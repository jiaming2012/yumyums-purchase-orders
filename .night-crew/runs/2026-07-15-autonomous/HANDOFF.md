# HANDOFF — overnight-20260715 (for the morning of 2026-07-15)

> **Run branch:** `overnight-20260715` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260715.md` (batch-signed 2026-07-14).
> **Scope:** Activity 4 — the **test-only prove-UNPROVEN sweep across all 5 apps** + a first wave
> of graduated fixes. **Concurrent dispatch** (Tracks A–E in parallel, ≤1 in-flight card/track).
> **Result:** **16/16 prove cards landed, every card G6-PASS, 0 REVISE, 0 crashes, 1 graduated
> fix (Inventory NFR-1).** ~52 flows proven GREEN, **1 RED found → fixed tonight**, the rest
> legitimately PARKED. 17 atomic merge commits (16 prove + 1 fix). Wall-clock ≈ **2h45m** of the
> 6h floor (prove sweep ~2h15m concurrent + the graduated fix ~30m serial at the tail). Reader =
> the operator; resolve via `/nc-morning-triage`.

## TL;DR (what changed)

- **The prove-UNPROVEN sweep is DONE — 16/16 cards, all G6-verified.** Every still-UNPROVEN flow
  across Operations, Purchasing, Users, Onboarding, Inventory now has a **real red-first assertion**
  naming its observable DB/UI behavior. Vacuous/tautological tests were rewritten into genuine
  guards (Ops flag-reject `:485-508`, Purchasing shopping tail, Inventory FR-11 `if`-guards).
- **The sweep's headline finding: the UNPROVEN flows were overwhelmingly UNTESTED, not BROKEN.**
  The scoping pass forecast ~34–40 of ~78 flows would go RED; **actual: exactly 1 deterministic RED**
  (Inventory NFR-1 item-edit normalization). The app code was already correct — including all four
  slate-flagged PRIORITY-risk flows (Inventory **NFR-8** slider reach-past-100 rollback, Stock
  **FR-12/13/14**), which proved WORKING with strong E2E assertions.
- **The one RED was fixed the same night (Eng KR-1: +1 → 0).** Inventory **NFR-1**: `UpdateItemHandler`
  wrote item descriptions raw (no `normalizeItemName`, unlike create); `ConfirmPendingPurchaseHandler`
  upserted vendor names raw. 2-line fix in `internal/inventory/handler.go` flipped the E4-committed
  RED test GREEN (red→green captured on pristine-vs-fixed binaries).
- **11 flows PARKED honestly** (photo/S3 plumbing ×3, offline/IndexedDB ×2, cron clock-seam ×4,
  a confirm-vendor 2nd gap folded into the fix, thumbnail ×1) + **1 UNTESTABLE** (Onboarding FR-28
  boot-time re-seed). Each parked with a committed `test.skip`+reason or an inline note — visible in
  the suite, not silently dropped. See `DECISIONS-NEEDED.md` §A/C.
- **3 future fix-WOs surfaced** (cron clock-seam, photo-S3 harness, offline-IndexedDB harness) for
  the planners to schedule — none is a tonight-scope test-only fix. See `DECISIONS-NEEDED.md` §B.

## Per-card outcome table (17 cards, concurrent, all G6-PASS)

| # | Card | Verdict | Merge | Flows |
|---|---|---|---|---|
| A1 | `ops-prove-checklists` | G6 PASS | `3513ba8` | FR-6/7/8 **GREEN** (idempotent submit no-dup, unsubmit submitter-authz 403, history last-50 DESC) |
| A2 | `ops-prove-approvals` | G6 PASS | `69f26f3` | FR-10/12/13 **GREEN** (+ rewrote vacuous `:485-508` flag test → real 3-hop assert) |
| A3 | `ops-prove-builder` | G6 PASS | `f066951` | FR-16/17/18 **GREEN** (DOW visibility, section day-condition, skip-logic show/hide round-trip) |
| A4 | `ops-prove-cross` | G6 PASS | `1879349` | NFR-2(presign-shape)/NFR-6/NFR-7(redirect)/FR-19/FR-20 **GREEN**; **PARK** NFR-5, NFR-2-PUT, NFR-7-draft |
| B1 | `purchasing-prove-order` | G6 PASS | `351d04e` | FR-1/2/4/6 **GREEN** (roll-to-next-week, qty stepper/qty-0, suggestions+bulk-add, cutoff admin+403) |
| B2 | `purchasing-prove-po-approval` | G6 PASS | `6658342` | FR-13/14/15/16 **GREEN** (locked read-only, admin-edit vs 403, approve-btn, snapshot + dual-409) |
| B3 | `purchasing-prove-shopping` | G6 PASS | `24c121e` | FR-12/17/24 **GREEN** (+ rewrote vacuous tail; 1 UI-timing-flaky rewrite flagged, retries:1) |
| B4 | `purchasing-prove-state-auth-scheduler` | G6 PASS | `3875ad8` | NFR-1/NFR-2/FR-23 **GREEN** + 3 Go units GREEN; **PARK** FR-19/20/21/22 crons (clock-seam) |
| C1 | `users-prove-security` | G6 PASS | `2ceb823` | NFR-1..5 **GREEN** (403×8, invite single-use, grant→/me/apps round-trip, notif-pref, 401); NFR-2 7d-expiry UNTESTABLE-leg |
| C2 | `users-prove-ui-access` | G6 PASS | `f43554b` | FR-2/4/6/9/10/11/15/16/17/18/19 **GREEN** (9 flows) |
| D1 | `onboarding-prove-assignments` | G6 PASS | `639bd41` | FR-26 **GREEN** (unassign idempotent + role-auto-assign survives) |
| D2 | `onboarding-prove-progress` | G6 PASS | `11b2010` | FR-10/14/21/29/NFR-3 **GREEN**; **PARK** FR-18 (thumbnail); **UNTESTABLE** FR-28 (re-seed) |
| E1 | `inventory-prove-purchases` | G6 PASS | `5484743` | FR-3/5/11 **GREEN** (item-picker persist, discard, vendor filter+pagination; ~8 guards → real seed) |
| E2 | `inventory-prove-stock` | G6 PASS | `c24270c` | FR-12/13/14/15 **GREEN** (threshold classify, COALESCE override, count+reason, reorder/View-in-Setup) — PRIORITY, proved WORKING |
| E3 | `inventory-prove-setup` | G6 PASS | `c472103` | FR-26/28/29/30/31 **GREEN** (CRUD title-case, low<high, vendors, tags, cross-pkg repurchase-reset); **PARK** FR-27 (photo) |
| E4 | `inventory-prove-recipes-cross` | G6 PASS | `bd1ee02` | FR-16/23/NFR-3/**NFR-8**(PRIORITY rollback)/NFR-9 **GREEN**; **NFR-1 RED** (graduatable) |
| FIX | `inventory-nfr1-normalize-fix` | G6 PASS | `748463c` | NFR-1 item-edit + confirm-vendor **RED→GREEN** (2-line `handler.go` fix) |

**Fix card (`748463c`) — G6 PASS.** The run's only production-code change: `+normalizeItemName(input.Description)`
in `UpdateItemHandler` (handler.go:1130) and `+normalizeItemName(input.VendorName)` in
`ConfirmPendingPurchaseHandler` (handler.go:660), mirroring the create-path idiom. G6 confirmed the
fix is minimal, idempotent (no create-path double-normalization regression), the RED→GREEN is
genuine (pristine binary returned `"nfr1 edited raw…"`, fixed binary returns `"Nfr1 Edited Raw…"`),
and no test asserts a raw name from the confirm-vendor path. Final integration `go build ./...` +
`go vet ./...` on the merged tree (with the fix): **CLEAN**.

**Every red→green / classification claim was independently re-verified by a separate fresh G6
subagent** against the diff + evidence only (not the implementer's reasoning). G6 confirmed each
new assertion is real (would go RED if the flow broke — several mutation-checked), each PARK is a
legitimate plumbing/refactor trigger (not an effort dodge), and the one RED is a genuine handler gap.

## Gate results on the final merged tree

- **Integration `go build ./...` + `go vet ./...`: CLEAN** on the run branch (the only Go added all
  run: B4's `internal/purchasing/scheduler_prove_test.go` + the fix's 2-line `internal/inventory/handler.go`).
- **B4 cron unit tests pass:** `go test ./internal/purchasing/ -run Prove` → `ok` (3/3).
- **`task sw` NOT run — correctly.** No card touched any `*.html` or JS app file (every change was a
  `tests/*.spec.js` or backend Go). The slate's "regen sw per HTML-touching merge" rule never fired.
  The frontend precache/semver is unchanged; `/save-project` handles version bumps at deploy.
- **No-new-reds vs baseline:** every card diffed its affected-seam subset against a fresh-DB baseline
  on its own ephemeral pg16 stack; all sibling flips confirmed pre-existing (HQ's documented ~37–41
  flaky-red pool: offline-sync, tab-persistence, cross-test DB-pollution). The **one intended new
  deterministic red** (E4's NFR-1) was flipped GREEN by the fix.
- **Footprint:** every card touched only its declared files; the 5 app tracks shared zero source
  files (verified per merge). Zero scope breaches, zero improvised forks.

## Cycle status-tally movement (KR denominators)

- **QA KR-1 (untested/vacuous → asserted):** ~52 UNPROVEN flows now carry real red-first guards;
  ≥4 vacuous/tautological tests rewritten into genuine assertions. This is the guaranteed deliverable
  and it landed in full.
- **Eng KR-1 (known-broken → fixed):** the sweep surfaced **1** broken flow (Inv NFR-1) and **fixed
  it the same night** (+1 → 0). The forecast "sweep raises known-broken before the fix wave lowers it"
  held, but at 1/78 the broken rate was far below the ~34–40 forecast — the flows were untested, not
  broken.
- **Delivery KR (flow coverage):** 16/16 prove cards, ~52 flows moved UNPROVEN → WORKING [E2E-proven]
  (or → the honest PARK/UNTESTABLE/BROKEN-then-fixed classification).

## Operational notes for the next slate (sizing + process)

- **Concurrency is a throughput win, at a per-card latency cost.** Running 3 ephemeral Docker envs +
  3 implementer agents at once inflated *per-card* wall-clock ~1.5–2× vs. the prior serial run
  (test-only cards 8–33 min here vs. 6–13 min serial) — shared CPU/IO across concurrent Docker
  builds + Playwright. But *total* wall-clock was **~2h15m for 16 cards** vs. the slate's serial
  estimate of ~8h30m. Net: concurrency ≈ 3.5× throughput. See `reference/card-actuals.md`.
- **Deliberate throttle to 3 in-flight (not 5).** The slate authorized up to 5 concurrent envs;
  I capped at a **rolling 3** as prudent Docker-strain management (this box runs Docker 20.10.14; the
  only prior clean data point was serial one-env-at-a-time). Result: **0 Docker crashes, 0 port
  collisions** across ~17 env cycles. 3 was comfortable; 4–5 is likely safe next time with the warm-
  cache pattern below.
- **Pre-warming the Docker build cache once, up front, was load-bearing.** A single `docker compose
  build` on the run branch before dispatch meant every per-worktree build reused the go-mod/deps
  layers — builds were seconds, not minutes. Do this every concurrent run.
- **⚠ SHARED-GIT-STASH HAZARD (fixed mid-run, must stay fixed).** All worktrees share one `.git`,
  so `refs/stash` is GLOBAL. Early on, one card's `git stash pop` yanked a concurrent card's WIP into
  the wrong worktree (recovered cleanly, no work lost). **Fix applied:** the implementer runbook now
  hard-forbids `git stash` and prescribes baseline-first / `cp` for baseline capture — every
  subsequent card obeyed it, zero further incidents. **Keep this rule in the standing runbook.**
- **Worktrees carry no `node_modules`.** Agents symlinked the main repo's (gitignored) — works, but
  I pre-symlinked it on worktree creation from mid-run on to save the dance. Consider a worktree-init
  step (or committing a `.gitignore` node_modules entry in worktrees) next run.
- **gopls "cannot find package in GOROOT" diagnostics on worktree Go files are false alarms** — the
  worktree module isn't in the IDE workspace. The real `go build`/`go vet` in the worktree pass.
  Don't chase them.

## Suggested triage order

1. Spot-check the **1 graduated fix** (Inv NFR-1, `internal/inventory/handler.go` 2-liner) — the only
   production-code change and the sole Eng-KR-1 movement. The RED→GREEN evidence is in the fix
   commit; G6 re-verified it at the diff.
2. Skim the **16 prove-card merges** (all G6-PASS, red-first honest). The all-GREEN result is the
   story: the UNPROVEN backlog was untested, not broken.
3. Note **`DECISIONS-NEEDED.md`** §A (11 parked flows) + §B (3 future fix-WOs: cron clock-seam,
   photo-S3 harness, offline-IndexedDB harness) — hand §B to the planners to schedule. **No fork
   blocks the merge.**
4. Sign off the roadmap rows (flip DRAFTING → DONE per the slate policy — done at triage, not
   overnight), then merge `overnight-20260715` → `dev` `--no-ff`. `dev` is then deploy-ready
   (`/save-project` bumps semver at deploy; this run left it untouched).
5. Record triage resolutions in `ledger.md`; the next slate = the surfaced fix-WOs (§B) + any
   remaining Activity-4 tail.

## This session is disposable — safe to clear.

The run branch holds everything; all worktrees are removed, all Docker stacks torn down `--volumes`.
Nothing in this conversation's context is needed to continue — resume from the branch + these docs
via `/nc-morning-triage`.
