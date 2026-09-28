# Spikes — supabase-schema-and-rls

Activity: Activity A — The attribution spine (the Supabase arbiter)

> This target repo has no `usm/roadmap.txt` story map (the layout the
> `night-crew spikes gate/run` verbs read), so those verbs cannot drive here —
> the established convention from last cycle's `activity-5-dev-complete` ledger.
> The artifacts are authored to the skill's paths anyway: this ledger, and
> runnable scripts that ARE the verdict (B-345), at
> `.night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/`.
>
> **Substrate note (the engineering call this sitting makes, stated):** no
> hosted Supabase project exists yet — `external-accounts-provision`
> (Activity 0, attended) creates it. The premises here are falsifiable TODAY
> against the committed throwaway substrate from last cycle
> (`docker-compose.supabase.yml`: `supabase/postgres:15.8.1.060` + PostgREST
> v12.2.12 + `supabase/realtime:v2.34.47` — the same engines a hosted project
> runs), so the spike does not wait on the account. What the substrate does NOT
> prove: hosted-project operational facts (dashboard-managed publication state,
> hosted connection pooling, project-level key rotation). Those are provisioning
> concerns, not schema/RLS/Realtime premises, and land with Activity 0.

## The goal, and which legs need a spike

The card (roadmap Activity A): the `campaigns` / `codes` / `scan_attempts`
schema (§4 + F2's `unverified_code`), `token_hash` never storing the raw token,
the `updated_at` checkpoint index, RLS per §7.2, and the tables in the
`supabase_realtime` publication (§7.1). done_when: "schema applies clean against
a fresh Supabase project and a row inserted on one client appears on a second
subscriber."

Three falsifiable premises, one spike each, in run order:

## Spike: schema-applies

- proves: the §4 schema (with F2's `unverified_code`) applies clean on
  Supabase-shaped Postgres — twice, fresh and warm — and every structural claim
  is present BY NAME: 3 tables, the boolean F2 column, the unique `token_hash`,
  the `codes(updated_at)` checkpoint index, the `(pos_business_date,
  pos_order_number)` join-key index, RLS enabled on all three, the three
  policies by name, and `public.codes` in the `supabase_realtime` publication.
  Enumerated, not sampled (B-216): each assertion prints the enumerating
  query's full result. The SQL fixture this spike applies IS the draft the
  card's in-repo `supabase/` migration starts from.
- plan: bring the substrate up via the proven `env-up.sh` (reconcile mode —
  never `--fresh` from a spike, so it cannot eat another session's stack),
  apply `sql/qr-schema.sql` twice via `docker exec psql -v ON_ERROR_STOP=1`,
  then run the named-object assertions against the catalogs.
- script: .night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/01-schema-applies.sh

## Spike: rls-per-device

- proves: the §7.2 RLS design discriminates through the REAL API surface
  (PostgREST + role-claim JWTs), positively and negatively: a device JWT reads
  the code replica; an anonymous request cannot; a device inserts a
  `scan_attempts` row AS ITSELF; the same device inserting under ANOTHER
  device_id is refused (the `with check` on the JWT `sub`); and the device role
  cannot SELECT `scan_attempts` at all (push-only, the §4 structural decision).
  The negative legs are what make this falsifiable — a let-everything-through
  policy passes the positives and fails the negatives.
- plan: substrate up, schema applied, one campaign + one code seeded
  server-side; mint an HS256 `authenticated`-role token with the existing
  `mintjwt` (secret read from the compose file, never re-typed); drive the six
  legs through PostgREST with curl; verify the accepted insert landed and the
  refused one did not, server-side.
- script: .night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/02-rls-per-device.sh

## Spike: realtime-second-subscriber

- proves: the card's done_when leg — a row changed by one client appears on a
  second subscriber — through the real Realtime websocket, with §7.1's
  publication membership and §7.2's RLS-per-subscriber both in play: an
  `authenticated`-role websocket client (the existing `rtprobe`, which
  distinguishes a join-ack from a real subscription and refuses to assert over
  a late SYS-ERR) subscribes to `public.codes`; a server-side redemption
  UPDATE (the §5.3 propagation event) then arrives as a `postgres_changes`
  frame within a bounded window.
- plan: substrate up, schema + seed; mint the subscriber token
  (role=authenticated, sub=device-b); build and background `rtprobe` bound to
  `public.codes` unfiltered; await `RTP READY` with JOIN-OK and no SYS-ERR;
  fire the UPDATE as "client one" via psql; wait out the window; assert the
  `RTP EVENT … type=UPDATE table=public.codes` line.
- script: .night-crew/spikes/activity-a-attribution-spine/supabase-schema-and-rls/03-realtime-second-subscriber.sh

## Verdict (run 2026-09-03, hand-run per the no-story-map convention)

- **schema-applies: passed** — exit 0 (second execution; the first was RED on a
  script assertion defect, see Corrections — the schema itself was correct on
  the first apply). Applied twice, clean both times. Enumerated by name:
  3 tables; `unverified_code:boolean`; `codes_token_hash_key` (unique) +
  `codes_updated_at_idx`; `scan_attempts_join_idx`; RLS enabled ×3; the three
  policies; `public.codes` present in `supabase_realtime`.
- **rls-per-device: passed** — exit 0, first run. Through PostgREST:
  device read 200 with the seeded row; anonymous 401 (`42501`); own-device
  insert 201; spoofed device_id 403 (`new row violates row-level security`);
  device SELECT on scan_attempts 403 (push-only holds); server-side counts
  confirm 1 device-a row, 0 device-b rows.
- **realtime-second-subscriber: passed** — exit 0, first run. JOIN-OK with a
  real postgres_changes subscription (no SYS-ERR), then the server-side
  redemption UPDATE arrived at the authenticated-role subscriber as
  `RTP EVENT … type=UPDATE table=public.codes id=55555555-…` within the 20s
  window. The card's done_when leg — "a row inserted/changed on one client
  appears on a second subscriber" — is proven with publication membership and
  per-subscriber RLS both live.

**Conclusion:** the card is buildable as designed. Build-facts it inherits:
(1) `sql/qr-schema.sql` in this spike's directory IS the working draft for the
in-repo `supabase/` migration — already proven to apply, discriminate, and
publish; (2) the RLS shape that held is: SELECT-for-authenticated on
campaigns/codes, INSERT-only on scan_attempts with `with check (device_id =
jwt sub)` — the scanner's push handler must therefore write `device_id` equal
to the token's `sub`; (3) hosted-project deltas (dashboard-managed publication,
key rotation) remain Activity 0 provisioning facts, not schema risks.

## Corrections

- **Assertion-shape defect in 01 (fixed, re-run green): a concatenated
  Postgres boolean casts to `true`/`false`, not psql's bare-column `t`/`f`.**
  First run of `01-schema-applies.sh` reported RED "RLS is not enabled" while
  its own output showed `campaigns:true / codes:true / scan_attempts:true` —
  the grep expected `:t$`. The schema was right; the expectation string was
  wrong. Fixed to `:true$` and re-run green. The same class was fixed in the
  sibling goal's race script (greps on `ok||'|'||…` output) before its first
  run. No premise changed.

## Review

- signed: operator, 2026-09-03 — covers 1 correction(s) (batch sitting with the
  sibling goal and Activity G; "Sign off all three" on the phrase-checked
  batch-review question).
