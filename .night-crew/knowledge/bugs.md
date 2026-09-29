# Bugs

> Scaffolded by `night-crew init`. Known bug classes and how the QA/E2E stage
> should treat them for this repo. Author before the first evening.

First authored 2026-09-29 from an attended operator walkthrough of the Inventory
review form on a phone (the 34 commits between `3fe14aa` and `6f79087` on `dev`).
Each class below is stated as what the QA/E2E stage should DO about it.

## Regressions — must be caught before they ship

**Silent numeric coercion presenting as a valid state.** `fmtMoney` did
`Number(n)||0`, so a malformed price (`"1.90.00"` — what you get typing `1.90`
into a pre-filled `0.00`) rendered a confident `$0.00` subtotal. Because
`NaN > 0.01` is false, the mismatch check then read **"Amounts match. Ready to
confirm."** and enabled Confirm on a receipt that did not balance. Introduced
and caught inside one session — by the operator photographing a field mid-typo,
NOT by any test, because every test used well-formed input.
*Handling:* any field that feeds an arithmetic gate needs a malformed-input case,
not just empty and valid. A non-finite total must render an em dash and disable
the gate, never a plausible number. See `tests/inventory.spec.js` — "a malformed
price never reads as $0.00 or as a matching total".

**A control that hides its own affordance.** "Retry parse" re-armed a row by
NULLing `parse_error` and emptying `items`, which were also the two fields
`renderPendingCard` gated the button on. Pressing it once removed the button,
the stated cause, and the pre-filled line items — so the row that most needed a
second reading was the one that could no longer ask. Fixed by `retry_requested_at`
(migration 0079) separating the signal from the data.
*Handling:* when a write re-arms a workflow, check whether the fields it clears
are also the fields the UI reads to offer that workflow.

**A cached answer masquerading as a live one.** `/api/v1/health` matches
`build-sw.js`'s `/api/` NetworkFirst rule, so on a flaky phone the SW served a
stale `storage: ok` from `api-cache` — the same account showed the storage
banner on desktop and nothing on mobile. Already recorded for the version line
in `build-sw.js` as "best-effort, not guaranteed live"; the banner inherited it.
*Handling:* any health/freshness probe must bypass the SW (`cache:'no-store'`
AND a unique query string — either alone has been observed insufficient on iOS).

**A UI change that moves a tap target under existing tests.** Rendering "Retry
parse" on every pending card put a full-width button mid-card, where a bare
`locator.click()` lands. 19 tests that opened the review form by clicking the
card centre broke at once. `openSeededReviewForm` had carried that warning for
parse-error cards; the change made it universal.
*Handling:* tests open a pending review form via `.event-vendor`, never the card
centre. A mass timeout in one screen's suite after a UI change is this shape.

## Not regressions — do not chase

**`internal/sync` failing with "THE SYNC SUBSTRATE COULD NOT BE RESOLVED".**
The RLS suite refuses to skip when no Supabase substrate is up (B-36). This is
the suite working as designed and is NOT a delivery failure — but it is also NOT
a pass. Report it as **unrun**. `HQ_SYNC_SUBSTRATE_OPTIONAL=1` silences it and
the suite itself says such a run is not evidence; do not set it to get green.

**`user can set store_location from Setup tab edit form` (tests/inventory.spec.js).**
Accepted-known flake, ~2/3 on repeat, present in runs predating 2026-09-29.
Not yet diagnosed. A single red here is not a signal; two in a row is.

**Go suite printing `ok` with no `DB_TEST_URL`.** Every DB-coupled test skips and
the package still reports `ok`. Check test COUNTS, not the `ok`/`FAIL` word.
