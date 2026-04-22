# Architecture Research

**Domain:** Purchase Order Workflow + Shopping Lists + Alerts + Cutoff Scheduling
**Researched:** 2026-04-22
**Confidence:** HIGH — derived directly from reading live codebase, not inferred from docs

---

## Existing Architecture (What We're Extending)

### System Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                         Browser (PWA)                             │
│  ┌──────────┐ ┌────────────────┐ ┌───────────────┐ ┌──────────┐ │
│  │index.html│ │purchasing.html │ │inventory.html │ │users.html│ │
│  │(launcher)│ │(MOCK — to wire)│ │(5-tab tool)   │ │(roles)   │ │
│  └────┬─────┘ └───────┬────────┘ └───────┬───────┘ └────┬─────┘ │
│       │               │                  │               │        │
│       └───────────────┴──────────────────┴───────────────┘        │
│                         fetch() + WebSocket                        │
└──────────────────────────────────────────────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Go HTTP Server      │
                    │  chi router          │
                    │  /api/v1/*           │
                    │  /ws (WebSocket hub) │
                    │  Background workers  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
    ┌─────────▼──────┐ ┌───────▼──────┐ ┌──────▼────────┐
    │  PostgreSQL     │ │ DO Spaces    │ │ External APIs │
    │  (pgx v5 pool) │ │ (S3 presign) │ │ Mercury bank  │
    │  goose migr.   │ │              │ │ Anthropic AI  │
    └────────────────┘ └──────────────┘ └───────────────┘
```

### Existing Table Schema (Inventory-relevant)

```
vendors           — food suppliers (name UNIQUE)
purchase_items    — canonical catalog items (description UNIQUE, group_id FK)
item_groups       — item categories (name, par_days, low/high thresholds)
item_group_tags   — m2m join for group tags
tags              — label taxonomy
purchase_events   — actual receipts/purchases (vendor_id FK, bank_tx_id UNIQUE)
purchase_line_items — line items on a purchase event (purchase_item_id FK)
pending_purchases — unreviewed receipts from Mercury worker (items JSONB)
stock_count_overrides — manual inventory count by item_description (TEXT PK)
users             — crew with roles[] TEXT[], status
sessions          — httpOnly cookie auth
hq_apps           — app tile registry (slug, name, icon)
app_permissions   — role_grants + user_grants per app slug
```

### Existing API Routes (Inventory)

```
GET  /api/v1/inventory/vendors           — list vendors
POST /api/v1/inventory/vendors           — create vendor
PUT  /api/v1/inventory/vendors           — update vendor name
POST /api/v1/inventory/vendors/merge     — merge two vendors

GET  /api/v1/inventory/purchases         — list purchase events (paginated)
POST /api/v1/inventory/purchases         — create purchase event
GET  /api/v1/inventory/purchases/pending — list unreviewed receipts
POST /api/v1/inventory/purchases/confirm — confirm pending purchase
POST /api/v1/inventory/purchases/discard — discard pending purchase
PUT  /api/v1/inventory/purchases/pending-items — save item selections on pending

GET  /api/v1/inventory/stock             — aggregated stock levels
POST /api/v1/inventory/stock/count       — upsert stock count override

GET  /api/v1/inventory/items             — list catalog items
POST /api/v1/inventory/items             — create item
PUT  /api/v1/inventory/items             — update item
POST /api/v1/inventory/items/merge       — merge items

GET  /api/v1/inventory/groups            — list item groups (with tags)
POST /api/v1/inventory/groups            — create group
PUT  /api/v1/inventory/groups            — update group thresholds
GET  /api/v1/inventory/tags              — list tags
```

### Background Workers (Existing Pattern)

The receipt worker (`internal/receipt/worker.go`) establishes the pattern for background work:
- `StartWorker(ctx, cfg)` — launches goroutine, runs immediately, then on ticker
- Config struct holds all dependencies (pool, API keys, presigner)
- Graceful skip if env vars missing
- Registered in `main.go` after server setup

The cutoff scheduler must follow this same pattern.

---

## New Components Required for v3.0

### 1. New Database Tables

```sql
-- Purchase Orders: the week's ordering document
CREATE TABLE purchase_orders (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  week_start   DATE NOT NULL UNIQUE,       -- Monday of the order week
  status       TEXT NOT NULL DEFAULT 'draft'
                 CHECK (status IN ('draft', 'locked', 'approved')),
  submitted_by UUID REFERENCES users(id),
  submitted_at TIMESTAMPTZ,
  approved_by  UUID REFERENCES users(id),
  approved_at  TIMESTAMPTZ,
  locked_at    TIMESTAMPTZ,               -- when cutoff fired (auto-lock)
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- PO Line Items: what crew requested for that week
CREATE TABLE po_line_items (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  po_id            UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
  purchase_item_id UUID NOT NULL REFERENCES purchase_items(id),
  quantity         NUMERIC(10,2) NOT NULL DEFAULT 0,
  unit             TEXT,                  -- lb, case, each (from item group)
  note             TEXT,
  added_by         UUID NOT NULL REFERENCES users(id),
  added_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (po_id, purchase_item_id)        -- one row per item per PO
);

-- Shopping Lists: generated from approved PO (1:1 with PO)
CREATE TABLE shopping_lists (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  po_id        UUID NOT NULL UNIQUE REFERENCES purchase_orders(id),
  assigned_to  UUID REFERENCES users(id), -- nullable — role-based or specific user
  assigned_role TEXT,                     -- 'manager', 'team_member', etc.
  completed_by UUID REFERENCES users(id),
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Shopping List Items: 1:1 with po_line_items, tracks check-off state
CREATE TABLE shopping_list_items (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shopping_list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  po_line_item_id  UUID NOT NULL REFERENCES po_line_items(id),
  checked          BOOLEAN NOT NULL DEFAULT false,
  checked_by       UUID REFERENCES users(id),
  checked_at       TIMESTAMPTZ,
  store_note       TEXT,                  -- editable inline location note
  UNIQUE (shopping_list_id, po_line_item_id)
);

-- Cutoff Config: admin-configurable weekly cutoff schedule
CREATE TABLE cutoff_config (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  day_of_week    INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Sun
  cutoff_time    TIME NOT NULL,           -- local time (e.g. 18:00)
  timezone       TEXT NOT NULL DEFAULT 'America/New_York',
  reminder_hours INTEGER[] NOT NULL DEFAULT '{2,24}', -- hours before cutoff
  enabled        BOOLEAN NOT NULL DEFAULT true,
  updated_by     UUID REFERENCES users(id),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Repurchase Badges: track "Repurchased +N" state per item
CREATE TABLE repurchase_badges (
  purchase_item_id UUID PRIMARY KEY REFERENCES purchase_items(id),
  quantity         NUMERIC(10,2) NOT NULL DEFAULT 0,
  reset_date       DATE,                  -- configurable reset date
  last_purchase_at TIMESTAMPTZ,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User Communication Preferences: Zoho Cliq / email
CREATE TABLE user_notification_prefs (
  user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  zoho_cliq_token TEXT,                  -- webhook URL or user token
  email        TEXT,                     -- can differ from login email
  prefer_cliq  BOOLEAN NOT NULL DEFAULT true,
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Alert Log: record of sent alerts for debugging / deduplication
CREATE TABLE alert_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_type TEXT NOT NULL,             -- 'cutoff_reminder', 'out_of_stock', 'shopping_complete'
  ref_id     UUID,                      -- po_id, shopping_list_id, etc.
  channel    TEXT NOT NULL,             -- 'zoho_cliq', 'email'
  recipient  TEXT NOT NULL,             -- user_id or email
  sent_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  payload    JSONB
);
```

**Migration numbering:** Next migration is `0034_purchase_orders.sql`. Each logical group gets its own file following the existing single-purpose migration pattern.

### 2. Purchase Items Extended

The existing `purchase_items` table needs two new columns added via a separate migration:

```sql
-- 0039_purchase_items_photo_store.sql
ALTER TABLE purchase_items
  ADD COLUMN photo_url TEXT,
  ADD COLUMN store_location TEXT;   -- e.g. "Restaurant Depot - Freezer aisle 3"
```

This is a non-breaking ALTER — existing rows get NULL for both columns.

### 3. New API Routes

All routes go under `/api/v1/purchasing/*` (new namespace, parallel to `/api/v1/inventory/*`):

```
-- Purchase Orders
GET    /api/v1/purchasing/orders                 — list POs (most recent first)
POST   /api/v1/purchasing/orders                 — create or get current week's PO
GET    /api/v1/purchasing/orders/:id             — get single PO with line items
POST   /api/v1/purchasing/orders/:id/lock        — admin: lock PO (manual or scheduled)
POST   /api/v1/purchasing/orders/:id/approve     — admin: approve locked PO

-- PO Line Items
PUT    /api/v1/purchasing/orders/:id/items       — upsert line items (qty=0 means remove)
GET    /api/v1/purchasing/orders/:id/suggestions — reorder suggestions from stock

-- Shopping Lists
GET    /api/v1/purchasing/shopping               — list shopping lists (RBAC filtered)
GET    /api/v1/purchasing/shopping/:id           — get list with items
POST   /api/v1/purchasing/shopping/:id/check     — check/uncheck item
POST   /api/v1/purchasing/shopping/:id/complete  — complete list + trigger missing-items alert
PUT    /api/v1/purchasing/shopping/:id/items/:item_id — update store_note

-- Cutoff Config (admin only)
GET    /api/v1/purchasing/cutoff                 — get current cutoff config
PUT    /api/v1/purchasing/cutoff                 — update cutoff config

-- Repurchase Badges
GET    /api/v1/purchasing/badges                 — get all repurchase badges
POST   /api/v1/purchasing/badges/reset           — reset badge for item
PUT    /api/v1/purchasing/badges/reset-date      — set global reset date

-- Notification Prefs (on /me namespace, not /purchasing)
GET    /api/v1/me/notification-prefs             — get my prefs
PUT    /api/v1/me/notification-prefs             — update my prefs

-- Notion Import (admin, idempotent)
POST   /api/v1/purchasing/import/notion          — import items from Notion export JSON
```

### 4. New Go Package: `internal/purchasing`

Mirror the structure of `internal/inventory`:

```
backend/internal/purchasing/
├── handler.go      — HTTP handlers for all purchasing routes
├── types.go        — PurchaseOrder, POLineItem, ShoppingList, etc.
├── service.go      — business logic (generate shopping list, lock PO, etc.)
├── scheduler.go    — cutoff scheduler goroutine (mirrors receipt/worker.go)
└── notifier.go     — Zoho Cliq + email alert sending
```

**Registration in `main.go`:**
- `purchasing.StartScheduler(ctx, schedulerCfg)` after `receipt.StartWorker`
- Mount `r.Route("/api/v1/purchasing", ...)` in the authenticated group

Optionally extract alert logic into `internal/notifier` if it becomes cross-cutting (future tools need alerts). For v3.0 scope, keeping it in `purchasing/notifier.go` is simpler and consistent with the codebase's single-responsibility-per-package pattern.

### 5. New Frontend Page: `purchasing.html` (Rewrite from Mock)

The current `purchasing.html` is a fully hardcoded mock (~90 LOC). It gets replaced with a real 3-tab tool. The file path and SW registration stay the same — no new page is needed.

```
Tab 1: "Order"    — current week PO form (reorder suggestions + search-add)
Tab 2: "Locked"   — locked/submitted view (read-only for crew, approve for admin)
Tab 3: "Shopping" — shopping checklists (RBAC filtered to assignee/role)
```

**Tab 1 (Order) data sources:**
- `GET /api/v1/purchasing/orders` — current week PO
- `GET /api/v1/purchasing/orders/:id/suggestions` — reorder suggestions from stock
- `GET /api/v1/inventory/items` — full catalog for search-add
- `GET /api/v1/inventory/groups` — group labels for display

**Tab 3 (Shopping) data sources:**
- `GET /api/v1/purchasing/shopping` — lists visible to current user
- Item photos served from `purchase_items.photo_url` (DO Spaces public URL)
- Store locations from `purchase_items.store_location`

### 6. Users Tab: Notification Preferences

Add a "Notifications" sub-section to `users.html` for the logged-in user to configure Zoho Cliq and email. This is not a new page — it's an additive UI section in the existing Users tool, available to all authenticated users (not admin-only) to configure their own preferences.

---

## Data Flow: PO Workflow

```
Crew (purchasing.html Tab 1)
  → PUT /api/v1/purchasing/orders/:id/items   (upsert quantities)
  → PO stored in purchase_orders + po_line_items

Cutoff fires (scheduler goroutine OR admin taps Lock)
  → POST /api/v1/purchasing/orders/:id/lock
  → purchase_orders.status = 'locked', locked_at = now()
  → Reminder alerts already sent 24h + 2h before cutoff time

Admin reviews locked PO (Tab 2)
  → POST /api/v1/purchasing/orders/:id/approve
  → purchase_orders.status = 'approved'
  → Server generates shopping_list + shopping_list_items (1:1 from po_line_items)
  → Shopping list becomes visible in Tab 3 per RBAC

Shopper (Tab 3)
  → POST /api/v1/purchasing/shopping/:id/check   (each item)
  → POST /api/v1/purchasing/shopping/:id/complete
  → If any items unchecked → alert fires for missing items via notifier
  → Checked items → update repurchase_badges (quantity += checked qty)
```

## Data Flow: Reorder Suggestions

```
GET /api/v1/purchasing/orders/:id/suggestions
  → Query purchase_items JOIN item_groups (low_threshold, high_threshold)
  → Query current stock: COALESCE(stock_count_overrides.quantity,
                                   SUM(purchase_line_items.quantity))
  → WHERE current_stock < item_groups.low_threshold
  → JOIN purchase_items for photo_url, store_location
  → Return sorted by group, then by stock deficit DESC
```

## Data Flow: Alerts

```
Scheduler goroutine (ticker every 15 min)
  → SELECT cutoff_config WHERE enabled = true
  → Calculate next cutoff timestamp in configured timezone
  → For each reminder_hours threshold:
      IF (cutoff_time - now()) <= reminder_hours AND
         NOT EXISTS (SELECT 1 FROM alert_log WHERE alert_type = 'cutoff_reminder'
                     AND ref_id = po_id AND sent_at > now() - interval '1 hour')
      THEN send alert

Notifier (per user in target role)
  → SELECT user_notification_prefs WHERE user_id = ?
  → IF prefer_cliq AND zoho_cliq_token IS NOT NULL → POST to Cliq webhook
  → ELSE → send email
  → INSERT INTO alert_log (for dedup on next tick)
```

## Data Flow: Repurchase Badges

```
POST /api/v1/purchasing/shopping/:id/complete
  → For each shopping_list_item WHERE checked = true:
      UPSERT repurchase_badges
        SET quantity = quantity + po_line_item.quantity,
            last_purchase_at = now()
  → GET /api/v1/inventory/stock response includes repurchase badge data via LEFT JOIN

Badge reset (two modes):
  1. Manual: POST /api/v1/purchasing/badges/reset {purchase_item_id}
             → UPDATE repurchase_badges SET quantity = 0
  2. Scheduled: PUT /api/v1/purchasing/badges/reset-date {date}
             → Scheduler checks daily; zeros all badges on that date
```

---

## Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `purchasing/handler.go` | HTTP I/O, auth checks, JSON marshal | `purchasing/service.go`, `pgxpool` |
| `purchasing/service.go` | Business logic: generate shopping list, lock PO, approval side effects | `pgxpool`, `purchasing/notifier.go` |
| `purchasing/scheduler.go` | Time-based cutoff checks, reminder scheduling, badge reset | `service.go`, `notifier.go` |
| `purchasing/notifier.go` | Send Zoho Cliq + email alerts, write alert_log | External Cliq webhook, SMTP |
| `purchasing.html` | PO form, locked view, shopping checklist | `/api/v1/purchasing/*`, `/api/v1/inventory/*` |
| `users.html` | Notification prefs section (additive) | `/api/v1/me/notification-prefs` |

---

## Architectural Patterns

### Pattern 1: Upsert-by-natural-key for PO Line Items

**What:** `PUT /api/v1/purchasing/orders/:id/items` accepts the full item list and uses `ON CONFLICT (po_id, purchase_item_id) DO UPDATE`. Quantity = 0 means the frontend stepper was decremented to zero — backend deletes or skips it on read.

**When to use:** Any time a frontend stepper/checkbox represents the full desired state (not a delta). Matches the existing `pending_purchases.items` JSONB pattern and `stock_count_overrides` upsert.

**Trade-offs:** Sends full state on every save (small payload given ~20-50 items/week). Avoids three separate create/update/delete endpoints.

**Constraint:** Only allowed when `purchase_orders.status = 'draft'`. Writes when status is `locked` or `approved` return 409 for non-admin users.

### Pattern 2: Shopping List Generated at Approval Time (Snapshot)

**What:** Shopping list is a snapshot created when admin approves. `shopping_list_items` rows are created once from `po_line_items` at approval time inside a single Postgres transaction.

**When to use:** Any time you need an immutable record of what was approved. Prevents PO edits from silently changing an in-progress shop.

**Trade-offs:** Slight data duplication (shopping_list_items mirrors po_line_items). Worth it for correctness.

**Implementation:** `POST /api/v1/purchasing/orders/:id/approve` opens one transaction: set status='approved', INSERT shopping_lists, bulk INSERT shopping_list_items from po_line_items.

### Pattern 3: Alert Deduplication via alert_log

**What:** Before sending any alert, check `alert_log` for a matching `(alert_type, ref_id, channel, recipient)` within the relevant time window. Insert into `alert_log` before (or immediately after) send.

**When to use:** Any recurring scheduler that fires on a ticker — same as `bankTxIDExists()` in the receipt worker.

**Trade-offs:** Adds one SELECT before each alert. Acceptable at 1-5 users.

**Implementation:** Partial unique index on `alert_log (alert_type, ref_id, recipient)` WHERE `sent_at > now() - interval '1 day'`. Use `INSERT ... ON CONFLICT DO NOTHING` as the dedup primitive.

### Pattern 4: Scheduler Follows receipt/worker.go Model

**What:** `purchasing.StartScheduler(ctx, cfg)` runs in a goroutine, uses `time.NewTicker`, runs immediately then on each tick, skips gracefully if config env vars are missing.

**When to use:** Any periodic background job in this codebase.

**Trade-offs:** Simple; no external job scheduler needed. Loses state on restart (timer resets). Acceptable for weekly cadence — worst case: a reminder fires slightly late after a deploy.

---

## Integration Points: New vs. Modified

### New (net-new files, no existing code modified)

| Component | Type | Notes |
|-----------|------|-------|
| `backend/internal/purchasing/` | New Go package | handler, service, types, scheduler, notifier |
| `backend/internal/db/migrations/0034_purchase_orders.sql` | New migration | purchase_orders, po_line_items |
| `backend/internal/db/migrations/0035_shopping_lists.sql` | New migration | shopping_lists, shopping_list_items |
| `backend/internal/db/migrations/0036_cutoff_config.sql` | New migration | cutoff_config, alert_log |
| `backend/internal/db/migrations/0037_notification_prefs.sql` | New migration | user_notification_prefs |
| `backend/internal/db/migrations/0038_repurchase_badges.sql` | New migration | repurchase_badges |
| `backend/internal/db/migrations/0039_purchase_items_photo_store.sql` | New migration | ALTER TABLE purchase_items |

### Modified (changes to existing files)

| File | Change | Impact |
|------|--------|--------|
| `backend/cmd/server/main.go` | Mount `/api/v1/purchasing/*`, start cutoff scheduler | Low — additive only, no existing routes touched |
| `backend/internal/inventory/handler.go` | `GetStockHandler` JOIN to `repurchase_badges` | Low — additive response fields, zero-valued for items with no badge |
| `backend/internal/inventory/types.go` | `StockItem` gets `RepurchasedQty *float64`, `ResetDate *string` fields | Low — nullable, backward-compatible |
| `purchasing.html` | Full rewrite from mock to real API-backed 3-tab tool | High — complete replacement of ~90 LOC mock |
| `users.html` | Add notification prefs sub-section | Low — additive UI section |
| `sw.js` | Rebuild via `node build-sw.js` after any HTML change | Mandatory step per project convention |

---

## External Service Integration

| Service | Integration Pattern | Config | Notes |
|---------|---------------------|--------|-------|
| Zoho Cliq | POST to incoming webhook URL | `ZOHO_CLIQ_WEBHOOK_URL` env var | One webhook for the shared channel. No per-user tokens needed for channel-level alerts. |
| Email (SMTP) | Standard `net/smtp` or SendGrid HTTP API | `SMTP_HOST`, `SMTP_USER`, `SMTP_PASS` env vars | Fallback channel when user prefers email or has no Cliq token. |
| Notion API | One-shot import via `POST /api/v1/purchasing/import/notion` | Request body: Notion exported JSON | Maps Notion DB properties to `purchase_items` fields. Idempotent via `ON CONFLICT (description) DO UPDATE`. |

---

## Build Order (Phase Dependencies)

Features have hard dependencies that constrain sequencing:

```
Phase 1: Data Foundation
  ├── Migrations 0034–0039 (all new tables + ALTER)
  └── Extend /inventory/items API to return photo_url, store_location

Phase 2: PO Form (purchasing.html rewrite, Tab 1)
  ├── Requires: Phase 1 (purchase_orders + po_line_items tables)
  ├── POST /api/v1/purchasing/orders             — get-or-create current week PO
  ├── GET  /api/v1/purchasing/orders/:id/suggestions — reorder from stock
  └── PUT  /api/v1/purchasing/orders/:id/items   — save quantities

Phase 3: Notion Import
  ├── Requires: Phase 1 (photo_url + store_location columns exist)
  └── POST /api/v1/purchasing/import/notion       — seed catalog with photos

Phase 4: Cutoff + Lock
  ├── Requires: Phase 2 (PO exists to lock)
  ├── GET/PUT  /api/v1/purchasing/cutoff          — cutoff config CRUD
  ├── POST /api/v1/purchasing/orders/:id/lock     — manual lock endpoint
  └── purchasing/scheduler.go                    — auto-lock on cutoff time
          + purchasing.html Tab 2 (locked view)

Phase 5: Admin Approval + Shopping List
  ├── Requires: Phase 4 (locked PO to approve)
  ├── POST /api/v1/purchasing/orders/:id/approve  — approve + generate list
  ├── GET  /api/v1/purchasing/shopping            — list shopping lists (RBAC)
  ├── GET  /api/v1/purchasing/shopping/:id        — single list with items
  ├── POST /api/v1/purchasing/shopping/:id/check  — check/uncheck items
  ├── POST /api/v1/purchasing/shopping/:id/complete — complete list
  └── purchasing.html Tab 3 (shopping checklist)

Phase 6: Alerts + Notification Prefs
  ├── Requires: Phase 4 (scheduler exists to send reminders)
  ├── Requires: Phase 5 (shopping complete triggers missing-items alert)
  ├── purchasing/notifier.go                     — Zoho Cliq + email send
  ├── GET/PUT /api/v1/me/notification-prefs       — prefs endpoints
  └── users.html notification prefs section

Phase 7: Repurchase Badges
  ├── Requires: Phase 5 (shopping completion exists)
  ├── Update GetStockHandler to JOIN repurchase_badges
  ├── Update StockItem type with badge fields
  └── GET/POST/PUT /api/v1/purchasing/badges      — badge CRUD
```

**Critical dependency:** Phases 1-4 can be built and tested independently. Phase 5 (shopping list generation) is the gate for Tab 3, alerts, and badges. Ship each phase as a deployable increment.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Per-Item CRUD Endpoints for PO

**What people do:** Separate `POST` (add item), `DELETE` (remove item), `PATCH` (update quantity) endpoints.

**Why wrong:** Three endpoints for one stepper UX. The existing inventory stepper (stock count overrides) uses a single upsert. The `pending_purchases.items` JSONB also sends full state.

**Do instead:** Single `PUT /api/v1/purchasing/orders/:id/items` with full item list. Backend upserts all rows; quantity=0 means remove.

### Anti-Pattern 2: Live Shopping List (No Snapshot)

**What people do:** Shopping list reads directly from `po_line_items` in real time with a JOIN.

**Why wrong:** If the PO is edited after approval (edge case or bug), the shopping list silently changes mid-shop. The checked state stored per `po_line_item_id` would also break if a row is deleted.

**Do instead:** Snapshot into `shopping_list_items` at approval time inside a transaction. PO is already frozen (locked) before approval, so snapshot matches approved state exactly.

### Anti-Pattern 3: Frontend Polls for Lock Status

**What people do:** `purchasing.html` polls `GET /api/v1/purchasing/orders` every 30 seconds to detect when the PO transitions to locked.

**Why wrong:** Unnecessary load. The existing WebSocket hub is already connected.

**Do instead:** When the scheduler (or admin) locks a PO, broadcast a WebSocket message: `{type: "PO_LOCKED", po_id: "..."}`. The frontend re-renders Tab 1 to the locked state. This matches the existing op-log/WebSocket pattern in `workflows.html`.

### Anti-Pattern 4: Duplicate Notifier Logic

**What people do:** Inline Zoho Cliq HTTP POST in three places: the cutoff scheduler, the shopping completion handler, and the out-of-stock checker.

**Why wrong:** Three copies of HTTP client setup, error handling, and dedup logic. Changes (e.g., new Cliq API format) require three edits.

**Do instead:** Single `notifier.Send(ctx, pool, payload)` function in `purchasing/notifier.go`. All three callers use this single entry point. It handles channel selection, alert_log dedup, and graceful skip when env vars are missing.

### Anti-Pattern 5: Notion Data as a SQL Migration

**What people do:** Write Notion export data directly into a goose migration file as INSERT statements.

**Why wrong:** Migrations run on every startup (`goose.Up` is idempotent but still queries the schema). Seed data mixed with schema changes is fragile. Notion data will be edited in the app after import — running the migration again could overwrite manual edits.

**Do instead:** `POST /api/v1/purchasing/import/notion` endpoint with idempotent `ON CONFLICT (description) DO UPDATE`. Admin triggers it once. Re-triggering is safe and picks up Notion updates without touching other data. Same pattern as `inventory.SeedInventoryFixtures` which uses upserts and is idempotent.

---

## Scaling Considerations

At 1-5 users and weekly ordering frequency, none of these features create load concerns.

| Concern | Now (1-5 users) | Future consideration |
|---------|-----------------|----------------------|
| PO concurrent writes | Single-writer fine (crew adds items one at a time on phones) | Optimistic locking if conflicts emerge (use `updated_at` version check) |
| Alert delivery | Fire-and-forget acceptable; ticker runs 15min | Add retry table if Cliq/SMTP flakiness becomes a problem |
| Reorder suggestions query | Inline JOIN is fast (~100 items) | Materialize if `purchase_line_items` grows beyond ~50k rows |
| Shopping list check-off | Direct Postgres write per tap | Fine for 1-5 users; add optimistic UI if latency is felt on slow connections |

---

## Sources

- Live codebase read directly:
  - `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/inventory/handler.go`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/inventory/types.go`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/worker.go`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/sync/hub.go`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0024_inventory.sql`
  - `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0033_stock_count_overrides.sql`
  - `/Users/jamal/projects/yumyums/hq/purchasing.html`
- Project requirements: `.planning/PROJECT.md` (v3.0 feature list)
- Existing patterns: `stock_count_overrides` upsert, `bankTxIDExists` idempotency, `receipt.StartWorker` scheduler model

---

*Architecture research for: Yumyums HQ v3.0 — Purchase Orders, Shopping Lists, Alerts, Cutoff Scheduling*
*Researched: 2026-04-22*
