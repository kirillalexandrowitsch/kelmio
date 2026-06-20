#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
QA_DIR=$(mktemp -d "${TMPDIR:-/tmp}/team-task-tracker-operations-smoke.XXXXXX")
RESTORE_SERVICE=restore-drill-postgres
RESTORE_WAS_RUNNING=false

fail() {
	printf '%s\n' "operations smoke failed: $*" >&2
	exit 1
}

cleanup() {
	rm -rf "$QA_DIR"
	if [ "$RESTORE_WAS_RUNNING" = false ]; then
		docker compose --profile monitoring stop "$RESTORE_SERVICE" >/dev/null 2>&1 || true
		docker compose --profile monitoring rm -f "$RESTORE_SERVICE" >/dev/null 2>&1 || true
	fi
}

trap cleanup EXIT INT TERM
cd "$ROOT_DIR"

if [ -n "$(docker compose --profile monitoring ps -q "$RESTORE_SERVICE" 2>/dev/null)" ]; then
	RESTORE_WAS_RUNNING=true
fi

printf '%s\n' 'Checking source database state...'
docker compose exec -T postgres sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null
source_before=$(docker compose exec -T postgres sh -c \
	'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT concat_ws('"'"'|'"'"', (SELECT count(*) FROM workspaces), (SELECT count(*) FROM users), (SELECT count(*) FROM issues));"')

printf '%s\n' 'Building operations worker...'
docker compose --profile monitoring build backup-worker >/dev/null

printf '%s\n' 'Creating and verifying a scheduled backup...'
BACKUP_DIR="$QA_DIR" BACKUP_RETENTION_COUNT=2 docker compose --profile monitoring run --rm backup-worker --once

scheduled=$(find "$QA_DIR" -maxdepth 1 -type f -name 'team-task-tracker-scheduled-*.sql.gz' -print)
[ "$(printf '%s\n' "$scheduled" | sed '/^$/d' | wc -l | tr -d ' ')" = "1" ] || fail 'expected exactly one scheduled backup'
valid_backup=$scheduled
state_file="$QA_DIR/restore-drill-state.json"
[ -r "$state_file" ] || fail 'restore drill state was not created'
[ "$(jq -r '.last_result' "$state_file")" = "success" ] || fail 'initial restore drill did not succeed'
[ "$(jq -r '.last_success_migration_version' "$state_file")" -gt 0 ] || fail 'migration version was not verified'
first_success=$(jq -r '.last_success_at' "$state_file")

printf '%s\n' 'Rejecting a corrupted latest backup...'
corrupted="$QA_DIR/team-task-tracker-scheduled-20990101-000000.sql.gz"
printf '%s\n' 'corrupted backup' >"$corrupted"
if BACKUP_DIR="$QA_DIR" docker compose --profile monitoring run --rm backup-worker --restore-only; then
	fail 'corrupted backup unexpectedly passed restore drill'
fi
[ "$(jq -r '.last_result' "$state_file")" = "failure" ] || fail 'failed restore result was not persisted'
[ "$(jq -r '.last_error_code' "$state_file")" = "artifact_invalid" ] || fail 'unexpected corrupted backup error code'
[ "$(jq -r '.last_success_at' "$state_file")" = "$first_success" ] || fail 'previous successful restore state was lost'

printf '%s\n' 'Recovering with the previous valid backup...'
rm -f "$corrupted"
BACKUP_DIR="$QA_DIR" docker compose --profile monitoring run --rm backup-worker --restore-only
[ "$(jq -r '.last_result' "$state_file")" = "success" ] || fail 'restore drill did not recover'
[ "$(jq -r '.last_backup_file' "$state_file")" = "$(basename "$valid_backup")" ] || fail 'recovery used an unexpected backup'

printf '%s\n' 'Checking isolated target cleanup and source isolation...'
target_tables=$(docker compose --profile monitoring exec -T "$RESTORE_SERVICE" psql -U restore_drill -d restore_drill -Atc \
	"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';")
[ "$target_tables" = "0" ] || fail "isolated restore target retained $target_tables tables"
source_after=$(docker compose exec -T postgres sh -c \
	'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT concat_ws('"'"'|'"'"', (SELECT count(*) FROM workspaces), (SELECT count(*) FROM users), (SELECT count(*) FROM issues));"')
[ "$source_before" = "$source_after" ] || fail 'source database changed during restore drill'

printf '%s\n' 'Operations smoke passed.'
