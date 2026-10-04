# DECISIONS-NEEDED — overnight run `20261003`

> **RESOLVED 2026-10-04 — recorded as `.night-crew/knowledge/ledger.md` T-64, decision 203.**
> D-1 → **ship as built.** The operator has not used QR campaigns yet, so there is no scan history
> whose totals matter, and both reports count scans from a chosen start date forward. The live scan
> table was NOT counted; the basis is the operator's statement. The card stays parked on its branch
> until its full browser-suite run and merge on the next slate. Kept as the analysis record.

## D-1 · OPEN · Card 4 `atomic-scan-dedupe` is PARKED: its migration deletes past scan rows on first deploy

**As the owner, I want to choose what happens to scan rows already in the table when the
double-tap fix goes live, so that my past campaign scan counts only change if I meant them to.**

**What raised it.** The card is built, and its independent review approved the mechanism
(12 simultaneous taps → 1 row, five runs out of five; a repeat tap still lands the customer on
the signup page). The review's one P2: to build its unique index, migration
`0087_qr_scans_dedupe_bucket.sql` (lines 51–63) first cleans up rows the old statement let
through, and that cleanup is permanent — the Down does not bring anything back.

**What the migration does today, on the first deploy:**
- repeat taps from the same phone on the same code inside one ten-minute bucket, with no signup
  attached → **deleted** (the earliest is kept);
- repeat taps that DID lead to a signup → kept, phone fingerprint (`ip_hash`) blanked, so they
  stay in the count as anonymous scans;
- anonymous scans (no fingerprint) and everything else → untouched.
On the reviewer's 19-row seed, 13 rows survived. **How many production rows this touches is
unmeasured** — the run may not read the production cluster.

**Why the run did not decide it.** The signed slate scope does not mention deleting existing
rows; past scan counts are read on the Campaigns and BI report screens
(`campaigns.go:366`, `stats.go:410`). Routed through `night-crew decisions log` → **`verdict:
park`** (medium severity, citation floor unmet; record in `.night-crew/knowledge/decisions/20261003.jsonl`).

**The choice:**
1. **Ship as built** — past counts drop to what the ten-minute rule always meant them to be.
   Every deleted row is one the rule would have refused; no signup-linked row is lost.
2. **Keep every row** — change the cleanup to blank the fingerprint on all surplus rows instead of
   deleting the unclaimed ones. Past counts stay exactly as they read today (still over-counted
   for the past); only new taps are deduped. A small fix round on the card.
3. **Measure first** — count the affected production rows (one read-only query at triage), then
   pick 1 or 2.

**State:** branch `card/i4-atomic-scan-dedupe` @ `8251ac9` (code unchanged since the reviewed `c1fc1c4`; `8251ac9` adds gate logs only; cut from Card 3's tip), worktree
preserved at `/home/jcole/projects/hq-worktrees/i4-atomic-scan-dedupe`, NOT merged to the run
branch. Review verdict APPROVE-WITH-FINDINGS (0 P1, 1 P2 = this, 3 P3). Its roadmap line reads
`LANDED` only on its own branch; on the run branch card I4 stays `PLANNED`.

**Gates on the parked branch:** G1 exit 0; full Go suite `EXIT_TEST=0`, 454 pass / 0 fail / 3 skip,
15 packages; `sw.js` 51, unchanged. **The full Playwright suite was NOT RUN** on this branch (the
orchestrator pulled it off the suite lock once the card parked) — it is owed at the re-gate,
whichever option is chosen. Without SOME cleanup step the index cannot be built on a table holding
duplicates (`red-migration-without-cleanup.log`), so "no cleanup" is not an option.
