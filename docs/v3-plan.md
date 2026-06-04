# Team Task Tracker V3 Plan

## 1. Product Goal

V1 закрыла крепкий localhost MVP: auth, workspace team, projects, issues, board/list, labels, comments, activity log, filters, seed data, Docker и базовый тестовый контур.

V2 превратила MVP в более сильный инструмент планирования: routing, hierarchy, links, sprints, story points, dashboard sprint summary, saved filters, in-app notifications, расширенный seed, API smoke, browser e2e и финальный QA.

Цель V3: подготовить проект к реальному self-hosted использованию небольшой командой, не ломая localhost-first основу и не превращая продукт в SaaS раньше времени.

Ключевой принцип V3: не добавлять новые Jira-like фичи ради масштаба, а сделать текущий функционал безопасным, поддерживаемым и готовым к запуску на собственном сервере.

V3 должна ответить на вопросы:

- как безопасно запустить проект не только на localhost;
- как настроить production environment без случайно небезопасных defaults;
- как обновлять, бэкапить и восстанавливать данные;
- как понимать, что приложение живо и работает корректно;
- как не сломать V1/V2 behavior при переходе к production-ready режиму.

## 2. V3 Boundary

### Что входит в V3

- production config validation;
- self-hosted Docker runtime для backend, frontend и PostgreSQL;
- production reverse proxy через Caddy или аналогичный легкий proxy;
- single-origin production app setup;
- TLS-ready deployment documentation;
- security hardening для cookies, CORS, headers и unsafe requests;
- CSRF protection для cookie-based auth;
- login rate limiting;
- session cleanup;
- invite-based onboarding без email;
- backup и restore workflow для PostgreSQL;
- production logging и request IDs;
- version/build info endpoint или metadata;
- pagination и performance indexes для тяжелых списков;
- GitHub Actions CI для базовых проверок;
- production smoke checks;
- README/deployment documentation;
- финальный V3 QA polish.

### Что сознательно не входит в V3

- managed cloud deployment под конкретного провайдера;
- SaaS billing;
- multi-tenant billing модель;
- mobile app;
- email notifications;
- real-time updates через WebSocket;
- Redis;
- background workers/message queues;
- file attachments/object storage;
- полноценный time tracking;
- automation rules;
- custom workflows per project;
- external integrations;
- advanced permission matrix шире текущих `admin/member`;
- переход на GraphQL;
- microservices.

V3 остается self-hosted шагом. Cloud-specific deploy, object storage, email, background jobs и automation должны планироваться отдельно только после того, как появится реальная потребность.

## 3. Main User Scenarios

V3 считается полезной, если хорошо закрывает такие сценарии:

1. Владелец проекта поднимает приложение на сервере через production Docker Compose.
2. Backend, frontend и PostgreSQL доступны через один публичный origin.
3. Production config не стартует с небезопасными значениями.
4. Пользователь логинится с secure cookie flow.
5. Admin приглашает нового участника через invite link без email-инфраструктуры.
6. Новый участник принимает invite, задает password и входит в workspace.
7. Admin делает backup PostgreSQL перед обновлением.
8. Admin восстанавливает backup в тестовую или локальную среду.
9. Разработчик видит request IDs и структурированные logs для диагностики.
10. GitHub Actions проверяет pull/push baseline до ручного запуска.
11. V1/V2 сценарии продолжают проходить после production hardening.

Если новая задача не помогает этим сценариям, ее нужно либо отложить, либо вынести в V4.

## 4. Architecture Direction

V3 не меняет базовую архитектуру:

- backend: Go modular monolith;
- API: REST JSON;
- database: PostgreSQL;
- auth: server-side sessions в PostgreSQL с HttpOnly cookie;
- frontend: React + TypeScript + Vite;
- local infrastructure: Docker Compose;
- production target: self-hosted Docker Compose.

### Что можно добавить в V3

- production-specific config fields;
- build-time version metadata;
- Caddy как reverse proxy для local production-like setup;
- отдельный production Dockerfile stage для backend;
- static frontend serving для production image;
- CSRF token mechanism;
- in-memory login rate limiter, достаточный для single-node self-hosted версии;
- scripts для backup/restore;
- GitHub Actions workflow;
- pagination helpers без breaking changes;
- JSON logs в production mode.

### Что не нужно делать в V3

- переписывать backend router;
- вводить `/api/v2`;
- менять auth model на JWT;
- добавлять Redis только ради rate limiting;
- делать Kubernetes manifests;
- добавлять Terraform;
- строить cloud-specific deploy;
- переписывать frontend state management;
- добавлять heavy observability stack.

API version остается `/api/v1`, потому что V3 не должна ломать существующий frontend contract.

## 5. Domain And Config Additions

V3 добавляет минимум новых domain сущностей и больше operational-конфигурации.

### Production config

Новые или усиленные config values:

- `APP_ENV`;
- `BACKEND_PORT`;
- `DATABASE_URL`;
- `PUBLIC_APP_URL`;
- `FRONTEND_URL` для development compatibility;
- `TRUSTED_ORIGINS`;
- `SESSION_TTL`;
- `SESSION_COOKIE_SECURE`;
- `CSRF_SECRET`;
- `RATE_LIMIT_LOGIN_PER_MINUTE`;
- `APP_VERSION`;
- `BUILD_COMMIT`;

Правила:

- `APP_ENV=production` должен fail-fast при localhost-only или пустых unsafe значениях;
- production cookies должны быть `HttpOnly`, `SameSite=Lax` и `Secure`;
- CORS должен разрешать только доверенные origins;
- secrets не должны попадать в logs;
- development defaults должны остаться удобными для localhost.

### `team_invites`

Новая таблица для invite-based onboarding:

- `id`;
- `workspace_id`;
- `email`;
- `role` (`admin`, `member`);
- `token_hash`;
- `created_by`;
- `created_at`;
- `expires_at`;
- `accepted_at`;
- `revoked_at`;

Правила:

- invite создает только admin;
- invite token хранится только как hash;
- invite можно принять один раз;
- expired/revoked invite нельзя принять;
- accept invite создает или активирует user в текущем workspace;
- username и password задаются при принятии invite;
- email delivery в V3 не добавляется, admin копирует invite link вручную.

### Session cleanup

V3 должна добавить безопасное удаление expired sessions:

- при login/logout/me можно opportunistic очищать старые sessions;
- отдельный background worker в V3 не нужен;
- cleanup не должен замедлять обычные API requests заметно для localhost/self-hosted сценария.

### Pagination metadata

Для тяжелых list endpoints добавляется backward-compatible pagination:

- `limit`;
- `cursor`;
- `next_cursor`;

Первыми кандидатами являются:

- `GET /api/v1/issues`;
- `GET /api/v1/notifications`;
- `GET /api/v1/issues/:id/activity`;

Если параметры не переданы, текущий behavior должен сохраниться настолько, насколько это нужно frontend и smoke/e2e тестам.

## 6. API Surface Additions

V3 добавляет endpoints без удаления V1/V2 endpoints.

### Runtime Metadata

- `GET /api/v1/version`

Ответ:

- `version`;
- `commit`;
- `environment`;
- `build_time` если доступен.

### CSRF

- `GET /api/v1/auth/csrf`

Правила:

- unsafe methods `POST`, `PATCH`, `PUT`, `DELETE` требуют `X-CSRF-Token`;
- token привязан к session или безопасному cookie flow;
- public unauthenticated endpoints должны быть явно проверены, чтобы не сломать login/invite accept.

### Team Invites

- `GET /api/v1/team/invites`;
- `POST /api/v1/team/invites`;
- `POST /api/v1/team/invites/:id/revoke`;
- `GET /api/v1/auth/invites/:token`;
- `POST /api/v1/auth/invites/:token/accept`.

### Backward-Compatible Pagination

Для выбранных list endpoints:

- query params: `limit`, `cursor`;
- response field: `next_cursor`;
- existing response arrays остаются на прежних ключах.

## 7. Frontend Screens For V3

V3 должна сохранить все V1/V2 экраны и добавить:

- admin invite management block в Team;
- invite accept page;
- production-safe auth error states;
- visible deployment/version metadata в Account или footer для admin/debug;
- backup/restore documentation links в README, не обязательно отдельный UI;
- clearer empty/error states для production-like failures.

V3 не должна превращать UI в админ-панель сервера. Operational actions вроде backup/restore остаются CLI/documentation flow.

## 8. UX Principles

V3 UX должен быть безопасным, но не тяжелым.

- Security не должна ломать быстрый localhost development flow.
- Production errors должны быть понятными без раскрытия внутренних details.
- Invite flow должен быть проще, чем ручное создание user + reset password.
- Admin-only controls должны оставаться явно отделены от member-readable flows.
- Backup/restore должны быть documented commands, а не скрытая магия.
- Production deployment docs должны быть короткими и повторяемыми.

## 9. Development Phases

## Phase 0. V3 Planning Baseline

- добавить официальный `docs/v3-plan.md`;
- убедиться, что V2 checks проходят;
- зафиксировать V3 как production-ready self-hosted scope;
- не менять V1/V2 behavior в planning commit.

Результат: есть официальный план V3, от которого дальше закрываются задачи.

## Phase 1. Production Config Foundation

- расширить config loader;
- добавить validation для production mode;
- добавить tests для safe/unsafe config combinations;
- обновить `.env.example` production-related comments;
- не менять Docker runtime в этом phase.

Результат: приложение явно различает development и production config.

## Phase 2. Production Docker Runtime

- добавить multi-stage backend production image;
- добавить frontend production build/static serving image;
- добавить `docker-compose.prod.yml`;
- сохранить текущий `docker-compose.yml` для development;
- проверить production compose config.

Результат: проект можно собрать как production containers без dev server внутри runtime.

## Phase 3. Reverse Proxy And Single-Origin Setup

- добавить Caddy config;
- проксировать API и frontend через один origin;
- подготовить TLS-ready локальный/серверный шаблон;
- описать ports и domains;
- проверить health/readiness через proxy.

Результат: production-like запуск не требует browser CORS между разными localhost ports.

## Phase 4. Cookie, CORS And Security Headers

- включить secure cookies в production;
- усилить CORS trusted origins;
- добавить security headers;
- добавить request size limits;
- покрыть tests.

Результат: базовый HTTP security hardening работает без ломки V1/V2 flows.

## Phase 5. CSRF Protection

- добавить CSRF token endpoint/mechanism;
- требовать `X-CSRF-Token` для unsafe authenticated requests;
- обновить frontend API client;
- обновить smoke/e2e helpers;
- покрыть auth/security tests.

Результат: cookie-based auth получает защиту от CSRF в production-ready контуре.

## Phase 6. Login Rate Limiting And Session Cleanup

- добавить in-memory rate limiter для login;
- добавить conservative defaults;
- добавить opportunistic expired session cleanup;
- покрыть tests;
- документировать ограничения single-node подхода.

Ограничение: login limiter в V3 является in-memory single-node защитой. Он сбрасывается при restart backend и не синхронизируется между несколькими backend instances; Redis/distributed limiter остается вне scope V3.

Результат: basic brute-force protection есть без Redis и без background worker.

## Phase 7. Invite-Based Onboarding Backend

- добавить миграцию `team_invites`;
- добавить backend invite CRUD/accept API;
- добавить token hashing;
- добавить validation и access checks;
- покрыть unit/integration tests.

Результат: admin может безопасно пригласить пользователя без email-инфраструктуры.

## Phase 8. Invite-Based Onboarding UI

- добавить invite management в Team для admin;
- добавить invite accept route/page;
- добавить form validation и clear error states;
- скрыть admin-only controls от members;
- добавить frontend tests/e2e scenario.

Результат: onboarding можно пройти end-to-end через UI.

## Phase 9. Backup And Restore Workflow

- добавить scripts для PostgreSQL backup;
- добавить scripts для restore в local/test database;
- добавить Makefile commands;
- документировать before-update flow;
- добавить smoke check для backup artifact и restore verification.

Результат: self-hosted admin имеет понятный путь защиты данных перед обновлениями.

## Phase 10. Observability And Runtime Metadata

- добавить request IDs;
- добавить JSON logs в production;
- добавить version/build endpoint;
- убедиться, что logs не содержат secrets;
- обновить README troubleshooting.

Результат: production инциденты можно диагностировать по request id, logs и version.

## Phase 11. Pagination And Performance Hardening

- добавить нужные indexes;
- добавить backward-compatible pagination для тяжелых list endpoints;
- обновить frontend там, где нужно;
- сохранить текущие default flows;
- покрыть tests.

Результат: приложение лучше ведет себя на больших локальных/self-hosted наборах данных.

## Phase 12. GitHub Actions CI

- добавить workflow для backend tests;
- добавить workflow для frontend tests/build;
- добавить shell syntax checks;
- добавить Docker Compose config validation;
- не требовать external services, которые сложно поднять в CI, если это замедляет старт.

Результат: baseline checks запускаются автоматически при push/PR.

## Phase 13. Production Smoke And E2E Coverage

- расширить smoke для production-like headers/config там, где возможно;
- добавить invite flow в e2e;
- проверить существующие V1/V2 browser smoke после CSRF/security changes;
- добавить docs для локального production QA.

Результат: production hardening покрыт тестами, а V1/V2 сценарии не сломаны.

## Phase 14. README Deployment Documentation

- обновить README статус до V3 in progress/completed по мере готовности;
- добавить self-hosted deployment flow;
- добавить update flow;
- добавить backup/restore flow;
- добавить troubleshooting;
- описать security config requirements.

Результат: новый пользователь может поднять production-like инстанс по документации.

## Phase 15. Final V3 QA Polish

- выполнить полный V1/V2/V3 automated baseline;
- выполнить manual localhost и production-like QA;
- исправить найденные blocker/polish defects;
- обновить README финальным V3 статусом;
- явно зафиксировать, что V3 завершена.

Результат: V3 можно считать production-ready self-hosted версией для небольшой команды.

## 10. Testing Strategy

Минимальный V3 testing baseline:

- все V1/V2 checks продолжают проходить;
- backend unit tests для config validation, cookies, CORS, CSRF, rate limiting, invites, pagination;
- backend integration tests для новых миграций и invite lifecycle;
- API smoke для production-sensitive flows;
- browser e2e:
  - admin creates invite;
  - invited user accepts invite;
  - invited user logs in;
  - V2 issue/sprint/saved filters/notifications flows still work after CSRF;
- production compose config validation;
- backup/restore smoke verification;
- GitHub Actions green baseline.

Перед финальным V3 закрытием нужно выполнить:

```sh
make setup-db
make smoke-api
make frontend-e2e
make verify
GOCACHE=/private/tmp/team-task-tracker-gocache make backend-integration-test
git diff --check
```

Дополнительно для V3:

```sh
make prod-config-check
make prod-compose-check
make backup
make restore-check
```

Названия новых команд могут быть уточнены при реализации, но смысл проверок должен сохраниться.

## 11. Definition Of Done For V3

V3 считается завершенной, когда:

1. V1 функционал не сломан.
2. V2 функционал не сломан.
3. Development localhost flow продолжает работать.
4. Production config validation предотвращает небезопасный запуск.
5. Production Docker images собираются без dev server runtime.
6. `docker-compose.prod.yml` поднимает self-hosted stack.
7. Reverse proxy дает single-origin access к frontend и API.
8. Production cookies, CORS и security headers работают.
9. CSRF protection включена для unsafe authenticated requests.
10. Login rate limiting работает.
11. Expired sessions очищаются безопасно.
12. Invite onboarding работает end-to-end.
13. Backup и restore workflow documented и проверен.
14. Production logs содержат request IDs и version/build metadata.
15. Тяжелые list endpoints имеют pagination/performance hardening.
16. GitHub Actions CI проходит.
17. README описывает self-hosted deployment, update, backup/restore и troubleshooting.
18. `make verify`, API smoke, backend integration tests и browser e2e проходят.
19. Production-like smoke/QA проходит.
20. После финального QA нет известных V3 blocker bugs.

## 12. Risks And Anti-Patterns

Основные риски V3:

- начать cloud-specific deploy до готовности generic self-hosted flow;
- добавить Redis/Kubernetes/Terraform раньше необходимости;
- сломать localhost development ради production hardening;
- сделать CSRF/security слишком сложными и хрупкими;
- не проверить backup restore реально;
- смешать production readiness с большими product features;
- оставить README неполным и сделать deploy зависимым от памяти разработчика.

Как защищаемся:

- работаем маленькими commits;
- каждый security шаг сопровождается тестами;
- development и production configs разделяются явно;
- Caddy/production compose добавляются отдельно от dev compose;
- new product features не добавляются в V3 без отдельного решения;
- backup считается готовым только после restore check;
- финальный V3 QA должен проходить и localhost, и production-like сценарии.

## 13. Proposed Commit Order

Ожидаемый размер V3: примерно 16-20 небольших коммитов.

Практический порядок:

1. add V3 plan;
2. add production config validation;
3. add production Docker images;
4. add production compose and Caddy proxy;
5. harden cookies CORS and security headers;
6. add CSRF protection;
7. add login rate limiting and session cleanup;
8. add invite schema and backend API;
9. add invite management UI;
10. add backup and restore scripts;
11. add production logging and request IDs;
12. add runtime version metadata;
13. add pagination and performance indexes;
14. add GitHub Actions CI;
15. extend smoke and e2e for V3;
16. update README for V3 deployment flow;
17. final V3 QA polish.

Количество коммитов может измениться, но порядок должен оставаться примерно таким: сначала planning, затем production foundation, затем security, затем onboarding/ops, затем performance, CI, docs и final QA.

## 14. Decision For Next Step

Следующий практический шаг после этого плана:

начать Phase 1 с production config validation.

Причина: до Docker production runtime, Caddy, CSRF и deployment docs нужно сначала четко определить, какие settings считаются безопасными в development и production. Если начать с Docker или proxy раньше config validation, можно получить production контур, который технически запускается, но допускает небезопасные defaults.
