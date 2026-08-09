# DECISIONS-NEEDED — run `20260810`

## No parks. No operator forks.

Both cards decided and filed within the run's authority. No card hit a PARK trigger (no new
`night-crew.toml` key/token, no new terminal status, no substrate schema change beyond decision
111's rows, no scope widening beyond decisions 105/126, no touch of the still-open My Checklists
read path T-43(b)). The one engineer-level call — Card 2's gate-harness form — was decided
(standalone spike-style spec gated on its own exit, form (a), B-345-aligned) and recorded in the
merge-intent, not escalated.

## Operator-awareness (not a decision — a heads-up)

- **The milestone close bar is now runnable and meaningful.** With both cards landed, the RxDB sync
  capability runs persistently in the dev environment and is proven usable in the app. The remaining
  attended act, `dev-complete-attestation` (decision 161), is the operator's own: `task sync:dev:up`,
  open `workflows.html` in `dev:tailscale`, see a field sync in the app, record the ledger line.
  🛑 That run is the ONLY sanctioned place the dev HQ's real `:5433` coordinate is touched, by the
  operator knowingly (`HQ_SYNC_DEV_ALLOW_5433=1`); the unattended run never went near it.

## Advisory findings (candidates, not blockers — triage/planner call per T-10/T-12)

1. **Latent hardening — bare `npx playwright test` in the spike scripts.** Card 2's setup burned two
   could-not-run rounds because a fresh worktree's `node_modules/.bin/playwright` symlink was missing
   (npm's "Exit handler never called" skipped bin-linking), so bare `npx playwright test` fell through
   to a **foreign PATH `playwright`** (`/Users/jamal/miniconda3/bin/playwright`, no `test` subcommand).
   Card 2's own harness is now hardened (`node node_modules/@playwright/test/cli.js` + a preflight
   assertion), but `spike-f-browser-live.sh`, `sync-dev-proof.sh`, and the other spike scripts still
   use bare `npx playwright test` and carry the same vulnerability (they got lucky with a populated
   `.bin`). **Backlog candidate:** apply Card 2's deterministic-CLI-resolution pattern across the
   spike scripts.

2. **G6 cosmetic nits (Card 2), non-blocking, no fix owed:** (a) the merge-intent describes the
   spike-f→repo promotion as a "rename" when it is actually a *strengthening* — spike-f keyed its red
   on a different assertion (`absent`); the card collapsed to ONE always-`served` test achieving the
   red purely by withholding the carrier, which is what done_when demands; (b) throwaway
   scratch-container dev creds are inline in `sync-app-proof.sh` (banner-documented; not a real secret).

3. **G6 must-fix (Card 1) — already applied `167bc7e`:** FDW SQL comment named the `:5433` override
   `HQ_FDW_ALLOW_5433`; the guard reads `HQ_SYNC_DEV_ALLOW_5433`. Fixed (fails safe either way).
   Noted here only so triage sees it was caught and closed within the run.
