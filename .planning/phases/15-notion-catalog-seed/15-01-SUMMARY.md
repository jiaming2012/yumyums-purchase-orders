---
phase: 15-notion-catalog-seed
plan: "01"
subsystem: backend/inventory
tags: [seed, migration, notion, import, do-spaces]
dependency_graph:
  requires: [14-po-backend-order-form]
  provides: [purchase_items_full_name_migration, notion_import_script, purchase_item_groups_extended]
  affects: [backend/internal/inventory, backend/cmd/import-notion]
tech_stack:
  added: [golang.org/x/text/cases, encoding/csv]
  patterns: [go:embed seed fixture, goose migration, direct S3 PutObject]
key_files:
  created:
    - backend/internal/db/migrations/0036_purchase_items_full_name.sql
    - backend/cmd/import-notion/main.go
    - backend/internal/inventory/fixtures/purchase_items.yaml
  modified:
    - backend/internal/inventory/fixtures/purchase_item_groups.yaml
    - .gitignore
decisions:
  - "Used direct S3 PutObject (not presigned PUT) in import script — presigned URLs are for browser-side uploads; CLI batch tool should use server-side PutObject"
  - "Category mapping treats Frozen/Meat as Proteins, Dry Groceries/Sauces as Sauces — consistent with D-01"
  - "Unknown Notion categories default to Dry Goods group (catch-all)"
  - "Placeholder purchase_items.yaml committed as items:[] so codebase compiles before user runs the script"
metrics:
  duration_minutes: 3
  completed_date: "2026-04-22"
  tasks_completed: 2
  files_changed: 5
---

# Phase 15 Plan 01: Notion Catalog Seed — Foundation Summary

One-liner: DB migration for full_name column, two new item groups (Seasoning/Sauces), and a Go CLI that reads Notion CSV export, uploads photos to DO Spaces via PutObject, and writes purchase_items.yaml.

## What Was Built

**Task 1: Migration, group fixtures, gitignore**

- `0036_purchase_items_full_name.sql`: goose migration adding `full_name TEXT` column to `purchase_items` table, wrapped in BEGIN/COMMIT matching existing migration patterns
- `purchase_item_groups.yaml`: Added Seasoning group (par_days: 14, tags: Seasoning/Spices) and Sauces group (par_days: 14, tags: Sauces/Condiments) after existing Cleaning Supplies group — total 10 groups
- `.gitignore`: Added `notion-export/` entry with comment so Notion source files (CSV + images) are never committed

**Task 2: Conversion script and placeholder YAML**

- `backend/cmd/import-notion/main.go`: Standalone Go command that:
  - Parses Notion CSV using `encoding/csv` with column index discovery
  - Maps categories to groups per D-01 (9 category mappings + default "Dry Goods")
  - URL-decodes photo paths and finds images under `--images` directory
  - Uploads PNGs to DO Spaces via direct `s3.Client.PutObject` with `public-read` ACL
  - Uses `photos.PublicURL()` to build permanent DO Spaces URLs (no expiring presigned URLs)
  - Title-case normalizes item names via `cases.Title(language.English)` matching `normalizeItemName()`
  - Converts descriptions to URL-safe slugs for S3 keys (`items/{slug}.png`)
  - Writes YAML grouped by item group in insertion order
  - Prints summary: total items, photos uploaded, items per group
  - Flags: `--csv` (required), `--images` (required), `--output`, `--dry-run`
  - Env vars: `DO_SPACES_KEY`, `DO_SPACES_SECRET`, `DO_SPACES_ENDPOINT`, `DO_SPACES_REGION`, `DO_SPACES_BUCKET`
- `purchase_items.yaml`: Placeholder file (`items: []`) so the codebase compiles before the user runs the script

## Commits

| Task | Commit | Files |
|------|--------|-------|
| 1: Migration + groups + gitignore | 5e8bdd1 | backend/internal/db/migrations/0036_purchase_items_full_name.sql, backend/internal/inventory/fixtures/purchase_item_groups.yaml, .gitignore |
| 2: Import script + placeholder YAML | 6380e47 | backend/cmd/import-notion/main.go, backend/internal/inventory/fixtures/purchase_items.yaml |

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written, with one acknowledged deviation from D-06 already noted in the plan itself:

**D-06 acknowledged deviation:** The plan specified "same presigned PUT pattern from Phase 12" but notes in the same section that direct `PutObject` is more appropriate for a batch CLI tool (presigned URLs are designed for browser-side uploads). The script uses `s3.Client.PutObject` directly as specified in the plan's implementation details.

## Verification Results

All checks pass:
- `grep -q "full_name" backend/internal/db/migrations/0036_purchase_items_full_name.sql` → PASS
- `grep -q "Seasoning" backend/internal/inventory/fixtures/purchase_item_groups.yaml` → PASS
- `grep -q "Sauces" backend/internal/inventory/fixtures/purchase_item_groups.yaml` → PASS
- `grep -q "notion-export" .gitignore` → PASS
- `go build ./cmd/import-notion/` → PASS
- `test -f internal/inventory/fixtures/purchase_items.yaml` → PASS
- No Notion S3 URLs in committed fixtures → PASS

## Known Stubs

`backend/internal/inventory/fixtures/purchase_items.yaml` contains `items: []` — intentional placeholder. Plan 15-02 will either run the import script to generate real data, or the user will run it manually with their Notion export.

## Next Steps (Plan 15-02)

Plan 15-02 should extend `SeedInventoryFixtures()` in `service.go` to read `purchase_items.yaml` and seed items with `photo_url`, `store_location`, and `full_name` fields via `ON CONFLICT (description) DO NOTHING`.

## Self-Check: PASSED

- `backend/internal/db/migrations/0036_purchase_items_full_name.sql` — EXISTS
- `backend/cmd/import-notion/main.go` — EXISTS
- `backend/internal/inventory/fixtures/purchase_items.yaml` — EXISTS
- Commit 5e8bdd1 — EXISTS (git log confirmed)
- Commit 6380e47 — EXISTS (git log confirmed)
