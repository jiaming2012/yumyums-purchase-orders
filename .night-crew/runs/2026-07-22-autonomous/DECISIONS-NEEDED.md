# DECISIONS-NEEDED — overnight-20260722

> Open forks from the autonomous run of 2026-07-22. Reader = the operator.
> Resolve via `/nc-morning-triage`. The run did not decide any of these.

---

## F1 — `trends-spend-by-group-endpoint` — **PARKED** (card's own PARK trigger fired)

**Status:** implementation complete and technically sound, **not merged**. Worktree preserved at
`scratchpad/wt-f1`, branch `card/f1-trends-endpoint` @ `88cab9d` (2 commits: red-first test
`fc0f08d`, implementation `88cab9d`). Nothing from F1 is on `overnight-20260722`.

**Why parked:** the card's PARK trigger reads — *"if proration can't reconcile with
`period-summary` on the shared fixture (AC-6 identity breaks) → PARK with the failing
reconciliation case; do not ship a number that disagrees with payroll's."* The identity held on
the fixture as authored, but the G6 adversarial reviewer demonstrated that **the fixture is rigged
on every axis that matters** (one line item per event, `Σlines == total − tax` by construction,
every event hardcoded `mercury_category='COGS'`, zero pending rows). With realistic fixtures the
identity breaks **five independent ways**.

**This is not an implementer error.** The implementer followed the SIGNED design §2.2 verbatim and
self-flagged the `mercury_category` divergence rather than silently patching it — the procedurally
correct call. **The design is what's defective.** The fix is a design amendment, which is
operator-only; the run correctly refused to improvise it.

### The failing reconciliation cases (G6-measured, Trends vs `period_summary.cogs_incl_tax`)

| Probe | Scenario | Trends | period-summary | Δ |
|---|---|---|---|---|
| **B1** | Partial line coverage — event total 110.00, tax 10.00, only 90.00 itemized (a delivery fee on the receipt, not a catalog line) | 99.00 | 100.00 | **−1.00** |
| **B2** | One reviewable `pending_purchases` row in window | 110.00 | 185.00 | **−75.00** |
| **B3** | One `mercury_category='Software'` event (500.00) beside a COGS event | 610.00 | 110.00 | **+500.00** |
| **B4** | 20 cells, 4-dp prices, 8.25% tax — per-bucket `ROUND` vs whole-window `ROUND` | 220.0000 | 220.0300 | **−0.03** |
| **B5** | Event with `total == tax` (25.00/25.00) — `NULLIF` guard ⇒ factor 1, lines are 0 | 0.00 | 25.00 | **−25.00** |

**B1 alone is disqualifying** — it is the *normal* case on real receipts (any unitemized fee), not
an exotic edge.

**Root cause:** Trends and `period-summary` are structurally different computations.
`period-summary` = `Σ(qty×price) + Σ(tax) + Σ|pending.bank_total|`; Trends =
`Σ(qty×price × total/(total−tax))`. These are algebraically equal **only** when every event's lines
sum exactly to `total − tax` **and** there are no pending rows. Real receipts satisfy neither.

### The fork — three operator decisions, all design amendments to §2.2

1. **`mercury_category` filtering.** `period-summary` filters `mercury_category = ANY($allowlist)`
   at three CTEs (`handler.go:1335`, `:1350`, `:1406`). Design §2.2's SQL sketch has **no such
   filter**, so Trends has none. On production data the gap is **unbounded** — it is the entire
   non-COGS half of the Mercury feed (software, equipment, fuel, insurance, rent), landing on a tab
   labelled "spend by group" that sits next to payroll's COGS number.
   **G6 recommendation:** add `AND pe.mercury_category = ANY($3)` with the same allowlist
   `period-summary` uses, threaded through `TrendsHandler(pool, cogsAllowlist)`.
   *Magnitude on real data is unmeasured — G6 had no production access. The direction
   (Trends over-reports) is certain; the size is not.*
2. **`pending_purchases`.** Trends never reads them; `period-summary` counts unreviewed reviewable
   rows as COGS. Does Trends include them, or carry a completeness note (as it already does for
   `unlinked`, which is a different gap)?
3. **What "reconciles with `period-summary`" MEANS when `Σlines ≠ total − tax`** — which on real
   receipts is the common case. Until this is stated, AC-6 is not a testable contract. §2.1 also
   claims Trends reconciles with `menu-cogs` (a *third* distinct number, since `menu-cogs` applies
   `AND pli.purchase_item_id IS NOT NULL` plus recipe allocation) — **untested by this card and
   unexamined by G6.**

### Knock-on: **F3 `trends-tab-frontend` could not run tonight**

Per the slate's own dependency note — *"if F1 PARKs, F3 cannot run (no API) — skip to F2/F4."*
F3 was not dispatched. Track F ran F2 → F4 → F5 instead.

### To un-park

Amend design §2.2 on the three points above, then re-dispatch F1 as a card whose red-first test
uses an **unrigged** fixture — G6's B1/B2/B3/B5 are ready-made cases to seed it with. The existing
implementation is a good base; the SQL needs the category filter, a pending-rows decision, and a
rounding-site change.

---

## S1 tail — `:1198` determinism — **ANSWERED BY THE RUN, and the answer is "genuinely flaky"**

S1 parked its de-flake tail (b) unable to distinguish *"still flaky"* from *"the box was saturated
by Track F."* Both S1 and its G6 said a quiet re-run was owed and neither would claim determinism.
**The orchestrator ran that quiet streak after Track F finished** (all subagents done, no Docker
builds, no competing Playwright, load average **0.84** at start — the first genuinely quiet window
all night). Isolated pg16 (`nc-quiet`, Docker-assigned `:33040`), `CI=1`, `--retries=0`,
fresh-dropped `hq_test` per leg, full `tests/sync.spec.js` (58 tests):

| Leg | Start load | Result | Wall |
|---|---|---|---|
| 1 | 0.84 | **58/58 PASSED** | 6.2m |
| 2 | 3.96 | **57/58 — `:1198` RED** | 6.4m |
| 3 | 6.48 | **ABORTED** — killed by the orchestrator's own 10m command timeout, not a result | — |

**Conclusion: `sync.spec.js:1198 › temperature answer converges` is NOT deterministic on a quiet
box.** It reddened 1-of-2 quiet legs with no competing load. This converts the open question into a
settled fact:

- **The PARK was correct.** The tail is a real card, not a load artifact. S1's PARK trigger
  anticipated exactly this ("plumbing beyond the gate + test-side deterministic waits").
- **S1's gate did not cause it** — its G6 independently refuted the implementer's self-flagged
  exposure mechanism (`fillState.activeTemplate` is already set when the wait is armed, so the gate
  does not suppress in that window), and `:1198` is a pre-existing intermittent going back to the
  07-21 run.
- **Do not read leg 1's 58/58 as reassurance.** A single green streak leg is exactly the evidence
  that made this flake look load-caused for two runs running.

**Next step (per S1's own recommendation, now evidence-backed):** a structural look at
`survivalCell`'s 12s `CONVERGE_TIMEOUT` budget versus the real WS round trip. Price it as its own
card. **Also fold in `:525 FLD-LIVE-02`** — S1's G6 found it fails 3/3 in isolation *and at the
pre-gate baseline `ffc474d`*, i.e. a pre-existing **order-dependent** test unrelated to the gate.
The `sync.spec.js` flake surface is broader than "just `:1198`", and the de-flake card should be
scoped to both.

*Honesty note: leg 3 was terminated mid-run by an orchestrator timeout. It is reported as aborted,
not as a pass or a fail. Two quiet legs is thinner evidence than the 5-leg streak the card wanted —
but one red on a quiet box is sufficient to settle the question in the negative, which is the
direction that matters.*

---

## F2 — `cost-margin-endpoint` — **MERGED**, two forks routed to you

**Status:** G6 verdict **PASS-WITH-FINDINGS**, merged. The fixture was independently confirmed
*genuinely unrigged* — it survived 8 adversarial fixtures (over-allocation to 160%, refunds,
duplicate menu names, $9.9M values, 20-ingredient rounding drift) and the red→green test-file diff
is **0 lines** (tests were not weakened to pass). Both deviations from signed §2.3 were adjudicated
**legitimate on the design's own text** (line 360-362: *"revenue + units shown, cost/margin marked,
never a silent 0"* — unsatisfiable under menu-cogs's INNER JOIN, so the wider join is required).
The live HTTP path was verified 401-unauthenticated / 200-authenticated, closing the gap F1's
reviewer could not check.

Two HIGH bugs were fixed in a scoped revision round (negative-revenue guard + an untested movers
tie-break). **Two questions are yours** — the run declined to decide either:

### Fork F2-a — "recipe exists, zero window spend" publishes a flattering 0

A menu item that **has** a recipe, but whose ingredients had no purchases inside the 12-week
window, returns:

```json
{"menu_item_name":"Bread Basket","units_sold":50,"revenue":250,
 "ingredient_cost_total":0,"margin":250,"food_cost_pct":0,"unallocated":null}
```

0% food cost, 100% margin, `unallocated: null` asserting the row is fine. This is **the same
"flattering meaningless number"** the implementer cited to justify its null-not-zero deviation —
that path was guarded, this one was not. G6 rates it **arguably the more likely prod case** (bulk
buys landing outside a rolling window). Cost *amplifies* it: menu-cogs reports `0.00` cost but
publishes no margin or pct, so no consumer ever sees "0% food cost" — **Cost invents that number.**

**Why the run did not fix it:** the correct fix needs a third `unallocated` reason string, and
menu-cogs's vocabulary is only `"no recipe"` / `"partial allocation (X%)"`. Coining a third is a
design amendment, i.e. yours. The behavior is unchanged and a code comment now names the gap at the
site so the next reader does not read it as intended.

**Your call:** coin a reason string (e.g. `"no spend in window"`) with null cost/margin/pct, or
accept the 0 as correct-for-the-window and state that in §2.3.

### Fork F2-b — no residual field on the envelope (G6 rates this the card's most substantive gap)

`menu-cogs` publishes **`unallocated_cogs`** — the residual dollars not attributed to any recipe.
**Cost publishes no equivalent.** An operator summing the Cost tab's `ingredient_cost_total` column
gets a number structurally *below* true COGS with nothing indicating a residual exists. This is
**not** inherited — menu-cogs *has* the field and Cost dropped it. §2.1's "money never silently
dropped" is the relevant clause. Adding it is a contract-shape decision.

### Context you should have: Cost diverges from `period-summary` on FOUR axes, not two

The implementer named two; G6 found two more. Cost inherits menu-cogs's semantics, so **every dollar
Cost reports is a dollar menu-cogs already reports** — but none of these agree with payroll's number:

| Axis | `period-summary` | Cost (inherited from menu-cogs) |
|---|---|---|
| `mercury_category` | filters `= ANY(allowlist)` | **no filter** |
| `pending_purchases` | counted as COGS | **ignored** |
| tax | sums `qty*price` raw + event-level tax | **prorates per line** |
| NULL `purchase_item_id` lines | counted | **dropped entirely** |

**The run's position (G6-endorsed, and I agree): this is NOT a park.** F1 was parked for *asserting
an identity that was false on realistic data*. F2 **declines** to assert Cost ↔ period-summary
agreement and instead **pins the divergence in a named test**
(`TestCost_InheritsMenuCogs_NoCategoryFilter_NoPendingRows`) so either side changing breaks loudly.
Agreeing with the signed live contract `menu-cogs` consumes-and-is-consumed-by is the correct
tiebreak — agreeing with period-summary instead would create a *new three-way* inconsistency.
**But the Cost tab is where an operator would naturally compare to payroll**, so resolve this before
the tab ships, alongside F1's amendment. The two cards are asking you the same question.

---

## Standing bugs surfaced (not forks — backlog candidates)

These need no decision tonight but should be routed at triage.

1. **HIGH — `NULLIF(total−tax,0)` silently zeroes money** (F1 probe B5). An event with
   `total == tax`, or lines unparsed while tax is recorded, contributes 0.00 to Trends and its full
   total to `period-summary`. Design §2.1 explicitly promises *"money never silently dropped."*
   Applies to the parked F1 code; check whether the same guard pattern exists elsewhere.
2. **MED — money is `float64` end-to-end** across the inventory package (`period-summary` too, so
   this is a pre-existing repo convention, not an F1 regression). JSON can emit
   `23.099999999999998`; it compounds the B4 per-bucket rounding drift. Cents-as-int or a decimal
   string is the repo-wide correct fix.
3. **MED — parallel-suite cross-package DB interference is a standing hazard.** G6 verified
   independently at the **base** commit (F1 absent): `go test ./...` under default parallel `-p` is
   broadly red — `internal/inventory` (6), `internal/purchasing` (4), `internal/receipt` (9),
   `internal/recipes` (5+). Multiple packages `TRUNCATE` a shared DB concurrently. `-p 1` is green.
   **This means `-p 1` is load-bearing for every card's G1/G2 signal** and should be written into
   the standing run mechanics, not rediscovered per card.
4. **HIGH (run-mechanics) — `.gitignore` lets a `node_modules` symlink into the index.** The line
   is `node_modules/` with a trailing slash, which matches a *directory* but **not a symlink** named
   `node_modules`. Worktrees have no `node_modules`, so symlinking the main repo's install is the
   natural move — and it slips straight past `.gitignore` into any `git add -A`. S1 caught and
   reverted its own instance (`35694ec`), but **the next implementer will hit the same trap.**
   Fix: drop the trailing slash (`node_modules`). Outside every card's footprint, so not fixed
   tonight.
5. **MED (run-mechanics) — the `:8199` port latch recurred, third run running.** A `go run` child
   survives `kill` of its parent and holds the port. S1 worked around it by moving to a private
   `:8231`. Fix the standing recipe to kill the **listener PID** (`ss -ltnp`), not the `go run`
   parent, and assign concurrent tracks distinct `TEST_PORT`s up front.
6. **LOW — `time.Now()` is server-local, not `America/Chicago`.** `trendsWindow` uses the
   container's local zone while `period-summary` is explicit about Chicago (`handler.go:1346`,
   `:1412`, `:1496`). On a UTC container, 18:00–24:00 Chicago computes the window a day ahead; when
   that crosses a Monday it shifts `from` by a **full week bucket**.

---

## F5 — `inventory-tab-gating` — **NOT DISPATCHED** (deliberate budget-discipline drop)

**Not a park, not a failure — an executed instruction.** The slate's budget clause reads: *"prefer a
clean early exit over starting a card you cannot finish cleanly. If quota tightens, drop from the
tail: F5 first (it's the next-morning-friendly card — endpoints ship logged-in-only for one day),
then F4."*

**The call, made at 2h42m elapsed:** F5 is first-of-kind (net-new `RequirePermission` spine across
`users`/`me`/`auth`/`workflow` + `users.html` + `inventory.html`), slate-estimated 45–75m impl **plus
a budgeted revision round** plus G6 — realistically 1.5–2h more, with two live PARK triggers of its
own. Starting it would have risked a **half-landed permission spine**, which is strictly worse than
none: a partially-applied gate reads as protection while leaving holes. The remaining budget went to
closing S1's owed determinism evidence and a proper handoff instead.

**What this means for tomorrow — the exposure is real but bounded:**

- `GET /api/v1/inventory/cost` (F2, merged) ships **cookie-auth logged-in-only**. Any logged-in user
  can read per-menu-item cost/margin data. The slate anticipated exactly this ("endpoints ship
  logged-in-only for one day").
- The **Cost tab (`#s6`) is visible to every logged-in user.** No per-tab gate exists.
- `GET /api/v1/inventory/trends` does **not** exist (F1 parked), so there is no ungated Trends
  surface — F1's park incidentally shrank tonight's exposure.
- **B5 is NOT closed.** `ApproveSubmissionHandler` / `RejectItemHandler`
  (`workflow/handler.go:728-753, 793+`) remain ungated. The operator-specified role rule (approver
  assignment ∨ admin ∨ superadmin, slate header 2026-07-20) is recorded and ready to implement, but
  **no code enforces it tonight.** This was a pre-existing gap (surfaced at T-18/B5), not a
  regression introduced by this run — but it stays open.

**Prep the run leaves for F5** (so the next dispatch is cheaper):

1. **A page-breaking hazard was found and fixed pre-emptively.** F4's reviewer caught that
   `document.getElementById('cost-container').addEventListener(...)` is unguarded at top level — if
   F5 removes `#cost-container` for un-granted users, it throws a TypeError **at parse time and
   breaks the entire inventory page** for exactly those users, every tab. F4 was sent back to make
   it null-safe. **F5 must still check the same pattern for any other element it hides.**
2. **Exact integration points** (from F4, verified): tab button `inventory.html:283`
   (`<button id="t6" onclick="show(6)">`); section `inventory.html:320` (`<div id="s6">`); load
   trigger `show(n)` at `:561`, single line `:574` `if(n===6){loadCost();}`. Gate by guarding that
   line and hiding `#t6`/`#s6` — F4's render code (`:1063–1300`) is self-contained and will not
   collide.
3. **Its states-spec row has a reserved home:** a `test.describe('Cost tab — gating')` block at the
   bottom of `tests/states-cost.spec.js`, with a header comment already spelling out the assertions
   (no `#t6`/`#s6`; direct endpoint → 403) and the `shot()` helper in scope for `edge-ungated.png`.
4. The `<!-- TODO: gate Cost to manager+ via backend roles (INTG-01) -->` in-code marker was removed
   with the Cost stub. Intent survives in the spec NOTE, but the inline breadcrumb is gone.

**Recommendation:** dispatch F5 first next run, before any further tab work. It is now the only thing
standing between the Cost tab and per-tab access control, and it carries B5.

---

## Run-discipline incident — a subagent ran `git stash` in a worktree (forbidden)

**Self-disclosed by the F4 implementer, unprompted.** It embedded `git stash` in a command line as
an intended no-op; the command executed and stashed its uncommitted `tests/inventory.spec.js` edit.
It caught this immediately, recovered with `git stash pop stash@{0}`, and reported it.

**Independently verified by the orchestrator:** the main repo's pre-existing entry
`stash@{0}: WIP on dev: acd2c7f refactor: migrate all server logging from log to slog NDJSON output`
**is intact and untouched.** No work was lost. G6 was additionally tasked with a residue check.

**Why this matters even though nothing broke:** the rule exists because worktrees **share the stash
ref with the main repo** — a stash/pop in a worktree can silently consume or reorder the operator's
own stashed work. This time the recovery was clean and the disclosure was immediate and voluntary,
which is the behavior you want when a rule is broken. But it is a real violation of a standing
prohibition (07-15 hazard) and is recorded here rather than buried in a card report.

**Suggested hardening (not done tonight — outside every footprint):** the prohibition currently
lives only in prose in the slate and subagent briefs. Consider a guard that makes it mechanical —
e.g. a repo-local `git` alias or a pre-command hook that refuses `stash` when `git rev-parse
--git-common-dir` differs from `--git-dir` (i.e. when running inside a linked worktree).

---

## Documentation defect — the inherited-rules pointer is dangling

Every slate from 07-15 through 07-22 inherits its standing rules and gates G1–G6 "unchanged by
reference from `reference/overnight-run-plan-20260707.md`". **That file does not exist** and does
not appear anywhere in the repo. The gates' real origin is
`.night-crew/runs/2026-07-09-attended/slate-20260710.md` § "Run mechanics" (G1–G4, G6), with the
app-code adaptation restated inline in `slate-20260714.md` — which is the form applicable to code
cards. This run executed against that reconstruction.

**Fix at triage:** either create `reference/overnight-run-plan-20260707.md` as the real standing-rules
document, or correct the pointer in the slate template. A dangling inherit in the one document that
defines the run's gates is a latent single point of failure.
