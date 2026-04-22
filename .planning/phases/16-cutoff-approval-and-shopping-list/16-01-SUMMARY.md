---
phase: 16-cutoff-approval-and-shopping-list
plan: "01"
subsystem: purchasing
tags: [database, migrations, go-types, shopping-list, cutoff]
dependency_graph:
  requires: []
  provides: [cutoff_config table, extended PO status, vendor FK on purchase_items, shopping list tables, Phase 16 Go types]
  affects: [purchase_orders, purchase_items, purchasing/types.go]
tech_stack:
  added: []
  patterns: [goose migrations, Go struct types]
key_files:
  created:
    - backend/internal/db/migrations/0037_cutoff_config.sql
    - backend/internal/db/migrations/0038_po_status_extend.sql
    - backend/internal/db/migrations/0039_purchase_items_vendor.sql
    - backend/internal/db/migrations/0040_shopping_lists.sql
  modified:
    - backend/internal/purchasing/types.go
decisions:
  - "vendor_name snapshotted at shopping list creation time so display works even if vendor is later deleted"
  - "shopping_list_vendor_sections allows NULL vendor_id; items with no vendor go into Unassigned section"
  - "cutoff_config has single-row upsert pattern (no UNIQUE constraint needed — service enforces it)"
metrics:
  duration: "~2 min"
  completed: "2026-04-22"
  tasks: 2
  files: 5
---

# Phase 16 Plan 01: Database Migrations and Go Types Summary

**One-liner:** Four Goose migrations (cutoff_config, extended PO status + timestamps, vendor FK on purchase_items, three shopping list tables) plus updated purchasing/types.go with CutoffConfig, ShoppingList, ShoppingListVendorSection, ShoppingListItem, and updated PurchaseOrder/POLineItem structs.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create database migrations 0037-0040 | 8f51051 | 0037-0040_*.sql (4 files) |
| 2 | Add Phase 16 Go types to types.go | 3ed1955 | backend/internal/purchasing/types.go |

## What Was Built

### Migration 0037 — cutoff_config
New table storing the weekly order cutoff schedule: `day_of_week` (0-6), `cutoff_time` (TIME), `timezone` (default `America/Chicago`). Single-row upsert pattern; service layer enforces the single-row constraint.

### Migration 0038 — po_status_extend
Extended `purchase_orders` with three new columns (`locked_at`, `approved_at`, `approved_by`) and replaced the existing 3-value CHECK constraint with a 5-value one: `draft | locked | approved | shopping_active | completed`. Used `DROP CONSTRAINT IF EXISTS` for safety.

### Migration 0039 — purchase_items_vendor
Added nullable `vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL` to `purchase_items` with an index. Enables vendor grouping in the PO tab (D-09) via LEFT JOIN.

### Migration 0040 — shopping_lists
Created three tables forming the shopping list snapshot structure:
- `shopping_lists` — 1:1 with approved PO, has `assigned_to`/`assigned_role` and completion timestamp
- `shopping_list_vendor_sections` — per-vendor grouping with snapshotted `vendor_name`; UNIQUE(shopping_list_id, vendor_id) prevents duplicates
- `shopping_list_items` — individual items snapshotted from PO line items, with `checked`/`checked_by`/`checked_at` for shopping workflow

### Go Types (types.go)
- `PurchaseOrder`: added `LockedAt`, `ApprovedAt`, `ApprovedBy` optional fields; updated Status comment to list all 5 values
- `POLineItem`: added `VendorName *string` for vendor grouping (D-09)
- `CutoffConfig`: new struct with `DayOfWeek`, `CutoffTime`, `Timezone`, `UpdatedAt`
- `ShoppingList`: new struct with `VendorSections []ShoppingListVendorSection`
- `ShoppingListVendorSection`: new struct with `Items []ShoppingListItem`
- `ShoppingListItem`: new struct with `Checked`, `CheckedBy`, `CheckedByName`, `CheckedAt`

## Verification

- `go build ./cmd/server/...` — passes
- `go vet ./internal/purchasing/...` — passes
- All 4 migration files exist in `backend/internal/db/migrations/`
- Migration numbering 0037-0040 is sequential with no gaps

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan is schema-only; no UI or API stubs.

## Self-Check: PASSED

Files verified:
- FOUND: backend/internal/db/migrations/0037_cutoff_config.sql
- FOUND: backend/internal/db/migrations/0038_po_status_extend.sql
- FOUND: backend/internal/db/migrations/0039_purchase_items_vendor.sql
- FOUND: backend/internal/db/migrations/0040_shopping_lists.sql
- FOUND: backend/internal/purchasing/types.go

Commits verified:
- FOUND: 8f51051 (feat(16-01): create migrations 0037-0040...)
- FOUND: 3ed1955 (feat(16-01): add Phase 16 Go types...)
