# Phase 5: Onboarding Builder - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 05-onboarding-builder
**Areas discussed:** Builder layout, Item type picker, Section configuration, Template management

---

## Builder Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror workflows builder (Recommended) | Same pattern: flat sections, inline items, tap to expand, + Add Item | ✓ |
| Card-based sections | Collapsible cards per section | |
| You decide | Claude picks | |

**User's choice:** Mirror workflows builder pattern.

### Follow-up: Drag Reorder

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, drag reorder (Recommended) | SortableJS for items and sections | ✓ |
| No drag, up/down arrows | Arrow buttons instead | |
| You decide | Claude picks | |

**User's choice:** Yes, SortableJS drag-to-reorder.

---

## Item Type Picker

**User's input:** Checkbox (with sub-items similar to operations/workflows) + Video Series. Two item types only.
**Notes:** User specifically requested sub-items on checkboxes, referencing the operations/workflows sub-steps feature.

### Follow-up: Video Series Editing

| Option | Description | Selected |
|--------|-------------|----------|
| Inline part list | Parts expand inline with editable title/desc/URL. + Add Part button. | ✓ |
| Modal editor per part | Tap to open overlay for editing parts | |
| You decide | Claude picks | |

**User's choice:** Inline part list.

---

## Section Configuration

| Option | Description | Selected |
|--------|-------------|----------|
| Inline toggles on section header | SO and FAQ toggles directly on header row | ✓ |
| Tap section to expand settings | Settings panel behind a gear icon | |
| You decide | Claude picks | |

**User's choice:** Inline toggles. When FAQ is toggled on, section switches to Q&A editor.

---

## Template Management

**User's multi-select:** Create new, Edit existing, Delete. Did NOT select Duplicate.

### Follow-up: Template List

| Option | Description | Selected |
|--------|-------------|----------|
| Simple list with + New | Card per template, name/role/sections, + New Template button | ✓ |
| You decide | Claude picks | |

**User's choice:** Simple list with + New Template.

---

## Claude's Discretion

- Template name input and role selector placement
- Delete button and confirmation dialog style
- Empty state for builder tab
- Sub-item visual nesting in builder
- Drag handle styling

## Deferred Ideas

- Template duplication — future enhancement
- Template versioning — out of scope for mocks
