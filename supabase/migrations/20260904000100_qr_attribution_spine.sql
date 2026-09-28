-- 20260904000100_qr_attribution_spine.sql
-- Activity A, card `supabase-schema-and-rls` (night-crew run 20260904).
--
-- The attribution spine's arbiter schema: campaigns / codes / scan_attempts
-- (handoff §4, incl. F2's `unverified_code`), RLS per the spike-proven shape
-- (§7.2), `public.codes` in the `supabase_realtime` publication (§7.1), and
-- the #5 settings surface (`marketing_settings.requires_online_threshold_cents`,
-- operator-signed at the slate-20260904 sitting).
--
-- Lineage: design-adopted from the PROVEN spike fixture
-- `.night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/sql/qr-schema.sql`
-- (7/7 spikes green 2026-09-03) with one deliberate structural difference:
-- this migration is ADDITIVE AND IDEMPOTENT, never drop-first. The spike
-- fixture dropped and recreated on purpose (every spike run starts from the §4
-- text); a migration that drops tables would destroy production data on
-- re-apply. Idempotency here means: `if not exists` on tables/indexes,
-- `drop policy if exists` + `create policy` (Postgres has no CREATE POLICY IF
-- NOT EXISTS), guarded publication add, and conflict-ignoring seeds — so the
-- file applies clean on a bare substrate AND on top of its own output.
--
-- Apply order and target discipline: see supabase/README.md. Verified by
-- supabase/verify/01-structure.sh (every claim below is asserted BY NAME).

-- ---------------------------------------------------------------------------
-- campaigns — one row per marketing campaign. `requires_online` is the §8
-- policy flag; it is DERIVED AT CAMPAIGN CREATION from face value vs
-- marketing_settings.requires_online_threshold_cents (#5, below) and stored
-- here so every offline device sees a stable policy, not a live computation.
-- ---------------------------------------------------------------------------
create table if not exists public.campaigns (
  id              uuid primary key,
  name            text not null,
  face_value      numeric not null,
  requires_online boolean not null default false,  -- §8 policy flag, derived per #5
  updated_at      timestamptz not null default now()
);

-- ---------------------------------------------------------------------------
-- codes — one row per issued QR code. `token_hash` NEVER stores the raw
-- token (§4): the device hashes the scanned identity token with WebCrypto
-- before any lookup, so a dumped replica never yields live codes. There is
-- deliberately no `token` column, and the verify harness asserts that
-- negatively by name.
-- ---------------------------------------------------------------------------
create table if not exists public.codes (
  id            uuid primary key,
  token_hash    text not null unique,   -- never the raw token (§4)
  campaign_id   uuid not null references public.campaigns(id),
  expires_at    timestamptz not null,
  redeemed_at   timestamptz,
  redeemed_by   text,
  updated_at    timestamptz not null default now(),
  _deleted      boolean not null default false    -- RxDB soft-delete contract
);
-- the replication checkpoint key — hit on every pull tick (§4)
create index if not exists codes_updated_at_idx on public.codes (updated_at);

-- ---------------------------------------------------------------------------
-- scan_attempts — device-owned, push-only (§4's key structural decision).
-- `unverified_code` is F2 (§19): an offline override on a code not in the
-- device's replica. The (pos_business_date, pos_order_number) index is the
-- Toast reconciliation join key (§13).
-- ---------------------------------------------------------------------------
create table if not exists public.scan_attempts (
  id                uuid primary key,        -- generated on device
  code_id           uuid not null,
  device_id         text not null,
  scanned_at        timestamptz not null,
  status            text not null default 'pending',  -- pending | accepted | rejected
  reason            text,                             -- already_used | expired | not_found
  offline_override  boolean not null default false,   -- §13 permissioned override
  override_by       text,
  unverified_code   boolean not null default false,   -- §19 F2
  pos_order_number  text,
  pos_business_date date not null,
  redeemed_value    numeric,
  match_status      text not null default 'unmatched' -- unmatched | matched | orphan
);
create index if not exists scan_attempts_join_idx
  on public.scan_attempts (pos_business_date, pos_order_number);

-- ---------------------------------------------------------------------------
-- marketing_settings — the #5 resolution (operator-signed, slate-20260904):
-- a structurally SINGLE-ROW settings table. Campaign creation consults
-- `requires_online_threshold_cents` to derive a new campaign's
-- `requires_online` (face value in cents >= threshold → true). Changing the
-- threshold is an UPDATE, never a migration — and the conflict-ignoring seed
-- below is what keeps a re-applied migration from clobbering an
-- operator-changed value (the verify harness proves that leg).
--
-- Server-side only: RLS is enabled with NO policies and NO client grants.
-- Devices never read this table — they read the derived `requires_online`
-- on campaigns. Only service_role (BYPASSRLS) / admin tooling touches it.
-- ---------------------------------------------------------------------------
create table if not exists public.marketing_settings (
  id                              smallint    primary key default 1,
  requires_online_threshold_cents integer     not null default 2000,  -- $20.00
  updated_at                      timestamptz not null default now(),
  constraint marketing_settings_singleton        check (id = 1),
  constraint marketing_settings_threshold_nonneg check (requires_online_threshold_cents >= 0)
);
insert into public.marketing_settings (id, requires_online_threshold_cents)
  values (1, 2000)
  on conflict (id) do nothing;   -- never clobber an operator-set value

-- ---------------------------------------------------------------------------
-- RLS (§7.2) — the spike-proven shape:
--   campaigns/codes: SELECT for any authenticated device (the replica is the
--     full bounded active-code set by design, §5.3 — the bound is the pull
--     filter, RLS grants visibility);
--   scan_attempts: INSERT only, and only AS the device itself (device_id must
--     equal the JWT sub). Push-only means no SELECT grant at all — a device
--     never reads other devices' attempts; outcomes come back via redeem()'s
--     return value (Card 2) and the codes pull (§6).
--   marketing_settings: RLS on, no policies, no client grants — server-only.
--
-- Note (spike sharp-edge 8): predicates read
-- current_setting('request.jwt.claims')::json ->> 'sub' directly. auth.uid()
-- does not work here — it reads the legacy singular GUC and casts to uuid,
-- and HQ device ids are not uuids.
-- ---------------------------------------------------------------------------
alter table public.campaigns          enable row level security;
alter table public.codes              enable row level security;
alter table public.scan_attempts      enable row level security;
alter table public.marketing_settings enable row level security;

revoke all on public.campaigns, public.codes, public.scan_attempts, public.marketing_settings
  from public, anon, authenticated;
grant select on public.campaigns to authenticated;
grant select on public.codes     to authenticated;
grant insert on public.scan_attempts to authenticated;
-- marketing_settings: no grants, deliberately.

drop policy if exists campaigns_select_device on public.campaigns;
create policy campaigns_select_device on public.campaigns
  for select to authenticated using (true);

drop policy if exists codes_select_device on public.codes;
create policy codes_select_device on public.codes
  for select to authenticated using (true);

drop policy if exists scan_attempts_insert_own on public.scan_attempts;
create policy scan_attempts_insert_own on public.scan_attempts
  for insert to authenticated
  with check (device_id = (current_setting('request.jwt.claims', true)::json ->> 'sub'));

-- ---------------------------------------------------------------------------
-- §7.1 — the publication toggle, "the usual reason people think Realtime is
-- broken". codes is the pull-replicated table the second subscriber watches.
-- Guarded: adding a table already in the publication is an error.
-- ---------------------------------------------------------------------------
do $$
begin
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime'
                   and schemaname = 'public' and tablename = 'codes') then
    alter publication supabase_realtime add table public.codes;
  end if;
end $$;

-- Nudge PostgREST's schema cache so freshly-created objects are servable
-- immediately (the Supabase image also ships DDL event triggers for this;
-- the explicit notify removes a timing flake class, and is a no-op when
-- nothing listens).
notify pgrst, 'reload schema';
