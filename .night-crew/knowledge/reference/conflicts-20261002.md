# Merge conflict log — run `20261002`

Required by §15ad.66: **every** merge to `overnight-20261002` gets an entry here, clean or
conflicted — the cards involved, files and hunks, the intents read, the resolution taken, the
precache count after `task sw`, and the gate result after it. A clean merge gets a one-line
entry, so an empty log can never read as "no conflicts" when it means "the logging never ran".

Slate: `reference/slate-20261002.md` (7 cards, Activity H, concurrent 3-track after a solo
Wave 0). Shared-surface resolution rules are that slate's "Shared surfaces and serialization"
table; the orchestrator resolves against **intent, not text**.

Baseline at branch cut: `c4f6db4`, precache count **48**.

---

## Launch legs (not merges — recorded so the log reads in order)

| When | Leg | Result |
|---|---|---|
| 2026-10-01T18:37:24-04:00 | `env-up.sh` reconcile (never `--fresh`) | **GREEN, `EXIT=0`** — db=53222 rest=52932 realtime=52074, project `spike-supabase` |
| 2026-10-01T18:38:36-04:00 | base-reds leg started on detached base `c4f6db4` | Go + full Playwright under `flock /tmp/hq-full-suite.lock` |
| 2026-10-01T18:55:25-04:00 | base-reds **Go** | **CLEAN, `EXIT_TEST=0`** — 13 pkgs, 368 top-level (623 incl. subtests), 0 fail, 2 opt-in skips; `internal/workflow` ran 39 (ladder floor 35), `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` both unset in-log. `logs/base-go-2.log` |
| — | base-reds Go, **first attempt discarded** | `EXIT_TEST=1`, three `internal/sync` reds — caused by the orchestrator minting `HQ_RLS_TEST_DB=hq_test_rls_…`, which the fixture-ownership guard refuses (`^hq_rls_…$`, B-141/decision 155). **Not a base red.** Ladder stanza never stated the shape; fixed in `dbbc1a8`. `logs/base-go.log` |
| 2026-10-01T19:13-04:00 | base-reds **Playwright** | **24 failed / 965 passed, 34.8m, exactly ONE summary block, `EXIT_PW=1`** — this is the night's no-new-reds set. Composition and the 4 reds new to the record: `bugs.md` addendum, `09c6586`. `logs/base-pw.log` |

---

## Merges

<!-- one section per merge, in landing order -->
