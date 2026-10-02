# Decisions needed — run `20261002`

> **RESOLVED 2026-10-02 — recorded as `.night-crew/knowledge/ledger.md` T-62, decisions 194–200.**
> All six forks settled with the operator at morning triage; the run branch is merged to `dev` at
> `5753ddf`. Kept as the analysis record — the reproductions and option tables below are the
> evidence the decisions were made against, and three of them were corrected at triage:
>
> - **D-1 → decision 194.** *Teach the merge about the new tables, AND blank the link as a
>   backstop.* 🛑 **D-1's reachability claim here is FALSE** — no production frontend calls the
>   dish-merge endpoint, so this is API-only, not the UI regression this file and HANDOFF.md
>   assert. Re-pointing is also the house convention, not a deviation from it. `0085`'s PII
>   erasure block decided with it, as this file asked.
> - **D-2 → decision 195.** *Tumbling 10-minute bucket + unique index on `(short, ip_hash,
>   bucket)` + `ON CONFLICT DO NOTHING`.* Taken as an engineer-level call; the operator was
>   offered it and handed it back as plumbing. Reproduced independently at 3/7/8/8/8 rows.
> - **D-3 → decision 196.** *BLOCKED ON EVIDENCE, value untouched, check named:* read one real
>   Toast export's `Opened` against a known order time. Only the ±30-minute hint is at risk.
> - **D-4 → decision 197.** *Hold attribution, state the limit on the page; the real question —
>   whether Supabase should be the single source of truth for the attribution spine — goes to the
>   roadmap round,* under decision 187's own "revisable at the first morning triage" rider. 🛑
>   **This file's "closes by itself later … no further code change" is an OVERSTATEMENT** —
>   `mirror.go` writes the campaign tag as a literal `NULL` and omits it from its `ON CONFLICT`
>   update list, so nothing in the tree can ever populate either arm. **Q-KR2 and the per-slice
>   half of P-KR3 are NOT MEASURABLE this cycle** — now a recorded consequence of a decision
>   rather than an open defect.
> - **D-5 → decision 198.** *Ship both figures* — `orphan_rate` keeps the literal specification
>   meaning so the 10% line stays comparable, and a second named figure carries the wider signal.
> - **D-6 → decision 199, and decision 191 is AMENDED.** *A genuinely offline phone serves an
>   unseen code, flags it, and the owner reviews it; behaviour at the window does not change.*
>   🛑 **D-6 is substantially NARROWER than presented here.** The clause everyone argued about
>   distinguishes "names a campaign" from "names no campaign" — and **no code naming no campaign
>   can exist**, since `qr_codes.campaign_id` and `public.codes.campaign_id` are both
>   `not null references`. The only real population is a code the device has **never seen**,
>   which `submit-flow.js:391` never puts through the predicate at all and which ratified
>   decision 166 already governs.
>
> Two things found at triage that are not in this file: **`[SP-03b]` is a second false gate**
> (B-465) leaving decision 191's distinguishing branch guarded by nothing, and **an online phone
> never verifies a code it has not synced** (B-468), promoted by the operator to the next slate.

Two open operator forks, both raised by Card 1's G6 adversarial review, both routed through
`night-crew decisions log --run 20261002` and both returned **`verdict: park`** — *escalated,
top severity, voted top by all three roles; "top-severity questions are the operator's — no
number of citations clears them."* Recorded in
`.night-crew/knowledge/decisions/20261002.jsonl`.

Neither is in Card 1's narrow PARK note. Both are nonetheless the operator's by the ladder's own
verdict, so the run did **not** choose. Card 1 **merged anyway** (`40b5846`) — see the bottom of
this file for why that was the orchestrator's call and not a silent override.

---

## D-1 · Migration 0083's `menu_items` foreign keys block the Recipes dish merge

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h1-campaign-codes-api` · **Reproduced:** yes, by G6, against both FK shapes

`0083_campaigns_admin.sql:40` and `:57` declare `campaigns_admin.item_id` and `qr_codes.item_id`
as `references menu_items(id)` with **no `ON DELETE` clause**. Both pre-existing `menu_items`
FKs (migrations 0061, 0062) are `ON DELETE CASCADE`.

**The failure, reachable today via the API:** a manager `POST`s a campaign with
`item_id=<wings>`; someone then opens Inventory › Recipes › By dish and merges "Wings" into
"6pc Wings"; `recipes/repository.go:162`'s `DELETE FROM menu_items` raises
`23503 campaigns_admin_item_id_fkey`; the merge returns an **opaque 500** and stays broken until
the campaign is deleted. G6 reproduced the exact UPDATE/DELETE sequence.

🛑 **This becomes reachable through the UI the moment Card 2 (`campaigns-tab-ui`) lands**, which
makes choosing an `item_id` a routine operator action.

**Why it is yours and not ours:** the `0083` DDL is **verbatim** from the signed handoff §4, so
every available fix deviates from the signed spec somewhere. §4 omitted the clause rather than
choosing none — it never considered the merge path.

| Option | What you would see |
|---|---|
| **`ON DELETE SET NULL` on both FKs** *(run's recommendation)* | The dish merge succeeds. The campaign and its QR codes survive with a null item; `utm_content` degrades to empty on an already-printed code. Adds a clause §4 omitted. |
| `ON DELETE CASCADE`, matching 0061/0062 | The merge succeeds, but **silently deletes the campaign and its `qr_codes` rows** — a dish rename destroys scan history. |
| Re-point `item_id` inside `MergeMenuItem` | `0083` stays byte-verbatim to the spec, but the fix lives in the `recipes` package and every future table referencing `menu_items` must remember to join the merge path. |

**The run's recommendation:** `ON DELETE SET NULL`. A campaign must not be destroyed by a dish
rename (rules out CASCADE) and a dish merge must not be blocked by a campaign (rules out the
status quo). Losing one UTM parameter is the cheapest honest degradation.

---

## D-2 · The 10-minute scan dedupe is not atomic, and `scans` is P-KR3's denominator

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h1-campaign-codes-api` · **Measured:** yes, 5 runs

`landing.go:306-312` dedupes with `INSERT … SELECT … WHERE NOT EXISTS` — a read-then-write under
READ COMMITTED. G6 drove **12 concurrent `GET /q/{short}`** with one identical
`CF-Connecting-IP`: **8 `qr_scans` rows in 4 of 5 runs** (1 row only on the cold-pool first run).

Card 1 enforces dedupe at **write** time only — a deliberate, stated choice so the §5 metric is a
plain `count(*)` with one definition in one place. The consequence is that **no read-side
correction is available** to Cards 4 and 5, which consume `scans`. `scans` is the **denominator of
the orphan rate P-KR3 is graded on**, so the KR inflates by an unknown factor under concurrent
double-taps.

| Option | What you would see |
|---|---|
| **Tumbling 10-minute bucket column + unique index on `(short, ip_hash, bucket)` + `ON CONFLICT DO NOTHING`** *(run's recommendation)* | The count is correct under concurrency. Adds one column and one index that §4 does not enumerate. |
| Leave as measured | `scans` over-counts; P-KR3 is graded on an inflated denominator; later cards inherit a number they cannot correct. |
| Advisory lock around the existing read-then-write | No schema change, correctness restored, at the cost of a serialization point on an unauthenticated public route. |

**Note on grading:** if you take option 2, P-KR3's orphan rate for this cycle should be read as an
**upper bound**, not a measurement.

---

## D-3 · `toast_orders.opened_at` is parsed in a zone nothing in the repo establishes

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h3a-toast-orders-and-mirror` · **Blast radius quantified by:** card `h3b`

`backend/internal/toast/orderdetails.go` hardcodes `America/Chicago`. Its justification comment
claimed purchasing and the recipes drift check already use Chicago — **G6 proved that false**:
`purchasing/service.go:64`, `recipes/scheduler.go:59`, `recipes/cost.go:97` and
`inventory/handler.go:27` all read `users.DefaultTimezone` = **`America/New_York`** (ledger T-26
decision 83, migration `0072_app_timezone_new_york.sql`). Chicago is only their *pre-changeover
production* behaviour, which 0072 ends on the next deploy. And **nothing in the repo establishes
which wall clock Toast exports `Opened` in** — spike 02 parsed it naive, measuring digits and not
offset.

The comment is corrected and carries a `TODO` naming this decision. **The value is untouched**,
because settling it needs a real export sample the night cannot read.

**What is NOT at risk** (quantified by Card 4, which consumes the field): `business_date` is
`opened.Date()` of the wall-clock string and so is **zone-independent** — G6 constructed the 00:30
and 23:50 cases and both land on their own calendar day under any zone. So the **matched** bucket,
revenue, discount, net, every slice, **and the orphan rate P-KR3 is graded on do not move.**

**What is at risk:** only the ±30-minute nearest-order suggestion. Under a one-hour offset every
real order falls outside the window. Card 4 hardened this without touching the fixed rule — the
±30-minute rule stays rung 1, and a second rung matches on the zone-independent `business_date`
and labels itself `basis:"business_date"` with `gap_seconds`, so an offset surfaces as a
labelled ~3600s hint instead of as silence. It **cannot** reclassify a bucket: `bucket()` reads
the order number, never the suggestion.

| Option | What you would see |
|---|---|
| **Confirm the zone from a real Toast export sample, then set it** *(run's recommendation)* | The only option that can actually be right. Needs an attended look at the export. |
| Switch to `users.DefaultTimezone` (New York) | One constant, one source of truth, consistent with decision 83. If Toast exports Central, `opened_at` is then an hour late instead of an hour early. |
| Keep Chicago | Preserves today's pre-changeover behaviour, but leaves a literal disagreeing with the repo's own constant — which is how this became invisible. |

---

## D-4 · 🛑 Campaign attribution cannot resolve on live data — the Stats slices will read "unattributed"

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** card `h3b`
`reconciliation-and-stats-engine`, from inside the card that consumes it · **This is the night's
most consequential finding.**

Card 3's `mirror.go` states that *"H3b resolves campaign through `code_id` when it needs it."*
**It cannot.** `scan_attempts.code_id` is the **Supabase `public.codes`** id — the per-customer
redemption-token row (`supabase/migrations/20260904000100_qr_attribution_spine.sql:45`) — and
**HQ Postgres holds no copy of that table.** `qr_codes` is a different id space, keyed by `short`.
Nothing maps `code_id → campaign_id`.

**Consequence on real data:** every accepted attempt is unattributable. The **campaign** slice
reads `unattributed`; **channel** and **item** read `direct`; `discount_unknown_rows` becomes the
whole set. Revenue and discount stay **correct per row**, and the funnel, the queue and the orphan
rate are all fine — but the per-campaign / per-channel / per-dish attribution, which is the point
of the Stats tab and what *"spend per channel and per dish"* means, collapses to one bucket.

**What the night shipped anyway, so this closes by itself later:** Card 4 implemented the
`COALESCE(mirror.campaign_id, qr_codes.campaign_id)` ladder joined on `qr_codes.id = code_id`, so
attribution starts working **the instant either side is populated** — no further code change. It
also pinned the degraded shape as behaviour (`TestUnresolvableCodeIDBucketsAsDirectAndCountsAsUnknown`)
rather than leaving it to be discovered, and made the unknown explicit on the wire via
`discount_unknown_rows` (non-zero ⇒ the discount total is a **floor**, not a figure).

| Option | What you would see |
|---|---|
| **Populate `campaign_id` in the mirror at write time** *(run's recommendation)* | Resolution happens on the Supabase side, where `public.codes` actually lives. One card, no new HQ table, no second copy of customer tokens, and Card 4's ladder consumes it the moment it appears. |
| Mirror `public.codes` into HQ as a second keyset poller | HQ gains the missing id space and can attribute historically — at the cost of a second mirror table, a second poller, and another copy of customer-token rows in HQ. |
| Ship Stats with attribution degraded and say so on the page | The funnel, revenue, discount and orphan rate are correct, so the tab is useful today; the three slices read `unattributed` until a later card closes it, and the UI says so rather than implying a campaign earned nothing. |

🛑 **Grading note:** **Q-KR2 and the per-slice half of P-KR3 should be read as NOT YET MEASURABLE on
live data**, independently of whether Cards 4 and 5 landed. The totals are measurable; the
attribution is not.

### D-1's defect class recurs in `0085` (routed here, not opened as a new fork)

Card 6's G6 proved on `:5434` that `0085`'s FKs are `NO ACTION` **both** ways: deleting a
`qr_codes` row a subscriber first-touched is **blocked**, and deleting a subscriber is blocked by
`subscriber_events`. The DDL is **§4 verbatim**, so this is inherited from the signed spec rather
than invented — but it is D-1's shape in a new table, and on a **PII** table it means a
right-to-erasure request cannot be served by a plain `DELETE`. Decide it with D-1.

---

## D-5 · Which scans count as orphans? The rate reads 40% or 60% on the same data, against a 10% line

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h3b-reconciliation-and-stats-engine`

Slate and handoff §5 define the metric as **orphans ÷ accepted, counting declines except
`duplicate_scan`**, and define an **orphan** as an accepted attempt with **no order number**.
Card 4 implemented the numerator as `!matched && !dup_decline` (`reconciliation.go:219-228`),
which **also counts `unmatched` rows** — where an order number *was* typed but no Toast order
matched it.

On G6's fixture that is **3 of 5 = 60%** where the literal reading gives **2 of 5 = 40%** —
**against a 10% threshold, on the figure P-KR3 is graded against.** The spike's fixture models no
unmatched row, so it never disambiguated this, and the card did not surface the choice.

**Tonight's handling:** the rule is **unchanged** (changing it is decision-190 territory and a park
in its own right). The card was instructed instead to make the figure **self-describing on the
wire**, state both numbers in its merge-intent, and **pin the current numerator in a test** — so
whichever way you rule, the change is a deliberate edit to an asserted value rather than a silent
drift.

| Option | What you would see |
|---|---|
| **Literal spec: orphans only (+ declines except `duplicate_scan`)** *(run's recommendation)* | The lower number, and the 10% line keeps the meaning it was set against. An `unmatched` row is its own bucket with its own queue section and its own suggestion, so it is already visible elsewhere. |
| As shipped: every accepted attempt not matched and not duplicate-declined | The higher number. Arguably truer to the metric's *purpose* — a scan nobody has tied to a sale is unreconciled whether or not someone typed a number — but wider than the words say. |
| Ship both | `orphan_rate` keeps the defined meaning for the threshold, and a second named figure carries the wider health signal. |

🛑 **Do not read the current `orphan_rate` against the 10% line until this is settled** — the number
and the threshold may not be measuring the same thing.

---

## D-6 · Ratified decision 191 contradicts itself, and a card had to pick a reading to ship

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** card
`h6-scanner-polish` · **Not a park the card could take** — the slate states B-436's shape is
**DECIDED (191) and is NOT a park**, so the card was required to implement it and found the text
would not resolve.

**The contradiction.** Decision 191 carries two clauses that do not agree about the no-policy-source
case: *"no offline override for **any** code"* versus *"**uniform** with the source's own
predicate."*

**What shipped — the uniform reading**, with both arms pinned by tests (`[SP-03]`, `[SP-03b]`, and
two `campaigns-harness.mjs` leg-3 checks) so the choice is visible rather than implied:
- a code that **names a campaign** → refused when no policy source resolves;
- a code that **names no campaign** → stays as ratified **decision 166** left it.

**The card's grounds, which are worth weighing:** reading *"any code"* literally would **repeal
decision 166** on precisely the devices least able to recover — a crew phone with no policy data —
and that is an affordance withdrawal nobody asked for. B-436's own harm statement is about every
**known** code.

🛑 **Its G6 went further and read the ledger itself: the shipped reading is not merely defensible,
it is the ONLY one consistent with ratified decision 171** (`ledger.md:3912`), which rules
explicitly that 166's `unknown → false` covers genuinely-unknown **codes**, and that reading it
broadly *"would convert an operator's narrow yes into standing cover."* So the literal *"any code"*
reading would contradict **two** ratified decisions, not one. G6 also confirmed by reading the code
that neither untouchable is weakened — the B-432 predicate line is untouched and both of the card's
changes move strictly **toward** refusal.

**One honest consequence the card flagged:** B-436 makes a population **strictly smaller**. Rows
shaped `(unverified_code=false, offline_override=true, policy_unresolved=true)` — a *known* code
force-submitted offline on a no-source device — are now refused and never written. Fewer
unauthorised overrides; `policy_unresolved = true` still means exactly "nothing was resolved".

| Option | What you would see |
|---|---|
| **Keep the uniform reading as shipped, and amend 191's wording to say so** *(run's recommendation)* | A crew phone with no policy data refuses the scans that name a campaign and behaves as 166 ratified for the rest. The contradiction stops recurring because the text is fixed, not re-derived by the next card. |
| Literal "any code" | A crew phone with no policy data refuses **every** scan. Strictly fewer unauthorised overrides, at the cost of withdrawing an affordance decision 166 ratified, on the devices least able to recover. |
| Amend the wording first, then restate the behaviour | Settles reading and text in one act rather than leaving two clauses that disagree. |

🛑 **What is NOT in question:** the `requires_online = true` refusal and the B-432 fail-closed
predicate are **untouchable** by the card's own PARK note, and weakening either would have been a
park rather than an interpretation. Card 7's G6 is checking the **code** against that, independently
of this wording question.

---

## Not decisions — recorded so triage does not mistake them for forks

- **Four LOW/INFO G6 findings on Card 1**, all engineer-level and none parked: the public landing
  `503`s and logs at ERROR on a malformed `short` (`landing.go:256` — unauthenticated, unthrottled
  log flood; the admin routes have the analogous `isBadUUID` guard, the landing has none);
  `PATCH /codes/{id}` accepts an undocumented `item_id` and answers `404 code_not_found` for a code
  that exists (`codes.go:325`), and its `COALESCE` means `landing`/`placement`/`variant` can never
  be cleared back to NULL; one code is minted per channel **entry** rather than per distinct
  channel (`campaigns.go:191`); and `live()` ignores `starts_at`/`status='scheduled'`
  (`landing.go:212`), unreachable today but live the moment Card 2 adds scheduling. These are
  follow-ups, not questions.
- **Card 1's three non-baseline Playwright reds are flake, not a fork.** Re-run green confined at
  the card HEAD twice, independently (orchestrator + G6); one baseline red went green in the same
  run, so the set flakes both ways under load. No causal path from `0083` or the `main.go` mounts
  to the Inventory/Recipes frontend surfaces was found by either reviewer.
- **The `dev` browser baseline is 24 reds, not the 4 of B-433.** Measured tonight on a detached
  `dev` worktree at `c4f6db4` (24 failed / 965 passed). Mostly the 17-test undiagnosed cluster
  `bugs.md` already carried; **four reds were new to the record** and are now named there
  (`09c6586`). Not tonight's, and no card owns that surface.
- **Two gate-ladder defects were corrected mid-run** (`dbbc1a8`), not parked: the per-leg isolation
  stanza never stated that `HQ_RLS_TEST_DB` must match `^hq_rls_[a-z0-9_]+$`, which cost the
  base-reds leg a discarded run whose three `internal/sync` reds read exactly like a substrate
  defect; and G4's precache count read **43** against a tree holding **48** — the second recurrence
  of that staleness in that row, which would have let a genuine drop to 43 read as the documented
  number.

---

## Why Card 1 merged with D-1 and D-2 open

Stated plainly so triage can disagree with it. The ladder parked **which fix to apply**; it did not
rule that the card's delivered scope was wrong. Card 1's five `done_when` tests, its DDL-vs-§4
equivalence, its Down round-trip and its live projection were each independently re-verified by G6
in a fresh context. Against that, parking Card 1 would have ended the night at **zero** merged
cards, because the remaining six structurally cannot start without `0083` and the `marketing`
package seam (the launch prompt's own instruction for that case is to close out early). Nothing
deploys tonight: `main` is untouched, the deploy is independently held, and no UI yet reaches the
`item_id` path that D-1 needs. Both findings carry a reproduction, three options and a
recommendation, so each is a short conversation at triage rather than an investigation.
