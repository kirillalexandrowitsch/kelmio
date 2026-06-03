# Team Task Tracker

Локальный team task tracker для небольших команд.

Текущий статус: localhost V2 feature set реализован по [docs/v2-plan.md](docs/v2-plan.md). Остался финальный плановый шаг V2: `final V2 QA polish`.

## Stack

- Backend: Go, `net/http`, `pgx`
- Frontend: React + TypeScript, Vite, обычный CSS
- Database: PostgreSQL
- Local infrastructure: Docker Compose
- Tests: Go tests, backend PostgreSQL integration tests, API smoke script, Playwright browser e2e smoke

Проект остается localhost-first: без cloud deployment, email, WebSocket, Redis и внешних интеграций. V2 расширяет V1 функциональность, но сохраняет простой Go modular monolith, REST JSON API, PostgreSQL, React + TypeScript и Docker Compose.

## V2 Features

- Route-level navigation для основных разделов приложения.
- Issue hierarchy: `epic`, `story/task`, `subtask`, parent/children UI и activity.
- Issue links: `blocks` и `relates` связи между задачами.
- Sprints: list/detail, backlog planning, start/complete flow, active sprint board.
- Story points и sprint summary на dashboard: progress, done/open points, workload by assignee.
- Saved issue filters/views с обработкой missing project/sprint/assignee/label values.
- In-app notifications без email/WebSocket: unread count, dropdown, page, mark read/read all.
- Расширенные V2 seed data: DEMO issues, sprints, links, saved filters и notifications.

## Local Development

Локально доступны базовые сервисы:

- frontend: `http://localhost:5173`
- backend health: `http://localhost:8080/healthz`
- PostgreSQL: `localhost:15432`

Опционально можно создать локальный `.env` из шаблона:

```sh
cp .env.example .env
```

Docker Compose использует эти переменные автоматически; если `.env` не создан, применяются localhost defaults из `docker-compose.yml`.

Первый запуск с нуля:

```sh
make doctor

# terminal 1
make dev

# terminal 2, after postgres/backend/frontend containers are up
make setup-db
make smoke-api
```

После seed можно войти во frontend:

```text
url: http://localhost:5173
username: admin
password: admin12345
```

Команды:

```sh
make doctor
make dev
make down
make logs
make db-up
make wait-db
make migrate-up
make seed
make setup-db
make backend-dev
make backend-test
make backend-integration-test
make frontend-install
make frontend-dev
make frontend-build
make frontend-test
make frontend-e2e-install
make frontend-e2e
make smoke-api
make verify
```

Backend health:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

API smoke test:

```sh
# terminal 1
make dev

# terminal 2, after backend is up
make setup-db
make smoke-api
```

`make smoke-api` проверяет V1 flow и V2 сценарии: readiness -> auth guard -> admin login -> member access guards -> team/users list -> project/issue/label/comment/activity flow -> hierarchy -> issue links -> sprint lifecycle -> saved filters -> notifications. По умолчанию используется `http://localhost:8080`, `admin` / `admin12345` и `demo_member` / `demo12345`; при необходимости можно переопределить `API_BASE_URL`, `ADMIN_LOGIN`, `ADMIN_PASSWORD`, `MEMBER_LOGIN` и `MEMBER_PASSWORD`.

Перед commit/push удобно запускать:

```sh
make verify
```

`make verify` выполняет non-destructive проверки: local toolchain doctor, shell syntax для smoke-скрипта, backend tests, frontend tests, frontend build и проверку Docker Compose config.

Backend integration test:

```sh
# terminal 1
make dev

# terminal 2
make backend-integration-test
```

`make backend-integration-test` проверяет миграции и базовые PostgreSQL операции в изолированной временной schema, затем удаляет ее. Рабочие localhost-данные не очищаются.

Browser e2e smoke test:

```sh
# one-time browser install
make frontend-e2e-install

# terminal 1
make dev

# terminal 2, after setup-db has seeded admin/demo data
make setup-db
make frontend-e2e
```

`make frontend-e2e` запускает Playwright browser smoke: V1 login/project/issue/board/comment flow плюс V2 hierarchy/links, sprint workflow, saved filters и notifications. По умолчанию используется `http://localhost:5173`, backend API `http://localhost:8080`, `admin` / `admin12345`; при необходимости можно переопределить `E2E_BASE_URL`, `E2E_API_BASE_URL`, `E2E_ADMIN_LOGIN` и `E2E_ADMIN_PASSWORD`.

## V2 Local QA Flow

Для полной локальной проверки V2:

```sh
# terminal 1
make dev

# terminal 2, after postgres/backend/frontend containers are up
make setup-db
make smoke-api
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
```

Ручные localhost сценарии для V2:

- Открыть `http://localhost:5173`, войти как `admin` / `admin12345`, проверить direct navigation между Dashboard, Issues, Board, Sprints, Notifications, Team, Labels, Account.
- В Issues открыть задачу, создать subtask, проверить hierarchy block и activity.
- В issue detail добавить linked issue с типом `blocks` или `relates`, проверить linked issues block и activity.
- В Sprints создать sprint, добавить backlog issue, start sprint, поменять status на active sprint board, complete sprint.
- В issue list выставить filters, сохранить view, очистить filters, применить saved filter, удалить saved filter.
- Создать/использовать member, назначить issue и добавить comment с `@username`, затем войти под member и проверить notification badge/dropdown/page, mark read и mark all read.

Auth API smoke test:

```sh
curl -i -c /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin","password":"admin12345"}' \
  http://localhost:8080/api/v1/auth/login

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/auth/me

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"display_name":"Updated Name"}' \
  http://localhost:8080/api/v1/auth/profile

# Use a test account cookie here unless you intentionally want to change admin.
curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"current_password":"current-password","new_password":"new-password123"}' \
  http://localhost:8080/api/v1/auth/password

curl -i -b /tmp/team-task-tracker.cookies \
  -X POST http://localhost:8080/api/v1/auth/logout
```

Team API smoke test:

```sh
curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"email":"member@example.com","username":"member","display_name":"Member","password":"member12345","role":"member"}' \
  http://localhost:8080/api/v1/team/members

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/team/members

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/users

MEMBER_ID="$(curl -s -b /tmp/team-task-tracker.cookies http://localhost:8080/api/v1/team/members \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).members.find((member) => member.username === "member").id));')"

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"role":"admin","is_active":true}' \
  "http://localhost:8080/api/v1/team/members/$MEMBER_ID"

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"password":"member54321"}' \
  "http://localhost:8080/api/v1/team/members/$MEMBER_ID/password"
```

Labels API smoke test:

```sh
curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"name":"frontend","color":"#4e795d"}' \
  http://localhost:8080/api/v1/labels

curl -b /tmp/team-task-tracker.cookies \
  http://localhost:8080/api/v1/labels

DELETE_LABEL_ID="$(curl -s -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"temp-label-$(date +%s)\",\"color\":\"#923c2d\"}" \
  http://localhost:8080/api/v1/labels \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).id));')"

curl -i -X DELETE -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/labels/$DELETE_LABEL_ID"
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

ARCHIVE_PROJECT_ID="$(curl -s -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"key\":\"TMP$(date +%s | tail -c 5)\",\"name\":\"Temporary Project\",\"description\":\"Archive smoke project\"}" \
  http://localhost:8080/api/v1/projects \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).id));')"

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"name":"Updated Temporary Project","description":"Updated project description"}' \
  "http://localhost:8080/api/v1/projects/$ARCHIVE_PROJECT_ID"

curl -i -X POST -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/projects/$ARCHIVE_PROJECT_ID/archive"
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
  "http://localhost:8080/api/v1/issues?q=first"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?sort=priority_desc"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?due=overdue"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?due=today"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?due=due_soon"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?due=no_due"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?project_id=$PROJECT_ID&status=todo&priority=high"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?assignee_id=unassigned"

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

COMMENT_ID="$(curl -s -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/comments" \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).comments[0].id));')"

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"body":"Looks good after editing the comment."}' \
  "http://localhost:8080/api/v1/comments/$COMMENT_ID"

DELETE_COMMENT_ID="$(curl -s -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"body":"Temporary comment to delete."}' \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/comments" \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).id));')"

curl -i -X DELETE -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/comments/$DELETE_COMMENT_ID"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/activity"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}' \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/transition"

ASSIGNEE_ID="$(curl -s -b /tmp/team-task-tracker.cookies http://localhost:8080/api/v1/users \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).members.find((member) => member.username === "demo_member").id));')"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"assignee_id\":\"$ASSIGNEE_ID\"}" \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/assign"

curl -i -X PATCH -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d '{"title":"Create first task with details","description":"Updated from smoke test.","issue_type":"task","priority":"medium","due_date":"2026-05-31"}' \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID"

LABEL_ID="$(curl -s -b /tmp/team-task-tracker.cookies http://localhost:8080/api/v1/labels \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).labels[0].id));')"

curl -i -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Create labeled task\",\"priority\":\"medium\",\"label_ids\":[\"$LABEL_ID\"]}" \
  http://localhost:8080/api/v1/issues

curl -i -X PUT -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"label_ids\":[\"$LABEL_ID\"]}" \
  "http://localhost:8080/api/v1/issues/$ISSUE_ID/labels"

curl -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues?label_id=$LABEL_ID"

ARCHIVE_ISSUE_ID="$(curl -s -b /tmp/team-task-tracker.cookies \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Archive smoke task\",\"priority\":\"low\"}" \
  http://localhost:8080/api/v1/issues \
  | node -e 'let data=""; process.stdin.on("data", c => data += c); process.stdin.on("end", () => console.log(JSON.parse(data).id));')"

curl -i -X POST -b /tmp/team-task-tracker.cookies \
  "http://localhost:8080/api/v1/issues/$ARCHIVE_ISSUE_ID/archive"
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

demo user:
email: demo.member@example.com
username: demo_member
password: demo12345

demo project:
key: DEMO
labels: frontend, backend, bug
issues: DEMO-1 ... DEMO-10
sprints: Demo Active Sprint, Demo Next Sprint, Demo Completed Sprint
links: DEMO-9 blocks DEMO-6, DEMO-7 relates DEMO-6
saved filters: admin and demo_member V2 planning views
notifications: seeded unread admin/demo_member notifications
```

## Environment

Шаблон переменных окружения находится в `.env.example`.

Для Docker Compose можно не создавать `.env`: development defaults уже заданы в `docker-compose.yml`.

Если нужно поменять порты, credentials или seed-пользователей, создай `.env` из `.env.example` и измени нужные значения локально. Файл `.env` не коммитится.

Для запуска backend вне Docker используется `DATABASE_URL` с host `localhost`. Внутри Docker Compose backend получает database URL автоматически из `POSTGRES_DB`, `POSTGRES_USER` и `POSTGRES_PASSWORD`.

Если меняешь `FRONTEND_PORT`, обнови также `FRONTEND_URL` и `VITE_API_BASE_URL`, чтобы CORS и frontend API calls указывали на актуальные localhost-адреса.
