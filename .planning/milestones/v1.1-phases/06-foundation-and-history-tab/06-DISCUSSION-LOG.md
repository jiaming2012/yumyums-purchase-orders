# Phase 6: Foundation and History Tab - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 06-foundation-and-history-tab
**Areas discussed:** History tab UX, Mock data design, Tab layout & navigation, HQ tile placement

---

## History Tab UX

| Option | Description | Selected |
|--------|-------------|----------|
| Card list with expand | Card per event, tap to expand line items | ✓ |
| Compact table | Dense table with columns, tap row to expand | |
| You decide | Claude picks | |

**User's choice:** Card list with expand.

### Follow-up: Vendor Filter

| Option | Description | Selected |
|--------|-------------|----------|
| Horizontal chip bar | Tappable chips at top, scrollable | |
| Dropdown select | Single dropdown, cleaner but extra tap | ✓ |
| You decide | Claude picks | |

**User's choice:** Dropdown select.

---

## Mock Data Design

| Option | Description | Selected |
|--------|-------------|----------|
| Use real Baserow data | Export actual DB into JS constants | |
| Fabricate realistic data | Invent events with realistic items | |
| Mix: real schema, fabricated values | Baserow structure with fabricated values for 3+ months | ✓ |

**User's choice:** Mix approach. Focus on UI realism, data can be sourced later. Use Baserow for accuracy in determining realistic layouts.

---

## Tab Layout & Navigation

### Tab Access

| Option | Description | Selected |
|--------|-------------|----------|
| All visible for now | No RBAC gating in mock, add later | ✓ |
| Gate Trends + Cost to manager+ | Shows intended RBAC design | |
| You decide | Claude picks | |

**User's choice:** All visible. Add note for backend to add RBAC later.

### Placeholders

| Option | Description | Selected |
|--------|-------------|----------|
| Coming Soon message | Centered message describing upcoming feature | ✓ |
| Empty with tab name only | Blank content area | |
| You decide | Claude picks | |

**User's choice:** Coming Soon messages.

---

## HQ Tile Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Replace BI 'Soon' tile | BI becomes Inventory | |
| Keep BI, add new tile | 8th tile, grid stays 4×2 | ✓ |
| You decide | Claude picks | |

**User's choice:** Keep BI as Soon, add Inventory as 8th tile with 📦 emoji.

---

## Claude's Discretion

- Card styling for purchase events
- Case badge display on line items
- Coming Soon placeholder styling
- Mock data item names and prices
- Chart.js download method

## Deferred Ideas

- Tab RBAC gating — backend later
- Metabase iframe embed — architecture in place, not activated
- Real data sourcing — mock for now
