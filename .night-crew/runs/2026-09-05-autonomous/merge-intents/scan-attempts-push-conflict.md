# Merge intent — Card 3 · `scan-attempts-push-conflict` (run 20260905)

Branch: `wo-scan-attempts-push-conflict` (cut from the Track B base @ 786be13 — includes
Card 1's marketing shell + seed, Card 7's arbitration machine + POST /api/v1/marketing/redeem,
and Card 2's marketing/sync pull replicas + widened vendor bundle). Card authority:
slate-20260905 Card 3. Footprint: rxdb replica (`marketing/sync/` — extended, no page files,
no sw.js, no backend).

## Shared files touched

- `marketing/sync/push-replication.js` (NEW) — the device-owned, push-only `scan_attempts`
  mechanism (design §4's opposite replication direction): local queue schema + `enqueueAttempt`
  + the push handler (pending → `POST /rpc/redeem` as the device JWT → land the attempt row →
  resolve the local row) + `startScanAttemptsReplica`. Dependency-injected like its siblings —
  zero imports, same file runs under the Node harness and the browser bundle. Cards 5/6 wire
  exactly this file's exports (surface below).
- `marketing/sync/harness/push-run.sh` + `marketing/sync/harness/push-harness.mjs` (NEW) —
  this card's standalone gate (B-345 precedent), a SIBLING of Card 2's `run.sh`/`harness.mjs`,
  deliberately not an edit to them: Card 2's landed gate stays byte-identical and re-runnable.
  Reuses Card 2's scaffolding patterns (lib.sh substrate discipline, JWT minting, the
  node_modules symlink, red/green mode contract) and drives Card 2's PRODUCTION
  `startCodesReplica` as the loser's display-data source.
- `marketing/sync/pull-replication.js`, `marketing/sync/replicas.js` — **read-only imports**
  from the new harness (the codes-side pull replica the loser's flip renders from). Zero edits.
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the `scan-attempts-push-conflict`
  card (PLANNED → DRAFTING (overnight-20260905)), the run's flip convention. No other line moves.
- `.night-crew/knowledge/spikes/activity-b-offline-first-replica/scan-attempts-push-conflict.md`
  — GAP-1's `validated:` line appended under `## Comebacks` (the card's owed validation run —
  the land-fails-after-redeem window), nothing else edited.
- `.night-crew/runs/2026-09-05-autonomous/merge-intents/scan-attempts-push-conflict.md` — this
  file (amended as evidence lands).
- `.night-crew/runs/2026-09-05-autonomous/card3-*.log` — committed whole gate logs, each ending
  in its `EXIT=` line.
- `marketing.html`, `index.html`, `sw.js`, `tests/*.spec.js`, `backend/`, `supabase/`,
  `night-crew.toml`, `package.json`, `vendor/` — **nothing here.** The push module is UNWIRED
  tonight (Cards 5/6 wire and state their precache moves); `supabase/` migrations are read-only
  inputs; the vendor bundle already exports `replicateRxCollection` (Card 2's widening) so no
  regen is needed for the browser wiring later.

## What must survive any merge — the API surface Cards 5/6 build against

`marketing/sync/push-replication.js` exports (renaming or re-shaping breaks Card 5's scanner):

- **`SCAN_ATTEMPTS_SCHEMA`** / **`SCAN_ATTEMPTS_COLLECTION`** (= `'scan_attempts'`) /
  **`scanAttemptsCollectionSpec()`** — the local collection. `scanAttemptsCollectionSpec()` is
  the `addCollections()` argument, same pattern as Card 2's `marketingCollectionSpec()`.
- **Queue shape (one local row per scan attempt):** server-taxonomy fields exactly as §4 —
  `id` (a REAL uuid, `crypto.randomUUID()` — spike build-fact, app-prefixed strings draw 400),
  `code_id`, `device_id`, `scanned_at`, `status` (`pending | accepted | rejected` — the §4
  taxonomy, UNCHANGED), `reason` (`already_used | expired | not_found | null`),
  `offline_override`, `override_by`, `unverified_code`, `pos_order_number`,
  `pos_business_date`, `redeemed_value` — **plus five LOCAL-ONLY fields that never reach the
  server row:** `burn_ok`/`burn_reason` (the persisted `redeem()` outcome — GAP-1 belt 1),
  `landed` (server attempt row confirmed inserted), `winner_device`/`winner_at` (the loser's
  render data, filled FROM THE CODES PULL REPLICA — never a scan_attempts read-back).
- **`enqueueAttempt(attemptsCollection, fields, opts?)`** — how the scanner records a scan.
  `fields` = `{code_id, device_id, offline_override?, override_by?, unverified_code?,
  pos_order_number?, redeemed_value?}`; returns `{doc, deduped}`. Insert-only and offline-safe:
  no network is touched; the push replica drains the queue when connectivity exists. **Dedupe
  rule:** at most one live (`pending`/`accepted`) attempt per `code_id` per device — a repeat
  enqueue answers the existing doc with `deduped: true` (this is also load-bearing for GAP-1
  belt 2's own-device test, see Engineering calls). A `rejected` attempt does NOT block a fresh
  scan. `opts` = `{now?, generateId?}` (clock/uuid injection).
- **`makePushHandler({restUrl, bearer, deviceId, fetchImpl, attemptsCollection,
  codesCollection, requestLog?, winnerWaitMs?, winnerPollMs?, now?})`** — the §6 handler.
  `codesCollection` is Card 2's codes-replica RxCollection (the ONLY place winner data and
  own-device arbitration reads come from). `requestLog` enumerates every outbound request
  (B-216).
- **`startScanAttemptsReplica({replicateRxCollection, collection, pushHandler,
  replicationIdentifier?, batchSize?, waitForLeadership?, retryTime?})`** — push-only
  `replicateRxCollection` wrapper; browser wiring passes the vendored bundle's
  `replicateRxCollection` (already exported since Card 2).
- **The UI contract:** the scanner observes the LOCAL attempt doc (`doc.$`). `pending` =
  queued; `accepted` = redeemed ✓; `rejected` + `reason='already_used'` + `winner_device`/
  `winner_at` = "already used at {winner_at} by {winner_device}". A transient landing failure
  can never flip an accepted-in-fact winner to rejected (GAP-1 — the two belts below).
- The GAP-1 `validated:` line in the spike ledger's `## Comebacks`; the roadmap flip; the
  `card3-*.log` evidence; this intent.

## What is safe to drop

- `marketing/sync/harness/node_modules` symlink (runtime-created, gitignored).
- Nothing else in this branch is scratch.

## Red-first

To be captured to `card3-red.log` BEFORE the production module exists in the tree (git history
is the chronology, Card 2's pattern): the naive handler shape (redeem → land → patch, no
persisted burn outcome, no own-device arbitration) with an INJECTED network failure on the
first `scan_attempts` landing insert AFTER a successful `redeem()` — the push retry re-runs
`redeem()`, the re-burn answers `already_used` to the device that in fact WON, and the naive
handler flips the winner's local row to `rejected/already_used`. That mis-flip failing the
"winner stays accepted" assertion (EXIT=1) is GAP-1's exact window, demonstrated. Then green:
the production module against the same injection.

## Engineering calls (recorded for the merge record; evidence lands below when gates run)

1. **GAP-1 — two belts, both shipped.** Belt 1 (the card's "persists the burn outcome locally
   before landing"): the handler patches `burn_ok`/`burn_reason` onto the local row IMMEDIATELY
   after `redeem()` answers, before the landing insert — a retry after a failed landing skips
   `redeem()` entirely (request log proves exactly one redeem per attempt) and retries only the
   landing. Belt 2 (the card's "treats `already_used` where codes.redeemed_by == own device as
   ACCEPTED"): covers the window belt 1 cannot — the redeem RESPONSE lost after the server
   committed the burn (or a device death between redeem and the burn patch). On
   `already_used`, the handler reads `redeemed_by` from the LOCAL codes pull replica (bounded
   await, `winnerWaitMs`); own device named ⇒ accepted.
2. **Retry mechanics:** handler throws on any unresolvable step (redeem HTTP error, landing
   network failure, winner unresolvable) → RxDB's own push retry (`retryTime`, injectable)
   re-runs it; the local row stays honestly `pending`. Landing idempotency: a 409 duplicate on
   our own uuid means a previous landing succeeded but its response was lost — treated as
   landed. Resolved rows (`status !== 'pending'`) re-push as no-ops, so the resolution patch
   cannot loop the handler (spike-proven bounded invocations, re-asserted by the harness).
3. **Winner-unresolvable = retry, never guess:** if `already_used` arrives and the codes
   replica cannot name a winner within `winnerWaitMs`, the handler throws and retries rather
   than flipping — marking rejected on a guess is GAP-1's harm; `pending` is the honest state.
4. **Dedupe at enqueue** (local persistence shape — this card's call): one live attempt per
   code per device makes belt 2 sound (an own-device `already_used` can only mean THIS
   attempt's lost burn, not a second scan racing our own earlier win) and gives Card 5
   double-scan handling for free.
5. **`pos_business_date` = UTC date of `scanned_at`** (spike's convention). Toast
   business-date semantics belong to §13's reconciliation card, not tonight.
