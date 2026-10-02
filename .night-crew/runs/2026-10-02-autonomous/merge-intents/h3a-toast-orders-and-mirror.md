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
| `backend/internal/marketing/zz_migration_down_test.go` | **ONE LINE in H1's test.** Its final leg was `db.MigrateTo(pool, 83)`, which left the whole test database pinned at 83 — i.e. with 0084's tables DROPPED for every package that runs after `internal/marketing` under `-p 1`. Changed to `db.Migrate(pool)` so it leaves the schema at the LATEST, which is what its own comment already promised ("no other test sees a half-migrated database"). Every future card that adds a migration needs this; take this side at merge. |
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
  `MirrorStart`, `MirrorReadCursor` — all `Mirror`-prefixed so no sibling card collides. It adds
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

## Red-first

Observed, in three passes, all on `:5434` / role `hqtest` / db `hq_test_go_h3a` (never `:5433`).
Logs under `.night-crew/runs/2026-10-02-autonomous/logs/h3a/`.

**RED-1 — the true pre-change tree** (`RF-red-1-pre-change-tree.log`, `EXIT=1`): base `94ad3b3`
production files restored, only the card's new test files kept.

- `internal/toast` — `FAIL … [build failed]`: `undefined: IngestOrderDetails`, `undefined:
  parseOrderDetails`, `undefined: OrderRow`, `undefined: orderTimeZone`.
- `internal/marketing` — `FAIL … [build failed]`: `undefined: MirrorPollOnce`, `undefined:
  MirrorStart`, `undefined: mirrorCursorLog`.
- `internal/redemption` — behavioural: `racelost_dedupe_test.go:48: replayed emit must not ERROR
  …: insert race_lost_notifications: ERROR: duplicate key value violates unique constraint
  "race_lost_notifications_dedupe_uq" (SQLSTATE 23505)`. (That database still carried 0084 from an
  earlier green run — so this red is the *index present, clause absent* half of B-424, which is
  itself worth having on the record.)

**RED-2 — the same tree with the database rolled back to goose 83** (`RF-red-2-pre-change-tree-db-at-83.log`,
`EXIT=1`), i.e. what a fresh clone of the base produces:

- `racelost_dedupe_test.go:69: B-424: a replayed reconciliation created 3 notifications, want 1`.

**RED-3 — the card's Go code present, migration 0084 withheld** (`RF-red-3-no-migration-0084.log`,
`EXIT=1`), which is the behavioural red for the two rows RED-1 could only show as a compile error:

- `TestOrderDetailsUpsertIsIdempotent` — `truncate toast_orders: ERROR: relation "toast_orders"
  does not exist (SQLSTATE 42P01)`.
- `TestScanAttemptsMirrorKeysetResumes` — `truncate mirror: ERROR: relation
  "reconciliation_decisions" does not exist (SQLSTATE 42P01)`.
- `TestRaceLostNotificationDedupe` — `ERROR: there is no unique or exclusion constraint matching
  the ON CONFLICT specification (SQLSTATE 42P10)`, **and the pre-existing
  `TestF4_RaceLostReconciledEmitsNotification` fails with the same code** — the measured proof that
  the index and the `ON CONFLICT` clause must land together or the F4 path breaks.

**GREEN** (`RF-green.log`, `EXIT=0`): `go test -p 1 -count=1 -v ./internal/toast/
./internal/marketing/ ./internal/redemption/` → 86 PASS / 0 FAIL / 1 SKIP (the skip is H1's
`TestProjectionConfiguredUpsertsToSubstrate`, which gates on its own env var).

## `TestScanAttemptsMirrorKeysetResumes` — LIVE substrate or recorded fixture?

**LIVE. Not a fixture, not a stub.** The local `spike-supabase` stack, discovered the way spike 03
discovers it — `docker compose -p spike-supabase --project-directory <repo> -f
docker-compose.supabase.yml port rest 3000`, resolved to `http://127.0.0.1:52932` on run
`20261002` — with a `service_role` HS256 JWT minted in-test from the committed throwaway
`JWT_SECRET` (stdlib `crypto/hmac`, the same ten lines `.night-crew/qa/spike-supabase/mintjwt` is).
The green log line:

```
mirror_test.go:356: LIVE substrate http://127.0.0.1:52932 — poll1 fetched=1
  cursor=2099-01-01T12:00:00Z/11111111-…; poll2 fetched=2
  from=2099-01-01T12:00:00Z/11111111-… to=2099-01-01T12:01:00Z/33333333-…;
  A's mirrored_at unchanged
```

**How production code is proven to have done the read.** The test's only writes are into the
substrate's OWN `public.scan_attempts` over PostgREST (the DEVICE's job). Everything on the HQ side
— building the keyset predicate, the PostgREST GET, decoding, and the `scan_attempts_mirror` upsert
— is the shipped `marketing.MirrorPollOnce`. The witness that the RESUME happened in the QUERY and
was not merely hidden by the idempotent upsert is row A's `mirrored_at`: it is read after poll 1 and
asserted UNCHANGED after poll 2. And `poll2 fetched=2`, not 3 and not 1, is the two-sided assertion
— 3 would mean no resume, 1 would mean the `(scanned_at, id)` tie-break dropped row B, which shares
A's `scanned_at` to the microsecond.

The gate follows `internal/sync`'s asymmetric pattern: `HQ_MARKETING_MIRROR_LIVE` unset + substrate
down → a loud SKIP; **set + substrate down → FAIL**. It was armed (`=1`) for every run above. The
test deletes its three rows from the substrate on the way out through `context.Background()`, not
`t.Context()` — the first run proved `t.Context()` is already cancelled in `t.Cleanup`, which left
far-future rows on a SHARED substrate; the substrate was verified back to its single pre-existing
row afterwards.

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
- `scan_attempts_mirror.campaign_id` is mirrored as **NULL** tonight — see Decisions below.

## Decisions this card made that the slate left open

1. **`scan_attempts_mirror.code_id` is NULLABLE and `token_hash` is ADDED** — the two stated
   deviations from §4, argued at length in migration 0084's own header. Upstream dropped that NOT
   NULL in `supabase/migrations/20260906000200_…` for F2, so a verbatim NOT NULL mirror would have
   silently refused exactly the unverified-code overrides F4 treats as highest priority. Upstream's
   CHECK travels with the data as `scan_attempts_mirror_names_a_code`.
2. **`campaign_id` mirrors as NULL.** Upstream `scan_attempts` has no `campaign_id` and `code_id`
   carries no `references codes`, so PostgREST has nothing to embed through. Filling it needs a
   second `codes` pull, which is not this card's. The column exists per §4 so H3b need not migrate.
3. **B-424's index is `(code_token_hash, device_id, scanned_at)`** — migration 0077's real column
   names for the slate's conceptual `(code_id, losing_device, scanned_at)`. The migration DELETEs
   pre-existing duplicates (keeping the earliest per key) first, because a unique index cannot be
   created over the duplicates the missing dedupe produced.
4. **The OrderDetails leg rides `SyncDate`'s already-open client** rather than taking a second dial
   per date, and reports through `slog` at three severities (absent file → WARN and skip; parse/DB
   failure → ERROR; rows landed → INFO with the count) WITHOUT touching `SyncDate`'s `(bool, error)`
   or the B-146 health machinery. An OrderDetails problem must never mark the SALES transport dead.
5. **`business_date` = `Opened`'s calendar date in `America/Chicago`**, and `opened_at`/`closed_at`
   are parsed in that zone. The report prints no offset and `opened_at` is compared against a
   device's real `scanned_at` by H3b's ±30-minute suggestion, so an hour of drift here is a wrong
   suggestion there. Same derivation spike 02's proven upsert key used.
6. **`Closed` and `Order Source` are OPTIONAL; the seven money/identity columns are REQUIRED.**
   Spike 02 read `Order #`, `Order Id`, `Opened`, `Amount`, `Discount Amount`, `Total`, `Voided`
   and treated `Order Source` as optional; it never read `Closed`, which handoff §3 names on the
   export but nothing in this repo has proven. §4 has `closed_at` nullable, so an absent column is
   handled natively, logged at WARN, and recorded here as the one **stated gap** — not a park (the
   PARK note is explicit that a missing COLUMN falls back and only a missing SOURCE parks). A
   missing REQUIRED column is a hard parse failure, because landing orders with `amount_cents=0`
   would poison every money figure H3b computes.
7. **`voided` is a COLUMN, not a filter** — unlike `parseItemSelectionDetails`, which drops voided
   lines per D-06. Reconciliation has to be able to say "that order was voided" about a scan.
8. **Keyset page size 500, max 50 pages per tick, 5-minute period, 30-second per-page budget**, and
   the cursor derived from `max (scanned_at, id)` in the mirror itself rather than a checkpoint
   table (rationale in `mirror.go`'s header).
9. **A NAMED GAP, not a silent one:** a late-arriving offline attempt whose `scanned_at` predates
   the cursor is never mirrored by a strict `(scanned_at, id)` keyset. The card binds the mechanism
   and the done_when pins "resumes after the first's last (scanned_at, id)", so that is what ships —
   with the gap and its cheap fix (resume from `cursor.scanned_at − replay window`; the upsert
   already makes overlap free) written into `mirror.go` rather than left to be rediscovered. Worth a
   backlog line at triage.

## Gate outcomes (for the orchestrator, full logs under `logs/h3a/`)

| Gate | Result |
|---|---|
| G1 | `go build ./...` + `go vet ./...` from `backend/` — both exit 0. |
| G2 (Go) | `go test -p 1 -count=1 -v ./...` — **655 PASS / 0 FAIL / 3 SKIP** (658 results; base 651 → +7, exactly this card's seven). `internal/workflow` ran **39**, so `DB_TEST_URL` took effect. All three SKIPs are pre-existing opt-in gates. `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` were both **UNSET**. |
| G2 (Playwright) | ONE full suite under `flock /tmp/hq-full-suite.lock` — **exactly one summary block**: 26 failed / 6 skipped / 963 passed, EXIT=1. 22 of the 26 are baseline members; 2 baseline members went green; **4 fell outside the 24 and ALL FOUR pass confined.** Not diff-attributable — see the log's RED-SET DIFF block. |
| G4 | `node build-sw.js` twice — precache **48** both times, `sw.js` byte-unchanged (this card touches no precached asset). |
| RF | Three reds observed before the fix, then green — see Red-first above. |

### The four non-baseline reds, for the triage record

`inventory.spec.js:1469` (medium shows n/a), `:2700` (create new item via Items tab — **named verbatim
in `bugs.md`'s own cluster list**), `:2919` (create item without group shows alert),
`states-inventory-nav.spec.js:297` (hub painted before boot). All four green confined; `:2919` needed
true isolation rather than the four-test subset. Same Inventory Setup-tab / hub surface as the
undiagnosed 17-test cluster, and the same run saw two *baseline* members (`:2767`, `:2931`) flip
green — B-45/B-437's "a distribution of about four, not a fixed four", moving. Worth a line on the
cluster rather than a card.
