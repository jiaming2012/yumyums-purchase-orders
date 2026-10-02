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

*Re-measured 2026-10-01 at the launch of run `20261002`* — the base-reds leg this
entry's handling note prescribes, run for real on a detached `dev` worktree at
`c4f6db4` (`logs/base-pw.log`, one summary block, `EXIT_PW=1`): **24 failed / 965
passed in 34.8m.** The Go suite on the same tree is **clean** (368 top-level tests,
0 fail, 2 opt-in skips) — so this is a browser-surface condition only.

The 24 decompose as: this entry's cluster, still red and still undiagnosed; the
**three** surviving reds of B-433's stale four (`SYNC-FC-01`, `SYNC-RF-01`,
`SYNC-RF-02` — `B1-XT-01` passed, consistent with 20260906-2 and 20260907);
`store_location` (B-453, named flaky above); and **four this record did not carry**:

| Test | Note |
|---|---|
| `tests/receipt-carousel.spec.js:54` — single attachment renders simple iframe overlay without carousel nav | new to the record |
| `tests/receipt-carousel.spec.js:123` — multi attachment renders carousel with prev/next and counter | new to the record |
| `tests/purchasing.spec.js:455` — No photo badge shows on checked item without photo and disappears after photo upload | new to the record |
| `tests/workflows.spec.js:1641` — Tab Persistence › inventory tab persists on reload | new to the record; names the Inventory tab, so plausibly the same hub cause |

Three of those four touch the same receipt/attachment and Inventory-tab surfaces as
the cluster, which is suggestive but **not diagnosed** — recorded, not concluded.
`inventory.spec.js:1404` (B-437) and `FILL-04` (B-443) were **green** on this base.

🛑 **This set of 24 is run `20261002`'s no-new-reds baseline.** A card tonight is
judged against it, never against green, and a red inside it is not that card's.

### One member of the cluster is now DIAGNOSED — `inventory.spec.js:2186` (Setup alias chips)

Named by Card 2's G6 on 2026-10-01, and it is **a real race in pre-existing Inventory
code, not load flake**. The test fails as `expect('.alias-chip').toHaveCount(2)` →
`Received: 1`, and it flips **both ways in isolation on a quiet box**, which load
contention cannot explain.

**Mechanism.** `ALL_ITEMS` has **three unsequenced writers**, all last-write-wins with no
request versioning: `inventory.html:2179` (`loadItems()`), `:2835` (the `DOMContentLoaded`
preload) and `:2550` (the alias handler's own refetch). The test does `reload()` →
`waitForLoadState('networkidle')` → `goTab(7)`, and **`goTab(7)` fires a fresh `GET /items`
that `networkidle` does not cover**. The Add handler then runs `POST /items/aliases` →
`GET /items` → `ALL_ITEMS` becomes the 2-alias snapshot → `refreshAliasChips()` paints 2
chips. If the *earlier* in-flight `GET` — issued before the POST — resolves **last**, it
overwrites `ALL_ITEMS` with the 1-alias snapshot and `loadItems()` re-renders the list from
it, producing exactly the observed count of 1. Timing-dependent on response ordering, hence
both-ways.

**Fix direction:** sequence or version the items fetch (ignore a response older than the
latest issued request), rather than retrying the test.

**Proven NOT attributable to any card tonight:** `playwright.config.js:65` sets
`serviceWorkers: 'block'` repo-wide, so a `sw.js` change has no path to it; `inventory.spec.js`
loads neither `marketing.html` nor `marketing/campaigns.js`; and with `workers: 1` and
alphabetical file order it runs **before** every spec added tonight, so none can pollute it.

*Handling:* treat a `:2186` red as this race until the fetch is sequenced. It is a product
defect in the Inventory page's state management, not a test defect — the test is correct to
expect 2. **Filed as B-459** with the reproduction recipe and fix direction.

**Reproduced first-hand, not merely reasoned about** (same G6, amended report): three isolated
passes at one HEAD on an otherwise idle box — `PASS_1_EXIT=0`, `PASS_2_EXIT=1`, `PASS_3_EXIT=1`,
each red with the identical shape (`.alias-chip` `toHaveCount(2)` → `Received: 1` at
`inventory.spec.js:2214`). **Final tally 1 green / 2 red: alone on an idle box it fails MORE
OFTEN THAN IT PASSES**, which puts it closer to a hard red than to a flake. **That settles it:
load contention is not the cause**, and "flaky under load" was the wrong label — the one this run
first applied, and corrected. (This entry first recorded the tally as "green then red" from the
first two passes; the third landed red and is folded in here rather than left to imply a 50/50.)

🛑 **The user-facing hazard is worse than the test failure.** A manager who adds a nickname
while the Setup list is still loading sees the chip **silently vanish**, while the server has
kept the alias. The write is durable; only the view lies — so the manager re-adds a nickname the
database already holds. That is the dangerous shape of this defect class, and it is why this one
is a product bug rather than a test nuisance.

**Reproduction recipe:** `npx playwright test tests/inventory.spec.js -g "Setup item editor shows
alias chips"` **alone, repeatedly.**

**Worth testing against the other 16:** if the rest of the cluster shares this writer-ordering
shape, one sequencing fix may close most of it. That is a lead, not a conclusion.

### 🛑 Method note: a ONE-SAMPLE baseline cannot classify a high-rate race

Established by a base-commit control on run `20261002` (G6, Card 2). The same test, three
isolated passes each, same box, same command:

| tree | pass 1 | pass 2 | pass 3 | rate |
|---|---|---|---|---|
| base `overnight-20261002` | red | green | red | **2 red / 1 green** |
| card HEAD `f29a8b7` | green | red | red | **2 red / 1 green** |

Identical rate, identical failure shape, with and without the card. Two consequences for how
any run judges a card, and both bit this one:

1. **The run's 24-red baseline is a single sample.** `:2186` was *green* in that one base run and
   *red* in the card's full suite, which is exactly why it was classified "non-baseline" and read
   as possibly the card's. For a test that fails ~⅔ of the time in isolation, **its presence or
   absence in any single full-suite run is a coin flip and carries no information about the
   card.** A single-run baseline diff is necessary but not sufficient.
2. **A single confined GREEN does not establish "flake, not regression" either.** It is the same
   error with the sign flipped: one pass of a ⅓-pass race proves nothing. Tonight
   `inventory.spec.js:2406`, `:2919` and `recipes.spec.js:216` were cleared as "flake pool" on
   exactly one confined green apiece — which is weaker evidence than it looked, and they may be
   ⅓-pass races rather than flakes.

**The standard that actually settles attribution** is the control G6 ran: repeat the test in
isolation on **both** the card HEAD **and** the base, several passes each, and compare the
*rates*. A mechanical exclusion argument (this diff cannot reach that code) is stronger still
when it holds — but when it does not, nothing short of a two-tree rate comparison distinguishes
"the card broke it" from "it was already broken this often".

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
