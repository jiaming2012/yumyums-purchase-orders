# DECISIONS NEEDED — overnight-20260717

> **No open forks block triage. 0 cards parked. All 9 slate cards landed G6-verified.**
> Everything below is a **surfaced follow-up** — an out-of-footprint scoped deferral or a
> test-hardening note. None blocks merging `overnight-20260717` → `dev`. Each is a clean candidate
> for a future card; the operator decides accept-to-`BACKLOG` vs schedule now.

---

## F-A · W-3 two-device convergence cells flake under no-retry (the one worth a real look)

**What:** the convergence-matrix suite (the Delivery-KR red→green pair) is **green under the repo's
shipped `retries:1`** config — orchestrator re-verified `repro-cut-task.spec.js tests/sync.spec.js`
= **36/36, 0 flaky, twice** (incl. on an accumulated DB). But the independent G6-W6 reviewer,
running `sync.spec.js + broadcast-rerender.spec.js` under **no-retry**, saw the two-device
`text answer converges` / `temperature answer converges` cells fail ~**3 of 6** single attempts
(reproduced on base `733fa16`, so pre-existing to W-6 — a harness WS-timing sensitivity, NOT a
product defect). W-3 was already hardened once this run (server-side persistence gate + deterministic
fail-note bundle write); the residual is asymptotic two-device WS-timing determinism.

**Why it matters:** this suite is the FR-7/A-5 convergence proof and the Delivery KR. Under
`retries:1` it passes; but it is not robustly deterministic without retries, so it can't yet be
leaned on as a *hard, no-retry* gate.

**Decision:** accept as-is (green under the shipped config) and log a **convergence-cell hardening
card** for a future slate, OR schedule that hardening before this suite gates anything. Recommended:
log to BACKLOG — chasing two-device WS timing to zero-flake is real work, disproportionate to
reopen mid-cycle. **Also:** only text/textarea convergence is tested on the W-6 *conflict* branch;
the other 6 field types ride the same `applyOp` path untested there — a small coverage-extension
worth folding into the same card.

## F-B · Transactional op emission not yet at full INV-1 parity (W-2 follow-up)

**What:** W-2 moved the **`updateTemplate`** op emission into the write txn (`EmitOpTx`), but
`CreateTemplateHandler` and `ArchiveTemplateHandler` still use the fire-and-forget `EmitOp`
goroutine. Full INV-1 "0 accepted writes whose op is not durably queued" parity wants those two
converted too.

**Decision:** schedule a small follow-up card converting Create/Archive to the transactional emit
(mirrors W-2's pattern; no schema change). Backlog.

## F-C · Approval + feedback is not atomic (W-4 follow-up)

**What:** `approveSubmission` commits `status='approved'` **before** the feedback loop, so a
feedback-persist failure now correctly returns **500 `feedback_persist_failed`** (the card's goal)
but with the submission **already `approved`** — a partial commit. G6 ruled the card's requirement
(no false "Approved" for unstored feedback) genuinely MET and atomicity an out-of-footprint follow-up.
Minor: a retrying approver sees a less-specific `internal_error` on the 2nd attempt (idempotent
against double-approval via the `status='pending'` guard).

**Decision:** schedule a follow-up threading a `tx` through `approveSubmission` (repository.go) so
status + feedback commit atomically. Backlog.

## F-D · `workbox-build` is an undeclared build dependency (infra gap)

**What:** `build-sw.js` (run by `task sw`, and transitively by `task prod:deploy`) `require`s
`workbox-build`, but it is **not in `package.json` devDependencies** and was absent from this dev
checkout's `node_modules`. I installed it `--no-save` (transient, not committed) to run `task sw`
per merge. Prod already has it (07-14 deploys ran `task sw`), so deploy is not at risk — this is a
**clean-checkout** gap.

**Decision:** add `workbox-build` (+ pin) to `package.json` devDependencies and commit the lockfile
so `task sw` / `task test` / `task prod:deploy` work on a fresh clone. Low effort, do at triage.

## F-E · Two onboarding persistence tests use a fixed flush wait (T-2 minor)

**What:** in `tests/onboarding.spec.js`, two converted persistence tests use `waitForTimeout(1500)`
instead of `waitForResponse('/saveProgress')`. The load-bearing proof is still the post-reload
assertion, so the guard isn't weakened — but a fixed wait is a small flake-surface.

**Decision:** optional tidy-up — switch to `waitForResponse` on the save POST in a future
test-hardening pass. Low priority.

---

### Minor observations (no action needed, recorded for completeness)
- **W-3:** the discard-warning / schedule-drop check uses browser-local `getDay()` — a theoretical
  timezone-boundary-day edge (consistent with the existing `myChecklists?dow=` convention).
- **W-2:** `DraftHolderCountHandler` does no server-side UUID validation; a malformed `field_ids`
  param would 500 rather than 400 (admin-only, frontend only ever sends UUIDs — low risk).
- **W-6:** `sync.js:537` (offline `submitChecklist`-archived whole-submission conflict) was
  deliberately left untouched — different semantics from field-value LWW, outside FR-9's scope.
