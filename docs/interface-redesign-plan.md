# Kelmio Interface Redesign Plan

## 1. Product Goal

Before V6, rebuild the complete Kelmio frontend into a cohesive **light**
workspace driven by the supplied "Aurora" mockups. The result must feel modern,
calm and command-first while staying compact enough for daily work with
projects, issues, boards, sprints, workflows and automation.

The redesign is **behavior-preserving**. Existing routes, permissions, API
calls, controllers, database behavior and V1-V5 user flows remain unchanged.

This is the second redesign attempt. The first was dark-first and was fully
reverted; the current tree matches the pre-rebuild state. The new mockups define
a different, light visual language, so the work is restarted from the approved
mockups. The architectural approach of the previous attempt (owned UI
primitives, layered CSS, local fonts, a single icon dependency, extracting page
composition from `application-controller.tsx`) is reused.

## 2. Design Direction

- light surfaces (`#FFFFFF`, `#F7F8FB`, `#F4F6F9`, `#ECEEF3`, `#E6E9F0`) with
  hairline borders and soft shadows;
- signature accent gradient iris `#6E5CF0` -> cyan `#22C3E6` for primary actions,
  branding and progress;
- semantic colors for data only: Critical (red), Overdue (amber), Done (green),
  Info (blue);
- Space Grotesk for display, headings and numerics; Manrope for interface and
  body; monospace for issue keys, points and shortcuts;
- rounded cards, pill tags, contextual toolbars and purposeful, reduced-motion
  aware transitions;
- original Kelmio layouts, components and assets. The product is named
  **Kelmio**; "Aurora" is only the internal design-system codename and is not
  shown in the UI chrome.

The external visual reference informs principles only. No third-party assets,
branding or page structures are copied.

## 3. Scope

The redesign covers every existing screen:

- authentication, invite acceptance and password recovery;
- application shell, grouped navigation, notifications and account controls;
- a working command palette (Cmd/Ctrl+K) built on existing navigation, create
  and recent-issue actions;
- dashboard, projects, issues, board, sprints, team, labels and account;
- issue filters, creation, detail (peek slide-over), activity, comments,
  hierarchy and links;
- project members, workflow and automation settings;
- responsive desktop, tablet and mobile layouts;
- keyboard navigation, focus states, contrast and reduced motion;
- component and visual regression coverage.

The redesign does **not** add product capabilities, backend endpoints, database
migrations, a theme switcher, fabricated data/controls, PWA behavior or
localization. A full day-by-day burndown history is a V12 reporting capability
(`REPORT-001`) and is not faked here; the dashboard burndown panel uses only
real available numbers plus an ideal guide line.

## 4. Frontend Architecture

- Add local fonts and a single icon dependency:
  `@fontsource-variable/space-grotesk`, `@fontsource-variable/manrope`,
  `lucide-react`. No runtime CDN requests.
- Build Kelmio-owned primitives in `frontend/src/ui` (button, field, surface,
  tabs, overlay, feedback, icon).
- Split styling into layers under `frontend/src/styles` (tokens, reset,
  primitives, shell and per-feature layers) with an `index.css` entry. The
  legacy `frontend/src/styles.css` stays in place for not-yet-migrated screens
  and is removed in the final cleanup commit.
- Keep controllers and the API layer as the source of domain state. Extract page
  composition from `application-controller.tsx` without duplicating business
  logic.
- Preserve stable routes and existing smoke/e2e contracts.

## 5. Hard Constraints

1. Preserve accessible names and ARIA roles that Playwright e2e and component
   tests depend on (field labels, button names, `role="alert"`, etc.). Restyle
   markup and classes, but keep labels, button names and roles stable.
2. Do not change routes (`lib/routing.ts`) or existing navigation/create
   handlers; admin/lead/contributor/viewer/member permissions behave as before.
3. No new backend endpoints, migrations, PWA, localization or non-functional
   controls.
4. After every commit `npm test` and `npm run build` are green and e2e is not
   broken.

## 6. Commit Order

1. `Add Kelmio interface redesign plan` (this document + roadmap gate).
2. `Add Kelmio design system foundation` (fonts, icons, tokens, layered CSS, UI
   primitives).
3. `Rebuild authentication experience`.
4. `Rebuild application shell and navigation`.
5. `Add Kelmio command palette`.
6. `Redesign dashboard`.
7. `Redesign projects and settings`.
8. `Redesign issue list filters and creation`.
9. `Redesign issue detail and collaboration`.
10. `Redesign project and sprint boards` (board).
11. `Redesign sprints`.
12. `Redesign automations`.
13. `Redesign team labels notifications and account`.
14. `Complete responsive accessibility and motion polish` (remove legacy CSS).
15. `Finalize Kelmio interface redesign QA`.

Commits are implemented one at a time, each green before the next.

## 7. Testing Strategy

- Run frontend unit/component tests and the production build after every commit.
- Preserve all V1-V5 Playwright flows; run affected specs on the relevant
  commits and the full suite in the final QA commit.
- Add component coverage for primitives, command palette, shell states and
  permission-based controls.
- Visually compare each screen against its mockup.

## 8. Definition Of Done

- Every existing screen uses the shared light Kelmio design system.
- Legacy shell styling and inconsistent form/button implementations are gone.
- Desktop and mobile critical flows remain fully usable.
- Admin, lead, contributor, viewer and member behavior is unchanged.
- Keyboard focus, contrast and reduced-motion behavior are verified.
- Public API, database schema, migrations and backend behavior are unchanged.

## 9. Decision For Next Step

V6 planning (`docs/v6-plan.md`, `PLAT-006`/`PLAT-007`) starts only after the
final interface redesign QA commit is green. Capability statuses in the
[capability baseline](product-capability-baseline.md) are unchanged by this
redesign.

## 10. Completion Status

The redesign is complete. Every screen runs on the light Kelmio design system,
the legacy `styles.css` has been removed, and the production CSS bundle dropped
from ~115 KB to ~67 KB.

Final QA gate:

- Frontend unit/component tests green (`npm test`, 32 files / 139 tests).
- Production build green (`tsc --noEmit && vite build`).
- Full Playwright suite green against a local stack (all V1-V5 specs). The gate
  also caught and fixed CSS-class selector contracts that several specs locate
  by (`.issue-filters`, `.project-form`, `.project-row`, `.project-member-card`,
  `.saved-filter-card`, `.workflow-status-card`) and restored the notification
  quick dropdown (`.notification-toggle` / `.notification-badge` / the
  "Notification dropdown" region) inside the new sidebar.
- Backend, database schema, migrations and API behavior unchanged; `go test`
  remains green in CI.

V6 planning may now begin.
