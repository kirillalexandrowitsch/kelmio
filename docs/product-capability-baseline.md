# Kelmio Product Capability Baseline

## 1. Purpose

Этот документ является измеримым источником истины для долгосрочной цели
Kelmio: создать оригинальную localhost-only платформу для управления работой,
планирования, service operations, product discovery и administration.

Baseline фиксирует значимые пользовательские, административные и operations
capabilities, которые должны работать как единая независимая система.

Каждый будущий version plan должен:

- ссылаться на capability IDs, которые он закрывает;
- не менять статус capability на `complete` до успешного Definition of Done;
- добавлять новую capability только вместе с целевой версией и acceptance
  criteria;
- обновлять этот baseline в финальном QA-коммите версии.

## 2. Product Boundary

### Включено

- configurable work management и agile delivery;
- service management и customer self-service;
- product discovery и portfolio planning;
- assets и configuration management;
- встроенные workflows, automation, forms, reports и administration;
- provider-neutral AI assistance для значимых product scenarios;
- optional реальные integrations с localhost mocks для воспроизводимого QA;
- responsive PWA как mobile-capability;
- multi-organization и multi-workspace модель внутри одной локальной установки.

### Не включено

- отдельные системы для документов, video collaboration, source hosting и team chat;
- паритет с тысячами сторонних marketplace applications;
- копирование чужого branding, визуального дизайна, proprietary code и API;
- SaaS billing, managed cloud, real hosting provider и public deployment;
- native iOS/Android applications;
- public-cloud infrastructure scale, external data residency и formal compliance certifications;
- новые capabilities, которые не были явно приняты в baseline.

### Правило завершения capability

Capability считается `complete`, когда Kelmio предоставляет полный
end-to-end результат, permission model, error handling, administrative controls
и automated/manual QA.

## 3. Status Model

| Status | Meaning |
|---|---|
| `complete` | Capability реализована, документирована и прошла финальный QA своей версии |
| `planned` | Capability назначена будущей версии |
| `in_progress` | Capability разрабатывается в текущей версии |
| `deferred` | Capability включена в baseline, но целевая версия требует пересмотра |
| `not_applicable` | Capability явно исключена из выбранного functional-parity scope |

## 4. Capability Matrix

### Platform, Identity And Operations

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| PLAT-001 | Local Docker development and reproducible QA | V1-V4 | `complete` | Application, database, seed, smoke, integration and browser QA run locally |
| PLAT-002 | Session authentication and account management | V1-V4 | `complete` | Secure login/logout/session lifecycle and self-service account editing |
| PLAT-003 | Production-like security and operations baseline | V3 | `complete` | CSRF, secure config, logs, request IDs, backups, restore and clean-room stack QA |
| PLAT-004 | Account recovery and durable system email | V5 | `complete` | Password reset, SMTP outbox, worker, diagnostics and reliable retries |
| PLAT-005 | Metrics, alerts and scheduled restore verification | V5 | `complete` | Local Prometheus/Grafana/Alertmanager and automated backup restore drill |
| PLAT-006 | Organizations, workspaces and site administration | V6 | `planned` | Multiple isolated organizations/workspaces with global and workspace admins |
| PLAT-007 | Groups, directories and reusable role assignments | V6 | `planned` | Users and groups can be assigned consistently across workspaces and projects |
| PLAT-008 | Personal preferences, profiles and notification preferences | V13 | `planned` | Users manage locale, appearance, communication and notification settings |

### Work Items, Fields And Workflows

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| WORK-001 | Core work items, hierarchy, comments, labels and links | V1-V2 | `complete` | Issues support CRUD, hierarchy, subtasks, comments, labels, links and activity |
| WORK-002 | Project workflows and transition graphs | V4 | `complete` | Custom statuses, strict transitions and archive-with-replacement work end-to-end |
| WORK-003 | Project roles and permission isolation | V4 | `complete` | Lead, contributor, viewer and no-membership access are enforced server-side |
| WORK-004 | Configurable work types and hierarchy schemes | V7 | `planned` | Admins define work types, icons, levels and allowed hierarchy |
| WORK-005 | Custom fields and field contexts | V7 | `planned` | Typed fields support scopes, defaults, validation and searchable values |
| WORK-006 | Screens, layouts and field configuration schemes | V7 | `planned` | Create/edit/view layouts and required/hidden fields are configurable |
| WORK-007 | Workflow schemes, conditions, validators and post-functions | V7 | `planned` | Reusable workflows and transition policies can be assigned by project/type |
| WORK-008 | Issue security levels and fine-grained visibility | V21 | `planned` | Work-item visibility can be restricted independently of project access |

### Search, Filters And Data Exchange

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| SEARCH-001 | Issue filters and personal saved filters | V1-V2 | `complete` | Users can filter issue lists and persist personal views |
| SEARCH-002 | Advanced query language and builder | V8 | `planned` | Structured and text queries cover fields, functions, history and relations |
| SEARCH-003 | Shared filters, subscriptions and permissions | V8 | `planned` | Filters can be shared, scheduled and permission-controlled |
| SEARCH-004 | Bulk edit, transition, move, archive and delete | V8 | `planned` | Large result sets support safe previewed bulk operations |
| SEARCH-005 | CSV import/export and field mapping | V8 | `planned` | Work items can be imported/exported with validation and error reporting |
| SEARCH-006 | Cross-domain global search | V22 | `planned` | Search spans work items, projects, requests, assets, knowledge and ideas |

### Agile, Boards And Delivery

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| AGILE-001 | Scrum sprints, backlog and story points | V2 | `complete` | Plan, start and complete sprints with issue assignment and point summaries |
| AGILE-002 | Dynamic project and active sprint boards | V4 | `complete` | Boards use project workflows and enforce transition permissions |
| AGILE-003 | Configurable boards from saved queries | V9 | `planned` | Multiple Scrum/Kanban boards use configurable filters and locations |
| AGILE-004 | Board columns, swimlanes, quick filters and card layouts | V9 | `planned` | Board presentation and grouping are configurable per board |
| AGILE-005 | Ranking, WIP limits and Kanban backlog | V9 | `planned` | Teams can rank work and enforce/visualize flow constraints |
| AGILE-006 | Components, versions and releases | V10 | `planned` | Projects track ownership, release versions, release notes and status |
| AGILE-007 | Development and deployment status | V10 | `planned` | Commits, branches, pull requests, builds and deployments link to work |

### Plans, Goals, Reports And Dashboards

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| PLAN-001 | Basic sprint and dashboard summaries | V2 | `complete` | Dashboard and sprint summaries expose current delivery state |
| PLAN-002 | Cross-project plans and dependency timeline | V11 | `planned` | Plans combine teams, releases, hierarchy, dependencies and scenarios |
| PLAN-003 | Goals, initiatives and outcome alignment | V11 | `planned` | Work and plans connect to measurable organizational goals |
| PLAN-004 | Capacity, estimates and scenario planning | V11 | `planned` | Teams model capacity, dates and alternative delivery scenarios |
| REPORT-001 | Agile reports | V12 | `planned` | Burndown, burnup, velocity, sprint, CFD, control and cycle-time reports |
| REPORT-002 | Custom dashboards and gadgets | V12 | `planned` | Users compose permission-aware dashboards from configurable gadgets |
| REPORT-003 | Cross-project analytics and exports | V12 | `planned` | Reporting aggregates projects, teams, releases and custom fields |

### Collaboration, Forms And Notifications

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| COLLAB-001 | In-app notifications and mentions | V2 | `complete` | Users receive and manage assignment, comment, sprint and automation events |
| COLLAB-002 | Rich text editor, attachments and media | V13 | `planned` | Rich descriptions/comments and secure local object storage work end-to-end |
| COLLAB-003 | Watchers, votes, reactions and sharing | V13 | `planned` | Users follow, endorse and share work with permission-aware updates |
| COLLAB-004 | Forms and structured request intake | V13 | `planned` | Configurable forms create or update work items with validation |
| COLLAB-005 | Email and real-time notifications | V13 | `planned` | Preferences drive durable email and live in-app updates |
| COLLAB-006 | Calendar, list, timeline and reusable views | V13 | `planned` | Teams create, configure, share and permission multiple work views |

### Automation And Extensibility

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| AUTO-001 | Project synchronous automation | V4 | `complete` | Typed project rules run atomically, single-pass, with activity and notifications |
| AUTO-002 | Organization/global automation and templates | V14 | `planned` | Reusable rules span projects and support scoped administration |
| AUTO-003 | Scheduled, webhook and service triggers | V14 | `planned` | Rules support time, inbound event and service-management triggers |
| AUTO-004 | Branching, smart values and audit logs | V14 | `planned` | Rules process related entities with expressions and explainable execution logs |
| DEV-001 | Stable authenticated REST API and API tokens | V15 | `planned` | External clients use documented scoped tokens and versioned APIs |
| DEV-002 | Webhooks and integration administration | V15 | `planned` | Signed outbound webhooks have retries, diagnostics and permission controls |
| DEV-003 | GitHub, GitLab and Slack integrations | V15 | `planned` | Optional real integrations and localhost mocks cover key collaboration flows |
| DEV-004 | Extension points without Marketplace parity | V15 | `planned` | Documented internal extension contracts support selected custom integrations |

### Service Management

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| JSM-001 | Customer portal and request types | V16 | `planned` | Customers submit and track requests through configurable portals |
| JSM-002 | Queues, agent workspace and customer permissions | V16 | `planned` | Agents triage permission-aware requests with configurable queues |
| JSM-003 | SLAs, calendars and breach management | V16 | `planned` | SLA goals track working calendars, pause states and breaches |
| JSM-004 | Approvals, organizations and request participants | V16 | `planned` | Requests support approval chains and customer collaboration |
| JSM-005 | Incident and major incident management | V17 | `planned` | Teams coordinate incidents, responders, timelines and post-incident review |
| JSM-006 | Problem and change management | V17 | `planned` | Problems, changes, risk, approvals and related incidents are managed end-to-end |
| JSM-007 | Services, on-call schedules and escalation | V17 | `planned` | Services map ownership and route alerts through schedules and escalations |

### Assets And Knowledge

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| ASSET-001 | Object schemas, types, attributes and references | V18 | `planned` | Admins model typed assets and relationships with validation |
| ASSET-002 | Object CRUD, history, roles and AQL-equivalent search | V18 | `planned` | Users manage, audit and query assets under granular permissions |
| ASSET-003 | Asset imports, mappings and scheduled synchronization | V18 | `planned` | CSV/JSON/API imports validate, reconcile and record execution history |
| ASSET-004 | Link assets, services and configuration items to work | V18 | `planned` | Requests, incidents and changes use live asset context |
| KNOW-001 | Knowledge spaces, articles and permissions | V19 | `planned` | Users author, review, publish and permission knowledge articles |
| KNOW-002 | Portal knowledge search and deflection | V19 | `planned` | Request intake suggests relevant articles and measures deflection |
| KNOW-003 | Article lifecycle, feedback and analytics | V19 | `planned` | Knowledge owners manage reviews, feedback and usefulness metrics |

### Product Discovery

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| JPD-001 | Ideas, hierarchies, archive and merge | V20 | `planned` | Product teams capture, organize, merge and restore ideas |
| JPD-002 | Insights, feedback and delivery links | V20 | `planned` | Ideas collect evidence and connect to delivery work and progress |
| JPD-003 | Product fields, formulas and prioritization | V20 | `planned` | Typed fields and formulas support transparent prioritization |
| JPD-004 | List, board, matrix, timeline and roadmap views | V20 | `planned` | Product teams configure, share and publish multiple discovery views |
| JPD-005 | Discovery permissions, import and export | V20 | `planned` | Contributors/stakeholders collaborate with controlled access and data exchange |

### Governance, AI And Experience

| ID | Capability | Target | Status | Acceptance summary |
|---|---|---:|---|---|
| GOV-001 | Global permission schemes and administrative controls | V21 | `planned` | Organization/workspace/project governance is centrally configurable |
| GOV-002 | Audit log, retention and compliance-ready exports | V21 | `planned` | Sensitive actions are searchable, exportable and retention-controlled |
| GOV-003 | SSO/OIDC, session controls and security policies | V21 | `planned` | Optional identity provider integration and administrative security policies work locally |
| GOV-004 | Sandboxes, configuration export/import and change safety | V21 | `planned` | Admins test, migrate and roll back configuration safely |
| AI-001 | Provider-neutral AI configuration and local mock mode | V22 | `planned` | AI features work with configured providers and deterministic localhost QA |
| AI-002 | Work summarization, drafting and natural-language actions | V22 | `planned` | Users summarize, draft and perform permission-checked actions with AI assistance |
| AI-003 | Intelligent search, triage and virtual service agent | V22 | `planned` | AI improves search, request routing and self-service without bypassing permissions |
| UX-001 | Responsive PWA and offline-safe shell | V23 | `planned` | Core flows work across desktop/mobile web with installable PWA behavior |
| UX-002 | Accessibility and keyboard navigation | V23 | `planned` | Critical flows meet documented accessibility and keyboard acceptance criteria |
| UX-003 | Localization foundation and locale-aware formatting | V23 | `planned` | UI supports translation catalogs and locale-aware dates/numbers |
| UX-004 | Large-dataset performance and scale QA | V23 | `planned` | Defined datasets and concurrency targets pass performance budgets locally |
| CLOSE-001 | Full product capability audit | V24 | `planned` | Every included capability is complete and full V1-V24 QA is green |

## 5. Version Completion Rule

A version is complete only when:

1. every capability assigned to it meets its acceptance summary;
2. its detailed plan, operations documentation and this baseline match actual
   behavior;
3. relevant unit, component, integration, smoke and browser tests pass;
4. full regression confirms completed earlier capabilities remain intact;
5. this matrix is updated in the final QA commit.

The project is functionally closed only after `CLOSE-001` is complete and no
included capability remains `planned`, `in_progress` or `deferred`.

The root `README.md` is the final-state product cover and does not track
per-version implementation status. V24 closure includes a final reconciliation
of that cover with the completed capability matrix.

## 6. Baseline Governance

This matrix is the sole source of implementation status for Kelmio. Capability
scope changes require an explicit version target, acceptance summary and QA
evidence. V24 performs the final reconciliation between this baseline, the
product roadmap, operational documentation and actual behavior.
