---
phase: 09-foundation-auth
plan: 02
subsystem: backend/db
tags: [postgres, goose, migrations, pgxpool, config, superadmin]
dependency_graph:
  requires: [09-01]
  provides: [db-pool, goose-migrations, superadmin-config]
  affects: [09-03, 09-04]
tech_stack:
  added:
    - github.com/jackc/pgx/v5 v5.9.1 (pgxpool + stdlib adapter)
    - github.com/pressly/goose/v3 v3.27.0 (embedded SQL migrations)
    - gopkg.in/yaml.v3 v3.0.1 (superadmin YAML config)
  patterns:
    - goose embedded FS migrations (goose.SetBaseFS + goose.Up)
    - stdlib.OpenDBFromPool wrapper (goose requires *sql.DB, not pgxpool)
    - LoadSuperadmins returns map[email]SuperadminEntry for O(1) lookup at API layer
key_files:
  created:
    - backend/internal/db/db.go
    - backend/internal/db/migrations/0001_users.sql
    - backend/internal/db/migrations/0002_sessions.sql
    - backend/internal/db/migrations/0003_invite_tokens.sql
    - backend/internal/db/migrations/0004_hq_apps.sql
    - backend/internal/db/migrations/0005_app_permissions.sql
    - backend/internal/config/config.go
    - backend/config/superadmins.yaml
  modified:
    - backend/cmd/server/main.go
    - backend/Makefile
decisions:
  - "sessions.expires_at is nullable (per D-03) — sessions live indefinitely until explicit logout"
  - "0004_hq_apps.sql has schema only, no seed data (per D-10) — db-seed Makefile target handles seeding"
  - "stdlib.OpenDBFromPool used as goose adapter — goose requires *sql.DB not pgxpool.Pool"
metrics:
  duration_minutes: 3
  tasks_completed: 2
  tasks_total: 2
  files_created: 8
  files_modified: 2
  completed_date: "2026-04-15"
---

# Phase 09 Plan 02: Postgres + Goose Migrations + Superadmin Config Summary

**One-liner:** pgxpool connection setup with embedded goose SQL migrations (5 tables) and YAML-parsed superadmin bootstrap config wired into main.go startup sequence.

## What Was Built

### Task 1: pgxpool setup, goose migrations, Makefile DB targets

**`backend/internal/db/db.go`** — two exported functions:
- `NewPool(ctx, connStr)` — creates a pgxpool.Pool with MaxConns=10, pings on startup
- `Migrate(pool)` — wraps pool with `stdlib.OpenDBFromPool`, calls `goose.Up` against embedded migrations FS

**5 migration files** in `backend/internal/db/migrations/`:
- `0001_users.sql` — users table with role/status CHECK constraints, nullable password_hash
- `0002_sessions.sql` — sessions table with nullable `expires_at` (per D-03), `sessions_user_idx` index
- `0003_invite_tokens.sql` — invite_tokens with `expires_at NOT NULL`, nullable `used_at`
- `0004_hq_apps.sql` — hq_apps schema only, no seed data (per D-10)
- `0005_app_permissions.sql` — app_permissions with XOR constraint (`role_or_user CHECK`) and two partial unique indexes

**`backend/Makefile`** — added targets:
- `db-start` — runs postgres:16 Docker container with yumyums-pgdata volume
- `db-stop` — stops and removes container
- `db-reset` — db-stop + volume removal + db-start
- `db-seed` — inserts 7 hq_apps rows (purchasing, payroll, scheduling, hiring, bi, users, operations) via `ON CONFLICT (slug) DO NOTHING`

### Task 2: Superadmin config parser + main.go wiring

**`backend/internal/config/config.go`** — `LoadSuperadmins(path)` reads YAML file, returns `map[string]SuperadminEntry` keyed by email for O(1) lookup at auth layer.

**`backend/config/superadmins.yaml`** — bootstrap config with jamal@yumyums.com as initial superadmin.

**`backend/cmd/server/main.go`** — startup sequence now:
1. Resolve static file source (disk vs embed.FS)
2. Load superadmin config (fatal if unreadable)
3. Connect to DB via `db.NewPool` (fatal if DB_URL missing or unreachable)
4. Run goose migrations via `db.Migrate` (fatal if any migration fails)
5. Start chi router + HTTP server

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan is pure infrastructure (schema + config). No UI or data stubs.

## Self-Check: PASSED

All 11 files confirmed present. Both task commits verified (9777fa0, 350161e).
