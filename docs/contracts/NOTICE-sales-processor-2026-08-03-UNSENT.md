# 🛑 UNSENT DRAFT — outbound notice to the sales-processor maintainer

| | |
|---|---|
| **Status** | **UNSENT. NOTHING HAS BEEN DELIVERED.** |
| **Drafted** | 2026-08-03, night-crew run `20260803`, card P6 `period-summary-contract-notice` |
| **Drafted by** | HQ (automated audit). **Not reviewed. Not approved. Not sent.** |
| **Sending is** | **the operator's act, and the operator's alone.** |
| **Channel** | undecided — the operator picks |
| **Supersedes** | nothing, but it **absorbs** the other notice owed to sales-processor: the timezone notice from card A1, which is **owed and was never drafted** (ledger decision 106 records it as owed; no draft has ever existed in this repo). Per decision 128 this is now **one combined notice**. See "Sequencing" below. |

**To whoever reads this file next:** do not treat any part of it as communicated. The audit that produced it (`docs/contracts/inventory-period-summary.md` §0 and `docs/contracts/inventory-menu-cogs.md` §0) is merged into the contract documents; **the counterparty has not been told.** Until someone sends this, sales-processor is still working from the pre-audit contracts.

---

## Before sending — two things the operator must decide

The draft below is written to be sendable **as-is**, but two decisions are embedded in it and neither is HQ's to make unilaterally:

1. **Whether to reconcile past runs.** The draft says HQ is *not* proposing a restatement. If the operator wants past weeks re-examined, §1's closing line and §5 both need rewriting (and §4's "What we are not doing" says the same thing for the changeover). **The audit deliberately makes no claim about whether any payroll run was actually blocked** — only that it became possible on 2026-06-06. HQ retains no log that would settle it.
2. **The `menu_item_name` / `name` question** (menu-cogs §8 Q2). The draft *asks* the counterparty rather than announcing a fix, because the right answer depends on what sales-processor actually built. If the operator would rather just change HQ's key, §3 becomes an announcement instead of a question.

*(There were three. The sequencing question is no longer one of them — see below.)*

## Sequencing — DECIDED, not open

🛑 **This was an open question in the drafted version and it is now settled: ledger decision 128 (triage 2026-08-02) rules ONE COMBINED NOTICE, amending decision 106.** The draft below is that combined notice. Nothing here is asking the operator to re-choose sequencing; what remains is whether to send it.

Two notices were owed:

- **A1's timezone notice** — the `America/Chicago` → `America/New_York` changeover, which requires a **coordinated two-repo release** and is therefore time-sensitive against the next HQ deploy. **Owed and undrafted:** decision 106 records it as owed, and no draft has ever existed in this repo — a search of `git log --all` finds no trace. *(The drafted version of this file asserted it was "drafted 2026-08-01". That was wrong and is corrected here; it is the B3 item of the P6 fix-forward checklist.)*
- **This notice** — eight weeks of undisclosed contract drift, not tied to a deploy date but larger.

**Why 128 amended 106 rather than honouring it.** Decision 106 had ruled two notices, the June drift first and alone. Three things surfaced afterwards: this audit covered every `:NN` row of both documents — **111 rows, 45 wrong** — and only a minority had *drifted*, with **22 menu-cogs rows never true at all**; A1's own notice carries an error *this* audit found (it attributes a timezone claim to `/menu-cogs`, which contains no `AT TIME ZONE` at all), so sending it alone would **propagate a fresh error while apologising for old ones**; and A1's notice was never drafted, so "send it first" had nothing to send. The owed timezone correction is therefore **folded into this message** rather than chased separately.

✅ **A1's timezone content IS now written into the draft below, as §4** (added 2026-08-03 at morning triage, after the P6 fix-forward checklist was discharged). Decision 128's requirement is therefore satisfied: this is the one combined notice, carrying both the eight weeks of drift and the forward-looking changeover. §4 also carries the correction to A1's own false claim that `/menu-cogs` moves with the changeover — verified at source: there is not one `AT TIME ZONE` anywhere in the code path behind that endpoint.

🛑 **What is still NOT decided, and it now gates the deploy rather than the notice.** §4 tells sales-processor the changeover happens on "the first HQ deploy that carries it" and that HQ will supply the date. That deploy has not been scheduled, and **`main` does not yet carry the change** — production still computes in `America/Chicago` (five literal sites in `inventory/handler.go` at `origin/main`). So the ordering is: send this, then promote and deploy, then tell them the date. Promoting before sending is the failure this notice exists to prevent.

---
---

# DRAFT MESSAGE BEGINS

**Subject: HQ inventory API — corrections to both contract documents, one that may have affected your payroll gate, and one upcoming change that needs a matching change on your side**

Hi,

We audited both HQ→sales-processor contract documents line by line against the shipped code on 2026-08-03. This is the first time that has been done since they were written. We found enough that was wrong that I want to walk you through it rather than just point at a diff.

Short version: **the code was fine throughout — the documents we handed you were not.** Both are now corrected, each with a full row-by-row audit section (`§0`) recording what was checked, what was confirmed accurate, and what was wrong, with the commit responsible in each case.

## 1. The one that may have actually cost you something

**`/period-summary`'s `completeness.pending_review_ids` — the field your payroll gate keys on — changed twice on 2026-06-06 and we told you about neither.**

You were handed this rule: a pending receipt blocks payroll if its `created_at`, read as a Chicago calendar date, falls in the period.

What actually shipped that day was two changes pulling in opposite directions:

- **`cf959bd` widened it.** The filter became `COALESCE(event_date, created_at::date)`. A receipt discovered late — a May 29 purchase ingested on June 2 — used to fall outside the May window and *not* block May payroll. After this change its extracted `event_date` puts it inside the window, and it **does** block.
- **`d41faef` narrowed it, much further.** Two clauses were added: the row must be a COGS-category transaction **and** must have `reason = 'no_attachment_on_bank_tx'` — i.e. no receipt attached at all. Everything else that used to block stopped blocking.

**Where the money that stopped blocking actually went — there are three cases, and only one of them moves a number you see.** We want to be exact about this because it decides where it is worth your time to look:

1. **COGS-category, receipt attached but the parser failed on it.** These stopped blocking **and** began contributing their full `ABS(bank_total)` to `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count` and `by_vendor`. This is the only bucket that moves money into your report.
2. **Anything whose Mercury category is not in our COGS allowlist — including transactions with no category at all.** Both our blocking query and our COGS query filter on that allowlist, so these rows block nothing **and contribute nothing**. They are invisible to every COGS figure we send you. The one place they remain visible is `tracked_bank_tx_ids`, which we deliberately do not filter — that is the field to diff against Mercury if you want to find them.
3. **COGS-category but with a null `reason`.** Our blocking filter is `reason = '…'` and our COGS filter is `reason != '…'`; against a NULL both are NULL rather than true, so these rows fall through **both**. They block nothing and contribute nothing either.

**The practical consequence for you:** since 2026-06-06 you may have received `ready: false` on runs where our published contract said you should have received `ready: true` — a blocked payroll run with no documented cause — and, in the other direction, `ready: true` on periods containing unreviewed receipts that the old rule would have caught.

I want to be precise about what we do and do not know here. **We know this became possible on 2026-06-06. We do not know whether it happened**, and HQ keeps no log that would tell us. If you have run history on your side showing unexplained gate failures since June 2026, that is where the answer is.

**We are not proposing to restate any past figures.** If you would rather we did, say so and we will work out what that means.

## 2. Things we have been returning without telling you (`/period-summary`)

Three response fields have been live since June 2026 and appeared in no document. All are **additive** — your decoder has been ignoring them, nothing of yours is broken:

- **`by_vendor`** — per-vendor COGS breakdown with trip counts.
- **`tracked_bank_tx_ids`** — every Mercury `bank_tx_id` we have touched for the period, in any state. Deliberately **unfiltered by category**. If you want to detect "Mercury has this transaction and HQ has not ingested it," this is the field for it.
- **`completeness.pending_review_details`** — the same rows as `pending_review_ids` but with vendor, date, amount and reason attached, so you can render a useful blocked-payroll message without a second call.

**And one input we never documented at all:** an env var, `HQ_COGS_CATEGORY_ALLOWLIST` (default `COGS`), which gates four of the response fields. The aggregate you receive is not "all purchasing" — it is "purchasing our operator categorised as COGS in Mercury." A miscategorised transaction is invisible to the COGS figures. It still shows up in `tracked_bank_tx_ids`.

Also worth flagging: **`purchase_event_count` no longer counts `purchase_events`.** It now includes unconfirmed pending rows too — but only the ones in case 1 above, i.e. COGS-category with a non-null `reason` other than `no_attachment_on_bank_tx`. A week with 4 confirmed receipts and 3 such rows in the review queue reports `7`; the same week with 3 uncategorised rows in the queue still reports `4`. And `cogs_excl_tax` includes those same case-1 rows at their full bank amount, which means the COGS number moved **up** on 2026-06-06 relative to what we documented.

## 3. `/menu-cogs` — this one is worse, and I am sorry

The menu-cogs contract did not drift. **Most of it was wrong the day we wrote it**, on 2026-06-04. It was written from the phase plan rather than from the shipped handler, and nobody diffed the two.

**Four of the nine per-menu-item fields have never matched what we return — but they are wrong in four different ways, and only one of them is a wrong name.** The distinction matters because each breaks your client differently. If you implemented against the struct we published — and we published a copy-paste-ready Go struct — then:

| We told you | We actually send | What kind of wrong |
|---|---|---|
| `name` | `menu_item_name` | **a wrong name** — your decoder finds nothing under `name` |
| `menu` | **nothing — this field does not exist** | **a phantom field** — we documented something we never built |
| `menu_subgroup: null` when absent | the key is **omitted** entirely | **wrong nullability** — present-and-null vs absent |
| *(not mentioned)* | `toast_master_id` | **undocumented** — we send it and never said so |

**So if you built it as documented, you have been rendering empty menu-item names since June 2026.** If that matches something you have seen and worked around, that is why.

Three more, all in the same document:

- **`ingredient_cost_total`'s formula was wrong.** We published a division by the sum of usage percentages across menu items. That division does not exist in our code — we compute `spend × (usage_pct / 100)` and leave the remainder in `unallocated_cogs`. The two only agree when an ingredient's percentages sum to exactly 100.
- **The row-selection rule was backwards in both directions.** We said "one row per menu item with > 0 sales." It is actually one row per menu item **with a recipe**, sales or not. So a menu item that sold with no recipe attached **does not appear at all** — which also means the acceptance scenario we gave you for that case (Scenario 2) could never have passed. If you wrote that test, it was failing because of us.
- **The reconciliation invariant between the two endpoints is broken.** We told you `sum(menu_items[].ingredient_cost_total) + unallocated_cogs ≈ period-summary cogs_incl_tax`, within rounding. That stopped being true on 2026-06-06 when `/period-summary` gained the category allowlist and the pending-row rollup and `/menu-cogs` did not. **The two now diverge in both directions by unbounded amounts.** If you have a reconciliation check on this, it is firing spuriously or silently passing for the wrong reason — please turn it off until we sort it out.

**One question I need to put to you rather than decide for you:** on the `name` / `menu_item_name` mismatch, we can either change HQ to send `name` as documented, or leave it and you adjust. Changing our side **fixes** a client built from the doc and **breaks** one built from the observed wire format, and we cannot see which you have. Tell us which you built against and we will move.

We can also start returning `menu` — the data is in our database, the endpoint just never selected it — if that is useful to you. That one is purely additive and breaks nothing.

## 4. One change still coming, and this one needs a matching change on your side

Everything above is about the past. This part is about a deploy that has not happened yet, and it is the only item in this message that asks you to do something.

**HQ's operating timezone is moving from `America/Chicago` to `America/New_York`.** The reasoning was simple: a payroll week and a food-cost week have to describe the same seven days, and we had been running two zones at once — our user-level default was already New York while the money queries had Chicago written into them as a literal.

**Status, stated precisely, because getting this wrong in either direction causes the problem:**

- We have **written and merged** the change.
- We have **not deployed it.** What is running in production today still computes in `America/Chicago`.
- It takes effect on **the first HQ deploy that carries it**, and **that deploy is not yet scheduled.** We will tell you the date. Afterwards it is recoverable exactly — it is the date migration `0072` first ran.

🛑 **Please do not ship your side on the assumption that we have already moved.** If you switch before we deploy, you create the same one-hour disagreement at every period edge that this change exists to end — just in the opposite direction, and while we are not looking for it.

### What actually moves, and it is narrower than you might assume

The zone enters our `/period-summary` query at **exactly one point**: a fallback that reads a pending receipt's `created_at` when the receipt parser could not extract a purchase date from the receipt itself. It applies to **pending, unconfirmed receipts with no parsed purchase date, and nothing else** — when a real purchase date exists, it wins and no zone is involved.

**The confirmed half of the endpoint has no timezone dependency at all.** `purchase_events.event_date` is a plain SQL `DATE`, compared without any cast. So the bulk of your COGS figure does not move.

But that one expression period-filters the pending rows, so where it applies it reaches the money as well as the gate: `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count`, `by_vendor` **and** `completeness.pending_review_ids`.

**Which direction it moves.** New York is an hour ahead of Central, so the day boundary lands an hour earlier in absolute terms. A pending receipt with no parsed purchase date, created between 23:00 and midnight Central, counts toward **that** day today; after the changeover the same row counts toward **the next** day. At a period edge that means such a row can drop out of one week and into the next. It is a shift, not a widening — a row leaves one end as another enters the other.

Realistically the affected set is small: only unconfirmed receipts, only those the parser could not read a date from, only those created in that one hour. We are flagging it anyway because **your week and our week have to agree on where they end**, and a disagreement here is invisible until a figure looks wrong.

### 🛑 A correction to something we told you about this already

On 2026-07-31 we added a timezone assumption to the menu-cogs contract document — assumption A10, and a matching line near the top — saying **`/menu-cogs` shares `/period-summary`'s date semantics and moves with it on the changeover deploy. That was wrong.** If you read the document between then and now, that is what it told you.

**`/menu-cogs` has no timezone dependency whatsoever.** There is not a single timezone cast anywhere in the code path behind it. It will not move on the changeover deploy, and nothing on your side needs to change for that endpoint. Both contract documents now say so.

We caught this in the audit described above, which is why this message is one message: sending the timezone note on its own would have handed you a fresh error while apologising for old ones.

### What we are not doing

**We are not restating any weekly COGS or payroll figure produced before that deploy.** They were already acted on. One boundary moves, once, on a known date — a future reader comparing two weeks either side of it will find exactly that, and this is the explanation.

## 5. What we changed, and what we did not

**We changed no code.** Every correction moved the *documents* to match the shipped behaviour, so that what you are reading is now true. We did not quietly alter any endpoint to match the old prose — that would have changed behaviour under you without warning, which is the thing that caused this in the first place.

The one exception worth naming: on `/menu-cogs`, correcting the doc to match the code arguably makes it *honest* rather than *right* — several of those items are places where the document described more useful behaviour than we implemented. Those are recorded as open questions in each document's `§8` and we would rather decide them with you than around you.

## 6. Why this went unnoticed for eight weeks

Both documents claimed their integration tests were "the executable proof that the HQ side matches this contract." They are not, and that claim is what stopped anyone looking. The tests decode into the same Go structs the handlers marshal — so a field name the *document* gets wrong is invisible to every one of them. The suite was green the entire time. Both documents now say so plainly, and we have filed work to add a test that asserts on raw JSON keys instead.

Happy to get on a call about any of this — particularly §1, the `name` question in §3, and the changeover timing in §4.

— HQ

# DRAFT MESSAGE ENDS

---
---

## Provenance

Every claim above traces to a row of the audit tables in the two contract documents. **Counting unit: one row of a `§0` table = one audited unit, whatever line range it anchors. Count the rows; these numbers match.**

- `docs/contracts/inventory-period-summary.md` §0 — **47 rows**: 24 confirmed byte-accurate, 3 not verifiable from HQ, 1 operational (not a code claim), 18 drifted/wrong/stale, 1 input never published at all. (24 + 3 + 1 + 18 + 1 = 47.)
- `docs/contracts/inventory-menu-cogs.md` §0 — **64 rows**: 35 confirmed byte-accurate, 1 not verifiable from HQ, 1 superseded, 27 wrong/unreachable/broken/stale. (35 + 1 + 1 + 27 = 64.) Of the 27, **5 are drift and 22 were never true**.

**Combined: 111 rows audited, 45 of them wrong** (18 + 27) plus the one missing input.

🛑 **These figures replace an earlier set — "76 rows audited, 26 wrong" (32 and 44 per document) — which were wrong in every component.** The audit ran in two passes on the same day. The first pass left three regions of each document unaudited while claiming full coverage, and its stated totals did not reconcile against its own tables. The second pass closed the gaps, added 15 rows to one document and 17 to the other, and restated the counting unit so the totals are checkable by counting. Nothing in the draft above depended on the earlier numbers, but they appeared in this block and in the roadmap card, and both are corrected.

Commits named in the draft: `cf959bd`, `d41faef`, `a726029`, `518a395`, `f730485`, `1c260f0` — all 2026-06-05/06.

Response shapes quoted were **observed** by marshalling `inventory.PeriodSummary` and `recipes.MenuCOGSResponse` at HEAD, not transcribed from the handlers.

## Reminder

**This file is a draft. It has not been sent. Sending it, and deciding whether past `ready:false` runs need reconciling, are the operator's calls.**
