---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 02
subsystem: backend/internal/toast
tags: [sftp, port, deps, backend]
requires:
  - sales-processor/sftp/default.go (verbatim source)
provides:
  - "package toast in backend/internal/toast (SFTPConfig, Client, New, Download, Close, Create, Upload, Info)"
  - "github.com/pkg/sftp v1.13.5 as direct dep"
  - "golang.org/x/crypto v0.52.0 as direct dep"
affects:
  - backend/go.mod
  - backend/go.sum
tech_stack:
  added:
    - github.com/pkg/sftp v1.13.5
  patterns:
    - "package-internal SFTP transport (sibling of receipt/, purchasing/)"
key_files:
  created:
    - backend/internal/toast/sftp.go
  modified:
    - backend/go.mod
    - backend/go.sum
decisions:
  - "Renamed Config -> SFTPConfig on port (Plan 03 reserves type Config for worker)"
  - "Kept HostKeyCallback returning nil (matches sales-processor production behavior; documented in file header)"
  - "Preserved Create/Upload/Info methods even though Phase 22 doesn't call them — keeps the port honest"
metrics:
  duration_sec: 135
  duration_min: 2
  tasks_completed: 2
  files_changed: 3
  completed_at: 2026-06-03T05:40:11Z
---

# Phase 22 Plan 02: Port SFTP Client + Promote Deps Summary

**One-liner:** Ported sales-processor SFTP client verbatim into `backend/internal/toast` (3 documented edits: package, struct rename, doc header) and promoted `github.com/pkg/sftp v1.13.5` + `golang.org/x/crypto v0.52.0` to direct deps.

## What Was Built

Plan 03's ingest layer now has a working SFTP transport. The file `backend/internal/toast/sftp.go` exposes:

```go
toast.SFTPConfig{Username, Password, PrivateKey, Server, KeyExchanges, Timeout}
toast.New(SFTPConfig) (*Client, error)
client.Download(filePath) (io.ReadCloser, error)
client.Close()
// also Create, Upload, Info (preserved for parity)
```

The file is 180 lines (10 lines of doc header + 170 lines verbatim from sales-processor). `diff` against the original source shows only the three documented changes:

1. `package sftp` → `package toast` (with file-level doc comment explaining the rename)
2. `type Config struct` → `type SFTPConfig struct`
3. `func New(config Config)` → `func New(config SFTPConfig)` + matching `Client.config` field rename

## Commits

| Task | Commit  | Files                                  |
| ---- | ------- | -------------------------------------- |
| 1    | 87a2fff | backend/internal/toast/sftp.go         |
| 2    | 64f9c25 | backend/go.mod, backend/go.sum         |

## Verification

| Check                                                            | Result |
| ---------------------------------------------------------------- | ------ |
| `cd backend && go build ./...`                                   | exit 0 |
| `cd backend && go vet ./internal/toast/`                         | exit 0 |
| `cd backend && go mod verify`                                    | all modules verified |
| `grep -c 'package toast' backend/internal/toast/sftp.go`         | 1      |
| `grep -c 'type SFTPConfig struct' backend/internal/toast/sftp.go`| 1      |
| `grep -c 'type Config struct' backend/internal/toast/sftp.go`    | 0      |
| `grep -c 'func New(config SFTPConfig)' .../sftp.go`              | 1      |
| `grep -c 'func (c \*Client) Download(filePath string)' .../sftp.go` | 1   |
| `grep -c 'func (c \*Client) Close()' .../sftp.go`                | 1      |
| `grep -c 'github.com/pkg/sftp v1.13.5' backend/go.mod`           | 1      |
| `grep 'github.com/pkg/sftp' backend/go.mod \| grep -c indirect`  | 0      |
| `grep 'golang.org/x/crypto' backend/go.mod \| grep -c indirect`  | 0      |

## Decisions Made

- **Renamed Config → SFTPConfig** (deviates from sales-processor naming). Reason: Plan 03 introduces `toast.Config` for the worker; keeping both names distinct prevents a confusing collision inside the package.
- **Kept HostKeyCallback returning nil** (no `ssh.FixedHostKey`). Reason: Toast's AWS Transfer Family endpoint doesn't publish a stable host key. sales-processor has run this way in production for years. Documented in the file header. Threat T-22-03 explicitly accepts this risk.
- **Pinned `github.com/pkg/sftp` to v1.13.5 explicitly** (not `@latest`). Reason: matches sales-processor's pinned version, so behavior is identical.
- **Upgraded `golang.org/x/crypto` to `@latest` (v0.50.0 → v0.52.0)**. The ssh subpackage surface used (`Dial`, `ClientConfig`, `Config{KeyExchanges}`, `ParsePrivateKey`, `PublicKeys`, `Password`) has been stable for years; no risk.
- **Preserved Create/Upload/Info methods** even though Phase 22 doesn't call them. Reason: keeps the port honest (verbatim, plus three deliberate edits) and lets future phases reuse the surface without re-porting.

## Deviations from Plan

None — plan executed exactly as written.

## Authentication Gates

None.

## Deferred Issues

None.

## Threat Flags

None — the threat surface introduced (`SFTPConfig` struct holding a PEM string, HostKeyCallback returning nil) is fully covered by the plan's `<threat_model>` (T-22-03, T-22-04, T-22-05).

## Self-Check: PASSED

- backend/internal/toast/sftp.go: FOUND
- backend/go.mod: MODIFIED (sftp v1.13.5 direct, crypto v0.52.0 direct)
- backend/go.sum: MODIFIED
- Commit 87a2fff: FOUND
- Commit 64f9c25: FOUND
