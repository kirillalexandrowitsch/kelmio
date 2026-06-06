#!/bin/sh
set -eu

PROJECT_NAME="${PROD_QA_PROJECT_NAME:-team-task-tracker-v3-qa-$$}"
HTTP_PORT="${PROD_QA_HTTP_PORT:-18080}"
HTTPS_PORT="${PROD_QA_HTTPS_PORT:-18443}"
PUBLIC_HOST="${PROD_QA_PUBLIC_HOST:-tasks.localhost}"
ADMIN_USERNAME="${PROD_QA_ADMIN_USERNAME:-production_admin}"
ADMIN_PASSWORD="${PROD_QA_ADMIN_PASSWORD:-qa-admin-password-123}"
POSTGRES_PASSWORD="${PROD_QA_POSTGRES_PASSWORD:-qa:strong@password#value}"
QA_DIR="$(mktemp -d "${TMPDIR:-/tmp}/team-task-tracker-prod-qa.XXXXXX")"
ENV_FILE="$QA_DIR/production.env"
BACKUP_DIR="$QA_DIR/backups"
PUBLIC_URL="https://$PUBLIC_HOST:$HTTPS_PORT"

compose() {
	docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" -f docker-compose.prod.yml "$@"
}

cleanup() {
	compose --profile tools down -v --remove-orphans >/dev/null 2>&1 || true
	rm -rf "$QA_DIR"
}
trap cleanup EXIT INT TERM

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		fail "Missing required command: $1"
	fi
}

require_command curl
require_command docker
require_command grep

cat >"$ENV_FILE" <<EOF
PUBLIC_APP_HOST=$PUBLIC_HOST
PUBLIC_APP_URL=$PUBLIC_URL
TRUSTED_ORIGINS=$PUBLIC_URL
VITE_API_BASE_URL=/api

POSTGRES_DB=team_task_tracker
POSTGRES_USER=team_task_tracker
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_SSLMODE=disable

PRODUCTION_HTTP_PORT=$HTTP_PORT
PRODUCTION_HTTPS_PORT=$HTTPS_PORT

SESSION_TTL=24h
SESSION_COOKIE_SECURE=true
CSRF_SECRET=qa-private-csrf-secret-0123456789abcdef
RATE_LIMIT_LOGIN_PER_MINUTE=10

APP_VERSION=v3-final-qa
BUILD_COMMIT=clean-room
BUILD_TIME=2026-06-06T00:00:00Z

BOOTSTRAP_WORKSPACE_NAME=V3 Production QA
BOOTSTRAP_ADMIN_EMAIL=production-admin@example.com
BOOTSTRAP_ADMIN_USERNAME=$ADMIN_USERNAME
BOOTSTRAP_ADMIN_DISPLAY_NAME=Production Admin
BOOTSTRAP_ADMIN_PASSWORD=$ADMIN_PASSWORD
EOF

printf 'Validating isolated production Compose config...\n'
compose --profile tools config >/dev/null

printf 'Building production images...\n'
compose build backend frontend

printf 'Starting isolated PostgreSQL and applying migrations...\n'
compose up -d postgres
compose run --rm backend /app/bin/migrate

printf 'Bootstrapping first production admin...\n'
compose --profile tools run --rm bootstrap-admin

printf 'Checking repeated bootstrap refusal...\n'
if compose --profile tools run --rm bootstrap-admin >"$QA_DIR/repeat-bootstrap.log" 2>&1; then
	fail "Repeated production admin bootstrap unexpectedly succeeded"
fi
if ! grep -q "bootstrap refused because production data already exists" "$QA_DIR/repeat-bootstrap.log"; then
	cat "$QA_DIR/repeat-bootstrap.log" >&2
	fail "Repeated bootstrap did not report the expected refusal"
fi

printf 'Starting isolated production application stack...\n'
compose up -d backend frontend caddy

attempts=60
until curl -kfsS "$PUBLIC_URL/readyz" >/dev/null 2>&1; do
	attempts=$((attempts - 1))
	if [ "$attempts" -le 0 ]; then
		compose ps >&2 || true
		compose logs --tail=100 backend caddy postgres >&2 || true
		fail "Timed out waiting for isolated production stack"
	fi
	sleep 1
done

printf 'Checking SPA direct route through Caddy...\n'
curl -kfsS "$PUBLIC_URL/issues" | grep -q '<div id="root"></div>' || fail "Production SPA direct route did not return the frontend"

printf 'Running production hardening smoke through Caddy TLS...\n'
CURL_INSECURE=true \
API_BASE_URL="$PUBLIC_URL" \
TRUSTED_ORIGIN="$PUBLIC_URL" \
ADMIN_LOGIN="$ADMIN_USERNAME" \
ADMIN_PASSWORD="$ADMIN_PASSWORD" \
RATE_LIMIT_LOGIN_PER_MINUTE=10 \
EXPECT_SECURE_COOKIE=true \
EXPECT_HSTS=true \
sh scripts/smoke-production.sh

printf 'Checking production backup and isolated restore...\n'
COMPOSE_PROJECT_NAME="$PROJECT_NAME" \
COMPOSE_FILE=docker-compose.prod.yml \
ENV_FILE="$ENV_FILE" \
BACKUP_DIR="$BACKUP_DIR" \
sh scripts/backup-db.sh

backup_path="$(find "$BACKUP_DIR" -type f -name '*.sql.gz' -print -quit)"
if [ -z "$backup_path" ]; then
	fail "Production QA backup was not created"
fi
BACKUP="$backup_path" sh scripts/restore-check-db.sh

printf 'Isolated production stack QA passed\n'
