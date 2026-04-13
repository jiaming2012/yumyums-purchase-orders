# Project Research Summary

**Project:** Yumyums HQ — Workflow/Checklist Engine
**Domain:** Mobile-first food safety workflow/checklist builder PWA (vanilla JS, no build step)
**Researched:** 2026-04-12
**Confidence:** HIGH

## Executive Summary

This project adds a workflow/checklist engine to an existing vanilla JS PWA with no build step. The feature lives in a new `workflows.html` page following the exact same single-file, no-framework conventions as `purchasing.html`. The core product is a Lumiform-style system with two distinct runtime modes: Fill Mode (crew completes assigned checklists) and Build Mode (owner/manager creates and edits templates). Research into Lumiform, iAuditor, GoAudits, and Jolt confirms this is a well-mapped domain with established patterns — the architecture is not novel, and the risk is entirely in implementation details rather than design discovery.

The recommended approach is to build in four strict layers: data model first (schema and pure logic functions), then Fill Mode end-to-end on mock data, then Build Mode, then wiring the two together. The central data design decision is the strict separation of Template (the blueprint, never mutated at fill-out time) from Response (the per-submission record, keyed by stable field IDs). JsonLogic is the recommended conditional logic library because it serializes to JSON and has a Go port — making it the right choice for the future backend without forcing a rewrite. SortableJS (1.15.7) covers drag-to-reorder in the builder without a bundler.

The two highest risks are both implementation-level: (1) vanilla JS state management — the DOM must never be used as source of truth, and a single canonical state object must be established before any conditional logic is written; and (2) iOS PWA behavior — keyboard/viewport bugs and photo capture single-use bugs that only manifest on a real iPhone in standalone mode, not in browser emulation. Both are preventable with deliberate design choices made in Phase 1, before UI is built around them.

---

## Key Findings

### Recommended Stack

The existing codebase constrains this to vanilla JS ES2020+, native CSS custom properties, and no build step — all new code must work as CDN includes or plain JS modules. Within those constraints, two external libraries are needed: SortableJS 1.15.7 (CDN via cdnjs) for drag-to-reorder in the template builder, and JsonLogic (CDN, verify at cdnjs.com/libraries/json-logic-js) for conditional rule evaluation. Everything else is vanilla.

The future Go + Postgres backend is not in scope for this milestone but informs data model decisions now. The full backend stack (chi v5.2.5, pgx v5.9.1, sqlc 1.30.0, golang-migrate v4, Postgres 16) is pre-selected and consistent with the existing `infra/` Temporal stack. The template schema stored as `jsonb` in Postgres means the frontend JSON data model must be finalized correctly — changing field ID schemes after the backend arrives is costly.

**Core technologies:**
- Vanilla JS ES2020+ (existing): all interactivity — matches codebase, no framework allowed
- CSS custom properties (existing): theming, dark mode — already established, extend not replace
- SortableJS 1.15.7 (CDN): drag-to-reorder in builder — only touch-native option without a bundler
- JsonLogic (CDN): conditional rule evaluation — language-portable, has Go port, rules are data
- `<input type="file" capture="environment">`: photo capture — opens rear camera on iOS/Android PWA without getUserMedia complexity
- Go 1.22+ / chi v5 / pgx v5 / sqlc 1.30 (future): API backend — already chosen, consistent with infra repo

See `.planning/research/STACK.md` for full rationale and version details.

### Expected Features

Research into the competitive landscape confirms two clear tiers: table stakes that every food service checklist app has, and differentiators that justify building vs. buying.

**Must have (table stakes):**
- Checkbox, yes/no, text note, temperature input — core field types; absence makes the product feel incomplete
- Section grouping — checklists beyond 8-10 items without sections are unusable
- Fill-out view with user attribution and timestamp — the primary crew experience
- Template builder with role-based access gate — how checklists are created; restricted to owner/manager
- Completion progress indicator — "7 of 12 items" reduces anxiety for crew mid-checklist
- Submission history / audit trail — required for health inspections; any serious food service tool has this

**Should have (competitive differentiators):**
- Fail triggers with inline corrective action prompts — the key food safety differentiator; Lumiform's "response triggers action"; required for documented HACCP compliance
- Temperature range validation with pass/fail feedback — inline green/red on number fields; auto-triggers corrective action
- Day-of-week conditions — food trucks have different procedures by day; evaluated at fill-out time, stored as day-mask on the template
- Skip logic / conditional fields — show/hide fields based on prior answers; requires stable field IDs from day one

**Defer to post-mock / backend phase:**
- Photo capture (backend must store it; mock uses placeholder only)
- Signature capture (same; needs storage)
- Offline draft persistence (requires IndexedDB; out of scope for mock phase)
- Push notifications (requires backend + push subscription)
- Analytics / BI dashboard (requires meaningful historical data volume)

See `.planning/research/FEATURES.md` for full taxonomy including food service workflow types.

### Architecture Approach

The system has two runtime modes — Fill Mode and Build Mode — that share a single data schema but are not the same UI rendered differently. They have fundamentally different interaction models. Fill Mode is a sequential execution engine; Build Mode is an editor. The architectural decision that matters most: Template and Response are strictly separate objects. The Template is read-only at fill-out time. Responses are keyed by stable field IDs (never array index, never display label).

The recommended build order flows strictly by dependency: data model and pure logic functions first, Fill Mode second (validates the schema is livable before investing in the editor), Build Mode third, integration fourth.

**Major components:**

Fill Mode:
1. `AssignmentList` — filters MOCK_TEMPLATES by active_days for today; entry point for crew
2. `ChecklistRunner` — renders one template; manages answer state; wires all fill-mode sub-components
3. `ConditionEvaluator` — determines visible fields given current answers; pure function, no DOM
4. `FailTriggerEvaluator` — determines if an answer fires a fail trigger; pure function
5. `FieldRenderer` — renders one field as the correct input element; handles all field types
6. `ResponseCollector` — accumulates answers, tracks completion, produces final response on submit

Build Mode:
7. `TemplateList` — entry point; shows all templates, "New template" button
8. `TemplateEditor` — outer shell: name, active_days, role assignments
9. `SectionEditor` — add/remove/reorder sections via SortableJS; edit section title
10. `FieldEditor` — add/remove/reorder fields; configure label, type, required, validation
11. `ConditionEditor` — UI for setting field/section conditions; most complex piece; enforces forward-reference constraint

Shared pure utilities: `evaluateCondition()`, `evaluateFailTrigger()`, `getVisibleFields()`, `isDayActive()`, `computeCompletionStatus()`

See `.planning/research/ARCHITECTURE.md` for full data schemas, data flow diagrams, and the scalability table showing mock-to-backend transition points.

### Critical Pitfalls

1. **DOM as source of truth** — Reading field values from `document.querySelector()` inside business logic makes conditional evaluation break on any re-render. Prevention: maintain a single canonical state object (`responses[fieldId] = value`); the DOM is a rendering artifact only. Establish this pattern before writing any conditional logic.

2. **Conditional logic without dependency ordering** — Naive re-evaluation of all conditions on every change works until conditions reference each other, producing infinite evaluation loops. Prevention: enforce a forward-reference constraint (a field's condition can only reference fields earlier in the form); clear hidden field values from state immediately when a field is hidden.

3. **Schema mismatch between template and responses** — Storing response answers by array position or display label breaks silently when templates are edited. Prevention: every field gets a stable UUID-style ID at creation time; responses are always `responses[field_id]`, never positional. Use hardcoded stable IDs in mock data from day one.

4. **Event listener accumulation on dynamic UI** — In vanilla JS, re-rendering a dynamic list (drag reorder, add/delete field) without removing old listeners causes handlers to fire multiple times or capture stale closures. Prevention: use event delegation on parent containers; never attach listeners inside loops; capture field ID in closures, never array index.

5. **iOS Safari keyboard/viewport breakage in standalone mode** — `100vh`, `position: fixed; bottom: 0`, and fixed footer patterns break on iPhone when the virtual keyboard opens in installed PWA mode. Prevention: use `-webkit-fill-available` instead of `100vh`; test every field type on a real iOS device in standalone mode before marking it complete.

6. **Fail trigger UX that interrupts fill-out flow** — Modal dialogs and full-page redirects for corrective actions cause crew to lose their place and skip the corrective action. Prevention: inline expansion below the triggering field; never block checklist progression with a modal.

See `.planning/research/PITFALLS.md` for 13 pitfalls with full prevention strategies and phase assignments.

---

## Implications for Roadmap

Based on research, the architecture research explicitly defines a build order that must be followed — each layer depends on the layer below it being stable. The suggested phase structure maps directly to that order.

### Phase 1: Data Foundation and Template Builder

**Rationale:** Everything depends on the data model being correct. Field IDs must be stable UUIDs from the first line of code. The template schema (Template object shape, field types, condition structure, fail trigger structure) must be frozen before the fill-out view is built on top of it. The builder is also where event delegation, drag-to-reorder, and role gating patterns are established — getting these right first prevents rewrites later.

**Delivers:** `workflows.html` with Tab 2 (Build Mode) functional end-to-end on mock data; `MOCK_TEMPLATES[]` in the final schema with 2-3 real food service templates; all pure logic utilities tested in browser console; SortableJS integrated for section and field reorder.

**Addresses:** Template builder (table stakes), section grouping, field type configuration, role-based access gate, day-of-week condition definition (stored on template, not evaluated yet).

**Avoids:** DOM as source of truth (Pitfall 1), schema mismatch (Pitfall 3), event listener accumulation (Pitfall 4), CSS-only role gating (Pitfall 13), multi-column mobile layout (Pitfall 11), select dropdown UX (Pitfall 12).

**Research flag:** Standard patterns — no additional research phase needed. Architecture is well-documented.

### Phase 2: Fill-Out View and Conditional Logic

**Rationale:** Fill Mode validates the data model is actually usable before investing further in Build Mode polish. The conditional logic engine (ConditionEvaluator, FailTriggerEvaluator) must be built as pure functions before any UI is wired around them — this is the highest-complexity piece and the one most likely to accumulate technical debt if rushed.

**Delivers:** Tab 1 (Fill Mode) functional end-to-end; crew can open today's assigned checklists, complete all field types, encounter fail triggers with inline corrective action prompts, and submit with timestamp + user attribution. Completion progress indicator included. Responses written to `MOCK_RESPONSES[]`.

**Addresses:** All core field types (checkbox, yes/no, text, temperature/number), fill-out view UX, completion progress indicator, fail trigger + corrective action (inline expansion), temperature range validation, day-of-week condition evaluation at render time.

**Avoids:** Conditional logic without dependency ordering (Pitfall 2), fail trigger UX interrupting flow (Pitfall 6), skip logic breaking completion tracking (Pitfall 8), day-of-week logic in wrong layer (Pitfall 7), iOS keyboard/viewport breakage (Pitfall 5).

**Research flag:** Standard patterns for the logic engine. iOS PWA testing is essential — allocate real device testing time for every field type added. No research phase needed but flag iOS device access as a hard dependency.

### Phase 3: Photo Capture and Submission History

**Rationale:** Photo is deferred from Phase 2 because it has a known iOS-specific bug (single-use input) that requires dedicated testing time on a real device, separate from the core fill-out flow. Submission history (audit trail) is deferred because it requires `MOCK_RESPONSES[]` to be populated first — which happens in Phase 2.

**Delivers:** Photo field type fully functional in Fill Mode (with re-capture support); submission history view filterable by checklist and date; service worker cache updated to include new workflow pages.

**Addresses:** Photo capture (table stakes), submission history / audit trail (table stakes), service worker cache management.

**Avoids:** Photo capture single-use iOS bug (Pitfall 9), stale service worker serving old UI (Pitfall 10).

**Research flag:** Standard patterns. iOS photo re-capture testing is mandatory on a real device — do not accept emulator-only test results.

### Phase 4: Advanced Conditions and Backend Handoff Prep

**Rationale:** Skip logic and day-of-week-conditional sections are differentiators but add builder complexity. They are deferred until the core builder and fill-out flow are proven stable. This phase also prepares the data layer for backend swap — verifying that replacing `MOCK_TEMPLATES[]` with `GET /api/templates` requires no architectural changes.

**Delivers:** Skip logic (conditional field show/hide) in both builder ConditionEditor and fill-out ConditionEvaluator; day-of-week section-level conditions fully functional; data layer abstraction verified to support backend swap; `templateVersion` tracked on responses for future compatibility.

**Addresses:** Skip logic / conditional fields (differentiator), day-of-week conditions (differentiator), template versioning (required for backend integrity).

**Avoids:** Conditional logic without dependency ordering (Pitfall 2 — final hardening), forward-reference constraint violations.

**Research flag:** May benefit from a `/gsd:research-phase` for the ConditionEditor UI pattern — specifically, how other vanilla JS form builders handle the condition picker when conditions can chain. Not critical but worth 30 minutes of targeted research before implementation.

### Phase Ordering Rationale

- Data model and builder come before fill-out because fill-out's correctness depends entirely on field IDs being stable — if you build fill-out first on an ad-hoc schema and then correct the IDs, you rewrite both layers.
- Fill Mode comes before Build Mode polish because it is the validation step: if the template schema makes fill-out awkward, the schema is wrong and should be fixed before investing further in the editor.
- Photo and audit trail are isolated to Phase 3 because they have external dependencies (real device testing, populated response data) that would block or slow Phase 2 if included.
- Advanced conditions are last because they build on top of stable field IDs, a working condition evaluator, and a proven builder interaction model — adding them earlier increases the chance of needing to revisit the evaluator design.

### Research Flags

Phases needing deeper research during planning:
- **Phase 4 (ConditionEditor):** How condition-chaining UI should work in a vanilla JS builder when conditions can reference multiple prior fields. A 30-minute targeted research session before implementation is recommended, not a full research phase.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Template Builder):** Drag-to-reorder with SortableJS is thoroughly documented. Event delegation pattern is standard. Role gating pattern is explicit in PROJECT.md and existing codebase conventions.
- **Phase 2 (Fill-Out View):** Checklist runner pattern is well-understood. The condition evaluator logic is fully specified in ARCHITECTURE.md. iOS quirks are documented in PITFALLS.md with concrete prevention steps.
- **Phase 3 (Photo + History):** Camera input pattern is documented. Service worker cache busting pattern is partially implemented in the existing repo.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Frontend choices are the existing conventions (not new). SortableJS is canonical for this use case. JsonLogic CDN version needs verification at cdnjs.com before embedding. |
| Features | HIGH | Sourced from multiple competing production products (Lumiform, iAuditor, GoAudits, Jolt). Food service workflow taxonomy is grounded in HACCP guidelines. |
| Architecture | HIGH | Template/response separation is an established pattern confirmed by JSON Forms official docs and Form.io architecture docs. The condition evaluation approach is deliberate and matches the existing codebase's no-eval, no-framework constraints. |
| Pitfalls | HIGH | iOS PWA bugs are confirmed against published bug trackers and device-specific behavior documentation. Vanilla JS state pitfalls are confirmed against multiple independent sources. Event delegation pattern is standard. |

**Overall confidence:** HIGH

### Gaps to Address

- **JsonLogic CDN availability:** The STACK.md researcher noted the specific CDN version was not verified. Verify at https://cdnjs.com/libraries/json-logic-js before embedding. If unavailable on cdnjs, the npm package can be included via unpkg. This is a 5-minute check, not a blocker.

- **Backend JSONB schema finalization:** The frontend template schema must be finalized in Phase 1 because it becomes the Postgres `jsonb` structure when the backend arrives. Any field added to the schema after the backend is built requires a migration. Treat the Phase 1 data model as a contract.

- **iOS device availability:** Multiple pitfalls (Pitfalls 5, 9) require testing on a real iPhone in standalone PWA mode. Desktop Chrome DevTools device emulation does not reproduce these bugs. Confirm a test device is available before starting Phase 2.

- **Service worker cache busting ownership:** The existing `sw.js` uses a cache key that needs incrementing on each deploy. The workflow pages need to be added to the precache list. This is an operational concern that needs an owner before Phase 3 deploy.

---

## Sources

### Primary (HIGH confidence)
- Existing codebase: `purchasing.html`, `index.html`, `sw.js`, `docs/user-management-api.md` — conventions, patterns, existing data model
- [JSON Forms Architecture](https://jsonforms.io/docs/architecture/) — template vs. instance separation, renderer model
- [SortableJS GitHub](https://github.com/SortableJS/Sortable) — no jQuery, touch-native, framework-agnostic confirmed
- [go-chi/chi v5.2.5](https://github.com/go-chi/chi/releases) — Feb 2025 release verified
- [jackc/pgx v5.9.1](https://github.com/jackc/pgx/tags) — Mar 2026 release verified
- [Temporal GitHub releases v1.30.3](https://github.com/temporalio/temporal/releases) — current stable

### Secondary (MEDIUM confidence)
- [Lumiform Features / Template Builder](https://lumiformapp.com/features) — field types, corrective actions, skip logic confirmed
- [GoAudits Restaurant Checklist Apps](https://goaudits.com/blog/restaurant-checklist-apps/) — accountability, timestamps, offline sync
- [Jolt Operations Management](https://www.jolt.com/) — per-employee task tagging model
- [JsonLogic](https://jsonlogic.com/) — framework-agnostic, Go + JS implementations
- [LogRocket: Offline-first frontend apps 2025](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/) — IndexedDB + Background Sync patterns
- [Form.io Logic & Conditions](https://help.form.io/userguide/forms/form-building/logic-and-conditions) — conditional logic patterns
- [Frontend System Design: Dynamic Form Builder](https://shivambhasin29.medium.com/mastering-frontend-system-design-building-a-dynamic-form-builder-from-scratch-0dfdd78d31d6) — component decomposition

### Tertiary (LOW confidence, needs validation)
- [pgx vs sqlx performance](https://dasroot.net/posts/2025/12/go-database-patterns-gorm-sqlx-pgx-compared/) — 300% bulk insert claim; directional signal only
- [iOS photo capture re-trigger bug](https://github.com/PWA-POLICE/pwa-bugs/issues/12) — documented but requires confirmation on current iOS version

---
*Research completed: 2026-04-12*
*Ready for roadmap: yes*
