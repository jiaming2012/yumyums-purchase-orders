---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 01
subsystem: backend/db
tags: [migrations, schema, toast-ingest]
provides:
  - menu_items table with master_id UNIQUE
  - daily_menu_sales table with composite PK
  - FK from daily_menu_sales.menu_item_id → menu_items.id ON DELETE CASCADE
requires:
  - pgx/pgxpool migration runner (db.Migrate)
  - gen_random_uuid() (pgcrypto extension already in use across project)
affects:
  - backend startup migration run (next deploy creates both tables)
tech_stack_added: []
patterns_added:
  - aggregate-at-ingest table with composite natural PK (mirrors low_stock_alert_log composite-unique precedent)
key_files_created:
  - backend/internal/db/migrations/0060_menu_items.sql
  - backend/internal/db/migrations/0061_daily_menu_sales.sql
key_files_modified: []
decisions:
  - composite PK on daily_menu_sales (no surrogate UUID) — no other table references a single sales row
  - menu_subgroup nullable per D-07 — rare today but Toast may populate later
  - last_seen DATE (not TIMESTAMPTZ) — granularity matches Toast's business_date
metrics:
  duration_minutes: 1
  tasks_completed: 2
  files_changed: 2
  completed_date: 2026-06-03
---

# Phase 22 Plan 01: Migrations — menu_items + daily_menu_sales Summary

Two goose migrations landed (0060, 0061) that anchor the HQ Toast ingest pipeline: `menu_items` stores the 3-level Toast menu hierarchy keyed on `master_id`, and `daily_menu_sales` stores per-(item, date) aggregates with last-pull-wins upsert semantics. Both follow the project's `BEGIN/COMMIT` + matching Down convention from `0024_inventory.sql`; `db.Migrate(pool)` picks them up automatically on next server start.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Create 0060_menu_items.sql migration | 967deae | backend/internal/db/migrations/0060_menu_items.sql |
| 2 | Create 0061_daily_menu_sales.sql migration | 38b065a | backend/internal/db/migrations/0061_daily_menu_sales.sql |

## What Shipped

### 0060_menu_items.sql

- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `master_id TEXT UNIQUE NOT NULL` — stable Toast identifier; ingest upserts target it
- `name TEXT NOT NULL`
- `menu TEXT NOT NULL`, `menu_group TEXT NOT NULL` — required levels of D-07's 3-level hierarchy
- `menu_subgroup TEXT` (nullable) — most rows today have no subgroup
- `last_seen DATE NOT NULL` — bumped to `MAX(business_date)` when units_sold > 0
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Indexes: `menu_items_last_seen_idx` (DESC), `menu_items_menu_group_idx`
- Down: `DROP TABLE IF EXISTS menu_items;` wrapped in BEGIN/COMMIT

### 0061_daily_menu_sales.sql

- `menu_item_id UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE`
- `business_date DATE NOT NULL`
- `units_sold INTEGER NOT NULL`
- `gross_amount NUMERIC(10,2) NOT NULL` — matches `purchase_events.total` precision
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` — last-pull wins per D-05
- `PRIMARY KEY (menu_item_id, business_date)` — natural composite key, no surrogate ID
- Index: `daily_menu_sales_business_date_idx` (DESC) for the week-window read endpoint
- Down: `DROP TABLE IF EXISTS daily_menu_sales;` wrapped in BEGIN/COMMIT

## Verification

- `cd backend && go build ./...` exit 0 after each task
- All grep acceptance criteria from the plan pass:
  - `CREATE TABLE menu_items` = 1
  - `master_id     TEXT UNIQUE NOT NULL` = 1
  - `menu_subgroup TEXT,` = 1 (nullable, no NOT NULL)
  - `menu_items_last_seen_idx` present
  - `-- +goose Up` / `-- +goose Down` markers present
  - `DROP TABLE IF EXISTS menu_items;` present
  - `CREATE TABLE daily_menu_sales` = 1
  - `REFERENCES menu_items(id) ON DELETE CASCADE` = 1
  - `PRIMARY KEY (menu_item_id, business_date)` = 1
  - `gross_amount  NUMERIC(10,2) NOT NULL` = 1
  - `daily_menu_sales_business_date_idx` present
- Migration files use the next free sequential numbers (0060/0061; previous high was 0059_cleanup_photo_items.sql)
- `git status --short` clean after both commits — no stray build artifacts

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Composite PK on daily_menu_sales (no surrogate UUID) | No other table references a single sales row; saves one column + one index. Differs from `low_stock_alert_log` which keeps a surrogate UUID alongside a unique constraint — Phase 22 promotes the natural key to PK directly. |
| menu_subgroup nullable | Per D-07; most Toast rows have no subgroup today but Toast may populate it later. Avoids a separate schema bump if/when that happens. |
| `last_seen DATE` not TIMESTAMPTZ | Toast's CSV business_date is calendar-day granularity; storing as DATE matches the source and simplifies the `GREATEST(menu_items.last_seen, EXCLUDED.last_seen)` upsert. |
| `gross_amount NUMERIC(10,2)` | Matches the project convention from `purchase_events.total` (line 39 of 0024_inventory.sql). |
| `ON DELETE CASCADE` on the FK | Dropping a menu_item should also clear its sales rows — operator-driven cleanup, the worker itself never deletes menu_items. |

## Deviations from Plan

None — plan executed exactly as written. The two SQL files match the plan's specified content verbatim.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or trust-boundary changes introduced. Both migrations are schema-only under the existing `db.Migrate` trust boundary already modelled in the plan's `<threat_model>` (T-22-01 mitigated via BEGIN/COMMIT envelopes + `DROP TABLE IF EXISTS` in Down).

## Self-Check: PASSED

- FOUND: backend/internal/db/migrations/0060_menu_items.sql
- FOUND: backend/internal/db/migrations/0061_daily_menu_sales.sql
- FOUND commit: 967deae
- FOUND commit: 38b065a
