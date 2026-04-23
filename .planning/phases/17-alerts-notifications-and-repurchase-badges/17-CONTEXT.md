# Phase 17: Alerts, Notifications, and Repurchase Badges - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver async alert notifications for cutoff reminders, out-of-stock items, and shopping list completion reports via Zoho Cliq incoming webhook or email. Add user notification preference configuration to the Users tab. Show repurchase badges on inventory Stock tab items after shopping completion, with admin-configurable reset schedule.

</domain>

<decisions>
## Implementation Decisions

### Alert Delivery
- **D-01:** Zoho Cliq incoming webhook is the primary delivery channel; email is the fallback alternative
- **D-02:** Zoho Cliq integration uses a single incoming webhook URL configured at the app/admin level (not per-user)
- **D-03:** Email delivery uses a simple SMTP or transactional email service (e.g., SendGrid, Mailgun)
- **D-04:** Alert queue is async — alerts are enqueued and delivered in the background, not blocking user actions

### Notification Preferences
- **D-05:** Each user has a notification preference field: "zoho_cliq" (default) or "email"
- **D-06:** At least one channel is required — users cannot disable all notifications
- **D-07:** Preference is configured in the Users tab (admin edits for all users, user edits for self)

### Cutoff Reminders
- **D-08:** Single reminder sent 24 hours before the configured cutoff time
- **D-09:** Reminder goes to all users with "order" permission (crew who can add to PO)
- **D-10:** Reminder message includes: week label, current item count, cutoff day/time

### Shopping Completion Alerts
- **D-11:** When a shopping list is fully completed, an alert is sent listing any missing/unchecked items (per Phase 16 D-23)
- **D-12:** Alert goes to admin(s) and the shopper who completed the list
- **D-13:** Alert includes: vendor name, missing item names, quantities, who completed it

### Repurchase Badges
- **D-14:** Inventory Stock tab items show a "Repurchased +[Qty]" badge after the shopping list containing that item is completed
- **D-15:** Badge quantity comes from the shopping list item's quantity (not PO quantity)
- **D-16:** Admin can configure a reset date/schedule (weekly, matching cutoff cycle)
- **D-17:** Badge resets for all items on the configured reset date

### Claude's Discretion
- Alert queue implementation (in-process goroutine channel vs external queue)
- Email template formatting
- Zoho Cliq message card formatting (plain text vs rich card)
- Database schema for alert history/log

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 16 Context
- `.planning/phases/16-cutoff-approval-and-shopping-list/16-CONTEXT.md` — D-23 defines missing items behavior on completion

### Requirements
- `.planning/REQUIREMENTS.md` §Alerts & Notifications — ALRT-01 through ALRT-06
- `.planning/REQUIREMENTS.md` §Repurchase Tracking — REP-01, REP-02

### Existing Code
- `backend/internal/purchasing/service.go` — CompleteVendorSection, shopping list completion logic
- `backend/internal/purchasing/scheduler.go` — Cutoff scheduler (15m tick) — extend for reminder alerts
- `backend/internal/users/db.go` — User model, role queries
- `inventory.html` — Stock tab rendering (badge insertion point)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `purchasing/scheduler.go`: Existing cutoff scheduler with 15-minute tick — can be extended for reminder checks
- `users/db.go`: User queries with role filtering — reuse for "who gets notified"
- `inventory/handler.go`: GetStockHandler — badge data can be added to stock response
- `photos/spaces.go`: DO Spaces client pattern — similar HTTP client pattern for Zoho webhook

### Established Patterns
- Background goroutines for periodic tasks (scheduler.go)
- `api()` wrapper in frontend for authenticated requests
- Badge rendering in inventory.html (stock-badge CSS class)
- Toast pattern in purchasing.html for transient notifications

### Integration Points
- Shopping list completion (CompleteVendorSection) — trigger completion alert
- Cutoff scheduler tick — add reminder check
- Users tab — add notification preference field
- Inventory Stock tab — add repurchase badge to stock items
- GetStockHandler response — add repurchase data

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches for webhook delivery and badge rendering.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 17-alerts-notifications-and-repurchase-badges*
*Context gathered: 2026-04-22*
