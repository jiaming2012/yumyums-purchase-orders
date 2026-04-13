# Requirements: Yumyums HQ — Workflow Engine

**Defined:** 2026-04-13
**Core Value:** A workflow engine that lets the owner build checklist templates and have crew members fill them out on mobile — with accountability and smart conditions.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Fill-Out View

- [ ] **FILL-01**: Crew member can see a list of available workflow checklists to complete
- [ ] **FILL-02**: Crew member can open a checklist and see items grouped by sections (e.g., Equipment, Food Prep, Cleaning)
- [ ] **FILL-03**: Crew member can check off checkbox / yes-no items with large touch targets
- [ ] **FILL-04**: Crew member can enter text notes on any item
- [ ] **FILL-05**: Crew member can enter temperature readings with unit display (°F)
- [ ] **FILL-06**: Crew member can capture a photo using the device camera for a photo field
- [ ] **FILL-07**: Each completed item shows who checked it (user attribution)
- [ ] **FILL-08**: Crew member can see completion progress (e.g., "7 of 12 items complete")

### Conditional Logic

- [ ] **COND-01**: When a temperature reading is out of the defined range, an inline corrective action prompt appears (fail trigger)
- [ ] **COND-02**: When an item is answered "No", an inline corrective action prompt appears requiring notes
- [ ] **COND-03**: Sections or items can be configured to only appear on certain days of the week
- [ ] **COND-04**: Items can be conditionally shown/hidden based on a prior answer (skip logic)

### Template Builder

- [ ] **BLDR-01**: Owner/manager can create a new workflow template with a name and sections
- [ ] **BLDR-02**: Owner/manager can add field types to a section: checkbox, yes/no, text, temperature, photo
- [ ] **BLDR-03**: Owner/manager can reorder fields within a section via drag
- [ ] **BLDR-04**: Owner/manager can delete fields from a template
- [ ] **BLDR-05**: Owner/manager can set temperature range thresholds (min/max) for fail trigger
- [ ] **BLDR-08**: Owner/manager can mark a photo field as required (must upload before submission)
- [ ] **BLDR-06**: Owner/manager can configure day-of-week conditions on sections or fields
- [ ] **BLDR-07**: Owner/manager can configure skip logic rules (if field X = Y, show/hide field Z)

### Approval

- [ ] **APRV-01**: Owner/manager can mark a template as requiring manager approval before it's considered complete
- [ ] **APRV-02**: When a checklist requires approval, the completed submission shows a "Pending Approval" status
- [ ] **APRV-03**: Manager can approve or reject a completed checklist with optional notes

### Integration

- [ ] **INTG-01**: Workflows app appears as an active tile on the HQ launcher (index.html)
- [ ] **INTG-02**: Fill-out tab is visible to all roles; builder tab is restricted by User Management permissions
- [ ] **INTG-03**: Mock includes 2-3 pre-built food truck templates (Opening Checklist, HACCP Temp Log, Closing Checklist)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Persistence & Sync

- **SYNC-01**: Completed checklists are saved to backend (Go + Postgres)
- **SYNC-02**: In-progress checklists auto-save to local storage (draft/resume)
- **SYNC-03**: Offline completion queues submissions for sync on reconnection

### History & Compliance

- **HIST-01**: User can view submission history (past completed checklists)
- **HIST-02**: Submission history is filterable by template and date
- **HIST-03**: Each submission is an immutable audit record

### Advanced Features

- **ADVN-01**: Signature capture on checklist completion
- **ADVN-02**: Required-field enforcement (prevent submission until critical items answered)
- **ADVN-03**: Repeatable sections (check multiple fridges with same fields)
- **ADVN-04**: Push notifications when a scheduled checklist is due

## Out of Scope

| Feature | Reason |
|---------|--------|
| Multi-location dashboard | Single food truck; irrelevant complexity |
| Scoring / audit percentages | Not how a small crew thinks about checklists; use pass/fail |
| Bluetooth temperature sensors | Enterprise hardware dependency; manual entry sufficient |
| AI-generated templates | Adds LLM cost for something owner can do in 10 minutes |
| Real-time collaboration | 1 person per checklist; last-write-wins is acceptable |
| Training / SOP modules | Out of scope; keep workflow engine focused |
| Analytics / BI dashboard | Separate BI tile exists; audit trail covers immediate need |
| Barcode / QR scanning | No SKU-linked checklist items at this scale |
| Backend implementation | UI mocks only for this milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| — | — | Pending |

**Coverage:**
- v1 requirements: 26 total
- Mapped to phases: 0
- Unmapped: 26 ⚠️

---
*Requirements defined: 2026-04-13*
*Last updated: 2026-04-13 after initial definition*
