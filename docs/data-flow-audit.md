# Workflows Data Flow Audit

Last updated: 2026-04-16

## User-Entered State Variables

| # | Variable | Type | User enters via | Persisted? | Save path | Restore path | Status |
|---|----------|------|-----------------|------------|-----------|--------------|--------|
| 1 | `FIELD_RESPONSES[fieldId].value` | checkbox boolean | Tap checkbox | Yes | `autoSaveField` → `POST /saveResponse` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 2 | `FIELD_RESPONSES[fieldId].value` | yes/no boolean | Tap Yes/No pill | Yes | `autoSaveField` → `POST /saveResponse` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 3 | `FIELD_RESPONSES[fieldId].value` | text string | Type in textarea | Yes | `autoSaveField` (on blur) → `POST /saveResponse` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 4 | `FIELD_RESPONSES[fieldId].value` | temperature number | Type/spin number | Yes | `autoSaveField` (on change) → `POST /saveResponse` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 5 | `FIELD_RESPONSES[fieldId].value` | sub-step state | Tap sub-step checks | Yes | `autoSaveField` → `POST /saveResponse` → `DRAFT_RESPONSES` | `hydrateFieldState` ← `DRAFT_RESPONSES` / `MY_SUBMISSIONS` | OK |
| 6 | `FAIL_NOTES[fieldId].note` | string | Type in fail card textarea | Yes | Bundled via `autoSaveField` as `{_v, _fail_note}` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_fail_note` → `FAIL_NOTES` | OK |
| 7 | `FAIL_NOTES[fieldId].severity` | string | Tap Minor/Major/Critical pill | Yes | Bundled via `autoSaveField` as `{_v, _fail_note}` → `DRAFT_RESPONSES` | `hydrateFieldState` unpacks `_fail_note` → `FAIL_NOTES` | OK |
| 8 | `FAIL_NOTES[fieldId].photo` | blob URL | Camera capture (Phase 12) | No | Not implemented — Phase 12 deferred | N/A | DEFERRED |
| 9 | `REJECTION_FLAGS[fldId]` | object | Manager flags field for rejection | No | Server-side via `POST /rejectItem` | Never loaded from server into `REJECTION_FLAGS` | BUG (separate issue) |
| 10 | `WAS_REJECTED[tplId]` | boolean | N/A (should be derived) | No | Never written | Never read meaningfully | DEAD CODE |
| 11 | `state.activeTemplate` | object | Builder editor edits | Yes | `POST /createTemplate` or `PUT /updateTemplate` on save | `GET /templates` → `fromApiTemplate` | OK |
| 12 | `fillState.activeTemplate` | object | Selecting a checklist to fill | No (transient) | N/A — navigational state | Set from `MY_CHECKLISTS` on click | OK (transient by design) |

## Persistence Rule

**Every user-entered value MUST flow through this path:**

```
User action
    → Update in-memory state (FIELD_RESPONSES, FAIL_NOTES)
    → autoSaveField(fieldId, value)
        → POST /saveResponse (server persists to Postgres)
        → Update DRAFT_RESPONSES (in-memory cache for reopen)
    → On checklist open: hydrateFieldState(filterFieldIds)
        → Reads DRAFT_RESPONSES + MY_SUBMISSIONS
        → Populates FIELD_RESPONSES + FAIL_NOTES
```

**The rule:** If a user can enter it, `autoSaveField` must be called. If `autoSaveField` is called, `DRAFT_RESPONSES` must be updated. If `DRAFT_RESPONSES` is updated, `hydrateFieldState` must restore it.

## Items Requiring Future Work

- **#8 Photo capture**: Deferred to Phase 12. Will need blob-to-server upload + URL persistence.
- **#9 REJECTION_FLAGS**: Manager rejections are persisted server-side but never loaded back into the crew member's UI on reload. The crew member sees the rejection only in the current session. Needs: load `submission.rejections[]` into `REJECTION_FLAGS` in `hydrateFieldState`.
- **#10 WAS_REJECTED**: Dead code — should be derived from `MY_SUBMISSIONS[].status === 'rejected'`.

## Total Count

- **7 user-entered states** flow through the save path (items 1-7)
- **1 deferred** (photo, Phase 12)
- **1 bug** (rejection flags not loaded from server)
- **1 dead code** (WAS_REJECTED)
