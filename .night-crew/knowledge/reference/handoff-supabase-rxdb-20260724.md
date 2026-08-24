# Handoff — evaluate a Supabase + RxDB sync refactor

**Written:** 2026-07-24, from the night-crew development clone (`~/projects/night-crew`).
**For:** an agent working in `~/projects/hq`.
**Status:** exploration brief. Nothing is decided, no card exists, no PRD exists.

---

## What the operator asked

Three things, in one breath:

1. Refactor HQ's sync feature onto **Supabase + RxDB**.
2. Refactor the other models onto whatever pattern that establishes.
3. Make sure **future** code changes conform to it.

The question was originally put to the night-crew repo — "can the architecture help with
this?" The answer there was yes in principle, but the useful version of the answer needs
HQ's actual code. That is your job.

## What you are being asked to produce

Not a plan and not an implementation — an **assessment**, in plain language, covering:

- What HQ's sync actually is today, and what Supabase + RxDB would displace.
- Whether the per-model migration decomposes into independent work orders or has to
  serialize — this determines whether night-crew can parallelize it at all.
- What belongs in `.night-crew/knowledge/rulebook.md` so future sessions can't drift off
  the pattern (it is still a scaffolded TODO stub — see "Conformance" below).
- What the operator must decide up front vs. what an overnight run can just execute.
- Your recommendation on **timing** — see "The timing problem", which is the sharpest
  issue in this brief.

## What I already verified about HQ (do not re-derive; do verify anything you rely on)

**Stack.** Go backend under `backend/internal/` (16 packages: alerts, auth, config, db,
inventory, me, onboarding, photos, purchasing, receipt, recipes, sync, toast, users,
version, workflow). Frontend is a vanilla-JS PWA — no ES modules, `window.*` globals,
workbox service worker, page-per-app HTML at the repo root. Tests are Playwright +
playwright-bdd (`npx playwright test`); there is no JS unit-test runner in `package.json`.

**Current sync.** `sync.js` at the repo root, 694 lines, "Shared Reactive Sync Module,"
extracted from `workflows.html` in Plan 01 of Phase 10.2. It wraps `fetch` against
`/api/v1/workflow/*` with cookie credentials, redirects to `login.html` on 401, and
handles 409 as a conflict carrying a `winner`. Server side is `backend/internal/sync`,
built on an **append-only ops journal** — the `ops` table never shrinks. There is prior art
you must read: `backend/pwa-sync-architecture.pdf`.

**night-crew is already scaffolded here** — `night-crew.toml` is filled in, not a stub:
ephemeral env via `docker-compose.nc.yml`, review lenses `["adversarial", "edge-case"]`,
Playwright as the e2e suite with seam-based subsetting.

## Three constraints that will shape any answer

### 1. Every sync card runs the full e2e suite

`night-crew.toml`'s `[e2e.seams]` maps HQ's five disjoint apps to their specs, and it
deliberately leaves shared packages — `auth`, `db`, **`sync`**, `receipt` — unmapped. The
comment is explicit: touching them de-confines the card and falls back to the full suite.

So there is no subset optimization available for this work. The sync rewrite pays full
suite, and **every per-model migration card pays full suite too**, because each one touches
the shared sync seam by construction. Price that into any parallelism proposal.

### 2. night-crew's per-test gate granularity is Go-only

night-crew's gate runner executes arbitrary `sh -c`, so Playwright runs fine. But only
`go test` output is parsed down to individual test failures (`internal/gotestjson`).
`npx playwright test` collapses to a single failure atom.

That matters because the gate compares a card's failures **against the baseline failure
set**. With one atom, the comparison is all-or-nothing: night-crew cannot distinguish "your
card broke this" from "this was already red." Given the known intermittent at
`sync.spec.js:1198` (below), this is not a theoretical concern — it is aimed directly at
these cards. Flag it; a Playwright JSON parser on the night-crew side may be a prerequisite.

### 3. Supabase is a hosted service; overnight runs cannot reach one

night-crew's hard prohibitions are absolute for unattended work: no production DB, no
deploy, no push, no infrastructure provisioning. Local Docker only — which HQ already does
via `docker-compose.nc.yml`.

So any night that builds against Supabase builds against a **self-hosted, containerized**
Supabase in that compose file. Cutting over real data, and anything touching a hosted
project, is the operator's own attended act and must be scoped out of every work order
explicitly, or the cards will hit a wall at 3am.

## The timing problem — read this before recommending anything

HQ's **current cycle is about sync**. `.night-crew/knowledge/roadmap.md` opens: *"Prove &
surface — trust the sync · surface the numbers."* The evidence of active work is everywhere:

- Cards `waiver1-isolation-fix`, `sync-pkg-unit-coverage`, and `syncspec-deflake` are all
  DONE this cycle, all in the sync subsystem.
- Live branches `card/s1-syncspec-deflake` and `card/d1-syncspec-deflake`.
- A known intermittent — `sync.spec.js:1198`, "temperature answer converges" — pre-existing,
  load-sensitive, with a reproduction record at
  `.night-crew/knowledge/reference/1198-flake-reproduction-20260721.md`.
- Operator preference **P1** (STRONG): run on a clean DB with fixtures for determinism. The
  stated reason is *the append-only ops journal accumulating across runs* — a test passes at
  98 ops and fails at 614+. That preference exists because of the very design a Supabase +
  RxDB move would replace.

Two readings, and you should take a position on which is right:

- **Rewriting is the fix.** The flakes and the isolation work are symptoms of the ops-journal
  design; replacing it with a real replication protocol dissolves the class. In that case the
  current cycle's cards are stabilizing something scheduled for deletion.
- **Rewriting mid-cycle is the mistake.** The cycle's goal is *trust* in sync, and you cannot
  earn trust in a subsystem you are simultaneously replacing. In that case the correct home
  is a milestone-boundary planning round, not an interrupt.

night-crew's own promotion bar (DESIGN §15k) is deliberately asymmetric and worth applying
here: mid-roadmap promotion is for items that **block future architecture** — a planned card
cannot be built correctly until it is resolved, or would be built on a foundation needing
later rip-out. "Useful," "cleaner," and "hardens correctness" explicitly do not qualify. The
normal destiny of an item this size is the next roadmap, authored at a milestone boundary.

State which reading you hold and why. Do not split the difference.

## Conformance — what exists here, and what does not

The operator's third ask is the one with the least infrastructure in place today. Be precise
about the gap:

**Available in HQ now:**

- **`.night-crew/knowledge/rulebook.md`** — the standing rules injected into every future
  session's work-order context. It is currently a **scaffolded TODO stub**. This is the
  natural home for "never touch the Supabase client outside the sync layer; go through the
  collection helper," and the single highest-leverage artifact for ask #3.
- **`.night-crew/knowledge/preferences.md`** — real content, operator-owned, weighted not
  binding (P1–P3 on test isolation and determinism). Note the distinction: if the operator
  wants Supabase + RxDB to be *non-negotiable*, it belongs in a PRD requirement, not here —
  a preference can be deviated from with a stated justification.
- **The pre-implementation design note + adversarial lens.** Every night-crew dev session
  writes interfaces-touched / dependency-direction / layer-ownership before coding, and the
  adversarial lens (configured in `night-crew.toml`) reviews the diff against what the note
  claimed. Silent architectural drift is a finding on its own terms.

**NOT available in HQ:**

- **Machine-checkable import-direction enforcement.** night-crew's `arch-rules.toml` +
  `task lint:arch` is that repo's own tool — `night-crew init` scaffolds only
  `night-crew.toml`, `team.toml`, `knowledge/`, and `qa/`. HQ has no `arch-rules.toml` and
  no archlint. Since HQ's backend *is* Go, pointing archlint at `backend/` is plausible, but
  it would be a change to night-crew, not a configuration step here. Treat it as a proposal
  to price, not a capability to assume.

So today, conformance for HQ rests on the rulebook plus the design-note/lens pair — human-
and model-legible, not mechanically enforced. If the operator wants a hard gate, say so and
say what it would cost.

## Open questions the operator must answer (do not guess these)

- **Auth boundary.** HQ authenticates by cookie against its own Go backend. Supabase brings
  its own auth. Are these being unified, or does Supabase run as a dumb sync substrate behind
  the existing session model?
- **RLS strategy.** HQ has a real authorization surface — there are branches for grant
  enforcement parity and ops authz coverage. Does row-level security in Postgres replace that
  logic, mirror it, or sit under it?
- **Offline conflict resolution.** RxDB's replication has opinions. The current design has an
  append-only journal and a 409-with-winner convention. Which one wins, and what happens to
  the journal — is it retained as an audit record or deleted?
- **The service worker.** Workbox is already doing offline caching. RxDB brings its own
  local persistence. Two offline stories is a bug factory; one of them has to go.

## Suggested first moves

1. Read `backend/internal/sync`, `sync.js`, and `backend/pwa-sync-architecture.pdf` before
   forming any opinion.
2. Read `.night-crew/knowledge/roadmap.md` for the cycle's actual goal and the sync cards'
   state, and `1198-flake-reproduction-20260721.md` for the failure that is currently live.
3. Then answer the operator's three questions concretely, and take a position on timing.

---

*Provenance: written from the night-crew clone by inspecting HQ's tree read-only. Every claim
above was checked against a file in this repo on 2026-07-24 — but verify anything you build
on; branches move.*
