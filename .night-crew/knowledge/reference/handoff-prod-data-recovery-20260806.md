# HANDOFF — Production data recovery (B-145)

> **Written:** 2026-08-06, at the operator's direction, from the session that filed B-145.
> **For:** a fresh agent session, working attended (the operator is reachable — Phase 3 requires
> them). **Governs:** the recovery of the production data destroyed on 2026-08-06.
> **Reads first:** this file, then `BACKLOG.md` B-141/B-143/B-145, then HANDOFF
> `.night-crew/runs/2026-08-06-autonomous/HANDOFF.md` §"The incident".

## What happened, in three sentences

During run `20260806`, an adversarial review probe set `HQ_RLS_TEST_DB=yumyums` and the RLS test
suite dropped the production database on the shared :5433 cluster (B-141 — a blocklist guard that
could not contain the names that matter). No backup of any kind existed (B-143 — no PITR, no
dump, empty alternate volume), so prod was rebuilt **structurally empty** the same night: schema
migrated clean via goose, zero rows. The crew has been operating on that empty console since,
and the gap widens daily (B-145).

## 🛑 Ground rules — read before any command

1. **The test cluster IS the production cluster.** Everything on :5433 is production. Do not run
   any Go test suite, any `DROP`/`TRUNCATE`, or anything that resolves a database name from an
   env var against :5433 during this work. You do not need to run tests to do this recovery.
2. **Nothing destructive, ever, without the operator watching.** This handoff contains zero
   destructive steps by design. If you believe one is needed, stop and ask.
3. **Phase order is load-bearing.** The backup floor (Phase 1) lands before any reconstruction
   effort (Phase 4) — rebuilt data must be protected from day one. Do not reorder.
4. **Operator rulings are Phase 3 and only Phase 3.** Do not improvise a restore/write-off call.
   Everything before Phase 3 is measurement; everything after executes what was ruled.
5. **Deploy path:** code changes (the backup Taskfile target) go dev → main → push →
   `task prod:deploy` (`Taskfile.yml:178-221`; prod builds from the clone at
   `/mnt/c/Users/jcole/projects/yumyums-purchase-orders`, pinned to origin/main). Cron lines and
   `postgresql.conf` changes act on the live box directly and are not part of the image.
6. **Verify every mechanism named here before relying on it** — this file was written from code
   reading (citations inline), not from executing the recovery. If a cite is stale, say so in
   your report; do not push through it.

## Phase 0 — Establish the actual current state (~20 min, read-only)

Everything below is `SELECT`/`ls`/`GET` only.

1. **What prod holds now.** Find the container (`docker ps` — expect the compose-managed
   Postgres on :5433), then per-table counts on database `yumyums`:
   `users`, `sessions`, `workflow templates` + `submissions`/responses tables,
   `purchases`, `pending_purchases`, `items`, `vendors`, `stock_count_overrides`, `recipes`,
   `menu_items`, `daily_menu_sales`. Record the counts — some manual re-entry by the crew since
   2026-08-06 is likely, and Phase 4 must not clobber it.
2. **Login works.** Confirm the operator can authenticate. `backend/cmd/seed/main.go` seeds
   superadmins from `config/superadmins.yaml` (env: `DB_URL`, optional `SUPERADMIN_CONFIG`) —
   if no superadmin exists in prod, running the seed is the first fix, and it is additive.
3. **Spaces bucket inventory.** The bucket named by prod's `DO_SPACES_BUCKET` (wiring:
   `backend/cmd/server/main.go:327-353`) should hold: uploaded receipt files, fail/correction
   photos, and the Toast CSV archive (`internal/toast/sync.go:26-35` — cache file is written
   first and removed if the Spaces upload fails, so **the Spaces archive is the durable copy**).
   List each prefix, count objects, note date ranges. This is the single most important
   measurement: it bounds what is recoverable without any external party.
4. **Mercury API reachable.** With prod's `MERCURY_API_KEY`, list one page of transactions
   (`internal/receipt/mercury.go` — date-ranged, 500/page). Note how far back history goes.
5. **Toast SFTP reachable.** Defaults in `internal/toast/config.go:50-53` (AWS Transfer host,
   user `YumYumsExportUser`, export 113866; key at `TOAST_SFTP_KEY_PATH`). List available export
   date-directories. If SFTP has aged out older days, the Spaces archive from step 3 covers them.

## Phase 1 — The backup floor, before anything else (~1-2h)

This is the immediate half of card `prod-backup-floor-and-pitr` (decision 154, ledger §T-38).
Build it now so everything reconstructed in Phase 4 is protected.

1. **`task prod:backup`** — new Taskfile target: `pg_dump` of `yumyums` from the prod container
   (`docker exec <container> pg_dump -U yumyums -Fc yumyums > <dest>`) to a timestamped file
   **outside the Docker volume** — put it under a new directory on the Windows-side disk (a
   path that survives WSL and container rebuilds), keep the last 14 dumps.
2. **Cron line** invoking it nightly. This box runs WSL2 — verify cron actually fires here (or
   use a Windows scheduled task calling `wsl -- task prod:backup`); a backup job that silently
   never runs is B-143 shape again. Prove one scheduled firing before calling this done.
3. **Restore drill, against a scratch database on a scratch container — never against :5433.**
   `pg_restore` the first dump into a throwaway container and count tables. A dump nobody has
   restored is not a backup.
4. **PITR (`archive_mode=on` + local WAL archive) is the card's second half** — do it if time
   allows, but do not let it block Phase 2. Note it in your report either way.
5. Commit the Taskfile target on a branch off `dev`, PR/merge per the normal flow, and record
   in the report exactly which halves of decision 154 are now real.

## Phase 2 — Measure the windows (~30 min, read-only)

Turn Phase 0's reachability checks into a bounded statement per source:

| Source | What to measure |
|---|---|
| Mercury API | earliest transaction date retrievable; count of card transactions since the business's start |
| Spaces: receipts prefix | object count + date range |
| Spaces: Toast archive | date-directory range; any gaps |
| Toast SFTP | date-directories still present |
| Crew re-entry since 08-06 | from Phase 0 step 1 counts — what already exists and must be preserved |

## Phase 3 — 🛑 The operator's rulings (attended — this is the decision moment)

Present the Phase 2 table, then ask for a ruling per class. Frame each as restore / rebuild /
write off, with measured cost. Record every ruling in `ledger.md` as a numbered decision.

| Class | Where truth survives | Likely ruling to propose |
|---|---|---|
| Purchases + receipts | Mercury API + Spaces receipts | **Restore** via backfill (Phase 4.1) — but ask how far back; each receipt re-enters the *pending review* queue and costs manual review time |
| Item catalog + vendors | Rebuilds through receipt review | Rides the purchases ruling — items are created from receipts by design |
| Toast menu + daily sales | Spaces CSV archive + SFTP | **Restore** via re-import (Phase 4.2) |
| Users + grants | `config/superadmins.yaml` + operator's knowledge | **Rebuild**: seed superadmins, re-invite crew via users.html |
| Workflow templates | Nowhere — operator's memory | **Rebuild by hand** in the Builder; ask which templates existed and in what priority |
| Submissions + responses | Nowhere | Propose **write off** — historical accountability records; state it explicitly |
| Stock count overrides | Nowhere (point-in-time counts) | Propose **write off** — a fresh count is more accurate than a restored stale one |
| Recipes (`usage_pct`) | Nowhere | **Rebuild by hand** in the Recipes tab; flag that COGS attribution is wrong until done |
| Sessions | Nowhere | Write off (users just log in again) |

Also ask, as its own question: **what do we tell sales-processor's owner?** The contract
endpoints (`docs/contracts/inventory-period-summary.md`, `inventory-menu-cogs.md`) are computing
over a purchase history with a hole — the current payroll period's COGS is understated until the
backfill lands. The message and its timing are the operator's call; drafting it is yours.

## Phase 4 — Execute per ruling

**4.1 Mercury/receipt backfill.** Mechanism verified in code: the worker fetches a date-ranged
window and *"poll backfills any row inside the lookback window"*
(`internal/receipt/worker.go:157-164, :200`). Set `MERCURY_LOOKBACK_DAYS` (read at
`cmd/server/main.go:380-387`, default 14) to cover the ruled horizon, restart the container, and
either wait one `RECEIPT_WORKER_INTERVAL` (default 6h) or trigger the on-demand
`POST /api/v1/inventory/sync-receipts`. Watch for Anthropic parse costs at volume (Haiku per
receipt — fine; note the count in the report). Backfilled rows land in **pending review** —
tell the operator the queue size; the review/confirm work is the crew's, over days, not yours.
Afterward, set the lookback back to its normal value.

**4.2 Toast re-import.** Two commands exist: `backend/cmd/sync-toast` (the pull) and
`backend/cmd/migrate-toast-archive` (works the archive). Read both `main.go`s before running —
determine which re-imports historical date-dirs from the **Spaces archive** into
`daily_menu_sales`/`menu_items`, and whether either is idempotent over days that already have
rows (Phase 0 crew re-entry). Run for the ruled range; verify per-day row counts.

**4.3 Users.** Run the seed (additive) if Phase 0 found no superadmin; re-invite the crew
through users.html per the operator's list. Grants are per-tab (the go-forward convention) —
have the operator state each person's grants rather than guessing from memory.

**4.4 Hand rebuilds (templates, recipes).** These are the operator's/crew's authoring work.
Your job: confirm the tools work end-to-end on prod (create one test template, set one recipe
slider, then delete the test artifacts with the operator watching) and record the rebuild as
started, not done.

**4.5 The sales-processor notice.** Draft it from the rulings: which weeks' `period-summary` /
`menu-cogs` numbers were affected, what the backfill restored, what was written off. The
operator sends it. Note `process/C-1` (pending preference: one complete correction beats a drip
of partial ones) — if it has been adopted by the time you run, it binds this notice.

## Phase 5 — Verify and close out

1. Re-run Phase 0's counts; diff against Phase 2's targets per class.
2. `task health:prod` green; a receipt visible in the Purchases tab; a Toast day visible in Menu.
3. Confirm last night's cron dump actually appeared on disk (Phase 1 step 2's proof, repeated).
4. Write the outcome into `ledger.md` (one section: rulings, what was executed, counts, what
   remains open) and update **B-145** in `BACKLOG.md` — per-class status, not just "done".
5. Report back to the operator: TLDR first (what is restored, what is written off, what work
   remains and whose it is), then the evidence.

## What this handoff does NOT cover

- The A3 attended re-gate (B-141 prefix guard + B-142) — separate card, roadmap Activity 1.
- `test-cluster-separation` (decision 155's card) — overnight-slate material, not this session.
- Tonight's slate. If you are the same session that later plans it, finish Phase 3 first —
  the rulings change what the slate says about sales-processor and Activity 0.
