---
phase: 02-fill-out-and-conditional-logic
plan: 02
subsystem: fill-out-runner
tags: [fail-triggers, corrective-action, yes-no, temperature, submit, css]
dependency_graph:
  requires: [02-01]
  provides: [fail-triggers, corrective-action-cards, submit-confirmation]
  affects: [workflows.html]
tech_stack:
  added: []
  patterns: [event-delegation, no-rerender-on-text-input, inline-fail-card]
key_files:
  created: []
  modified:
    - workflows.html
    - sw.js
decisions:
  - "Kept existing MOCK_RESPONSES flat dict and added FAIL_NOTES dict alongside it — no restructuring needed"
  - "Named mutation functions (setYesNo, setTextAnswer, setSeverity, submitChecklist) added for traceability"
  - "submit shows inline confirm + disables button; fireworks only on all-done path"
  - "temp-input handler re-renders for fail card reactivity, restores focus+cursor after re-render"
metrics:
  duration: "~8 min"
  completed: "2026-04-13"
  tasks: 1
  files: 2
---

# Phase 02 Plan 02: Fail Triggers, Corrective Action Cards, and Submit Summary

**One-liner:** Inline fail detection for yes/no "No" answers and out-of-range temperature, corrective action cards with severity picker and photo stub, and inline submit confirmation.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Yes/no, text, temperature fields, fail triggers, corrective action, and submit | bc88068 | workflows.html, sw.js |

## Task 2 — PENDING HUMAN VERIFICATION

Task 2 is a `checkpoint:human-verify` gate. No code changes in Task 2 — user must verify in browser.

See verification steps in `.planning/phases/02-fill-out-and-conditional-logic/02-02-PLAN.md` Task 2 `<how-to-verify>` section.

## What Was Built

### evaluateFailTrigger(fail_trigger, value)

Pure function. Returns `false` for empty/null/undefined values (Pitfall 7 prevention). For `out_of_range` type, checks `value < min || value > max` only when both sides are set and value is a valid number.

### renderFailCard(fld, failNote)

Renders the `.fail-card` inline card below triggering fields. Structure:
1. "Corrective action required" header (12px, #c0392b, weight 500)
2. Textarea with `data-action="fail-note-input"` (no re-render on input)
3. Photo stub: disabled button + "Coming in Phase 3" label
4. Severity pills: Minor/Major/Critical with `.on` class for active state

### Yes/No Field Updates

Replaced `.yn-btn` buttons with `.yn-pill` buttons using `data-action="set-yes"` / `data-action="set-no"` (Pitfall 6: no full-row toggle, only pills are tap targets). "No" answer renders `.fail-card` below field. "Yes" removes fail card and clears `FAIL_NOTES[fldId]`.

### Temperature Field Updates

Added `data-action="temp-input"` to the number input. Input handler re-renders the field for fail card reactivity, then restores focus + cursor position. Out-of-range shows `.temp-warn` amber label and `.fail-card`.

### Submit Confirmation

`submitChecklist(tplId)` sets `SUBMITTED_TEMPLATES[tplId] = true` and re-renders. Runner shows disabled button + `.submit-confirm` span: "Checklist submitted. Thanks, JM!". Fireworks fire on all-done path.

### Named Mutation Helpers

Added: `setYesNo`, `setTextAnswer`, `setFailNote`, `setSeverity`, `submitChecklist` — wired into event delegation, not called from render functions.

### CSS Added

`.fail-card`, `.yn-pills`, `.yn-pill`, `.yn-pill.on`, `.severity-pills`, `.severity-pill`, `.severity-pill.on`, `.temp-warn`, `.temp-input-row`, `.photo-stub-btn`, `.submit-confirm`, `.fail-note-input` — all with dark mode support for `.fail-card`.

### SW Cache

Bumped from v16 to v17.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing] Named mutation functions added**

- **Found during:** Task 1
- **Issue:** The plan described `setYesNo`, `setTextAnswer`, `setSeverity`, `submitChecklist` as mutation functions. The existing code used inline logic. Added named stubs that also serve as the actual implementation to satisfy plan verification grep check and improve traceability.
- **Fix:** Added all named functions before the `evaluateFailTrigger` function; they are called from event handlers.
- **Files modified:** workflows.html
- **Commit:** bc88068

**2. [Rule 1 - Existing code] Kept MOCK_RESPONSES flat dict**

- **Found during:** Task 1
- **Issue:** Plan 02-02 context described `fillState.responses[tplId]` structure, but existing code (added after Plan 02-01 by the user) uses `MOCK_RESPONSES` flat dict. Restructuring would break existing checkbox/text/temp behavior.
- **Fix:** Added `FAIL_NOTES` dict alongside `MOCK_RESPONSES` and `SUBMITTED_TEMPLATES` set. No state migration needed for mock data.
- **Files modified:** workflows.html
- **Commit:** bc88068

## Known Stubs

- Photo capture in `.fail-card` is a disabled button stub with "Coming in Phase 3" label. This is intentional per plan spec — Phase 3 will implement real photo capture.

## Self-Check: PASSED

- workflows.html: FOUND
- sw.js: FOUND
- 02-02-SUMMARY.md: FOUND
- commit bc88068: FOUND
