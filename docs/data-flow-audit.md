# Workflows Data Flow Audit

Last updated: 2026-08-04 (card A2, run `20260804` — B-65: every save-path cell below named a function that does not exist)

## User-Entered State Variables

| # | Variable | Type | User enters via | Persisted? | Save path | Restore path | Status |
|---|----------|------|-----------------|------------|-----------|--------------|--------|
| 1 | `FIELD_RESPONSES[fieldId].value` | checkbox boolean | Tap checkbox | Yes | `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 2 | `FIELD_RESPONSES[fieldId].value` | yes/no boolean | Tap Yes/No pill | Yes | `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 3 | `FIELD_RESPONSES[fieldId].value` | text string | Type in textarea | Yes | `debouncedSaveField` (on blur) → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 4 | `FIELD_RESPONSES[fieldId].value` | temperature number | Type/spin number | Yes | `debouncedSaveField` (on change) → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 5 | `FIELD_RESPONSES[fieldId].value` | sub-step state | Tap sub-step checks | Yes | `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 6 | `FAIL_NOTES[fieldId].note` | string | Type in fail card textarea | Yes | Bundled by `debouncedSaveField` as `{_v, _fail_note}` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_fail_note` → `FAIL_NOTES` | OK |
| 7 | `FAIL_NOTES[fieldId].severity` | string | Tap Minor/Major/Critical pill | Yes | Bundled by `debouncedSaveField` as `{_v, _fail_note}` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_fail_note` → `FAIL_NOTES` | OK |
| 8 | `FAIL_NOTES[fieldId].photo` | https:// URL | Camera capture on a fail card | Yes | presign → S3 PUT → bundled by `debouncedSaveField` as `{_v, _fail_note}` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_fail_note.photo` → `FAIL_NOTES` | OK (was silently broken until B-65 — the chain called the phantom `autoSaveField` and its own `.catch()` ate the ReferenceError) |
| 8b | `CORRECTION_PHOTOS[fldId]` | https:// URL | Camera capture to satisfy a `require_photo` rejection | Yes | presign → S3 PUT → bundled by `debouncedSaveField` as `{_v, _correction_photo}` → `POST /ops` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_correction_photo` → `CORRECTION_PHOTOS` | OK |
| 9 | `REJECTION_FLAGS[fldId]` | object | Manager flags field for rejection | No | Server-side via `POST /rejectItem` | Never loaded from server into `REJECTION_FLAGS` | BUG (separate issue) |
| 10 | `WAS_REJECTED[tplId]` | boolean | N/A (should be derived) | No | Never written | Never read meaningfully | DEAD CODE |
| 11 | `state.activeTemplate` | object | Builder editor edits | Yes | `POST /createTemplate` or `PUT /updateTemplate` on save | `GET /templates` → `fromApiTemplate` | OK |
| 12 | `fillState.activeTemplate` | object | Selecting a checklist to fill | No (transient) | N/A — navigational state | Set from `MY_CHECKLISTS` on click | OK (transient by design) |

## Persistence Rule

**Every user-entered value MUST flow through this path:**

```
User action
    → Update in-memory state (FIELD_RESPONSES, FAIL_NOTES)
    → debouncedSaveField(fieldId, value)          // workflows.html:389 — 400ms debounce
        → submitOp('SET_FIELD', fieldId, 'field_response', {value, field_id})
            → POST /api/v1/workflow/ops           // sync.js:781, Lamport-stamped
            → workflowOpRouter → workflow.SaveResponseFunc (persists to Postgres)
        → Update the draftResponses store (DRAFT_RESPONSES is its live alias)
    → On checklist open: hydrateFieldState(filterFieldIds)
        → Reads DRAFT_RESPONSES + MY_SUBMISSIONS
        → Populates FIELD_RESPONSES + FAIL_NOTES
```

**The rule:** If a user can enter it, `debouncedSaveField` must be called. If `debouncedSaveField` is called, `DRAFT_RESPONSES` must be updated. If `DRAFT_RESPONSES` is updated, `hydrateFieldState` must restore it.

🛑 **The function is `debouncedSaveField`, and the transport is `POST /ops`.** Every save-path
cell in the table above named `autoSaveField` → `POST /saveResponse` until 2026-08-04. Neither
half was true: no `autoSaveField` is defined anywhere in the tree, and no frontend code posts
to `/saveResponse` (the endpoint exists on the backend and the test suites drive it directly,
but the op journal is the single write channel — D-08). The one place production code actually
*called* `autoSaveField` — the fail-photo upload chain — threw a `ReferenceError` that its own
`.catch()` swallowed, silently dropping the photo. B-65, fixed by card A2 of run `20260804`.

## Items Requiring Future Work

- **#9 REJECTION_FLAGS**: Manager rejections are persisted server-side but never loaded back into the crew member's UI on reload. The crew member sees the rejection only in the current session. Needs: load `submission.rejections[]` into `REJECTION_FLAGS` in `hydrateFieldState`.
- **#10 WAS_REJECTED**: Dead code — should be derived from `MY_SUBMISSIONS[].status === 'rejected'`.

## Submission Validation Rules

Every validation must be enforced on BOTH client and server.

| # | Rule | Client enforcement | Server enforcement |
|---|------|-------------------|-------------------|
| 1 | Template with `requires_approval` must have at least one approver assignment | Toast: "Select at least one approver" | 400 `requires_approver` |
| 2 | Fields with triggered `fail_trigger` must have a corrective action (note + severity) | Toast: "Corrective action required for N field(s)" | 400 `corrective_action_required` |

When adding a new validation rule:
1. Add client-side check before the submit API call
2. Add server-side check in the handler before calling the repository function
3. Write TWO regression tests: one for client rejection, one for server rejection

## Total Count

- **7 user-entered states** flow through the save path (items 1-7)
- **1 deferred** (photo, Phase 12)
- **1 bug** (rejection flags not loaded from server)
- **1 dead code** (WAS_REJECTED)
