# Phase 7: Stock and Reorder Tab - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 07-stock-and-reorder-tab
**Areas discussed:** Stock level display, Estimation algorithm, Reorder suggestions UX, Manual override flow

---

## Stock Level Display

| Option | Description | Selected |
|--------|-------------|----------|
| Card list by tag category | Items grouped under tag headers with badges | ✓ |
| Flat list sorted by urgency | All items in one list, Low first | |
| You decide | Claude picks | |

**User's choice:** Card list by tag category.

### Follow-up: Item Detail

| Option | Description | Selected |
|--------|-------------|----------|
| Inline expand with last purchase info | Shows last purchased, vendor, frequency, price, override button | ✓ |
| No detail view | Just badge + override button on row | |
| You decide | Claude picks | |

**User's choice:** Inline expand with purchase details.

---

## Estimation Algorithm

| Option | Description | Selected |
|--------|-------------|----------|
| Ratio-based (time) | daysSince / avgInterval | |
| Fixed day thresholds | Same for all items | |
| You decide | Claude picks | |

**User's choice:** Ratio-based, BUT based on orders sold and ingredient ratios, NOT time-based. Stock = purchased - consumed (sales × ingredient ratios).

### Follow-up: Mock Data

| Option | Description | Selected |
|--------|-------------|----------|
| Add mock sales data | MOCK_SALES with weekly counts, derive consumed from ingredient ratios | ✓ |
| Time-based for now, design for sales-based | Simpler heuristic, UI ready for real data later | |
| You decide | Claude picks | |

**User's choice:** Add mock sales data and implement the real algorithm.

---

## Reorder Suggestions UX

| Option | Description | Selected |
|--------|-------------|----------|
| Separate section at top | Highlighted section with Low/Medium items | |
| Inline badges only | +PO badge next to items | |
| You decide | Claude picks | ✓ |

**User's choice:** Claude's discretion.

---

## Manual Override Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Inline level picker + reason | Radio buttons + reason text + Save in expanded detail | ✓ |
| Tap badge to cycle levels | Cycle through levels on tap, prompt for reason | |
| You decide | Claude picks | |

**User's choice:** Inline level picker with reason input.

---

## Claude's Discretion

- Reorder suggestions layout
- Exact stock formula and thresholds
- Quantity display format
- Badge color implementation
- MOCK_SALES data structure

## Deferred Ideas

- Backend journal persistence for overrides
- Purchasing app integration for actual reorders
- isCase weighting in consumption
