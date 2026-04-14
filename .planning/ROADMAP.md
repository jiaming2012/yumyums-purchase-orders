# Roadmap: Yumyums HQ — Workflow Engine

## Overview

Three phases deliver a complete Lumiform-style checklist engine as a vanilla JS UI mock. Phase 1 locks the data model and builds the template editor (builder-first because fill-out correctness depends on stable field IDs). Phase 2 builds the crew fill-out experience and wires the conditional logic engine. Phase 3 completes the surface area: photo capture (isolated due to iOS single-use input bug), manager approval flow, and HQ integration.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Template Builder** - Lock the data model and build the owner/manager template editor with section grouping, all field types, drag-to-reorder, and day-of-week + skip logic conditions
- [x] **Phase 2: Fill-Out and Conditional Logic** - Build the crew fill-out experience end-to-end: checklist runner, all non-photo field types, fail triggers, inline corrective actions, and completion tracking (completed 2026-04-13)
- [x] **Phase 3: Photo, Approval, and Integration** - Add photo capture (isolated iOS testing), manager approval flow, and wire the app into the HQ launcher with pre-built food truck templates (completed 2026-04-13)
- [x] **Phase 4: Onboarding App** - Standalone onboarding tool for new crew member onboarding: checklist per new hire, training progress tracking, owner sign-off (completed 2026-04-14)

## Phase Details

### Phase 1: Template Builder
**Goal**: Owner/manager can create and edit checklist templates with sections, all field types, conditional logic rules, and role-gated access — with a frozen data schema that downstream phases build on
**Depends on**: Nothing (first phase)
**Requirements**: BLDR-01, BLDR-02, BLDR-03, BLDR-04, BLDR-05, BLDR-06, BLDR-07, BLDR-08
**Success Criteria** (what must be TRUE):
  1. Owner can create a new template, add named sections, and add fields of every supported type (checkbox, yes/no, text, temperature, photo) to each section
  2. Owner can drag fields to reorder them within a section
  3. Owner can delete a field and set temperature min/max thresholds and mark a photo field as required
  4. Owner can configure day-of-week conditions on a section or field, and skip logic rules (if field X = Y, show/hide field Z)
  5. Builder tab is only accessible to roles with builder permission; crew-only users see the tab absent or disabled
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md — Page scaffold, data model, template list, editor view, section CRUD
- [x] 01-02-PLAN.md — Field type picker, field CRUD, expand/collapse settings, temperature min/max, photo required
- [x] 01-03-PLAN.md — SortableJS drag-to-reorder, day-of-week chips, skip logic editor, sw.js update

**UI hint**: yes

### Phase 2: Fill-Out and Conditional Logic
**Goal**: Crew members can open today's assigned checklists, complete all non-photo field types, encounter inline corrective actions when items fail, and submit with user attribution and progress tracking
**Depends on**: Phase 1
**Requirements**: FILL-01, FILL-02, FILL-03, FILL-04, FILL-05, FILL-07, FILL-08, COND-01, COND-02, COND-03, COND-04
**Success Criteria** (what must be TRUE):
  1. Crew member sees a list of checklists active today (filtered by day-of-week) and can open one to see items grouped by section
  2. Crew member can check off checkbox/yes-no items, enter text notes, and enter temperature readings with °F display
  3. When a temperature is out of range or a yes/no is answered "No", an inline corrective action prompt appears below the field without blocking form progression
  4. Fields and sections hidden by skip logic or day-of-week conditions are not visible; hidden field answers are cleared from state
  5. Each checked item shows the name of who checked it, and a progress indicator shows "X of Y items complete"
**Plans**: TBD
**UI hint**: yes

### Phase 3: Photo, Approval, and Integration
**Goal**: The workflow app is fully integrated into HQ, photo capture works on iOS with re-capture support, manager approval is available on submissions, and 2-3 pre-built food truck templates are included
**Depends on**: Phase 2
**Requirements**: FILL-06, APRV-01, APRV-02, APRV-03, INTG-01, INTG-02, INTG-03
**Success Criteria** (what must be TRUE):
  1. Crew member can tap a photo field, open the rear camera, capture a photo, and re-capture it if needed — on iOS in standalone PWA mode
  2. Owner can mark a template as requiring manager approval; completed submissions show "Pending Approval" status
  3. Manager can open a pending submission and approve or reject it with optional notes
  4. Workflows tile appears on the HQ launcher (index.html) and links to workflows.html
  5. Opening workflows.html shows 2-3 pre-built food truck templates (e.g., Opening Checklist, HACCP Temp Log, Closing Checklist) ready to fill out
**Plans**: 2 plans

Plans:
- [x] 03-01-PLAN.md — Photo capture: CSS, openCamera utility, photo preview modal, standalone field + corrective action card photo
- [x] 03-02-PLAN.md — Three-tab navigation, approval flow, pre-built templates, INTG verification, sw.js bump, human-verify

**UI hint**: yes

### Phase 4: Onboarding App
**Goal:** Standalone onboarding tool (onboarding.html) with role-based views, video-based training checklists, sequential section progression with manager sign-off, and FAQ gate. Mock data only.
**Depends on:** Phase 3
**Requirements**: D-01, D-02, D-03, D-04, D-05, D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13, D-14, D-15, D-16
**Success Criteria** (what must be TRUE):
  1. Crew member sees assigned onboarding checklists with progress and can complete sections sequentially
  2. Video training items link to external videos; crew checks off parts after watching
  3. FAQ section must be viewed before checklist is considered complete
  4. Manager can view hire progress, approve/send-back section sign-offs, and assign additional trainings
  5. Onboarding tile is active on HQ launcher and page is cached by service worker
**Plans**: 3 plans

Plans:
- [x] 04-01-PLAN.md — Page scaffold, data model, My Trainings tab with checklist runner, section lock, video items, FAQ gate
- [x] 04-02-PLAN.md — Manager tab with hire cards, read-only drill-down, sign-off approve/send-back, assign training
- [x] 04-03-PLAN.md — HQ integration (index.html tile, sw.js cache, users.html APPS) + human-verify checkpoint

**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Template Builder | 2/3 | In Progress|  |
| 2. Fill-Out and Conditional Logic | 2/2 | Complete   | 2026-04-13 |
| 3. Photo, Approval, and Integration | 2/2 | Complete   | 2026-04-13 |
| 4. Onboarding App | 3/3 | Complete   | 2026-04-14 |
