#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
PROMETHEUS_IMAGE=prom/prometheus:v3.12.0
ALERTMANAGER_IMAGE=prom/alertmanager:v0.33.0
SMOKE_ALERT=TeamTaskTrackerMonitoringSmoke
ALERT_POSTED=false

fail() {
  printf '%s\n' "monitoring check failed: $*" >&2
  exit 1
}

wait_for_url() {
  name=$1
  url=$2
  attempts=0
  while ! curl -fsS "$url" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 60 ]; then
      fail "$name did not become ready at $url"
    fi
    sleep 1
  done
}

wait_for_prometheus_value() {
  description=$1
  query=$2
  pattern=$3
  attempts=0
  while :; do
    response=$(curl -fsS --get --data-urlencode "query=$query" "$PROMETHEUS_URL/api/v1/query")
    if printf '%s' "$response" | grep -Eq "$pattern"; then
      return 0
    fi
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 60 ]; then
      fail "$description did not become available"
    fi
    sleep 1
  done
}

published_port() {
  service=$1
  container_port=$2
  port=$(docker compose --profile monitoring port "$service" "$container_port" 2>/dev/null | sed -n '1s/.*://p')
  [ -n "$port" ] || fail "cannot resolve published port for $service:$container_port"
  printf '%s' "$port"
}

future_timestamp() {
  if date -u -v+5M '+%Y-%m-%dT%H:%M:%SZ' >/dev/null 2>&1; then
    date -u -v+5M '+%Y-%m-%dT%H:%M:%SZ'
  else
    date -u -d '+5 minutes' '+%Y-%m-%dT%H:%M:%SZ'
  fi
}

resolve_smoke_alert() {
  [ "$ALERT_POSTED" = true ] || return 0
  now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  payload=$(printf '[{"labels":{"alertname":"%s","service":"monitoring-check","severity":"info"},"annotations":{"summary":"Synthetic localhost monitoring check"},"startsAt":"%s","endsAt":"%s"}]' "$SMOKE_ALERT" "$now" "$now")
  curl -fsS -H 'Content-Type: application/json' -d "$payload" "$ALERTMANAGER_URL/api/v2/alerts" >/dev/null 2>&1 || true
}

if [ -n "${METRICS_AUTH_TOKEN:-}" ]; then
  fail "the localhost monitoring profile requires METRICS_AUTH_TOKEN to be empty"
fi
if [ -f "$ROOT_DIR/.env" ] && grep -Eq '^[[:space:]]*METRICS_AUTH_TOKEN=.+$' "$ROOT_DIR/.env"; then
  fail "the localhost monitoring profile requires METRICS_AUTH_TOKEN to be empty in .env"
fi

cd "$ROOT_DIR"

printf '%s\n' 'Validating Prometheus and alert rules...'
docker run --rm \
  --entrypoint /bin/promtool \
  -v "$ROOT_DIR/deploy/monitoring/prometheus:/etc/prometheus:ro" \
  "$PROMETHEUS_IMAGE" \
  check config /etc/prometheus/prometheus.yml >/dev/null
docker run --rm \
  --entrypoint /bin/promtool \
  -v "$ROOT_DIR/deploy/monitoring/prometheus:/etc/prometheus:ro" \
  "$PROMETHEUS_IMAGE" \
  check rules /etc/prometheus/alerts.yml >/dev/null

printf '%s\n' 'Validating Alertmanager configuration...'
docker run --rm \
  --entrypoint /bin/amtool \
  -v "$ROOT_DIR/deploy/monitoring/alertmanager:/etc/alertmanager:ro" \
  "$ALERTMANAGER_IMAGE" \
  check-config /etc/alertmanager/alertmanager.yml >/dev/null

PROMETHEUS_PORT=$(published_port prometheus 9090)
GRAFANA_PORT=$(published_port grafana 3000)
ALERTMANAGER_PORT=$(published_port alertmanager 9093)
PROMETHEUS_URL="http://127.0.0.1:$PROMETHEUS_PORT"
GRAFANA_URL="http://127.0.0.1:$GRAFANA_PORT"
ALERTMANAGER_URL="http://127.0.0.1:$ALERTMANAGER_PORT"

trap resolve_smoke_alert EXIT HUP INT TERM

printf '%s\n' 'Waiting for monitoring services...'
wait_for_url Prometheus "$PROMETHEUS_URL/-/ready"
wait_for_url Grafana "$GRAFANA_URL/api/health"
wait_for_url Alertmanager "$ALERTMANAGER_URL/-/ready"

printf '%s\n' 'Checking Prometheus scrape targets...'
for job in backend email-worker backup-worker prometheus; do
  query=$(printf 'up{job="%s"}' "$job")
  wait_for_prometheus_value "Prometheus target $job" "$query" '"value":\[[^]]*,"1"\]'
done

printf '%s\n' 'Checking Prometheus alert rules and Alertmanager discovery...'
rules=$(curl -fsS "$PROMETHEUS_URL/api/v1/rules?type=alert")
for rule in BackendMetricsUnavailable DatabaseNotReady EmailWorkerUnavailable EmailWorkerHeartbeatStale EmailDeliveryFailures EmailWorkerBatchErrors BackupRunnerUnavailable BackupFailed BackupStale RestoreDrillStale; do
  printf '%s' "$rules" | grep -q "\"name\":\"$rule\"" || fail "Prometheus did not load alert rule $rule"
done
alertmanagers=$(curl -fsS "$PROMETHEUS_URL/api/v1/alertmanagers")
printf '%s' "$alertmanagers" | grep -q 'alertmanager:9093' || fail 'Prometheus did not discover Alertmanager'

printf '%s\n' 'Checking scheduled backup metrics...'
wait_for_prometheus_value \
  'backup worker successful scheduled backup result' \
  'team_task_tracker_backup_last_result{job="backup-worker",result="success"}' \
  '"value":\[[^]]*,"1"\]'
wait_for_prometheus_value \
  'backup worker retained scheduled artifact' \
  'team_task_tracker_backup_artifacts{job="backup-worker"}' \
  '"value":\[[^]]*,"[1-9][0-9]*"\]'

printf '%s\n' 'Checking Grafana provisioning...'
grafana_health=$(curl -fsS "$GRAFANA_URL/api/health")
printf '%s' "$grafana_health" | grep -q '"database"' || fail 'Grafana health response is incomplete'
datasource=$(curl -fsS "$GRAFANA_URL/api/datasources/uid/team-task-tracker-prometheus")
printf '%s' "$datasource" | grep -q '"uid":"team-task-tracker-prometheus"' || fail 'Grafana Prometheus datasource is missing'
dashboards=$(curl -fsS "$GRAFANA_URL/api/search?query=Team%20Task%20Tracker%20Operations")
printf '%s' "$dashboards" | grep -q '"uid":"team-task-tracker-operations"' || fail 'Grafana operations dashboard is missing'

printf '%s\n' 'Checking Alertmanager with a temporary synthetic alert...'
now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
ends_at=$(future_timestamp)
payload=$(printf '[{"labels":{"alertname":"%s","service":"monitoring-check","severity":"info"},"annotations":{"summary":"Synthetic localhost monitoring check"},"startsAt":"%s","endsAt":"%s"}]' "$SMOKE_ALERT" "$now" "$ends_at")
curl -fsS -H 'Content-Type: application/json' -d "$payload" "$ALERTMANAGER_URL/api/v2/alerts" >/dev/null
ALERT_POSTED=true

attempts=0
while :; do
  alerts=$(curl -fsS "$ALERTMANAGER_URL/api/v2/alerts?active=true&silenced=false&inhibited=false")
  if printf '%s' "$alerts" | grep -q "\"alertname\":\"$SMOKE_ALERT\""; then
    break
  fi
  attempts=$((attempts + 1))
  if [ "$attempts" -ge 15 ]; then
    fail 'Alertmanager did not expose the synthetic alert'
  fi
  sleep 1
done

resolve_smoke_alert
ALERT_POSTED=false
trap - EXIT HUP INT TERM

printf '%s\n' 'Monitoring check passed.'
printf '%s\n' "Prometheus: $PROMETHEUS_URL"
printf '%s\n' "Grafana: $GRAFANA_URL"
printf '%s\n' "Alertmanager: $ALERTMANAGER_URL"
