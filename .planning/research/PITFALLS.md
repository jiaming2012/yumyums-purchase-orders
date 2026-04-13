# Domain Pitfalls

**Domain:** Mobile-first workflow/checklist builder PWA — vanilla JS, no framework, no build step
**Researched:** 2026-04-12
**Scope:** Conditional logic engines, form builder data models, vanilla JS state management, iOS PWA quirks, food-service checklist UX

---

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

---

### Pitfall 1: DOM as Source of Truth

**What goes wrong:** Vanilla JS code reads field values directly from the DOM — `document.querySelector('#field-3').value` — rather than maintaining a separate in-memory state object. The DOM becomes the only record of what's in the form.

**Why it happens:** It feels like the obvious shortcut when there's no framework. No boilerplate, just grab the value.

**Consequences:** When you need to evaluate conditional logic ("show section B if field 3 is 'Fail'"), you end up querying the DOM inside logic evaluators. Reordering, hiding, or re-rendering any element breaks evaluation. Adding drag-to-reorder or section collapse requires a full rewrite of the logic layer. Debugging becomes impossible because state is implicit.

**Prevention:**
- Maintain a single plain JS object as the canonical state: `{ templateId, sections: [...], responses: {...} }`
- The DOM is a rendering artifact — always derive it from state, never read back from it
- For the fill-out view: `responses[fieldId] = value` on every input event; never query `.value` inside business logic
- For the builder: `template.sections[i].fields[j]` is the field definition; DOM reflects it, doesn't own it

**Warning signs:**
- Any function that calls `document.querySelector` and then passes the result into a condition evaluator
- Any function that re-reads the form DOM to figure out "current state"

**Phase:** Template builder (Phase 1) and fill-out view (Phase 2) — establish the state model before writing any conditional logic

---

### Pitfall 2: Conditional Logic Without a Dependency Graph

**What goes wrong:** Conditional rules are evaluated naively — on every change, scan all fields and re-evaluate all conditions. Works until a condition references a field that is itself conditionally shown/hidden. Then you get: Field A shows B, B shows C, C hides A → infinite evaluation loop. Or: A hides B, but B's value is still in `responses` and still affects other conditions downstream.

**Why it happens:** Simple condition evaluation is easy. Dependency tracking is not. The problem only manifests once you have 5+ fields with overlapping logic.

**Consequences:** Infinite loops that freeze the UI, or worse — silent incorrect behavior where hidden fields still influence shown fields. Both are catastrophic for a food safety checklist where a wrong corrective action trigger has real consequences.

**Prevention:**
- Model conditions as a directed graph: each field lists which other fields it depends on
- Topologically sort the evaluation order so dependencies resolve before dependents
- When a field is hidden by a condition, clear its value from `responses` immediately — hidden fields must not influence logic
- Set a maximum re-evaluation depth (e.g., 3 passes) and log a warning if it's hit
- For this project's scope (skip logic + fail triggers + day-of-week), conditions are field-level and rule-based. Do not add cross-section logic until the simple cases are proven stable.

**Warning signs:**
- A condition evaluator that calls itself recursively without a depth counter
- Hiding a field but leaving its value in state
- Any `while(changed)` loop in the condition evaluation code

**Phase:** Conditional logic engine — must be addressed before building the builder UI around it

---

### Pitfall 3: Schema Mismatch Between Template Definition and Fill Responses

**What goes wrong:** A template is defined once (builder) and filled many times (crew). When the template changes — a field is renamed, a section reordered, a new field inserted — existing in-progress or completed fill responses no longer map correctly to the template schema.

**Why it happens:** Early implementations store field references by position (`section[0].field[2]`) or by display label. Both break silently when the template is edited.

**Consequences:** Completed checklists display wrong values. Corrective actions appear attached to the wrong items. The entire fill history becomes untrustworthy.

**Prevention:**
- Every field in every template gets a stable, unique ID generated at creation time (e.g., `field_` + `Date.now()` or a short UUID). Never use array index as an identifier.
- Responses are keyed by field ID, not position: `responses["field_abc123"] = "Pass"`
- When a template is edited after submissions exist, treat it as a new version — store `templateVersion` on each filled response
- For mock phase: use hardcoded field IDs in your mock data. Don't use array indices.

**Warning signs:**
- Any code that references fields as `template.sections[i].fields[j]` outside of rendering loops
- Mock data with responses stored as arrays (`["Pass", "Fail", "N/A"]`) rather than objects keyed by field ID
- A rename of a field label that also changes how responses are looked up

**Phase:** Data model design — must be settled in Phase 1 (template builder) before building fill-out view

---

### Pitfall 4: Event Listener Accumulation on Dynamic UI

**What goes wrong:** The builder lets users add, reorder, and delete fields. Each time a field is re-rendered (e.g., after a drag-drop reorder), new event listeners are attached to the new DOM nodes — but the old ones are not removed if stale nodes linger, or new ones stack on nodes that were re-used.

**Why it happens:** In vanilla JS without a VDOM, re-rendering a list means removing and recreating DOM nodes, or mutating them in place. Both approaches have listener lifecycle traps.

**Consequences:** A button fires its handler 3x instead of 1x. A delete operation deletes the wrong field because the closure captured a stale index. Input events fire on fields the user didn't touch.

**Prevention:**
- Use event delegation: attach a single listener to the parent container (`#field-list`), inspect `event.target` with `.closest('[data-field-id]')` to identify which field was acted on. Never attach listeners to individual field rows.
- If you must attach per-element listeners, always call `removeEventListener` before re-attaching, or use `{ once: true }` for single-fire actions
- Never capture array index in closures — capture field ID (the stable identifier from Pitfall 3). The index can change; the ID cannot.
- When deleting a field row from the DOM, also delete its entry from state in the same operation

**Warning signs:**
- `addEventListener` calls inside a loop that also creates DOM nodes
- A delete button that fires once on first load, twice after adding a field, three times after adding two fields
- Any closure that captures `i` or `index` from a `forEach`

**Phase:** Template builder (Phase 1) — establish delegation pattern before the first field list is built

---

### Pitfall 5: iOS Safari Keyboard and Viewport Breakage in Standalone Mode

**What goes wrong:** On iOS, when a user taps a text input or temperature field in a PWA installed to the home screen (standalone mode), the virtual keyboard opens but Safari does not resize the viewport. Instead, it pushes the layout up. Elements positioned with `100vh`, `position: fixed`, or `bottom: 0` overlap the keyboard or scroll off screen.

**Why it happens:** Safari's standalone PWA mode has a known long-standing bug where `window.innerHeight` does not update when the keyboard appears. `100vh` also includes the Safari bottom bar, causing layout calculation errors.

**Consequences:** The submit button or action bar is hidden behind the keyboard. Users on iOS (the primary device for food truck crew) cannot complete checklist items. The app feels broken.

**Prevention:**
- Never use `100vh` for full-screen layout. Use `-webkit-fill-available` as the height value, with `100vh` as a fallback: `height: -webkit-fill-available`
- Keep action bars (`position: fixed; bottom: 0`) minimal. Test with keyboard open on a real iOS device.
- Ensure all critical actions (submit, save, next item) are reachable by scrolling, not only visible in a fixed footer
- For temperature inputs and text notes, make sure the focused field scrolls into view with `el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })`
- Add `meta viewport` with `viewport-fit=cover` and use `env(safe-area-inset-bottom)` for bottom padding

**Warning signs:**
- Any fixed-position bottom bar tested only in a browser tab, never as an installed PWA
- Layout that passes desktop and Chrome mobile emulation but breaks on iPhone with Safari

**Phase:** Fill-out view (Phase 2) — test on a real device before declaring a field type complete

---

### Pitfall 6: Fail Trigger / Corrective Action UX That Interrupts Flow

**What goes wrong:** A fail trigger fires (e.g., temperature out of range), and the app immediately opens a modal or redirects to a corrective action screen. The crew member loses their place in the checklist, cannot see what item triggered it, and abandons the corrective action.

**Why it happens:** Copying desktop UX patterns (modal dialogs, full-page redirects) onto a mobile checklist that is designed for fast sequential completion.

**Consequences:** Corrective actions are skipped or filled in after the fact. The accountability value of the feature — documenting who did what when — is lost. Crew members learn to avoid the fields that trigger the interruption.

**Prevention:**
- Inline expansion, not modals: when a fail is detected, expand a corrective action sub-form inline below the triggering item. The checklist continues above it; the action is anchored to the item that triggered it.
- Never block checklist progression — corrective action should be completable immediately or flagged for follow-up, not a gate
- Show a persistent visual indicator (e.g., badge count) on the section header for uncompleted corrective actions, so crew can return to them
- The corrective action fields (what was done, who did it) must be as simple as possible — a single text field + timestamp is better than a 5-field sub-form

**Warning signs:**
- Corrective action implemented as a `window.location` redirect or a `display:block` overlay that covers the checklist
- A corrective action that requires more than 2 inputs from mobile keyboard

**Phase:** Conditional logic / fill-out view integration — address during the fail-trigger implementation sprint

---

## Moderate Pitfalls

---

### Pitfall 7: Day-of-Week Logic Implemented in the Wrong Layer

**What goes wrong:** Day-of-week conditions (`show Section C only on Mondays`) are evaluated in the template builder (where the condition is configured) rather than in the fill-out view (where the condition is evaluated at runtime). Or, the current day is read at page load rather than rechecked when a checklist is submitted.

**Why it happens:** It's tempting to embed "active today" logic into the template data itself, rather than separating condition definition from condition evaluation.

**Prevention:**
- Condition definitions live in the template: `{ type: "day-of-week", days: ["Monday", "Wednesday"] }`
- Condition evaluation happens in the fill-out view at render time, using `new Date().getDay()`
- Never mutate the template definition based on the current day — templates are immutable definitions, not runtime state
- For mock purposes, add a `__debugDay` override variable to test Monday vs Friday behavior without waiting for the actual day

**Warning signs:**
- Template JSON that contains `"active": true/false` fields that are set at save time
- Any reference to `new Date()` inside the template builder code

**Phase:** Template builder data model (Phase 1), enforced during fill-out view (Phase 2)

---

### Pitfall 8: Skip Logic That Doesn't Account for Skipped Values in Completion Tracking

**What goes wrong:** A section has 10 items. 3 are skipped due to skip logic conditions (e.g., "skip if previous answer is N/A"). The completion tracker shows "7/10 items completed" and flags the checklist as incomplete, even though the 3 skipped items were correctly skipped.

**Prevention:**
- Completion percentage denominator must be: all required items minus currently-skipped items
- Skipped items must be recorded in responses as `{ value: null, skipped: true, reason: "condition:field_abc123" }` — not absent
- "Submit" button enablement is based on: `requiredItems.every(id => responses[id]?.value !== undefined || responses[id]?.skipped === true)`

**Warning signs:**
- Completion logic counts total fields rather than required+applicable fields
- Skipped fields are simply absent from the responses object (indistinguishable from unanswered)

**Phase:** Fill-out view completion logic (Phase 2)

---

### Pitfall 9: Photo Capture Field Stalls on iOS After One Use

**What goes wrong:** `<input type="file" accept="image/*" capture="environment">` works once. If the user dismisses the camera without taking a photo, or takes a photo and then tries to re-capture, the input does not open the camera again. This is a documented iOS 12+ WebKit bug that persists into recent iOS versions for installed PWAs.

**Prevention:**
- Do not rely on `capture="environment"` for re-triggering. Instead, use `accept="image/*"` without `capture` — this opens the iOS share sheet which lets users choose camera or photo library and is more reliable for re-use
- After a successful capture, reset the input's value (`el.value = ''`) so the `change` event fires again next time
- Test photo re-capture on a real device, not emulator — this bug does not appear in Chrome DevTools device mode

**Warning signs:**
- Camera input tested only once per page load
- `capture="environment"` used without a reset strategy

**Phase:** Photo field implementation within fill-out view (Phase 2)

---

### Pitfall 10: Service Worker Caching Serving Stale App After Updates

**What goes wrong:** The existing PWA has a service worker. When the workflow pages are added, crew phones that installed the PWA earlier continue serving the old cached version — sometimes for days — because the service worker intercepts all fetches and returns stale assets. Crew members are unknowingly using an outdated checklist UI while the owner has deployed a fixed template builder.

**Prevention:**
- The existing `sw.js` cache key must be incremented on every deploy that changes the workflow pages (already partly addressed per git history — `ptr.js` references cache busting)
- Add the new workflow HTML files (`workflow.html`, `builder.html`) to the service worker's precache list explicitly
- Expose a visible app version string in the UI — even a build date in the footer — so crew can confirm they have the latest
- Consider a "new version available — tap to refresh" notification pattern using the service worker `waiting` state

**Warning signs:**
- Deploying a template builder fix and crew still seeing the old UI 24 hours later
- Service worker precache list hardcoded without workflow page filenames

**Phase:** Any phase that introduces new HTML pages — update sw.js cache list as part of the page delivery checklist

---

## Minor Pitfalls

---

### Pitfall 11: Multi-Column Layouts in the Builder That Break on Mobile

**What goes wrong:** The template builder shows field configuration options in a two-column layout (field type on left, options on right). On a 375px iPhone screen, columns either overflow or collapse in a way that makes tap targets unreachable.

**Prevention:** The existing project convention is single-column, 480px max-width, mobile-first. Apply it to the builder UI too. All builder controls go in a single-column stacked layout. Avoid side-by-side panels.

**Phase:** Template builder UI (Phase 1)

---

### Pitfall 12: Dropdown Menus for Field Type Selection on Mobile

**What goes wrong:** `<select>` elements for choosing field type (checkbox vs. yes/no vs. temperature) are technically functional but provide poor mobile UX — small hit targets, OS-native styling that clashes with the existing dark-mode design, and no icon support.

**Prevention:** Use a tappable card grid or segmented button row for field type selection in the builder. Reserve `<select>` only for options with more than ~6 choices. Match the interaction patterns already used in `purchasing.html`.

**Phase:** Template builder UI (Phase 1)

---

### Pitfall 13: Role-Based Tab Visibility Implemented Only in CSS

**What goes wrong:** The builder tab is restricted to certain roles. If visibility is controlled only with `display: none` (CSS), any user can open DevTools and remove the class to access the builder. On a food truck this is low risk, but it sets a bad precedent for the mock that will mislead the future backend implementation.

**Prevention:** For the mock, gate builder tab access with a JS check against the mock user object (`if (!currentUser.permissions.includes('workflow_builder')) return`). Use CSS to reflect the state, not enforce it. This trains the correct pattern for when real auth is added.

**Phase:** Template builder access gating (Phase 1)

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Template builder data model | Schema mismatch (Pitfall 3), DOM as source of truth (Pitfall 1) | Define field ID scheme and state shape before writing any UI |
| Conditional logic engine | Infinite evaluation loop (Pitfall 2), hidden fields keeping stale values | Build a condition evaluator with topological sort and value-clear-on-hide before wiring to UI |
| Fail trigger UX | Modal/redirect interruption (Pitfall 6) | Prototype inline expansion on mobile before committing to the interaction pattern |
| Fill-out view | Keyboard/viewport breakage (Pitfall 5), completion tracking (Pitfall 8) | Real iOS device test after every field type added |
| Photo capture field | Single-use input bug (Pitfall 9) | Test re-capture on a real iPhone in standalone mode specifically |
| Day-of-week conditions | Logic in wrong layer (Pitfall 7) | Enforce: template stores definition, fill-out view evaluates at render time |
| New HTML page deploy | Stale service worker cache (Pitfall 10) | Update sw.js cache key and precache list on every page-adding PR |
| Builder tab access | CSS-only gating (Pitfall 13) | JS permission check required, CSS reflects it |

---

## Sources

- [JotForm: Conditional logic not triggering when value changed from JS](https://www.jotform.com/answers/12198181-conditional-logic-not-triggering-when-value-changes-from-javascript)
- [Form.io: Form JSON Schema vs Submission — separation of definition from data](https://form.io/form-json-schema-vs-submission/)
- [Form.io versioning: versioning against submissions](https://github.com/formio/formio/issues/123)
- [LogRocket: Dynamically creating JS elements with event handlers — delegation pattern](https://blog.logrocket.com/dynamically-create-javascript-elements-event-handlers/)
- [PWA bugs: iOS input type file camera issue (iOS 12.2+)](https://github.com/PWA-POLICE/pwa-bugs/issues/12)
- [MagicBell: PWA iOS Limitations and Safari Support 2026](https://www.magicbell.com/blog/pwa-ios-limitations-safari-support-complete-guide)
- [Martijn Hols: How to detect on-screen keyboard in iOS Safari](https://martijnhols.nl/blog/how-to-detect-the-on-screen-keyboard-in-ios-safari)
- [Fixing Safari mobile resizing bug — viewport height issues](https://medium.com/@krutilin.sergey.ks/fixing-the-safari-mobile-resizing-bug-a-developers-guide-6568f933cde0)
- [Namastedev: State management strategies without frameworks](https://namastedev.com/blog/state-management-strategies-without-frameworks-vanilla-patterns-that-scale/)
- [web.dev: Capturing images from the user — input type file patterns](https://web.dev/media-capturing-images/)
- [MyFieldAudits: Food safety checklist design — complexity and corrective action](https://www.myfieldaudits.com/blog/food-safety-management-system-audit-checklist)
- [FormsonFire: 13 mobile form design best practices](https://www.formsonfire.com/blog/mobile-form-design)
