# UI Bug Post-Mortem — 2026-04-26 Session

Patterns and root causes from bugs found during this session. Use these to prevent similar issues in future development.

---

## 1. Silent async handler failures

**Bug:** Onboarding Builder buttons stopped working — typing in inputs still worked but clicking any button did nothing.

**Root cause:** The `builder-body` click handler is an `async function`. When `obBuilderState.localCopy` was null, accessing `.sections` threw a `TypeError` that was silently swallowed as an unhandled promise rejection. The input handler had a null guard (`if (!obBuilderState.localCopy) return;`) but the click handler did not.

**Pattern:** Async event delegation handlers silently swallow errors. A thrown error in one branch doesn't crash the page or show any visible feedback — the user just sees "nothing happens."

**Prevention:**
- Every async event delegation handler should have a null/state guard at the top, matching the most defensive sibling handler.
- Consider wrapping async delegation handlers in try/catch with `console.error` so failures are visible in logs.

---

## 2. Frontend-only data structures never persisted

**Bug:** Onboarding sub-items (e.g., "Set timer for 15 minutes") disappeared after saving and reopening a template.

**Root cause:** Sub-items existed only in frontend JavaScript state. The backend `CreateItemInput` Go struct had no `sub_items` field, the DB had no `ob_sub_items` table, and Go silently ignores unknown JSON fields during unmarshaling. No error was ever raised — the data was silently dropped.

**Pattern:** Frontend can model data structures that the backend silently discards. Go's `json.Unmarshal` ignores unknown fields by default, so there's no signal that data is being lost.

**Prevention:**
- When adding a new data concept to the frontend (sub-items, tags, metadata), always trace the full round-trip: frontend state -> API serialization -> Go struct -> SQL insert -> SQL select -> Go struct -> JSON response -> frontend hydration.
- Add a persistence regression test immediately: create -> save -> reload -> assert data is still there.

---

## 3. DB CHECK constraint violations from missing field defaults

**Bug:** Saving a workflow template with sub-steps failed with `checklist_fields_type_check` constraint violation.

**Root cause:** The frontend's "add sub-step" action created `{ id: ..., label: '' }` without a `type` field. The backend inserted this as `type = ""` which violated `CHECK (type IN ('checkbox','yes_no','text','temperature','photo'))`.

**Pattern:** When child records reuse the same DB table as parent records (sub-steps stored in `checklist_fields` alongside regular fields), the child must satisfy all the same constraints. The frontend only set required fields for the parent context, not the child.

**Prevention:**
- When creating child/nested records that share a table with parents, explicitly set all constrained fields with sensible defaults.
- Add the default at creation time AND in the serialization layer (belt and suspenders) so pre-existing in-memory objects also get the default.

---

## 4. Foreign key breaks after full-replace update strategy

**Bug:** Rejecting a submitted checklist item returned a 500 — `submission_rejections_field_id_fkey` violation.

**Root cause:** `replaceTemplate` deletes all sections/fields and re-inserts with new UUIDs on every template save. Submitted checklists store a `template_snapshot` with the OLD field UUIDs. When an approver rejects a field, the rejection references the old UUID which no longer exists in `checklist_fields`.

**Pattern:** Full-replace update strategies (delete all + re-insert) break foreign keys from other tables that reference the replaced rows. This is fine for parent->child relationships (CASCADE handles it) but breaks sibling/cross-table references.

**Prevention:**
- Tables that reference entities subject to full-replace updates should NOT use FK constraints, or should use `ON DELETE SET NULL`.
- Alternatively, use an upsert/diff strategy (like the onboarding `UpdateTemplate`) that preserves IDs instead of full replace.
- When a table stores snapshot/historical references (rejections, audit logs), those FKs should be relaxed or removed since the referenced data may be legitimately deleted.

---

## 5. Progress count mismatch with sub-step fields

**Bug:** Workflow checklist showed "2 of 3 items complete" when only 1 sub-step was checked, because the parent field was counted as "answered."

**Root cause:** `countCompletedFields()` checked `FIELD_RESPONSES[f.id] !== undefined`. For fields with sub-steps, toggling the first sub-step creates the parent response with `{ value: false, sub_steps: {...} }`. The response EXISTS but `value` is `false` — the field is not truly complete. The count treated it as complete.

**Pattern:** Progress counting that checks for response existence (`!== undefined`) breaks when partial responses are valid states. Sub-step fields have a three-state lifecycle: no response -> partial (some subs done, value=false) -> complete (all subs done, value=true).

**Prevention:**
- For fields with sub-steps/children, always check `response.value === true` not just `response !== undefined`.
- Extract counting logic into a shared helper (`isFieldAnswered()`) used by all views (list, runner, progress bar) to avoid divergence.
- When unchecking all sub-steps, clean up the parent response entirely (`delete FIELD_RESPONSES[id]`) rather than leaving a stale `{ value: false }`.

---

## 6. Query JOIN gaps when adding new entity types

**Bug:** Onboarding sub-item checked state didn't persist across page navigations — checking sub-items worked but reopening the training showed them unchecked.

**Root cause:** `GetHireTraining` builds a `progressMap` from a query that JOINs `ob_progress.item_id` through `ob_items` and `ob_video_parts` to find the section. When `sub_item` progress was added, the query didn't JOIN through `ob_sub_items`, so sub-item progress rows were silently excluded (the JOIN to `ob_sections` failed, dropping the row).

**Pattern:** Polymorphic ID columns (`item_id` references multiple tables depending on `progress_type`) require a LEFT JOIN for EACH possible target table. Adding a new target type requires updating every query that resolves the polymorphic reference.

**Prevention:**
- When adding a new progress/reference type to a polymorphic ID column, grep for ALL queries that JOIN through that column and add the new JOIN path.
- Search for the table name (`ob_progress`) and audit every query that filters by section/template — each one likely needs the new JOIN.

---

## 7. Naming inconsistency across the system

**Bug:** "Purchasing" appeared in the users/permissions page while the tile said "Purchase Orders."

**Root cause:** The display name was set in two places — `backend/cmd/seed/main.go` (initial seed) and `backend/internal/db/db.go` (startup upsert). Both had the old name "Purchasing." The tile in `index.html` had already been renamed but the database records were not updated.

**Pattern:** Display names stored in the database diverge from UI labels when only one side is updated. The DB name is the source of truth for dynamic UIs (permissions page), while the static HTML has its own label.

**Prevention:**
- When renaming a feature, grep the entire codebase for the old name — including seed files, migration data, and DB upsert statements.
- The startup upsert (`ON CONFLICT DO UPDATE`) is the fix mechanism: it automatically propagates name changes on next server restart.

---

## 8. Section defaults that don't match user expectations

**Bug:** New workflow sections defaulted to "Custom days" instead of "Same as schedule." New onboarding sections defaulted to `requires_sign_off: false`.

**Root cause:** `createNewSection()` in workflows.html copied the schedule's active days into the section condition, making it "Custom days." `createNewOBSection()` in onboarding.html set `requires_sign_off: false`.

**Pattern:** Defaults are chosen by developers based on what's technically neutral, not what's operationally expected. "No condition" (null) is the correct default for inheriting schedule days. Sign-off is almost always required for onboarding.

**Prevention:**
- When adding create-new functions, ask: "What would the user have to change 90% of the time?" Set that as the default.
- Prefer inheriting from parent context (null/inherit) over copying parent values into the child.

---

## 9. Attribution rendered outside its visual container

**Bug:** Sub-step completion attribution ("Ben J. - 1:03p") appeared below the divider line separating sub-steps, instead of above it.

**Root cause:** The attribution HTML was concatenated AFTER the `sub-step-row` div which had `border-bottom`. The attribution div was a sibling rendered below the border.

**Pattern:** When a list item has both content and metadata (attribution, timestamps), the metadata must be inside the bordered container or the border must move to below the metadata.

**Prevention:**
- When adding attribution/metadata to list items, place it INSIDE the row container, or remove the row's border-bottom and add it to the attribution div instead.
- Visual rule: all content that belongs to an item should appear between the item's top and bottom borders.

---

## 10. Progress count divergence across views

**Bug:** "Monday setup" showed 1/2 items in the list view but the runner showed 2/2 when opened. "Setup Checklist" showed 0/5 items but only 3 fields were actually visible due to conditional logic.

**Root cause:** Five independent code paths compute "X of Y items complete," each with different rules:

| Location | Applies visibility filters? | Handles sub-steps? | Data source |
|----------|---------------------------|--------------------|--------------------|
| List view row | No | Yes | FIELD_RESPONSES vs live template |
| List view aggregate | No | Yes | Sum of rows |
| Runner progress bar | Yes | Yes | FIELD_RESPONSES vs live template |
| Runner updateProgress() | Yes | Yes | countCompletedFields() |
| Submitted list view | No | N/A | Submission responses vs snapshot |

The list view counts ALL fields (`tpl.sections.flatMap(s => s.fields)`), including conditionally-hidden ones. The runner filters by `isSectionVisible()` and `isFieldVisible()`. So the list says "0/5" while the runner says "0/3" for the same checklist.

Additionally, after a template update (`replaceTemplate`), live field UUIDs change but draft/submission responses reference old UUIDs, causing further mismatches.

**Pattern:** When the same metric is computed independently in multiple places with subtly different logic, they inevitably diverge. Each fix to one view doesn't propagate to others.

**Prevention:**
- **Single function:** All views that display progress must call ONE shared `getProgress(tpl, submission)` function that returns `{ answered, total }`.
- **Always filter visibility:** The denominator must apply `isSectionVisible()` and `isFieldVisible()` — hidden fields should never inflate the total.
- **Snapshot-first for submitted checklists:** Any read-only view (list, runner, approvals) should derive data from the snapshot + responses, never from the live template which may have changed.
- **Invariant test:** Test that the list view count matches the runner count for the same checklist, rather than testing each view in isolation.

---

## 11. Sentinel values in skip logic not handled at runtime

**Bug:** A checkbox field configured with "Show this field only when: Describe starting conditions → Not empty" never appeared, even after the text field was filled in and synced.

**Root cause (two layers):**

1. **Sentinel value `_notempty` not interpreted.** The builder dropdown stores "Not empty" as `{ operator: 'not_equals', value: '_notempty' }`. But `isFieldVisible()` compared the actual response value (e.g., `"Lets talk about money"`) against the literal string `"_notempty"` — which never matches. The sentinel was never expanded into a runtime check like "value is non-empty."

2. **Stale field_id in condition.** The template was saved before the `replaceTemplate` ID remapping fix. The condition's `field_id` referenced a UUID that no longer existed, so `FIELD_RESPONSES[field_id]` was always `undefined`, and the field was hidden regardless of the sentinel issue.

**Why the regression test didn't catch it:** The test used raw API conditions `{ operator: 'not_equals', value: '' }` instead of `{ operator: 'not_equals', value: '_notempty' }`. It bypassed the builder's dropdown which stores the sentinel. The test exercised a code path that works (comparing against empty string) while the real UI uses a different code path (the sentinel).

**Pattern:** When a UI component stores sentinel/magic values (like `_notempty`, `_any`, `_none`) to represent abstract concepts, the runtime evaluator must handle those sentinels explicitly. If the evaluator just does raw value comparison, sentinels become unreachable conditions.

**Prevention:**
- **Test through the UI when possible.** Tests that bypass UI interactions (setting conditions via API) miss the actual data shapes the UI produces. For skip logic, the test should select from the builder dropdown, not construct raw JSON.
- **Document sentinel values.** When a dropdown maps display labels to stored values, document the mapping and ensure the runtime evaluator has explicit handling for each sentinel.
- **Round-trip assertions.** After setting a condition via the builder UI, assert the stored condition matches the expected shape (including sentinels), then assert runtime behavior separately.

---

## Recommendations for improving UI test robustness

### 1. Prefer UI-driven tests over API-constructed data
Tests that construct template data via raw API calls bypass the actual code paths users exercise. The skip logic bug wasn't caught because the test set `value: ''` directly, while the real UI stores `value: '_notempty'`. When a feature involves a builder UI that produces data, the test should drive the builder.

### 2. Test the full user flow, not isolated steps
Instead of testing "does the runner show a field," test the complete flow: "user configures skip logic in builder → saves → opens checklist → fills dependent field → conditional field appears." This catches bugs at every handoff point (builder → save → load → runtime eval).

### 3. Assert intermediate data shapes
When a test sets up conditions via API, assert what the stored condition actually looks like after save+reload. For example: after saving a template with skip logic, reload it and assert `field.condition.value === '_notempty'` and `field.condition.field_id` is a valid UUID that exists in the template. This catches stale ID and sentinel issues before testing runtime behavior.

### 4. Test with real template saves, not just creates
Many bugs only appear after `replaceTemplate` (update), not `insertTemplate` (create). Tests should create → save → update → reload to exercise the full-replace logic that generates new UUIDs.

### 5. Pair every sentinel value with a runtime test
For every value in a dropdown's option list (e.g., `_notempty`, `true`, `false`, `''`), write a test that verifies the runtime evaluator handles it correctly. Sentinel values are an implicit contract between the builder and the runner — both sides need tests.
