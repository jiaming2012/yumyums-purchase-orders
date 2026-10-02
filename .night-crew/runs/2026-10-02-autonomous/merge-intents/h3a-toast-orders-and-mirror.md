# Merge intent — Card H3a · `toast-orders-and-mirror`

Run `20261002` · branch `card/h3a-toast-orders-and-mirror` off `overnight-20261002` at `94ad3b3`
(i.e. **after** Card 1 / H1 merged, so `backend/internal/marketing/` already exists).
Track C, first half. H3b (`reconciliation-and-stats-engine`) and H6 (`scanner-polish`) land behind it.

## Shared files touched

| File | Why |
|---|---|
| `backend/cmd/server/main.go` | **ONE new call site** — `marketing.MirrorStart(ctx, pool, mktDeps.Projection)` inside the existing `schedulersDisabled` region, immediately before the Toast worker block. Beside H1's three. Undeclared seam in `night-crew.toml` → this card owes the FULL Playwright suite. |
| `backend/internal/marketing/mirror.go` | **NEW FILE in H1's package.** Add-a-file, no restructure. Every package-level identifier carries the `Mirror…` prefix the slate assigned this card. |
| `backend/internal/marketing/routes.go` | **NOT TOUCHED.** The mirror registers no HTTP route — it is a poller, not an endpoint. Nothing appended; no labelled block. If a merge offers a `routes.go` hunk from this card, it is wrong. |
| `backend/internal/db/migrations/0084_toast_orders_reconciliation.sql` | New file, next free number (0083 is H1's). |
| `backend/internal/redemption/store.go` | B-424: `PGRaceLostStore.Emit`'s INSERT gains `ON CONFLICT … DO NOTHING`. One statement; the F4 observer path is otherwise untouched. |
| `backend/internal/toast/sync.go` | The OrderDetails leg rides the **already-open** SFTP client at the end of `SyncDate`. Addition-only — see "ItemSelection untouched" below. |
| `backend/internal/toast/orderdetails.go` | New file: parser + upsert. |
| `backend/internal/toast/main_test.go` | New file: the package's first `TestMain` (DB-backed tests need a migrated pool; `internal/testdb`'s asymmetric gate, copied from `internal/marketing/helpers_test.go`). |
| `backend/internal/toast/testdata/OrderDetails.csv` | Fixture of the real export's column set and formats (see "Decisions"). |
| `.night-crew/knowledge/roadmap.md` | H3a's card line PLANNED → LANDED. Every card edits this file; expect a conflict and take both sides. |
| `.night-crew/knowledge/BACKLOG.md` | B-424 `closed → toast-orders-and-mirror` (a done_when row). Line-local; take both sides. |
| `.night-crew/runs/2026-10-02-autonomous/logs/h3a/*`, this note | Gate logs + merge intent. |

**No frontend file. No `sw.js` content change** (`build-sw.js` is still re-run and the regenerated
file committed per B-13, but no precached asset is added or removed — the count stays **48**).
**No `night-crew.toml` key** — `backend/internal/toast`, `backend/internal/marketing` and
`backend/cmd/server` are all already unmapped in `[e2e.seams]`, which is what de-confines this
card to the full suite. **No new app grant, no new env var** (the mirror reads the same
`HQ_SYNC_REST_URL` / `HQ_SYNC_SERVICE_KEY` pair H1's projection already reads, through H1's own
`marketing.ProjectionConfig`).

## What must survive any merge

- **`backend/internal/marketing/mirror.go` as a FILE.** It declares `MirrorInterval`,
  `MirrorPageSize`, `MirrorCursor`, `MirrorAttempt`, `MirrorResult`, `MirrorPollOnce`,
  `MirrorStart`, `MirrorLiveEnv` — all `Mirror`-prefixed so no sibling card collides. It adds
  **no field to `Deps`** and **no route to `Mount`/`MountReports`**: `MirrorStart` takes
  `(ctx, *pgxpool.Pool, ProjectionConfig)` so H1's `Deps` shape is untouched.
- **The one `main.go` call site**, inside the `schedulersDisabled` region:
  `if schedulersDisabled { slog.Info("scan-attempts mirror disabled", …) } else { marketing.MirrorStart(ctx, pool, mktDeps.Projection) }`.
  Placed before the Toast worker block so the two background subsystems read as one region.
- **Migration 0084 with ALL THREE tables** — `toast_orders`, `scan_attempts_mirror`,
  `reconciliation_decisions` — Down included. H3b must add no migration; it only reads/writes
  these. `reconciliation_decisions.attempt_id` references `scan_attempts_mirror(id)`, so the
  two tables cannot be split across migrations.
- **B-424's unique index** `race_lost_notifications_dedupe_uq (code_token_hash, device_id, scanned_at)`
  **together with** the `ON CONFLICT (code_token_hash, device_id, scanned_at) DO NOTHING` in
  `PGRaceLostStore.Emit`. Either half alone is a regression: the index without the clause turns a
  replay into a 23505 error at the arbitration response, and the clause without the index does not
  compile against Postgres (no matching constraint) — it is a single change in two files.
- **The upsert key `(business_date, order_number)`** on `toast_orders` and the
  `ON CONFLICT … DO UPDATE` in `UpsertOrders`. Spike 02 proved it unique over 77 real orders.
- **`SyncDate`'s existing `(bool, error)` contract and its `ErrSFTPMiss` / `ErrSFTPUnavailable`
  classification.** The OrderDetails leg is appended at the end and cannot change either.

## What is safe to drop

- `backend/internal/toast/testdata/OrderDetails.csv`'s exact contents and
  `TestParseOrderDetailsCoercions` — hermetic hygiene around tonight's parse coercions, not a
  done_when row. If a later card gets the real export to commit, it replaces both.
- The `orders_upserted` key in the per-date `slog.Info` line.
- `TestMigration0084DownAndUpRoundTrip` — proves 0084's Down; if a later migration makes the
  down-to-83 leg awkward, update it rather than preserving this spelling.
- `backend/internal/toast/main_test.go`'s fallback DSN, if a later card centralises TestMain.

## `## Red-first`

Filled after the reds were observed; see the commit that follows this note on this branch.

## `TestScanAttemptsMirrorKeysetResumes` — LIVE substrate or recorded fixture?

Stated in the commit that fills `## Red-first`, and in the card report. The intent is **LIVE**:
the local `spike-supabase` stack (compose project `spike-supabase`, ports resolved with
`docker compose -p spike-supabase port rest 3000`) with a `service_role` JWT minted from the
committed throwaway `JWT_SECRET`, exactly as spike 03 does. The test seeds rows into the
substrate's **own** `public.scan_attempts` (the device's job) and then calls the **shipped**
`marketing.MirrorPollOnce` to do the PostgREST read and the HQ-Postgres upsert; it writes nothing
the shipped code is supposed to write.

## What the mirror makes visible (the F4 `scan_attempts`-status acceptance bullet, B-424)

Until tonight, `scan_attempts.status` / `match_status` lived **only** on the device-facing Supabase
substrate, which RLS makes write-only for devices and readable by `service_role` alone — so HQ had
no server-side copy of "what happened at the counter", and F4's acceptance bullet ("the attempt's
status is visible") was owned by no card. The mirror is that copy. Per attempt it lands:

- **`status`** (`pending` / `accepted` / `rejected`) and **`reason`** (`already_used` / `expired` /
  `not_found` / …) — so a refusal at the counter is inspectable in HQ without a substrate query.
- **`match_status`** (`unmatched` / `matched` / `orphan`) as stored upstream — the bucket H3b's
  queue and the orphan rate are computed from.
- **`offline_override` + `override_by`** — which redemptions were taken offline on someone's say-so,
  and whose.
- **`unverified_code` + `token_hash`** — F2: an override on a code the device's replica could not
  verify, which is the highest-priority follow-up in F4.
- **`policy_unresolved`** — B-432/B-436's audit column: the override happened because the campaign
  policy could not be resolved, not because the code was genuinely unknown.
- **`pos_order_number` + `pos_business_date`** — the §13 join key into `toast_orders`, which is why
  this card lands both halves at once: with only one of them there is nothing to reconcile.

Concretely: a replayed reconciliation now produces **one** `race_lost_notifications` row (B-424's
dedupe), and the attempt behind it is visible in HQ Postgres with its status, its reason and its
override provenance — which is the acceptance bullet.

## Not in this card (stated so H3b is not surprised)

- `backend/internal/marketing/reconciliation.go`, `GET /reconciliation/*`, `POST
  /reconciliation/{id}/{match,decline,reopen,verify,reject}`, `/stats/*`, the `MountReports` body
  and the `money` block — **all H3b's**. H3a lands the two tables they read, the third table they
  write, and nothing that serves HTTP.
- `scan_attempts_mirror.campaign_id` is mirrored as **NULL** tonight — see "Decisions".
