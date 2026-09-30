# Bugs

> Scaffolded by `night-crew init`. Known bug classes and how the QA/E2E stage
> should treat them for this repo. Author before the first evening.

First authored 2026-09-29 from an attended operator walkthrough of the Inventory
review form on a phone (the 34 commits between `3fe14aa` and `6f79087` on `dev`).
Each class below is stated as what the QA/E2E stage should DO about it.

## Regressions — must be caught before they ship

**Seventeen inventory.spec tests fail on `dev` today, and did on yesterday's hub
commit too.** Found 2026-09-30 by the BI card's footprint (WO-2b, branch
`wo-bi-hub`), then reproduced by title on `dev` at `8ec1882` AND on the hub
commit `538c1ba` in fresh `:5434` databases — so it is not the BI card, and it
is not new code since the hub either. The set: every `stubPending(...)` test
(parse-failure cause, Retry parse offered, stale re-parse flag, queued
re-parse, pre-filled line items, transient failure), the four sync/cancel
tests, "a running sync shows elapsed time", "back link navigates to HQ", "the
amount is centred on the card", "navigating with newItem hash prefills", "create
new item via Items tab" (the auto-opened edit form never appears; reproduced on
`dev` alone), and "user can set store_location" (flaky). The saved page snapshot for the pre-fill
test shows the Receipts page fully rendered with **"Needs review (0)"** and
**"No purchases yet"** — the stubbed pending row never reaches the page, though
the route regex matches the URL the page fetches. The sync-chip test sees
`#sync-receipts-chip` empty and hidden, i.e. its mocked running status is not
applied either. Two routes mocked with `page.route(...)` before `page.goto`,
neither taking effect, is the common shape; a clock-dependent filter (today is
the first run after 2026-09-29's changes) is the other candidate. NOT
diagnosed — recorded so the next inventory-touching card does not read these
sixteen as its own.
*Handling:* run `npx playwright test tests/inventory.spec.js -g "a running sync
shows elapsed time|a parse failure states its cause"` on a clean `dev`
worktree before blaming a card; if they are red there, the card is not the
cause. Diagnose from the snapshot (route hit count vs. page fetch), not from
the assertion.

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
*Tail, same day:* the server-side fix left the page's optimistic update behind —
the click handler still did `row.parse_error=''` (and `items=[]` on mismatch),
so "Why:" vanished on tap until the next reload, and the koi test that asserted
the cause survives was red on `dev`. An optimistic update is a second copy of
the server's write; when the write changes, grep the client for its mirror.
The same card also told the operator to "Tap Retry Parse (All Receipts)" — an
instruction standing in for a fix, because the per-card retry ran a Mercury
sync that cannot reach a charge older than the 14-day lookback. Replaced by
`POST /purchases/pending/{id}/reprocess` (the storage re-read, one row) and
`tests/inventory.spec.js` — "Inline reparse (260929)".

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

**A test that reads a UI label as proof of identity.** purchasing.spec.js
"Item card Setup deep link" asserted the PO card's label equals the Setup Name
box, as a proxy for "the right item opened". That equality only ever held
because no item had a nickname; the first feature to give an item a display
name distinct from its description (promoted aliases, 2026-09-29) tripped it,
and the test picked `items[0]` from a description-sorted list, so the seeded
row landed first. Found by a peer session; fixed by asserting the edit form's
`data-item-id` and checking the Name box separately.
*Handling:* identity is an id. A test that needs "the right record opened"
asserts the id (or a seeded unique tag), and asserts the label as a second,
separate fact. Any change that splits what a thing is CALLED from what it IS
should grep the specs for label-equality assertions before it lands.

**Assertions against a database nobody resets.** `playwright.config.js` runs
`workers: 1`, and the E2E DB is reset once per RUN (`scripts/reset-e2e-db.js` in
`webServer.command`), never between tests — so every test inherits everything
the tests before it seeded. Assertions written against global counts are
therefore order-dependent: FR-11 asserted that its freshly seeded event sat on
UNFILTERED page 1 of `/purchases` and not on page 2 — true only while fewer than
50 events dated on or after its seed date existed at that point in the run (26
after a full run on 2026-09-29, and `retries: 1` re-seeds on every retry). Its
`toHaveCount(1)` on `$10.00` was NOT this class — the vendor filter scopes that
read server-side to a vendor the test created — but two other things in the
same test were defects: the list query ordered by `event_date` alone, so a page
boundary inside a same-date group could show a row on both pages or on neither
(a real pagination bug, not a test one — 51 same-date rows reproduced it red
first), and a non-retrying `cards.count()` snapshot raced the vendor-filter
reload and read 0 mid-render, which is the B-156 shape. Both fixed 2026-09-29:
FR-11 pages 51 same-date rows for its own vendor, and `ListPurchaseEventsHandler`
orders by `event_date DESC, created_at DESC, id DESC`.
On 2026-09-29 four full runs at the same commit produced four different
failing sets (view-receipt, PDF-iframe, Confirm-disabled ×2, FR-11, No-photo-
badge, Setup-deep-link, alias-chips — every one green when run alone), and the
set shifts whenever anyone adds a test that seeds. Found by a peer session's
attribution runs while two sessions were also racing on the same DB (see the
shared-tree memory) — the two effects look identical from one run's output.
*Handling:* a test that asserts a count must scope it to what IT seeded (a
unique vendor/tag in the locator, or a filter that only its rows satisfy), never
to the whole list. Before blaming a change for a multi-spec red, run each red
alone: a test that passes in isolation and fails in the suite is this class,
not a regression — but it is still a defect in the test, not weather. And a
LIMIT/OFFSET query needs a total order: `ORDER BY` a date alone is a flake
generator in the backend, not in the suite.

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
