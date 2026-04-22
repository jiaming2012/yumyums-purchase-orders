---
gsd_state_version: 1.0
milestone: v3.0
milestone_name: Purchase Orders & Shopping Lists
status: executing
stopped_at: Completed 16-02-PLAN.md
last_updated: "2026-04-22T18:29:24.050Z"
last_activity: 2026-04-22
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 9
  completed_plans: 7
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-22)

**Core value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.
**Current focus:** Phase 16 — cutoff-approval-and-shopping-list

## Current Position

Phase: 16 (cutoff-approval-and-shopping-list) — EXECUTING
Plan: 4 of 5
Status: Ready to execute
Last activity: 2026-04-22

Progress: [░░░░░░░░░░] 0% (v3.0)

## Performance Metrics

**Velocity (v3.0):**

- Total plans completed: 0
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:** —

*Updated after each plan completion*
| Phase 14 P01 | 3m | 3 tasks | 6 files |
| Phase 14 P02 | 3m | 1 tasks | 2 files |
| Phase 15 P02 | 10 | 1 tasks | 1 files |
| Phase 16 P01 | 2m | 2 tasks | 5 files |
| Phase 16 P03 | 3m | 2 tasks | 3 files |
| Phase 16 P02 | 8m | 2 tasks | 4 files |

## Accumulated Context

### Decisions

- **[v3.0 roadmap]:** Shopping list must be an immutable snapshot created at approval time — not a live JOIN view of the PO
- **[v3.0 roadmap]:** Cutoff enforcement requires Postgres WHERE clause (not only Go guard clauses); `version` column on `purchase_orders` from migration 0034
- **[v3.0 roadmap]:** Zoho Cliq alerts write to `outgoing_alerts` queue table in same DB transaction and deliver async — never block the API handler
- **[v3.0 roadmap]:** Notion image URLs expire in 1 hour; must re-host to DO Spaces during import — raw Notion S3 URLs never stored
- **[v3.0 roadmap]:** Badge reset uses `business_timezone` from settings table — not UTC truncation
- [Phase 14]: America/Chicago timezone for week_start computation in PO backend
- [Phase 14]: UpsertLineItems uses transaction: verify draft, delete removed items, upsert remaining; added_by only set on INSERT
- [Phase 14]: Full rewrite of purchasing.html (not incremental) — 89-line mock had no reusable state management
- [Phase 15]: nilIfEmpty helper converts optional YAML string fields to *string (NULL in Postgres) for photo_url, store_location, full_name
- [Phase 15]: Unknown groups in purchase_items.yaml log warning and skip rather than crashing the seed
- [Phase 16]: vendor_name snapshotted at shopping list creation time so display works even if vendor is later deleted
- [Phase 16]: shopping_list_vendor_sections allows NULL vendor_id; unassigned items go into Unassigned section
- [Phase 16]: loadShoppingListSections loads all items in one query then distributes by vendor_section_id map to avoid N+1 per section
- [Phase 16]: UpdateShoppingItemLocation and UpdateShoppingItemPhoto update both shopping_list_items snapshot and purchase_items catalog so future lists see updates
- [Phase 16]: LockPO uses WHERE id=$1 AND status='draft' for optimistic locking (Pitfall 2)
- [Phase 16]: ApprovePO creates shopping list snapshot atomically in single TX with active-list check (D-11, D-15)

### Pending Todos

None yet.

### Blockers/Concerns

- **[Phase 14]:** Confirm business timezone (likely `America/Chicago`) before writing migration 0034 default
- **[Phase 16]:** PO state machine transition diagram required before any shopping list endpoint is built — admin unlock after list generation is highest-risk unspecified path
- **[Phase 17]:** Confirm email provider in use for v2.0 invite flow (Resend or Postmark) before specifying go-mail SMTP config in Phase 17
- **[Phase 17]:** Zoho Cliq service account + webhook token must be created by owner before Phase 17 is testable (operational prerequisite)

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260415-axs | Convert backend/Makefile to backend/Taskfile.yml (go-task format) | 2026-04-15 | 691e616 | [260415-axs-convert-backend-makefile-to-backend-task](./quick/260415-axs-convert-backend-makefile-to-backend-task/) |
| 260417-x0g | Add user display name and logout button to index.html | 2026-04-17 | 5edc1e1 | [260417-x0g-add-user-display-name-and-logout-button-](./quick/260417-x0g-add-user-display-name-and-logout-button-/) |
| 260418-0tz | Multi-role support for users and training templates | 2026-04-18 | bf6a7c2 | [260418-0tz-multi-role-support-for-users-and-trainin](./quick/260418-0tz-multi-role-support-for-users-and-trainin/) |
| 260421-im4 | Rename History tab to Purchases in inventory.html | 2026-04-21 | f46a88c | [260421-im4-rename-history-tab-to-purchases-in-inven](./quick/260421-im4-rename-history-tab-to-purchases-in-inven/) |
| 260421-iug | Move Stock tab before Trends in inventory.html | 2026-04-21 | ab6806f | [260421-iug-move-the-stock-tab-before-trends-in-inve](./quick/260421-iug-move-the-stock-tab-before-trends-in-inve/) |

## Session Continuity

Last session: 2026-04-22T18:29:24.041Z
Stopped at: Completed 16-02-PLAN.md
Resume file: None
