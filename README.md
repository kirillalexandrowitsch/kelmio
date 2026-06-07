# Team Task Tracker

Free self-hosted team task tracker for small teams, built with Go, React + TypeScript, PostgreSQL, and Docker.

Current status: V1, V2, V3, and the post-V3 maintainability cleanup are fully completed. The project is ready for V4 planning.

## Features

- Workspace authentication, admin/member roles, team management, and invite-link onboarding.
- Projects, issues, board/list views, labels, comments, activity, hierarchy, subtasks, and issue links.
- Sprint planning, active sprint board, story points, dashboard summary, and saved filters.
- In-app notifications with unread state and read/read-all actions.
- Production config validation, secure cookies, trusted-origin CORS, CSRF protection, login rate limiting, and security headers.
- Production Docker images, single-origin Docker Compose stack, Caddy-managed HTTPS, backup/restore scripts, and first-admin bootstrap.
- JSON production logs, `X-Request-ID`, runtime version metadata, pagination, performance indexes, GitHub Actions CI, API smoke, and Playwright e2e.

V1, V2, and V3 scope and decisions are documented in:

- [docs/mvp-plan.md](docs/mvp-plan.md)
- [docs/v2-plan.md](docs/v2-plan.md)
- [docs/v3-plan.md](docs/v3-plan.md)
- [docs/v1-v3-cleanup-plan.md](docs/v1-v3-cleanup-plan.md)

## Stack

- Backend: Go modular monolith, `net/http`, `pgx`
- Frontend: React + TypeScript, Vite
- Database: PostgreSQL 16
- Infrastructure: Docker Compose, Caddy for production HTTPS
- Tests: Go unit/integration/race tests, Vitest/React Testing Library, API and production hardening smoke, Playwright e2e, fast and full GitHub Actions workflows

## Local Development

Requirements: Docker with Compose, Go, Node.js/npm, and `curl`.

```sh
make doctor

# terminal 1
make dev

# terminal 2, after services are up
make setup-db
make smoke-api
```

Open `http://localhost:5173` and sign in:

```text
username: admin
password: admin12345
```

Development endpoints:

- frontend: `http://localhost:5173`
- backend health: `http://localhost:8080/healthz`
- backend readiness: `http://localhost:8080/readyz`
- runtime metadata: `http://localhost:8080/api/v1/version`
- PostgreSQL: `localhost:15432`

`make setup-db` applies migrations and runs the idempotent localhost demo seed. The seed creates `admin`, `demo_member`, DEMO issues, sprints, saved filters, and notifications. Do not use the demo seed in production.

Useful commands:

```sh
make help
make dev
make down
make logs
make setup-db
make smoke-api
make smoke-production
make prod-stack-qa
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
```

## Self-Hosted Production

Use the production deployment guide:

- [Self-hosted deployment](docs/self-hosted-deployment.md)
- [Backup and restore](docs/backup-restore.md)
- [V3 local and production QA](docs/v3-local-production-qa.md)

The production flow uses `docker-compose.prod.yml`, a private ignored `deploy/production.env`, Caddy HTTPS, explicit migrations, and a one-time first-admin bootstrap. The bootstrap refuses to run when workspace or user data already exists.

The backend accepts an explicit `DATABASE_URL` override. When it is absent, the production stack passes separate `POSTGRES_*` values and the backend safely constructs the PostgreSQL URL, including URL-encoding arbitrary strong passwords.

Minimum production security requirements:

- a real HTTPS origin for `PUBLIC_APP_URL` and exact `TRUSTED_ORIGINS`;
- `SESSION_COOKIE_SECURE=true`;
- private PostgreSQL password and 32+ character `CSRF_SECRET`;
- private, strong bootstrap admin password removed from the env file after first use;
- verified backup before every update.

## Verification

Baseline checks:

```sh
make prod-config-check
make prod-compose-check
make prod-stack-qa
make smoke-production
make smoke-api
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
git diff --check
```

`make prod-stack-qa` creates and removes an isolated production Compose stack with a special-character PostgreSQL password, migrations, first-admin bootstrap, internal TLS, hardening smoke, backup, and restore-check. `make smoke-api` covers V1/V2 business flows and V3 pagination/version regression. `make smoke-production` covers request IDs, security headers, CORS, cookies, CSRF, request size limits, and login rate limiting. Playwright covers V1/V2 flows plus V3 invite onboarding.

The fast GitHub Actions workflow runs on every push and pull request. The separate Full QA workflow can be started manually and also runs weekly; it covers the complete development, integration, browser, and isolated production-stack baseline.

## Operations And Observability

- Every backend response includes `X-Request-ID`; use it to correlate an incident with backend logs.
- `APP_ENV=production` enables JSON logs without query strings, headers, cookies, bodies, passwords, or tokens.
- `GET /api/v1/version` returns deployment version, commit, environment, and optional build time.
- The login limiter is in-memory and single-node; it resets after backend restart and is not synchronized between backend instances.
- Backups contain sensitive application data and must be stored privately outside Git.

## V1-V3 Completion Status

V1, V2, and V3 are fully completed. On June 7, 2026, the post-V3 cleanup audit again passed the complete automated baseline, two consecutive idempotent database setups, frontend component tests, production config and Compose validation, isolated production-stack TLS/security/bootstrap/backup/restore QA, API smoke, backend integration and race tests, all Playwright e2e scenarios, production image builds, static checks, and dependency audit. No known V1-V3 blocker bugs remain.
