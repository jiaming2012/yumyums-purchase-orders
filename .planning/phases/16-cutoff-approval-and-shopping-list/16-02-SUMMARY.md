---
phase: 16-cutoff-approval-and-shopping-list
plan: "02"
subsystem: purchasing
tags: [go-backend, state-machine, scheduler, cutoff, approval, shopping-list]
dependency_graph:
  requires: [16-01]
  provides: [PO state machine, cutoff scheduler, shopping list snapshot, locked PO query]
  affects: [purchasing/service.go, purchasing/handler.go, purchasing/scheduler.go, cmd/server/main.go]
tech_stack:
  added: []
  patterns: [optimistic locking, single-TX approval snapshot, DST-safe scheduler, admin RBAC]
key_files:
  created:
    - backend/internal/purchasing/scheduler.go
  modified:
    - backend/internal/purchasing/service.go
    - backend/internal/purchasing/handler.go
    - backend/cmd/server/main.go
decisions:
  - "LockPO uses WHERE id=$1 AND status='draft' in the UPDATE for optimistic locking (Pitfall 2 from research)"
  - "UnlockPO checks approved/shopping_active/completed before allowing unlock (D-13)"
  - "ApprovePO creates shopping list snapshot atomically in one TX — checks active list first (D-11, D-15)"
  - "scheduler uses time.LoadLocation for DST-safe timezone handling (Pitfall 1 from research)"
  - "cutoffTimeParts moved into scheduler.go to keep strings import collocated with its only user"
  - "GetOrdersByStatus route registered as GET /orders before POST /orders and {id} wildcard to avoid routing conflicts"
metrics:
  duration: "~8 min"
  completed: "2026-04-22"
  tasks: 2
  files: 4
---

# Phase 16 Plan 02: PO State Machine + Cutoff Scheduler Summary

**One-liner:** PO state machine (LockPO/UnlockPO/ApprovePO), cutoff config CRUD, 15-minute DST-safe scheduler, simulate-cutoff endpoint, atomic shopping list snapshot on approval, and locked PO query endpoint.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add state machine + cutoff service functions to service.go | 921d891 | backend/internal/purchasing/service.go |
| 2 | Create scheduler.go and add handlers + routes | b3c8cc0 | scheduler.go, handler.go, main.go |

## What Was Built

### State Machine Functions (service.go)

**Sentinel errors:** `ErrPONotLocked`, `ErrPOAlreadyApproved`, `ErrActiveShoppingListExists`, `ErrUnlockAfterApproval`

**isAdmin helper:** Checks `IsSuperadmin` flag and `Roles` slice for "admin" — used by all admin-only handlers.

**GetOrderLineItems updated:** Added `LEFT JOIN vendors v ON v.id = pi.vendor_id` and `v.name AS vendor_name` to the SELECT + scan (D-09).

**GetCutoffConfig / UpsertCutoffConfig:** Single-row table pattern — DELETE all then INSERT in a TX. Returns `*CutoffConfig` (nil if not set yet).

**LockPO:** TX with `UPDATE ... WHERE id=$1 AND status='draft'` (Pitfall 2 — optimistic locking). Creates next week's draft via `ON CONFLICT DO NOTHING`. Returns `ErrPONotDraft` if PO is already locked.

**UnlockPO:** Checks status before attempting update — returns `ErrUnlockAfterApproval` if approved/shopping_active/completed (D-13). Returns `ErrPONotLocked` if not in locked state.

**ApprovePO:** Atomic TX: (1) check no active shopping list, (2) set PO to `shopping_active`, (3) INSERT shopping_list, (4) query distinct vendors, (5) INSERT vendor sections, (6) INSERT item snapshots per vendor. Returns shopping list ID. Handles "unassigned" vendor bucket for items with no vendor_id.

**GetOrdersByStatus:** Returns most recent PO by status with line items. Used by frontend PO tab.

### Scheduler (scheduler.go)

`StartScheduler` follows the `receipt.StartWorker` goroutine pattern: runs immediately on start, then ticks every 15 minutes.

`runCutoffCheck`: Loads cutoff config → validates timezone with `time.LoadLocation` (DST-safe) → computes most recent occurrence of `day_of_week + HH:MM` in the configured timezone → finds draft PO → calls `LockPO` if cutoff has passed.

### Handlers (handler.go)

All 7 new handlers added:

| Handler | Method | Path | Admin? |
|---------|--------|------|--------|
| `GetCutoffConfigHandler` | GET | `/purchasing/cutoff` | No |
| `UpsertCutoffConfigHandler` | PUT | `/purchasing/cutoff` | Yes |
| `SimulateCutoffHandler` | POST | `/purchasing/simulate-cutoff` | Yes |
| `GetOrdersByStatusHandler` | GET | `/purchasing/orders?status=` | No |
| `LockPOHandler` | POST | `/purchasing/orders/{id}/lock` | Yes |
| `UnlockPOHandler` | POST | `/purchasing/orders/{id}/unlock` | Yes |
| `ApprovePOHandler` | POST | `/purchasing/orders/{id}/approve` | Yes |

Service error → HTTP status mapping:
- `ErrPONotDraft` → 409
- `ErrPONotLocked` → 409
- `ErrUnlockAfterApproval` → 409
- `ErrActiveShoppingListExists` → 409

### Routes + Scheduler (main.go)

All 7 routes registered under `/purchasing`. `GET /orders` registered before `POST /orders` and `GET /orders/{id}` to avoid routing conflicts. `purchasing.StartScheduler(ctx, pool)` called after `receipt.StartWorker`.

## Verification

- `go build ./cmd/server/...` — passes
- `go vet ./internal/purchasing/...` — passes
- `LockPO` confirmed with `WHERE id = $1 AND status = 'draft'` (optimistic locking)
- All 7 routes confirmed in main.go
- `StartScheduler` confirmed in main.go

## Deviations from Plan

**[Rule 2 - Auto-add] Moved cutoffTimeParts to scheduler.go:** The plan placed `cutoffTimeParts` in service.go, but it's only used by the scheduler and Go's import linter (goimports) kept removing the `strings` import from service.go because nothing else in the file used it. Moving `parseCutoffTime` to scheduler.go keeps the import collocated with its only consumer and avoids the linter conflict.

**[Context] service.go pre-populated by parallel agent:** Another parallel agent had already added shopping list service functions (GetActiveShoppingList, loadShoppingListSections, etc.) and handler functions to the files before this plan ran. This plan's additions were appended without conflict.

## Known Stubs

None — this plan is backend-only; no UI stubs.

## Self-Check: PASSED

Files verified:
- FOUND: backend/internal/purchasing/scheduler.go
- FOUND: backend/internal/purchasing/service.go (with LockPO, UnlockPO, ApprovePO, GetCutoffConfig, UpsertCutoffConfig, GetOrdersByStatus, isAdmin)
- FOUND: backend/internal/purchasing/handler.go (with SimulateCutoffHandler, LockPOHandler, UnlockPOHandler, ApprovePOHandler, GetCutoffConfigHandler, UpsertCutoffConfigHandler, GetOrdersByStatusHandler)
- FOUND: backend/cmd/server/main.go (with 7 routes + purchasing.StartScheduler)

Commits verified:
- FOUND: 921d891 (feat(16-02): add PO state machine + cutoff service functions)
- FOUND: b3c8cc0 (feat(16-02): create scheduler.go and add cutoff/approval handlers + routes)
