# HANDOFF — overnight run 20260906

Slate: `.night-crew/knowledge/reference/slate-20260906.md` (signed 2026-09-05).
Branch: `overnight-20260906` off `dev` @ `4facd5e`. Dispatch: SERIAL, 1 card (the
slate's whole content — Activities E/F refused by the per-goal spike gate, no stretch
tier). Conflict log: `.night-crew/knowledge/reference/conflicts-20260906.md` (1 clean
merge entry). Decisions journal: **none this run** — no gray area needed the resolver;
every mechanics call (publication-vs-fan-out, constraint form, harness design) was
pre-delegated to the night by the signed slate, and the §9/§19 re-read resolved inside
the existing taxonomy. **1 of 1 cards merged. Nothing parked.**

## Per-card outcomes

| # | Card | Outcome | Gates (final tree) | G6 |
|---|------|---------|--------------------|-----|
| 1 | requires-online-replication | ✅ merged `2eafa55` | RF both reds proven pre-change (branch-3 `override` where `requires-online` owed; f2 poison 10 redeem calls/0 landings) → both green; harness f2 green=0 + red-unflagged=1 (discriminator load-bearing), campaigns=0, regressions c2/c3/clock=0; machine conformance 18/18 (460 pairs/23 states) + strictness 9/9; marketing spec 30 passed ×3 (impl, G6, post-merge final tree); full e2e ×2 — 858/2/6 (impl) and 857/3/6 (G6 own run) — sole common red = base-proven DBL-05 (B-421), remaining reds shift between runs, green in isolation, on surfaces the diff doesn't touch; G4 precache idempotent at **43**; migrations apply clean twice, Activity A's byte-untouched | **APPROVE** (all legs independently re-run in own detached checkout) |

Velocity: 1/1 landed, zero fix rounds, zero regressions known at closeout.
Implementer ~44m + G6 ~37m + merge/closeout — inside the slate's 60–100m band.

## What landed, functionally

The decision-166 rider is discharged: a dedicated **campaigns pull replica** (same
shipped GAP-1 keyset mechanism as codes/offers; `buildPullUrl` expiry bound now
optional, bounded callers byte-identical) feeds `setCampaignPolicy` real data — a
`requires_online=true` campaign's code scanned offline now REFUSES the override even
for an entitlement holder ("can't verify — try again"), where pre-change the $40 code
was overridable exactly like the $2 one. Campaigns are replication-visible via
migration `20260906000100` (supabase_realtime publication membership + touch trigger —
chosen over codes-RESYNC fan-out because a campaign-only write emits no codes frame); a
post-flip downgrade re-delivers on next RESYNC (the leg that killed the embed
alternative). The **F-2 landing path** (migration `20260906000200`): `scan_attempts.code_id`
nullable + `token_hash` (SHA-256, raw token never persisted) + check constraint; an
unverified override skips `redeem()` and lands directly with
`offline_override=true, unverified_code=true, status='accepted'` — no new terminal
status (§9 already names offline overrides as accepted-pending-reconciliation; G6
independently concurred). The poison path is dead: pre-change 12 redeem retries /
0 landings / legit attempt stuck `pending`; now 0 redeem calls for the unverified
attempt, both attempts land. GAP-1's validation debt is paid — spike 03 re-executed
against the shipped guard, `validated:` line under the GAP-1 comeback in the goal
ledger. Both push belts and card 6's 460-pair strictness proof survive untouched.
**Activity B is now 4/4 — closed.**

## Next actions (operator, morning)

1. **`/nc-morning-triage`** — review the run branch, merge to dev. Roadmap line reads
   DONE "awaiting morning triage"; triage appends the merged-to-dev note per convention.
2. **Milestone remainder (6 cards, all gated on attended work):** Activity 0
   (`redemption-unknowns-spike`, `external-accounts-provision`) is strictly next and
   attended-by-design; Activities E/F unlock only after Activity 0 + `/nc-spike-open`
   sittings on their goals. The still-armed ATTENDED live-camera check (card 5, run
   20260905) also remains.
3. **Carried residuals (named, untouched by design):** GAP-2 (unstamped campaign write
   invisible to replicas — future provisioning-surface card; NOTE tonight's touch
   trigger mechanically overlaps its fix but its validation debt — spike 01 flip #1
   re-execution — is NOT claimed), GAP-3 (embedded-offer path stays policy-unknown —
   Activity E's payload question; do not count it as Q-KR1 coverage), B-423 (belt-2
   cross-session double-serve), B-429 (harness HTTP-failure injection gap).
4. **Small FYIs:** scratch test DB `hq_test_e2e_c1x` was created on :5434 during a
   parallel-stack repeat (test cluster only — drop at leisure); load-flake pattern in
   the full suite (FLD-03/04 one run, inventory:2616 + sync:3062 the other, all green
   in isolation, none diff-attributable) — backlog candidate if it recurs; two
   test-only hardenings rode along (F3-online real-probe race fix, tolerant
   `seedLocal` campaigns seeding); `submit-flow.js:122` discards the policy-source
   `stop()` handle (page-lifetime subscription, consistent with existing replicas —
   G6 noted, non-blocking).
5. **Worktrees:** card worktree `wt-20260906-c1` (merged branch
   `wo-requires-online-replication`) left in place per precedent, as is last run's
   `wt-20260905-c6`; G6's detached checkout was removed by the reviewer. The two
   launch-guard findings (June scheduling-app workspace, pinned `hq-main` checkout)
   were shown to the operator at launch and left untouched on their answer.
