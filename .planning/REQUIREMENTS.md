# Requirements: Yumyums HQ v3.0

**Defined:** 2026-04-22
**Core Value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.

## v3.0 Requirements

Requirements for Purchase Orders & Shopping Lists milestone. Each maps to roadmap phases.

### Purchase Order Form

- [x] **PO-01**: User can view reorder suggestions from inventory Stock tab on the PO form
- [x] **PO-02**: User can tap a reorder suggestion to add it to the purchase order
- [x] **PO-03**: User can search and add items from the inventory catalog (Setup)
- [x] **PO-04**: Each PO line item shows the item's photo, store location note, and suggested quantity
- [x] **PO-05**: User can adjust quantity on each line item before submission
- [x] **PO-06**: PO form is backed by real API data (replaces current purchasing.html mock)
- [x] **PO-07**: Each PO line item shows who added it to the order
- [x] **PO-08**: When adding an item, user can see if it's already on the PO and its current quantity

### Cutoff & Approval

- [x] **CUT-01**: Admin can configure a recurring weekly cutoff schedule (day + time)
- [ ] **CUT-02**: After cutoff time, PO is locked — only admin can edit
- [x] **CUT-03**: Backend provides a test command to simulate cutoff for easy testing
- [x] **CUT-04**: Admin can approve a locked PO, which generates a shopping checklist

### Shopping List

- [x] **SHOP-01**: Approved PO generates a shopping checklist (1:1 mapping)
- [x] **SHOP-02**: Shopping checklist tab with RBAC — assignable to specific members or roles
- [x] **SHOP-03**: Shopping list shows each item's photo and store location (tap icon to reveal)
- [ ] **SHOP-04**: User can edit store location notes inline from the shopping list
- [x] **SHOP-05**: User can mark items as checked off while shopping
- [x] **SHOP-06**: "Complete" button sends alert for any missing/unchecked items
- [ ] **SHOP-07**: When checking off an item without a photo or location, shopper is prompted to add them (can skip but must confirm each time)
- [ ] **SHOP-08**: Shopper can upload a photo for an item that doesn't have one from the shopping list

### Alerts & Notifications

- [ ] **ALRT-01**: System sends reminder alerts before cutoff time
- [ ] **ALRT-02**: System sends alerts when items are out of stock
- [ ] **ALRT-03**: Alerts delivered via Zoho Cliq channel (default) or email
- [ ] **ALRT-04**: Users configure communication preference in Users tab (at least one required, Zoho Cliq default)
- [ ] **ALRT-05**: Zoho Cliq channel integration via incoming webhook
- [ ] **ALRT-06**: Missing items alert sent on shopping list completion via configured channels

### Data Import

- [x] **IMP-01**: Notion CSV export is converted to a YAML seed file following existing fixture patterns
- [x] **IMP-02**: Seed re-hosts Notion images to DO Spaces (URLs expire after 1 hour); raw Notion URLs never stored

### Repurchase Tracking

- [ ] **REP-01**: Inventory items show "Repurchased +[Qty]" badge after purchase via shopping list completion
- [ ] **REP-02**: Badge resets on a configurable date (admin-settable reset schedule)

## Future Requirements

### Advanced PO Features

- **PO-F01**: Multi-vendor PO splitting (separate orders per store)
- **PO-F02**: PO history and reorder from previous orders
- **PO-F03**: Price tracking and cost comparison across vendors

## Out of Scope

| Feature | Reason |
|---------|--------|
| Auto-submit PO to vendor | Removes human judgment; wrong quantities waste hundreds on perishables |
| Auto-deduct inventory on purchase | Same — human verification required for food service |
| Multi-location support | Single food truck operation |
| Recurring Notion sync | One-time import is sufficient; catalog managed in-app after import |
| Vendor portal / EDI | Overkill for small crew ordering from Restaurant Depot |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PO-01 | Phase 14 | Complete |
| PO-02 | Phase 14 | Complete |
| PO-03 | Phase 14 | Pending |
| PO-04 | Phase 14 | Complete |
| PO-05 | Phase 14 | Complete |
| PO-06 | Phase 14 | Complete |
| PO-07 | Phase 14 | Complete |
| PO-08 | Phase 14 | Pending |
| CUT-01 | Phase 16 | Complete |
| CUT-02 | Phase 16 | Pending |
| CUT-03 | Phase 16 | Complete |
| CUT-04 | Phase 16 | Complete |
| SHOP-01 | Phase 16 | Complete |
| SHOP-02 | Phase 16 | Complete |
| SHOP-03 | Phase 16 | Complete |
| SHOP-04 | Phase 16 | Pending |
| SHOP-05 | Phase 16 | Complete |
| SHOP-06 | Phase 16 | Complete |
| SHOP-07 | Phase 16 | Pending |
| SHOP-08 | Phase 16 | Pending |
| ALRT-01 | Phase 17 | Pending |
| ALRT-02 | Phase 17 | Pending |
| ALRT-03 | Phase 17 | Pending |
| ALRT-04 | Phase 17 | Pending |
| ALRT-05 | Phase 17 | Pending |
| ALRT-06 | Phase 17 | Pending |
| IMP-01 | Phase 15 | Complete |
| IMP-02 | Phase 15 | Complete |
| REP-01 | Phase 17 | Pending |
| REP-02 | Phase 17 | Pending |

**Coverage:**
- v3.0 requirements: 30 total
- Mapped to phases: 30
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-22*
*Last updated: 2026-04-22 after roadmap creation*
