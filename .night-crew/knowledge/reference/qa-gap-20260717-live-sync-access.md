# QA-gap report card — live-sync recipient access (found 2026-07-17, on dev)

> **Class:** escaped defect + QA coverage gap. Found by the operator while exercising `dev`
> before the dev→main promotion, on the "Nothing silently lost" cycle's own convergence work.
> Recorded per operator request (2026-07-17) so future QA closes the gap that let it through.

## The defect (fixed)

**Symptom (operator):** editing a checkbox on device A did not propagate live to device B (both
devices logged in as the operator, a **superadmin**); a page reload showed the change. Nothing
synced live — check *or* uncheck.

**Root cause:** live-sync ops fan out only to a checklist's **assignees**. `ResolveEntityAccess`
(`backend/internal/sync/ops.go`) resolved a `field_response` op's recipients to the template's
assignee users; the op **author was added only as a fallback when that set was empty**
(`listener.go`). The operator's "Friday checklist" was assigned to role `team_member` / `manager`;
the operator (`{admin}`, superadmin) is **not** in that set, so their own edits fanned out to the
team_members and **never reached their own second device**. A reload worked because it re-hydrates
from the DB (`myChecklists`/drafts), which is independent of the WS fan-out.

**Asymmetry that hid it:** `myChecklists` grants admins/superadmins **view-all**
(`roles && ARRAY['admin','superadmin']`), but the live-sync fan-out did **not** mirror that access.
So an admin could *view and edit* any checklist but was excluded from its live ops.

**Fix (`overnight`/interactive, red-first):**
- `ResolveEntityAccess` now unions in **all admins/superadmins** (mirrors `myChecklists` view-all).
- The listener now **always** includes the op **author** (a user's own edits always converge on
  their other devices, regardless of assignment).
- Regression test: `backend/internal/sync/access_test.go`
  (`TestResolveEntityAccess_AdminReceivesLiveOps`) — RED before, GREEN after.

## The QA gap (the point of this card)

1. **No cross-user access-matrix testing.** Every convergence/live-sync test in `sync.spec.js`
   (incl. this cycle's W-3 convergence matrix) drives **the assignee editing their own assigned
   checklist** — the editor is always in the recipient set, so the fan-out's recipient resolution
   was **never exercised for a non-assignee editor**. The bug lived precisely in the untested cell:
   **editor ∉ assignees** (admin/superadmin viewing/editing others' checklists).
2. **The entire `sync` package had ZERO Go tests.** `ResolveEntityAccess`, the hub, and the
   listener — the whole op fan-out/recipient-resolution layer — shipped with no unit coverage.
   (This fix adds the first `sync` package test.)
3. **The convergence "matrix" was a field-TYPE matrix, not an ACCESS matrix.** It proved all 7
   field types converge for an assignee; it did not vary *who* edits vs *who* observes vs *their
   role/assignment*.

## Recommendation for future QA (feeds next `/nc-okr-session`)

Add a **cross-user access matrix** to the QA bar for any collaborative/real-time feature — the
Cartesian product of **{who can view} × {who edits} × {who observes} × {their role/assignment}**,
asserting both **access** (can they view/edit) and **propagation** (does a live op reach them). At
minimum for the checklist engine:

| Editor role | Observer role | Checklist assigned to | Expect live op reaches observer? |
|---|---|---|---|
| assignee | same-assignee (2nd device) | that assignee's role | yes (was tested) |
| **admin/superadmin** | **same admin (2nd device)** | **a different role (team_member)** | **yes (this bug — was NOT tested)** |
| admin | team_member assignee | team_member | yes |
| team_member | admin observer | team_member | yes (admin view-all) |
| non-assignee, non-admin | — | — | no access (negative test) |

Graduate as a test-hardening WO (backlog): **"cross-user live-sync access matrix + `sync`-package
unit coverage."** Pair the E2E matrix with Go unit tests on `ResolveEntityAccess` for every
role/assignment combination.
