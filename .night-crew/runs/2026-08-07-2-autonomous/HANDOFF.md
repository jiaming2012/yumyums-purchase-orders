# HANDOFF — run `overnight-20260807-2`

**2 of 2 cards merged. Zero parks. Zero open forks. All four D-KR1 spike verdicts are now
recorded — A GREEN, B GREEN, C GREEN, D GREEN — and the Activity 3–5 build gate is OPEN.**
Run window: ~00:05–02:05 EDT, 2026-08-07 (the true overnight of the 2026-08-06 evening
sign-off). Serial dispatch as signed. Final tree: see closeout commit on `overnight-20260807-2`.

## Per-card outcomes

| Card | Outcome | Merge | G6 verdict | Planned → actual (incl. G6) |
|---|---|---|---|---|
| C `spike-c-round-trip` | ✅ MERGED — **verdict GREEN, exit 0** (3rd of 4 D-KR1 spike verdicts) | `76801aa` | MERGE-WITH-NOTES | 90–150m → ~125m implementation + spike runs; ~2h05m wall with the full gate ladder (the suite is the overage, not the mechanism) |
| D `spike-d-realtime-live` | ✅ MERGED — **verdict GREEN, exit 0** (4th of 4; answers B-62) | `7101b1c` | MERGE-WITH-NOTES | 60–120m → ~75m wall |

Both merges were clean (all-additions; disjoint Taskfile stanzas). Conflict log:
`.night-crew/knowledge/reference/conflicts-20260807-2.md`, §1–§2 — clean merges logged per rule.

## What is now true

- **Spike C GREEN — the HQ-Postgres → substrate → RxDB-read path EXISTS, mechanism proven:
  a LISTEN/NOTIFY relay.** An `AFTER INSERT OR UPDATE` trigger on `submission_responses` fires
  `pg_notify`; a Go relay (`pgxlisten`, same dependency `internal/sync/listener.go` already
  uses) re-reads the row, transforms it to the sync contract's shape, writes it into
  `hq_sync_checklists` through PostgREST **with a service identity** (the lane spike B measured
  as the only viable bulk lane); RxDB's Realtime pull delivers it. Round trip **248 ms**
  (G6 re-execution: 136 ms) against a 20 000 ms bound. The write was genuinely real: HQ's own
  binary, all 74 migrations into a scratch Postgres, real login cookie, `POST
  /api/v1/workflow/saveResponse` → 204, arriving RxDB doc carried the real user uuid +
  field value. Candidates rejected with reasons in the script header: polling relay (fallback,
  strictly worse latency), logical replication (no transform stage for HQ's shape), PostgREST
  forward-writer in the handler (puts the substrate on the crew's write path — contradicts
  decision 126's shape). Spike code is loudly marked; `RunSpikeCRelay` referenced nowhere in
  `cmd/server`.
- **Spike D GREEN — the live Realtime server honours the replication filter** in all three
  clause shapes `sync-rxdb/client.js` production code emits (`submitted_at=gte.<iso>`,
  `id=eq.<id>`, `id=in.(a,b)` — the `in.(…)` acceptance was the likeliest refusal and is now
  measured). Suppression is attributable: the identical out-of-scope row arrived on an
  unfiltered subscription to the same table over the same socket in the same window. The
  `responses` unfiltered residual is now **measured**, not asserted. `HQ_SYNC_REST_URL` never
  read/set — the interlock stayed armed. `sync-rxdb/client.js` untouched; the spike drives the
  already-exported `realtimeFilterFor()`, so the wire string IS the production string, and
  `tests/sync-rxdb-client.spec.js` stayed green byte-for-byte (55/55).
- **The Activity 3–5 build gate is OPEN.** The roadmap's STOP ("no build card may be cut until
  spike C is green") does not fire. Cutting the build cards is the **next slate's act**, per
  the slate's own rule — nothing was cut mid-run.
- Roadmap flipped on this branch: both cards → DONE (`8a45e51`), with verdicts and run id.

## Gate evidence on the final tree

Final HEAD differs from D's gated card tip only by docs (conflict log §2, roadmap flip,
this closeout). The code state IS card D's gated tree, whose base included card C's merge:
- **G1** clean (backend module root; card logs `card-c-g1.log`, `card-d-g1.log`).
- **G2 (Go)** 9 packages, 456 `=== RUN` / 0 FAIL (246 top-level: 244 PASS + 2 documented
  SKIPs), `internal/workflow` = **35**. Env attestation **captured in the log artifact**
  (`card-d-g2-go.log` head: `HQ_SYNC_SUBSTRATE_OPTIONAL` / `HQ_SYNC_GATE_CHILD` `<unset>`,
  `DB_TEST_URL` on :5434) — the artifact-capture G6 asked for on C.
- **G2 (Playwright)** full suite, `--retries=0`, exactly one summary block, both cards:
  C: 790 passed / 2 failed / 6 skipped, 25.7m (fails: LST-17 the named armed red + FR-11, see
  follow-up 1). D: **792 passed / 6 skipped / 0 failed**, 25.5m — both armed reds passed
  (green-direction; G6 roster-diffed the two logs: zero test-identity differences).
  Wall times inside the 21–26m re-baselined band — do not re-arm.
- **G4** on merged HEAD after each merge: exit 0 twice, precache **31**, `sw.js`
  byte-identical, 1.4.0 three-way parity.
- **RF** both cards: C `--no-relay` → exit 1 with the relay absent from the tree at that
  commit (`card-c-rf-red.log`); D `--no-filter` → exit 1 on the suppression leg only
  (`card-d-rf-red.log`), teardown VERIFIED on the red path — B-148's un-rehearsed-red-path
  residual is discharged for this harness. Both reds reproduce on demand via the scripts'
  flags; both G6s re-executed them.
- **G6** both cards MERGE-WITH-NOTES; both verdicts survived adversarial re-execution
  (all exit-code contract paths reproduced: 0 / 1 / 2 / 64, and C's occupied-port refusal).
- **G4 discipline greps: N/A-VACUOUS — neither package exists in this repo (B-14).**
- Suite lock honoured structurally: serial dispatch, one suite in flight, ever. Spike A's
  stack consumed in reconcile mode by all four legs (2 implementers + 2 G6s); substrate
  restore VERIFIED byte-identical each time; containers never restarted ("Up 33 hours" at D's
  G6). Isolation: unique TEST_PORT/TEST_DB_NAME/HQ_RLS_TEST_DB/DB_TEST_URL per leg, all on
  :5434; **nothing resolved any name against :5433 all night** (spike DBs rode ephemeral
  ports).

## Follow-ups the run leaves (none blocking; fresh branches off `dev` where code is implied)

1. **FR-11 needs a triage ruling under the flake protocol** — `tests/inventory.spec.js:3577`
   failed its UI half (cardCount 0 at `:3626`, 1.5s assertion mismatch — NOT B-32's 30s-timeout
   shape) at whole-suite position in C's run, then **passed** in D's run. It is on no armed
   list (B-27 is `:883`, B-32's inventory member is `:2908`). Candidate new named member;
   neither card can have caused it (both diffs have zero frontend/spec files).
2. **LST-17, the named armed red, passed in BOTH suites tonight.** Either the underlying
   defect got fixed (B-147/B-148 landed today) or "armed" is doing less work than assumed.
   Decision 100 / T-31 decision 120 bind: retired by diagnosis, never by passing — but two
   consecutive full-suite passes are evidence triage should weigh.
3. **Relay double-fire, for the cutover card:** one `/saveResponse` call fires the relay twice
   per row — `workflow/repository.go:826` (the save) + `sync/ops.go:148` (EmitOp's lamport
   stamp UPDATE). Any CDC mechanism on this table sees 2× write volume; harmless tonight only
   because the projection is an idempotent upsert. Recorded beside the code in
   `spikec_relay.go`.
4. **`app_slug` is a constant in the relay** — HQ still stores no template→app association
   (spike B's finding #1 resurfaced; labelled in code).
5. **Spike-script hardening, one line each (cutover/next-touch):** C's `rxdb/spike-c-read.js`
   checks `replErrors` only pre-write (a post-write infra death could mislabel exit 1); D's
   suppression leg would be backstopped by asserting rtprobe's `events==1` COUNT lines.
6. **`spike-d-realtime.sh --fresh-substrate` is a documented destroy footgun** — forwards
   `--fresh` to `env-up.sh` (`down --volumes`) inside a reconcile-only discipline. Opt-in,
   never default; triage may want it renamed or guarded.
7. **Evidence hygiene (resolved for D, noted for C's record):** C's env-unset attestation
   lives only in a commit message and `card-c-g2-bddgen.log` is zero bytes — D's implementer
   established the cause (no `node_modules` until `npm ci`; bddgen's message goes to stderr)
   and captured both properly. No re-gate needed; the indirect in-suite evidence supports C's
   claim (G6 judged it true-but-unproven-by-artifact).
8. **Harness lesson worth propagating:** `( … ) &` makes `$!` the subshell pid → orphaned
   server holds the port and the next run's health poll goes green against a foreign DB.
   Fixed both ways in C's harness (env-exec + occupied-port refusal); same class as
   `playwright.config.js`'s `reuseExistingServer` note.
9. **B-62 can close** — answered GREEN by card D. Its recorded destination
   (`sync-hard-cutover`, "first moment this is testable") was pessimistic: testable now
   against the spike stack, no deploy config touched. Triage's flip to make.

## Attended work still waiting (unchanged by tonight; standing rule 1 kept the run out of all of it)

- **A3 re-gate** (`gate-rls-fixture-ownership`) — attended by ruling (decision 155); branch
  `card/a3-rls-fixture-own` + worktree preserved and untouched tonight.
- **Decision-156 Mercury backfill** (attended).
- **B-146 SFTP key fix** (attended).
- **Decision-159 `archive_mode` enablement residual** (attended).
- **Decision-158 sales-processor message** (attended).
- **B-145 recovery Phase 1** (attended queue).

## Next actions

1. `/nc-morning-triage` on this branch — audit the conflict log, rule on follow-ups 1, 2 and
   9, merge to `dev`.
2. Next slate: the Activity 3–5 build cards are un-gated by C's GREEN — cutting them is that
   slate's act, on the recorded verdicts.
