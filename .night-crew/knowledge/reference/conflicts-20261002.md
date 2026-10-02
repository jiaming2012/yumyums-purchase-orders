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

### Merge 1 — `card/h1-campaign-codes-api` → `overnight-20261002` (`40b5846`), 2026-10-01 ~20:15

**Cards involved:** H1 only (Wave 0, solo by design).

**Conflicted:** nothing. `git merge --no-ff --no-commit` reported *"Automatic merge went well"*.
Recorded anyway — a clean merge gets an entry so an empty log cannot read as "no conflicts".

**Files and hunks.** 32 files, ~12900 insertions: the new `backend/internal/marketing/` package
(13 files), migration `0083_campaigns_admin.sql`, `backend/cmd/server/main.go` (+60: three call
sites — `Mount`, the no-op `MountReports` in the BI block, and `MountPublic` for `/q/{short}`
outside auth), `go.mod`/`go.sum` (+`skip2/go-qrcode`), plus the card's merge-intent note and 8
gate logs.

**Intents read.** H1's merge-intent (`merge-intents/h1-campaign-codes-api.md`) — it declares the
package shape and the `routes.go` seam as *must survive*, names `main.go` as **three** call sites
rather than the two the dispatch brief assumed (the public landing cannot sit inside either gated
block), and records the shipped `POST /campaigns` JSON verbatim so H2 need not re-derive it.
No second intent existed to read: H1 is Wave 0 and merged alone.

**Resolution taken.** None required by git. The one thing that needed *checking* rather than
resolving: the card branch was cut at `c4f6db4` and so predated six orchestrator commits on
knowledge/run files, which made `bugs.md`, `conflicts-20261002.md`, `gate-ladder.md` and
`timings.log` appear as **removals** in `git diff overnight..card`. A diff is not a merge — the
merge kept the run branch's side on all four, verified by assertion after the fact: 8 timings
entries present, the gate-ladder `hq_rls_` + precache-48 fix present, tonight's base-red
re-measurement present. `roadmap.md` took both sides (H1 → **LANDED**).

**`task sw` at merged HEAD.** `node build-sw.js` ×2 → **48 files precached** both runs, byte-identical,
`git status` shows `sw.js` and `version.json` **unmodified**. Count unchanged 48 → **48**, as the
slate predicted for H1 (it adds no precached asset). Log `logs/merge1-G4-sw.log`.

**Gate result after the merge.** G1 `go build` + `go vet` exit 0; **G2 (Go) `EXIT_TEST=0`**, 14/14
packages `ok` including the new `internal/marketing`. Log `logs/merge1-G1-G2go.log`.

🛑 **G2 (Playwright) was NOT re-run at the merged HEAD, deliberately and with the reason stated.**
`git diff card/h1-campaign-codes-api..HEAD` over `backend/`, `tests/`, `*.html`, `*.js`, `*.json`
and `night-crew.toml` is **empty** — the merged tree and the card HEAD are identical in every
product file, differing only in four knowledge/log files. The card's suite (26 failed / 963 passed,
one summary block, `EXIT_PW=1`, log `logs/h1/G2-playwright.log`) therefore *is* this tree's result;
re-running would measure byte-identical product code at the cost of ~35 minutes of exclusive box
time inside a concurrent wave. Orchestrator call, stated rather than silent.

**Reds judged.** 23 of the card's 26 are exact matches in tonight's measured 24-red baseline. The
3 non-baseline reds (`inventory.spec.js:2186`, `recipes.spec.js:216`,
`states-inventory-nav.spec.js:297`) re-ran **green** confined at the card HEAD **twice and
independently** — once by the orchestrator (`logs/h1-isolation.log`, `3 passed`, `EXIT_PW=0`) and
once by G6 in its own isolated stack. One baseline red (`inventory.spec.js:2931`, B-453) was
**green** in the card's run, so the set flakes both ways under load. Classified **load flake, not
regression**; G6 independently looked for a causal path from 0083 or the `main.go` mounts to the
Inventory/Recipes frontend surfaces and found none.

**G6:** APPROVE-WITH-FINDINGS. Two MEDIUM findings **parked** to the operator (both came back
`verdict: park` at top severity from `night-crew decisions log`) — see `DECISIONS-NEEDED.md`.
Merged notwithstanding: the contracted scope is delivered and twice-verified, parking H1 would end
the night at zero cards (the other six cannot start without 0083 and the package seam), and
nothing deploys tonight.
