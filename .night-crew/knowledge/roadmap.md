# Roadmap — "Sync foundation" cycle (replace the op-log/WebSocket sync layer with RxDB + self-hosted Supabase)

> **Cycle:** Sync foundation — replace `workflows.html`'s hand-rolled WebSocket + Postgres
> LISTEN/NOTIFY + Lamport-clock op-log sync layer with RxDB (client-side replication) against a
> self-hosted Supabase stack (Realtime + PostgREST, alongside the existing Postgres on the
> Windows box), unifying the field-autosave (`/saveResponse`) and live-broadcast (`sync.js`)
> write paths into one store. Root cause: a recurring fetch-storm/stale-hydration failure class
> diagnosed across the just-closed "Prove & surface" cycle (T-18 fetch-storm bug class, the
> retired-but-recurring `sync.spec.js` flakiness, cross-user hydration divergence) — the
> mechanism RxDB's checkpoint-based replication is built not to have.
> **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA,
> authored alongside this roadmap). **Produced:** 2026-07-24/25 attended `/nc-roadmap-round`,
> scoped to this activity at the operator's direction. Previous cycle archived at
> `reference/roadmap-2026-07-24-prove-and-surface.md`.
> **Trigger:** BACKLOG.md entry "`workflows.html` sync: migrate to RxDB + self-hosted Supabase"
> (operator explore session 2026-07-24, decisions on conflict resolution/scope/auth-bridge/
> cutover recorded there) plus the untracked handoff brief
> `reference/handoff-supabase-rxdb-20260724.md` (written from the night-crew development clone,
> asking for exactly this assessment — its "milestone boundary is the correct home" reading is
> now the actual state, since the milestone closed the same day).
> **Scope note:** this round walked ONE backlog group (sync fragility) at the operator's
> request. ~19 other `· new` backlog items across security/infra hygiene, grants follow-ups,
> test/run-mechanics hygiene, product/display nuance, and money-precision were proposed as
> groups but **not walked this round** — they remain `new` in BACKLOG.md, available for the next
> roadmap-round pass (either continuing this same round or a future one), not dropped.

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan` sizes to a
  night. Cards carry a **module footprint** (for parallel tracks) and a **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** Cards-per-night is the planner's call against
  the night budget + quality bar (budget is a floor, not a ceiling).
- **Build order within Activity 1.** The feasibility spike gates everything else — self-hosted
  Supabase + the JWT bridge must be proven before schema/replication and the auth endpoint are
  sized. Schema+replication and the JWT-bridge endpoint have disjoint footprints (frontend RxDB
  vs. backend auth) and may run in parallel once the spike clears. Hard cutover is serialized
  last — it depends on both.
  **Fan-out 2026-07-25 (`/nc-slate-plan`, §1 split rule):** the feasibility spike was split into
  **two** cards before slating, because the operator directed it to (a) exercise RxDB itself, not
  only the stack beneath it, and (b) leave a runbook the operator can run by hand. Those are two
  mechanisms of normal-card size — Docker/JWT infra and an RxDB replication client — with
  different failure modes and park triggers, so the card is `sync-spike-stack-and-jwt-bridge` +
  `sync-spike-rxdb-replication`. **The cycle's gate is the stack card alone**; the RxDB card
  deepens the verdict and de-risks `sync-rxdb-schema-and-replication`, but blocks nothing.
  Activity 1 therefore holds **6** cards, not 5 — the Delivery per-card-timing KR denominator
  moves with it.
- **Red-first is mandatory on every fix card.**
- **Per-card wall-clock timing is a standing output** on every build card, continuing the
  "Prove & surface" cycle's practice (T-14 baseline N=23 / 22m28s; last-measured median 94m
  N=12, population shift noted at that cycle's close).
- **night-crew CI constraint (from the handoff brief, verify before slating):** this repo's
  `[e2e.seams]` config leaves `sync` unmapped for subset optimization — touching it de-confines
  a card to the full Playwright suite. Every card in this activity touches the sync seam by
  construction, so every card here pays full-suite, not a subset. Additionally, night-crew's
  per-test gate granularity is Go-only; Playwright collapses to a single failure atom, so the
  gate cannot distinguish "this card broke it" from "already red" — price this into slating,
  especially given the known `sync.spec.js` load-sensitivity history.
- **Self-hosted only for unattended work.** night-crew's overnight runs prohibit production DB/
  deploy/push/infrastructure provisioning. Every card here builds against the self-hosted
  Supabase in the repo's own Docker compose — never a hosted Supabase project. Cutting over real
  prod data is the operator's own attended act, scoped out of every work order explicitly.

## Module footprints (independent → parallelizable)

| Track | Frontend | Backend | Tests |
|---|---|---|---|
| Sync rewrite | `sync.js`, `workflows.html`, new RxDB client layer | `backend/internal/sync` (retired by cutover), new self-hosted Supabase compose service, new JWT-bridge endpoint | `sync.spec.js` and successors, new RxDB replication tests |
| Independent fix | — | `backend/internal/workflow` (`checklist_submissions.status` default) | `workflow` package tests |

---

## Activity 1 — Sync foundation: RxDB + self-hosted Supabase

- **`sync-spike-stack-and-jwt-bridge`** · **DONE — verdict GO** (2026-07-25, run
  `overnight-20260725`, merged `51d0c02`; G6 PASS-WITH-FINDINGS, all non-blocking). Self-hosted
  Supabase (postgres + postgrest + realtime; **Kong/Studio/GoTrue proved unnecessary**) accepts a
  stdlib-only Go-minted HS256 token on both PostgREST and Realtime, with RLS **demonstrably
  discriminating** — verified twice, once by the card and once independently by G6 against the
  live stack, with a `service_role` BYPASSRLS control ruling out the empty-table explanation.
  Verdict at `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`; runbook half 1 at
  `.night-crew/qa/spike-supabase/README.md`; stack left running deliberately.
  🛑 **The verdict still has to reach `ledger.md` at morning triage before the three downstream
  cards may dispatch** — Product KR1 / Delivery KR1 measure the ledger timestamp, and the ledger is
  an attended artifact this run cannot write. · *(½ of the fanned-out
  `sync-rxdb-feasibility-spike` — the cycle's Wave-0 gate)* · Stand up self-hosted Supabase
  (Realtime + PostgREST, via Docker) in a **new, separate `docker-compose.supabase.yml`** —
  never by extending `docker-compose.nc.yml`, which would boot Supabase for every night-crew run
  in this repo. Prove the Go backend can mint its own HS256 JWTs (`role: authenticated`, `sub`,
  `exp`, signed with Supabase's configured `JWT_SECRET`) using **stdlib only** (`crypto/hmac`;
  no new module dependency) and have self-hosted PostgREST/Realtime accept them for RLS —
  without GoTrue/Supabase Auth. The PostgREST proof must show the policy **discriminating** (an
  authorized read succeeds *and* an unauthorized one is refused); a 200 alone proves nothing.
  Realtime is proven over `github.com/coder/websocket`, already a direct dependency; self-hosted
  Realtime needs a **tenant row**, expected to be the sharpest edge. Confirm the
  self-hosted-specific table contract: text PK, `_deleted` boolean, `_modified` trigger, RLS
  enabled, and manual `ALTER PUBLICATION supabase_realtime ADD TABLE` (no dashboard toggle in
  self-hosted) — note the per-table cost, which is what sizes the schema card. Output: a written
  **GO or NO-GO** at `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` (a NO-GO is a
  completed card, not a failed one) plus **half 1 of the operator runbook**. Footprint: new
  compose file + `.night-crew/qa/spike-supabase/**`; **no production code, and `go.mod` / root
  `package.json` / `docker-compose.nc.yml` / `Taskfile.yml` HARD-untouched.** *(from BACKLOG
  "`workflows.html` sync: migrate to RxDB + self-hosted Supabase")*

- **`sync-spike-rxdb-replication`** · **PLANNED** (depends on the stack card's GO) · *(½ of the
  fanned-out `sync-rxdb-feasibility-spike`; operator ask 2026-07-25 — the spike must exercise
  RxDB itself, and leave something runnable)* · Drive an actual RxDB collection against the
  stack card's Supabase, from an **isolated Node harness** at `.night-crew/qa/spike-supabase/rxdb/`
  with its **own** `package.json` (the repo-root one is the Playwright environment for every card
  and stays HARD-untouched). Prove replication in **both directions separately** — local RxDB
  write visible in Postgres via PostgREST, and a direct Postgres write converging into RxDB
  without a client restart; a one-directional proof is how this class of spike fools itself.
  **Observe last-write-wins rather than assuming it**: construct one concurrent-write case and
  record what actually happens, including which clock decides and whether a write is silently
  lost — a divergence from the assumed LWW is a finding to route, not something to correct in
  code. **Answer two go/no-go inputs:** (a) which RxDB storage the real PWA would use in a browser
  and whether it is free or premium-licensed under RxDB's current terms (verify against the
  license page; a paid dependency is a cost the operator must know before four cards are built on
  it), and (b) whether Supabase replication is a supported plugin or a documented example built on
  `replicateRxCollection` that we would maintain ourselves. Uses **one throwaway table, not the
  real checklist domain** — modelling that is the schema card's job. States plainly what a
  Node-side proof does **not** establish (browser storage, service-worker interaction, PWA offline
  semantics). Output: **half 2 of the operator runbook** (append-only) + the RxDB half of the
  verdict, sizing `sync-rxdb-schema-and-replication`. Footprint:
  `.night-crew/qa/spike-supabase/rxdb/**` only.

- **`sync-rxdb-schema-and-replication`** · **PLANNED** (depends on `sync-spike-stack-and-jwt-bridge`'s
  go-decision reaching `ledger.md`; sized by `sync-spike-rxdb-replication` where it ran) · Define
  RxDB collections for checklists, templates, responses, and approvals (mirroring the current
  Postgres domain model), each satisfying the self-hosted table contract above. Wire RxDB's
  Supabase replication plugin client-side. Last-write-wins conflict resolution (per the explore
  session: checklist edit conflicts are rare, no custom conflict handler). Footprint:
  `workflows.html`, new RxDB client layer.

- **`sync-jwt-bridge-endpoint`** · **PLANNED** (depends on `sync-spike-stack-and-jwt-bridge`'s
  go-decision reaching `ledger.md`; disjoint footprint
  from the schema card, may build in parallel) · Go backend endpoint that mints the
  Supabase-compatible JWT from the existing session/bearer-token auth and grant data, bridging
  existing permissions into the `role`/`sub` claims Supabase's RLS policies read — no adoption
  of Supabase Auth/GoTrue. Footprint: `backend/internal/auth` (or a new package), `backend/internal/sync`.

- **`sync-hard-cutover`** · **PLANNED** (depends on schema+replication AND jwt-bridge) · Replace
  BOTH current write paths in `workflows.html` — `autoSaveField`→`POST /saveResponse` and
  `sync.js`'s WebSocket/ops-log broadcast — with the RxDB store as the single write path. Retire
  `sync.js`, `backend/internal/sync/`, and `/saveResponse` entirely. Hard swap, no parallel run
  (per the explore session — no need to keep the old system live during cutover). Reconcile the
  existing Workbox service-worker offline caching against RxDB's own local persistence so there
  is exactly one offline story, not two (Workbox keeps owning static-asset caching; RxDB owns
  data). Footprint: `workflows.html`, `sync.js` (deleted), `backend/internal/sync` (deleted),
  `backend/internal/workflow` (`/saveResponse` removed).

- **`workflow-submission-status-default`** · **PLANNED — server half merged, CLIENT HALF REQUIRED
  AND MISSING** (2026-07-25, run `overnight-20260725`). Server fix merged at `53e921d` and its Go
  gates are green; the seam-confined subset leg was also green (102 passed / 1 skipped / 6 m 18 s).
  **But the subset was the wrong suite:** the full suite reds `tests/repro-cut-task.spec.js:153`
  and `tests/sync.spec.js:1581`, both proven by measurement to be an F1 regression (pass on `dev`,
  fail with F1). `workflows.html` does not recognise the new `'completed'` status, so
  `.submit-confirm` never renders. **Parked as a contract question — F1's own park trigger (ii)
  — see `runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` FORK 1.** Stays PLANNED per the run rule
  that a card parking without a verdict does not flip. · (independent footprint, no dependency
  on the sync cards) · `checklist_submissions.status` defaults to `'pending'` and
  `submitChecklist` never updates it for `requires_approval:false` submissions, so no-approval
  submissions read `'pending'` server-side forever. Harmless today (UI derives status from other
  fields) but a trap for any future server-side status consumer. Normalize on submit or document
  the invariant explicitly; red-first regression test. Footprint: `backend/internal/workflow`.
  *(from BACKLOG "`checklist_submissions.status` never set for `requires_approval:false`
  submissions")*

---

## Backlog dispositions this round

| Backlog item (`· new`) | Disposition |
|---|---|
| `workflows.html` sync: migrate to RxDB + self-hosted Supabase | promoted → ~~`sync-rxdb-feasibility-spike`~~ → **`sync-spike-stack-and-jwt-bridge` + `sync-spike-rxdb-replication`** (fanned out 2026-07-25 at slating), `sync-rxdb-schema-and-replication`, `sync-jwt-bridge-endpoint`, `sync-hard-cutover` (Activity 1) |
| `checklist_submissions.status` never set for `requires_approval:false` submissions | promoted → `workflow-submission-status-default` (Activity 1) |
| Replay fetch-storm class is NOT fully closed | dropped — superseded by the RxDB/Supabase migration (symptom of the mechanism being replaced) |
| `sync.js` catch-up fetch-storm gate | dropped — superseded by the RxDB/Supabase migration |
| Rejected-field hydrate quirk: new answer visually clears on reload until resubmission | dropped — superseded by the RxDB/Supabase migration |
| Cross-user checklist hydration divergence (approved-vs-rejected ghost state) | left `new` — needs a product ruling on desired cross-user semantics; routed to the next `/nc-pm-session` intake, not resolved this round |

All other `· new` backlog items (security/infra hygiene, grants follow-ups, test/run-mechanics
hygiene, product/display nuance, money-precision) were grouped for this round's walk but not
yet walked — left untouched, still `new`.
