# Phase 16: Cutoff, Approval, and Shopping List - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-04-22
**Phase:** 16-cutoff-approval-and-shopping-list
**Areas discussed:** Cutoff config UX, PO lock + approval, Shopping check-off, State machine

---

## Cutoff Config UX

| Option | Description | Selected |
|--------|-------------|----------|
| PO tab header | Admin taps cutoff pill to edit day/time inline | ✓ |
| Order tab header | Same but on Order tab | |
| Separate settings | Gear icon opens config panel | |

**User's choice:** PO tab header

**Test simulation:**

| Option | Description | Selected |
|--------|-------------|----------|
| API endpoint | POST /api/v1/purchasing/simulate-cutoff — same production logic | ✓ |
| CLI command | Standalone script | |

**User's notes:** Same endpoint used in production. Truncate DB to test again.

---

## PO Lock + Approval

| Option | Description | Selected |
|--------|-------------|----------|
| Read-only cards + admin edit | Photo cards, disabled steppers, admin Edit toggle, vendor totals, Approve button | ✓ |
| Compact list view | Simpler read-only list | |

---

## Shopping Check-off

| Option | Description | Selected |
|--------|-------------|----------|
| Bottom sheet prompt | Slide-up card, skip requires confirm | |
| Inline expand | Card expands with photo/location fields | |
| Toast + badge | Item checks off immediately, toast prompts, badge shows missing info | ✓ |

**User's notes:** Non-blocking — toast can be dismissed, badge persists as reminder.

---

## State Machine

| Option | Description | Selected |
|--------|-------------|----------|
| Linear flow | No reverse transitions | |
| With admin unlock | Admin can unlock locked → draft, but not after approval | ✓ |

**Partial completion:**

| Option | Description | Selected |
|--------|-------------|----------|
| Per-vendor status | Each vendor section independent, overall complete when all done | ✓ |
| All-or-nothing | One Complete button for entire list | |

---

## Claude's Discretion

- Scheduler implementation pattern
- Shopping list snapshot schema
- History tab sort order
- Approve button animation
- Toast auto-dismiss timing

## Deferred Ideas

- Alert notifications — Phase 17
- Repurchase badges — Phase 17
