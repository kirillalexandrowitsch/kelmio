#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres}"

cookie_jar="$(mktemp "${TMPDIR:-/tmp}/kelmio-email-diagnostics.XXXXXX")"
cleanup() {
	rm -f "$cookie_jar"
}
trap cleanup EXIT INT TERM

login_payload=$(printf '{"login":"%s","password":"%s"}' "$ADMIN_LOGIN" "$ADMIN_PASSWORD")
if curl -fsS "$API_BASE_URL/readyz" >/dev/null 2>&1 &&
	curl -fsS -c "$cookie_jar" -H "Content-Type: application/json" -d "$login_payload" "$API_BASE_URL/api/v1/auth/login" >/dev/null 2>&1; then
	curl -fsS -b "$cookie_jar" "$API_BASE_URL/api/v1/email/diagnostics"
	printf '\n'
	exit 0
fi

printf '%s\n' "Backend diagnostics unavailable; reading a safe email_outbox summary from PostgreSQL." >&2

if [ -n "${ENV_FILE:-}" ]; then
	docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T "$POSTGRES_SERVICE" sh -c '
		psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
			-c "SELECT status, count(*) AS count FROM email_outbox GROUP BY status ORDER BY status;" \
			-c "SELECT id::text, email_type, regexp_replace(recipient_email, '\''(^.).*(@.*$)'\'', '\''\1***\2'\'') AS recipient_email, attempt_count, left(regexp_replace(coalesce(last_error, '\'''\''), '\''(password|token|secret|api[_-]?key)=([^[:space:]]+)'\'', '\''\1=[redacted]'\'', '\''gi'\''), 160) AS last_error, updated_at FROM email_outbox WHERE status = '\''failed'\'' ORDER BY updated_at DESC, id DESC LIMIT 10;"
	'
else
	docker compose -f "$COMPOSE_FILE" exec -T "$POSTGRES_SERVICE" sh -c '
		psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
			-c "SELECT status, count(*) AS count FROM email_outbox GROUP BY status ORDER BY status;" \
			-c "SELECT id::text, email_type, regexp_replace(recipient_email, '\''(^.).*(@.*$)'\'', '\''\1***\2'\'') AS recipient_email, attempt_count, left(regexp_replace(coalesce(last_error, '\'''\''), '\''(password|token|secret|api[_-]?key)=([^[:space:]]+)'\'', '\''\1=[redacted]'\'', '\''gi'\''), 160) AS last_error, updated_at FROM email_outbox WHERE status = '\''failed'\'' ORDER BY updated_at DESC, id DESC LIMIT 10;"
	'
fi
