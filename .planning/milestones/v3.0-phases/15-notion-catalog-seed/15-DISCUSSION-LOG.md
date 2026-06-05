# Phase 15: Notion Catalog Seed - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-22
**Phase:** 15-notion-catalog-seed
**Areas discussed:** Category mapping, Photo handling, Item name format, Seed vs import

---

## Category Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-map + new groups | Map obvious matches, create Seasoning and Sauces groups | ✓ |
| Merge into existing | Force-fit all into 8 existing groups | |
| You decide | Claude's discretion | |

**User's choice:** Auto-map + new groups
**Notes:** Seasoning and Sauces are new groups. Cooking Supplies maps to Packaging.

---

## Photo Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Conversion script | Go script uploads to DO Spaces, writes YAML with URLs | ✓ |
| Embed images in repo | Copy PNGs into fixtures/, upload on seed | |
| Skip photos for now | Seed without photos | |

**User's choice:** Conversion script
**Notes:** Notion source files should be copied into the project but added to .gitignore. Generated YAML with DO Spaces URLs is committed.

---

## Item Name Format

| Option | Description | Selected |
|--------|-------------|----------|
| Short name (Name) | "Brisket Rolls" as description, full name as separate field | ✓ |
| Full name | "4\" Brioche Roll - 12 ct Pack" as description | |
| Full name with short fallback | Full name primary, short name when full is empty | |

**User's choice:** Short name with full name fallback
**Notes:** Short name → description, full name → full_name field. Use full name as fallback when short name is empty.

---

## Seed vs Import

| Option | Description | Selected |
|--------|-------------|----------|
| YAML seed (Recommended) | Embedded go:embed, runs on every startup idempotently | ✓ |
| Separate seed command | Only seed when explicitly running task seed | |

**User's choice:** YAML seed

---

## Claude's Discretion

- DO Spaces key path for seed photos
- Image normalization/resizing
- Items with no category → default group
- Conversion script CLI flag names

## Deferred Ideas

None
