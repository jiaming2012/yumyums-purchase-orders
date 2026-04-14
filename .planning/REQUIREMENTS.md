# Requirements: Yumyums HQ — Inventory App

**Defined:** 2026-04-14
**Core Value:** Food cost intelligence through purchase analytics — estimate costs from purchase frequency and ingredient proportions without manual inventory counting.

## v1.1 Requirements

Requirements for the Inventory App milestone. Each maps to roadmap phases.

### Purchase History

- [ ] **HIST-01**: User can browse purchase events sorted by date, expandable to see line items (name, qty, price, case flag)
- [ ] **HIST-02**: User can filter purchase events by vendor
- [x] **HIST-03**: Inventory tile appears on HQ launcher and links to inventory.html
- [x] **HIST-04**: Inventory page is cached by service worker for offline PWA use

### Spending Trends

- [ ] **TRND-01**: User can see a bar chart of spending by tag category (Beef, Produce, Supplies, etc.) for a selected time range
- [ ] **TRND-02**: User can see a pie/doughnut chart showing spending proportion by tag
- [ ] **TRND-03**: User can see spending over time (weekly or monthly bar/line chart)
- [ ] **TRND-04**: User can filter trends by specific tags to drill into one category

### Stock & Reorder

- [ ] **STCK-01**: User can see low/medium/high stock level indicators per item group based on purchase recency
- [ ] **STCK-02**: Items at low/medium stock are flagged as "recommended for next PO" (display only)
- [ ] **STCK-03**: User can manually override a stock level with a reason (mock journal entry for backend later)

### Food Cost Intelligence

- [ ] **COST-01**: User can see estimated cost per menu item (e.g., Cheesesteak = $X) with ingredient proportion table — mock data, real calculation by backend later
- [ ] **COST-02**: For a menu item, user can see which purchase items contribute (beef, rolls, onions) with proportions
- [ ] **COST-03**: For a purchase item, user can see which menu items use it — relative percentages showing cost, revenue, and return on purchase

### Integration

- [x] **INTG-01**: Each section is its own tab (4 tabs: History / Trends / Stock / Cost) for future RBAC gating
- [ ] **INTG-02**: Architecture supports future replacement of Trends/Cost tabs with embedded Metabase reports

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### Backend Integration
- **BEND-01**: Purchase events synced from backend (replace mock data)
- **BEND-02**: Stock level overrides create journal entries in backend
- **BEND-03**: Food cost calculations performed by backend (potentially AI-assisted)
- **BEND-04**: Reorder suggestions integrated with Purchasing app

### Analytics
- **ANLT-01**: Embedded Metabase reports for Trends and Cost tabs (replace native charts)
- **ANLT-02**: Sales data integration for revenue/margin calculations

### Advanced
- **ADVN-01**: Recipe/BOM (bill of materials) editor for menu items
- **ADVN-02**: Waste tracking and variance reporting

## Out of Scope

| Feature | Reason |
|---------|--------|
| Barcode scanning | Hardware dependency, overkill for food truck scale |
| Physical inventory counts | Too much staff effort — purchase frequency estimation is the core approach |
| Real-time stock sync | No backend yet; mock data only |
| Actual ordering from inventory | Future backend integration with Purchasing app |
| Recipe costing engine | Backend computation, potentially AI-assisted — UI schema designed but not calculated |
| Multi-location inventory | Single food truck operation |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| HIST-01 | Phase 6 | Pending |
| HIST-02 | Phase 6 | Pending |
| HIST-03 | Phase 6 | Complete |
| HIST-04 | Phase 6 | Complete |
| INTG-01 | Phase 6 | Complete |
| STCK-01 | Phase 7 | Pending |
| STCK-02 | Phase 7 | Pending |
| STCK-03 | Phase 7 | Pending |
| TRND-01 | Phase 8 | Pending |
| TRND-02 | Phase 8 | Pending |
| TRND-03 | Phase 8 | Pending |
| TRND-04 | Phase 8 | Pending |
| COST-01 | Phase 8 | Pending |
| COST-02 | Phase 8 | Pending |
| COST-03 | Phase 8 | Pending |
| INTG-02 | Phase 8 | Pending |

**Coverage:**
- v1.1 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-04-14*
*Last updated: 2026-04-14 — traceability mapped to phases 6-8*
