# Kelmio Product Roadmap

## 1. Product Direction

V1-V5 сформировали стабильную основу: task tracking, agile planning,
production-like self-hosted foundation, project workflows, project permissions,
synchronous automation, account recovery и localhost operations.

Долгосрочная цель Kelmio: создать оригинальную localhost-only платформу,
которая объединяет work management, agile delivery, service operations,
product discovery, assets, reports, administration и provider-neutral AI.

Целевой scope включает конфигурируемые work items и workflows, planning,
service management, product discovery, Assets, automation, reports, forms,
administration и provider-neutral AI assistance.

Kelmio сохраняет собственные название, интерфейс, архитектуру, product model
и исходный код.

До полного закрытия продукта запланированы V5-V24. Ориентировочная оценка:
**300-450 небольших проверяемых коммитов**. Полнота измеряется через
[product capability baseline](product-capability-baseline.md).

## 2. Product Principles

- Go backend, React + TypeScript frontend, PostgreSQL и Docker остаются
  обязательной технологической основой.
- Modular monolith сохраняется, пока измеримая сложность не докажет
  необходимость выделения runtime-компонента.
- Приложение и весь release QA остаются воспроизводимыми на localhost.
- Production-like Compose используется для security, reliability и operations
  QA, а не как обязательство реального deployment.
- Каждая версия имеет отдельный planning document, небольшие коммиты и финальный
  QA release-gate.
- Каждый version plan ссылается на capability IDs из baseline.
- Новые возможности не должны ломать завершенные V1-V5 workflows.
- Optional внешние integrations поддерживают реальные credentials, но имеют
  локальные mocks для полного QA.
- AI capabilities остаются provider-neutral и должны иметь детерминированный
  localhost test mode.

### Interface Rebuild Gate Before V6

После завершения V5 и до планирования V6 Kelmio проходит отдельный
behavior-preserving rebuild всего frontend. Этот этап не является новой
product version, не сдвигает V6-V24 и не закрывает будущие PWA, localization
или formal accessibility capabilities. Его scope и release gate определены в
[Kelmio Interface Rebuild Plan](interface-rebuild-plan.md).

### README Policy

- Root `README.md` является final-state product cover, а не источником текущего
  implementation status.
- Фактическое состояние capabilities определяется только
  [capability baseline](product-capability-baseline.md) и version plans.
- V6-V23 не включают отдельные README-update commits; они обновляют baseline,
  version documentation и соответствующие operations guides.
- V24 final parity audit сверяет README с завершенным продуктом и исправляет
  только фактические, визуальные или legal расхождения.

## 3. Product Boundary

### Входит в целевой функциональный паритет

- configurable work management и agile delivery;
- service management и customer self-service;
- product discovery и portfolio planning;
- Assets и configuration management;
- multi-organization и multi-workspace administration;
- significant built-in administration, automation, reporting, forms и AI
  scenarios;
- responsive PWA вместо отдельных native mobile applications.

### Не входит

- отдельные products для document collaboration, team chat, video и source hosting;
- паритет со всеми сторонними marketplace applications;
- копирование чужого branding, source code и визуального интерфейса;
- SaaS billing, managed multi-tenant cloud и real hosting provider;
- public deployment и production pilot;
- public-cloud infrastructure scale, external data residency и formal compliance certifications;
- capabilities, которые не были явно приняты в product baseline.

## 4. V5: Account Recovery And Operations Foundation

Цель V5: закрыть account recovery и надежную локальную operations foundation.

Статус: полностью завершена 22 июня 2026 года. Operations и email runbooks:
[Email Delivery And Account Recovery](email-delivery.md) и
[Local Operations](local-operations.md).

Основные результаты:

- generic SMTP configuration и Mailpit;
- durable email outbox и delivery worker;
- password reset и invite email delivery;
- email diagnostics;
- Prometheus metrics, Grafana и Alertmanager;
- scheduled backups, retention и automated restore drill.

Capability groups: `PLAT-004`, `PLAT-005`.

## 5. V6: Organizations, Multi-Workspace And Identity Administration

Цель V6: превратить single-workspace foundation в локальную multi-organization
platform с несколькими изолированными организациями и workspaces.

Основные результаты:

- organization и workspace lifecycle;
- global, organization и workspace administration;
- groups, directories и reusable role assignments;
- workspace switching и isolation;
- safe migration существующего workspace;
- organization-aware seed, backup и audit foundation.

Capability groups: `PLAT-006`, `PLAT-007`.

## 6. V7: Configurable Work Items, Fields And Schemes

Цель V7: сделать модель work items и workflows полностью конфигурируемой.

Основные результаты:

- configurable work types и hierarchy schemes;
- typed custom fields и field contexts;
- create/edit/view screens и layouts;
- field configuration schemes;
- reusable workflow schemes;
- transition conditions, validators и post-functions.

Capability groups: `WORK-004`-`WORK-007`.

## 7. V8: Advanced Search, Filters, Bulk Operations And Data Exchange

Цель V8: обеспечить эффективную работу с большими наборами work items.

Основные результаты:

- advanced query language и visual query builder;
- shared filters, permissions и subscriptions;
- bulk edit, transition, move, archive и delete;
- CSV import/export и field mapping;
- saved searches для новых configurable fields;
- search performance и large-result QA.

Capability groups: `SEARCH-002`-`SEARCH-005`.

## 8. V9: Advanced Boards, Backlogs And Agile Configuration

Цель V9: закрыть configurable Scrum и Kanban workflows.

Основные результаты:

- несколько boards на основе saved queries;
- configurable columns, swimlanes, quick filters и card layouts;
- issue ranking;
- WIP limits;
- Kanban backlog;
- board estimation и working-day configuration.

Capability groups: `AGILE-003`-`AGILE-005`.

## 9. V10: Releases, Components And Delivery Tracking

Цель V10: связать планирование work items с release и engineering delivery.

Основные результаты:

- components и component ownership;
- versions, releases и release notes;
- release readiness и unresolved-work checks;
- development information для commits, branches и pull requests;
- build и deployment status;
- provider-neutral delivery integration contracts.

Capability groups: `AGILE-006`, `AGILE-007`.

## 10. V11: Plans, Goals, Capacity And Portfolio Planning

Цель V11: добавить cross-project и portfolio-level planning.

Основные результаты:

- cross-project plans и dependency timeline;
- initiatives и configurable hierarchy levels;
- goals и outcome alignment;
- teams, capacity и estimates;
- scenario planning;
- plan permissions, saved views и exports.

Capability groups: `PLAN-002`-`PLAN-004`.

## 11. V12: Reports, Dashboards And Analytics

Цель V12: закрыть advanced reporting и decision support.

Основные результаты:

- burndown, burnup, velocity, sprint и cumulative-flow reports;
- control, cycle-time и release reports;
- configurable dashboards и gadgets;
- cross-project analytics;
- scheduled report exports;
- permission-aware reporting datasets.

Capability groups: `REPORT-001`-`REPORT-003`.

## 12. V13: Collaboration, Rich Content, Forms And Notifications

Цель V13: сделать приложение полноценной collaboration platform.

Основные результаты:

- rich text, attachments и media;
- watchers, votes, reactions и sharing;
- configurable forms;
- durable email notifications и user preferences;
- real-time updates;
- reusable list, board, calendar и timeline views.

Capability groups: `PLAT-008`, `COLLAB-002`-`COLLAB-006`.

## 13. V14: Enterprise Automation And Rule Platform

Цель V14: расширить project automation до reusable organization-level rule
platform.

Основные результаты:

- organization/global rules и templates;
- scheduled, webhook и service triggers;
- branching и related-entity actions;
- smart values и expressions;
- execution audit log, diagnostics и limits;
- safe rule migration и conflict handling.

Capability groups: `AUTO-002`-`AUTO-004`.

## 14. V15: Developer Platform, APIs, Webhooks And Integrations

Цель V15: предоставить стабильный integration и extension surface.

Основные результаты:

- scoped API tokens;
- documented versioned REST API;
- signed webhooks с retries и diagnostics;
- GitHub, GitLab и Slack integrations;
- localhost integration mocks;
- selected extension contracts без Marketplace parity.

Capability groups: `DEV-001`-`DEV-004`.

## 15. V16: Service Management Request Portal, Queues And SLAs

Цель V16: реализовать основу service management.

Основные результаты:

- customer portal и request types;
- configurable request forms;
- agent workspace и queues;
- customers, organizations и request participants;
- approvals;
- SLA goals, calendars, pause states и breach handling.

Capability groups: `JSM-001`-`JSM-004`.

## 16. V17: Service Management Incidents, Problems, Changes And On-Call

Цель V17: закрыть основные ITSM operational workflows.

Основные результаты:

- incident и major incident management;
- responders, timelines и post-incident reviews;
- problem management и known workarounds;
- change management, risk и approvals;
- services и ownership;
- on-call schedules, alerts и escalation policies.

Capability groups: `JSM-005`-`JSM-007`.

## 17. V18: Assets And Configuration Management

Цель V18: реализовать Assets-equivalent для ITAM и configuration context.

Основные результаты:

- object schemas, types, attributes и references;
- object lifecycle, history и permissions;
- AQL-equivalent search;
- import mappings и scheduled synchronization;
- links между assets, services, requests, incidents и changes;
- asset reports и dashboards.

Capability groups: `ASSET-001`-`ASSET-004`.

## 18. V19: Knowledge Management And Self-Service

Цель V19: добавить встроенную knowledge base и service self-help.

Основные результаты:

- knowledge spaces и article lifecycle;
- authoring, review, publishing и permissions;
- portal search и article suggestions;
- request deflection;
- feedback и knowledge analytics;
- links между knowledge, requests, incidents и problems.

Capability groups: `KNOW-001`-`KNOW-003`.

## 19. V20: Product Discovery, Ideas, Insights And Roadmaps

Цель V20: закрыть product discovery workflows.

Основные результаты:

- ideas, hierarchies, archive и merge;
- insights, feedback и delivery links;
- custom fields, formulas и prioritization;
- list, board, matrix, timeline и roadmap views;
- view publishing и stakeholder permissions;
- discovery import/export.

Capability groups: `JPD-001`-`JPD-005`.

## 20. V21: Governance, Security, Audit And Administration

Цель V21: завершить platform-wide administrative и governance controls.

Основные результаты:

- global permissions и reusable permission schemes;
- issue security levels;
- organization audit log;
- retention и compliance-ready exports;
- optional OIDC/SSO и session policies;
- sandboxes, configuration export/import и safe change workflows.

Capability groups: `WORK-008`, `GOV-001`-`GOV-004`.

## 21. V22: AI Assistance, Intelligent Search And Virtual Agent

Цель V22: предоставить provider-neutral AI assistance для значимых product scenarios.

Основные результаты:

- AI provider configuration и deterministic localhost mock mode;
- work summarization и drafting;
- natural-language actions с permission checks;
- cross-domain intelligent search;
- assisted triage и prioritization;
- virtual service agent и knowledge-assisted self-service.

Capability groups: `SEARCH-006`, `AI-001`-`AI-003`.

## 22. V23: PWA, Accessibility, Localization, Performance And Scale QA

Цель V23: завершить пользовательское качество и доказать устойчивость полного
локального продукта.

Основные результаты:

- responsive installable PWA;
- accessibility и keyboard navigation;
- localization foundation;
- locale-aware formatting;
- large-dataset profiling и performance budgets;
- concurrency, recovery и long-running stability QA.

Capability groups: `UX-001`-`UX-004`.

## 23. V24: Final Product Capability Audit And Closure

Цель V24: доказать полноту выбранного capability baseline и закрыть
проектную разработку.

Основные результаты:

- audit каждой included capability;
- устранение blocker/polish defects без расширения snapshot;
- полный V1-V24 automated и manual regression;
- security, dependency, accessibility и performance audit;
- финальные localhost operations, upgrade, rollback и recovery guides;
- зафиксированный capability baseline без незакрытых included entries.

Capability group: `CLOSE-001`.

## 24. Definition Of Project Closure

Проект считается функционально закрытым после V24, когда:

1. Все included capabilities baseline имеют статус `complete`.
2. Нет capability со статусом `planned`, `in_progress` или `deferred`.
3. V1-V24 Definition of Done полностью выполнены.
4. Полный automated и manual V1-V24 QA зеленый.
5. Backup, restore, upgrade, rollback и recovery локально воспроизводимы.
6. Security, dependency, accessibility и performance audits не имеют
   незакрытых blocker/high-severity проблем.
7. Optional реальные integrations имеют рабочие adapters и localhost mocks.
8. AI capabilities работают provider-neutral и имеют deterministic test mode.
9. Документация соответствует фактическим setup, administration и workflows.
10. Нет известных blocker bugs во включенном product baseline.

Реальный deployment, hosting provider, public domain и production pilot не
являются условиями закрытия проекта.

## 25. Decision For Next Step

V5 завершена. Следующий этап должен начаться с отдельного `docs/v6-plan.md`,
который детализирует capabilities `PLAT-006` и `PLAT-007` до начала runtime
изменений.
