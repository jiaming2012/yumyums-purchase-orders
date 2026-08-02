# Phase 999.2 — sales-processor ↔ HQ HTTP Contract (menu-cogs)

**Status:** Authored 2026-06-04 by the planner. **Hand this document to the sales-processor maintainer.**

This document is the contract the sales-processor repo must implement against to satisfy Phase 999.2's acceptance criteria. The HQ-side of the phase (this repo) is shipped and exercised by integration tests; the sales-processor side is NOT planned here per the developer's decision to keep that work in its own repo.

If sales-processor differs from any assumption below, raise a question against this doc — do NOT silently diverge.

This contract is the sibling of `21-SALES-PROCESSOR-CONTRACT.md`. The two endpoints share the same Bearer-auth pattern, the same 503-if-unset behavior, and the same env var. *(Corrected 2026-08-03: they do **not** share date semantics — see §0 — and since 2026-06-06 `/menu-cogs` no longer decomposes the Phase 21 aggregate. The DIFFERENCE is narrower than this page used to claim: `/period-summary` returns aggregate COGS over allowlisted confirmed events plus pending rows; `/menu-cogs` attributes confirmed, catalogued line-item spend to menu items.)*

---

## 0. Drift Audit — 2026-08-03

**Why this section exists.** On 2026-08-03 every line of this document that publishes a code-level claim was diffed against the code at HEAD, name by name and expression by expression, rather than read for plausibility.

**This section was itself audited, and it was itself incomplete.** The first pass (same day) skipped the three example response states, all of §3 past the client struct, and all of §4 — while claiming the whole document had been diffed, and while its own row counts did not reconcile against its own table. The second pass closed those gaps, added seventeen rows, reclassified two verdicts and corrected two attributions. Where the second pass corrected the first, the row says so explicitly rather than quietly restating.

**The finding that matters most:** unlike its sibling, **this document did not drift — most of it was wrong on the day it was written.** It was authored 2026-06-04 (`b0119c0`) from the phase plan, while the handler and structs had landed the same day from the same plan (`3d9362c`, `b283f5f`) — and the two disagree. **Four of the nine per-item JSON field names have never matched the code.** A sales-processor client built literally against the previous version of this page decodes `name` and `menu` as empty strings on every row and has done since June 2026. 🛑 **And the first pass of this audit corrected that in the field table and the copy-paste struct, and left it standing in the State A example** — see `:123`–`:140`.

Line numbers below (`:NN`) are as this document stood **before** this revision. Corrections are applied inline.

**Method.** Published name/expression → the code at HEAD, named by file and symbol. The response shape was **observed** by marshalling `recipes.MenuCOGSResponse`, `recipes.MenuCOGSRow`, `recipes.IngredientAlloc`, `recipes.UnallocatedBreakdown` and `recipes.UnallocatedDetail` and printing the JSON, not transcribed by eye — which is the only reason the `omitempty` findings and the number formatting surfaced at all. A row that **drifted** names the commit that drifted it; a row that was **never true** says so instead of naming one, because there is none to name.

| `:NN` | Row / claim | Verdict |
|---|---|---|
| `:9` | "the two endpoints share … the same date semantics" | **FALSE.** `/menu-cogs` has no timezone cast anywhere; `/period-summary` has one. Corrected inline. |
| `:18` | Path `GET /api/v1/inventory/menu-cogs` | **CONFIRMED** — `main.go` route registration |
| `:24` | `GET`, idempotent, no body | **CONFIRMED** |
| `:27`–`:28` | `from` / `to` required, `YYYY-MM-DD`, inclusive | **CONFIRMED** — `time.Parse("2006-01-02")` on both |
| `:29` | `breakdown` optional, `true` enables detail | **CONFIRMED byte-accurate** — `r.URL.Query().Get("breakdown") == "true"`; any other value is false |
| `:30` | "Both dates interpreted in `America/New_York`" | **FALSE.** No `AT TIME ZONE` in any of the four queries. `pe.event_date` and `dms.business_date` are plain `DATE`. Corrected inline. |
| `:31` | "shares date semantics with `/period-summary`, so it moves with it" on the changeover deploy | **FALSE.** Nothing here moves on that deploy. Added by card A1 in `1be1a59`, **2026-07-31** (merged the same day in `67b2c9f`), and not checked against the code. Corrected inline. *(Attribution corrected 2026-08-03, second pass: the first pass cited "`67b2c9f`, 2026-08-01". `67b2c9f` is the merge commit, and both its author and commit dates are 2026-07-31; the text itself landed in `1be1a59`, also 07-31. The "2026-08-01" is the date the prose writes about itself, not a date in git.)* |
| `:32` | Same `HQ_INVENTORY_SERVICE_TOKEN` as Phase 21 | **CONFIRMED** — one middleware group, one env var, both routes |
| `:47` | `Cache-Control: private, max-age=3600` | **CONFIRMED byte-accurate** |
| `:51`–`:68` | Summary-mode JSON example | **WRONG SINCE AUTHORING.** Four field names never matched; one documented field does not exist; one real field was never documented. Replaced with observed output. |
| `:105` | `from` echo | **CONFIRMED** |
| `:106` | `to` echo | **CONFIRMED** |
| `:107` | "One row per menu item with > 0 sales in the window" | **WRONG SINCE AUTHORING.** `JOIN alloc` is an **inner** join on `recipes`; sales are LEFT-joined. It is one row per menu item **with a recipe**. Items that sold but have no recipe are absent; items with a recipe and no sales are present. Wrong in both directions. `b283f5f`. |
| `:108` | `menu_item_id` = `menu_items.id` | **CONFIRMED** |
| `:109` | `menu_items[].name` | **WRONG SINCE AUTHORING.** The key is `menu_item_name`. Decodes to `""`. |
| `:110` | `menu_items[].menu` — "top-level Toast menu" | **WRONG SINCE AUTHORING — the field does not exist.** Not selected, not in `MenuCOGSRow`. The `menu_items.menu` *column* does exist (`0060`), so the data is in HQ but unreachable via this endpoint. See §8 Q1. |
| `:111` | `menu_items[].menu_group` | **CONFIRMED** |
| `:112` | `menu_subgroup` "string or null; null when absent" | **WRONG.** `omitempty` on a pointer → **the key is omitted**, never `null`. Only detectable by marshalling. |
| `:113` | `units_sold` = `SUM(daily_menu_sales.units_sold)`, **integer** | **WRONG SINCE AUTHORING — type.** The SQL is right; the Go field is `float64`, not an int. Whole values serialise as `42`, not `42.0`, but the declared type is wrong and a strict integer decoder is not guaranteed. 🛑 **RECLASSIFIED 2026-08-03 (second pass). The first pass marked this row DRIFTED and counted it among "only 3 are drift" — but it carries no drifting commit, because there is none.** `MenuCOGSRow.UnitsSold` was born `float64` in `3d9362c` at 10:18 on 2026-06-04; the sentence saying "integer" landed in `b0119c0` at 23:50 **the same day**, thirteen hours later. The document was wrong the moment it was written. A row filed as drift with no commit to name is exactly the defect this audit exists to catch, so it is restated here rather than quietly renumbered. |
| `:114` | `ingredient_cost_per_unit`, null when `units_sold == 0` | **CONFIRMED byte-accurate** — SQL `CASE … THEN NULL`, Go `*float64` with no `omitempty`, verified `null` on the wire |
| `:115` | `ingredient_cost_total` formula | **WRONG SINCE AUTHORING — materially.** Published with a `/ SUM(usage_pct across all menu items using that purchase_item)` normalisation term. **That division does not exist in the code.** Actual: `spend * (usage_pct / 100.0)`. The two agree only when an ingredient's percentages sum to exactly 100; below that the published formula would rescale the residual onto menu items, whereas the code leaves it in `unallocated_cogs`. The tax-proration half (D-11) **is** accurate. |
| `:116` | `ingredients` "empty array `[]` when no recipe rows" | **WRONG.** `omitempty` → key omitted. Moot in practice: such a menu item never appears at all. |
| `:117` | `unallocated_cogs`, summary mode | **CONFIRMED** — set as a non-nil pointer, so present despite `omitempty` |
| `:118`–`:119` | `unallocated` object / `.total`, breakdown mode | **CONFIRMED** |
| `:120` | `by_ingredient[].reason` = `"no recipe"` / `"partial allocation (X%)"` | **CONFIRMED byte-accurate** — both literals match the SQL `CASE` exactly |
| `:122` | Reconciliation invariant against `period-summary cogs_incl_tax` | **BROKEN 2026-06-06 — by the sibling, not by this endpoint.** `a726029` added a category allowlist to `/period-summary` only; `d41faef` folded pending rows into `/period-summary` only. Divergence now runs both ways and is unbounded. Corrected inline; see §8 Q2. |
| `:123`–`:140` | **State A example** — "fully allocated week" | 🛑 **WRONG SINCE AUTHORING — and still published, uncorrected, after the first pass of this audit.** All four bad keys, on **two** menu items: `name` (the key is `menu_item_name`), `menu` (no such field), `menu_subgroup: null` (the key is omitted), and no `toast_master_id`. **This is verbatim the defect the first pass called "the most directly harmful item on the page" at `:243`–`:253` — and its table jumped `:122` → `:153`, so it never audited the example that repeats it.** A reader who corrected their struct from the top of the page and then checked it against State A would have concluded the correction was wrong. Replaced with observed marshalled output. |
| `:142`–`:151` | **State B example** — no recipe coverage | **CONFIRMED.** `{from, to, menu_items: [], unallocated_cogs}` is exactly what `recipes.MenuCOGSResponse` marshals when `MenuItems` is an empty slice and `UnallocatedCogs` is a non-nil pointer. The *prose* under it at `:153` is a separate row and is half wrong. |
| `:169` | "per-unit is the literal JSON `null`, NOT `0`" | **CONFIRMED byte-accurate** — survives the withdrawal of State C above it and is carried into State C′. `*float64` with no `omitempty`. |
| — | **JSON number formatting in every example** | **WRONG SINCE AUTHORING — no line anchor, because it spans every example block on the page.** Go's `encoding/json` emits the shortest round-tripping form of a `float64`, so a SQL `ROUND(x, 4)` of `2.7500` goes on the wire as `2.75`, `115.50` as `115.5`, and `0.00` as `0`. Every example here published trailing-zero literals the endpoint has never emitted; a fixture transcribed from them fails. Corrected throughout, and the field table's "2 decimals"/"4 decimals" clarified as server-side rounding rather than wire format. |
| `:153` | "`menu_items` empty when no recipes exist OR when no menu items have sales" | **HALF WRONG.** Empty when no recipes — yes. Not empty merely for want of sales. |
| `:155`–`:169` | **State C** — "menu item sold but no recipe linked" | **UNREACHABLE — the endpoint has never done this.** Filtered out by the inner join before sales are considered. No test covers it. Withdrawn and replaced with State C′. |
| `:175`–`:180` | Six error envelopes (3×400, 401, 500, 503) | **CONFIRMED byte-accurate** — all six strings identical to Phase 21's, as `:182` claims |
| `:186` | No completeness gate; always 200 given valid dates + auth | **CONFIRMED** |
| `:198` | Env loaded in `main.go` via `os.Getenv` | **CONFIRMED** |
| `:199` | Startup log text | 🛑 **DRIFTED — cosmetic, and the first pass got this one backwards.** The claim it *checked* — that the line names **both** endpoints — is true, and the Phase 21 page had that half wrong. But the string this page publishes is not the string in the code: it prefixes `WARNING: ` and uses an em-dash where `backend/cmd/server/main.go` has a comma and no prefix. `slog.Warn` carries the severity as a structured level, not in the message, so an operator grepping the literal `WARNING:` finds nothing. **Drifting commit: `acd2c7f` (2026-06-22), the `log` → `slog` NDJSON migration** — this page's string was byte-accurate when written (`9f28197`, 2026-06-04) and stopped being so eighteen days later. **The first pass marked this row CONFIRMED while marking the byte-identical string in the sibling document DRIFTED — cosmetic. Same string, two verdicts.** Corrected inline. |
| `:214` | `crypto/subtle.ConstantTimeCompare` | **CONFIRMED** |
| `:243`–`:253` | Published `MenuItemCOGS` client struct | **WRONG SINCE AUTHORING — four bad json tags.** This is the most directly harmful item on the page: it is copy-paste-ready and does not work. Corrected inline. |
| `:202` | "Same token as Phase 21 … both endpoints expose the same trust boundary" | **CONFIRMED** — one `serviceToken := os.Getenv(...)` in `main.go`, one `auth.ServiceTokenMiddleware` group, both routes inside it. |
| `:216` | "the endpoint is wired but inactive until called" | **CONFIRMED** — the route is registered unconditionally; the middleware, not the router, decides 503. No new secret is needed if Phase 21 is already configured. |
| `:235`–`:241` | Published `MenuCOGS` envelope client struct | **CONFIRMED byte-accurate** — `from`, `to`, `menu_items`, `unallocated_cogs,omitempty`, `unallocated,omitempty`: five tags, all matching `recipes.MenuCOGSResponse`. The envelope was right; only the per-item struct below it was wrong. |
| `:255`–`:270` | Published `IngredientAlloc` / `UnallocatedBlock` / `UnallocatedDetail` client structs | **CONFIRMED byte-accurate** — nine json tags across three types, all matching `recipes.IngredientAlloc`, `recipes.UnallocatedBreakdown` and `recipes.UnallocatedDetail`. **Added by the second pass**, which is worth stating plainly: the first pass audited `:243`–`:253` and stopped, so the three structs immediately beneath the one it condemned were never checked. They happen to be right. |
| `:272`–`:282` | §3.1 client implementation notes | **CONFIRMED** — the four status codes it branches on match the error-envelope table, `?breakdown=true` matches the handler's exact-string test, and the `Cache-Control` value matches `:47`. Client-side guidance otherwise. |
| `:296` | §3.2 `MenuCOGS []MenuItemCOGS // one entry per menu item with sales in the week` | 🛑 **WRONG SINCE AUTHORING.** The same inner-join defect as `:107`, restated in a second place. **The first pass's table jumped `:253` → `:380`, so the whole of §3 past the struct, and all of §4, went unaudited.** Corrected inline. |
| `:312` | §3.2 `Show()` sample line — `Iced Tea (30 units) n/a (no recipe linked)` | 🛑 **UNREACHABLE — it renders the State C the first pass withdrew.** A sold-but-uncosted menu item never appears in `menu_items`, so this row cannot be printed from this endpoint's data. Replaced with the reachable case (recipe present, zero sales). |
| `:321` | §3.2 "A footer total that reconciles against `COGSInclTax` (within rounding)" | 🛑 **BROKEN 2026-06-06 — the invariant again, published a second time.** Same cause as `:122`; the first pass corrected the statement at `:122` and left both restatements standing. Withdrawn inline. |
| `:390` | Scenario 1 expected — "one row per menu item that has sales" | 🛑 **WRONG SINCE AUTHORING.** Third publication of the `:107` defect, this time as an acceptance criterion HQ handed the counterparty. Corrected inline. |
| `:393` | Scenario 1 expected — "Sum of per-menu costs + unallocated ≈ Phase 21's `COGSInclTax`" | 🛑 **BROKEN 2026-06-06 — the invariant a third time, and the only one written as a test that runs.** A sales-processor team asserting this against a real HQ sees it fail, with nothing on this page explaining why. Withdrawn inline; see §8 Q4. |
| `:409`–`:416` | Scenario 3 — breakdown-mode expectations | **CONFIRMED** — `loadBreakdown` populates `ingredients` per row with `usage_pct` + `allocated_cost`, and `loadUnallocatedBreakdown` populates `unallocated.by_ingredient` with a `reason`. All three expectations are producible. |
| `:418`–`:427` | Scenario 4 — 503 when the env var is unset | **CONFIRMED** — both routes sit in the same `ServiceTokenMiddleware` group reading the same `HQ_INVENTORY_SERVICE_TOKEN`, so `/period-summary` returns 503 too. |
| `:429`–`:437` | Scenario 5 — 401 on token mismatch | **CONFIRMED** — same middleware, same constant-time compare, byte-identical envelope. |
| `:380` | "9 tests covering 200/400/401/503/units=0/cache header" | **CONFIRMED** — exactly nine `Test` functions, categories match |
| `:395`–`:405` | **Scenario 2** — acceptance test for "sold but no recipe" | **CANNOT PASS.** Follows from `:107` / State C. Withdrawn, replaced with Scenario 2′. |
| `:445`–`:448` | A1–A4 (sales-processor repo layout, `WeeklySummary`, CLI, non-fatal fetch) | **NOT VERIFIABLE FROM HQ** — they describe a repo not present here |
| `:449` | A5 `units_sold` truth source is HQ | **CONFIRMED** — the query joins `daily_menu_sales` server-side |
| `:450` | A6 rounding tolerance | **SUPERSEDED** by the `:122` finding — the discrepancy is no longer a rounding matter |
| `:451` | A7 null per-unit must decode to a nullable type | **CONFIRMED** |
| `:452` | A8 single-tenant, no `tenant_id` | **CONFIRMED** — no tenant parameter anywhere |
| `:453` | A10 timezone — "shares date semantics, so it moves with it" | **FALSE.** Same defect as `:31`, same origin — card A1, `1be1a59`, **2026-07-31**, merged in `67b2c9f` the same day. Corrected inline, and the misdated attribution corrected with it. |
| `:454` | A9 drift signal is out of band | **CONFIRMED** — `drift.go` + scheduler; the endpoint carries no drift field |
| `:465` | "D-21 locks 10%/20% in HQ code" | **CONFIRMED** — `drift.go`: unallocated at `SUM(usage_pct) < 90` (i.e. >10% unallocated), divergence at `abs(configured − actual) > 20` |
| `:474`–`:479` | §7 reference-implementation paths | **DRIFTED — stale.** All five `.planning/phases/999.2-…` paths deleted by `34f8c7e` (2026-07-26). Replaced. |
| `:480` | "integration tests are the executable proof that the HQ side matches this contract" | **FALSE, and causally responsible.** The tests decode into the same struct the handler marshals, so every wrong field name on this page is invisible to them. Green suite, wrong contract, fourteen months. Corrected inline; test-shape gap filed as **B-71**. *(Corrected 2026-08-03, second pass: the first pass cited **B-73** here and at §7. B-73 is the broken cross-endpoint invariant; the test-shape gap is **B-71**. Wrong ID, twice, in an external-facing contract.)* |
| `:486`–`:505` | Four smoke-test `curl` commands | **CONFIRMED as invocations** — paths, params and expected status codes are right. ⚠️ But `:490`'s "Expected: 200, JSON with from/to/menu_items[]/unallocated_cogs" describes the envelope only; the **per-item keys** it implies are the wrong ones. Envelope confirmed, item shape corrected above. |

**Counting unit — stated so the totals are checkable.** One **row of the table above** = one audited unit, whatever line range it anchors. Count the rows; the numbers below must match. Every row carries exactly one verdict bucket.

**Rows audited: 64** — 63 line-anchored entries spanning `:9`–`:505`, plus one row with no line anchor (a defect spanning every example block). **35 CONFIRMED byte-accurate · 1 NOT VERIFIABLE FROM HQ · 1 SUPERSEDED · 27 WRONG, unreachable, broken or stale.** 35 + 1 + 1 + 27 = 64.

🛑 **The first pass of this audit reported "44 rows · 24 CONFIRMED · 4 not verifiable · 16 WRONG", and none of those four numbers was right under any consistent counting unit — its own table already held 47 rows (26 CONFIRMED, 1 not verifiable, 1 superseded, 19 wrong).** More to the point, its coverage did not match its claim. The table jumped `:122` → `:153` and `:253` → `:380`, leaving unaudited: the **three example response states** — including State A, which still published all four wrong per-item keys on two menu items after the same pass had called that defect "the most directly harmful item on the page" — and the whole of **§3 past the client struct** and **§4**. Seventeen rows were added on the second pass, **seven** of them wrong, and one existing row (`:199`, the startup-log string) was reclassified **CONFIRMED → DRIFTED**. Three of the seven are the `:107` inner-join defect and the `:122` broken invariant, each republished in places the first pass never opened. The arithmetic from the first pass's own table: 26 + 1 + 1 + 19 = 47 → −1 CONFIRMED (`:199` reclassified) + 10 CONFIRMED, + 7 wrong = 35 + 1 + 1 + 27 = 64.

**Of the 27, 5 are drift** in the sense the sibling document has it — a claim that was true and stopped being true: the reconciliation invariant, published in **three** places (`:122`, `:321`, `:393`) and broken in all three by the same pair of sibling-side commits on 2026-06-06; the stale `.planning` paths at `:474`–`:479`; and the startup-log string at `:199`, which was byte-accurate when written and drifted in `acd2c7f` on 2026-06-22. **The other 22 were never true.** 🛑 **`:113` (`units_sold` declared `integer`) has moved out of the drift bucket on the second pass: it carried no drifting commit because there is none.** `UnitsSold` was born `float64` at 10:18 on 2026-06-04 (`3d9362c`); the sentence calling it an integer landed at 23:50 the same day (`b0119c0`). Never true, never drift — and a drifted row with no commit to name is the exact defect this audit exists to catch.

The never-true bucket is a different failure from the sibling's and it wants a different remedy: the sibling drifted because code moved without the doc; this one was born wrong because the doc was written from the plan and nobody diffed it against the shipped handler — and the page's own claim at `:480` that the tests proved otherwise is what kept anyone from looking.

**No HQ code was changed by this revision.** Every correction moved the document to match the code. Where that is arguably the wrong direction — and here, unlike the sibling, it may genuinely be — see §8.

---

## 1. The Endpoint

### Request

```
GET /api/v1/inventory/menu-cogs?from=YYYY-MM-DD&to=YYYY-MM-DD HTTP/1.1
Host: hq.yumyums.kitchen
Authorization: Bearer <HQ_INVENTORY_SERVICE_TOKEN>
Accept: application/json
```

- **Method:** `GET` (idempotent, no body, cacheable).
- **Path:** exactly `/api/v1/inventory/menu-cogs`.
- **Query params:**
  - `from` (required) — start date, format `YYYY-MM-DD`, inclusive.
  - `to` (required) — end date, format `YYYY-MM-DD`, inclusive.
  - `breakdown` (optional) — `true` for per-ingredient detail per menu item; default `false` (summary only).
  - The two dates define an inclusive calendar window. For a Monday–Sunday workweek "May 25–31, 2026", send `from=2026-05-25&to=2026-05-31`.
  - 🛑 **CORRECTED 2026-08-03 — this endpoint has NO timezone dependency at all.** The previous text said the dates are "interpreted in `America/New_York`" and that the endpoint "shares its date semantics with Phase 21's `/period-summary`, so it moves with it" on the timezone changeover deploy. **Both statements are wrong for this endpoint.** Every date comparison here is against a plain SQL `DATE` column with **no timezone cast anywhere in any of the four queries** — `purchase_events.event_date` for spend, `daily_menu_sales.business_date` for units. There is no `AT TIME ZONE` in this handler. **Nothing about this endpoint's output changes on the `America/New_York` changeover deploy**, and sales-processor needs to make no change here on account of it.
  - **What is still true:** the *operator-facing* week — which seven days sales-processor asks for — should stay aligned with Phase 21's, and Phase 21 **does** have a genuine (if narrow) zone dependency. So read assumption **A5 of `21-SALES-PROCESSOR-CONTRACT.md` §5** for the coordinated-release requirement; just do not expect this endpoint to move when it happens. See also A10 in §5 below, corrected the same day.
- **Auth header:** `Authorization: Bearer <token>`. The token is an opaque string. Sales-processor reads it from the env var `HQ_INVENTORY_SERVICE_TOKEN`. **Same token as Phase 21's `/period-summary`** — there is one HQ inventory service token, shared by both endpoints.

### Base URL — confirm with operator

The HQ base URL is the same domain used by the PWA. The expected value:

```
HQ_BASE_URL=https://hq.yumyums.kitchen
```

**[ACTION REQUIRED — user/operator confirmation]:** Same as Phase 21's contract — confirm that `hq.yumyums.kitchen` is the correct hostname for sales-processor to reach (it routes through Cloudflare Tunnel per project memory). If sales-processor runs on the same Windows box as HQ, a LAN/Tailscale address may be preferred to avoid the tunnel round-trip. The base URL is shared across all HQ endpoints — confirm once.

### Response — success (200 OK)

`Content-Type: application/json`
`Cache-Control: private, max-age=3600`

🛑 **CORRECTED 2026-08-03 — the examples below were WRONG FROM THE DAY THIS DOCUMENT WAS WRITTEN (2026-06-04), not drifted into wrongness later.** This contract was authored from the phase plan rather than from the shipped handler, and **four of the nine per-item field names never matched the code.** A client built literally against the previous version of this page decodes `name` and `menu` as empty strings on every row. The shapes below were **observed** by marshalling `recipes.MenuCOGSResponse` at HEAD and printing the JSON. See §0 for the row-by-row audit.

**Summary mode (default):**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    {
      "menu_item_id": "ae56f5eb-8399-4dbb-8872-70fe16132d16",
      "toast_master_id": "M-1",
      "menu_item_name": "Smashburger",
      "menu_group": "Sandwiches",
      "units_sold": 42,
      "ingredient_cost_per_unit": 2.75,
      "ingredient_cost_total": 115.5
    }
  ],
  "unallocated_cogs": 89
}
```

Note what is **not** there: no `name` (it is `menu_item_name`), no `menu` at all, and **no `menu_subgroup` key whatsoever** when the item has no subgroup — the field is `omitempty`, so it is omitted rather than emitted as `null`. `toast_master_id` is present and was never documented.

**Breakdown mode (`?breakdown=true`):**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    {
      "menu_item_id": "ae56f5eb-8399-4dbb-8872-70fe16132d16",
      "toast_master_id": "M-1",
      "menu_item_name": "Smashburger",
      "menu_group": "Sandwiches",
      "menu_subgroup": "Burgers",
      "units_sold": 42,
      "ingredient_cost_per_unit": 2.75,
      "ingredient_cost_total": 115.5,
      "ingredients": [
        { "purchase_item_description": "Ground Beef 80/20", "usage_pct": 50, "allocated_cost": 75 },
        { "purchase_item_description": "Brioche Buns", "usage_pct": 100, "allocated_cost": 40.5 }
      ]
    }
  ],
  "unallocated": {
    "total": 89,
    "by_ingredient": [
      { "purchase_item_description": "Olive Oil", "amount": 60, "reason": "no recipe" },
      { "purchase_item_description": "Tortillas", "amount": 29, "reason": "partial allocation (40%)" }
    ]
  }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `from` | string | Echo of the input `from` (YYYY-MM-DD). |
| `to` | string | Echo of the input `to` (YYYY-MM-DD). |
| `menu_items` | array of objects | 🛑 **CORRECTED 2026-08-03 — this row was never accurate.** It is **not** "one row per menu item with > 0 sales." The query inner-joins `menu_items` to `recipes`, then LEFT-joins sales. So it is **one row per menu item that has at least one recipe row**, with sales attached where they exist. Two consequences the previous text got backwards: **(a)** a menu item that sold units but has **no recipe row is absent entirely** — not present with a null cost; **(b)** a menu item with a recipe but **zero sales in the window is present**, with `units_sold: 0` and a null per-unit. Empty array `[]` when no recipes exist at all. |
| `menu_items[].menu_item_id` | string (UUID) | Stable `menu_items.id`. Use this when correlating across reports — not the display name (which can collide across menus per D-09). |
| `menu_items[].toast_master_id` | string | **NEVER PUBLISHED UNTIL 2026-08-03.** `menu_items.master_id` — Toast's own stable identifier, unique per menu item. Present on every row since the endpoint shipped. |
| `menu_items[].menu_item_name` | string | Display name from Toast. 🛑 **The JSON key is `menu_item_name`. This document said `name` from 2026-06-04 until 2026-08-03 and the code never emitted that key.** A client decoding `json:"name"` gets `""` on every row. |
| ~~`menu_items[].menu`~~ | — | 🛑 **THIS FIELD DOES NOT EXIST AND NEVER DID.** Published as "Top-level Toast menu (e.g. 'Lunch Menu')" on 2026-06-04; the handler has never selected it and `MenuCOGSRow` has no such field. The underlying `menu_items.menu` **column** does exist (migration `0060`), so the data is in HQ — it is simply not returned. Whether to start returning it is an open question, see §8 Q1. Until then, **a client cannot get the top-level menu from this endpoint.** |
| `menu_items[].menu_group` | string | Toast menu group (e.g. "Sandwiches"). |
| `menu_items[].menu_subgroup` | string, or **key absent** | 🛑 **CORRECTED 2026-08-03.** Published as "string or null; null when the menu item has no subgroup." The field is `omitempty` on a pointer, so when there is no subgroup **the key is omitted from the object entirely** — it is never the JSON literal `null`. Harmless for a decoder targeting a nullable type (both yield nil), but a client that checks for key *presence*, or that round-trips the object, will see something the contract did not describe. |
| `menu_items[].units_sold` | number | `SUM(daily_menu_sales.units_sold)` over the window, where `daily_menu_sales.business_date BETWEEN from AND to`. Per D-13, HQ is the truth source — sales-processor must NOT aggregate from Toast directly when this field is present. **Corrected 2026-08-03: published as `integer`; the Go field is a `float64`.** It serialises without a decimal point for whole values, so `42` not `42.0`, but a non-integral total would serialise as a decimal and a strict integer decoder would fail. |
| `menu_items[].ingredient_cost_per_unit` | number (4 decimals) or null | `ingredient_cost_total / units_sold`. **Null when `units_sold == 0`** — JSON literal `null` (not 0). Verified byte-accurate: the SQL `CASE` returns NULL and the Go field carries no `omitempty`, so `null` is emitted. |
| `menu_items[].ingredient_cost_total` | number (2 decimals) | Tax-inclusive ingredient cost for the week. 🛑 **CORRECTED 2026-08-03 — the published formula was never the implemented one.** It is **not** `usage_pct * window_spend / SUM(usage_pct across all menu items using that purchase_item)`. There is **no division by the sum of usage percentages** anywhere in the code. The actual computation, summed over each recipe row linking this menu item to a `purchase_item`, is simply `window_spend_for_that_purchase_item * (usage_pct / 100.0)`. The two agree only in the special case where the percentages for an ingredient sum to exactly 100. **Where they sum to less than 100 — the normal case — the published formula would have rescaled the remainder onto the menu items; the code instead leaves it in `unallocated_cogs`, which is the entire point of that field.** The code is self-consistent; the formula on this page was not. The "tax-inclusive" piece is accurate and is documented in D-11 — tax is pro-rated to each line item by its share of the event subtotal (`total / NULLIF(total - tax, 0)`). |
| `menu_items[].ingredients` | array (breakdown mode only) | Per-ingredient breakdown of `ingredient_cost_total`. **Corrected 2026-08-03:** the field is `omitempty`, so when a menu item has no recipe rows **the key is omitted**, not emitted as `[]`. In practice this case is unreachable — a menu item with no recipe rows does not appear in `menu_items` at all (see that row above). |
| `unallocated_cogs` | number (summary mode only, 2 decimals) | Dollar residual of window spend not allocated to any menu item. |
| `unallocated` | object (breakdown mode only) | Replaces `unallocated_cogs` in breakdown mode. |
| `unallocated.total` | number | Same dollar residual. |
| `unallocated.by_ingredient` | array | One row per `purchase_item` contributing to the residual. `reason` is either `"no recipe"` (no recipe rows reference the purchase_item) or `"partial allocation (X%)"` (sum of usage_pct < 100). |

**Invariant — 🛑 BROKEN SINCE 2026-06-06. Do not assert on it.** This document published: `SUM(menu_items[].ingredient_cost_total) + unallocated_cogs ≈ period-summary cogs_incl_tax` for the same window, tolerating rounding under $0.10. That held when both endpoints were written. It stopped holding on 2026-06-06 and **the two sides now disagree by unbounded amounts**, because `/period-summary` changed and this endpoint did not:

- **`a726029` (2026-06-05)** made `/period-summary` filter `purchase_events` by `mercury_category = ANY(HQ_COGS_CATEGORY_ALLOWLIST)`. This endpoint applies **no category filter** — it costs every line item with a `purchase_item_id`. So `/menu-cogs` sees spend that `/period-summary` excludes.
- **`d41faef` (2026-06-06)** made `/period-summary` fold unconfirmed `pending_purchases` into `cogs_excl_tax` / `cogs_incl_tax` at their full `ABS(bank_total)`. This endpoint reads only `purchase_line_items`, which pending rows do not have yet. So `/period-summary` sees spend that `/menu-cogs` cannot.

The divergence runs in **both** directions and its size depends on how much of the period is miscategorised and how much is still in the review queue — neither of which is bounded, and neither of which is knowable from either response alone. **A sales-processor reconciliation check written against the old invariant will fire spuriously.** The two endpoints answer genuinely different questions now; treat `/menu-cogs` as an attribution view over *confirmed, catalogued* line items, not as a decomposition of the Phase 21 aggregate. Whether they should be re-aligned is an open question — see §8 Q2.

### Response — example states

🛑 **CORRECTED 2026-08-03 (second pass) — State A below was still publishing all four wrong keys, on two menu items, after the first pass of this audit had already called that exact defect "the most directly harmful item on the page."** It carried `name` instead of `menu_item_name`, a `menu` field that does not exist, `menu_subgroup: null` where the key is omitted, and no `toast_master_id`. The first pass's table jumped `:122` → `:153` and never mentioned it, so the page corrected the shape at the top and then contradicted itself three screens down. Replaced below with the observed marshalled output of `recipes.MenuCOGSResponse`.

📐 **Read the numbers literally.** Go's `encoding/json` emits the shortest representation that round-trips a `float64`, so a value SQL rounded to `2.7500` goes on the wire as `2.75`, `115.50` as `115.5`, and `0.00` as `0`. "4 decimals" / "2 decimals" in the field table describe the **server-side `ROUND()`**, not the wire format. Every example on this page previously published trailing-zero literals the endpoint has never emitted; they are gone.

**State A: fully allocated week**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    {
      "menu_item_id": "ae56f5eb-8399-4dbb-8872-70fe16132d16",
      "toast_master_id": "M-1",
      "menu_item_name": "Smashburger",
      "menu_group": "Sandwiches",
      "units_sold": 42,
      "ingredient_cost_per_unit": 2.75,
      "ingredient_cost_total": 115.5
    },
    {
      "menu_item_id": "5b1f0c72-2f4d-4a63-9a7e-1c3a5d8e9012",
      "toast_master_id": "M-2",
      "menu_item_name": "Tacos al Pastor",
      "menu_group": "Tacos",
      "units_sold": 80,
      "ingredient_cost_per_unit": 1.5,
      "ingredient_cost_total": 120
    }
  ],
  "unallocated_cogs": 0
}
```

Neither row carries a `menu_subgroup` key: both items have none, and the field is `omitempty` on a pointer, so it is **omitted, not `null`**. Neither carries `menu`, which does not exist. Both carry `toast_master_id`, which no version of this page documented before 2026-08-03.

**State B: week with no recipe coverage at all (everything unallocated)**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [],
  "unallocated_cogs": 235.5
}
```

(**Corrected 2026-08-03:** `menu_items` is empty when **no recipes exist** — that much is right. It is **not** empty merely because no menu items have sales: a menu item with a recipe and zero sales still appears, with `units_sold: 0`. Sales-processor must NOT assume non-empty, and must NOT assume every returned row sold something.)

**State C — 🛑 THIS STATE IS UNREACHABLE. The example below is what the code does NOT do.**

This document published State C as "menu item sold but no recipe linked (units_sold > 0, per-unit is null)", with this response:

```json
{
  "menu_items": [
    { "menu_item_id": "...", "name": "Iced Tea", "menu": "Beverages", "menu_group": "Drinks", "menu_subgroup": null,
      "units_sold": 30, "ingredient_cost_per_unit": null, "ingredient_cost_total": 0.00 }
  ],
  "unallocated_cogs": 18
}
```

**The endpoint has never behaved this way.** The summary query inner-joins `menu_items` to the recipe allocation set, so a menu item with **no** recipe rows is filtered out before sales are ever considered. An Iced Tea that sold 30 units with no recipe linked does not appear in `menu_items` at all — it is silently absent, and the operator sees no row telling them it is uncosted. (Its ingredient spend, if any of its ingredients were catalogued, still lands in `unallocated_cogs` — but with nothing naming the menu item.) There is no HQ-side test covering this case, which is why it went unnoticed for fourteen months; the nearest test, `TestMenuCogs_UnitsSoldZero_PerUnitIsNull`, exercises the **opposite** case (recipe present, no sales).

Whether this is a defect worth fixing is an open question — see §8 Q3. **Sales-processor should not rely on this endpoint to enumerate menu items lacking recipes.**

**State C′ (the reachable one): menu item with a recipe but no sales in the window**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    { "menu_item_id": "...", "toast_master_id": "M-9", "menu_item_name": "Iced Tea", "menu_group": "Drinks",
      "units_sold": 0, "ingredient_cost_per_unit": null, "ingredient_cost_total": 4.2 }
  ],
  "unallocated_cogs": 18
}
```

(per-unit is the literal JSON `null`, NOT `0`. Decode into `*float64` / `Optional<Double>` / equivalent; rendering should show "—" or "n/a" rather than "$0.00". Note that `ingredient_cost_total` can be **non-zero while `units_sold` is zero** — ingredient spend is allocated by recipe percentage regardless of whether anything sold.)

### Response — error states

| HTTP | Body | When |
|------|------|------|
| 400 | `{"error":"from must be YYYY-MM-DD"}` | `from` query param malformed or missing |
| 400 | `{"error":"to must be YYYY-MM-DD"}` | `to` query param malformed or missing |
| 400 | `{"error":"from must be <= to"}` | `from > to` lexicographic |
| 401 | `{"error":"unauthorized"}` | `Authorization` header missing, malformed (no `Bearer ` prefix), or token mismatch |
| 500 | `{"error":"internal_error"}` | DB error on the HQ side; details only in HQ server logs |
| 503 | `{"error":"service_token_not_configured"}` | HQ has `HQ_INVENTORY_SERVICE_TOKEN` unset — operator must configure |

**Byte-for-byte identical to Phase 21's `/period-summary` error envelope** per D-18. Sales-processor's error-handling for the two endpoints can share one branch.

Sales-processor should distinguish 503 (config error on HQ side — surface to operator, do not retry blindly) from 500 (transient — may retry once).

**No completeness gate field.** Per D-15, this endpoint always returns 200 (given valid dates + auth). Drift surfaces in-app via the Recipes-tab banner and a weekly Cliq alert — NOT as an endpoint-level boolean. Sales-processor renders whatever `menu_items` it gets, even if the underlying recipes are incomplete; if the operator cares about coverage, they look at the HQ Recipes tab.

---

## 2. Env Var Contract

### HQ side (this repo — already shipped)

```
HQ_INVENTORY_SERVICE_TOKEN=<opaque-string>
```

- **Where loaded:** `backend/cmd/server/main.go` via `os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")` (single env var serves both endpoints).
- **Empty behavior:** server logs `HQ_INVENTORY_SERVICE_TOKEN not set, /api/v1/inventory/period-summary AND /api/v1/inventory/menu-cogs will return 503` at startup, BOTH endpoints return 503 on every request (fail-closed). *(Corrected 2026-08-03, second pass: this page published the message with a `WARNING: ` prefix and an em-dash where the code has a comma. `backend/cmd/server/main.go` calls `slog.Warn` with the string above — the severity is `slog`'s structured level field, not part of the message, so an operator grepping for the literal `WARNING:` finds nothing. The published form was accurate on 2026-06-04 (`9f28197`) and drifted on 2026-06-22 in `acd2c7f`, the `log` → `slog` migration. The first pass marked this row **CONFIRMED** while marking the byte-identical claim in the sibling document **DRIFTED — cosmetic**; same string, two verdicts, and this page was the one that was wrong.)*
- **Format:** opaque string, no whitespace, no encoding requirements. Recommend 32+ random bytes hex- or base64-encoded.
- **Storage:** managed as an env var in the Cloudflare Tunnel / docker-compose / systemd unit running the HQ backend on the Windows box. NOT committed to the repo.
- **Same token as Phase 21.** Do NOT rotate to a per-endpoint token — that would double the operator's secret-management burden for no security gain (both endpoints expose the same trust boundary).

### sales-processor side (separate repo — sales-processor team implements)

The sales-processor must read TWO env vars:

```
HQ_BASE_URL=https://hq.yumyums.kitchen
HQ_INVENTORY_SERVICE_TOKEN=<must match HQ's value byte-for-byte>
```

- **`HQ_BASE_URL`:** the protocol + host where HQ is reachable. No trailing slash. Sales-processor's HTTPClient appends `/api/v1/inventory/menu-cogs?...` to this.
- **`HQ_INVENTORY_SERVICE_TOKEN`:** the SAME secret as on the HQ side. Same one Phase 21 already uses — no new secret needed if sales-processor already has Phase 21 wired up. Comparison on HQ uses `crypto/subtle.ConstantTimeCompare` (timing-safe).

**[ACTION REQUIRED — operator]:** If Phase 21 is already deployed, `HQ_INVENTORY_SERVICE_TOKEN` is already set on both sides. This phase requires NO new secret. Just deploy the new HQ binary (endpoint is wired but inactive until called).

---

## 3. Sales-Processor Implementation Contract

### 3.1 HTTPClient (`service/external/hq.go`)

If Phase 21's `HQClient` already exists, EXTEND it with a `GetMenuCOGS` method rather than creating a sibling client. The two endpoints share auth, base URL, and error semantics — one client is the right factoring.

```go
// PSEUDOCODE — adapt to sales-processor's actual project layout and HTTP conventions.
package external

type HQClient interface {
    GetPeriodSummary(ctx context.Context, from, to time.Time) (*PeriodSummary, error)
    GetMenuCOGS(ctx context.Context, from, to time.Time, breakdown bool) (*MenuCOGS, error)
}

type MenuCOGS struct {
    From            string             `json:"from"`
    To              string             `json:"to"`
    MenuItems       []MenuItemCOGS     `json:"menu_items"`
    UnallocatedCOGS *float64           `json:"unallocated_cogs,omitempty"` // summary mode
    Unallocated     *UnallocatedBlock  `json:"unallocated,omitempty"`      // breakdown mode
}

// CORRECTED 2026-08-03. The previous version of this struct had FOUR wrong
// json tags and would have decoded Name and Menu as "" on every row. If
// sales-processor already implemented against it, THIS is the fix.
type MenuItemCOGS struct {
    MenuItemID            string              `json:"menu_item_id"`
    ToastMasterID         string              `json:"toast_master_id"`       // was undocumented
    MenuItemName          string              `json:"menu_item_name"`        // was published as `json:"name"` — wrong
    MenuGroup             string              `json:"menu_group"`
    MenuSubgroup          *string             `json:"menu_subgroup"`         // key ABSENT when there is no subgroup, never null
    UnitsSold             float64             `json:"units_sold"`            // was published as int — HQ emits a JSON number from a float64
    IngredientCostPerUnit *float64            `json:"ingredient_cost_per_unit"` // null when units_sold == 0
    IngredientCostTotal   float64             `json:"ingredient_cost_total"`
    Ingredients           []IngredientAlloc   `json:"ingredients,omitempty"` // breakdown mode only
}

// There is no `Menu` field. HQ does not return the top-level Toast menu on
// this endpoint, despite the previous version of this document listing it.


type IngredientAlloc struct {
    PurchaseItemDescription string  `json:"purchase_item_description"`
    UsagePct                float64 `json:"usage_pct"`
    AllocatedCost           float64 `json:"allocated_cost"`
}

type UnallocatedBlock struct {
    Total        float64               `json:"total"`
    ByIngredient []UnallocatedDetail   `json:"by_ingredient"`
}

type UnallocatedDetail struct {
    PurchaseItemDescription string  `json:"purchase_item_description"`
    Amount                  float64 `json:"amount"`
    Reason                  string  `json:"reason"` // "no recipe" or "partial allocation (X%)"
}

// Implementation:
//   - Use net/http.Client with explicit Timeout (e.g. 10s). Do NOT use http.DefaultClient (no timeout).
//   - Format dates as "2006-01-02".
//   - Append ?breakdown=true to the URL when the caller asks for it.
//   - Set Authorization: Bearer <token>.
//   - On 200, decode JSON into MenuCOGS.
//   - On 401, return a typed error so the caller can surface "HQ rejected our token".
//   - On 503, return a typed error so the caller can distinguish HQ-misconfigured.
//   - On 500, treat as transient — caller may retry once.
//   - On 400, treat as a programmer error — surface and fail.
//   - Honor the Cache-Control: private, max-age=3600 header if the caller has a cache layer.
```

### 3.2 WeeklySummary fields + Show() rendering

The sales-processor's existing `WeeklySummary` struct (already extended in Phase 21 with `COGS` + `COGSInclTax`) must gain a new field for the per-menu-item breakdown:

```go
type WeeklySummary struct {
    // ... Phase 21 fields ...
    COGS              float64           // tax-EXCLUDED COGS for the week
    COGSInclTax       float64           // tax-INCLUDED COGS for the week

    // Phase 999.2: per-menu-item breakdown.
    MenuCOGS          []MenuItemCOGS    // CORRECTED 2026-08-03 (second pass): NOT "one entry per
                                        // menu item with sales in the week". One entry per menu
                                        // item that HAS A RECIPE — sales or no sales. Same
                                        // inner-join defect as the `menu_items` row in §1.
    UnallocatedCOGS   float64           // residual not attributed to any menu item
}
```

The `Show()` method renders these AFTER the existing Net Sales / COGS lines. Exact label text is sales-processor's choice but should match the existing report style. Example:

```
Net Sales:           $5,432.10
COGS (excl tax):     $1,234.56
COGS (incl tax):     $1,334.56
Gross Margin:        $4,197.54        // (Net Sales - COGS excl tax)

COGS by Menu Item:
  Smashburger          (42 units)    $115.50    ($2.75/unit)
  Tacos al Pastor      (80 units)    $120.00    ($1.50/unit)
  Iced Tea              (0 units)        n/a    (recipe, no sales this week)
  Unallocated                         $89.00
                                   ---------
  Total                              $324.50
```

Layout details are sales-processor's call. The data points the operator needs:
- Per-menu-item: name, units sold, cost total, cost per unit (or "n/a" when null).
- Unallocated total — surfaces incomplete recipes (the food-truck operator's signal to add more recipe rows).
- ~~A footer total that reconciles against `COGSInclTax` (within rounding).~~ 🛑 **WITHDRAWN 2026-08-03 (second pass). Do not build this.** It restates, in a region the first pass of this audit never looked at, the reconciliation invariant that broke on 2026-06-06 — see the Invariant note in §1 and §8 Q4. `/period-summary`'s `COGSInclTax` is category-filtered and includes unconfirmed pending rows; this endpoint's totals are neither. A footer built to reconcile the two will not, by an amount nothing bounds. Render the footer as this endpoint's own total, and do not compare it to Phase 21's.

### 3.3 Endpoint selection — when to use summary vs breakdown

```go
// PSEUDOCODE — adapt to the actual CLI framework and entry point.
func runWeekly(ctx context.Context, from, to time.Time) error {
    // Phase 21: always fetch the aggregate.
    summary, err := hqClient.GetPeriodSummary(ctx, from, to)
    if err != nil { return err }

    // Phase 999.2: fetch summary mode by default. Breakdown mode is only
    // needed when --breakdown / --verbose is requested or when an operator
    // wants the per-ingredient drilldown printed.
    menuCogs, err := hqClient.GetMenuCOGS(ctx, from, to, false /* breakdown */)
    if err != nil {
        // Phase 999.2 should NOT block the report — Phase 21's gate is the
        // hard-fail. If menu-cogs fails (e.g. 500 transient), log a warning
        // and proceed with aggregate-only output.
        log.Printf("warning: menu-cogs fetch failed (continuing with aggregate-only): %v", err)
    } else {
        weekly.MenuCOGS = menuCogs.MenuItems
        if menuCogs.UnallocatedCOGS != nil {
            weekly.UnallocatedCOGS = *menuCogs.UnallocatedCOGS
        }
    }

    // Phase 21 gate is unchanged.
    if !summary.Completeness.Ready && !forcePayroll {
        // Same gate as Phase 21 — menu-cogs has no completeness gate (D-15).
        return errors.New("receipts not ready for payroll")
    }

    return writePayrollArtifacts(ctx, weekly)
}
```

### 3.4 Optional `--breakdown` CLI flag (sales-processor team's call)

If the sales-processor maintainer wants per-ingredient drilldown in the report, add a `--breakdown` flag and route it through `GetMenuCOGS(ctx, from, to, true)`. Render `MenuItemCOGS.Ingredients` as a nested table:

```
COGS by Menu Item (with ingredient breakdown):
  Smashburger          (42 units)    $115.50    ($2.75/unit)
    Ground Beef 80/20  50%           $75.00
    Brioche Buns       100%          $40.50
  Tacos al Pastor      (80 units)    $120.00    ($1.50/unit)
    ...
  Unallocated                         $89.00
    Olive Oil          no recipe      $60.00
    Tortillas          partial (40%)  $29.00
```

This is optional — the summary mode alone covers the primary "COGS by Menu Item" report line item. Breakdown mode exists for the operator's debugging / recipe-tuning workflow.

---

## 4. Acceptance Scenarios

Sales-processor must demonstrate each scenario passes. The HQ side already provides integration tests for the endpoint behavior (`backend/internal/recipes/menu_cogs_test.go` — 9 tests covering 200/400/401/503/units=0/cache header); the scenarios below are end-to-end through sales-processor.

### Scenario 1 — Fully-allocated week renders COGS by menu item

**Setup:** HQ has recipes covering all purchase_items for the week; menu items have sales in `daily_menu_sales`.

**Invocation:** `./sales-processor weekly --from 2026-05-25 --to 2026-05-31`

**Expected:**
- Exit code: zero (Phase 21 gate passes).
- Report contains a "COGS by Menu Item" section with one row per menu item **that has a recipe** — 🛑 **corrected 2026-08-03 (second pass); this line said "that has sales", the same inner-join defect the first pass corrected in §1 and did not look for here.** Items that sold with no recipe are absent from the response entirely (see State C); items with a recipe and no sales are present with `units_sold: 0`.
- Each row shows units, total, and per-unit (or "n/a").
- Unallocated line is shown when > $0.
- ~~Sum of per-menu costs + unallocated ≈ Phase 21's `COGSInclTax` (within rounding).~~ 🛑 **WITHDRAWN 2026-08-03 (second pass). This assertion FAILS against a real HQ and has since 2026-06-06** — it is the broken invariant again, published a third time, here as an acceptance criterion. A sales-processor team that wrote this check would see it fail with no way to tell from this document whether the fault was theirs. See §8 Q4.

### Scenario 2 — 🛑 WITHDRAWN 2026-08-03. This scenario cannot pass and never could.

It was published as: *"Menu item sold but no recipe — null per-unit handled. Setup: HQ has at least one menu_item with `units_sold > 0` but NO recipe rows linking to it. Expected: that menu item appears in the COGS-by-menu-item table, per-unit shows n/a, cost-total shows $0.00."*

**HQ does not return that menu item at all** — see State C above. A sales-processor team that wrote this acceptance test against a real HQ would have found it failing and had no way to tell from this document whether the fault was theirs or HQ's. It was HQ's, and it was in this page rather than in the code.

**Replacement — Scenario 2′: menu item with a recipe but no sales — null per-unit handled**

**Setup:** HQ has at least one `menu_item` with recipe rows but **no** `daily_menu_sales` rows in the window.

**Invocation:** Same as Scenario 1.

**Expected:**
- That menu item appears in the COGS-by-menu-item table with `units_sold: 0`.
- Per-unit column shows "n/a" / "—" (NOT "$0.00").
- Cost-total column may be **non-zero** — ingredient spend is allocated by recipe percentage regardless of sales.

This is the case HQ actually covers, by `TestMenuCogs_UnitsSoldZero_PerUnitIsNull`.

### Scenario 3 — Breakdown mode (if implemented)

**Setup:** Same as Scenario 1.

**Invocation:** `./sales-processor weekly --from 2026-05-25 --to 2026-05-31 --breakdown`

**Expected:**
- Per-menu rows expand to show ingredient sub-rows.
- Each ingredient sub-row shows usage_pct + allocated $.
- Unallocated section expands to show `by_ingredient` detail with reason.

### Scenario 4 — HQ returns 503 (env var unset)

**Setup:** HQ has `HQ_INVENTORY_SERVICE_TOKEN` unset.

**Invocation:** Same as Scenario 1.

**Expected:**
- Phase 21's `/period-summary` ALSO returns 503 (same env var).
- Sales-processor exits non-zero with clear message: "HQ inventory service token not configured; ask the operator to set HQ_INVENTORY_SERVICE_TOKEN".
- The Phase 21 gate trips first and the menu-cogs fetch never runs.

### Scenario 5 — HQ returns 401 (wrong token)

**Setup:** `HQ_INVENTORY_SERVICE_TOKEN` on the two sides do not match.

**Invocation:** Same as Scenario 1.

**Expected:**
- Sales-processor exits non-zero with clear message: "HQ rejected our token (HTTP 401). Verify HQ_INVENTORY_SERVICE_TOKEN matches on both sides."
- Same failure mode as Phase 21 — the troubleshooting checklist is identical.

---

## 5. Open Assumptions — sales-processor team must confirm

These are assumptions the HQ planner could not verify because the sales-processor repo is not present in this codebase. Each MUST be checked before merging the sales-processor PR.

- [ ] **A1: `HQClient` from Phase 21 exists and is extensible** — Phase 21's `service/external/hq.go` (or wherever sales-processor put it) has a `GetPeriodSummary` method that this phase extends with `GetMenuCOGS`. If Phase 21 put the client elsewhere, follow that convention.
- [ ] **A2: `WeeklySummary` struct exists** — Phase 21 added `COGS` + `COGSInclTax`. This phase adds `MenuCOGS` + `UnallocatedCOGS`. If the struct/method names differ, adjust 3.2 to match.
- [ ] **A3: CLI framework** — confirm sales-processor uses stdlib `flag`, `cobra`, `urfave/cli`, or another. The optional `--breakdown` flag (3.4) follows whatever idiom Phase 21 used for `--force-payroll`.
- [ ] **A4: menu-cogs fetch failure is non-fatal** — Per 3.3, this endpoint returning 500/timeout should NOT block the report. The aggregate from Phase 21 still produces a usable payroll PDF. If sales-processor wants a stricter "no report without menu breakdown" mode, that's a future enhancement.
- [ ] **A5: `units_sold` source of truth is HQ** — D-13: sales-processor must NOT aggregate Toast directly when this field is present. If sales-processor had a parallel Toast aggregation path before Phase 999.2, retire it.
- [ ] **A6: rounding tolerance** — HQ rounds at the Go-decode boundary (not in SQL). Small discrepancies (< $0.10) between menu_items[].ingredient_cost_total totals and Phase 21's COGSInclTax are tolerated. If sales-processor enforces strict equality, it will need to relax the assertion.
- [ ] **A7: null per-unit handling** — `ingredient_cost_per_unit` is the literal JSON `null` when `units_sold == 0`. Sales-processor must decode into a nullable type (Go `*float64`) and render "n/a" / "—". Decoding into a plain `float64` will get `0.0`, which is the wrong value to display.
- [ ] **A8: no per-tenant scoping** — Single-tenant model. The endpoint returns COGS for THE food truck; there is no `tenant_id` query parameter.
- [ ] **A10 (ADDED 2026-07-31 in `1be1a59` — this entry previously said "2026-08-01", which is not the date of any commit that touched it; CORRECTED 2026-08-03): the operating timezone will be `America/New_York` — but THIS ENDPOINT IS NOT AFFECTED BY THE CHANGE.** The operator ruled that HQ's app timezone becomes `America/New_York`; it is `America/Chicago` in production today, and **HQ has built and merged the change but has NOT deployed it — the changeover is the first HQ deploy after that merge, date TBD.**

  🛑 **The claim this entry originally made — that "this endpoint shares its `from` / `to` date semantics with Phase 21's `/period-summary`, so it does not get to hold an independent opinion about which day a receipt belongs to" — is false, and was checked against the code for the first time on 2026-08-03.** All four `/menu-cogs` queries compare against plain SQL `DATE` columns (`purchase_events.event_date`, `daily_menu_sales.business_date`) with **no `AT TIME ZONE` cast anywhere**. This endpoint has no timezone dependency to move. Its output on the day after the changeover deploy is byte-identical to its output on the day before, for the same window.

  **What sales-processor should take from this:** make the Phase 21 change (assumption **A5 of `21-SALES-PROCESSOR-CONTRACT.md` §5** is still authoritative and still requires a coordinated two-repo release) — but do not schedule, gate, or test anything for `/menu-cogs` on account of the timezone. There is nothing here to coordinate.
- [ ] **A9: drift signal is OUT of band** — Per D-15, this endpoint never indicates "your recipes are stale." That signal lives in the HQ Recipes tab banner + a weekly Cliq message. If sales-processor wants to surface "X% unallocated" as a warning in the report, it can compute that from the response itself (`unallocated_cogs / sum(menu_items.ingredient_cost_total + unallocated_cogs)`).

---

## 6. Out of Scope (deferred to future phases)

- **Token rotation.** v1 uses a single long-lived shared secret. Same as Phase 21.
- **Real-time updates.** Sales-processor pulls once per weekly run. No streaming / websocket / push.
- **Per-unit BOM (gram-level recipes).** Explicitly out of scope per ROADMAP. Future phase if rough % stops being good enough.
- **Multi-vendor / multi-size purchases of the same ingredient.** Explicitly out of scope. Assume one canonical `purchase_item` per ingredient.
- **Bulk-buy distortion smoothing.** A single olive-oil purchase covering 6 weeks shows as $X spend in week 1 and $0 in weeks 2-6. Accepted as a known limitation; window-sum basis preserves Phase 21 consistency. Could be addressed with a trailing-N-week smoothing option in a future phase.
- **Configurable drift thresholds.** D-21 locks 10%/20% in HQ code. Endpoint behavior is unaffected; this only matters for the HQ-side scheduler.
- **Sales-processor side of the recipe edit flow.** Sales-processor is read-only. The Recipes tab in HQ is the only edit surface.

---

## 7. HQ-side reference implementation

*(Corrected 2026-08-03: the five `.planning/phases/999.2-…` SUMMARY.md paths this section used to list **no longer exist**. Archived by `875e26c` (2026-06-05), then deleted outright by `34f8c7e` (2026-07-26) along with the rest of the GSD artifacts. A reader following those links since July 2026 got nothing.)*

The HQ-side implementation lives in:

- `backend/internal/recipes/handler.go` — `MenuCogsHandler`, `loadBreakdown`, `loadUnallocatedBreakdown`
- `backend/internal/recipes/types.go` — `MenuCOGSResponse`, `MenuCOGSRow`, `IngredientAlloc`, `UnallocatedBreakdown`
- `backend/internal/recipes/drift.go` — the out-of-band drift check (D-15, D-21)
- `backend/internal/auth/service_token.go` — `ServiceTokenMiddleware` (the 401 / 503 envelope)
- `backend/cmd/server/main.go` — route wiring, shared `HQ_INVENTORY_SERVICE_TOKEN`
- Migrations `0060_menu_items.sql`, `0061_daily_menu_sales.sql`, `0062_recipes.sql`

🛑 **The integration tests in `backend/internal/recipes/menu_cogs_test.go` are NOT proof that the HQ side matches this document, and this page claiming they were is how four wrong field names survived from 2026-06-04 to 2026-08-03.** The nine tests decode into `recipes.MenuCOGSResponse` — the same struct the handler marshals. A JSON key that this document gets wrong is therefore **invisible** to every one of them: the test and the handler agree with each other and both disagree with the page. The suite was green throughout.

**Any contract change requires updating this doc, the integration tests, AND re-running the §0 audit** — diffing published names and expressions against the code, which no test here does for you. If you want a test that *would* have caught this, it has to assert on raw JSON keys, not on a decoded struct. Filed as **B-71**. *(Corrected 2026-08-03, second pass — this cited B-73, which is the broken cross-endpoint invariant, not the test-shape gap.)*

### Smoke test commands

When `HQ_INVENTORY_SERVICE_TOKEN` is set on the server:

```sh
# Summary mode
curl -s -H "Authorization: Bearer $HQ_INVENTORY_SERVICE_TOKEN" \
  "https://hq.yumyums.kitchen/api/v1/inventory/menu-cogs?from=2026-05-25&to=2026-05-31"
# Expected: 200, JSON with from/to/menu_items[]/unallocated_cogs

# Breakdown mode
curl -s -H "Authorization: Bearer $HQ_INVENTORY_SERVICE_TOKEN" \
  "https://hq.yumyums.kitchen/api/v1/inventory/menu-cogs?from=2026-05-25&to=2026-05-31&breakdown=true"
# Expected: 200, JSON with menu_items[].ingredients[] populated and unallocated{total, by_ingredient[]} object

# Missing Bearer → 401
curl -s "https://hq.yumyums.kitchen/api/v1/inventory/menu-cogs?from=2026-05-25&to=2026-05-31"
# Expected: 401, {"error":"unauthorized"}

# Invalid date → 400
curl -s -H "Authorization: Bearer $HQ_INVENTORY_SERVICE_TOKEN" \
  "https://hq.yumyums.kitchen/api/v1/inventory/menu-cogs?from=not-a-date&to=2026-05-31"
# Expected: 400, {"error":"from must be YYYY-MM-DD"}
```

When `HQ_INVENTORY_SERVICE_TOKEN` is UNSET on the server, all the above paths return 503 `{"error":"service_token_not_configured"}` — same fail-closed behavior as Phase 21.

---

## 8. Open questions raised by the 2026-08-03 audit — OPERATOR DECISION REQUIRED

§0 corrected this document to match the code. **On this page, unlike its sibling, that direction is genuinely arguable** — several of the findings are places where the *document* described the more useful behaviour and the code simply never implemented it. Correcting the doc to match the code makes the page honest; it does not make the behaviour right.

**These are the HQ operator's calls. None has been made, and no code has been changed.**

- **Q1 — Should `/menu-cogs` return the top-level `menu` field it has promised since 2026-06-04?** The `menu_items.menu` column exists and is populated; the handler simply never selects it. Adding it is **additive and backward-compatible** — no existing consumer breaks — and it would make the code match what HQ published. This is the cheapest of the three and the only one with no downside identified. *Recommendation: yes, but it is still an external-contract change and therefore the operator's to authorise.*
- **Q2 — Should the `menu_items[].name` key be renamed to match the contract, or the contract left pointing at `menu_item_name`?** Renaming the JSON key **fixes** any sales-processor client that was built from this page (they are currently getting `""`) and **breaks** any client built from the observed wire format. HQ cannot see which exists. `MenuCOGSRow` is used by no other HQ endpoint, so the blast radius is entirely on the sales-processor side. **This one cannot be decided without the counterparty** — it is a question for the notice, not for HQ alone.
- **Q3 — Should a menu item that sold units with no recipe appear in the response?** Today it is silently absent (State C above). The operator therefore has no endpoint-visible signal that a seller is uncosted — the money shows up in `unallocated_cogs` with nothing naming it. Changing the inner join to a left join would surface these rows with `ingredient_cost_total: 0.00` and a null per-unit, which is exactly what this document originally promised. It is a **behaviour change to an endpoint an external system consumes** and would add rows to every response. *Not proposed here.*
- **Q4 — Should the reconciliation invariant between the two endpoints be restored?** Since 2026-06-06 they answer different questions (§0, `:122`). Either `/menu-cogs` grows the category allowlist and pending-row handling that `/period-summary` gained, or the two are formally declared non-reconcilable and sales-processor stops checking. Today it is neither: the invariant is published, broken, and unmonitored.

Nothing in this section has been actioned. Nothing has been sent to the sales-processor maintainer — see `NOTICE-sales-processor-2026-08-03-UNSENT.md`, which is a **draft**.
