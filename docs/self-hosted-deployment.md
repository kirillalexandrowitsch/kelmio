# Self-Hosted Deployment

This guide deploys Team Task Tracker on a single Linux host with Docker Compose, PostgreSQL, and Caddy-managed HTTPS.

The production stack is intended for a small team on one backend instance. It is not a multi-node or managed SaaS deployment.

## Prerequisites

- A Linux host with current Docker Engine and Docker Compose.
- A real DNS hostname pointing to the host, for example `tasks.example.com`.
- Inbound TCP ports `80` and `443` open for Caddy and certificate provisioning.
- A private location for production secrets and PostgreSQL backups.
- Repository access on the host.

Do not expose PostgreSQL publicly. `docker-compose.prod.yml` keeps PostgreSQL and application services on the internal Compose network and publishes only Caddy ports.

## Production Configuration

Create a private env file:

```sh
cp deploy/production.env.example deploy/production.env
```

`deploy/production.env` is ignored by Git. Replace every placeholder before starting production:

- set `PUBLIC_APP_HOST`, `PUBLIC_APP_URL`, and `TRUSTED_ORIGINS` to the same real HTTPS origin;
- use a private PostgreSQL password; arbitrary strong passwords with URL-special characters are supported;
- keep `SESSION_COOKIE_SECURE=true`;
- use a private `CSRF_SECRET` of at least 32 characters;
- set a strong one-time `BOOTSTRAP_ADMIN_PASSWORD`;
- set `APP_VERSION`, `BUILD_COMMIT`, and `BUILD_TIME` for deployment diagnostics.

By default, the backend safely constructs its connection URL from `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_SSLMODE`. An explicit non-empty `DATABASE_URL` overrides those values and must contain a PostgreSQL host and database name without a URL fragment.

The public Caddy ports default to `80` and `443`. Set `PRODUCTION_HTTP_PORT` and `PRODUCTION_HTTPS_PORT` only when the host needs different published ports, such as isolated production-like QA.

Validate the real private configuration and Caddyfile:

```sh
ENV_FILE=deploy/production.env make prod-compose-check
```

Production config fails fast when required HTTPS, trusted origin, secure cookie, database, or CSRF settings are unsafe.

## First Deployment

Build the production images and start PostgreSQL:

```sh
docker compose --env-file deploy/production.env -f docker-compose.prod.yml build
docker compose --env-file deploy/production.env -f docker-compose.prod.yml up -d postgres
```

Apply migrations:

```sh
docker compose --env-file deploy/production.env -f docker-compose.prod.yml \
  run --rm backend /app/bin/migrate
```

Create the first workspace and admin:

```sh
docker compose --env-file deploy/production.env -f docker-compose.prod.yml \
  --profile tools run --rm bootstrap-admin
```

The bootstrap command only succeeds when `workspaces`, `users`, and `workspace_members` are empty. It never updates an existing user or resets an existing password. After it succeeds, remove `BOOTSTRAP_ADMIN_PASSWORD` from `deploy/production.env`.

Do not run `/app/bin/seed` in production. The seed command creates demo records and known localhost credentials.

Start the full stack:

```sh
docker compose --env-file deploy/production.env -f docker-compose.prod.yml up -d
docker compose --env-file deploy/production.env -f docker-compose.prod.yml ps
```

Verify the public endpoints:

```sh
curl -fsS https://tasks.example.com/healthz
curl -fsS https://tasks.example.com/readyz
curl -fsS https://tasks.example.com/api/v1/version
```

Open the configured HTTPS URL and sign in with the bootstrap admin username and password.

## Update Flow

Create and verify a backup before every update:

```sh
COMPOSE_FILE=docker-compose.prod.yml ENV_FILE=deploy/production.env make backup
BACKUP=backups/team-task-tracker-YYYYMMDD-HHMMSS.sql.gz make restore-check
```

Then update and rebuild:

```sh
git pull --ff-only
ENV_FILE=deploy/production.env make prod-compose-check

docker compose --env-file deploy/production.env -f docker-compose.prod.yml build
docker compose --env-file deploy/production.env -f docker-compose.prod.yml \
  run --rm backend /app/bin/migrate
docker compose --env-file deploy/production.env -f docker-compose.prod.yml up -d
```

After the update, check health, readiness, runtime version, logs, and the production smoke:

```sh
API_BASE_URL=https://tasks.example.com \
TRUSTED_ORIGIN=https://tasks.example.com \
ADMIN_LOGIN=production_admin \
ADMIN_PASSWORD='<production-admin-password>' \
EXPECT_SECURE_COOKIE=true \
EXPECT_HSTS=true \
make smoke-production
```

Use the configured `RATE_LIMIT_LOGIN_PER_MINUTE` value when it differs from `10`.

## Backup, Restore, And Rollback

The full backup and destructive restore workflow is documented in [backup-restore.md](backup-restore.md).

For rollback:

1. Keep the verified pre-update backup and previous Git commit available.
2. Deploy the previous commit and rebuild the production images.
3. Restore the verified backup if the newer migrations or application changed stored data.
4. Re-run health, readiness, version, and production smoke checks.

There is no automatic migration downgrade. Never restore an older application over newer database data without a verified backup and an intentional rollback plan.

## Troubleshooting

Inspect service state and logs:

```sh
docker compose --env-file deploy/production.env -f docker-compose.prod.yml ps
docker compose --env-file deploy/production.env -f docker-compose.prod.yml logs --tail=200 backend
docker compose --env-file deploy/production.env -f docker-compose.prod.yml logs --tail=200 caddy
docker compose --env-file deploy/production.env -f docker-compose.prod.yml logs --tail=200 postgres
```

Operational signals:

- `/healthz` confirms that the backend process responds.
- `/readyz` confirms that the backend can reach PostgreSQL.
- `/api/v1/version` identifies the deployed version, commit, environment, and build time.
- Every backend response includes `X-Request-ID`; production backend logs are JSON and include the same request ID.
- Caddy needs correct DNS plus reachable ports `80` and `443` to provision TLS.

Request logs intentionally exclude query strings, headers, cookies, bodies, passwords, session tokens, CSRF tokens, and invite tokens.

## Production QA

Before the first real deployment, run `make prod-stack-qa` to verify an isolated clean-room stack. Use [v3-local-production-qa.md](v3-local-production-qa.md) for config validation, localhost hardening smoke, real HTTPS checks, and the full V1/V2/V3 regression baseline.
