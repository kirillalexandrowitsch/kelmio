# Kelmio V6 Plan

## 1. Product Goal

V1-V5 закрыли task tracking, agile planning, production-ready self-hosted
foundation, project workflows, project permissions, synchronous automation,
account recovery и localhost operations foundation.

Цель V6: превратить текущую single-workspace установку в локальную
**multi-organization** платформу с несколькими изолированными организациями,
несколькими workspaces внутри организации, тремя уровнями администрирования
(global/site, organization, workspace) и переиспользуемыми группами и
role assignments.

V6 должна дать пользователям и администраторам:

- несколько изолированных организаций в одной локальной установке;
- несколько workspaces внутри организации с изоляцией данных;
- понятную модель ролей на уровнях site, organization и workspace;
- группы пользователей и directory, назначаемые согласованно между workspaces
  и проектами;
- безопасное переключение между workspaces без утечки данных;
- безопасную миграцию существующего single workspace в новую структуру;
- organization-aware seed, backup и audit foundation.

Ключевой принцип V6: модель остается **modular monolith** и полностью
воспроизводится на localhost. Реальный hosting, managed multi-tenant cloud,
SaaS billing и public deployment не входят в целевую модель проекта.

V6 закрывает capabilities `PLAT-006` и `PLAT-007` из
[product capability baseline](product-capability-baseline.md).

## 2. V6 Boundary

### Что входит в V6

- таблица `organizations` и organization lifecycle (create/rename/archive);
- multi-workspace модель: несколько workspaces в организации;
- workspace lifecycle (create/rename/archive) внутри организации;
- роли на трех уровнях: site (global) admin, organization admin, workspace admin
  и workspace member;
- `groups` и `group_members`, переиспользуемые в пределах организации;
- directory активных пользователей организации;
- reusable role assignments: назначение пользователя или группы на workspace и
  на project role;
- workspace switching в UI с серверной изоляцией всех запросов;
- безопасная idempotent migration существующего workspace в default organization;
- organization/workspace-aware seed и bootstrap;
- organization-aware backup и audit foundation (минимальный, расширяется позже);
- global/organization/workspace administration screens;
- unit, component, integration, smoke и browser e2e coverage;
- обновление operations и setup документации;
- финальный V1-V6 QA polish.

### Что сознательно не входит в V6

- реальный hosting provider, domain, managed cloud или public deployment;
- SaaS billing и подписки;
- внешние identity providers, SSO/OIDC (это V21 `GOV-003`);
- полноценный organization audit log и compliance export (V21 `GOV-002`);
- cross-organization sharing данных (организации остаются изолированными);
- configurable work types, custom fields и schemes (V7);
- advanced search/bulk operations (V8);
- per-user notification/locale preferences (V13 `PLAT-008`);
- любые capabilities, не входящие в `PLAT-006`/`PLAT-007`.

## 3. Main User Scenarios

V6 считается полезной, если хорошо закрывает такие сценарии:

1. После апгрейда существующие данные оказываются в default organization и
   default workspace без потерь и без ручной правки базы.
2. Site admin создает новую организацию и назначает organization admin.
3. Organization admin создает несколько workspaces и видит только свою
   организацию.
4. Workspace admin управляет участниками своего workspace, но не видит чужие
   workspaces или организации.
5. Пользователь, состоящий в нескольких workspaces, переключается между ними, и
   все списки (projects, issues, board, sprints) показывают только данные
   текущего workspace.
6. Запрос к ресурсу чужого workspace или организации отклоняется на сервере.
7. Organization admin создает группу, добавляет участников и назначает группе
   роль в workspace; все участники получают доступ согласованно.
8. Удаление пользователя из группы синхронно убирает производный доступ.
9. Directory показывает активных пользователей организации для назначения без
   раскрытия пользователей других организаций.
10. Архивация workspace скрывает его из switcher, но сохраняет данные для audit.
11. Seed создает воспроизводимую multi-organization демо-структуру для QA.
12. Все V1-V5 workflows продолжают работать внутри default workspace.

Если новая задача не помогает этим сценариям, ее нужно отложить до следующей
версии.

## 4. Architecture Direction

V6 сохраняет текущую архитектуру:

- backend: Go modular monolith;
- API: REST JSON `/api/v1`;
- database: PostgreSQL;
- auth: server-side sessions;
- frontend: React + TypeScript + Vite;
- local runtime: Docker Compose;
- existing production-like QA runtime: Docker Compose и Caddy.

### Что можно добавить в V6

- новый identity/tenancy module для organizations, workspaces, groups и role
  assignments;
- PostgreSQL migrations, которые добавляют `organization_id` к workspace-уровню
  и backfill существующих данных;
- session-scoped «active workspace» с серверной валидацией на каждом запросе;
- reusable authorization layer, который объединяет direct и group-derived роли;
- additive API endpoints и frontend screens;
- organization-aware seed и bootstrap-admin расширение.

### Что не нужно делать в V6

- вводить отдельный per-tenant database или schema-per-tenant;
- добавлять Redis, message broker или новый runtime-компонент;
- выделять tenancy в отдельный сервис (monolith сохраняется);
- реализовывать SSO/OIDC, audit export или compliance certifications;
- ломать существующую workspace-scoped изоляцию projects/issues/labels;
- хранить cross-organization связи данных;
- менять завершенные V1-V5 product boundaries.

## 5. Domain And Schema Additions

Текущая схема уже содержит `workspaces`, `workspace_members`
(role `admin`/`member`) и `workspace_id` scoping для projects, labels, issues и
т.д. V6 надстраивает organization-слой и groups над этой основой. Все изменения
должны быть additive и idempotent, с backfill существующего workspace.

### `organizations`

- `id`;
- `name`;
- `slug` (уникальный, человекочитаемый идентификатор);
- `status` (`active`/`archived`);
- `created_by`;
- timestamps.

Правила:

- организация — верхний уровень изоляции;
- ресурсы одной организации недоступны из другой;
- архивированная организация скрыта, но сохраняет данные.

### `workspaces` (расширение)

Добавить:

- `organization_id` (NOT NULL, REFERENCES organizations);
- `status` (`active`/`archived`);
- `slug` уникален в пределах организации.

Правила:

- каждый workspace принадлежит ровно одной организации;
- `UNIQUE (organization_id, slug)`;
- существующий «Local Workspace» получает default organization при migration.

### `organization_members`

- `organization_id`;
- `user_id`;
- `role` (`org_admin`/`org_member`);
- timestamps;
- PRIMARY KEY (organization_id, user_id).

### Site (global) admin

- флаг site-admin на пользователе (`users.is_site_admin`) либо отдельная
  таблица site administrators;
- site admin может управлять организациями и назначать organization admins;
- bootstrap-admin расширяется, чтобы создавать первого site admin.

### `groups` и `group_members`

`groups`:

- `id`;
- `organization_id`;
- `name` (уникально в организации);
- `description`;
- timestamps.

`group_members`:

- `group_id`;
- `user_id`;
- timestamps;
- PRIMARY KEY (group_id, user_id).

### Reusable role assignments

`workspace_role_assignments` и `project_role_assignments` (или единая
полиморфная таблица) хранят назначение **subject** (user или group) на роль:

- `id`;
- `scope` (`workspace`/`project`);
- `scope_id`;
- `subject_type` (`user`/`group`);
- `subject_id`;
- `role`;
- timestamps;
- уникальность по (scope, scope_id, subject_type, subject_id).

Правила:

- эффективная роль пользователя = максимум из direct и group-derived ролей;
- изменение состава группы синхронно меняет производный доступ;
- workspace_members остается совместимым представлением для direct membership;
- project membership (V4 `WORK-003`) продолжает работать и может питаться из
  assignments.

### Migration существующего workspace

- создать default organization;
- привязать существующий workspace к ней;
- преобразовать существующих workspace admins в organization admins там, где это
  требуется;
- сохранить все project memberships и роли;
- migration должна быть idempotent (повторный `setup-db` безопасен).

## 6. API Additions

Все endpoints — additive, под `/api/v1`, с серверной проверкой scope.

### Organizations (site admin)

- `GET /api/v1/organizations` — список (site admin видит все, прочие — свои);
- `POST /api/v1/organizations` — создать;
- `PATCH /api/v1/organizations/{id}` — переименовать/архивировать;
- управление organization admins.

### Workspaces

- `GET /api/v1/workspaces` — workspaces, доступные текущему пользователю в
  активной организации;
- `POST /api/v1/workspaces` — создать (organization admin);
- `PATCH /api/v1/workspaces/{id}` — rename/archive;
- `POST /api/v1/session/active-workspace` — выбрать активный workspace; сервер
  валидирует доступ и переписывает scope последующих запросов.

### Groups и directory

- `GET /api/v1/groups`, `POST`, `PATCH`, добавление/удаление участников;
- `GET /api/v1/directory` — активные пользователи организации для назначения;
- `GET/POST/DELETE` role assignments для workspace и project с subject
  user/group.

### Правила

- каждый business endpoint резолвит организацию и активный workspace из сессии;
- любой доступ к чужому scope возвращает `403`/`404` без раскрытия данных;
- существующие V1-V5 endpoints продолжают работать в активном workspace.

## 7. Frontend Screens And Behavior

### Workspace / organization switcher

- в shell добавляется switcher (организация + workspace);
- переключение workspace перезагружает данные текущего раздела с новым scope;
- switcher показывает только доступные пользователю организации/workspaces.

### Administration

- **Site administration**: список организаций, создание, назначение org admins
  (виден только site admin);
- **Organization administration**: workspaces, organization members, groups,
  directory;
- **Workspace administration**: участники и role assignments (расширяет текущий
  Team/Project members UI).

### Поведение и изоляция

- все существующие разделы (Dashboard, Projects, Issues, Board, Sprints, Team,
  Labels) работают в пределах активного workspace;
- права admin/lead/contributor/viewer/member сохраняются;
- визуальный язык — текущая light Aurora design system;
- администрирование скрыто от пользователей без соответствующей роли.

## 8. UX Principles

- Пользователь всегда понимает, в какой организации и workspace он работает.
- Переключение workspace не смешивает данные разных scope.
- Администрирование разнесено по уровням и видно только нужным ролям.
- Группы упрощают массовое назначение доступа и остаются единственным источником
  производного доступа.
- Существующие task/agile flows не усложняются для single-workspace
  пользователя.
- Default установка остается простой: один organization + workspace «из
  коробки», multi-tenancy не навязывается.

## 9. Development Phases

### Phase 1: Organization и workspace schema + migration

- добавить `organizations`, расширить `workspaces`, добавить
  `organization_members`;
- backfill default organization и привязать существующий workspace;
- idempotent migration и обновленный seed/bootstrap.

Результат: данные живут в organization/workspace структуре без потерь.

### Phase 2: Tenancy authorization foundation

- session-scoped active organization/workspace;
- серверная проверка scope для всех существующих endpoints;
- site/org/workspace роли в authorization layer.

Результат: изоляция данных гарантируется на сервере.

### Phase 3: Organization lifecycle и site administration

- organizations API и site admin screens;
- назначение organization admins;
- bootstrap первого site admin.

Результат: можно создавать и администрировать организации локально.

### Phase 4: Workspace lifecycle и switching

- workspaces API и organization workspace management;
- workspace switcher в shell;
- archive/rename workspace.

Результат: несколько workspaces с безопасным переключением.

### Phase 5: Groups, directory и reusable role assignments

- groups/group_members API и UI;
- directory организации;
- workspace/project role assignments для user и group;
- объединение direct и group-derived ролей.

Результат: согласованное переиспользуемое назначение доступа.

### Phase 6: Organization-aware operations foundation

- organization-aware seed для QA;
- backup/restore учитывают organization scope;
- минимальный audit foundation для административных действий.

Результат: operations и QA работают в multi-organization модели.

### Phase 7: V6 test expansion и final QA

- integration tests изоляции и migration idempotency;
- component/e2e для switcher, admin и groups;
- полный V1-V6 regression и Full QA;
- обновление документации и фиксация итога V6.

Результат: V6 завершена и готова стать основой V7.

## 10. Testing Strategy

### Backend unit tests

- organization/workspace/group validation и slug rules;
- effective-role resolution (direct + group-derived, максимум);
- scope-resolution из сессии;
- migration backfill normalization.

### Backend integration tests

- cross-scope доступ отклоняется (workspace/organization isolation);
- workspace switching меняет видимые данные;
- group membership изменения меняют производный доступ;
- idempotent migration и два последовательных `setup-db`;
- сохранение V1-V5 project membership и ролей.

### Frontend tests

- switcher поведение и scope-aware загрузка;
- site/organization/workspace admin permission states;
- groups и role assignment UI;
- regression component tests V1-V5.

### Smoke и browser e2e

- migration smoke: существующие данные доступны в default org/workspace;
- создать организацию, workspace, переключиться и проверить изоляцию;
- создать группу, назначить роль, проверить согласованный доступ;
- полный V1-V6 business regression.

### Full QA

- два последовательных database setup;
- unit, component, integration, race и vet checks;
- API, production-sensitive и operations smoke;
- Playwright e2e;
- Docker Compose config и clean-room production-stack regression;
- dependency audit и `git diff --check`.

## 11. Definition Of Done

V6 полностью завершена, когда:

1. Существующая установка мигрирует в default organization/workspace без потерь
   и idempotently.
2. Поддерживаются несколько изолированных организаций.
3. Поддерживаются несколько workspaces в организации с изоляцией данных.
4. Роли site/organization/workspace admin работают и проверяются на сервере.
5. Группы и directory переиспользуются для согласованного назначения доступа.
6. Эффективная роль корректно объединяет direct и group-derived роли.
7. Workspace switching не смешивает данные разных scope.
8. Любой доступ к чужому scope отклоняется сервером.
9. Seed создает воспроизводимую multi-organization структуру для QA.
10. Все V1-V5 regressions остаются зелеными.
11. Полный V1-V6 QA и GitHub Full QA успешны.
12. V6 setup/administration docs соответствуют фактическому поведению.
13. Нет известных V6 blocker bugs.

После выполнения этих критериев `PLAT-006` и `PLAT-007` отмечаются `complete`
в [product capability baseline](product-capability-baseline.md).

## 12. Risks

- небезопасная migration существующего workspace и потеря данных или ролей;
- неполная серверная проверка scope и утечка данных между organizations или
  workspaces;
- рассинхронизация direct и group-derived доступа;
- избыточно сложная role-assignment модель, мешающая single-workspace
  пользователю;
- регрессия существующей workspace-scoped изоляции projects/issues/labels;
- попытка ввести per-tenant database/schema или отдельный сервис;
- расширение V6 в сторону SSO/audit export/compliance (это V21);
- усложнение default localhost setup для простого сценария.

## 13. Proposed Commit Order

1. `Add organization and workspace schema with migration`
2. `Add tenancy authorization and session scope`
3. `Add organization lifecycle and site administration`
4. `Add workspace lifecycle and switching`
5. `Add groups directory and reusable role assignments`
6. `Add organization-aware seed and operations foundation`
7. `Extend V6 integration component and e2e tests`
8. `Finalize V6 QA polish`

Каждый commit должен быть небольшим, проходить релевантные tests и сохранять
работоспособность завершенных V1-V5 flows.
