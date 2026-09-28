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

## Merge 3 — `wo-backlog-machine-migration` (Card 3)

- **Cards involved:** Card 3 onto the Track-A tree. Shared-surface note: Card 3 edits
  `COMMANDS.md` (triage step 6, the §4.5 backlog gate); Card 4 will edit the same file's
  closeout moment — Card 4 was cut AFTER this merge, so it inherits this text (serial
  dispatch removes the collision the slate planned for).
- **Files/hunks:** clean merge, no conflicts — `BACKLOG.md` (207→209 entries, canonical
  form), `COMMANDS.md` (§4.5 gate stanza in the triage step), one roadmap card flip,
  run-dir evidence (merge-intent with red baseline + preservation proof,
  `card3-*.log`, `card3-preservation-proof.py`, `card3-reshape.py`).
- **Intents read:** Card 3's merge-intent (its COMMANDS.md merge note — "keep both
  additions" — recorded for Card 4's merge).
- **Resolution:** none needed.
- **Gate result after merge:** G6 PASS-WITH-NOTES on identical tree content. Independent
  re-run: check exit 0 "valid — 209 entries", list 209==209 by the reviewer's own count;
  reviewer's OWN preservation proof (3 tokenizations, case-sensitive) lost = 0, plus a
  per-entry containment check — 0 per-entry loss, nothing invented, B-77 the only
  (declared) fold; 15 sampled entries all meaning-preserved. Notes to triage: stale HTML
  comment after line 147 still claims the document is invalid (now false — append a
  correction on next touch, preservation kept the tokens); B-76/B-92 head `new` while
  carrying `✅ RESOLVED` continuation blocks (faithful to the before-document; triage
  should reconcile the heads); B-145 `done —` and B-412 `promoted →` were implementer
  judgment calls, verified as carrying the residuals verbatim.

## Merge 4 — `wo-team-records-from-hand-runs` (Card 4)

- **Cards involved:** Card 4 onto the tree carrying Cards 1–3. Shared surface with
  Card 3: `COMMANDS.md` — Card 4 adds a closeout-scorecard stanza under step 5; Card 3's
  §4.5 triage stanza under step 6 verified byte-untouched by Card 4's G6. Serial
  dispatch removed the collision the slate planned for.
- **Files/hunks:** clean merge, no conflicts — `scorecard/TEMPLATE.md` (new, .md-inert
  by design: a .jsonl template would render as a real run), `COMMANDS.md` (one +26-line
  stanza), one roadmap card flip, run-dir evidence (merge-intent + 3 `card4-*.log`).
- **Intents read:** Card 4's merge-intent (its COMMANDS.md note honors Card 3's "keep
  both additions").
- **Resolution:** none needed.
- **Gate result after merge:** G6 PASS-WITH-NOTES on identical tree content. Independent
  re-run: inert render "No runs to show." exit 0; validation record rebuilt from the
  template's own instructions with no improvised field, 4/4 roles record-backed, exit 0;
  closing-artifact negative checked per-commit (none, tip or history);
  `milestones.jsonl` blob-identical to base. Notes to triage: template's
  filename-stem==run_id rule was deviated from by the validation file (CLI reads run_id
  from content — safer for a transient, moot at a real closeout); team-line points split
  has no rounding guidance for uneven totals (two closeouts could split differently).
- **NOT emitted by the card (deliberate):** `scorecard/20260904.jsonl` — it is a closing
  artifact in the run-evidence oracle; the run's closeout emits it (first consumer
  proves the producer), immediately after this merge.
