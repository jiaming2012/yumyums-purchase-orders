# Phase 21 — sales-processor ↔ HQ HTTP Contract

**Status:** Authored 2026-06-02 by the planner. **Hand this document to the sales-processor maintainer.**

This document is the contract the sales-processor repo must implement against to satisfy Phase 21's acceptance criteria. The HQ-side of the phase (this repo) is planned and executable; the sales-processor side is NOT planned here per the developer's decision to keep that work in its own repo.

If sales-processor differs from any assumption below, raise a question against this doc — do NOT silently diverge.

---

## 1. The Endpoint

### Request

```
GET /api/v1/inventory/period-summary?from=YYYY-MM-DD&to=YYYY-MM-DD HTTP/1.1
Host: hq.yumyums.kitchen
Authorization: Bearer <HQ_INVENTORY_SERVICE_TOKEN>
Accept: application/json
```

- **Method:** `GET` (idempotent, no body).
- **Path:** exactly `/api/v1/inventory/period-summary`.
- **Query params** (both required):
  - `from` — start date, format `YYYY-MM-DD`, inclusive.
  - `to` — end date, format `YYYY-MM-DD`, inclusive.
  - Both are interpreted in `America/Chicago` (the food-truck operating timezone). The two dates define an inclusive calendar window. For a Monday–Sunday workweek "May 25–31, 2026", send `from=2026-05-25&to=2026-05-31`.
- **Auth header:** `Authorization: Bearer <token>`. The token is an opaque string. Sales-processor reads it from the env var `HQ_INVENTORY_SERVICE_TOKEN`. The exact value must be agreed out-of-band with the HQ operator and stored as a secret in sales-processor's runtime environment.

### Base URL — confirm with operator

The HQ base URL is the same domain used by the PWA. The expected value:

```
HQ_BASE_URL=https://hq.yumyums.kitchen
```

**[ACTION REQUIRED — user/operator confirmation]:** Confirm that `hq.yumyums.kitchen` is the correct hostname for sales-processor to reach (it routes through Cloudflare Tunnel per project memory). If sales-processor runs on the same Windows box as HQ, a LAN/Tailscale address may be preferred to avoid the tunnel round-trip.

### Response — success (200 OK)

`Content-Type: application/json`

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "cogs_excl_tax": 1234.56,
  "cogs_incl_tax": 1334.56,
  "purchase_event_count": 7,
  "completeness": {
    "ready": false,
    "pending_review_ids": ["7c2e9a1b-...", "9f0a2b5c-..."],
    "unlinked_line_item_ids": ["3d8b1c4e-..."]
  }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `from` | string | Echo of the input `from` (YYYY-MM-DD). |
| `to` | string | Echo of the input `to` (YYYY-MM-DD). |
| `cogs_excl_tax` | number (float, 2 decimal places) | `SUM(quantity * price)` over `purchase_line_items` joined to `purchase_events` where `event_date BETWEEN from AND to`. Rounded server-side in SQL. Zero if no events. |
| `cogs_incl_tax` | number (float, 2 decimal places) | `cogs_excl_tax + SUM(tax)` over the same `purchase_events`. Zero if no events. |
| `purchase_event_count` | integer | `COUNT(*)` of `purchase_events` in the range. Zero if none. |
| `completeness.ready` | boolean | `true` iff BOTH `pending_review_ids` AND `unlinked_line_item_ids` are empty. |
| `completeness.pending_review_ids` | array of strings (UUIDs) | `pending_purchases.id` rows where `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN from AND to` AND `confirmed_at IS NULL` AND `discarded_at IS NULL`. Always present, empty array `[]` when none. |
| `completeness.unlinked_line_item_ids` | array of strings (UUIDs) | `purchase_line_items.id` rows where the parent `purchase_events.event_date BETWEEN from AND to` AND `purchase_line_items.purchase_item_id IS NULL`. Always present, empty array `[]` when none. |

### Response — example states

**State A: ready=true (fully ingested period)**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "cogs_excl_tax": 1234.56,
  "cogs_incl_tax": 1334.56,
  "purchase_event_count": 7,
  "completeness": {
    "ready": true,
    "pending_review_ids": [],
    "unlinked_line_item_ids": []
  }
}
```

**State B: ready=false with pending review queue**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "cogs_excl_tax": 800.00,
  "cogs_incl_tax": 860.00,
  "purchase_event_count": 4,
  "completeness": {
    "ready": false,
    "pending_review_ids": ["7c2e9a1b-9c1a-4f2e-bd0c-1234567890ab"],
    "unlinked_line_item_ids": []
  }
}
```

**State C: ready=false with unlinked line items**

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "cogs_excl_tax": 1100.00,
  "cogs_incl_tax": 1180.00,
  "purchase_event_count": 6,
  "completeness": {
    "ready": false,
    "pending_review_ids": [],
    "unlinked_line_item_ids": ["3d8b1c4e-2a5f-4e8b-9c0d-abcdef012345"]
  }
}
```

Note: even when `ready=false`, `cogs_excl_tax` / `cogs_incl_tax` are still returned. This is intentional — `--force-payroll` callers can render COGS anyway.

### Response — error states

| HTTP | Body | When |
|------|------|------|
| 400 | `{"error":"from must be YYYY-MM-DD"}` | `from` query param malformed or missing |
| 400 | `{"error":"to must be YYYY-MM-DD"}` | `to` query param malformed or missing |
| 400 | `{"error":"from must be <= to"}` | `from > to` lexicographic |
| 401 | `{"error":"unauthorized"}` | `Authorization` header missing, malformed (no `Bearer ` prefix), or token mismatch |
| 500 | `{"error":"internal_error"}` | DB error on the HQ side; details only in HQ server logs |
| 503 | `{"error":"service_token_not_configured"}` | HQ has `HQ_INVENTORY_SERVICE_TOKEN` unset — operator must configure |

Sales-processor should distinguish 503 (config error on HQ side — surface to operator, do not retry blindly) from 500 (transient — may retry once).

---

## 2. Env Var Contract

### HQ side (this repo — already planned)

```
HQ_INVENTORY_SERVICE_TOKEN=<opaque-string>
```

- **Where loaded:** `backend/cmd/server/main.go` via `os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")`.
- **Empty behavior:** server logs `WARNING: HQ_INVENTORY_SERVICE_TOKEN not set — /api/v1/inventory/period-summary will return 503` at startup, endpoint returns 503 on every request (fail-closed).
- **Format:** opaque string, no whitespace, no encoding requirements. Recommend 32+ random bytes hex- or base64-encoded.
- **Storage:** managed as an env var in the Cloudflare Tunnel / docker-compose / systemd unit running the HQ backend on the Windows box. NOT committed to the repo.

### sales-processor side (separate repo — sales-processor team implements)

The sales-processor must read TWO env vars:

```
HQ_BASE_URL=https://hq.yumyums.kitchen
HQ_INVENTORY_SERVICE_TOKEN=<must match HQ's value byte-for-byte>
```

- **`HQ_BASE_URL`:** the protocol + host where HQ is reachable. No trailing slash. Sales-processor's HTTPClient appends `/api/v1/inventory/period-summary?...` to this.
- **`HQ_INVENTORY_SERVICE_TOKEN`:** the SAME secret as on the HQ side. Comparison on HQ uses `crypto/subtle.ConstantTimeCompare` (timing-safe).

**[ACTION REQUIRED — operator]:** Generate the secret once, configure it on BOTH sides, restart BOTH services. Whenever rotated, both sides must be updated atomically (or sales-processor will get 401 until it sees the new value).

---

## 3. Sales-Processor Implementation Contract

### 3.1 HTTPClient (`service/external/hq.go`)

```go
// PSEUDOCODE — adapt to sales-processor's actual project layout and HTTP conventions.
package external

type HQClient interface {
    GetPeriodSummary(ctx context.Context, from, to time.Time) (*PeriodSummary, error)
}

type PeriodSummary struct {
    From               string             `json:"from"`
    To                 string             `json:"to"`
    COGSExclTax        float64            `json:"cogs_excl_tax"`
    COGSInclTax        float64            `json:"cogs_incl_tax"`
    PurchaseEventCount int                `json:"purchase_event_count"`
    Completeness       CompletenessBlock  `json:"completeness"`
}

type CompletenessBlock struct {
    Ready                bool     `json:"ready"`
    PendingReviewIDs     []string `json:"pending_review_ids"`
    UnlinkedLineItemIDs  []string `json:"unlinked_line_item_ids"`
}

// Implementation:
//   - Use net/http.Client with explicit Timeout (e.g. 10s). Do NOT use http.DefaultClient (no timeout).
//   - Format dates as "2006-01-02".
//   - Set Authorization: Bearer <token>.
//   - On 200, decode JSON into PeriodSummary.
//   - On 401, return a typed error so the gate layer can show a clear "HQ rejected our token" message.
//   - On 503, return a typed error so the gate can distinguish HQ-misconfigured from transient.
//   - On 500, treat as transient — caller may retry once.
//   - On 400, treat as a programmer error (we sent bad input) — surface and fail.
```

### 3.2 WeeklySummary fields + Show() rendering

The sales-processor's existing `WeeklySummary` struct must gain two new fields:

```go
type WeeklySummary struct {
    // ... existing fields like Net Sales, Tax Collected, Tips, etc. ...
    COGS        float64  // tax-EXCLUDED COGS for the week
    COGSInclTax float64  // tax-INCLUDED COGS for the week
}
```

The `Show()` method renders these AFTER the existing Net Sales line. Exact label text is sales-processor's choice but should match the existing report style. Example:

```
Net Sales:           $5,432.10
COGS (excl tax):     $1,234.56
COGS (incl tax):     $1,334.56
Gross Margin:        $4,197.54        // (Net Sales - COGS excl tax)
```

(Gross Margin is optional — listed as a likely use case, not a required field.)

### 3.3 Payroll Gate Logic

```go
// PSEUDOCODE — adapt to the actual CLI framework and entry point.
func runWeekly(ctx context.Context, from, to time.Time, forcePayroll bool) error {
    summary, err := hqClient.GetPeriodSummary(ctx, from, to)
    if err != nil {
        return fmt.Errorf("fetch HQ period summary: %w", err)
    }

    weekly.COGS = summary.COGSExclTax
    weekly.COGSInclTax = summary.COGSInclTax

    if !summary.Completeness.Ready && !forcePayroll {
        // Hard-fail: print blocker IDs and exit non-zero. Do NOT generate PDF/CSV/transfers.
        log.Println("HQ receipts not fully ingested for this period. Pass --force-payroll to override.")
        if len(summary.Completeness.PendingReviewIDs) > 0 {
            log.Printf("  Pending review (%d): %v", len(summary.Completeness.PendingReviewIDs), summary.Completeness.PendingReviewIDs)
        }
        if len(summary.Completeness.UnlinkedLineItemIDs) > 0 {
            log.Printf("  Unlinked line items (%d): %v", len(summary.Completeness.UnlinkedLineItemIDs), summary.Completeness.UnlinkedLineItemIDs)
        }
        return errors.New("receipts not ready for payroll")
    }

    // Proceed with the existing flow. If --force-payroll was used, the report
    // proceeds but the COGS lines may be under-counted; that's the operator's
    // explicit choice.
    return writePayrollArtifacts(ctx, weekly)
}
```

### 3.4 `--force-payroll` CLI flag

- **Name:** `--force-payroll`
- **Default:** `false`
- **Type:** boolean flag (no value).
- **Effect:** when `true`, the gate at 3.3 is bypassed AND a warning is logged. PDF/CSV/Mercury transfers proceed using whatever COGS data was returned (which may be incomplete).

Sales-processor's CLI framework is unknown to the HQ planner. The flag MUST be visible in `--help` output so operators discover it.

---

## 4. Acceptance Scenarios (from the roadmap)

Sales-processor must demonstrate each scenario passes. The HQ side already provides integration tests for the endpoint behavior; the scenarios below are end-to-end through sales-processor.

### Scenario 1 — Week with unconfirmed pending purchases

**Setup:** HQ has at least one `pending_purchases` row in the week with `confirmed_at IS NULL` and `discarded_at IS NULL`.

**Invocation:** `./sales-processor weekly --from 2026-05-25 --to 2026-05-31` (no force flag).

**Expected:**
- Exit code: non-zero
- stderr/log: lists the pending UUIDs
- No PDF / OnPay CSV / Mercury transfers created

### Scenario 2 — Same week + `--force-payroll`

**Setup:** Same as Scenario 1.

**Invocation:** `./sales-processor weekly --from 2026-05-25 --to 2026-05-31 --force-payroll`

**Expected:**
- Exit code: zero
- PDF generated with COGS lines populated from the HQ response
- Warning logged that force-payroll was used

### Scenario 3 — Fully-ingested week

**Setup:** HQ has all receipts confirmed, no pending, all line items linked.

**Invocation:** `./sales-processor weekly --from 2026-05-25 --to 2026-05-31`

**Expected:**
- Exit code: zero
- PDF generated, COGS lines populated

---

## 5. Open Assumptions — sales-processor team must confirm

These are assumptions the HQ planner could not verify because the sales-processor repo is not present in this codebase. Each MUST be checked before merging the sales-processor PR.

- [ ] **A1: file `service/external/hq.go` doesn't exist yet** — the sales-processor will CREATE it. If sales-processor already has a different convention for external HTTP clients (e.g. `internal/clients/`), use that instead.
- [ ] **A2: `WeeklySummary` struct exists with a `Show()` method** — if the struct/method names differ, adjust 3.2 to match.
- [ ] **A3: CLI framework** — confirm sales-processor uses stdlib `flag`, `cobra`, `urfave/cli`, or another, and add `--force-payroll` per that framework's idiom.
- [ ] **A4: static shared-secret bearer token is acceptable** for v1 (no HMAC, no timestamp binding, no rotation). If the operator wants HMAC+timestamp, the HQ middleware needs a redesign (out of scope for Phase 21).
- [ ] **A5: `America/Chicago` is the correct operating timezone** — this matches `backend/internal/purchasing/repurchase.go:71` in HQ. If the food truck moves to a different TZ, both repos must update.
- [ ] **A6: sales-processor and HQ communicate over a trusted network** (Cloudflare Tunnel, Tailscale, or LAN) — HTTPS + bearer token without mTLS or IP allowlist is sufficient.
- [ ] **A7: "unlinked line item" semantics** — `purchase_line_items.purchase_item_id IS NULL` for confirmed events. Items inside `pending_purchases.items` JSONB are reported via `pending_review_ids`, not `unlinked_line_item_ids` (no double-counting).
- [ ] **A8: HTTP 200 with `ready:false` is the right shape** — gate logic lives on the sales-processor side. HQ does NOT return non-2xx for "not ready" (that would conflate transport errors with business state).
- [ ] **A9: Discarded `pending_purchases` (`discarded_at IS NOT NULL`) are treated as resolved** — they do NOT block `ready`. Confirmed by roadmap constraint and integration-tested in HQ.

---

## 6. Out of Scope (deferred to future phases)

- **Token rotation.** v1 uses a single long-lived shared secret. Rotation is a manual op (update both env vars + restart both services). A future phase MAY add a token-rotation endpoint or move to HMAC-signed requests with timestamp binding.
- **Per-menu-item COGS attribution.** Phase 21 returns aggregate COGS only. Phase 999.2 (backlog) will add a `/menu-cogs` endpoint that breaks COGS down by menu item using a recipe/BOM table.
- **Real-time updates.** sales-processor pulls once per weekly run. There is no streaming / websocket / push notification when receipts complete.
- **Multi-tenant.** Single-tenant model. The endpoint returns COGS for THE food truck; there is no `tenant_id` query parameter.

---

## 7. HQ-side reference implementation

The HQ-side implementation is tracked in:

- `.planning/phases/21-cogs-in-sales-processor-report-receipt-completeness-gate-bef/21-01-PLAN.md` — types + handler
- `.planning/phases/21-cogs-in-sales-processor-report-receipt-completeness-gate-bef/21-02-PLAN.md` — service-token middleware + unit tests
- `.planning/phases/21-cogs-in-sales-processor-report-receipt-completeness-gate-bef/21-03-PLAN.md` — main.go wiring + integration tests + CLAUDE.md docs

The integration tests in `backend/internal/inventory/period_summary_test.go` are the executable proof that the HQ side matches this contract. Any contract change requires updating both this doc AND the integration tests.
