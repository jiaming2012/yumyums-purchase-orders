# HANDOFF — overnight-20260722 (for the morning of 2026-07-22)

> **Run branch:** `overnight-20260722` (cut from `dev` at `ffc474d`; **never pushed, `main` untouched**).
> **Slate:** `.night-crew/knowledge/reference/slate-20260722.md` (batch-signed 2026-07-20).
> **Scope:** Activity 3 remainder (Track S, 1 card) + Activity 4 Feature track (Track F, 5 cards),
> dispatched CONCURRENTLY, one in-flight card per track. Per-card worktree + fresh implementer
> subagent + **separate** fresh G6 adversarial subagent; orchestrator alone merged.
> **Result: 3 cards MERGED · 1 PARKED · 1 BLOCKED by the park · 1 DROPPED by budget discipline.**
> **4/4 G6 reviews returned findings that changed the outcome** — one park, two revision rounds.
> Reader = the operator; resolve via `/nc-morning-triage`.

## TL;DR

- **S1 `replay-fetchstorm-gate` — MERGED (PARTIAL).** The ungated `SUBMIT_CHECKLIST` replay
  re-fetch is gated (`sync.js`); fetch count on a 4-op catch-up went **5 → 1**. The
  `checkAllWithRepair` revert and the B4 approver-inclusion contract flip landed. De-flake tail (b)
  **PARKED per the card's own trigger** — and the orchestrator then **answered the parked question**
  with a quiet streak: **`:1198` is genuinely flaky, not load-caused** (see below). G6 found the
  card's premise is half-wrong — **the storm class is NOT fully closed.**
- **F1 `trends-spend-by-group-endpoint` — PARKED, unmerged.** Implementation was sound and
  faithful to signed design §2.2, but G6 broke its AC-6 reconciliation **five ways** with realistic
  fixtures; the red-first fixture was rigged on every axis simultaneously. The card's own PARK
  trigger — *"do not ship a number that disagrees with payroll's"* — fired. **The design is what's
  defective**, and amending it is operator-only.
- **F3 `trends-tab-frontend` — NOT DISPATCHED.** Blocked by F1's park per the slate's own dependency
  note (*"if F1 PARKs, F3 cannot run (no API) — skip to F2/F4"*).
- **F2 `cost-margin-endpoint` — MERGED.** `GET /api/v1/inventory/cost` (sibling endpoint;
  `menu-cogs` byte-unchanged). Fixture independently confirmed **unrigged** — survived 8 adversarial
  fixtures. Revision round fixed a negative-revenue row rendering **`food_cost_pct: -500000` ranked
  #1 "best"**, plus one vacuous assertion. Two forks routed to you.
- **F4 `cost-tab-frontend` — MERGED.** `#s6` built, 14 tests. **The screenshot ritual was genuinely
  performed and independently verified pixel-by-pixel** — and it caught two defects code review
  would have missed. Revision round pinned the card's best insight, which had **zero test coverage
  in either direction**, and defused a hazard that would have **broken the entire inventory page**
  once F5 lands.
- **F5 `inventory-tab-gating` — DROPPED** at 2h42m per the slate's budget clause (F5 is the named
  first drop). A half-landed permission spine is worse than none.

## Per-card wall-clock (harness-measured, Delivery KR3 — the standing output)

| Card | Impl | G6 | Revision + merge | Outcome |
|---|---|---|---|---|
| S1 `replay-fetchstorm-gate` | **47m23s** | **44m52s** | 0m24s | MERGED (PARTIAL) |
| F1 `trends-spend-by-group-endpoint` | **9m55s** | **11m14s** | — | **PARKED** |
| F2 `cost-margin-endpoint` | **12m23s** | **15m42s** | **5m31s** | MERGED |
| F4 `cost-tab-frontend` | **58m34s** | **47m55s** | **22m09s** | MERGED |
| F3 | — | — | — | blocked by F1 |
| F5 | — | — | — | dropped (budget) |

**Run total: 184m54s (~3h05m)** from `RUN_START` to F4's merge, plus the quiet determinism streak
(~13m) and closeout. Epoch-stamped legs in `timings.log`.

**Estimate vs actual:** the backend endpoint cards beat their 25–40m estimate (F1 9m55s, F2 12m23s
impl); the UI card ran over (F4 58m34s vs 35–55m). **G6 was the dominant unbudgeted cost** — the
slate priced G6 at 2–5m for F-cards; actuals were **11–48m**. That is not overrun, it is the gate
doing real work: every G6 this run ran its own mutations, built its own adversarial fixtures, and
three of four booted their own database. **Reprice G6 for code cards at ~15–45m, not 2–5m.**

## Gate evidence (on the final merged tree)

- **G1/G2** — `go build ./... && go vet ./...` from `backend/`: **both exit 0** on the final tree
  (also captured green at baseline before any card landed).
- **G3 (red-first)** — independently re-verified by G6 on **every merged card**, by checking out the
  red commit and re-running it, not by trusting the report:
  - S1: red reproduced verbatim (5 fetches vs `<= 2`); `git diff` red→green on the test file **EMPTY**.
  - F2: 8 tests assertion-level red; test-file diff red→green **0 lines**.
  - F4: 9 failed / 1 passed against the stub (the lone pass = a layout guard the stub trivially satisfies).
  - **No card weakened a test to make it pass.**
- **G4** — affected-subset suites green under `-p 1`: `recipes` + `inventory` both `ok` (F2);
  `states-cost + inventory + recipes` → **182 passed / 1 skipped / 0 failed** on a fresh DB (F4).
- **Quiet determinism streak** (orchestrator-run, post-Track-F, load 0.84 at start): leg 1
  **58/58 green**, leg 2 **57/58 — `:1198` RED**, leg 3 aborted by an orchestrator timeout.
- **`task sw`** — run **twice, orchestrator-only**, once per HTML/JS-touching merge (S1's `sync.js`,
  F4's `inventory.html`). 22 files precached, frontend 1.0.3.
- **Footprint** — `git diff --stat dev..overnight-20260722` = **14 files, +2264/−45**. Zero files
  outside the union of declared card footprints, except the one **adjudicated-justified** deviation
  (`tests/inventory.spec.js`, forced by stub deletion; G6 ruled parking would have meant landing a
  knowingly-red suite).
- **Ephemeral-env discipline** — host `:5432` **never touched**; every leg on Docker-assigned
  loopback ports (`nc-s1/f1/f2/f4`, `nc-g6f1/g6f2/g6s1/g6f4`, `nc-quiet`), all torn down
  `--volumes`. **0 `nc-` containers remain.**

## ✅ Triaged 2026-07-20 — recorded as `ledger.md` §T-19

Merged to `dev` `--no-ff` (`05dc053`). Independent re-verification on the merged tree: build, vet,
and **`go test ./...` all exit 0**; G4 discipline greps clean (structurally N/A for HQ);
replay/testdata untouched.

**Settled (decisions 29–33):** Trends filters to food spend only · unreviewed receipts become a
completeness note, not a chart bar (they *cannot* be bucketed — review is what links line items) ·
Trends reports **attributed** spend with unattributed money in the completeness note instead of
prorated across groups, redefining the payroll identity to one that holds on messy receipts ·
F2's non-positive-revenue guard ratified · the residual-money gap resolved by consistency.
**Decisions 29–31 amend signed design §2.2 and un-park F1, which unblocks F3.**

**Left open as investigations (deliberately not forced):**
- **Food cost as a drifting long-term average** with a direction of travel, rather than a
  fixed-12-week snapshot. Dissolves the 0%-food-cost bug instead of patching it. F2-a unresolved;
  no third reason string coined.
- **Margin with and without discounting.** **Blocked on data that does not exist** — verified at
  triage: `daily_menu_sales` stores only `units_sold` + `gross_amount`, no discount/comp field.
  Needs Toast sync to capture it first. F4's red-negative fork unresolved.

Both are the same shape — single numbers where a comparison is wanted. Routed to the next PM session
as a product thread, not cleanup.

**Deferred, not asked:** F5 priority and the attended two-device check (recommendations below stand).

## ⚠ Standing flags (post-triage)

- **`task sw` was run → the attended two-device convergence check is RE-ARMED** (production
  `sync.js` changed). This is yours in the morning.
- **Prod deploy NOT done** (attended, rides the cycle gate).
- **Frontend semver untouched** (1.0.3) — bump belongs to `/save-project` at deploy time.
- **The Cost tab and `/inventory/cost` ship LOGGED-IN-ONLY** (F5 dropped). Anticipated by the slate
  ("endpoints ship logged-in-only for one day"), but it is live exposure — see DECISIONS-NEEDED F5.
- **B5 remains unclosed** — approve/reject handlers still ungated. Pre-existing, not a regression.

## What the G6 gate actually bought this run (the headline)

Every one of the four G6 reviews changed the outcome. This is worth reading before deciding how much
G6 is worth next slate:

1. **F1 — caught a rigged fixture and stopped a wrong number from shipping.** The identity held on
   the authored fixture; G6 rebuilt the fixture honestly and broke it five ways. The minimal breaker
   is **a receipt with an unitemized delivery fee** — the normal case, not an edge case.
2. **S1 — refuted the implementer's self-criticism, and found the card's premise half-wrong.** It
   traced the wait's arming point and showed the claimed exposure mechanism does not exist; then
   enumerated every branch in `applyOp` and found **`loadPendingApprovals()` and `loadTemplates()`
   are still ungated**. The storm is fixed for one op type, not for the class.
3. **F2 — found a bug the implementer's own reasoning implied but missed**, and named the gap the
   implementer had not: `menu-cogs` publishes `unallocated_cogs`; **Cost dropped it**, so summing
   the tab under-reports true COGS with nothing indicating a residual.
4. **F4 — proved the ritual was real, then found its blind spot.** It read all 10 PNGs itself and
   confirmed every claim true of the actual pixels — then showed that **removing the loss-red
   entirely, or restoring it under sparsity, both shipped green.** The card's best insight was its
   least protected assertion.

**Pattern worth keeping:** an implementer grading its own homework passes; a fresh reviewer with
*only* the contract, the diff, and the evidence does not. Three of four reviewers ran their own
mutations rather than re-reading the tests.

## Commits on `overnight-20260722`

Per card: impl commit(s) → `--no-ff` merge carrying the G6 verdict in the body → roadmap flip.
`4df240a` (S1 merge) + `a51bf95` (sw) + `ad4a5f5` (flip) ·
`fb1995c` (F2 merge) + `6121f20` (flip) ·
F4 merge + sw + flip · `2d76ecc` (F1 park record) · + this closeout.
**Parked work preserved unmerged on `card/f1-trends-endpoint` @ `88cab9d`** (2 commits).

## For the morning reader (triage order)

1. **Merge `overnight-20260722` → `dev`** (`--no-ff`). Then **re-arm the two-device convergence
   check** — production `sync.js` changed.
2. **Amend design §2.2 so F1 can un-park** — three decisions: `mercury_category` filtering,
   `pending_purchases` inclusion, and *what "reconciles with period-summary" means when
   `Σlines ≠ total − tax`* (which on real receipts is the common case). G6's B1/B2/B3/B5 probes are
   ready-made fixture cases. **F3 unblocks the moment F1 does.**
3. **Answer F2's two forks** (DECISIONS-NEEDED F2-a / F2-b) — the "recipe exists, zero window spend"
   flattering 0%, and the missing residual field. **Both should be settled before the Cost tab is
   operator-facing**, because the tab is where you would compare to payroll.
4. **Ratify F2's `<= 0` guard extension** — a written change to a signed rule, flagged rather than
   absorbed silently.
5. **Answer F4's fork** — loss-red is applied window-level but not row-level.
6. **Dispatch F5 first next run.** It is now the only thing between the Cost tab and access control,
   and it carries B5. Integration points and a reserved spec block are documented in DECISIONS-NEEDED.
7. **Open a de-flake card for `sync.spec.js`** — scoped to **both** `:1198` (proven flaky on a quiet
   box) **and** `:525 FLD-LIVE-02` (pre-existing order-dependent, fails at baseline).
8. **Open a follow-up for the unclosed storm class** — `loadPendingApprovals()` / `loadTemplates()`.
9. **Fix `.gitignore`** (`node_modules/` → `node_modules`) — a symlink slips past the trailing slash
   into `git add -A`. One implementer already hit it.
10. **Fix the dangling standing-rules pointer** — every slate since 07-15 inherits gates from
    `reference/overnight-run-plan-20260707.md`, **which does not exist.**

## Run-mechanics notes for the next slate

- **Reprice G6 at 15–45m for code cards** (measured 11–48m against a 2–5m estimate).
- **`-p 1` is load-bearing** for Go suites — verified at base commit that parallel `-p` reddens four
  packages via concurrent `TRUNCATE`s. Put it in the standing mechanics, don't rediscover it per card.
- **The `:8199` latch recurred (third run).** Kill the **listener PID** via `ss -ltnp`, not the
  `go run` parent, and assign concurrent tracks distinct `TEST_PORT`s up front.
- **A subagent ran `git stash` in a worktree** (forbidden) — self-disclosed, fully recovered, main
  repo stash independently verified intact. Consider a mechanical guard; prose is not holding.
- **Orchestrator command timeouts must exceed the leg they wrap.** A 10m timeout killed quiet streak
  leg 3 mid-run. Reported as aborted, not as a result.
- **Concurrent dispatch worked**, but it cost S1 its determinism proof for the whole run — the quiet
  window only existed after Track F finished. If a card's deliverable *is* a flake proof, either
  serialize it or reserve the quiet window up front.

## Deviations from the slate (declared)

1. **Merge order.** F2 merged before S1, not in slate order — S1's G6 was still running and the two
   are file-disjoint (`sync.js`/tests vs `backend/internal/recipes`). The rule's purpose
   (serializing shared surfaces) held; holding F2 would have stalled Track F for nothing.
2. **Two revision rounds** (F2, F4) were dispatched for in-footprint G6 findings that were
   violations of already-signed text. The slate budgets a revision round only for F5. Judged as
   executing the spec rather than deciding anything; both are documented in the merge bodies.
3. **The orchestrator ran the quiet determinism streak itself** rather than leaving S1's parked tail
   unanswered — the rules of engagement forbid leaving that proof half-claimed.
