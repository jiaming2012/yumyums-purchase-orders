# Milestones

## v2.1 Onboarding Video Upgrade (Shipped: 2026-04-20)

**Phases completed:** 2 phases, 6 plans, 10 tasks

**Key accomplishments:**

- Go backend for video upload (presigned PUT URLs), FFmpeg conversion + thumbnail extraction, and watch-position persistence — extending ob_progress and ob_video_parts schemas
- XHR-backed video upload UI in the onboarding builder — upload/URL toggle, progress bar, and custom thumbnail override per video part
- One-liner:
- workflows.html page scaffold with MOCK_TEMPLATES data model, template list/editor navigation, section CRUD, and state-first render architecture using event delegation
- Full field CRUD with inline type picker, expand/collapse settings panels, temperature min/max config, and photo required toggle — all wired via state-first mutations and existing event delegation
- SortableJS drag-to-reorder, Mon-first day-of-week chips on sections and fields, and two-dropdown skip logic editor — all wired via state-first mutations and existing event delegation, completing all BLDR requirements

---

## v2.0 Backend (Shipped: 2026-04-19)

**Phases completed:** 7 phases, 31 plans, 35 tasks

**Key accomplishments:**

- Task 1: Go Server Shell
- One-liner:
- httpOnly cookie session auth with bcrypt, opaque tokens, chi middleware, login/logout handlers, and /me endpoint — superadmin bootstrapped from YAML on startup
- login.html wired to POST /api/v1/auth/login with httpOnly cookie redirect; index.html 401 guard; seed script sets superadmin test password via bcrypt; SW bumped to v48
- Postgres schema for the full workflows domain — 9 goose migrations (templates through rejections), Go model structs, and YAML-driven template seed infrastructure
- 1. [Rule 1 - Bug] Used errors.Is instead of os.IsNotExist for wrapped error
- workflows.html rewritten from mock data to live API — zero MOCK_ constants remain, all 3 tabs load from Go REST endpoints with skeleton screens, auto-save, and inline error retry
- IndexedDB offline queue with drain-on-reconnect, persistent sync banner, and per-item pending badges — full offline submit → queue → online → drain cycle implemented
- Playwright test suite rewritten against real Go server — 54 mock-data tests deleted, 15 new full-stack E2E tests covering Builder CRUD, checklist fill/submit, approval flow, offline sync, and access control
- Real-time delivery pipeline: pgxlisten on ops_channel fans out to per-user WebSocket clients via channel-based hub, with /ops/since catch-up for reconnecting devices
- One-liner:
- LamportClock + WebSocket client with drain-before-catchup, applyOp state-first rendering, and 409 conflict revert — all wired into workflows.html
- Blue flash animation (@keyframes sync-flash) and 500ms-batched grouped toast (enqueueSyncToast/flushSyncToast) for real-time field sync visibility in light and dark mode
- One-liner:
- workflows.html refactored to use sync.js Store for all state — globals replaced by store collections, autoSaveField replaced by debouncedSaveField via submitOp, 5 store.on subscribers drive reactive re-renders
- One-liner:
- 0017_users_naming.sql
- One-liner:
- users.html fully API-backed with invite/edit/access management, and login.html extended with token-based set-password flow
- Task 1 — onboarding.html fully API-backed
- 23 Playwright E2E tests rewritten/created against real Go server: 11 onboarding + 12 users admin, fixing a latent parseInt(UUID) bug in users.html click handler
- One-liner:
- DO Spaces presigned PUT upload for checklist photo fields — Go backend generates URLs, browser uploads blobs directly, persistent https:// URLs stored in field responses
- One-liner:
- inventory.html — fully API-backed:
- One-liner:
- One-liner:
- One-liner:
- One-liner:

---

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
