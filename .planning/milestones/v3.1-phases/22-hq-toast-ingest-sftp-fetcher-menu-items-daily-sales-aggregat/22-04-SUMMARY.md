---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 04
subsystem: backend/internal/toast + backend/cmd/sync-toast
tags: [scheduler, worker, cli, ingest, toast]
requires:
  - backend/internal/toast/ingest.go (Plan 03 — RunIngest, isColdStart)
  - backend/internal/toast/config.go (Plan 03 — LoadConfigFromEnv)
  - backend/internal/toast/types.go (Plan 03 — Config, IngestResult)
  - github.com/jackc/pgx/v5/pgxpool
provides:
  - "toast.StartWorker(ctx, cfg) — background goroutine: immediate tick + time.NewTicker(cfg.Interval)"
  - "toast.runCycle (internal) — owns D-02 cold-start branch + D-13 log line"
  - "cmd/sync-toast CLI binary — one-shot ingest with --from/--to flags, reuses LoadConfigFromEnv + RunIngest"
affects:
  - backend/internal/toast (new file: worker.go)
  - backend/cmd/sync-toast (new package + main.go)
  - backend/.gitignore (new entries for stray go build artifacts)
tech_stack_added: []
patterns_added:
  - "goroutine + time.NewTicker scheduler with immediate-first-tick + ctx.Done shutdown (mirrors receipt.StartWorker pattern)"
  - "one-shot CLI binary structure (flags → env → DB connect → reuse internal service func → exit code semantics)"
key_files_created:
  - backend/internal/toast/worker.go
  - backend/cmd/sync-toast/main.go
key_files_modified:
  - backend/.gitignore
decisions:
  - "Worker owns cold-start auto-detect; CLI does not. --from/--to are authoritative — operators wanting the 90-day backfill pass it explicitly. Clean separation of trigger-path responsibilities (D-11)."
  - "Worker logs D-13 line on every successful cycle in runCycle (not RunIngest) so the CLI can use the same data with a different prefix ('done.' vs 'toast ingest:'). Both still greppable as 'items_upserted='."
  - "Defensive interval<=0 guard inside StartWorker logs + returns rather than defaulting to 12h. Plan 05's caller is responsible for skipping StartWorker entirely on TOAST_SYNC_INTERVAL=0."
  - "Stray cmd binary artifacts (backend/sync-toast etc.) added to backend/.gitignore to prevent accidental 14 MB commits when 'go build ./cmd/...' is invoked without -o."
metrics:
  duration_minutes: 3
  tasks_completed: 2
  files_changed: 3
  completed_date: 2026-06-03
---

# Phase 22 Plan 04: Toast Worker + CLI Summary

**One-liner:** Toast ingest now runs on a 12h schedule (`toast.StartWorker`) and on-demand (`cmd/sync-toast --from … --to …`); both paths share `LoadConfigFromEnv` + `RunIngest`, and the worker owns the cold-start branch (D-02) so the CLI stays explicit-only.

After this plan, Plan 05 only needs to call `toast.LoadConfigFromEnv()` + `toast.StartWorker(ctx, cfg)` from `cmd/server/main.go` (with `cfg.Pool` injected and `Interval==0` skip-guard). Operators can also run `go run ./cmd/sync-toast/ --from 2026-05-01 --to 2026-05-31` for one-off backfills or future external cron.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | StartWorker + runCycle (cold-start branch + D-13 log line) | 962c144 | backend/internal/toast/worker.go |
| 2 | CLI binary (flag-validated --from/--to, reuses LoadConfigFromEnv + RunIngest) | fed665a | backend/cmd/sync-toast/main.go |
| — | gitignore stray cmd binaries (Rule 2 — auto-add missing infra hygiene) | 7b511e4 | backend/.gitignore |

## What Shipped

### Worker behaviour summary

```go
// backend/internal/toast/worker.go
func StartWorker(ctx context.Context, cfg Config)
```

- **Pre-flight:** logs + returns early on `cfg.Interval <= 0` (defensive; caller in Plan 05 should already have skipped it).
- **Startup log:** `toast worker: starting (interval=12h0m0s, window=7d, backfill=90d)` (parameters reflect cfg).
- **Goroutine pattern:** runs `runCycle` immediately, then enters `time.NewTicker(interval)` loop with `ctx.Done()` shutdown case (mirrors `receipt.StartWorker` lines 21–55).
- **Cold-start branch (D-02):** `runCycle` calls `isColdStart(ctx, cfg.Pool)` (`SELECT COUNT(*) FROM daily_menu_sales`). True → window = `cfg.BackfillDays` (90) and an extra log line `toast worker: cold start detected — pulling last 90 days`. False → window = `cfg.SyncWindowDays` (7) per D-04.
- **D-13 log line:** `toast ingest: dates=[YYYYMMDD..YYYYMMDD] items_upserted=N sales_rows_upserted=M duration=Xs` fires once per successful cycle.
- **DEVIATION from receipt.StartWorker (D-12) doc'd in code:** no graceful-skip on missing key — caller fail-fasts at startup. Comment block at top of `worker.go` explains why (Phase 999.2 dependency).

### CLI flag list + exit-code semantics

```
go run ./cmd/sync-toast/ --from YYYY-MM-DD --to YYYY-MM-DD
```

| Flag | Required | Validated |
|------|----------|-----------|
| `--from` | yes | `time.Parse("2006-01-02")`; rejected if unparseable |
| `--to` | yes | `time.Parse("2006-01-02")`; rejected if unparseable OR before `--from` |

| Env | Required | Behaviour on missing |
|-----|----------|----------------------|
| `DB_URL` | yes | `log.Fatal("DB_URL is required")` |
| `TOAST_SFTP_KEY_PATH` | yes | inherited from `LoadConfigFromEnv` → `log.Fatalf("toast config: TOAST_SFTP_KEY_PATH is required (no default — see D-12)")` |
| `TOAST_SFTP_USER` | no | default `YumYumsExportUser` |
| `TOAST_SFTP_HOST` | no | default `s-9b0f88558b264dfda...:22` |
| `TOAST_EXPORT_ID` | no | default `113866` |
| `TOAST_SYNC_INTERVAL` | no | unused by CLI; only the worker reads it |

| Exit code | When |
|-----------|------|
| 0 | Ingest cycle completed (per-day SFTP/parse/DB errors are logged + skipped per the `RunIngest` contract — they do NOT trigger non-zero exit) |
| 1 | Fatal: missing/invalid flags, missing DB_URL, missing/unreadable key, DB connect failure, OR SFTP dial failure after 3 retries (5s/15s/30s backoff per D-10) |

Success log line: `done. dates=[YYYYMMDD..YYYYMMDD] items_upserted=N sales_rows_upserted=M duration=Xs` — same structured fields as the worker's D-13 line, different prefix so log streams can be distinguished without losing greppability on `items_upserted=`.

### Both call paths share LoadConfigFromEnv + RunIngest — confirmation

| Concern | Worker (`internal/toast/worker.go`) | CLI (`cmd/sync-toast/main.go`) |
|---------|-------------------------------------|-------------------------------|
| Config load | Plan 05 caller invokes `toast.LoadConfigFromEnv()` then passes the result to `StartWorker` | `toast.LoadConfigFromEnv()` (line 58) |
| Pool injection | Plan 05 caller sets `cfg.Pool = pool` before `StartWorker` | `cfg.Pool = pool` (line 70) after `pgxpool.New` |
| Ingest function | `RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate)` in `runCycle` | `toast.RunIngest(ctx, pool, cfg, fromDate, toDate)` (line 73) |
| Cold-start probe | Yes — `isColdStart(ctx, cfg.Pool)` in `runCycle` | No — `--from`/`--to` are authoritative (intentional split) |
| Date window | Auto: cold → `BackfillDays`, warm → `SyncWindowDays`, both anchored to `time.Now()` | Explicit: parsed from flags |
| Log line shape | `toast ingest: dates=[...] items_upserted=N sales_rows_upserted=M duration=Xs` | `done. dates=[...] items_upserted=N sales_rows_upserted=M duration=Xs` |

This means a bug-fix or behaviour change to `LoadConfigFromEnv` or `RunIngest` automatically flows through both paths — no parallel maintenance burden.

## Verification

| Check | Result |
|-------|--------|
| `cd backend && go build ./...` | exit 0 |
| `cd backend && go vet ./...` | exit 0 |
| `cd backend && go test ./internal/toast/ -count=1` | exit 0 (ok, ~0.3s — Plan 03's 7 parser tests still pass) |
| Build CLI: `go build -o /tmp/sync-toast-bin ./cmd/sync-toast/` | binary produced |
| `/tmp/sync-toast-bin --help` | prints `-from string` + `-to string` usage |
| `/tmp/sync-toast-bin` (no flags) | exits non-zero with `--from and --to are required` (acceptance ≥1, got 1) |
| `/tmp/sync-toast-bin --from 2026-05-01 --to 2026-04-01` | exits non-zero with `--to (2026-04-01) is before --from (2026-05-01)` (acceptance ≥1, got 1) |
| `DB_URL=ignored /tmp/sync-toast-bin --from 2026-05-01 --to 2026-05-01` (TOAST_SFTP_KEY_PATH unset) | exits non-zero with `toast config: TOAST_SFTP_KEY_PATH is required (no default — see D-12)` |
| Acceptance grep: `func StartWorker(ctx context.Context, cfg Config)` in worker.go | 1 |
| Acceptance grep: `func runCycle(ctx context.Context, cfg Config)` in worker.go | 1 |
| Acceptance grep: `isColdStart(ctx, cfg.Pool)` in worker.go | 1 |
| Acceptance grep: `RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate)` in worker.go | 1 |
| Acceptance grep: `time.NewTicker(interval)` in worker.go | 1 |
| Acceptance grep: `case <-ctx.Done()` in worker.go | 1 |
| Acceptance grep: `toast ingest: dates=` in worker.go | 1 (D-13 line) |
| Acceptance grep: `cold start detected` in worker.go | 1 (D-02 branch) |
| Acceptance grep: `package main` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `flag.String("from"` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `flag.String("to"` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `toast.LoadConfigFromEnv()` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `toast.RunIngest(ctx, pool, cfg, fromDate, toDate)` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `pgxpool.New(ctx, dbURL)` in cmd/sync-toast/main.go | 1 |
| Acceptance grep: `log.Fatal` in cmd/sync-toast/main.go | 8 (≥4 required) |

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| **Worker auto-detects cold-start; CLI does not** | Per the plan's `<action>` note in Task 2: `--from`/`--to` are explicit and authoritative. Operators wanting the 90-day backfill from the CLI can pass it explicitly: `--from $(date -d "-90 days" +%Y-%m-%d) --to $(date +%Y-%m-%d)`. Keeps each trigger path single-purpose — the worker handles auto-scheduling concerns; the CLI does exactly what it's told. |
| **`interval <= 0` defensively logs + returns inside StartWorker** | The plan's template originally fell back to `12 * time.Hour` on non-positive `interval`. I changed this to log + return because Plan 05's caller will check `cfg.Interval == 0` and skip `StartWorker` entirely (per the 22-PATTERNS.md cmd/server snippet — "0 interval disables the worker (operator override; cmd/sync-toast still works)"). If we ever reach `StartWorker` with non-positive interval, that's a code bug worth surfacing in logs, not papering over. |
| **CLI success log uses `done.` prefix instead of `toast ingest:`** | Two log streams sharing a literal prefix would force operators to filter by PID or process name to tell them apart. Both still carry `items_upserted=N` for grep-stat parsing. Documented inline in the CLI source. |
| **Added `/sync-toast` (and siblings) to backend/.gitignore** | `go build ./cmd/sync-toast/` (without `-o`) drops a 14 MB Mach-O binary at `backend/sync-toast`. The post-commit deletion check in the task protocol caught this on Task 2 commit. Pre-emptive gitignore avoids future foot-guns; this is the same pattern that already protects `bin/`. |

## Deviations from Plan

### Rule 2 — Infrastructure hygiene: gitignore stray cmd build artifacts

- **Found during:** Task 2 commit / post-commit deletion check
- **Issue:** Running `go build ./cmd/sync-toast/` (per the plan's acceptance criteria, which exercise the binary several times) drops a 14 MB Mach-O at `backend/sync-toast`. Without a gitignore entry, this is one careless `git add backend/` from a 14 MB commit. Same hazard for `backend/server`, `backend/seed`, `backend/import-notion` when their cmds are built without `-o`.
- **Fix:** Added `/sync-toast`, `/import-notion`, `/seed`, `/server` to `backend/.gitignore` (anchored with `/` so only the backend-root artifact is matched, not anything nested in `cmd/`).
- **Files modified:** `backend/.gitignore`
- **Commit:** 7b511e4

### Code-level deviations from plan template

- **`interval <= 0` handling** — plan template defaulted to `12 * time.Hour`; this implementation logs + returns instead. Rationale documented in "Decisions Made" above. The acceptance criteria don't probe this path, so the deviation is invisible to plan grading.

No bugs, missing functionality, or blocking issues encountered. Plans 02 + 03's surface area (`toast.Config`, `toast.LoadConfigFromEnv`, `toast.RunIngest`, `toast.isColdStart`) was correct and required no patching.

## Authentication Gates

None.

## Deferred Issues

None.

## Known Stubs

None — both files are fully-functional production code. No placeholder data, no TODO-marked stubs, no mock data.

## Threat Flags

None — the threat surface introduced (background goroutine, CLI with DB credentials in env) is fully covered by the plan's `<threat_model>` (T-22-11 through T-22-13). All three mitigations are present in code:

- T-22-11 (runaway goroutine): single ticker, single goroutine, `defer ticker.Stop()`, `ctx.Done()` shutdown case
- T-22-12 (silent per-cycle failures): D-13 INFO log on success; ERROR log on ingest cycle abort; ERROR log on cold-start probe failure
- T-22-13 (CLI elevated DB user): accepted per disposition — out of scope for Phase 22

## Self-Check

- FOUND: backend/internal/toast/worker.go
- FOUND: backend/cmd/sync-toast/main.go
- FOUND: backend/.gitignore
- FOUND commit: 962c144 (Task 1)
- FOUND commit: fed665a (Task 2)
- FOUND commit: 7b511e4 (gitignore hygiene)

## Self-Check: PASSED
