# Team Task Tracker V2 Plan

## 1. Product Goal

V1 закрыла базовый localhost task tracker: auth, workspace team, projects, issues, board/list, labels, comments, activity log, filters, seed data, Docker и тестовый контур.

Цель V2: превратить базовый task tracker в более сильный инструмент планирования работы небольшой команды, не ломая простоту V1 и не уходя в cloud/deploy раньше времени.

Ключевой принцип V2: добавлять только тот функционал, который усиливает ежедневную командную работу:

- планирование задач по итерациям;
- декомпозиция больших задач;
- прогнозирование объема работы;
- сохранение рабочих представлений;
- понятная командная видимость изменений.

## 2. V2 Boundary

### Что входит в V2

- URL-level navigation для основных экранов приложения;
- постепенное разделение большого frontend-кода на feature modules;
- issue hierarchy: epic, story/task, subtask;
- базовые issue links: blocks, relates;
- lightweight sprints/iterations;
- sprint backlog и active sprint board;
- story points для оценки задач;
- sprint summary/progress на dashboard;
- saved filters/views для списка задач;
- in-app notifications без email и без WebSocket;
- улучшенная activity visibility для hierarchy, sprint, estimate и links;
- дополнительные backend integration tests;
- расширенный browser e2e smoke для ключевых V2 сценариев;
- обновленный README с V2 командами и сценариями.

### Что сознательно не входит в V2

- cloud deployment;
- production hosting;
- mobile app;
- external integrations;
- email notifications;
- real-time updates через WebSocket;
- automation rules;
- полностью custom workflows per project;
- advanced permission matrix;
- file attachments/object storage;
- полноценный time tracking;
- billing или multi-tenant SaaS модель.

V2 должна остаться localhost-first версией. Сервер, деплой и production security hardening планируются отдельно после завершения V2.

## 3. Main User Scenarios

V2 считается полезной, если хорошо закрывает такие сценарии:

1. Пользователь открывает прямую ссылку на нужный раздел, проект, задачу или sprint.
2. Пользователь создает epic и привязывает к нему задачи.
3. Пользователь разбивает задачу на subtasks.
4. Пользователь связывает задачи отношениями blocks/relates.
5. Команда создает sprint, добавляет в него задачи и запускает sprint.
6. Команда видит active sprint на board и в list view.
7. Пользователь выставляет story points и видит общий объем sprint.
8. Пользователь сохраняет часто используемый фильтр задач.
9. Пользователь получает in-app notification о назначении, комментарии или важном изменении задачи.
10. Команда видит progress по sprint и нагрузку по исполнителям.

Если новая задача не помогает этим сценариям, ее нужно либо отложить, либо вынести в V3.

## 4. Architecture Direction

V2 не требует переписывать V1. Основная архитектура сохраняется:

- backend: Go modular monolith;
- API: REST JSON;
- database: PostgreSQL;
- auth: server-side sessions в PostgreSQL с HttpOnly cookie;
- frontend: React + TypeScript + Vite;
- local infrastructure: Docker Compose.

### Что можно добавить в V2

- React Router для URL-level navigation;
- небольшую feature-based структуру во frontend;
- отдельные backend domain modules для `sprints`, `notifications`, `issue_links`;
- расширение миграций без пересоздания V1 schema;
- дополнительные Makefile commands только если они реально нужны;
- polling для notification count вместо WebSocket.

### Что не нужно делать в V2

- переписывать backend на другую архитектуру;
- делать microservices;
- добавлять Redis/message queue;
- переходить на GraphQL;
- мигрировать UI на тяжелый design system;
- вводить `/api/v2` только из-за версии продукта.

API version может остаться `/api/v1`, пока нет breaking changes для существующего frontend contract.

## 5. Domain Model Additions

V2 должна расширять V1 schema миграциями.

### `issues`

Новые поля:

- `parent_issue_id` для subtasks;
- `story_points`;
- `sprint_id`;

Новые значения:

- `issue_type`: добавить `epic` и `subtask`.

Правила:

- `subtask` должен иметь `parent_issue_id`;
- `epic` не должен иметь parent;
- нельзя создать цикл parent-child;
- parent и child должны быть в одном workspace;
- closed parent не должен блокировать просмотр subtasks.

### `issue_links`

- `id`;
- `source_issue_id`;
- `target_issue_id`;
- `link_type` (`blocks`, `relates`);
- `created_by`;
- `created_at`.

Правила:

- нельзя связать задачу саму с собой;
- одинаковая связь не должна дублироваться;
- обе задачи должны быть доступны в одном workspace.

### `sprints`

- `id`;
- `workspace_id`;
- `project_id`;
- `name`;
- `goal`;
- `status` (`planned`, `active`, `completed`);
- `start_date`;
- `end_date`;
- `created_by`;
- `created_at`;
- `completed_at`.

Правила:

- в рамках проекта может быть только один active sprint;
- sprint можно завершить без удаления задач;
- задачи completed sprint остаются доступными в истории;
- issue может находиться максимум в одном текущем sprint.

### `saved_filters`

- `id`;
- `workspace_id`;
- `user_id`;
- `name`;
- `filters` as `jsonb`;
- `created_at`;
- `updated_at`.

Правила:

- saved filter принадлежит пользователю;
- фильтры не должны ломаться, если label/project/user архивирован;
- UI должен понятно показывать missing/archived values.

### `notifications`

- `id`;
- `workspace_id`;
- `user_id`;
- `actor_id`;
- `issue_id`;
- `notification_type`;
- `payload` as `jsonb`;
- `read_at`;
- `created_at`.

Правила:

- пользователь не должен получать notification на собственное действие;
- notification не заменяет activity log;
- read/unread state хранится отдельно от issue state.

## 6. API Surface Additions

V2 добавляет endpoints без удаления V1 endpoints.

### Issue Hierarchy

- `POST /api/v1/issues/:id/subtasks`
- `PATCH /api/v1/issues/:id/parent`
- `GET /api/v1/issues/:id/children`

### Issue Links

- `GET /api/v1/issues/:id/links`
- `POST /api/v1/issues/:id/links`
- `DELETE /api/v1/issues/:id/links/:linkId`

### Sprints

- `GET /api/v1/sprints`
- `POST /api/v1/sprints`
- `GET /api/v1/sprints/:id`
- `PATCH /api/v1/sprints/:id`
- `POST /api/v1/sprints/:id/start`
- `POST /api/v1/sprints/:id/complete`
- `POST /api/v1/sprints/:id/issues`
- `DELETE /api/v1/sprints/:id/issues/:issueId`

### Saved Filters

- `GET /api/v1/saved-filters`
- `POST /api/v1/saved-filters`
- `PATCH /api/v1/saved-filters/:id`
- `DELETE /api/v1/saved-filters/:id`

### Notifications

- `GET /api/v1/notifications`
- `GET /api/v1/notifications/unread-count`
- `POST /api/v1/notifications/:id/read`
- `POST /api/v1/notifications/read-all`

## 7. Frontend Screens For V2

V2 должна сохранить все V1 экраны и добавить:

- route-based app shell;
- issue detail route/drawer state via URL;
- sprint list page;
- sprint detail page;
- active sprint board;
- backlog planning panel;
- saved filters panel;
- notifications dropdown/page;
- hierarchy block in issue detail;
- linked issues block in issue detail;
- sprint summary cards on dashboard.

## 8. UX Principles

V2 UI должен стать сильнее, но не тяжелее.

- Любой новый flow должен быть доступен за 1-2 понятных действия.
- Sprint planning не должен превращаться в сложную Jira-конфигурацию.
- Hierarchy должна помогать понимать работу, а не скрывать задачи.
- Notifications должны быть полезными, а не шумными.
- Saved filters должны ускорять работу, а не требовать настройки.
- Board/list/detail должны продолжать использовать один backend source of truth.

## 9. Development Phases

## Phase 0. V2 Planning Baseline

- зафиксировать V2 plan;
- убедиться, что V1 checks проходят;
- не менять V1 behavior без причины;
- завести V2 checklist в документации.

Результат: есть официальный план V2, от которого дальше закрываются задачи.

## Phase 1. Frontend Structure And Routing

- подключить route-level navigation;
- сохранить текущие разделы как routes;
- добавить direct links для project, issue, board, team, labels, account;
- начать выносить код из большого `App.tsx` в feature modules;
- сохранить V1 UI behavior.

Результат: приложение готово расти без превращения frontend в один огромный файл.

## Phase 2. Issue Hierarchy

- добавить миграцию для `parent_issue_id` и новых issue types;
- добавить backend validation against cycles;
- добавить API для subtasks/children;
- добавить UI для epic/subtask relations;
- добавить activity events для hierarchy changes;
- покрыть backend tests и один frontend/e2e сценарий.

Результат: большие задачи можно разбивать на понятную структуру.

## Phase 3. Issue Links

- добавить таблицу `issue_links`;
- добавить API для create/list/delete links;
- добавить UI блок linked issues;
- добавить link validation и activity events;
- добавить tests.

Результат: задачи можно связывать отношениями blocks/relates.

## Phase 4. Sprints Core

- добавить таблицу `sprints`;
- добавить `sprint_id` в issues;
- добавить sprint CRUD;
- добавить start/complete sprint flow;
- добавить правила одного active sprint per project;
- добавить backend tests.

Результат: команда может планировать работу итерациями.

## Phase 5. Sprint UI And Planning

- добавить sprint list/detail screens;
- добавить backlog planning panel;
- добавить перенос задач в sprint;
- добавить active sprint board;
- добавить sprint filters в issue list;
- синхронизировать sprint board/list/detail state.

Результат: sprint workflow можно использовать в ежедневной работе.

## Phase 6. Story Points And Sprint Summary

- добавить `story_points`;
- добавить UI для оценки задач;
- добавить суммарный объем sprint;
- добавить dashboard cards: sprint progress, points done/open, workload by assignee;
- добавить validation и tests.

Результат: команда видит объем работы и progress sprint.

## Phase 7. Saved Filters

- добавить таблицу `saved_filters`;
- добавить CRUD API;
- добавить UI для сохранения текущих issue filters;
- добавить применение saved filter одним действием;
- добавить обработку archived/missing filter values.

Результат: пользователь быстро возвращается к своим рабочим представлениям.

## Phase 8. In-App Notifications

- добавить таблицу `notifications`;
- создавать notifications на assignment, mention-like comment text, direct comment on assigned/reported issue, sprint start/complete;
- добавить unread count;
- добавить notifications dropdown/page;
- добавить mark read/read all;
- использовать polling, без WebSocket.

Результат: пользователь видит важные изменения без email и внешних сервисов.

## Phase 9. V2 Hardening

- расширить API smoke script V2 сценариями;
- расширить Playwright e2e;
- добавить integration tests для новых миграций;
- проверить access checks;
- проверить empty/error/loading states;
- обновить seed data;
- обновить README;
- финальный ручной localhost QA.

Результат: V2 можно стабильно использовать на localhost.

## 10. Testing Strategy

Минимальный V2 testing baseline:

- все V1 checks продолжают проходить;
- backend unit tests для hierarchy, issue links, sprints, saved filters, notifications;
- backend integration tests для новых миграций;
- API smoke сценарии для sprint и hierarchy;
- browser e2e:
  - login -> create epic -> create child issue -> create subtask;
  - create sprint -> add issue -> start sprint -> move issue -> complete sprint;
  - save issue filter -> apply saved filter;
  - assign issue -> notification appears -> mark read.

## 11. Definition Of Done For V2

V2 считается завершенной, когда:

1. V1 функционал не сломан.
2. Приложение поднимается локально через Docker.
3. Есть URL-level navigation для основных разделов.
4. Можно создавать epic и subtasks.
5. Можно связывать задачи blocks/relates.
6. Можно создать sprint, добавить задачи, запустить и завершить sprint.
7. Есть active sprint board/list.
8. Можно задавать story points.
9. Dashboard показывает sprint progress и базовую нагрузку.
10. Можно сохранить и применить issue filter.
11. Работают in-app notifications.
12. README описывает V2 сценарии.
13. `make verify`, API smoke, backend integration tests и browser e2e проходят.
14. После финального QA нет известных V2 blocker bugs.

## 12. Risks And Anti-Patterns

Основные риски V2:

- расползание scope до "почти Jira";
- слишком ранний переход к deployment;
- слишком сложный sprint workflow;
- попытка добавить custom workflows одновременно со sprint planning;
- перегруз notifications;
- рост `App.tsx` без разбиения на feature modules;
- недостаточные migration tests.

Как защищаемся:

- работаем маленькими commits;
- каждая фаза должна иметь проверяемый результат;
- не начинаем следующую большую область, пока предыдущая не работает end-to-end;
- сохраняем localhost-first подход;
- не добавляем infrastructure complexity без реальной причины.

## 13. Proposed Commit Order

Ожидаемый размер V2: примерно 18-24 небольших коммита.

Практический порядок:

1. add V2 plan;
2. add routing foundation;
3. split frontend API/types/helpers by feature;
4. split first frontend sections out of `App.tsx`;
5. add issue hierarchy migration;
6. add hierarchy backend API/tests;
7. add hierarchy UI;
8. add issue links migration;
9. add issue links backend API/tests;
10. add issue links UI;
11. add sprints migration;
12. add sprints backend API/tests;
13. add sprint list/detail UI;
14. add backlog planning UI;
15. add active sprint board;
16. add story points and sprint summary;
17. add saved filters backend/UI;
18. add notifications backend/UI;
19. extend seed data;
20. extend smoke/e2e tests;
21. update README;
22. final V2 QA polish.

Количество коммитов может измениться, но порядок должен оставаться примерно таким: сначала foundation, затем domain, затем UI, затем tests/docs.

## 14. Decision For Next Step

Следующий практический шаг после этого плана:

начать Phase 1 с route-level navigation и аккуратного разбиения frontend-кода, потому что V2 добавит много новых экранов. Если сначала добавить sprints/hierarchy поверх текущего большого `App.tsx`, frontend быстро станет сложным и дорогим для поддержки.
