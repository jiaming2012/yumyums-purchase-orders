# OKRs

<!-- ✅ SIGNED OFF by the operator 2026-07-25 (attended /nc-roadmap-round).

Cycle: "Sync foundation" — replace the hand-rolled WebSocket + Postgres LISTEN/NOTIFY +
Lamport-clock op-log sync layer in `workflows.html` with RxDB (client-side replication) against
a self-hosted Supabase stack, unifying the field-autosave and live-broadcast write paths into
one store. Opened 2026-07-24/25 (attended `/nc-roadmap-round`) after "Prove & surface" closed
2026-07-24 (15 MET · 1 N/A, ledger T-21f, cycle-closeout-20260724.md).

Traces to the roadmap's single authored activity so far (Activity 1: Sync foundation — 4 sync
cards + 1 independent fix card). This round was scoped to ONE backlog group at the operator's
direction; ~19 other `· new` items remain unrouted for a future roadmap-round pass, so these
OKRs cover only what was actually roadmapped, not a full-cycle theme the way "Prove & surface"
did.

OKR-authoring lesson carried from the prior cycle's P4 grading (decision 47): write KR metrics
that cannot be failed by desirable behavior — e.g. deferring the cross-user-hydration item to
the PM session, or dropping the three superseded backlog items, must not redden a KR here.

Previous cycle archived at reference/okrs-2026-07-24-prove-and-surface.md +
reference/roadmap-2026-07-24-prove-and-surface.md. -->

## Product

### Objective: The sync rewrite ships against a de-risked, operator-approved plan — not an assumed one — and the one open product question it surfaces is routed, not guessed at by engineering.
- 0 of the schema/replication or JWT-bridge WOs are dispatched before the feasibility spike (`sync-rxdb-feasibility-spike`) records a written go/no-go verdict (self-hosted Supabase reachable, Go-minted JWTs accepted by PostgREST/Realtime for RLS without GoTrue) in `ledger.md` — measured by comparing the ledger verdict timestamp against `.night-crew/runs/` dispatch timestamps for `sync-rxdb-schema-and-replication` and `sync-jwt-bridge-endpoint`.
- 0 `sync-hard-cutover` WOs are dispatched while the BACKLOG.md entry for "Cross-user checklist hydration divergence" still reads plain `new` — it must carry a PM-session-routed disposition first, not an engineering-guessed fix, measured by the entry's disposition text at the time `sync-hard-cutover`'s WO is dispatched.
- The hard-swap cutover decision (no parallel run) is carried into the `sync-hard-cutover` WO record as an explicit constraint, not rediscovered mid-build. Measured by: the `sync-hard-cutover` WO record in `.night-crew/runs/` stating the no-parallel-run constraint verbatim — 0 build WOs propose a parallel-run alternative.

## Delivery

### Objective: The migration ships with the same discipline the last cycle earned — a proven spike before build, measured cadence, and real prod verification with a returning-client check.
- Feasibility-gate-before-build: 0 of the 3 downstream WOs (schema+replication, JWT-bridge, hard-cutover) dispatch before the spike's go decision lands. Measured by: `ledger.md` decision timestamp vs. `.night-crew/runs/` WO dispatch timestamps (same audit method as the prior cycle's D1, ledger T-18).
- **D-KR2a** — Prod parity is verified this cycle, **independent of any cutover**: `task version` shows prod backend/frontend == local `version.go` constants with 0 drift. Measured by: `task version` output recorded in the ledger. *(REWORDED 2026-08-03 — see the amendment note at the foot of this file.)*
- **D-KR2b** — After each deploy, a **returning** client — an installed PWA that saw the previous version, never a fresh load, never a cleared cache, never incognito (the T-21d lesson, by name) — displays a frontend version read from the **precached** `version.json` that matches the deployed `version.go` `Frontend` constant. Measured by: the version line read on that client and recorded in the ledger. 🛑 Reading the version from `/api/v1/health` does **not** satisfy this KR and never will — the server's value cannot be stale, which is the entire property under test. *(REWORDED 2026-08-03.)*
- **D-KR3** — A **defensible** per-card cycle-time median is computed against the prior cycle's baseline (N=12 / 94m), with (a) N ≥ 8, (b) every excluded card listed with its reason, and (c) a stated sensitivity check showing the median is robust to those exclusions. Measured by: the median table, the Excluded-with-reasons section, and the Sensitivity section in `reference/card-actuals.md`. *(REWORDED 2026-08-03 — stricter than the original, which permitted a median with no sensitivity analysis.)*

## Engineering

### Objective: The new sync layer is provably correct where the old one wasn't — no fetch-storm class, no stale hydration, and the JWT/RLS bridge actually bounds data the way the bearer-token auth did.
- 0 of the 2 superseded fetch-storm-class backlog items (replay-fetch-storm, `sync.js` catch-up gate) reproduce against the new architecture — evidenced by exactly 1 regression test per item in `tests/sync.spec.js` (or its cutover-era successor) asserting no ungated re-fetch storm on catch-up, OR, where no such test is constructible, exactly 1 reviewed architectural-argument note per item in `.night-crew/knowledge/designs/`.
- 0 attack variants bypass RLS under a JWT-bridge attack-variant suite comparable in rigor to `grant-enforcement-parity`'s 13 variants (invalid `role` claim, expired token, missing `sub`, wrong signature, token replay after grant revocation) — measured by a new Go test file under `backend/internal/sync` (or the auth package hosting the JWT-bridge endpoint), modeled on `tests/grant-enforcement-parity.spec.js`'s attack-variant structure.
- **E-KR3** — Exactly 1 documented owner exists per offline data class **and direction** — static assets → Workbox; checklist offline *reads* → one named owner; checklist offline *writes* → one named owner — with **0 classes whose owner is unstated or ambiguous**. Any deliberate two-store split is named as such, with its reason and its divergence risk stated. Measured by: 1 design note in `.night-crew/knowledge/designs/`, reviewed at the cycle gate, cross-checked against the `build-sw.js` / RxDB-init diff. **Gradeable whether or not a cutover shipped.** *(REWORDED 2026-08-03 — the original demanded "0 dual ownership", which decision 126's deliberate parallel run makes unmeetable by construction.)*

## QA

### Objective: This migration does not repeat "Prove & surface"'s QA gap — a sync-adjacent package shipping with zero tests until an escape forced the issue — and night-crew's CI constraints on this activity are priced in up front, not discovered at 3am.
- New code carries real coverage from its first WO, not retrofitted later: the RxDB replication layer and the JWT-bridge endpoint each ship with unit/integration tests in the SAME WO that introduces them (the `sync` package's 0-Go-tests gap from last cycle does not recur in its replacement). Measured by: each card's WO record in `.night-crew/runs/` and its merged diff showing test files alongside implementation files, not in a later card.
- **Q-KR2** — Every WO **whose deliverable includes a code change** carries red-run evidence: the named test, the tree or commit it was captured red against, and the green after. Measured by: a mandatory `## Red-first` section in each card's `merge-intent.md`, asserted by the run's launch prompt. A WO whose deliverable is documentation, audit or a spike records `n/a — no code change` explicitly, and an **absent** section grades the KR down. *(REWORDED 2026-08-03. The original named a field that has never existed in the record format — of every merge-intent this cycle, exactly one carries a `## Red-first` heading. **This cycle grades UNAUDITABLE, not MET**; the reword takes effect from the next run.)*
- **Q-KR3** — Every slate document carries, once, a **Gate cost** section stating the expected full-suite runtime and the load-sensitivity risks in play for that night's cards. Measured by: that section's presence in `reference/slate-<runid>.md`. 🛑 **Backfilling a signed slate is prohibited** — a missing section grades the KR down and is not repaired retroactively. *(REWORDED 2026-08-03 — per WO → per slate, which is where the note actually lives. **This cycle grades PARTIAL and stays PARTIAL**: `slate-20260801.md` and `slate-20260802.md` lack it and will not be edited.)*

---

## 🛑 Amendment — five KRs reworded mid-cycle, 2026-08-03 (operator sign-off)

**What changed:** D-KR2 (split a/b), D-KR3, E-KR3, Q-KR2, Q-KR3. Full before/after text, the
evidence each grade rests on, and the reasoning: `reference/okr-completion-plan-20260804.md` §4.

**Why a mid-cycle reword is legitimate here, and where the line was drawn.** These OKRs were
authored **2026-07-25** against an architecture disproved **2026-08-03**: card S1b
`sync-hard-cutover` parked on the measurement that RxDB replicates to a *second* Postgres and
nothing carries a checklist row back, and ledger decision **126** then retired the card and chose
reads-on-RxDB / writes-on-REST. Four of the five above were written to measure the retired shape
or to read a field the process never built. The test applied to each:

> **A reword is honest when it changes what is *measured* while preserving what is *protected*.
> It is laundering when it lowers the bar on the thing the KR exists to guarantee.**

Every reword above preserves its protected property, and two are **stricter** than what they
replace — D-KR3 now forbids silent exclusions the original permitted, and D-KR2b now names the one
shortcut (`/api/v1/health`) that would silently defeat it.

🛑 **E-KR1 was NOT reworded, deliberately.** Its subject did not move under it: `sync.js` is still
in the tree and both fetch-storm mechanisms are live at `:443-454` and `:475-479`. Rewording it to
grade green would be the laundering case. **It grades NOT MET**, its two "superseded" backlog items
are un-dropped, and the class carries into the next cycle. This entry exists so that nobody reading
a page of tidy rewords mistakes the set for a clean sweep — one KR was simply missed, and it is the
one the migration was launched to satisfy.

**Grades that no reword repairs:** Q-KR2 UNAUDITABLE this cycle · Q-KR3 PARTIAL, permanently ·
E-KR1 NOT MET. Signed slates are not edited to fix any of them.
