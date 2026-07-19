# Phase 8: Trends and Cost Intelligence Tabs - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-04-14
**Phase:** 08-trends-and-cost-intelligence-tabs
**Areas discussed:** Chart types & layout, Tag filter UX, Cost Intelligence views, Metabase swap architecture

---

## Chart Types & Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Stacked vertical | All 3 charts scrollable vertically | |
| Tab within tab | Sub-tabs: By Category / Over Time | ✓ |
| You decide | Claude picks | |

**User's choice:** Sub-tabs inside Trends tab.

---

## Tag Filter UX

| Option | Description | Selected |
|--------|-------------|----------|
| Tappable chip bar | Multi-select chips above charts | ✓ |
| Dropdown | Single select, consistent with History vendor filter | |
| You decide | Claude picks | |

**User's choice:** Tappable chip bar with multi-select.

---

## Cost Intelligence Views

| Option | Description | Selected |
|--------|-------------|----------|
| Sub-tabs: Menu Items / Ingredients | Two sub-tabs, each with expandable cards | ✓ |
| Single scrollable list | Both sections in one scroll | |
| You decide | Claude picks | |

**User's choice:** Sub-tabs for clean separation.

---

## Metabase Swap Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Container div + config flag | Runtime switching between native and iframe | |
| Just a TODO comment | Comment only, manual replacement later | ✓ |
| You decide | Claude picks | |

**User's choice:** TODO comment only — no runtime switching infrastructure.

---

## Claude's Discretion

- Chart color palette (CSS variables for dark mode)
- Bar chart orientation
- Timeline granularity (weekly vs monthly)
- Return on purchase display format
- Chip styling
- Chart.js tooltip config
- Sale price field on MOCK_MENU_ITEMS

## Deferred Ideas

- Metabase iframe embed — TODO only
- AI-assisted cost estimation — backend
- CSV export — future
- Date range picker — future
