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

---

## § 2026-07-18 addendum — two more escaped defects, same class (approval-state ops)

Continued operator play on `dev` surfaced two further live-sync escapes of the **same root class**
(a live op arrives but the client re-renders from a **stale in-memory cache** instead of reconciling
the changed state). Recorded here so the WO above is scoped correctly.

### The defects (fixed, red-first)

1. **Rejection reason never reaches the submitter's other device live.** A manager rejects an item
   with a comment. The backend emits a `REJECT_ITEM` op and it *does* reach the submitter's device,
   but `applyOp`'s `REJECT_ITEM` branch called **only `loadPendingApprovals()`** — the approver's
   queue, a no-op for a non-approver. It never refreshed `MY_SUBMISSIONS`, so `hydrateFieldState`
   (which builds the `REJECTION_FLAGS` `⚠ Rejected` correction banner **only** from a submission whose
   status is `rejected`) never ran. The reason stayed invisible until a hard reload — and even an
   in-app reopen missed it, because the runner re-opens from the stale cached submission
   (`workflows.html` row-click reads `MY_SUBMISSIONS`, still `pending_approval`).

2. **Observer's list count frozen on the submission snapshot.** For a `pending_approval` /
   `submitted` / `approved` submission, `getProgress` counts the **frozen `submission.responses`
   snapshot**, not live state. After a rejection the submitter goes back to editing and unchecks a
   field; that `SET_FIELD` op reaches the observer and re-renders their list, but `getProgress` still
   reads the stale frozen snapshot (the observer's cached submission is still `pending_approval`,
   never refreshed to `rejected`), so the count never moves off the pre-rejection number.

**Fix (broad, operator-chosen):** `applyOp` now routes `APPROVE_ITEM` / `REJECT_ITEM` through
`loadMyChecklists` (in addition to `loadPendingApprovals`), gated like the `SAVE_TEMPLATE` branch to
avoid a catch-up fetch storm. `loadMyChecklists` re-fetches `MY_SUBMISSIONS`, re-hydrates
field/rejection state, re-renders the list, and re-renders an open runner in place — so on every
receiving device the correction banner, edit-vs-readonly mode, and list progress count all converge
live. Regression tests: `tests/sync.spec.js` `RJT-LIVE-01` (rejection reason live), `RJT-LIVE-02`
(observer count not frozen), `RJT-LIVE-03` (approval live). RED before, GREEN after; full `sync`
suite (43) green, no regressions.

### The QA gap this widens

The 2026-07-17 gap was "no cross-user access matrix." This addendum shows a second axis was missing:
**op type.** Every convergence test drove `SET_FIELD` (field-edit) ops. The **submission-lifecycle**
ops — submit / approve / reject — were **never tested cross-device**, and no test asserted that a
status change reconciles a *derived* view (the correction banner, readonly mode, or the list progress
count) on a second device. Both bugs lived in exactly that untested intersection: a lifecycle op ×
a derived view × an observer who isn't the actor.

### Recommendation update (feeds the same WO)

Extend the cross-user access matrix with an **op-type axis** and assert **live convergence of derived
views**, not just the raw field value:

| Op type | Actor | Observer | Derived view that must converge live |
|---|---|---|---|
| `SET_FIELD` | assignee | 2nd device / admin | field value + list count (was tested) |
| `REJECT_ITEM` | approver | submitter's 2nd device | correction banner + edit mode (was NOT tested) |
| `REJECT_ITEM` | approver | admin/manager observer | list progress count off the frozen snapshot (was NOT tested) |
| `APPROVE_ITEM` | approver | submitter's 2nd device | Approved badge / approved readonly state (was NOT tested) |
| `SUBMIT_CHECKLIST` | submitter | approver + 2nd device | Approvals queue + pending badge |

The single WO now reads: **"cross-user × op-type live-sync matrix (assert derived-view convergence)
+ `sync`-package unit coverage."**
