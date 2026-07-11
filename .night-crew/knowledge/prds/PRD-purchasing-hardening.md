# PRD — Purchasing app hardening

> **Cycle:** HQ hardening — first night-crew guinea-pig run.
> **App:** Purchasing — `purchasing.html`, `/api/v1/purchasing/*`, Go + Postgres.
> **Depth:** *Enumerate + mark only.* This PRD is the honest flow map with per-flow
> status and a falsifiable definition of "working." It does **not** fix anything —
> each BROKEN/UNPROVEN flow becomes a work order after sign-off.
> **Role of this doc:** the 5th of the cycle's five app hardening PRDs; copies the
> shape of the signed Operations exemplar (`PRD-operations-hardening.md`).
> **Enumeration provenance (for the ≥90% recall KR):** flows were derived by reading
> `purchasing.html` (4 tabs, 1,078 lines), `tests/purchasing.spec.js` (756 lines,
> 26 tests), and the Go package `backend/internal/purchasing/*`
> (`handler.go`, `service.go`, `scheduler.go`, `repurchase.go`, `types.go`) plus the
> chi router in `backend/cmd/server/main.go:552-585` (**21 `/api/v1/purchasing/*`
> routes** enumerated in §Verification — close to the ~20 the card assumed), then
> **cross-checked by a second independent pass (G5)** angled at the named blind spot:
> the scheduler/cutoff cron, the repurchase-reset logic, and the shopping-list state
> transitions — backend timing/state behavior the UI-first pass cannot see. Recall is
> measured post-build as `enumerated ÷ (enumerated + discovered-during-WO-build)`.
> **Scope note — corrects a stale CLAUDE.md label (resolved 2026-07-09):** the root
> CLAUDE.md tables list Purchasing as **"Mockup."** *That premise is stale and is
> corrected here.* Purchasing is a **real app**: a real, tested Go backend
> (`backend/internal/purchasing/*` — 21 routed endpoints across orders, cutoff,
> suggestions, shopping lists, repurchase-reset, plus a 15-minute background
> scheduler), a 1,078-line `purchasing.html` with a live 4-tab UI wired to those
> endpoints, and a 31 KB `tests/purchasing.spec.js`. This PRD enumerates Purchasing
> **as a real app**, on exactly the same footing as the other four. The old
> "Mockup" note predates the backend build and should be dropped from CLAUDE.md when
> a doc-update WO next touches it (not in scope here).
> **"Critical flows" interpretation (resolves G2):** the OKR's phrase "critical
> flows" is read as **all** real user-facing/operator-triggered flows — no
> sub-selection — plus the backend-only guarantees a real person depends on
> (auto-lock, alerts, snapshot immutability). Dropping the "critical" filter removes
> an undefined term that made the recall metric gameable.

## Objective

Produce a written, agreed enumeration of every end-to-end flow in the Purchasing
app — from a crew member stepping quantities on this week's draft order, through the
weekly cutoff auto-locking the PO, an admin approving it into an immutable shopping
list, the crew checking items off vendor-by-vendor, to the completed run landing in
history and the repurchase badges resetting — with each flow honestly marked
**WORKING**, **UNPROVEN**, or **BROKEN**, and each backed by a falsifiable
definition of what "working" means. It advances the **Product objective**: KR-1
(5/5 apps have a hardening PRD enumerating their E2E flows — this is app 5/5) and
KR-2 (enumeration recall ≥ 90%). Its status tally is the denominator the
**Engineering** (0 known-broken flows at cycle end) and **QA** (every flow has a
real, asserting test) objectives grade against.

## Operators & users

- **Crew member (team_member)** — opens the Order tab, adds catalog items via the
  fullscreen picker, steps quantities up/down, accepts restock suggestions. During
  a shopping run, opens the Shopping tab and checks items off per vendor section,
  edits in-store aisle locations, snaps item photos, and completes each vendor.
  Primary user of Tabs 1 and 2.
- **Admin / owner** — configures the weekly cutoff schedule (day/time/timezone),
  edits a locked PO, and **approves** a locked PO into a shopping list from the PO
  tab. Admin-only endpoints also cover simulate-cutoff, lock/unlock, and the
  repurchase-reset schedule — several of which have **no UI surface** (API/backend
  only; see below).
- **The scheduler (indirect actor)** — a 15-minute background goroutine
  (`StartScheduler`) that auto-locks the draft PO at cutoff, sends a 24h cutoff
  reminder, batches low-stock alerts, and auto-resets repurchase badges. No human
  triggers it; it acts on the admin's stored config.
- **The overnight crew (indirect)** — consumes this enumeration + status as the
  work-order backlog; a flow marked BROKEN/UNPROVEN here is a candidate WO.

## Requirements

Status legend: **WORKING** = flow works E2E *and* a test drives it and asserts
observable DB/UI state · **UNPROVEN** = flow appears to work in code but no test
verifies the behavior it names (missing or vacuous test) · **BROKEN** = flow is
**confirmed** incomplete/stubbed by code inspection, or a test reveals it does not
do what it claims. Every requirement traces to an OKR key result.

**G1 resolution (high-risk card):** BROKEN is reserved for *confirmed* breakage
only. The "Mockup" reputation is **not** grounds for a BROKEN mark — a flow that
looks like a stub stays **UNPROVEN with a confirm-absence step** unless code
inspection actually proves it missing/stubbed. Exactly **one** flow is marked
BROKEN (FR-18, the History tab), and it is cited to the exact stub line.

### Tab 1 — Order (weekly draft PO)

- **FR-1** — On load, the system gets-or-creates the current week's **draft** PO;
  if this week's PO is already locked/approved it rolls to next week's draft, so the
  Order tab always shows an editable draft, never a locked PO. *(POST
  `/orders`; `service.go:73-132 GetOrCreateOrder`)* — **UNPROVEN** (init renders and
  the suggestions regression test asserts the page loads, but no test drives the
  roll-to-next-week branch or asserts a draft is returned) — traces to Engineering
  KR-1, QA KR-2.
- **FR-2** — A crew member steps a line item's quantity up/down; qty 0 dims/removes
  it. The change is debounced and saved by **replacing** the PO's line items
  (delete-not-in-set + upsert qty>0). *(PUT `/orders/{id}/items?require_draft=true`;
  `service.go:213-274 UpsertLineItems`, `purchasing.html:784-836`)* — **UNPROVEN**
  (stepper mutates DOM + debounced-saves in code; no test enters a qty and asserts
  the persisted line-item set after reload) — traces to Product KR-1, QA KR-1.
- **FR-3** — Adding an item requires the catalog item to have a `store_location`;
  a location-less item shows "Set location in Setup" in the picker and
  `addItemToPO` blocks it with a toast. *(`purchasing.html:839-865 addItemToPO`,
  picker `renderPickerList` 921-1003)* — **WORKING** (5 item-picker tests drive the
  guard: "shows Set location…", "shows Add button", grouped-by-location headers,
  30+ render cap, `addItemToPO` toast) — traces to Engineering KR-1, QA KR-2.
- **FR-4** — The system lists restock **suggestions** (catalog items below their
  group's high threshold, stock>0, not already on the PO) in a collapsible card;
  "Add Selected" bulk-adds them. *(GET `/orders/{id}/suggestions`;
  `service.go:659-724 GetSuggestions`, `purchasing.html:456-500, 877-904`)* —
  **UNPROVEN** (the suggestions regression test asserts the endpoint loads without a
  JS error and the Order renders, but does **not** assert the correct suggestion set
  or that "Add Selected" writes the items — note the memory-logged open bug
  "suggestions count 4 vs 3") — traces to Product KR-1, QA KR-1.
- **FR-5** — Tapping an item's info area deep-links to Inventory Setup with that
  item expanded (`inventory.html#tab=5&item=…`). *(`purchasing.html:1046`)* —
  **WORKING** ("clicking item info on Order tab navigates to Inventory Setup with
  item expanded" drives the nav + asserts the expanded edit form) — traces to
  Product KR-1, QA KR-2.
- **FR-6** — An admin sets the weekly **cutoff** (day-of-week + HH:MM + timezone)
  from the Order tab's cutoff pill; non-admins cannot save (403). *(GET/PUT
  `/cutoff`; `handler.go:311-369`, `purchasing.html:242-290 renderCutoffForm/
  saveCutoff`)* — **UNPROVEN** (form renders + PUT wired; no test saves a config and
  asserts it round-trips, nor asserts the non-admin 403) — traces to Engineering
  KR-1, QA KR-1.

### Tab 2 — Shopping (active run)

- **FR-7** — When a shopping list is active, the tab renders it grouped into vendor
  sections with per-item check buttons, thumbnails, and aisle locations; when none
  is active it shows the "…after the PO is approved" stub. *(GET `/shopping/active`;
  `service.go:277-297`, `purchasing.html:542-596`)* — **WORKING** (empty-state test +
  `seedShoppingList` drives the populated render) — traces to Product KR-1, QA KR-2.
- **FR-8** — Checking/unchecking an item persists `checked`/`checked_by`/
  `checked_at` and survives reload. *(POST `/shopping/{id}/check`;
  `service.go:461-476 CheckShoppingItem`, `purchasing.html:610-637`)* — **WORKING**
  ("shopping item check-off survives page reload" seeds, clicks, reloads, asserts
  ✅) — traces to Product KR-1, QA KR-2.
- **FR-9** — Editing an item's **in-store aisle** location persists to
  `shopping_list_items.store_location` **only** — it must not overwrite the catalog
  `purchase_items.store_location`. *(PUT `/shopping/{id}/items/{itemId}/location`;
  `service.go:481-489`, `purchasing.html:639-668`)* — **WORKING** (two tests: "store
  location edit persists after reload" + "aisle location does not overwrite catalog
  store_location") — traces to Engineering KR-1, QA KR-2.
- **FR-10** — Snapping/uploading an item photo persists `photo_url` to **both**
  `shopping_list_items` and the catalog `purchase_items`, clears the "No photo"
  badge, and shows the thumbnail on reload. *(PUT `/shopping/{id}/items/{itemId}/
  photo`; `service.go:492-506`, `purchasing.html:735-774`)* — **WORKING** ("No photo
  badge shows… and disappears after photo upload" asserts badge→thumbnail
  transition) — traces to Engineering KR-1, QA KR-2.
- **FR-11** — Checking an item with no photo/location surfaces a toast prompting
  "Add Now." *(`purchasing.html:599-609 showShoppingToast`)* — **WORKING** ("toast
  appears when checking item without photo" asserts toast text + Add-Now button) —
  traces to Product KR-1, QA KR-2.
- **FR-12** — Completing a vendor section marks it `completed` (with completed_by),
  and when the **last** pending section completes it cascades the shopping list →
  `completed` and its PO → `completed`. *(POST `/shopping/{id}/vendors/
  {vendorSectionId}/complete`; `service.go:510-581 CompleteVendorSection`,
  `purchasing.html:670-696`)* — **UNPROVEN** ("vendor section completion persists
  after reload" clicks Complete and reloads, but its final assertion is only
  `expect(content).toBeTruthy()` — it does **not** assert the section shows completed
  nor the list/PO cascade; a vacuous tail — see §Vacuous tests) — traces to
  Engineering KR-1, QA KR-1.

### Tab 3 — PO (locked order → approval)

- **FR-13** — When a locked PO exists, the tab renders it read-only with a status
  badge (locked/approved/shopping). *(GET `/orders?status=locked`;
  `handler.go:431-456 GetOrdersByStatus`, `purchasing.html:292-352 renderPOTab`)* —
  **UNPROVEN** *(inline mark corrected WORKING → UNPROVEN by the Activity-2 sweep
  2026-07-11, G6-confirmed, to match the authoritative tally: `renderPOTab` is present
  and real, but the "PO tab shows stub or locked PO" test only asserts content renders
  — it does not seed a locked PO and assert the read-only render + status badge, so it
  fails the WORKING bar. Present-but-untested.)* — traces to Product KR-1, QA KR-2.
- **FR-14** — An admin can edit a **locked** PO (add items via the PO-target
  picker, step quantities) — non-admins cannot; the edit path sends `allowLocked`
  (no `require_draft`). *(PUT `/orders/{id}/items`; `handler.go:82-99`,
  `service.go:232-234`, `purchasing.html:343-374, 804-836`)* — **UNPROVEN** (admin
  edit path + the `po_locked_admin_only` 403 for non-admins are both untested) —
  traces to Engineering KR-1, QA KR-1.
- **FR-15** — An admin approves a locked PO; the button is shown only to admins and
  only while status is `locked`. *(`purchasing.html:314, 347-350, 470-490`)* —
  **UNPROVEN** ("PO tab shows approve button for admin when PO is locked" *renders*
  the button but its assertion is guarded — `if (locked.status === 'locked')` — so
  when no locked PO is seeded it asserts nothing; button visibility for a
  guaranteed-locked PO is not driven) — traces to Product KR-1, QA KR-1.
- **FR-16** — Approving transitions the PO `locked → shopping_active`, and in one
  transaction creates an **immutable shopping-list snapshot**: one vendor section
  per distinct vendor, items snapshotted from PO line items (name/photo/location/
  qty/unit frozen). Refused if an active list already exists (409
  `active_shopping_list_exists`) or the PO isn't locked (409 `po_not_locked`).
  *(POST `/orders/{id}/approve`; `service.go:851-969 ApprovePO`, `handler.go:534-564`)*
  — **UNPROVEN** (`seedShoppingList` *exercises* approve as a test **helper**, so the
  happy path runs, but no test **asserts** the snapshot contents, the vendor-section
  grouping, or the two 409 refusals as named behaviors) — traces to Engineering
  KR-1, QA KR-1.

### Tab 4 — History

- **FR-17** — The backend returns all completed shopping lists with per-vendor
  sections and missing counts. *(GET `/shopping/history`;
  `service.go:396-458 GetShoppingListHistory`, `handler.go:153-172`)* —
  **UNPROVEN** (endpoint implemented and returns rows; but nothing that *asserts* its
  output — the History tests all `test.skip` when history is empty and otherwise
  probe DOM that doesn't exist — see FR-18) — traces to Product KR-1, QA KR-1.
- **FR-18** — The History **tab UI** shows past shopping runs (vendor breakdown,
  missing counts, tap-to-expand item detail). — **BROKEN (confirmed)** — the History
  tab is a **hardcoded static stub**: `purchasing.html:156` is literally
  `<div id="s4" …><div class="stub">Past shopping runs will appear here</div></div>`,
  and `show(n)` (`purchasing.html:776-782`) only toggles `display` — **there is no
  `renderHistory()`, no `#history-content` element, and no call to
  `GET /shopping/history` anywhere in the frontend** (`grep` for
  `history-content|history-card|shopping/history|renderHistory` in `purchasing.html`
  returns nothing). The four History tests assert against `#history-content`,
  `.history-card`, `.history-hd`, `.history-detail`, `.history-mt` — DOM that is
  never produced — so they either time out or `test.skip`; they do not prove the
  flow. The backend (FR-17) works, but the UI flow it feeds is absent. — traces to
  Product KR-1, Engineering KR-1, QA KR-1/KR-2.

### Cross-cutting (non-functional / platform guarantees)

- **NFR-1 (PO state machine)** — `draft → locked → shopping_active → completed`,
  with `unlock` returning `locked → draft` (blocked after approval, D-13). Each
  transition is optimistic-locked (`WHERE status = …`, `version = version + 1`) and
  bumps a next-week draft on lock. *(`service.go:772-845 LockPO/UnlockPO`,
  `handler.go:460-530`)* — **UNPROVEN** (transitions run inside seed/approve helpers;
  no test asserts a status change or an illegal-transition 409 as its named
  behavior) — traces to Engineering KR-1, QA KR-1.
- **NFR-2 (Admin authorization tier)** — cutoff PUT, simulate-cutoff, lock, unlock,
  approve, and all repurchase-reset endpoints are admin-gated (`isAdmin` →
  superadmin or `admin` role); non-admins get 403. *(`service.go:43-56`,
  handlers throughout)* — **UNPROVEN** (the guard exists on every admin handler; no
  test drives a non-admin session against any of them to assert 403 — the suite
  logs in as an admin only) — traces to Engineering KR-1, QA KR-1.
  *(Endpoint-accounting note: `POST /simulate-cutoff` — the admin-only manual
  lock-now that immediately transitions the draft PO to `locked` (or returns 409
  `locked_po_pending_approval` / `po_not_draft`), `handler.go:373-429` — has no
  dedicated FR line; its state-transition behavior is the manual counterpart of the
  auto-lock cron FR-19, and its admin gate is covered here under NFR-2. So the
  "every endpoint → a flow" audit finds all 21 routes accounted for.)*

## Additional flows — G5 second-pass cross-check

A second, independent enumeration angled at the named blind spot (the scheduler/
cutoff cron, repurchase-reset logic, and shopping-list state transitions) found
**6 flows the first UI-first pass missed** — the empirical basis for the recall
note in §Success metrics. All 6 are backend-only (no UI surface) and untested:

- **FR-19 (auto-lock cron)** — Every 15 min the scheduler loads the cutoff config,
  computes the most-recent cutoff occurrence in the config timezone (DST-safe via
  `time.LoadLocation`), and — if it has passed and no locked PO is pending approval —
  auto-locks the current draft PO. *(`scheduler.go:167-243 runCutoffCheck`, launched
  `main.go:621`)* — **UNPROVEN** — traces to Engineering KR-1, QA KR-1.
- **FR-20 (cutoff reminder cron)** — Within the 24h-to-23h window before cutoff the
  scheduler sends a single, idempotent (per `alert_log` on `week_start`) reminder to
  crew-alert contacts, including the current draft item count. *(`scheduler.go:54-163
  runReminderCheck`)* — **UNPROVEN** — traces to Engineering KR-1, QA KR-1.
- **FR-21 (low-stock alert cron)** — Each tick batches items below their group low
  threshold and alerts admins once per week per item (idempotent via
  `low_stock_alert_log`). *(`scheduler.go:247-360 runLowStockCheck`)* — **UNPROVEN**
  — traces to Engineering KR-1, QA KR-1.
- **FR-22 (repurchase-reset config + auto-reset)** — An admin sets a weekly badge
  reset schedule (single-row upsert); each scheduler tick advances `last_reset_at`
  once the configured weekday/time has passed, hiding badges for items repurchased
  before it. *(GET/PUT `/repurchase-reset[/config]`; `handler.go:588-655`,
  `repurchase.go:81-177 GetRepurchaseResetConfig/Upsert…/runRepurchaseResetCheck`)*
  — **UNPROVEN** (no UI surface, no test) — traces to Engineering KR-1, QA KR-1.
- **FR-23 (manual repurchase reset)** — An admin can immediately reset badges via
  `POST /repurchase-reset`, updating `last_reset_at = now()` (inserting a default
  config row if none exists). *(`handler.go:566-586`, `repurchase.go:60-77
  TriggerRepurchaseReset`)* — **UNPROVEN** — traces to Engineering KR-1, QA KR-1.
- **FR-24 (repurchase log on vendor complete)** — Completing a vendor section
  records its checked items to `repurchase_log` (best-effort, post-commit) so the
  repurchase badges have data. *(`service.go:568-572`, `repurchase.go:16-55
  RecordRepurchase`)* — **UNPROVEN** — traces to Engineering KR-1, QA KR-1.

*(A 7th backend behavior — `NotifyVendorComplete` shopping-completion alert with the
missing-items list, `service.go:583-654` — is the alert half of FR-12/FR-21 and is
covered by the FR-12 completion flow + NFR admin-alert path; not counted separately
to keep the tally clean.)*

## Acceptance criteria

Surface-anchored, Given/When/Then. These define "working" for representative flows;
every enumerated flow inherits the pattern of *drive-the-real-flow +
assert-observable-state* (the WORKING bar).

- **AC-1 (FR-2, order stepper persists):** *Given* the Order tab with a draft PO,
  *When* a crew member steps an item to qty 3, waits for the debounced save, and
  reloads, *Then* the persisted PO line-item set contains that item at qty 3 and
  qty-0 items are absent.
- **AC-2 (FR-3, location guard — WORKING):** *Given* a catalog item with no
  `store_location`, *When* `addItemToPO` is invoked for it, *Then* a toast "Set a
  store location in Inventory Setup…" appears and the item is not added.
- **AC-3 (FR-8, check-off persists — WORKING):** *Given* an active shopping list,
  *When* an item is checked and the page reloaded, *Then* the item still renders ✅
  and `checked_by`/`checked_at` are set in the DB.
- **AC-4 (FR-12, vendor-complete cascade — currently UNPROVEN):** *Given* a list
  whose last pending vendor section has all items checked, *When* that section is
  completed, *Then* the section shows `completed`, the shopping list is `completed`,
  and its PO is `completed` in the DB. *(A red-first test must show this failing if
  the cascade breaks — replacing the current vacuous `toBeTruthy()` tail.)*
- **AC-5 (FR-16, approve snapshot — currently UNPROVEN):** *Given* a locked PO with
  items across two vendors, *When* an admin approves it, *Then* the PO is
  `shopping_active`, exactly one shopping list exists with one vendor section per
  vendor and items snapshotted from the PO line items; *When* approve is called
  again, *Then* it returns 409 `active_shopping_list_exists`.
- **AC-6 (FR-18, History tab — BROKEN):** *Given* at least one completed shopping
  list, *When* the crew opens the History tab, *Then* a history card renders with
  the week label, vendor breakdown, and missing count, and tapping it expands item
  detail. *(This currently cannot pass — `#s4` is a static stub and never calls
  `GET /shopping/history`; the WO is a frontend build + regression test, not a
  test-only fix.)*
- **AC-7 (FR-19, auto-lock cron — currently UNPROVEN):** *Given* a cutoff config
  whose time is in the past and a draft PO with no locked PO pending, *When* the
  scheduler tick runs, *Then* the draft PO transitions to `locked` and next week's
  draft is created.
- **AC-8 (NFR-2, admin gate):** *Given* a non-admin session, *When* it calls
  `PUT /cutoff` (or simulate-cutoff / lock / unlock / approve / repurchase-reset),
  *Then* the request is refused (403) and no state changes.

## Verification plan

- **Environment:** localhost Postgres (`brew postgresql@16`) — the E2E suite
  requires a local DB; the remote Windows DB is too slow for the suite (N+1 ×
  50ms RTT). Playwright blocks service workers (`serviceWorkers: 'block'`). The
  scheduler (FR-19..FR-24) is time-driven; its WO tests must invoke the check
  functions directly (Go unit test seeding config + a past cutoff) rather than
  waiting 15 minutes.
- **Suites:** `tests/purchasing.spec.js` (26 tests). Run per-flow during iteration
  (`npx playwright test tests/purchasing.spec.js -g "<name>"`), full suite
  (`task test`) at gate. Cron flows want Go tests under
  `backend/internal/purchasing/`.
- **This PRD specifies the test each flow needs; it does not write them (resolves
  G4).** Writing/repairing a test is itself a work order.
- **What each status turns into downstream:**
  - **WORKING** flows: a **test-audit WO** — spot-check the existing test is
    non-vacuous. If vacuous, it drops to UNPROVEN.
  - **UNPROVEN** flows: a **test-only WO first (resolves G3)** — write a real
    seeded, red-first assertion per the bug-fix protocol. The flow graduates to a
    **fix WO only if that test goes red** — we do not pre-judge an untested flow as
    needing code changes. **This is the operative rule for every "mockup"-suspected
    backend flow here** (FR-1, FR-4, FR-6, FR-14, FR-15, FR-16, NFR-1, NFR-2,
    FR-19..FR-24): each is present in real code and stays UNPROVEN, not BROKEN.
  - **BROKEN** flow (FR-18): a **frontend build WO** (implement `renderHistory` +
    wire `GET /shopping/history` into the History tab) **plus** a real regression
    test — the existing History tests are rewritten to drive the new UI.
- **Endpoints in scope (21 routes, `main.go:552-585`):** GET `/cutoff`, PUT
  `/cutoff`, POST `/simulate-cutoff`, GET `/orders?status=`, POST `/orders`, GET
  `/orders/{id}`, PUT `/orders/{id}/items`, GET `/orders/{id}/suggestions`, POST
  `/orders/{id}/lock`, POST `/orders/{id}/unlock`, POST `/orders/{id}/approve`, GET
  `/shopping/active`, GET `/shopping/history`, GET `/shopping/{id}`, POST
  `/shopping/{id}/check`, PUT `/shopping/{id}/items/{itemId}/location`, PUT
  `/shopping/{id}/items/{itemId}/photo`, POST
  `/shopping/{id}/vendors/{vendorSectionId}/complete`, GET `/repurchase-reset`, POST
  `/repurchase-reset`, PUT `/repurchase-reset/config`. *(`GET /orders/{id}` and
  `GET /shopping/{id}` are fetch-by-id helpers behind the tabs; the `simulate-cutoff`
  and `unlock`/`lock` routes are admin/API-only with no UI trigger — see below.)*
- **API-only endpoints with no UI surface (flagged, not a defect):** `POST
  /simulate-cutoff`, `POST /orders/{id}/lock`, `POST /orders/{id}/unlock`, and all
  three `/repurchase-reset*` routes have **no button in `purchasing.html`** (0
  references). They are reachable only via direct API (or, for lock, the scheduler).
  Their WOs must decide whether each needs a UI affordance or is intentionally
  admin-API-only; the enumeration marks them UNPROVEN, not BROKEN, because the
  handlers are real and correct.

### Status tally (the denominator downstream objectives grade against)

Total requirements enumerated: **26** (24 FR + 2 NFR) — 18 first-pass (FR-1..FR-18)
+ 2 first-pass NFR (NFR-1, NFR-2) = 20 first-pass, + 6 from the G5 cross-check
(FR-19..FR-24). **Every ID below is counted exactly once; the three counts sum to
26.**

| Status | Count | Flows |
|---|---|---|
| **WORKING** | 7 | FR-3, FR-5, FR-7, FR-8, FR-9, FR-10, FR-11 |
| **UNPROVEN** | 18 | FR-1, FR-2, FR-4, FR-6, FR-12, FR-13, FR-14, FR-15, FR-16, FR-17, NFR-1, NFR-2, FR-19, FR-20, FR-21, FR-22, FR-23, FR-24 |
| **BROKEN** | 1 | FR-18 *(History tab — confirmed static stub, `purchasing.html:156`; no `renderHistory`/`GET /shopping/history` in the frontend)* |

*Sum check: 7 + 18 + 1 = **26** = total.* (18 UNPROVEN + 1 BROKEN = the 19-flow
candidate work-order backlog. Every one must have a shipped WO by cycle end —
Delivery KR-1 — and reach 0 known-broken — Engineering KR-1. FR-18 is the only
confirmed code-fix WO; the rest open as test-only WOs and graduate to a fix WO only
if their red-first test fails. Recall denominator = 26.)

### Activity-2 confirm-absence sweep record (2026-07-11, G6-passed)

Two-pass static audit (pass 1 UI-flow, pass 2 scheduler/cron/state-machine cross-check)
of all 18 UNPROVEN flows against `purchasing.html` + `backend/internal/purchasing/*`;
adversarial G6 stub-hunt of every cron + `ApprovePO` + the 5 D-1 handlers. **Result: 0
graduations — all 18 stay UNPROVEN; FR-18 remains the only BROKEN.** Tally unchanged:
WORKING 7 · UNPROVEN 18 · BROKEN 1. **Pass 2 was load-bearing for NOT mis-marking** the
backend-only cron/approve surface BROKEN: all four scheduler checks and `ApprovePO` are
real implementations (no no-op/TODO/empty body), just untested.

**D-1 honored:** the 5 no-UI admin/cron endpoints have real handler bodies
(`SimulateCutoffHandler` `handler.go:373-429`, `LockPOHandler` `:460-491`,
`UnlockPOHandler` `:495-530`, `RepurchaseResetHandler` `:568-586`,
Get/Upsert-RepurchaseResetConfig `:590-655`) — absent UI is by-design, so they stay
UNPROVEN, not BROKEN. **FR-13 inline mark corrected WORKING → UNPROVEN** (doc-consistency;
tally was already correct). FR-18 stub re-confirmed at `purchasing.html:156` (grep for
`renderHistory|history-content|shopping/history` → 0 frontend hits).

| Flow | Present at | Confirm-note |
|---|---|---|
| FR-1 | `service.go:73-132` | `GetOrCreateOrder` roll-to-next-week branch real; untested |
| FR-2 | `service.go:213-274`; `purchasing.html:784-836` | stepper debounced-save (delete-not-in-set + upsert) present |
| FR-4 | `service.go:659-724`; `purchasing.html:477` | suggestions query + Add-Selected present |
| FR-6 | `handler.go:335-369`; `purchasing.html:242,272` | admin-gated cutoff PUT (403) + form present |
| FR-12 | `service.go:510-581`; `purchasing.html:670-696` | vendor-complete cascade→list→PO present (test tail vacuous) |
| FR-13 | `purchasing.html:292-352` | `renderPOTab` present; inline mark reconciled to UNPROVEN |
| FR-14 | `handler.go:82-99`; `service.go:232-234`; `purchasing.html:314-345` | admin locked-PO edit path + 403 present |
| FR-15 | `purchasing.html:314,347-350` | approve button gated admin+locked present |
| FR-16 | `service.go:851-969`; `handler.go:534-564` | `ApprovePO` snapshot + both 409s present |
| FR-17 | `service.go:396-458`; `handler.go:153-172` | `GetShoppingListHistory` backend real (UI = FR-18 BROKEN) |
| NFR-1 | `service.go:772-845` | `LockPO`/`UnlockPO` optimistic-lock state machine present |
| NFR-2 | `service.go:43-56`; guards at `handler.go:342,380,467,502,541,575,597,624` | `isAdmin` on every admin handler present |
| FR-19 | `scheduler.go:167-243` | `runCutoffCheck` (DST-safe auto-lock) real |
| FR-20 | `scheduler.go:54-163` | `runReminderCheck` (idempotent `alert_log`) real |
| FR-21 | `scheduler.go:247-360` | `runLowStockCheck` (idempotent `low_stock_alert_log`) real |
| FR-22 | `repurchase.go:81-125,129-177` | reset config upsert + auto-reset real (D-1 no-UI) |
| FR-23 | `handler.go:566-586`; `repurchase.go:60-77` | `TriggerRepurchaseReset` manual reset real (D-1 no-UI) |
| FR-24 | `service.go:568-572`; `repurchase.go:16-55` | `RecordRepurchase` on vendor-complete real |

## Out of scope

- The other four apps (Operations, Inventory, Onboarding, Users) — their own PRDs.
- **Fixing** any flow — this PRD enumerates and marks; work orders fix (including
  building the History tab UI for FR-18).
- Any net-new feature (hardening only, per the brief). Deciding whether the
  API-only admin endpoints (simulate-cutoff, lock, unlock, repurchase-reset) *should*
  get a UI is a WO/product call, not this doc's.
- Editing CLAUDE.md's stale "Mockup" label — flagged in the Scope note for a
  doc-update WO; not touched here.
- The inventory item catalog, groups/thresholds, and stock-count machinery that
  Purchasing reads from (different app) — only the purchasing-side reads/writes are
  in scope.

## Success metrics

- **Enumeration recall ≥ 90%** — `enumerated ÷ (enumerated + discovered-during-WO-
  build) ≥ 0.90`. Denominator: the **26** requirements above plus any flow the build
  surfaces. *(Product KR-2.)*
  - **Empirical finding (guinea-pig signal):** the first (UI-first + endpoint) pass
    enumerated 20; the G5 cross-check angled at the scheduler/repurchase blind spot
    found 6 more → **single-pass recall ≈ 20/26 = 77%, well under the 90% bar.** The
    two-pass total (26) clears it. This is the *lowest* single-pass recall of the
    cycle so far (Operations was 85%) — the backend-heavy scheduler/cron surface is
    exactly what a UI-first read misses. Reinforces the exemplar's lesson: **one
    pass is not enough; the blind-spot cross-check is mandatory.**
- **5/5 apps gate** — this is app **5 of 5**; all five PRDs now share one shape.
  *(Product KR-1.)*
- **0 known-broken flows** across Purchasing at cycle end — the 1 BROKEN (FR-18) and
  the 18 UNPROVEN (incl. any that graduate to BROKEN via a red-first test) either
  reach WORKING or are explicitly waived. *(Engineering KR-1.)*
- **100% of UNPROVEN/BROKEN flows have a shipped WO** by cycle end. *(Delivery KR-1.)*
- **Every WORKING flow's test is non-vacuous** and every repaired flow carries a
  red-first proof — starting with the FR-12 vendor-complete test whose current
  `toBeTruthy()` tail is replaced by a real cascade assertion. *(QA KR-1, KR-3.)*
