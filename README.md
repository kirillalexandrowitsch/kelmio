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

Team API smoke test:

```sh
curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/team/members
```

Projects API smoke test:

```sh
curl -i -c /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin","password":"admin12345"}' \
  http://localhost:8080/api/v1/auth/login

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"key":"CORE","name":"Core Platform","description":"Main product workspace"}' \
  http://localhost:8080/api/v1/projects

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/projects
```

Issues API smoke test:

```sh
PROJECT_ID="$(curl -s -b /tmp/team-task-tracker.cookies http://localhost:8080/api/v1/projects \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).projects[0].id));')"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Create first task\",\"priority\":\"high\"}" \
  http://localhost:8080/api/v1/issues

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?project_id=$PROJECT_ID"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?project_id=$PROJECT_ID&status=todo&priority=high"

ISSUE_ID="$(curl -s -b /tmp/team-task-tracker.cookies "http://localhost:8080/api/v1/issues?project_id=$PROJECT_ID" \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).issues[0].id));')"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"body":"Looks good for the first pass."}' \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/comments"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/comments"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/activity"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}' \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/transition"
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
