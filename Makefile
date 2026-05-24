SHELL := /bin/sh

.PHONY: help doctor dev down logs ps db-up wait-db migrate-up seed setup-db backend-dev backend-test frontend-install frontend-dev frontend-build frontend-test smoke-api verify

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
	@printf '%s\n' '  make backend-dev      Run backend locally'
	@printf '%s\n' '  make backend-test     Run Go tests'
	@printf '%s\n' '  make frontend-install Install frontend dependencies'
	@printf '%s\n' '  make frontend-dev     Run frontend dev server locally'
	@printf '%s\n' '  make frontend-build   Build frontend'
	@printf '%s\n' '  make frontend-test    Run frontend tests'
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

backend-dev:
	cd backend && go run ./cmd/api

backend-test:
	cd backend && go test ./...

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm test

smoke-api:
	./scripts/smoke-api.sh

verify:
	sh -n scripts/smoke-api.sh
	sh -n scripts/doctor.sh
	./scripts/doctor.sh
	cd backend && go test ./...
	cd frontend && npm test
	cd frontend && npm run build
	docker compose config >/dev/null
