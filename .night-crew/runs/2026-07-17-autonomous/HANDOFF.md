# HANDOFF — overnight-20260717 (for the morning of 2026-07-17)

> **✅ TRIAGED 2026-07-17** (`/nc-morning-triage`) — merged `→ dev --no-ff` (`22cb7dd`) after cold
> re-verification (build/vet/tests green; G4 greps N/A in subject repo; replay/testdata +
> package.json untouched). Flags updated: the **⚠️ #2 `workbox-build` gap is CLEARED** (F-D fixed
> now — `chore(build)` `3b1be67`, `^7.4.1` declared + lockfile). **F-A** convergence-cell flake →
> scheduled as roadmap card `editprop-convergence-cell-hardening` (Activity 6) with the operator
> rider "no no-retry hard gate on this suite until it lands"; **F-B/F-C/F-E** → BACKLOG. Full
> record: `ledger.md` §"Morning-triage resolutions (2026-07-17)". `dev` pushed at triage close.

> **Run branch:** `overnight-20260717` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260717.md` (batch-signed 2026-07-16).
> **Design gate:** FR-1 edit-propagation design operator-signed 2026-07-16
> (`designs/editprop-frozen-at-submit.md`; C5 = warned-live-removal) — authorized the 3 editprop cards.
> **Scope:** Activities 2 + 4-signed-5 + 6 — the full editprop chain + 2 engine-trust fixes + 2
> carried fixes + test-debt retirement. **Serial** dispatch (one in-flight card at a time).
> **Result:** **9/9 cards DONE, every card G6-verified, 0 parked, 0 footprint breaches.** Two cards
> (W-2, W-3) took one G6-driven revision loop each and then passed. 9 atomic commits.
> Reader = the operator; resolve via `/nc-morning-triage`.

## ⚠️ Read first — two things that differ from the usual overnight

1. **Roadmap cards were flipped straight to `DONE` at merge**, per this run's explicit launch
   instruction ("roadmap card flip to DONE"), NOT to `DRAFTING`-for-triage as in prior overnights.
   The `roadmap.md` already shows all 9 rows `DONE ✅ overnight-20260717` with the G6 SHA + evidence.
   **You should still review this HANDOFF + the diffs** and may revert any card's status if you want
   to re-examine it — the flip is a recording convenience, not a substitute for your review.
2. **`workbox-build` is not a declared dependency** and was missing from this dev checkout's
   `node_modules`, so `task sw` failed until I `npm install --no-save workbox-build` (transient, NOT
   committed — `package.json`/lockfile unchanged). `task sw` then ran cleanly for every HTML/JS-
   touching merge. **Follow-up:** add `workbox-build` to `package.json` devDependencies so `task sw`
   (and `task prod:deploy`, which depends on it) works on a clean checkout. Prod already has it
   (07-14 ran `task sw`), so deploy is not at risk — this is a clean-checkout gap only.

## TL;DR (what changed)

- **The frozen-at-submit edit-propagation architecture landed in full (Activity 5, 3 cards):**
  - **Stable field identity** — `updateTemplate` diff/upserts by Builder-sent field IDs; the
    `replaceTemplate` delete-and-reinsert path is **deleted from the codebase**; a surviving field
    keeps one permanent `checklist_fields.id` for life (the Friday P0 field-ID-churn root cause is
    now structurally impossible). A write to a field absent from the current template → **422
    `{"error":"unknown_field"}`** (app-level check, no restored FK); the runner rolls back the
    optimistic checkmark.
  - **Broadcast re-render** — `applyOp` handles `SAVE_TEMPLATE`: re-fetch + re-render the open
    unsubmitted checklist with surviving answers intact (all 7 types + sub-steps), **silent on
    catch-up** replay; **transactional op emission** (op row commits in the write txn); **C5 warned
    live removal** for schedule-drops-today; **INV-6 Builder discard warning naming the crew count**
    before a save that cuts a field / drops today (cut-field unsubmitted drafts discarded in-txn).
  - **Convergence matrix** — a two-device E2E matrix proving convergence on the *observing* device
    for all 7 types + sub-steps + photo + submit/unsubmit + list-view progress + denominator, live +
    catch-up; **AC-6a** (mid-run edit re-renders open device, answers intact) is a real red→green
    bug-guard; **AC-6b** (submitted record byte-identical after later edits) is a frozen-snapshot
    coverage-lock. Surfaced + fixed an unsubmit-broadcast gap in-footprint.
- **Two engine-trust fixes (Activity 2):** approval-with-feedback now returns **500
  `feedback_persist_failed`** instead of a false "Approved" when the comment doesn't persist
  (FR-8); a field write that loses LWW (409) now **re-renders the DB-winning value** so the screen
  never keeps a value the DB rejected (FR-9 — fixed a double-wrap in `api()`'s 409 return).
- **Carried backend gate:** direct-API resubmit of a `require_photo`-rejected field is now
  **blocked server-side** (400 `resubmit_photo_required`) at both submit paths — closes the
  07-14-deferred hole, resolved from the DB with no client-controllable escape.
- **Hygiene + test-debt:** removed the dead `users.html #s3` orphan div; added a `now func()
  time.Time` **clock seam** + 13 mock-time cron-decision tests to the 4 purchasing `run*Check`
  funcs (unblocks Purchasing FR-19–22); **retired vacuous waiver #2** — 16 vacuous test guards
  converted to real seeded assertions (2 more already hardened at base).

## Per-card outcome table (9 cards, serial, all G6-verified)

| # | Card | Verdict | Commit | Red→green evidence (ephemeral pg16) |
|---|---|---|---|---|
| W-1 | `editprop-stable-field-identity` | **G6 PASS** | `86bd09c` | Go 422 `unknown_field` + cross-device E2E identity go red on pristine (churn), green on fix. `replaceTemplate` deleted; app-level existence check, no FK. G6 re-reproduced both reds. main.go +6 (SET_FIELD 422 routing) = necessary wiring (runner saves via /ops). |
| W-2 | `editprop-broadcast-rerender` | **G6 PASS** (after 1 revision) | `186e14c` | All 5 sub-behaviors red→green: SAVE_TEMPLATE re-render (surviving answers), silent-on-catch-up, C5 warned live removal, transactional emission, INV-6 discard warning. **Revision:** INV-6 warning was initially parked as "out of footprint"; orchestrator sent it back — it's in-scope (FR-4/5+INV-3/6) and in-footprint — and it landed + fixed an orphaned-draft defect. |
| W-3 | `editprop-convergence-matrix` | **G6 PASS** (after 1 revision) | `3e5b921` | Full two-device matrix (7 types+sub-steps+photo+submit/unsubmit+list-progress+denominator, live+catch-up); AC-6a bug-guard + AC-6b snapshot-lock red→green; unsubmit-broadcast gap fixed in-footprint. **Revision (G6 FAIL-REVISE→PASS):** landed a parked denominator cell (in-footprint sync.js fix) + de-flaked two two-device cells. Orchestrator re-verified 36/36 twice under combined load. |
| W-4 | `engine-approval-feedback-loud` | **G6 PASS** | `f50dd32` | Failed `submission_rejections` persist → 500 `feedback_persist_failed` (was false "Approved"). Forced via invalid-UUID field_id; red 200→green 500. `ON CONFLICT DO NOTHING` removed (proven behavior-neutral). G6 re-reproduced. |
| W-5 | `ops-nfr3-resubmit-photo-gate` | **G6 PASS** | `733fa16` | Direct-API resubmit of a require_photo-rejected field: 201 bypass → 400 `resubmit_photo_required`. Server-side from `submission_rejections` on the submitter's most-recent prior submission; both submit paths. Lineage verified sound; no client escape. main.go +3 = necessary wiring (2nd direct-API path). |
| W-6 | `engine-conflict-refetch` | **G6 PASS** | `fc0ed6b` | LWW loser re-renders the DB-winning value from the 409 body via `applyOp` (fixed a double-wrap in `api()`). Deterministic red (loser shows `undefined`) → green (`WINNER`), G6-reproduced 3/3. No backend/schema change. |
| U-1 | `users-s3-orphan-cleanup` | **DONE** (slate-scoped inline verify) | `a11a58f` | Single dead `#s3` div removed; no `#s3` refs remain; Users E2E 33/0. No red-first per the slate's card spec (zero behavior change); orchestrator inline-verified the diff + grep + `task sw`. |
| T-1 | `carried-fix-wos-sweep` (cron-clock-seam) | **G6 PASS** | `c5aede8` | `now func() time.Time` seam on all 4 `run*Check` funcs (production still `time.Now`) + 13 mock-time cron-decision subtests. Behavioral red (body ignoring injected clock locks a PO it must not) → green, G6-reproduced. Unblocks Purchasing FR-19–22. |
| T-2 | `vacuous-tests-18-to-0` | **G6 PASS** | `3fd4d3f` | 18 = 16 converted (Onboarding 6 + Inventory 10, each a real seeded assertion downstream of exercised state — G6-verified non-tautological) + 2 Ops items already hardened at base. Tests-only; 204 passed / 2 S3-parks / 0 reds re-verified. Retires waiver #2. |

**Every red→green claim was independently re-verified by a separate fresh G6 subagent** against the
diff + evidence only (not the implementer's reasoning). G6 re-confirmed each new test genuinely went
red pre-fix and each fix stayed in (or justifiably extended) its declared footprint.

## Gate results on the final merged tree

- **Integration build/vet (all 9 merged):** `go build ./...` + `go vet ./...` **clean** — the serial
  `handler.go`/`repository.go`/`main.go`/`sync/ops.go` edits across W-1…W-6 integrate coherently.
- **Merged-tree Go tests:** `go test -p 1 ./internal/workflow/... ./internal/purchasing/...` → both
  `ok` — all cards' backend test files (stable-identity, broadcast-emit, approval-feedback,
  resubmit-photo-gate, cron-seam) coexist and pass together.
- **`task sw`** regenerated once per HTML/JS-touching merge (W-1, W-2, W-3, W-6, U-1) in landing
  order; a final idempotent regen showed no drift → `sw.js` reflects the final tree. Frontend semver
  unchanged (1.0.3 — bump belongs to `/save-project` at deploy, not overnight).
- **No new reds vs baseline:** each card judged on its NEW test going red→green + introducing no new
  reds vs HQ's ~37–41-red known pool (not a globally clean suite). Affected specs re-verified per card.
- **Footprint:** every card stayed inside its declared footprint. Three cards extended into
  `cmd/server/main.go` by ≤6 lines of pure error/gate routing (W-1, W-5) — necessary wiring the
  planner's footprint list missed (the runner/2nd-path saves via `/ops`), adjudicated PASS by G6 on
  the W-1 precedent. No schema migration anywhere (NFR-2 held).

## Roadmap state

All 9 rows flipped **PLANNED → DONE** in `roadmap.md` with the G6 SHA + one-line evidence (see the
⚠️ note above — this run flips to DONE at merge, not DRAFTING). Activity 5 (editprop build), the
Activity-2 engine-trust + carried fixes, and Activity 6 (test-debt) are complete.

## Decisions needed (operator)

**No open forks block triage. 0 cards parked.** See `DECISIONS-NEEDED.md` for 5 surfaced follow-ups
(all out-of-footprint scoped deferrals or test-hardening notes — none blocks merge):
the **W-3 convergence-cell flake under no-retry** is the one worth a real look (it's the Delivery-KR
suite); the rest are clean future cards (Create/Archive transactional emission, approval+feedback
atomicity, the `workbox-build` devDependency gap).

## Suggested triage order

1. **Skim the editprop chain (W-1→W-2→W-3)** — the cycle's headline. The commits carry the red/green
   detail; G6 independently reproduced each. Note W-2's INV-6 and W-3's revision loops (the run
   pushed both back to complete in-scope work rather than accept a premature park).
2. **Note the 5 follow-ups** in `DECISIONS-NEEDED.md` — accept as `BACKLOG` cards or schedule. The
   W-3 two-device convergence flake (green under `retries:1`, ~3/6 under no-retry) is the most
   substantive; decide whether it wants a dedicated hardening card before this suite is leaned on as
   a hard gate.
3. **Sign off the 9 DONE rows** (or hold any you want to re-read — revert its status if so).
4. **Merge `overnight-20260717` → `dev` `--no-ff`.** Then `dev` is ready for a normal
   `task prod:deploy` (this run left frontend semver at 1.0.3 — `/save-project` bumps at deploy).
5. **Add `workbox-build` to `package.json` devDependencies** (see ⚠️ #2) so `task sw` works on a
   clean checkout.
6. Record triage resolutions in `ledger.md`.

## Notes for the next slate (sizing signal)

- All 9 fork-free cards landed; the slate was coverage-complete (nothing spilled). App-fix +
  red-first cards ran ~15–30m impl; the two XL editprop cards (broadcast, matrix) ran longest and
  each needed one revision loop — the revision cost is real, budget for it on first-of-kind
  structural cards.
- The **two-device WS-convergence E2E cells are timing-sensitive** — green under `retries:1` but not
  robustly deterministic. Any future work leaning on that suite as a hard gate should first harden
  those cells (a candidate card is noted in `DECISIONS-NEEDED.md`).
- The ephemeral pg16 Docker env (`docker-compose.nc.yml` postgres service, one container per
  worktree on a Docker-assigned port) held with **zero crashes** across ~15 subagent runs; the
  Playwright-spawned `go run` webServer (working-tree code) is the clean red/green iteration path.
