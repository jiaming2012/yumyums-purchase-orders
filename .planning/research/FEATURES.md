# Feature Research

**Domain:** Food service purchase order and shopping list workflows (food truck operations, v3.0 milestone)
**Researched:** 2026-04-22
**Confidence:** MEDIUM-HIGH — Standard food service procurement patterns verified via MarketMan/BlueCart feature analysis. Zoho Cliq webhook API verified via official docs. Cutoff enforcement, badge, and shopping list patterns inferred from existing mockup, industry norms, and current codebase review.

## Context: What Already Exists

This is a subsequent milestone. Do not re-build what is already shipped.

**Already built (do not redesign):**
- Inventory Stock tab with par-level thresholds (low/high per group), reorder suggestions, stock count overrides with required reason
- Item catalog: items, groups, tags, vendor assignments, item photos
- Receipt ingestion pipeline (Mercury banking → AI parse → pending review → confirm)
- `purchasing.html` mockup: 3-tab layout (Form / Locked / PO), stepper qty inputs, category grouping, "Cutoff Sun 6pm" pill, locked view showing contributor initials, PO breakdown by vendor
- User roles (superadmin, admin, manager, member) with RBAC on all app tiles
- Fullscreen picker modal pattern (established in inventory item linking)
- Stock count overrides table with configurable reset date

**v3.0 goal:** Connect the purchasing mockup to real inventory data and add the full loop: PO form → cutoff lock → admin approval → shopping checklist → completion alert → repurchase badge.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| PO form pre-populated from inventory reorder suggestions | Every food service procurement tool (MarketMan, BlueCart) auto-populates from par levels — a blank form feels like going backward | MEDIUM | Pull items where `current_stock < low_threshold` from existing Stock tab data; `suggested_qty = low_threshold - current_stock`; items grouped by catalog group (Proteins, Produce, etc.) — same categories as purchasing.html mockup |
| Item search / add from catalog | Users need to add non-reorder items (opportunistic buys, seasonal items, forgotten items) | LOW | Fullscreen picker modal — already established pattern in inventory item linking; search by name/group/tag against existing item catalog |
| Item photo + store location on PO form row | Crew needs visual confirmation they are buying the right thing at the right store; reduces wrong-item purchases | LOW | Item photo and vendor/location already exist in catalog; surface both on each PO row — photo thumbnail left, item name + vendor right |
| Stepper qty input on each PO item | Touch-optimized quantity adjustment — established in existing mockup, crew expects it | LOW | Already in purchasing.html mockup; wire to real data with suggested qty as the starting value |
| Cutoff enforcement — form locks after deadline | Food truck ordering has a hard supplier deadline; late changes break the vendor order | MEDIUM | `status` field on PO: `draft` → `locked` → `approved`; backend cron job transitions `draft` → `locked` when current time passes configured cutoff; non-admin users see read-only view after lock |
| Admin-only edit after cutoff | Owner needs escape hatch for emergency corrections or last-minute items | LOW | Role check on edit endpoint (`admin` or `superadmin`); UI shows edit controls only for qualifying roles; admin can force `locked` → `draft` with a reason |
| Weekly cutoff schedule configurable by admin | "Sunday 6pm" is hardcoded in the mockup pill — production requires self-serve config | LOW | Admin settings: day-of-week picker + time; stored in backend config table; cron scheduler reads this on startup and after config change |
| PO approval by admin generates shopping checklist | Approval is the bridge between "what to order" and "who goes shopping" — without it, there is no execution record or accountability | HIGH | `POST /api/v1/purchasing/orders/{id}/approve` creates `shopping_list` + `shopping_list_items` records from PO line items; shopping list inherits `item_id`, `qty`, `vendor_id`, `location_notes` |
| Shopping list item check-off | Crew needs to mark items as bought while shopping; this is the core execution UX | LOW | Checkbox per row; persist immediately (same pattern as workflow checklist field responses); tap again to uncheck |
| Shopping list assignability | Someone must own the shopping run — ambiguity = nobody goes | LOW | Assign to role or specific user; same RBAC model as workflow checklists (existing pattern) |
| Missing items report on shopping completion | When shopping is done, owner needs to know what could not be sourced | MEDIUM | "Complete shopping" button shows summary of unchecked items; user adds a note per missing item before confirming; creates `shopping_completion` record with `missing_items` array |
| Cutoff reminder alert | Staff forget deadlines; reminder must arrive before the window closes, not after | MEDIUM | Scheduled job fires N hours before cutoff (configurable; default 2 hours); sends to configured notification channel |
| Out-of-stock / low-stock alert | Owner needs to know when something is critically low before the PO window even opens | MEDIUM | Trigger when `current_stock < low_threshold` (threshold already exists per group); alert sent to configured channel; de-duplicate (don't alert on every stock check — once per transition below threshold) |
| Zoho Cliq channel notification delivery | Zoho Cliq is the team's primary communication tool — alerts must arrive there, not in a buried in-app notification | MEDIUM | POST to channel webhook: `https://cliq.zoho.com/api/v2/channelsbyname/{channel}/message?zapikey={token}`; JSON payload `{"text": "..."}` (MEDIUM confidence — official docs confirm endpoint format but note outgoing webhooks are deprecated; incoming webhook tokens remain supported for posting) |
| Email as fallback notification channel | Not everyone opens Cliq; email is universally accessible and expected as a backup | LOW | Standard SMTP via Go `net/smtp` or transactional email service (Resend/Postmark already used for invite emails in v2.0); user sets preference in Users tab |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Reorder suggestion tap-to-add at top of PO form | MarketMan auto-populates; our version surfaces suggestions explicitly and lets crew tap to accept or adjust — keeps human judgment in the loop while providing speed | LOW | Suggestions list as top section of Form tab; "Add all" CTA + individual tap-to-add per item; items not yet in PO show "Add" button, added items show stepper |
| Contributor attribution on locked PO view | "3 contributors · 14 items" — owner sees who added what; accountability without surveillance | LOW | Track `added_by` on each PO line item; show initials in locked view (already in purchasing.html mockup as "JM MK TR") |
| Item location notes editable inline on shopping list | Crew can update "aisle 7" or "ask at butcher counter" notes while shopping — institutional knowledge lives in the app | LOW | Inline tap-to-edit text field per shopping list item; same pattern as stock group threshold inline editing |
| Shopping list grouped by store / vendor | Items from different vendors are bought at different stores; grouping by store reduces backtracking during the shopping run | MEDIUM | Group shopping list items by `vendor_id`; collapsible sections per store; same collapse/expand pattern as inventory Stock tab group sections |
| "Repurchased +[Qty]" badge on inventory Stock items | Closes the loop between purchasing and inventory; makes it immediately visible that an item was just restocked | MEDIUM | After shopping list is confirmed, write to `stock_count_overrides` with reason "Repurchased" and delta qty; badge = any override with reason "Repurchased" within reset window; existing `COALESCE(override, sum)` query already picks up overrides automatically |
| Per-user notification preference config | Users choose Cliq, email, or both (at least one required); prevents alerts going to the wrong channel | LOW | Notification preference field in Users tab (extending existing user profile); at least one channel required (validation); Zoho Cliq is the default |
| Backend cutoff simulation command | Developer and owner need to test the lock/unlock flow without waiting a week for real Sunday 6pm | LOW | `POST /api/v1/purchasing/simulate-cutoff` with admin auth; triggers the same cron job logic immediately; essential for QA and owner demos |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Auto-submit PO to vendor via API or EDI | Seems like efficiency win — skip the manual call to the supplier | Supplier APIs vary wildly; EDI integration is a 3-6 month project; errors are hard to recover from; food truck at this scale should maintain a human touchpoint with vendors | Lock the PO and show a vendor summary view (already in PO tab of mockup); owner manually places order using the summary as reference — this is the right level of automation for a 1-5 person operation |
| Barcode scanning for shopping list receive | "Scan items when they arrive" sounds fast | Camera-based barcode scanning in a PWA is unreliable without a native scanner; requires a UPC food database; wrong scans are hard to detect | Shopping list check-off is the right UX here: crew taps each item as they put it in the cart, not after delivery; existing receipt ingestion pipeline handles receiving acknowledgment |
| Automatic inventory deduction on PO approval | Avoids manual stock counting | Theoretical deduction diverges from reality quickly (wrong deliveries, partial orders, supplier substitutions); creates false confidence in stock levels | Keep existing model: stock is updated through confirmed receipts and manual overrides with reason; "Repurchased +[Qty]" badge is informational context, not a stock count mutation |
| PO comment threads / in-app discussion | "We need to discuss the order" | Adds significant complexity (threading, notifications per comment, read states); team already has Zoho Cliq for this | Send PO summary link to Cliq channel at cutoff and approval; discussion happens in Cliq, decisions flow back into the PO form |
| Multi-location / multi-store PO | Owner asks about expansion or running two stops | Single food truck is the stated constraint; adding `location_id` to every PO table now creates schema complexity with zero current value | Document as explicitly out-of-scope with "add in v4.x when second truck is operational" |
| Inventory auto-reorder (fully automatic PO creation and send) | Zero-touch ordering sounds appealing | Removes human oversight from food ordering; wrong quantities can waste hundreds of dollars of perishable inventory; not appropriate at 1-5 person scale | Auto-populate suggestions on the PO form with one-tap accept — same speed benefit with human judgment retained |
| Notion data import as a feature | "Seed the catalog from our Notion database (~100 items)" | A one-time import is an operation, not a feature; building a Notion sync UI adds ongoing complexity with no ongoing value | Run a one-shot Go migration script against the Notion API at the start of the milestone to seed item catalog; not a recurring feature |

---

## Feature Dependencies

```
[Inventory item catalog + stock levels + vendor assignments]   ← already exists
    └──required by──> [PO form with reorder suggestions + item search]
                          └──required by──> [Cutoff enforcement (draft → locked)]
                                               └──required by──> [Admin approval flow]
                                                                     └──required by──> [Shopping list generation]
                                                                                           └──required by──> [Shopping list check-off + completion]
                                                                                                                └──required by──> [Missing items alert]
                                                                                                                └──required by──> [Repurchased badge on inventory items]

[Admin-configurable cutoff schedule]
    └──required by──> [Cutoff enforcement] (cron job needs day/time config)
    └──required by──> [Cutoff reminder alert] (scheduler needs to know when to fire)

[Notification channel config (Zoho Cliq webhook URL, SMTP settings)]
    └──required by──> [Cutoff reminder alert]
    └──required by──> [Out-of-stock alert]
    └──required by──> [Missing items alert on shopping completion]

[Per-user notification preference in Users tab]
    └──enhances──> [All alert routing] (determines which channel each user gets alerted on)

[Backend cutoff simulation command]
    └──enables──> [QA and testing of entire lock/approve/shopping flow without waiting for real cutoff]
```

### Dependency Notes

- **PO form requires existing inventory catalog.** Item photos, store locations, par levels, vendor assignments all live in the current catalog/inventory data model. PO form is a projection of that data — not a new data source.
- **Shopping list requires approved PO.** Shopping list is a derived artifact created at approval time. It cannot be created independently. Approval is the write trigger.
- **Repurchased badge requires shopping list completion.** Badge is written when a shopping list is marked complete (actual purchase confirmed), not at PO approval time (planned purchase). This ensures the badge reflects what was actually bought.
- **All alerts require notification channel config.** Cliq webhook URL or SMTP must be configured before any alert fires. This is a backend config, not a user setting, except for per-user channel preference which routes the alert.
- **Cutoff enforcement requires configurable schedule.** "Sunday 6pm" hardcoded in the mockup is fine for prototype only. The cron job must read from a config table to be production-ready.
- **Backend simulation command is a development dependency.** Required before QA can test the lock/approval/shopping flow without waiting for the real weekly cutoff.

---

## MVP Definition

This is a subsequent milestone. MVP means the minimum needed to close the ordering-to-shopping loop end-to-end.

### Launch With (v3.0 — close the loop)

- [ ] PO form backed by real inventory data — reorder suggestions pre-loaded, stepper qty, item photos, store locations
- [ ] Item search / add from catalog (fullscreen picker modal)
- [ ] Admin-configurable weekly cutoff schedule
- [ ] Cutoff enforcement — backend lock after configured day/time; admin edit override
- [ ] PO approval by admin generates shopping checklist with 1:1 line items
- [ ] Shopping list: item check-off, store location notes, assignable to role/user
- [ ] Shopping list completion with missing items note
- [ ] Zoho Cliq channel alert on shopping completion (missing items report)
- [ ] Cutoff reminder alert (N hours before deadline, sent to Cliq channel)
- [ ] "Repurchased +[Qty]" badge on inventory Stock tab items after shopping confirmed
- [ ] Backend cutoff simulation command (admin-only, for QA)

### Add After Validation (v3.x)

- [ ] Out-of-stock / low-stock alerts — add after owner confirms cutoff reminders are useful; validate alert fatigue first
- [ ] Email as fallback notification channel — add only if owner reports missing Cliq alerts on mobile
- [ ] Shopping list grouped by store — add when crew reports backtracking pain during shopping runs
- [ ] Per-user notification preference config — add when team size grows beyond owner + 1 crew member

### Future Consideration (v4+)

- [ ] Notion item catalog import — run as a one-shot Go migration script before v3.0 launch; not a recurring feature
- [ ] PO history / spending trend by PO (extend existing Chart.js spending trends in Trends tab)
- [ ] Multi-location PO when second truck is operational
- [ ] Vendor EDI / API integration (if supplier relationships justify it at scale)

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| PO form from inventory data | HIGH | MEDIUM | P1 |
| Admin-configurable cutoff schedule | HIGH | LOW | P1 |
| Cutoff enforcement + admin lock | HIGH | MEDIUM | P1 |
| Item search / add from catalog | MEDIUM | LOW | P1 |
| PO approval → shopping list generation | HIGH | HIGH | P1 |
| Shopping list check-off + completion | HIGH | LOW | P1 |
| Missing items report + Cliq alert | HIGH | MEDIUM | P1 |
| Cutoff reminder alert | HIGH | MEDIUM | P1 |
| Repurchased badge on inventory | MEDIUM | MEDIUM | P1 |
| Backend cutoff simulation command | MEDIUM | LOW | P1 |
| Contributor attribution (locked view) | LOW | LOW | P2 |
| Shopping list grouped by store | MEDIUM | MEDIUM | P2 |
| Out-of-stock alerts | MEDIUM | LOW | P2 |
| Email fallback notification | LOW | LOW | P2 |
| Per-user notification preference | MEDIUM | LOW | P2 |
| Item location notes inline edit | LOW | LOW | P3 |
| PO history spending trends | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for v3.0 launch — required to close the ordering-to-shopping loop
- P2: Add post-launch when triggered by real usage or owner request
- P3: Future consideration; deferred until product-market fit on v3.0 features

---

## Competitor Feature Analysis

| Feature | MarketMan | BlueCart | Yumyums v3.0 Approach |
|---------|--------------|--------------|--------------|
| PO generation | Auto-generated from par levels; "fill to par" one-click | Auto-generated from par/sales data | Show suggestions explicitly, tap to accept — keeps human judgment; appropriate for 1-5 person operation |
| Shopping / receiving | Digital receiving checklist with check-off at delivery | Integrated with supplier catalog | Shopping checklist check-off while shopping (pre-delivery); separate from receipt ingestion (post-delivery) |
| Cutoff enforcement | Purchasing budgets + price limits; user permissions gate submission | Par-based auto-order; no explicit cutoff | Explicit weekly cutoff with admin lock; configurable day/time; maps to real food truck supplier ordering rhythm |
| Alerts | Alerts for shorts, subs, credits, billing irregularities | Low-stock auto-reorder triggers | Zoho Cliq channel webhook + email; cutoff reminders + missing items + out-of-stock |
| Vendor consolidation | Multi-vendor, centralized ordering UI | Supplier marketplace | PO tab already groups by vendor; shopping list grouped by store in v3.x |
| Mobile UX | Web + mobile app (not PWA) | Web-focused | Mobile-first PWA, touch-optimized, 480px — better field UX than competitors for crew on phones |

---

## Implementation Notes

### PO Form Data Source
Pull from existing `GET /api/v1/inventory/stock` endpoint (already returns items with current stock, par thresholds, group, vendor). Reorder suggestions = items where `current_stock < low_threshold`. PO form pre-loads with `suggested_qty = max(1, low_threshold - current_stock)`.

### Cutoff Enforcement
Add `purchasing_config` table: `cutoff_day` (0-6, Sun=0), `cutoff_hour`, `cutoff_minute`. Add `purchase_orders` table with `status` enum (`draft`, `locked`, `approved`). Backend cron job (1-minute tick) transitions all `draft` POs to `locked` when current weekday + time matches config. Admin can revert via `PATCH /api/v1/purchasing/orders/{id}/status` with `{"status": "draft", "reason": "..."}`.

### Shopping List Generation
`POST /api/v1/purchasing/orders/{id}/approve` creates `shopping_lists` + `shopping_list_items` from PO line items. Shopping list inherits `item_id`, `qty_planned`, `vendor_id`, `location_notes`. Status: `open` → `complete`.

### Repurchased Badge
After `POST /api/v1/purchasing/shopping-lists/{id}/complete`, write `stock_count_overrides` row per purchased item with `reason = "Repurchased"` and `qty = qty_purchased`. Existing `COALESCE(override, sum)` stock query picks this up automatically. Badge rendered in Stock tab = any override with `reason = "Repurchased"` within the last N days (configurable, default 7).

### Zoho Cliq Webhook
```
POST https://cliq.zoho.com/api/v2/channelsbyname/{channel_unique_name}/message?zapikey={token}
Content-Type: application/json

{"text": "Shopping complete. Missing: Salmon fillet (2 lb), Shrimp 16/20 (4 lb)."}
```
Token and channel name stored as backend environment variables. Go implementation: `http.NewRequest("POST", url, bytes.NewBuffer(payload))` with `Content-Type: application/json` header. No Go SDK needed — plain HTTP POST.

### Email Fallback
Reuse existing SMTP/transactional email infrastructure from v2.0 invite flow (Resend or Postmark). Alert email is plain text. Template: subject "Yumyums: [alert type]", body = same text as Cliq message.

---

## Dependencies on Existing v2.0 Features

| Existing Feature | How v3.0 Depends On It |
|-----------------|------------------------|
| Inventory item catalog (items, groups, vendors, photos) | PO form data source — all item metadata lives here |
| Stock levels + low/high thresholds per group | Reorder suggestion logic — `current_stock < low_threshold` triggers suggestion |
| Stock count overrides table + reset date | Repurchased badge writes here; existing COALESCE query picks up badge data automatically |
| Fullscreen picker modal pattern | Item search / add in PO form reuses this exact UX pattern |
| User roles (admin, manager, member) | PO approval gated to admin+; shopping list assignment uses existing role model |
| SMTP / transactional email (invite flow) | Email fallback notification reuses same email infrastructure |

---

## Sources

- [MarketMan Purchase & Order Management](https://www.marketman.com/platform/restaurant-purchasing-software-and-order-management) — food service procurement feature patterns (MEDIUM confidence)
- [Zoho Cliq Channel Webhooks (official docs)](https://www.zoho.com/cliq/help/platform/channel-webhooks.html) — webhook integration; note outgoing webhooks are deprecated, incoming webhook tokens for posting remain active (HIGH confidence)
- [Zoho Cliq Post to Channel API](https://www.zoho.com/cliq/help/platform/post-to-channel.html) — endpoint URL `https://cliq.zoho.com/api/v2/channelsbyname/{channel}/message` and zapikey auth (HIGH confidence)
- [PAR Level inventory alerts — Shopventory](https://shopventory.com/par-inventory/) — par-based alert trigger patterns (MEDIUM confidence)
- [Restaurant inventory management best practices — Sage](https://www.sage.com/en-us/blog/restaurant-inventory-management-best-practices/) — ordering workflow patterns (MEDIUM confidence)
- [Purchase order approval workflow — Opstream](https://www.opstream.ai/blog/purchase-order-workflow-steps-process-and-best-practices/) — approval flow UX and state machine patterns (MEDIUM confidence)
- [Lumiform restaurant checklist apps 2026](https://lumiformapp.com/comparisons/best-restaurant-checklist-apps) — competitor checklist completion patterns (MEDIUM confidence)
- Existing `purchasing.html` mockup — established UI patterns, tab layout, cutoff pill, contributor initials, locked view (HIGH confidence — is the direct UI spec)
- Existing `inventory.html` — stock levels, item catalog, fullscreen picker modal, stock override patterns (HIGH confidence — already implemented)
- `.planning/PROJECT.md` — v3.0 active requirements list (HIGH confidence — primary source of truth)

---

*Feature research for: Yumyums HQ v3.0 Purchase Orders and Shopping Lists*
*Researched: 2026-04-22*
