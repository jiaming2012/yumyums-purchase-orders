# Phase 3: Photo, Approval, and Integration - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete the workflow app: make photo capture functional (standalone photo fields + corrective action card photos), add manager approval flow with a third tab, wire the Operations tile on the HQ launcher, and include 2 pre-built food truck templates (Setup Checklist, Closing Checklist).

</domain>

<decisions>
## Implementation Decisions

### Photo Capture UX
- **D-01:** Uses `<input type="file" capture="environment">` — opens rear camera directly. No JS camera API. Works in iOS standalone PWA.
- **D-02:** Two-step flow after capture: (1) Full-size preview with checkmark icon and retake button. (2) After checkmark is pressed, collapses to small thumbnail inline in the field row with a "Retake" button next to it.
- **D-03:** Corrective action card photo stub becomes functional — the "Add photo" button in fail cards also opens the camera for evidence capture. Same two-step preview flow.

### Approval Flow
- **D-04:** New third tab: "Approvals" — sits between My Checklists and Builder. Shows list of submitted checklists awaiting manager review.
- **D-05:** Review view is a summary card: checklist name, who submitted, when, completion %, any fail triggers fired, and photos captured. Approve/Reject buttons at bottom. Not a full read-only runner.
- **D-06:** Approval is one-tap (toast "Approved ✓"). Rejection requires the manager to type a reason before confirming (modal or inline text input + confirm).
- **D-07:** After approval/rejection, submission moves to a completed/rejected state. No notification to crew (mock only).

### Pre-Built Templates
- **D-08:** Two pre-built templates: Setup Checklist and Closing Checklist.
- **D-09:** Minimal placeholder items — enough to demonstrate the feature, not realistic food truck items.

### Integration
- **D-10:** Operations tile already exists on index.html (added previously). INTG-01 is already satisfied — just verify it links correctly.
- **D-11:** INTG-02 (role-based tab access) — Builder tab already gated. Approvals tab should also be gated to manager+ roles. My Checklists visible to all.

### Claude's Discretion
- Exact summary card layout and information density
- Tab ordering (My Checklists / Approvals / Builder)
- How "requires approval" flag from builder connects to the approval queue
- Photo thumbnail size and retake button styling

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Prior Phase Artifacts
- `.planning/phases/01-template-builder/01-CONTEXT.md` — Data model, builder layout decisions
- `.planning/phases/02-fill-out-and-conditional-logic/02-CONTEXT.md` — Fill-out UX decisions, fail trigger behavior, corrective action card structure
- `.planning/phases/01-template-builder/01-RESEARCH.md` — Photo capture approach (`<input capture="environment">`), iOS single-use input bug (Pitfall 9)

### Research
- `.planning/research/PITFALLS.md` — iOS Safari photo capture single-use bug (Pitfall 9), iOS standalone keyboard bugs (Pitfall 5)
- `.planning/research/FEATURES.md` — Table stakes: photo capture, submission history, role-based access

### Existing Code
- `workflows.html` — Current implementation (~1100+ lines): builder tab, fill-out runner, MOCK_TEMPLATES, MOCK_RESPONSES, FAIL_NOTES, SUBMITTED_TEMPLATES, fireworks, toast, conditional logic
- `index.html` — Operations tile already present and linking to workflows.html
- `sw.js` — Already includes workflows.html in ASSETS (currently v19)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Tab switching** (`show(n)`): Currently 2 tabs. Adding a 3rd tab for Approvals requires updating the show function to handle `[1,2,3]`.
- **SUBMITTED_TEMPLATES**: Already tracks which templates have been submitted. Approval flow builds on this.
- **MOCK_CURRENT_USER**: Has `name`, `initials`, `role`, `permissions`. Can check `role === 'admin'` or `role === 'manager'` for approval tab visibility.
- **Toast system** (`showToast()`): Reusable for approval/rejection confirmations.
- **Fireworks**: Already triggers on fully-completed submit.
- **Fail card** (`.fail-card`): Photo stub already rendered as disabled button. This phase makes it functional.
- **Event delegation pattern**: Single click + input listeners on container divs.

### Established Patterns
- State-first rendering: mutate JS state → call render function
- In-memory only (D-11 from Phase 1) — approval state also in-memory
- CSS inline in `<style>` block, dark mode via `@media(prefers-color-scheme:dark)`

### Integration Points
- `renderRunner()` photo fields: currently render "Photo capture coming in Phase 3" placeholder — replace with functional capture
- `renderFailCard()` photo stub: currently disabled button — make functional
- Tab bar: add Approvals tab, update `show(n)` for 3 tabs
- `submitChecklist()`: needs to check `requires_approval` flag and route to approval queue instead of immediately completing

</code_context>

<specifics>
## Specific Ideas

- Photo full-size preview should overlay the current view (not navigate away) — modal-style with dark backdrop, checkmark (✓) and retake (↻) buttons at bottom
- After confirm, thumbnail should be ~60-80px square with rounded corners, retake as a small text link
- Approval summary card should show photo thumbnails in a horizontal scroll if multiple photos were captured
- "Requires approval" toggle already exists in the builder's meta card — connect it to the approval flow
- Setup Checklist and Closing Checklist can replace the current MOCK_TEMPLATES (Morning Opening Checklist and Closing Checklist) or be added alongside them

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-photo-approval-and-integration*
*Context gathered: 2026-04-13*
