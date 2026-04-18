---
phase: 12-inventory-photos-tile-permissions
plan: "01"
subsystem: backend/inventory
tags: [inventory, postgres, migrations, seed-data, api]
dependency_graph:
  requires: []
  provides:
    - inventory DB schema (vendors, item_groups, tags, purchase_items, purchase_events, purchase_line_items, pending_purchases)
    - GET /api/v1/inventory/vendors
    - GET /api/v1/inventory/purchases
    - POST /api/v1/inventory/purchases
    - GET /api/v1/inventory/purchases/pending
    - POST /api/v1/inventory/purchases/confirm
    - POST /api/v1/inventory/purchases/discard
    - GET /api/v1/inventory/stock
    - SeedInventoryFixtures (called on startup)
    - hq_apps 'inventory' row
  affects:
    - backend/cmd/server/main.go (routes + seed call)
tech_stack:
  added: [gopkg.in/yaml.v3 (already in go.mod)]
  patterns:
    - goose migrations with BEGIN/COMMIT wrappers
    - //go:embed for YAML fixtures inside package directory
    - pgx.Rows for generic query results
    - pgxpool.Pool transaction for multi-table inserts
key_files:
  created:
    - backend/internal/db/migrations/0024_inventory.sql
    - backend/internal/db/migrations/0025_pending_purchases.sql
    - backend/internal/inventory/types.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/service.go
    - backend/internal/inventory/fixtures/purchase_item_groups.yaml
    - backend/config/fixtures/purchase_item_groups.yaml
  modified:
    - backend/cmd/server/main.go
decisions:
  - "YAML fixtures embedded inside inventory package directory (not config/fixtures/) because Go embed does not support .. path traversal — config/fixtures/ copy kept for reference"
  - "pgx.Rows interface used in ListPurchaseEventsHandler to handle conditional vendor_id filter with a single variable"
  - "N+1 line item loading kept for simplicity — inventory fetch volume is low; JOIN + scan is premature optimization at this stage"
metrics:
  duration: 3
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_changed: 8
---

# Phase 12 Plan 01: Inventory DB Schema, API, and Seed Data Summary

**One-liner:** Postgres-backed inventory API with 7 tables, 7 HTTP handlers, YAML seed fixtures, and idempotent startup seeding.

## What Was Built

Two goose migrations create the full inventory schema: `vendors`, `tags`, `item_groups`, `item_group_tags`, `purchase_items`, `purchase_events`, `purchase_line_items`, and `pending_purchases`. The `hq_apps` row for the inventory tile is inserted in migration 0024.

The Go `inventory` package provides domain types, 7 HTTP handlers, and a seed function that loads YAML fixtures on startup. All routes are registered inside the `/api/v1` protected group in `main.go`. Seed data covers 5 vendors (Restaurant Depot, US Foods, Sysco, Costco, Local Farm) and 8 item groups (Proteins, Produce, Dairy, Dry Goods, Packaging, Beverages, Bread, Cleaning Supplies) with tagged purchase items.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Inventory DB migrations and types | 2b86fda | 0024_inventory.sql, 0025_pending_purchases.sql, types.go |
| 2 | Handlers, service, seed data, route wiring | 483f970 | handler.go, service.go, fixtures/purchase_item_groups.yaml, main.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Embed path does not support parent directory traversal**
- **Found during:** Task 2
- **Issue:** Plan specified `//go:embed ../../../config/fixtures/purchase_item_groups.yaml` but Go's embed directive does not allow `..` path components.
- **Fix:** Created `backend/internal/inventory/fixtures/purchase_item_groups.yaml` (copy of the fixtures file inside the package) and used `//go:embed fixtures/purchase_item_groups.yaml`. The `backend/config/fixtures/` copy is kept for reference.
- **Files modified:** Added `backend/internal/inventory/fixtures/purchase_item_groups.yaml`
- **Commit:** 483f970

**2. [Rule 1 - Bug] Rows variable interface missing Err() method**
- **Found during:** Task 2
- **Issue:** Used a local anonymous interface `interface{ Next() bool; Scan(...any) error; Close() }` for the conditional query result variable, but the code called `rows.Err()` which is not on that interface.
- **Fix:** Changed variable type to `pgx.Rows` which includes `Err()`.
- **Files modified:** handler.go
- **Commit:** 483f970

## Known Stubs

None. This is a pure backend plan — no frontend wiring in scope.

## Self-Check: PASSED

Files verified:
- `backend/internal/db/migrations/0024_inventory.sql` — FOUND
- `backend/internal/db/migrations/0025_pending_purchases.sql` — FOUND
- `backend/internal/inventory/types.go` — FOUND
- `backend/internal/inventory/handler.go` — FOUND
- `backend/internal/inventory/service.go` — FOUND
- `backend/internal/inventory/fixtures/purchase_item_groups.yaml` — FOUND
- `backend/config/fixtures/purchase_item_groups.yaml` — FOUND
- `backend/cmd/server/main.go` — FOUND with `r.Route("/inventory"` and `inventory.SeedInventoryFixtures`

Commits verified:
- `2b86fda` — FOUND
- `483f970` — FOUND

Build: `go build ./cmd/server/...` — PASS
