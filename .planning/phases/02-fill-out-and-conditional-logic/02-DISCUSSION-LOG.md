# Phase 2: Fill-Out and Conditional Logic - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 02-fill-out-and-conditional-logic
**Areas discussed:** Checklist completion UX, Fail trigger behavior, User attribution display, Day filtering & checklist list

---

## Checklist Completion UX

| Option | Description | Selected |
|--------|-------------|----------|
| Tap the whole row | Large touch target, fast for kitchen use | ✓ |
| Tap a checkbox | Explicit checkbox, smaller target | |
| Swipe to complete | Satisfying gesture but less discoverable | |

**User's choice:** Tap the whole row

---

| Option | Description | Selected |
|--------|-------------|----------|
| Muted with checkmark | Text dims, green checkmark appears | |
| Strikethrough | Line through text, stays in place | ✓ |
| Collapse to one line | Shrinks completed items | |

**User's choice:** Strikethrough with green checkmark to the right

---

| Option | Description | Selected |
|--------|-------------|----------|
| Inline input | Text/number input directly in the row | ✓ |
| Tap to open input | Expand area below the row | |

**User's choice:** Inline input

---

## Fail Trigger Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Inline slide-down | Red card slides down below field, doesn't block | ✓ |
| Full-width banner | Red banner at top of screen | |
| You decide | Claude's discretion | |

**User's choice:** Inline slide-down

---

| Option | Description | Selected |
|--------|-------------|----------|
| Text note only | Just a text input | |
| Text + photo | Text plus photo capture stub | |
| Text + severity | Text plus severity picker | |
| All three | Text + photo + severity | ✓ |

**User's choice:** All three options (text note + photo stub + severity picker)

---

## User Attribution Display

| Option | Description | Selected |
|--------|-------------|----------|
| Initials + time | 'JM · 2:15p' badge, compact | ✓ |
| Full name + time | 'Jamal M. · 2:15 PM' below label | |
| Initials only | Just 'JM' badge, no timestamp | |

**User's choice:** Initials + time

---

| Option | Description | Selected |
|--------|-------------|----------|
| User picker dropdown | Switch between mock users for demo | |
| Hardcoded single user | Always 'Jamal M.' | ✓ |

**User's choice:** Hardcoded single user
**Notes:** User asked for explanation of the feature before deciding

---

## Day Filtering & Checklist List

| Option | Description | Selected |
|--------|-------------|----------|
| Card per checklist | Name, section count, progress bar | ✓ |
| Simple list rows | Flat list with arrow | |
| You decide | Claude's discretion | |

**User's choice:** Card per checklist

---

| Option | Description | Selected |
|--------|-------------|----------|
| Friendly message | 'No checklists for today. Enjoy your day off!' | ✓ |
| Show all with filter | Grey out non-today checklists | |
| You decide | Claude's discretion | |

**User's choice:** Friendly message

---

| Option | Description | Selected |
|--------|-------------|----------|
| Both | Progress on list card + inside checklist | ✓ |
| Inside only | Progress only when checklist is open | |
| You decide | Claude's discretion | |

**User's choice:** Both

---

## Claude's Discretion

- Progress bar styling and placement
- Section rendering style in fill-out mode
- Submit button behavior (mock confirmation)

## Deferred Ideas

None — discussion stayed within phase scope.
