# Merge intent — inventory-setup-races (K1, run 20261007)

Card: what a manager types or adds in Inventory Setup stays on screen (BACKLOG B-459 — a
nickname added in Setup that disappears; B-478 — a typed item name wiped by a late response).

## Shared files I touch (outside my own area)

- `inventory.html` — the card's subject. Only the Setup › Items block (`loadItems`,
  `gotoSetupItem`, `renderItemsList`, the `#items-list` click handler) plus the
  `DOMContentLoaded` preload and the review-form item modal's `GET /items` assignment. K4
  (scanner-refusal-seam-and-pick-feedback, track A, after K1) does not edit this file.
- `tests/inventory.spec.js` — four new tests `[IS-01]`…`[IS-04]` appended as ONE new
  `test.describe('Setup — late responses (K1)')` block at the end of the file. No existing
  test is edited (the three flaky tests are the measurement).
- `.night-crew/knowledge/roadmap.md` — one word on this card's Activity K line,
  `PLANNED` → `LANDED`. Append-only otherwise.

## What must survive any merge

- The item-list request sequence in `inventory.html`: `ITEMS_SEQ` / `ITEMS_APPLIED`, the
  `fetchItemsSeq()` helper every `GET /items` writer goes through, and the read-only
  `window.InventorySetup` debug object `[IS-02]` reads.
- The add bar living OUTSIDE the re-rendered list body (`#items-add-bar` built once,
  `#items-list-body` re-rendered), and `syncAddBarGroups()` rebuilding only the `<select>`'s
  options.
- The edit-form draft carry-over in `renderItemsList()` (what is typed in an open item editor,
  including the nickname box, is put back after a re-render) — this is the `[IS-04]` fix.
- The dead-photo fallback swapping the failed `<img>` IN PLACE instead of calling
  `renderItemsList()` — this is the mechanism `[IS-04]` found (one full list re-render per dead
  catalog photo, back to back, rebuilding the open editor between the manager's typing and the
  Add tap). A merge that restores the full re-render brings the dropped add back.
- The empty-name alert ("Enter an item name first.").
- The `[IS-01]`…`[IS-04]` describe block.

## What is safe to drop

- Nothing in the code change. Under `.night-crew/runs/2026-10-07-autonomous/logs/inventory-setup-races/`
  the raw Playwright logs are evidence, not behaviour — a merge may keep either side's copy.

## Stubs and fixtures, per done-when clause

| Clause | Rides a stub or fixture? |
|---|---|
| `[IS-01]` typed name survives a late groups response; create POSTs | Network TIMING only — `page.route` delays `GET /groups` 600 ms and continues to the real server. Real page, real server, real database. |
| `[IS-02]` two `GET /items` released out of order leave `ALL_ITEMS` at the newer | Network TIMING only — `page.route` sends each of the two real requests to the real server the moment the page makes it (`route.fetch()`) and hands that same real response back later, newer first (`route.fulfill({response})`). No body is written by the test. Both requests are fired by the page's own writer (`loadItems`, on two Setup opens). |
| `[IS-03]` empty-name create → dialog names the name field | No stub. |
| `[IS-04]` nickname added while the first `GET /items` is held, 5× | Network TIMING only — `page.route` holds the Setup open's first `GET /items` and continues it to the real server. |
| Measurement leg (10× / 10× / 5×), full Playwright suite, `sw.js` 51, G6 | The orchestrator's — not run by this card. |

No stub of the behaviour under test. No migration. No `night-crew.toml` key. No Go change.
`sw.js` / `version.json` are not in the diff (generated from git HEAD by the orchestrator).

## Notes for the merge

- Nothing outside the card's footprint was edited (`inventory.html`, `tests/inventory.spec.js`,
  the roadmap line, this run directory).
- The confined seam run (`logs/inventory-setup-races/confined-seam.log`) is 261 passed / 16
  failed / 1 skipped. The 16 are not this card's: 15 fail identically with `inventory.html` put
  back to HEAD (`base-comparison-16-reds.log`) and are `bugs.md`'s standing inventory cluster
  (the stubbed-pending tests, the sync/cancel tests, "back link navigates to HQ", "the amount is
  centred on the card", "navigating with newItem hash prefills"); the 16th is FR-11 (vendor
  filter + pagination), green alone on both trees — the recorded order-dependent flake
  (BACKLOG B-156).
