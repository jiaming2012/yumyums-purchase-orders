# HANDOFF — run `overnight-20260807`

> **TRIAGED 2026-08-06 — merged to `dev`, recorded as ledger T-40.** Adversarial review
> reproduced all gates and mutation-verified both bug fixes; spike GREEN recorded unverified
> by triage (G6-executed during the run). Follow-ups 1–8 below graduated to BACKLOG.md as
> **B-147–B-155** (+B-149, a new triage finding: the uid-mismatch half of B-89 is unguarded
> by test); B-147 and B-148 are flagged fix-before-next-run. The worktree standing rule is
> decision 160 / pending candidate `operations/C-3`. `card/d1-syncspec-deflake` cut under the
> net-zero rule. The run executed 2026-08-06 by wall clock; this file's dates carry the run
> id. The "Attended work still waiting" section remains live and untouched.

**3 of 3 cards merged. Zero parks. Zero open forks. Decision 155 is now real.**
Run window: ~09:25–12:55 EDT, 2026-08-07 (the "overnight" ran in daylight — launched by the
operator mid-morning). Serial dispatch as signed. Final tree: `ee109b1` on `overnight-20260807`.

## Per-card outcomes

| Card | Outcome | Merge | G6 verdict | Planned → actual (incl. G6) |
|---|---|---|---|---|
| W0 `test-cluster-separation` | ✅ MERGED | `a03c6bc` | MERGE-WITH-NOTES | 90–150m → ~68m (implementer 54m; part of the gap was waiting out a foreign :5433 suite, see Incidents) |
| A2 `shipped-bug-sweep` | ✅ MERGED — B-89 + B-132 closed | `405a52f` | MERGE-WITH-NOTES | 70–105m → ~91m wall (inflated by a mid-card implementer stall that was resumed; work time within estimate) |
| S `spike-b-migration-rehearsal` (stretch) | ✅ MERGED — **verdict GREEN, exit 0** (2nd of 4 D-KR1 spike verdicts) | `61eb4af` | MERGE-WITH-NOTES | 60–150m → ~65m |

Stretch gate was computed, not vibed: at 11:31 EDT both full cards were landed clean ~2.1h in;
S's 150m high end + 30m closeout projected a ~14:30 finish inside the sign-off envelope.

## What is now true

- **Decision 155 is executed and proven.** Test suites run against `yumyums-test-pg`
  (postgres:16, **:5434**, `hqtest`/`hqtest`, volume `yumyums-test-pgdata`,
  `docker-compose.test.yml`, `task test:db:up|down`). Every test-path default re-pointed:
  Taskfile `test:*` env blocks, `rowvisibility_rls_test.go` constants, `verify-test-harness.sh`,
  `scripts/reset-e2e-db.js` (the actual DROP site), plus `backend:db-test`'s dev-host guard
  armed (`ALLOW_TEST_DB_ON_DEV_HOST=0`). B-141's structural half closed. Full Go suite
  (9 pkgs, workflow=35, sync subtest guards PASS) + full Playwright (791→792/6/0) green on it.
- **B-89 closed**: `cachedGrantSlugs()` reads the `{uid, apps}` envelope, verifies `uid`
  against the identity token, mirrors `index.html`'s shipped mismatch behavior (→ `[]`) —
  explicitly not a new product decision. The lying bare-array fixture fixed in the same commit.
- **B-132 closed**: radius clamped; the "frozen overlay" is settled — it was a fully
  transparent leaked canvas, not visible frozen confetti (screenshots committed). Was firing
  **2482×** per suite run (stale 28× figure corrected in BACKLOG). New regression test = the
  +1 in 792.
- **Spike B GREEN**: HQ-shaped data (8-table subset, 3 users/2 apps/3 templates/7 submissions)
  migrated into spike A's Supabase substrate via PostgREST and surfaced in RxDB with RLS
  discriminating correctly (2/2/1 rows per client, negative control invisible). 48 named
  assertions; genuinely red-first (run 1 red on `jsonb_array_length`); G6 reproduced the GREEN
  independently and verified byte-identical substrate restore + full teardown.

## Gate evidence on the final tree

Final HEAD `ee109b1` differs from S's card tip only by docs (conflict log §3 + a 49→48
assertion-count correction). The code state IS S's gated tree: G1 clean, G2 Go 9 packages
(counts in `s-logs/g2-go-counts.log`, env attested unset by expansion), G2 Playwright
**792 passed / 6 skipped / 0 failed**, one summary block, `--retries=0`, 20.9m
(`s-logs/g2-playwright.log`), G4 sw.js unchanged / 31 precached / 1.4.0 three-way parity.
Suite lock honoured structurally: serial dispatch, one suite in flight, ever.
G4 discipline greps: **N/A-VACUOUS — neither package exists in this repo (B-14).**
Full-suite wall-clock re-baselined on the new container: 26.1m (W0), 25.1m (A2), 20.9m (S)
vs ~24.2m historical — no material move; do not re-arm.

## Incidents (for triage, both recovered, nothing lost)

1. **The orchestrator's first W0 merge landed on `dev` and was recovered.** The main checkout
   had been switched from the run branch to `dev` by a **concurrent attended session** — the
   same one whose commit `eb8e415` (T-39 de-confinement gate) landed mid-card. Recovery:
   verified the mis-merge (`3820cc9`) sat alone on `eb8e415`, `reset --hard` removed exactly
   it, and the identical merge was re-executed on `overnight-20260807` in a dedicated run
   worktree (`hq-worktrees/run-20260807`), where all subsequent run git ops happened.
   Conflict log §1 carries the full account. **Two sessions sharing the main checkout has now
   moved a git write** — the same hazard class the slate's amendment note flagged (signed on
   stale premises). Worth a standing rule: runs operate from a dedicated worktree, never the
   main checkout.
2. **A foreign full Playwright suite ran against :5433 while W0 was in flight** (the attended
   session's de-confinement gate, landed as `eb8e415`). Standing rule 2 bound this run — W0
   waited it out rather than kill an attended process. Silver lining: 791/6 on :5433 vs 791/6
   on :5434, identical figures across clusters — clean evidence the move changed *where* the
   suite runs, not what it proves.

## Follow-ups the run leaves (none blocking, all G6-found; fresh branches off `dev`)

Priority-ordered; the first two should land **before the next overnight run**:

1. 🛑 **A2/G6-1:** the permanent B-132 test hardcodes its screenshot into
   `.night-crew/runs/2026-08-07-autonomous/a2-logs/` — every future full-suite run rewrites
   this run's committed evidence (already rewrote it once, bytes happened identical). Drop the
   screenshot step or point it at `test-results/`.
2. 🛑 **S/G6-1+2 (paired):** on a *red* run of the spike-b user-lane leg, the substrate-restore
   guarantee does not hold (probe row `probe-ok-<run>` escapes the manifest, contaminates
   `hq_sync_checklists` persistently → reds `TestJWTBridgeRLS`) AND the printed recovery
   command is inoperable (`SPIKE_B_HQ_CID` import guard). Fix both before anyone runs the
   script red.
3. **W0/G6-1:** Taskfile v3 `deps` run in parallel — the composed `task test:*` targets'
   "db up first" ordering is unproven from cold (fail-loud, second invocation succeeds).
   Serialize (db-test into `cmds` or hung off `test:db:up`) + one captured cold run.
4. **W0/G6-3:** the armed `backend:db-test` guard fails OPEN on psql connection error —
   distinguish empty-result from non-zero exit.
5. **W0/G6-2:** seven TestMain fallback DSNs still carry `yumyums:yumyums@localhost:5432`
   (a different live cluster, not the hq-serving one — but prod's credential pair in test
   files aimed at a live system). Re-point at `hqtest@5434` or delete.
6. **S/G6-3:** sharpen the spike's archived-exclusion assertion (`=== 6` + specific ids);
   also hq-reset's seed check is `> 0` not exact, and compose-down is warn-and-continue.
7. **A2 obs:** `night-crew.toml [e2e.seams]` has no row for `sync-rxdb/**` — undeclared paths
   de-confine to full suite (safe default, but declare it deliberately).
8. **Gate-log convention:** A2's logs carried no command/env echo (results were corroborated,
   the format was thin). Consider requiring the attestation header W0/S used.

## Attended work still waiting (operator's own, untouched tonight per standing rule 1)

- **The A3 re-gate** (decision 155's second half): branch `card/a3-rls-fixture-own` + worktree
  preserved untouched. W0's merge-intent states the two re-pointed constants as
  what-must-survive — the rebase contract is written down.
- **Decision-156 Mercury backfill** to 2026-03-01 — its cursor-pagination fix is now
  **committed** (`4efd265`, with `cmd/backfill-receipts`); the backfill itself has not run.
- **B-146**: prod Toast sync silently dead since the 07-28 rebuild — SFTP key never ships in
  the prod image.
- **Decision-159 residual**: `archive_mode` enablement on the live cluster (brief restart,
  operator present) — unless the 08-06 attended session already did it.
- **Decision-158**: the sales-processor message — held by the operator, no draft.
- **B-145 recovery Phase 1 (shrunken)** per the launch prompt's triage note.

## Housekeeping notes for triage

- `card/d1-syncspec-deflake` is a **net-zero** branch (fix `4ab162c` + revert `6ee45e0`,
  diff 0 lines vs base) — a proven fix+revert closeout note with a destination
  (run-20260724 vintage), not a question. Route or delete at triage per the standing
  net-zero rule; the run left it untouched.
- `hq_test_w0`/`hq_test_a2*`/`hq_test_s` + matching `hq_rls_test_*` DBs remain on :5434 —
  scratch by construction; leave or drop freely.
- The `yumyums-test-pg` container is `restart: unless-stopped` — deliberately permanent,
  credentials `hqtest`/`hqtest` published on 0.0.0.0:5434 (same exposure pattern :5433 has;
  recorded by W0's G6 as a NOTE).
- Card branches `card/w0-test-cluster-separation`, `card/a2-shipped-bug-sweep`,
  `card/s-spike-b-migration` are fully merged into the run branch; worktrees removed.
- No `night-crew decisions log` invocation was needed all night: no gray area reached the
  routing threshold — every judgment call fell inside card scope (A2's uid-mismatch was
  resolved by mirroring shipped behavior per the card's own carve-out, stated in its report
  and G6-confirmed). Nothing was delegated, so nothing awaits ratification.

## Milestone position (cards, not key results)

18 cards; 6 were DONE at sign-off; tonight lands **3 more (W0, A2, S)** → **9 DONE, 9
remaining**. `spike-c-round-trip` (load-bearing, 3rd of 4 D-KR1 verdicts) leads the next
slate per the slate's own recommendation. `gate-rls-fixture-ownership` stays BLOCKED-attended.

## Next actions

1. `/nc-morning-triage` on this branch — audit conflict log §1–3 (all clean merges; §1
   carries the mis-merge incident), rule on the follow-up list above, route the d1 net-zero
   branch, merge `overnight-20260807` → `dev`.
2. Attended queue as listed above.
