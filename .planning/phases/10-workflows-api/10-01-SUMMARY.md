---
phase: 10-workflows-api
plan: 01
subsystem: database
tags: [postgres, goose, migrations, go-structs, yaml, seed]

# Dependency graph
requires:
  - phase: 09-foundation-auth
    provides: users table, pgxpool.Pool pattern, goose migration runner

provides:
  - 9 goose migration files (0006–0014) for full workflow domain schema
  - Go model structs for Template, Section, Field, Submission, FieldResponse, FailNote, Rejection
  - TemplateInput and other input types for API requests
  - LoadTemplateConfig + SeedTemplates functions following superadmins/SeedHQApps pattern
  - backend/config/templates.yaml with Setup Checklist sample template

affects: [10-02, 10-03, 10-04, 10-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "goose migration: one table per numbered file, wrapped in BEGIN/COMMIT"
    - "JSONB for conditions/config/fail_trigger columns (D-01)"
    - "soft delete via archived_at TIMESTAMPTZ nullable column (D-07)"
    - "seed pattern: LoadXConfig(path) + SeedX(ctx, pool, items, createdBy)"

key-files:
  created:
    - backend/internal/db/migrations/0006_checklist_templates.sql
    - backend/internal/db/migrations/0007_checklist_schedules.sql
    - backend/internal/db/migrations/0008_template_assignments.sql
    - backend/internal/db/migrations/0009_checklist_sections.sql
    - backend/internal/db/migrations/0010_checklist_fields.sql
    - backend/internal/db/migrations/0011_checklist_submissions.sql
    - backend/internal/db/migrations/0012_submission_responses.sql
    - backend/internal/db/migrations/0013_submission_fail_notes.sql
    - backend/internal/db/migrations/0014_submission_rejections.sql
    - backend/internal/workflow/model.go
    - backend/internal/workflow/seed.go
    - backend/config/templates.yaml
  modified: []

key-decisions:
  - "JSONB for conditions, config, fail_trigger, and template_snapshot — flexible schema without migrations per new field type"
  - "parent_field_id self-reference in checklist_fields for sub-steps (D-03)"
  - "submission_responses_draft_idx partial unique index WHERE submission_id IS NULL for draft support (D-21)"
  - "SeedTemplates is idempotent on template name — safe to run on every startup"
  - "insertField is recursive — handles arbitrary sub-step depth"

patterns-established:
  - "Workflow seed pattern: LoadTemplateConfig returns []TemplateInput; SeedTemplates iterates and calls seedTemplate per item"
  - "seedTemplate uses pool.Begin transaction for atomicity; Rollback deferred; Commit at end"
  - "marshalNullableJSON helper: nil if empty/null RawMessage, []byte otherwise for pgx JSONB params"

requirements-completed: [WKFL-01]

# Metrics
duration: 3min
completed: 2026-04-15
---

# Phase 10 Plan 01: Workflow Schema and Models Summary

**Postgres schema for the full workflows domain — 9 goose migrations (templates through rejections), Go model structs, and YAML-driven template seed infrastructure**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-04-15T19:40:33Z
- **Completed:** 2026-04-15T19:43:51Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- 9 migration files (0006–0014) covering all workflow tables with correct JSONB, foreign keys, indexes, and goose Up/Down markers
- Go model structs in `backend/internal/workflow/model.go` mapping 1:1 to schema columns with correct nullable types and JSON tags
- `LoadTemplateConfig` + `SeedTemplates` functions following the established LoadSuperadmins/SeedHQApps pattern, with a sample Setup Checklist in `templates.yaml`

## Task Commits

1. **Task 1: Create 9 goose migration files** - `25b5cb9` (feat)
2. **Task 2: Create Go model structs in workflow/model.go** - `f52a43d` (feat)
3. **Task 3: Create YAML template seed infrastructure** - `b830563` (feat)

## Files Created/Modified

- `backend/internal/db/migrations/0006_checklist_templates.sql` - Templates table with soft delete (archived_at)
- `backend/internal/db/migrations/0007_checklist_schedules.sql` - Schedules table with active_days INTEGER[]
- `backend/internal/db/migrations/0008_template_assignments.sql` - Assignment junction table with role/user type check
- `backend/internal/db/migrations/0009_checklist_sections.sql` - Sections with JSONB condition
- `backend/internal/db/migrations/0010_checklist_fields.sql` - Fields with self-ref parent_field_id and JSONB config/fail_trigger/condition
- `backend/internal/db/migrations/0011_checklist_submissions.sql` - Submissions with template_snapshot JSONB and idempotency_key UNIQUE
- `backend/internal/db/migrations/0012_submission_responses.sql` - Responses with draft partial index
- `backend/internal/db/migrations/0013_submission_fail_notes.sql` - Fail notes with severity enum
- `backend/internal/db/migrations/0014_submission_rejections.sql` - Per-field rejections
- `backend/internal/workflow/model.go` - All domain structs + input types, compiles cleanly
- `backend/internal/workflow/seed.go` - LoadTemplateConfig + SeedTemplates with pgx.Tx transactions
- `backend/config/templates.yaml` - Setup Checklist sample with 2 sections and sub-steps

## Decisions Made

- `marshalNullableJSON` helper function introduced to convert `json.RawMessage` to `[]byte` or `nil` for pgx JSONB parameters — avoids passing empty strings to nullable JSONB columns
- Used `pgx.Tx` directly in `insertField` parameter type rather than a custom interface, avoiding type mismatch with `pgconn.CommandTag` return from `Exec`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pgx.Tx interface mismatch in insertField**
- **Found during:** Task 3 (YAML template seed infrastructure)
- **Issue:** Custom interface with `Exec(...) (pgxResult, error)` did not match `pgx.Tx.Exec(...)` which returns `(pgconn.CommandTag, error)` — build failed
- **Fix:** Changed `insertField` parameter type from anonymous interface to `pgx.Tx` directly
- **Files modified:** backend/internal/workflow/seed.go
- **Verification:** `go build ./internal/workflow/` exits 0
- **Committed in:** b830563 (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Required type correction for compilation. No scope creep.

## Issues Encountered

None beyond the interface type mismatch documented above.

## Next Phase Readiness

- Schema foundation complete — all 9 tables ready for goose to apply
- Go models compile and are importable by repository/handler packages
- SeedTemplates not yet wired into main.go (per plan — happens in Plan 03)
- Plan 02 can now build repository queries against the schema

---
*Phase: 10-workflows-api*
*Completed: 2026-04-15*
