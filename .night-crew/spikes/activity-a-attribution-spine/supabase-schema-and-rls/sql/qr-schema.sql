-- qr-schema.sql — Activity A draft schema: §4 of docs/qr-offline-redemption-handoff.md
-- verbatim, plus the §19 F2 `unverified_code` column (already folded into §4 by the
-- addendum). This fixture IS the draft the card's in-repo `supabase/` migration will
-- start from — the spike applies it, so what the card inherits has already run.
--
-- ⚠ SPIKE FIXTURE. Applied ONLY to the throwaway spike-supabase substrate by the
-- scripts in this directory. Drop-and-recreate on purpose: every spike run starts
-- from the §4 text, not from whatever a previous run left behind. The real card's
-- migration will be additive, not drop-first — that difference is deliberate and
-- noted in the ledger.

drop table if exists public.scan_attempts cascade;
drop table if exists public.codes cascade;
drop table if exists public.campaigns cascade;

create table public.campaigns (
  id              uuid primary key,
  name            text not null,
  face_value      numeric not null,
  requires_online boolean not null default false,  -- §8 policy flag
  updated_at      timestamptz not null default now()
);

create table public.codes (
  id            uuid primary key,
  token_hash    text not null unique,   -- never store the raw token (§4)
  campaign_id   uuid not null references public.campaigns(id),
  expires_at    timestamptz not null,
  redeemed_at   timestamptz,
  redeemed_by   text,
  updated_at    timestamptz not null default now(),
  _deleted      boolean not null default false
);
-- the replication checkpoint key — hit on every pull tick (§4)
create index codes_updated_at_idx on public.codes (updated_at);

create table public.scan_attempts (
  id                uuid primary key,        -- generated on device
  code_id           uuid not null,
  device_id         text not null,
  scanned_at        timestamptz not null,
  status            text not null default 'pending',  -- pending | accepted | rejected
  reason            text,                             -- already_used | expired | not_found
  offline_override  boolean not null default false,   -- §13 permissioned override
  override_by       text,
  unverified_code   boolean not null default false,   -- §19 F2: override on a code not in the replica
  pos_order_number  text,
  pos_business_date date not null,
  redeemed_value    numeric,
  match_status      text not null default 'unmatched' -- unmatched | matched | orphan
);
create index scan_attempts_join_idx on public.scan_attempts (pos_business_date, pos_order_number);

-- ---------------------------------------------------------------------------
-- RLS (§7.2) — policies let each device see only what it needs:
--   codes/campaigns: SELECT for any authenticated device (the replica is the
--     full bounded active-code set by design, §5.3 — the bound is the pull
--     filter, RLS grants visibility);
--   scan_attempts: INSERT only, and only AS the device itself (device_id must
--     equal the JWT sub). Push-only means no SELECT grant at all — a device
--     never reads other devices' attempts; outcomes come back via redeem()'s
--     return value and the codes pull (§6).
-- ---------------------------------------------------------------------------
alter table public.campaigns     enable row level security;
alter table public.codes         enable row level security;
alter table public.scan_attempts enable row level security;

revoke all on public.campaigns, public.codes, public.scan_attempts
  from public, anon, authenticated;
grant select on public.campaigns to authenticated;
grant select on public.codes     to authenticated;
grant insert on public.scan_attempts to authenticated;

create policy campaigns_select_device on public.campaigns
  for select to authenticated using (true);
create policy codes_select_device on public.codes
  for select to authenticated using (true);
create policy scan_attempts_insert_own on public.scan_attempts
  for insert to authenticated
  with check (device_id = (current_setting('request.jwt.claims', true)::json ->> 'sub'));

-- ---------------------------------------------------------------------------
-- §7.1 — the publication toggle, "the usual reason people think Realtime is
-- broken". codes is the pull-replicated table the second subscriber watches.
-- Idempotent: adding a table already in the publication is an error, so guard.
-- ---------------------------------------------------------------------------
do $$
begin
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime'
                   and schemaname = 'public' and tablename = 'codes') then
    alter publication supabase_realtime add table public.codes;
  end if;
end $$;
