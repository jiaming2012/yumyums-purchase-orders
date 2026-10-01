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
| 2026-10-01T18:38:36-04:00 | base-reds leg started on detached base `c4f6db4` | Go + full Playwright under `flock /tmp/hq-full-suite.lock`; logs `logs/base-go.log`, `logs/base-pw.log` |

---

## Merges

<!-- one section per merge, in landing order -->
