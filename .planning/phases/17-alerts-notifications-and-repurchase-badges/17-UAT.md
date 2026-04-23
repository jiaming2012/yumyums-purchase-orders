---
status: complete
phase: 17-alerts-notifications-and-repurchase-badges
source: [17-01-SUMMARY.md, 17-02-SUMMARY.md, 17-03-SUMMARY.md]
started: 2026-04-23T00:15:00.000Z
updated: 2026-04-23T04:20:00.000Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Kill the running backend. Restart with `cd backend && task run`. Server boots without errors, migrations apply (0041-0044), alert queue starts ("Alert queue started" in logs). Hit http://localhost:8080/api/v1/me — returns user data.
result: pass

### 2. User Notification Preference in Edit Form
expected: Open Users tab. Tap a user to edit. An "Alert Channel" dropdown appears with options "Zoho Cliq" and "Email" (Zoho Cliq selected by default). Change to Email, save. Reopen the user — Email is still selected.
result: issue
reported: "Alert channel does show up; however, the user should be able to select more than one (both) alert channels"
severity: major

### 3. Repurchase Badge on Stock Tab
expected: Open Inventory → Stock tab. If a shopping list was recently completed, items that were on it show a blue "Repurchased +N" badge next to their stock level badge. Items not on any shopping list show no repurchase badge.
result: issue
reported: "Works however there's a large space between stock level badge and repurchase badge that should be removed"
severity: cosmetic

### 4. Badge Reset Config in Setup Tab (Admin Only)
expected: Open Inventory → Setup tab (as admin). A "Badge Reset Schedule" card appears at the bottom. It shows day-of-week chips and a time input. Select a day, set a time, tap Save. Reload — config persists. Non-admin users should NOT see this card.
result: issue
reported: "Cannot change timezone from America/Chicago. Timezone should be a per-user setting defaulting to America/New_York"
severity: major

### 5. Shopping Completion Alert
expected: Complete a shopping list vendor section. Server logs show the alert being dispatched. The alert message lists any unchecked/missing items with vendor name and quantities.
result: pass

### 6. Low Stock Alert in Scheduler Logs
expected: With the server running for 15+ minutes, check logs. If any items are at "low" stock level, you should see a log entry about low-stock alert dispatch. No duplicate alerts for the same item in the same week.
result: pass

## Summary

total: 6
passed: 3
issues: 3
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Users can select multiple alert channels (both Zoho Cliq and Email)"
  status: failed
  reason: "User reported: Alert channel dropdown is single-select, should allow multi-select (both channels)"
  severity: major
  test: 2
  artifacts: [users.html, backend/internal/users/db.go]
  missing: [multi-select notification preference]

- truth: "No extra spacing between stock level badge and repurchase badge"
  status: failed
  reason: "User reported: large gap between Low badge and Repurchased +1 badge"
  severity: cosmetic
  test: 3
  artifacts: [inventory.html]
  missing: [CSS spacing fix]

- truth: "Timezone is configurable per user, not hardcoded to America/Chicago"
  status: failed
  reason: "User reported: cannot change timezone, should be per-user setting defaulting to America/New_York"
  severity: major
  test: 4
  artifacts: [backend/internal/purchasing/scheduler.go, users.html, backend/internal/users/db.go]
  missing: [user timezone column, timezone selector in Users tab, scheduler uses per-user timezone]
