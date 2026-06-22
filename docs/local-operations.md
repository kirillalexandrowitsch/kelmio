# Local Operations

Kelmio includes a localhost-only operations profile for metrics, dashboards,
alerts, scheduled backups and automated restore verification. The profile is
optional and remains stopped during normal development unless operations QA is
required.

## Service Lifecycle

Start the application stack:

```sh
docker compose up -d --build
make setup-db
```

Start and verify the monitoring profile:

```sh
make monitoring-up
make monitoring-check
```

Stop only monitoring services:

```sh
make monitoring-down
```

Stop the complete local stack while preserving volumes:

```sh
docker compose --profile monitoring down
```

Do not add `-v` unless local database and monitoring data should be deleted.

## Monitoring Endpoints

Default localhost endpoints are:

| Service | URL |
|---|---|
| Backend metrics | `http://localhost:8080/metrics` |
| Email worker metrics | `http://localhost:9091/metrics` |
| Backup worker metrics | `http://localhost:9092/metrics` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |
| Alertmanager | `http://localhost:9093` |

`make monitoring-check` validates service health, Prometheus targets and alert
rules, Grafana datasource/dashboard provisioning, backup/restore metrics and a
temporary Alertmanager alert.

The localhost profile requires `METRICS_AUTH_TOKEN` to be empty. Production-like
configuration requires a private token of at least 32 characters whenever
metrics are enabled.

## Collected Signals

- normalized HTTP route, method, status, request count and duration;
- PostgreSQL readiness;
- login outcomes without user identifiers;
- email outbox state totals;
- email worker heartbeat, attempts and results;
- backup attempt/success time, duration, result and artifact count;
- restore drill attempt/success time, duration, result and verified artifact.

Metrics must not contain raw paths, query strings, issue titles, email
addresses, usernames, cookies, request bodies or invite/reset tokens.

## Alerts

Prometheus loads alerts for:

- backend metrics or database readiness failure;
- unavailable/stale email worker and terminal delivery failures;
- email worker batch errors;
- unavailable, failed or stale scheduled backups;
- unavailable, failed or stale restore drills.

Alertmanager uses a local receiver only. No email, webhook or chat destination
is contacted by the default profile.

## Scheduled Backup Cycle

The backup worker starts with the monitoring profile and immediately runs a
full cycle:

1. Create a compressed PostgreSQL dump using an atomic temporary file.
2. Restore it into the isolated `restore-drill-postgres` service.
3. Verify migrations, core tables and required workspace/user data.
4. Clean the isolated schema.
5. Apply retention only after successful verification.

Defaults:

| Variable | Default |
|---|---|
| `BACKUP_INTERVAL` | `24h` |
| `BACKUP_RETRY_INTERVAL` | `5m` |
| `BACKUP_RETENTION_COUNT` | `7` |
| `BACKUP_DIR` | `backups` |
| `BACKUP_METRICS_PORT` | `9092` |
| `RESTORE_DRILL_ENABLED` | `true` |
| `RESTORE_DRILL_TIMEOUT` | `5m` |

Scheduled artifacts use `kelmio-scheduled-*.sql.gz`. Retention never removes
manual `make backup` artifacts and always preserves the latest verified copy.
Restore state is written atomically to `backups/restore-drill-state.json` with
mode `0600`.

See [Backup And Restore](backup-restore.md) for manual backup and destructive
restore commands.

## One-Off Operations

Create and verify one scheduled backup:

```sh
make backup-runner-once
```

Re-run the restore drill for the latest scheduled artifact:

```sh
make restore-drill-once
```

Run the isolated corruption/recovery scenario:

```sh
make smoke-operations
```

The smoke uses a temporary backup directory, rejects a corrupted artifact,
preserves the previous successful state, recovers with a valid artifact and
confirms that the source database was not changed.

## Full Local Operations QA

Use this sequence after operations changes:

```sh
make smoke-production
make smoke-api
make smoke-email-delivery
make smoke-operations
make monitoring-up
make monitoring-check
make monitoring-down
make prod-config-check
make prod-compose-check
make prod-stack-qa
```

`make prod-stack-qa` creates an isolated production-like Compose project,
applies migrations, bootstraps an admin, validates Caddy TLS/security behavior,
creates a backup and verifies its restore. It removes its temporary containers,
volumes, credentials and backup artifacts on exit.

## Failure Triage

1. Use `X-Request-ID` to correlate API failures with backend logs.
2. Inspect `/api/v1/version` to confirm runtime metadata.
3. Run `make email-diagnostics` for outbox failures.
4. Run `make monitoring-check` for metrics, alerts and provisioning failures.
5. Inspect `backups/restore-drill-state.json` for the stable restore error code.
6. Reproduce backup/restore behavior with `make smoke-operations` before any
   destructive restore.

Local operations logs and state files must never contain database/SMTP
passwords, authentication cookies, CSRF values or raw reset/invite tokens.
