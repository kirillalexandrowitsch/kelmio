# Team Task Tracker V5 Plan

## 1. Product Goal

V1-V4 закрыли task tracking, planning, production-ready self-hosted foundation,
project workflows, project permissions и synchronous automation.

Цель V5: добавить account recovery и локально воспроизводимую operations
foundation для долгосрочного Jira family functional-parity roadmap.

V5 должна дать пользователям и администраторам:

- безопасное восстановление пароля через email token;
- автоматическую доставку invite email и повторную отправку приглашения;
- durable delivery системных писем без потери при временной SMTP-ошибке;
- понятную диагностику email delivery;
- application metrics и локально проверяемые alerts;
- автоматические backups с retention;
- регулярный автоматический restore drill;
- полный localhost QA всех operations flows.

Ключевой принцип V5: все новые operations capabilities должны быть
provider-neutral и полностью проверяться на localhost. Реальный hosting,
public deployment, домен и production pilot не входят в целевую модель
проекта.

V5 закрывает capabilities `PLAT-004` и `PLAT-005` из
[Jira family capability baseline](jira-family-capability-baseline.md).

## 2. V5 Boundary

### Что входит в V5

- generic SMTP configuration;
- Mailpit в development Docker Compose;
- durable PostgreSQL email outbox;
- отдельный email delivery worker;
- retry policy и terminal failure state для email;
- password reset request, preview и complete flow;
- password reset UI;
- invite email delivery и resend API/UI;
- admin-readable email delivery diagnostics;
- Prometheus application metrics;
- локальные Prometheus, Grafana и Alertmanager;
- localhost alerts для API readiness, email failures и backup freshness;
- scheduled backup runner;
- backup retention;
- автоматический isolated restore drill;
- V5 seed/test fixtures;
- unit, component, integration, smoke и browser e2e coverage;
- operations documentation;
- финальный V1-V5 QA polish.

### Что сознательно не входит в V5

- реальный hosting provider, domain или public deployment;
- hosting-specific configuration и production pilot;
- общие email notifications для issue/comment/sprint событий;
- user-configurable notification preferences;
- custom fields;
- advanced search;
- reports и roadmap;
- file attachments;
- external integrations;
- real-time updates;
- SaaS billing;
- multi-workspace UI.

## 3. Main User Scenarios

V5 считается полезной, если хорошо закрывает такие сценарии:

1. Пользователь запрашивает password reset, не раскрывая существование email.
2. Пользователь открывает действительный reset link и устанавливает новый
   пароль.
3. Использованный, просроченный или отозванный reset token нельзя применить.
4. Admin создает invite, а письмо надежно попадает в локальный Mailpit.
5. Admin повторно отправляет pending invite без создания дубликата invite.
6. Временная SMTP-ошибка не теряет письмо: worker повторяет доставку.
7. Terminal email failure виден admin без раскрытия secret или полного token.
8. Метрики показывают состояние HTTP API, outbox, worker и backup workflow.
9. Alertmanager получает alert при искусственно созданной failure-ситуации.
10. Scheduled backup создается, старые artifacts удаляются по retention policy.
11. Restore drill восстанавливает последний backup в isolated PostgreSQL и
    проверяет core data.
12. Все V1-V4 workflows продолжают работать после V5 изменений.

Если новая задача не помогает этим сценариям, ее нужно отложить до следующей
версии.

## 4. Architecture Direction

V5 сохраняет текущую архитектуру:

- backend: Go modular monolith;
- API: REST JSON `/api/v1`;
- database: PostgreSQL;
- auth: server-side sessions;
- frontend: React + TypeScript + Vite;
- local runtime: Docker Compose;
- existing production-like QA runtime: Docker Compose и Caddy.

### Что можно добавить в V5

- backend modules для email outbox, mail delivery, password reset и metrics;
- отдельный Go worker binary, использующий ту же database и config foundation;
- PostgreSQL migrations, indexes и row-locking для reliable outbox processing;
- Mailpit, Prometheus, Grafana и Alertmanager только в localhost/operations
  profiles;
- scripts/commands для scheduled backup и restore drill;
- provider-neutral SMTP interface;
- additive API endpoints и frontend screens.

### Что не нужно делать в V5

- выбирать hosting provider или строить provider-specific deployment;
- добавлять Redis или message broker;
- отправлять email непосредственно из HTTP handler;
- хранить raw password reset или invite tokens в database;
- делать email delivery частью пользовательской transaction после durable
  outbox insert;
- создавать generic background-job framework для будущих неизвестных задач;
- добавлять email notifications ко всем existing notification types;
- менять завершенные V1-V4 product boundaries.

## 5. Domain And Configuration Additions

### SMTP configuration

Добавить provider-neutral configuration:

- `SMTP_HOST`;
- `SMTP_PORT`;
- `SMTP_USERNAME`;
- `SMTP_PASSWORD`;
- `SMTP_FROM_EMAIL`;
- `SMTP_FROM_NAME`;
- `SMTP_TLS_MODE`;
- `EMAIL_DELIVERY_ENABLED`;
- `EMAIL_WORKER_POLL_INTERVAL`;
- `EMAIL_MAX_ATTEMPTS`;
- `PASSWORD_RESET_TTL`.

Правила:

- development defaults направляют письма в Mailpit;
- production config требует валидный sender и безопасный TLS mode, если email
  delivery включена;
- secrets никогда не выводятся в logs, metrics или API;
- HTTP API создает outbox records независимо от текущей доступности SMTP;
- worker можно безопасно перезапускать без потери или двойной доставки.

### `email_outbox`

Новая таблица хранит durable email delivery:

- `id`;
- `workspace_id`, nullable для pre-auth системных писем;
- `email_type`;
- `recipient_email`;
- `template_data jsonb`;
- `status`;
- `attempt_count`;
- `next_attempt_at`;
- `last_error`;
- `sent_at`;
- timestamps;
- deterministic deduplication key, где это требуется.

Статусы:

- `pending`;
- `processing`;
- `sent`;
- `failed`.

Правила:

- worker получает записи через transaction и `FOR UPDATE SKIP LOCKED`;
- retry использует ограниченный exponential backoff;
- зависшая `processing` запись может быть безопасно возвращена в очередь;
- terminal failure не удаляется автоматически;
- `last_error` очищается от credentials, tokens и unsafe provider response data;
- email body не хранится, если достаточно template data.

### `password_reset_tokens`

Новая таблица хранит password reset requests:

- `id`;
- `user_id`;
- `token_hash`;
- `created_at`;
- `expires_at`;
- `used_at`;
- `revoked_at`;
- request metadata без sensitive headers.

Правила:

- database хранит только SHA-256 hash случайного token;
- raw token появляется только в reset URL внутри outbox template data;
- новый request отзывает предыдущие active tokens пользователя;
- request endpoint всегда возвращает одинаковый публичный response;
- complete atomарно обновляет password hash, отмечает token использованным и
  удаляет все active sessions пользователя;
- token нельзя использовать повторно;
- reset request имеет отдельный rate limit.

### Operations state

Backup freshness и restore drill result должны быть доступны как machine-readable
artifacts/metrics без новой business schema, если durable database state для
этого не требуется.

## 6. API Additions

### Password reset

`POST /api/v1/auth/password-reset/request`

Request:

```json
{
  "email": "member@example.com"
}
```

Behavior:

- public pre-login endpoint;
- всегда возвращает `202 Accepted` с одинаковым message;
- active matching user получает новый token и outbox email;
- unknown/inactive email не раскрывается;
- endpoint exempt от CSRF и защищен отдельным rate limit.

`GET /api/v1/auth/password-reset/{token}`

Behavior:

- public safe endpoint;
- возвращает минимальный preview для valid token;
- unknown token получает `404 password_reset_not_found`;
- expired, used и revoked token получают понятные stable error codes.

`POST /api/v1/auth/password-reset/{token}/complete`

Request:

```json
{
  "password": "new-password",
  "confirm_password": "new-password"
}
```

Behavior:

- public pre-login endpoint и CSRF exempt;
- использует существующие password validation rules;
- invalid token state не изменяет пользователя;
- success возвращает `204 No Content`;
- после success пользователь входит обычным login flow.

### Invite resend

`POST /api/v1/team/invites/{id}/resend`

Behavior:

- доступен workspace admin;
- только pending, non-expired и non-revoked invite можно отправить повторно;
- создает новый durable outbox record;
- не создает новый invite и не меняет accept token;
- применяет cooldown/rate limit;
- response возвращает invite metadata без raw token.

### Metrics

`GET /metrics`

Behavior:

- Prometheus text exposition;
- endpoint выключен или защищен shared bearer token вне development;
- не содержит email addresses, usernames, issue titles, tokens или других
  high-cardinality/sensitive labels;
- не входит в public product API `/api/v1`.

Минимальные metrics:

- HTTP request count/duration/status;
- database readiness;
- auth login outcomes без identifier labels;
- email outbox pending/failed counts;
- email delivery attempts/results;
- email worker heartbeat;
- backup age/result;
- restore drill result/duration.

## 7. Frontend Screens And Behavior

### Sign-in и password reset

- Sign-in получает ссылку `Forgot password?`.
- `/forgot-password` содержит email form и одинаковый success state независимо
  от существования пользователя.
- `/reset-password?token=...` загружает preview и показывает password/confirm
  form.
- Invalid, expired, used и revoked token имеют понятные error states.
- После успешного reset пользователь получает переход к sign-in.

### Team invite management

- Pending invite получает действие `Resend email`.
- UI показывает pending/sent/failed delivery state, если backend предоставляет
  admin-safe summary.
- Existing copy-link и revoke flows сохраняются.
- Failed resend не изменяет invite state.

### Email diagnostics

- Admin видит компактный diagnostics block в Team или Account admin section.
- Показываются counts `pending`, `sent`, `failed` и последние terminal failures.
- Recipient email можно маскировать.
- Retry terminal failure разрешается только admin и только если это будет
  добавлено отдельным явно протестированным endpoint; иначе remediation остается
  CLI/worker flow.

Operations dashboards остаются в Grafana, а не дублируются внутри product UI.

## 8. UX Principles

- Password reset не раскрывает существование пользователя.
- System email errors не показывают secrets или raw provider responses.
- Пользовательские actions не ждут SMTP delivery.
- Admin получает конкретное состояние доставки и понятный remediation path.
- Operations tooling не перегружает обычный product UI.
- Localhost setup остается простым: базовый `docker compose up` продолжает
  работать, а monitoring может быть отдельным profile.
- Existing auth, invite и team flows сохраняют текущий визуальный язык.

## 9. Development Phases

### Phase 1: SMTP config и Mailpit

- добавить typed SMTP config и validation;
- добавить Mailpit в development Compose;
- добавить SMTP client abstraction и локальный delivery test;
- документировать Mailpit UI и disabled-delivery mode.

Результат: localhost может безопасно отправлять и просматривать тестовые письма.

### Phase 2: Durable email outbox и worker

- добавить outbox migration и indexes;
- реализовать atomic enqueue;
- добавить worker binary, locking, retry/backoff и graceful shutdown;
- добавить worker health/heartbeat logging.

Результат: системные письма не теряются при временных ошибках и рестартах.

### Phase 3: Password reset backend

- добавить token schema и service;
- добавить request/preview/complete endpoints;
- добавить reset-specific rate limit;
- отзывать sessions после успешной смены password.

Результат: пользователь может безопасно восстановить доступ.

### Phase 4: Password reset UI

- добавить public routes и forms;
- добавить token-state и validation errors;
- покрыть component и browser flows.

Результат: password reset полностью доступен без ручных API-запросов.

### Phase 5: Invite email delivery

- enqueue invite email при create;
- добавить admin resend flow;
- сохранить copy-link fallback;
- добавить cooldown и delivery tests.

Результат: admin может доставлять invite через email и повторять отправку.

### Phase 6: Email diagnostics

- добавить admin-safe delivery summaries;
- добавить worker/outbox troubleshooting commands;
- добавить readable logs и failure handling.

Результат: email delivery можно диагностировать без прямого изменения database.

### Phase 7: Application metrics

- добавить Prometheus registry и middleware;
- покрыть HTTP, database, auth, outbox и worker signals;
- защитить metrics endpoint;
- проверить отсутствие sensitive/high-cardinality labels.

Результат: приложение отдает стабильные machine-readable operational metrics.

### Phase 8: Local monitoring и alerting

- добавить localhost operations profile с Prometheus, Grafana и Alertmanager;
- добавить versioned dashboards и alert rules;
- проверить API, email и backup failure alerts.

Результат: monitoring и alerts воспроизводятся локально.

### Phase 9: Scheduled backups и retention

- добавить reusable backup runner;
- добавить schedule для localhost operations profile;
- добавить configurable retention;
- не удалять последний successful backup.

Результат: backups создаются и обслуживаются автоматически.

### Phase 10: Restore drill

- автоматически выбирать последний backup;
- восстанавливать его в isolated PostgreSQL;
- проверять migrations/core data и cleanup;
- публиковать machine-readable result и metrics.

Результат: backup считается успешным только вместе с проверяемым restore path.

### Phase 11: V5 test expansion

- расширить API smoke;
- добавить operations smoke;
- добавить password reset/invite email Playwright flows;
- добавить component/integration tests;
- включить V5 checks в Full QA.

Результат: V5 regressions воспроизводимо обнаруживаются локально и в CI.

### Phase 12: Operations documentation и final QA

- добавить email и localhost operations guides;
- выполнить полный V1-V5 QA;
- исправить только blocker/polish defects;
- зафиксировать итог V5.

Результат: V5 полностью завершена и готова стать основой V6.

## 10. Testing Strategy

### Backend unit tests

- SMTP config validation;
- email template rendering;
- outbox normalization и retry/backoff;
- token generation/hash/state;
- password validation и request privacy;
- metrics label safety;
- backup retention rules.

### Backend integration tests

- outbox enqueue и concurrent worker locking;
- retry, terminal failure и stuck-processing recovery;
- password reset lifecycle и session revocation;
- invite create/resend delivery;
- transaction rollback behavior;
- metrics values после controlled actions;
- two consecutive migrations/setup runs.

### Frontend tests

- forgot/reset password validation и states;
- invite resend behavior;
- admin/member permission behavior;
- email diagnostics presentation;
- regression component tests V1-V4.

### Smoke и browser e2e

- request reset, получить link из Mailpit API, завершить reset и войти;
- создать invite, проверить email, accept и sign-in;
- временно остановить Mailpit, проверить retry, затем восстановить delivery;
- проверить metrics endpoint и sensitive-label absence;
- создать scheduled backup и выполнить restore drill;
- полный V1-V5 business regression.

### Full QA

- два последовательных database setup;
- unit, component, integration, race и vet checks;
- API, production-sensitive и operations smoke;
- Playwright e2e;
- backup/restore/retention drill;
- Docker Compose config и clean-room production-stack regression;
- dependency audit;
- `git diff --check`.

## 11. Definition Of Done

V5 полностью завершена, когда:

1. Development Compose предоставляет рабочий Mailpit flow.
2. Системные письма доставляются только через durable outbox и worker.
3. Worker безопасно обрабатывает retries, restarts и concurrent execution.
4. Password reset не раскрывает существование email и отзывает sessions.
5. Invite create/resend доставляет email и сохраняет copy-link fallback.
6. Admin может диагностировать terminal email failures без sensitive data.
7. Prometheus metrics покрывают API, database, email и backup operations.
8. Local Grafana dashboards и Alertmanager rules воспроизводимо работают.
9. Scheduled backups соблюдают retention и сохраняют последний successful
   backup.
10. Restore drill автоматически подтверждает пригодность backup.
11. Все V1-V4 regressions остаются зелеными.
12. Полный V1-V5 QA и GitHub Full QA успешны.
13. V5 operations docs соответствуют фактическим commands.
14. Нет известных V5 blocker bugs.

## 12. Risks

- отправлять SMTP внутри HTTP request и ухудшить надежность/latency;
- создать слишком общий background-job framework;
- допустить двойную доставку при restart/concurrency;
- раскрыть существование пользователя через password reset;
- записать raw tokens, SMTP credentials или recipient data в logs/metrics;
- использовать high-cardinality metric labels;
- считать backup успешным без restore verification;
- retention ошибочно удаляет последний usable backup;
- monitoring profile усложняет базовый localhost flow;
- начать hosting-specific или public deployment работу;
- расширить V5 до общих email notifications и потерять управляемый scope.

## 13. Proposed Commit Order

1. `Add SMTP config and Mailpit development stack`
2. `Add durable email outbox and worker`
3. `Add password reset backend API`
4. `Add password reset UI`
5. `Add invite email delivery and resend flow`
6. `Add email delivery diagnostics`
7. `Add Prometheus application metrics`
8. `Add local monitoring and alerting stack`
9. `Add scheduled backup runner and retention`
10. `Add operations restore drill and smoke tests`
11. `Extend V5 component integration and e2e tests`
12. `Finalize V5 QA polish`

Каждый commit должен быть небольшим, проходить релевантные tests и сохранять
работоспособность завершенных V1-V4 flows.

## 14. Decision For Next Step

Следующий шаг: реализовать commit `Finalize V5 QA polish`.

После V5 проект продолжает разрабатываться и проверяться на localhost по
roadmap V6-V24. Hosting provider, реальный deployment и production pilot не
входят в критерии закрытия проекта.
