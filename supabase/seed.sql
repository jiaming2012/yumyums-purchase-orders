-- seed.sql — TEST FIXTURES ONLY.
--
-- ⚠ LOCAL / TEST SUBSTRATES ONLY. NEVER apply this file to a production
-- Supabase project. It seeds two TEST campaigns and five TEST codes that
-- exist so Card 2 (`redeem-rpc-race-proof`) and the verify harnesses
-- (supabase/verify/) have named fixtures to drive. The PRODUCTION
-- welcome-offer campaign is Activity E's to create (#8) — it does not and
-- must not appear here.
--
-- Idempotent by `on conflict (id) do nothing`: re-applying never duplicates
-- rows and never resets state a test run has legitimately moved (e.g. a race
-- round that redeemed a code).
--
-- Fixture contract (fixed UUIDs — Card 2 and the verify legs reference these
-- BY VALUE; do not renumber):
--
--   campaigns
--     a0000000-0000-4000-8000-000000000001  TEST low  ($2.00  < $20 threshold → requires_online=false)
--     a0000000-0000-4000-8000-000000000002  TEST high ($40.00 >= $20 threshold → requires_online=true)
--
--   codes
--     c0000000-…-0001  LOW  active, unredeemed   (RLS read leg + Realtime touch target)
--     c0000000-…-0002  LOW  active, unredeemed   (Card 2: race / happy-path target)
--     c0000000-…-0003  LOW  EXPIRED, unredeemed  (Card 2: `expired` arm)
--     c0000000-…-0004  LOW  active, REDEEMED     (Card 2: `already_used` arm)
--     c0000000-…-0005  HIGH active, unredeemed   (the requires_online=true campaign's code)
--
-- The `requires_online` values are written literally here, but each one is the
-- value campaign creation WOULD derive from marketing_settings (#5): face
-- value in cents vs requires_online_threshold_cents (seeded 2000).
--
-- token_hash values are sha256 digests of the fixture LABELS
-- ("card1-test-code-fixture-1" … "-5") — hash-shaped on purpose, and there is
-- no raw token anywhere: none was ever minted for these fixtures.

insert into public.campaigns (id, name, face_value, requires_online) values
  ('a0000000-0000-4000-8000-000000000001', 'TEST — $2 wing discount (offline-eligible)', 2.00,  false),
  ('a0000000-0000-4000-8000-000000000002', 'TEST — $40 catering credit (online-only)',   40.00, true)
on conflict (id) do nothing;

insert into public.codes (id, token_hash, campaign_id, expires_at, redeemed_at, redeemed_by) values
  ('c0000000-0000-4000-8000-000000000001',
   'c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680',
   'a0000000-0000-4000-8000-000000000001', '2028-01-01T00:00:00Z', null, null),
  ('c0000000-0000-4000-8000-000000000002',
   '5ee84bb426f2ef8d34d90ddec7e443e0b7755dfccbc44ef2bc31ee9bad2aedc1',
   'a0000000-0000-4000-8000-000000000001', '2028-01-01T00:00:00Z', null, null),
  ('c0000000-0000-4000-8000-000000000003',
   'f9e197925d9d0901b3fdc7197e91972aa7a225744e8434104a7d4fa5a7c3ddd9',
   'a0000000-0000-4000-8000-000000000001', '2026-01-01T00:00:00Z', null, null),
  ('c0000000-0000-4000-8000-000000000004',
   'a939afc9a3040327594b0f3c1d3db90a317f93188c114bac807ffdc64eb09097',
   'a0000000-0000-4000-8000-000000000001', '2028-01-01T00:00:00Z',
   '2026-09-01T12:00:00Z', 'test-device-seed'),
  ('c0000000-0000-4000-8000-000000000005',
   '60f4743622b18f559fb115e1c3329fad70e0168a3b05328361792477615db7cf',
   'a0000000-0000-4000-8000-000000000002', '2028-01-01T00:00:00Z', null, null)
on conflict (id) do nothing;
