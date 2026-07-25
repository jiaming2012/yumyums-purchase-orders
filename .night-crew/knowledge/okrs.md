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
- Prod cutover parity: once `sync-hard-cutover` ships, `task version` shows prod backend/frontend == local `version.go` constants with 0 drift, AND 2/2 tab screenshots are verified on a **returning** client, not just a fresh load — the T-21d lesson applies here by name. Measured by: `task version` command output + operator-provided screenshots recorded in the ledger (same artifact type that settled D2 last cycle, T-21e).
- Per-card wall-clock timing is recorded for all 5 cards in this activity (4 sync + 1 independent fix), and a median is computed against the prior cycle's baseline (N=12 / 94m). Measured by: the timing field in each card's WO record under `.night-crew/runs/` (the `percard-timing-instrumentation` standing output from the prior cycle).

## Engineering

### Objective: The new sync layer is provably correct where the old one wasn't — no fetch-storm class, no stale hydration, and the JWT/RLS bridge actually bounds data the way the bearer-token auth did.
- 0 of the 2 superseded fetch-storm-class backlog items (replay-fetch-storm, `sync.js` catch-up gate) reproduce against the new architecture — evidenced by exactly 1 regression test per item in `tests/sync.spec.js` (or its cutover-era successor) asserting no ungated re-fetch storm on catch-up, OR, where no such test is constructible, exactly 1 reviewed architectural-argument note per item in `.night-crew/knowledge/designs/`.
- 0 attack variants bypass RLS under a JWT-bridge attack-variant suite comparable in rigor to `grant-enforcement-parity`'s 13 variants (invalid `role` claim, expired token, missing `sub`, wrong signature, token replay after grant revocation) — measured by a new Go test file under `backend/internal/sync` (or the auth package hosting the JWT-bridge endpoint), modeled on `tests/grant-enforcement-parity.spec.js`'s attack-variant structure.
- Exactly 1 documented owner exists per offline data class after cutover (static assets → Workbox, checklist data → RxDB) — 0 classes with dual or ambiguous ownership — measured by 1 design note in `.night-crew/knowledge/designs/`, reviewed at the cycle gate, cross-checked against the `build-sw.js`/RxDB-init diff.

## QA

### Objective: This migration does not repeat "Prove & surface"'s QA gap — a sync-adjacent package shipping with zero tests until an escape forced the issue — and night-crew's CI constraints on this activity are priced in up front, not discovered at 3am.
- New code carries real coverage from its first WO, not retrofitted later: the RxDB replication layer and the JWT-bridge endpoint each ship with unit/integration tests in the SAME WO that introduces them (the `sync` package's 0-Go-tests gap from last cycle does not recur in its replacement). Measured by: each card's WO record in `.night-crew/runs/` and its merged diff showing test files alongside implementation files, not in a later card.
- 100% of this activity's WOs carry red-run evidence in the WO record (red-first: the test fails before the fix/feature lands). Measured by: the "red-first" evidence field in each `.night-crew/runs/` WO record, same convention as the prior cycle's QA2.
- Every WO for this activity states its expected runtime under the full Playwright suite (the `sync` seam is unmapped for subset optimization, so every card here pays full-suite) and explicitly flags `sync.spec.js` load-sensitivity risk before dispatch. Measured by: the slate document (`reference/slate-<runid>.md`) for each card carrying an explicit full-suite-runtime + load-sensitivity note.
