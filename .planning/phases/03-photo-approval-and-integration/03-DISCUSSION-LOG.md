# Phase 3: Photo, Approval, and Integration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 03-photo-approval-and-integration
**Areas discussed:** Photo capture UX, Approval flow, Pre-built templates

---

## Photo Capture UX

| Option | Description | Selected |
|--------|-------------|----------|
| Native camera | `<input type="file" capture="environment">`, opens rear camera | ✓ |
| Camera + gallery choice | Standard iOS/Android picker | |
| You decide | Claude's discretion | |

**User's choice:** Native camera

---

| Option | Description | Selected |
|--------|-------------|----------|
| Thumbnail + re-capture | Small 80x80 thumbnail inline with retake | |
| Full-width preview | Photo spans full card width | |
| Custom two-step | Full preview first, then thumbnail after confirm | ✓ |

**User's choice:** Full-size preview with checkmark/retake → after confirm, small thumbnail inline with retake
**Notes:** User specified a two-step flow not in the original options

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, make functional | Fail card photo stub becomes real camera | ✓ |
| Keep as stub | Leave as "Coming soon" | |

**User's choice:** Yes, make it functional

---

## Approval Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Separate tab | Third tab "Approvals" between My Checklists and Builder | ✓ |
| Badge on My Checklists | Pending section at top of existing tab | |
| You decide | Claude's discretion | |

**User's choice:** Separate tab

---

| Option | Description | Selected |
|--------|-------------|----------|
| Read-only runner | Full runner view, all fields read-only | |
| Summary card | Condensed card with key info + photos | ✓ |
| You decide | Claude's discretion | |

**User's choice:** Summary card — with photos included
**Notes:** User specified photos should be visible in the summary

---

| Option | Description | Selected |
|--------|-------------|----------|
| Toast + status change | Simple toast, no notes | |
| Toast + notes on reject | Approval one-tap, rejection requires reason | ✓ |
| You decide | Claude's discretion | |

**User's choice:** Toast + notes required on reject

---

## Pre-Built Templates

| Option | Description | Selected |
|--------|-------------|----------|
| Opening + HACCP + Closing | Three templates with food safety focus | |
| Opening + Closing only | Two daily bookend checklists | |
| Custom | User specified | ✓ |

**User's choice:** Setup Checklist + Closing Checklist
**Notes:** User renamed "Opening" to "Setup"

---

| Option | Description | Selected |
|--------|-------------|----------|
| Realistic | Real food truck items | |
| Minimal placeholders | Generic items, enough to show feature | ✓ |

**User's choice:** Minimal placeholders

---

## Claude's Discretion

- Summary card layout and information density
- Tab ordering
- "Requires approval" flag connection to queue
- Photo thumbnail size and retake styling

## Deferred Ideas

None — discussion stayed within phase scope.
