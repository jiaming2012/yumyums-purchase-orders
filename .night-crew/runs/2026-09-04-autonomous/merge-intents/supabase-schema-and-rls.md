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

- RED: **captured 2026-09-04T05:35Z, before any migration file existed in the
  tree.** Fixture action `supabase/verify/reset-bare.sh` (EXIT=0,
  `card1-red-00-reset-bare.log`) dropped only the card's four objects from the
  throwaway substrate; probe `supabase/verify/01-structure.sh --assert-only`
  → **EXIT=1** (`card1-red-01-structure-assert.log`): first assertion
  "expected 4 tables (campaigns, codes, marketing_settings, scan_attempts),
  got: <empty>". The probe does not invert its meaning — red is a genuine
  assertion failure against genuinely absent objects. The harness commit
  carrying this evidence contains NO migration SQL; the migration lands in the
  following commit, so the ordering is auditable from git alone.
- GREEN (fresh + warm): `supabase/verify/01-structure.sh` → **EXIT=0**
  (`card1-green-01-structure.log`). Fresh apply over bare, full named
  assertion set (4 tables, F2 boolean, no raw-token column, unique
  `codes_token_hash_key` + `codes_updated_at_idx`, `scan_attempts_join_idx`,
  RLS ×4, 3 policies + zero on `marketing_settings`, zero client grants on
  `marketing_settings`, singleton constraint by name, 1 row @ 2000, 2 TEST
  campaigns, 5 TEST codes with expired/redeemed fixtures, `public.codes` in
  `supabase_realtime`); warm re-apply, same set again; plus the #5 leg:
  operator-set 2500 survives a third re-apply (then restored to 2000).
- RLS six legs: `supabase/verify/02-rls-six-legs.sh` → **EXIT=0**
  (`card1-green-02-rls.log`). Through PostgREST: (1) device read 200 with
  seeded code …0001; (2) anonymous 401/42501; (3) own-device insert 201;
  (4) spoofed device_id 403 (`new row violates row-level security`);
  (5) device SELECT on scan_attempts 403 (push-only holds); (6) server-side
  counts 1 own / 0 spoof (per-run device ids, so re-runs stay exact).
- Realtime second subscriber: `supabase/verify/03-realtime-second-subscriber.sh`
  → **EXIT=0** (`card1-green-03-realtime.log`). Authenticated-role rtprobe:
  JOIN-OK, READY subscribed=1 failed=0, no SYS-ERR; server-side
  `updated_at = now()` touch on code …0001 arrived as
  `RTP EVENT label=codes type=UPDATE table=public.codes id=c0000000-…-0001`
  inside the 20s window; rtprobe exit=0.

All three green legs printed the resolved substrate coordinates read-only
before any write: compose project `spike-supabase`, db container
`b4d825247a2d…`, db host port 55342 / rest 55368 / realtime 55371
(Docker-assigned), role `supabase_admin` via `docker exec` — NOT :5433,
NOT :5434, no hosted project.
