# Phase 4: Onboarding App - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Standalone onboarding tool (`onboarding.html`) for new crew member onboarding. Auto-assigns role-based training checklists, supports video-based training with multi-part series, per-section sign-off by managers, and FAQ sections. Mock data only — no backend.

</domain>

<decisions>
## Implementation Decisions

### Screen Structure
- **D-01:** Two role-based views: "My Trainings" tab (all roles — crew sees their own onboarding checklists) and "Manager" tab (manager+ only — sees all hires with Active/Completed sub-views).
- **D-02:** Manager Active view shows card-per-hire with name, role, start date, and progress bar (e.g., "60% — 12/20 tasks").
- **D-03:** Manager taps a hire's card to see the full checklist view (same view crew sees, but read-only for manager).

### Assignment Model
- **D-04:** Onboarding checklists are auto-assigned by role. A "Line Cook" gets the "Line Cook Onboarding" template automatically.
- **D-05:** Managers can also manually assign additional training checklists to a hire beyond the role default.

### Checklist Design
- **D-06:** Simpler format than workflows — checkboxes grouped by sections (e.g., Paperwork, Training, Equipment). No temperature, skip logic, or fail triggers.
- **D-07:** Primary content is video-based training. Each training topic is a series of video parts (e.g., "Kitchen Cleanup" → Part 1, Part 2, Part 3). Each part has a title, short description, and embedded video link (YouTube/Vimeo URL).
- **D-08:** Expandable sections — tap a training topic to expand and see video parts listed with title + description. Check off each part after watching. Section header shows progress count (e.g., "[2/3]").

### FAQ Section
- **D-09:** Each onboarding checklist can have an FAQ section with Q&A pairs.
- **D-10:** Viewing the FAQ is required — crew member must expand/view the FAQ before the checklist can be considered complete.

### Sign-off Flow
- **D-11:** Per-section sign-off, optional per section (configurable in mock template data).
- **D-12:** Sections must be completed in order — can't start section 2 until section 1 is done + signed off (if sign-off required for that section).
- **D-13:** Items within a section can be completed in any order.
- **D-14:** When sign-off is required, crew completes all items → section enters "pending sign-off" state → manager approves or sends back → next section unlocks.

### Progress Tracking
- **D-15:** Manager Active tab: card per hire with progress bar and task count.
- **D-16:** "My Trainings" view for crew: shows their assigned checklists with per-checklist progress.

### Claude's Discretion
- FAQ interaction pattern (accordion vs gate page vs other — must enforce "viewed" requirement)
- Sign-off request/approval interaction (submit-for-review button vs auto-notify)
- Empty state messaging for both crew and manager views
- Exact card styling, progress bar appearance, section expand/collapse animation
- How manager assigns additional training checklists (UI for manual assignment)
- Mock template content (realistic food truck onboarding items with video titles/descriptions)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing App Patterns
- `workflows.html` — Three-tab layout, event delegation, state-first rendering pattern, MOCK_TEMPLATES data shape. Reference for established UI patterns but onboarding has its own simpler data model.
- `users.html` — Role/permission model, USERS array with role field. Reference for how roles are defined (superadmin, admin, manager, team_member).
- `index.html` — Launcher grid where new tile must be added.

### Infrastructure
- `sw.js` — Service worker cache list. Must add `onboarding.html` to ASSETS array and bump cache version.
- `.planning/codebase/CONVENTIONS.md` — Naming patterns, CSS variable block, PWA boilerplate, code style.
- `.planning/codebase/STRUCTURE.md` — "Where to Add New Code" section for new tool page checklist.

### Prior Phase Context
- `.planning/phases/01-template-builder/01-CONTEXT.md` — Data model decisions, builder layout patterns.
- `.planning/phases/02-fill-out-and-conditional-logic/02-CONTEXT.md` — Fill-out UX decisions (D-01 through D-11), user attribution pattern (CURRENT_USER hardcoded mock).
- `.planning/phases/03-photo-approval-and-integration/03-CONTEXT.md` — Approval flow pattern, tab gating by role.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- CSS variable block (`:root` with `--bg`, `--card`, `--txt`, `--mut`, `--brd`, dark mode override) — copy from any existing page
- PWA boilerplate (viewport meta, manifest link, SW registration, dblclick prevention) — identical on every page
- `ptr.js` — pull-to-refresh utility, load via script tag
- Tab switching pattern (`show(n)` function) — established in purchasing.html, users.html, workflows.html

### Established Patterns
- State-first rendering: mutate JS state → call render function → DOM updates from state
- Event delegation with `data-action` attributes (established in workflows.html)
- `CURRENT_USER` mock for attribution (from Phase 2)
- Cards with progress bars (established in workflows.html My Checklists tab)

### Integration Points
- `index.html` — Add active tile linking to `onboarding.html` (change a "Soon" tile or add new one)
- `sw.js` — Add `'./onboarding.html'` to ASSETS array, bump CACHE version
- `users.html` — Add `{slug:'onboarding', name:'Onboarding', icon:'🎓'}` to APPS array for permission management

</code_context>

<specifics>
## Specific Ideas

- Video-first training: most checklist items are watching video series, not just checking boxes
- Each video part has: title, short written description, embedded video link (YouTube/Vimeo URL that opens in browser/app)
- Sequential section progression enforced (can't skip ahead) but items within a section are free-order
- FAQ viewing is mandatory before checklist completion — not optional

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-onboarding-app*
*Context gathered: 2026-04-13*
