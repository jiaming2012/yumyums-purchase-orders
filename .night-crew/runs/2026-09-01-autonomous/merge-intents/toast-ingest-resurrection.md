# Merge-intent — `toast-ingest-resurrection` (Card 3, Track A)

Closes **B-146 (mechanism half)**. Ships the SFTP-key delivery mechanism + resurrects
the daily Toast sync config, provable dev-side. The real key is placed by the operator,
attended, on the prod box — NOT in this run.

## Shared files touched

### `docker-compose.prod.yml` — CHANGED
- Added a `volumes:` bind-mount of an operator-placed, git-ignored SFTP key file
  (`./id_rsa` beside the compose file) → `/app/id_rsa` **read-only** in the container.
- Pinned `TOAST_SFTP_KEY_PATH: /app/id_rsa` in the `environment:` block (absolute,
  in-container path — no longer relies on a `.env.prod` line that was silently absent).
- Pinned `TOAST_SYNC_INTERVAL: "12h"` in `environment:` to **enable** the in-process
  worker (resurrection). `0` disables; `12h` is the documented default.
- **What must survive:** the exact `DB_URL` line (search_path=production isolation),
  `env_file: .env.prod required:false`, `STORAGE_REQUIRED`, `ALERTS_ENABLED`, the
  `image`/`container_name`/`networks` block. I only ADDED a `volumes:` key and two
  `environment:` lines; I changed nothing else.
- **Safe to drop:** nothing. All three additions are load-bearing for resurrection.
- **No other Card touches `docker-compose.prod.yml`** in this slate (Card 2's
  fail-loud landed in Go + main.go, not this compose). Clean file.

### `.gitignore` — NOT CHANGED (already covers it)
- `id_rsa` was ALREADY on line 10 and `.env.*` on line 17. The key path is already
  git-ignored; no edit needed. (Verified: `git check-ignore id_rsa` → ignored.)

### Root `Taskfile.yml` — NOT TOUCHED
- Card 6 `sync-dev-one-command` also edits root Taskfile in a DIFFERENT section. I did
  not touch it — no collision. (Noted here per slate instruction only to confirm the
  non-overlap.)

## New files (additive, no conflict risk)
- `.night-crew/knowledge/reference/toast-archive-gap-20260901.md` — day-by-day archive
  gap enumeration + attended-steps note (deliverables 3 & 4).

## What G6 / orchestrator must know
- **NO PROD CREDENTIAL ENTERED THIS RUN.** No `id_rsa` file was created, copied, or
  committed. `git status` shows only tracked changes; no key artifact.
- The dev-provable half is DONE; the **prod proof** (a current date-directory from prod's
  next scheduled sync + `toast_sync` health flipping to `ok`) is ATTENDED post-deploy,
  per the roadmap flip note.
- The archive gap has **10 aged-out days permanently lost** (2026-07-25 → 2026-08-03) and
  **28 recoverable days** (2026-08-04 → 2026-08-31, as of tonight). Recovery of the
  recoverable window is a SEPARATE attended action (the worker's default 7-day window
  will not reach back 28 days on its own — see the ref doc's recovery note).
