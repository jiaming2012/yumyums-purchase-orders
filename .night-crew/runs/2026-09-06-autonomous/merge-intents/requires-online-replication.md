# Merge intent — Card 1 · `requires-online-replication` (run 20260906)

Branch: `wo-requires-online-replication` (cut from `overnight-20260906` @ 4facd5e —
carries all of Activity A + Activity B cards 2/3/4 + Activity C cards 5/6 + Card 7,
merged to dev at triage 20260905). Card authority: slate-20260906 (the signed single
card; decision-166 rider, ledger T-53, scoped by decision 167). Spike authority:
`.night-crew/knowledge/spikes/activity-b-offline-first-replica/requires-online-replication.md`
(+ `.extraction.md`) — 3 spikes green 2026-09-05, review signed, 2 corrections.
Mechanism DECIDED by the spike with evidence: a dedicated **campaigns pull replica**
on the shipped `pull-replication.js` mechanism; the codes-embed alternative is CLOSED
(post-flip staleness measured, `max(codes.updated_at)` byte-identical across the
campaign write). Design authority: docs/qr-offline-redemption-handoff.md §7/§8/§9,
§19 F2/F4 — §9/§19 re-read verdict recorded below (build call 1).

## Shared files touched

Each line: a file outside this card's own new files, and why it moves.

- `marketing/sync/pull-replication.js` (EDIT — Card 2's module, ADDITIVE) —
  `buildPullUrl()` / `makePullHandler()`'s expiry bound becomes **OPTIONAL, never
  removed**: a null/absent `windowBound` omits the `expires_at=gt.<windowIso>`
  fragment (campaigns has no `expires_at`; the shipped handler answers HTTP 400 on
  that table today — spike build-fact 1). Callers that pass a bound are
  byte-identical; the GAP-1 keyset checkpoint is untouched.
- `marketing/sync/replicas.js` (EDIT — Card 2's module, ADDITIVE) — the campaigns
  replica: `CAMPAIGNS_COLLECTION` + minimal `CAMPAIGNS_REPLICA_SCHEMA`
  (id, requires_online, updated_at — §10 minimal, no name/face_value),
  `marketingCollectionSpec()` gains the campaigns collection,
  `startCampaignsReplica()` (no expiry bound, same keyset mechanism), and
  `createCampaignPolicySource()` — the sync-readable policy lookup
  (RxDB reactive-query-fed Map; `policyFor(campaignId)` → `{requiresOnline}` | null).
- `marketing/sync/push-replication.js` (EDIT — Card 3's module) — the **F-2 guard,
  placed BEFORE `redeem()`**: an `unverified_code=true` pending attempt skips the
  burn entirely (no code row exists to burn — `/rpc/redeem` refuses it first,
  HTTP 400 `22P02`, spike-measured) and lands directly on the distinct path
  (`code_id: null, token_hash: <the 64-hex the local row carries as code_id>`,
  `status: 'accepted'`, flags carried). GAP-1's two belts and belt-2's
  block-on-unknown-winner are untouched — the guard diverts ONLY unverified rows,
  before the belts' code path begins.
- `marketing/scan-page.js` (EDIT — Card 5's file, ADDITIVE) — `startSync()` starts
  the campaigns replica beside codes/offers; `resync()` nudges it too (the
  reachability-driven RESYNC fan-out that exists today reaches campaigns the same
  way it reaches codes/offers).
- `marketing/submit-flow.js` (EDIT — Card 6's file) — the policy seam's DEFAULT:
  `CAMPAIGN_POLICY` starts as `createCampaignPolicySource(collections.campaigns)
  .policyFor` instead of the shipped `null` literal. `setCampaignPolicy(fn)` stays
  the injection seam and still overrides. New debug/test read:
  `window.MarketingSubmit.campaignPolicyFor(id)`. Unknown campaign → null → false:
  the ratified unknown→false default survives for genuinely-unknown codes.
- `supabase/migrations/20260906000100_campaigns_replication.sql` (NEW numbered file —
  Activity A's migrations untouched) — campaigns visible to replication:
  `supabase_realtime` publication membership (guarded, the §7.1 pattern) + a touch
  trigger stamping `updated_at` on every UPDATE. Decision recorded below (build call 2).
- `supabase/migrations/20260906000200_scan_attempts_unverified_landing.sql` (NEW
  numbered file) — the F-2 landing path: `scan_attempts.code_id` drops NOT NULL,
  `token_hash text` column added, check constraint `scan_attempts_names_a_code`
  (`code_id is not null or (unverified_code and token_hash is not null)`).
- `marketing/sync/harness/campaigns-run.sh` + `campaigns-harness.mjs` (NEW) — the
  card's standalone gate, replication half: optional bound (campaigns HTTP 200,
  codes URL still bounded), initial sync, unstamped-write flip re-delivered on next
  RESYNC (trigger under test), no-polling leg, publication membership enumerated,
  and the spike-02 four-run policy matrix on the PRODUCTION policy source
  (mode 'throw' — zero undeclared pairs).
- `marketing/sync/harness/f2-run.sh` + `f2-harness.mjs` (NEW) — the guard half AND
  the owed GAP-1 validation run: spike 03 re-executed against the SHIPPED guard
  (no wrapper), fresh live code minted per leg (build-fact 5). Red mode
  `red-unflagged` demonstrates the assertion catches the defect class.
- `tests/marketing.spec.js` (EDIT) — the branch-3 e2e **flips from seam-injected to
  real data**: seeds the HIGH fixture (code …0005 / campaign …0002,
  `requires_online=true` — the committed seed literals) into the local replicas and
  asserts the refusal with NO `setCampaignPolicy` injection. `seedLocal` gains a
  campaigns leg (tolerant of the collection's absence so the red-first run observes
  the actual overridable-today behavior, not a seeding crash).
- `sw.js` (REGENERATED after code commits — build-sw.js reads git HEAD) — module
  edits only; **expected precache count stays 43** (no file added or removed). If it
  moves, that is B-37 — investigate, don't ship.
- `.night-crew/knowledge/spikes/activity-b-offline-first-replica/requires-online-replication.md`
  (EDIT — append-only) — the `validated:` sub-bullet under the GAP-1 comeback
  (the entry that owes the spike-03 re-execution).
- `.night-crew/knowledge/roadmap.md` — the card's own line flips PLANNED → DONE
  (overnight-20260906), same change set as the closing work. No other line moves.
- This file (amended as evidence lands); `card1-*.log` gate logs beside it.
- **Nothing here:** `backend/**` (no Go change — no new endpoint, the browser reads
  the replica), `marketing/submit-machine.js` (zero machine changes — the spike
  proved zero new (state,event) pairs with replica-fed policy), `night-crew.toml`
  (the `supabase/` seam stays deliberately undeclared → full-suite gate, the B-37
  safe direction), `vendor/`, `lib/`, `sync-rxdb/`, `index.html`, other specs.

## What must survive any merge

1. **The codes/offers replicas' expiry bounds** — `startCodesReplica` /
   `startOffersReplica` still pass their windowBounds and their pull URLs still
   carry `expires_at=gt.` (harness run.sh green legs + the campaigns harness's
   bounded-URL assertion). The bound became optional, NEVER removed.
2. **GAP-1's two push belts** (burn-persist before landing; own-device
   `already_used` arbitrated as accepted) and **belt-2's block-on-unknown-winner**
   (a design property, build-fact 5 — not a defect). The F-2 guard diverts only
   `unverified_code=true` rows and sits before the belts' path; push-run.sh green
   unchanged.
3. **Card 6's 460-pair strictness proof** — the machine is untouched; the policy
   swap feeds the same RESOLVED{requiresOnline} event. The campaigns harness
   re-runs the four-run matrix in mode 'throw' (zero undeclared pairs, actor alive).
4. **The §8 refusal on real data** (the done_when): the flipped branch-3 e2e +
   the harness policy leg. `requires_online=true` → no override, for anyone.
5. **The F-2 landing shape**: `code_id null` + `token_hash` + check constraint;
   guard BEFORE redeem(); the audit-flagged attempt REACHES the server
   (skip-until-arbitration is REJECTED — spike leg c1 — and must not come back).
6. Both NEW numbered migrations (Activity A's two files byte-identical).
7. The `validated:` ledger line, the roadmap flip, the red/green gate logs, this intent.

## What is safe to drop

- `campaignPolicyFor` debug surface if a later card replaces the policy plumbing
  wholesale.
- Harness timing constants (retryTime/wait windows) — any values keep the
  assertions correct.
- Nothing else on this branch is scratch.

## Red-first

Both reds demonstrated on the PRE-CHANGE tree (product files untouched; the new
test/harness files exercise the shipped modules), captured to `card1-red.log`
and committed before any production change:

- (a) branch-3 refusal e2e, real-data form: on the pre-change tree the $40
  `requires_online=true` code's offline gate offers the OVERRIDE branch to an
  entitlement holder — `Expected: "requires-online", Received: "override"` — i.e.
  overridable exactly like the $2 one (branch-2 stays green beside it).
- (b) F-2 poison: `f2-run.sh` (green mode) against the pre-change tree — the
  shipped handler redeem-first-400s on the 64-hex `code_id`, the legitimate
  attempt behind it never lands (stays `pending`, zero landing requests), EXIT≠0.

## The card's recorded build calls

1. **§9/§19 re-read — the unverified-override landing status is `accepted` +
   flags; NO new terminal status, nothing weakened.** §9 names offline overrides
   as "the only **accepted** attempts that can still turn into a real
   double-redeem" and orders them reconciled FIRST — i.e. the taxonomy already
   expects an accepted-at-the-window row distinguished by `offline_override=true`;
   `unverified_code=true` + `code_id IS NULL` + `token_hash` make the F-2 row
   strictly more distinguishable, not a new status. §19 F2's "queued for sync
   arbitration … reconcile as a lost/invalid override" and F4's
   status-reflects-the-loss are the SERVER-side reconciliation flow (Activity D's
   arbitration surface, keyed on exactly these flags) — they happen after landing,
   not at it. The `requires_online=true` refusal path is untouched (unknown codes
   have no campaign row by construction; the refusal arms for replicated
   campaigns). PARK conditions not met.
2. **campaigns visibility: publication membership + touch trigger, NOT a codes
   RESYNC fan-out.** Measured basis (spike build-facts 2/3): a campaign-only
   write emits no codes frame — fanning the codes channel into the campaigns
   replica delivers a flip only on the next reconnect/codes event, i.e. possibly
   never while codes sit still; publication membership makes campaigns
   self-announcing through the SAME `wireRealtimeResync` mechanism codes already
   uses, and the touch trigger guarantees the stamped `updated_at` the keyset
   checkpoint requires (a plain `update campaigns set requires_online=…` is
   otherwise invisible to EVERY checkpointed replica). The reachability-driven
   `resync()` additionally nudges campaigns (scan-page wiring) — belt, not the
   mechanism. NOTE: the trigger overlaps GAP-2's named fix mechanically, but
   GAP-2 stays CARRIED as recorded — its validation debt (re-executing spike 01's
   flip #1 leg, which must stop being blind) belongs to the provisioning-surface
   card and is NOT claimed here.
3. **Constraint form:** `check (code_id is not null or (unverified_code and
   token_hash is not null))` — every attempt either names a real code or is an
   audit-flagged unverified override carrying the hash it actually has; a row with
   neither is structurally impossible. `code_id` stays `uuid` (a 64-hex hash can
   never masquerade as a code id server-side).
4. **Guard discriminator = `unverified_code`** (the enqueue's F2 flag,
   submit-flow.js:240's engineering call: an unknown code's local `code_id` IS its
   token_hash). The red-unflagged harness mode demonstrates the poison recurs for
   a row that loses the flag — the discriminator is load-bearing and tested.

## Gate evidence (run in this worktree; logs committed beside this file)

- **RF** — `card1-red.log` (committed `e83a0f6`, BEFORE any production file):
  (a) the flipped branch-3 e2e reds `Expected: "requires-online" /
  Received: "override"` — the $40 code overridable offline exactly like the
  $2 one (branches 1–2 green beside it), **EXIT=1**; (b) f2-run.sh green mode
  reds with the head-of-line poison enumerated — **10 redeem calls for the
  64-hex row, 0 for the legit code behind it, both stuck `pending`, +0 server
  rows, 6 named disagreements, EXIT=1**.
- **THE CARD'S OWN GATES** (spike-supabase substrate, reconcile mode):
  - `card1-f2-harness.log`: f2-run.sh **green EXIT=0** — THE owed GAP-1
    validation run (spike 03 vs the shipped guard, wrapper-free): 0 redeem
    calls for the unverified attempt, exactly 1 for the legitimate one, both
    landed, +2 server rows, audit row
    `(null)|<64-hex>|true|true|accepted`; **red-unflagged EXIT=1** —
    the discriminator stripped, the poison recurs, all 6 assertions red
    (the assertion catches the defect class).
  - `card1-campaigns-harness.log`: campaigns-run.sh **EXIT=0** — campaigns
    pull unbounded HTTP 200 / codes URL still `expires_at=gt.`-bounded; both
    seeded flags landed; the four-run policy matrix on the PRODUCTION source
    (mode 'throw'): replica+HIGH → `overrideAvailable=false` →
    `blockedOffline`, replica+LOW → `overrideConfirm`, none+HIGH/LOW → both
    overridable, **alive=true trips=0 in all four runs**; the DOWNGRADE flip
    via plain UPDATE: `updated_at` trigger-stamped (moved=true), 2s no-nudge
    → still stale (no polling), next RESYNC → re-delivered and the policy
    source follows. Publication membership + trigger enumerated by name in
    the shell half.
- **Regression on landed gates**: `card1-regression-c2.log` run.sh **EXIT=0**
  (bounded keyset replicas, GAP-1 boundary walked); `card1-regression-c3.log`
  push-run.sh **EXIT=0** (exactly-one-winner, both GAP-1 windows, write-only
  RLS); clock-run.sh **EXIT=0** (uncommitted spot-check); Activity A verify
  01-structure.sh **EXIT=0** — all four migrations apply clean twice
  (fresh + warm), every structural claim by name.
- **Machine**: `node tests/machine/run-conformance.mjs` **EXIT=0** — ALL 18
  SEQUENCES HELD, **460 declared pairs across 23 states** (the strictness
  proof unchanged); `strictness.mjs` **EXIT=0** (9/9).
- **Marketing spec**: `npx playwright test tests/marketing.spec.js
  --retries=0` → **30 passed**, including the flipped branch-3 (5/5 on a
  parallel-stack repeat). One test-only hardening: F3-online forces online
  through the real probe (the ONLINE direction of the 9c6f04e race — 3/3
  green in isolation pre-fix, red once under whole-file load; assertions
  unchanged).
- **E2E** — `card1-e2e.log` (FULL suite — `supabase/` is an undeclared seam,
  deliberate de-confine, B-37 safe direction; :5434 via task test:db:up,
  `--retries=0`): **858 passed / 2 failed / 6 skipped (19.3m), EXIT=1.**
  Verdicts: `workflows DBL-05` — the armed baseline red (B-421,
  base-proven, expected); `workflows FLD-03/FLD-04` (sub-step attribution
  before divider) — **3/3 green in isolation on the same tree** (targeted
  repeat) and nothing in this card's diff touches workflows → the known
  load-shaped flake class in exactly the spec the seam notes call
  load-sensitive. **Zero unexplained reds; zero diff-attributable reds.**
- **SW** — `card1-g4.log`: `node build-sw.js` **EXIT=0, 43 files precached
  (2797.3 KB)** — count unchanged (module edits only); reachability 30 files
  parsed / 47 references resolved / 0 outside the precache.
