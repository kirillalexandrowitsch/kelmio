# Team Task Tracker MVP Plan

## 1. Product Goal

Сделать локальный трекер задач для небольшой команды, который:

- бесплатен в использовании;
- запускается полностью на localhost через Docker;
- покрывает ежедневную работу команды без перегруза;
- закладывает архитектурную основу для будущего роста до более сильного аналога Jira.

Ключевой принцип первой версии: не строить "всю Jira", а собрать крепкий, быстрый и расширяемый MVP.

## 2. MVP Boundary

### Что входит в V1

- локальная аутентификация по email или username и password;
- одна workspace-модель с возможностью расширения на несколько workspaces позже;
- управление пользователями внутри workspace;
- проекты;
- задачи с типом, статусом, приоритетом, описанием, исполнителем, автором и дедлайном;
- backlog/list view задач;
- kanban-board по статусам;
- comments внутри задачи;
- labels/tags;
- базовые фильтры по статусу, исполнителю, проекту, приоритету и label;
- activity log по ключевым изменениям задачи;
- базовая роль `admin/member`;
- Docker-based локальный запуск.

### Что сознательно не входит в V1

- sprints;
- story points;
- time tracking;
- file attachments;
- email notifications;
- real-time updates через WebSocket;
- advanced permissions;
- custom workflows per project;
- automation rules;
- subtasks и epics;
- external integrations;
- mobile version;
- deployment в cloud.

Это не отказ от функционала, а защита проекта от расползания scope.

## 3. Main User Scenarios

Первая версия должна хорошо решать ровно такие сценарии:

1. Пользователь логинится в систему.
2. Пользователь создает проект.
3. Пользователь создает задачу в проекте.
4. Задача получает статус, приоритет, исполнителя и labels.
5. Команда видит задачи списком и на kanban-board.
6. Пользователь открывает карточку задачи, читает описание, комментарии и историю изменений.
7. Пользователь меняет статус задачи и переназначает исполнителя.
8. Админ управляет участниками workspace.

Если сценарий не помогает этим восьми пунктам, он почти наверняка не нужен для V1.

## 4. Architecture Direction

## Technology Selection Policy

Жесткие требования проекта:

- backend должен быть написан на Go;
- frontend должен быть написан на React + TypeScript;
- основная база данных должна быть PostgreSQL;
- локальный запуск должен работать через Docker.

Остальные технологии можно добавлять прагматично, если они дают реальную пользу:

- сокращают ручную работу;
- повышают типобезопасность;
- упрощают тестирование;
- делают локальную разработку стабильнее;
- не превращают V1 в тяжелую enterprise-систему раньше времени.

Для V1 разрешенные дополнительные технологии:

- `chi` для HTTP routing в Go;
- `pgx` для PostgreSQL driver/pool;
- `sqlc` для типобезопасного SQL-кода;
- `goose` для миграций;
- `bcrypt` или `argon2id` для password hashing;
- `Vite` для React dev/build tooling;
- `TanStack Query` для server state;
- `React Hook Form` + `Zod` для форм и валидации;
- `Tailwind CSS` для быстрой и консистентной UI-разработки;
- `Playwright` для smoke/e2e проверок;
- `Makefile` для единых команд разработки.

Технологии вроде Redis, WebSocket, message queues, OpenTelemetry, background workers и object storage откладываются до момента, когда появится реальная потребность.

### Примечание по реализации V1

Список выше фиксирует разрешенные технологии, а не обязательные зависимости.
Финальная localhost V1 использует более легкий набор там, где это снижает сложность без потери сценариев MVP:

- backend routing сделан на стандартном `net/http`, потому что текущий REST API небольшой и не требует `chi`;
- SQL написан напрямую поверх `pgx`, без `sqlc`, чтобы не добавлять генерацию кода до появления более широкой SQL-поверхности;
- миграции используют собственный легкий runner с `-- +goose Up/Down` markers, без отдельной зависимости от `goose` binary;
- frontend state и forms реализованы на React state + TypeScript validation helpers, без `TanStack Query`, `React Hook Form`, `Zod` и `Zustand`, потому что V1 пока помещается в простой локальный UI контур;
- стили написаны в обычном CSS с design tokens, без Tailwind build layer;
- browser e2e smoke покрыт Playwright.

Это осознанное упрощение V1. При расширении продукта эти инструменты можно добавить точечно, когда они начнут уменьшать сложность, а не увеличивать ее.

## Backend

- язык: Go;
- формат: modular monolith;
- API: REST JSON;
- router: стандартный `net/http` для V1, `chi` остается разрешенным при росте API;
- database driver: `pgx`;
- migrations: легкий внутренний runner с goose-compatible markers;
- SQL generation: handwritten SQL поверх `pgx` для V1, `sqlc` остается разрешенным при росте количества запросов;
- auth: server-side sessions в PostgreSQL с HttpOnly cookie;
- logging: structured logs;
- config: `.env` + typed config loader.

Почему так:

- modular monolith быстрее и безопаснее для старта, чем microservices;
- REST проще в отладке и быстрее поднимается, чем GraphQL;
- `sqlc` + PostgreSQL дают строгую типизацию без тяжелого ORM;
- session-based auth проще и надежнее для localhost MVP, чем сразу строить JWT ecosystem.

## Frontend

- React + TypeScript;
- bundler/dev server: Vite;
- routing: локальное section state для V1, React Router остается разрешенным, когда понадобится URL-level навигация;
- server state: explicit fetch helpers и React state для V1, TanStack Query остается разрешенным при росте сложности caching/invalidation;
- forms: controlled React forms с TypeScript validation helpers для V1;
- UI state: локальный React state, Zustand только если позже понадобится shared local state;
- styling: обычный CSS с project design tokens и admin-oriented UI system.

## Database

- PostgreSQL;
- одна база для приложения;
- отдельные миграции для schema evolution;
- seed script для локальной инициализации admin user и demo data.

## Infrastructure

- Docker Compose для `postgres`, `backend`, `frontend`;
- локальный reverse proxy в V1 не обязателен;
- `.env.example` для конфигурации;
- Makefile для типовых команд.

## 5. Repository Structure

Рекомендуемая структура монорепозитория:

```text
team-task-tracker/
  backend/
    cmd/api/
    internal/
      auth/
      config/
      db/
      http/
      projects/
      issues/
      comments/
      users/
      activity/
      common/
    migrations/
    sql/
    tests/
  frontend/
    src/
      app/
      pages/
      features/
      components/
      lib/
      hooks/
      types/
  deploy/
    docker/
  docs/
  .env.example
  docker-compose.yml
  Makefile
  README.md
```

## 6. Domain Model

Минимальные сущности V1:

### `workspaces`

- `id`
- `name`
- `created_at`

### `users`

- `id`
- `email`
- `username`
- `password_hash`
- `display_name`
- `is_active`
- `created_at`

### `workspace_members`

- `workspace_id`
- `user_id`
- `role` (`admin`, `member`)
- `joined_at`

### `projects`

- `id`
- `workspace_id`
- `key` например `CORE`
- `name`
- `description`
- `created_by`
- `created_at`
- `archived_at`

### `issues`

- `id`
- `project_id`
- `number` локальный номер внутри проекта
- `issue_key` например `CORE-12`
- `title`
- `description`
- `issue_type` (`task`, `bug`, `story`)
- `status` (`backlog`, `todo`, `in_progress`, `done`, `blocked`)
- `priority` (`low`, `medium`, `high`, `critical`)
- `reporter_id`
- `assignee_id`
- `due_date`
- `created_at`
- `updated_at`

### `labels`

- `id`
- `workspace_id`
- `name`
- `color`

### `issue_labels`

- `issue_id`
- `label_id`

### `comments`

- `id`
- `issue_id`
- `author_id`
- `body`
- `created_at`
- `updated_at`

### `sessions`

- `id`
- `user_id`
- `token_hash`
- `expires_at`
- `created_at`

### `activity_log`

- `id`
- `entity_type`
- `entity_id`
- `action`
- `actor_id`
- `payload`
- `created_at`

## 7. API Surface

V1 не требует идеального публичного API, но требует стабильного frontend contract.

### Auth

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`

### Users

- `GET /api/v1/users`
- `POST /api/v1/users`
- `PATCH /api/v1/users/:id`

### Projects

- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/:id`
- `PATCH /api/v1/projects/:id`

### Issues

- `GET /api/v1/issues`
- `POST /api/v1/issues`
- `GET /api/v1/issues/:id`
- `PATCH /api/v1/issues/:id`
- `POST /api/v1/issues/:id/transition`

### Comments

- `GET /api/v1/issues/:id/comments`
- `POST /api/v1/issues/:id/comments`
- `PATCH /api/v1/comments/:id`
- `DELETE /api/v1/comments/:id`

### Labels

- `GET /api/v1/labels`
- `POST /api/v1/labels`

## 8. Frontend Pages For V1

- login page;
- dashboard page;
- projects page;
- project detail page;
- issue list page with filters;
- kanban board page;
- issue detail drawer or modal;
- users/settings page для admin.

## 9. UX Principles

Нужно сразу держать правильную планку:

- быстрый create/edit flow без лишних экранов;
- максимум полезной информации в карточке задачи;
- board и list должны опираться на один и тот же backend source of truth;
- фильтры должны быть простыми и мгновенно понятными;
- UI должен быть строгим и рабочим, а не декоративным.

## 10. Development Phases

## Phase 0. Foundation

- инициализировать монорепозиторий;
- поднять `backend`, `frontend`, `postgres` через Docker Compose;
- завести Makefile;
- завести `.env.example`;
- описать README с локальным запуском.

Результат: проект запускается локально одной-двумя командами.

## Phase 1. Backend Skeleton

- создать Go service с health endpoint;
- подключить config loader;
- подключить PostgreSQL;
- завести миграции;
- подключить `sqlc`;
- разложить код по доменным модулям.

Результат: backend жив, ходит в БД, схема управляется миграциями.

## Phase 2. Auth And Users

- таблицы `users`, `workspaces`, `workspace_members`, `sessions`;
- login/logout/me;
- seed admin user;
- middleware аутентификации;
- middleware авторизации по ролям.

Результат: можно войти в систему и получить защищенный API.

## Phase 3. Projects

- CRUD проектов;
- project list;
- project detail shell;
- генерация `project key`.

Результат: есть рабочий контур управления проектами.

## Phase 4. Issues Core

- CRUD задач;
- статусы, приоритеты, типы;
- assignee, reporter, due date;
- генерация `issue_key`;
- activity log на создание и обновление.

Результат: система уже полезна как task tracker.

## Phase 5. Comments, Labels, Filters

- комментарии;
- labels;
- issue filtering и query params;
- list view.

Результат: задачами можно реально пользоваться внутри команды.

## Phase 6. Board UI

- kanban columns;
- drag-and-drop смена статуса;
- синхронизация board/list/detail view;
- optimistic updates там, где это оправдано.

Результат: появляется привычный Jira-like daily workflow.

## Phase 7. Hardening

- валидация на backend и frontend;
- error handling;
- access checks;
- базовые unit/integration tests;
- smoke e2e сценарий;
- demo seed data;
- cleanup API contract и UI polish.

Результат: V1 можно стабильно использовать на localhost.

## 11. Testing Strategy

Минимум, который нужен уже в первой версии:

- backend unit tests для domain/services;
- backend integration tests для repository + PostgreSQL;
- frontend component tests для ключевых форм;
- один e2e smoke test: login -> create project -> create issue -> move issue -> add comment.

Без тестов получится демо, а не продукт.

## 12. Definition Of Done For V1

Первая версия считается завершенной, когда:

1. проект поднимается локально через Docker;
2. есть admin login;
3. можно создать пользователя;
4. можно создать проект;
5. можно создать, отредактировать и закрыть задачу;
6. можно смотреть задачи списком и на доске;
7. можно писать комментарии;
8. работают базовые фильтры;
9. есть seed/demo data;
10. есть README с понятным локальным запуском.

## 13. Risks And Anti-Patterns

Чего нельзя делать на старте:

- не начинать с microservices;
- не добавлять Kafka, Redis и WebSocket без реальной необходимости;
- не проектировать "идеальную Jira schema" на десять версий вперед;
- не перегружать права доступа;
- не строить сначала сложный UI kit вместо самого продукта;
- не уходить в cloud/deploy до завершения localhost MVP.

## 14. Proposed Order Of Work

Практический порядок работ:

1. scaffold monorepo;
2. docker compose и база;
3. backend skeleton;
4. auth;
5. projects;
6. issues;
7. frontend auth shell;
8. issue list;
9. issue detail;
10. board;
11. comments/labels/filters;
12. tests, polish, docs.

## 15. Decision For Next Step

Следующий правильный шаг после этого плана:

сразу создать базовый scaffold проекта и инфраструктуру Phase 0:

- `backend/`
- `frontend/`
- `docker-compose.yml`
- `.env.example`
- `Makefile`
- начальный `README.md`

После этого можно перейти к первой практической реализации без повторного перепроектирования.
