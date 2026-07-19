# QA Coverage Findings — why five escaped defects slipped, and whether our agents could have caught them

**Date:** 2026-07-18
**Investigation brief:** `reference/handoff-qa-coverage-investigation-20260718.md`
**Scope:** read-only coverage analysis. The five defects (`5c423ac`, `93c74a4`, `08ebb4f`, `9b1aa32`, `b224c79`) are already fixed on `dev`; this report explains the *coverage gap*, not the fixes.

---

## TL;DR

1. **The night-crew QA process missed all five defects because its coverage is authored per-phase and per-component, while every defect lived in a *cross-cutting combination* of state dimensions (role × assignment × observer × op-type × nesting × derived-view) that no phase's State Enumeration Table, and no pre-existing test, ever enumerated.** The pre-existing suite ran one identity (admin) playing assignee *and* approver, drove only field-edit ops cross-device, and never rejected a sub-step or a non-photo field with require-photo.
2. **ui-jury, as configured, could not have found any of the five.** It captures 7 pages × 1 `at-rest` state, one identity, default tab only, no `db_seed`, no interactions, single browser session — and its critics are negative-heuristic detectors (visual hierarchy, mobile layout, console/network errors) with **no expected-content oracle**. Four of the five defects are completely *silent* (no console/network/backend error), so even a perfect capture of the buggy state would produce a plausible-looking screenshot and a clean log.
3. **Reconfigured in principle**, ui-jury could *reach* defects 3–5's states (seeded lifecycle fixtures + per-role passes) and could plausibly *detect* only defect 4's resubmit-block (the one noisy signal: HTTP 400). It could not reach defects 1–2 at all (no multi-session capability exists anywhere in the pipeline) and could not detect 3 or 5 even if reached (silent semantic defects, no oracle).
4. **The operator's thesis is confirmed with evidence:** there are entire state dimensions — concurrent second identity, cross-device propagation, seeded submission lifecycle, sub-step rejection state, and expected-content semantics — that no current automated tool reaches. Manual play found the bugs precisely because a human varies several of these axes at once, for free.

---

## 1. Why the night-crew QA process missed all five

### 1.1 The process's coverage shape (what it actually tests)

**The Definition of Done is phase-scoped and component-scoped.** `CLAUDE.md` requires a State Enumeration Table (empty/loading/error/success + ≥2 edge rows), a screenshot self-verification ritual, and a verifier-subagent gate — but each of these is authored **by hand, per phase, for the phase's component**. Nothing in the process enumerates the *cross-phase* combinatorial space (roles × lifecycle × nesting × observers). A phase that builds the rejection banner enumerates the banner's states; a phase that builds sub-steps enumerates sub-step states; **no artifact ever enumerates "rejection banner × sub-step"** — which is exactly where defects 3 and 5 lived. The structural blind spot is that the enumeration discipline is strong *within* a component and absent *between* components.

**The pre-existing automated suite had a very specific shape** (evidence from `tests/` + git provenance, verified against the parent of `5c423ac`):

- **Identity:** every test logs in as `jamal@yumyums.kitchen`, DB role `admin` (`sync.spec.js:4-5`, `workflows.spec.js:4-5,77`). Zero tests log in as superadmin. A genuine manager/team_member lane exists only in `workflows.spec.js`'s "Approval Flow" and attribution blocks — and role-switching there is **sequential re-login on one page**, never two concurrent sessions.
- **Assignment:** the dominant pattern is a template assigned to role `admin` as **both assignee and approver** — the same logged-in admin plays submitter and approver (e.g. `sync.spec.js:604-607`; `persistence.spec.js:952-955`). The "neither-but-can-view" relationship (an admin editing a checklist assigned to someone else) was never driven on the JS side.
- **Second session (observer):** 26 `newContext` two-device tests, **all in `sync.spec.js`**, and pre-regression they covered only field-edit / structural-edit / submit / unsubmit convergence. **Approve/reject was never asserted on a second device.** `workflows.spec.js` and `persistence.spec.js` have effectively zero concurrent-observer coverage.
- **The W-3 convergence matrix** (`sync.spec.js` ~1011+) is a **field-TYPE matrix**: it proves checkbox/yes-no/text/temperature/sub-step/fail-note/photo each converge — for *one admin assignee editing their own checklist*. It varies axis F while holding A, B, C(actor=observer identity), and D (`SET_FIELD` only) fixed.
- **Backend:** `backend/internal/sync/` had **zero Go tests** before `5c423ac` (`git log --all --diff-filter=A -- 'backend/internal/sync/*_test.go'` returns only that commit). The entire fan-out/recipient-resolution layer shipped untested. The resubmit-photo gate had Go coverage only for **photo-type fields** with URL values.

### 1.2 Per-defect: the untested cell each bug lived in

| # | Defect | The cell outside the tested box |
|---|---|---|
| 1 | Cross-user live sync (`5c423ac`) | **Editor ∉ assignees.** Every convergence test's editor was the assignee, so `ResolveEntityAccess`'s recipient resolution was never exercised for a non-assignee editor. (Already root-caused in `qa-gap-20260717-live-sync-access.md`.) |
| 2 | Reject/approve stale cache (`93c74a4`) | **Lifecycle op × second device × derived view.** All cross-device tests drove `SET_FIELD`; `APPROVE_ITEM`/`REJECT_ITEM` were only ever driven single-session, so nothing asserted that a status change reconciles the banner, readonly mode, or list count on an observer device. |
| 3 | Sub-step rejection feedback invisible (`08ebb4f`) | **Nesting × rejection.** Sub-steps were tested for *convergence* (W-3, W-6b); rejection was tested at *top level* only. No test ever rejected a sub-step. |
| 4 | Require-photo dead-end (`9b1aa32`) | **Field-type × rejection-modifier.** The gate's Go tests used photo-type fields whose *value* is a URL; no test rejected a checkbox/text/yes-no/temp field with `require_photo=true`. The sub-step hard-block adds the nesting axis on top. |
| 5 | Rejected sub-step stays checked (`b224c79`) | **Nesting × rejection × reopen.** The uncheck-on-reopen behavior was tested (implicitly) top-level; sub-step done-state lives in the parent's `sub_steps` map — a storage asymmetry no test touched. Latent until defect 3's fix made sub-step rejections visible at all. |

**The common structure:** each defect sits at the intersection of at least two axes the suite held fixed. The suite is dense along single axes (7 field types converge; 7 field types persist) and empty at the intersections. The operator found all five in two days of play because manual use naturally varies role, device, nesting, and lifecycle *simultaneously*.

### 1.3 What the regression commits fixed — and the residual test-shape gap

The five commits added the first sync-package Go tests, the first cross-device reject/approve tests (`RJT-LIVE-01/02/03`), and the first sub-step-rejection tests (`APR-SUBSTEP-*`). But note: **the RJT-LIVE tests still run one admin identity playing both submitter and approver across two devices.** No test yet combines a second live device with a *distinct* actor role (e.g. a real manager rejecting while a real team_member's device observes). The BACKLOG's consolidated WO ("cross-user × op-type live-sync matrix + sync-package unit coverage") is the right home for closing that; it is not yet done.

---

## 2. Could ui-jury have found them?

### 2.1 What ui-jury is (capability), verified from the agent defs

ui-jury is a **visual/runtime review pipeline** over a *declared* surface:

- **Cartographer** — BFS from seed URLs (defaults: depth 2, 50 routes, 300s). Extracts anchor links from the a11y tree and clicks up to **10 non-anchor nav elements per page** (`button:not([type="submit"]), [role="tab"], [role="menuitem"], a:not([href])`). **No login/credential facility at all** — it crawls unauthenticated, one isolated session, one pass. Its Step 2.5 "multi-fixture impact observation" *can* run `db:reset`/`db:seed` hooks and semantically diff a11y snapshots per fixture — but its prerequisite check (`yq '.db_seed | keys | length'` = 0) makes it **self-skip in hq** ("impact.skipped: no fixtures in hooks.yaml").
- **Seeder** — pure planner. With no `db_seed` map it emits one `default` group covering all states and the orchestrator **skips seeding entirely**.
- **Driver** — per state: desktop+mobile screenshot/DOM/a11y, console NDJSON, failures-only network NDJSON. **Single browser context** (recycled every 5 routes), **one identity** from the top-level `setup` block. It can execute up to 10 *declared* `interactions[]` per state — hq declares **none**.
- **Critics** — hierarchy (visual hierarchy/brand, desktop), mobile (touch targets/responsive collapse at 393×852), correctness (**reads only console/network/backend logs, sees no pixels**; severity keyed to `pageerror`→Blocker, 5xx/requestfailed→Blocker, 4xx→High). All three are **problem-detectors with an explicit "zero findings is valid" contract. There is no expected-content field anywhere in routes.yaml, hooks.yaml, the app-map, or any critic's input contract.** The only content-comparison capability in the pipeline is Cartographer 2.5's baseline-vs-fixture *change* diff — which detects that content changed, not that expected content is missing, and is skipped in hq anyway.

### 2.2 How it is configured in hq (the crux)

From `docs/ui-jury.md` (quoting its own "Known caveats (v1)"): *"Tab-default state only… Other tabs need their own `states[]` entries… deferred to v2"*; *"No `db_seed:` map. Single default fixture for v1."*

- **7 routes × 1 `at-rest` state each.** `workflows.html` captured on the default "My Checklists" tab; Approvals and Builder never captured.
- **One identity:** `jamal@yumyums.kitchen` via the top-level setup block. No role variation.
- **`db_reset` truncates exactly `purchase_line_items`, `purchase_events`, `pending_purchases`** — it preserves users and templates but seeds **no workflow instances**. No submitted / rejected / rejected-with-photo / approved state ever exists in a run.
- **No interactions declared** on any state; single session; annotator re-navigates un-seeded and logged-out.

### 2.3 The defect × tool matrix

"Reachable" = could the tool arrive at the buggy state. "Detectable" = could it flag the defect once there. *(as-cfg = as currently configured; recfg = in principle, reconfigured within the tool's actual capabilities.)*

| Defect | night-crew E2E (pre-existing) | Cartographer (discover) | Driver (capture) | Critics (detect) |
|---|---|---|---|---|
| **1** cross-user live sync | **Not reached.** Had the two-context harness but never varied editor-vs-assignee. | **No, as-cfg or recfg.** One unauthenticated session; cannot be a non-assignee editor, cannot observe fan-out. | **No, as-cfg or recfg.** Single browser context by construction; a fan-out bug is invisible without a second concurrent session. | **No.** Defect is **silent** — the op is simply never sent to the device; no console/network error. No convergence oracle exists. |
| **2** reject/approve stale cache | **Not reached.** Approve/reject never driven cross-device. | **No.** Requires an approver acting on a submitter's live submission while a third session observes — three concurrent sessions. | **No.** Same single-session limit; also no rejected state exists (no seed). | **No.** **Silent** — `loadPendingApprovals()` runs cleanly and is a no-op; the stale banner/count produce no error and a plausible screenshot. |
| **3** sub-step banner missing | **Not reached.** No test rejected a sub-step. | **As-cfg: no** (no seeds → 2.5 skipped; BFS can't produce a rejected submission by clicking). **Recfg: reachable** — a `rejected-substep` `db_seed` fixture + a crew-login pass puts the state one navigation away (sub-steps render expanded by default; `workflows.html:2286-2292`). | **As-cfg: no. Recfg: yes** — captures the reopened runner with the (missing) banner. | **No, even recfg.** **Silent.** The screenshot shows a normal-looking sub-step row; nothing tells a critic a banner *should* be there. Oracle gap. |
| **4** require-photo dead-end | **Not reached.** Gate tests used photo-type fields only. | **As-cfg: no. Recfg: reachable** via a `rejected-require-photo-on-checkbox` fixture + crew pass. | **As-cfg: no. Recfg: yes**, and with a declared `interactions[]` step that presses Submit it can *trigger* the block. | **Partially, recfg only.** The dead-end itself (no attach control) is **silent**. But the resubmit attempt is **noisy**: HTTP 400 `resubmit_photo_required` (`handler.go:189,671`) → failures-only network log → correctness critic maps status≥400 to High/Blocker. **This is the single defect of the five that a reconfigured ui-jury could plausibly flag** — and only the gate half, not the missing-control half. |
| **5** sub-step stays checked | **Not covered** (latent behind defect 3). | Same as defect 3 — recfg reachable via seed. | Same — recfg capturable. | **No, even recfg.** **Silent.** A checked checkbox is a fully plausible screenshot; only a "rejected ⇒ unchecked after reopen" invariant catches it, and no such oracle exists. |

### 2.4 Verdict on ui-jury

The handoff's hypothesis is **confirmed, with one refinement**. ui-jury is a *visual/runtime regression* tool over a *declared, single-identity, default-tab, unseeded* surface:

- **As configured, it could not have found any of the five.** Every defect lived behind at least one of: a lifecycle state that never exists in a run (no seeds), a second concurrent session (doesn't exist), or a semantic expectation (no oracle).
- **Reconfigured, it could reach 3–5 and detect only 4** (the gate half, via the 400). Defects 1–2 are unreachable in principle: nothing in any stage's definition supports two concurrent sessions or identities, and single-session capture is blind to fan-out by construction.
- **Refinement to the hypothesis:** the nesting axis is *not* an interaction-depth problem for ui-jury — sub-steps render expanded in the runner, so once a rejected-sub-step state is *seeded*, the driver captures it with zero clicks. The sub-list gap is really a **seeded-lifecycle gap plus an oracle gap**, not a crawl-depth gap. (Cartographer crawl depth matters for tab/route discovery, not for sub-steps.)
- One honest caveat in ui-jury's favor: Cartographer Step 2.5 is a real, dormant capability — with lifecycle fixtures it would at least *discover* that workflows.html's content varies by fixture and suggest interactions. It is a change-detector, though, not an expectation-checker.

---

## 3. Capability vs gap: the operator's thesis, confirmed

> *"There are many states in the app that are never looked at until I manually go through different roles, sub-lists, and combinations."*

**Confirmed.** The gap taxonomy, each axis with today's coverage and the defect(s) that prove it:

| # | Gap axis | Who covers it today | Proof |
|---|---|---|---|
| **G1** | **Multi-identity** — more than one concurrent role/identity | **Nobody.** E2E: one admin plays all parts (even the new RJT-LIVE tests); role switches are sequential re-logins. ui-jury: one identity, hard-coded. Cartographer: unauthenticated single session. | Defects 1, 2 |
| **G2** | **Cross-device / live-propagation** — a second session observing while the first acts | **Partially:** `sync.spec.js` only, and pre-regression only for field-edit/submit ops. ui-jury: structurally single-session, no stage supports two contexts. | Defects 1, 2 |
| **G3** | **Seeded lifecycle** — submitted → rejected (→ require-photo) → approved states | **Nobody, outside hand-written per-bug tests.** ui-jury has no `db_seed`; these states are unreachable by single-session clicking from a clean DB (they need an approver acting on a submitter's submission). The E2E suite creates them ad hoc per test, never as a reusable enumerated fixture set. | Defects 2, 3, 4, 5 |
| **G4** | **Nesting / sub-lists** — sub-step versions of every field-level behavior (rejection, uncheck, gating) | **Partially:** sub-step *convergence* and *persistence* are tested; sub-step *rejection-lifecycle* states existed nowhere until the regression tests. No enumeration forces "whatever is asserted at field level must also be asserted at sub-step level." | Defects 3, 4 (hard-block), 5 |
| **G5** | **Behavioral oracle** — asserting expected content/state, not absence of errors | **Only hand-written test assertions.** ui-jury's critics are negative-heuristic by contract ("zero findings is a valid output"); 4 of the 5 defects are fully silent — plausible screenshot, clean console, clean network. No invariant layer exists ("a rejection with a comment must render a banner"; "a rejected id must be un-done after reopen"). | Defects 1, 2, 3, 5, and 4's dead-end half |
| **G6** *(bonus, observed)* | **Derived-view reconciliation** — the *thing that must update* (banner, count, checked-state, readonly flip) as a distinct assertion target from the raw value | Pre-regression: only raw field values and list counts, and only for field-edit ops. | Defect 2 (both symptoms), 5 |

The state space the operator walks by hand is roughly `{4 roles} × {3 assignment relations} × {observer yes/no} × {4 op types} × {2 nesting levels} × {7 field types × 3 rejection modifiers} × {5 derived views}`. Automated coverage today is dense along exactly one axis at a time and empty at the intersections. **Every one of the five escaped defects is an intersection cell.**

---

## 4. Recommendations, ranked by leverage

**R1 — Build the state-model matrix generator (highest leverage; the automation of what the operator does by hand).**
Enumerate `{role} × {assignment} × {op-type} × {nesting} × {observer} × {derived-view}` for the checklist engine and emit a driveable Playwright plan. The BACKLOG's consolidated WO ("cross-user × op-type live-sync matrix + sync-package unit coverage") **is the right home** — but extend its axes: as written it covers G1/G2 (access × op-type); add nesting (G4) and derived-view (G6) columns so sub-step cells are generated, not hand-remembered. Most cells are cheap: the W-3 matrix already proves the pattern of generating tests from a table. Even a pruned matrix (pairwise rather than full Cartesian) would have contained all five defects' cells.

**R2 — Generalize the two-session observer harness (closes G2 permanently, and G1 with it).**
`sync.spec.js` already builds two contexts ad hoc 26 times. Extract it: a helper that wraps *any* mutation with "and assert convergence on a second context within N seconds, on the derived views, not just the value." Crucially, let the second context log in as a **different role** than the actor — today even the new RJT-LIVE tests use one admin on both devices, so the distinct-approver/distinct-observer cells are still unmodeled. Once the helper exists, every future lifecycle feature gets cross-device coverage nearly for free.

**R3 — Add a behavioral-invariant layer (closes G5; the only thing that catches silent semantic defects).**
Distinct from both the visual critics and per-bug regression tests: engine-level post-condition invariants asserted after any test (or any seeded state), e.g.:
- every `submission_rejections` row with a comment ⇒ a correction banner renders for that field-or-sub-step id on reopen;
- every rejected id ⇒ un-done state on reopen (top-level *and* inside any parent's `sub_steps` map);
- every `require_photo` rejection ⇒ an attach control exists in the DOM;
- rejected status ⇒ runner is editable, list count is live (not snapshot).
Defects 3, 4 (dead-end half), and 5 are each a one-line violation of one of these. Invariants scale where enumerations don't: they check every state any test happens to reach.

**R4 — `db_seed` lifecycle fixtures + per-role passes for ui-jury (closes G3 for the visual pipeline; cheap, bounded win).**
Fixtures: `submitted`, `rejected-with-comment`, `rejected-require-photo` (on a non-photo field *and* on a sub-step), `approved-with-feedback` — crossed with crew/manager login passes and per-tab `states[]` (Approvals, Builder). This un-skips Cartographer Step 2.5, lets the hierarchy/mobile critics finally *see* these states, and with one declared Submit interaction would have flagged defect 4's 400. Be honest about the ceiling: by itself this catches layout/runtime problems in lifecycle states, not silent semantic ones — it's the *reach* half; R3 is the *detect* half.

**R5 — Cartographer depth (multi-role crawl, sub-state discovery) — lowest leverage for this defect class.**
Useful for route/tab discovery drift, but the five defects show the bottleneck is state *creation* (seeding) and *judgment* (oracles), not discovery. Do R4 first; 2.5's fixture-impact observation then gives most of this for free.

---

## 5. Residual: what stays economically un-auto-findable

Even with R1–R4 in place, manual play remains the backstop for:

- **"What should this mean" defects** — behavior that is *wrong* but matches no encodable invariant (e.g. a rejection flow that is technically consistent but confusing; the *decision* that sub-step require-photo should be advisory rather than gating was a product judgment, not a derivable rule). An invariant layer only catches violations of rules someone thought to write.
- **Cell explosion at full fidelity** — the full Cartesian space is thousands of cells; automation will run a pruned (pairwise/risk-weighted) slice. A defect requiring three-way interaction of axes that pairwise pruning separated can still slip. (Defect 5 needed nesting × rejection × reopen — a genuine 3-way cell.)
- **Real-device/timing texture** — iOS PWA quirks, WS reconnection under real network jitter, and races outside Playwright's deterministic timing. The LWW/convergence tests approximate but don't reproduce a phone on truck Wi-Fi.
- **Novel-state genesis** — the first time a new feature creates a state *category* (as correction-photos did), no fixture, invariant, or matrix row exists yet by definition. The process fix is that R1's matrix and R3's invariants must be updated as part of a phase's Definition of Done — otherwise each new axis re-opens the gap this report closes.

---

## Appendix: evidence sources

- Prior root-cause: `reference/qa-gap-20260717-live-sync-access.md` + 2026-07-18 addendum; `BACKLOG.md` §"Escaped defect + QA gap" (WO 1 + WO 2 = the consolidated matrix WO; WO 3 resolved).
- Commit messages: `git log 5c423ac~1..b224c79` (full root-cause detail per fix).
- Test-shape evidence: `tests/sync.spec.js` (W-3 ~1011+, FLD-LIVE, RJT-LIVE), `tests/workflows.spec.js` (APR-*, Approval Flow ~2531), `tests/persistence.spec.js` (FLD-CORRECTION-PHOTO), `backend/internal/sync/access_test.go`, `backend/internal/workflow/resubmit_photo_gate_test.go`; provenance via `git log -- tests/sync.spec.js`.
- ui-jury capability: agent defs (`ui-jury-{cartographer,seeder,driver,critic-*,verifier,annotator}.md`), `docs/ui-jury.md` caveats, `.ui-jury/routes.template.yaml`, `.ui-jury/hooks.yaml` → `db-reset.sh` → `task backend:db-reset-inventory` (truncates only `purchase_line_items`, `purchase_events`, `pending_purchases`).
- Product-code verification (reach/observe/signal per defect): `backend/internal/sync/ops.go:474-557`, `backend/internal/sync/listener.go:56-71`, `sync.js:444-468`, `workflows.html:383, 1535-1577, 2065-2068, 2172-2204, 2271, 2286-2294, 2337, 2409-2420, 2743-2758`, `backend/internal/workflow/handler.go:138-224, 671`.
