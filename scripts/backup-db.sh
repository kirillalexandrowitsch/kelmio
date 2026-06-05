#!/bin/sh
set -eu

BACKUP_DIR=${BACKUP_DIR:-backups}
POSTGRES_SERVICE=${POSTGRES_SERVICE:-postgres}
POSTGRES_DB_OVERRIDE=${POSTGRES_DB:-}
POSTGRES_USER_OVERRIDE=${POSTGRES_USER:-}

compose() {
	if [ -n "${ENV_FILE:-}" ]; then
		docker compose --env-file "$ENV_FILE" "$@"
	else
		docker compose "$@"
	fi
}

mkdir -p "$BACKUP_DIR"

timestamp=$(date -u +"%Y%m%d-%H%M%S")
backup_path="${BACKUP_DIR%/}/team-task-tracker-${timestamp}.sql.gz"
if [ -e "$backup_path" ]; then
	backup_path="${BACKUP_DIR%/}/team-task-tracker-${timestamp}-$$.sql.gz"
fi

tmp_sql="${backup_path}.tmp.sql"
tmp_gz="${backup_path}.tmp"

cleanup() {
	rm -f "$tmp_sql" "$tmp_gz"
}

trap cleanup EXIT INT TERM

printf 'Creating PostgreSQL backup from service "%s"...\n' "$POSTGRES_SERVICE"

compose exec -T "$POSTGRES_SERVICE" sh -c '
	db_name=${1:-${POSTGRES_DB:-team_task_tracker}}
	db_user=${2:-${POSTGRES_USER:-team_task_tracker}}
	exec pg_dump --no-owner --no-privileges --schema=public -U "$db_user" -d "$db_name"
' sh "$POSTGRES_DB_OVERRIDE" "$POSTGRES_USER_OVERRIDE" >"$tmp_sql"

gzip -c "$tmp_sql" >"$tmp_gz"
mv "$tmp_gz" "$backup_path"
rm -f "$tmp_sql"
trap - EXIT INT TERM

printf 'Backup written: %s\n' "$backup_path"
