# Backup And Restore

V3 keeps backup and restore as an operational CLI flow. There is no in-app admin panel for database backups.

Backups contain application data from the PostgreSQL `public` schema. Treat every backup file as sensitive data and store it outside the Git repository.

## Local Development

Start or prepare the local database first:

```sh
make setup-db
```

Create a compressed SQL backup:

```sh
make backup
```

The command writes a file like:

```text
backups/team-task-tracker-20260605-120000.sql.gz
```

Verify a backup safely in an isolated temporary PostgreSQL container:

```sh
BACKUP=backups/team-task-tracker-20260605-120000.sql.gz make restore-check
```

Restore into the currently selected Compose PostgreSQL database:

```sh
BACKUP=backups/team-task-tracker-20260605-120000.sql.gz RESTORE_CONFIRM=I_UNDERSTAND make restore
```

`make restore` is destructive. It drops and recreates the selected database `public` schema before loading the backup.

## Scheduled Local Backups

The localhost monitoring profile includes a dedicated backup worker. It creates
one backup immediately, then runs every 24 hours by default:

```sh
make monitoring-up
```

Scheduled files use a separate name so retention never removes manual backups:

```text
backups/team-task-tracker-scheduled-20260620-120000.sql.gz
```

Create one scheduled-format backup without starting the monitoring stack:

```sh
make backup-runner-once
```

The worker keeps the latest seven scheduled backups by default. Retention runs
only after a successful dump, never removes the newest scheduled backup, and
does not touch files created by `make backup`.

Prometheus scrapes backup status from the worker. Grafana displays the latest
result, backup age, and retained artifact count. The backup worker belongs only
to the localhost `monitoring` profile; production-like Compose keeps the manual
backup flow documented below.

## Production Compose

For the production stack, point the commands at `docker-compose.prod.yml` and the private env file used to run production:

```sh
COMPOSE_FILE=docker-compose.prod.yml ENV_FILE=deploy/production.env make backup
```

Before restoring to production, validate the backup away from production:

```sh
BACKUP=backups/team-task-tracker-20260605-120000.sql.gz make restore-check
```

Then restore only if you intentionally want to replace the selected production database schema:

```sh
COMPOSE_FILE=docker-compose.prod.yml ENV_FILE=deploy/production.env BACKUP=backups/team-task-tracker-20260605-120000.sql.gz RESTORE_CONFIRM=I_UNDERSTAND make restore
```

## Before-Update Flow

Use this flow before updating a self-hosted instance:

1. Confirm the current production stack is healthy.
2. Run `COMPOSE_FILE=docker-compose.prod.yml ENV_FILE=deploy/production.env make backup`.
3. Run `BACKUP=<created-file> make restore-check`.
4. Store the verified backup somewhere durable and private.
5. Continue with the application update only after the restore check passes.

## Configuration

The scripts support these variables:

- `COMPOSE_FILE`: Compose file selection, for example `docker-compose.prod.yml`.
- `ENV_FILE`: optional Compose env file passed as `docker compose --env-file`.
- `POSTGRES_SERVICE`: Compose service name, default `postgres`.
- `POSTGRES_DB`: optional database override.
- `POSTGRES_USER`: optional database user override.
- `BACKUP_DIR`: output directory for `make backup`, default `backups`.
- `BACKUP_INTERVAL`: delay after a successful scheduled backup, default `24h`.
- `BACKUP_RETRY_INTERVAL`: delay after a failed scheduled backup, default `5m`.
- `BACKUP_RETENTION_COUNT`: number of scheduled backups to retain, default `7`.
- `BACKUP_METRICS_PORT`: localhost backup-worker metrics port, default `9092`.
- `BACKUP`: backup path for `make restore` and `make restore-check`.
- `RESTORE_CONFIRM`: must be `I_UNDERSTAND` for destructive restore.
