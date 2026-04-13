# Yumyums HQ — Operations Console

## What This Is

A mobile-first PWA operations console for a food truck business. One app shell with a launcher grid that links to independent workflow tools — purchasing, user management, and a workflow/checklist engine inspired by Lumiform. Each tool is a standalone HTML page inside a shared PWA, designed for a small crew (1-5 people) to use on their phones.

## Core Value

A workflow engine that lets the owner build checklist templates and have crew members fill them out on mobile — with accountability (who checked what) and smart conditions (day-of-week, fail triggers, skip logic).

## Requirements

### Validated

- ✓ PWA shell with launcher grid — existing (`index.html`)
- ✓ Purchasing mockup with weekly order form, locked view, PO view — existing (`purchasing.html`)
- ✓ User management mockup with roles, invitations, app access control — existing (`users.html`)
- ✓ Login screen mockup — existing (`login.html`)
- ✓ Pull-to-refresh on iOS standalone — existing (`ptr.js`)

### Active

- [ ] Workflow fill-out view — crew sees assigned checklists, checks off items, notes who completed each item
- [ ] Workflow template builder — create/edit templates with sections, field types, and conditional logic
- [ ] Field types: checkboxes, text notes, yes/no, temperature input, photo capture, timestamps
- [ ] Conditional logic: fail triggers (out-of-range → corrective action), skip logic, day-of-week conditions
- [ ] Section grouping — items organized under headers (Equipment, Food Prep, Cleaning, etc.)
- [ ] Role-based access — fill-out tab visible to all roles, builder tab restricted via User Management permissions

### Out of Scope

- Backend / API implementation — UI mocks only for now, backend design docs exist but no server code
- Real authentication — login.html is a mockup, no session management
- Offline sync / IndexedDB — future concern, not needed for mocks
- Multi-location support — single food truck operation
- Reporting / analytics dashboard — may come later as a BI tool
- Replacing the existing purchasing app — purchasing stays separate, workflows handle other operations

## Context

- **Business type:** Food truck / mobile food operation
- **Team size:** 1-5 people (owner + small crew)
- **Existing codebase:** Static HTML/CSS/JS PWA, no build step, no framework, deployed on Digital Ocean App Platform
- **Design system:** CSS variables with dark mode, mobile-first (480px max), inline styles, system font stack
- **Inspiration:** Lumiform (lumiformapp.com/product) — form/checklist builder with templates, assignments, and mobile completion
- **Two-tab architecture:** Tab 1 is the fill-out view (all roles), Tab 2 is the template builder (restricted roles)
- **User management integration:** Access to the builder tab is controlled via the existing Users app permissions system (role-based + individual grants)
- **Backend design exists:** `docs/user-management-api.md` has data model and API contracts for future Go + Postgres backend

## Constraints

- **Static only:** No build step, no framework — plain HTML, CSS, vanilla JS (matches existing convention)
- **PWA:** Must work as installed app on iOS and Android, offline-capable via service worker
- **Mobile-first:** All UI designed for 480px max-width, touch-optimized
- **Design consistency:** Must use existing CSS variables and dark mode support from other HQ pages
- **Mocks only:** All data is hardcoded JavaScript arrays — no localStorage, no API calls

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| UI mocks only (no backend) | Owner wants to validate the UX before investing in backend | — Pending |
| Keep purchasing separate | Purchasing app works well as-is, no need to force it into the workflow engine | — Pending |
| Two-tab layout (fill-out + builder) | Matches mental model — most users just fill out, few need to build templates | — Pending |
| Lumiform-style template builder | Proven pattern for field ops checklists, familiar UX | — Pending |
| Day-of-week conditions | Food truck has different procedures per day (deep clean Monday, inventory Friday) | — Pending |
| Fail triggers with corrective actions | Food safety requires documenting what you did when something fails | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-13 after initialization*
