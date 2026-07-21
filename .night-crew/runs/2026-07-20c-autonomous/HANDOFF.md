# HANDOFF — overnight-20260720c

> ## ✅ TRIAGED 2026-07-21 — merged to `dev` (`c2cfc13`), recorded as ledger T-20
>
> **Standing flags after triage:**
>
> - **Attended two-device convergence check — STILL ARMED, NOT RUN.** Armed by the 07-22
>   `sync.js` change. D1 left `sync.js`/`sw.js` byte-identical, so this cycle neither
>   re-armed nor cleared it. Re-arms again on any change to production sync or the
>   verify/merge path.
> - **Prod deploy — NOT done.** Frontend semver untouched (1.0.3); the bump belongs to
>   `/save-project` at deploy time.
> - **DB isolation — materially improved, NOT closed.** Go and Playwright now own separate
>   databases (`hq_test_go` / `hq_test_e2e`), proven concurrent-safe by execution
>   (decision 38). Surfaces #2, #5, #9 fixed (decisions 39–41). **Surface #4 remains
>   OPEN and is now the sharpest item**: dev, prod and test share one Postgres cluster
>   under one role and one password, separated only by `search_path`.
> - **`grant-enforcement-parity` — NEW URGENT CARD, blocks go-live** (decision 36). The
>   Users tab offers 11 grants; the backend enforces 2. No live exposure today.
> - **`stash@{0}`** still holds unattributed WIP in a slot shared by five worktrees.
> - **FR-12 Cliq-dup watch** continues over the cycle (nothing observed).
>
> **Corrections to this document, made at triage:**
>
> 1. §D1 and §"Read this first" state that `:1198` is "very likely **not flaky**" and that
>    the card's premise needs re-deriving. **RETRACTED — the test IS flaky** (16% overall,
>    20% under a concurrent suite). The card is KEPT and RE-AIMED, not dropped. See
>    `reference/1198-flake-reproduction-20260721.md` and preference P3a.
> 2. §"Verification state" reports 515 passed. Re-verified on the merged tree at triage:
>    **528 passed / 6 skipped / 0 failed / 0 flaky** (the sweep and triage added tests).
> 3. The run's own §D1 note that a concurrent session held the box is confirmed and
>    generalised into preference P2 — "the box is quiet" is measured, never assumed.


**Run branch:** `overnight-20260720c`, cut from `dev` @ `688f74b`. **Never pushed. `main` untouched.**
**Slate:** `.night-crew/knowledge/reference/slate-20260720c.md` (batch-signed 2026-07-20).
**Wall clock:** 407 min (~6h47m). Slate envelope was 6h10m–9h40m — **in band.**
**Outcome: 3 of 4 cards merged, 1 PARKED correctly.**

---

## Read this first

Two results matter more than the rest:

1. **F5's G6 caught a live authentication bypass** that the card, as written, would have shipped.
   A non-approver with zero assignments forged an approval that broadcast over the sync hub as
   legitimate. Fixed, re-verified by live exploit, merged. Details below.
2. **D1 PARKED, and in doing so refuted the evidence that promoted it.** `sync.spec.js:1198` is
   very likely **not flaky** — ~20 consecutive greens across two independent parties against a
   claimed 50% failure rate. **The `cycle-gate` card's premise needs re-deriving before it is
   scheduled.** See DECISIONS-NEEDED §0a–0d.

**Open forks: 10.** All in `DECISIONS-NEEDED.md`, ordered by consequence. Nothing there blocked a
merge; §0a–0d block the `cycle-gate` card.

---

## Per-card outcomes

| Card | Outcome | Impl | G6 | Merge | Commit |
|---|---|---|---|---|---|
| Wave 0 | DONE | 1m | n/a | direct | `ecb12e0` |
| F1 · trends-spend-by-group-endpoint | **MERGED** | 24m (+18m revision) | REVISE → PASS | 1m | `a7d09b6` |
| F3 · trends-tab-frontend | **MERGED** | 55m | PASS first pass | 1m | `be5ffb0` |
| F5 · inventory-tab-gating | **MERGED** | 60m (+50m revision) | **FAIL** → PASS | 1m | `c1a2393` |
| D1 · syncspec-deflake | **PARKED** | 180m | diagnosis partly confirmed; **PARK correct** | — | net-zero diff |

Estimates ran long on every card that landed: F1 52m vs 50–95m, F3 55m vs 100–150m, F5 110m vs
110–180m. F3 and F5 both credited prior cards for leaving reserved test blocks and delegation-safe
containers exactly where the slate promised — the prep was real and it compounded. **D1 was the
inverse:** 180m against 80–125m, because the honest path (full-suite verification, then revert) is
slower than the dishonest one.

**G6 pricing was correct.** The slate repriced G6 at 15–45m per code card rather than 2–5m; actuals
were 18m / 18m / 8m plus two revision rounds. **Both revision rounds were load-bearing** — one
caught a payroll-disagreeing rounding bug, one caught an auth bypass. Budgeting a revision for
every card, not just first-of-kind ones, was the right call and should stand.

---

## Wave 0

`.gitignore:1` `node_modules/` → `node_modules`. The trailing slash matched a directory but not a
symlink; worktrees have no `node_modules`, so symlinking the main install slipped past `git add -A`.
Committed `ecb12e0`. **All four worktrees were removed as their cards closed except D1's, which is
preserved per PARK rules.**

---

## F1 · trends-spend-by-group-endpoint — MERGED `a7d09b6`

`GET /api/v1/inventory/trends` per design §2.2 as amended (decisions 29/30/31). Re-derived against
the amended spec, not resumed from the parked branch.

**The identity holds exactly** — `Σcells + Σunlinked + pending_total == period_summary.cogs_excl_tax`
— asserted by calling `period-summary` in the same test on the window the handler itself reported,
never against a constant. All five G6 breakers present and each carrying non-zero weight.

**G6 REVISE round, and it mattered.** The reviewer confirmed the fixture was honest, then went
further and *constructed a new breaking case*: `purchase_line_items.price` is `NUMERIC(10,4)`;
Trends rounded per week×group cell and summed the rounded cells, while `period-summary` rounds once
over all lines. `Σ(round) ≠ round(Σ)` — two lines of `1 × 4.9950` in different weeks produced a **1¢
disagreement with payroll's number.** Reachable in production today: `receipt/worker.go:778` rounds
`quantity` at the write boundary but passes the LLM's `price` through unrounded, and the parser
prompt explicitly asks for *unit* prices on weight-priced lines.

Fixed by computing the published figure from one window-wide sum. The revision also **separated two
claims that had been conflated**: the payroll-facing number is now asserted exact with no tolerance,
while per-cell display rounding gets an explicit derived drift budget (`0.005 × N`, triangle
inequality — the reviewer verified it was derived a priori, not reverse-fitted). An **anti-vacuity
guard** fails the test if the sub-cent fixture ever stops exercising the bug — a direct answer to
why this card parked last cycle.

Gate evidence: red-first genuine (verified by checkout); four independent mutations of `trends.go`
each caught; `go build`/`vet` green; `CI=1 go test -p 1 ./...` green, 8 packages.

---

## F3 · trends-tab-frontend — MERGED `be5ffb0`

Trends tab (`#s5`): inline SVG weekly spend-by-group chart + table, 12-week window. **G6 PASS on the
first pass** — the only card that needed no revision.

**It reconciles on the right number.** `completeness.reconciles_to_cogs_excl_tax` (the payroll-facing
figure), not the sum of penny-rounded display cells, and it discloses the drift when they diverge.
Picking wrong here would have made the UI disagree with payroll — the reviewer treated this as a
ship-blocker check and it passed.

The completeness note renders per decisions 30/31, and renders *honestly*: net-zero cells show as
absent (`—` / "no cell"), never as a measured `$0.00`; `unitemized_remainder` is labelled **(net)**
with an open disclosure explaining that offsetting gaps cancel. The card was told to pair the
remainder with a count or drill-down — **the endpoint exposes neither**, so it discharged the
requirement with explicit non-reassuring labelling and flagged the gap rather than inventing a
backend change out of footprint. Correct call.

Three defects were found *by reading the screenshots back*, not from code: a price splitting
mid-number at 390px, an axis choice that flattened week-to-week variation, and a `$0.00` total on
empty weeks. That is the self-verification ritual doing exactly what it exists for.

Gate evidence: red-first genuine (10 of 11 failed at the test-only commit). **The 11th — the
null-safety hazard guard — passed vacuously, and G6 caught that**, then forced the forbidden
pattern to prove the guard actually fires. `task sw` verified real by md5 against the precache
revision. 161 passed / 1 pre-existing skip.

**Footprint deviations, both judged necessary:** `tests/inventory.spec.js` (+13/-4, retargeting a
smoke test that asserted the stub this card deletes — F4 set the precedent) and the mockup under
gitignored `.planning/` (see DECISIONS-NEEDED §3).

---

## F5 · inventory-tab-gating — MERGED `c1a2393` — **the important one**

Option (i): two per-tab slugs (`inventory-trends`, `inventory-cost`) via `SeedHQApps`, no migration.
`RequirePermission` passes on tab slug ∨ umbrella `inventory` slug ∨ superadmin, and **fails closed
on DB error**. B5 fold-in gates approve/reject on approver assignment ∨ admin ∨ superadmin.

### G6 FAIL — a live bypass, reproduced

`POST /api/v1/workflow/ops` routed `APPROVE_ITEM`/`REJECT_ITEM` straight to the *unexported
repository functions* via exported aliases, **from the same cookie-auth group** as the REST route —
identical reachability, zero extra privilege. The card gated the two handlers; the op router called
the same mutations with no check at all.

Reproduced live as a `team_member` with zero assignments:

```
POST /workflow/approveSubmission -> 403   status stays "pending"     <- gate worked
POST /workflow/ops APPROVE_ITEM  -> 200   status="approved", reviewed_by=<eve>
POST /workflow/rejectItem        -> 403   0 rejection rows
POST /workflow/ops REJECT_ITEM   -> 200   status="rejected", 1 rejection row
```

Both forged mutations **emitted an op and broadcast over the sync hub**, so a forged approval
propagated to every connected client as legitimate.

**Why the card missed it:** it was scoped verbatim to two named handlers, and gated exactly those.
This codebase exposes the same mutations through a second path. `handler.go:41` already carried a
comment about another function needing "the same gate as the REST handler" — the dual-path hazard
had bitten before and was papered over locally rather than structurally.

**Fixed by closing the class, not the instance.** `requireReviewAuthz` now runs inside
`approveSubmission`/`rejectItem` themselves, so all four call sites inherit it. Patching the router
branch would have fixed the instance and left the trap armed for the next caller.

**A near-miss worth recording:** with a low `lamport_ts` the forged op returned **409 Conflict while
the mutation still landed** — the router runs before `EmitOpWithConflictCheck`. A naive reading of
that 409 would look like the gate working. G6 verified the fix by deliberately racing a superadmin
op ahead of the forged one: post-fix it returns 403 regardless of lamport ordering, because the
refusal happens before the conflict machinery is consulted.

### Gate evidence on the final tree

- Tab gate attacked across 13 variants and held: path normalization (`//`, `/./`, `%6e`, `;`, `%20`
  with `--path-as-is`), case-sensitive slugs, disabled `hq_apps` rows, orphaned grants, wrong-app
  grants, service-token crossover, no/bogus cookie (401, envelope distinct from 403 per §1.2).
- **Fails closed on DB error** — verified by renaming `app_permissions` mid-flight: 500, handler
  unreached. Superadmin short-circuits *before* the query, so an outage doesn't lock out the one
  role that would need to act during one. Both directions now test-guarded.
- **0 logged-in-only bypass** proven by `curl` from a gated session, no browser involved. Client
  hiding is UX only.
- No over-gating: approver, admin, superadmin all succeed through both doors.
- Null-safety: all 14 top-level listeners in `inventory.html` verified to bind into `#s1`/`#s7`
  (ungated, always rendered). `#s5`/`#s6` hold one node each, both document-delegated. **Zero
  listeners orphaned** — the un-granted page parses clean, asserted via `pageerror`.
- Mixed Trends-only case (§1.6) exercised both directions, screenshot-verified.
- A4 untouched — the branch modifies zero files under `.night-crew/`.

**Self-disclosed git incident:** the implementer mis-aimed a `git commit --amend`, clobbering a
commit message, and recovered from reflog. I verified independently — `git diff 4729c85 f077e70` is
empty, content byte-identical, SHA-only change, no stash, no force-push. Harmless, and the
disclosure is the behavior you want.

**Footprint:** `internal/db/db.go` judged in-scope (the card's scope sentence names `SeedHQApps`
even though the file list omits it). `tests/workflows.spec.js` (+154) judged necessary — `package
main` is unimportable, so Go can prove the mutation refuses but not that the router maps
`ErrNotAuthorized` → 403.

**Held out of scope deliberately:** `unsubmitChecklist` and `OpSaveTemplate`/`OpArchiveTemplate` are
the same class of unauthorized mutation but outside B5's named scope. **Not fixed. See
DECISIONS-NEEDED §1** — that item is the one that prevents a third recurrence.

---

## D1 · syncspec-deflake — **PARKED** (worktree preserved)

Worktree: `…/scratchpad/wt-d1`, branch `card/d1-syncspec-deflake`.
**Net diff against base is empty** — the attempted fix and its revert are both preserved in history
so the diagnosis isn't lost. `sync.js`/`sw.js` byte-identical to base, so **the attended two-device
check is NOT re-armed by this card** (it remains armed from the 07-22 change, per the slate's own
Preconditions).

**PARK trigger fired exactly as written:** the real fix requires reworking catch-up re-render
semantics in production `sync.js`, beyond a narrow determinism seam. The attempted coalescing fix
made the two target tests green but turned previously-green tests red — coalescing **moved** the
clobber from "many, early" to "one, late" rather than removing it. The implementer ran the
full-suite leg, saw the regression, and reverted. That honesty cost ~24 minutes and is why the card
ran 180m against an 80–125m estimate. **It is the correct outcome**, and G6 independently confirmed
`PARK CORRECT` — reproducing its own (disjoint) regression set, which establishes the regressions
are nondeterministic and therefore that no narrow fix can converge against them.

**The implementer disclaimed its own 10/10 green streak as invalid evidence.** G6 confirmed and
sharpened this: because the regression set is nondeterministic, *no* streak over a target subset can
establish safety in this suite at all. That should inform how `cycle-gate`'s "suite-green
attestation" is defined.

**Diagnosis corrections — read DECISIONS-NEEDED §0a–0d before scheduling `cycle-gate`.** In short:
`:1198` is probably not flaky (§0a); `:525` is a real product defect but fails only against a
carried-over DB, so the gate's DB precondition decides whether it's even in scope (§0b, §0c); and
the reported third blocker `RUN-10` did not reproduce (§0d).

### My error, stated plainly

**I briefed D1 that the box would be quiet. It was not.** A concurrent night-crew run in a separate
Claude session held the box for much of D1's window; only streak legs 3–10 ran at genuinely low
load, and D1's G6 measured 1.5–3.4 throughout — all *above* the slate's 0.84 reference. Serial
dispatch guarantees *this run* is serial; it does not guarantee the machine is idle. The slate's
"quiet box" deliverable needs a load precondition that is checked, not assumed. This does not change
the PARK verdict — the regression that caused it is deterministic in kind, if not in membership —
but it does weaken every timing-derived number D1 produced.

---

## Verification state of the final tree

Last full green legs, on the merged tree at `c1a2393` (F5's confirm pass):

- `CI=1 go test -p 1 -count=1 ./...` — **8 packages ok, 0 FAIL**, ephemeral pg16 on a Docker-assigned
  port. Host `:5432` never touched by any card.
- Playwright, isolated stack — **515 passed, 6 skipped, 0 failed**.
- `go build ./... && go vet ./...` green.

`-p 1` was used on every Go leg (standing, not discovery). `CI=1` on every suite/G6 leg. The `:8199`
latch was cleared by killing the listener PID where it recurred. No suite run was backgrounded. No
`git stash` in any worktree. No `node_modules` symlink committed.

**Not run, and still the operator's:** `task sandbox:e2e` / the attended two-device convergence
check (armed since the 07-22 `sync.js` change), and the prod deploy. Frontend semver 1.0.3 is
untouched — the bump belongs to `/save-project` at deploy time.

---

## Next actions, in the order I'd take them

1. **Read `DECISIONS-NEEDED.md` §0a–0d before touching `cycle-gate`.** Its premise is partly
   refuted; scheduling it as written would burn a card.
2. **Decide §1** — `unsubmitChecklist` and the template ops are the same class as the bypass F5's G6
   caught. The standing-rule option (every `/ops` branch carries the same authz as its REST twin) is
   what stops a third recurrence.
3. **Decide §2** — the slate template's OpenSpec clause doesn't match this repo. Four silent
   workarounds in one night.
4. Triage §5 (the `NUMERIC(10,2)`/`(10,4)` mismatch) alongside the known `float64` item — they may be
   one card.
5. Merge `overnight-20260720c` → `dev` at morning triage, then flip the three DRAFTING roadmap rows
   to DONE.
6. Remove `…/scratchpad/wt-d1` once D1's diagnosis has been read and acted on.
