---
status: resolved
trigger: "Fix timezone/time handling and JS crash in loadMyChecklists"
created: 2026-04-15T00:00:00Z
updated: 2026-04-15T00:00:00Z
---

## Current Focus

hypothesis: Two JS crashes confirmed. Line 1620: operator precedence bug — `submission && submission.status === 'pending_approval' || submission.status === 'pending'` evaluates `submission.status === 'pending'` when submission is null, throwing TypeError. Same pattern on line 1644 in renderRunner. These crash renderChecklistList and cause the "Couldn't load checklists" error.
test: Fix operator precedence on both lines, run test suite
expecting: All fixme tests pass after fix
next_action: Apply fixes to workflows.html, remove .fixme from persistence tests, run full suite

## Symptoms

expected: My Checklists renders without error
actual: "Couldn't load checklists" error shown; JS crash in loadMyChecklists
errors: TypeError: Cannot read property 'status' of null (uncaught, swallowed by try/catch)
reproduction: Load My Checklists tab when at least one template has no submission for today
started: After renderChecklistList was updated to check submission status

## Eliminated

- hypothesis: hydrateFieldState crashes
  evidence: hydrateFieldState code is correct — null guards present throughout
  timestamp: 2026-04-15

## Evidence

- timestamp: 2026-04-15
  checked: workflows.html line 1620
  found: `var pendingBadge = submission && submission.status === 'pending_approval' || submission.status === 'pending'` — JS operator precedence means `submission.status === 'pending'` is evaluated independently even when submission is null
  implication: TypeError thrown for every template that has no submission, crashes renderChecklistList, caught by try/catch as "Couldn't load checklists"

- timestamp: 2026-04-15
  checked: workflows.html line 1644
  found: Same operator precedence bug in renderRunner: `var isPending = submission && submission.status === 'pending_approval' || submission.status === 'pending'`
  implication: Same crash would occur when opening a checklist runner with no submission

- timestamp: 2026-04-15
  checked: repository.go cleanupOldDrafts
  found: Uses `current_date` (UTC midnight) which is correct — drafts from previous UTC days are cleaned up. Client DOW is used for schedule matching.
  implication: cleanupOldDrafts is fine as-is; timezone handling is already correct in backend

- timestamp: 2026-04-15
  checked: persistence.spec.js fixme tests
  found: Four .fixme tests — all UI integration tests checking My Checklists renders correctly after reload. They will pass once renderChecklistList no longer crashes.
  implication: Remove .fixme after fixing the crash

## Resolution

root_cause: Three bugs found:
  1. workflows.html line 1620: operator precedence — `submission && A || B` evaluates B even when submission is null, crashing renderChecklistList for any template without a submission.
  2. workflows.html line 1644: same operator precedence bug in renderRunner.
  3. repository.go myChecklists: did not call hydrateSubmission for today's submissions, so submission.responses was empty — hydrateFieldState couldn't restore FIELD_RESPONSES, causing check-btns to render unchecked on reload.
  4. tests/workflows.spec.js createTestTemplate: called with requires_approval:true but no approver assignment when todayDOW is undefined, causing 400 errors that silently returned {error:...} instead of {id:...}, so templates were never created.
fix:
  1+2: Wrapped pendingBadge and isPending conditions with proper parentheses
  3: Added hydrateSubmission call for each submission in myChecklists
  4: Made requires_approval conditional on todayDOW presence in createTestTemplate
  5: Removed .fixme from 4 persistence tests
  6: Bumped sw.js cache to v61
verification: All 104 tests pass (npx playwright test)
files_changed: [workflows.html, backend/internal/workflow/repository.go, tests/persistence.spec.js, tests/workflows.spec.js, sw.js]
