-- spike-d-teardown.sql — the exact inverse of spike-d-fixture.sql.
--
-- Dropping the tables also removes them from the `supabase_realtime`
-- publication, which is why publication membership is one of the two things
-- spike-d-realtime.sh snapshots before it creates anything and asserts
-- byte-identical afterwards. A restore that is not verified is a claim; B-148's
-- residual is precisely that spike B's recovery path was never re-rehearsed
-- after its fix, so this card verifies its own — on the RED path as well as the
-- green one.
--
-- Idempotent: safe to run when the fixture was never applied.
drop table if exists public.spike_d_responses;
drop table if exists public.spike_d_submissions;
drop table if exists public.spike_d_templates;

notify pgrst, 'reload schema';
