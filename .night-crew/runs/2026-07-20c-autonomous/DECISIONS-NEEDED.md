# DECISIONS-NEEDED — overnight-20260720c

These are questions the run deliberately did **not** answer, per the standing rule that the run
executes and never decides. Ordered by consequence.

**One card PARKED: D1 · syncspec-deflake.** Its items are 0a–0d below, first because the
`cycle-gate` card depends on them.

---

## 0a. The `:1198` premise that promoted D1 does not survive — re-derive or drop it

**Status: the slate's evidence is refuted by two independent parties.**

The slate promoted D1 on the basis that `sync.spec.js:1198` (`survivalCell`) is *"proven flaky on a
quiet box — red 1-of-2 `--retries=0` legs at load 0.84."* That is a ~50% failure rate.

D1's implementer could not produce a red in 15+ observations. D1's G6 reviewer independently could
not produce one in 5 more, spanning journal depth 0 → ~1100 ops and load 1.5 → 3.4. **Combined:
~20 consecutive greens, zero reds, two parties.** Against a claimed 50% rate that is p ≈ 1×10⁻⁶.

Note the 07-22 streak that produced the original claim ran under concurrent load — which the slate
itself warns proves nothing.

**Decision needed:** re-derive the `:1198` flake evidence, or drop it. As written it is the weakest
link in the slate and will consume another card if carried forward unexamined. Raising
`CONVERGE_TIMEOUT` was diagnosed as the *wrong* fix regardless — the mechanism its own de-flake
comment names is a clobber race, not latency.

---

## 0b. Does the cycle-gate run on a reset DB or a carried-over one? This changes what it means

**Status: currently ambiguous, and the answer decides whether `:525` is even in scope.**

D1's G6 found the slate's characterization of `:525` (`FLD-LIVE-02`) as *"fails 3/3 in isolation and
at the pre-gate baseline"* is true **only against a dirty carried-over database.**

Measured: at the point `:525` actually executes in alphabetical order, the journal holds **98 ops**
— and at 98 ops **it passes**. It fails at 614+. The journal only reaches that depth through
**cross-run accumulation**: `task test` drops and recreates `hq_test`; a bare `npx playwright test`
does not.

So the real order-dependence is **cross-run journal accumulation, not within-run ordering** — which
also means a cycle-gate invoked via `task test` may never see this red at all.

**Decision needed:** define the gate's DB precondition deliberately rather than by accident. A gate
that resets is measuring different behavior than one that doesn't, and only one of them exercises
the defect in 0c.

---

## 0c. `:525` is a genuine product defect — scope it wider than D1 was scoped

**Status: confirmed real by both parties. Not a test-cleanliness item.**

Mechanism: the suite shares one server + DB, so the `ops` journal grows monotonically. A browser
context created mid-suite starts at Lamport 0, so `wsCatchUp` replays the **entire** journal. With a
runner open, replayed ops each fire a full `loadMyChecklists` re-fetch. `FLD-LIVE-02` is the
**two-tab** test — both tabs share one browser context and therefore one per-origin connection pool
— so tab B's fetch starves and `#s1` renders empty. `FLD-LIVE-01` (separate contexts) passes, which
is exactly why the two diverge.

**Correction to D1's own diagnosis, found by its G6:** the implementer reported that only the
`SAVE_TEMPLATE`/`ARCHIVE_TEMPLATE` branch was ungated. False. The baseline gate on
`SUBMIT_CHECKLIST` (`sync.js:457`) and `APPROVE`/`REJECT_ITEM` (`:482`) is
`(fillState.activeTemplate || !silent)` — with a runner **open**, which is precisely the failing
condition, those branches storm too. Server logs show frames from `:457`, `:482` **and** `:492`.
`SAVE_TEMPLATE` is the largest contributor *by volume* (structural ops ≈ 53% of the journal), not
the only one.

**This is why D1's attempted fix was insufficient:** `4ab162c` coalesced only the `SAVE_TEMPLATE`
branch, leaving three storm paths intact.

**Decision needed:** scope the follow-up card against **all four** ungated `applyOp` branches. Expect
it to need a merge-aware or edit-aware reconcile inside `loadMyChecklists` — production sync-engine
work that **re-arms the attended two-device check**. Two candidate directions, and the run
deliberately did not choose: **(a)** make the deferred reconcile non-clobbering (preserve dirty local
fields across a replay re-render), or **(b)** bound the replay server-side (cap `ops/since`, or
snapshot/compact the journal) — arguably the better architectural fix, and invisible to the client.

---

## 0d. `workflows.spec.js:2223` (`RUN-10`) — reported as a third blocker, NOT reproduced

D1's implementer reported `RUN-10` (unsubmit → editable draft) failing at the pre-fix baseline as an
independent blocker on the no-retry gate. **D1's G6 could not reproduce it** — passed in a
full-suite-order leg (114/114) and targeted at maximum journal depth.

**Decision needed:** none yet, but **do not schedule work against this without a fresh repro.** It is
either environment-specific to the implementer's stack or already resolved.

---

## 1. Two more unauthorized mutation paths, same class as F5's blocker — gate them or not?

**Found by:** F5's G6 reviewer, while confirming the `/ops` APPROVE_ITEM/REJECT_ITEM bypass.
**Status:** deliberately NOT fixed tonight — outside B5's named scope.

F5's B5 fold-in was scoped verbatim to `ApproveSubmissionHandler` / `RejectItemHandler`. While
proving that gate, the reviewer found the same *class* of hole elsewhere:

- **`unsubmitChecklist`** (`backend/cmd/server/main.go:494`) — an adjacent submission-state
  mutation with no authz check at all.
- **`OpSaveTemplate` / `OpArchiveTemplate`** — template mutations in the same `workflowOpRouter`,
  likewise unauthorized.

I held the run out of these. Gating `unsubmitChecklist` is arguably the same decision you already
made for approve/reject (it moves a submission back out of the approval queue). **Template
mutation is a genuinely unasked question** — it may be intended to be broadly writable.

**Decision needed:** (a) gate `unsubmitChecklist` under the same approver∨admin∨superadmin rule,
(b) decide whether template mutation needs any gate at all, and (c) whether the `/ops` router
deserves a standing rule that every op branch must carry the same authz as its REST twin. (c) is
the one that prevents a third recurrence — the dual-path trap is now documented as having bitten
once and been found twice.

---

## 2. The slate's OpenSpec clause does not match this repo

**Found by:** F1's implementer; confirmed by me directly.

Every card tonight was dispatched with the slate's per-change mechanics: *"draft, `openspec
validate`, implement through tasks.md, atomic commits with the OpenSpec-Change trailer, archive,
flip the roadmap card."*

**There is no `openspec/` directory in this repo,** and `CLAUDE.md`'s enforcement section
describes the GSD workflow instead (`/gsd:quick`, `/gsd:execute-phase`, …).

Cards kept what survives translation — red-first, atomic commits, the `OpenSpec-Change:` trailer,
roadmap flip in the same change set — and skipped scaffolding a tree that does not exist, which I
judged correct under "stay in footprint." But the clause will misfire on every future card.

**Decision needed:** adopt OpenSpec here (`/spec-init`), or amend the slate template to cite the
GSD workflow this repo actually uses. Four silent workarounds in one night is the signal.

---

## 3. `.planning/` is gitignored but the Definition of Done requires committing into it

**Found by:** F3's implementer.

`CLAUDE.md` Definition of Done: *"Commit the mockup … at `.planning/.../<phase>/mockup.html`."*
`.gitignore:5`: `.planning/`.

F3 used `git add -f`. G6 judged that acceptable on the grounds that hundreds of `.planning/` files
are already force-added repo-wide — so the convention is real, just undocumented and at odds with
the ignore rule.

**Decision needed:** narrow the `.gitignore` rule (e.g. ignore scratch subpaths, track planning
artifacts), or drop the mockup clause. Low stakes, but it forces a `-f` on every UI card.

---

## 4. Design §2.2 prose is now stale against the shipped endpoint

**Found by:** F3's G6 reviewer.

`.night-crew/knowledge/designs/prove-surface-gating-and-endpoints.md` §2.2 defines
`reconciles_to_cogs_excl_tax` as `Σcells + Σunlinked + pending_total`.

F1 deliberately ships it as `round(Σlines) + pending_total`, because `Σ(round) ≠ round(Σ)` — the
literal formula disagreed with payroll by a cent on sub-cent unit prices (see HANDOFF §F1). The
struct field doc in `trends.go` states the correct definition and F3 followed it.

**Decision needed:** amend the design prose so a future consumer doesn't reconcile per the stale
text. The code and its field docs are right; only the design document is behind. **This is a
documentation fix, not a behavior question** — flagged rather than done because the design file is
a signed artifact and the run does not edit signed artifacts.

---

## 5. `purchase_events.total` is NUMERIC(10,2) while `price` is NUMERIC(10,4)

**Found by:** F1's implementer, confirmed by its G6 reviewer. Suggested as its own backlog card.

`receipt/worker.go:778` rounds `quantity` at the write boundary but passes the LLM's `price`
through **unrounded** into the 4-decimal column, and `parser.go:24,182` instructs the model to emit
*unit* prices for weight-priced lines (`10.13 lbs @ $5.30/lb`) — so `5.2996` is exactly what the
prompt invites.

Consequence: every receipt with a sub-cent unit price accrues a small permanent discrepancy
between the stored event total and Σ(its lines), which **surfaces in the new Trends tab as
`unitemized_remainder`** — money labelled "we can't account for this" that is actually a schema
artifact. Operators will see cents of phantom coverage noise.

The number Trends publishes is correct either way; this is about not misattributing noise.

**Decision needed:** round `price` at the write boundary, or widen `total`/`tax` to `NUMERIC(10,4)`.
Adjacent to the slate's already-noted "money is `float64` end-to-end" item, which is explicitly out
of scope — these may want to be one card.

---

## 5b. `reviewerID` / context-identity seam in the approve/reject mutations

**Found by:** F5's G6 reviewer, on the confirm pass. Latent, not live.

`approveSubmission` / `rejectItem` now authorize `auth.UserFromContext(ctx)` but attribute the write
to a separately-passed `reviewerID` / `rejectedBy` parameter. Both current call sites pass `user.ID`
from the same context, so they cannot diverge today — but a future caller could authorize one user
and attribute the write to another.

**Decision needed:** low urgency. Deriving the reviewer from the context user inside the mutation
collapses the seam — a one-line follow-up, not a defect today. Worth folding into whatever card
addresses item 1.

---

## 6. Is the Cost grant meant to be confidentiality, or UI tidiness?

**Found by:** F5's G6 reviewer.

The Cost tab's content is **partly reconstructable** from the ungated `/inventory/recipes` and
`/menu-items` routes. This follows from the signed design scoping the gate to the two new
aggregation endpoints, not from any card defect — F5 built exactly what §1 specifies.

But it means `inventory-cost` currently hides a *view*, not the underlying data.

**Decision needed:** if the intent is that un-granted users must not learn per-item cost, the gate
needs to extend to those routes and that is a new card. If the intent is tidiness — keeping a
money-dense tab out of the way of crew who don't need it — F5 is complete as shipped.

---

## Deliberately left open (NOT decisions for tonight)

Per the slate, these stay unresolved and no card touched them:

- **Food cost as a drifting long-term average** (dissolves the 0%-food-cost bug rather than
  patching it) — F2-a stays open.
- **Margin with/without discounting** — F4's red-negative fork stays open. **Blocked on data that
  does not exist:** `daily_menu_sales` has no discount or comp field; needs Toast sync upstream.
- **Design §5 / decision A4** — verified untouched by F5 (the branch modifies zero files under
  `.night-crew/`).
