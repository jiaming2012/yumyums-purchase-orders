# Phase 4: Onboarding App - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-13
**Phase:** 04-onboarding-app
**Areas discussed:** Screen structure, Checklist design, Progress tracking, Sign-off flow

---

## Screen Structure

| Option | Description | Selected |
|--------|-------------|----------|
| Two tabs (Recommended) | Tab 1: Active Onboardings. Tab 2: Completed. Tap a hire to see checklist. | ✓ (with modifications) |
| Three tabs | Active / Completed / Templates. Mirrors workflows 3-tab pattern. | |
| Single page | No tabs — one scrollable page with inline progress. | |

**User's choice:** Two tabs, but with role-based separation: "My Trainings" for crew (own checklists), "Manager" tab with Active/Completed for manager+ roles.
**Notes:** User clarified that crew should have their own list view, not just the manager's overview.

### Follow-up: New Hire Creation

| Option | Description | Selected |
|--------|-------------|----------|
| Manager tab has + New Hire | Manager creates onboarding from template | |
| New hires are pre-populated | Mock data only, no creation flow | |

**User's choice:** Neither — onboarding checklists are created by role, not per-hire.
**Notes:** User clarified that checklists are role-based (e.g., "Line Cook Onboarding"), auto-assigned when someone has that role.

### Follow-up: Assignment Model

| Option | Description | Selected |
|--------|-------------|----------|
| Auto by role | Crew sees onboarding checklist matching their role automatically | ✓ |
| Manager assigns | Manager picks a hire and assigns a template | |

**User's choice:** Auto by role, but manager can also assign additional trainings manually.

---

## Checklist Design

| Option | Description | Selected |
|--------|-------------|----------|
| Simpler format (Recommended) | Checkboxes grouped by section. No temperature/skip logic/fail triggers. | ✓ |
| Reuse workflows fields | Full field type support from workflows builder. | |
| Checkboxes + notes | Checkboxes with optional text notes per item. | |

**User's choice:** Simpler format, but primarily video-based. Tasks are broken into video series with title and description per part.

### Follow-up: Video Display

| Option | Description | Selected |
|--------|-------------|----------|
| Expandable sections | Tap topic to expand, see video parts with title + description. Check off each. | ✓ |
| Flat list with grouping | All parts shown flat under headers. | |
| Step-through flow | Focused view, one part at a time. | |

**User's choice:** Expandable sections with progress count on header.

### Follow-up: Video Embedding

| Option | Description | Selected |
|--------|-------------|----------|
| Mock only — just checkboxes | No video embedding, honor system. | |
| Embed video links | YouTube/Vimeo URLs, tapping opens video. | ✓ |

**User's choice:** Embed video links.

### FAQ Addition (User-initiated)

**User's input:** Each checklist should have an FAQ section with Q&A pairs. Viewing the FAQ must be required before completion.

| Option | Description | Selected |
|--------|-------------|----------|
| Expandable Q&A accordion | Each Q tappable to expand answer. Must expand all before submit. | |
| Single FAQ page gate | Scroll through FAQ, tap "I've read this" at bottom. | |
| You decide | Claude picks the best FAQ interaction. | ✓ |

**User's choice:** Claude's discretion on FAQ interaction, but must enforce "viewed" requirement.

---

## Progress Tracking

| Option | Description | Selected |
|--------|-------------|----------|
| Card per hire with progress bar | Name, role, start date, progress bar. Tap for detail. | ✓ |
| Table/list view | Compact table format. | |
| You decide | Claude picks best display. | |

**User's choice:** Card per hire with progress bar.

### Follow-up: Detail View

| Option | Description | Selected |
|--------|-------------|----------|
| Section-level breakdown | Per-section progress, tap to see items. | |
| Full checklist view | Same view as crew member, read-only for manager. | ✓ |
| You decide | Claude picks. | |

**User's choice:** Full checklist view (same as crew sees, but read-only).

---

## Sign-off Flow

| Option | Description | Selected |
|--------|-------------|----------|
| One final sign-off per hire | Manager signs off after all items complete. | |
| Per-section sign-off | Manager signs off each section independently. | ✓ |
| Auto-complete, no sign-off | All items checked → auto-complete. | |

**User's choice:** Per-section sign-off with these rules:
- Sign-off is optional per section (configurable)
- Sections must be completed in order (sequential gating)
- Items within a section can be completed in any order
- When sign-off required: crew completes → manager approves → next section unlocks

### Follow-up: Sign-off Interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Submit for review | Crew taps "Submit for Review", manager sees and approves/sends back. | |
| Auto-notify, one-tap approve | Auto-notification to manager on section completion. | |
| You decide | Claude picks. | ✓ |

**User's choice:** Claude's discretion on the sign-off interaction pattern.

---

## Claude's Discretion

- FAQ interaction pattern (must enforce "viewed" requirement)
- Sign-off request/approval interaction
- Empty state messaging
- Card styling and progress bar appearance
- Manager's UI for assigning additional trainings
- Mock template content

## Deferred Ideas

None — discussion stayed within phase scope
