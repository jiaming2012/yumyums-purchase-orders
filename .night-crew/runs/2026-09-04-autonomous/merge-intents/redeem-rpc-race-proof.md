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
