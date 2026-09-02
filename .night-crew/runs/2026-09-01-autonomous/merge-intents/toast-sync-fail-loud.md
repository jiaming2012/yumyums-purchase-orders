# Merge-intent — `toast-sync-fail-loud` (Card 2, Track A)

Split (a) of roadmap card `pipeline-fail-loud`; closes the **B-146 silent-death
class** (the fail-loud half — Card 3 ships the SFTP key separately).

## What this card does

Prod's Toast sync has been silently dead since the 2026-07-28 image rebuild
because the SFTP private key never shipped. NOTHING surfaced it: an SFTP
dial/auth failure in `SyncDate` (`internal/toast/sync.go:51-56`) was downgraded
to `ErrSFTPMiss`, which the worker treats as an EXPECTED, silent miss (INFO log,
no failure counter, no alert). A dead transport was indistinguishable from
"this date isn't on the server yet."

This card makes the silent death loud:

1. **New sentinel `ErrSFTPUnavailable`** (`sync.go`) distinct from `ErrSFTPMiss`.
   A dial/auth failure returns `ErrSFTPUnavailable` (transport dead), NOT
   `ErrSFTPMiss` (date-not-found, still graceful per D-05).
2. **New `SyncStatus` in `internal/toast`** (`syncstatus.go`) — a testable,
   concurrency-safe last-sync tracker mirroring `photos.StorageHealth` shape.
   The worker writes it; `/api/v1/health` reads it. States: `ok` (last cycle
   reached SFTP), `failing` (SFTP unreachable/auth-failed this cycle),
   `stale` (no successful sync within a staleness window), `unknown` (never run).
3. **Worker fires the Cliq alert IMMEDIATELY** on `ErrSFTPUnavailable` (SFTP dead
   = whole pipeline dead, unlike per-date Spaces flakiness which keeps its 3-tick
   threshold). Reuses the existing `alerts.Queue` enqueue path — no new mechanism.
4. **`/api/v1/health` gains a `toast_sync` field** — see shape below.

## New /health field shape (orchestrator + Card 3 need this)

The `toast_sync` field mirrors the existing `storage` string field's spirit but
carries timestamps, so it is an object:

```json
"toast_sync": {
  "status": "ok" | "failing" | "stale" | "unknown",
  "last_success": "2026-09-01T12:00:00Z",   // RFC3339, omitted if never
  "last_error": "2026-09-01T12:05:00Z",     // RFC3339, omitted if none
  "last_error_summary": "sftp dial/auth failed: ..."  // omitted if none
}
```

Rendered by `toast.SyncStatus.Snapshot()` → a `toast.SyncStatusView` struct with
json tags. Health handler adds one key: `"toast_sync": toastSync.Snapshot()`.

## Shared files touched

- `backend/cmd/server/main.go` — TWO edits, both additive:
  1. Construct `toastSync := toast.NewSyncStatus(staleAfter)` and wire it into
     the worker (`toastCfg.SyncStatus = toastSync`, or `toast.SetSyncStatus`)
     right beside the existing `toast.SetAlertQueue(alertQ)` at ~line 870.
  2. Add `"toast_sync": toastSync.Snapshot()` to the `/health` map at ~line 491.
  `main.go` is a shared/hot file — kept to two additive touches, no reordering of
  existing init, no signature changes to anything else. Reason it must be touched:
  the health handler is inline in `main.go` and the worker is wired there; there
  is no other seam.
- `.night-crew/knowledge/roadmap.md` — flip `pipeline-fail-loud` to DONE (both
  halves now landed). Sibling (b) already noted itself and explicitly did NOT flip.

## Own-module files (not shared)

- `backend/internal/toast/syncstatus.go` — NEW. The `SyncStatus` type.
- `backend/internal/toast/sync.go` — `ErrSFTPUnavailable` sentinel + return it on
  dial failure instead of `ErrSFTPMiss`.
- `backend/internal/toast/worker.go` — route `ErrSFTPUnavailable` to loud path
  (record status + immediate alert); record `ok` on any successful cycle.
- `backend/internal/toast/syncstatus_test.go`, `worker_faliloud_test.go` — NEW
  red-first tests (fake SFTP dialer/config forcing dial failure; fake alert sink;
  assert health status + alert enqueue red-before / green-after).

## What must survive any merge

- The `/health` `toast_sync` field (status + timestamps) — the fail-loud signal.
- The immediate Cliq alert on SFTP-unavailable via the existing `alerts.Queue`.
- `ErrSFTPMiss` semantics UNCHANGED for genuine date-not-found (D-05 graceful skip
  must still be silent — we only split OUT the transport-failure case).
- The existing 3-consecutive-Spaces-failure degraded-alert path (do not disturb).

## What is safe to drop

- Exact status string spellings (`failing`/`stale`) are cosmetic — but if Card 3
  or the orchestrator has assumed a spelling, keep mine.

## Nothing-here

No frontend, no HTML/JS, no Playwright specs, no migrations, no new env vars, no
new dependencies. Go-only.
