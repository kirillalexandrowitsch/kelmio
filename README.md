# Team Task Tracker

Free self-hosted team task tracker for small teams, built with Go, React + TypeScript, PostgreSQL, and Docker.

Current status: V3 feature implementation is complete. The final V3 QA polish step from [docs/v3-plan.md](docs/v3-plan.md) remains before V3 is declared fully completed.

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

## Stack

- Backend: Go modular monolith, `net/http`, `pgx`
- Frontend: React + TypeScript, Vite
- Database: PostgreSQL 16
- Infrastructure: Docker Compose, Caddy for production HTTPS
- Tests: Go unit/integration tests, API smoke, production hardening smoke, Playwright e2e, GitHub Actions

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
make smoke-production
make smoke-api
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
git diff --check
```

`make smoke-api` covers V1/V2 business flows and V3 pagination/version regression. `make smoke-production` covers request IDs, security headers, CORS, cookies, CSRF, request size limits, and login rate limiting. Playwright covers V1/V2 flows plus V3 invite onboarding.

## Operations And Observability

- Every backend response includes `X-Request-ID`; use it to correlate an incident with backend logs.
- `APP_ENV=production` enables JSON logs without query strings, headers, cookies, bodies, passwords, or tokens.
- `GET /api/v1/version` returns deployment version, commit, environment, and optional build time.
- The login limiter is in-memory and single-node; it resets after backend restart and is not synchronized between backend instances.
- Backups contain sensitive application data and must be stored privately outside Git.

## V3 Completion Status

V3 implementation phases through production smoke/e2e and deployment documentation are complete. The remaining planned step is `17. final V3 QA polish`: run the full automated and manual localhost/production-like audit, fix any blocker or polish defects, then explicitly mark V3 fully completed.
