# Conflict log — run 20260906

Every merge to `overnight-20260906` gets an entry, clean or conflicted, so an empty log
never reads as "no conflicts" when it means "the logging never ran" (§15ad.66).

## Merge 1 — `wo-requires-online-replication` (Card 1, the night's only card)

- **Cards involved:** Card 1 only (first and only merge of the night; run branch at
  `89a1b39`, two timings-docs commits past the slate sign-off commit `4facd5e`).
- **Files/hunks:** clean merge, no conflicts — `marketing/sync/pull-replication.js`
  (optional expiry bound; bounded callers byte-identical), `marketing/sync/replicas.js`
  (`startCampaignsReplica` + `createCampaignPolicySource`),
  `marketing/sync/push-replication.js` (F-2 guard before `redeem()`, pure insertion —
  55 additions, 0 deletions, both GAP-1 belts byte-identical), `marketing/scan-page.js`
  + `marketing/submit-flow.js` (replica-fed policy seam wiring),
  `supabase/migrations/20260906000100_campaigns_replication.sql` +
  `20260906000200_scan_attempts_unverified_landing.sql` (NEW numbered files; Activity A's
  untouched), `marketing/sync/harness/` (campaigns + f2 harnesses, NEW),
  `tests/marketing.spec.js` (branch-3 e2e flipped to real replica data; 30 passing),
  `sw.js` (revision-hash-only, precache stays 43), roadmap flip, GAP-1 `validated:` line
  in the goal ledger, merge-intent + card1 evidence logs.
- **Intents read:** Card 1's merge-intent only (no other side exists on a first merge).
  Its three must-survive items — codes/offers expiry bounds, GAP-1's two push belts,
  card 6's 460-pair strictness proof — all verified intact by G6's independent runs.
- **Resolution:** none needed.
- **Gate result after merge:** merged tree content-identical to the reviewed card branch
  outside `.night-crew/` (`git diff wo-requires-online-replication overnight-20260906
  -- . ':(exclude).night-crew'` empty), so the branch evidence transfers: implementer
  full suite 858/2/6 (19.3m) and G6's independent full suite 857/3/6 (19.0m) — sole
  common red DBL-05 (B-421, base-proven); remaining reds shift between runs, green in
  isolation, on surfaces the diff doesn't touch (load flake). Harness legs f2 green=0 /
  red-unflagged=1, campaigns=0, regressions c2/c3/clock=0, machine conformance 18/18
  (460 pairs / 23 states) + strictness 9/9, `tests/marketing.spec.js` 30 passed (both
  sittings). G6 verdict: **APPROVE**. Post-merge re-run of `tests/marketing.spec.js` on
  the final tree: **30 passed (41.1s), exit 0**.
