# Phase 12: Inventory + Photos + Tile Permissions - Research

**Researched:** 2026-04-18
**Domain:** Go backend (inventory persistence, receipt ingestion pipeline, presigned uploads), DO Spaces (S3-compatible), Claude Haiku AI parsing, vanilla JS frontend (big-bang API swap, tile filtering)
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Photo Storage**
- D-01: DO Spaces for all photo/file storage — S3-compatible, $5/mo for 250GB, already on Digital Ocean.
- D-02: Type-prefixed bucket paths: `checklists/{submission_id}/{field_id}.jpg`, `receipts/{event_id}/original.pdf`, `receipts/{event_id}/parsed.json`.
- D-03: Presigned PUT URLs from Go backend — frontend uploads directly to Spaces. No file proxying through the server.

**Receipt Ingestion Pipeline**
- D-04: Background Go worker (cron or Temporal) pulls transactions from Mercury banking API. NOT an in-app upload flow.
- D-05: Claude Haiku for receipt parsing — extracts vendor, line items (description, quantity, unit, price), summary (subtotal, tax, total). Replaces Gemini from the baserow project.
- D-06: Validation checks item calculated total vs bank transaction amount. Mismatches go to a PendingPurchases review queue.
- D-07: In-app review queue in inventory.html — pre-filled form showing parsed receipt data. User corrects errors, confirms to save. Nothing saves without human approval.
- D-08: Subtotal/total mismatch warning (ported from baserow-revenue-api's ValidateReceiptData pattern) — warns user if line item sum doesn't match receipt subtotal.
- D-09: Auto-creates vendors and purchase items if they don't exist (fuzzy match existing via DerivePurchaseItem pattern from baserow).
- D-10: Mercury is first provider. Architecture should support adding Xero and other providers later.

**Inventory Schema + API**
- D-11: Normalized tables: vendors, item_groups, purchase_items, purchase_events, purchase_line_items, tags (with junction table). Matches Phase 10 patterns.
- D-12: Big-bang mock→API swap for inventory.html — replace all MOCK_ arrays with API calls. Delete all mock data.
- D-13: History tab wired to real purchase events from Postgres. Stock and Trends tabs compute from purchase data only — sales data integration deferred.

**Checklist Photo Wiring**
- D-14: Replace blob URLs with presigned upload immediately after camera capture. Request presigned URL → upload blob to Spaces → store Spaces URL in field response. Existing thumbnail/preview UI unchanged.

**Inventory Seed Data**
- D-15: YAML fixtures following the baserow-revenue-api purchase_item_groups.yaml pattern. Go seed function loads on startup. Vendors, item groups with tags, and purchase items.

**Tile Permissions**
- D-16: index.html fetches GET /api/v1/me/apps on load. Tiles not in the response are hidden completely (removed from grid).
- D-17: Cache last-known /me/apps response in localStorage. Show cached tiles immediately on load, refresh in background. Works offline with last-known permissions.

### Claude's Discretion
- Inventory table schema details (exact columns, indexes, constraints)
- API endpoint naming (RPC-style matching Phase 10/11 patterns)
- Mercury API integration details (auth, pagination, date range handling)
- Receipt parsing prompt design for Claude Haiku
- DO Spaces bucket name, region, CORS policy
- Background worker scheduling (cron interval or Temporal workflow)
- Review queue UI layout in inventory.html
- E2E test approach for the pipeline (mock Mercury API responses)

### Deferred Ideas (OUT OF SCOPE)
- Xero integration as additional receipt provider — future phase
- POS/sales data integration for true food cost calculations — future milestone (COST-01, COST-02, COST-03)
- Real-time stock counting / barcode scanning — out of scope
- Push notifications for pending review items — future phase

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INVT-01 | Vendors, purchase events, and line items persisted to Postgres (replacing mock data) | Schema design section, goose migration patterns, seed YAML fixture pattern from baserow-revenue-api |
| INVT-02 | inventory.html fetches purchase data from API for History, Stock, Trends, and Cost tabs | Big-bang mock→API swap pattern from Phase 10, RPC handler pattern from workflow package |
| INVT-03 | Receipt ingestion pipeline — upload receipt image, OCR, map to purchase items, human review | Full baserow-revenue-api port documented: Mercury → Claude Haiku → validate → PendingPurchases queue |
| PHOT-01 | Photo upload via presigned URLs (DO Spaces) | AWS SDK v2 presign pattern, DO Spaces CORS requirements, existing photo-modal UI in workflows.html |
| PHOT-02 | Photos stored and retrievable for checklist evidence and corrective action documentation | Presigned GET URL pattern, Spaces URL stored in submission_responses.value as JSONB string |
| TILE-01 | index.html launcher grid filtered by GET /api/v1/me/apps — users only see tiles for apps they have permission to access | /me/apps endpoint already implemented (me/handler.go), index.html checkAuth() extension pattern |

</phase_requirements>

---

## Summary

Phase 12 is a large backend expansion with three loosely-coupled concerns: (1) inventory persistence replacing all mock data in inventory.html with real Postgres tables and API endpoints, (2) a background receipt ingestion pipeline ported from the existing baserow-revenue-api Go project, and (3) wiring photo capture to DO Spaces presigned uploads and filtering the HQ launcher by user permissions.

The receipt pipeline is the most complex piece. The baserow-revenue-api/src/main.go is the complete reference implementation — it already contains all the algorithms needed: `FetchReceipts` (Mercury HTTP client), `ParseReceipt` (AI multipart call with JSON extraction), `ValidateReceiptData` (two-layer numeric validation against bank amount + line item sum), `DerivePurchaseItem` (Jaro-Winkler fuzzy matching at 0.85 threshold), and the `PendingPurchases` review queue pattern. The port replaces Baserow API calls with pgxpool queries, and replaces `google.golang.org/genai` (Gemini) with `github.com/anthropics/anthropic-sdk-go` (Claude Haiku). The validation logic is pure math — port verbatim.

The photo upload flow is a two-step presign-then-upload pattern: backend generates a short-lived PUT presigned URL via the AWS SDK v2 (DO Spaces is fully S3-compatible with a custom endpoint), and the browser fetches the blob from the capture modal and PUTs it directly to the URL. The existing `photo-modal` / `photo-confirm-btn` / `photo-thumb` UI in workflows.html is unchanged — only the confirm handler needs rewiring to call the presign endpoint first. The tile permission filter extends the already-working `checkAuth()` function in index.html to also call `/me/apps` (which is already implemented in `me/handler.go`) and remove non-matching tiles from the DOM.

**Primary recommendation:** Implement in 4 waves: (1) DB migrations + inventory CRUD API + YAML seed, (2) inventory.html big-bang swap, (3) DO Spaces presigned upload wiring + photo field, (4) receipt pipeline background worker + review queue UI. Each wave is independently testable.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/anthropics/anthropic-sdk-go | v1.37.0 | Claude Haiku API calls for receipt parsing | Official Go SDK; verified installable from module proxy; replaces Gemini per D-05 |
| github.com/aws/aws-sdk-go-v2/service/s3 | v1.99.1 | Presigned PUT/GET URL generation for DO Spaces | DO Spaces is S3-compatible; AWS SDK v2 is the standard way to generate presigned URLs; verified installable |
| github.com/aws/aws-sdk-go-v2/config | (bundled with above) | AWS credential config with custom endpoint | Required to point SDK at DO Spaces endpoint |
| github.com/antzucaro/matchr | (baserow project version) | Jaro-Winkler fuzzy matching for DerivePurchaseItem | Already used in baserow-revenue-api; pure Go, no external dep |
| gopkg.in/yaml.v3 | v3.0.1 | YAML fixture loading for seed data | Already in go.mod; used by existing config package |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| time.AfterFunc / ticker | stdlib | Background worker scheduling | Sufficient for a polling cron interval; Temporal deferred |
| net/http | stdlib | Mercury API HTTP client | Same pattern as baserow-revenue-api/services/mercury.go |
| encoding/json | stdlib | JSON parse/encode for receipt data | Same as all existing handlers |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| AWS SDK v2 for presigned URLs | minio-go SDK | minio-go is simpler but less maintained; AWS SDK v2 is the canonical S3 interface |
| Claude Haiku | gemini-2.5-flash-lite | Decision locked (D-05); Claude Haiku has structured JSON output via tool_use which is more reliable than regex-based JSON extraction |
| time.Ticker for background worker | Temporal workflow | Temporal adds operational complexity; for a single cron job on a low-volume app, stdlib ticker is correct |

**Installation (new deps only):**
```bash
cd backend
go get github.com/anthropics/anthropic-sdk-go@v1.37.0
go get github.com/aws/aws-sdk-go-v2/service/s3@v1.99.1
go get github.com/aws/aws-sdk-go-v2/config@latest
go get github.com/antzucaro/matchr@latest
go mod tidy
```

---

## Architecture Patterns

### Recommended Project Structure (new files only)
```
backend/
├── internal/
│   ├── inventory/
│   │   ├── handler.go           # RPC-style HTTP handlers (listVendors, getPurchaseEvents, etc.)
│   │   ├── service.go           # Business logic (fuzzy match, validation, seed)
│   │   └── types.go             # Input/output structs, domain types
│   ├── receipt/
│   │   ├── worker.go            # Background worker: ticker + run loop
│   │   ├── mercury.go           # FetchReceipts (ported from baserow-revenue-api)
│   │   ├── parser.go            # ParseReceipt using Claude Haiku (replaces gemini.go)
│   │   ├── validate.go          # ValidateReceiptData (port verbatim from services.go)
│   │   └── fuzzy.go             # DerivePurchaseItem (port from fuzzy.go)
│   └── photos/
│       └── spaces.go            # GeneratePresignedPutURL, GeneratePresignedGetURL
├── internal/db/migrations/
│   ├── 0024_inventory.sql       # vendors, item_groups, tags, purchase_items, purchase_events, purchase_line_items
│   └── 0025_pending_purchases.sql # pending_purchases review queue table
└── config/
    └── fixtures/
        └── purchase_item_groups.yaml  # Seed data (ported from baserow-revenue-api/fixtures/)
```

### Pattern 1: RPC-Style Inventory Handlers (matches workflow/ pattern)

**What:** Handler functions accept `*pgxpool.Pool`, return `http.HandlerFunc`, use `auth.UserFromContext`, write JSON.
**When to use:** All inventory read/write endpoints.

```go
// Source: backend/internal/workflow/handler.go (existing pattern)
func ListPurchaseEventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := auth.UserFromContext(r.Context())
        if user == nil {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }
        // vendor filter from query param
        vendorID := r.URL.Query().Get("vendor_id")
        events, err := listPurchaseEvents(r.Context(), pool, vendorID)
        if err != nil {
            log.Printf("ListPurchaseEventsHandler: %v", err)
            http.Error(w, `{"error":"internal_error"}`, http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(events)
    }
}
```

### Pattern 2: Goose Migration (matches 0001-0023 pattern)

**What:** Each migration is a numbered `.sql` file with `-- +goose Up` / `-- +goose Down` markers and `BEGIN`/`COMMIT` wrappers.

```sql
-- Source: backend/internal/db/migrations/0021_ob_signoffs.sql (existing pattern)
-- 0024_inventory.sql
-- +goose Up
BEGIN;

CREATE TABLE vendors (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
  id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL
);

CREATE TABLE item_groups (
  id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name     TEXT NOT NULL,
  par_days INTEGER
);

CREATE TABLE item_group_tags (
  group_id UUID NOT NULL REFERENCES item_groups(id) ON DELETE CASCADE,
  tag_id   UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, tag_id)
);

CREATE TABLE purchase_items (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  description TEXT UNIQUE NOT NULL,
  group_id    UUID REFERENCES item_groups(id)
);

CREATE TABLE purchase_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id     UUID NOT NULL REFERENCES vendors(id),
  bank_tx_id    TEXT UNIQUE NOT NULL,
  event_date    DATE NOT NULL,
  tax           NUMERIC(10,2) NOT NULL DEFAULT 0,
  total         NUMERIC(10,2) NOT NULL,
  receipt_url   TEXT,             -- Spaces URL for original receipt
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE purchase_line_items (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purchase_event_id UUID NOT NULL REFERENCES purchase_events(id) ON DELETE CASCADE,
  purchase_item_id  UUID REFERENCES purchase_items(id),
  description       TEXT NOT NULL,    -- raw name from receipt, before fuzzy match
  quantity          INTEGER NOT NULL,
  price             NUMERIC(10,4) NOT NULL,
  is_case           BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX purchase_events_vendor_idx ON purchase_events(vendor_id);
CREATE INDEX purchase_events_date_idx ON purchase_events(event_date DESC);
CREATE INDEX purchase_line_items_event_idx ON purchase_line_items(purchase_event_id);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS purchase_line_items;
DROP TABLE IF EXISTS purchase_events;
DROP TABLE IF EXISTS purchase_items;
DROP TABLE IF EXISTS item_group_tags;
DROP TABLE IF EXISTS item_groups;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS vendors;
COMMIT;
```

### Pattern 3: Presigned PUT URL (AWS SDK v2 + DO Spaces)

**What:** Backend generates a presigned PUT URL; browser uploads the blob directly.
**When to use:** Any file upload — checklist photos, receipt originals.

```go
// Source: AWS SDK v2 documentation + DO Spaces S3-compatible endpoint
// backend/internal/photos/spaces.go
import (
    "context"
    "time"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewSpacesPresigner(accessKey, secretKey, endpoint, region string) (*s3.PresignClient, error) {
    cfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion(region),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
        config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
            func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{URL: endpoint}, nil
            },
        )),
    )
    if err != nil {
        return nil, err
    }
    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true  // DO Spaces requires path-style addressing
    })
    return s3.NewPresignClient(client), nil
}

func GeneratePresignedPutURL(ctx context.Context, presigner *s3.PresignClient, bucket, key string, ttl time.Duration) (string, error) {
    req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    }, s3.WithPresignExpires(ttl))
    if err != nil {
        return "", err
    }
    return req.URL, nil
}
```

**CRITICAL — DO Spaces CORS policy must allow browser PUTs:**
```json
[
  {
    "AllowedOrigins": ["https://hq.yumyums.com", "http://localhost:8080"],
    "AllowedMethods": ["GET", "PUT"],
    "AllowedHeaders": ["*"],
    "MaxAgeSeconds": 3000
  }
]
```
Set this via the DO Spaces dashboard → bucket → Settings → CORS before attempting browser uploads.

### Pattern 4: Claude Haiku Receipt Parsing (replaces Gemini)

**What:** Download receipt file, send to Claude Haiku with a structured JSON prompt.
**When to use:** For each new Mercury transaction with an attachment URL.

```go
// Source: Anthropic Go SDK + baserow-revenue-api/services/gemini.go port
import (
    anthropic "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

func ParseReceiptWithClaude(ctx context.Context, apiKey string, receiptURL string) ([]ReceiptItem, ReceiptSummary, error) {
    client := anthropic.NewClient(option.WithAPIKey(apiKey))

    // Download file bytes (same as existing gemini.go pattern)
    fileBytes, contentType, err := downloadReceiptFile(receiptURL)
    if err != nil {
        return nil, ReceiptSummary{}, fmt.Errorf("ParseReceiptWithClaude: %w", err)
    }

    // Build content with file + instructions
    msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaude_Haiku_4_5),
        MaxTokens: anthropic.F(int64(2048)),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.NewUserMessage(
                anthropic.NewImageBlockParam(anthropic.ImageBlockParamSourceBase64{
                    MediaType: anthropic.ImageMediaType(contentType),
                    Data:      anthropic.String(base64.StdEncoding.EncodeToString(fileBytes)),
                }),
                anthropic.NewTextBlockParam(`Parse this receipt. Return ONLY a JSON object with:
{
  "items": [{"name": string, "quantity": int, "price": float, "is_case": bool}],
  "summary": {"vendor": string, "total_units": int, "total_cases": int, "tax": float, "total": float}
}`),
            ),
        }),
    })
    if err != nil {
        return nil, ReceiptSummary{}, fmt.Errorf("ParseReceiptWithClaude: API call failed: %w", err)
    }

    // Extract JSON from response text (same parseJSONBody logic as gemini.go)
    return parseJSONBody(msg.Content[0].Text)
}
```

**Note:** For PDF receipts, use `anthropic.NewDocumentBlockParam` instead of `NewImageBlockParam`. Claude Haiku supports PDF documents natively.

### Pattern 5: DerivePurchaseItem Fuzzy Match (port verbatim)

**What:** Exact-match first, then Jaro-Winkler at 0.85 threshold, then create new with title-case.
**When to use:** Matching raw receipt item names against existing `purchase_items.description`.

The port is direct — replace `models.BaserowData` interface with a simple `map[string]string` (name → UUID) since Postgres returns structured data, not Baserow rows.

```go
// Source: baserow-revenue-api/src/services/fuzzy.go
// Port: replace BaserowData interface with simple string map
func DerivePurchaseItemID(rawName string, existingMap map[string]string) (id string, name string, isNew bool) {
    lower := strings.ToLower(rawName)
    for existingName, existingID := range existingMap {
        if strings.ToLower(existingName) == lower {
            return existingID, existingName, false
        }
    }
    var bestMatch string
    var bestID string
    var highest float64
    for existingName, existingID := range existingMap {
        score := matchr.JaroWinkler(lower, strings.ToLower(existingName), true)
        if score > highest {
            highest = score
            bestMatch = existingName
            bestID = existingID
        }
    }
    if highest >= 0.85 {
        return bestID, bestMatch, false
    }
    titleName := cases.Title(language.English).String(rawName)
    return "", titleName, true
}
```

### Pattern 6: ValidateReceiptData (port verbatim)

From `baserow-revenue-api/src/services/services.go`. Two validation layers:
1. `summary.Total == -mercuryTx.Amount` — receipt total matches bank debit (note negative sign: bank debit amounts are negative in Mercury API)
2. `|sum(item.price * item.qty) - (summary.Total - summary.Tax)| <= 0.01` — line items sum to subtotal
3. `sum(item.qty) == summary.TotalUnits + summary.TotalCases` — item count matches

Failed validation → insert into `pending_purchases` table for human review (D-06).

### Pattern 7: Background Worker (stdlib ticker)

```go
// Source: Go stdlib — time.NewTicker
// backend/internal/receipt/worker.go
func StartWorker(ctx context.Context, pool *pgxpool.Pool, mercuryKey, anthropicKey, spacesConfig SpacesConfig, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                if err := runIngestCycle(ctx, pool, mercuryKey, anthropicKey, spacesConfig); err != nil {
                    log.Printf("receipt worker: ingest cycle error: %v", err)
                }
            case <-ctx.Done():
                return
            }
        }
    }()
    log.Printf("receipt worker: started (interval=%v)", interval)
}
```

**Scheduling recommendation:** 6-hour interval (4x/day). Mercury transactions settle within hours; this is sufficient for a food truck. Set via `RECEIPT_WORKER_INTERVAL` env var (default `6h`).

### Pattern 8: Tile Permission Filtering (index.html)

**What:** Extend existing `checkAuth()` in index.html to also fetch `/me/apps` and filter tiles.
**When to use:** On every page load of index.html.

```javascript
// Source: index.html checkAuth() existing pattern + docs/user-management-api.md D-16/D-17
const APPS_CACHE_KEY = 'hq_apps_cache';

async function filterTilesByPermissions(apps) {
  const allowedSlugs = new Set(apps.map(a => a.slug));
  // Map tile href → slug
  const tileMap = {
    'purchasing.html': 'purchasing',
    'workflows.html': 'operations',
    'users.html': 'users',
    'inventory.html': 'inventory',
    'onboarding.html': 'onboarding',
  };
  document.querySelectorAll('.grid .tile').forEach(tile => {
    const href = tile.getAttribute('href');
    const slug = tileMap[href];
    if (slug && !allowedSlugs.has(slug)) {
      tile.remove();  // remove from DOM entirely, grid reflows
    }
  });
}

async function checkAuth() {
  // ... existing /api/v1/me fetch unchanged ...

  // Tile filtering — show cached immediately, refresh in background
  const cached = localStorage.getItem(APPS_CACHE_KEY);
  if (cached) {
    try { filterTilesByPermissions(JSON.parse(cached)); } catch(e) {}
  }
  try {
    const res = await fetch('/api/v1/me/apps');
    if (res.ok) {
      const apps = await res.json();
      localStorage.setItem(APPS_CACHE_KEY, JSON.stringify(apps));
      filterTilesByPermissions(apps);
    }
  } catch(e) {
    // offline — cached version already applied above
  }
}
```

### Anti-Patterns to Avoid
- **Storing blob: URLs in submission_responses.value** — blob URLs are session-scoped and die on page reload. After presigned upload, replace with the Spaces `https://` URL. The existing code in workflows.html correctly uses `blob:` as a temporary preview only; the confirm flow must swap it out.
- **Proxying file uploads through Go server** — violates D-03 and wastes memory. Always presign → frontend PUT directly to Spaces.
- **Running receipt parsing synchronously in an HTTP handler** — Claude Haiku API calls take 2-10 seconds. Always run in a goroutine or background worker, never blocking an HTTP response.
- **Querying all purchase events without pagination** — History tab will grow indefinitely. Add `LIMIT 50 OFFSET $n` from the start; the API should accept `page` or `before` cursor parameter.
- **Using latest tag on any SDK** — pin exact versions in go.mod to prevent unexpected breaking changes.
- **Seeding YAML fixtures on every request** — seed functions must be idempotent (use `INSERT ... ON CONFLICT DO NOTHING`) and run only on server startup, not per-request.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Presigned URL generation | Custom HMAC signing against DO API | `github.com/aws/aws-sdk-go-v2/service/s3` PresignClient | Signature V4 has edge cases (canonical headers, content hash); SDK handles all of it |
| Fuzzy string matching | Custom edit-distance or n-gram | `github.com/antzucaro/matchr` JaroWinkler | Same library already proven in baserow-revenue-api; Jaro-Winkler is specifically better than Levenshtein for short strings like item names |
| Claude API client | Direct HTTP to api.anthropic.com | `github.com/anthropics/anthropic-sdk-go` | Handles auth, retries, streaming; SDK is official |
| JSON extraction from AI response | Regex on raw text | Structured JSON prompt + `encoding/json` unmarshal | The baserow project used regex to strip markdown code fences — Claude Haiku with a strict JSON-only prompt instruction is more reliable |
| Goose migration embedding | Custom SQL loader | `github.com/pressly/goose/v3` (already in go.mod) | Already wired in `db.Migrate(pool)`; just add new .sql files |

**Key insight:** The baserow-revenue-api is 90% of the implementation already. Port the algorithms; replace the storage layer.

---

## Common Pitfalls

### Pitfall 1: Mercury API Amount Sign
**What goes wrong:** `ValidateReceiptData` fails for all transactions with "receipt total X does not match transaction amount Y".
**Why it happens:** Mercury returns debit amounts as negative floats. `summary.Total` is positive. The check must be `summary.Total == -mercuryTx.Amount` (note the negation), exactly as in baserow-revenue-api's `services.go` line 12.
**How to avoid:** Port the negation directly. Add a unit test with a sample Mercury transaction where `amount = -248.40` and `total = 248.40`.
**Warning signs:** All new transactions immediately land in pending_purchases.

### Pitfall 2: DO Spaces Presigned PUT Fails with 403
**What goes wrong:** Browser PUT to presigned URL returns 403 Forbidden or CORS error.
**Why it happens:** Either (a) the bucket CORS policy hasn't been configured to allow `PUT` from the app origin, or (b) the AWS SDK is using virtual-hosted addressing (`https://bucket.region.digitaloceanspaces.com`) while DO Spaces requires path-style (`https://region.digitaloceanspaces.com/bucket`).
**How to avoid:** Set `UsePathStyle: true` in s3.Options (documented in Pattern 3 above). Set CORS policy on the bucket before first test.
**Warning signs:** Presigned URL generation succeeds but PUT request fails in browser console.

### Pitfall 3: Claude Haiku JSON Extraction
**What goes wrong:** `parseJSONBody` fails because the AI response wraps JSON in markdown code fences or includes explanatory text.
**Why it happens:** Without explicit instruction, Claude sometimes responds with `Here is the JSON: \`\`\`json ... \`\`\``.
**How to avoid:** Use `"Return ONLY a JSON object, no markdown, no explanation"` in the prompt. Additionally, keep the baserow project's regex fallback (```` ```json ... ``` ````) as a secondary parser for resilience.
**Warning signs:** `ParseReceipt` returns parse errors on valid-looking receipts.

### Pitfall 4: Blob URL in autoSaveField
**What goes wrong:** Photo field value saved to Postgres as `blob:http://localhost/...` — the URL is session-scoped and dies on page reload.
**Why it happens:** The existing `handlePhotoCaptureClick` stub in workflows.html currently does nothing (D-26 placeholder). When the presigned flow is wired in, the confirm handler must replace the blob with the Spaces URL before calling `autoSaveField`.
**How to avoid:** The presigned upload sequence must be: (1) user confirms photo → (2) request presigned PUT URL from backend → (3) PUT blob to Spaces → (4) only on 200 response, call `autoSaveField(fieldId, spacesURL)`. Never save the `blob:` URL.
**Warning signs:** Photos show as broken images on checklist reopen.

### Pitfall 5: Tile Filter Race on Cold Load
**What goes wrong:** All tiles briefly flash as "Soon" on first load, then snap to the correct filtered set — jarring UX.
**Why it happens:** If no cache exists and the DOM initially renders all tiles as active, the filter runs after a 200ms+ fetch round-trip.
**How to avoid:** On first load with no cache, show all tiles as `.tile.soon` placeholder style until the `/me/apps` response arrives, then build the real grid from the response. The filter function should re-render the grid, not just remove nodes.
**Warning signs:** Users see ghost tiles for a moment after install.

### Pitfall 6: Inventory Seed Running on Every Request
**What goes wrong:** Vendors/tags/item_groups are duplicated in DB or throw unique constraint errors.
**Why it happens:** Seed function called outside startup, or `INSERT ... ON CONFLICT` omitted.
**How to avoid:** Seed function must use `INSERT INTO vendors (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`. Call only from `main.go` once at startup, same pattern as `SeedHQApps` and `SeedOnboardingTemplates`.

---

## Code Examples

### Mercury Transaction Fetch (port from baserow-revenue-api)
```go
// Source: baserow-revenue-api/src/services/mercury.go
// GET https://api.mercury.com/api/v1/transactions?start=2026-01-01&end=2026-01-07
// Authorization: Bearer {MERCURY_API_KEY}
// Response: {"transactions": [{id, amount, bankDescription, status, kind, attachments, note, createdAt}]}
// Note: status filter = "sent"; kind filter = credit/debit card transactions only
// Note: 1000-record limit — pagination needed at scale (not required for food truck volume)
```

### DO Spaces Endpoint
```
Region:   nyc3 (or sgp1 — wherever the existing DO account is)
Endpoint: https://nyc3.digitaloceanspaces.com
Bucket:   yumyums-hq (suggested name — confirm with DO account)
Key pattern for checklist photos: checklists/{submission_id}/{field_id}.jpg
Key pattern for receipts:         receipts/{bank_tx_id}/original.pdf
```

### Pending Purchase DB Schema
```sql
-- 0025_pending_purchases.sql
-- +goose Up
BEGIN;
CREATE TABLE pending_purchases (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bank_tx_id        TEXT NOT NULL,
  bank_total        NUMERIC(10,2) NOT NULL,
  vendor            TEXT NOT NULL,
  event_date        DATE,
  tax               NUMERIC(10,2),
  total             NUMERIC(10,2),
  total_units       INTEGER,
  total_cases       INTEGER,
  receipt_url       TEXT,
  reason            TEXT,            -- validation failure message
  item_name         TEXT,            -- NULL for header row
  item_quantity     INTEGER,
  item_price        NUMERIC(10,4),
  item_is_case      BOOLEAN,
  confirmed_at      TIMESTAMPTZ,
  confirmed_by      UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX pending_purchases_bank_tx_idx ON pending_purchases(bank_tx_id);
COMMIT;
-- +goose Down
BEGIN;
DROP TABLE IF EXISTS pending_purchases;
COMMIT;
```

### Frontend api() wrapper (existing pattern — use as-is)
```javascript
// Source: workflows.html — already handles 401 redirect and 204 short-circuit
async function api(path, opts) {
  const res = await fetch(path, opts);
  if (res.status === 401) { window.location.href = '/login.html'; throw new Error('unauthorized'); }
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw Object.assign(new Error(data.error || 'api_error'), {status: res.status, data});
  return data;
}
// Use for all inventory.html API calls — identical to workflows.html pattern
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Gemini 2.5 Flash Lite (genai SDK) | Claude Haiku (anthropic-sdk-go) | Phase 12 (D-05) | Swap AI client in receipt/parser.go; algorithm identical |
| Baserow API for storage | Postgres (pgxpool) | Phase 12 | Replace all `baserowClient.CreateRow` / `ListRows` calls with pgxpool queries |
| blob: URLs for photo fields | DO Spaces https:// URLs | Phase 12 (D-14) | presign endpoint + browser PUT — workflows.html photo confirm handler rewired |
| Static tile grid | Permission-filtered tiles from /me/apps | Phase 12 (D-16) | index.html checkAuth() extended; /me/apps already implemented |

**Deprecated/outdated:**
- `MOCK_VENDORS`, `MOCK_PURCHASE_EVENTS`, `MOCK_PURCHASES`, `MOCK_ITEM_GROUPS`, `MOCK_PURCHASE_ITEMS`, `MOCK_TAGS`, `MOCK_MENU_ITEMS`, `MOCK_SALES` — all 8 mock arrays in inventory.html to be deleted in the big-bang swap (D-12).
- `handlePhotoCaptureClick` stub in workflows.html (line 1453-1456) — currently shows placeholder; replace entirely with presigned flow.

---

## Open Questions

1. **DO Spaces bucket name and region**
   - What we know: Decision D-01 says DO Spaces, $5/mo plan; existing project is on Digital Ocean
   - What's unclear: Exact bucket name and which DO region the account is in (nyc3 vs sgp1 vs sfo3)
   - Recommendation: Create bucket `yumyums-hq` in nyc3 (closest to Hetzner box in Germany — low latency). Planner should add a Wave 0 task to confirm bucket exists and set CORS policy before implementation.

2. **Mercury API date range for initial sync**
   - What we know: The baserow-revenue-api uses a rolling 2-week window (previous Monday to last Sunday)
   - What's unclear: For the initial run, how far back should the pipeline fetch? All-time or last 90 days?
   - Recommendation: Accept `MERCURY_LOOKBACK_DAYS` env var (default 90 for initial run, 14 for ongoing). First run fetches 90 days; subsequent runs use last 14 days or a watermark timestamp stored in DB.

3. **Anthropic API key management**
   - What we know: Existing `AI_API_KEY` env var pattern in baserow-revenue-api; project uses `os.Getenv` for all secrets
   - What's unclear: Should the key be named `ANTHROPIC_API_KEY` (SDK default) or match the existing `AI_API_KEY` convention?
   - Recommendation: Use `ANTHROPIC_API_KEY` — that is the SDK's default env var name, so `anthropic.NewClient()` picks it up automatically with no explicit `option.WithAPIKey`.

4. **Inventory HQ app slug**
   - What we know: `hq_apps` table has seed rows; inventory.html exists and has an active tile in index.html
   - What's unclear: Does `hq_apps` currently have a row for `inventory`? The seed data in the docs shows 7 apps (purchasing, payroll, scheduling, hiring, bi, users, operations) — `inventory` is not listed.
   - Recommendation: Migration 0024 or a separate seed update should `INSERT INTO hq_apps (slug, name, icon, enabled) VALUES ('inventory', 'Inventory', '📦', true) ON CONFLICT (slug) DO NOTHING`. Planner must add this.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25.5 | All backend code | Yes | 1.25.5 | — |
| github.com/anthropics/anthropic-sdk-go | receipt/parser.go | Yes (installable) | v1.37.0 | — |
| github.com/aws/aws-sdk-go-v2/service/s3 | photos/spaces.go | Yes (installable) | v1.99.1 | — |
| github.com/antzucaro/matchr | receipt/fuzzy.go | Yes (installable) | latest | — |
| DO Spaces bucket | Photo storage | Unknown | — | Cannot proceed without bucket + credentials |
| Mercury API key | Receipt ingestion | Unknown | — | Worker gracefully skips cycle if key absent |
| Anthropic API key | Receipt parsing | Unknown | — | Worker logs error + skips if key absent |
| Playwright | E2E tests | Yes | (existing) | — |

**Missing dependencies with no fallback:**
- DO Spaces bucket must exist with CORS policy set before photo upload tests can run. Planner should add a Wave 0 infrastructure task.

**Missing dependencies with fallback:**
- Mercury API key and Anthropic API key: if absent, the background worker logs and skips gracefully. Inventory CRUD and photo upload can be tested independently.

---

## Validation Architecture

> `nyquist_validation` is `false` in `.planning/config.json` — this section is skipped.

---

## Sources

### Primary (HIGH confidence)
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/main.go` — Complete receipt ingestion pipeline reference; all algorithms confirmed by direct code read
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/services/mercury.go` — Mercury API HTTP client pattern
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/services/services.go` — ValidateReceiptData exact logic
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/services/fuzzy.go` — DerivePurchaseItem with matchr.JaroWinkler at 0.85 threshold
- `/Users/jamal/projects/yumyums/hq/backend/internal/me/handler.go` — /me/apps already implemented; confirmed queryUserApps logic
- `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go` — Route registration pattern, startup seed pattern
- `/Users/jamal/projects/yumyums/hq/backend/go.mod` — Current deps; confirmed aws-sdk-go-v2 and anthropic-sdk-go not yet present
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0021_ob_signoffs.sql` — Migration format
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0023_multi_role.sql` — Latest migration number (0024/0025 are next)
- `/Users/jamal/projects/yumyums/hq/inventory.html` — 933-line mock data inventory; confirmed all MOCK_ arrays to delete
- `/Users/jamal/projects/yumyums/hq/workflows.html` — Photo modal/capture UI at lines 413-467; presign stub at 1453-1456
- `/Users/jamal/projects/yumyums/hq/index.html` — Current checkAuth() at line 119; tile structure confirmed
- `go get` probe — Confirmed `github.com/anthropics/anthropic-sdk-go v1.37.0` and `github.com/aws/aws-sdk-go-v2/service/s3 v1.99.1` resolve and download cleanly (probe reverted)

### Secondary (MEDIUM confidence)
- `docs/user-management-api.md` — /me/apps contract; GET /api/v1/me/apps response shape; PWA offline localStorage fallback described
- `.planning/phases/12-inventory-photos-tile-permissions/12-UI-SPEC.md` — UI contract for review queue, tile filter, photo upload states; confirmed component reuse list

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries probed against module proxy; versions confirmed
- Architecture: HIGH — patterns directly derived from existing codebase (Phase 10/11) and reference baserow project
- Pitfalls: HIGH — identified from direct code inspection (Mercury amount sign in services.go, blob: URL stub in workflows.html, DO Spaces path-style requirement)
- Receipt pipeline port: HIGH — source code read line by line; algorithms are deterministic and well-tested

**Research date:** 2026-04-18
**Valid until:** 2026-05-18 (30 days — libraries stable; Mercury/Anthropic APIs stable)
