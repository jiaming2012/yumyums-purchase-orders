---
phase: 11-onboarding-users-admin
plan: "04"
subsystem: backend/onboarding
tags: [onboarding, go, api, postgres, seed]
dependency_graph:
  requires:
    - ob_templates, ob_sections, ob_items, ob_video_parts tables (Plan 01)
    - ob_progress, ob_signoffs, ob_template_assignments tables (Plan 01)
    - auth.UserFromContext / auth.User (Phase 09)
    - chi router, pgxpool (existing)
  provides:
    - backend/internal/onboarding package (db.go + handler.go + seed.go)
    - GET /api/v1/onboarding/templates
    - GET /api/v1/onboarding/templates/{id}
    - GET /api/v1/onboarding/myTrainings
    - GET /api/v1/onboarding/hireTraining/{hireId}
    - GET /api/v1/onboarding/managerHires
    - POST /api/v1/onboarding/saveProgress
    - POST /api/v1/onboarding/signOff
    - POST /api/v1/onboarding/createTemplate
    - PUT /api/v1/onboarding/updateTemplate/{id}
    - POST /api/v1/onboarding/assignTemplate
    - POST /api/v1/onboarding/unassignTemplate
    - "Kitchen Basics Training" onboarding seed template
  affects:
    - backend/cmd/server/main.go (routes + seed call added)
    - Plan 05 frontend (will call these endpoints)
tech_stack:
  added: []
  patterns:
    - RPC-style HTTP handlers (same pattern as workflow/handler.go)
    - Full replace pattern for UpdateTemplate (delete sections + reinsert)
    - Idempotent seeding via existence check before INSERT
    - Server-side section state computation (locked/active/complete/signed_off)
key_files:
  created:
    - backend/internal/onboarding/db.go
    - backend/internal/onboarding/handler.go
    - backend/internal/onboarding/seed.go
  modified:
    - backend/cmd/server/main.go
decisions:
  - "isManagerOrAdmin helper used for sign-off and management endpoints (manager role can sign off per D-05)"
  - "insertSectionsTx helper accepts pgx.Tx to share between CreateTemplate and UpdateTemplate"
  - "SeedOnboardingTemplates uses name-based existence check (same pattern as workflow SeedTemplates)"
  - "isSectionComplete returns true for is_faq sections so they never block progression"
  - "GetHireTraining fetches progress/signoffs in separate queries then computes section states in Go"
metrics:
  duration_seconds: 420
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_modified: 4
---

# Phase 11 Plan 04: Onboarding Backend API Summary

Onboarding backend with template CRUD, progress auto-save, manager sign-offs, and seed data — all routes registered and server compiling.

**One-liner:** Go onboarding package with 11 DB functions, 11 HTTP handlers, server-side section state computation (locked/active/complete/signed_off), and "Kitchen Basics Training" seed template wired into main.go.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Create onboarding package with db.go, handler.go, and seed.go | 2f01c0f | 3 new files |
| 2 | Wire onboarding routes and seed call into main.go | d923229 | backend/cmd/server/main.go |

## What Was Built

### db.go — SQL Query Functions

All 11 functions against the ob_* schema from Plan 01 migrations:

- **GetTemplates** / **GetTemplate** — Nested SELECT via getSections → getItems → getVideoParts helpers, returning full Template tree
- **GetHireTraining** — Fetches template + progress rows + signoffs, then computes section states in Go: signed_off (signoff exists) → complete (all non-faq items checked) → active (first section or all prior complete/signed_off) → locked (otherwise). Mirrors `isSectionComplete()` + `canActivateSection()` logic from onboarding.html
- **GetMyTrainings** — Hire's assigned templates with progress percentage; subquery counts checked items and total items (video parts counted individually)
- **GetManagerHires** — All hires with assignments; uses COALESCE display_name expression consistent with auth package
- **SaveProgress** — INSERT ON CONFLICT DO NOTHING (checked=true) or DELETE (checked=false) from ob_progress
- **SignOff** — INSERT ON CONFLICT (section_id, hire_id) DO NOTHING — idempotent
- **CreateTemplate** / **UpdateTemplate** — Transaction-wrapped with insertSectionsTx helper; UpdateTemplate deletes sections first (FK cascades to items/video_parts) then reinserts
- **AssignTemplate** / **UnassignTemplate** — Simple INSERT ON CONFLICT DO NOTHING / DELETE on ob_template_assignments

### handler.go — HTTP Handlers

11 handlers following the same pattern as workflow/handler.go:

- `writeJSON`, `writeError`, `isAdmin`, `isManagerOrAdmin` helpers
- **SaveProgressHandler** — Reads `{item_id, progress_type, checked}`, validates non-empty item_id and progress_type, calls SaveProgress with user.ID as hire_id
- **SignOffHandler** — Admin/manager only; validates notes non-empty and rating ∈ {ready, needs_practice, struggling} before calling SignOff
- **HireTrainingHandler** — Requires `templateId` query param; allows both hire and manager to view training detail
- All management endpoints (create/update/assign/unassign templates, sign-off, list hires) restricted to manager/admin

### seed.go — Seed Data

`SeedOnboardingTemplates` seeds "Kitchen Basics Training" template on startup:

- Section 1: "Safety & Hygiene" (requires_sign_off: true) — 4 checkboxes
- Section 2: "Equipment Training" (requires_sign_off: true) — video_series "Grill Operation" (3 parts) + 2 checkboxes
- Section 3: "Menu Knowledge" (requires_sign_off: false) — 3 checkboxes
- Section 4: "FAQ" (is_faq: true) — 3 FAQ items with answers

Idempotent via name existence check before INSERT. `strPtr` helper for nullable string fields.

### main.go Updates

- `onboarding` package import added
- `r.Route("/onboarding", ...)` registered after workflow routes with all 11 endpoints
- `onboarding.SeedOnboardingTemplates` call added after workflow seed, before server start

## Deviations from Plan

None - plan executed exactly as written.

The plan specified seeding from MOCK_OB_TEMPLATES data in onboarding.html, but also explicitly listed the "Kitchen Basics Training" template structure in the seed.go action section. The plan action section was used as the authoritative seed spec.

## Known Stubs

None — this plan creates backend Go code only; no UI or mock data introduced.

## Self-Check: PASSED
