#!/bin/sh
set -eu

BACKUP=${BACKUP:-${1:-}}
POSTGRES_SERVICE=${POSTGRES_SERVICE:-postgres}
POSTGRES_DB_OVERRIDE=${POSTGRES_DB:-}
POSTGRES_USER_OVERRIDE=${POSTGRES_USER:-}

usage() {
	printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz RESTORE_CONFIRM=I_UNDERSTAND make restore'
	printf '%s\n' '   or: RESTORE_CONFIRM=I_UNDERSTAND scripts/restore-db.sh backups/file.sql.gz'
}

compose() {
	if [ -n "${ENV_FILE:-}" ]; then
		docker compose --env-file "$ENV_FILE" "$@"
	else
		docker compose "$@"
	fi
}

if [ -z "$BACKUP" ]; then
	usage >&2
	exit 2
fi

if [ ! -r "$BACKUP" ]; then
	printf 'Backup file is not readable: %s\n' "$BACKUP" >&2
	exit 2
fi

case "$BACKUP" in
	*.sql | *.sql.gz) ;;
	*)
		printf 'Backup file must end with .sql or .sql.gz: %s\n' "$BACKUP" >&2
		exit 2
		;;
esac

if [ "${RESTORE_CONFIRM:-}" != "I_UNDERSTAND" ]; then
	printf '%s\n' 'Refusing destructive restore.' >&2
	printf '%s\n' 'Set RESTORE_CONFIRM=I_UNDERSTAND to replace the selected database schema.' >&2
	exit 2
fi

if [ "${BACKUP##*.}" = "gz" ]; then
	gzip -t "$BACKUP"
fi

printf 'Restoring PostgreSQL backup into service "%s"...\n' "$POSTGRES_SERVICE"

compose exec -T "$POSTGRES_SERVICE" sh -c '
	db_name=${1:-${POSTGRES_DB:-team_task_tracker}}
	db_user=${2:-${POSTGRES_USER:-team_task_tracker}}
	psql -q -v ON_ERROR_STOP=1 -U "$db_user" -d "$db_name" \
		-c "SET client_min_messages TO WARNING; DROP SCHEMA IF EXISTS public CASCADE;"
' sh "$POSTGRES_DB_OVERRIDE" "$POSTGRES_USER_OVERRIDE" >/dev/null

if [ "${BACKUP##*.}" = "gz" ]; then
	gzip -dc "$BACKUP" | compose exec -T "$POSTGRES_SERVICE" sh -c '
		db_name=${1:-${POSTGRES_DB:-team_task_tracker}}
		db_user=${2:-${POSTGRES_USER:-team_task_tracker}}
		exec psql -q -v ON_ERROR_STOP=1 -U "$db_user" -d "$db_name"
	' sh "$POSTGRES_DB_OVERRIDE" "$POSTGRES_USER_OVERRIDE" >/dev/null
else
	compose exec -T "$POSTGRES_SERVICE" sh -c '
		db_name=${1:-${POSTGRES_DB:-team_task_tracker}}
		db_user=${2:-${POSTGRES_USER:-team_task_tracker}}
		exec psql -q -v ON_ERROR_STOP=1 -U "$db_user" -d "$db_name"
	' sh "$POSTGRES_DB_OVERRIDE" "$POSTGRES_USER_OVERRIDE" <"$BACKUP" >/dev/null
fi

printf 'Restore complete: %s\n' "$BACKUP"
