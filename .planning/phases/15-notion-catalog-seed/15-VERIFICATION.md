---
phase: 15-notion-catalog-seed
verified: 2026-04-22T17:30:00Z
status: gaps_found
score: 6/8 must-haves verified
gaps:
  - truth: "Notion CSV is converted to a YAML seed file with DO Spaces photo URLs"
    status: partial
    reason: "All artifacts exist and compile correctly in git commits on worktree-agent-a8c95582 but have NOT been merged into main. purchase_items.yaml is a placeholder (items: []) — intentionally so per D-05; the import script must be run manually with the Notion export to populate it."
    artifacts:
      - path: "backend/internal/inventory/fixtures/purchase_items.yaml"
        issue: "Placeholder only (items: []) — script must be run with actual Notion CSV to generate real data"
    missing:
      - "Merge worktree-agent-a8c95582 into main (4 commits: 5e8bdd1, 6380e47, c9dfda7, e883a0f)"
      - "Run go run ./cmd/import-notion/ with Notion CSV + images to populate purchase_items.yaml with real items and DO Spaces URLs"
      - "Commit the populated purchase_items.yaml to main"
  - truth: "Items seed idempotently on every server startup"
    status: failed
    reason: "The extended SeedInventoryFixtures (with purchase_items.yaml embedding and item seeding loop) exists only in commit e883a0f on worktree-agent-a8c95582, not in main. The main branch service.go has no go:embed for purchase_items.yaml and no item seeding loop."
    artifacts:
      - path: "backend/internal/inventory/service.go"
        issue: "Main branch version lacks the purchase_items.yaml embed and item seeding loop added in Phase 15"
    missing:
      - "Merge Phase 15 commits into main so server seeds items on startup"
human_verification:
  - test: "Run import-notion script with real Notion export and verify DO Spaces URLs"
    expected: "purchase_items.yaml populated with ~106 items, all photo_url fields pointing to *.digitaloceanspaces.com (not amazonaws.com or Notion URLs)"
    why_human: "Requires actual Notion CSV export files and DO Spaces credentials; cannot verify programmatically without running the script"
  - test: "Start server and verify items appear in inventory catalog"
    expected: "Items seeded from purchase_items.yaml appear in /api/v1/inventory/items with photo_url, store_location, full_name populated; no duplicates on restart"
    why_human: "Requires live DB connection and populated YAML; behavior verified at runtime"
  - test: "Confirm photos still load after 2+ hours"
    expected: "Item photos load correctly in inventory UI after 2+ hours, confirming DO Spaces (permanent) URLs not Notion (1-hour expiry) URLs"
    why_human: "Time-dependent check; requires real upload to DO Spaces followed by a wait"
---

# Phase 15: Notion Catalog Seed Verification Report

**Phase Goal:** Convert Notion CSV export into a YAML seed file with images re-hosted to DO Spaces; seeds on first run like other YAML fixtures
**Verified:** 2026-04-22T17:30:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | Notion CSV is converted to a YAML seed file with DO Spaces photo URLs | PARTIAL | Script exists in commit 6380e47 and is substantive (344 lines, full logic). purchase_items.yaml is a placeholder (items: []). Both are only on worktree branch, not main. |
| 2 | Raw Notion S3 URLs are never stored in the YAML | VERIFIED | purchase_items.yaml is placeholder with no URLs; import-notion script writes only photos.PublicURL (DO Spaces) output. No amazonaws.com or notion.so URLs anywhere in committed files. |
| 3 | New groups (Seasoning, Sauces) exist in group fixtures | VERIFIED | Confirmed in commit 5e8bdd1: purchase_item_groups.yaml has Seasoning (par_days:14, tags:[Seasoning,Spices]) and Sauces (par_days:14, tags:[Sauces,Condiments]) appended after existing 8 groups. |
| 4 | full_name column exists on purchase_items table | VERIFIED | Migration 0036_purchase_items_full_name.sql in commit 5e8bdd1: ALTER TABLE purchase_items ADD COLUMN full_name TEXT with matching Down. Follows same BEGIN/COMMIT pattern as 0035. |
| 5 | Notion source files (CSV + images) are excluded from git | VERIFIED | .gitignore in commit 5e8bdd1 contains `notion-export/` entry with comment. |
| 6 | Items seed idempotently on every server startup | FAILED | service.go on main branch has NO go:embed for purchase_items.yaml and NO item seeding loop. The extended version exists only in commit e883a0f on worktree-agent-a8c95582 (not merged). |
| 7 | Re-runs do not duplicate items | VERIFIED (in branch) | Commit e883a0f service.go uses ON CONFLICT (description) DO NOTHING for item inserts. But this code is not on main. |
| 8 | Seeded items have photo_url, store_location, and full_name populated | PARTIAL | INSERT in commit e883a0f includes all 5 columns (description, full_name, photo_url, store_location, group_id). purchase_items.yaml is items:[] so no real data flows until import script is run. |

**Score:** 6/8 truths verified (2 failed/partial due to unmerged commits + placeholder YAML)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/internal/db/migrations/0036_purchase_items_full_name.sql` | ALTER TABLE adding full_name column | VERIFIED (branch only) | Exists in commit 5e8bdd1. Correct goose Up/Down with BEGIN/COMMIT. Not on main. |
| `backend/cmd/import-notion/main.go` | One-time conversion script using DO Spaces | VERIFIED (branch only) | 344 lines in commit 6380e47. All required flags (--csv, --images, --output, --dry-run). Category mapping, title case, S3 PutObject, photos.PublicURL. Not on main. |
| `backend/internal/inventory/fixtures/purchase_items.yaml` | Seed data with DO Spaces URLs | STUB | Exists as placeholder: `items: []`. Intentional per D-05 — must run import script to populate. |
| `backend/internal/inventory/fixtures/purchase_item_groups.yaml` | Group fixtures including Seasoning and Sauces | VERIFIED (branch only) | Seasoning and Sauces groups added in commit 5e8bdd1. Not on main (main has 8 groups only). |
| `.gitignore` | Notion source exclusion | VERIFIED (branch only) | notion-export/ entry in commit 5e8bdd1. Not on main. |
| `backend/internal/inventory/service.go` | Extended SeedInventoryFixtures with purchase_items.yaml | VERIFIED (branch only) | Commit e883a0f has: 2x go:embed, itemsFile/itemSeedGroup/itemSeedEntry structs, nilIfEmpty helper, full item seeding loop with ON CONFLICT DO NOTHING. Not on main. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `backend/cmd/import-notion/main.go` | `backend/internal/photos/spaces.go` | S3 client for DO Spaces upload | PARTIAL | Script uses `photos.PublicURL(endpoint, bucket, key)` — it does NOT use `photos.SpacesConfig` or `photos.NewSpacesPresigner` as the key_link pattern specifies. This is the documented D-06 deviation: script uses direct s3.New() + PutObject instead of presigned PUT. The photos package IS imported and used (PublicURL), but the specific pattern in must_haves.key_links does not match. D-06 acknowledges this is intentional. |
| `backend/internal/inventory/service.go` | `backend/internal/inventory/fixtures/purchase_items.yaml` | go:embed | VERIFIED (branch only) | `//go:embed fixtures/purchase_items.yaml` in commit e883a0f. Not on main. |
| `backend/internal/inventory/service.go` | `purchase_items table` | INSERT ON CONFLICT DO NOTHING | VERIFIED (branch only) | `ON CONFLICT (description) DO NOTHING` in item seeding loop in commit e883a0f. Not on main. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `purchase_items.yaml` | items[] | go run ./cmd/import-notion/ | No — placeholder only | STATIC — items: [] until import script is run with Notion export |
| `service.go SeedInventoryFixtures` | items.Items | go:embed purchase_items.yaml | No — placeholder YAML | HOLLOW — wired correctly but data source is empty by design until script runs |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| import-notion compiles | `go build ./cmd/import-notion/` | Cannot run — worktree directory pruned, commits not on main | SKIP — all dependencies confirmed present in go.mod; code structure is syntactically valid per inspection |
| server compiles | `go build ./cmd/server/` | Not testable on main (Phase 15 changes not merged) | SKIP |
| purchase_items.yaml has no Notion URLs | `grep amazonaws purchase_items.yaml` | No matches — file is items: [] | PASS |
| service.go on main has go:embed for items | grep on main service.go | Only 1 go:embed (groups yaml); items yaml NOT embedded | FAIL |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| IMP-01 | 15-01, 15-02 | Notion CSV export converted to YAML seed file following existing fixture patterns | PARTIAL | Script is complete and correct in commits. YAML placeholder exists. Neither merged to main nor populated with real data yet. Script must be run and result committed. |
| IMP-02 | 15-01, 15-02 | Seed re-hosts Notion images to DO Spaces; raw Notion URLs never stored | VERIFIED | import-notion script produces only photos.PublicURL output (DO Spaces). No Notion/amazonaws URLs in any committed file. Mechanically correct — needs real run to prove end-to-end. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `purchase_items.yaml` (commit 6380e47) | 3 | `items: []` | Info | Intentional placeholder per D-05. Not a bug — requires manual import script run. Documents as "DO NOT EDIT" with re-run instructions. |
| Key commits not on main | — | 4 commits on worktree-agent-a8c95582 not merged | Blocker | All Phase 15 work is inaccessible from the main branch and would not be deployed. |

### Human Verification Required

#### 1. Run import-notion and verify DO Spaces URLs

**Test:** Run `go run ./cmd/import-notion/ --csv "/path/to/All Items.csv" --images "/path/to/All Items/" --output "internal/inventory/fixtures/purchase_items.yaml"` with real DO Spaces credentials
**Expected:** purchase_items.yaml populated with ~106 items; all photo_url fields point to `*.digitaloceanspaces.com` URLs; no Notion/amazonaws.com URLs
**Why human:** Requires actual Notion CSV export files and live DO Spaces credentials

#### 2. Server seeds items on startup

**Test:** Start server after merging Phase 15 commits and running import script; check item catalog
**Expected:** Items appear in inventory Setup tab with photo_url, store_location, full_name; second restart does not duplicate items
**Why human:** Requires live DB and populated YAML

#### 3. Photos load after 2+ hours

**Test:** Upload photos via import script, wait 2+ hours, navigate to inventory item list in browser
**Expected:** Photos still load (confirming DO Spaces permanent URLs, not Notion 1-hour-expiry URLs)
**Why human:** Time-dependent; requires real DO Spaces upload

### Gaps Summary

The Phase 15 implementation is **code-complete** and **architecturally correct** in commits on the `worktree-agent-a8c95582` branch. Four commits (5e8bdd1, 6380e47, c9dfda7, e883a0f) contain all required artifacts:

- Migration 0036 (full_name column) — correct
- purchase_item_groups.yaml with Seasoning and Sauces — correct
- .gitignore with notion-export/ — correct
- import-notion/main.go (344 lines, all flags, category mapping, DO Spaces upload) — correct
- service.go extended with dual go:embed, new structs, nilIfEmpty, item seeding loop — correct

**Two gaps block goal achievement:**

1. **Unmerged commits**: All 4 Phase 15 commits are on `worktree-agent-a8c95582` and have NOT been merged into `main`. The main branch is missing the migration, the group additions, the gitignore entry, the import script, and the service.go extension. Deploy would not include any Phase 15 changes.

2. **Placeholder YAML**: `purchase_items.yaml` intentionally contains `items: []` (D-05). The import-notion script must be run with the Notion CSV export to generate real data. Until this is done and the populated YAML committed, the seed produces 0 items and the phase goal ("convert Notion CSV into a seed file") is not met end-to-end.

**The key_link deviation (D-06)** — script uses `photos.PublicURL` directly rather than `photos.SpacesConfig`/`NewSpacesPresigner` — is documented and intentional. Direct PutObject is the correct pattern for a batch CLI tool; presigned URLs are for browser-side uploads. Not a gap.

---

_Verified: 2026-04-22T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
