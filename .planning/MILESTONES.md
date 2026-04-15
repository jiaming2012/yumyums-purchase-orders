# Milestones

## v1.1 Inventory App (Shipped: 2026-04-15)

**Phases completed:** 3 phases, 6 plans, 13 tasks

**Key accomplishments:**

- One-liner:
- renderHistory() with event cards sorted newest-first, expandable line items showing name/qty/price/CASE badge, vendor dropdown filter, and 18 Playwright E2E tests covering all features
- 1. [Rule 1 - Bug] MOCK_SALES quantities scaled to ingredient-unit purchase scale
- Chart.js horizontal bar, doughnut, and monthly line charts with multi-select tag filter chips across two sub-tabs (By Category / Over Time) replacing the Trends Coming Soon placeholder
- Cost Intelligence tab with menu item cost breakdown (ingredient proportions + margin) and ingredient reverse-lookup (usage by menu item with ROI), plus full Playwright test coverage for TRND-01 through COST-03

---

## v1.0 Operations Console MVP (Shipped: 2026-04-14)

**Phases completed:** 5 phases, 12 plans, 17 tasks

**Key accomplishments:**

- workflows.html page scaffold with MOCK_TEMPLATES data model, template list/editor navigation, section CRUD, and state-first render architecture using event delegation
- Full field CRUD with inline type picker, expand/collapse settings panels, temperature min/max config, and photo required toggle — all wired via state-first mutations and existing event delegation
- SortableJS drag-to-reorder, Mon-first day-of-week chips on sections and fields, and two-dropdown skip logic editor — all wired via state-first mutations and existing event delegation, completing all BLDR requirements
- Crew fill-out experience: today's checklist list filtered by day-of-week, runner view with all non-photo field types (checkbox, yes/no, text, temperature), user attribution, and live progress tracking
- One-liner:
- iOS-safe photo capture with two-step preview modal, blob-URL thumbnail display, and corrective action card evidence photos — all wired via event delegation, no disabled buttons remaining.
- Three-tab nav (My Checklists / Approvals / Builder) with manager approval flow — one-tap approve, two-step reject with required reason — two pre-built food truck templates, and role-gated tab access.
- Crew-facing onboarding HTML page with sequential section state machine, checkbox+video-series items, FAQ gate, and sign-off request flow — all event-delegated and state-first rendered
- Manager tab for onboarding.html — read-only hire checklist drill-down, Active/Completed sub-views, sign-off approve/send-back actions, and inline training assignment via shared renderRunnerContent(readOnly) refactor
- Onboarding tile activated in HQ launcher, onboarding.html added to SW cache (v40), and onboarding registered in Users permission APPS — human-verified end-to-end across all 19 steps.
- One-liner:
- Full onboarding builder human-verified and approved: checkbox/sub-item editing, video series multi-part editor, FAQ Q&A, SortableJS drag-to-reorder, and 35 E2E tests

---

## v1.0 Workflow Engine MVP (Shipped: 2026-04-13)

**Phases completed:** 3 phases, 7 plans, 10 tasks

**Key accomplishments:**

- workflows.html page scaffold with MOCK_TEMPLATES data model, template list/editor navigation, section CRUD, and state-first render architecture using event delegation
- Full field CRUD with inline type picker, expand/collapse settings panels, temperature min/max config, and photo required toggle — all wired via state-first mutations and existing event delegation
- SortableJS drag-to-reorder, Mon-first day-of-week chips on sections and fields, and two-dropdown skip logic editor — all wired via state-first mutations and existing event delegation, completing all BLDR requirements
- Crew fill-out experience: today's checklist list filtered by day-of-week, runner view with all non-photo field types (checkbox, yes/no, text, temperature), user attribution, and live progress tracking
- One-liner:
- iOS-safe photo capture with two-step preview modal, blob-URL thumbnail display, and corrective action card evidence photos — all wired via event delegation, no disabled buttons remaining.
- Three-tab nav (My Checklists / Approvals / Builder) with manager approval flow — one-tap approve, two-step reject with required reason — two pre-built food truck templates, and role-gated tab access.

---
