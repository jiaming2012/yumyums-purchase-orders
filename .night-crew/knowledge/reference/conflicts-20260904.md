# Conflict log — run 20260904

Every merge to `overnight-20260904` gets an entry, clean or conflicted, so an empty log
never reads as "no conflicts" when it means "the logging never ran" (§15ad.66).

## Merge 1 — `wo-supabase-schema-and-rls` (Card 1)

- **Cards involved:** Card 1 only (first merge of the night; run branch was at the slate
  sign-off commit `c158311`).
- **Files/hunks:** clean merge, no conflicts — `supabase/` (all-new: 2 migrations dir
  files, seed.sql, README, 5 verify scripts), run-dir evidence (merge-intent +
  5 `card1-*.log`), one roadmap card row flip (PLANNED → DRAFTING).
- **Intents read:** Card 1's merge-intent only (no other side exists on a first merge).
- **Resolution:** none needed.
- **Gate result after merge:** G6 PASS-WITH-NOTES on the identical tree content
  (independent re-run: reset-bare 0 → assert-only-on-bare 1 (RED) → 01-structure 0 →
  02-rls-six-legs 0 → 03-realtime 0). Notes carried to HANDOFF for triage: merge-intent
  says "PLANNED → DONE" but diff flips to DRAFTING (prose wrong, diff right); committed
  logs lack EXIT= lines (codes verified by re-run); 01-structure lacks a named negative
  for "no SELECT policy/grant on scan_attempts" (handed to Card 2 as a cheap add);
  anon-on-campaigns untested (exposure nil — anon fully revoked); seed.sql has no
  structural guard against a future hosted apply (flagged for Activity 0's
  `external-accounts-provision`).

## Merge 2 — `wo-redeem-rpc-race-proof` (Card 2)

- **Cards involved:** Card 2 onto Card 1's merged tree (serial by design — Card 2 extends
  Card 1's migration and harness).
- **Files/hunks:** clean merge, no conflicts — new migration
  (`20260904000200_redeem_rpc.sql`), `04-redeem-race.sh` harness, additive extensions to
  `01-structure.sh`/`lib.sh`/README, run-dir evidence (merge-intent + `card2-*.log`),
  one roadmap card flip, one append to the spike ledger's `## Comebacks` (GAP-1
  `validated:` line).
- **Intents read:** Card 2's merge-intent (Card 1's consulted for the fixture-UUID
  contract — honored, seed.sql untouched).
- **Resolution:** none needed.
- **Gate result after merge:** G6 PASS-WITH-NOTES on identical tree content. Independent
  re-run: 01/02/03 all 0, race red-analog 1 (winners=2 observed), race gate 0 (20/20
  rounds winners=1), warm assert 0; three mutation probes each red 1 and were restored.
  Notes to triage: `updated_at = now()` in the winning UPDATE is a named semantic delta
  from the signed v2 text (judged in-remit: enforcement predicate/atomicity/taxonomy
  byte-equivalent; it advances the replication checkpoint Card 1's schema indexes);
  `reset_bare` not extended to drop `public.redeem` (stale-function false-green risk for
  future red probes); red-analog runs leave their per-run race code until a full
  01-structure restore (disclosed in harness header).
