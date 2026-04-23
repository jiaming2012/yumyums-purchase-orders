# Phase 14: PO Backend + Order Form - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-22
**Phase:** 14-po-backend-order-form
**Areas discussed:** PO form layout, Reorder flow, Added-by + dupes, Tab structure

---

## PO Form Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Photo card | Photo thumbnail + name + store + stepper. ~80px per item | ✓ |
| Compact row | Name + unit + stepper in one row, photo on tap. ~44px per item | |
| Hybrid | Compact rows, photo when available, tap to expand | |

**User's choice:** Photo card
**Notes:** Items on the PO should be grouped by category. Follow-up confirmed sticky category headers (matches existing patterns).

---

## Reorder Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-populate | Items below threshold auto-added to PO | |
| Suggestion banner | Banner with count, tap opens checklist to select which to add | ✓ |
| Inline section | Reorder section at top of form, tap item to add | |

**User's choice:** Suggestion banner
**Notes:** More intentional — user selects which suggestions to add rather than auto-populating.

---

## Added-by + Duplicates

| Option | Description | Selected |
|--------|-------------|----------|
| Block + show existing | Prevent adding, show toast with qty and who added | |
| Badge in picker | Show "On PO: qty" badge, tapping increments | |
| Both | Badge in picker AND popup on tap with details | ✓ |

**User's choice:** Both (Option C)
**Notes:** User initially asked for detailed example — all three options were explained with concrete Maria/Jamal scenario before selection.

---

## Tab Structure

User provided specific vision rather than selecting from options:

| Tab | Purpose |
|-----|---------|
| Order | Always open. After cutoff, fresh list starts for next week |
| Shopping | Current week's approved shopping list, organized by vendor |
| PO | Cutoff list, admin-editable, vendor-grouped pricing, approve button |
| History | Past shopping lists with missing items and who completed |

**Tab order:** Order / Shopping / PO / History (user selected)

**Additional decisions from discussion:**
- Shopping list has one submit button per vendor section (supports multiple shoppers)
- History shows who completed each shopping list
- Admin blocked from approving if previous shopping list still active — shows message
- PO tab pricing matches current mock vendor-grouped layout

---

## Claude's Discretion

- Empty state design for Order tab
- Stepper button styling
- Search input placement and debounce in item picker
- Skeleton/loading states

## Deferred Ideas

None — discussion stayed within phase scope
