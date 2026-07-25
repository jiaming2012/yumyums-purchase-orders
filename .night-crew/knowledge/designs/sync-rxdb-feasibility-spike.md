# GO — self-hosted Supabase accepts a Go-minted HS256 token, with no GoTrue, on both PostgREST and Realtime, and its RLS demonstrably discriminates.

**Card:** W1 `sync-spike-stack-and-jwt-bridge` (wave-0 gate, run 2026-07-25-autonomous)
**Scope of the verdict:** the *substrate*. This says the stack works and can be
driven from Go. It says nothing about RxDB, which is card W2's question, and
nothing about HQ's policy semantics, which is `sync-jwt-bridge-endpoint`'s.
**Reproduction:** [`.night-crew/qa/spike-supabase/README.md`](../../qa/spike-supabase/README.md) — every command, with its real output.

---

## The verdict in one paragraph

A three-service self-hosted Supabase stack (`supabase/postgres:15.8.1.060` +
`postgrest/postgrest:v12.2.12` + `supabase/realtime:v2.34.47`) came up locally
and accepted a JWT minted by a ~10-line Go program that imports nothing but
`crypto/hmac`, `crypto/sha256`, `encoding/base64` and `encoding/json`. Through
PostgREST that token performed an authorized read and an authorized write, and
was **refused** on an unauthorized write (403) and an unauthorized read (empty
result / 401 for anon) — with a `service_role` BYPASSRLS control proving the
scoping was RLS rather than an empty table. Over WebSocket, the same token
subscribed to Realtime and received a live `postgres_changes` event, and a
second subscriber proved Realtime enforces the same RLS per subscriber.
**Kong, Studio, and GoTrue were not deployed and none proved necessary.** The
sharpest edge — the Realtime tenant row — was resolved by an environment
variable, not by hand-crafting encrypted rows. Nothing found during the spike
argues against the architecture.

---

## Evidence

Full commands and untrimmed context are in the runbook. What follows is the
evidence chain, each item an observed fact.

### The stack came up

Three attempts. Attempts 1 and 2 failed for concrete, now-encoded reasons
(below); attempt 3 was green and the stack has been up continuously since.

```
NAME                        IMAGE                          SERVICE    STATUS                    PORTS
spike-supabase-db-1         supabase/postgres:15.8.1.060   db         Up (healthy)              0.0.0.0:46011->5432/tcp
spike-supabase-realtime-1   supabase/realtime:v2.34.47     realtime   Up                        0.0.0.0:46355->4000/tcp
spike-supabase-rest-1       postgrest/postgrest:v12.2.12   rest       Up                        0.0.0.0:46233->3000/tcp
```

Image pull: 3.3 GB total, of which `supabase/postgres` is 3.0 GB. Pull was clean
and took under five minutes on a cold cache.

### The token is minted in Go with stdlib only

```
{"alg":"HS256","typ":"JWT"}
{"exp":1784995399,"iat":1784991799,"role":"authenticated","sub":"user-alice"}
```

`.night-crew/qa/spike-supabase/mintjwt/main.go`. `backend/go.mod` was **not
touched**; no JWT module was added. The `sign()` function is ten lines.

### PostgREST: read, write, and a policy that actually discriminates

Both halves of the discrimination requirement were obtained. Stated plainly:
**yes, an authorized request succeeded AND an unauthorized one was refused.**

| Proof | Request | Observed |
|---|---|---|
| P1 | GET all, token `sub=user-alice` | `[{"id":"note-alice-1","owner_id":"user-alice",...}]` · **200** |
| P2 | GET all, token `sub=user-bob` (identical URL) | `[{"id":"note-bob-1","owner_id":"user-bob",...}]` · **200** |
| P3 | GET all, **no token** | `{"code":"42501",...,"message":"permission denied for table spike_notes"}` · **401** |
| P4 | GET `id=eq.note-bob-1` as alice | `[]` · **200** |
| P5 | POST own row as alice | **201**, and `_modified` returned as `2026-07-25T15:03:41` despite the client sending `1999-01-01` |
| P6 | POST row with `owner_id=user-bob` as alice | `{"code":"42501",...,"message":"new row violates row-level security policy for table \"spike_notes\""}` · **403** |
| P7 | PATCH bob's row as alice | `[]` · **200**; bob's row verified unchanged afterwards |
| P8 | GET with an expired token | `{"code":"PGRST301",...,"message":"JWT expired"}` · **401** |
| P9 | GET with a token signed by the wrong secret | `{"code":"PGRST301",...,"message":"JWSError JWSInvalidSignature"}` · **401** |
| P10 | GET with `role: service_role` (BYPASSRLS control) | **all three rows across both owners** · **200** |

P10 is what makes P1/P2/P4 mean something: it rules out "the table was empty" and
"the URL was wrong" as explanations for the scoped results. The rows were always
there; RLS was hiding them.

P5 additionally establishes that the `_modified` trigger overrides a
client-supplied timestamp — the property that makes a pull checkpoint safe
against device clock skew.

### Realtime: tenant, subscription, and a delivered change event

Negative control first — dialing with `Host: localhost` (tenant `localhost`,
which does not exist):

```
rtwatch: handshake failed: failed to WebSocket dial: expected handshake response
status code 101 but got 403 (HTTP 403 Forbidden)
```

Then, with the tenant's name as the first host label:

```
CONNECTED ws://realtime-dev.localhost -> 127.0.0.1:46355 (HTTP 101 Switching Protocols)
RECV event=system           payload={"message":"Subscribed to PostgreSQL","status":"ok",...}
RECV event=postgres_changes payload={"data":{"table":"spike_notes","type":"INSERT",
  "record":{"_deleted":false,"_modified":"2026-07-25T15:04:28.376289+00:00",
            "body":"realtime probe row","id":"note-alice-rt-1","owner_id":"user-alice"},
  "columns":[...],"errors":null,"schema":"public",
  "commit_timestamp":"2026-07-25T15:04:28.377Z"},"ids":[130398987]}
```

And Realtime enforces RLS per subscriber. With **bob** subscribed, an alice-owned
row and a bob-owned row were inserted in the same batch. Bob's socket received
exactly one:

```
RECV event=postgres_changes payload={...,"id":"note-bob-rt-1","owner_id":"user-bob"...}
```

The alice row was inserted **first** and never arrived. The token scopes the
change stream, not just REST reads.

WebSocket client: `github.com/coder/websocket v1.8.14` — verified before use as
an existing **direct** dependency at `backend/go.mod:11`. Nothing new entered
HQ's supply chain.

### The tenant row — expected to be the sharpest edge, and it was, but it yielded

Self-hosted Realtime is multi-tenant even with one tenant, and resolves the
tenant from the **first dot-separated label of the HTTP `Host` header**. Hosted
Supabase hides this because every project is `<ref>.supabase.co`.

The tenant row was **not** hand-inserted. `SEED_SELF_HOST=true` +
`SELF_HOST_TENANT_NAME=realtime-dev` in the compose file made the container seed
both `_realtime.tenants` and its `postgres_cdc_rls` `_realtime.extensions` row on
boot:

```
 external_id  |     name     | has_secret
--------------+--------------+------------
 realtime-dev | realtime-dev | t
```

This mattered because **`_realtime.tenants.jwt_secret` is encrypted at rest**
with `DB_ENC_KEY` (which must be exactly 16 characters — AES-128). A hand-written
`INSERT` would have needed the secret pre-encrypted in Realtime's own scheme. The
Go client's job then reduced to presenting the right `Host` header, which
`rtwatch` does with a custom `http.Transport.DialContext` that splits the Host
header from the TCP address.

**Recorded exactly, since the card asked:** what it took was one environment
variable pair (`SEED_SELF_HOST` / `SELF_HOST_TENANT_NAME`) and one client-side
Host-header split. It cost **zero** hand-written encrypted rows and zero retries
once the mechanism was understood. If a tenant ever needs renaming,
`UPDATE _realtime.tenants SET external_id = ...` preserves the encrypted secret.

---

## The per-table contract — the number that sizes `sync-rxdb-schema-and-replication`

All six verified as observed facts on `public.spike_notes`.

| # | Requirement | Per-table cost | Verified by |
|---|---|---|---|
| 1 | `id text primary key` | column | table created and written through PostgREST (P5) |
| 2 | `_deleted boolean not null default false` | column | round-tripped `true` via PATCH; present in the Realtime payload |
| 3 | `_modified timestamptz not null default now()` | column | present in every REST and Realtime payload |
| 4 | `BEFORE INSERT OR UPDATE` trigger stamping `_modified` | **1 trigger per table** (the function is shareable) | P5 — client sent `1999-01-01`, server returned `2026-07-25` |
| 5 | `ENABLE ROW LEVEL SECURITY` + `REVOKE ... FROM anon` + `GRANT ... TO authenticated` + policy set | **per table**; predicates repeat | P1–P3, P6 |
| 6 | `ALTER PUBLICATION supabase_realtime ADD TABLE` + `REPLICA IDENTITY FULL` | **per table, manual, no UI in self-hosted** | `pg_publication_tables` shows `spike_notes` present and the deliberately-skipped `spike_unpublished` absent |

**Concrete sizing: three mechanically-repeated DDL statements plus one policy set
per table.** Items 1–3 are ordinary column definitions. Items 4–6 are the
recurring work. Critically, **none of the six required per-table judgement** —
they are uniform enough to generate from a table list, which is the good news for
`sync-rxdb-schema-and-replication`. The `supabase_realtime` publication already
exists in a fresh stack (the image's own `00000000000001-initial-schema.sql` runs
`create publication supabase_realtime;`), so it is only ever `ADD TABLE`.

**The failure mode of forgetting item 6 is worse than an error.** Measured, and
this corrects a guess we had written down earlier: it is *not* silent, but the
`phx_join` reply still returns `{"status":"ok"}` with a subscription id. The
error arrives afterwards as a separate `system` frame:

```
{"status":"error","extension":"postgres_changes",
 "message":"Unable to subscribe to changes with given parameters. Please check
            Realtime is enabled for the given connect parameters: [...]"}
```

A client that resolves its "subscribed" state on the join reply will believe it
is subscribed to a table that will never fire.

---

## Sizing paragraph for `sync-jwt-bridge-endpoint`

**Small — call it half a day of build, with one design conversation in front of
it that this spike deliberately did not have.**

The mechanical part is nearly free and is already written. Minting is
`mintjwt/main.go`'s `sign()`: ten lines, `crypto/hmac` + `crypto/sha256` +
`encoding/base64` + `encoding/json`, **no new dependency** (`backend/go.mod`
stays untouched). The endpoint itself is a handler that reads the current HQ
session, builds a claims map, signs it with a new `HQ_SUPABASE_JWT_SECRET`, and
returns the token plus its `exp`. PostgREST and Realtime share that one secret;
there is no key distribution problem, no JWKS, no rotation machinery required for
v1. Verified: `exp` is enforced (P8) and a wrong-secret signature is rejected
(P9), so the bridge fails closed on both axes without extra work.

**Three things will actually consume the time, and two of them are decisions, not
code:**

1. **Claim shape is an open operator decision** (see below). Until it is settled,
   the endpoint cannot be finished — only the signing half can.
2. **`auth.uid()` and `auth.jwt()` are unusable here and every hosted-Supabase
   policy example uses them.** Without GoTrue's migrations the `auth` schema
   ships only `email`, `role`, `uid`; and `auth.uid()` is
   `current_setting('request.jwt.claim.sub', true)::uuid` — it reads the **legacy
   singular** GUC that PostgREST only sets when `PGRST_DB_USE_LEGACY_GUCS=true`
   (this stack sets it `false`), and it **casts to `uuid`**, which HQ user ids
   may not be. Policies must use
   `current_setting('request.jwt.claims', true)::json ->> '<claim>'` directly.
   Budget for the fact that copy-pasted policies from Supabase's docs will not
   work, and that this will not be obvious from the error.
3. **Token lifetime and refresh.** Offline-first is the entire premise; a device
   can be offline across an expiry. Nothing about that was tested here. It is a
   design question for the endpoint card, and it is the one most likely to grow.

**Risk to the estimate is concentrated in (1), not in the crypto.** The crypto is
done and proven.

---

## Open question — recorded, deliberately NOT answered

Per the card's instruction, RLS *policy design* belongs to `sync-jwt-bridge-endpoint`
and to the operator. The spike stopped at the boundary rather than crossing it.

The `owner_id = current_setting('request.jwt.claims',true)::json->>'sub'` policy
in `sql/spike-fixture.sql` is a **proof device**, chosen because it is the
simplest predicate that can be observed either admitting or refusing a request.
It is not a proposal.

What the spike surfaced but did not decide:

- HQ's Users-tab grants are **per-tab / per-feature, not per-app** (established
  convention). Does each grant become a claim in the token, or does the token
  carry identity only while policies join against an HQ-side grants table? The
  first bloats the token and makes revocation wait for expiry; the second puts a
  join in every policy.
- **HQ rows frequently have more than one legitimate reader.** A checklist
  submission belongs to a submitter *and* an approver. A single-subject predicate
  cannot express that, and the shape of the multi-party case will drive the claim
  design more than the single-owner case will.
- **`service_role` is a `BYPASSRLS` god-token.** Used here only as the P10
  control. What, if anything, is ever permitted to mint one?

---

## Sharp edges (all hit for real; each is encoded as a comment where it bites)

1. **Setting `POSTGRES_USER` breaks the image's own bootstrap** (bring-up attempt
   1). The image defaults it to `supabase_admin`; overriding it to `postgres`
   makes `migrate.sh` fail with `Role "supabase_admin" does not exist` and the
   container exits 2. Omit it.
2. **The image creates `authenticator` with no password** (bring-up attempt 2).
   PostgREST loops on `password authentication failed for user "authenticator"`
   and exits 1. Supabase ships the fix as a separate `roles.sql`, not in the
   image. Ours is `initdb/99-roles.sql`; the `99-` prefix is load-bearing.
3. **`pg_isready` is not a readiness check for this image** — it answers on the
   temporary bootstrap server during `initdb`, so dependents start too early. The
   healthcheck uses an authenticated `psql -c 'select 1'`.
4. **`_realtime` schema must pre-exist**; Realtime's migrator does not create it.
5. **`docker-entrypoint.sh: ignoring .../init-scripts` is a red herring** — the
   image's `migrate.sh` processes that directory afterwards.
6. **Realtime tenant routing is by `Host` header**, and the failure is a bare
   `403` with no body — indistinguishable at a glance from an auth problem.
7. **`DB_ENC_KEY` must be exactly 16 characters** and changing it orphans every
   existing tenant row.
8. **`auth.jwt()` does not exist and `auth.uid()` is wrong for HQ** — see the
   sizing section. This is the one most likely to cost someone an afternoon.

## What was NOT established

Stated plainly rather than softened:

- **Nothing about RxDB.** No RxDB client was run. Whether RxDB's replication
  protocol maps cleanly onto this substrate is card W2's question, untouched here.
- **Nothing about token refresh across an offline period.** Not tested.
- **Nothing about scale, WAL retention, or slot growth.** Realtime created two
  logical replication slots (`wal2json` and `pgoutput`); their behaviour under a
  disconnected subscriber over days was not exercised.
- **Nothing about HQ's real schema.** One purpose-built fixture table was used.
  The per-table cost above is measured on that one table and extrapolated by
  counting steps, not by applying it to HQ's tables.
- **No `DELETE` event was observed.** `REPLICA IDENTITY FULL` was set and INSERT
  and UPDATE were exercised; a hard `DELETE` change event was not, because the
  replication model is soft-delete by design.
