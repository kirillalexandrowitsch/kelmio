#!/bin/sh
set -eu

BACKUP=${BACKUP:-${1:-}}
CONTAINER_NAME=${RESTORE_CHECK_CONTAINER:-kelmio-restore-check-$(date -u +"%Y%m%d%H%M%S")-$$}
VOLUME_NAME=${RESTORE_CHECK_VOLUME:-${CONTAINER_NAME}-data}
POSTGRES_IMAGE=${RESTORE_CHECK_POSTGRES_IMAGE:-postgres:16-alpine}
CHECK_DB=restore_check
CHECK_USER=restore_check
CHECK_PASSWORD=restore_check

usage() {
	printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz make restore-check'
	printf '%s\n' '   or: scripts/restore-check-db.sh backups/file.sql.gz'
}

cleanup() {
	docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
	docker volume rm "$VOLUME_NAME" >/dev/null 2>&1 || true
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

if [ "${BACKUP##*.}" = "gz" ]; then
	gzip -t "$BACKUP"
fi

trap cleanup EXIT INT TERM

printf 'Starting isolated PostgreSQL restore check...\n'
docker volume create "$VOLUME_NAME" >/dev/null
docker run -d \
	--name "$CONTAINER_NAME" \
	-e POSTGRES_DB="$CHECK_DB" \
	-e POSTGRES_USER="$CHECK_USER" \
	-e POSTGRES_PASSWORD="$CHECK_PASSWORD" \
	-v "$VOLUME_NAME":/var/lib/postgresql/data \
	"$POSTGRES_IMAGE" >/dev/null

attempts=60
# The image's temporary init server accepts socket connections before it
# shuts down. TCP is available only after the final server has started.
until docker exec "$CONTAINER_NAME" pg_isready -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" >/dev/null 2>&1; do
	attempts=$((attempts - 1))
	if [ "$attempts" -le 0 ]; then
		printf '%s\n' 'Timed out waiting for restore-check PostgreSQL.' >&2
		exit 1
	fi
	sleep 1
done

docker exec "$CONTAINER_NAME" psql -q -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" -c "SET client_min_messages TO WARNING; DROP SCHEMA IF EXISTS public CASCADE;" >/dev/null

if [ "${BACKUP##*.}" = "gz" ]; then
	gzip -dc "$BACKUP" | docker exec -i "$CONTAINER_NAME" psql -q -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" >/dev/null
else
	docker exec -i "$CONTAINER_NAME" psql -q -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" <"$BACKUP" >/dev/null
fi

core_table_count=$(docker exec "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" -tAc "
	SELECT count(*)
	FROM information_schema.tables
	WHERE table_schema = 'public'
	  AND table_name IN ('workspaces', 'users', 'workspace_members', 'projects', 'issues');
")

if [ "$core_table_count" != "5" ]; then
	printf 'Restore check failed: expected 5 core tables, found %s.\n' "$core_table_count" >&2
	exit 1
fi

docker exec "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" -c "SELECT count(*) FROM workspaces;" >/dev/null
docker exec "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$CHECK_USER" -d "$CHECK_DB" -c "SELECT count(*) FROM issues;" >/dev/null

printf 'Restore check passed: %s\n' "$BACKUP"
