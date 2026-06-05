---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 06
subsystem: inventory.html (frontend) + tests/inventory.spec.js + sw.js
tags: [frontend, tab-ui, toast, menu, sw-rebuild, test-migration]
requires:
  - "GET /api/v1/inventory/menu-items?since=YYYY-MM-DD (Plan 05) — already live"
  - "menu_items + daily_menu_sales tables (Plan 02) — already migrated"
  - "Toast worker / sync-toast (Plans 03/04) — populates the data the UI reads"
  - "inventory.html existing helpers: api(), escHtml(), formatDate(), showSkeleton(), showInlineError()"
  - "tab.js (data-tabs= attribute reader)"
provides:
  - "Menu top-level tab in inventory.html between Stock and Setup (D-08)"
  - "loadMenu() / renderMenu() / MENU_DATA — card content per item: name + menu_group + last_seen + units_sold_this_week (D-09)"
  - "Empty-state UI (No menu items) when API returns []"
  - "Renumbered Setup tab: t5→t6, s5→s6 — every hardcoded reference migrated (8 edits in 1 file)"
  - "Two new Playwright regression tests (Menu empty + Menu populated) using page.route stubs"
  - "Regenerated sw.js (Workbox precache) — 21 files, 1334.7 KB, new workbox-cb48cba7 runtime"
affects:
  - inventory.html (8 inline edits)
  - tests/inventory.spec.js (renamed tab-count test, shifted Trends/Cost, migrated 28+ Setup-tab selectors, added 2 new tests)
  - sw.js (regenerated — content-hashed precache)
  - workbox-cb48cba7.js (new Workbox runtime — companion to sw.js)
tech_stack_added: []
patterns_added:
  - "Tab insertion pattern: tabs/sections/data-tabs/show()/render() + every ACTIVE_TAB consumer (goto-setup-item, hashTab deep-link) must migrate atomically"
  - "page.route stub pattern for read-only tabs: register stub BEFORE goto, fulfill with deterministic JSON — keeps tests independent of dev DB state"
key_files_created:
  - .planning/phases/22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat/22-06-SUMMARY.md
  - workbox-cb48cba7.js (Workbox runtime — generated)
key_files_modified:
  - inventory.html
  - tests/inventory.spec.js
  - sw.js
decisions:
  - "Honored step B.6 (Cost test uses #t5/#s5 for the new Cost position) over the conflicting strict acceptance grep `grep -c '#t5\\|#s5\\|show(5)' = 0`. The grep is unachievable now that Cost legitimately lives at index 5 — applying it would have broken the Cost test."
  - "Inserted loadMenu/renderMenu immediately BEFORE loadStock/renderStock to keep load/render pairs grouped logically (plan said 'alongside loadStock/renderStock')."
  - "Used `if(!list)return;` guard in renderMenu — defensive against being called before s3 div is rendered (mirrors existing pattern)."
metrics:
  duration_minutes: 18
  tasks_completed: 3
  files_changed: 4
  completed_date: 2026-06-03
---

# Phase 22 Plan 06: Menu tab in inventory.html Summary

**One-liner:** Plans 02–05 produced the menu_items + daily_menu_sales tables and the cookie-auth read endpoint; Plan 06 surfaces them in a new "Menu" top-level tab between Stock and Setup, with two Playwright regression tests stubbing the endpoint via page.route — and migrates every hardcoded `5` in the Setup-tab routing (show, render, goto-setup-item handler, hashNewItem deep-link) to `6` atomically so existing magic links survive the renumber.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Apply 8 edits to inventory.html (tab bar + sections + show/render + loadMenu/renderMenu + goto-setup-item + hashNewItem) | aa91cc1 | inventory.html |
| 1b | Migrate test selectors t5→t6 / s5→s6 + add 2 Menu-tab regression tests + rename tab-count test | fb6f14e | tests/inventory.spec.js |
| 2 | Rebuild sw.js (Workbox) for the updated inventory.html | fc1d276 | sw.js, workbox-cb48cba7.js |
| 3 | Human verification (Menu tab on phone/Chrome DevTools) | — pending checkpoint — | (returned to orchestrator) |

## What Shipped

### inventory.html — 8 edits, atomic renumber

| Edit | Location (approx line) | What changed |
|------|------------------------|--------------|
| 1 | Tab bar 183–189 | Insert `<button id="t3" onclick="show(3)">Menu</button>`; rename Trends t3→t4, Cost t4→t5, Setup t5→t6 |
| 2 | Section divs 200–233 | Insert new `<div id="s3"><div id="menu-list"></div></div>` between s2 and the old s3; rename Trends s3→s4, Cost s4→s5, Setup s5→s6 |
| 3 | `<script src="tab.js" data-tabs="N">` line 235 | `data-tabs="5"` → `data-tabs="6"` |
| 4 | show() 271–284 | Loop `[1,2,3,4,5]` → `[1,2,3,4,5,6]`; add `if(n===3){loadMenu();}`; Setup loader now triggers on `n===6` (was 5) |
| 5 | render() 286–295 | Add `if(ACTIVE_TAB===3)renderMenu();`; Trends shifts to 4, Cost to 5, Setup catalog branch shifts to 6 |
| 6 | MENU_DATA + loadMenu/renderMenu (new ~298 + ~549) | New `let MENU_DATA=[]` after STOCK_DATA; new `async function loadMenu()` + `function renderMenu()` inserted immediately before `async function loadStock()` |
| 7 | goto-setup-item handler ~886-903 | `ACTIVE_TAB=5`→6; loop array `[1,2,3,4,5]`→`[1,2,3,4,5,6]`; both `i===5` comparisons→`i===6` |
| 8 | hashNewItem bootstrap line 1407 | `hashTab='5'`→`hashTab='6'` |

### Menu tab UI contract (D-08, D-09)

```
GET /api/v1/inventory/menu-items?since=<7-days-ago-YYYY-MM-DD>
  → MENU_DATA = response array (already ordered by last_seen DESC server-side)
  → Each row:
       Bold name (escHtml'd)
       menu_group · last sold <YYYY-MM-DD via formatDate>
       <units_sold_this_week> (right column)
       "this week" subtitle
  → Empty state when MENU_DATA.length === 0:
       "No menu items"
       "Toast ingest has not populated the menu yet. Run `sync-toast` or wait for the next 12h cycle."
```

XSS posture (T-22-19 mitigated): every interpolated string passes through `escHtml()`; numeric `units_sold_this_week` is coerced via `(m.units_sold_this_week||0)`; `formatDate()` returns a known-safe display string.

### tests/inventory.spec.js — selector migration + 2 new tests

| Migration | Before | After | Count |
|-----------|--------|-------|-------|
| Tab-count test | "shows 4 tabs: Purchases, Stock, Trends, Cost" | "shows 6 tabs: Purchases, Stock, Menu, Trends, Cost, Setup" | 1 |
| Trends test | `#t3` / `#s3` | `#t4` / `#s4` | 1 block (lines 334-338) |
| Cost test | `#t4` / `#s4` | `#t5` / `#s5` | 1 block (lines 342-346) |
| Setup-tab tests | `#t5` / `#s5` (~28 occurrences from line 451 onward) | `#t6` / `#s6` | bulk |
| View-in-Setup magic link | `#t5` (line 1969) | `#t6` (line 2020 after insertions) | 1 |
| hashNewItem test | `getElementById('t5')` + "tab 5" comments | `getElementById('t6')` + "tab 6 — was 5 before Phase 22" | 1 block (lines 2607-2627) |
| Comment | `5th tab` | `6th tab` | 2 |

**Two new regression tests added** (inserted immediately after the Cost test, lines 352-399):

1. `Menu tab renders empty state when API returns []` — `page.route` stub fulfills with `[]`, asserts `#s3` visible + `#menu-list` contains "No menu items".
2. `Menu tab renders rows when API returns data` — `page.route` stub fulfills with one fixture row (`Jerk Sliders` / `Sandwiches` / 12 units), asserts all three values appear in `#menu-list`.

Pattern mirrors `tests/users.spec.js:59-78` — route registered BEFORE `page.goto`.

### Targeted Playwright run — all green

```
DB_HOST=100.70.200.55 DB_PORT=5433 \
TOAST_SFTP_KEY_PATH=/Users/jamal/projects/yumyums/sales-processor/creds/id_rsa \
TOAST_SYNC_INTERVAL=0 \
npx playwright test tests/inventory.spec.js \
  -g "Menu tab|shows 6 tabs|Trends tab|Cost tab"          # 5 passed (16.7s)

npx playwright test tests/inventory.spec.js \
  -g "Setup tab|View in Setup|prefills item name|hashNewItem"   # 6 passed (16.0s)
```

The Tailscale Postgres on the Windows box ran migrations 0060+0061 (Plans 02's tables) cleanly. The server booted with `TOAST_SYNC_INTERVAL=0` (Plan 05's escape hatch) which skipped the in-process Toast worker but still passed `LoadConfigFromEnv` key validation — proving the Plan 05 wiring composes correctly with the Plan 06 frontend.

### sw.js rebuild

```
$ node build-sw.js
SW built: 21 files precached (1334.7 KB)
```

- New `workbox-cb48cba7.js` runtime committed alongside sw.js (existing convention — five `workbox-*.js` files already tracked).
- `git diff sw.js` shows the `inventory.html` precache revision hash changed (1 line: 2 matches of the string "inventory.html" — old + new revision entries before/after).
- No manual cache-version bump (Workbox handles content-hashing, per CLAUDE.md).
- ptr.js auto-reload-on-controllerchange (existing) means installed PWAs pick up the new SW on next launch — T-22-21 mitigated.

## Verification

| Check | Result |
|-------|--------|
| `grep -c 'onclick="show(3)">Menu' inventory.html` | 1 |
| `grep -c '<button id="t6" onclick="show(6)">Setup' inventory.html` | 1 |
| `grep -c 'data-tabs="6"' inventory.html` | 1 |
| `grep -c 'data-tabs="5"' inventory.html` | 0 |
| `grep -c '<div id="s3" style="display:none">' inventory.html` | 1 |
| `grep -c '<div id="s6" style="display:none">' inventory.html` | 1 |
| `grep -c 'id="menu-list"' inventory.html` | 1 |
| `grep -c 'async function loadMenu()' inventory.html` | 1 |
| `grep -c 'function renderMenu()' inventory.html` | 1 |
| `grep -c 'let MENU_DATA=' inventory.html` | 1 |
| `grep -c '/api/v1/inventory/menu-items?since=' inventory.html` | 1 |
| `grep -c 'if(n===3){loadMenu()' inventory.html` | 1 |
| `grep -c 'if(ACTIVE_TAB===3)renderMenu()' inventory.html` | 1 |
| `grep -c 'if(n===6){loadItems()' inventory.html` | 1 |
| `grep -c 'if(n===5){loadItems()' inventory.html` | 0 |
| `grep -c 'ACTIVE_TAB=5;' inventory.html` | 0 |
| `grep -c 'ACTIVE_TAB=6;' inventory.html` | 1 |
| `grep -c "hashTab='5'" inventory.html` | 0 |
| `grep -c "hashTab='6'" inventory.html` | 1 |
| `grep -c '\[1,2,3,4,5\]' inventory.html` | 0 |
| `grep -c '\[1,2,3,4,5,6\]' inventory.html` | 2 |
| `grep -c "i===6?'':'none'" inventory.html` | 1 |
| `grep -c "i===5?'':'none'" inventory.html` | 0 |
| HTML parse sanity | OK (`<html>` + `</html>` both present) |
| `grep -c 'inventory.html' sw.js` | 1 |
| `git diff sw.js | grep -c 'inventory.html'` | 2 (revision hash before/after) |
| `grep -c "Menu tab" tests/inventory.spec.js` | 3 (2 test names + 1 section header) |
| `grep -c "menu-list" tests/inventory.spec.js` | 2 |
| `grep -c "api/v1/inventory/menu-items" tests/inventory.spec.js` | 2 |
| `grep -c "shows 6 tabs" tests/inventory.spec.js` | 1 |
| `grep -c "shows 4 tabs" tests/inventory.spec.js` | 0 |
| `grep -c "5th tab" tests/inventory.spec.js` | 0 |
| `grep -c "6th tab" tests/inventory.spec.js` | 2 |
| `grep -c '#t6' tests/inventory.spec.js` | 29 |
| 5 targeted Menu/tab tests | PASSED (16.7s) |
| 6 Setup-tab + magic-link tests | PASSED (16.0s) |

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| **Honor step B.6 (Cost test uses #t5/#s5) over the conflicting strict acceptance grep** | The plan's acceptance criterion `grep -c '#t5\|#s5\|show(5)' tests/inventory.spec.js = 0` is logically incompatible with step B.6 ("Cost: change `#t4`→`#t5` ... in JUST that test block"). Cost legitimately moved from index 4 to index 5; the test for Cost MUST use `#t5`. Applying the strict grep would have left no test exercising the Cost-tab content. The intent of the grep was clearly "no stale Setup-tab references" — that intent is satisfied: every `#t5`/`#s5` remaining (lines 117, 345-347) is Cost. Documented as Rule 1 deviation in commit fb6f14e. |
| **Insert loadMenu/renderMenu immediately before loadStock/renderStock** | The plan said "alongside loadStock/renderStock" — interpreting "alongside" as "immediately before" keeps the load/render pair grouped with its analog (Stock), matches the natural reading order of show()'s if-chain (1→2→3), and minimizes diff noise around the Stock code block. |
| **Defensive `if(!list)return;` guard in renderMenu** | Mirrors a pattern that appears elsewhere in inventory.html — render functions called in odd sequences (e.g. from `render()` while ACTIVE_TAB transitions) should fail soft, not throw, if the section's DOM node isn't yet present. Cheap insurance. |
| **Used Tailscale Windows-box Postgres for test verification** | The local laptop has no Postgres (CLAUDE.md memory: "Remote dev Postgres on Windows box via Docker, accessed over Tailscale/LAN"). Playwright auto-spawns the server pointed at the Tailscale DB. Migrations 0060/0061 (Plans 02) applied cleanly on this DB, proving the full Plan 02→06 stack composes. |
| **TOAST_SYNC_INTERVAL=0 + the sales-processor SSH key path for Playwright** | The server's startup fails fast on missing `TOAST_SFTP_KEY_PATH` (Plan 05's D-12). For test runs that don't need to actually pull from Toast, point at the existing key file at `/Users/jamal/projects/yumyums/sales-processor/creds/id_rsa` (per CONTEXT.md: "User confirmed: SSH key portable") and set `TOAST_SYNC_INTERVAL=0` (Plan 05's escape hatch) to skip the in-process worker. |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Internal Inconsistency] Cost-tab test acceptance grep**

- **Found during:** Task 1b (test migration)
- **Issue:** The plan's acceptance criterion `grep -c '#t5\|#s5\|show(5)' tests/inventory.spec.js = 0` contradicted step B.6 of the plan's own action section ("Cost: change `#t3`→`#t4` ... `#t4`→`#t5` ... in JUST that test block"). After applying step B.6 correctly, `#t5` legitimately remains in the Cost test (line 117 + lines 345-347).
- **Fix:** Honored the action instructions (B.6), which produce correct tests, over the conflicting strict grep. The Cost test now exercises `#t5`/`#s5` because Cost moved to index 5. All other `#t5`/`#s5` references (which all belonged to Setup) were migrated to `#t6`/`#s6` via a Python-bounded bulk replace from line 400 onward — confirmed by grep audit showing only Cost test lines remaining.
- **Files modified:** tests/inventory.spec.js
- **Commit:** fb6f14e
- **Verification:** Targeted Playwright run with both `Cost tab` and `Setup tab` test filters — all 11 tests across the two filters passed.

No other deviations — the 8 inline inventory.html edits applied exactly as specified, sw.js rebuilt cleanly on the first invocation, and the two new Menu-tab tests passed on the first run against a live server.

## Authentication Gates

None. All test verification ran against the Tailscale Windows-box Postgres with `TOAST_SFTP_KEY_PATH` pointed at the existing sales-processor SSH key (no auth gate — file already on disk, per CONTEXT.md confirmation that the key is portable to the dev laptop).

## Deferred Issues

None. The Cost-tab acceptance-grep inconsistency was resolved inline (Rule 1) without leaving any deferred items.

## Known Stubs

None.

The Menu tab is fully wired end-to-end:
- Frontend reads the live `GET /api/v1/inventory/menu-items` endpoint (Plan 05)
- Empty state renders correctly when the API returns `[]` (verified by the "renders empty state" test passing against a stub)
- Populated state renders all four card fields (name, menu_group, last_seen, units_sold_this_week) (verified by the "renders rows" test)
- No placeholder data, no hardcoded empty arrays that flow to UI, no "coming soon" content (Trends/Cost remain coming-soon as before — that's unchanged scope)

The Toast worker's actual data population is Plan 04's responsibility; Plan 06 is the read path. On a fresh DB the Menu tab shows the empty state until Plan 04's worker (or `cmd/sync-toast`) runs.

## Threat Flags

None. The two trust boundaries in the plan's threat model are both mitigated in the shipped code:

| Threat | Mitigation in code |
|--------|-------------------|
| T-22-19 (XSS via innerHTML) | Every dynamic string (`m.name`, `m.menu_group`, `m.menu_subgroup`) wrapped in `escHtml()`; `formatDate(m.last_seen)` returns a known-safe display string; `(m.units_sold_this_week||0)` is a numeric coercion |
| T-22-20 (info disclosure across tenants) | Accepted — single-team app, cookie-auth gate on the endpoint (Plan 05) |
| T-22-21 (stale sw.js precache) | Workbox rebuild in Task 2 changes the inventory.html precache hash; ptr.js auto-reload-on-controllerchange picks up the new SW on next PWA launch |

No new threat surface beyond what the plan's threat register covers.

## Self-Check

- FOUND: inventory.html (modified — 61 insertions, 17 deletions)
- FOUND: tests/inventory.spec.js (modified — 94 insertions, 43 deletions)
- FOUND: sw.js (regenerated)
- FOUND: workbox-cb48cba7.js (new — Workbox runtime)
- FOUND: .planning/phases/22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat/22-06-SUMMARY.md (this file)
- FOUND commit: aa91cc1 (Task 1)
- FOUND commit: fb6f14e (Task 1b)
- FOUND commit: fc1d276 (Task 2)

## Self-Check: PASSED
