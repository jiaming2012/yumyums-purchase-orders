# Phase 21 — sales-processor ↔ HQ HTTP Contract

**Status:** Authored 2026-06-02 by the planner. **Hand this document to the sales-processor maintainer.**

This document is the contract the sales-processor repo must implement against to satisfy Phase 21's acceptance criteria. The HQ-side of the phase (this repo) is planned and executable; the sales-processor side is NOT planned here per the developer's decision to keep that work in its own repo.

If sales-processor differs from any assumption below, raise a question against this doc — do NOT silently diverge.

---

## 0. Drift Audit — 2026-08-03

**Why this section exists.** On 2026-08-03 every line of this document that publishes a code-level claim was diffed against the code at HEAD, expression by expression, rather than read for plausibility. That had never been done. The prose had been *read* many times over fourteen months and had been believed each time; **ten rows were wrong and one input was missing entirely**. Five of the ten are material, and one of those had been wrong since 2026-06-06 in a way that could block a payroll run without explanation.

Line numbers below (`:NN`) are as this document stood **before** this revision — the state a sales-processor reader would have been working from. The corrections are applied inline in the sections that follow.

**Method.** Published expression → the code at HEAD, named by file and symbol. Response shapes were **observed** by marshalling `inventory.PeriodSummary` and printing the JSON, not transcribed by eye. Each drifted row names the commit that drifted it.

| `:NN` | Row / claim | Verdict |
|---|---|---|
| `:16` | Path `GET /api/v1/inventory/period-summary` | **CONFIRMED** — `main.go` route registration |
| `:18` | `Authorization: Bearer <token>` | **CONFIRMED** — `auth/service_token.go`, prefix `"Bearer "` |
| `:25`–`:26` | `from` / `to` required, `YYYY-MM-DD`, inclusive | **CONFIRMED** — `time.Parse("2006-01-02")` on both |
| `:27` | "Both are interpreted in `America/New_York`" | **DRIFTED — over-broad.** The zone applies to **one** expression: the `created_at` fallback inside `pendingPeriodDateExpr`. `purchase_events.event_date` is a plain `DATE` and is compared without any cast, so the COGS aggregate has **no timezone dependency at all**. Corrected inline. |
| `:28` | "CHANGING — was `America/Chicago`, built not deployed" | **CONFIRMED** — `users.DefaultTimezone = "America/New_York"` in source; migration `0072_app_timezone_new_york.sql` present and unapplied in prod |
| `:45`–`:58` | Response JSON example | **DRIFTED — three fields missing.** `by_vendor` (`518a395`, 2026-06-05), `tracked_bank_tx_ids` (`f730485`, 2026-06-06), `completeness.pending_review_details` (`1c260f0`, 2026-06-06). All additive. |
| `:62` | `from` echo | **CONFIRMED** |
| `:63` | `to` echo | **CONFIRMED** |
| `:64` | `cogs_excl_tax` = `SUM(quantity*price)` over the period | **DRIFTED ×2.** `a726029` (2026-06-05) added the `mercury_category` allowlist filter. `d41faef` (2026-06-06) added a second summand — `SUM(ABS(bank_total))` over unconfirmed pending rows. Neither published. COGS went **up**. |
| `:65` | `cogs_incl_tax` = `cogs_excl_tax + SUM(tax)` | **DRIFTED — by inheritance.** The stated arithmetic still holds exactly; its base does not. Same two commits. |
| `:66` | `purchase_event_count` = `COUNT(*)` of `purchase_events` | **DRIFTED.** Now allowlist-filtered **and** includes unconfirmed pending rows. Same two commits. A 4-confirmed / 3-pending week reports `7`. |
| `:67` | `completeness.ready` = true iff both lists empty | **CONFIRMED byte-accurate** — `len(pendingIDs) == 0 && len(unlinkedIDs) == 0` |
| `:68` | `completeness.pending_review_ids` | **DRIFTED — and the previous revision's correction was itself incomplete.** `cf959bd` (2026-06-06) added the `COALESCE`, widening the gate; that half was stated. `d41faef` (2026-06-06) added `mercury_category = ANY(...)` **and** `reason = 'no_attachment_on_bank_tx'`, narrowing it much further, and **that half was never stated by anyone until now.** See the corrected row. |
| `:69` | `completeness.unlinked_line_item_ids` | **CONFIRMED byte-accurate** — clause for clause, including the `purchase_events` join and the `IS NULL` test |
| `:124` | COGS still returned when `ready=false` | **CONFIRMED** — the response struct is populated unconditionally |
| `:130`–`:132` | Three 400 envelopes | **CONFIRMED** — all three strings byte-identical |
| `:133` | 401 `unauthorized` | **CONFIRMED** — byte-identical |
| `:134` | 500 `internal_error` | **CONFIRMED** — byte-identical |
| `:135` | 503 `service_token_not_configured` | **CONFIRMED** — byte-identical |
| `:149` | Loaded via `os.Getenv` in `main.go` | **CONFIRMED** |
| `:150` | Startup log text | **DRIFTED — cosmetic.** The real line names **both** `/period-summary` and `/menu-cogs`. Harmless; corrected for accuracy. |
| — | **`HQ_COGS_CATEGORY_ALLOWLIST`** | 🛑 **MISSING ROW — never published in either document.** Not a drifted row; an absent one. It gates four response fields. Added to §2 by this revision. `a726029`. |
| `:164` | `crypto/subtle.ConstantTimeCompare` | **CONFIRMED** |
| `:316`–`:318` | A1–A3 (sales-processor repo layout, `WeeklySummary`, CLI framework) | **NOT VERIFIABLE FROM HQ** — they describe a repo not present here. Unchanged. |
| `:319` | A4 static bearer token, no HMAC / rotation | **CONFIRMED** — the middleware is a constant-time compare against one static string |
| `:320`–`:331` | A5 timezone assumption | **CONFIRMED** — matches `users/db.go`, `handler.go` changeover comment, and migration `0072` |
| `:332` | A6 trusted network | **OPERATIONAL** — not a code claim |
| `:333` | A7 unlinked-vs-pending semantics | **DRIFTED.** First sentence accurate. Second sentence wrong since `d41faef` — most pending rows now appear in *neither* completeness list while still moving COGS. Corrected inline. |
| `:334` | A8 HTTP 200 with `ready:false` | **CONFIRMED** — no non-2xx path for business state |
| `:335` | A9 discarded rows treated as resolved | **CONFIRMED** — `discarded_at IS NULL` present in every gate query |
| `:352`–`:354` | §7 reference-implementation paths | **DRIFTED — stale.** All three `.planning/phases/21-…` paths were deleted by `34f8c7e` (2026-07-26). Replaced with live paths. |
| `:356` | "integration tests are the executable proof" | **FALSE as written.** The tests assert against the same Go structs the handler marshals, so they cannot detect a doc-vs-code mismatch — which is exactly why these drifts survived a green suite for fourteen months. Corrected inline. |

**Rows audited: 32** — 31 line-anchored entries spanning `:16`–`:356`, plus one missing-input finding that has no line number because it was never written down. **19 CONFIRMED byte-accurate · 2 not verifiable from this repo (they describe the sales-processor repo) · 10 DRIFTED or false · 1 missing.**

**The single most consequential finding:** `:68` drifted **twice on the same day**, in **opposite directions**, and the previous revision of this document corrected only one of them. Since 2026-06-06 the completeness gate is wider than published in one dimension (late-discovered receipts now block) and much narrower in another (only COGS-category receipts with no attachment at all block). **The net effect on any given period cannot be derived from the previously published contract.**

**No HQ code was changed by this revision.** The defect was documentary: in every case the code was self-consistent and tested, and the document described an earlier version of it. Where the audit raised a question about whether the *code* is right — see §8.

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
  - The two dates define an inclusive calendar window. For a Monday–Sunday workweek "May 25–31, 2026", send `from=2026-05-25&to=2026-05-31`.
  - **Timezone — narrowed 2026-08-03.** This document used to say "both are interpreted in `America/New_York`" without qualification. That is over-broad. **Most of this endpoint has no timezone dependency at all:** `purchase_events.event_date` is a plain SQL `DATE` and both `cogs_excl_tax` and `unlinked_line_item_ids` compare against it with no cast. The operating zone (`America/New_York`, the food-truck's zone) enters at exactly **one** point — the `created_at` fallback inside the pending-purchases date expression, which applies only to rows where the receipt parser extracted no `event_date`. See `completeness.pending_review_ids` below and assumption **A5**.
  - 🛑 **CHANGING — was `America/Chicago`. HQ has BUILT this change; HQ has NOT DEPLOYED it.** The zone moves **on the first HQ deploy that follows this document's merge — date TBD, and it has not happened yet.** Do not read this page as a description of what HQ is running today. See assumption **A5** in §5 for the coordinated-release requirement. **Until sales-processor ships its matching change, the two repos will disagree by one hour at each period edge**, on `pending_purchases` rows that carry no extracted `event_date`.
- **Auth header:** `Authorization: Bearer <token>`. The token is an opaque string. Sales-processor reads it from the env var `HQ_INVENTORY_SERVICE_TOKEN`. The exact value must be agreed out-of-band with the HQ operator and stored as a secret in sales-processor's runtime environment.

### Base URL — confirm with operator

The HQ base URL is the same domain used by the PWA. The expected value:

```
HQ_BASE_URL=https://hq.yumyums.kitchen
```

**[ACTION REQUIRED — user/operator confirmation]:** Confirm that `hq.yumyums.kitchen` is the correct hostname for sales-processor to reach (it routes through Cloudflare Tunnel per project memory). If sales-processor runs on the same Windows box as HQ, a LAN/Tailscale address may be preferred to avoid the tunnel round-trip.

### Response — success (200 OK)

`Content-Type: application/json`

🛑 **This example was wrong from 2026-06-05 until 2026-08-03.** It omitted three fields HQ had been returning for fourteen months. The shape below is the **observed** shape — produced by marshalling `inventory.PeriodSummary` at HEAD, not transcribed from the handler. Added fields are marked.

```json
{
  "from": "2026-05-25",
  "to": "2026-05-31",
  "cogs_excl_tax": 1234.56,
  "cogs_incl_tax": 1334.56,
  "purchase_event_count": 7,
  "by_vendor": [
    {
      "vendor_id": "v1",
      "vendor_name": "Restaurant Depot",
      "total_excl_tax": 900,
      "total_incl_tax": 970,
      "trip_count": 3
    }
  ],
  "tracked_bank_tx_ids": ["mercury-tx-1"],
  "completeness": {
    "ready": false,
    "pending_review_ids": ["7c2e9a1b-..."],
    "pending_review_details": [
      {
        "id": "7c2e9a1b-...",
        "bank_tx_id": "mercury-tx-1",
        "vendor": "",
        "event_date": "2026-05-29",
        "bank_total": -84.21,
        "reason": "no_attachment_on_bank_tx"
      }
    ],
    "unlinked_line_item_ids": []
  }
}
```

All three additions are **additive** — a decoder written against the old shape still works; it just ignores data it was never told about. None of them was ever published until this revision.

| Field | Type | Notes |
|-------|------|-------|
| `from` | string | Echo of the input `from` (YYYY-MM-DD). |
| `to` | string | Echo of the input `to` (YYYY-MM-DD). |
| `cogs_excl_tax` | number (float, 2 decimal places) | **CORRECTED 2026-08-03 — this row was wrong since 2026-06-06.** Two summands, not one: (a) `SUM(quantity * price)` over `purchase_line_items` joined to `purchase_events` where `event_date BETWEEN from AND to` **AND `purchase_events.mercury_category = ANY(HQ_COGS_CATEGORY_ALLOWLIST)`**, plus (b) `SUM(ABS(bank_total))` over **unconfirmed `pending_purchases`** in the period that are allowlisted and whose `reason != 'no_attachment_on_bank_tx'`. Summand (a)'s allowlist filter arrived in `a726029` (2026-06-05); summand (b) arrived in `d41faef` (2026-06-06). Neither was published. **Direction of the change: COGS went UP** — receipts still sitting in the review queue now contribute their full bank amount. Rounded server-side in SQL. Zero if no events. |
| `cogs_incl_tax` | number (float, 2 decimal places) | `cogs_excl_tax + SUM(tax)` over the **allowlisted confirmed** `purchase_events` only. The arithmetic relation to `cogs_excl_tax` is unchanged and still holds; what changed is the base — it inherits both corrections to `cogs_excl_tax` above. Note the asymmetry introduced by `d41faef`: an unconfirmed pending row contributes its `bank_total` to **both** figures and contributes no separate tax, so for that row the "incl" and "excl" numbers are the same. Zero if no events. |
| `purchase_event_count` | integer | **CORRECTED 2026-08-03 — this row was wrong since 2026-06-06.** Not `COUNT(*)` of `purchase_events`. It is `COUNT(*)` of **allowlisted** `purchase_events` in the range **plus** `COUNT(*)` of the unconfirmed non-blocking `pending_purchases` folded into `cogs_excl_tax` summand (b). A period with 4 confirmed receipts and 3 in the review queue reports `7`. Allowlist filter from `a726029`; pending addend from `d41faef`. Zero if none. |
| `by_vendor` | array of objects | **NEVER PUBLISHED UNTIL 2026-08-03.** Added by `518a395` (2026-06-05). Per-vendor COGS breakdown: `{vendor_id, vendor_name, total_excl_tax, total_incl_tax, trip_count}`. `trip_count` counts distinct `purchase_events`; tax is allocated per event, not per line item. Unconfirmed pending rows are folded in — matched to a vendor by `LOWER(TRIM(vendors.name)) = LOWER(TRIM(pending_purchases.vendor))`, and when no vendor matches the row gets its own entry with `vendor_id: ""` and `vendor_name` of the raw receipt text (or `"(unknown vendor)"`). Ordered by `total_excl_tax DESC, vendor_name ASC`. |
| `tracked_bank_tx_ids` | array of strings | **NEVER PUBLISHED UNTIL 2026-08-03.** Added by `f730485` (2026-06-06). Every Mercury `bank_tx_id` HQ has touched for the period across all states — confirmed in `purchase_events`, or pending/confirmed/discarded in `pending_purchases`. **No `mercury_category` filter** — this list is for completeness detection, not COGS. Intended use: diff it against Mercury's own transaction list for the same period to find "Mercury has it, HQ has not ingested it yet" gaps. |
| `completeness.ready` | boolean | `true` iff BOTH `pending_review_ids` AND `unlinked_line_item_ids` are empty. |
| `completeness.pending_review_ids` | array of strings (UUIDs) | `pending_purchases.id` rows where `COALESCE(event_date, (created_at AT TIME ZONE 'America/New_York')::date) BETWEEN from AND to` AND `confirmed_at IS NULL` AND `discarded_at IS NULL` AND `mercury_category = ANY(HQ_COGS_CATEGORY_ALLOWLIST)` AND `reason = 'no_attachment_on_bank_tx'`. Always present, empty array `[]` when none. 🛑 **THREE separate corrections to this row — read all three.** **(1) The zone.** It was `America/Chicago`; it becomes `America/New_York` **on the first HQ deploy that follows this document's merge — date TBD, not yet deployed** (assumption A5). **(2) The `COALESCE` — a behaviour change of 2026-06-06 that HQ never published, and which WIDENED the gate.** Phase 21 published `(created_at AT TIME ZONE 'America/Chicago')::date` with no `COALESCE`, and that was accurate to the code as it then stood. Quick task `260606-0gh`, commit `cf959bd` (2026-06-06), added `COALESCE(event_date, …)` to the live query and did not touch this contract. The difference is not cosmetic: under the **published** expression a late-discovered receipt — a May 29 purchase ingested June 2 — was filtered by its `created_at`, fell outside the May window, and did **not** block May payroll; under the shipped code its extracted `event_date` puts it inside the May window and it **does**. **(3) ADDED 2026-08-03 — the two filter clauses `mercury_category = ANY(...)` and `reason = 'no_attachment_on_bank_tx'`, a SECOND unpublished behaviour change of the same day that NARROWED the gate, in the opposite direction to (2).** Commit `d41faef` (2026-06-06, quick task `260606-jvs`) restricted the blocking set to pending rows that are COGS-category **and have no receipt attached at all**. Every other pending row — food purchases whose receipt attached but failed to parse, non-food purchases, rows with a NULL category — **stopped blocking payroll on that date** and instead began contributing to `cogs_excl_tax` (see that row). ⚠️ **The previous revision of this document stated only (2). A reader who acted on it would have concluded the gate was strictly wider than Phase 21's, which is not true — since 2026-06-06 it is wider in one dimension and much narrower in another, and the net effect on any given period cannot be predicted from this document alone.** Note how (2) bounds (1): an extracted `event_date` wins the `COALESCE`, so **only rows with no extracted `event_date` are exposed to the zone at all**. |
| `completeness.pending_review_details` | array of objects | **NEVER PUBLISHED UNTIL 2026-08-03.** Added by `1c260f0` (2026-06-06). One object per entry in `pending_review_ids`, same order: `{id, bank_tx_id, vendor, event_date, bank_total, reason}`. `vendor` is `""` when the receipt parser could not extract one. `event_date` is the `COALESCE` expression above rendered as `YYYY-MM-DD`, so it is the row's *effective* period date, not necessarily its extracted one. `reason` is omitted when NULL. Exists so a service-token caller can render a meaningful blocked-payroll message without a second round trip to the cookie-auth-only `/purchases/pending` endpoint. |
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
- **Empty behavior:** server logs `HQ_INVENTORY_SERVICE_TOKEN not set, /api/v1/inventory/period-summary AND /api/v1/inventory/menu-cogs will return 503` at startup, both endpoints return 503 on every request (fail-closed). *(Corrected 2026-08-03: the log line names both endpoints — `/menu-cogs` was added to the same middleware group in Phase 999.2 and shares this token.)*
- **Format:** opaque string, no whitespace, no encoding requirements. Recommend 32+ random bytes hex- or base64-encoded.
- **Storage:** managed as an env var in the Cloudflare Tunnel / docker-compose / systemd unit running the HQ backend on the Windows box. NOT committed to the repo.

#### `HQ_COGS_CATEGORY_ALLOWLIST` — 🛑 ADDED 2026-08-03, never previously published

```
HQ_COGS_CATEGORY_ALLOWLIST=COGS      # comma-separated; this is the default
```

This env var did not exist when Phase 21 was published and **has never appeared in either contract document**, yet it is an input to four of the response fields — `cogs_excl_tax`, `cogs_incl_tax`, `purchase_event_count`, and `completeness.pending_review_ids`. Introduced by `a726029` (2026-06-05).

- **Where loaded:** `backend/cmd/server/main.go`, `envOrDefault("HQ_COGS_CATEGORY_ALLOWLIST", "COGS")`, split on `,` with each element trimmed.
- **Meaning:** which Mercury `categoryData.name` values count toward COGS. A `purchase_event` or `pending_purchase` whose `mercury_category` is not in this list — **including NULL** — stays in the database for bookkeeping but does not roll up into any food-cost number this endpoint returns, and cannot block payroll.
- **Why sales-processor needs to know:** the aggregate it receives is not "all purchasing" — it is "purchasing HQ's operator has categorised as COGS in Mercury". A miscategorised transaction is silently invisible to this endpoint. It will, however, still appear in `tracked_bank_tx_ids`, which is deliberately unfiltered — **that is the field to diff against Mercury if you want to detect this class of miss.**
- **Operator note:** changing this value changes historical figures for every period, retroactively, with no migration and no signal to sales-processor.

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
- [ ] **A5 (AMENDED — HQ has BUILT this, HQ has NOT DEPLOYED it): `America/New_York` will be the operating timezone. It is `America/Chicago` in production today, and this is a COORDINATED TWO-REPO RELEASE — sales-processor must make the matching change.**

  🛑 **This is the one assumption on this page that is not merely "confirm and proceed." It requires a change on the sales-processor side.**

  🛑 **Do not read this entry as "HQ is already on New York."** At the time of writing, HQ has written the code and merged it. **Nothing has been deployed.** The changeover happens **on the first HQ deploy that follows this merge, and that deploy has not been scheduled — date TBD.** If sales-processor ships its side on the assumption that HQ has already moved, it creates the exact one-hour disagreement this entry exists to prevent.

  - **What is changing and why.** The operator ruled that the app's timezone is `America/New_York` (HQ ledger T-26 decision 83, re-affirmed against this contract as T-28 decision 93). The framing was: *a payroll week and a food-cost week must describe the same seven days.* HQ had been running two zones at once — `users.DefaultTimezone` was already New York while the money paths were hardcoded Chicago — so the previous A5 was describing only half of HQ.
  - **What HQ has BUILT (merged, not deployed).** Every hardcoded zone in HQ now reads the single constant `users.DefaultTimezone` (`America/New_York`) **in the merged source**. The old anchor site this assumption named, `backend/internal/purchasing/repurchase.go:71`, is one of them. Migration `0072_app_timezone_new_york.sql` re-points the `cutoff_config.timezone` and `repurchase_reset_config.timezone` column defaults **and** updates the already-written rows — **the migration runs on that deploy, not before it.**
  - **What sales-processor must do.** Interpret `from` / `to` for `/period-summary` and `/menu-cogs` in `America/New_York`, and compute the weekly payroll window on the same zone. If sales-processor derives its Monday–Sunday window from a hardcoded `America/Chicago`, change it.
  - **Sequencing — read this before deploying either side.** Until BOTH repos have shipped, one of them is wrong — **and as of this writing NEITHER has**. The disagreement is **one hour at each period edge**, and it is bounded: it can only move a `pending_purchases` row that has **no extracted `event_date`** (see `completeness.pending_review_ids` above) across a period boundary. It cannot move a confirmed `purchase_event`, whose `event_date` is a plain `DATE`. **Agree the HQ deploy date with the HQ operator before scheduling the sales-processor deploy**; the changeover date is that deploy's date, and it is not yet known.
  - **Fix forward only.** Weekly COGS and payroll figures produced before the changeover deploy were already acted on and are **NOT** restated. A reader comparing two weeks either side of that deploy will find one boundary that moved by an hour, exactly once.
  - **If the food truck moves to a different TZ again, both repos must update — together, and this assumption must be amended again.**
- [ ] **A6: sales-processor and HQ communicate over a trusted network** (Cloudflare Tunnel, Tailscale, or LAN) — HTTPS + bearer token without mTLS or IP allowlist is sufficient.
- [ ] **A7 (CORRECTED 2026-08-03): "unlinked line item" semantics** — `purchase_line_items.purchase_item_id IS NULL` for confirmed events. That half is still byte-accurate. **The second sentence was wrong from 2026-06-06:** it read *"Items inside `pending_purchases.items` JSONB are reported via `pending_review_ids`, not `unlinked_line_item_ids` (no double-counting)."* Since `d41faef` **most pending rows are reported by neither list.** Only pending rows that are COGS-category **and** carry `reason = 'no_attachment_on_bank_tx'` reach `pending_review_ids`; the rest are absorbed silently into `cogs_excl_tax`. The no-double-counting guarantee survives — nothing is counted twice — but the implied *completeness* guarantee does not: **a pending row can now affect the COGS number while appearing in no completeness list at all.**
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

*(Corrected 2026-08-03: the three `.planning/phases/21-…` PLAN.md paths this section used to list **no longer exist**. They were archived by `875e26c` (2026-06-05) and then deleted outright by `34f8c7e` (2026-07-26), which removed the GSD artifacts. A reader following those links since July 2026 got nothing.)*

The HQ-side implementation lives in:

- `backend/internal/inventory/handler.go` — `PeriodSummaryHandler`, all four queries
- `backend/internal/inventory/types.go` — `PeriodSummary`, `VendorCOGS`, `CompletenessBlock`, `PendingReviewDetail`
- `backend/internal/auth/service_token.go` — `ServiceTokenMiddleware` (the 401 / 503 envelope)
- `backend/cmd/server/main.go` — route wiring, `HQ_INVENTORY_SERVICE_TOKEN`, `HQ_COGS_CATEGORY_ALLOWLIST`

🛑 **The integration tests in `backend/internal/inventory/period_summary_test.go` are NOT proof that the HQ side matches this document.** They assert against the Go structs, which are the same structs the handler marshals — so a claim in this document that disagrees with the struct is invisible to them. That is precisely how the drifts catalogued in §0 survived fourteen months of a green test suite. **Any contract change requires updating this doc, the integration tests, AND re-running the §0 audit** — diffing published expressions against the code, which no test does for you.

---

## 8. Open questions raised by the 2026-08-03 audit — OPERATOR DECISION REQUIRED

The audit in §0 corrected this document to match the code. In three places it is not obvious that the *code* is the thing that should have won. **These are the HQ operator's calls, not the auditor's, and none of them has been made.** They are recorded here so the sales-processor maintainer knows the questions are open, and so a future reader does not mistake "the doc now matches the code" for "the behaviour was ratified."

- **Q1 — Should a pending row be able to move COGS without appearing in any completeness list?** Since `d41faef` a food purchase whose receipt attached but failed to parse contributes its full `bank_total` to `cogs_excl_tax` and blocks nothing. That is defensible (the money did leave the account) and it is also how a period can read `ready: true` while containing an unreviewed figure. **No change is proposed here.**
- **Q2 — Do the `ready:false` runs between 2026-06-06 and today need reconciling?** Explicitly out of scope for the audit and reserved to the operator. The audit establishes only that undisclosed `ready:false` results were *possible* from 2026-06-06; it makes no claim about whether any occurred, and HQ retains no log that would settle it.
- **Q3 — Should `HQ_COGS_CATEGORY_ALLOWLIST` changes be versioned?** Editing it silently restates every historical period this endpoint can report, with no migration and no signal to sales-processor. Today it is an unversioned env var.

Nothing in this section has been actioned. Nothing has been sent to the sales-processor maintainer — see `NOTICE-sales-processor-2026-08-03-UNSENT.md`, which is a **draft**.
