# Team Task Tracker

Free self-hosted team task tracker for small teams, built with Go, React + TypeScript, PostgreSQL, and Docker.

Current status: V1-V3 and the post-V3 maintainability cleanup are fully completed. The V4 feature set and testing baseline are implemented; final V4 QA polish remains.

## Features

- Workspace authentication, admin/member roles, team management, and invite-link onboarding.
- Projects, issues, board/list views, labels, comments, activity, hierarchy, subtasks, and issue links.
- Sprint planning, active sprint board, story points, dashboard summary, and saved filters.
- In-app notifications with unread state and read/read-all actions.
- Project-specific workflows with custom statuses, enforced transition graphs, and atomic archive-with-replacement.
- Project roles, dynamic project and active sprint boards, and synchronous single-pass automation with activity and notifications.
- Production config validation, secure cookies, trusted-origin CORS, CSRF protection, login rate limiting, and security headers.
- Production Docker images, single-origin Docker Compose stack, Caddy-managed HTTPS, backup/restore scripts, and first-admin bootstrap.
- JSON production logs, `X-Request-ID`, runtime version metadata, pagination, performance indexes, GitHub Actions CI, API smoke, and Playwright e2e.

V1-V4 scope and decisions are documented in:

- [docs/mvp-plan.md](docs/mvp-plan.md)
- [docs/v2-plan.md](docs/v2-plan.md)
- [docs/v3-plan.md](docs/v3-plan.md)
- [docs/v1-v3-cleanup-plan.md](docs/v1-v3-cleanup-plan.md)
- [docs/v4-plan.md](docs/v4-plan.md)

## Stack

- Backend: Go modular monolith, `net/http`, `pgx`
- Frontend: React + TypeScript, Vite
- Database: PostgreSQL 16
- Infrastructure: Docker Compose, Caddy for production HTTPS
- Tests: Go unit/integration/race tests, Vitest/React Testing Library, API and production hardening smoke, Playwright e2e, fast and full GitHub Actions workflows

## V4 Workflows, Permissions, And Automation

Each project has its own ordered workflow statuses and allowed transition graph. Used statuses can be archived only with an active replacement, which moves affected issues atomically. The project board and active sprint board use the project's live workflow.

The API keeps the legacy immutable issue `status` key for V1-V3 compatibility, while the UI and additive API fields use workflow statuses. Sprint and dashboard completion metrics use workflow category `done`, not a hardcoded status key.

| Role | Project access |
|---|---|
| Workspace admin | Full access to every project |
| Project lead | Manage project members, workflow, automation, issues, comments, and sprints |
| Contributor | Create and update issues, comments, and sprints |
| Viewer | Read-only project access |
| No project membership | Cannot see project data |

Automation rules execute synchronously, atomically, and single-pass inside the originating issue transaction. Rule actions do not trigger additional rules; applied changes are recorded in issue activity and can create final-result notifications.

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

`make setup-db` applies migrations and runs the idempotent localhost demo seed. The V4 seed makes `admin` a DEMO project lead and `demo_member` a contributor, adds the custom `review` status, DEMO-11/12, automation rules, readable automation activity, and automation notifications alongside the existing V1-V3 demo data. Do not use the demo seed in production.

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

## V4 Local QA Flow

```sh
make setup-db
make smoke-api
make frontend-e2e
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
GOCACHE=/private/tmp/team-task-tracker-gocache make verify
```

Manual V4 checks:

- In Projects, manage lead/contributor/viewer roles and confirm viewer/no-membership restrictions.
- In Workflow, create, edit, reorder, and archive a status with replacement, then save a restricted transition graph.
- Open the project board and active sprint board and confirm both use the project's custom workflow columns.
- Create an automation rule, trigger it from an issue, and verify the resulting issue activity and notification.

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

`make prod-stack-qa` creates and removes an isolated production Compose stack with a special-character PostgreSQL password, migrations, first-admin bootstrap, internal TLS, hardening smoke, backup, and restore-check. `make smoke-api` covers V1-V4 business flows, including custom workflows, project roles, automation, activity, notifications, and V4 seed checks. `make smoke-production` covers request IDs, security headers, CORS, cookies, CSRF, request size limits, and login rate limiting. Playwright covers V1-V4 flows, including project membership, workflow settings and boards, permissions, automation management, activity, and notifications.

The fast GitHub Actions workflow runs on every push and pull request. The separate `V1 V2 V3 V4 full QA` job can be started manually and also runs weekly; it covers the complete development, integration, browser, and isolated production-stack baseline.

## Operations And Observability

- Every backend response includes `X-Request-ID`; use it to correlate an incident with backend logs.
- `APP_ENV=production` enables JSON logs without query strings, headers, cookies, bodies, passwords, or tokens.
- `GET /api/v1/version` returns deployment version, commit, environment, and optional build time.
- The login limiter is in-memory and single-node; it resets after backend restart and is not synchronized between backend instances.
- Backups contain sensitive application data and must be stored privately outside Git.

## V1-V4 Implementation Status

V1, V2, V3, and the post-V3 cleanup are fully completed. The complete V4 feature set and its component, smoke, integration, and Playwright testing baseline are implemented. V4 is not yet declared fully completed: the remaining planned step is the final V4 QA polish and completion audit.
