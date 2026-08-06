# Conflict log — run `overnight-20260807`

Every merge to the run branch gets an entry, clean or conflicted (§15ad.66). An empty
log means the logging never ran, not "no conflicts".

## §1 — W0 `test-cluster-separation` → `overnight-20260807` (CLEAN, with an orchestrator incident)

- **Merge commit:** `a03c6bc` (no-ff, ort strategy), card branch `card/w0-test-cluster-separation`
  tip `9a3a917`, base `379d859`. **Zero conflicted hunks** — the run branch had not moved since
  the card branched.
- **Intents read:** W0's only (`merge-intents/w0-test-cluster-separation.md`) — first merge of
  the night, no counterpart intent exists. What-must-survive (the re-pointed
  `defaultHQAdminURL`/`defaultFDWPort` constants, the A3 re-gate rebase contract) survives
  verbatim; nothing was dropped.
- **Files:** 24 (Taskfile.yml, backend/Taskfile.yml, docker-compose.test.yml new,
  playwright.config.js, scripts/reset-e2e-db.js, scripts/verify-test-harness.sh,
  rowvisibility_rls_test.go, CLAUDE.md, roadmap/BACKLOG flips, merge-intent + w0-logs).
- **Gate after merge:** merged tree is content-identical to the card tip for every code path
  (base advanced zero commits), so the card's committed gate evidence carries over: G1 clean,
  G2 Go 9 pkgs / 246 top-level / 456 with subtests (internal/workflow 35, subtest-count
  assertions PASS, `HQ_SYNC_SUBSTRATE_OPTIONAL`/`HQ_SYNC_GATE_CHILD` attested unset), G2
  Playwright 791 passed / 6 skipped / 0 failed, one summary block, 26.1m, on the new :5434
  container. G4 independently reproduced by G6 on the identical tree (31 precached, 1.4.0
  parity, idempotent).
- **G6 verdict:** MERGE-WITH-NOTES. Three concerns recorded for morning triage, none blocking:
  (1) Taskfile v3 runs `deps` in parallel, so the composed `task test:*` targets' "db up
  first" ordering is unproven from cold — follow-up fix off `dev` recommended; (2) the
  "no `yumyums:yumyums` in any test-path default" claim is overstated — seven TestMain
  fallback DSNs still carry it aimed at :5432 (a different live cluster, not the hq-serving
  one); (3) the newly-armed `backend:db-test` guard fails OPEN on psql connection error.
- 🛑 **Orchestrator incident, recovered:** the first execution of this merge landed on `dev`
  (as `3820cc9`) because the **main checkout had been switched from `overnight-20260807` to
  `dev` by a concurrent attended session** — the same session whose de-confinement gate
  commit `eb8e415` landed on `dev` mid-card. Recovery: verified `3820cc9`'s first parent was
  exactly `eb8e415` and that the mis-merge was the only foreign commit, `git reset --hard
  eb8e415` on `dev` (removing only the orchestrator's own minutes-old merge; the attended
  commit untouched), then re-executed the identical merge on `overnight-20260807` in a
  dedicated run worktree (`hq-worktrees/run-20260807`), where all subsequent run-branch git
  operations happen. Nothing of the attended session's work was altered. For triage: two
  sessions sharing the main checkout is the same hazard class the slate's amendment note
  already flagged (slate signed on stale premises, 2026-08-06) — now it has moved a git
  write. Also of record: that attended session ran a full Playwright suite against :5433
  while W0 was in flight — standing rule 2 bound this run, which waited; the attended act
  was the operator's own.

## §2 — A2 `shipped-bug-sweep` → `overnight-20260807` (CLEAN)

- **Merge commit:** `405a52f` (no-ff, ort), card branch `card/a2-shipped-bug-sweep` tip
  `f4dab6a`, base `35ad2ec`. **Zero conflicted hunks** — the run branch had not moved since
  the card branched.
- **Intents read:** A2's (`merge-intents/a2-shipped-bug-sweep.md`) and W0's — no file
  overlap (A2 stayed off Taskfile/compose/scripts entirely); W0's what-must-survive
  (rowvisibility constants) untouched by A2. Nothing dropped.
- **Files:** sync-rxdb/bootstrap.js, tests/sync-rxdb-client.spec.js, workflows.html,
  tests/workflows.spec.js, sw.js (regenerated, content-hash only), roadmap/BACKLOG flips,
  merge-intent + a2-logs.
- **Gate after merge:** merged tree content-identical to card tip (base unmoved), so the
  card's evidence carries: G1 clean; G2 Go 9 pkgs ok (DB-coupled execution corroborated by
  durations, G6); G2 Playwright **792 passed / 6 skipped / 0 failed**, one summary block,
  25.1m, +1 vs baseline = the card's own new B-132 regression test; G4 reproduced
  byte-identical by G6 at HEAD (31 precached, 1.4.0 parity).
- **G6 verdict:** MERGE-WITH-NOTES. For triage: (1) 🛑 the permanent B-132 test hardcodes a
  screenshot path into `.night-crew/runs/2026-08-07-autonomous/a2-logs/` — every future
  full-suite run rewrites this run's committed evidence; fix off `dev` (drop the screenshot
  step or point it at test-results/) BEFORE the next overnight run; (2) gate logs carry no
  command/env attestation (results corroborated by durations/flaky-absence, but the format
  is thin — convention item); (3) merge-intent footprint omitted tests/workflows.spec.js
  while naming it in what-must-survive. Also: before/after B-132 PNGs are byte-identical —
  the pre-fix state stands on b132-red.log (IndexSizeError at 10:50:54, pre-fix-commit),
  not the PNG pair.
