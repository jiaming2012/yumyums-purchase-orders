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
`[build failed]`, exit 1. The green run is logged beside it as `g2-go.log`.

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
