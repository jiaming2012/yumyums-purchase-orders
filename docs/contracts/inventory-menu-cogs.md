# Phase 999.2 — sales-processor ↔ HQ HTTP Contract (menu-cogs)

**Status:** Authored 2026-06-04 by the planner. **Hand this document to the sales-processor maintainer.**

This document is the contract the sales-processor repo must implement against to satisfy Phase 999.2's acceptance criteria. The HQ-side of the phase (this repo) is shipped and exercised by integration tests; the sales-processor side is NOT planned here per the developer's decision to keep that work in its own repo.

If sales-processor differs from any assumption below, raise a question against this doc — do NOT silently diverge.

This contract is the sibling of `21-SALES-PROCESSOR-CONTRACT.md`. The two endpoints share the same Bearer-auth pattern, the same 503-if-unset behavior, the same env var, and the same date semantics. The DIFFERENCE: Phase 21's `/period-summary` returns AGGREGATE COGS; Phase 999.2's `/menu-cogs` breaks that aggregate down BY menu item.

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
  - Both dates are interpreted in `America/New_York` (the food-truck operating timezone). The two dates define an inclusive calendar window. For a Monday–Sunday workweek "May 25–31, 2026", send `from=2026-05-25&to=2026-05-31`.
  - 🛑 **CHANGING — was `America/Chicago`. BUILT, NOT DEPLOYED: the zone moves on the first HQ deploy that follows this document's merge, date TBD.** This endpoint shares its date semantics with Phase 21's `/period-summary`, so it moves with it. The authoritative statement and the sequencing requirement live in **assumption A5 of `21-SALES-PROCESSOR-CONTRACT.md` §5** — it is a **coordinated two-repo release**, and until sales-processor ships its matching change the two repos disagree by one hour at each period edge. Read A5 before deploying either side.
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

**Summary mode (default):**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    {
      "menu_item_id": "ae56f5eb-8399-4dbb-8872-70fe16132d16",
      "name": "Smashburger",
      "menu": "Lunch Menu",
      "menu_group": "Sandwiches",
      "menu_subgroup": null,
      "units_sold": 42,
      "ingredient_cost_per_unit": 2.7500,
      "ingredient_cost_total": 115.50
    }
  ],
  "unallocated_cogs": 89.00
}
```

**Breakdown mode (`?breakdown=true`):**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    {
      "menu_item_id": "ae56f5eb-8399-4dbb-8872-70fe16132d16",
      "name": "Smashburger",
      "menu": "Lunch Menu",
      "menu_group": "Sandwiches",
      "menu_subgroup": null,
      "units_sold": 42,
      "ingredient_cost_per_unit": 2.7500,
      "ingredient_cost_total": 115.50,
      "ingredients": [
        { "purchase_item_description": "Ground Beef 80/20", "usage_pct": 50.00, "allocated_cost": 75.00 },
        { "purchase_item_description": "Brioche Buns", "usage_pct": 100.00, "allocated_cost": 40.50 }
      ]
    }
  ],
  "unallocated": {
    "total": 89.00,
    "by_ingredient": [
      { "purchase_item_description": "Olive Oil", "amount": 60.00, "reason": "no recipe" },
      { "purchase_item_description": "Tortillas", "amount": 29.00, "reason": "partial allocation (40%)" }
    ]
  }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `from` | string | Echo of the input `from` (YYYY-MM-DD). |
| `to` | string | Echo of the input `to` (YYYY-MM-DD). |
| `menu_items` | array of objects | One row per menu item with > 0 sales in the window. Empty array `[]` when no sales. |
| `menu_items[].menu_item_id` | string (UUID) | Stable `menu_items.id`. Use this when correlating across reports — not `name` (which can collide across menus per D-09). |
| `menu_items[].name` | string | Display name from Toast. |
| `menu_items[].menu` | string | Top-level Toast menu (e.g. "Lunch Menu"). |
| `menu_items[].menu_group` | string | Toast menu group (e.g. "Sandwiches"). |
| `menu_items[].menu_subgroup` | string or null | Toast subgroup; null when the menu item has no subgroup. |
| `menu_items[].units_sold` | integer | `SUM(daily_menu_sales.units_sold)` over the window. Per D-13, HQ is the truth source — sales-processor must NOT aggregate from Toast directly when this field is present. |
| `menu_items[].ingredient_cost_per_unit` | number (4 decimals) or null | `ingredient_cost_total / units_sold`. **Null when `units_sold == 0`** — JSON literal `null` (not 0). |
| `menu_items[].ingredient_cost_total` | number (2 decimals) | Tax-inclusive ingredient cost for the week. Computed as: for each recipe row linking this menu item to a `purchase_item`, take `usage_pct * window_spend_for_that_purchase_item / SUM(usage_pct across all menu items using that purchase_item)`. The "tax-inclusive" piece is documented in D-11 — tax is pro-rated to each line item by its share of the event subtotal. |
| `menu_items[].ingredients` | array (breakdown mode only) | Per-ingredient breakdown of `ingredient_cost_total`. Empty array `[]` when the menu item has no recipe rows. |
| `unallocated_cogs` | number (summary mode only, 2 decimals) | Dollar residual of window spend not allocated to any menu item. |
| `unallocated` | object (breakdown mode only) | Replaces `unallocated_cogs` in breakdown mode. |
| `unallocated.total` | number | Same dollar residual. |
| `unallocated.by_ingredient` | array | One row per `purchase_item` contributing to the residual. `reason` is either `"no recipe"` (no recipe rows reference the purchase_item) or `"partial allocation (X%)"` (sum of usage_pct < 100). |

**Invariant:** `SUM(menu_items[].ingredient_cost_total) + unallocated_cogs ≈ period-summary cogs_incl_tax` for the same `[from..to]` window. Phase 21's `period-summary` is the aggregate; this endpoint breaks it down. Small rounding discrepancies (< $0.10) are tolerated — Phase 21 rounds at the SQL level, this endpoint rounds at the Go-decode level.

### Response — example states

**State A: fully allocated week**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    { "menu_item_id": "...", "name": "Smashburger", "menu": "Lunch", "menu_group": "Sandwiches", "menu_subgroup": null,
      "units_sold": 42, "ingredient_cost_per_unit": 2.7500, "ingredient_cost_total": 115.50 },
    { "menu_item_id": "...", "name": "Tacos al Pastor", "menu": "Lunch", "menu_group": "Tacos", "menu_subgroup": null,
      "units_sold": 80, "ingredient_cost_per_unit": 1.5000, "ingredient_cost_total": 120.00 }
  ],
  "unallocated_cogs": 0.00
}
```

**State B: week with no recipe coverage at all (everything unallocated)**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [],
  "unallocated_cogs": 235.50
}
```

(`menu_items` is empty when no recipes exist OR when no menu items have sales. Sales-processor must NOT assume non-empty.)

**State C: menu item sold but no recipe linked (units_sold > 0, per-unit is null)**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "menu_items": [
    { "menu_item_id": "...", "name": "Iced Tea", "menu": "Beverages", "menu_group": "Drinks", "menu_subgroup": null,
      "units_sold": 30, "ingredient_cost_per_unit": null, "ingredient_cost_total": 0.00 }
  ],
  "unallocated_cogs": 18.00
}
```

(per-unit is the literal JSON `null`, NOT `0`. Decode into `*float64` / `Optional<Double>` / equivalent; rendering should show "—" or "n/a" rather than "$0.00".)

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
- **Empty behavior:** server logs `WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary AND /api/v1/inventory/menu-cogs will return 503` at startup, BOTH endpoints return 503 on every request (fail-closed).
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

type MenuItemCOGS struct {
    MenuItemID            string              `json:"menu_item_id"`
    Name                  string              `json:"name"`
    Menu                  string              `json:"menu"`
    MenuGroup             string              `json:"menu_group"`
    MenuSubgroup          *string             `json:"menu_subgroup"`         // null when absent
    UnitsSold             int                 `json:"units_sold"`
    IngredientCostPerUnit *float64            `json:"ingredient_cost_per_unit"` // null when units_sold == 0
    IngredientCostTotal   float64             `json:"ingredient_cost_total"`
    Ingredients           []IngredientAlloc   `json:"ingredients,omitempty"` // breakdown mode only
}

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
    MenuCOGS          []MenuItemCOGS    // one entry per menu item with sales in the week
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
  Iced Tea             (30 units)        n/a    (no recipe linked)
  Unallocated                         $89.00
                                   ---------
  Total                              $324.50
```

Layout details are sales-processor's call. The data points the operator needs:
- Per-menu-item: name, units sold, cost total, cost per unit (or "n/a" when null).
- Unallocated total — surfaces incomplete recipes (the food-truck operator's signal to add more recipe rows).
- A footer total that reconciles against `COGSInclTax` (within rounding).

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
- Report contains "COGS by Menu Item" section with one row per menu item that has sales.
- Each row shows units, total, and per-unit (or "n/a").
- Unallocated line is shown when > $0.
- Sum of per-menu costs + unallocated ≈ Phase 21's `COGSInclTax` (within rounding).

### Scenario 2 — Menu item sold but no recipe — null per-unit handled

**Setup:** HQ has at least one menu_item with `units_sold > 0` but NO recipe rows linking to it.

**Invocation:** Same as Scenario 1.

**Expected:**
- That menu item appears in the COGS-by-menu-item table.
- Per-unit column shows "n/a" / "—" (NOT "$0.00").
- Cost-total column shows $0.00.
- The unallocated row absorbs the corresponding ingredient spend.

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
- [ ] **A10 (ADDED): the operating timezone will be `America/New_York`, and changing it is a COORDINATED TWO-REPO RELEASE.** It is `America/Chicago` in production today; **HQ has built and merged the change but has NOT deployed it — the changeover is the first HQ deploy after that merge, date TBD.** This endpoint shares its `from` / `to` date semantics with Phase 21's `/period-summary`, so it does not get to hold an independent opinion about which day a receipt belongs to. **The authoritative statement, the sequencing requirement, and the bounded blast radius are all in assumption A5 of `21-SALES-PROCESSOR-CONTRACT.md` §5** — confirm that one, not this one. This entry exists so a reader who only ever opens the menu-cogs contract still learns that the zone moved.
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

The HQ-side implementation is tracked in:

- `.planning/phases/999.2-per-menu-item-cogs-attribution-via-recipe-bom-mapping/999.2-01-SUMMARY.md` — migrations (0062 recipes, 0063 drift_check_results) + internal/recipes package scaffold
- `.planning/phases/999.2-per-menu-item-cogs-attribution-via-recipe-bom-mapping/999.2-02-SUMMARY.md` — MenuCogsHandler + 9 integration tests (this is THE contract reference)
- `.planning/phases/999.2-per-menu-item-cogs-attribution-via-recipe-bom-mapping/999.2-03-SUMMARY.md` — recipes CRUD handlers (orthogonal to this contract — owner-facing only)
- `.planning/phases/999.2-per-menu-item-cogs-attribution-via-recipe-bom-mapping/999.2-04-SUMMARY.md` — drift scheduler + Cliq alert (out-of-band signal per D-15)
- `.planning/phases/999.2-per-menu-item-cogs-attribution-via-recipe-bom-mapping/999.2-05-SUMMARY.md` — Recipes tab UI (frontend; orthogonal to this contract)

The integration tests in `backend/internal/recipes/menu_cogs_test.go` are the executable proof that the HQ side matches this contract. Any contract change requires updating both this doc AND the integration tests.

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
