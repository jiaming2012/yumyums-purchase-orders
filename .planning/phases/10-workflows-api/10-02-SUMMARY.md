---
phase: 10-workflows-api
plan: "02"
subsystem: backend/workflow
tags: [go, api, repository, handlers, chi, postgres]
dependency_graph:
  requires: [10-01]
  provides: [workflow-api-endpoints]
  affects: [backend/cmd/server/main.go]
tech_stack:
  added: []
  patterns:
    - Pool-closure handler pattern (same as me/handler.go)
    - Transaction-based CRUD with defer rollback
    - pgx v5 ON CONFLICT for idempotency
    - Soft-delete via archived_at
key_files:
  created:
    - backend/internal/workflow/repository.go
    - backend/internal/workflow/handler.go
  modified:
    - backend/cmd/server/main.go
decisions:
  - errors.Is(err, os.ErrNotExist) used instead of os.IsNotExist for wrapped error from LoadTemplateConfig
  - cleanupOldDrafts fired as goroutine in MyChecklistsHandler (fire-and-forget, non-blocking)
  - hydrateTemplate and hydrateSubmission extracted as shared helpers for DRY loading
metrics:
  duration: "~5 min"
  completed: "2026-04-15"
  tasks: 3
  files_modified: 3
---

# Phase 10 Plan 02: Workflow API Layer Summary

Complete Go API layer for workflow CRUD — repository queries, HTTP handlers, and route wiring providing all 11 endpoints under `/api/v1/workflow/` with idempotency, auto-save drafts, and admin-gated builder endpoints.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create repository.go with all DB query functions | 3c9dfc9 | backend/internal/workflow/repository.go |
| 2 | Create handler.go with HTTP handlers for all endpoints | ea2d8e2 | backend/internal/workflow/handler.go |
| 3 | Wire workflow routes and seed into main.go | d81dc6d | backend/cmd/server/main.go |

## What Was Built

### repository.go (802 lines)

13 functions implementing the full data access layer:

- `insertTemplate` — transactional insert with sections, fields (recursive sub-steps), schedules, assignments
- `replaceTemplate` — D-09 full replace: delete children, update header, re-insert all children
- `archiveTemplate` — D-07 soft-delete via `archived_at = now()`
- `listTemplates` / `getTemplateByID` — fully hydrated via `hydrateTemplate` helper
- `submitChecklist` — D-15 `ON CONFLICT (idempotency_key)`, moves drafts, inserts fail notes
- `saveResponse` — D-21 draft upsert `ON CONFLICT ON CONSTRAINT submission_responses_draft_idx`
- `myChecklists` — D-22 filters by `template_assignments` + `checklist_schedules.active_days` for today's DOW
- `myHistory` — last 50 submissions with template name JOIN
- `pendingApprovals` — D-23 filters by `assignment_role = 'approver'` matching user or role
- `approveSubmission` — UPDATE status to 'approved' where status = 'pending'
- `rejectItem` — D-06 inserts `submission_rejections`, sets status = 'rejected'
- `cleanupOldDrafts` — deletes `submission_responses` where `submission_id IS NULL AND answered_at < current_date`
- `ErrTemplateArchived` — sentinel error for archived template conflict (D-14)

### handler.go (340 lines)

11 HTTP handlers following the `pool *pgxpool.Pool` closure pattern from `me/handler.go`:

- `isAdmin` helper — `Role == "admin" || IsSuperadmin` (D-11)
- `ListTemplatesHandler` — GET, returns `[]Template` (empty array if none)
- `CreateTemplateHandler` — POST, admin-only 403, returns `{"id":"..."}`
- `UpdateTemplateHandler` — PUT `/{id}`, admin-only 403, full replace
- `ArchiveTemplateHandler` — DELETE `/{id}`, admin-only 403
- `MyChecklistsHandler` — GET, fires `cleanupOldDrafts` goroutine, returns `{templates, submissions}`
- `MyHistoryHandler` — GET, returns `[]Submission`
- `SaveResponseHandler` — POST, returns 204 No Content
- `SubmitChecklistHandler` — POST, returns 409 on `ErrTemplateArchived`
- `PendingApprovalsHandler` — GET, returns hydrated `[]Submission`
- `ApproveSubmissionHandler` — POST `{"submission_id":"..."}`
- `RejectItemHandler` — POST `RejectItemInput`

### main.go changes

- Added `workflow` import
- Template config loading from `config/templates.yaml` (optional, `errors.Is(err, os.ErrNotExist)`)
- Template seeding using first superadmin as creator after `SeedHQApps`
- `r.Route("/workflow", ...)` with all 11 endpoints inside the protected `auth.Middleware` group

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Used errors.Is instead of os.IsNotExist for wrapped error**
- **Found during:** Task 3
- **Issue:** `LoadTemplateConfig` wraps the OS error with `fmt.Errorf("read templates config: %w", err)`. `os.IsNotExist` does not unwrap, so it would always return false for wrapped errors.
- **Fix:** Used `errors.Is(err, os.ErrNotExist)` which unwraps the error chain correctly. Added `"errors"` import.
- **Files modified:** backend/cmd/server/main.go
- **Commit:** d81dc6d

## Known Stubs

None — all repository functions are fully implemented against real schema. No mock data, no placeholder returns.

## Self-Check

### Files exist:
- backend/internal/workflow/repository.go: FOUND
- backend/internal/workflow/handler.go: FOUND
- backend/cmd/server/main.go: modified (FOUND)

### Commits exist:
- 3c9dfc9: feat(10-02): create workflow repository with all DB query functions
- ea2d8e2: feat(10-02): create workflow HTTP handlers for all 11 endpoints
- d81dc6d: feat(10-02): wire workflow routes and template seeding into main.go

### Build verification:
- `go build ./internal/workflow/`: PASSED
- `go build ./cmd/server/`: PASSED

## Self-Check: PASSED
