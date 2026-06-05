# Phase 1: Template Builder - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 01-template-builder
**Areas discussed:** Builder layout, Field configuration UX, Drag-reorder interaction, Template data shape

---

## Builder Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Single scroll | One scrollable page showing sections + fields inline. Matches existing tab pattern. | ✓ |
| Split view | Left side shows structure, right side shows editor. More desktop-oriented. | |
| You decide | Claude's discretion | |

**User's choice:** Single scroll
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| + Add button per section | Each section has '+ Add field' button that opens field type picker | ✓ |
| Floating action button | Single FAB at bottom of screen | |
| You decide | Claude's discretion | |

**User's choice:** + Add button per section
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Collapsible cards | Each section is a card that can collapse/expand | |
| Flat headers | Sections are bold headers with divider line, matches purchasing.html | ✓ |
| You decide | Claude's discretion | |

**User's choice:** Flat headers
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Editor only | No preview pane, fill-out view is the preview | ✓ |
| Side-by-side preview | Mini preview pane next to editor | |
| Toggle preview mode | Button to switch between edit and preview | |

**User's choice:** Editor only
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Top of page | Template name + approval toggle at the very top | ✓ (preferred) |
| Settings drawer | Gear icon that opens template-level settings | |

**User's choice:** "Let Claude decide: prefer top of page if there is enough space"
**Notes:** Deferred to Claude's discretion with preference for top of page

---

## Field Configuration UX

| Option | Description | Selected |
|--------|-------------|----------|
| Tap to expand | Tap field row to expand inline settings panel below it | ✓ |
| Slide-up sheet | Bottom sheet with settings | |
| You decide | Claude's discretion | |

**User's choice:** Tap to expand
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Simple dropdowns | 'Show this field only when [field] is [value]' — two dropdowns | ✓ |
| Natural language style | Readable sentence with tappable parts | |
| You decide | Claude's discretion | |

**User's choice:** Simple dropdowns
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Toggle chips | Row of day abbreviations as toggleable chips | ✓ |
| Checkboxes | Standard checkbox list with full day names | |
| You decide | Claude's discretion | |

**User's choice:** Toggle chips: M T W T F Sa Su
**Notes:** User specified two-letter abbreviations to avoid T/T ambiguity

---

## Drag-Reorder Interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Drag handle | Grip icon (☰) on left side, uses SortableJS | ✓ |
| Up/down buttons | Arrow buttons, no drag library needed | |
| Long-press to drag | Long-press to enter drag mode, cleaner look | |

**User's choice:** Drag handle
**Notes:** None

---

## Template Data Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Match backend schema | UUID field IDs, nested sections, typed fields | ✓ |
| Keep it simple | Simple JS arrays like purchasing.html | |
| You decide | Claude's discretion | |

**User's choice:** Match backend schema
**Notes:** None

---

| Option | Description | Selected |
|--------|-------------|----------|
| Editable in memory | Changes work during session, reset on refresh | ✓ |
| LocalStorage persist | Changes survive refresh | |
| Read-only demo | Templates hardcoded and can't be modified | |

**User's choice:** Editable in memory
**Notes:** None

---

## Claude's Discretion

- Template name/approval toggle placement (prefer top of page if space allows)

## Deferred Ideas

None — discussion stayed within phase scope.
