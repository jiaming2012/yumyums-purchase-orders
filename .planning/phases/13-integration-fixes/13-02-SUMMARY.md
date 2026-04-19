---
phase: 13-integration-fixes
plan: "02"
subsystem: onboarding-ui, workflows-photos
tags: [onboarding, manager-tab, template-assignment, fail-photos, presign, persistence, regression-test]
dependency_graph:
  requires: []
  provides: [onboarding-template-assignment-ui, fail-photo-presign-pipeline]
  affects: [onboarding.html, workflows.html, tests/persistence.spec.js]
tech_stack:
  added: []
  patterns: [presign-put-upload, data-action-event-delegation, hydrateFieldState-photo-restore]
key_files:
  created: []
  modified:
    - onboarding.html
    - workflows.html
    - tests/persistence.spec.js
    - sw.js
decisions:
  - Hire detail view replaces auto-open-first-template for better discoverability
  - Photo URL included in _fail_note bundle (autoSaveField + hydrateFieldState) for persistence
  - Test debounce timing: wait 600ms before injecting photo to let initial No-answer save fire first
metrics:
  duration_seconds: 583
  completed_date: "2026-04-19"
  tasks_completed: 3
  files_modified: 4
---

# Phase 13 Plan 02: Integration Fixes (Onboarding Template Assignment + Fail Photo Pipeline) Summary

**One-liner:** Template assignment UI in Manager hire-detail view + fail card photos routed through presign-to-Spaces pipeline with photo URL restored on reopen.

## What Was Built

### Task 1: Template Assignment UI in Manager Tab

Added a hire detail drill-down view between the hire list and the training runner in the Manager tab of `onboarding.html`.

- **`renderHireDetail(hire)`**: Shows assigned templates with progress bars, "View Training" and "Unassign" buttons for each, plus an "+ Assign Template" button
- **`renderAssignPicker(hireId, assignedIds)`**: Shows available templates (not yet assigned), each with an "Assign" button
- **Updated `open-hire` handler**: Now navigates to hire detail view instead of auto-opening the first template
- **New action handlers**: `view-training`, `show-assign-picker`, `assign-template`, `unassign-template`, `back-to-hire-detail`
- Assign/unassign both call backend APIs and refresh `MANAGER_HIRES` afterward

### Task 2: Fail Card Photos Through Presign Pipeline

Replaced the broken blob: URL implementation of `handleFailPhotoCaptureClick` in `workflows.html` with a version that mirrors `handlePhotoCaptureClick`.

- Shows photo preview with confirm/retake UX
- Shows "Uploading..." spinner during upload
- Calls `/api/v1/photos/presign` with `path_prefix: 'checklists'` and filename `fail-{fldId}.jpg`
- PUTs the file to the presigned URL
- Stores `presignResp.public_url` (https://) in `FAIL_NOTES[fldId].photo` (not blob:)
- Calls `autoSaveField` after upload to persist the fail note (with photo URL) to backend
- Error handling re-renders the fail card to allow retry

### Task 3: Regression Test — Fail Photo Persistence

Added `'fail photo survives back-to-list and reopen as https:// URL'` to `tests/persistence.spec.js` under "Draft response persistence" section.

- Creates template with yes/no field
- Clicks "No" to trigger the fail card
- Waits 600ms for initial debounce to fire (important: prevents overwrite)
- Injects photo URL via direct API saveResponse call
- Goes back to list and reopens
- Asserts `img.photo-thumb` src starts with `https://` not `blob:`
- Test passes (1 passed, 5.9s)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] hydrateFieldState did not restore photo from _fail_note on reopen**
- **Found during:** Task 3 (test investigation)
- **Issue:** `hydrateFieldState` at line 1343 only restored `note` and `severity` from `_fail_note` — `photo` was silently dropped, so fail card photos never showed after back-and-reopen even with the presign pipeline in place
- **Fix:** Added `photo: val._fail_note.photo || null` to the FAIL_NOTES assignment in `hydrateFieldState`
- **Files modified:** `workflows.html` (line 1343)
- **Commit:** c9f86fe

**2. [Rule 1 - Bug] autoSaveField did not include photo in bundled _fail_note**
- **Found during:** Task 3 (test investigation)
- **Issue:** `autoSaveField` at line 202 only included `note` and `severity` in the `_fail_note` bundle, never `photo` — so even if photo was set in `FAIL_NOTES[fldId].photo`, it wasn't persisted to backend
- **Fix:** Added `photo: fn.photo || null` to the `_fail_note` bundle in `autoSaveField`, and changed the trigger condition from `fn.note || fn.severity` to `fn.note || fn.severity || fn.photo`
- **Files modified:** `workflows.html` (lines 202-203)
- **Commit:** c9f86fe

## Known Stubs

None — all photo URLs are real presigned Spaces https:// URLs after upload.

## Self-Check: PASSED

- FOUND: onboarding.html
- FOUND: workflows.html
- FOUND: tests/persistence.spec.js
- FOUND: .planning/phases/13-integration-fixes/13-02-SUMMARY.md
- FOUND commit df733ca: feat(13-02): add template assignment/unassignment UI in Manager tab
- FOUND commit 57adc08: feat(13-02): wire fail card photos through presign-to-Spaces pipeline
- FOUND commit c9f86fe: feat(13-02): regression test for fail photo persistence + fix hydration bug
