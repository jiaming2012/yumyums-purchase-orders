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

### Merge 2 — `card/h2-campaigns-tab-ui` → `overnight-20261002` (`f245259`), 2026-10-01 ~23:00

**Cards involved:** H2 only. H3a, H3b and H5 were in flight in their own worktrees; none had merged.

**Conflicted:** nothing. `git merge --no-ff --no-commit` reported *"Automatic merge went well"*.

**Files and hunks.** 20 files. Product: `marketing.html` (**two hunks, both inside the `#s2`
wrapper** — the section itself and one `.mc-note-bad` rule in `#s2`'s own `<style>`),
`marketing/campaigns.js` (new, 1173 lines), `tests/marketing-campaigns.spec.js` (new),
`tests/states-marketing-campaigns.spec.js` (new), `tests/marketing.spec.js` (narrow edit),
`sw.js`, `night-crew.toml` (roll-call comment only), `.gitignore` (one block). Plus the
merge-intent, its fix-round appendix, the gate logs and **18 committed state screenshots**.

**Intents read.** H2's merge-intent plus its appended fix-round section. It declares `#s2` and
only `#s2`; names `tests/marketing.spec.js` as a declared out-of-footprint edit with the
instruction *"keep Card 7's content, keep `#s2` asserting `#mc-root`/`#mc-locked`"*; and flags
that its own `#mc-f-item` is the surface making D-1 reachable. No second intent met it on any
file, so nothing had to be resolved **against** intent at this merge — the declarations were
read and checked, not arbitrated.

**Resolution taken.** None required by git. What needed *checking*: the card branched at
`94ad3b3`, so against the current tip it showed **phantom deletions** in
`.night-crew/knowledge/bugs.md` (−48), `BACKLOG.md` (−2) and `timings.log` (−7) — pure
staleness, 0 insertions and 0 commits by the card on any of them. H2's G6 warned explicitly
that **a squash-from-diff would have silently deleted B-459**, filed after the card branched.
This was a normal 3-way merge and it did not. Verified by assertion afterwards rather than
assumed: B-459 present, the one-sample-baseline method note present, the B-459 tally correction
present, **15** timings entries, the Merge-1 entry intact.
`roadmap.md` took both sides (H2 → **LANDED**).

**`task sw` at merged HEAD.** `node build-sw.js` ×2 → **49 files precached** (2990.0 KB) both
runs, byte-identical, reachability *"36 parsed, 63 refs, 0 outside"*, `git status` clean for
`sw.js` and `version.json`. Count **48 → 49**, exactly the one new module, as the slate
predicted. Log `logs/merge2-G4-sw.log`.

**Gate result after the merge.** G1 `go build` + `go vet` exit 0; **G2 (Go) `EXIT_TEST=0`**.
Log `logs/merge2-G1-G2go.log`.

🛑 **G2 (Playwright) was NOT re-run at the merged HEAD**, on the same stated basis as Merge 1:
`git diff card/h2-campaigns-tab-ui..HEAD` over `backend/`, `tests/`, `*.html`, `marketing/`,
`*.js`, `*.json` and `night-crew.toml` is **empty** — the merged tree is identical to the card
HEAD in every product file. The card's suite (28 failed / 6 skipped / 977 passed, one summary
block) therefore *is* this tree's result. Orchestrator call, stated rather than silent.

**Reds judged.** 23 of 28 are exact baseline matches; one baseline red (`inventory.spec.js:2931`)
went **green**. Of the 5 non-baseline: `tests/marketing.spec.js:141` was **the card's own and
expected** — it asserted `#s2` was a labelled placeholder, which this card makes false by
construction — and the card updated it narrowly, leaving the Scan-live and `#s3`/`#s4`
assertions byte-identical. The other four were cleared confined. 🛑 **`inventory.spec.js:2186`
was investigated properly rather than waved through**: H2's G6 reproduced it **1 green / 2 red**
in isolation at the card HEAD **and 2 red / 1 green on the base**, an identical rate with and
without the card — so it is conclusively not this card's, and it is now filed as **B-459**, a
diagnosed `ALL_ITEMS` race in pre-existing Inventory code. That control also produced tonight's
**method note**: a one-sample baseline cannot classify a high-rate race, and a single confined
green does not establish "flake" either.

**G6:** APPROVE-WITH-FINDINGS → **fix round → re-verified**. One P1 (the "Other" channel could
not be named — reproduced, and invisible to both specs because they used `page.fill()`), plus a
failed-write-reported-as-failed-read that wiped the detail view, plus non-durable screenshot
evidence. All three fixed red-first on the card branch before this merge; `[MC-06]` and `[MC-07]`
landed as permanent coverage.

### Merge 3 — `card/h3a-toast-orders-and-mirror` → `overnight-20261002` (`799a748`), 2026-10-01 ~23:30

**Cards involved:** H3a. H3b was being built **on this branch** concurrently; H5 was in flight.

**Conflicted:** nothing. Git auto-merged `.night-crew/knowledge/BACKLOG.md` and `roadmap.md`
line-locally and reported *"Automatic merge went well"*.

**Files and hunks.** 25 files. Product: `backend/internal/toast/orderdetails.go` (new, 497 lines)
+ `sync.go` (**49 insertions / 0 deletions**), `backend/internal/marketing/mirror.go` (new, 405
lines, a FILE added to H1's package with every identifier `Mirror…`-prefixed),
`backend/internal/db/migrations/0084_toast_orders_reconciliation.sql`,
`backend/internal/redemption/store.go` (one `ON CONFLICT` clause), plus one line of H1's
`zz_migration_down_test.go`. **`routes.go` NOT touched** — the mirror registers no HTTP route, so
a `routes.go` hunk from this card would have been wrong. `main.go`: exactly one call site.

**Intents read.** H3a's merge-intent plus its fix-round appendix. Load-bearing claims checked
rather than taken: `routes.go` untouched (confirmed), one `main.go` call site inside the existing
`schedulersDisabled` region (confirmed), and the `zz_migration_down_test.go` edit — which its
intent explains as a **cross-package defect fix**: the old final leg `db.MigrateTo(pool, 83)` left
the shared Go test DB pinned at 83, so under `-p 1` every alphabetically-later package
(`redemption`, `toast`) got a database with 0084's tables and B-424's index dropped. **Took this
side**, as the intent asks and as its G6 independently confirmed keeps every assertion.

**Resolution taken.** Nothing to arbitrate — no second intent met it on any file. The
phantom-deletion check was run again and is clean: B-459, the one-sample-baseline method note, 18
timings entries, **Card 2's entire frontend** (`marketing/campaigns.js` 1173 lines, `#s2` intact)
and B-424's `closed → toast-orders-and-mirror` disposition all survive. 🛑 One check of the
orchestrator's own was **wrong**: a grep for `closed → toast` returned 0 and briefly looked like a
missing disposition — the text carries a backtick (`closed → \`toast-orders-and-mirror\``) and the
pattern missed it. Card and G6 were both right; the check was not. Recorded because a false
negative at merge time is exactly how a real one would be dismissed.

**`task sw` at merged HEAD.** `node build-sw.js` ×2 → **49 files precached** both runs,
byte-identical, `git status` clean. Count **49 → 49**: this card adds no precached asset, as the
slate predicted. Log `logs/merge3-G4-sw.log`.

**Gate result after the merge.** G1 `go build` + `go vet` exit 0; **G2 (Go) `EXIT_TEST=0`**.
Log `logs/merge3-G1-G2go.log`. G2 (Playwright) not re-run: the card is backend-only, its own
full suite measured 26/963 with one summary block, and no frontend file moved in this merge.

**Reds judged.** 22 of 26 baseline matches, **two** baseline reds went green, 4 non-baseline — all
in B-459's `ALL_ITEMS` cluster. G6 pushed on the most suspicious (`inventory.spec.js:1469`, red
3/3 confined but green on the base) and got it to flip **3 failed / 1 passed** with
`--repeat-each=4` at one HEAD — which is B-459's own reproduction criterion, and its failure is
the auto-open reading a stale `ALL_ITEMS`, the same mechanism.

**G6:** APPROVE-WITH-FINDINGS → fix round → re-verified. Six findings; the two that mattered were
a **false justification comment** (the file claimed purchasing and recipes use America/Chicago;
they all read `users.DefaultTimezone` = America/New_York, decision 83 / migration 0072) and an
**unconfirmed zone** feeding H3b's ±30-minute matcher, now parked as **D-3**. The zone *value* is
untouched; only the comment was corrected, with a `TODO` naming the parked decision. Also fixed
red-first: a business-date/export-directory disagreement now warns; the upsert count reported rows
*parsed* rather than *landed*; and `parseCents` accepted a blank required money cell as a confident
`0` plus `NaN`/`Inf`/`1e3`. G6's F4 ("half the gate artifacts are missing") was a **wrong-SHA
finding** — all 16 exist at the tip, each with an `EXIT=` marker inside, verified independently at
merge; the card's real defect was citing paths without naming the carrying SHA, now fixed with a
committed artifact inventory.

