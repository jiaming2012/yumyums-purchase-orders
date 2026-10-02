-- +goose Up
BEGIN;

-- ===========================================================================
-- 0084 — the Toast join + the attempts mirror + the decision log (card H3a,
-- `toast-orders-and-mirror`, run 20261002). Decision 188.
--
-- ALL THREE tables land here on purpose: H3b (`reconciliation-and-stats-engine`)
-- adds NO migration. `reconciliation_decisions.attempt_id` references
-- `scan_attempts_mirror(id)`, so splitting the two across migrations would make
-- 0085 depend on a table 0084 might not have; and a queue card that cannot
-- write its own decisions is a card that cannot ship.
--
-- DDL below is docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md
-- §4's H3 block, with goose framing, the repo's BEGIN/COMMIT convention, and
-- the TWO stated deviations recorded at scan_attempts_mirror below.
-- ===========================================================================

-- ---------------------------------------------------------------------------
-- toast_orders — one row per Toast order, parsed from OrderDetails.csv on the
-- SFTP export HQ already syncs (internal/toast/orderdetails.go). Decision 188;
-- the file's presence on 20260928/29/30 was confirmed by read-only listing
-- 2026-10-01, so the fixture-loader fallback was not needed.
--
-- `(business_date, order_number)` is the PRIMARY KEY because it is the UPSERT
-- key: the same daily report WILL arrive twice (§13 — the worker re-pulls a
-- 7-day window every tick). Spike 02 enumerated the duplicate set over 77 real
-- orders and found it empty, so this is a measured key, not an assumed one.
-- `order_number` is TEXT: the real sample carries 1-, 2- and 4-digit numbers,
-- digits only, variable length (handoff #2 answered) — no numeric coercion, no
-- zero-padding, matched exactly.
--
-- Money is integer CENTS everywhere, parsed from the report's dollar strings.
-- ---------------------------------------------------------------------------
create table toast_orders (
  business_date   date not null,
  order_number    text not null,                     -- Toast "Order #", stored as text (digits only, 1-4 long in the real sample)
  order_id        text not null,
  opened_at       timestamptz not null,
  closed_at       timestamptz null,
  amount_cents    integer not null,                  -- "Amount" (pre-discount)
  discount_cents  integer not null default 0,        -- "Discount Amount"
  total_cents     integer not null,
  voided          boolean not null default false,
  order_source    text null,
  ingested_at     timestamptz not null default now(),
  primary key (business_date, order_number)          -- upsert key: the same report WILL arrive twice (§13)
);

-- Not in §4, load-bearing for H3b: the ±30-minute nearest-order suggestion for
-- an unmatched attempt scans a business date by open time.
create index toast_orders_opened_idx on toast_orders (business_date, opened_at);

-- ---------------------------------------------------------------------------
-- scan_attempts_mirror — HQ's server-side copy of Supabase public.scan_attempts,
-- pulled by internal/marketing/mirror.go every 5 minutes with the service key,
-- keyset on (scanned_at, id). Devices have INSERT and no SELECT on the upstream
-- table (proven at Activity A, re-proven by spike 03: service_role 200, device
-- 403), so the server is the only reader and this is the only place the counter's
-- outcome becomes visible inside HQ. That visibility IS B-424's F4
-- scan_attempts-status acceptance bullet — see the card's merge-intent.
--
-- 🛑 TWO DELIBERATE DEVIATIONS FROM §4, both forced by the upstream shape:
--
--   1. `code_id` is NULLABLE here; §4 wrote `not null`. Upstream dropped that
--      NOT NULL in supabase/migrations/20260906000200_scan_attempts_unverified_landing.sql
--      for F2: an offline override on a code the device's replica could not
--      verify has NO code_id, only a token_hash. A NOT NULL mirror would reject
--      exactly the attempts F4 cares about most, i.e. the mirror would be
--      silently lossy on its highest-priority rows. The upstream CHECK
--      (`code_id is not null or (unverified_code and offline_override and
--      token_hash is not null)`) is mirrored below so the invariant travels with
--      the data instead of being lost in the copy.
--   2. `token_hash` is ADDED (§4 has no such column). Without it an
--      unverified_code attempt in the mirror names no code at all, and it is
--      also the key B-424's race_lost_notifications dedupe is written on. An
--      addition widens nothing a device can see — this table is HQ-only.
--
-- `campaign_id` stays in the shape per §4 but is mirrored as NULL: upstream
-- scan_attempts has no campaign_id column and no FK PostgREST could embed
-- through (code_id has no `references codes`), so there is nothing to read it
-- from. H3b resolves campaign through code_id when it needs it; filling this
-- column would need a second `codes` pull, which is not this card's.
-- ---------------------------------------------------------------------------
create table scan_attempts_mirror (
  id uuid primary key, code_id uuid null, campaign_id uuid null, device_id text not null,
  scanned_at timestamptz not null, status text not null, reason text null,
  offline_override boolean not null, override_by text null, unverified_code boolean not null,
  policy_unresolved boolean not null default false,
  token_hash text null,                              -- deviation 2: F2's only code identifier
  pos_order_number text null, pos_business_date date not null, redeemed_value numeric null,
  match_status text not null,                        -- as stored upstream: unmatched | matched | orphan
  mirrored_at timestamptz not null default now(),
  constraint scan_attempts_mirror_names_a_code
    check (code_id is not null or (unverified_code and offline_override and token_hash is not null))
);

-- The poller's keyset cursor is `max (scanned_at, id)` already mirrored — there
-- is no separate checkpoint table, so this index IS the cursor read.
create index scan_attempts_mirror_keyset_idx on scan_attempts_mirror (scanned_at, id);
-- §13's Toast join key, same shape as upstream's scan_attempts_join_idx.
create index scan_attempts_mirror_join_idx on scan_attempts_mirror (pos_business_date, pos_order_number);

-- ---------------------------------------------------------------------------
-- reconciliation_decisions — §4 verbatim. WRITTEN BY H3b, not by this card;
-- it lands here so H3b adds no migration (see the header).
-- ---------------------------------------------------------------------------
create table reconciliation_decisions (
  id          bigserial primary key,
  attempt_id  uuid not null references scan_attempts_mirror(id),
  decision    text not null check (decision in ('matched','declined','reopened','verified','rejected')),
  order_number text null,                            -- for 'matched'
  reason      text null check (reason in ('no_such_order','customer_left','comped','duplicate_scan','no_discount_applied','other')),
  note        text null,                             -- required when reason='other'
  decided_by  uuid not null references users(id),
  decided_at  timestamptz not null default now()
);
create index on reconciliation_decisions (attempt_id, decided_at desc);

-- ---------------------------------------------------------------------------
-- B-424 — race_lost_notifications had no dedupe, so a REPLAYED reconciliation
-- (the arbiter re-processing the same synced offline_override attempt) pinged
-- the Shift Manager twice for one loss. The dedupe key is the F4 identity of
-- the event: WHICH code (the §4 token hash — never a raw token), WHICH device
-- lost the race, and WHEN the code was accepted at the counter. The slate names
-- it `(code_id, losing_device, scanned_at)`; migration 0077's actual columns are
-- `code_token_hash`, `device_id`, `scanned_at` — same three facts, the shipped
-- spelling.
--
-- `staff` is deliberately NOT in the key: the same loss re-synced by a second
-- manager is the same loss.
--
-- The DELETE first is not optional. A unique index cannot be created over
-- existing duplicates, and prod may hold some precisely because the dedupe was
-- missing — that is the bug. Keep the earliest row per key (the one the Shift
-- Manager has most likely already seen) and drop the later replays.
-- internal/redemption/store.go's INSERT gains the matching
-- ON CONFLICT ... DO NOTHING in the same change set; the index and the clause
-- are one change in two files.
-- ---------------------------------------------------------------------------
DELETE FROM race_lost_notifications r
 USING race_lost_notifications keep
 WHERE r.code_token_hash = keep.code_token_hash
   AND r.device_id       = keep.device_id
   AND r.scanned_at      = keep.scanned_at
   AND ( r.created_at > keep.created_at
      OR (r.created_at = keep.created_at AND r.id > keep.id) );

CREATE UNIQUE INDEX race_lost_notifications_dedupe_uq
  ON race_lost_notifications (code_token_hash, device_id, scanned_at);

COMMENT ON INDEX race_lost_notifications_dedupe_uq IS
  'B-424: one notification per (code, losing device, accept time). PGRaceLostStore.Emit '
  'inserts ON CONFLICT DO NOTHING against this index, so a replayed reconciliation '
  'pings the Shift Manager once. Dropping this index breaks that INSERT.';

COMMENT ON TABLE toast_orders IS
  'Toast orders parsed from OrderDetails.csv on the SFTP export (decision 188, card H3a). '
  'Upsert key (business_date, order_number) — the same daily report arrives on every tick.';
COMMENT ON TABLE scan_attempts_mirror IS
  'HQ-side mirror of Supabase public.scan_attempts, pulled by internal/marketing/mirror.go '
  '(service key, keyset on scanned_at,id). The only place a counter attempt''s status, reason '
  'and override provenance are visible inside HQ (B-424 F4 bullet).';
COMMENT ON TABLE reconciliation_decisions IS
  'Append-only log of a human''s reconciliation call on a mirrored attempt. Written by H3b.';

COMMIT;

-- +goose Down
BEGIN;
-- Reverse dependency order: reconciliation_decisions -> scan_attempts_mirror.
-- toast_orders is independent. The indexes go with their tables; B-424's index
-- is on a table 0077 owns, so it is dropped explicitly — and the matching
-- ON CONFLICT in PGRaceLostStore.Emit stops deduping once it is gone, which is
-- the honest reverse of this migration rather than a half-rollback.
DROP TABLE IF EXISTS reconciliation_decisions;
DROP TABLE IF EXISTS scan_attempts_mirror;
DROP TABLE IF EXISTS toast_orders;
DROP INDEX IF EXISTS race_lost_notifications_dedupe_uq;
COMMIT;
