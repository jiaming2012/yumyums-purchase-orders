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

- **`shipped-bug-sweep`** · **DONE** (run `20260807`, card **A2**, branch `card/a2-shipped-bug-sweep`; commits `08dad2d` (B-89), `e65deb6` (B-132), `f2aeb0c` (sw.js regen)) · Closed **B-89** and **B-132** —
  routed to "the next night" by T-34 decision 137 and never promoted to a card, which is exactly the
  channel gap **B-38** describes. (a) `cachedGrantSlugs()` returned `[]` unconditionally on every real
  client (`index.html` writes `hq_apps` as `{uid, apps}`; `bootstrap.js` `Array.isArray`-gated it) —
  now reads the envelope and verifies `uid` against the identity token, mirroring `index.html`'s own
  uid-mismatch handling (not a PARK). Landed before Activity 4 arms `startHQReplication`. (b)
  `workflows.html`'s `fireworks()` threw an uncaught `IndexSizeError` on every completed submission
  (measured tonight at 2482× per suite run, not the stale 28×/run figure) — radius now clamped
  (`Math.max(0, p.size*p.life)`). Screenshot evidence settled what was never established: the "frozen
  overlay" renders fully transparent (not a visible frozen confetti burst) — the real defect was a
  leaked, invisible `<canvas>` DOM node. Both RF'd red-first; see BACKLOG.md B-89/B-132 for full
  evidence + log paths. Footprint: page wiring + sync client, exactly as declared.

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

- **`spike-b-migration-rehearsal`** · **DONE** (run `20260807`, card S, verdict **GREEN** — exit 0
  from `.night-crew/qa/spike-supabase/spike-b-migration.sh`; D-KR1 now has 2 of its 4 spike
  verdicts) · *Spike B — the operator's own.* Create one Postgres whose schema **mimics HQ's with a
  small subset of fields**, add a data fixture, stand up fresh Supabase + RxDB instances, and
  **migrate the fixtured data across**. Proves HQ-shaped data actually lands in the substrate and
  surfaces in RxDB. This is the leg nine nights were built on top of without ever testing.
  Footprint: spike scripts.
  **Verdict delivered:** an 8-table subset of HQ's real schema (transcribed from migrations
  0001/0004/0005/0006/0009/0010/0011/0012) on a fresh scratch Postgres, 6 of 7 fixtured submissions
  transformed and loaded **through PostgREST** into the substrate, byte-identical to source with
  HQ's `uuid` keys intact through the cast into the text-keyed sync contract; RLS discriminated over
  **those migrated rows** on both axes; three RxDB clients each replicated **exactly** the migrated
  rows they were entitled to (2 / 2 / 1) and none received the nobody-visible control row. 48 named
  assertions, ~25 s end to end, re-runnable from nothing (scratch container created and destroyed
  each run; spike A's stack consumed in reconcile mode, never destroyed).
  **Two findings for the cutover card, neither of which blocks it:** (1) HQ stores **no
  template→app association**, so there is nothing to populate the sync contract's `app_slug` from —
  the spike added the column explicitly and labelled it a deviation rather than hardcode
  `'operations'`; where that association should live is an open question the cutover card inherits.
  (2) A bulk migration **cannot** run on per-user tokens: `hq_sync_checklists_insert`'s `WITH CHECK`
  refuses a row whose owner holds no live grant on its app, and real datasets contain exactly such
  rows. The bulk lane must be a service identity — measured viable on this stack with no schema
  change (`service_role` already has `rolbypassrls=t` and full table grants from the
  supabase/postgres image's default privileges). The `authenticated` user lane was rehearsed
  separately after the load and still refuses correctly (HTTP 403, row genuinely absent).

- **`spike-c-round-trip`** · **DONE** (run `20260807-2`, card C, verdict **GREEN** — exit 0 from
  `.night-crew/qa/spike-supabase/spike-c-roundtrip.sh`, round trip 248 ms against a 20 000 ms
  bound; mechanism proven: LISTEN/NOTIFY relay → PostgREST service-identity write → RxDB
  Realtime pull; D-KR1 now has 3 of its 4 spike verdicts) · 🛑 **LOAD-BEARING — if this cannot go green, STOP and
  re-plan before any card is cut.** One row written through the **real** write path
  (`/saveResponse`) must appear in an **RxDB-served read** within bounded time. This is precisely
  the premise decision 126 measured false: RxDB replicates from a *second, different* Postgres
  (the Supabase substrate) and **nothing carries a row from the substrate back into HQ's
  Postgres** — the FDW bridge is one-directional and carries *permissions, not data*. The spike's
  job is to establish whether the HQ-Postgres → substrate → RxDB-read path exists **at all**, and
  by what mechanism. A red here is a **successful spike**, not a failed card. Footprint: spike
  scripts + backend sync.

- **`spike-d-realtime-live`** · **DONE** (run `20260807-2`, card D, verdict **GREEN** — exit 0
  from `.night-crew/qa/spike-supabase/spike-d-realtime.sh`; the live Realtime server honours the
  filter in all three clause shapes production emits, suppression proven attributable via
  same-socket unfiltered control; all 4 D-KR1 spike verdicts now recorded, B-62 answered) ·
  Close **B-62**. The Realtime `filter` is proved at
  the **config**, never against a live server — every existing test injects a fake. Drive the
  replication filter against real infrastructure. 🛑 `HQ_SYNC_REST_URL` being unset is **the
  interlock working, not evidence of correctness**. Footprint: spike scripts + sync client.

- **`spike-e-reconnect-catchup`** · **DONE** (run `20260808`, card E, verdict **GREEN** — exit 0
  from `.night-crew/qa/spike-supabase/spike-e-reconnect.sh`; a severed RxDB client recovered
  **all three** dark-window changes on reconnect via checkpoint pull in 1 ms of a 20 s bound,
  **including the mandatory UPDATE to a row it already held**, corroborated by the substrate
  primary key staying the same and field B holding exactly 1 draft row in HQ's Postgres; the
  first post-reconnect pull was observed resuming FROM the sever-time checkpoint, not doing a
  full re-read; red-first `--no-pull` exit 1 missed all three with the liveness control still
  arriving; teardown VERIFIED byte-identical on both paths) · **B-161 answered.** Carried
  finding: `checklist_submissions.submitted_at` never advances after INSERT (0 user triggers;
  approve/reject set `status`/`reviewed_by`/`reviewed_at` only) while
  `submission_responses.answered_at` does — but the pull checkpoints on **neither**, it
  resumes on the substrate's trigger-stamped `_modified` with a strict `gt` + id tie-breaker.
  That independence is *why* the UPDATE recovered, so the green is conditional on the carrier
  re-projecting on every change: a future relay that polls HQ on a business watermark instead
  of NOTIFY reintroduces the miss exactly. · Prove the disconnect/reconnect/catch-up cycle no
  existing spike touches: C proved
  the round trip and D proved the filter, but nothing has ever severed a replicating client,
  written rows while it was dark (including an UPDATE to an existing row — the
  `submitted_at=gte.<iso>` checkpoint's weak spot), reconnected it, and measured whether
  checkpoint pull recovers everything. Reuse spike C's harness (real write path + relay) and
  spike D's substrate discipline; exit status is the verdict, sibling contract (0/1/2/64 +
  restore-failure code). Natural red: pull leg disabled, realtime-only — it MUST miss the
  dark-window rows or the assertion set is vacuous. 🛑 A red here is a **successful spike**:
  it means the build cards need an explicit resync step, and finding that out costs one night
  now versus a crew member's phone sleeping through a write in production. Footprint: spike
  scripts only.

---

## Activity 3 — The walking skeleton (one row, end to end, behind a flag)

> Only the legs Activity 2 proved. The skeleton exists from here on and **grows into** the demo
> script rather than being authored at the end. **Trace:** Product + Engineering objectives.

- **`skeleton-one-row-end-to-end`** · **PLANNED** (slated: `20260808-2` C2) · Thread **one** checklist row from the real
  write path to an RxDB-served read in dev, behind an explicit flag, with the two list views and
  the fill view all still on REST. The first production call site of `createHQSyncDatabase()` and
  `startHQReplication()` in the repo's history. Must carry decision 126's shape verbatim (reads
  on RxDB, `/saveResponse` and `/submit` keep owning writes) and cite it, per the standing rule
  that a build WO may not propose the split itself. Footprint: page wiring + sync client.

- **`skeleton-offline-ownership-honesty`** · **DONE** (run `20260808-2`, card **C1**, branch
  `card/c1-skeleton-offline-ownership-honesty`; commits `b1d1bb7` (merge-intent + RF, first),
  `69f6543` (the fix)) · Closed **B-88**. The three `expect(src).not.toContain(…)` source-text
  assertions in `tests/sync-rxdb-client.spec.js` (formerly lines 1497-1499) — which never named
  `window.HQSync.db`, the fourth read route `workflows.html:3589`'s `defaultStore()` actually
  uses — are replaced with an object-level browser test: load `/workflows.html` for real, assert
  `window.HQSync.db === undefined` and that no RxDB/Dexie-backed IndexedDB database exists. **This
  is the flag-off contract `skeleton-one-row-end-to-end` (C2) must keep green** — with the sync
  flag off, page load must leave `window.HQSync.db` undefined; C2 may only set it inside a
  flag-gated branch. RF: Probe A showed the shipped guard blind to the literal `HQSync.db` already
  present in `workflows.html` (a hypothetical fourth assertion reds on the unmodified tree, the
  shipped three do not); Probe B showed the new test is a real gate — red (exit 1) when
  `sync-rxdb/bootstrap.js` was temporarily made to set `HQSync.db` on every load, green (exit 0)
  before and after, tree left byte-identical. Gates: G1 clean; G2 (Go) 9/9 packages ok, 0 FAIL,
  454 total `--- PASS:` lines (`internal/workflow` 35 exactly), `HQ_SYNC_SUBSTRATE_OPTIONAL` /
  `HQ_SYNC_GATE_CHILD` both unset; G2 (Playwright) de-confined to the full suite (799 tests, one
  summary block) — 791 passed / 2 failed / 6 skipped in 22.5m, the 2 failures
  (`inventory.spec.js:3124`, `sync.spec.js:1327`) are NOT the armed baseline (which all passed:
  `inventory.spec.js:883` B-27, `sync.spec.js:446` LST-17, `receipt-carousel.spec.js:123` B-162)
  but did not reproduce on an isolated `tests/`-anchored rerun (2/2 passed, exit 0) and this card's
  diff touches only the test file — ruled flake-trail, not a card failure, and left named rather
  than silently dropped; G4 idempotent, tree clean both runs, precache count 31. Logs:
  `.night-crew/runs/2026-08-08-2-autonomous/c1-gates/`.

---

## Activity 4 — Activate the read path (decision 126's shape)

> **Trace:** Product objective. Carries the riders the retired card left behind: **B-63, B-64,
> B-66–B-69, B-79**.

- **`activate-fill-view-reads`** · **PLANNED** (slated: `20260808-2` C3) · The checklist fill
  view reads from RxDB, scoped per-open-checklist (T-29 decision 105 — replication scope is
  **never** all collections at once, and no card may widen it without a recorded decision).
  🛑 **Hard requirement (operator, ledger T-43c):** crew members work multiple checklists
  concurrently (setup + food prep) — multiple live fill replications at once ARE the design,
  one per open checklist, cancelled on close; regression test drives two concurrent fill
  scopes (B-63's lead). Carries spike E's condition (T-42): trusted checkpoint catch-up is
  valid only while the relay stays trigger/NOTIFY-driven. Footprint: page wiring + sync client.

- **`activate-list-views-or-state-they-stay-rest`** · **PLANNED** (slated: `20260808-2` S1,
  reshaped as `list-views-decision-recording`) · **B-43 partially ruled at ledger T-43:**
  Approvals stays on re-fetch (recorded); 🛑 **the My Checklists read path is deliberately
  OPEN — the operator declined to rule it, and no card may decide it.** The card's remaining
  job: record the Approvals ruling and the open remainder in the code that lies about them
  today — fix B-64's stale `bootstrap.js` scope banner and restate the cancel rule as
  *"cancel before re-scoping THE SAME shape"* (B-63's corrected wording, T-43c) — after the
  fill-view lifecycle exists. Closes B-64; closes B-63 jointly with `activate-fill-view-reads`'
  concurrent-fill test. Footprint: page wiring + sync client.

---

## Activity 5 — Dev complete (the close bar)

> **Trace:** Delivery objective. This activity is the milestone's definition of done.

- **`demo-sync-target`** · **PLANNED** (slated: `20260808-2` S2, budget-gated stretch) · Ship `task demo:sync` as a **first-class deliverable**:
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
