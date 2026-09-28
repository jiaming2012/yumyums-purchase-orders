-- 20260906000200_scan_attempts_unverified_landing.sql
-- Activity B, card `requires-online-replication` (night-crew run 20260906).
--
-- The F-2 DISTINCT LANDING PATH (handoff §19 F2, §9; spike
-- requires-online-replication leg c2). An unknown-code offline override has
-- no code row to name — the device queues `code_id = <64-hex token_hash>`
-- (submit-flow.js's recorded engineering call), and the shipped push handler's
-- redeem-first path drew a deterministic HTTP 400 (`22P02` on `p_code uuid`)
-- that HEAD-OF-LINE POISONED the queue: 12 redeem attempts, 0 landing
-- requests, every later redemption on the device stranded (spike-measured).
-- The client-side guard (push-replication.js) diverts unverified attempts
-- BEFORE redeem(); this migration gives them somewhere honest to land:
--
--   * `code_id` drops NOT NULL — an unverified attempt names no code, and
--     that is the truth. The column stays `uuid`: a 64-hex hash can never
--     masquerade as a code id.
--   * `token_hash` — what the attempt actually has; the §19 F2 arbitration
--     clause ("when sync arbitrates…") resolves it server-side later.
--   * the check constraint — every attempt either names a real code or is an
--     audit-flagged unverified override carrying its hash; a row with neither
--     is structurally impossible.
--
-- Status taxonomy (§9/§19 re-read, recorded in the run's merge intent): the
-- row lands `status='accepted'` + `offline_override=true` +
-- `unverified_code=true` — NO new terminal status. §9 names offline overrides
-- as "the only accepted attempts that can still turn into a real
-- double-redeem" and orders them reconciled FIRST; F4's
-- status-reflects-the-loss is the server-side reconciliation flow (Activity
-- D's arbitration surface), after landing, not at it. Skip-until-arbitration
-- was run and REJECTED (spike leg c1): it drains the queue but strands the
-- audit row on-device, falsifying decision 166's own reasoning.
--
-- Idempotent per the Activity A convention: DROP NOT NULL is a no-op when
-- already nullable; ADD COLUMN IF NOT EXISTS; drop+add constraint. Applies
-- clean on a bare substrate AND on top of its own output. Activity A's
-- migrations are untouched (new numbered file only).

alter table public.scan_attempts alter column code_id drop not null;
alter table public.scan_attempts add column if not exists token_hash text;

alter table public.scan_attempts drop constraint if exists scan_attempts_names_a_code;
alter table public.scan_attempts add constraint scan_attempts_names_a_code
  check (code_id is not null or (unverified_code and token_hash is not null));

-- Nudge PostgREST's schema cache (Card 1's convention; a no-op when nothing
-- listens).
notify pgrst, 'reload schema';
