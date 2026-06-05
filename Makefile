SHELL := /bin/sh

.PHONY: help doctor dev down logs ps db-up wait-db migrate-up seed setup-db backup restore restore-check backend-dev backend-test backend-integration-test frontend-install frontend-dev frontend-build frontend-test frontend-e2e-install frontend-e2e smoke-api verify

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
	@printf '%s\n' '  make restore          Restore BACKUP into selected PostgreSQL database'
	@printf '%s\n' '  make restore-check    Verify BACKUP in isolated temporary PostgreSQL'
	@printf '%s\n' '  make backend-dev      Run backend locally'
	@printf '%s\n' '  make backend-test     Run Go tests'
	@printf '%s\n' '  make backend-integration-test Run Go integration tests against local PostgreSQL'
	@printf '%s\n' '  make frontend-install Install frontend dependencies'
	@printf '%s\n' '  make frontend-dev     Run frontend dev server locally'
	@printf '%s\n' '  make frontend-build   Build frontend'
	@printf '%s\n' '  make frontend-test    Run frontend tests'
	@printf '%s\n' '  make frontend-e2e-install Install Playwright browser dependencies'
	@printf '%s\n' '  make frontend-e2e     Run browser e2e smoke against localhost frontend'
	@printf '%s\n' '  make smoke-api        Run API smoke test against localhost backend'
	@printf '%s\n' '  make verify           Run local non-destructive verification checks'

doctor:
	./scripts/doctor.sh

dev:
	docker compose up --build

down:
	docker compose down

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

restore:
	@if [ -z "$(BACKUP)" ]; then printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz RESTORE_CONFIRM=I_UNDERSTAND make restore' >&2; exit 2; fi
	sh scripts/restore-db.sh "$(BACKUP)"

restore-check:
	@if [ -z "$(BACKUP)" ]; then printf '%s\n' 'Usage: BACKUP=backups/file.sql.gz make restore-check' >&2; exit 2; fi
	sh scripts/restore-check-db.sh "$(BACKUP)"

backend-dev:
	cd backend && go run ./cmd/api

backend-test:
	cd backend && go test ./...

backend-integration-test: wait-db
	cd backend && go test -tags=integration ./internal/...

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

verify:
	sh -n scripts/smoke-api.sh
	sh -n scripts/doctor.sh
	sh -n scripts/backup-db.sh
	sh -n scripts/restore-db.sh
	sh -n scripts/restore-check-db.sh
	./scripts/doctor.sh
	cd backend && go test ./...
	cd frontend && npm test
	cd frontend && npm run build
	docker compose config >/dev/null
