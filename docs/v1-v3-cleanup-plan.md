# Team Task Tracker V1-V3 Cleanup Plan

## 1. Goal

V1, V2, and V3 are functionally complete. This cleanup closes the remaining
maintainability and verification debt before V4 planning begins.

The cleanup is behavior-preserving:

- no new product features;
- no public API changes;
- no database schema or migration changes;
- no permission or workflow changes;
- no replacement state-management framework.

## 2. Audit Findings

The final V1-V3 audit found no blocker bugs. The complete local and production-like
QA baseline passed, but these maintainability gaps remain:

- `frontend/src/App.tsx` owns too much state, effect, and action orchestration;
- key frontend forms and role-dependent screens lack direct component tests;
- large issue and sprint backend handlers mix HTTP, validation, query, and domain concerns;
- the fast GitHub Actions workflow does not run the full integration, browser, and
  production-stack QA baseline;
- V1 and V2 planning documents still end with obsolete implementation-start instructions.

## 3. Cleanup Work

### Frontend Test Foundation

- use Vitest as the single frontend unit/component test runner;
- add React Testing Library, user-event, and jsdom;
- keep existing helper tests;
- add component coverage for sign-in, issue creation, saved filters, sprint actions,
  notifications, and admin/member Team behavior.

### Frontend Orchestration

- reduce `App.tsx` to a small application orchestration entrypoint;
- extract typed controllers/hooks for session/account, workspace administration,
  issues, sprints, and notifications;
- use explicit typed callbacks to synchronize shared collections;
- preserve current routes, API calls, loading states, errors, and UI behavior.

### Backend Organization

- preserve the current Go packages, `Handler` types, routes, and SQL behavior;
- split issue and sprint handlers into concern-focused files;
- keep transactions and validation behavior unchanged;
- strengthen targeted tests for validation, errors, and transaction-sensitive operations.

### Full QA Automation

- keep the fast CI workflow on every push and pull request;
- add a separate full-QA workflow for manual and weekly runs;
- run development setup twice, production/API smoke, backend integration, browser e2e,
  and isolated production-stack QA;
- always clean Docker resources after the workflow.

### Documentation

- replace obsolete V1/V2 next-step sections with completion summaries;
- record the final cleanup result in V3 and README;
- start V4 planning only after the complete cleanup QA is green.

## 4. Proposed Commit Order

1. `Add V1 V2 V3 cleanup plan`
2. `Add frontend component test foundation`
3. `Extract frontend session and admin controllers`
4. `Extract frontend issue sprint notification controllers`
5. `Split issue backend handler by concern`
6. `Split sprint backend handler and strengthen tests`
7. `Add scheduled full QA workflow`
8. `Finalize V1 V2 V3 cleanup audit`

## 5. Definition Of Done

The cleanup is complete when:

1. `App.tsx` is no longer the central owner of all feature state and actions.
2. Key frontend forms and role-dependent views have direct component tests.
3. Issue and sprint backend handlers are split by concern without behavior changes.
4. Fast CI remains green and a manual/weekly full-QA workflow exists.
5. V1, V2, V3, README, and this document describe the actual completed state.
6. Two consecutive database setups, all smoke suites, backend integration tests,
   browser e2e, production Compose checks, isolated production-stack QA, race tests,
   static checks, and dependency audit pass.
7. No known V1-V3 blocker bugs remain.

## 6. Current Status

Cleanup is in progress. V4 planning must not start until the Definition of Done above
is fully verified and this status is updated to completed.
