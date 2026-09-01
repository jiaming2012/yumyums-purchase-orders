# DECISIONS-NEEDED — run `20260901`

> **RESOLVED 2026-09-01 — recorded as ledger T-47.** Triage reviewed this file: **0 operator
> forks, 0 parks**, nothing to resolve. Run merged to `dev` at `8c2ea02`. The three
> triage-awareness items below (B-178, B-177, B-179) are filed in BACKLOG; the two operator
> ACTS (counterparty-notice SEND before the deploy, Toast key placement post-deploy) are carried
> forward in T-47. File kept as the analysis record.

## ✅ No operator forks. No parks. Nothing blocks the morning.

All 11 slated cards landed within the run's authority. None of the slate's three named PARK
conditions triggered:
- **Card 3 `toast-ingest-resurrection`** — the SFTP key rode an untracked bind-mounted file
  (the `.env.prod` pattern), exactly the recorded fix shape. No credential-home fork. Not parked.
- **Card 8 `app-slug-association`** — the association-home was a clean 1:N FK column
  (`checklist_templates.app_id → hq_apps`), no new operator-facing concept or app-behavior
  choice. Decided under standing authority, recorded in the merge-intent (E-KR4). Not parked.
- **Card 11 `deploy-hygiene-honesty`** — the version.json unification was byte-identical
  (same artifacts ship, generated once consistently). No release-flow shape change. Not parked.

## For triage awareness (NOT decisions the run needed — all filed in BACKLOG, summarized in HANDOFF)

These are surfaced so triage sees them, not because the run needs a ruling:
- **B-178** — the Spike C relay (pid 31802, → live `:5433`) contaminates the sync RLS test
  fixture; it reddened `internal/sync/TestJWTBridgeRLS` every leg tonight (environmental, not a
  card defect). Attended cleanup (stop the relay + clean its `spikec-*` rows) + a fixture-isolation
  fix (fold into `gate-rls-fixture-ownership`) would restore a green sync gate. Not urgent; no card
  depends on it.
- **B-177** — the parseable-path COGS-zone gap waived from Card 1; scope alongside the 0072
  changeover.
- **B-179** — workbox-build 7.3.0 (node_modules) vs 7.4.1 (lockfile); decide the authoritative
  version before relying on a fresh-clone `npm ci` to regenerate sw.js.

The two operator ACTS the morning needs (the counterparty-notice SEND before the deploy, and the
attended Toast key placement) are in HANDOFF.md's "DO THESE FIRST" — those are scoped acts, not
open decisions.
