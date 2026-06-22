# Kelmio V4 Plan

## 1. Product Goal

V1 закрыла стабильный task-tracking baseline: auth, team, projects, issues,
list/board, labels, comments, activity, filters и Docker-based localhost flow.

V2 добавила planning workflow: hierarchy, links, sprints, story points, saved
filters, notifications и расширенный QA.

V3 подготовила приложение к production-ready self-hosted использованию:
security hardening, invites, backup/restore, observability, pagination,
production Compose, CI и deployment documentation.

Post-V3 cleanup укрепил frontend/backend структуру, component tests и полный QA.

Статус на 15 июня 2026 года: V4 полностью завершена. Все плановые фазы,
Definition of Done и финальный V1-V4 QA закрыты.

Цель V4: сделать workflow каждого проекта гибким и управляемым, сохранив простой
self-hosted runtime и совместимость завершенных V1-V3 сценариев.

V4 должна дать небольшой команде:

- собственные статусы и переходы для каждого проекта;
- понятные project-level роли и границы доступа;
- project-specific board, отражающий реальный workflow;
- простые синхронные automation rules без workers и очередей;
- безопасную миграцию существующих проектов, задач, filters и sprint metrics.

Ключевой принцип V4: добавить управляемую гибкость, но не превращать продукт в
enterprise workflow engine.

## 2. V4 Boundary

### Что входит в V4

- project-level workflow statuses;
- настраиваемый граф разрешенных переходов;
- status name, immutable key, color, category и position;
- безопасное архивирование status с обязательным replacement;
- project roles `lead`, `contributor`, `viewer`;
- project membership и access enforcement;
- project-specific Kanban board;
- active sprint board на workflow выбранного проекта;
- dynamic status controls и filters;
- синхронные issue automation rules;
- automation activity visibility;
- V4 seed data, component tests, integration tests, smoke и browser e2e;
- обновленные README и V4 QA documentation;
- финальный V4 QA polish.

### Что сознательно не входит в V4

- multi-workspace UI или переключение между workspace;
- Redis;
- background workers и message queues;
- cascading automation rules;
- scheduled/time-based automation;
- file attachments и object storage;
- email notifications;
- real-time updates через WebSocket;
- полноценный time tracking;
- external integrations;
- custom issue fields;
- детальная permission matrix на каждое отдельное действие;
- workflow templates, общие между несколькими проектами;
- SaaS billing и managed cloud platform;
- новый `/api/v2`.

V4 остается single-workspace self-hosted этапом. Новые runtime-компоненты не
добавляются.

## 3. Main User Scenarios

V4 считается полезной, если хорошо закрывает такие сценарии:

1. Admin создает проект и получает стандартный workflow.
2. Project lead переименовывает статусы, меняет цвета и порядок.
3. Project lead добавляет custom status и настраивает разрешенные переходы.
4. Contributor создает задачу и переводит ее только по разрешенным переходам.
5. Project board показывает реальные статусы выбранного проекта.
6. Active sprint board использует тот же project workflow.
7. Project lead назначает contributors и viewers.
8. Viewer открывает project/issues/sprints, но не изменяет данные.
9. Участник без project membership не видит project data.
10. Lead архивирует используемый status, выбирает replacement, и задачи
    переносятся атомарно.
11. Lead создает automation rule для issue trigger, conditions и actions.
12. Подходящая automation rule выполняется атомарно вместе с пользовательским
    изменением и отображается в activity.
13. Старые V1-V3 задачи, saved filters, boards и sprint metrics продолжают
    работать после миграции.

Если новая задача не помогает этим сценариям, ее нужно отложить до следующей
версии.

## 4. Architecture Direction

V4 сохраняет текущую архитектуру:

- backend: Go modular monolith;
- API: REST JSON `/api/v1`;
- database: PostgreSQL;
- auth: server-side sessions;
- frontend: React + TypeScript + Vite;
- local и production runtime: Docker Compose;
- production proxy: Caddy.

### Что можно добавить в V4

- новые backend modules для workflow, project membership и automation;
- PostgreSQL migrations и indexes;
- shared permission helper для project-scoped access;
- workflow-aware issue queries и response models;
- synchronous automation evaluator внутри issue transactions;
- новые project settings screens и typed frontend controllers;
- additive API fields и endpoints.

### Что не нужно делать в V4

- переписывать modular monolith;
- менять auth model;
- добавлять event bus;
- выполнять automation асинхронно;
- разрешать automation actions запускать другие rules;
- строить generic expression language;
- вводить полностью custom roles;
- заменять существующий frontend state management;
- ломать существующие V1-V3 API fields без необходимости.

Workflow, permissions и automation должны оставаться отдельными concerns, даже
если они вызываются из одного issue transaction.

## 5. Domain Model Additions

### `project_workflow_statuses`

Новая таблица хранит статусы конкретного проекта:

- `id`;
- `project_id`;
- `key`;
- `name`;
- `color`;
- `category` (`backlog`, `todo`, `in_progress`, `done`);
- `position`;
- `created_at`;
- `updated_at`;
- `archived_at`.

Правила:

- `key` immutable после создания;
- `key` уникален внутри проекта;
- key использует lowercase letters, numbers и underscore;
- name обязателен и уникален среди active statuses проекта;
- color хранится как валидный hex color;
- position определяет порядок board и controls;
- active project должен иметь минимум один status category `done`;
- default workflow: `backlog`, `todo`, `in_progress`, `blocked`, `done`;
- default `blocked` относится к category `in_progress`;
- archived status нельзя использовать для новых задач и переходов.

### `project_workflow_transitions`

Новая таблица хранит разрешенные переходы:

- `project_id`;
- `from_status_id`;
- `to_status_id`;
- `created_at`.

Правила:

- source и target относятся к одному проекту;
- self-transition не хранится;
- duplicate transition запрещен;
- default workflow получает логичный граф переходов;
- transition API строго проверяет граф;
- workspace admin может выполнить любой переход только через те же API и не
  обходит workflow graph.

### `project_members`

Новая таблица хранит project-scoped доступ:

- `project_id`;
- `user_id`;
- `role` (`lead`, `contributor`, `viewer`);
- `created_at`;
- `updated_at`.

Правила:

- workspace admin всегда имеет полный project access независимо от записи;
- active workspace member может быть project member;
- каждый active project должен иметь минимум одного active lead или workspace
  admin;
- lead управляет project members, workflow и automation;
- contributor создает и изменяет issues/comments/sprints;
- viewer имеет только read access;
- участник без project membership не видит project-scoped данные.

Migration/backfill:

- все active workspace members становятся contributors существующих проектов;
- creator каждого существующего проекта становится lead;
- новый project автоматически добавляет creator как lead;
- все остальные active workspace members нового project добавляются contributors.

### `issues`

Добавить `workflow_status_id`, связанный с active или historical status проекта.

Backward compatibility:

- существующее response field `status` остается стабильным status key;
- добавить `workflow_status` object:
  - `id`;
  - `key`;
  - `name`;
  - `color`;
  - `category`;
- существующие statuses мигрируют в project workflow без потери key;
- старый database `status` column удаляется только после успешного backfill и
  полного перехода queries на `workflow_status_id`;
- completion/open logic использует category, а не конкретный key.

### `automation_rules`

Новая таблица хранит project-scoped rules:

- `id`;
- `project_id`;
- `name`;
- `trigger_type`;
- `conditions jsonb`;
- `actions jsonb`;
- `position`;
- `is_enabled`;
- `created_by`;
- `created_at`;
- `updated_at`.

Поддерживаемые triggers:

- `issue_created`;
- `status_changed`;
- `assignee_changed`;
- `priority_changed`.

Поддерживаемые conditions:

- issue type;
- workflow status;
- priority;
- assignee;
- reporter;
- label.

Поддерживаемые actions:

- change workflow status;
- change assignee;
- change priority;
- add label;
- remove label.

Execution semantics:

- rules выбираются по project и trigger;
- matching проверяется по snapshot после пользовательского изменения;
- enabled rules выполняются по position;
- более поздняя action имеет приоритет при изменении одного поля;
- rules выполняются один раз;
- automation actions не запускают новые rules;
- исходное изменение, automation actions и activity записи выполняются в одной
  transaction;
- ошибка action откатывает всю transaction;
- rule нельзя сохранить с invalid dependency;
- если dependency позже архивирована или стала недоступна, rule отключается и
  показывает причину.

## 6. API Surface Additions

API version остается `/api/v1`.

### Project Workflows

- `GET /api/v1/projects/{id}/workflow`
- `POST /api/v1/projects/{id}/workflow/statuses`
- `PATCH /api/v1/projects/{id}/workflow/statuses/{statusID}`
- `PUT /api/v1/projects/{id}/workflow/statuses/order`
- `POST /api/v1/projects/{id}/workflow/statuses/{statusID}/archive`
- `PUT /api/v1/projects/{id}/workflow/transitions`

Status archive request требует `replacement_status_id`. Backend атомарно
переносит issues, удаляет связанные transitions, архивирует status и записывает
activity.

Transitions обновляются одним atomic request с полным набором разрешенных пар.

### Project Members

- `GET /api/v1/projects/{id}/members`
- `PUT /api/v1/projects/{id}/members/{userID}`
- `DELETE /api/v1/projects/{id}/members/{userID}`

Workspace admin и project lead управляют membership. Нельзя удалить последнего
доступного lead, если нет workspace admin.

### Issues And Filters

Additive changes:

- issue responses получают `workflow_status`;
- create/subtask/transition принимают `workflow_status_id`;
- legacy `status` key остается fallback для совместимости;
- если переданы оба значения, `workflow_status_id` имеет приоритет;
- key должен существовать в workflow проекта;
- `GET /api/v1/issues` поддерживает `workflow_status_id`;
- legacy `status` filter продолжает фильтровать по status key;
- saved filters поддерживают additive `workflowStatusId`;
- missing/archived workflow status отображается как missing value и может быть
  очищен.

Forbidden transition возвращает `409 transition_not_allowed`.

### Automation Rules

- `GET /api/v1/projects/{id}/automation-rules`
- `POST /api/v1/projects/{id}/automation-rules`
- `PATCH /api/v1/projects/{id}/automation-rules/{ruleID}`
- `DELETE /api/v1/projects/{id}/automation-rules/{ruleID}`
- `PUT /api/v1/projects/{id}/automation-rules/order`

Rules API доступен workspace admin и project lead.

### Project Access Enforcement

Project membership применяется ко всем project-scoped endpoints:

- projects;
- issues, hierarchy, links, comments и activity;
- sprints;
- workflow;
- project members;
- automation rules.

Cross-project issue links разрешены только если пользователь имеет доступ к обоим
проектам. Workspace-level team, labels, account, notifications и invite API
сохраняют текущую модель доступа.

## 7. Frontend Screens For V4

### Project Details

Расширить project detail:

- summary;
- members;
- workflow;
- automation.

Tabs `Members`, `Workflow`, `Automation` доступны workspace admin и project lead.
Viewer видит только read-only project summary.

### Project Members

- список active project members и roles;
- add workspace member;
- change role;
- remove member;
- понятные permission notes;
- защита последнего lead.

### Workflow Settings

- ordered status list;
- create/edit status;
- color и category controls;
- drag/reorder statuses;
- transition matrix или компактный transition editor;
- archive status modal с обязательным replacement;
- warning о влиянии на issues, saved filters и rules.

### Project Board

- Board требует выбранный project;
- route хранит selected project;
- columns строятся из active workflow statuses;
- cards переходят только по разрешенным transitions;
- недоступные drop targets визуально disabled;
- archived historical status не создает новую board column.

### Issues And Sprints

- create/detail/filter controls используют statuses выбранного project;
- cross-project issue list показывает status name/color;
- active sprint board использует workflow sprint project;
- sprint progress считает category `done`;
- legacy status keys не показываются пользователю как основной label.

### Automation Settings

- список rules с enabled state и position;
- create/edit rule form;
- trigger selector;
- typed conditions/actions builder;
- dependency validation;
- disabled reason;
- clear single-pass atomic execution note.

## 8. UX Principles

- custom workflow не должен усложнять обычную работу contributor;
- board всегда показывает workflow одного project;
- category остается технической основой metrics, но name/color определяют UI;
- destructive status archive всегда требует replacement и confirmation;
- forbidden actions скрываются или disabled с понятным объяснением;
- viewer UI явно read-only;
- automation builder поддерживает только известные typed conditions/actions;
- automation не должна выглядеть асинхронной: результат виден сразу после request;
- activity показывает, какие изменения сделал пользователь, а какие automation;
- migration не должна менять знакомые названия/default workflow существующих
  проектов.

## 9. Development Phases

## Phase 0. V4 Planning Baseline

- добавить `docs/v4-plan.md`;
- зафиксировать decisions, scope и commit order;
- не менять runtime behavior.

Результат: V4 начинается от decision-complete плана.

## Phase 1. Project Workflow Schema And Backfill

- добавить workflow statuses и transitions;
- добавить `issues.workflow_status_id`;
- создать default workflow для каждого project;
- backfill существующих issue statuses;
- добавить migration integration tests.

Результат: существующие данные безопасно используют project workflows.

## Phase 2. Workflow Backend API

- добавить workflow read/status CRUD/order/archive API;
- добавить atomic transition graph update;
- покрыть validation, replacement и transaction behavior.

Результат: workflow полностью управляется через backend.

## Phase 3. Project Membership Foundation

- добавить `project_members`;
- backfill leads/contributors;
- добавить project member API;
- добавить shared project permission checks.

Результат: project-level доступ имеет стабильную backend основу.

## Phase 4. Project Permission Enforcement

- применить access checks ко всем project-scoped endpoints;
- проверить links, sprints, comments и notifications;
- сохранить workspace admin full access.

Результат: lead/contributor/viewer правила реально защищаются backend.

## Phase 5. Project Member Management UI

- добавить members tab;
- добавить role management и read-only states;
- добавить component/e2e coverage.

Результат: lead/admin управляет доступом без прямых API calls.

## Phase 6. Workflow-Aware Issue API

- перевести issue reads/writes/filters на workflow statuses;
- сохранить legacy status compatibility;
- перевести sprint/dashboard metrics на category;
- обновить saved filters.

Результат: V1-V3 issue и sprint behavior работает поверх новой модели.

## Phase 7. Dynamic Issue Controls

- заменить hardcoded status options;
- добавить dynamic create/detail/filter controls;
- показать status name/color/category;
- обработать missing/archived values.

Результат: issue UI отражает workflow проекта.

## Phase 8. Project And Sprint Workflow Boards

- сделать project selection обязательным для Board;
- строить columns из workflow;
- ограничить drag/drop transition graph;
- перевести active sprint board.

Результат: Kanban работает с custom statuses безопасно и предсказуемо.

## Phase 9. Workflow Settings UI

- добавить status editor, reorder, transition editor;
- добавить archive with replacement;
- добавить permissions/error/loading states.

Результат: lead/admin управляет workflow end-to-end.

## Phase 10. Automation Schema And API

- добавить `automation_rules`;
- добавить validation и CRUD/order/enable-disable API;
- проверить invalid dependencies и permissions.

Результат: automation rules можно безопасно хранить и настраивать.

## Phase 11. Synchronous Automation Engine

- добавить trigger matching;
- добавить typed condition evaluator;
- добавить single-pass action execution;
- интегрировать engine в issue transactions;
- записывать automation activity.

Результат: простые rules выполняются атомарно без workers.

## Phase 12. Automation Management UI

- добавить rules list и builder;
- добавить enable/disable/reorder;
- показать validation и disabled reasons.

Результат: lead/admin создает automation без ручного JSON.

## Phase 13. Activity, Notifications And Seed

- расширить activity presentation;
- обновить notifications там, где automation меняет assignee/status;
- добавить V4 demo project workflow, roles и rules;
- сохранить idempotent seed.

Результат: V4 behavior видимо в demo и daily workflow.

## Phase 14. V4 Hardening

- расширить component, API smoke, integration и Playwright coverage;
- проверить permission isolation;
- проверить migration/backfill;
- обновить README;
- выполнить полный V1-V4 QA.

Результат: V4 можно стабильно использовать на localhost и self-hosted production.

## 10. Testing Strategy

Минимальный V4 testing baseline:

- все V1-V3 checks продолжают проходить;
- migration integration tests для workflow, membership и automation schema;
- workflow backfill сохраняет status keys и issue counts;
- workflow API tests для CRUD/order/transitions/archive/replacement;
- permission tests для admin/lead/contributor/viewer/no-membership;
- issue integration tests для allowed/forbidden transitions;
- sprint/dashboard tests для category-based done metrics;
- saved filter tests для legacy status и `workflowStatusId`;
- automation unit tests для triggers, conditions, ordering и action precedence;
- automation integration tests для atomic success/rollback и no-cascade behavior;
- frontend component tests для project members, workflow editor, dynamic status
  controls, board transitions и automation builder;
- API smoke для custom workflow, project roles и automation;
- Playwright e2e:
  - lead configures workflow and transition graph;
  - contributor moves issue through allowed transition;
  - forbidden transition is rejected;
  - viewer remains read-only;
  - status archive replaces issues;
  - automation changes issue and writes activity;
- full QA workflow остается зеленым.

После каждой фазы:

```sh
git diff --check
GOCACHE=/private/tmp/kelmio-gocache make verify
```

Перед финальным V4 закрытием:

```sh
GOCACHE=/private/tmp/kelmio-gocache make setup-db
GOCACHE=/private/tmp/kelmio-gocache make setup-db
make smoke-production
make smoke-api
make frontend-e2e
GOCACHE=/private/tmp/kelmio-gocache make backend-integration-test
make prod-compose-check
make prod-stack-qa
GOCACHE=/private/tmp/kelmio-gocache make verify
git diff --check
```

## 11. Definition Of Done For V4

V4 считается завершенной, когда:

1. V1, V2 и V3 behavior не сломан.
2. Каждый active project имеет валидный workflow.
3. Existing issues и statuses мигрированы без потери данных.
4. Lead/admin управляет statuses, order и transition graph.
5. Используемый status безопасно архивируется через replacement.
6. Board и active sprint board используют project workflow.
7. Backend строго проверяет разрешенные transitions.
8. Project roles `lead`, `contributor`, `viewer` работают end-to-end.
9. Участник без membership не видит project data.
10. Workspace admin сохраняет полный доступ.
11. Legacy `status` API и saved filters продолжают работать.
12. Sprint/dashboard completion metrics используют category `done`.
13. Automation rules создаются и управляются через UI.
14. Automation выполняется single-pass, атомарно и без cascade.
15. Automation activity понятно отображается.
16. Seed демонстрирует V4 workflow, roles и automation.
17. Component, unit, integration, smoke и browser e2e checks проходят.
18. Production/self-hosted V3 flow не сломан.
19. README описывает V4 behavior и permissions.
20. После финального QA нет известных V4 blocker bugs.

## 12. Risks And Anti-Patterns

Основные риски V4:

- слишком быстро удалить legacy status contract;
- смешать workflow category и пользовательский status key;
- сделать permissions только frontend-проверкой;
- допустить project data leakage через links, sprints или notifications;
- разрешить удаление status без atomic replacement;
- превратить automation в generic programming language;
- допустить automation cascade или циклы;
- считать sprint progress по имени/key custom status;
- построить cross-project board, который скрывает реальные workflow различия;
- расширить scope до multi-workspace или custom permission matrix.

Как защищаемся:

- сначала schema/backfill и integration tests;
- сохраняем additive `/api/v1` compatibility;
- permission checks централизуются и проверяются backend;
- каждый destructive workflow action выполняется transactionally;
- automation остается typed, synchronous и single-pass;
- board всегда project-specific;
- работаем маленькими commits;
- полный V1-V4 QA обязателен перед завершением.

## 13. Proposed Commit Order

Практический порядок V4:

1. `Add V4 plan`
2. `Add project workflow schema and backfill`
3. `Add workflow status and transition backend API`
4. `Add project membership schema and API`
5. `Enforce project permissions`
6. `Add project member management UI`
7. `Migrate issue API to workflow statuses`
8. `Add dynamic status filters and issue controls`
9. `Add project and sprint workflow boards`
10. `Add workflow settings UI`
11. `Add automation rules schema and API`
12. `Add synchronous automation engine`
13. `Add automation management UI`
14. `Extend activity notifications and seed data`
15. `Extend V4 component smoke integration and e2e tests`
16. `Update README for V4`
17. `Finalize V4 QA polish`

Порядок должен сохраняться: сначала migration/backfill, затем permissions и
workflow-aware domain behavior, затем UI, automation, tests/docs и final QA.

## 14. Final V4 QA Result

V4 полностью завершена 15 июня 2026 года.

Финальный audit подтвердил:

- два последовательных idempotent `make setup-db`;
- полный V1-V4 unit, component, integration, race и vet baseline;
- project workflow, transition graph, archive replacement и legacy status
  compatibility;
- project roles, permission isolation, dynamic project и active sprint boards;
- synchronous single-pass automation, activity и notifications;
- API smoke и все Playwright e2e сценарии;
- production config, Compose, production images и isolated clean-room stack с
  TLS, bootstrap, hardening smoke, backup и restore-check;
- dependency audit без известных уязвимостей после обновления Vite до `8.0.16`;
- отсутствие известных V4 blocker bugs.

Следующий этап проекта должен начинаться с отдельного V5 planning документа, без
расширения завершенного V4 scope.
