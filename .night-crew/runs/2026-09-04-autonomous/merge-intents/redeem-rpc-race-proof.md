# Merge intent — Card 2 · `redeem-rpc-race-proof` (run 20260904)

Branch: `wo-redeem-rpc-race-proof` (cut from `overnight-20260904` @ a1a3401, AFTER
Card 1's merge — its `supabase/` migration, seed and verify harness are inherited).
Card authority: slate-20260904 Card 2. Footprint: supabase arbiter — extends Card 1's
`supabase/` tree with a later-sorting migration + a fourth verify harness; spike dir
read-only.

## Shared files touched (everything outside my own `supabase/` additions)

- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `redeem-rpc-race-proof` card (PLANNED → DRAFTING), same change set as the
  implementation, matching Card 1's flip convention. No other roadmap line moves.
- `.night-crew/knowledge/spikes/activity-a-attribution-spine/redeem-rpc-race-proof.md`
  — ONE appended `validated:` line under `## Comebacks`, closing GAP-1 (the card's
  explicit obligation: one re-validation run of the spike's successor harness against
  the BUILT migration, recorded in this card's sitting). Append-only; no existing
  line of that ledger changes.
- `.night-crew/runs/2026-09-04-autonomous/merge-intents/redeem-rpc-race-proof.md`
  — this file (amended as red-first evidence lands).
- `.night-crew/runs/2026-09-04-autonomous/card2-*.log` — committed whole logs
  (red + green evidence, each ending in its `EXIT=` line per Card 1's G6 finding).
- `supabase/verify/01-structure.sh` — Card 1's file, extended ADDITIVELY inside
  `assert_all` per the handed-down G6 finding: (a) a named NEGATIVE assertion that
  `scan_attempts` is push-only at the structural layer (no SELECT/ALL policy, no
  SELECT grant to client roles — mirroring the marketing_settings zero-policy
  negative), and (b) this card's own new objects asserted by name (the `redeem`
  function: signature, security definer, pinned search_path, execute grants
  positive for `authenticated` / negative for `anon`). No existing assertion is
  weakened or removed.
- `supabase/README.md` — Card 1's file: the Verifying list gains
  `04-redeem-race.sh`, plus a short `redeem()` section. Additive.
- `night-crew.toml` — **nothing here** (same reasoning as Card 1, deliberately NOT
  touched: `supabase/**` matches no `[e2e.seams]` key; the gate is the standalone
  harness exit codes, no Playwright spec can exercise SQL against the throwaway
  substrate).
- App code, `sw.js`, `package.json`, `backend/` — **nothing here** (untouched; no
  build-sw leg, no Go-suite leg, no Playwright leg owed).
- `.night-crew/spikes/**` and `.night-crew/qa/spike-supabase/**` — **nothing here**
  (read-only; the harness *runs* `env-up.sh` and `mintjwt` from the QA dir but
  modifies no byte of them; the spike's `redeem-fns.sql` was design input only —
  the shipped harness is self-contained under `supabase/verify/`).

## What must survive any merge

- `supabase/migrations/20260904000200_redeem_rpc.sql` — the atomic `redeem()` with
  the operator-signed v2 body (explicit `already_used`/`expired` arms,
  `coalesce(…, 'not_found')`). This is E-KR1's artifact.
- `supabase/verify/04-redeem-race.sh` — the repeatable race harness (20×2, red-analog
  mode) that is the point of the card.
- The additive `assert_all` extensions in `supabase/verify/01-structure.sh` (the
  push-only structural negative + the redeem-by-name assertions). If a conflict
  forces a choice inside `assert_all`, keep BOTH Card 1's assertions and these —
  they are disjoint blocks.
- The `validated:` GAP-1 line in the spike ledger's `## Comebacks` — it is the
  recorded closure of a comeback; dropping it re-opens GAP-1 silently.
- The roadmap status flip for `redeem-rpc-race-proof`, and the committed
  `card2-*.log` evidence files.
- Card 1's seed fixture UUIDs stay by-value (…0002 race/happy target, …0003 expired,
  …0004 pre-redeemed) — this card's harness references them literally.

## What is safe to drop

- Nothing in this branch is scratch. Temp files live only under the session
  scratchpad and are not committed. Inside
  `.night-crew/runs/2026-09-04-autonomous/`, logs are append-only evidence — in a
  conflict keep both sides.

## Red-first

(Filled as evidence lands. Greenfield form, per the card: the SAME shipped
assertion legs must red against the naive check-then-update analog — installed AS
`public.redeem`, identical signature — BEFORE the atomic v2 migration exists in
the tree, with observed double-wins in the log; then green against the built
migration.)

- RED (structural): **captured 2026-09-04T05:53Z, before any redeem migration
  existed in the tree.** `supabase/verify/01-structure.sh --assert-only` →
  **EXIT=1** (`card2-red-01-structure-assert.log`): every Card 1 assertion plus
  the new push-only structural negatives pass, then
  "· redeem(p_code uuid, p_device text) — Card 2's atomic arbiter, by name:
  `<absent>`" reds. The probe does not invert its meaning — red is a genuine
  named-object absence.
- RED (behavioral — the card's red-first): **captured 2026-09-04T05:57Z, same
  pre-migration tree.** `supabase/verify/04-redeem-race.sh --red-analog` →
  **EXIT=1** (`card2-red-04-race-naive.log`): the naive check-then-update body
  (TOCTOU window widened with `pg_sleep(0.4)`) installed AS `public.redeem`,
  then the shipped race leg observed **round 01: winners=2** — BOTH concurrent
  clients redeemed the same code — and the double-win assertion reds
  ("a round produced TWO winners — the single-use premise is falsified").
  The commit carrying this evidence contains NO migration SQL; the migration
  lands in the following commit, so the ordering is auditable from git alone.
  (First red-analog attempt died silently instead of red: with zero `false|`
  lines on a double-win, the loser-reason pipeline failed under
  `set -euo pipefail` before the assertion could speak. Fixed in the harness —
  `|| true` on the two extraction pipelines, with comments — so the double-win
  is now reported by the ASSERTION, loudly. No assertion was weakened; the
  detection got stronger.)
- GREEN (structural, fresh + warm): `supabase/verify/01-structure.sh` →
  **EXIT=0** (`card2-green-01-structure.log`). Both migrations apply clean over
  bare AND over their own output; the full Card 1 assertion set plus the new
  push-only structural negatives and the redeem-by-name block (signature,
  SECURITY DEFINER, `search_path=""`, `authenticated:true anon:false`) hold on
  both passes; the #5 config-survives-reapply leg still green. **The anon:false
  assertion caught a real hole on the first green attempt:** this substrate's
  `ALTER DEFAULT PRIVILEGES` hands `anon` an EXPLICIT execute grant at CREATE
  FUNCTION time, so the migration's original `revoke … from public` left
  `anon:true` (EXIT=1). Fixed by revoking from `public, anon` both — the
  assertion did exactly what the G6 finding wanted the structural layer to do.
- GREEN (the card's gate): `supabase/verify/04-redeem-race.sh` → **EXIT=0**
  (`card2-green-04-redeem-race.log`). Leg R: naive analog double-won on try 1
  (winners=2), analog dropped. Leg G: **20 rounds × 2 concurrent clients,
  exactly one winner every round, 0 double-wins, 0 zero-win rounds, every
  loser `already_used`**. Leg E: expired fixture …0003 refused both clients
  with `expired`, row stayed unredeemed. Leg A: pre-redeemed fixture …0004 →
  `already_used`. Leg N: unknown uuid → `(false, not_found)` — GAP-1 closed
  against the BUILT migration. Leg H: fixture …0002 happy path,
  `updated_at` ADVANCED. Leg P: device JWT `POST /rpc/redeem` → 200
  `{"ok":true}`, re-call 200 `already_used`, anonymous → 401 `42501`.
  Cleanup: 22 per-run codes seeded, 0 remaining.
- GREEN (Card 1 regression): `02-rls-six-legs.sh` → **EXIT=0**
  (`card2-green-02-rls.log`); `03-realtime-second-subscriber.sh` → **EXIT=0**
  (`card2-green-03-realtime.log`). Card 1's behavioral surface is unchanged by
  the new migration.

All runs printed the resolved substrate coordinates read-only before any write:
compose project `spike-supabase`, db container `b4d825247a2d…`, db host port
55342 / rest 55368 (Docker-assigned), role `supabase_admin` via `docker exec` —
NOT :5433, NOT :5434, no hosted project.
