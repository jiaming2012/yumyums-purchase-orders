---
phase: 18-add-photos-to-onboarding-checklists
plan: 01
subsystem: onboarding-backend
tags: [photo, migration, progress, backend]
dependency_graph:
  requires: []
  provides: [photo-progress-type, value-column, photo-url-in-api]
  affects: [ob_progress-table, onboarding-api]
tech_stack:
  added: []
  patterns: [sql-null-string-scan, coalesce-upsert]
key_files:
  created:
    - backend/internal/db/migrations/0056_ob_progress_photo.sql
  modified:
    - backend/internal/onboarding/db.go
    - backend/internal/onboarding/handler.go
decisions:
  - "COALESCE(EXCLUDED.value, ob_progress.value) in upsert preserves existing value if new is null"
  - "database/sql.NullString used for scanning nullable value column from progress query"
  - "Photo items use same incomplete-item detection as checkboxes (absence of progress row)"
metrics:
  duration_seconds: 159
  completed: "2026-05-21T02:51:00Z"
---

# Phase 18 Plan 01: Onboarding Photo Backend Support Summary

**One-liner:** DB migration adding value column + photo progress_type, Go backend SaveProgress/GetHireTraining/isSectionComplete with full photo type handling across all 7 SQL filters.

## What Was Done

### Task 1: DB Migration (0056_ob_progress_photo.sql)
- Added nullable `value TEXT` column to `ob_progress` table
- Updated CHECK constraint to include `'photo'` in allowed progress_type values
- Down migration reverts both changes
- **Commit:** 0227581

### Task 2: Go Backend Updates (db.go + handler.go)
- **Item struct:** Added `PhotoURL string` field with `json:"photo_url,omitempty"` tag
- **SaveProgress:** Extended signature with `value *string` parameter; INSERT now includes `value` column with COALESCE upsert to preserve existing value
- **GetHireTraining:** Progress query SELECTs `op.value`; scan loop builds `valueMap` using `sql.NullString`; new `photo` branch in item type switch sets `Checked` and `PhotoURL`
- **isSectionComplete:** New `photo` branch increments `totalCheckable` and checks progress
- **All 7 SQL type filters:** Updated from `IN ('checkbox', 'video_series', 'faq')` to `IN ('checkbox', 'video_series', 'faq', 'photo')` across sectionIncompleteItem, GetMyTrainings, GetManagerHires (4 locations), and IsSectionLockedForEdits
- **handler.go:** SaveProgressHandler body struct accepts `Value *string`; passes `body.Value` to SaveProgress call
- **Commit:** 1c3ed41

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `go build ./...` passes with zero errors
- Zero occurrences of `IN ('checkbox', 'video_series', 'faq')` without 'photo' remain in db.go
- Migration file contains correct Up/Down sections with BEGIN/COMMIT wrapping

## Self-Check: PASSED
