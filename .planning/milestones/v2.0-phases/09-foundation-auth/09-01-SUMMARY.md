---
phase: 09-foundation-auth
plan: 01
subsystem: backend-shell
tags: [go, chi, embed-fs, service-worker, pwa, infrastructure]
requirements: [INFRA-01, INFRA-04]

dependency_graph:
  requires: []
  provides: [go-server-shell, sw-fetch-partition]
  affects: [all-future-api-plans]

tech_stack:
  added:
    - go module: github.com/yumyums/hq
    - go-chi/chi/v5 v5.2.5 — HTTP router
    - embed.FS — prod static file serving
    - os.DirFS — dev static file serving
  patterns:
    - dual-mode static serving via STATIC_DIR env var
    - chi router with Logger + Recoverer middleware
    - embed.FS with all:public pattern (dir relative to main.go)
    - SW fetch handler partitioned: /api/* network-first, static cache-first

key_files:
  created:
    - backend/cmd/server/main.go — Go HTTP server with chi router, embed.FS + os.DirFS switching
    - backend/go.mod — Go module definition (github.com/yumyums/hq)
    - backend/go.sum — dependency checksums
    - backend/Makefile — dev and build targets
    - backend/.gitignore — excludes bin/
    - backend/cmd/server/public/.gitkeep — marks embed directory (build artifacts not committed)
  modified:
    - sw.js — partitioned fetch handler, bumped to yumyums-v42

decisions:
  - "public/ directory is cmd/server/public/ not backend/public/ — embed.FS requires path relative to source file"
  - "Go embed does not allow gitignoring the files it embeds — removed gitignore from public/ to allow embed to work"
  - "SW cache bumped from v41 to v42 (worktree was at v41, not v46 as plan expected)"

metrics:
  duration: 12 minutes
  completed_date: "2026-04-15"
  tasks_completed: 2
  tasks_total: 2
  files_created: 6
  files_modified: 1
---

# Phase 09 Plan 01: Go Server Shell + SW Fetch Partition Summary

Go HTTP server with chi router that serves the existing PWA from embed.FS (prod) or os.DirFS (dev), with a /api/v1/health endpoint, and service worker fetch handler partitioned to prevent API response caching.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Go server shell with chi router and dual-mode static file serving | 34c7a08 | backend/cmd/server/main.go, backend/go.mod, backend/go.sum, backend/Makefile, backend/.gitignore, backend/cmd/server/public/.gitkeep |
| 2 | Partition SW fetch handler for API vs static routes | 7e93772 | sw.js |

## What Was Built

**Task 1: Go Server Shell**

A Go binary at `backend/cmd/server/main.go` that:
- Uses chi router with Logger and Recoverer middleware
- Serves static PWA files in two modes:
  - Dev: `STATIC_DIR` env var → `os.DirFS(dir)` — live file serving, no rebuild for frontend changes
  - Prod: `STATIC_DIR` unset → `fs.Sub(embeddedFS, "public")` — embed.FS baked into binary
- Exposes `GET /api/v1/health` → `{"status":"ok"}` (outside any auth middleware)
- Reads PORT (default 8080) and DB_URL env vars
- Makefile `dev` target: `STATIC_DIR=.. go run ./cmd/server`
- Makefile `build` target: copies frontend files to `cmd/server/public/`, then `go build -o bin/server`

**Task 2: SW Fetch Partition**

Updated `sw.js` with partitioned fetch strategy:
- `/api/*` paths: network-first, offline returns `{"error":"offline"}` with 503 status
- All other paths: cache-first (unchanged behavior)
- Cache version bumped to v42 (forces reinstall on all PWA clients)

## Verification Results

- `go build ./cmd/server` — PASS (compiles without errors)
- Health endpoint: `{"status":"ok"}` 200 — PASS
- Static files served from disk (STATIC_DIR mode) — PASS
- `/api/*` network-first — PASS (pathname check in sw.js)
- Static cache-first — PASS (unchanged branch preserved)
- All 54 Playwright E2E tests — PASS

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] embed.FS path must be relative to source file, not module root**

- **Found during:** Task 1
- **Issue:** Plan specified `backend/public/` as the embed directory, but Go's `//go:embed` directive resolves paths relative to the source file containing the directive. Since `main.go` is at `backend/cmd/server/main.go`, the embed path `all:public` looks for `backend/cmd/server/public/` not `backend/public/`. Build failed with `pattern all:public: no matching files found`.
- **Fix:** Moved the `public/` directory to `backend/cmd/server/public/`. Updated Makefile build target to copy frontend files to `cmd/server/public/` instead of `public/`.
- **Files modified:** backend/Makefile (paths updated), backend/cmd/server/public/.gitkeep (new location)
- **Commits:** 34c7a08

**2. [Rule 1 - Bug] Go embed respects .gitignore — gitignore in public/ breaks embed**

- **Found during:** Task 1
- **Issue:** Plan specified adding `backend/public/.gitignore` with `*` and `!.gitignore` to prevent committing build artifacts. However, Go's embed tool respects `.gitignore` patterns. With `*` in the gitignore, ALL files in `public/` were excluded from embedding, causing `pattern all:public: no matching files found`.
- **Fix:** Removed gitignore from `public/`. Used `backend/cmd/server/public/.gitkeep` instead to mark the directory as intentional. Build artifacts (copied HTML files) are not committed because `make build` runs before deployment, not before `git commit`.
- **Files modified:** Removed gitignore from public/; added .gitkeep
- **Commits:** 34c7a08

**3. [Rule 1 - Bug] SW cache version adapted to worktree state**

- **Found during:** Task 2
- **Issue:** Plan says to bump from `yumyums-v46` to `yumyums-v47`. The worktree's sw.js was at `yumyums-v41` (different from main repo state). Applying `v47` would skip 5 versions.
- **Fix:** Bumped from `v41` to `v42` — correct increment for this worktree's state.
- **Files modified:** sw.js
- **Commits:** 7e93772

## Self-Check: PASSED

Files exist:
- backend/cmd/server/main.go — FOUND
- backend/go.mod — FOUND
- backend/go.sum — FOUND
- backend/Makefile — FOUND
- backend/.gitignore — FOUND
- backend/cmd/server/public/.gitkeep — FOUND
- sw.js (modified) — FOUND

Commits exist:
- 34c7a08 — FOUND (feat(09-01): Go server shell...)
- 7e93772 — FOUND (feat(09-01): partition SW fetch handler...)
