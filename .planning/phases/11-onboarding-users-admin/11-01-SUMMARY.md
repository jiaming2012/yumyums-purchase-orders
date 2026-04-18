---
phase: 11-onboarding-users-admin
plan: "01"
subsystem: backend/database
tags: [migrations, auth, postgres, onboarding]
dependency_graph:
  requires: []
  provides:
    - users table with first_name/last_name/nickname columns
    - ob_templates, ob_sections tables
    - ob_items, ob_video_parts tables
    - ob_progress table
    - ob_signoffs table
    - ob_template_assignments table
    - auth.User.DisplayName derived from SQL COALESCE expression
    - DeleteAllSessionsByUserID function
  affects:
    - backend/internal/auth/service.go (all user queries)
    - all downstream plans 02-06 that need onboarding tables
tech_stack:
  added: []
  patterns:
    - goose migrations with BEGIN/COMMIT blocks
    - COALESCE(NULLIF(nickname,''), first_name || ' ' || LEFT(last_name,1) || '.') for display_name derivation
key_files:
  created:
    - backend/internal/db/migrations/0017_users_naming.sql
    - backend/internal/db/migrations/0018_ob_templates.sql
    - backend/internal/db/migrations/0019_ob_items.sql
    - backend/internal/db/migrations/0020_ob_progress.sql
    - backend/internal/db/migrations/0021_ob_signoffs.sql
    - backend/internal/db/migrations/0022_ob_template_assignments.sql
  modified:
    - backend/internal/auth/service.go
decisions:
  - "displayNameExpr constant used in all SELECT queries — single source of truth for derived display_name"
  - "splitName helper added to auth package to parse config.SuperadminEntry.DisplayName into first/last components"
  - "ob_progress uses discriminator column (progress_type) rather than FK constraint to support both item and video_part progress in one table"
  - "0017 Down migration wraps in BEGIN/COMMIT for consistency (plan spec omitted it — added for correctness)"
metrics:
  duration_seconds: 126
  completed_date: "2026-04-17"
  tasks_completed: 2
  files_modified: 7
---

# Phase 11 Plan 01: DB Migrations and Auth Updates Summary

Foundation migrations and auth package updates for the onboarding/users/admin phase.

**One-liner:** 6 goose migrations (users naming refactor + 5 onboarding tables) with auth queries updated to derive display_name from first_name/last_name/nickname via COALESCE.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Create all 6 database migrations (0017-0022) | fadc7e1 | 6 new SQL files |
| 2 | Update auth/service.go queries for derived display_name | 2d88708 | backend/internal/auth/service.go |

## What Was Built

### Migrations (0017-0022)

**0017_users_naming.sql** — Users table schema migration: adds `first_name`, `last_name`, `nickname` columns; migrates existing `display_name` data by splitting at first space; drops `display_name`. Down migration reconstructs `display_name` from the COALESCE expression.

**0018_ob_templates.sql** — `ob_templates` (id, name, role, created_by, created_at, updated_at) and `ob_sections` (id, template_id FK, title, sort_order, requires_sign_off, is_faq) with index on template_id.

**0019_ob_items.sql** — `ob_items` (checkbox/video_series/faq type enum, label, answer, sort_order) and `ob_video_parts` (item_id FK, title, description, url, sort_order) with indexes.

**0020_ob_progress.sql** — `ob_progress` with discriminator `progress_type` CHECK ('item', 'video_part') and UNIQUE(hire_id, item_id, progress_type) to prevent duplicate progress records across both item types.

**0021_ob_signoffs.sql** — `ob_signoffs` with `rating` CHECK ('ready', 'needs_practice', 'struggling'), `notes` TEXT, UNIQUE(section_id, hire_id) to enforce one sign-off per section per hire.

**0022_ob_template_assignments.sql** — `ob_template_assignments` linking hires to templates with assigned_by and assigned_at, UNIQUE(hire_id, template_id).

### auth/service.go Updates

- `displayNameExpr` constant replaces raw `u.display_name` in all SELECT queries
- `LookupSession` and `AuthenticateUser` both use `fmt.Sprintf` with the constant
- `UpsertSuperadmins` now INSERTs `first_name`/`last_name` (split from config Name field) instead of `display_name`
- `splitName` helper splits at first space; handles single-word names gracefully
- `DeleteAllSessionsByUserID` added for session revocation (D-20)
- `strings` import added; `fmt` import already present

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing correctness] 0017 Down migration missing BEGIN/COMMIT**
- **Found during:** Task 1 implementation review
- **Issue:** Plan spec showed Down migration without transaction wrapping; all other Down migrations should be consistent
- **Fix:** Wrapped Down migration in BEGIN/COMMIT block for consistency with Up migration and other migration files
- **Files modified:** backend/internal/db/migrations/0017_users_naming.sql
- **Commit:** fadc7e1

## Known Stubs

None — this plan creates DB migrations and updates Go auth code only; no UI stubs introduced.

## Self-Check: PASSED

Files verified:
- FOUND: backend/internal/db/migrations/0017_users_naming.sql
- FOUND: backend/internal/db/migrations/0018_ob_templates.sql
- FOUND: backend/internal/db/migrations/0019_ob_items.sql
- FOUND: backend/internal/db/migrations/0020_ob_progress.sql
- FOUND: backend/internal/db/migrations/0021_ob_signoffs.sql
- FOUND: backend/internal/db/migrations/0022_ob_template_assignments.sql
- FOUND: backend/internal/auth/service.go (updated)

Commits verified:
- FOUND: fadc7e1 (Task 1 migrations)
- FOUND: 2d88708 (Task 2 auth update)
