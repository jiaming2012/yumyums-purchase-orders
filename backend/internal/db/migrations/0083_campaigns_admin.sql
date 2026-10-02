-- +goose Up
BEGIN;

-- ===========================================================================
-- 0083 — campaign admin lands in HQ Postgres (decision 187, card H1).
--
-- Tablets need four columns offline; managers need twenty online. Putting the
-- twenty in Supabase would widen the replica and the RLS surface for no
-- offline benefit, so the manager-facing record lives HERE and the Supabase
-- `campaigns` row becomes a PROJECTION written by the same handler
-- (internal/marketing/projection.go) after the local transaction commits.
--
-- `campaigns_admin.id == Supabase campaigns.id` — the id IS the projection key,
-- which is why it has no DEFAULT here that the projector could disagree with:
-- the INSERT supplies gen_random_uuid() explicitly and the handler projects
-- that same value.
--
-- `qr_codes.short` is the public payload key (decision 189): 6 characters from
-- 23456789ABCDEFGHJKLMNPQRSTUVWXYZ — 0/1/I/O are excluded because a human
-- reads these off a printed sign. The CHECK is the database's half of that
-- promise; internal/marketing/codes.go's generator is the other half, and the
-- tests assert the alphabet independently of the constant.
--
-- `short` is also the FK target H5's 0085_subscribers.sql points
-- `subscribers.source_short` at (first-touch attribution), so this migration
-- must sort before that one.
--
-- DDL below is docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md
-- §4's H1 block, verbatim, with goose framing and the repo's BEGIN/COMMIT
-- convention added.
-- ===========================================================================

create table campaigns_admin (
  id              uuid primary key,
  slug            text not null unique,              -- "wing-wednesday", for UTM + payload preview
  name            text not null,
  offer_text      text not null,                     -- "$2 off any 6pc wings"
  face_value_cents integer not null check (face_value_cents >= 0),
  requires_online boolean not null,                  -- derived: face_value_cents >= marketing_settings threshold (#5)
  item_id         uuid null references menu_items(id),
  landing         text not null default 'signup' check (landing in ('signup','menu','offer','directions')),
  starts_at       timestamptz not null default now(),
  ends_at         timestamptz not null,
  status          text not null default 'live' check (status in ('scheduled','live','paused','ended')),
  projected_at    timestamptz null,                  -- last successful Supabase projection; NULL = "not on tablets yet"
  created_by      uuid not null references users(id),
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
);

create table qr_codes (
  id          uuid primary key default gen_random_uuid(),
  short       text not null unique check (short ~ '^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$'),
  campaign_id uuid not null references campaigns_admin(id),
  channel     text not null check (channel in ('truck_sign','flyer','table_tent','menu_board','instagram','google_ads','receipt','sms','other')),
  channel_label text null,                           -- required when channel='other'
  item_id     uuid null references menu_items(id),   -- defaults to the campaign's item
  placement   text null,
  variant     text null,
  landing     text null,                             -- NULL = inherit campaign.landing
  active      boolean not null default true,
  v           smallint not null default 1,
  created_by  uuid not null references users(id),
  created_at  timestamptz not null default now()
);

create table qr_scans (
  id          bigserial primary key,
  short       text not null references qr_codes(short),
  scanned_at  timestamptz not null default now(),
  ua_family   text null,                             -- coarse: ios / android / desktop / bot
  referrer    text null,
  ip_hash     text null,                             -- sha256(ip + daily salt); for 10-minute dedupe only
  subscriber_id uuid null                            -- set when the visit converts (H5 joins by source_short)
);
create index on qr_scans (short, scanned_at);

-- Not in §4, and load-bearing for the list route: every read of a campaign's
-- codes is by campaign_id.
create index qr_codes_campaign_idx on qr_codes (campaign_id);

COMMIT;

-- +goose Down
BEGIN;
-- Reverse dependency order: qr_scans -> qr_codes -> campaigns_admin. The
-- indexes go with their tables.
DROP TABLE IF EXISTS qr_scans;
DROP TABLE IF EXISTS qr_codes;
DROP TABLE IF EXISTS campaigns_admin;
COMMIT;
