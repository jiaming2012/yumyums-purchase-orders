# Merge intent — Card 1 · `supabase-schema-and-rls` (run 20260904)

Branch: `wo-supabase-schema-and-rls` (cut from `overnight-20260904` / dev @ c158311).
Card authority: slate-20260904 Card 1. Footprint: `supabase/` NEW + spike harness read-only.

## Shared files touched (everything outside my own new `supabase/` dir)

- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `supabase-schema-and-rls` card (PLANNED → DONE), in the same change set as the
  implementation, per the slate's mechanics rule. No other roadmap line moves.
- `.night-crew/runs/2026-09-04-autonomous/merge-intents/supabase-schema-and-rls.md`
  — this file.
- `.night-crew/runs/2026-09-04-autonomous/card1-*.log` — committed harness logs
  (red + green evidence) so morning triage can audit figures.
- `night-crew.toml` — **nothing here** (checked, deliberately NOT touched):
  `supabase/**` matches no `[e2e.seams]` key, so any later card touching it
  de-confines to the full suite — the safe default. This card is SQL + bash
  against the throwaway substrate; no Playwright spec can exercise it, and the
  slate's adapted gate is the standalone harness exit codes (B-345 precedent).
  Adding a seam key/token would be exactly the kind of new footprint concept the
  PARK note warns about.
- App code, `sw.js`, `package.json`, `backend/` — **nothing here** (untouched;
  no build-sw leg, no Go leg, no Playwright leg owed).
- `.night-crew/spikes/**` and `.night-crew/qa/spike-supabase/**` — **nothing
  here** (read-only per the footprint; the harness *runs* `env-up.sh`, `mintjwt`
  and `rtprobe` from the QA dir but modifies no byte of them).

## What must survive any merge

- The whole new `supabase/` tree: `migrations/20260904000100_qr_attribution_spine.sql`,
  `seed.sql`, `README.md`, `verify/*.sh`. Card 2 (`redeem-rpc-race-proof`) is
  serial after this card and extends this exact directory (its migration file
  sorts after mine; its harness reuses `verify/lib.sh`).
- The roadmap status flip for `supabase-schema-and-rls`.
- The committed `card1-*.log` evidence files and this merge-intent (the RF gate
  evidence lives here and in those logs).
- The seed fixture UUIDs in `supabase/seed.sql` — Card 2's harness and the RLS
  legs reference them by value (they are the contract between Card 1 and Card 2
  tonight). Do not renumber them in a conflict resolution.

## What is safe to drop

- Nothing in this branch is scratch. Temp files live only under the session
  scratchpad and are not committed. If a conflict forces a choice inside
  `.night-crew/runs/2026-09-04-autonomous/`, the logs are append-only evidence —
  keep both sides.

## Red-first

(Filled in as evidence lands. Greenfield form: the named-object assertion
harness must run RED against a bare substrate — objects don't exist — BEFORE the
migration exists, then GREEN after it applies, fresh and warm.)

- RED: pending
- GREEN (fresh + warm): pending
- RLS six legs: pending
- Realtime second subscriber: pending
