# Sales-processor notice + pre-deploy checklist — FINALISED 2026-09-01

| | |
|---|---|
| **Status** | **FINALISED 2026-09-01 by the operator.** This is the contract record of what HQ changed under the HQ→sales-processor contract, and the checklist that gates the `0072`-carrying deploy. The operator maintains both sides (B-137), so "finalised" means *recorded as the spec of record* — there is no outside party to transmit it to. |
| **Prepared** | 2026-09-01, night-crew run `20260901`, Card 10 `counterparty-notice-prep` (Track C); reviewed + finalised by the operator 2026-09-01 (morning triage T-47). |
| **Source** | HQ automated audit + re-verification against the pre-deploy tree. Finalised: framing settled, §3 decided (below). |
| **Supersedes** | `docs/contracts/NOTICE-sales-processor-2026-08-03-UNSENT.md`. That file is left in place as cited evidence; this is the record of account. |
| **Deploy** | **Done 2026-09-02** (`task prod:deploy`, prod `git_sha` `a347bde`). Migration `0072` ran; changeover date recorded in §4. sales-processor side **verified already aligned** (host-local `America/New_York`); no change required — see §4. |
| **Closed on finalisation** | **B-29** (the undisclosed 2026-06-06 gate change) and **B-137** (counterparty-process) — closed 2026-09-01, ledger T-48. |

## Read this first — framing

**The operator maintains sales-processor themselves (B-137, established 2026-08-03).** There is no third-party maintainer to apologise to. This record is written **first-person, as a changelog + pre-deploy checklist to yourself** — the artifact that records what HQ changed under the contract and what must be true on both sides before the deploy — not as an outward apology to an external party. That reframing is the substance of B-137's fix; the audit body it carries is unchanged and stands on its own.

**Why it still matters even though the reader is the author:**
- **§1 is an operational finding about the operator's own payroll gate**, not a courtesy. Since 2026-06-06 the gate could return `ready:false` on periods the published rule called clear, and `ready:true` on periods holding unreviewed receipts. (Resolved 2026-08-04 for the exposure window — no spurious block found — see §1; carried here so the record is complete.)
- **The coordinated-release constraint is real.** Owning both sides removes the negotiation, not the one-hour disagreement: deploying HQ's `America/Chicago` → `America/New_York` changeover while sales-processor still computes Chicago produces the same edge disagreement. §4 is the checklist that gates the deploy.
- **The contract documents are the spec sales-processor is built and maintained against**, so a wrong contract is a wrong spec regardless of who reads it.

## Pre-deploy sequencing — the one hard ordering

1. ~~Review + finalise this notice (resolve the §3 `name`/`menu_item_name` decision).~~ **DONE 2026-09-01** — §3 resolved below (`menu_item_name` stays on the wire; the corrected document is the spec).
2. **Then** merge `dev` → `main`, promote, and `task prod:deploy` (Activity 2, `ship-dev-to-main`). Production still computes in `America/Chicago` until that deploy; `main` does not yet carry the change.
3. **Then** record the changeover date here — it is the date migration `0072` first ran (recover from goose's `goose_db_version.tstamp` for version 72, or the deploy log).

🛑 **Deploying HQ's New-York changeover before the sales-processor side is aligned (§4) is the failure this record exists to prevent.** The consumer half of sales-processor must not switch to New York on the assumption HQ has already moved, nor stay on Chicago after HQ has.

---

# RECORD — what changed under the contract

**Subject: HQ inventory API — contract corrections, the payroll-gate change of 2026-06-06, and the timezone changeover that needs the sales-processor side aligned**

Both HQ→sales-processor contract documents were audited line-by-line against the shipped code on 2026-08-03 (111 rows across the two, 45 wrong), and **re-verified against the pre-deploy tree on 2026-09-01** — the tree about to ship, including run `20260901`'s receipt-worker and observability fixes. The 2026-08-03 corrections still hold byte-for-byte; nothing on the wire changed since. **The code was fine throughout — the documents were not.** Both are now corrected, each with a full row-by-row `§0` audit recording what was checked and what was wrong, with the responsible commit named.

## 1. The one that touched the payroll gate (B-29)

**`/period-summary`'s `completeness.pending_review_ids` — the field the payroll gate keys on — changed twice on 2026-06-06, and neither change was documented.**

The published rule was: a pending receipt blocks payroll if its `created_at`, read as a Chicago calendar date, falls in the period.

What shipped that day was two changes pulling opposite ways:

- **`cf959bd` widened it.** The filter became `COALESCE(event_date, created_at::date)`. A late-discovered receipt — a May 29 purchase ingested June 2 — used to fall outside the May window on `created_at` and *not* block May payroll. After the change its extracted `event_date` puts it inside the window and it **does** block.
- **`d41faef` narrowed it, much further.** Two clauses were added: the row must be a COGS-category transaction **and** carry `reason = 'no_attachment_on_bank_tx'` (no receipt attached at all). Everything else that used to block stopped blocking.

**Where the money that stopped blocking went — three buckets, and only one moves a number you see:**

1. **COGS-category, receipt attached but the parser failed.** Stopped blocking **and** began contributing full `ABS(bank_total)` to `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count` and `by_vendor`. The only bucket that moves money into the report.
2. **Anything whose Mercury category is not in the COGS allowlist (including no category).** Both the blocking query and the COGS query filter on the allowlist, so these block nothing **and contribute nothing** — invisible to every COGS figure. They remain only in `tracked_bank_tx_ids` (deliberately unfiltered).
3. **COGS-category with a NULL `reason`.** `reason = '…'` and `reason != '…'` are both NULL against a NULL column, so these fall through both filters — block nothing, contribute nothing.

**Practical consequence:** since 2026-06-06 the gate could return `ready:false` where the published contract said `ready:true`, and vice-versa.

> **✅ RESOLVED 2026-08-04 — §1 exposure window checked, no restatement.** The gate went live 2026-06-05, one day before the 06-06 changes. The consumer's run history shows an unbroken weekly run 2026-05-31 → 2026-07-19 (reports + transfer ledgers, no gaps), and the two later weeks (07-26, 08-02) were operator skips (business closed), for which HQ's `/period-summary` returns `ready:true`, zero pending, zero unlinked. **No spurious block anywhere in the window. No past figure is restated.** Two class-defects surfaced from this and are now fixed/filed: a skipped week is byte-for-byte indistinguishable on disk from a blocked week (consumer now runs payroll even on closed weeks), and HQ could not see a blocked week either — filed and now **closed by Card 9 (B-139): `/period-summary` logs every served response and its `ready` verdict** (see §7).

## 2. Fields HQ returns that were never documented (`/period-summary`)

Three response fields live since June 2026, in no document, all **additive** (a decoder built on the old shape ignores them — nothing breaks):

- **`by_vendor`** — per-vendor COGS breakdown with trip counts.
- **`tracked_bank_tx_ids`** — every Mercury `bank_tx_id` HQ touched for the period, any state. **Deliberately unfiltered by category** — this is the field to diff against Mercury to find "Mercury has it, HQ hasn't ingested it." Do not add a category filter to it.
- **`completeness.pending_review_details`** — the `pending_review_ids` rows with vendor, date, amount and reason attached, so a blocked-payroll message renders without a second call.

**And one undocumented input:** env var `HQ_COGS_CATEGORY_ALLOWLIST` (default `COGS`), which gates four response fields. The aggregate is "purchasing categorised as COGS in Mercury," not "all purchasing." A miscategorised transaction is invisible to the COGS figures but still appears in `tracked_bank_tx_ids`.

Also: **`purchase_event_count` no longer counts `purchase_events` alone** — it includes the case-1 pending rows above. A week with 4 confirmed receipts + 3 such queued rows reports `7`. And `cogs_excl_tax` includes those rows at full bank amount, so the COGS number moved **up** on 2026-06-06 relative to the doc.

## 3. `/menu-cogs` — most of it was wrong the day it was written

The menu-cogs contract did not drift — **most of it was wrong on 2026-06-04**, written from the phase plan rather than the shipped handler, never diffed. Note: **no consumer has ever been built against `/menu-cogs`** (confirmed 2026-08-04), so nothing here is load-bearing today — but if a consumer is built later it will be built from the corrected document, so the corrections land before a line of client code exists.

**Four of the nine per-item fields never matched the code, four different ways:**

| Documented | Actually sent | Kind of wrong |
|---|---|---|
| `name` | `menu_item_name` | wrong name — decodes to `""` |
| `menu` | **nothing — field does not exist** | phantom field (the `menu_items.menu` column exists but the handler never selects it) |
| `menu_subgroup: null` when absent | key **omitted** entirely | wrong nullability |
| *(not mentioned)* | `toast_master_id` | undocumented — sent but never stated |

Three more in the same document:

- **`ingredient_cost_total`'s formula was wrong.** The doc published a division by the sum of usage percentages; that division does not exist. The code is `spend × (usage_pct / 100)` and leaves the remainder in `unallocated_cogs`. The two agree only when an ingredient's percentages sum to exactly 100.
- **Row selection was backwards both ways.** Not "one row per menu item with sales" — it is one row per menu item **with a recipe**, sales or not. A menu item that sold with no recipe **does not appear at all** (so the old Scenario 2 could never pass).
- **The cross-endpoint reconciliation invariant is broken.** `sum(menu_items[].ingredient_cost_total) + unallocated_cogs ≈ period-summary cogs_incl_tax` stopped holding on 2026-06-06 when `/period-summary` gained the category allowlist and pending-row rollup and `/menu-cogs` did not. The two now diverge in both directions by unbounded amounts. Any reconciliation check on this fires spuriously — turn it off until the two are re-aligned or formally declared non-reconcilable (§8 Q4 in the doc).

**Decision — RESOLVED 2026-09-01 (operator).** On `name` vs `menu_item_name`: **leave the wire as-is.** HQ continues to send `menu_item_name`; the corrected `inventory-menu-cogs.md` is the spec of record, and any future consumer is built against the document, not against the old `name` prose. `name` is **not** adopted (changing the wire to match stale documentation would be the original sin — quietly altering an endpoint to match old prose — repeated). HQ **may** additionally return `menu` (purely additive) if a later consumer needs it; until then it stays unshipped.

## 4. The timezone changeover — this one needs the sales-processor side aligned before the deploy

Everything above is the past. This section is HQ's timezone changeover — **deployed to production on 2026-09-02**. It required action on both sides; the sales-processor side was verified 2026-09-02 and is **already aligned** — see the resolution below.

**HQ's operating timezone is moving from `America/Chicago` to `America/New_York`** (ledger T-26 decision 83). A payroll week and a food-cost week must describe the same seven days, and HQ had been running two zones at once — the user-level default was already New York while the money queries had Chicago as a literal.

**Status, stated precisely (updated post-deploy 2026-09-02):**
- The change is **written, merged, and deployed** (`users.DefaultTimezone = "America/New_York"`, `pendingPeriodDateExpr` reads that single constant, migration `0072_app_timezone_new_york.sql` re-points the `cutoff_config` / `repurchase_reset_config` defaults and updates existing rows).
- It is **deployed.** Production computes `America/New_York` as of 2026-09-02 (`task prod:deploy`, image built `2026-09-02T21:39:32Z`, prod `git_sha` `a347bde`). Before this date production computed `America/Chicago`.
- **Changeover date: 2026-09-02.** The date migration `0072` first ran in production — the single day every weekly/daily boundary moved one hour earlier (Chicago → New York), once. Authoritative source: `goose_db_version.tstamp` for version 72. Weekly COGS/payroll figures produced before this date were computed in `America/Chicago` and are **not restated** (fix-forward, per the migration).

✅ **sales-processor side — resolved 2026-09-02, already aligned.** sales-processor carries no explicit timezone: it anchors the pay week on host-local time (`main.go` `now := time.Now()` → `service.GetDateLastSunday` → `service.GetDatesStartingFromPreviousMonday`, no `.In(zone)`, no `TZ` in `.env`, no `America/*` literal). It is run via `go run main.go` on an `America/New_York` host, so it already computes New York — it always did. **HQ's Chicago → New_York move brought HQ into line with sales-processor, not the reverse; there is no one-hour disagreement to fix.** Latent fragility: sales-processor's zone is *implicit* in the run-host, so running it on a UTC/Central host would silently shift its weeks out of alignment — the same class of bug HQ's `0072` retired. Hardening it to an explicit `time.LoadLocation("America/New_York")` is a filed follow-up (handoff in the sales-processor repo: `docs/handoff-timezone-hardening.md`), not a blocker.

**What actually moves, and it is narrow.** The zone enters `/period-summary` at exactly one point: the `created_at` fallback inside `pendingPeriodDateExpr`, which applies **only** to pending, unconfirmed receipts whose parser extracted no purchase date. The confirmed half has no zone dependency (`purchase_events.event_date` is a plain `DATE`). But that one expression period-filters the pending rows, so where it applies it reaches the money as well as the gate — `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count`, `by_vendor` and `completeness.pending_review_ids`. New York is an hour ahead, so a boundary receipt created 23:00–midnight Central with no parsed date moves from one week to the next — a shift, not a widening.

🛑 **`/menu-cogs` does NOT move on this deploy.** There is no `AT TIME ZONE` cast anywhere in its code path. A 2026-07-31 amendment (assumption A10) wrongly said it shares `/period-summary`'s date semantics; that is corrected in the doc and stated here so the correction is not lost. Nothing on the sales-processor side needs changing for that endpoint.

**Adjacent open zone question (context, not a contract change — B-177).** Run `20260901`'s receipt-worker fix (Card 1, B-28) corrected the *unparseable* `event_date` fallback from server-local to the business zone. A follow-on finding, **B-177 (filed, deferred)**, is that the *parseable* path still stamps `event_date` from the timestamp's own zone (UTC) rather than the business zone — and `event_date` wins the `pendingPeriodDateExpr` COALESCE, so a parseable boundary receipt can file to the adjacent COGS week. It is pre-existing, entangled with this changeover, and deliberately scoped to land alongside the `0072` zone decision. It does **not** change what prod serves today; flagged only so the two zone decisions are made coherently.

**What is NOT being done:** no weekly COGS or payroll figure produced before the deploy is restated. One boundary moves, once, on a known date.

## 5. What changed, what did not

**No HQ code was changed by the audit.** Every correction moved the *documents* to match the shipped behaviour — no endpoint was quietly altered to match old prose (that would be the original sin repeated). On `/menu-cogs`, correcting the doc to match the code arguably makes it honest rather than right; those are recorded as open questions in each `§8`.

## 6. Why this went unnoticed for eight weeks

Both documents claimed their integration tests were "the executable proof that the HQ side matches this contract." They are not — the tests decode into the same Go structs the handlers marshal, so a field name the *document* gets wrong is invisible to all of them. A test that would have caught it must assert on raw JSON keys. Filed as **B-71**.

## 7. New in run `20260901` — HQ can now see a served request (Card 9, B-139)

`/period-summary` and `/menu-cogs` each now emit one `slog.Info` at the end of the success path (`"period-summary served"` with `from, to, ready, pending_review_count, unlinked_line_item_count`; `"menu-cogs served"` with `from, to, menu_item_count, breakdown`). **These are server-side log lines, not response-shape changes** — the wire JSON is unchanged, so no decoder is affected. They matter to the integration because a blocked payroll week is now greppable from HQ's own logs, independent of what the consumer writes to disk — closing the "HQ cannot see a block either" half of the §1 finding.

# RECORD ENDS

---

## Provenance

Every claim traces to a `§0` audit row in `docs/contracts/inventory-period-summary.md` (47 rows) and `inventory-menu-cogs.md` (64 rows) — **111 rows, 45 wrong**. Response shapes were observed by marshalling `inventory.PeriodSummary` / `recipes.MenuCOGSResponse` at HEAD. Both docs re-verified against the pre-deploy tree (Cards 1+9) on 2026-09-01; the corrections still hold. Commits named: `cf959bd`, `d41faef`, `a726029`, `518a395`, `f730485`, `1c260f0` (all 2026-06-05/06).

## B-137 counterparty-process lesson (recorded in ledger T-48)

> **B-137 lesson:** sales-processor is a peer with a contract of record maintained by the operator — not an external afterthought, and not an outside party inferable from the repo's absence from this tree. A published cross-repo contract has no mechanical link to the code it describes, so any change to a handler named in a `*-CONTRACT.md` can silently invalidate the contract; the durable fix is to treat those docs as the spec of record — re-diff a contract's stated SQL against its handler whenever the handler changes, and notify the peer (even when the peer is yourself) before the change ships, because the notice is the pre-deploy checklist, not a courtesy.

## Status — finalised

This record was **finalised 2026-09-01** (operator, morning triage T-47/T-48; B-29 and B-137 closed) and **the deploy landed 2026-09-02** (`task prod:deploy`, prod `git_sha` `a347bde`): migration `0072` ran, HQ now computes `America/New_York`, and the changeover date (2026-09-02) is recorded in §4. The sales-processor side was **verified already aligned** the same day (it computes host-local `America/New_York`; the deploy brought HQ into line with it, §4). That closes every thread of the `counterparty-combined-notice` roadmap card. Optional follow-up (non-blocking): harden sales-processor's implicit timezone to an explicit `America/New_York` — handoff at `sales-processor/docs/handoff-timezone-hardening.md`.
