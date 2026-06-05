# Phase 15: Notion Catalog Seed - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Convert the Notion CSV export (~100 items with photos, categories, stores) into a YAML seed file following existing fixture patterns. A one-time conversion script uploads images to DO Spaces and writes the YAML. The YAML seeds on every server startup idempotently via `go:embed` + `ON CONFLICT DO NOTHING`.

</domain>

<decisions>
## Implementation Decisions

### Category Mapping
- **D-01:** Auto-map Notion categories to existing item_groups where obvious, create new groups for unmatched:
  - Produce → Produce (exists)
  - Bread → Bread (exists)
  - Dairy → Dairy (exists)
  - Frozen / Meat → Proteins (exists)
  - Drinks → Beverages (exists)
  - Cleaning → Cleaning Supplies (exists)
  - Cooking Supplies → Packaging (exists)
  - Seasoning → Seasoning (NEW group)
  - Dry Groceries / Sauces → Sauces (NEW group)
- **D-02:** New groups (Seasoning, Sauces) are added to `purchase_item_groups.yaml` with appropriate tags and par_days

### Photo Handling
- **D-03:** One-time Go conversion script at `cmd/import-notion/` reads Notion CSV + local image files, uploads PNGs to DO Spaces, writes `purchase_items.yaml` with DO Spaces URLs
- **D-04:** Notion source files (CSV + images) copied into project directory but added to `.gitignore`
- **D-05:** The generated YAML with DO Spaces URLs IS committed to the repo (it's the seed artifact)
- **D-06:** Photos are uploaded to DO Spaces using the same presigned PUT pattern from Phase 12

### Item Name Format
- **D-07:** Short name (Notion "Name" column) becomes `description` — e.g. "Brisket Rolls"
- **D-08:** Full name (Notion "Full Name" column) stored as `full_name` field — e.g. "4\" Brioche Roll - 12 ct Pack"
- **D-09:** If short name is empty, fall back to full name for description
- **D-10:** `purchase_items` table needs a `full_name` column (ALTER TABLE migration)

### Seed Pattern
- **D-11:** YAML seed file embedded via `go:embed`, runs on every startup idempotently — matches `superadmins.yaml` and `purchase_item_groups.yaml` patterns
- **D-12:** Items seeded with `ON CONFLICT (description) DO NOTHING` — re-runs don't duplicate
- **D-13:** Store location from Notion "Store" column maps to `purchase_items.store_location`

### Claude's Discretion
- Exact DO Spaces key path for seed photos (e.g. `items/{slug}.png`)
- Whether to normalize/resize images before upload
- How to handle items with no category in the Notion CSV (assign to a default group or skip)
- Conversion script CLI flag names

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing seed infrastructure
- `backend/internal/inventory/service.go` lines 1-100 — `SeedInventoryFixtures()` function, `go:embed`, YAML parsing, idempotent insert pattern
- `backend/internal/inventory/fixtures/purchase_item_groups.yaml` — Current YAML structure with vendors, item_groups, tags, purchase_items

### Notion source data
- `/Users/jamal/Downloads/ExportBlock-99603bef-0b4f-4395-91a5-37036f190626-Part-1/All Items 136643f80250811f980cda0a9127c6d8.csv` — CSV with columns: Name, Category, Full Name, Last Purchased, Photo, Purchased, Quantity, Store
- `/Users/jamal/Downloads/ExportBlock-99603bef-0b4f-4395-91a5-37036f190626-Part-1/All Items/` — Image subdirectories per item

### Schema
- `backend/internal/db/migrations/0035_purchase_items_photo_store.sql` — Added photo_url, store_location to purchase_items
- `backend/internal/db/migrations/0024_inventory.sql` — purchase_items table definition

### Photo upload pattern
- `backend/internal/photos/handler.go` — DO Spaces presigned PUT pattern (reuse for seed uploads)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **SeedInventoryFixtures()**: Existing seed function — extend to handle purchase_items with photo_url, store_location, full_name
- **purchaseItemFixture struct**: Already exists with `description` field — add `photo_url`, `store_location`, `full_name`
- **DO Spaces upload**: `photos/handler.go` has presigned PUT — conversion script can use same S3 client

### Established Patterns
- **go:embed fixtures/**: YAML files embedded at compile time
- **ON CONFLICT DO NOTHING**: Idempotent seeding
- **normalizeItemName()**: Title case normalization via `cases.Title(language.English)` — apply to seeded item names

### Integration Points
- `purchase_item_groups.yaml` gets two new groups (Seasoning, Sauces) with their tags
- `purchase_items` table gets new `full_name` column via migration
- `SeedInventoryFixtures()` extended to insert items under their groups

</code_context>

<specifics>
## Specific Ideas

- The conversion script should be a standalone Go command (`go run ./cmd/import-notion/ --csv ... --images ...`) that outputs the YAML file
- Items with no photo in the CSV should have `photo_url: null` in the YAML
- Items with no category should be assigned to "Dry Goods" as default (catch-all for miscellaneous)
- The YAML should nest items under their groups (matching existing structure)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 15-notion-catalog-seed*
*Context gathered: 2026-04-22*
