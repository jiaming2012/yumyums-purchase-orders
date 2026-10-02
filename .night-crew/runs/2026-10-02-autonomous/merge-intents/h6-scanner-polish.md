# Merge intent — card `h6-scanner-polish` (Card 7, roadmap H6, run `20261002`)

Branch `card/h6-scanner-polish`, cut from `overnight-20261002` at `28984ab`
(Cards 1, 2, 3 and 6 already merged). Serial tail of Activity H: four scanner
findings — **B-446**, **B-447**, **B-440**, **B-436** (decision 191).

Spec: `docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md` §6 row H6,
§7. Goal ledger: `.night-crew/knowledge/spikes/activity-h-designed-tabs/scanner-polish.md`
(2/2 passed; both build-facts binding).

---

## Shared files touched, and why each

| File | Why |
|---|---|
| `marketing/sync/replicas.js` | **B-447**: `name` joins `CAMPAIGNS_SELECT` + `CAMPAIGNS_REPLICA_SCHEMA` (→ **version 1** + a migration strategy, the `SCAN_ATTEMPTS_SCHEMA` precedent), and the policy source's Map now carries the name so `nameFor()` can answer synchronously at render time. |
| `marketing/sync/push-replication.js` | **B-440**: the F-2 divert predicate at `:303` becomes `unverified_code && offline_override`, matching the constraint run `20260906-2` tightened; the mismatched shape is quarantined loudly instead of being handed to a 400. |
| `marketing/submit-flow.js` | **B-446**: the `requires_online` refusal renders at scan-resolve (same copy, earlier) in place of the order-#/Submit affordances; the post-submit guard stays as the belt. **B-436**: `policyFor`'s `!CAMPAIGN_POLICY` arm fails CLOSED. |
| `marketing/scan-page.js` | **B-447**: the offer card renders the campaign name + the code's last four; the campaigns pull's `replicationIdentifier` is bumped so an already-synced device re-pulls once and gets names. |
| `marketing/sync/harness/f2-harness.mjs`, `f2-run.sh` | **B-440**'s poison-row case — a new `poison-mismatch` mode asserting the guard and the constraint AGREE. |
| `marketing/sync/harness/campaigns-harness.mjs` | **B-436**: leg 3's negative assertion flips (decision 191), plus the migration plugin the campaigns schema bump requires. |
| `marketing/sync/harness/harness.mjs`, `clock-harness.mjs`, `recovery-clear-harness.mjs` | one `addRxPlugin(RxDBMigrationSchemaPlugin)` each — mechanical, forced by the campaigns schema going to version 1. No leg, assertion or verdict changed. |
| `tests/marketing.spec.js` | `[SP-01]`/`[SP-01b]`/`[SP-02]`/`[SP-02b]`/`[SP-03]`/`[SP-03b]`, plus the two pre-existing branch-3 tests adapted to tap-free refusal (see below). |
| `sw.js` | regenerated at the merged HEAD. **No new file — the precache count must stay 50.** |
| `night-crew.toml` | a COMMENT only (the roll call records that H6 added no spec file and no token). No new key, no new token. |
| `.night-crew/knowledge/BACKLOG.md` | B-446/B-447/B-440/B-436 marked closed. |

Files **outside** the declared footprint: `marketing/sync/harness/harness.mjs`,
`clock-harness.mjs`, `recovery-clear-harness.mjs` are inside `marketing/sync/harness/*`
(declared). `night-crew.toml` and `.night-crew/knowledge/BACKLOG.md` are not in the
slate's Footprint line — declared here: both are comment/bookkeeping edits, and
`night-crew.toml` carries a union roll-call comment Cards 2, 4 and 6 also edited, so
expect an ordinary conflict there and resolve by union.

---

## What must survive any merge

1. 🛑 **The `requires_online = true` refusal.** B-446 moves it EARLIER; it must never
   become conditional, and the post-Submit guard must stay. `[SP-01]` asserts both
   halves in one test precisely so a future change cannot trade one for the other.
2. 🛑 **The B-432 fail-closed predicate** in `createCampaignPolicySource` — a KNOWN
   code whose campaign is absent from the Map answers `{requiresOnline:true,
   unresolved:true}`. Untouched by this card; `campaigns-harness.mjs` leg 3's
   `replica + HIGH` / `replica + LOW` rows and the `B-432` / `B-434(c)` e2e still pin it.
3. **Decision 166 by construction.** `policyFor`'s no-source arm answers "refuse" only
   for a campaignId that is non-empty — uniform with the source's own predicate, which
   answers `null` for a code naming no campaign. `[SP-03b]` and two leg-3 predicate
   checks pin it. A fail-closed arm that also swallowed the genuinely-unknown code
   would repeal decision 166 on the devices least able to recover it.
4. **The divert predicate and the constraint agree.** `unverified_code &&
   offline_override`, matching `scan_attempts_names_a_code`. `f2-run.sh poison-mismatch`
   enumerates the constraint's refusal (23514) in the same run that measures the guard,
   so the two can never drift apart silently again.
5. **`CAMPAIGNS_REPLICA_SCHEMA` version 1 travels with its migration strategy AND with
   `addRxPlugin(RxDBMigrationSchemaPlugin)` in every harness that builds the campaigns
   collection.** Drop any one of the three and `addCollections` throws — the Scan page
   renders "Scanner failed to start" on every device (the `SCAN_ATTEMPTS_SCHEMA` lesson,
   restated).
6. **Cards 2's and 6's assertions in `tests/marketing.spec.js`** — the placeholder test
   "marketing.html shows Scan and Campaigns live, Subscribers gated, and Stats as a
   labeled placeholder" and its `#s2`/`#s3`/`#s4` claims are untouched by this card.

## What is safe to drop

* The `night-crew.toml` roll-call comment line — informational.
* `[SP-01b]`, `[SP-02b]`, `[SP-03b]` are guard/edge tests, not done_when rows; losing
  them costs coverage, not correctness.
* The `replicationIdentifier` bump in `scan-page.js` (`marketing-campaigns-pull` →
  `-v2`) is a one-time re-pull convenience: without it an already-synced device shows
  the id-prefix fallback until a campaign is next touched. Honest either way.
* Nothing else. Items 1–6 above are not droppable.

---

## Red-first — OBSERVED, with exit codes

Three independent reds, all measured on this branch before any production file changed.

| What | Command | Observed | EXIT | Log |
|---|---|---|---|---|
| `[SP-01]` `[SP-02]` `[SP-02b]` `[SP-03]` | `npx playwright test tests/marketing.spec.js -g "SP-0" --retries=0` | **4 failed / 2 passed** — the four done_when rows red; `[SP-01b]` + `[SP-03b]` pass pre-change **by design** (they pin behavior this card must NOT change) | `EXIT=1` | `logs/h6/sp-red.log` |
| B-440 poison row (LIVE substrate) | `bash marketing/sync/harness/f2-run.sh poison-mismatch` | the constraint refuses the shape (`HTTP 400`, `23514`, `scan_attempts_names_a_code`) and the shipped guard **diverts it anyway — 10 `land-unverified` attempts, 0 server rows, the legitimate attempt behind it stranded `pending`**: B-440's head-of-line poison, reproduced as filed | `EXIT=1` | `logs/h6/f2-poison-red.log` |
| B-436 leg-3 flip (LIVE substrate) | `bash marketing/sync/harness/campaigns-run.sh` | `none + HIGH` and `none + LOW` both render `overrideAvailable=true`, `afterOVERRIDE_REQUEST=overrideConfirm` — a `requires_online=true` code offline-overridable on a no-source device | `EXIT=1` | `logs/h6/campaigns-leg3-red.log` |

Two pre-existing tests change shape rather than claim, and the change is the card:
`offline branch 3` and `B-432 …(branch-3 minus the campaigns: seed)` tapped
`[data-action="ms-submit"]` to reach the gate. Under B-446 that control no longer
exists on a refused scan, so both now assert the gate renders with **zero taps** and
keep every original claim (`data-branch`, the copy, no `ms-override`, zero POSTs),
plus an explicit belt assertion driven through the machine. Strictly stronger; nothing
dropped.

## Harness legs — live substrate, GRADED not narrated

The local `spike-supabase` substrate was UP and GREEN (compose project
`spike-supabase`, REST resolved by `docker compose -p spike-supabase port rest 3000`;
RECONCILE mode, never `--fresh`, never `:5433`, no hosted project). Every leg below
RAN; its `EXIT=` marker is inside its own log file (B-445), never piped through `tail`.

| Leg | Log | EXIT |
|---|---|---|
| `f2-run.sh poison-mismatch` (red-first) | `logs/h6/f2-poison-red.log` | `EXIT=1` (red as expected) |
| `f2-run.sh poison-mismatch` (green) | `logs/h6/f2-poison-green.log` | **`EXIT=0`** |
| `f2-run.sh` (green, the pre-existing primary gate — must not regress) | `logs/h6/f2-green.log` | **`EXIT=0`** |
| `campaigns-run.sh` (red-first, leg 3 flipped) | `logs/h6/campaigns-leg3-red.log` | `EXIT=1` (red as expected) |
| `campaigns-run.sh` (green) | `logs/h6/campaigns-green.log` | **`EXIT=0`** |

The campaigns schema bump forced a one-line plugin registration in four harnesses, so
all five remaining legs were re-run as a regression check (same substrate, same rules):

| Leg | Log | EXIT |
|---|---|---|
| `run.sh` | `logs/h6/regress-run.log` | `EXIT=0` |
| `clock-run.sh` | `logs/h6/regress-clock.log` | `EXIT=0` |
| `recovery-clear-run.sh` | `logs/h6/regress-recovery-clear.log` | `EXIT=0` |
| `refusal-run.sh` | `logs/h6/regress-refusal.log` | `EXIT=0` |
| `push-run.sh` | `logs/h6/regress-push.log` | `EXIT=0` |

🛑 **One box rule learned the hard way, for whoever runs the next marketing/sync card.**
The first G2(Go) attempt red with four `internal/sync` `TestRowVisibilityRLS` subtests
(`logs/h6/g2-go-collided.log`, `EXIT=1`) — duplicate-key on seed, a leftover `w14-during`
row. **Self-inflicted and not a code red:** `internal/sync`'s RLS suite and
`marketing/sync/harness/*` provision the SAME `spike-supabase` substrate, and every
harness script runs `reset_bare` + `apply_all`. They must not run concurrently. Re-run
alone: `EXIT=0`, 15 ok packages, 0 FAIL (`logs/h6/g2-go.log`).

## Gates at close

| Gate | EXIT | Log |
|---|---|---|
| G1 (`go build ./...`, `go vet ./...`) | `EXIT=0` | `logs/h6/g1.log` |
| G2 Go (`go test -p 1 -count=1 ./...`) | `EXIT=0` — 15 ok / 13 no-test-files / 0 FAIL | `logs/h6/g2-go.log` |
| G2 Playwright (full suite, under `flock /tmp/hq-full-suite.lock`) | `EXIT=1` — **27 failed / 1001 passed / 6 skipped**, ONE summary block | `logs/h6/g2-playwright.log` |
| G4 (`node build-sw.js`) | `EXIT=0` — **precache 50, unchanged**, idempotent on re-run | `logs/h6/g4-sw.log`, `g4-sw-idempotent.log` |

Red-set diff against the run's 24-red baseline (`logs/base-pw.log`): **23 of the 24 still
red, 1 went green** (`inventory.spec.js:2931`), **4 outside it**, none this card's:

| Red | Why not this card |
|---|---|
| `inventory.spec.js:2186` (alias chips) | **B-459**, diagnosed: the `ALL_ITEMS` three-writer race. bugs.md: "treat a `:2186` red as this race… proven NOT attributable to any card tonight". Reds even in confined isolation on this HEAD (`logs/h6/flakepool-head-sample.log`), matching its recorded ⅔-red rate. |
| `inventory.spec.js:2919` (create item without group) | tonight's recorded flake pool (Card 2's G6). **Mechanical exclusion:** `tests/inventory.spec.js` contains ZERO references to `marketing` (`grep -c` = 0), and `playwright.config.js:65` blocks service workers for it, so neither `sw.js` nor any `marketing/*` file this card changed is reachable from it. |
| `recipes.spec.js:216` (slider PUT) | same flake-pool record; same mechanical exclusion (`grep -c marketing` = 0 in `tests/recipes.spec.js`). |
| `sw-api-cache-partition.spec.js:92` `[B1-XT-01]` | the ONE spec that opts into `serviceWorkers:'allow'`, so `sw.js` CAN reach it — treated as the hard case. **Two-tree rate control, 3 isolated passes each:** HEAD **3 green / 0 red** (`logs/h6/b1xt01-head-control.log`), base `28984ab` **3 green / 0 red** (`logs/h6/b1xt01-base-control.log`) — identical rate, so the card does not change it. Reinforced byte-level: the whole `sw.js` delta is four `revision` hashes (`submit-flow`, `scan-page`, `sync/replicas`, `sync/push-replication`) plus a minifier-local variable rename in the AMD shim. The `api-cache` `NetworkFirst` route, its `cacheKeyWillBeUsed` identity partition, `cacheWillUpdate` and `handlerDidError` — the exact code this spec tests — are **byte-identical**, and precache membership and order are unchanged. Long-known intermittent: B-174, and B-433's stale four. |

