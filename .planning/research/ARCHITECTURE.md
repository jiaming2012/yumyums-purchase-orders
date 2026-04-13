# Architecture Patterns: Workflow/Checklist Builder

**Domain:** Mobile-first form builder + checklist fill-out engine (Lumiform-style)
**Researched:** 2026-04-12
**Confidence:** HIGH — patterns are well-established across multiple production systems

---

## Recommended Architecture

The system has two distinct runtime modes that share a single data schema: **Build mode** (template editor, owner-only) and **Fill mode** (checklist runner, all crew). These are not the same UI rendered differently — they are different interaction models on the same data.

```
┌─────────────────────────────────────────────────────────────────┐
│                        workflows.html                           │
│                                                                 │
│  ┌─────────────────────┐   ┌─────────────────────────────────┐  │
│  │   Tab 1: Fill Mode  │   │   Tab 2: Build Mode (gated)     │  │
│  │                     │   │                                 │  │
│  │  AssignmentList     │   │  TemplateList                   │  │
│  │       ↓             │   │       ↓                         │  │
│  │  ChecklistRunner    │   │  TemplateEditor                 │  │
│  │       ↓             │   │    ├── SectionEditor            │  │
│  │  ConditionEvaluator │   │    ├── FieldEditor              │  │
│  │       ↓             │   │    └── ConditionEditor          │  │
│  │  ResponseCollector  │   │                                 │  │
│  └─────────────────────┘   └─────────────────────────────────┘  │
│                                                                 │
│                    Shared: MOCK_DATA (JS arrays)                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Data Model

The entire system rests on two JSON shapes: **Template** (what to do) and **Response** (what was done). They must stay separate.

### Template Schema

A template is the blueprint. It is never mutated when someone fills it out.

```javascript
// Template — the unchanging definition
const template = {
  id: "tpl_morning_open",
  name: "Morning Opening Checklist",
  description: "Complete before first service",

  // Day-of-week scheduling: which days this template is active
  // Days: 0=Sun, 1=Mon, 2=Tue, 3=Wed, 4=Thu, 5=Fri, 6=Sat
  active_days: [1, 2, 3, 4, 5],   // Mon–Fri

  // Role access: who can fill this out
  assignable_to: ["admin", "manager", "team_member"],

  sections: [
    {
      id: "sec_equipment",
      title: "Equipment Check",
      order: 0,

      // Section-level condition: show/hide the whole section
      // null = always show
      condition: null,

      fields: [
        {
          id: "fld_grill_temp",
          type: "number",         // see Field Types below
          label: "Grill surface temperature (°F)",
          required: true,
          order: 0,
          config: {
            unit: "°F",
            min: 350,
            max: 500,
            placeholder: "Enter temp"
          },

          // Fail trigger: what counts as a failed response
          fail_trigger: {
            type: "out_of_range",  // "out_of_range" | "equals" | "not_equals"
            min: 350,
            max: 500
          },

          // Corrective action: shown only when fail_trigger fires
          corrective_action: {
            label: "Corrective action taken",
            type: "text",
            required: true,
            placeholder: "Describe what you did to fix it"
          },

          // Field-level condition: when to show this field
          // null = always show
          condition: null
        },
        {
          id: "fld_grill_clean",
          type: "checkbox",
          label: "Grill grates cleaned",
          required: true,
          order: 1,
          config: {},
          fail_trigger: null,
          corrective_action: null,
          condition: null
        },
        {
          id: "fld_grill_issue_note",
          type: "text",
          label: "Describe the issue",
          required: true,
          order: 2,
          config: { placeholder: "What did you find?" },
          fail_trigger: null,
          corrective_action: null,

          // Skip logic: only show if grill_clean was unchecked
          condition: {
            field_id: "fld_grill_clean",
            operator: "equals",      // "equals" | "not_equals" | "lt" | "gt" | "in_range"
            value: false
          }
        }
      ]
    },
    {
      id: "sec_food_temp",
      title: "Food Safety",
      order: 1,
      condition: null,
      fields: [
        {
          id: "fld_cold_storage_temp",
          type: "number",
          label: "Walk-in cooler temp (°F)",
          required: true,
          order: 0,
          config: { unit: "°F", min: 33, max: 40 },
          fail_trigger: { type: "out_of_range", min: 33, max: 40 },
          corrective_action: {
            label: "Action taken",
            type: "text",
            required: true,
            placeholder: "e.g. called repair, moved product to backup cooler"
          },
          condition: null
        },
        {
          id: "fld_photo_storage",
          type: "photo",
          label: "Photo of storage area",
          required: false,
          order: 1,
          config: {},
          fail_trigger: null,
          corrective_action: null,
          condition: null
        }
      ]
    }
  ]
};
```

### Field Types

| Type | Input Element | Config Keys | Notes |
|------|--------------|-------------|-------|
| `checkbox` | `<input type="checkbox">` | — | Boolean pass/fail. Most common type. |
| `yes_no` | Two buttons (Yes/No) | — | Better mobile UX than checkbox for binary. |
| `number` | `<input type="number">` | `unit`, `min`, `max` | Enables out-of-range fail triggers. |
| `text` | `<textarea>` or `<input>` | `placeholder`, `maxlength` | Free-form notes. |
| `photo` | `<input type="file" accept="image/*" capture="environment">` | — | Camera capture on mobile. Mock uses placeholder. |
| `timestamp` | Auto-populated | — | Records `Date.now()` when field is reached; display only. |
| `temperature` | Same as `number` with °F/°C toggle | `unit`, `min`, `max` | Semantic alias for number; renders unit label. |

### Response Schema

A response is one filled-out instance of a template. It is a separate object — never a mutated template.

```javascript
// Response — created when crew starts filling out a checklist
const response = {
  id: "resp_abc123",
  template_id: "tpl_morning_open",
  template_name: "Morning Opening Checklist",  // snapshot, not reference
  started_at: "2026-04-12T09:00:00Z",
  completed_at: null,                          // null until submitted
  completed_by: {
    user_id: "usr_jamal",
    display_name: "Jamal"
  },
  status: "in_progress",  // "in_progress" | "completed" | "completed_with_fails"

  // Answers keyed by field_id
  answers: {
    "fld_grill_temp": {
      value: 320,               // the raw input
      failed: true,             // fail_trigger fired
      corrective_action: "Turned up heat, waited 10 min, re-checked at 375°F",
      answered_at: "2026-04-12T09:03:00Z",
      answered_by: "usr_jamal"  // for multi-person fill-outs (future)
    },
    "fld_grill_clean": {
      value: false,
      failed: false,
      corrective_action: null,
      answered_at: "2026-04-12T09:04:00Z",
      answered_by: "usr_jamal"
    },
    "fld_grill_issue_note": {
      value: "Grates had residue from yesterday",
      failed: false,
      corrective_action: null,
      answered_at: "2026-04-12T09:05:00Z",
      answered_by: "usr_jamal"
    }
    // Fields that were hidden by conditions are absent from answers
    // Fields not yet answered are absent from answers
  }
};
```

**Key principle:** Fields hidden by conditions do not appear in `answers`. The fill-out runtime skips them entirely. This keeps the response clean and makes replaying the fill-out deterministic.

---

## Component Boundaries

### Fill Mode Components

| Component | Responsibility | Inputs | Outputs |
|-----------|---------------|--------|---------|
| `AssignmentList` | Show templates assigned for today; filter by day-of-week | `MOCK_TEMPLATES[]`, current day | Selected template ID |
| `ChecklistRunner` | Render one template for fill-out; manage answer state | Template object, initial answers (empty) | Completed response object |
| `ConditionEvaluator` | Determine which fields/sections are visible | Current answers, field/section condition objects | `Set<field_id>` of visible fields |
| `FailTriggerEvaluator` | Determine if a given answer triggers a fail | Field's `fail_trigger` config, answer value | `boolean` |
| `FieldRenderer` | Render one field as the appropriate input element | Field object, current answer, visible state | DOM element + change events |
| `ResponseCollector` | Accumulate answers, track completion, produce final response | Answer events from FieldRenderers | Response object on submit |

### Build Mode Components

| Component | Responsibility | Inputs | Outputs |
|-----------|---------------|--------|---------|
| `TemplateList` | Show all templates; entry point to editor | `MOCK_TEMPLATES[]` | Selected template or "new" |
| `TemplateEditor` | Outer shell: name, description, active_days, assignable_to | Template object | Updated template object |
| `SectionEditor` | Add/remove/reorder sections; edit section title and condition | Template sections array | Updated sections |
| `FieldEditor` | Add/remove/reorder fields; configure all field properties | Field object | Updated field object |
| `ConditionEditor` | UI for setting `condition` on a field or section | Available field IDs (preceding fields), current condition | Condition object |
| `FieldTypePicker` | Pick field type; updates config defaults on change | Current type | New type + default config |
| `TemplatePreview` | Render the template in read-only fill-mode appearance | Template object | Visual only |

### Shared Utilities (no UI, pure functions)

| Utility | Responsibility |
|---------|---------------|
| `evaluateCondition(condition, answers)` | Returns `true` if the field/section should be visible |
| `evaluateFailTrigger(fail_trigger, value)` | Returns `true` if the answer is a failure |
| `getVisibleFields(template, answers)` | Returns flat list of visible fields given current answers |
| `isDayActive(template, dayOfWeek)` | Returns `true` if template runs on this day |
| `computeCompletionStatus(template, answers)` | Returns `{ total, answered, failed, complete }` |

---

## Data Flow

### Fill Mode Flow

```
User taps checklist
        ↓
AssignmentList filters MOCK_TEMPLATES by active_days[today]
        ↓
ChecklistRunner initializes: { answers: {}, template: selected }
        ↓
For each render cycle:
  ConditionEvaluator(template.sections, answers) → visible field IDs
        ↓
  FieldRenderer renders each visible field
        ↓
User answers a field
        ↓
ResponseCollector.recordAnswer(field_id, value)
  → FailTriggerEvaluator fires → if fail: show corrective_action input
  → answers state updated
  → re-render (ConditionEvaluator runs again → may show/hide fields)
        ↓
All required visible fields answered
        ↓
ResponseCollector.submit() → response object written to MOCK_RESPONSES[]
        ↓
ChecklistRunner shows completion screen
```

### Build Mode Flow

```
Owner opens builder tab
        ↓
TemplateList shows MOCK_TEMPLATES[]
        ↓
Owner selects template (or creates new)
        ↓
TemplateEditor renders:
  - top-level fields (name, days, roles)
  - SectionEditor for each section
    - FieldEditor for each field
      - ConditionEditor (only if fields exist before this one in the section,
        or prior sections exist — conditions can only reference preceding fields)
        ↓
Any change → update MOCK_TEMPLATES[] in-place → TemplatePreview re-renders
        ↓
"Save" is implicit (mock state; real version would POST to API)
```

### Condition Evaluation Logic

Conditions are deliberately simple — no rule engines, no `new Function()` eval. Plain JS object comparison is sufficient for this domain:

```javascript
function evaluateCondition(condition, answers) {
  if (!condition) return true;  // no condition = always show

  const answer = answers[condition.field_id];
  if (!answer) return false;    // referenced field not yet answered = hide

  const val = answer.value;

  switch (condition.operator) {
    case "equals":     return val === condition.value;
    case "not_equals": return val !== condition.value;
    case "lt":         return val < condition.value;
    case "gt":         return val > condition.value;
    case "in_range":   return val >= condition.min && val <= condition.max;
    default:           return true;
  }
}
```

**Why not `new Function()` eval:** The codebase is mocks-only with hardcoded data. Eval-based approaches (seen in some generic form builders) add security surface area and complexity not needed here. The condition schema above covers all required use cases (skip logic, fail visibility) with zero eval.

**Condition reference constraint:** A field's condition can only reference fields that appear before it (earlier in the same section, or in a prior section). This is enforced in the ConditionEditor UI by only showing eligible field IDs in the picker. This avoids circular dependencies and makes evaluation order deterministic (top-to-bottom, left-to-right).

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Mutating the template during fill-out

**What goes wrong:** Storing answers inside the template object (e.g., `field.value = userInput`). This makes it impossible to fill out the same template twice without resetting state, breaks the builder, and conflates "what to do" with "what was done."

**Prevention:** Always keep `template` and `response` as separate objects. The template is read-only during fill-out.

### Anti-Pattern 2: Conditions referencing forward fields

**What goes wrong:** Field A is conditional on Field B, but B appears after A in the section. Evaluation order becomes undefined.

**Prevention:** ConditionEditor only offers fields that appear before the current field as targets. Enforce this in the builder UI, not just docs.

### Anti-Pattern 3: Flattening sections before render

**What goes wrong:** Converting `template.sections[].fields[]` into a flat array early, then trying to re-group for display. You lose section boundaries needed for headers and section-level conditions.

**Prevention:** Keep the nested `sections → fields` structure throughout rendering. The renderer iterates sections, not fields.

### Anti-Pattern 4: One HTML file per tool page trying to share a module

**What goes wrong:** Extracting `evaluateCondition` into a shared `utils.js` that all pages import — but since this is a no-build, static project, shared module state (like MOCK_DATA) becomes tricky across page navigations.

**Prevention:** For this mock phase, keep all data and logic inside `workflows.html` as a self-contained script block, matching the pattern in `purchasing.html`. When the backend arrives, replace MOCK_DATA with fetch calls — the component structure stays the same.

### Anti-Pattern 5: Encoding conditions as code strings

**What goes wrong:** Storing conditions as JavaScript code strings (`"return answers.fld_x === true"`) and evaluating with `new Function()`. Looks flexible, becomes a security and maintainability problem.

**Prevention:** Use the structured condition object schema defined above. All needed operators (equals, not_equals, lt, gt, in_range) are covered without eval.

---

## Suggested Build Order

Dependencies flow strictly downward. Each layer can be built and tested before the next.

### Layer 1: Data Foundation (no UI)

Build first because everything else depends on these shapes being stable.

1. Define `MOCK_TEMPLATES[]` with 2–3 real templates in the final schema
2. Define `MOCK_RESPONSES[]` as an initially-empty array
3. Implement `evaluateCondition()`, `evaluateFailTrigger()`, `getVisibleFields()` as plain JS functions
4. Manually test the pure functions in the browser console before building UI

### Layer 2: Fill Mode (Tab 1)

Build before the builder because it validates the data model is usable. If filling out is awkward, fix the data model before investing in the editor.

5. `AssignmentList` — renders today's templates from MOCK_TEMPLATES filtered by active_days
6. `FieldRenderer` — renders each field type correctly (start with checkbox + yes_no; add number/text/photo iteratively)
7. `ChecklistRunner` — wires FieldRenderer + ConditionEvaluator + ResponseCollector
8. Fail trigger + corrective action input (inline, appears below the failed field)
9. Completion screen

### Layer 3: Build Mode (Tab 2)

Build after fill mode works end-to-end on hardcoded data.

10. `TemplateList` — list view + "New template" button
11. `TemplateEditor` top-level fields (name, active_days, roles)
12. `SectionEditor` — add/remove/reorder sections
13. `FieldEditor` — add/remove fields, pick type, configure label/required
14. `ConditionEditor` — the most complex piece; build last when field IDs are stable
15. `TemplatePreview` — live re-render of fill mode as you build (nice-to-have, not blocking)

### Layer 4: Integration

16. Wire builder saves into MOCK_TEMPLATES[] so new templates immediately appear in Fill Mode
17. Wire fill-out submission into MOCK_RESPONSES[] (used by future history/reporting view)
18. Role gate: hide Tab 2 if current mock user role lacks builder permission (read from MOCK_CURRENT_USER)

---

## Integration with Existing PWA Shell

The workflow tool follows the same conventions as `purchasing.html`:

- Single HTML file (`workflows.html`) with all styles, markup, and script inline
- Back link to `index.html` using the existing `.back::before` chevron pattern
- Two-tab layout using the existing `.tabs` + `.tabs button.on` pattern from purchasing
- Same CSS variable set (`--bg`, `--card`, `--txt`, `--mut`, `--brd`, `--info-bg`, `--info-tx`)
- `if ('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js')` at bottom
- `<script src="ptr.js"></script>` for pull-to-refresh

The launcher tile in `index.html` gets an `<a class="tile active" href="workflows.html">` entry — same as the Purchasing tile.

**Tab 2 gating** (builder access): Read from `MOCK_CURRENT_USER.role` and compare against `MOCK_BUILDER_ROLES`. In the mock, the tab simply doesn't render if the role lacks access. The data model for this is already established in `user-management-api.md` via `app_permissions`.

---

## Scalability Considerations

This architecture is intentionally scoped to the mock phase. The component and data model boundaries are designed so the transition to a real backend is a data layer swap, not an architectural rewrite.

| Concern | Mock Phase | With Backend |
|---------|-----------|--------------|
| Template storage | `const MOCK_TEMPLATES = [...]` in script | `GET /api/templates` |
| Response storage | `const MOCK_RESPONSES = []` push on submit | `POST /api/responses` |
| Auth/roles | `const MOCK_CURRENT_USER = {role: "manager"}` | Session token, decoded on page load |
| Offline fill-out | Not needed for mocks | IndexedDB queue + background sync |
| Photo capture | `<input type="file">` renders file picker | Same element; upload to object storage |
| Day-of-week filter | `new Date().getDay()` | Same |

---

## Sources

- [Frontend System Design: Building a Dynamic Form Builder from Scratch](https://shivambhasin29.medium.com/mastering-frontend-system-design-building-a-dynamic-form-builder-from-scratch-0dfdd78d31d6) — component decomposition pattern (MEDIUM confidence, single source)
- [JSON Forms Architecture](https://jsonforms.io/docs/architecture/) — template vs instance separation, renderer model (HIGH confidence, official docs)
- [Form.io Logic & Conditions](https://help.form.io/userguide/forms/form-building/logic-and-conditions) — conditional logic patterns (MEDIUM confidence)
- [Dynamic Form Builder Part 3: Advanced Logic](https://medium.com/@charanvinaynarni/dynamic-form-builder-part-3-advanced-logic-apis-conditional-rendering-no-ai-involved-eabdfc42579c) — condition evaluation implementation (MEDIUM confidence)
- [Lumiform Template Builder](https://help.lumiformapp.com/en/knowledge/template-builder-lumiform) — product reference for fail triggers / corrective actions / skip logic (MEDIUM confidence)
- [OxMaint Inspection Management](https://www.oxmaint.com/blog/post/inspection-management-software-digital-checklists-compliance) — fail trigger → corrective action pattern (MEDIUM confidence)
- Existing codebase: `purchasing.html`, `index.html`, `docs/user-management-api.md` — conventions and data model patterns (HIGH confidence, primary source)
