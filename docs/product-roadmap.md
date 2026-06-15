# Team Task Tracker Product Roadmap

## 1. Product Direction

V1-V4 сформировали стабильную основу бесплатного task tracker для небольших
команд:

- V1 закрыла базовый task tracking;
- V2 добавила planning workflow;
- V3 подготовила production-ready self-hosted foundation;
- V4 добавила project workflows, project roles и automation.

Следующая цель продукта: последовательно закрыть обязательные возможности
Jira-подобного продукта для команд примерно до 50-100 пользователей, не
копируя весь Jira Cloud, Jira Service Management или Atlassian Marketplace.

До полного закрытия продукта запланированы еще шесть версий: V5-V10.
Ориентировочная оценка: 80-120 небольших проверяемых коммитов.

V5-V9 разрабатываются, тестируются и полностью проверяются на localhost.
Hosting provider, реальный домен, production deployment и pilot выбираются и
выполняются только в V10 на основе фактических требований готового продукта.

## 2. Product Principles

- Go backend, React + TypeScript frontend, PostgreSQL и Docker остаются
  обязательной технологической основой.
- Modular monolith сохраняется, пока реальная нагрузка не докажет необходимость
  другого подхода.
- Каждая версия имеет отдельный planning document, небольшие коммиты и финальный
  QA release-gate.
- Новые возможности не должны ломать завершенные V1-V4 workflows.
- Бесплатный self-hosted сценарий остается основным направлением продукта.
- До V10 не добавляется hosting-specific runtime или deployment configuration.
- Реальная эксплуатация не объявляется готовой без проверенных backup, restore,
  upgrade, rollback, monitoring и security procedures.

## 3. V5: Account Recovery And Operations Foundation

Цель V5: закрыть недостающие account recovery и operational foundations,
которые можно надежно разработать и проверить локально до реального deployment.

Основные результаты:

- generic SMTP configuration и локальная Mailpit-среда;
- durable email outbox и отдельный delivery worker;
- password reset API и UI;
- invite email delivery и resend flow;
- email delivery diagnostics;
- Prometheus application metrics;
- локальные Prometheus, Grafana и Alertmanager;
- scheduled backup runner, retention и автоматический restore drill;
- полный V1-V5 regression и operations QA.

После V5 приложение имеет проверенные механизмы восстановления доступа,
доставки системных писем и локально воспроизводимый operations baseline.

## 4. V6: Flexible Work Items And Search

Цель V6: сделать модель задач и поиск сопоставимыми с основным Jira workflow.

Основные результаты:

- custom fields и настраиваемые issue layouts;
- components и releases/versions;
- issue templates;
- advanced search/query builder;
- bulk issue operations;
- CSV import/export;
- сохранение совместимости workflow, automation и permissions.

После V6 команды могут адаптировать задачи под собственные процессы и работать
с большими наборами данных без ручного редактирования каждой задачи.

## 5. V7: Planning And Reports

Цель V7: закрыть planning, estimation и reporting для небольшой engineering
команды.

Основные результаты:

- timeline/roadmap view;
- calendar view;
- burndown, burnup, velocity и cumulative flow reports;
- sprint reports;
- capacity planning;
- расширенная dashboard analytics;
- time estimates и базовый time tracking.

После V7 продукт поддерживает регулярное планирование, прогнозирование и анализ
результатов команды.

## 6. V8: Collaboration And Integrations

Цель V8: встроить task tracker в ежедневную работу команды и внешние developer
workflows.

Основные результаты:

- file attachments через локальное или S3-compatible object storage;
- watchers;
- email notifications;
- real-time updates;
- API tokens;
- webhooks;
- базовые GitHub/GitLab и Slack integrations.

После V8 пользователи получают необходимые события и могут связывать задачи с
основными collaboration и development tools.

## 7. V9: Administration Security And Scale Readiness

Цель V9: подготовить продукт к безопасной эксплуатации несколькими небольшими
командами и более серьезным deployment-сценариям.

Основные результаты:

- workspace audit log;
- расширенные admin tools;
- более детальные permissions и security controls;
- optional OIDC/SSO;
- distributed-ready sessions и rate limits;
- external PostgreSQL compatibility;
- load testing и performance budgets;
- data retention и disaster-recovery procedures.

После V9 продукт имеет проверенную administrative, security и scale readiness,
но реальный deployment и pilot все еще остаются задачей V10.

## 8. V10: Deployment Pilot And Final Product Closure

Цель V10: выбрать подходящую hosting platform, развернуть готовый продукт и
подтвердить его качество в реальной эксплуатации.

Основные результаты:

- выбор hosting provider по фактическим runtime, database, storage, monitoring и
  budget requirements;
- real domain и production deployment;
- deploy/update/rollback automation;
- scheduled production backups и restore drill;
- production monitoring, alerts и incident workflow;
- responsive/PWA polish;
- accessibility и keyboard navigation;
- localization foundation;
- large-dataset performance profiling;
- полный security и dependency audit;
- pilot минимум с 2-3 реальными командами;
- финальные operations, support, upgrade и rollback guides;
- полный V1-V10 regression и production QA.

Только после V10 продукт может быть объявлен полностью завершенной
Jira-альтернативой для небольших команд.

## 9. Definition Of Project Closure

Проект считается функционально закрытым, когда выполнены все условия:

1. V1-V10 Definition of Done полностью выполнены.
2. Приложение работает на реальном сервере и домене.
3. Минимум 2-3 реальные команды прошли pilot без известных blocker bugs.
4. Основные issue, workflow, planning, reporting и integration workflows
   подтверждены реальными пользователями.
5. Production monitoring и alerts проверены контролируемым incident drill.
6. Backup, restore, upgrade и rollback регулярно воспроизводятся.
7. Security, dependency, accessibility и performance audits не имеют
   незакрытых blocker/high-severity проблем.
8. Полный automated и manual V1-V10 QA зеленый.
9. Документация позволяет новому администратору развернуть, обновить,
   диагностировать и восстановить приложение.
10. Дальнейшая работа преимущественно состоит из поддержки, исправления багов и
    необязательных улучшений, а не разработки обязательных функций.

## 10. Explicit Product Boundary

Даже после V10 проект не обязан копировать:

- весь Jira Cloud;
- Jira Service Management;
- Atlassian Marketplace;
- enterprise compliance для регулируемых отраслей;
- SaaS billing;
- managed multi-tenant cloud;
- AI/Rovo;
- неограниченную enterprise scalability.

Эти направления могут быть запланированы только после полного закрытия
основного продукта и подтвержденной потребности реальных пользователей.

## 11. Decision For Next Step

Следующий этап: выполнить `docs/v5-plan.md` по одному небольшому проверяемому
коммиту. До V10 все новые возможности должны оставаться полностью
воспроизводимыми и проверяемыми на localhost.
