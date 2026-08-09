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

> 🛑 **Bar corrected 2026-08-08 (operator, morning triage of `20260809`; decision 161).** The
> paragraph above is what was *chosen at the roadmap round* — a scripted-fresh throwaway
> environment. Triage established that `task demo:sync` clears that letter but leaves the sync
> capability **running in no persistent environment**: `HQ_SYNC_REST_URL`/`HQ_SYNC_REALTIME_URL`
> are set nowhere, so the in-server `/sync/*` proxy answers 503 in `dev`, `dev:tailscale` and
> prod; the substrate and relay exist only inside the demo's throwaway stack, torn down the moment
> it exits. The operator's own intent (lines 31–34) — *"the sync capability running in my dev
> environment … something I can actually use"* — is therefore **not** met by the demo alone.
> **`dev complete` now means: sync runs in the operator's persistent dev environment and is usable
> in the app.** The demo (`demo-sync-target`, DONE) stands as the data-plane proof; the cards
> `sync-live-in-dev-substrate` + `sync-live-in-dev-app-proof` below (fanned out from
> `sync-live-in-dev` at slate-20260810, §1.5) are the work that makes the capability live and
> usable, and they must land before `dev-complete-attestation`.

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

- **`skeleton-one-row-end-to-end`** · **DONE** (run `20260808-2`, card **C2**, branch
  `card/c2-skeleton-one-row-end-to-end`; commits `bf9ed24` (merge-intent + RF red, first),
  `dc6e43a` (the skeleton), `53e2fbd` (sw), `42e547c` (B-70 fix), `ba464c1` (sw)) · Threads **one**
  checklist row from the real write path (`POST /api/v1/workflow/saveResponse`) to an RxDB-served
  read on `/workflows.html`, behind an explicit flag, with both list views and the fill view still
  on REST. **The first production call site of `createHQSyncDatabase()` and `startHQReplication()`
  in this repo's history.**
  **THE FLAG (G6-F3 found "the sync flag" naming nothing in the tree): `hq_sync_read`**, defined as
  `SYNC_READ_FLAG` in `sync-rxdb/bootstrap.js`, stored in `localStorage` with value exactly `'on'`,
  settable+persistable from the URL (`?hq_sync_read=on` / `=off`) so a crew phone with no devtools
  can drive it, **default OFF in every environment**, resolved once synchronously at module load.
  **C1's flag-off contract is kept by construction, not by timing:** `openSyncScope()` refuses
  SYNCHRONOUSLY — it throws before returning a promise, before `createHQSyncDatabase` is
  referenced, before any `await` — so no path even *begins* async database creation with the flag
  off (answers G6-F1's "samples early and is timing-blind" directly). No memory-backed RxDB
  instance is introduced anywhere, so the guard's Dexie-only IndexedDB scan stays sufficient
  (G6-F2). Cites **decision 126** verbatim at the call site (RxDB serves READS; `/saveResponse` +
  `/submitChecklist` keep owning ALL writes — carried, not proposed), **decision 105** (scoped,
  never pulled whole), and **spike E's condition (T-42)** verbatim — this card polls nothing, so
  no explicit resync step is required, and the call site says so for whoever changes the relay.
  Shaped for **T-43(c)**: one shared promise-memoised database, one registry entry per scope,
  multiple different scopes live at once, the same scope twice returning the SAME handle, per-scope
  `cancel()` — **C3 builds the fill view on this without changing it**. Nothing here decides the My
  Checklists read path (**T-43(b)**, still OPEN); Approvals stays on re-fetch (**T-43(a)**).
  RF: the new end-to-end test RED on the pre-change tree (`3 failed`, EXIT=1) with the red landing
  only at the missing surface — the real `/saveResponse`, the submit that moves the draft onto a
  submission, and a psql read-back asserting exactly one persisted row with `value=true` all passed
  unmodified — then GREEN (`3 passed`, EXIT=0) after.
  🛑 **The first full suite went RED on this card's own defect and it is recorded, not buried:**
  `tests/repo-hygiene.spec.js:41` caught two raw `U+0000` bytes the card wrote into
  `sync-rxdb/bootstrap.js`'s `scopeKey()` — **B-70 recurring in a new file** (a raw NUL puts grep
  into binary mode, which is what makes `done_when: "grep returns nothing"` unreliable in the
  passing direction). Fixed to the `\0` escape (same byte at runtime, no key or identifier
  changes), commit `42e547c`. Gates: G1 build+vet clean; G2 (Go) 9/9 packages ok, 0 FAIL, 454
  `--- PASS:` lines, `internal/workflow` exactly 35, 2 live-proof skips,
  `HQ_SYNC_SUBSTRATE_OPTIONAL` / `HQ_SYNC_GATE_CHILD` both unset; G2 (Playwright) de-confined to
  the full suite, ONE summary block both legs — leg 1 (`dc6e43a`) 802 tests, 795 passed / 1 failed
  / 6 skipped in 23.0m with all three armed reds PASSING (`inventory:883` B-27, `sync:446` LST-17,
  `receipt-carousel:123` B-162), the single failure being the NUL defect above; leg 2 on the final
  tree recorded in the gate log. G4 idempotent (byte-identical `sw.js` across runs), precache count
  **31 — unchanged, and correctly so: this card adds no precached asset** (the dev surface is an
  inline module block; `sync-rxdb/bootstrap.js` was already precached). `night-crew.toml`'s
  `[e2e.seams]` roll-call gained `sync-one-row.spec.js` and `repo-hygiene`'s count went 9→10 — **no
  key and no token changed**; the guard fired as designed and an Operations-confined card now costs
  ten spec files. Logs: `.night-crew/runs/2026-08-08-2-autonomous/c2-gates/`.

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

- **`activate-fill-view-reads`** · **DONE** (run `20260808-2`, card **C3**, branch
  `card/c3-activate-fill-view-reads`; commits `ea1407a` (merge-intent, first), `5929f0f`
  (red-first tests, no production code), `61d13d0` (the implementation)) · The checklist FILL
  view's field values are served out of RxDB for the OPEN checklist, behind the `hq_sync_read`
  flag that is **OFF by default in every environment**. `hydrateFieldState` gains a **layer 4**
  — the open checklist's rows as RxDB holds them, overlaid on the REST snapshot they are a newer
  view of, **except over a `REJECTION_FLAGS` field**, which step 3 cleared on purpose and where
  the replicated row is precisely the stale answer the approver bounced. The `_v` /
  `_fail_note` / `_correction_photo` / `sub_steps` unpack was **extracted** from the draft loop
  into `applyResponseRow` and shared rather than copied — the replicated rows ARE
  `submission_responses` rows, and a second unpack is a second, drifting definition of one wire
  shape.
  **Lifecycle (`HQFillSync`):** one scope per open checklist, cancelled on close, and
  `FILL_SYNC_SCOPES` is a **MAP, not a slot** — ledger **T-43(c)**, the operator's own ruling
  that crew members work a setup checklist and a food-prep checklist concurrently. Opening a
  second does not cancel the first; `close()` cancels exactly one. Opened on checklist open
  **when the checklist has a submission row** (`checklist_submissions.id` is what decision 105
  scopes BY; an absent id is not permission to widen), closed at every exit from the runner —
  back button, all four post-submit exits, `show(1)`.
  **C2's two C3-facing G6 findings, both resolved.** **F-2:** `normalizeScope` now REQUIRES
  `userId` on the FILL scope and `scopeIdentity()` carries it, exactly as SCOPE-03 has for the
  LIST scope since S1a — the fill checkpoint had no crew member in its key, so crew member B on
  a device A used resumed A's `_modified` cursor and slept through B's own older draft rows,
  permanently. It appears in **no filter clause** (RLS is the gate; the client scope is the
  bound). 🛑 **A narrowing, not a widening** — more identifiers, each over a subset, every
  emitted query unchanged — so decision 105 is satisfied rather than amended and **decision
  111's four substrate rows are untouched**, which is why the PARK trigger did not fire.
  **F-1:** a rejected `createDatabase()` / `openSyncScope()` is now **evicted** from its memo
  instead of cached for the page's lifetime; `ensureDatabase()` calls `HQSync.createDatabase`
  (the property) so the failure G6 could not force — Dexie holds its own IndexedDB reference,
  which is why F-1 shipped PLAUSIBLE — is forceable and therefore testable.
  Decision **126** carried verbatim (RxDB serves READS; `/saveResponse` + `/submitChecklist`
  keep owning ALL writes; `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops`
  byte-untouched, no `autoSaveField` — B-65). **T-43(b) respected: the My Checklists read path
  is NOT decided here** — both list views stay on REST with the flag on as well as off, and the
  `[FILL-01]` test asserts the list row still renders `0/1` from REST while the runner reads
  `1 of 1` from RxDB. Spike E's condition (**T-42**) carried verbatim at the lifecycle: no
  polling, no interval, no business watermark, no resync step.
  RF, **same spec revision both legs** (the tests are their own commit `5929f0f`, no production
  code in it): RED `8 failed / 63 passed`, EXIT=1 — F-2 ×3, F-1's unforceable failure, and the
  fill view having no RxDB read path at all; GREEN `114 passed`, EXIT=0 on `61d13d0` with C2's
  `sync-one-row.spec.js` added to the leg. 🛑 **Stated rather than dressed up:** the card's
  *named* two-concurrent-fill regression test **passed pre-change** — two scopes for ONE crew
  member already minted distinct identifiers (SCOPE-02) — so it is a regression guard, and the
  red half of that requirement is the **shared-phone** case, where concurrency and F-2 meet.
  Gates: G1 build+vet exit 0; G2 Go **9 packages ok / 454 PASS / `internal/workflow` exactly
  35**, `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` both attested UNSET; G2 Playwright
  the FULL suite, **ONE summary block**, `806 passed / 1 failed / 6 skipped` in 22.8m with all
  three armed reds **PASSING** (`inventory:883` B-27, `sync:446` LST-17,
  `receipt-carousel:123` B-162). The single failure — `workflows.spec.js:3909` **GLB-01**,
  `page.goto ... interrupted by another navigation to /login.html` — is the test's own expected
  redirect racing its `waitUntil:"load"`, is **not** on the armed list and **not** previously
  recorded in this run's logs or `bugs.md`; **4/4 green on re-run** (3× isolated `-g`, plus the
  whole `tests/workflows.spec.js` file 87/87 exit 0). Reported as a **flake with its trail**,
  not as a pass. G4 idempotent (byte-identical `sw.js` across runs), precache **31 —
  unchanged**: this card adds one spec file and no precached asset. `night-crew.toml`'s
  `[e2e.seams]` roll-call gained `sync-fill-view.spec.js` and `repo-hygiene`'s count went 10→11
  — **no key and no token changed**; an Operations-confined card now costs eleven spec files.
  🛑 **G6 FIX ROUND — one CONFIRMED TOP defect, recorded not buried** (commits `276068b`
  red-first tests, `3e4397d` the fix). **F-A:** the overlay subscribed on `field_id` alone
  and applied every doc it got back. Three individually-correct premises made that a
  data-integrity defect — `checklist_fields.id` is per-template-VERSION and shared by every
  submission of that template; the `hq_sync` Dexie DB is persistent and `cancel()` purges
  nothing (**B-42**, RxDB's downstream only ADDS); and `submission_responses_select` is
  `hq_can_see_field(field_id)`, **field-level, not authorship-level**, deliberately, because
  a draft has no submission to scope a policy by. So yesterday's rows for a daily recurring
  checklist AND other crew members' drafts both sat resident and both marked today's blank
  checklist answered — **measured: `2 of 2 items complete` on a blank two-field checklist.**
  Fixed by `acceptedFillDocs()`, which is not a new policy but the REST hydrate's own,
  applied to the same rows arriving by a second road: `submission_id ===` the open checklist
  renders (shared per-submission, as `MY_SUBMISSIONS.responses` are), `submission_id == null
  && answered_by === me` renders (mine, as `DRAFT_RESPONSES` are), anything else is dropped.
  🛑 **The card's original "stated bound" is corrected in place rather than replaced: it
  reasoned about which scopes are open CONCURRENTLY and the defect was SEQUENTIAL — a bound
  on which scopes are live says nothing about which ROWS are resident.**
  **F-B:** `closeActiveFillScope()` moved out of `show()`'s `n === 1` branch to the top —
  every tab switch leaves the runner, and Approvals/Builder used to leave a Realtime channel
  and pull loop running for a checklist nobody had open. **F-C:** the deferred cancel is now
  identity-guarded ON THE HANDLE, so a close-then-fast-reopen no longer kills the scope the
  reopen believes it holds; testable without timing guesswork, because the symptom is
  `HQFillSync.openIds()` and `HQSync.openScopeKeys()` disagreeing. **F-D (prose):** "RLS is
  the gate" is a claim about VISIBILITY, not authorship — corrected in `client.js`'s F-2
  banner and `[SCOPE-05]`'s header, naming `acceptedFillDocs` as what actually excludes a
  foreign draft.
  **Fix-round gates, all on `3e4397d`:** G1 exit 0; G2 Go **9 packages ok / 454 PASS / 244
  top-level / `internal/workflow` exactly 35**, both env vars attested UNSET; G2 Playwright
  the FULL suite, ONE summary block, **811 passed / 0 failed / 6 skipped in 21.7m, EXIT=0**
  — all three armed reds passing, and **GLB-01 passed**, retiring the earlier occurrence as
  the flake it was reported as. RF: red `3 failed / 6 passed` EXIT=1 on `276068b`, green
  `74 passed` EXIT=0, spec byte-identical (0-line diff). G4 idempotent, precache **31**.
  Logs: `.night-crew/runs/2026-08-08-2-autonomous/c3-gates/`.

- **`list-views-decision-recording`** (formerly `activate-list-views-or-state-they-stay-rest`)
  · **DONE** (run `20260808-2`, card **S1**, branch `card/s1-list-views-decision-recording`;
  merge-intent `6a3b331` first, then this same commit set) · **B-43
  partially ruled at ledger T-43:** Approvals stays on re-fetch — recorded, in
  `sync-rxdb/bootstrap.js`'s rewritten `startReplication` banner and BACKLOG.md, not just this
  entry. 🛑 **The My Checklists read path stayed OPEN, as ruled** — this card states it as open
  in every banner it touches and predicts no outcome; the PARK trigger (recording requiring a
  decision on it) did not fire, because T-43(a) and T-43(c) are both fully recordable without
  touching (b).
  **B-64, found by content, not the slate's line numbers:** `bootstrap.js:80-86` (the slate's
  citation) is now `readIdentityToken()` — C2 added ~250 lines above it. The actual stale
  banner had moved to the comment on the `startReplication` property inside the `HQSync`
  object literal (~420-427): FILL-shape-only, silent on LIST, and closing with the pre-B-63
  full stop *"CANCEL the previous states before starting a re-scoped replication"* — which
  under T-43(c)'s concurrent shapes reads as "opening a second checklist cancels the first,"
  exactly the conclusion the ruling overturns. Rewritten to name both shapes (pointing at
  `client.js`'s docblock as the shape of record, per B-64's own lead, rather than a second
  copy), state what's live today (C2's `#sync-one-row` dev surface behind `hq_sync_read`; C3's
  per-open-checklist fill scopes, many at once), state what stays REST (both list views;
  Approvals BY RULING), state My Checklists OPEN, and restate the cancel rule.
  **`sync-rxdb/client.js`, two banners, both carrying the pre-B-63 wording:** the
  `startHQReplication` docblock (the one C3's merge-intent named explicitly as S1's to
  restate) and the older REPLICATION SCOPE design-block tail (same rule, predates the list
  scope, not named by C3 but found stale by the same reading). Both restated as *"cancel
  before re-scoping THE SAME shape"* (B-63's corrected wording, T-43c); C3's FILL-shape line
  `{userId, checklistId, templateId, fieldIds}` in the docblock is untouched, verbatim.
  **Closes B-64** (banner fixed). **Closes B-63 jointly with `activate-fill-view-reads`'s**
  (C3) `[SCOPE-05]` concurrent-fill regression test — the behavior half was already proven;
  this card supplied the documentation half.
  Docs-only diff: `sync-rxdb/bootstrap.js` + `sync-rxdb/client.js` comments, `BACKLOG.md`,
  this entry. No `workflows.html` touch (not owned; C3's merge-intent flags a conflict there
  as a mistake). RF: **n/a — non-code deliverable**, reason recorded in the card's own
  merge-intent (`.night-crew/runs/2026-08-08-2-autonomous/merge-intents/s1-list-views-decision-recording.md`)
  — no function body or test assertion changed, and B-63's behavior claim is proven by C3's
  own red-first tests, cited jointly above, not re-litigated here.

---

## Activity 5 — Dev complete (the close bar)

> **Trace:** Delivery objective. This activity is the milestone's definition of done.

- **`demo-sync-target`** · **DONE** (run `20260809`, branch `overnight-20260809`; verdict GREEN — round trip closed in 115 ms via `task demo:sync`; tri-state exits 0/1/2 all captured distinct) · Ship `task demo:sync` as a **first-class deliverable**:
  scripted-fresh environment (from Spike A), one field written through `/saveResponse`, surfacing
  in an RxDB-served read, on one real checklist. Non-zero exit on any failed leg. 🛑 **"Could not
  run" must render as an outcome distinct from "ran and failed"** — a demo that silently no-ops
  would reproduce the exact class this milestone exists to retire. Footprint: spike scripts +
  page wiring.

> **`sync-live-in-dev` fanned out into the two cards below at slate-20260810 (§1.5 split rule).**
> The decision-161 card bundled two normal-change-sized mechanisms in disjoint file families —
> persistent infra (legs 1+2 + the FDW-persistence finding) vs a novel-integration test (leg 3) —
> so an unattended run would have discovered mid-night that one card was two. The split maps 1:1
> onto the original four `done_when` items: items 1–2 → `-substrate`, item 3 → `-app-proof`, item 4
> (the attended attestation) → `dev-complete-attestation`. The spike gate is now GREEN and no
> longer blocks: leg 3 spiked GREEN (`spike-f-browser-live.sh` exit 0, run `f20260808232119`),
> legs 1 & 2 recorded "no spike needed" — ledger `spikes/activity-5-dev-complete/sync-live-in-dev.md`
> ("The goal is settled and slatable"). Two build-facts that ledger surfaced, inherited by both
> cards: (1) the proxy needs **four** `HQ_SYNC_*` vars, not two — `HQ_SYNC_JWT_SECRET` +
> `HQ_SYNC_REALTIME_HOST` are also required; (2) production per-user RLS resolves through the FDW
> server `hq_pg`, so the persistent env must arrange a persistent FDW→HQ pointing (with the
> `hq_sync_fdw` role given LOGIN), not only a scratch container.

- **`sync-live-in-dev-substrate`** · **PLANNED** (slate-20260810, Card 1; legs 1+2 + FDW
  persistence) · Make the RxDB data plane **run persistently in the operator's dev environment**
  and open the `/sync/*` proxy door. (1) **Persistent substrate + relay.** Bring up the Spike A
  substrate (PostgREST + Realtime) and the LISTEN/NOTIFY relay
  (`backend/internal/sync/spikec_relay.go` — NOT `cmd/spikec-relay`, which does not exist) as a
  **persistent dev service** (compose service + a `task` target), so the data plane stays up
  between runs instead of being spun up and torn down per `demo:sync`. (2) **Config wiring.** Set
  the **four** required vars — `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` / `HQ_SYNC_JWT_SECRET` /
  `HQ_SYNC_REALTIME_HOST` — in the `backend:dev` / `dev:tailscale` / `dev:lan` env so the in-server
  `/sync/*` proxy door **opens** (today it answers 503 everywhere). 🛑 Honor `proxy.go:78`'s
  ACTIVATION-ORDER guard — dev targets only, never `docker-compose.prod.yml`. (3) **FDW
  persistence.** Arrange a persistent FDW→HQ pointing: `hq_pg` resolves real per-user RLS against
  the operator's live `dev:tailscale` HQ (carrying the `hq_sync_*` source views + the `hq_sync_fdw`
  role given LOGIN), not only a scratch container as the spike did by repoint+restore.

  done_when:
  - Starting the dev stack opens the substrate door: with the persistent substrate + relay up and
    `HQ_SYNC_*` set, a request to `/sync/rest/…` through the running dev server returns **200, not
    503** — check: bring up the dev stack, `curl` the proxy path (explicit non-`:5433` coordinate),
    assert non-503.
  - The relay carries a real write to the substrate: a field written via `/saveResponse` appears
    in the substrate within the Spike-A convergence bound — check: write through the running dev
    server, poll the substrate, assert the row arrives.

  Footprint: persistent-substrate compose (`docker-compose.supabase.yml` or a new persistent-dev
  compose), root/`backend` Taskfile (`task` target + 4× `HQ_SYNC_*` dev env),
  `backend/internal/sync/spikec_relay.go` (persistent-service wiring), FDW/role SQL for the
  pointing. KR trace: Delivery objective — Activity 5, corrected close bar.

- **`sync-live-in-dev-app-proof`** · **PLANNED** (slate-20260810, Card 2; leg 3) · **Depends on
  `sync-live-in-dev-substrate`.** Prove the sync capability is **usable in the app**: promote
  spike-f's `browser-live/workflows-live.spec.js` into a repo **red-first** Playwright spec that
  drives the real `workflows.html` (flag `hq_sync_read` ON, **no `page.route` stub**) against the
  live persistent substrate, enters one field through the real `/saveResponse` path, and asserts it
  surfaces via RxDB replication **in the app** (`#sync-one-row` → `data-state="served"`) — replacing
  the demo's Node RxDB read client with the actual app surface (closes the read-surface gap T-44
  recorded).

  done_when:
  - The **app** shows the round trip: the spec drives `workflows.html` against the live substrate
    (no `page.route` stub), enters one field, and asserts it surfaces via RxDB replication in the
    app — **red-first**: the same spec fails when the relay is stopped — check: run the spec with
    the relay up (pass) and down (fail). Gate the red-first on the spec/script exit, never on
    `task` (B-163).

  Footprint: a new red-first Playwright spec + its config (promoting `browser-live/`), its
  `night-crew.toml` footprint/seam integration (+ `tests/repo-hygiene.spec.js` count bump if a
  roll-call name is added); `workflows.html` / `sync-rxdb/*` read path only if a source edit proves
  necessary (spike evidence says not). Gate-harness integration (standalone spike-style spec vs a
  self-skipping `tests/` seam — the live-substrate spec cannot run in the standard `:5434` harness)
  is an engineer-level decision the card records in its merge-intent; a new `night-crew.toml`
  KEY/TOKEN would PARK. KR trace: Delivery objective — Activity 5, corrected close bar.
  🛑 **Spike gate: GREEN, no longer blocking** — leg 3 (real browser against the real substrate)
  was spiked GREEN (`spike-f-browser-live.sh` exit 0). This is the integration the milestone had
  never spiked; it now has.

- **`dev-complete-attestation`** · **PLANNED** · 🛑 **Attended, and the operator's own act.** Per
  the corrected bar (decision 161), this is no longer "run `task demo:sync`" alone: the operator
  opens `workflows.html` in their dev environment (`dev:tailscale`, against the persistent
  substrate delivered by `sync-live-in-dev`), sees a field sync in the app, and records the outcome
  in `ledger.md`. `task demo:sync` remains the data-plane self-check; the attestation is now the
  app-surface read. **The milestone does not close without this line.** No card, grade or closeout
  substitutes for it.

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
