-- 20260906000100_campaigns_replication.sql
-- Activity B, card `requires-online-replication` (night-crew run 20260906).
--
-- Makes `public.campaigns` VISIBLE TO REPLICATION, closing spike build-facts
-- 2 and 3 (ledger requires-online-replication.md): the campaigns pull replica
-- (marketing/sync/replicas.js startCampaignsReplica) carries the §8
-- `requires_online` flag to devices, and a FLIP of it must reach a replica
-- that is already checkpointed — the operator downgrading a campaign while
-- its codes sit still is the decisive case (the codes-embed alternative is
-- CLOSED on exactly that measurement).
--
-- Two halves, both required for coherence — the mechanics decision recorded
-- in the run's merge intent (publication + trigger over a codes-channel
-- RESYNC fan-out, because a campaign-only write emits no codes frame):
--
--   1. TOUCH TRIGGER — the pull checkpoint is keyset {updated_at, id}
--      (§4/GAP-1), so a write that does not advance `updated_at` is invisible
--      to EVERY checkpointed replica (spike-measured: both mechanisms blind
--      to an unstamped flip). The trigger stamps every UPDATE, making a plain
--      `update campaigns set requires_online = …` deliverable — the same
--      rule decision 163 put on redeem()'s winning UPDATE for codes.
--      (GAP-2 — the future provisioning surface's write path — stays carried
--      as recorded; its validation run belongs to that card. This trigger is
--      owed item 2's machinery, authorized by the card text.)
--
--   2. PUBLICATION MEMBERSHIP (§7.1) — a pull replica does not poll
--      (spike-measured: 3s after a stamped write with no RESYNC, the replica
--      still read the old value), and a table outside `supabase_realtime`
--      emits no Realtime frame to nudge one. Guarded exactly like Card 1's
--      codes block: adding a table already in the publication is an error.
--
-- Idempotent per the Activity A convention: `create or replace` +
-- `drop trigger if exists` + guarded publication add — applies clean on a
-- bare substrate AND on top of its own output. Activity A's migrations are
-- untouched (new numbered file only). Apply order: supabase/README.md.

create or replace function public.campaigns_touch_updated_at()
returns trigger
language plpgsql
set search_path = ''
as $$
begin
  new.updated_at = now();   -- the replication checkpoint key must move (§4)
  return new;
end $$;

drop trigger if exists campaigns_touch_updated_at on public.campaigns;
create trigger campaigns_touch_updated_at
  before update on public.campaigns
  for each row execute function public.campaigns_touch_updated_at();

do $$
begin
  if not exists (select 1 from pg_publication_tables
                 where pubname = 'supabase_realtime'
                   and schemaname = 'public' and tablename = 'campaigns') then
    alter publication supabase_realtime add table public.campaigns;
  end if;
end $$;

-- Nudge PostgREST's schema cache (Card 1's convention; a no-op when nothing
-- listens).
notify pgrst, 'reload schema';
