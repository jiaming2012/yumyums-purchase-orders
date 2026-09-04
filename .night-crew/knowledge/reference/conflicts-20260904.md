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
