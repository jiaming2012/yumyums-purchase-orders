---
phase: 14-po-backend-order-form
plan: "01"
subsystem: backend/purchasing
tags: [go, postgres, migrations, rest-api, purchase-orders]
dependency_graph:
  requires: []
  provides: [purchasing-api, purchase-orders-schema, po-line-items-schema]
  affects: [backend/cmd/server/main.go, purchase_items-table]
tech_stack:
  added: [internal/purchasing package]
  patterns: [chi-closure-handlers, pgx-pool, goose-migrations, upsert-on-conflict]
key_files:
  created:
    - backend/internal/db/migrations/0034_purchase_orders.sql
    - backend/internal/db/migrations/0035_purchase_items_photo_store.sql
    - backend/internal/purchasing/types.go
    - backend/internal/purchasing/service.go
    - backend/internal/purchasing/handler.go
  modified:
    - backend/cmd/server/main.go
decisions:
  - "America/Chicago timezone used for week_start computation — confirmed from STATE.md blocker"
  - "UpsertLineItems deletes removed items and upserts remaining in a single transaction"
  - "added_by only set on INSERT (first adder wins), not updated on quantity change"
  - "GetSuggestions returns empty array (not null) for safe frontend consumption"
metrics:
  duration: "~3 minutes"
  completed_date: "2026-04-22"
  tasks_completed: 3
  files_created: 5
  files_modified: 1
---

# Phase 14 Plan 01: PO Backend — Order Form Summary

Purchase Orders Go backend with two Goose migrations (purchase_orders/po_line_items tables + photo_url/store_location columns) and a new `internal/purchasing` package exposing 4 REST endpoints for get-or-create PO, line item upsert with draft guard, and reorder suggestions.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create Goose migrations (0034, 0035) | c722109 | 0034_purchase_orders.sql, 0035_purchase_items_photo_store.sql |
| 2 | Create purchasing Go package | b768723 | types.go, service.go, handler.go |
| 3 | Register purchasing routes in main.go | 498146c | main.go |

## What Was Built

### Migrations

**0034_purchase_orders.sql:**
- `purchase_orders` table: UUID PK, `week_start DATE UNIQUE` (one PO per week), `status CHECK('draft'|'locked'|'approved')`, `version INTEGER` (optimistic locking), `created_at`
- `po_line_items` table: FK to purchase_orders (cascade delete), FK to purchase_items, FK to users (added_by), `quantity CHECK(>0)`, `UNIQUE(po_id, purchase_item_id)` enabling ON CONFLICT upsert, `updated_at`
- Index on `po_line_items(po_id)` for list queries

**0035_purchase_items_photo_store.sql:**
- `photo_url TEXT` — nullable, populated during Phase 15 Notion import
- `store_location TEXT` — nullable, populated during Phase 15 Notion import

### Go Package: internal/purchasing

**types.go:** PurchaseOrder, POLineItem, OrderSuggestion, UpsertLineItemInput, UpsertLineItemsRequest

**service.go:**
- `CurrentWeekStart()` — Monday of current week in America/Chicago
- `GetOrCreateOrder()` — INSERT ON CONFLICT (week_start) DO UPDATE, returns PO with line items
- `GetOrderByID()` — fetch PO by ID with line items; returns nil (not error) for 404
- `GetOrderLineItems()` — JOIN with purchase_items, item_groups, users; includes photo_url, store_location, group_name, added_by display name
- `UpsertLineItems()` — transaction: verify draft status, delete removed items, upsert remaining; returns ErrPONotDraft if locked/approved
- `GetSuggestions()` — items below group low_threshold not already on PO; uses COALESCE(stock_count_override, purchase_history_total)

**handler.go:**
- `GetOrCreateOrderHandler` — POST /orders, auth-checked
- `GetOrderHandler` — GET /orders/{id}, returns 404 for missing
- `UpsertLineItemsHandler` — PUT /orders/{id}/items, returns 409 `{"error":"po_not_draft"}` when locked
- `GetSuggestionsHandler` — GET /orders/{id}/suggestions, always returns array (never null)

### Route Registration

Added to main.go inside the authenticated r.Group, after the inventory block:
```
POST   /api/v1/purchasing/orders
GET    /api/v1/purchasing/orders/{id}
PUT    /api/v1/purchasing/orders/{id}/items
GET    /api/v1/purchasing/orders/{id}/suggestions
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

- `photo_url` and `store_location` on purchase_items are nullable with no data yet — Phase 15 Notion import will populate them. Frontend must handle null gracefully (plan notes this explicitly).
- `unit` field in OrderSuggestion is hardcoded to `''` — unit data comes from purchase history or Notion import in a later phase.

## Self-Check: PASSED

- [x] 0034_purchase_orders.sql exists with CREATE TABLE purchase_orders, CREATE TABLE po_line_items, UNIQUE constraint, added_by FK, version column, status CHECK
- [x] 0035_purchase_items_photo_store.sql exists with ALTER TABLE for photo_url and store_location
- [x] internal/purchasing/types.go: PurchaseOrder, POLineItem, OrderSuggestion, UpsertLineItemsRequest
- [x] internal/purchasing/service.go: CurrentWeekStart (America/Chicago), GetOrCreateOrder (ON CONFLICT week_start), GetSuggestions, UpsertLineItems, ErrPONotDraft
- [x] internal/purchasing/handler.go: all 4 handlers with auth.UserFromContext and chi.URLParam
- [x] main.go: purchasing import and r.Route("/purchasing") block
- [x] go build ./cmd/server/... exits 0
- [x] go vet ./internal/purchasing/... exits 0
- [x] Commits: c722109, b768723, 498146c all exist
