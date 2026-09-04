# supabase/ — the attribution spine's arbiter (Activity A)

The contract of record for the QR-redemption arbiter that lives in Supabase:
the `campaigns` / `codes` / `scan_attempts` schema, its RLS, the Realtime
publication membership, and the `marketing_settings` surface (#5). Created by
night-crew card `supabase-schema-and-rls` (run 20260904); extended by
`redeem-rpc-race-proof` (the atomic `redeem()` RPC).

## Layout

```
supabase/
  migrations/   one .sql file per change, additive + idempotent, timestamp-named
  seed.sql      TEST fixtures ONLY — never production
  verify/       the standalone gate harnesses (exit codes are the verdict)
```

## Apply order

1. **`migrations/*.sql` in lexicographic filename order.** Files are named
   `YYYYMMDDHHMMSS_name.sql` (Supabase CLI convention), so lexicographic order
   IS chronological order. Every file is additive and idempotent — it applies
   clean on a bare database and on top of its own output. No migration may
   drop a table: the spike fixture this tree grew from was drop-first *by
   design* (spike runs restart from the §4 text); a migration is not.
2. **`seed.sql` afterwards — on LOCAL/TEST substrates only.** It seeds the two
   TEST campaigns and five TEST codes the verify harnesses and Card 2's race
   proof drive (fixed UUIDs — the file header is the fixture contract). The
   production welcome-offer campaign is **Activity E's** to create; it must
   never appear here, and this file must never be applied to a production
   project.

`verify/lib.sh`'s `apply_all` implements exactly this order and is what the
harnesses (and Card 2's) use.

## Target discipline

- **Tonight's only target is the committed LOCAL throwaway substrate**:
  compose project `spike-supabase` (`docker-compose.supabase.yml`), brought up
  via `.night-crew/qa/spike-supabase/env-up.sh` in RECONCILE mode — never
  `--fresh` from a harness. Its credentials are throwaway values committed on
  purpose; never reuse them anywhere real.
- **No hosted Supabase project exists yet.** Provisioning one is Activity 0's
  attended card (`external-accounts-provision`); when it lands, migrations
  apply there in the same order, and the dashboard-side equivalents of the
  publication/key wiring are that card's checklist (spike extraction record).
- 🛑 **NEVER `:5433`** (the dev AND production HQ cluster — decision 155) and
  **never `:5434`** (the Playwright/Go test pg). The arbiter has nothing to do
  with either.

## Verifying

```
supabase/verify/01-structure.sh   # applies fresh + warm; every claim by name
supabase/verify/02-rls-six-legs.sh
supabase/verify/03-realtime-second-subscriber.sh
supabase/verify/04-redeem-race.sh # the race gate: 20×2 clients, 0 double-wins
supabase/verify/reset-bare.sh     # fixture action: drop ONLY this card's objects
```

🛑 The verdict is each script's **exit status**, never its prose
(`0` green, `1` red, `2` could-not-run). Capture whole logs
(`cmd > log 2>&1; echo EXIT=$?`) — never pipe a gate through `tail`/`head`.
Red-first evidence for the schema card (harness RED against a bare substrate
before the migration existed) is in
`.night-crew/runs/2026-09-04-autonomous/card1-red-*.log`.

## marketing_settings (#5) — how the threshold is consumed

`public.marketing_settings` is a structurally single-row table
(`marketing_settings_singleton` check). `requires_online_threshold_cents`
(seeded **2000** = $20.00) is the value **campaign creation consults** to
derive a new campaign's `requires_online` flag: face value in cents `>=`
threshold → `true` (no offline override, §8), else `false`. The derived flag
is stored on the campaign row; devices only ever read `campaigns.requires_online`.

- **Configurable without a migration**: changing the threshold is an `UPDATE`
  (service_role/admin tooling). The migration's seed is
  `on conflict do nothing`, so a re-applied migration never clobbers an
  operator-set value — `01-structure.sh` proves that leg explicitly.
- **Server-side only**: RLS enabled, zero policies, zero client grants.
  Devices never read this table.
- The threshold applies **at creation time**; already-created campaigns keep
  their stored flag (a threshold change is not retroactive by design — a
  campaign's offline policy must not flip under devices mid-flight).

## redeem() — the atomic arbiter (§6)

`public.redeem(p_code uuid, p_device text) → (ok boolean, reason text)`
(`migrations/20260904000200_redeem_rpc.sql`, card `redeem-rpc-race-proof`). The
single conditional `UPDATE … WHERE redeemed_by IS NULL AND expires_at > now()`
is the **only** thing enforcing single use in the whole design — two concurrent
callers get exactly one `ok=true`, arbitrated by Postgres row locking.

- **Reason taxonomy (complete, never NULL):** `already_used` · `expired` ·
  `not_found`. The `not_found` arm is GAP-1's fix (spike-proven, operator-signed
  v2 body): the handoff §6 text returned a NULL reason for a nonexistent code,
  which would make a forged code read as a system outage downstream.
- **The winning UPDATE also bumps `codes.updated_at`** — the replication
  checkpoint key. Without it a redemption would be invisible to the pull
  replica (Activity B) until some other write touched the row.
- **SECURITY DEFINER, `search_path` pinned empty**; EXECUTE granted to
  `authenticated` + `service_role` only. The migration revokes from **both**
  `public` and `anon`: this substrate (like hosted Supabase) carries default
  privileges that hand `anon` an *explicit* function grant at create time, so
  revoking PUBLIC alone leaves `anon` executable. Devices drive it via
  PostgREST `POST /rpc/redeem`.
- **Gate:** `verify/04-redeem-race.sh` — 20 rounds × 2 concurrent clients (one
  `docker exec psql` connection per client), 0 double-wins, every loser
  `already_used`; expired / already_used / not_found arms on the seed fixtures;
  `updated_at`-advances leg; device-JWT RPC leg. `--red-analog` installs a
  naive check-then-update body AS `public.redeem` and proves the same legs red
  on the defect class (exit 1 is the expected evidence there — red is red).
