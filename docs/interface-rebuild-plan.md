# Kelmio Interface Rebuild Plan

## 1. Product Goal

Before V6, rebuild the complete Kelmio frontend into a cohesive dark-first
enterprise workspace. The result must feel modern and distinctive while
remaining compact enough for daily work with projects, issues, boards,
sprints, workflows and automation.

The rebuild is behavior-preserving. Existing routes, permissions, API calls,
database behavior and V1-V5 user flows remain unchanged.

## 2. Design Direction

- dark graphite canvas with elevated charcoal surfaces;
- Kelmio mint as the primary accent, warm amber as a secondary accent;
- expressive display typography with dense, highly legible work surfaces;
- subtle gradients, hairline borders and restrained depth instead of heavy
  glass effects;
- purposeful motion for navigation and state changes with reduced-motion
  support;
- original Kelmio layouts, components, branding and visual assets.

The external visual reference informs principles only. No third-party assets,
branding or page structures are copied.

## 3. Scope

The rebuild covers:

- authentication, invite acceptance and password recovery;
- application shell, navigation, notifications and account controls;
- dashboard, projects, issues, board, sprints, team, labels and account;
- issue filters, creation, detail, activity, comments, hierarchy and links;
- project members, workflow and automation settings;
- desktop, tablet and mobile layouts;
- keyboard navigation, focus states, contrast and reduced motion;
- component, responsive and visual regression coverage.

The rebuild does not add product capabilities, backend endpoints, database
migrations, fake search controls, PWA behavior or localization.

## 4. Frontend Architecture

- Build Kelmio-owned primitives in `frontend/src/ui`.
- Split global styling into token, reset, primitive, shell and feature layers.
- Use local Space Grotesk and Manrope fonts without runtime CDN requests.
- Use `lucide-react` as the only icon dependency.
- Keep controllers and the API layer as the source of domain state.
- Extract page composition from `application-controller.tsx` without
  duplicating business logic.
- Preserve stable routes, accessible names and existing smoke/e2e contracts.
- Replace browser confirmations with an accessible Kelmio confirmation dialog.

## 5. Main Experience

Desktop uses a collapsible grouped sidebar, compact contextual topbar and
high-density work surfaces. Mobile uses a dedicated header and navigation
drawer. Boards remain horizontally scrollable workspaces rather than being
collapsed into card feeds.

List-heavy areas use consistent filters, toolbars, empty states and list/detail
composition. Manager-only areas retain existing permission rules and clearly
communicate read-only states.

## 6. Development Phases

1. Document the interface rebuild and roadmap gate.
2. Add fonts, tokens, icons and reusable UI primitives.
3. Rebuild authentication and public account-recovery screens.
4. Rebuild the application shell and navigation.
5. Redesign dashboard, notifications and account.
6. Redesign projects, team and labels.
7. Redesign issue lists, filters and creation.
8. Redesign issue details and collaboration.
9. Redesign project and sprint boards.
10. Redesign project members, workflow and automation settings.
11. Complete responsive, accessibility and motion polish.
12. Add stable visual regression coverage.
13. Run the final interface rebuild QA gate.

## 7. Testing Strategy

- Run frontend unit/component tests and production build after every phase.
- Preserve all V1-V5 Playwright flows.
- Add component coverage for primitives, shell states, dialogs, mobile
  navigation, notifications and permission-based controls.
- Verify desktop at 1440x900, tablet at 1024x768 and mobile at 390x844.
- Add deterministic visual baselines for auth, dashboard, issues, board and
  project settings.
- Run full repository verify, smoke, integration and Full QA checks before
  declaring the rebuild complete.

## 8. Definition Of Done

- Every existing screen uses the shared Kelmio design system.
- Legacy shell styling and inconsistent form/button implementations are gone.
- Desktop and mobile critical flows remain fully usable.
- Admin, lead, contributor, viewer and member permission behavior is unchanged.
- Keyboard focus, contrast and reduced-motion behavior are verified.
- Public API, database schema, migrations and backend behavior are unchanged.
- V6 planning starts only after the final interface rebuild QA is green.

## 9. Proposed Commit Order

1. `Add Kelmio interface rebuild plan`
2. `Add Kelmio design system foundation`
3. `Rebuild authentication experience`
4. `Rebuild application shell and navigation`
5. `Redesign dashboard notifications and account`
6. `Redesign projects team and labels`
7. `Redesign issue lists filters and creation`
8. `Redesign issue details and collaboration`
9. `Redesign project and sprint boards`
10. `Redesign project management settings`
11. `Complete responsive accessibility and motion polish`
12. `Add interface visual regression coverage`
13. `Finalize Kelmio interface rebuild QA`

## 10. Decision For Next Step

Implement `Add Kelmio design system foundation`. V6 scope and capability
assignments remain unchanged until the rebuild passes its final QA gate.
