# DECISIONS-NEEDED — overnight-20260720c

> # ✅ RESOLVED 2026-07-21 — recorded as ledger T-20 (decisions 34–41)
>
> All forks in this file are closed. Kept as the analysis record. Summary:
>
> | Item | Resolution |
> |---|---|
> | **0a** | `:1198` **IS flaky** — 16% / 20% under contention. The refutation was RETRACTED in-file (see below). Card KEPT and **RE-AIMED** test-side; scope widened to LST-17 + GATE-04. |
> | **0b** | CLOSED — gate runs on a clean DB (preference P1). |
> | **0c** | Direction set (P2, separate schemas). Audit delivered; surfaces #2/#5/#9 fixed at triage, #3 fixed by the `hq_test_go`/`hq_test_e2e` split (decision 38). **#4 remains open** — dev/prod/test still share one cluster. |
> | **0d** | CLOSED — `RUN-10` RESOLVED under P3. |
> | **1 (HIGH defect)** | FIXED before triage — `requireApprover` moved inside the template write (`0057638`). |
> | **1-B** | **Decision 34** — `SAVE_TEMPLATE` + `ARCHIVE_TEMPLATE` gated to admin, matching REST. `EXCEPTIONS` empties. |
> | **1-C** | **Decision 35** — STANDING RULE: `/ops` never diverges from its REST twin. `EXCEPTIONS` kept as a tripwire, not a waiver. |
> | **2** | **Decision 37** — a NIGHT-CREW tooling defect, pushed back to that repo as urgent. NOT "amend the slate to cite GSD" — the operator is refactoring away from GSD. Sequence: urgent backlog item → OpenSpec pros/cons investigation → formal hq refactor request only if OpenSpec is kept. No hq change. |
> | **3** | `.planning/` gitignore vs the mockup clause — left as-is; `git add -f` is the established repo-wide convention (hundreds of files). Recorded, not fixed. It bit again during triage: `git add` on a tracked `.planning/` file returns non-zero and short-circuits `&&` chains. |
> | **4** | Design §2.2 prose stale vs the shipped endpoint — documentation fix, still OPEN. The design file is a signed artifact; amending it is an attended act, not a run's. |
> | **5** | `NUMERIC(10,2)` vs `(10,4)` — remains a BACKLOG item, to be carded alongside the `float64`-money item as one change. |
> | **5b** | `reviewerID` context-identity seam — latent, folded into the `grant-enforcement-parity` card's authz sweep. |
> | **6** | **Decision 36** — confidentiality, and far broader than asked: the Users tab offers 11 grants and the backend enforces 2. New URGENT card `grant-enforcement-parity`, blocks go-live. |


These are questions the run deliberately did **not** answer, per the standing rule that the run
executes and never decides. Ordered by consequence.

**One card PARKED: D1 · syncspec-deflake.** Its items are 0a–0d below, first because the
`cycle-gate` card depends on them.

> ## ✅ OPERATOR RESOLUTIONS — 2026-07-21
>
> Given directly by the operator during triage. **0b and 0d are CLOSED. 0a and 0c are in
> progress.** The three standing preferences are now recorded in
> `.night-crew/knowledge/preferences.md` §"Test isolation and determinism" so future runs
> inherit them rather than re-asking.
>
> | Item | Resolution |
> |---|---|
> | **0a** | Re-test `:1198` **under concurrent load** — the one condition never tested. If still green, investigate the right disposition (e.g. mark not-flaky). **In progress:** `investigate/1198-under-load-20260721`. |
> | **0b** | **CLOSED — the gate runs on a clean DB.** Standing preference P1: clean DB with fixtures, unless there is a specific reason not to. A non-resetting entry point is a defect in the entry point, not a quirk for tests to absorb. |
> | **0c** | **Direction set — separate schemas, no cross-contamination** (P2), applied everywhere, not just to this defect. A full surface audit is **in progress:** `investigate/cross-contamination-20260721` → `reference/cross-contamination-surfaces-20260721.md`. The `:525` fix is scoped by that audit, not by D1's narrower reading. |
> | **0d** | **CLOSED — `RUN-10` marked RESOLVED.** Standing policy P3: what cannot be reproduced is resolved until it recurs. Reopening on recurrence is expected and is not a process failure. |
>
> **P3 carries a caveat that applies directly to 0a:** "not flaky" (no mechanism, no
> reproduction) and "rare, mechanism understood, bounded at N%" are different findings and
> must not be collapsed. `:1198` has a named mechanism in its own de-flake comment — a
> clobber race. If the load test stays green, that distinction decides the label.

---

## 0a. ~~The `:1198` premise does not survive~~ — **RETRACTED. The test IS flaky.**

> ## ⛔ THIS SECTION WAS WRONG — corrected 2026-07-21
>
> Everything below the line was written before the test was run **under contention**. It
> concluded the flake premise was "decisively refuted." **That conclusion was wrong** and is
> retracted. Full evidence: `reference/1198-flake-reproduction-20260721.md`.
>
> **REPRODUCED: 4 reds in 25 `--retries=0` legs (16%); 4/20 (20%) under a concurrent
> Playwright suite.**
>
> **Why ~20 prior greens proved nothing.** Both earlier parties sampled the *quiet*
> condition. A ~20% rate conditional on contention is entirely consistent with ~20 greens
> off-condition (p ≈ 1.2%). The p ≈ 1×10⁻⁶ figure below is arithmetically fine and
> epistemically worthless — it tested against a 50% *unconditional* rate that nobody had
> claimed, on samples drawn from the wrong condition. **A streak in the wrong condition is
> not evidence.**
>
> **Two corrections to the ORIGINAL claim — both real, neither fatal to it:**
>
> 1. **The load figure was misattributed.** The source table
>    (`runs/2026-07-22-autonomous/DECISIONS-NEEDED.md` §"S1 tail") reads: leg 1 load **0.84
>    PASSED**, leg 2 load **3.96 RED**. The phrase "proven flaky on a quiet box at load
>    0.84" assigned the *green* leg's load to the *red* one. **"Flaky" survives; "on a quiet
>    box" does not.** All four new reds started at load ≤ 7.21, while greens ran clean at
>    24.77, 21.46, 19.77 — so the discriminator is the **presence of a competing suite**,
>    not load magnitude.
> 2. **The card is aimed at the wrong wait.** Every red is a 12s timeout at
>    **`sync.spec.js:1119`** — the `POST /ops` **commit** wait — *not* `CONVERGE_TIMEOUT` and
>    not a convergence assert. The debounce is 400ms, a 30× margin, and the journal gains
>    **+3 rows on a red vs +5 on a green**: no `SET_FIELD` row. **The op never fired. It was
>    not slow. No timeout increase can fix this.**
>
> **Mechanism.** The prior de-flake *relocated* the clobber race rather than closing it. Its
> step 1 claims gating on the `POST /ops` 2xx is "race-free" — true against a clobber of an
> already-sent value, false against a re-render that stops the POST existing at all. A stray
> WS catch-up `loadMyChecklists` re-render detaches the input between `fill()` and
> `dispatchEvent('change')`, so the change handler never arms `debouncedSaveField`. The
> observable moved from `Received ""` to a commit-gate timeout.
>
> **Verdict under preference P3: "rare, mechanism understood, bounded at ~16–20%."** NOT
> "not flaky." This is precisely the distinction P3 was written to preserve, and it earned
> its keep within hours of being recorded.
>
> **Disposition: KEEP the card, RE-AIM it.** Fix is test-side (settle the runner before
> entry; re-assert and re-dispatch when no POST is observed). **No production change
> needed.** Tier A was 5/5 green but p(0 red in 5 | 20%) ≈ 0.33 — **a quiet box is NOT
> established as safe.**
>
> **`cycle-gate`'s no-retry premise STANDS.** It genuinely cannot pass at ~1-in-5. Item 0b's
> clean-DB resolution is unaffected and still correct.
>
> **Flakiness is recorded in 5 places** — `BACKLOG.md:427-437` (corrected in place),
> `slate-20260720c.md:159,165`, `slate-20260722.md:21,94,103`,
> `runs/2026-07-22-autonomous/` §S1-tail + HANDOFF, and this file. **No test-level
> quarantine exists** — only the suite-wide `retries: 1` in `playwright.config.js`.

---

### Original (wrong) analysis, retained for the record

**Status: ~~the slate's evidence is refuted by two independent parties.~~ RETRACTED — see above.**

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

## 1. `/ops` ↔ REST authorization parity — complete enumeration, one HIGH defect, one product fork

**Found by:** F5's G6 reviewer (incidentally, 2 items). **Enumerated exhaustively:** follow-up
sweep `followup/ops-authz-coverage-20260721`, 2026-07-21.
**Status:** enumerated and now guarded by a standing test. **Nothing was fixed** — no production
code was touched by the sweep. Two things came out of it and they are kept structurally apart:
one **HIGH defect** that needs a fix and no decision, and one **product fork** (§1-B) that is
still yours and on which the sweep holds no view.

The class: `POST /api/v1/workflow/ops` dispatches through `workflowOpRouter`
(`backend/cmd/server/main.go`) to the same workflow mutations the REST routes call, from the
**same cookie-auth group**. Two doors, one mutation. F5 shipped a handler-only gate and its
reviewer walked straight around it. The previous version of this section listed the two further
instances the reviewer happened to trip over. That list was incidental **and one entry was wrong.**
Here is the whole surface.

### Verification status of every claim below

One false claim was found in this section (the reachability bullet under the HIGH defect — struck,
see there). Finding one means the class was worth a sweep, so every remaining factual assertion here
has been re-checked and is labelled. This document is what the operator reads cold; nothing in it
should be believed on the author's say-so.

| Claim | How verified |
|---|---|
| The six-row enumeration — mutations, REST twins, and each side's authz | Read in source (`main.go`, `workflow/handler.go`, `workflow/repository.go`) **and** exercised at runtime by `tests/ops-authz-coverage.spec.js` |
| `SAVE_TEMPLATE` / `ARCHIVE_TEMPLATE` ungated on `/ops`, admin-only over REST | **Runtime** — unprivileged `team_member` got 403 at both REST routes, 200 at `/ops`; the archive leg genuinely archived a template |
| `default:` returns 400 `unknown_op_type`, so unlisted ops open no door | Source (`main.go:160`) **and** now a runtime assertion (see the guard section) |
| `unsubmitChecklist` enforces submitter-ownership and has no `/ops` twin | Source (`repository.go:1158`); no op type routes to it; `workflows.spec.js [RUN-11]` already pins the 403 |
| REST twins reject `requires_approval` with no approver (400 `requires_approver`) | Source — `handler.go:372` and `:429`, both calling `hasApprover` |
| The `/ops` `SAVE_TEMPLATE` branch never calls `hasApprover` | Source — the branch unmarshals and writes with no validation call |
| That gap leaves a submission stuck `pending` in nobody's queue | **Runtime, by G6** — reproduced end-to-end (200 via `/ops`, 201 submit, absent from `pendingApprovals`). Not independently re-run by this sweep; attributed |
| `/ops` update branch skips the transactional op-emit | Source — REST calls `updateTemplateAndEmit`, the router calls `UpdateTemplateFunc` = plain `updateTemplate`; `sync/handler.go` records the op after the router returns |
| `SET_FIELD` / `SUBMIT_CHECKLIST` succeed against unassigned templates | Source — neither `saveResponse` nor `submitChecklist` performs any assignment check (only field-existence / archived-template checks) |
| Neither op carries an attribution field | Source — `SaveResponseInput` is `{field_id, value}`; `SubmitChecklistInput` carries no user field; the writer id comes from the request context in both doors |
| `workflowOpRouter` cannot be imported by a Go test | Source — it is in `package main`, and `backend/cmd/server/` contains no `_test.go` files |
| F5's original bypass broadcast a forged approval over the sync hub | **Not re-verified by this sweep** — inherited from F5's G6 record and stated there. Load-bearing for none of the findings above |

### The enumeration — every op type `workflowOpRouter` handles

| Op type | Mutation | REST twin | Twin's authz | `/ops` path authz | Agree? |
|---|---|---|---|---|---|
| `SET_FIELD` | `saveResponse` — upserts a **draft** response row keyed `(field_id, answered_by)` | `POST /workflow/saveResponse` | authn only | authn only | ✅ yes — self-scoped, no privilege to escalate |
| `SUBMIT_CHECKLIST` | `submitChecklist` — creates a submission attributed to the caller, sweeps the caller's own drafts into it | `POST /workflow/submitChecklist` | authn + `validateFailNotes` + `validateResubmitPhoto` | authn + the **same two** validators (router calls `ValidateFailNotesFunc` / `ValidateResubmitPhotoFunc`) | ✅ yes |
| `APPROVE_ITEM` | `approveSubmission` — `status='approved'` | `POST /workflow/approveSubmission` | `requireReviewAuthz` (approver∨admin∨superadmin) | **same check, same function** — it lives inside the mutation as of `8c71022` | ✅ yes (was the F5 blocker) |
| `REJECT_ITEM` | `rejectItem` — inserts `submission_rejections` + `status='rejected'` | `POST /workflow/rejectItem` | `requireReviewAuthz` | same, inside the mutation | ✅ yes (was the F5 blocker) |
| `SAVE_TEMPLATE` | `insertTemplate` (create) / `updateTemplate` (update) — authors or **rewrites** a checklist template | `POST /workflow/createTemplate`, `PUT /workflow/updateTemplate/{id}` | **admin-only** (`isAdmin`, D-11) | **none** | ❌ **NO** |
| `ARCHIVE_TEMPLATE` | `archiveTemplate` — soft-deletes a template by id | `DELETE /workflow/archiveTemplate/{id}` | **admin-only** (`isAdmin`, D-11) | **none** | ❌ **NO** |

That is the complete set. `workflowOpRouter`'s `default:` branch returns 400 `unknown_op_type`, so
no other op type opens a door. **Both divergences were verified live**, not read off the source: a
freshly-invited `team_member` with zero grants was refused 403 at both template REST routes and
served 200 at `/ops` for both `SAVE_TEMPLATE` and `ARCHIVE_TEMPLATE` — the archive leg genuinely
archived a template it had no other way to touch.

### Correction: `unsubmitChecklist` is NOT in this class

The previous version of this section said `unsubmitChecklist` had "no authz check at all." **That
is false.** `unsubmitChecklist` (`backend/internal/workflow/repository.go:1158`) enforces
submitter-ownership inside the mutation — `submittedBy != userID` → `not the submitter` → 403
`not_submitter` — and additionally refuses once `status='approved'`. `tests/workflows.spec.js`
`[RUN-11]` already pins the non-submitter refusal. It also has **no `/ops` twin**: no op type
routes to it, so it has exactly one door and no parity question to answer. **Item (a) of the old
decision list is withdrawn — there is nothing to decide there.**

Whether ownership is the *right* rule (vs. also letting an approver pull a submission back out of
the queue) is a separate product question and is not raised here.

### ⛔ DEFECT (HIGH) — not a decision: `/ops` `SAVE_TEMPLATE` skips the `requires_approval` invariant

**This is separated from the product question below on purpose. It is a defect awaiting a fix, not
a decision awaiting an operator.** No product answer to §1-B makes "a template that demands
approval but names no approver" correct, so nothing here is blocked on that fork — it is
independently fixable today.

Both REST twins reject `requires_approval: true` with no `approver` assignment (400
`requires_approver`). The `/ops` `SAVE_TEMPLATE` branch skips that validation entirely.

**Proven end-to-end by G6**, not inferred: an unprivileged `team_member` created a template with
`requires_approval=true` and zero approver assignments (200 via `/ops`, 400 via REST); a submission
against that template was accepted (201); and the submission **does not appear in the admin's
`pendingApprovals`**. It sits `pending` in nobody's queue, with no in-app path to resolution.

Graded **high**, above the authz divergence below, because:

- it is **silent** — nothing surfaces to the crew member or the admin;
- it is **persistent** — the stuck submission does not age out or resolve;
- it **corrupts the approval ledger** rather than merely widening write access;
- it **requires no privilege** — any authenticated `team_member` can do it, with no grant, no
  assignment, and no trace beyond the ops journal.

**Reachability, stated accurately:** it requires a **deliberate, hand-crafted `/ops` call**. An
earlier draft of this section claimed the frontend routes template saves through `/ops`, so
ordinary crew would trip it by accident. **That was false and has been struck.** Verified against
source: `submitOp` (`sync.js:676`) has exactly ONE call site in the entire frontend
(`workflows.html:211`) and it emits only `SET_FIELD`. Template saves go over REST
(`workflows.html:1232` → `createTemplate` / `updateTemplate/{id}`; `:1295`, `:1318` →
`archiveTemplate`), and the *server* emits the `SAVE_TEMPLATE` / `ARCHIVE_TEMPLATE` ops from those
handlers. The `op_type` string matches in `sync.js:405-540` are the **inbound** `applyOp` consumer,
not outbound emitters. So: no privilege required, but not accidental either.

The grade stands at **high** on the four bullets above — the corrected reachability lowers the
*volume* of accidental exposure, not the severity of the corruption or the absence of any
privilege barrier.

Not fixed here (the sweep was tests + docs only, by rule). Whoever fixes it: the check is
`hasApprover(input.Assignments)` when `input.RequiresApproval`, already written and used by both
REST handlers — the `/ops` branch simply never calls it.

### Also on the same door (low) — the `/ops` update branch skips the transactional op-emit

`PUT /updateTemplate/{id}` calls `updateTemplateAndEmit`, which writes the template and queues its
`SAVE_TEMPLATE` re-render op in ONE transaction (FR-5, INV-1 — the op that tells other devices to
re-fetch can never be lost while the write is accepted). The `/ops` branch calls plain
`updateTemplate` and relies on `OpHandler` recording the op afterwards, outside the write's
transaction. A convergence gap, not a security one.

### What the sweep landed instead of a fix

`tests/ops-authz-coverage.spec.js` — a standing guard that encodes the invariant as
**parity between the two doors**, not as "everything must be gated":

- `PARITY_GATED` (`APPROVE_ITEM`, `REJECT_ITEM`) — both doors must refuse an unprivileged caller.
- `PARITY_OPEN` (`SET_FIELD`, `SUBMIT_CHECKLIST`) — both doors must accept. If either side is
  gated later without the other, this goes red too.
- `EXCEPTIONS` (`SAVE_TEMPLATE`, `ARCHIVE_TEMPLATE`) — documented divergence, each entry naming
  **§1-B below** as the open decision. The exception list is the living form of that question;
  it is a record, not a waiver.

Three anti-rot properties, each proven red before the file was committed:

- The covered op set is **derived by parsing the router's own `switch`** and asserted equal to it.
  Adding `case opsync.OpAnything:` without a coverage entry fails the build (proven: injected a
  `NUKE_EVERYTHING` op → red).
- Removing a gate fails (proven: deleted `requireReviewAuthz` → red; and re-armed the exact F5
  bypass — gate in the handler only — → red on the `/ops` assertion specifically).
- Exceptions assert the `/ops` path is **still open**, so a stale entry cannot pass as fiction
  (proven: gated `SAVE_TEMPLATE` while leaving its exception in place → red).
- Neutralising the derivation fails loudly rather than passing empty (proven → red).
- **`DEFAULT-REFUSES`**: a runtime leg posts an unrecognised `op_type` and requires a 4xx. G6 found
  that making `default:` permissive (`return nil, nil`) — accepting and broadcasting every unknown
  op — left the equality test green, because the derived and covered sets were untouched. No source
  parse can catch that; the HTTP leg does. Every "this op is not in the router, therefore it is a
  closed door" conclusion in this section depends on it, so it is now enforced rather than assumed.
- The derivation itself cannot pass vacuously. G6 defeated the first version — an end-of-line
  anchor made `case opsync.OpX: // comment`, `case opsync.OpX: return nil, nil`, and multi-line
  grouped labels *invisible* rather than erroneous, so the suite went green with a live uncovered
  door. Fixed by stripping Go comments and string literals, brace-matching the switch body, and
  matching each label up to its terminating colon rather than to end-of-line; plus a
  `MIN_ROUTED_OPS` floor so any future silent narrowing trips regardless of cause. All three
  variants and the floor re-proven red.

The guard is HTTP-level because `workflowOpRouter` lives in `package main`, which has no test files
and cannot be imported; and because "is this path actually ungated?" is a runtime question that no
amount of source reading answers.

### Decision needed — the product question (and only the product question)

Everything above this line is fact or defect. Everything below is genuinely yours.

**(B) Should crew be able to mutate templates at all?** The sweep did **not** answer this, did not
recommend an answer, and holds no view. The observable facts are in the enumeration; the call is
yours. Note the two shapes are separable — *authoring/editing* a template (`SAVE_TEMPLATE`) and
*archiving anyone's template by id* (`ARCHIVE_TEMPLATE`) may well not deserve the same answer.

**(C) Should the `/ops` router carry a standing rule** that every op branch enforces the same authz
as its REST twin? The guard now *measures* parity and forces a deliberate entry either way — but
whether a divergence is ever permissible is still a policy call. If the answer is "never,"
`EXCEPTIONS` should be emptied by resolving (B), and the bucket kept only so the next divergence
cannot ship silently.

**(A) is withdrawn** — see the `unsubmitChecklist` correction above. There is nothing to decide.

### Backlog notes (raised, not decisions for now)

- **Unassigned crew can fill and submit against any template.** `SET_FIELD` and `SUBMIT_CHECKLIST`
  succeed against templates the caller holds no assignment on. Both doors behave identically, so
  the parity invariant holds and their ✅ above is correct — and neither op carries an attribution
  field, so nothing can be written as another user. But whether unassigned crew *should* be able to
  submit against an arbitrary template is a scope question nobody has asked. Verified safe at the
  DB row level; raised only so it is on the record.
- **The guard is a source parse of one `switch`.** If dispatch ever moves to a map, a table, a
  second router, or a helper called from `default:`, those ops become invisible to
  `tests/ops-authz-coverage.spec.js` and it would go quietly green. Its `MIN_ROUTED_OPS` floor
  catches silent *narrowing*, not *relocation*. Anyone changing how `workflowOpRouter` dispatches
  must revisit that parser; this note exists because a code comment alone cannot enforce it.

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
