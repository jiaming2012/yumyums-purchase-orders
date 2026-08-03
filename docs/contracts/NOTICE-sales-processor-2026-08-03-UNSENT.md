# 🛑 UNSENT DRAFT — outbound notice to the sales-processor maintainer

| | |
|---|---|
| **Status** | **UNSENT. NOTHING HAS BEEN DELIVERED.** |
| **Drafted** | 2026-08-03, night-crew run `20260803`, card P6 `period-summary-contract-notice` |
| **Drafted by** | HQ (automated audit). **Not reviewed. Not approved. Not sent.** |
| **Sending is** | **the operator's act, and the operator's alone.** |
| **Channel** | undecided — the operator picks |
| **Supersedes** | nothing. This is the **second** notice owed to sales-processor. The first is the timezone notice from card A1 (2026-08-01), which is **also undelivered.** See "Sequencing" below. |

**To whoever reads this file next:** do not treat any part of it as communicated. The audit that produced it (`docs/contracts/inventory-period-summary.md` §0 and `docs/contracts/inventory-menu-cogs.md` §0) is merged into the contract documents; **the counterparty has not been told.** Until someone sends this, sales-processor is still working from the pre-audit contracts.

---

## Before sending — three things the operator must decide

The draft below is written to be sendable **as-is**, but three decisions are embedded in it and none is HQ's to make unilaterally:

1. **Whether to reconcile past runs.** The draft says HQ is *not* proposing a restatement. If the operator wants past weeks re-examined, §4 needs rewriting. **The audit deliberately makes no claim about whether any payroll run was actually blocked** — only that it became possible on 2026-06-06. HQ retains no log that would settle it.
2. **The `menu_item_name` / `name` question** (menu-cogs §8 Q2). The draft *asks* the counterparty rather than announcing a fix, because the right answer depends on what sales-processor actually built. If the operator would rather just change HQ's key, §3 becomes an announcement instead of a question.
3. **Whether to send this together with, or after, the A1 timezone notice.** See Sequencing.

## Sequencing

Two notices are now owed and **neither has been sent**:

- **A1's timezone notice** (drafted 2026-08-01) — the `America/Chicago` → `America/New_York` changeover, which requires a **coordinated two-repo release** and is therefore time-sensitive against the next HQ deploy.
- **This notice** — eight weeks of undisclosed contract drift, which is not tied to a deploy date but is larger.

They overlap: this audit **corrects** one of A1's own statements (A1 told sales-processor that `/menu-cogs` shares `/period-summary`'s date semantics and moves with it on the changeover deploy; it does not — `/menu-cogs` has no timezone dependency at all). **Sending A1's notice alone would propagate that error.** Recommended: send one combined message, or send this one first. *Operator's call.*

---
---

# DRAFT MESSAGE BEGINS

**Subject: HQ inventory API — corrections to both contract documents, including one that may have affected your payroll gate**

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

**Four of the nine per-menu-item field names have never matched what we return.** If you implemented against the struct we published — and we published a copy-paste-ready Go struct — then:

| We told you | We actually send |
|---|---|
| `name` | `menu_item_name` |
| `menu` | **nothing — this field does not exist** |
| `menu_subgroup: null` when absent | the key is **omitted** entirely |
| *(not mentioned)* | `toast_master_id` |

**So if you built it as documented, you have been rendering empty menu-item names since June 2026.** If that matches something you have seen and worked around, that is why.

Three more, all in the same document:

- **`ingredient_cost_total`'s formula was wrong.** We published a division by the sum of usage percentages across menu items. That division does not exist in our code — we compute `spend × (usage_pct / 100)` and leave the remainder in `unallocated_cogs`. The two only agree when an ingredient's percentages sum to exactly 100.
- **The row-selection rule was backwards in both directions.** We said "one row per menu item with > 0 sales." It is actually one row per menu item **with a recipe**, sales or not. So a menu item that sold with no recipe attached **does not appear at all** — which also means the acceptance scenario we gave you for that case (Scenario 2) could never have passed. If you wrote that test, it was failing because of us.
- **The reconciliation invariant between the two endpoints is broken.** We told you `sum(menu_items[].ingredient_cost_total) + unallocated_cogs ≈ period-summary cogs_incl_tax`, within rounding. That stopped being true on 2026-06-06 when `/period-summary` gained the category allowlist and the pending-row rollup and `/menu-cogs` did not. **The two now diverge in both directions by unbounded amounts.** If you have a reconciliation check on this, it is firing spuriously or silently passing for the wrong reason — please turn it off until we sort it out.

**One question I need to put to you rather than decide for you:** on the `name` / `menu_item_name` mismatch, we can either change HQ to send `name` as documented, or leave it and you adjust. Changing our side **fixes** a client built from the doc and **breaks** one built from the observed wire format, and we cannot see which you have. Tell us which you built against and we will move.

We can also start returning `menu` — the data is in our database, the endpoint just never selected it — if that is useful to you. That one is purely additive and breaks nothing.

## 4. What we changed, and what we did not

**We changed no code.** Every correction moved the *documents* to match the shipped behaviour, so that what you are reading is now true. We did not quietly alter any endpoint to match the old prose — that would have changed behaviour under you without warning, which is the thing that caused this in the first place.

The one exception worth naming: on `/menu-cogs`, correcting the doc to match the code arguably makes it *honest* rather than *right* — several of those items are places where the document described more useful behaviour than we implemented. Those are recorded as open questions in each document's `§8` and we would rather decide them with you than around you.

## 5. Why this went unnoticed for eight weeks

Both documents claimed their integration tests were "the executable proof that the HQ side matches this contract." They are not, and that claim is what stopped anyone looking. The tests decode into the same Go structs the handlers marshal — so a field name the *document* gets wrong is invisible to every one of them. The suite was green the entire time. Both documents now say so plainly, and we have filed work to add a test that asserts on raw JSON keys instead.

Happy to get on a call about any of this, particularly §1 and the `name` question in §3.

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
