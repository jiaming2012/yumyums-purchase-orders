-- spike-c-relay-trigger.sql — THE MECHANISM UNDER TEST, half 1 of 2.
--
-- Night-crew card C `spike-c-round-trip` (Spike C).
--
-- ⚠ LOCAL SPIKE ONLY. Applied by spike-c-roundtrip.sh into the throwaway
--   `spike-c-hq` scratch Postgres, which is created and destroyed inside one
--   script run. NEVER :5433 (that cluster is PRODUCTION), never :5434, never
--   HQ's real database. This file is NOT a migration and deliberately does not
--   live in backend/internal/db/migrations — see "why a trigger" below.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- WHAT THIS IS
--
-- Spike C must carry ONE row from HQ's Postgres into the Supabase substrate so
-- an RxDB client can read it. Four candidate mechanisms were inventoried
-- (see spike-c-roundtrip.sh's header for the full disposition). This file plus
-- backend/internal/sync/spikec_relay.go is the one that got proven:
--
--     LISTEN/NOTIFY relay
--     HQ Postgres --NOTIFY--> Go relay --PostgREST(service identity)--> substrate
--
-- ═══════════════════════════════════════════════════════════════════════════
-- 🛑 WHY A TRIGGER, AND NOT A LINE OF Go IN saveResponse()
--
-- Decision 126's shape is that `/saveResponse` and `/submit` KEEP OWNING WRITES
-- and only reads move to RxDB. A mechanism that edits the write handler puts
-- the substrate on `/saveResponse`'s critical path: a substrate outage would
-- then red a crew member's checkbox. This trigger changes NOTHING about the
-- write path — `backend/internal/workflow/repository.go:saveResponse` is not
-- touched by this card, and its SQL is byte-identical with the trigger present
-- or absent. The projection is a CONSEQUENCE of the write, observed from
-- outside it, which is the only shape that keeps writes and sync independent.
--
-- Note also what is reused rather than invented: HQ ALREADY runs a
-- LISTEN/NOTIFY fan-out. `backend/internal/sync/ops.go:204` fires
-- `pg_notify('ops_channel', ...)` and `backend/internal/sync/listener.go` holds
-- a `pgxlisten.Listener` against it. Every dependency this mechanism needs is
-- already a DIRECT dependency of backend/go.mod. That is a large part of why
-- this candidate was chosen over the other three.
--
-- 🛑 WHY THE PAYLOAD IS IDS ONLY
-- pg_notify's payload is capped at 8000 bytes and a `submission_responses.value`
-- is unbounded JSONB (a correction photo URL, a fail note). A payload that
-- carries the value would work on every fixture anyone writes and then throw
-- `payload string too long` on a real row — inside the writer's transaction.
-- The relay re-reads the row by id, exactly as listener.go's handler re-reads
-- the op with GetOpByID rather than trusting the notification body.
-- ═══════════════════════════════════════════════════════════════════════════

create or replace function public.spike_c_relay_notify()
returns trigger language plpgsql as $$
declare
  payload text;
begin
  payload := json_build_object(
    'response_id',   new.id,
    'field_id',      new.field_id,
    'answered_by',   new.answered_by,
    'submission_id', new.submission_id,
    'op',            tg_op
  )::text;
  perform pg_notify('spike_c_relay', payload);
  return new;
end;
$$;

drop trigger if exists spike_c_relay_notify on public.submission_responses;

-- AFTER, not BEFORE: a notification for a row whose INSERT is later rolled back
-- would send the relay chasing a row that does not exist. AFTER inside the same
-- transaction still queues the NOTIFY transactionally — Postgres delivers it at
-- COMMIT and discards it on ROLLBACK — so the relay is only ever told about
-- rows that really landed.
create trigger spike_c_relay_notify
  after insert or update on public.submission_responses
  for each row execute function public.spike_c_relay_notify();
