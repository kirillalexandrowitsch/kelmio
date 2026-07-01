#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
MAILPIT_BASE_URL="${MAILPIT_BASE_URL:-http://localhost:8025}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
COOKIE_JAR=$(mktemp "${TMPDIR:-/tmp}/kelmio-email-smoke.XXXXXX")
INVITE_ID=""
INVITE_TOKEN=""
CSRF_TOKEN=""

fail() {
	printf '%s\n' "email delivery smoke failed: $*" >&2
	exit 1
}

json_value() {
	node -e '
const expression = process.argv[1];
let input = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => input += chunk);
process.stdin.on("end", () => {
  const data = JSON.parse(input);
  const value = Function("data", `return ${expression}`)(data);
  if (value === undefined || value === null || value === false) process.exit(1);
  process.stdout.write(typeof value === "string" ? value : JSON.stringify(value));
});
' "$1"
}

cleanup() {
	docker compose up -d mailpit >/dev/null 2>&1 || true
	if [ -n "$INVITE_ID" ] && [ -n "$CSRF_TOKEN" ]; then
		curl -fsS -b "$COOKIE_JAR" \
			-H "Content-Type: application/json" \
			-H "X-CSRF-Token: $CSRF_TOKEN" \
			-d '{}' \
			"$API_BASE_URL/api/v1/team/invites/$INVITE_ID/revoke" >/dev/null 2>&1 || true
	fi
	rm -f "$COOKIE_JAR"
}

outbox_row() {
	docker compose exec -T postgres sh -c \
		"psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"SELECT id::text || '|' || status || '|' || attempt_count::text FROM email_outbox WHERE email_type = 'team_invite' AND template_data->>'invite_id' = '$INVITE_ID' ORDER BY created_at DESC, id DESC LIMIT 1;\""
}

wait_for_retry() {
	deadline=$(( $(date +%s) + 45 ))
	while [ "$(date +%s)" -lt "$deadline" ]; do
		row=$(outbox_row)
		status=$(printf '%s' "$row" | cut -d '|' -f 2)
		attempts=$(printf '%s' "$row" | cut -d '|' -f 3)
		if [ "$status" = "pending" ] && [ "${attempts:-0}" -ge 1 ]; then
			printf '%s' "$row" | cut -d '|' -f 1
			return 0
		fi
		sleep 1
	done
	return 1
}

wait_for_sent() {
	deadline=$(( $(date +%s) + 45 ))
	while [ "$(date +%s)" -lt "$deadline" ]; do
		row=$(outbox_row)
		status=$(printf '%s' "$row" | cut -d '|' -f 2)
		attempts=$(printf '%s' "$row" | cut -d '|' -f 3)
		if [ "$status" = "sent" ] && [ "${attempts:-0}" -ge 2 ]; then
			return 0
		fi
		sleep 1
	done
	return 1
}

trap cleanup EXIT INT TERM
cd "$ROOT_DIR"

command -v curl >/dev/null 2>&1 || fail 'curl is required'
command -v node >/dev/null 2>&1 || fail 'node is required'

printf '%s\n' 'Preparing email delivery services...'
docker compose up -d postgres backend email-worker mailpit >/dev/null
# Recreating the dev backend recompiles the mounted source via `go run`, so allow
# a generous readiness window for a cold compile on a loaded CI runner.
deadline=$(( $(date +%s) + 180 ))
until curl -fsS "$API_BASE_URL/readyz" >/dev/null 2>&1; do
	[ "$(date +%s)" -lt "$deadline" ] || fail 'backend did not become ready'
	sleep 1
done

curl -fsS -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
	-H 'Content-Type: application/json' \
	-d "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}" \
	"$API_BASE_URL/api/v1/auth/login" >/dev/null
CSRF_TOKEN=$(curl -fsS -b "$COOKIE_JAR" "$API_BASE_URL/api/v1/auth/csrf-token" | json_value 'data.csrf_token')

RUN_ID="$(date +%s)$$"
RECIPIENT="email-retry-$RUN_ID@example.com"

printf '%s\n' 'Stopping Mailpit and enqueueing an invite email...'
docker compose stop mailpit >/dev/null
CREATE_RESPONSE=$(curl -fsS -b "$COOKIE_JAR" \
	-H 'Content-Type: application/json' \
	-H "X-CSRF-Token: $CSRF_TOKEN" \
	-d "{\"email\":\"$RECIPIENT\",\"role\":\"member\"}" \
	"$API_BASE_URL/api/v1/team/invites")
INVITE_ID=$(printf '%s' "$CREATE_RESPONSE" | json_value 'data.id')
INVITE_TOKEN=$(printf '%s' "$CREATE_RESPONSE" | json_value 'data.accept_token')
printf '%s' "$CREATE_RESPONSE" | json_value 'data.email_delivery_status === "pending" && typeof data.email_queued_at === "string"' >/dev/null

OUTBOX_ID=$(wait_for_retry) || fail 'worker did not persist a retry while Mailpit was unavailable'

printf '%s\n' 'Restarting Mailpit and accelerating the isolated retry...'
docker compose up -d mailpit >/dev/null
deadline=$(( $(date +%s) + 30 ))
while ! curl -fsS "$MAILPIT_BASE_URL/api/v1/messages" >/dev/null 2>&1; do
	[ "$(date +%s)" -lt "$deadline" ] || fail 'Mailpit did not become ready'
	sleep 1
done

docker compose exec -T postgres sh -c \
	"psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1 -c \"UPDATE email_outbox SET next_attempt_at = NOW(), updated_at = NOW() WHERE id = '$OUTBOX_ID' AND status = 'pending';\"" >/dev/null
wait_for_sent || fail 'worker did not deliver the retried email'

curl -fsS "$MAILPIT_BASE_URL/api/v1/messages" | json_value \
	"Array.isArray(data.messages) && data.messages.some((message) => JSON.stringify(message).toLowerCase().includes(\"$RECIPIENT\"))" >/dev/null

WORKER_LOGS=$(docker compose logs --no-color email-worker)
if printf '%s' "$WORKER_LOGS" | grep -Fq "$RECIPIENT"; then
	fail 'worker logs leaked the recipient email'
fi
if printf '%s' "$WORKER_LOGS" | grep -Fq "$INVITE_TOKEN"; then
	fail 'worker logs leaked the invite token'
fi

printf '%s\n' 'Email delivery retry and recovery smoke passed.'
