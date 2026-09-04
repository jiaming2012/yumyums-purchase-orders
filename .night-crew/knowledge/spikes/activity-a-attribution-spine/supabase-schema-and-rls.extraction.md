# Extraction — supabase-schema-and-rls

Outcome: confirmed

Approach used: the §4 schema (with F2's `unverified_code`) applied to the
throwaway self-hosted substrate (supabase/postgres 15.8 + PostgREST v12 +
supabase/realtime v2.34 — the same engines a hosted project runs), with this
RLS shape: SELECT-for-authenticated on `campaigns`/`codes` (the replica is the
full bounded active set by design), INSERT-only on `scan_attempts` guarded by
`with check (device_id = jwt sub)` and no SELECT grant at all (push-only made
structural), and `public.codes` added to the `supabase_realtime` publication
via an idempotent guarded DO block. Second-subscriber proof by an
authenticated-role websocket client that distinguishes a join-ack from a real
subscription. The working fixture is
`.night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/sql/qr-schema.sql`
— a candidate seed for the card's in-repo `supabase/` migration, not an
adoption (NFR-6; the card's design.md decides).

Confirmed: the schema applies clean twice (fresh and warm); every structural
claim present by name (3 tables, F2 boolean, unique `token_hash`, the
`updated_at` checkpoint index, the join-key index, RLS ×3, 3 policies,
publication membership); RLS discriminates through the real API surface both
ways (device reads 200, anonymous 401, own-insert 201, spoofed-device 403,
device SELECT of attempts 403, server-side counts agree); and a redemption
UPDATE fired by one client arrived at a second Realtime subscriber inside the
20s window with publication membership and per-subscriber RLS both live — the
card's done_when leg, proven ahead of the card.

Learned: (nothing product-surprising). Harness-level only: a concatenated
Postgres boolean casts to `true`/`false`, not psql's bare `t`/`f` — cost one
false RED, signed at the batch review.

Plan change: two concrete inputs to downstream cards — (1) the scanner's push
handler (Activities B/C) MUST write `device_id` equal to the device token's
`sub` claim, or the insert policy refuses the row; put that in the replica
card's contract. (2) Hosted-project provisioning (Activity 0) owes the
dashboard-side equivalents of the publication + key wiring the fixture does
locally; the schema card itself needs no re-shaping — its migration can start
from the spike fixture as candidate.
