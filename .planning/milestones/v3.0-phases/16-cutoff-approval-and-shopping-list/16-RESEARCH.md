# Phase 16: Cutoff, Approval, and Shopping List - Research

**Researched:** 2026-04-22
**Domain:** Go backend state machine, Postgres scheduler, vanilla JS mobile UI — extending existing purchasing package
**Confidence:** HIGH — all findings derived from live codebase reads and verified prior research documents

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Cutoff Config UX**
- D-01: Admin configures cutoff by tapping the cutoff pill on the PO tab header — expands inline to show day dropdown + time dropdown + Save/Cancel
- D-02: Non-admin users see the cutoff pill as read-only
- D-03: Cutoff stored in a `cutoff_config` table (day_of_week, time, timezone) — one row, upserted

**Cutoff Test Simulation**
- D-04: Uses the same production endpoint: `POST /api/v1/purchasing/simulate-cutoff` — triggers the real lock logic (locks current draft, creates new draft for next week)
- D-05: Admin-only endpoint. To test again, user truncates the DB and re-seeds
- D-06: No separate test path — same code path as the scheduler

**PO Tab (Locked State)**
- D-07: Photo card layout (same as Order tab) but steppers disabled for non-admin crew
- D-08: Admin sees an "Edit ✎" toggle to unlock steppers for editing within locked state
- D-09: Items grouped by vendor with estimated totals per vendor (like current mock PO tab)
- D-10: "Approve & Create Shopping List" button at bottom — admin-only
- D-11: Approve blocked if previous week's shopping list is still active — shows message: "Complete last week's shopping list before approving this one"

**State Machine**
- D-12: PO lifecycle: draft → locked → approved → shopping_active → completed
- D-13: Admin can unlock a locked PO back to draft (e.g. cutoff was premature) — but NOT after approval
- D-14: On cutoff: current draft transitions to locked, new draft auto-created for next week
- D-15: On approve: shopping list snapshot created (immutable), PO status → approved, shopping list status → active
- D-16: On all vendor sections completed: shopping list status → completed, PO status → completed

**Shopping Check-off**
- D-17: Toast + badge pattern for missing photo/location: item checks off immediately, toast appears "Lemons needs photo & location [Add Now]", badge shows "⚠ No photo" on the item card
- D-18: User can dismiss toast (item stays checked off without photo/location) or tap "Add Now" to open photo capture + location input
- D-19: No skip confirmation required — toast is non-blocking, badge persists as a gentle reminder
- D-20: Photo capture uses same DO Spaces presigned PUT pattern from Phase 12

**Per-Vendor Completion**
- D-21: Each vendor section has its own status (pending/completed) and its own "Complete [Vendor]" button
- D-22: Completing a vendor section records who completed it and when
- D-23: Unchecked items in a completed vendor section are flagged as "missing" — reported in the completion alert (Phase 17)
- D-24: Overall shopping list transitions to "completed" only when ALL vendor sections are done

**History Tab**
- D-25: Shows past shopping lists with: week label, vendor breakdown, missing items count, who completed each vendor section
- D-26: Tappable to expand and see full item list with checked/unchecked status

### Claude's Discretion

- Scheduler implementation (goroutine ticker vs gocron — research recommended)
- Shopping list snapshot schema design
- History tab sort order (newest first assumed)
- Animation on approve button
- Toast auto-dismiss timing

### Deferred Ideas (OUT OF SCOPE)

- Alert notifications (cutoff reminder, out-of-stock, missing items on complete) — Phase 17
- Repurchase badges on inventory — Phase 17
- Email delivery — Phase 17
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CUT-01 | Admin can configure a recurring weekly cutoff schedule (day + time) | D-01 through D-03: inline pill UI, `cutoff_config` table (upsert one row) |
| CUT-02 | After cutoff time, PO is locked — only admin can edit | D-07/D-08: locked view with admin "Edit ✎" toggle; optimistic lock via version column in WHERE clause |
| CUT-03 | Backend provides a test command to simulate cutoff for easy testing | D-04/D-05/D-06: `POST /api/v1/purchasing/simulate-cutoff` triggers real lock logic |
| CUT-04 | Admin can approve a locked PO, which generates a shopping checklist | D-10/D-15: approve button, snapshot INSERT in single Postgres transaction |
| SHOP-01 | Approved PO generates a shopping checklist (1:1 mapping) | Snapshot pattern: INSERT shopping_list_items SELECT from po_line_items at approval time |
| SHOP-02 | Shopping checklist tab with RBAC — assignable to specific members or roles | `shopping_lists.assigned_to / assigned_role`; server-side role check on all shopping endpoints |
| SHOP-03 | Shopping list shows each item's photo and store location (tap icon to reveal) | `shopping_list_items` snapshots photo_url and store_location; tap-to-reveal reuses existing lightbox CSS |
| SHOP-04 | User can edit store location notes inline from the shopping list | `PUT /api/v1/purchasing/shopping/{id}/items/{item_id}` updates store_note on shopping_list_items |
| SHOP-05 | User can mark items as checked off while shopping | `POST /api/v1/purchasing/shopping/{id}/check` sets checked=true, checked_by, checked_at |
| SHOP-06 | "Complete" button sends alert for any missing/unchecked items | `POST /api/v1/purchasing/shopping/{id}/vendors/{vendor_id}/complete` — alert delivery deferred to Phase 17 |
| SHOP-07 | When checking off item without photo/location, shopper is prompted (can skip) | D-17/D-18/D-19: toast + badge UX; no confirm-each-skip required |
| SHOP-08 | Shopper can upload a photo for an item that doesn't have one | Reuse `POST /api/v1/photos/presign` with path_prefix "shopping"; UPDATE purchase_items.photo_url |
</phase_requirements>

---

## Summary

Phase 16 extends the existing purchasing package with three major capabilities: cutoff scheduling, admin approval, and a shopping checklist. All backend infrastructure (tables, router, service functions) follows established patterns from the receipt worker and inventory package. The purchasing.html already has 4 stub tabs (Order, Shopping, PO, History) — this phase wires up the 3 stub tabs.

The most critical architectural choice is the PO state machine. Six status values span two phases: `draft → locked → approved → shopping_active → completed`. Phase 16 owns all five transitions. The state machine has one high-risk edge case documented in prior research (Pitfall 6): admin unlock must be blocked after approval, which is already locked in via D-13.

The scheduler uses a goroutine ticker (same pattern as `receipt.StartWorker`) rather than `robfig/cron`. This avoids DST double-fire and missed-fire bugs. The `cutoff_config` table stores day_of_week + time in a named timezone (not UTC offset); the scheduler converts to UTC at runtime to compute the next fire time.

**Primary recommendation:** Build in three waves — (1) migrations + backend state machine, (2) PO tab (locked view + admin actions), (3) Shopping tab (check-off, photo, vendor completion). Keep Phase 17 alert hooks as no-op stubs so they compile and can be wired later.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| pgx/v5 | already in use | Postgres driver + pool | Project standard — all existing queries use pgxpool |
| chi v5 | already in use | HTTP router | Project standard — all routes use chi |
| Go stdlib `time` | stdlib | Scheduler ticker, timezone conversions | No external scheduler needed for weekly cadence |

### No New Dependencies Required

All Phase 16 backend functionality is achievable with the existing dependency set. The scheduler uses `time.NewTicker` + `time.LoadLocation` (both stdlib). No `robfig/cron` or `gocron` needed — and avoiding them eliminates the DST bug class entirely (per PITFALLS.md Pitfall 4).

**Installation:** none needed.

---

## Architecture Patterns

### Existing Database Schema (Phase 16 starts here)

Migrations through 0036 are applied. The next migration number is **0037**.

```
purchase_orders      — week_start, status (draft|locked|approved), version
po_line_items        — po_id, purchase_item_id, quantity, unit, added_by
purchase_items       — description, group_id, photo_url, store_location, full_name
item_groups          — name, par_days, low_threshold, high_threshold
vendors              — name (exists on purchase_events, NOT on purchase_items)
```

**Critical gap:** `purchase_items` has no `vendor_id` column. D-09 requires items grouped by vendor on the locked PO view. A migration must add `vendor_id` to `purchase_items` to support per-vendor grouping in the PO and shopping list.

### New Migrations Required

```
0037_cutoff_config.sql          — cutoff_config table
0038_shopping_lists.sql         — shopping_lists + shopping_list_vendor_sections + shopping_list_items
0039_purchase_items_vendor.sql  — ALTER TABLE purchase_items ADD COLUMN vendor_id UUID REFERENCES vendors(id)
```

### Pattern 1: PO State Machine — Status Transitions

**State values (ordered):**

```
draft → locked → approved → shopping_active → completed
```

**Transition rules:**
- `draft → locked`: scheduler OR `POST /simulate-cutoff` (admin-only); creates new draft for next week in same TX
- `locked → draft`: admin unlock (`POST /api/v1/purchasing/orders/{id}/unlock`); blocked if shopping list exists
- `locked → approved`: admin approve (`POST /api/v1/purchasing/orders/{id}/approve`); creates shopping list snapshot in same TX
- `approved → shopping_active`: triggered when shopping list is created (set immediately after approve TX)
- `shopping_active → completed`: all vendor sections completed

**DB constraint update needed:** Current migration 0034 has `CHECK (status IN ('draft', 'locked', 'approved'))`. A new migration must extend this to include `'shopping_active'` and `'completed'`. Alternatively, remove the CHECK constraint and enforce in Go.

**Recommended approach:** Drop the CHECK in migration 0037 or a dedicated migration. Status transitions enforced exclusively in Go (simpler to evolve). Add a `locked_at` and `approved_at` TIMESTAMPTZ column to `purchase_orders` for history/audit.

### Pattern 2: Snapshot at Approval Time

**What:** `POST /api/v1/purchasing/orders/{id}/approve` opens ONE Postgres transaction:
1. Verify PO status = 'locked'
2. Verify no active shopping list exists for a prior week (D-11)
3. `UPDATE purchase_orders SET status = 'shopping_active', approved_at = now(), approved_by = $userID WHERE id = $1 AND version = $currentVersion`
4. `INSERT INTO shopping_lists (...) VALUES (...) RETURNING id`
5. `INSERT INTO shopping_list_vendor_sections SELECT DISTINCT vendor_id, ...`
6. `INSERT INTO shopping_list_items SELECT id, $listID, photo_url, store_location, ... FROM po_line_items WHERE po_id = $1`
7. Commit

If any step fails, the entire TX rolls back. PO status remains 'locked'.

**Source:** ARCHITECTURE.md Pattern 2, prior research.

### Pattern 3: Cutoff Scheduler (Goroutine Ticker)

**What:** `purchasing.StartScheduler(ctx, cfg)` mirrors `receipt.StartWorker`:

```go
func StartScheduler(ctx context.Context, pool *pgxpool.Pool) {
    log.Println("cutoff scheduler: starting")
    go func() {
        runCutoffCheck(ctx, pool)           // run immediately
        ticker := time.NewTicker(15 * time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                runCutoffCheck(ctx, pool)
            }
        }
    }()
}
```

**`runCutoffCheck` logic:**
1. SELECT cutoff_config (single row)
2. Compute next cutoff UTC timestamp from day_of_week + time + timezone using `time.LoadLocation`
3. If `time.Now().UTC().After(nextCutoffUTC)` AND current draft PO exists AND PO is still 'draft':
   - BEGIN TX
   - UPDATE purchase_orders SET status = 'locked', locked_at = now() WHERE status = 'draft' AND week_start = $currentWeek
   - INSERT INTO purchase_orders (week_start, status) VALUES ($nextWeek, 'draft') ON CONFLICT DO NOTHING
   - COMMIT

**Timezone handling:** Store cutoff as `day_of_week INTEGER (0=Sun)` + `cutoff_time TIME` + `timezone TEXT`. At runtime:
```go
loc, _ := time.LoadLocation(cfg.Timezone)  // e.g. "America/Chicago"
// compute next occurrence of day_of_week + cutoff_time in loc
// convert to UTC for comparison with time.Now().UTC()
```

This avoids DST bugs because Go's `time.LoadLocation` handles DST transitions correctly. Never store the cutoff as a hardcoded UTC offset.

### Pattern 4: Per-Vendor Section Model

**What:** Shopping lists are organized by vendor. Each vendor section has its own status and completion record.

**New table: `shopping_list_vendor_sections`**

```sql
CREATE TABLE shopping_list_vendor_sections (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shopping_list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  vendor_id        UUID NOT NULL REFERENCES vendors(id),
  status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
  completed_by     UUID REFERENCES users(id),
  completed_at     TIMESTAMPTZ,
  UNIQUE (shopping_list_id, vendor_id)
);
```

**`POST /api/v1/purchasing/shopping/{id}/vendors/{vendor_id}/complete`:**
1. Verify current user has access
2. Mark unchecked items in this vendor section as "missing" (flag in shopping_list_items or just read at report time)
3. UPDATE shopping_list_vendor_sections SET status = 'completed', completed_by = $userID, completed_at = now()
4. Check if all vendor sections are completed → if yes, UPDATE shopping_lists SET status = 'completed' + UPDATE purchase_orders SET status = 'completed'
5. Return updated state

### Pattern 5: Admin Role Check

**Existing pattern:** `user.Roles` is `[]string`. Admin check:

```go
func isAdmin(user *auth.User) bool {
    if user.IsSuperadmin {
        return true
    }
    for _, r := range user.Roles {
        if r == "admin" || r == "manager" {
            return true
        }
    }
    return false
}
```

**Use for:** Cutoff config write, simulate-cutoff, PO lock/unlock, PO approve, "Edit ✎" toggle in UI.

### Pattern 6: Toast + Badge for Missing Photo/Location (Frontend)

**Decision D-17 through D-19:** Non-blocking toast pattern.

```javascript
// On item check-off:
function handleCheckItem(itemId) {
  var item = findItem(itemId);
  // Optimistic update
  item.checked = true;
  renderShoppingTab();
  // API call
  api('/api/v1/purchasing/shopping/' + LIST_ID + '/check', {
    method: 'POST',
    body: JSON.stringify({ item_id: itemId, checked: true })
  });
  // Show toast if missing photo or location
  var missing = [];
  if (!item.photo_url) missing.push('photo');
  if (!item.store_location) missing.push('location');
  if (missing.length) {
    showToast(item.item_name + ' needs ' + missing.join(' & '), itemId);
  }
}
```

**Toast auto-dismiss:** 5 seconds (Claude's discretion). Badge remains until photo/location added.

### Recommended Project Structure

New files for Phase 16:

```
backend/internal/purchasing/
├── handler.go          (EXTEND: add lock, approve, simulate-cutoff, shopping routes)
├── service.go          (EXTEND: LockPO, ApprovePO, GenerateShoppingList, etc.)
├── types.go            (EXTEND: ShoppingList, ShoppingListItem, CutoffConfig, etc.)
├── scheduler.go        (NEW: StartScheduler, runCutoffCheck)
└── cutoff.go           (NEW: timezone-aware next-cutoff computation — optional separate file)

backend/internal/db/migrations/
├── 0037_cutoff_config.sql
├── 0038_shopping_lists.sql
└── 0039_purchase_items_vendor.sql

purchasing.html         (EXTEND: wire up s2 Shopping tab, s3 PO tab, s4 History tab)
```

### Anti-Patterns to Avoid

- **Cutoff as Go guard only (not in DB WHERE clause):** Race condition window. The `UpsertLineItems` function already enforces status='draft' in the WHERE — do the same for lock and approve. Return 409 if version mismatch.
- **Inline alert send on vendor complete:** Phase 17 owns alerts. Phase 16 should write a no-op stub `purchasing.NotifyVendorComplete(ctx, pool, listID)` that logs "alert pending — Phase 17" and returns nil.
- **Shopping list as live JOIN:** Snapshot at approval time (Pattern 2). Never query live from po_line_items.
- **Blocking approve button during TX:** Approve creates the shopping list snapshot in ~10ms (O(n) inserts for n items). No async needed.
- **Admin unlock after approval:** D-13 explicitly blocks this. Handler must check: if status is 'shopping_active' or 'completed', return 409 with "cannot unlock after approval".

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Timezone-aware cron | Custom UTC arithmetic | `time.LoadLocation` + compute next occurrence | Go stdlib handles DST correctly; 10 lines beats a dependency |
| Photo upload | Custom S3 client | `photos.PresignUploadHandler` (already exists) | Reuse existing presigned PUT with path_prefix "shopping" |
| Toast UI component | Custom notification system | Adapt existing `#toast` / `#sync-toast` from workflows.html | Same CSS variables, same mobile UX |
| Role check | Per-handler ad-hoc string comparison | `isAdmin(user)` helper in purchasing package | Single definition, consistent across all purchasing handlers |
| Optimistic lock | Retry loop | `WHERE id = $1 AND version = $v RETURNING id` → 0 rows = 409 | One SQL clause, no loop needed |

---

## Common Pitfalls

### Pitfall 1: DST Skip / Double-Fire on Cutoff
**What goes wrong:** Scheduler fires at wrong wall-clock time after DST transition if cutoff is computed as a fixed UTC offset.
**Why it happens:** `time.Now().UTC()` doesn't know about local DST shifts.
**How to avoid:** Use `time.LoadLocation(timezone)` to compute the next cutoff occurrence in local time, then convert to UTC for comparison. Never store cutoff as a UTC offset.
**Warning signs:** Cutoff time stored as `INTEGER utc_hour` or computed with `time.FixedZone`.

### Pitfall 2: Cutoff Race — Edit Accepted After Deadline
**What goes wrong:** Crew opens PO 30s before cutoff. Guard check passes. Cutoff fires. Crew submits. Guard already evaluated — write goes through on locked PO.
**Why it happens:** Go guard before DB write has a time gap.
**How to avoid:** Include status check inside the UPDATE WHERE clause: `WHERE id = $1 AND status = 'draft' AND version = $2`. Return 409 if 0 rows updated.
**Warning signs:** `if status != 'draft' { return 403 }` before `db.Exec(UPDATE)`.

### Pitfall 3: State Drift — Admin Unlocks After Shopping List Created
**What goes wrong:** Admin unlocks an approved PO (after the shopping list exists), edits quantities, re-locks. Shopper's list shows old snapshot quantities.
**How to avoid:** D-13 is locked — block unlock if status is 'shopping_active' or 'completed'. Return 409 with "cannot unlock after approval".
**Warning signs:** No shopping list existence check before allowing unlock.

### Pitfall 4: `purchase_orders` CHECK Constraint Blocks New States
**What goes wrong:** Migration 0034 has `CHECK (status IN ('draft', 'locked', 'approved'))`. Inserting 'shopping_active' or 'completed' fails with a DB constraint error.
**How to avoid:** Migration 0037 must ALTER the CHECK constraint or drop and recreate it to include all five status values.
**Warning signs:** Unit test for approve passes but vendor complete throws a DB constraint violation.

### Pitfall 5: Vendor Grouping Without vendor_id on purchase_items
**What goes wrong:** D-09 requires items grouped by vendor on the locked PO view and shopping list. Current schema has no vendor association on `purchase_items` — only on `purchase_events`.
**How to avoid:** Migration 0039 adds `vendor_id UUID REFERENCES vendors(id)` to `purchase_items`. The Setup tab's item edit form in inventory.html must expose a vendor selector so items can be assigned.
**Warning signs:** PO tab shows all items in one flat list with no vendor sections.

### Pitfall 6: Shopping List Items Show Stale Photo/Location
**What goes wrong:** Shopping list snapshot stores a reference to `po_line_item_id` but the photo comes from `purchase_items.photo_url`. If the photo is updated after approval, the shopper sees the new photo — correct behavior. But if the item is deleted, the FK breaks.
**How to avoid:** Snapshot `photo_url` and `store_location` directly into `shopping_list_items` at approval time. Use these columns for display. SHOP-08 (upload photo from shopping list) updates `purchase_items.photo_url` directly so all future lists see the new photo too.
**Warning signs:** `shopping_list_items` has no `photo_url` or `store_location` column — relying on JOIN to purchase_items.

---

## Code Examples

Verified patterns from live codebase:

### Locking a PO (optimistic lock pattern)

```go
// Source: service.go (pattern), migration 0034 (version column)
func LockPO(ctx context.Context, pool *pgxpool.Pool, poID string) error {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil { _ = tx.Rollback(ctx) }
    }()

    var id string
    err = tx.QueryRow(ctx, `
        UPDATE purchase_orders
        SET status = 'locked', locked_at = now(), version = version + 1
        WHERE id = $1 AND status = 'draft'
        RETURNING id
    `, poID).Scan(&id)
    if errors.Is(err, pgx.ErrNoRows) {
        return ErrPONotDraft
    }
    if err != nil {
        return err
    }

    // Create next week's draft
    nextWeek := nextMondayAfter(poID) // compute from current week_start + 7 days
    _, err = tx.Exec(ctx, `
        INSERT INTO purchase_orders (week_start, status)
        VALUES ($1, 'draft')
        ON CONFLICT (week_start) DO NOTHING
    `, nextWeek)
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### Presigned upload reuse (SHOP-08)

```go
// Source: backend/internal/photos/handler.go — PresignUploadHandler
// Caller pattern: POST /api/v1/photos/presign
// Body: { "path_prefix": "shopping", "id": "<shopping_list_id>", "filename": "item_abc123.jpg" }
// Returns: { "url": "<presigned PUT>", "object_key": "...", "public_url": "..." }
// After upload: PUT /api/v1/purchasing/shopping/{id}/items/{item_id}/photo
//   → UPDATE purchase_items SET photo_url = $publicURL WHERE id = $purchaseItemID
```

### Role check pattern (admin-only endpoints)

```go
// Source: backend/internal/auth/service.go — User.Roles []string, User.IsSuperadmin bool
func requireAdmin(user *auth.User) bool {
    if user == nil { return false }
    if user.IsSuperadmin { return true }
    for _, r := range user.Roles {
        if r == "admin" { return true }
    }
    return false
}
```

### Scheduler pattern (mirrors receipt.StartWorker)

```go
// Source: backend/internal/receipt/worker.go — StartWorker pattern
func StartScheduler(ctx context.Context, pool *pgxpool.Pool) {
    log.Println("cutoff scheduler: starting (15m tick)")
    go func() {
        runCutoffCheck(ctx, pool)
        ticker := time.NewTicker(15 * time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                log.Println("cutoff scheduler: shutting down")
                return
            case <-ticker.C:
                runCutoffCheck(ctx, pool)
            }
        }
    }()
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Mock tabs in purchasing.html (stubs) | Real API-backed tabs | Phase 16 | s2 (Shopping), s3 (PO), s4 (History) need to be wired |
| purchase_orders CHECK (draft\|locked\|approved) | Extend to include shopping_active, completed | Phase 16 migration | Must ALTER constraint in 0037 migration |
| No vendor association on purchase_items | vendor_id FK to vendors table | Phase 16 migration | Enables per-vendor grouping in PO tab and shopping list |

---

## Open Questions

1. **Vendor grouping — is vendor_id on purchase_items the right model?**
   - What we know: `purchase_items` currently has no `vendor_id`. Vendors exist on `purchase_events` (receipts), not catalog items. The CONTEXT says D-09 requires vendor grouping.
   - What's unclear: A single catalog item (e.g., "Salmon fillet") could theoretically be purchased from multiple vendors over time. Assigning a single `vendor_id` to `purchase_items` implies "primary vendor."
   - Recommendation: Add `vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL` as nullable. Items with no vendor assigned show in an "Unassigned" section. The Setup tab can expose a vendor selector.

2. **Simulate-cutoff endpoint (D-04) — does it lock the CURRENT week's draft or a specific PO by ID?**
   - What we know: D-04 says `POST /api/v1/purchasing/simulate-cutoff`. No `{id}` in the path suggests it locks the current week's draft.
   - Recommendation: Lock whatever the current week's draft is (same logic as the scheduler). Admin can only simulate once per week — to test again, they truncate the DB per D-05.

3. **`purchase_orders` status CHECK constraint — drop or ALTER?**
   - What we know: Migration 0034 has `CHECK (status IN ('draft', 'locked', 'approved'))`.
   - Recommendation: Migration 0037 should `ALTER TABLE purchase_orders DROP CONSTRAINT purchase_orders_status_check` then `ADD CONSTRAINT purchase_orders_status_check CHECK (status IN ('draft', 'locked', 'approved', 'shopping_active', 'completed'))`. Postgres requires the exact constraint name — verify with `\d purchase_orders` or check goose Down section.

4. **Shopping list RBAC — who can see the Shopping tab?**
   - What we know: SHOP-02 says "assignable to specific members or roles." The CONTEXT doesn't lock a specific RBAC model.
   - Recommendation: All authenticated users can see the Shopping tab. Server-side filter returns only lists assigned to the current user's role or user_id. Admin can see all lists.

5. **History tab — shows shopping lists or POs?**
   - What we know: D-25 says "past shopping lists." D-26 says "tappable to expand and see full item list."
   - Recommendation: History tab (`s4`) shows `shopping_lists` ordered newest first with `week_start` label from the parent PO. Each entry is collapsible/expandable.

---

## Environment Availability

Step 2.6: Phase 16 is code/config changes extending existing infrastructure. No new external services. DO Spaces is already configured for photo uploads. SKIPPED.

---

## Sources

### Primary (HIGH confidence)
- Live codebase reads:
  - `backend/internal/purchasing/service.go` — existing PO service patterns
  - `backend/internal/purchasing/handler.go` — existing handler conventions
  - `backend/internal/purchasing/types.go` — existing type definitions
  - `backend/internal/photos/handler.go` — DO Spaces presign pattern
  - `backend/internal/receipt/worker.go` — scheduler goroutine pattern
  - `backend/internal/auth/service.go` — User struct, Roles, IsSuperadmin
  - `backend/cmd/server/main.go` — route registration, worker registration
  - `backend/internal/db/migrations/0034_purchase_orders.sql` — current schema
  - `backend/internal/db/migrations/0035_purchase_items_photo_store.sql` — photo_url, store_location columns
  - `purchasing.html` — existing 4-tab layout with stubs
- `.planning/research/ARCHITECTURE.md` — snapshot pattern, scheduler model, state machine
- `.planning/research/PITFALLS.md` — cutoff race, DST bugs, state drift after unlock

### Secondary (MEDIUM confidence)
- `.planning/phases/16-cutoff-approval-and-shopping-list/16-CONTEXT.md` — locked decisions D-01 through D-26

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages already in use, no new dependencies
- Architecture: HIGH — derived from live code + prior verified research
- Pitfalls: HIGH (race conditions, DST) — prior research documents verified against codebase; MEDIUM (vendor grouping gap) — inferred from schema read

**Research date:** 2026-04-22
**Valid until:** 2026-05-22 (stable Go codebase, no fast-moving ecosystem dependencies)
