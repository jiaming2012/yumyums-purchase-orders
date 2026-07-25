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

---
---

# GO (RxDB) — RxDB 17.4.0 replicates over the self-hosted stack in BOTH directions, on free/Apache-2.0 licensing, using a real shipped plugin. One assumption did not survive contact: the configuration is NOT last-write-wins.

*Card W2 `sync-spike-rxdb-replication`, 2026-07-25. Appended below W1's verdict,
which is settled and untouched. Reproduce everything here with
[`.night-crew/qa/spike-supabase/README.md`](../../qa/spike-supabase/README.md)
**half 2**; the harness is `.night-crew/qa/spike-supabase/rxdb/`.*

## The verdict in one paragraph

RxDB does what the migration needs it to do. Against the exact stack W1 left
running — self-hosted Postgres + PostgREST + Realtime, no GoTrue, no Kong, an
HS256 token minted by W1's stdlib-Go program — `rxdb@17.4.0`'s
`replicateSupabase` plugin **pushed** a locally-created document into Postgres
and **pulled** remote inserts, updates and soft-deletes into a *running* client
in 45–130 ms with no restart and no manual `reSync()`. Both directions were
proven by separate scripts, because a combined proof cannot tell you which
direction carried the data. The licensing question resolves in our favour: the
`rxdb` package is Apache-2.0, the Dexie browser storage is free, and premium buys
speed rather than capability — **no paid dependency is required to ship this**.
The Supabase replication is a genuine shipped plugin, not an example we would
maintain. **The one thing that did not survive contact with reality is the
conflict policy**: the explore session assumed last-write-wins, and the observed
behaviour is unconditional *master-wins* with the losing write discarded
silently. That is a finding for the operator, recorded below and routed to
DECISIONS-NEEDED, not something this card corrected.

## Evidence

Two independent runs, identical results. Real captured output for all of it is
transcribed in runbook half 2; the summary:

| Question | Observed | Where |
|---|---|---|
| Cold install works | `added 78 packages … in 20s`, 0 vulnerabilities. rxdb 17.4.0, @supabase/supabase-js 2.109.0, ws 8.21.1 | half 2, step 0 |
| Client reaches the gateway-less stack | REST rows returned, **bob's seed row absent** (RLS still discriminating through supabase-js); Realtime `SUBSCRIBED` | step 1a |
| **Push** | local `collection.insert()` → row present in Postgres, verified over an independent request: `HTTP 200`, `_modified` stamped by the server trigger | step 3 |
| **Pull, no restart** | remote INSERT converged 128 ms, remote UPDATE 129 ms, remote soft-delete 121 ms, all into a client that was never restarted and never `reSync()`-ed | step 4 |
| Conflict | later local write **discarded**, earlier server write survived, `error$` emitted **0** events | step 5 |
| Licensing | `rxdb@17.4.0` `"license": "Apache-2.0"`; Dexie free, IndexedDB `👑` premium | step 6 |
| Plugin vs example | real 247-line shipped plugin, `rxdb/plugins/replication-supabase` | step 6 |

The 45–130 ms convergence is itself the evidence that pull arrived over the
Realtime stream rather than a retry: `replicateSupabase`'s `retryTime` defaults
to 5000 ms.

## THE FINDING — conflict resolution is master-wins, not last-write-wins, and the loss is silent

**Read this before sizing anything else.**

Constructed case: one agreed document; client goes offline; **Postgres edited
first (T1)**; **RxDB edited second (T2 > T1)** — the local write strictly later
in wall-clock time; client reconnects on the same `replicationIdentifier`.

Observed:

```
local  body after reconnect : REMOTE-EDIT (written first, T1)
remote body after reconnect : REMOTE-EDIT (written first, T1)
replication errors surfaced : 0 []
conflict handler invocations: 1
    newDocumentState.body  : LOCAL-EDIT (written second, T2)    <- the local (later) write
    realMasterState.body   : REMOTE-EDIT (written first, T1)    <- what the server actually held
    handler CHOSE          : REMOTE-EDIT (written first, T1)
```

**Which clock decided: none.** The mechanism is optimistic concurrency, not
timestamp comparison. `replicateSupabase`'s push issues a compare-and-swap
(`UPDATE … WHERE id = … AND body = <assumed master value> AND …`); the remote
edit had already moved `body`, zero rows matched, and the plugin returned the
row as a conflict. RxDB's `defaultConflictHandler` then resolved it by its own
documented rule — *"The default conflict handler will always drop the fork state
and use the master state instead"* — returning `realMasterState`. `_modified`
never participated: it is the pull cursor, and with `_modified` absent from the
collection schema it is not even in the compare-and-swap. Skewing the client
clock in either direction changes nothing.

**Why it matters for HQ, concretely.** A crew member completes a checklist on a
phone with no signal in the truck. A manager edits the same submission from the
office. The phone reconnects. **The crew member's work is dropped, and nothing
is emitted on `error$` that the app could use to tell them.** From inside the app
the offline edit simply never happened. For a product whose stated core value is
"accountability — who checked what", silently losing the crew member's entry is
a product-level problem, not a tuning detail.

**This card did not fix it, deliberately.** The observing conflict handler used
to capture the evidence delegates every decision to the default and only prints
what happened; making it nicer would have destroyed the finding. RxDB supports a
per-collection custom `conflictHandler` and that is the correct hook for a real
policy — but *which* policy is an operator/product call. Routed to
`.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md`.

## Licensing and storage — the go/no-go input

**No paid dependency is required.** Verified against RxDB's current pages, not
taken on anyone's word:

- `rxdb@17.4.0` ships **Apache-2.0** (verified locally: `package.json`
  `"license"` field and the full Apache text in `LICENSE.txt`).
- **Dexie storage is free** and is the browser storage a real HQ PWA would use.
- **IndexedDB storage is premium.** <https://rxdb.info/rx-storage.html> marks it
  `👑 IndexedDB`, usable *"if you have 👑 premium access"*, and recommends:
  *"Use the LocalStorage storage for simple setup and small build size. For
  bigger datasets, use either the dexie.js storage (free) or the IndexedDB
  RxStorage if you have 👑 premium access."* Corroborated by
  <https://rxdb.info/premium/>, whose free tier is *"Default RxStorage (Dexie,
  Memory, LokiJS)"* and whose paid tiers add OPFS, IndexedDB, SQLite,
  Filesystem, Worker, SharedWorker, Sharding, Memory-Mapped and the Localstorage
  Meta Optimizer.

**The operator's reading was correct.** Premium buys performance, not
capability, and is an optimisation available later rather than a gate on
starting. Note the free Dexie storage is itself IndexedDB-backed via dexie.js;
the premium `storage-indexeddb` is a faster direct implementation.

## Plugin, not example — but young

**The BACKLOG's wording is right: it is a real shipped plugin**, so we do not
carry the replication protocol as a maintenance cost. Verified three ways:
it is exported from `rxdb@17.4.0`'s `package.json` as `./plugins/replication-supabase`
with 247 lines of implementation and full TypeScript types; `rxdb`'s own shipped
`README.md` lists Supabase among *"production-ready plugins"*; and
<https://rxdb.info/replication-supabase.html> documents
`import { replicateSupabase } from 'rxdb/plugins/replication-supabase'` with no
example/community/beta caveat.

**Size it as young, though.** `rxdb`'s shipped `CHANGELOG.md`:
`### 16.19.0 (4 September 2025) — ADD Supabase Replication Plugin (beta)`. It
entered as beta ~10 months before the 17.4.0 we are on, with active work since
(`FIX(supabase-replication) push.modifier is not used` in 16.20.0,
`feat: replication-supabase querybuilder` in 16.21.0). We are early adopters of
a young plugin on a self-hosted stack its author most likely tests against
hosted Supabase. Budget reading its source when something is odd — as this card
had to. 247 readable lines is a bounded risk.

## Second finding — supabase-js assumes Kong; we bridged it in the client

`@supabase/supabase-js` freezes `<baseUrl>/rest/v1` and `<baseUrl>/realtime/v1`
in its constructor — it assumes one origin behind Kong. W1 deliberately omitted
Kong, so a stock `createClient()` cannot reach either service in this stack.

The harness bridged it in ~25 lines using the two extension points supabase-js
already exposes (`global.fetch` to strip the `/rest/v1` prefix,
`realtime.transport` to re-point host:port, rewrite the path to
`/socket/websocket`, and set the tenant `Host` header). It worked first try, so
this is not a blocker — but it forces a decision the migration must make
deliberately: **run Kong, or ship a small permanent client-construction helper
in HQ that tracks a supabase-js internal.** Both viable; the second is one fewer
service to run and secure but is standing HQ code coupled to a library detail
that could move in a minor release. Belongs to whichever card owns client
construction.

## Sizing for `sync-rxdb-schema-and-replication`

W1 measured the per-table substrate cost. W2 adds the client-side cost.

**Small–medium, and the schema work is genuinely the easy half.** The mechanism
is proven, so that card is modelling plus one real decision, not research.

Per collection, once the pattern exists:
1. an RxDB JSON schema mirroring the table (text PK; **do not declare `_deleted`**;
   decide `_modified` deliberately, see below) — minutes;
2. one `replicateSupabase({ … })` call — minutes;
3. W1's per-table SQL contract (text PK, `_deleted`, `_modified` trigger, RLS,
   `alter publication`, `replica identity full`) — W1's measured cost.

The three things that will actually consume the time:

- **The conflict policy.** This is the card's real work and it is a design task,
  not a coding one, blocked on the operator decision above. Once decided, the
  implementation is a per-collection `conflictHandler` — a small function, but it
  needs a rule that covers HQ's actual multi-actor rows (a submission has a
  submitter *and* an approver; "owner" is not one field) and a way to surface a
  discarded write to the user, since the default surfaces nothing.
- **`_modified` in the schema: a semantics switch, not a formality.** Declaring
  it makes the plugin round-trip the server timestamp *and* makes
  `addDocEqualityToQuery` include `_modified` in the compare-and-swap, tightening
  conflict detection so any server-side touch is a conflict. Leaving it out keeps
  it a pure pull cursor (what this spike ran). Decide it; do not let it be
  decided by whether someone copied the field in.
- **A browser-side check.** Everything below is untested (see next section) and
  the storage and service-worker items are the two most likely to produce a nasty
  surprise. Budget a real browser spike inside that card rather than assuming the
  Node result transfers.

Do **not** size in a replication-protocol implementation. That is the plugin's.

## What a Node-side proof does NOT establish

Stated plainly rather than softened. This harness proves the **replication
protocol**. It does not prove:

- **Browser storage behaviour.** Runs used `getRxStorageMemory()`. Nothing here
  exercises Dexie/IndexedDB, quota limits, eviction under storage pressure,
  Safari's stricter eviction, or private browsing. A memory store cannot fail the
  way a browser store fails.
- **Persistence across reloads.** Memory storage starts empty every run.
- **Service-worker interaction.** HQ's Workbox `sw.js` is network-first for API
  calls with an offline JSON fallback. Its interaction with RxDB's fetches and
  with a long-lived Realtime WebSocket is completely untested, and an offline
  fallback answering a replication request with cached JSON is a plausible and
  nasty failure.
- **PWA offline semantics.** No airplane mode, no flaky network, no backgrounded
  tab, no iOS killing the page. The "offline" in the conflict proof was
  `rep.cancel()` — the friendliest possible version of going offline.
- **Multi-tab leader election.** `waitForLeadership: false` was set because the
  harness is one process; browsers default it to `true` and that surface is
  untested.
- **Token refresh across an offline period.** Still untested — W1 did not, and
  neither did W2. A token minted with a 1 h TTL and a truck that is offline
  longer than that is an open question.
- **HQ's real schema, volume or relations.** One flat fixture table with a
  single-owner predicate.
