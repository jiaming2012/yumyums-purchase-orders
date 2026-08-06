# Roadmap — "Sync, dev complete" cycle (make the sync layer real, and prove it by running it)

> **Cycle:** Sync, dev complete — take the tested-but-uncalled RxDB layer the "Sync foundation"
> cycle built and make it **serve reads in the operator's dev environment**, proven by a demo
> script the operator runs personally. **Traces to:** `.night-crew/knowledge/okrs.md` (Product /
> Delivery / Engineering / QA, authored in the same sitting per DESIGN §15j.42).
> **Produced:** 2026-08-05 attended `/nc-roadmap-round`, at the milestone boundary.
> Previous cycle archived at `reference/roadmap-2026-08-05-sync-foundation.md` +
> `reference/okrs-2026-08-05-sync-foundation.md`; its close at
> `reference/cycle-closeout-20260805.md` (ledger §T-37, decisions 148–152).
>
> **Trigger — stated plainly, because it is the whole reason this cycle exists.** The previous
> milestone closed **8 MET · 1 PARTIAL · 3 NOT MET · 1 UNAUDITABLE** with its named capability
> **undelivered**: nine overnight runs and 28 landed cards produced a library with **zero
> production call sites**. `createHQSyncDatabase()` and `startHQReplication()` are imported and
> re-exported by `sync-rxdb/bootstrap.js` and never called; `window.HQSync.db` is never assigned;
> the 495 KB bundle ships precached to every crew phone and does nothing. Every close artifact —
> card count, 13 KRs, closeout — was structurally unable to see it. This is the **second** time a
> milestone has had to be created to cover a predecessor that was not dev complete.
>
> **Governing input:** `reference/handoff-hq-sync-dev-complete-20260805.md` — written from the
> night-crew clone 2026-08-05 at the operator's direction, *for this round*. Its §5 spike series,
> §6 riders and §8 re-verification block are carried into the cards below. Its load-bearing claims
> were **re-run at this round**, not taken on trust: the NUL byte in `sync-rxdb/client.js` (1
> byte, confirmed), `night-crew.toml`'s `sync` token selecting **6** spec files while claiming 4,
> and `bootstrap.js:22` still gating activation on `sync-rxdb-row-visibility-rls` — merged days
> ago.

## The operator's acceptance criterion

> *As the operator who spent over a week of attended evenings on a cycle I could not use, I want
> this milestone to end **dev complete** — the sync capability running in my dev environment,
> demonstrated by a script I run myself — so that "everything is built" can never again mean
> something I cannot actually use without authoring yet another follow-on milestone.*

**The close bar, chosen at this round:** `task demo:sync` stands up a scripted-fresh environment,
writes one field through the **real** write path (`/saveResponse`), and shows it surfacing in an
**RxDB-served read** — one real checklist, round-trip. The script exits non-zero if any leg fails,
and renders **"could not run" as an outcome distinct from "ran and failed."** 🛑 **The milestone
may not close until the operator has personally run it and seen it pass.** No KR grade, card
count or closeout substitutes for that run.

## How this roadmap works

- **Activity-level cards**, WO-sized, each carrying a module footprint and a KR trace.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Build order is load-bearing this cycle.** Activity 1 makes every later gate reading mean
  something; Activity 2 proves the path is walkable *before* any build card is cut; Activities
  3–5 only exist on legs the spikes turned green.
- 🛑 **The spike↔script rule (operator, B-345 in the night-crew clone): every spike maps
  one-to-one onto a small-to-medium script. The script IS the verdict.** "Proven" means the
  script runs green — never a prose assertion. A spike that silently no-ops is the exact defect
  class the last milestone died of.
- 🛑 **Reject any `done_when:` of the form "grep returns nothing"** until the NUL byte in
  `sync-rxdb/client.js` is fixed (Activity 1). Today that shape is satisfiable by the file being
  unreadable.

## Module footprints (independent → parallelizable)

| Footprint | Files |
|---|---|
| **gate/harness** | `verify-test-harness.sh`, `night-crew.toml`, `backend/internal/**/*_test.go` `TestMain`, `reference/gate-ladder.md` |
| **spike scripts** | `.night-crew/qa/spike-supabase/**`, `Taskfile.yml` (new `demo:`/`spike:` targets) |
| **sync client** | `sync-rxdb/*.js`, `sync-schema/*.js`, `vendor/rxdb.bundle.js` |
| **page wiring** | `workflows.html`, `sync-rxdb/bootstrap.js` |
| **backend sync** | `backend/internal/sync/**`, `backend/internal/inventory/handler.go` |

---

## Activity 0 — The floor under everything (production posture)

> **Added at morning triage 2026-08-05** from run `20260806`'s incident, on the operator's D-2
> rulings (ledger §T-38 decisions 154–155). Not part of the sync close bar, but nothing above it
> is worth proving while a test mistake can erase the business's operating data with no restore
> path — which is not hypothetical: it happened on 2026-08-06 (B-141, B-143).

- **`prod-backup-floor-and-pitr`** · **DONE** (attended, 2026-08-06 morning, branch
  `fix/b145-prod-backup-floor` merged to `dev` at `0a764a9`; ledger §T-39. `task prod:backup` —
  dump + globals, keep 14, small-dump guard — proven by a restore drill (96 tables) and an
  observed scheduled firing (`YumyumsProdBackup`, nightly 03:30); PITR base via `pg_basebackup`
  + WAL pruning per decision 159, `archive_mode` enablement attended) · Closed **B-143**
  (decision 154). Nightly
  `pg_dump` of `yumyums` to a path outside the Docker volume — a Taskfile target plus a cron
  line — then `archive_mode=on` with a local WAL archive for point-in-time recovery. The dump is
  the immediate build; either half alone would have made 2026-08-06 a twenty-minute restore.
  Footprint: `Taskfile.yml` + compose files; no app code.

- **`test-cluster-separation`** · **DONE** (run `20260807`, card W0, branch
  `card/w0-test-cluster-separation`) · Closed the **structural half of B-141** (decision 155).
  Test suites got their own Postgres container: **`yumyums-test-pg`** — `docker-compose.test.yml`,
  service `postgres-test`, compose project `yumyums-test`, host port **`5434`** (not 5433, not
  5432), named volume `yumyums-test-pgdata`, role **`hqtest`** (deliberately *not* `yumyums`, so a
  stale default fails closed with `role "hqtest" does not exist` instead of authenticating against
  production). Lifecycle: `task test:db:up` (a dependency of every `test:*` target, idempotent,
  waits healthy — measured 7.7s cold) / `task test:db:down`; `task test:targets` prints every
  resolved coordinate read-only. Re-pointed: the `task test:*` env blocks *and* the
  `backend:db-test` deps *and* `test:all`'s post-run `psql` reset trap, the RLS suite's
  `defaultHQAdminURL` + `defaultFDWPort`, `scripts/verify-test-harness.sh`'s two `DEAD_URL` arms,
  and — found at implementation, outside the slated list — `scripts/reset-e2e-db.js`, which is the
  single place Playwright's coordinates are computed and which issues the `DROP DATABASE`.
  `backend/Taskfile.yml`'s `ALLOW_TEST_DB_ON_DEV_HOST` guard was armed (default `1` → `0`) now that
  the cluster its comment was waiting for exists. Proven by the full Go suite (9 packages, 246
  top-level / 456 incl. subtests, 0 failures) and the full Playwright suite run against the new
  container. Production topology untouched. **B-141's prefix-guard half and B-142 remain open on
  the attended `gate-rls-fixture-ownership` re-gate.** Footprint: gate/harness + compose files.

---

## Activity 1 — A green that means something (gate integrity)

> **Why this is first, and not sync.** The close bar is *"the operator ran it and saw it pass."*
> That verdict is only worth what the gate underneath it is worth — and today a gate can print
> `ok` having run nothing. Last cycle's A1 (`e2e-gate-database-isolation`) was exactly this shape
> and it is the reason the close could cite anything at all. **Trace:** QA objective.

- **`gate-rls-count-assertion`** · **DONE** (run `20260806`, merge `9b63958`; triaged to `dev` 2026-08-05) · Closed **B-36**.
  🛑 **Re-scoped at slate-20260806 on execution evidence.** B-36's mechanism is **already fixed**:
  commit `4615661` (2026-08-01, on `dev`) made an unresolvable substrate a hard `t.Fatalf` unless
  `HQ_SYNC_SUBSTRATE_OPTIONAL` is set. Re-probed at slate time in both directions — docker stripped
  from `PATH` ⇒ `EXIT=1`; opt-out set ⇒ `EXIT=0`. The bug was filed 07-31, fixed 08-01, and its
  backlog entry was never closed, so this round promoted a fixed defect (an instance of **B-38**'s
  channel gap). **The card's surviving half is the count assertion** — Q-KR1 requires the 59
  subtests *asserted rather than inferred*, and `grep` finds no count assertion anywhere in the
  package. Also pins the exit-code asymmetry as a test, and closes B-36's stale entry.
  Footprint: backend sync.

- **`gate-harness-check-b-per-package`** · **DONE** (run `20260806`, merge `b75ac53`; triaged to `dev` 2026-08-05) · Closed **B-22**; B-144 filed on its cost honesty at triage.
  🛑 **Split out of `gate-harness-honesty` at slate-20260806 per the §1.4 fan-out rule** — the
  original card bundled two mechanisms in two file families (shell vs Go). `scripts/verify-test-harness.sh`
  Check B runs **one** aggregate `go test` over seven packages and passes when it exits non-zero, so
  six of seven can report `ok` on a dropped database and the gate still prints PASS. Make it
  per-package, require all seven to exit non-zero, and assert the iterated package count so a
  shrinking `DB_PKGS` announces itself. Footprint: gate/harness.

- **`gate-rls-fixture-ownership`** · **BLOCKED** (run `20260806`: G6 verdict **DO NOT MERGE**; branch + worktree preserved at `card/a3-rls-fixture-own`) · Still closes **B-35**. 🛑 **Attended re-gate required** — fix B-141's prefix guard and B-142's two items as one card, then re-run the gates (ledger §T-38 decision 155; its G6 probe destroyed production, see B-141/B-143). The mechanism and evidence are otherwise the strongest of run `20260806`.
  🛑 **Split out of `gate-harness-honesty`** (see above). The standard gate command `go test ./...`
  **drops a database it does not own** — `rowvisibility_rls_test.go:400` drops/recreates
  `hq_test_b2_fdw` on entry, so any plain gate run destroys a concurrently-running card's fixture.
  B-16's failure mode baked into the primary gate command, on a project whose normal shape is
  concurrent dispatch. 🛑 **Narrower than first written:** the `HQ_RLS_TEST_DB` override *already
  exists* (`:233`); the defect is the shared **default**. Remedy per B-35's lead — prefer failing
  over defaulting. Footprint: backend sync.

- **`gate-ladder-completeness`** · **DONE** (run `20260806`, merge `c2a7e5c`; triaged to `dev` 2026-08-05; its D-1 fork ruled — red-first is now gate **RF**, ledger §T-38 decision 153) · Closed the surviving half of **B-26**, plus
  **B-14**. Decision 138 gave the ladder a home (`reference/gate-ladder.md`) but **G5 is still
  undefined** — the table runs G1, G2(Go), G2(Playwright), G3 (N/A), G4, **G6**. Either define G5
  or state in the file that there is none, so no future slate inherits a gap. And **B-14**: the
  morning-triage G4 discipline greps are **vacuous in hq and read as clean** — the same
  absence-reads-as-pass class this whole activity exists to retire. Footprint: gate/harness.

- **`shipped-bug-sweep`** · **PLANNED** (was slated on run `20260806` as the budget-gated stretch; **cut at 21:40Z, never dispatched** — re-slate it) · Close **B-89** and **B-132** — routed to "the next
  night" by T-34 decision 137 and never promoted to a card, which is exactly the channel gap
  **B-38** describes. (a) `cachedGrantSlugs()` returns `[]` unconditionally on every real client
  (`index.html` writes `hq_apps` as `{uid, apps}`; `bootstrap.js` `Array.isArray`-gates it) — 🛑
  **latent only while nothing calls `startHQReplication`, which Activity 4 changes**, so this must
  land before activation. (b) `workflows.html:708` throws an uncaught `IndexSizeError` on every
  completed submission, orphaning a full-screen canvas; fires 28× per suite run. Footprint: page
  wiring + sync client.

- **`repo-hygiene-preconditions`** · **DONE** (run `20260806`, merge `6f91863`; triaged to `dev` 2026-08-05; B-140 filed for the four residual stale-gate sites) · The three defects the handoff §6 re-verified,
  each one line, all blocking clean `done_when:` authoring downstream. (a) **One NUL byte in
  `sync-rxdb/client.js`** makes `grep` report nothing on strings present three times — any
  "grep returns nothing" criterion is currently satisfiable by unreadability. (b)
  **`night-crew.toml`'s `sync` token selects 6 spec files while claiming 4** — a false claim in
  the file that decides which tests a card must run. (c) **`bootstrap.js:22` still gates
  activation "until `sync-rxdb-row-visibility-rls` lands"** — that card merged; the banner is
  stale in the very file Activity 3 edits. Footprint: sync client + gate/harness.

---

## Activity 2 — Prove the route before the nights spend it (spikes A–D)

> 🛑 **No build card in Activities 3–5 may be cut until this activity's spikes are green.** The
> premise that killed `sync-hard-cutover` (decision 126 — "rows flow back from the substrate; no
> API boundary left to translate at") was falsifiable by a script on day one and was instead
> measured false on **night nine of nine**, after ~11,200 spec lines had been built on it.
> **Trace:** Delivery + Engineering objectives.

- **`spike-a-environment-up`** · **DONE** (run `20260806`, merge `76dc12b`, verdict **GREEN**; triaged to `dev` 2026-08-05 — D-KR1 now has 1 of its 4 spike verdicts) · *Spike A — the operator's own.* One script takes a
  clean machine to *"Supabase + RxDB both up, schema applied, healthy"* **unattended**. Proves the
  environment has no hand-configured, undocumented step. Seeds the dev-environment target that
  Activity 5's demo runs against. Verdict = the script's exit status. Footprint: spike scripts.

- **`spike-b-migration-rehearsal`** · **PLANNED** · *Spike B — the operator's own.* Create one
  Postgres whose schema **mimics HQ's with a small subset of fields**, add a data fixture, stand
  up fresh Supabase + RxDB instances, and **migrate the fixtured data across**. Proves HQ-shaped
  data actually lands in the substrate and surfaces in RxDB. This is the leg nine nights were
  built on top of without ever testing. Footprint: spike scripts.

- **`spike-c-round-trip`** · **PLANNED** · 🛑 **LOAD-BEARING — if this cannot go green, STOP and
  re-plan before any card is cut.** One row written through the **real** write path
  (`/saveResponse`) must appear in an **RxDB-served read** within bounded time. This is precisely
  the premise decision 126 measured false: RxDB replicates from a *second, different* Postgres
  (the Supabase substrate) and **nothing carries a row from the substrate back into HQ's
  Postgres** — the FDW bridge is one-directional and carries *permissions, not data*. The spike's
  job is to establish whether the HQ-Postgres → substrate → RxDB-read path exists **at all**, and
  by what mechanism. A red here is a **successful spike**, not a failed card. Footprint: spike
  scripts + backend sync.

- **`spike-d-realtime-live`** · **PLANNED** · Close **B-62**. The Realtime `filter` is proved at
  the **config**, never against a live server — every existing test injects a fake. Drive the
  replication filter against real infrastructure. 🛑 `HQ_SYNC_REST_URL` being unset is **the
  interlock working, not evidence of correctness**. Footprint: spike scripts + sync client.

---

## Activity 3 — The walking skeleton (one row, end to end, behind a flag)

> Only the legs Activity 2 proved. The skeleton exists from here on and **grows into** the demo
> script rather than being authored at the end. **Trace:** Product + Engineering objectives.

- **`skeleton-one-row-end-to-end`** · **PLANNED** · Thread **one** checklist row from the real
  write path to an RxDB-served read in dev, behind an explicit flag, with the two list views and
  the fill view all still on REST. The first production call site of `createHQSyncDatabase()` and
  `startHQReplication()` in the repo's history. Must carry decision 126's shape verbatim (reads
  on RxDB, `/saveResponse` and `/submit` keep owning writes) and cite it, per the standing rule
  that a build WO may not propose the split itself. Footprint: page wiring + sync client.

- **`skeleton-offline-ownership-honesty`** · **PLANNED** · Close **B-88**. The rule *"nothing may
  read from RxDB on a code path that can execute offline"* is enforced by three
  `expect(src).not.toContain(…)` assertions over **source text** — and `workflows.html:3590` reads
  `window.HQSync.db`, **a fourth route the guard does not name**. It is green today only because
  the database does not exist; **the first card that creates one breaks the rule with no diff to
  anything the guard watches**. That card is the one directly above. Assert on the **object**, not
  the spelling of identifiers. Footprint: page wiring.

---

## Activity 4 — Activate the read path (decision 126's shape)

> **Trace:** Product objective. Carries the riders the retired card left behind: **B-63, B-64,
> B-66–B-69, B-79**.

- **`activate-fill-view-reads`** · **PLANNED** · The checklist fill view reads from RxDB, scoped
  per-open-checklist (T-29 decision 105 — replication scope is **never** all collections at once,
  and no card may widen it without a recorded decision). Footprint: page wiring + sync client.

- **`activate-list-views-or-state-they-stay-rest`** · **PLANNED** · Resolve **B-43**, which has
  never been decided: `scope.checklistId` is **mandatory and singular**, but the page a crew
  member lands on is **My Checklists** — a list over *many* submissions — and Approvals is a
  second. Either those two views stay on REST (the cutover is partial **by design**, stated) or
  someone records a C-2 widening. 🛑 **Decide it; do not discover it while wiring.** Carries
  **B-63**: two concurrent replications over the same four local collections, where
  `client.js`'s standing *"CANCEL BEFORE RE-SCOPING"* banner would break the live list if followed
  literally. Footprint: page wiring + sync client.

---

## Activity 5 — Dev complete (the close bar)

> **Trace:** Delivery objective. This activity is the milestone's definition of done.

- **`demo-sync-target`** · **PLANNED** · Ship `task demo:sync` as a **first-class deliverable**:
  scripted-fresh environment (from Spike A), one field written through `/saveResponse`, surfacing
  in an RxDB-served read, on one real checklist. Non-zero exit on any failed leg. 🛑 **"Could not
  run" must render as an outcome distinct from "ran and failed"** — a demo that silently no-ops
  would reproduce the exact class this milestone exists to retire. Footprint: spike scripts +
  page wiring.

- **`dev-complete-attestation`** · **PLANNED** · 🛑 **Attended, and the operator's own act.** The
  operator runs `task demo:sync` in dev and records the outcome in `ledger.md`. **The milestone
  does not close without this line.** No card, grade or closeout substitutes for it.

---

## Backlog dispositions this round

**Walked:** the five groups below. **Not walked:** the remainder of the backlog, untouched and
still `new` — see the count note in the closeout. Group labels were this round's scaffolding and
are deliberately **not** written into `BACKLOG.md`; only each entry's status is.

| Group | Handles | Disposition |
|---|---|---|
| Gate integrity | B-22, B-35, B-36, B-14, B-26 | **promoted** → Activity 1 |
| Live shipped bugs | B-89, B-132 | **promoted** → `shipped-bug-sweep` |
| Handoff §6 preconditions | (NUL byte, toml count, stale banner) | **promoted** → `repo-hygiene-preconditions` |
| Sync activation riders | B-62, B-43, B-63, B-88 | **promoted** → Activities 2–4 |
| Observability cluster | B-139, B-81, B-82, B-86, B-93 | **left `new`** — walked, and waits. Same class as this cycle's subject and a strong candidate next round; kept off the critical path to dev-complete at the operator's direction |

**Deliberately left `new` and named out loud** (not silently dropped): **B-131** and the
load-sensitive flake family (**B-27**, **B-30**, **B-32**) stay armed — an armed red is retired by
diagnosis, never by passing once (decision 100; T-31 decision 120). **B-33**, **B-77**, **B-133**
and **B-12** are night-crew tooling defects, now filed clone-side as **B-346**/**B-347** and
tracked there. **E-KR1's two un-dropped fetch-storm items** carry forward: `sync.js` is still in
the tree with both mechanisms live at `:443-454` and `:475-479`, and this cycle does not remove
it — Activity 4 narrows what RxDB serves, it does not retire the old path.
