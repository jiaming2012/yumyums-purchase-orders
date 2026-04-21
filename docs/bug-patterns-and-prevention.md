# Bug Patterns and Prevention Guide

Recurring bug categories discovered during development of a Go + Postgres + vanilla JS PWA. Each pattern includes the root cause, how many times it appeared, and a concrete prevention rule.

---

## Pattern 1: Polymorphic ID Column Mismatch

**Frequency:** 6 occurrences (most common)

**Symptom:** Data is saved successfully but silently dropped on read. Writes return 200, but after page reload, some items appear unchecked/missing.

**Root cause:** A column stores IDs from multiple tables (e.g., `progress.item_id` can reference either `items.id` or `video_parts.id`). Queries use `INNER JOIN items ON items.id = progress.item_id`, which silently excludes rows where `item_id` is a video part UUID — it doesn't exist in the `items` table.

**Example:**
```sql
-- BUG: Only finds checkbox progress, silently drops video part progress
SELECT p.item_id FROM progress p
JOIN items i ON i.id = p.item_id  -- video_part IDs don't exist in items!
WHERE p.hire_id = $1

-- FIX: LEFT JOIN both possible source tables
SELECT p.item_id FROM progress p
LEFT JOIN items i ON i.id = p.item_id
LEFT JOIN video_parts vp ON vp.id = p.item_id
LEFT JOIN items vp_parent ON vp_parent.id = vp.item_id
JOIN sections s ON s.id = COALESCE(i.section_id, vp_parent.section_id)
WHERE p.hire_id = $1
```

**Prevention:**
- Document polymorphic columns with a comment: `-- polymorphic: references items.id OR video_parts.id`
- Always use LEFT JOINs against ALL possible source tables
- Create a helper function/query fragment for repeated polymorphic patterns
- **Test the read path, not just the write path.** A test that saves data and then reads it back would catch this immediately.

---

## Pattern 2: Incomplete Migration Propagation

**Frequency:** 4 occurrences

**Symptom:** Server returns 500 with `column "old_name" does not exist` on endpoints that weren't updated after a migration.

**Root cause:** SQL column renames (e.g., `role` → `roles`, `display_name` → `first_name/last_name`) don't break compilation because SQL queries are strings in Go. The compiler can't check them. Developers update the obvious handlers but miss secondary queries in other packages.

**Example:**
```go
// Migration: ALTER TABLE users RENAME COLUMN role TO roles, change TEXT to TEXT[]

// handler.go — updated ✓
rows, _ := pool.Query(ctx, `SELECT roles FROM users WHERE id = $1`, userID)

// sync/ops.go — forgotten ✗ (still references old column)
rows, _ := pool.Query(ctx, `... AND u.role = ta.assignee_id ...`)
// Runtime error: column "role" does not exist
```

**Prevention:**
- After every migration that renames or removes a column, run:
  ```bash
  grep -rn "old_column_name" backend/ --include="*.go"
  ```
- Fix EVERY match — no exceptions
- Add migration notes in the SQL file:
  ```sql
  -- MIGRATION NOTE: column "role" renamed to "roles"
  -- After applying, grep all Go files for ".role" and update to ".roles"
  ```
- Consider a CI step that runs all tests against a freshly migrated database

---

## Pattern 3: Optimistic UI Without Server Reconciliation

**Frequency:** 3 occurrences

**Symptom:** Data appears correct on the device that made the change, but other devices (or the same device after reload) show different values. Attribution shows the wrong user.

**Root cause:** Frontend makes optimistic updates for responsiveness (e.g., sets `answeredBy: getCurrentUser()` immediately on click) but never replaces the optimistic value with the server's authoritative response.

**Example:**
```javascript
// BUG: Uses current viewer's name, not the actual answerer from server
function hydrateFieldState(drafts) {
  drafts.forEach(d => {
    FIELD_RESPONSES[d.field_id] = {
      value: d.value,
      answeredBy: getUserName(),  // WRONG — always shows current user
      answeredAt: d.answered_at
    };
  });
}

// FIX: Use server-provided name, fall back to current user only for local actions
function hydrateFieldState(drafts) {
  drafts.forEach(d => {
    FIELD_RESPONSES[d.field_id] = {
      value: d.value,
      answeredBy: d.answered_by_name || getUserName(),  // Server truth first
      answeredAt: d.answered_at
    };
  });
}
```

**Prevention:**
- Every optimistic update should follow this pattern:
  1. Set optimistic value → render immediately
  2. Send API request
  3. On response, overwrite optimistic value with server data
  4. Re-render if server data differs
- Never use `getCurrentUser()` or local state for data that originated from another user
- Backend should return display-ready values (e.g., `answered_by_name`) not just IDs

---

## Pattern 4: HTML Attribute String Concatenation

**Frequency:** 2 occurrences

**Symptom:** Buttons appear in the UI but don't respond to clicks. No console errors.

**Root cause:** When building HTML via string concatenation, missing spaces between attributes cause the browser to merge them into one invalid attribute name.

**Example:**
```javascript
// BUG: 'disabled' and 'data-action' merge into 'disableddata-action'
html += '<button ' + (canConfirm ? '' : 'disabled') + 'data-action="confirm">OK</button>';
// Renders: <button disableddata-action="confirm">OK</button>
// The event delegation (closest('[data-action]')) never finds this button

// FIX: Put data-action first, append optional attrs with leading space
html += '<button data-action="confirm"' + (canConfirm ? '' : ' disabled') + '>OK</button>';
// Renders: <button data-action="confirm" disabled>OK</button>
```

**Also watch for:** Using display text as data values (e.g., `data-rating="Ready"` when the API expects `"ready"`). Always use the API's expected value in data attributes, not the human-readable label.

**Prevention:**
- Always put `data-action` as the FIRST attribute (it's the delegation target — if it's mangled, the button silently dies)
- Append optional boolean attributes (`disabled`, `checked`) with a leading space
- Use API-format values in `data-*` attributes, use separate text content for display
- Consider a template literal helper that handles attribute concatenation safely

---

## Pattern 5: Stale Client State Across User Sessions

**Frequency:** 2 occurrences

**Symptom:** After logging out and logging in as a different user, the new user sees the previous user's data or UI state.

**Root cause:** Client-side state (DOM nodes, localStorage, in-memory objects) persists across login/logout cycles. `element.remove()` permanently destroys DOM nodes that the next user might need.

**Example:**
```javascript
// BUG: Tiles removed from DOM — can't be restored for next user
function filterTiles(allowedApps) {
  document.querySelectorAll('.tile').forEach(tile => {
    if (!allowedApps.has(tile.slug)) tile.remove();  // Gone forever
  });
}

// FIX: Hide with CSS — DOM stays intact for re-evaluation
function filterTiles(allowedApps) {
  document.querySelectorAll('.tile').forEach(tile => {
    tile.style.display = allowedApps.has(tile.slug) ? '' : 'none';
  });
}
```

**Prevention:**
- Use `display: none` instead of `remove()` for permission-based filtering
- Clear user-specific localStorage keys on logout
- On login, always re-fetch and re-apply permissions from server (don't trust cached state)
- Test the multi-user flow: login as admin → logout → login as restricted user → logout → login as admin → verify admin sees everything

---

## Pattern 6: Client-Side State Machine Missing Transitions

**Frequency:** 2 occurrences

**Symptom:** UI shows correct state on server reload but fails to transition locally. E.g., completing a section doesn't unlock the next one until page refresh.

**Root cause:** State machine logic is split between server (authoritative, handles all cases) and client (optimistic, handles only the common cases). New entity types (e.g., FAQ sections) get added to the server state machine but not the client's.

**Example:**
```javascript
// BUG: FAQ sections excluded — never transition to "complete" on client
function recomputeSectionState(sec) {
  if (sec.is_faq) return;  // Skipped entirely!
  // ... only handles checkbox/video types
}

// FIX: Handle all section types, mirror server transitions
function recomputeSectionState(sec) {
  if (sec.is_faq) {
    var allViewed = sec.items.every(f => f.viewed);
    if (allViewed && sec.state === 'active') {
      sec.state = 'complete';
      unlockNextSection(sec);  // Must also trigger downstream effects
    }
    return;
  }
  // ... checkbox/video handling with unlock + collapse
}
```

**Prevention:**
- When adding a new entity type, grep for every state transition function and update it
- Client state machine should have the same branches as the server's `isSectionComplete()` / `canActivateSection()`
- After any state transition, check if downstream sections need updating (unlock next, collapse current)
- Write a test that covers the full lifecycle: create → complete each section → verify next unlocks → verify final state

---

## Testing Rules (From Experience)

1. **Test the read path, not just the write path.** Save data via API, then fetch it back and verify. This catches JOIN bugs, field name mismatches, and serialization issues.

2. **Test across user sessions.** Login as User A, make changes, logout, login as User B, verify. Then login as User A again and verify their data is intact.

3. **Write the regression test BEFORE the fix.** The test must fail first (proving the bug exists), then pass after the fix. This ensures the test actually guards against the regression.

4. **Test with all entity types.** If your system has checkboxes, video series, and FAQ items, test completion/progress with each type — not just the most common one.

5. **Test state transitions end-to-end.** Don't just test that a section can be marked complete — test that completing section N unlocks section N+1 and that the UI reflects this without a page reload.

---

*Derived from ~30 bug fixes during Yumyums HQ v2.0 backend development (Go + Postgres + vanilla JS PWA).*
