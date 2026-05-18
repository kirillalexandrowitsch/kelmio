# Team Task Tracker

Локальный team task tracker для небольших команд.

Текущий этап: базовый scaffold проекта. Бизнес-логика будет добавляться постепенно по плану из [docs/mvp-plan.md](docs/mvp-plan.md).

## Stack

- Backend: Go
- Frontend: React + TypeScript
- Database: PostgreSQL
- Local infrastructure: Docker Compose

## Local Development

Пока проект находится на Phase 0, доступны базовые сервисы:

- frontend: `http://localhost:5173`
- backend health: `http://localhost:8080/healthz`
- PostgreSQL: `localhost:15432`

Команды:

```sh
make dev
make down
make logs
make db-up
make migrate-up
make seed
make setup-db
make backend-dev
make backend-test
```

Backend health:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Auth API smoke test:

```sh
curl -i -c /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin","password":"admin12345"}' \
  http://localhost:8080/api/v1/auth/login

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/auth/me

curl -i -b /tmp/team-task-tracker.cookies \
  -X POST http://localhost:8080/api/v1/auth/logout
```

Для локального запуска frontend без Docker:

```sh
make frontend-install
make frontend-dev
```

Для локального запуска backend без полного Docker stack:

```sh
make setup-db
make backend-dev
```

Локальный seed создает:

```text
workspace: Local Workspace
email: admin@example.com
username: admin
password: admin12345
```

## Environment

Шаблон переменных окружения находится в `.env.example`.

Для Docker Compose сейчас используются безопасные development defaults. Позже, когда появятся миграции и подключение backend к базе, env-конфигурация будет расширена.
