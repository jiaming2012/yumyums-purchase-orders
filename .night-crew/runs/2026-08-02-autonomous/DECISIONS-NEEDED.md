# DECISIONS NEEDED — run 20260802

Forks this run could not resolve, plus calls it made that need ratifying.
**The run never decides a PRODUCT fork; it executes.** Batch sign-off (2026-08-01) covers the
signed specs and nothing else.

**Nothing parked. No card raised an operator-only fork.** Every item below is either a decision
the slate delegated and the run implemented, or a judgement the run made and is surfacing.

---

## R1 — B-13: the Taskfile is right, the doc is wrong · **RATIFY OR REVERSE**

The slate DECIDED this rather than parking it, under the delegation principle.
`precache-manifest-from-head` made `build-sw.js` read **git HEAD**, and the prod clone's HEAD after
`git reset --hard origin/main` **is** the shipped tree — so regenerating `sw.js` on the box is
redundant by construction, and `sw.js` committed with its source is the correct contract.

Card P1 fixed `CLAUDE.md` to match `Taskfile.yml:178-221`. G6 verified the correction against the
Taskfile line by line and found it right.

🛑 **P1 also corrected FOUR further false claims in the same deploy block**, none of which the
slate anticipated, all verified by G6 against `Taskfile.yml`:

| Old claim | Truth |
|---|---|
| deploy SSHes to a Windows box over Tailscale | **No SSH at all.** Prod builds on this box from a clone pinned to `origin/main` (`PROD_REPO`) |
| `task prod:ssh` / `PROD_SSH` exist | **Neither exists anywhere in the repo** |
| container is `yumyums-hq` | it is **`yumyums-prod`** |
| — | `task prod:rollback` and `PROD_COMPOSE` **exist and were undocumented** |

**Why this deserves a moment at triage rather than a nod:** the previous doc described a deploy
path that does not exist, and it is the doc an operator reads *while deploying*. Ratifying R1
ratifies these too.

---

## R2 — P1's scope call: the precache count moved **29 → 31** · **RATIFY**

The slate gave two different formulations: the **mechanism** (*"exit non-zero if any resolves to a
skipped path"*) and the **invariant** (*"nothing precached may import something not precached"*),
labelling the latter "the actionable" one. P1 implemented the **invariant**, so its guard also
fires on a target that was never globbed at all.

That surfaced a live defect with no synthetic case: **`log.js` is `src=`'d by all 7 precached pages
and `tab.js` by 5, and neither was in `globPatterns`.** They ship via `COPY *.html *.js`, so they
have always worked online and always failed offline — and `tab.js` applies `#tab=N` synchronously
**before paint**, so five of seven tools opened on a returning offline client with every tab
section visible at once and no switching.

**Tonight was under a hard scope freeze, and adding two files to the shipped precache manifest is a
real product change reaching every phone.** It is surfaced here rather than buried for that reason.

A fresh-context G6 verified the claim independently from git history (`git log -S` confirms this
card is the only commit ever to add `'log.js'` to `build-sw.js`), confirmed both files are already
staged into the image by the existing `COPY` so no `Dockerfile` change was needed, and judged it
**the right call, not scope creep** — the narrow reading exits 0 on the merged tree and leaves the
defect live, and that defect is **D-KR2's exact subject**.

**31 is the new invariant.** 🛑 It is stated in prose and enforced by nothing — see `B-54`.

---

## R3 — Ledger decision 108's reporting rule: **KEPT, and amended** · **CONFIRM**

Card A2 initially recorded that B-36's fix meant *"the package `ok` line means something again"* and
that decision 108's rule could be retired. **Its G6 proved that false**, and the fix round confirmed
it independently:

```
HQ_SYNC_SUBSTRATE_OPTIONAL=1  →  ok  github.com/yumyums/hq/internal/sync  0.012s   exit 0
                          -v  →  --- SKIP: TestJWTBridgeRLS / --- SKIP: TestRowVisibilityRLS
-run TestSpikeGate            →  ok, exit 0, zero attack variants run
```

B-36 closes **one** road to a silent skip; it does not close the road. **The rule stays**, amended:
an `internal/sync` result is reportable only when it cites `-run TestRowVisibilityRLS -v` with
subtests **executed** *and* states `HQ_SYNC_SUBSTRATE_OPTIONAL` was unset.

Had this been retired tonight it would have quietly reopened the hole card A2 exists to close.

---

## R4 — A2 needed a migration on a card told it would need none · **RATIFY**

The roadmap bullet said A2 *"must not need a `backend/migrations/` file"*. It added
**`0074_sync_fdw_approver_view.sql`**.

Reason: `assignment_role` deliberately does not cross the FDW (`sync-schema/sql/0002` §3a), so
decision 111's own `hq_can_approve_template` is **not evaluable on the substrate** without HQ-side
plumbing. The migration is **one read-only VIEW and one grant** — no table, column, constraint or
role.

G6 verified every part: measured un-widening (`0 rows` outside the relation that already crosses the
FDW), ACL is SELECT-to-`hq_sync_fdw`-only, Up/Down/re-Up idempotent, Down fails closed, and `V20`
asserts the substrate-side refusal directly rather than trusting the revoke. **Verdict: should not
have parked** — the slate's park condition is scoped to "a write predicate beyond decision 111's
four rows", which did not fire; `0074` is the *plumbing* for row 4, not a fifth row.

---

## R5 — `V18` was rewritten in place · **RATIFY**

`TestRowVisibilityRLS/V18` previously proved `submission_rejections` is deny-all even for admins.
A2 gives that table a SELECT policy, so V18 now asserts something different. **Rewriting a passing
test to match new behaviour is how a regression gets laundered**, so it was put to G6 explicitly.

G6's verdict: **authorised, not laundered.** Decision 111's four-row table says
`submission_rejections` *"gains a SELECT policy"*, and consequence (1) gives the mechanical reason
(a device that can write a row it cannot read back breaks replication). The rewrite also **changed
the subtest title**, so the old assertion cannot be silently mistaken for the new one, and the new
form is strictly stronger — it keeps a refusal half, keeps the `service_role` control, and adds
three-way discrimination.

---

## R6 — 🔴 `HQ_SYNC_REST_URL` remains ARMED and UNSET

Verified set **nowhere** in the merged tree by every card that touched the replication path and by
the orchestrator's own final gate. Every occurrence is a comment, a doc, or the constant that names
it (`backend/internal/sync/proxy.go`).

**It disarms only at triage, on evidence, never by a run asserting it.** `sync-hard-cutover` (S1) is
the card that first sets it in a real deploy.

---

## R7 — The armed reds all PASSED tonight, and none is retired

By FULL TITLE (decision 100 — never by line anchor):

- `list page progress decrements when another device unchecks a field [LST-17]` — 🛑 the bare tag
  matches **two** tests; the armed one is `tests/sync.spec.js:446`
- `a queued submission still lends its idempotency_key at 7:30pm CT [A1-TZ-02]`
- `submitted checklist survives builder edit with assignment change [LC-02]`
- **B-27** — `tests/inventory.spec.js:883 › Inventory › item modal pre-fills search with current
  line item text`

**B-27 passed in every single run tonight** — six-plus full suites across five legs. **Per decision
100 that retires nothing**, and no card claimed it fixed. It is recorded here because a reader of
the closeout will notice, and the honest reading is in `B-45`: **this suite's baseline is a
distribution, not a list.**

---

## Open forks

**None.** No card reached an operator-only question.

## Parked cards

**None.** No card parked.

**P2 `workflow-unsubmit-failnote-reattach` and P3 `sync-banner-builder-tab-scope` were never
started** — a budget decision by the control loop, not a park. See HANDOFF.md "What is NOT done"
for their destinations.
