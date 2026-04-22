# Yumyums HQ — Operations Console

## What This Is

A mobile-first PWA operations console for a food truck business. One app shell with a launcher grid linking to independent workflow tools — purchasing, user management, a workflow/checklist engine (Lumiform-style), crew onboarding/training, and inventory management. Each tool is a standalone HTML page inside a shared PWA, backed by a Go + Postgres REST API with real-time sync via WebSocket. Designed for a small crew (1-5 people) to use on their phones. Shipped v2.0 with Go backend, 28 API endpoints, real-time sync, offline support, receipt ingestion pipeline, and 108+ E2E tests.

## Core Value

Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability (who checked what), smart conditions (day-of-week, fail triggers), and structured onboarding (video training, sequential sign-off).

## Requirements

### Validated

- ✓ PWA shell with launcher grid — `index.html`
- ✓ Purchasing mockup — `purchasing.html`
- ✓ User management with roles, invitations, app access control — `users.html`
- ✓ Login screen mockup — `login.html`
- ✓ Pull-to-refresh on iOS standalone + auto-reload on deploy — `ptr.js`
- ✓ Template builder with sections, 5 field types (checkbox, yes/no, text, temperature, photo) — Phase 1
- ✓ Drag-to-reorder fields via SortableJS — Phase 1
- ✓ Day-of-week conditions on sections and fields — Phase 1
- ✓ Skip logic (show/hide based on prior answer) — Phase 1
- ✓ Temperature min/max thresholds with fail triggers — Phase 1
- ✓ Sub-steps for checkbox fields (recursive checklists) — post-Phase 3
- ✓ Assignable checklists (role/user pickers for assignees and approvers) — post-Phase 3
- ✓ Fill-out runner with day-filtered checklist list — Phase 2
- ✓ Checkbox, yes/no, text, temperature field completion — Phase 2
- ✓ User attribution (initials + timestamp) on completed items — Phase 2
- ✓ Progress tracking (per-checklist + aggregate) — Phase 2
- ✓ Inline corrective action cards on fail triggers (text + photo + severity) — Phase 2
- ✓ Conditional logic runtime (skip logic + day-of-week filtering) — Phase 2
- ✓ Photo capture via native camera with inline thumbnails — Phase 3
- ✓ Manager approval flow with 3-tab layout (My Checklists / Approvals / Builder) — Phase 3
- ✓ Item-level rejection with comments and optional photo requirements — post-Phase 3
- ✓ Approve-with-comment for incomplete checklists — post-Phase 3
- ✓ Unapprove with required reason — post-Phase 3
- ✓ Unsubmit functionality — post-Phase 3
- ✓ Correction loop (rejected items unchecked, crew re-completes with corrective action) — post-Phase 3
- ✓ Fireworks animation on fully completed submit — Phase 2
- ✓ Green checkmark animation on reapproval submit — post-Phase 3
- ✓ 2 pre-built templates (Setup Checklist, Closing Checklist) — Phase 3
- ✓ 54 Playwright E2E tests — post-Phase 3
- ✓ Onboarding app with role-based views, video training, sequential sections, FAQ gate, manager sign-off — Phase 4
- ✓ Onboarding builder with template CRUD, checkbox/sub-items, video series parts, FAQ Q&A editor, SortableJS drag-to-reorder — Phase 5
- ✓ 89 Playwright E2E tests (54 workflows + 35 onboarding) — Phase 5

- ✓ Inventory app with purchase history, vendor filter, Chart.js spending charts, stock estimation, food cost intelligence — v1.1
- ✓ 128+ Playwright E2E tests (54 workflows + 35 onboarding + 39 inventory) — v1.1

- ✓ Go backend with Postgres — REST API replacing all mock data — v2.0
- ✓ User auth + sessions — httpOnly cookies, login/logout/invite flows — v2.0
- ✓ Workflows persistence — templates, submissions, approvals, audit trail — v2.0
- ✓ Real-time cross-device sync — op-log, WebSocket hub, Lamport clocks — v2.0
- ✓ Onboarding persistence — training progress, sign-off journal entries — v2.0
- ✓ Inventory/Purchasing data pipeline — receipt ingestion, real purchase events — v2.0
- ✓ PWA offline mode — IndexedDB queue, drain-on-reconnect, sync banner — v2.0
- ✓ Photo upload via presigned URLs (DO Spaces) — v2.0
- ✓ Tile-based permissions — users see only permitted app tiles — v2.0
- ✓ Template assignment UI — managers assign training templates to hires — v2.0

### Active

- [ ] PO form pulls reorder suggestions from inventory Stock tab with item photos, store location, and suggested quantities
- [ ] Search and add items from inventory catalog (Setup tab) into PO form
- [ ] Notion data import — seed item catalog from existing Notion export (~100 items with photos, categories, stores)
- [ ] Recurring weekly cutoff schedule (admin-configurable day/time); after cutoff only admin can edit PO
- [ ] Backend command for easy cutoff testing
- [ ] Admin approval of PO generates a shopping checklist (1:1 mapping)
- [ ] Shopping checklist tab with RBAC — assignable to specific members/roles
- [ ] Shopping list shows item photo + store location (tap icon); location notes editable inline
- [ ] Shopping list "Complete" button sends alert for missing items via configured channel
- [ ] "Repurchased +[Qty]" badge on inventory items after purchase; configurable reset date
- [ ] Alert reminders before cutoff time; out-of-stock alerts
- [ ] Zoho Cliq channel integration for alerts (default); email as alternative
- [ ] Users configure communication preference in Users tab (at least one required, Zoho Cliq default)

### Out of Scope

- Multi-location support — single food truck operation
- Real-time stock counting / barcode scanning — too complex for this milestone
- HTMX frontend migration — may come in a future milestone
- Food cost calculations — deferred from v2.x, not in scope for v3.0
- Dev deployment via Tailscale — infrastructure concern, separate from PO workflow

## Current Milestone: v3.0 Purchase Orders & Shopping Lists

**Goal:** Connect the Purchase Orders tool to real inventory data with a full ordering-to-shopping workflow, cutoff enforcement, and multi-channel alerts via Zoho Cliq/email.

**Target features:**
- PO form from inventory with reorder suggestions, photos, store locations, suggested quantities
- Notion data import for item catalog seeding
- Weekly cutoff schedule with admin lock and approval flow
- Shopping checklist generation from approved POs with RBAC
- Repurchased badge on inventory items with configurable reset
- Multi-channel alerts (Zoho Cliq channel + email) with user preference config

## Shipped Milestones

- **v1.0** Operations Console MVP (2026-04-14) — Workflows engine + Onboarding app
- **v1.1** Inventory App (2026-04-14) — Purchase history, spending trends, stock estimation, food cost intelligence
- **v2.0** Backend (2026-04-19) — Go + Postgres API, auth, real-time sync, offline support, receipt ingestion, photo upload, tile permissions
- **v2.1** Onboarding Video Upgrade (2026-04-20) — DO Spaces video upload, FFmpeg conversion, inline player, watch enforcement, thumbnails

## Context

- **Business type:** Food truck / mobile food operation
- **Team size:** 1-5 people (owner + small crew)
- **Codebase:** Static HTML/CSS/JS PWA + Playwright tests, deployed on Digital Ocean App Platform
- **Design system:** CSS variables with dark mode, mobile-first (480px max), inline styles, system font stack
- **Inspiration:** Lumiform — form/checklist builder with templates, assignments, and mobile completion
- **Workflows architecture:** My Checklists (all roles) / Approvals (manager+) / Builder (restricted roles)
- **Onboarding architecture:** My Trainings (crew) / Manager (manager+) / Builder (manager+) — video-based, sequential sections, per-section sign-off
- **User management integration:** Access controlled via Users app permissions (role-based + individual grants)
- **Backend design:** `docs/user-management-api.md` — 7 tables (users, sessions, templates, submissions, responses, rejections, audit log), full REST API contracts, Go + Postgres planned
- **v1.0 shipped:** 2026-04-14 — 5 phases, 12 plans, 89 E2E tests, 3,695 LOC

## Constraints

- **Static only:** No build step, no framework — plain HTML, CSS, vanilla JS
- **PWA:** Must work as installed app on iOS and Android, offline-capable via service worker
- **Mobile-first:** All UI designed for 480px max-width, touch-optimized
- **Design consistency:** Shared CSS variables and dark mode across all pages
- **API-backed:** All data persisted in Postgres via Go backend REST API — no mock data remains
- **Testing:** Playwright E2E tests run via `npm test`

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| UI mocks only (no backend) | Validate UX before investing in backend | ✓ Good — full workflow validated |
| Keep purchasing separate | Purchasing works well as-is | ✓ Good |
| Three-tab layout | My Checklists / Approvals / Builder maps to crew → manager → owner | ✓ Good |
| Lumiform-style template builder | Proven pattern for field ops checklists | ✓ Good |
| Day-of-week conditions | Different procedures per day | ✓ Good |
| Fail triggers with corrective actions | Food safety documentation | ✓ Good |
| Sub-steps for checkboxes | Recipes with sub-tasks (e.g., sauce = sugar + ketchup + soy) | ✓ Good |
| Assignable checklists by role/user | Different checklists for different roles | ✓ Good |
| Item-level rejection (not whole-checklist) | Managers flag specific items for correction | ✓ Good |
| SortableJS via CDN | Only external dependency, touch-native drag | ✓ Good |
| Playwright for E2E tests | 89 tests, catches regressions, free/open source | ✓ Good |
| Standalone onboarding tool | Separate from workflows — different UX for training vs daily checklists | ✓ Good |
| Video-based training with sequential sections | Crew watches video series, manager signs off per section | ✓ Good |
| Onboarding builder mirrors workflows builder | Same flat-section pattern, SortableJS — consistent UX | ✓ Good |
| Tab switch re-renders fresh | Prevents stale state between tabs (bug found during verification) | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-22 after v3.0 milestone start*
