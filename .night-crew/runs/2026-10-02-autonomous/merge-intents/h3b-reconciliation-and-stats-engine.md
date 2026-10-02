# Merge intent — card h3b `reconciliation-and-stats-engine` (run 20261002, Activity H, Card 4)

Branch `card/h3b-reconciliation-and-stats-engine`.

## 🛑 Based on Card 3's UNMERGED branch

Cut from `card/h3a-toast-orders-and-mirror` at `2e17ae3`, **not** from `overnight-20261002`.
Card 3's migration `0084` carries the three tables this card writes
(`toast_orders`, `scan_attempts_mirror`, `reconciliation_decisions`), so this card adds
**no migration**. Consequences for the orchestrator:

- **Merge Card 3 first.** This branch's history contains Card 3's commit; merging this
  branch alone would land Card 3's work under this card's name.
- Card 3's files — `0084_toast_orders_reconciliation.sql`, `internal/toast/orderdetails.go`,
  `internal/marketing/mirror.go`, `internal/redemption/store.go` — are **untouched** by
  this card. One defect inherited from Card 3 is REPORTED, not fixed (see
  *Inherited from Card 3* below).

## Shared files touched (everything outside my two new files)

| File | Why |
|---|---|
| `backend/internal/marketing/routes.go` | **The append.** One labelled `── card H3b ──` block inside `Mount` (the queue + the five decision writes + the declined bucket + the two `/stats/*` reads); `MountReports` is FILLED (decision 192's BI mirror) via a new `MountReportsDeps`; `NewDeps` records its own result in a package-level pointer so `MountReports(r)`'s Card-1 signature still compiles and `main.go` stays untouched. |
| `backend/internal/marketing/types.go` | `moneyDTO` gains four ALWAYS-PRESENT keys: `discount_implied_cents`, `discount_actual_cents`, `discount_unknown_rows`. No existing key renamed, removed or retyped. `zeroMoney()` unchanged in meaning. |
| `backend/internal/marketing/campaigns.go` | The card's "`GET /campaigns`' money block becomes real": the three `zeroMoney()` call sites (list / get / patch) now take the computed per-campaign money, and `funnel.redeemed` / `funnel.signups` are filled from the same engine. `funnel.scans` is left exactly as Card 1 computed it. |
| `backend/internal/marketing/routes_test.go` | `TestMountReportsIsANoOpSeamToday` is INVERTED into `TestMountReportsRegistersTheBICampaignReports` — Card 1 wrote it as "it must register NOTHING **until H3b fills it**"; this is the card that fills it. |
| `night-crew.toml` | Roll-call COMMENT only — **no new key, no new token**. It records that `backend/internal/marketing/` is **not** a key in `[e2e.seams]` (the `marketing` key is a path prefix matching the FRONTEND tree), so every Activity-H backend card has an undeclared footprint and de-confines to the full Playwright suite. H3b ran the full suite on that basis rather than claiming its slate line's subset. Whether that path should get a key is a toml change and therefore the operator's. |

New files: `backend/internal/marketing/reconciliation.go`, `backend/internal/marketing/stats.go`,
plus tests `reconciliation_test.go`, `stats_test.go`, `recon_fixture_test.go`.

**`backend/cmd/server/main.go` is UNTOUCHED — expected and stated.** That is the whole
point of filling `MountReports` from inside the package (decision 192).
**No migration added** — `0085` belongs to Card 6.

## What must survive any merge

1. **The `routes.go` block is append-only and must coexist with Cards 1, 3 and 6's blocks.**
   One labelled block per card inside `Mount`; order among blocks is irrelevant (chi paths
   are disjoint). If a conflict appears, KEEP BOTH blocks.
2. **`GET /campaigns`' `money` is now REAL and replaces Card 1's zero shape.** Card 2's
   shipped UI renders from it and Card 5's reports read the same `moneyDTO`. A merge that
   restores `zeroMoney()` on the list/get/patch paths silently re-zeroes the manager's money
   column. **Card 1's key-set property is preserved: every key present, `null` = "no
   opinion".** Four keys are ADDED; none removed.
3. **`MountReports` must stay filled.** `/api/v1/bi/campaigns/{overview,by}` answers 404
   today only because the seam was a deliberate no-op. Reverting `MountReports` to a no-op
   un-ships the whole report half (decision 192) while leaving the queue working, which
   would look like a UI bug in Card 5.
4. **`NewDeps` records its result.** If that two-line append is dropped, `MountReports(r)`
   has no pool and the BI pair answers `503 reports_unavailable` — loud, not silent, but
   still broken.
5. **The manager tier asymmetry.** `/api/v1/marketing/stats/*` applies it; `/api/v1/bi/campaigns/*`
   must NOT (decision 192: `bi` grant alone). One handler pair, one `managerTier bool`.

## What is safe to drop

- The roll-call comment in `night-crew.toml` — nothing here.
- `recon_fixture_test.go`'s `subscribersTableExists` branch once Card 6's `subscribers` has
  landed everywhere; it is a compatibility shim, not a contract.
- Nothing else. There is no dead code and no scaffolding in this change set.

## Red-first

Observed on this branch BEFORE any of `reconciliation.go` / `stats.go` existed, with the four
named tests written first. Log: `.night-crew/runs/2026-10-02-autonomous/logs/h3b/rf-red-first.log`.

```
# github.com/yumyums/hq/internal/marketing [github.com/yumyums/hq/internal/marketing.test]
internal/marketing/reconciliation_test.go:38:10: undefined: ReconQueueResponse
internal/marketing/reconciliation_test.go:153:15: undefined: ReconDeclinedResponse
internal/marketing/stats_test.go:23:42: undefined: MountReportsDeps
internal/marketing/stats_test.go:131:9: undefined: StatsOverviewResponse
internal/marketing/stats_test.go:174:10: undefined: StatsByResponse
FAIL	github.com/yumyums/hq/internal/marketing [build failed]
EXIT=1
```

All four `done_when` tests — `TestQueueOrdersOverridesThenOrphansThenUnmatched`,
`TestDeclineOtherRequiresNote`, `TestOrphanRateCountsDeclinesExceptDuplicateScan`,
`TestSlicesReconcileToOverview` — are inside that build failure, i.e. red by
`[build failed]`, exit 1.

## Gate results

| Gate | Command | EXIT | Log |
|---|---|---|---|
| G1 | `go build ./...` + `go vet ./...` from `backend/` | 0 / 0 | `logs/h3b/g1-build-vet.log` |
| G2 Go | `go test -p 1 -count=1 -v ./...`, `DB_TEST_URL` set | 0 — **672 PASS / 0 FAIL / 3 SKIP** (base 655/0/3; +17 = 18 new test funcs − 1 inverted) | `logs/h3b/g2-go-final.log` |
| G2 Playwright | full suite under `flock /tmp/hq-full-suite.lock` | 1 — **22 failed / 5 flaky / 6 skipped / 962 passed**, ONE summary block | `logs/h3b/g2-playwright-full.log` |
| G2 PW delta | `tests/marketing.spec.js` at the last commit | 0 — 47 passed | `logs/h3b/g2-playwright-marketing-subset.log` |
| G4 | `node build-sw.js`, twice | 0 — **48 precached**, unchanged, `sw.js` not modified | `logs/h3b/g4-sw.log` |

**Playwright red-set diff against tonight's measured baseline of 24** (`logs/base-pw.log`):
my 22 are the baseline's 24 **minus two that now PASS** — `inventory.spec.js:2767`
("user can set store_location", the B-453 flake) and `inventory.spec.js:2931`
("creating item opens edit form with store location dropdown"). **ZERO new reds.**
The 5 flaky all passed on retry and are all Inventory/Recipes specs this card does
not touch. `tests/marketing.spec.js` — this card's own seam — is fully green in both
legs.

**Why the full suite and not the slate's subset:** `backend/internal/marketing/` is
not a key in `night-crew.toml` `[e2e.seams]` (the `marketing` key is a path prefix
over the FRONTEND tree), so this footprint is undeclared and de-confines. Adding the
key would be a new key = PARK, so the suite was run instead and the gap recorded as
a roll-call comment.

## Was `subscribers` present when I computed signups?

**ABSENT.** `backend/internal/db/migrations/` ends at `0084` on this branch; Card 6's
`0085_subscribers.sql` had not landed. Therefore:

- `funnel.signups` is **literally `0`**, and the overview carries
  `"signups_basis": "unavailable"` so a consumer **can** tell it apart from a real zero.
  When Card 6 lands, the same code path finds the table (`to_regclass`) and returns the real
  count with `"signups_basis": "subscribers"` — **no code change needed at merge**.
- `funnel.codes_sent` is **literally `0`** with `"codes_sent_basis": "unavailable"`
  (`subscriber_events` is in the same absent migration).
- `TestSlicesReconcileToOverview` and `TestSignupsStatesItsBasisWhenSubscribersIsAbsent`
  both branch on `to_regclass`, so they assert the stronger figure (signups = 5, basis
  `subscribers`) automatically once `0085` is in the migrated schema. **Re-run the Go suite
  after merging Card 6** — it is the stronger assertion, already written.

## Inherited from Card 3 — reported, NOT fixed

`scan_attempts_mirror.campaign_id` is mirrored as **always NULL** (Card 3's stated deviation
2). `mirror.go`'s header says *"H3b resolves campaign through code_id when it needs it"* —
**it cannot.** `scan_attempts.code_id` is the Supabase `public.codes` id (the per-customer
redemption token row), and HQ Postgres has **no `codes` table at all**: nothing maps
`code_id → campaign_id`. So on live data today every accepted attempt is unattributable and
the campaign/channel/item slices are all `unattributed`/`direct`. Details and blast radius
in the card's final report. **No fix attempted — Card 3's files are untouched.**

### `opened_at`'s timezone — blast radius, measured

Card H3a's G6 found `orderdetails.go` stamps the export's naive `Opened` with a
hardcoded `America/Chicago` while the rest of the tree reads
`users.DefaultTimezone = America/New_York` (T-26 decision 83, migration 0072), and
nothing establishes which wall clock Toast writes. Quantified against this card:

- **`matched`, revenue, discount, net, every slice, and the orphan rate: UNAFFECTED.**
  They all read `matched()`, which keys on equality of `pos_business_date` and
  `toast_orders.business_date`; `business_date` is `opened.Date()` of the wall-clock
  string and is zone-independent. **No number a manager sees moves.**
- **The ±30-minute suggestion is the only thing that moves.** A one-hour offset puts
  every real order outside the window.
- **It cannot reclassify anything.** `bucket()` reads the ORDER NUMBER, never the
  suggestion, so an unmatched attempt with no suggestion stays `unmatched` and never
  becomes an `orphan`. Pinned by `TestSuggestionSurvivesAOneHourOpenedAtOffset`.
- **Hardened without touching a spec rule:** `reconNearestOrder` keeps the card's
  ±30-minute rule as rung 1 (`basis:"window"`) and adds rung 2 — the nearest order on
  the attempt's own, zone-independent `business_date`, `basis:"business_date"` with
  `gap_seconds` — so a systematic offset surfaces as a labelled ~3600s hint instead of
  as silence. The suggestion is advisory by §8's framing, so widening the ADVICE
  changes nothing decision 190 fixes. Card H3a's code is untouched.

Also noted for triage: this card's orphan rate inherits `qr_scans`' **write-time** 10-minute
dedupe, which is **DECISIONS-NEEDED D-2** (read-then-write under READ COMMITTED). `scans` is
not the orphan-rate denominator (accepted attempts are), but it IS the funnel denominator a
manager reads beside it. Not fixed, by instruction.

---

# APPENDED 2026-10-02 — G6 fix round (F1 / F2 / F3)

G6 came back APPROVE-WITH-FINDINGS and **confirmed D-4 at source**
(`mirror.go:370` inserts `campaign_id` as a literal NULL and omits it from the
`ON CONFLICT SET` list; `public.codes.id` and `qr_codes.id = gen_random_uuid()`
are different id spaces, so the join never matches live). It also re-proved the
keystone on a harder fixture than mine — one including an **unmatched** row, a
shape my fixture never modelled — and Σ still equalled overview across four dims.

## Shared files touched by this round

Only files already declared above: `types.go`, `stats.go`, `campaigns.go`,
`routes.go` (one comment). New test file `fixround_test.go`. **No new file outside
`backend/internal/marketing/`, no migration, `main.go` still untouched, `sw.js`
unchanged (48).**

## F1 — `GET /campaigns` rendered a confident $0.00. FIXED.

G6's live shape: a campaign with 6 scans and one **matched $30.00 order whose
`campaign_id` is NULL** returned `funnel.redeemed 0 / revenue_cents 0 /
discount_cents 0 / discount_unknown_rows 0`. The money was real and sitting in the
invisible `unattributed` row of the by-campaign slice, and **Card 2's money strip
is already shipped against this block.**

Why the existing signal could not fire: `discount_unknown_rows` is a **per-row
count scoped to the group**, so on the campaign's own row 0 is legitimately
correct — the campaign has no unpriceable rows *of its own*. The missing fact is a
**period** fact.

**`moneyDTO` gains two always-present keys** (no Card-1 key renamed, retyped or
removed — all seven intact):

| Key | Type | Meaning |
|---|---|---|
| `unattributed_redeemed` | `*int` | accepted redemptions in the period that **no campaign could claim** |
| `unattributed_revenue_cents` | `*int` | the matched revenue sitting inside them |

Scope follows this block's existing null contract: **non-null** on the
period-scoped routes (`GET /campaigns`, `GET /campaigns/{id}`,
`PATCH /campaigns/{id}`, `GET /stats/overview`) — where **`0` is a stated fact**
meaning "every redemption found its campaign", which is what lets a UI render a
plain `$0.00` with confidence; **null** on `/stats/by` rows and totals, where a
slice has no opinion on a period fact and states the figure as its own
`unattributed` row instead. **Never sum them across rows.** One
`statsPeriodScope` struct now decides which routes state period facts, so the
campaigns routes and the overview cannot disagree.

**What Card 2 and Card 5 must render when it fires** (also written into
`logs/h3b/wire-shapes-for-card-h4.log`, which Card 5 is dispatched against):
Card 2 renders the campaign's own `$0.00` — it is a true zero — but **not alone**:
beside the strip, "1 redemption ($30.00) not attributed to any campaign". Card 5
renders the by-campaign slice's `unattributed` row as a real row and must never
filter it out, or the slice stops summing to the overview.

Live shape now returned on G6's fixture:
`{revenue_cents:0, …, discount_unknown_rows:0, unattributed_redeemed:1, unattributed_revenue_cents:3000, avg_order_cents_with:null, avg_order_cents_without:null}`.

## F2 — the orphan-rate numerator is wider than the words. DISCLOSED, NOT CHANGED.

🛑 **The rule is unchanged.** `night-crew decisions log` returned
**`verdict: park`, top severity**; it is **D-5** in `DECISIONS-NEEDED.md`.

**The undisclosed choice, stated plainly.** `reconAttempt.countsInOrphanRate`
counts **every accepted attempt with no matched Toast order**, minus declines whose
reason is `duplicate_scan`. That set is the `orphan` bucket (no order number at
all) **plus the `unmatched` bucket** (an order number *was* typed, Toast has no
such order). §5's words are "orphan rate = orphans ÷ accepted", which reads
narrower. On G6's five-row fixture:

| Reading | Numerator | Rate | vs the 10% threshold |
|---|---|---|---|
| **shipped** (orphans + unmatched, excl. `duplicate_scan`) | 3 of 5 | **60%** | 6× over |
| narrow literal (orphan bucket only, excl. `duplicate_scan`) | 2 of 5 | **40%** | 4× over |

Both breach the threshold on that fixture, so the ruling does not flip the banner
there — but it moves a figure **P-KR3 is graded on** by 20 points, and my spike
fixture models no unmatched row, so it never disambiguated. That omission was
mine and is now on the record.

**`StatsReconciliation` gains three keys** so the figure is self-describing:
`orphan_rate_basis` (the constant
`"unmatched_and_orphans_excl_duplicate_scan"`), `orphan_numerator`,
`orphan_denominator`. The basis is stated even when the rate is `null`.
`TestOrphanRateDisclosesItsNumerator` **pins 3/5** on G6's shape, so whichever way
D-5 is ruled the change is a deliberate edit to an asserted value, and a consumer
that keyed off the old basis string sees a NEW string rather than a silently
different number under the same name.

## F3 — a comment in my own code was false. CORRECTED.

`types.go` said of `discount_unknown_rows`: *"Non-zero here means the discount
total is a floor, not a figure."* **Untrue for a matched row** — G6's probe
returned `basis:"actual", discount_cents:400, discount_unknown_rows:1`, which is
exact. Corrected to say what is true: each counted row contributed **no face
value**, so `discount_implied_cents` is a floor; `discount_cents` is a floor only
for the counted rows that are **unmatched**, because a matched row with no campaign
still contributes its order's exact actual discount. Read it as "N redemptions had
no campaign to price them", never as "the total is understated by N rows".

## Also done, from the same review

- `reportDeps` now carries the comment you asked for: it is a **package singleton,
  last `NewDeps` wins**, correct for this process (main.go calls it once, before
  either mount) and the price of filling H1's one-argument seam without touching
  `main.go`. The escape hatch is named: `MountReportsDeps` already takes Deps, so
  a two-pool process deletes the variable — a `main.go` edit, so not this card's.
- **RF disclosure:** the card's original red-first was a **compile** red
  (`[build failed]`, undefined types) rather than behavioural — acceptable for a
  brand-new package where the handlers did not exist, and stated here rather than
  implied. **This fix round's red is behavioural**: the struct fields were added
  first with no wiring, the package compiled, and the tests then failed on the
  defects themselves (`unattributed_redeemed is null on the campaigns list`,
  `orphan_rate_basis = ""`), exit 1. Both stages are in
  `logs/h3b/rf-fixround-red.log`.
- A defect of my own, found while wiring F1 and fixed before it shipped: the first
  cut recovered a `statsAgg` from an already-rendered `StatsRow` to re-render it at
  period scope, guessing `matchedRows` for the `"mixed"` basis. That is a **second
  arithmetic** — the one thing `stats.go` exists to prevent. Replaced by
  `statsGroup`, which returns the accumulators, so the period-scoped render uses
  the *same* accumulator as the slice.

## Gate results, this round

| Gate | Command | EXIT | Log |
|---|---|---|---|
| RF stage 1 | the four new tests, pre-field | 1 — `[build failed]`, undefined `UnattributedRedeemed` / `OrphanBasisUnmatchedAndOrphans` | `logs/h3b/rf-fixround-red.log` |
| RF stage 2 | same, fields added, **nothing wired** | 1 — **behavioural**: `unattributed_redeemed is null`, `orphan_rate_basis = ""`, `orphan_numerator/denominator = 0/0` | same file |
| G1 | `go build ./...` + `go vet ./...` | 0 / 0 | — |
| G2 Go | `go test -p 1 -count=1 -v ./...` | 0 — **676 PASS / 0 FAIL / 3 SKIP** (was 672; +4 new tests) | `logs/h3b/g2-go-fixround.log` |
| G4 | `node build-sw.js` | 0 — **48 precached**, `sw.js` unchanged | `logs/h3b/g4-sw-fixround.log` |

**No Playwright this round**, by instruction: the box lock is Card 7's, the diff is
backend-only, and the card's own full suite already measured **22 failed / 962
passed = zero new reds** against the baseline 24.
