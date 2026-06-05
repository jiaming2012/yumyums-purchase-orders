# Phase 16: Cutoff, Approval, and Shopping List - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the weekly cutoff scheduler that auto-locks the PO, admin approval flow that generates an immutable shopping checklist, and the shopping tab where crew checks off items by vendor with photo/location capture prompts. Includes the PO tab (locked view), Shopping tab (active checklist), and History tab (past lists).

</domain>

<decisions>
## Implementation Decisions

### Cutoff Config UX
- **D-01:** Admin configures cutoff by tapping the cutoff pill on the PO tab header — expands inline to show day dropdown + time dropdown + Save/Cancel
- **D-02:** Non-admin users see the cutoff pill as read-only
- **D-03:** Cutoff stored in a `cutoff_config` table (day_of_week, time, timezone) — one row, upserted

### Cutoff Test Simulation
- **D-04:** Uses the same production endpoint: `POST /api/v1/purchasing/simulate-cutoff` — triggers the real lock logic (locks current draft, creates new draft for next week)
- **D-05:** Admin-only endpoint. To test again, user truncates the DB and re-seeds
- **D-06:** No separate test path — same code path as the scheduler

### PO Tab (Locked State)
- **D-07:** Photo card layout (same as Order tab) but steppers disabled for non-admin crew
- **D-08:** Admin sees an "Edit ✎" toggle to unlock steppers for editing within locked state
- **D-09:** Items grouped by vendor with estimated totals per vendor (like current mock PO tab)
- **D-10:** "Approve & Create Shopping List" button at bottom — admin-only
- **D-11:** Approve blocked if previous week's shopping list is still active — shows message: "Complete last week's shopping list before approving this one"

### State Machine
- **D-12:** PO lifecycle: draft → locked → approved → shopping_active → completed
- **D-13:** Admin can unlock a locked PO back to draft (e.g. cutoff was premature) — but NOT after approval
- **D-14:** On cutoff: current draft transitions to locked, new draft auto-created for next week
- **D-15:** On approve: shopping list snapshot created (immutable), PO status → approved, shopping list status → active
- **D-16:** On all vendor sections completed: shopping list status → completed, PO status → completed

### Shopping Check-off
- **D-17:** Toast + badge pattern for missing photo/location: item checks off immediately, toast appears "Lemons needs photo & location [Add Now]", badge shows "⚠ No photo" on the item card
- **D-18:** User can dismiss toast (item stays checked off without photo/location) or tap "Add Now" to open photo capture + location input
- **D-19:** No skip confirmation required — toast is non-blocking, badge persists as a gentle reminder
- **D-20:** Photo capture uses same DO Spaces presigned PUT pattern from Phase 12

### Per-Vendor Completion
- **D-21:** Each vendor section has its own status (pending/completed) and its own "Complete [Vendor]" button
- **D-22:** Completing a vendor section records who completed it and when
- **D-23:** Unchecked items in a completed vendor section are flagged as "missing" — reported in the completion alert (Phase 17)
- **D-24:** Overall shopping list transitions to "completed" only when ALL vendor sections are done

### History Tab
- **D-25:** Shows past shopping lists with: week label, vendor breakdown, missing items count, who completed each vendor section
- **D-26:** Tappable to expand and see full item list with checked/unchecked status

### Claude's Discretion
- Scheduler implementation (goroutine ticker vs gocron — research recommended)
- Shopping list snapshot schema design
- History tab sort order (newest first assumed)
- Animation on approve button
- Toast auto-dismiss timing

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 14 purchasing infrastructure
- `backend/internal/purchasing/service.go` — Existing PO service (get-or-create, upsert items, suggestions)
- `backend/internal/purchasing/handler.go` — Existing purchasing handlers (4 endpoints)
- `backend/internal/purchasing/types.go` — PurchaseOrder, POLineItem, OrderSuggestion types
- `purchasing.html` — Current 4-tab layout with Order tab interactive, 3 stub tabs

### Photo upload pattern
- `backend/internal/photos/handler.go` — DO Spaces presigned PUT pattern for photo upload (reuse for shopping photos)

### Research
- `.planning/research/ARCHITECTURE.md` — Shopping list as immutable snapshot, cutoff enforcement in WHERE clause
- `.planning/research/PITFALLS.md` — Optimistic locking via version column, timezone handling, cutoff race conditions

### Prior context
- `.planning/phases/14-po-backend-order-form/14-CONTEXT.md` — D-10 through D-16 (tab structure, per-vendor submit, history attribution)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **PO service**: `GetOrCreateOrder`, `UpsertLineItems`, `GetSuggestions` — extend with lock/approve/unlock
- **Photo upload**: `photos.PresignUploadHandler` — reuse for shopping list photo capture
- **Fullscreen lightbox**: Already in purchasing.html — reuse for shopping list photo view
- **Toast pattern**: Existing toast in workflows.html (`#toast` / `#sync-toast`) — adapt for shopping check-off prompts

### Established Patterns
- **State-first rendering**: Mutate state → render → DOM
- **Event delegation**: Single click listener with `data-action` routing
- **Optimistic locking**: `version` column on `purchase_orders` (from migration 0034)

### Integration Points
- `purchase_orders.status` transitions (draft → locked → approved)
- New `shopping_lists` + `shopping_list_items` tables
- New `cutoff_config` table
- Scheduler goroutine in `main.go` (like receipt worker pattern)

</code_context>

<specifics>
## Specific Ideas

- The cutoff pill is already rendered in purchasing.html ("Cutoff Sun 6pm") — make it tappable for admin
- Shopping list is a snapshot: copies PO line items at approval time, not a live view
- Vendor grouping in PO tab matches the current mock layout (vendor name + total on header, items below)
- "Complete [Vendor]" button style should match the existing "Submit additions" / "Add Selected" blue button pattern

</specifics>

<deferred>
## Deferred Ideas

- Alert notifications (cutoff reminder, out-of-stock, missing items on complete) — Phase 17
- Repurchase badges on inventory — Phase 17
- Email delivery — Phase 17

</deferred>

---

*Phase: 16-cutoff-approval-and-shopping-list*
*Context gathered: 2026-04-22*
