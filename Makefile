SHELL := /bin/sh

.PHONY: help doctor dev down logs ps db-up wait-db migrate-up seed setup-db backup backup-runner-once restore restore-check restore-drill-once smoke-operations smoke-email-delivery backend-dev email-worker email-diagnostics monitoring-up monitoring-check monitoring-down backend-test backend-integration-test frontend-install frontend-dev frontend-build frontend-test frontend-e2e-install frontend-e2e smoke-api smoke-production prod-config-check prod-compose-check prod-stack-qa verify

help:
	@printf '%s\n' 'Available commands:'
	@printf '%s\n' '  make doctor           Check local toolchain requirements'
	@printf '%s\n' '  make dev              Start local Docker stack'
	@printf '%s\n' '  make down             Stop local Docker stack'
	@printf '%s\n' '  make logs             Follow Docker logs'
	@printf '%s\n' '  make ps               Show Docker services'
	@printf '%s\n' '  make db-up            Start PostgreSQL only'
	@printf '%s\n' '  make wait-db          Wait until PostgreSQL is ready'
	@printf '%s\n' '  make migrate-up       Apply database migrations'
	@printf '%s\n' '  make seed             Create local admin seed data'
	@printf '%s\n' '  make setup-db         Start DB, migrate, and seed'
	@printf '%s\n' '  make backup           Create compressed PostgreSQL backup in backups/'
	@printf '%s\n' '  make backup-runner-once Create one scheduled-format backup through the worker'
	@printf '%s\n' '  make restore          Restore BACKUP into selected PostgreSQL database'
	@printf '%s\n' '  make restore-check    Verify BACKUP in isolated temporary PostgreSQL'
	@printf '%s\n' '  make restore-drill-once Verify the latest scheduled backup through the operations worker'
	@printf '%s\n' '  make smoke-operations Run backup and restore drill smoke checks'
	@printf '%s\n' '  make smoke-email-delivery Verify email worker retry and Mailpit recovery'
	@printf '%s\n' '  make backend-dev      Run backend locally'
	@printf '%s\n' '  make email-worker     Run email delivery worker locally'
	@printf '%s\n' '  make email-diagnostics Show read-only email outbox diagnostics'
	@printf '%s\n' '  make monitoring-up    Start local Prometheus, Grafana, and Alertmanager'
	@printf '%s\n' '  make monitoring-check Validate the running local monitoring stack'
	@printf '%s\n' '  make monitoring-down  Stop and remove local monitoring containers'
	@printf '%s\n' '  make backend-test     Run Go tests'
	@printf '%s\n' '  make backend-integration-test Run Go integration tests against local PostgreSQL'
	@printf '%s\n' '  make frontend-install Install frontend dependencies'
	@printf '%s\n' '  make frontend-dev     Run frontend dev server locally'
	@printf '%s\n' '  make frontend-build   Build frontend'
	@printf '%s\n' '  make frontend-test    Run frontend tests'
	@printf '%s\n' '  make frontend-e2e-install Install Playwright browser dependencies'
	@printf '%s\n' '  make frontend-e2e     Run browser e2e smoke against localhost frontend'
	@printf '%s\n' '  make smoke-api        Run API smoke test against localhost backend'
	@printf '%s\n' '  make smoke-production Run production-sensitive API smoke'
	@printf '%s\n' '  make prod-config-check Validate production backend config rules'
	@printf '%s\n' '  make prod-compose-check Validate production Compose and Caddy config'
	@printf '%s\n' '  make prod-stack-qa     Run isolated clean-room production stack QA'
	@printf '%s\n' '  make verify           Run local non-destructive verification checks'

doctor:
	./scripts/doctor.sh

dev:
	docker compose up --build

down:
	docker compose --profile monitoring down

logs:
	docker compose logs -f

ps:
	docker compose ps

db-up:
	docker compose up -d postgres

wait-db:
	docker compose exec -T postgres sh -c 'until pg_isready -U "$${POSTGRES_USER}" -d "$${POSTGRES_DB}"; do sleep 1; done'

migrate-up:
	cd backend && go run ./cmd/migrate

seed:
	cd backend && go run ./cmd/seed

setup-db: db-up wait-db migrate-up seed

backup:
	sh scripts/backup-db.sh

backup-runner-once:
	@mkdir -p "$${BACKUP_DIR:-backups}"
	docker compose --profile monitoring run --rm backup-worker --once

restore:
	@if [ -z "$(BACKUP)" ]; then printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz RESTORE_CONFIRM=I_UNDERSTAND make restore' >&2; exit 2; fi
	sh scripts/restore-db.sh "$(BACKUP)"

restore-check:
	@if [ -z "$(BACKUP)" ]; then printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz make restore-check' >&2; exit 2; fi
	sh scripts/restore-check-db.sh "$(BACKUP)"

restore-drill-once:
	@mkdir -p "$${BACKUP_DIR:-backups}"
	docker compose --profile monitoring run --rm backup-worker --restore-only

smoke-operations:
	sh scripts/smoke-operations.sh

smoke-email-delivery:
	sh scripts/smoke-email-delivery.sh

backend-dev:
	cd backend && go run ./cmd/api

email-worker:
	cd backend && go run ./cmd/email-worker

email-diagnostics:
	sh scripts/email-diagnostics.sh

monitoring-up:
	@mkdir -p "$${BACKUP_DIR:-backups}"
	docker compose --profile monitoring up -d restore-drill-postgres backup-worker alertmanager prometheus grafana

monitoring-check:
	sh scripts/check-monitoring.sh

monitoring-down:
	docker compose --profile monitoring stop grafana prometheus alertmanager backup-worker restore-drill-postgres
	docker compose --profile monitoring rm -f grafana prometheus alertmanager backup-worker restore-drill-postgres

backend-test:
	cd backend && go test ./...

backend-integration-test: wait-db
	cd backend && go test -tags=integration ./internal/... ./cmd/email-worker

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm test

frontend-e2e-install:
	cd frontend && npx playwright install chromium

frontend-e2e:
	cd frontend && npm run e2e

smoke-api:
	./scripts/smoke-api.sh

smoke-production:
	./scripts/smoke-production.sh

prod-config-check:
	cd backend && GOCACHE="$${GOCACHE:-$${TMPDIR:-/tmp}/team-task-tracker-gocache}" go test ./internal/config -run Production

prod-compose-check:
	docker compose --env-file "$${ENV_FILE:-deploy/production.env.example}" -f docker-compose.prod.yml config >/dev/null
	docker run --rm -v "$(CURDIR)/deploy/caddy/Caddyfile:/etc/caddy/Caddyfile:ro" caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile

prod-stack-qa:
	sh scripts/qa-production-stack.sh

verify:
	sh -n scripts/smoke-api.sh
	sh -n scripts/smoke-production.sh
	sh -n scripts/smoke-operations.sh
	sh -n scripts/smoke-email-delivery.sh
	sh -n scripts/qa-production-stack.sh
	sh -n scripts/doctor.sh
	sh -n scripts/backup-db.sh
	sh -n scripts/restore-db.sh
	sh -n scripts/restore-check-db.sh
	sh -n scripts/email-diagnostics.sh
	sh -n scripts/check-monitoring.sh
	./scripts/doctor.sh
	cd backend && go test ./...
	cd frontend && npm test
	cd frontend && npm run build
	docker compose config >/dev/null
