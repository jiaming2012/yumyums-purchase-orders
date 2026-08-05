# Supabase sync spike — runbook, half 1: the substrate and the JWT bridge

> ## ⚠ LOCAL ONLY
>
> Everything in this document runs on **your own machine, in Docker, against
> containers that exist only while you leave them running.** It is not
> production. It is not a hosted Supabase project — there is no `supabase.com`
> account involved at any point. It touches **no real HQ data**: not the prod
> database, not the night-crew ephemeral environment (`docker-compose.nc.yml`),
> not your `.env`.
>
> **Every credential in `docker-compose.supabase.yml` is a throwaway generated
> for this spike and committed to git in plain text on purpose**, so that this
> document reproduces byte-for-byte with no setup. Supabase's own self-hosted
> docs ship demo keys the same way. **Never reuse any of those values anywhere
> real.**

---

## What this document is

Night-crew card **W1 `sync-spike-stack-and-jwt-bridge`** is the wave-0 gate for
the RxDB-offline-sync cycle. Its job was to find out — *before* four more cards
get built on the assumption — whether a self-hosted Supabase substrate will
accept a JWT that HQ's own Go backend minted, with **no GoTrue**.

The verdict lives at
[`.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`](../../knowledge/designs/sync-rxdb-feasibility-spike.md).
**This file is the runbook: how to reproduce every proof behind that verdict
yourself.**

Every command below was actually run on 2026-07-25, and every **fact** asserted
under it was observed. Most output blocks are verbatim captures. **Six are not**
— they are annotated or abridged presentations of real runs — and they are
enumerated by name in [Integrity of the output blocks](#integrity-of-the-output-blocks)
below. Read that section before you treat any block here as byte-exact.
Where output is long it is trimmed to the line that matters, and the trim is marked.

**This half stands alone.** It covers the substrate (Postgres + PostgREST +
Realtime) and the token bridge. It does not involve RxDB at all, and it remains
a complete, runnable document whether or not half 2 is ever written.

### Integrity of the output blocks

*Added 2026-07-26 by card `sync-rxdb-browser-delivery-spike`, per T-22 decision
53. This document previously claimed, in two places, that every output block was
"the real captured output, not a reconstruction". That claim was **too strong**.
The claim has been narrowed rather than the blocks rewritten — see* ***Why
narrowed and not repaired*** *at the end of this section.*

**Every fact this document asserts re-verifies, and W1's GO stands.** What
follows is a correction to the *presentation*, not to any finding.

Six blocks are **hand-composed presentations of real runs** rather than
byte-exact captures. They are:

| # | Where | What was composed | Does it change any fact? |
|---|---|---|---|
| 1 | Proofs **P1–P10** (§ "Proving RLS actually discriminates") | The right-aligned `HTTP nnn` annotations — **ten of them** — were added by hand. The `curl` commands as printed carry no `-w`, `-i` or `-D -`, so they could not have emitted a status line. The status codes themselves were observed; they were observed on separate invocations. | No. To reproduce the annotation, append `-w ' HTTP %{http_code}\n'` to each `curl`. |
| 2 | Proof **R2** (Realtime, terminal 1) | Every `RECV` line is missing the `topic=` column. `rtwatch/main.go:144` prints `RECV event=%-18s topic=%-16s payload=%s` **unconditionally**, so a real capture always carries it. The column was dropped for page width. | No — the topic was `realtime:spike` throughout, which the `SENT phx_join topic=realtime:spike` line above it already shows. |
| 3 | Proof **R3** (the missing-`alter publication` negative proof) | Same missing `topic=` column, and `phx_reply` is padded to a **different width** than in R2 (`%-18s` would make both identical). Two paddings in one document is the tell. | No. |
| 4 | Step "apply the fixture" (psql command tags) | The block shows `DROP TRIGGER` but **not** the three `DROP POLICY` tags. `sql/spike-fixture.sql` runs one `drop trigger if exists` and three `drop policy if exists`; psql emits a command tag for each. Three tags were dropped, one kept. | No — the policies are created three lines further down in the same block (`CREATE POLICY` ×3). |
| 5 | Half 2, Step 1a (`node smoke.js`) | The elision read `... (5 more alice rows) ...`, implying the table held **6** user-alice rows. It held **8**. Corrected to a non-numeric elision, because the true number is not stable: sharp edge 13 means `spike_notes` **accumulates** rows on every run by design, so any count printed here is stale the moment another proof runs. | No — the fact being proven is that **zero `user-bob` rows appear**, which is unaffected by how many alice rows there are. |
| 6 | `.night-crew/runs/2026-07-25-autonomous/timings.log:34` | Records *"PostgREST half fully discriminating: **P1-P11** all as predicted"* against a runbook that documents **P1–P10**. There is no P11. | No. The historical run log is **left as written** — appending to or rewriting a closed run's timings is a worse defect than the off-by-one. The discrepancy is recorded here instead. |

**Why narrowed and not repaired.** Both were allowed. Repairing the
presentation would mean re-running W1's proofs and pasting fresh byte-exact
captures — and that cannot honestly be done now, because the stack is
**stateful and cumulative**: `spike_notes` has accumulated rows from every proof
run since (sharp edge 13), so a re-capture today produces *different bytes* for
the same commands. Pasting those under a claim of byte-exactness would recreate
the same defect one generation later. Narrowing the claim is the durable fix: it
tells a reader exactly which blocks to distrust as bytes and confirms that every
block is trustworthy as **fact**. The one place where a number was simply
**wrong** (row 5) was corrected outright rather than annotated.

### Prerequisites

- Docker with Compose v2 (`docker compose`, not `docker-compose`)
- Go on `PATH`. On this box that means `export PATH=/usr/local/go/bin:$PATH`
  first — a non-interactive shell does not have it.
- ~4 GB free disk for the images, and `curl`.

All paths below are relative to the repo root.

---

## 🛑 If you only want the environment up, run one command

**Added by card C1 `spike-a-environment-up`, run `20260806`.** Steps 0–2 below
are the *narrated* bring-up and they remain the explanation of record — read them
when you need to know **why** a step exists. But you no longer have to perform
them by hand, and you should not: everything through "schema applied and
healthy" is now one unattended script.

```bash
task spike:up          # idempotent: reconcile the stack, apply schema, verify
task spike:up:fresh    # destroy volumes and come up from nothing (the clean-machine test)
task spike:health      # verify only — touches nothing
task spike:down        # teardown, still a deliberate and separate act
```

or, with no `task` installed, the script the targets wrap:

```bash
./.night-crew/qa/spike-supabase/env-up.sh [--fresh|--health]
```

**🛑 The verdict is the script's exit status, never its prose.** `0` means the
substrate and RxDB are both genuinely up: three containers, all three fixture
schemas applied, PostgREST answering *and discriminating* by RLS, Realtime
`SUBSCRIBED`, and a real RxDB database created and round-tripped. Anything else
is non-zero with a named cause. No health assertion in it is advisory, and none
carries `|| true`.

Measured on 2026-08-06: **68.8 s** from destroyed volumes with no `node_modules`
and two of the three images deleted; **35.3 s** to reconcile a warm stack.

**Why this exists at all** — the run that wrote it found this stack sitting in
`docker ps` as three containers, `Up (healthy)`, **five days old, with no
schema**: `relation "public.spike_notes" does not exist`. Every eye-check of
`docker ps` over those five days would have read green. *"A container is
running"* is not *"the environment is up"*, and only an assertion that queries
the thing can tell the two apart. Capture:
[`captures/green-20260806-env-up.txt`](captures/green-20260806-env-up.txt).

---

## Step 0 — pull the images (do this first)

Nothing else works if the registry is unreachable, and finding that out costs
minutes here versus hours later.

```bash
docker pull supabase/postgres:15.8.1.060
docker pull postgrest/postgrest:v12.2.12
docker pull supabase/realtime:v2.34.47
```

Observed:

```
Status: Downloaded newer image for supabase/postgres:15.8.1.060
Status: Downloaded newer image for postgrest/postgrest:v12.2.12
Status: Downloaded newer image for supabase/realtime:v2.34.47

supabase/realtime:v2.34.47    250MB
supabase/postgres:15.8.1.060  3GB
postgrest/postgrest:v12.2.12  22.8MB
```

**3.3 GB, dominated entirely by `supabase/postgres`.** That image is a full
Postgres 15.8 with pgsodium, pgaudit, TimescaleDB, pg_cron and the rest of the
Supabase extension set baked in. Worth knowing before you plan a CI job around
it.

> **Only three services are involved. There is no Kong, no Studio, and no
> GoTrue.** Kong is an API gateway we do not need because we talk to PostgREST
> and Realtime on their own ports. Studio is a dashboard. GoTrue is an identity
> provider, and the whole premise of this migration — decided in the operator
> explore session of 2026-07-24 — is that **HQ mints its own tokens** and
> Supabase is a dumb sync substrate. Nothing in the bring-up turned out to need
> any of the three. That is a real finding, not an assumption: see
> [Sharp edges](#sharp-edges) for the two things the missing pieces *did* cost.

---

## Step 1 — bring the stack up (idempotent)

```bash
docker compose -p spike-supabase -f docker-compose.supabase.yml up -d
```

The `-p spike-supabase` project name is not decoration. HQ already runs a
night-crew Docker environment from `docker-compose.nc.yml`; pinning a distinct
project name is what guarantees this stack can never adopt, restart, or delete
those containers. **Re-running the command is safe** — Compose reconciles to the
same three containers rather than creating more.

Observed:

```
NAME                        IMAGE                          SERVICE    STATUS                    PORTS
spike-supabase-db-1         supabase/postgres:15.8.1.060   db         Up (healthy)              0.0.0.0:46011->5432/tcp
spike-supabase-realtime-1   supabase/realtime:v2.34.47     realtime   Up                        0.0.0.0:46355->4000/tcp
spike-supabase-rest-1       postgrest/postgrest:v12.2.12   rest       Up                        0.0.0.0:46233->3000/tcp
```

**Host ports are assigned by Docker, not fixed in the compose file** — the same
convention `docker-compose.nc.yml` uses, so two stacks can be up at once without
colliding. **Your port numbers will differ from the ones printed above.** Resolve
them, and keep these in your shell for the rest of the document:

```bash
export DBP=$(docker compose -p spike-supabase -f docker-compose.supabase.yml port db 5432 | cut -d: -f2)
export RESTP=$(docker compose -p spike-supabase -f docker-compose.supabase.yml port rest 3000 | cut -d: -f2)
export RTP=$(docker compose -p spike-supabase -f docker-compose.supabase.yml port realtime 4000 | cut -d: -f2)
echo "DB=$DBP REST=$RESTP RT=$RTP"
```

```
DB=46011 REST=46233 RT=46355
```

If a container is missing from `ps`, `docker compose -p spike-supabase -f
docker-compose.supabase.yml ps -a` will show it as `Exited`, and `docker logs
spike-supabase-<svc>-1` will say why. Both bring-up failures we hit are written
up under [Sharp edges](#sharp-edges) — read those before debugging your own.

---

## Step 2 — apply the fixture table

```bash
docker exec -i spike-supabase-db-1 \
  psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -f - \
  < .night-crew/qa/spike-supabase/sql/spike-fixture.sql
```

Observed (trimmed; `NOTICE: ... does not exist, skipping` lines are the
idempotent `drop if exists` guards and are expected on a first run):

```
CREATE TABLE
CREATE FUNCTION
DROP TRIGGER
CREATE TRIGGER
ALTER TABLE
REVOKE
GRANT
CREATE POLICY
CREATE POLICY
CREATE POLICY
DO
ALTER TABLE
INSERT 0 2
NOTIFY
```

[`sql/spike-fixture.sql`](sql/spike-fixture.sql) creates **one** table,
`public.spike_notes`, that carries the entire self-hosted contract an
RxDB-replicated table has to satisfy — text primary key, `_deleted`, a
`_modified` trigger, RLS enabled, publication membership, replica identity — and
seeds two rows owned by two different users. The file is commented line by line
with *why* each piece is load-bearing; read it rather than duplicating it here.
It is safe to re-run.

**Two rows, two owners, is the point.** A single-owner fixture cannot tell a
working policy apart from a policy that lets everything through. Every read
below is checked against *both*.

---

## Step 3 — mint a token in Go, with no JWT library

```bash
export PATH=/usr/local/go/bin:$PATH
cd .night-crew/qa/spike-supabase

export SECRET=2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c
export ALICE=$(go run ./mintjwt -secret "$SECRET" -sub user-alice)
export BOB=$(go run ./mintjwt -secret "$SECRET" -sub user-bob)
echo "$ALICE"
```

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODQ5OTUzOTksImlhdCI6MTc4NDk5MTc5OSwicm9sZSI6ImF1dGhlbnRpY2F0ZWQiLCJzdWIiOiJ1c2VyLWFsaWNlIn0.OrC2g-7lUeE_rYA7cEJeHI1ojWjomSENTj8lRXjpucE
```

Decode it — a JWT is not encrypted, and being able to eyeball the claims is most
of debugging this stack:

```bash
echo "$ALICE" | cut -d. -f1 | tr '_-' '/+' | base64 -d; echo
echo "$ALICE" | cut -d. -f2 | tr '_-' '/+' | base64 -d; echo
```

```
{"alg":"HS256","typ":"JWT"}
{"exp":1784995399,"iat":1784991799,"role":"authenticated","sub":"user-alice"}
```

**`$SECRET` above is literally the `JWT_SECRET` in
`docker-compose.supabase.yml`**, and it is simultaneously PostgREST's
`PGRST_JWT_SECRET` and Realtime's `API_JWT_SECRET`. That single shared secret is
the entire bridge. In the real migration it becomes one HQ backend secret; here
it is throwaway.

[`mintjwt/main.go`](mintjwt/main.go) imports **only** `crypto/hmac`,
`crypto/sha256`, `encoding/base64`, `encoding/json` (plus `flag`/`fmt`/`os`/
`time`). Its `sign()` function is ten lines. This matters because the natural
instinct is to add `github.com/golang-jwt/jwt` to HQ's `go.mod` for this, and
the whole point of the demonstration is that an HS256 JWT is a base64url-joined
header, payload, and HMAC-SHA256 — nothing more. **This would catch:** a
migration plan that treats "we need a JWT library" as a dependency decision
requiring review, when it is actually ten lines of stdlib.

The one non-obvious detail is `base64.RawURLEncoding` — base64url with **no
padding**. `StdEncoding` produces a token every verifier on earth rejects.

---

## Step 4 — PostgREST: prove the policy *discriminates*

This is the step that matters most, and the reason it is six proofs rather than
one. **A `200 OK` proves nothing.** A policy that lets everything through and a
policy that is doing real work return identical status codes. The only way to
tell them apart is to show the *same request* producing *different rows* for
different tokens, and to show an unauthorized attempt being **refused**.

### P1/P2 — same URL, two tokens, different rows

```bash
curl -s "http://localhost:$RESTP/spike_notes?select=id,owner_id,body" -H "Authorization: Bearer $ALICE"
curl -s "http://localhost:$RESTP/spike_notes?select=id,owner_id,body" -H "Authorization: Bearer $BOB"
```

```
[{"id":"note-alice-1","owner_id":"user-alice","body":"alice seed row"}]      HTTP 200
[{"id":"note-bob-1","owner_id":"user-bob","body":"bob seed row"}]            HTTP 200
```

Byte-identical requests apart from the token; disjoint results. **This would
catch:** an RLS policy accidentally written as `USING (true)`, which is the
single easiest way to ship a sync layer where every crew member can read every
other crew member's rows.

### P3 — no token at all

```bash
curl -s -w '\nHTTP %{http_code}\n' "http://localhost:$RESTP/spike_notes?select=id,owner_id,body"
```

```
{"code":"42501","details":null,"hint":null,"message":"permission denied for table spike_notes"}
HTTP 401
```

Refused at the **grant** layer, not the RLS layer — note `42501 permission
denied`, not an empty array. Grants and RLS are two independent gates and the
fixture closes both (`revoke all ... from anon`). **This would catch:** the very
common mistake of enabling RLS but leaving the default `anon` grant in place,
which leaves an unauthenticated hole that RLS alone does not close.

### P4 — asking for someone else's row by primary key

```bash
curl -s "http://localhost:$RESTP/spike_notes?id=eq.note-bob-1&select=id,owner_id,body" -H "Authorization: Bearer $ALICE"
```

```
[]      HTTP 200
```

Naming the row directly does not get you the row.

### P5 — an authorized write, and the server overriding the client's clock

```bash
curl -s -X POST "http://localhost:$RESTP/spike_notes" \
  -H "Authorization: Bearer $ALICE" -H 'Content-Type: application/json' \
  -H 'Prefer: return=representation' \
  -d '{"id":"note-alice-2","owner_id":"user-alice","body":"written by alice through PostgREST","_modified":"1999-01-01T00:00:00Z"}'
```

```
[{"id":"note-alice-2","owner_id":"user-alice","body":"written by alice through PostgREST","_deleted":false,"_modified":"2026-07-25T15:03:41.436515+00:00"}]
HTTP 201
```

Two proofs in one response. The write succeeded (**201**), and look at
`_modified`: **the client sent `1999-01-01` and the row came back stamped
`2026-07-25`.** The `BEFORE INSERT OR UPDATE` trigger overrode it. **This would
catch:** the failure where a device with a skewed clock writes a checkpoint
timestamp in the past — so every other replica silently re-pulls forever — or in
the future, so every other replica silently *misses* rows. The pull cursor is
only trustworthy if the server owns it.

### P6 — an unauthorized write is refused

```bash
curl -s -X POST "http://localhost:$RESTP/spike_notes" \
  -H "Authorization: Bearer $ALICE" -H 'Content-Type: application/json' \
  -d '{"id":"note-forged-1","owner_id":"user-bob","body":"alice trying to write as bob"}'
```

```
{"code":"42501","details":null,"hint":null,"message":"new row violates row-level security policy for table \"spike_notes\""}
HTTP 403
```

**403, not a silently-dropped row.** Alice authenticated perfectly well and was
still refused, because the `WITH CHECK` clause rejects a row she does not own.
**This would catch:** a client (or a compromised one) forging `owner_id` on push
to write into another user's dataset — which in a sync architecture is *the*
privilege-escalation path, because the client composes the rows.

### P7 — an unauthorized update affects nothing

```bash
curl -s -X PATCH "http://localhost:$RESTP/spike_notes?id=eq.note-bob-1" \
  -H "Authorization: Bearer $ALICE" -H 'Content-Type: application/json' \
  -H 'Prefer: return=representation' -d '{"body":"HIJACKED"}'
curl -s "http://localhost:$RESTP/spike_notes?id=eq.note-bob-1&select=id,body" -H "Authorization: Bearer $BOB"
```

```
[]                                                     HTTP 200
[{"id":"note-bob-1","body":"bob seed row"}]            HTTP 200
```

Note the asymmetry with P6: an UPDATE that matches no visible row is **not** an
error, it is an empty result. A client that only checks the status code would
believe the update landed. **This would catch:** a push handler that treats
`200` as "written" and never notices its writes are being silently scoped away.

### P8/P9/P10 — the token itself is actually being verified

```bash
export EXPIRED=$(go run ./mintjwt -secret "$SECRET" -sub user-alice -expired)
export WRONGSIG=$(go run ./mintjwt -secret "definitely-not-the-stacks-secret" -sub user-alice)
export SERVICE=$(go run ./mintjwt -secret "$SECRET" -sub svc -role service_role)

curl -s "http://localhost:$RESTP/spike_notes?select=id" -H "Authorization: Bearer $EXPIRED"
curl -s "http://localhost:$RESTP/spike_notes?select=id" -H "Authorization: Bearer $WRONGSIG"
curl -s "http://localhost:$RESTP/spike_notes?select=id,owner_id" -H "Authorization: Bearer $SERVICE"
```

```
{"code":"PGRST301","details":null,"hint":null,"message":"JWT expired"}                  HTTP 401
{"code":"PGRST301","details":null,"hint":null,"message":"JWSError JWSInvalidSignature"} HTTP 401
[{"id":"note-alice-1","owner_id":"user-alice"},
 {"id":"note-bob-1","owner_id":"user-bob"},
 {"id":"note-alice-2","owner_id":"user-alice"}]                                          HTTP 200
```

`exp` is enforced, and a token signed with the wrong secret is rejected — so the
shared secret is genuinely doing work, not being ignored.

**The third one is the control that makes all of P1–P7 mean something.**
`service_role` has `BYPASSRLS`. Handed a `role: service_role` claim, PostgREST
`SET ROLE`s into it and the *same table, same endpoint* returns **all three rows
across both owners**. That rules out the boring explanation for every scoped
result above — that the table was nearly empty, or the URL was wrong. The rows
were always there; RLS was hiding them.

> ⚠ `role: service_role` is a **BYPASSRLS god-token**. It is used here purely as
> a control. It must never be mintable by anything a client can reach.

---

## Step 5 — Realtime over the same token

Realtime is where self-hosted diverges hardest from hosted Supabase, and it was
the sharpest edge of the spike. The mechanism is worth understanding before you
run anything.

**Self-hosted Realtime is multi-tenant even when you have exactly one tenant,
and it resolves the tenant from the HTTP `Host` header** — specifically the
first dot-separated label. Hosted Supabase hides this because every project is
`<ref>.supabase.co`. Self-hosted, `localhost` means "tenant `localhost`", which
does not exist.

### R1 — the negative control: wrong tenant

```bash
export RT=$(docker compose -p spike-supabase -f docker-compose.supabase.yml port realtime 4000 | cut -d: -f2)
go run ./rtwatch -addr "127.0.0.1:$RT" -host localhost -token "$ALICE" -timeout 15s
```

```
rtwatch: handshake failed: failed to WebSocket dial: expected handshake response
status code 101 but got 403 (HTTP 403 Forbidden)
```

**A bare 403 with no body and no hint.** This is the error you will spend an
afternoon on if you do not know the mechanism, because it looks exactly like an
auth failure — and your token is fine. **This would catch:** hours lost
regenerating perfectly valid JWTs while the actual problem is a DNS name.

### R2 — the real subscription

The tenant row was created for us by `SEED_SELF_HOST=true` in the compose file.
Confirm it exists:

```bash
docker exec spike-supabase-db-1 psql -U supabase_admin -d postgres \
  -c "select external_id, name, jwt_secret is not null as has_secret from _realtime.tenants;"
```

```
 external_id  |     name     | has_secret
--------------+--------------+------------
 realtime-dev | realtime-dev | t
```

Now subscribe with the tenant's name as the first label of the host, and insert
a row while it is listening:

```bash
# terminal 1
go run ./rtwatch -addr "127.0.0.1:$RT" -host realtime-dev.localhost -token "$ALICE" -timeout 60s

# terminal 2, once terminal 1 prints "Subscribed to PostgreSQL"
docker exec spike-supabase-db-1 psql -U supabase_admin -d postgres \
  -c "insert into public.spike_notes (id, owner_id, body)
      values ('note-alice-rt-1','user-alice','realtime probe row');"
```

Observed in terminal 1 (payloads trimmed for width):

```
CONNECTED ws://realtime-dev.localhost -> 127.0.0.1:46355 (HTTP 101 Switching Protocols)
SENT phx_join topic=realtime:spike postgres_changes=public.spike_notes
RECV event=phx_reply      payload={"status":"ok","response":{"postgres_changes":[{"id":130398987,...}]}}
RECV event=presence_state payload={}
RECV event=system         payload={"message":"Subscribed to PostgreSQL","status":"ok",
                                   "extension":"postgres_changes","channel":"spike"}
RECV event=postgres_changes payload={"data":{"table":"spike_notes","type":"INSERT",
  "record":{"_deleted":false,"_modified":"2026-07-25T15:04:28.376289+00:00",
            "body":"realtime probe row","id":"note-alice-rt-1","owner_id":"user-alice"},
  "columns":[...],"errors":null,"schema":"public",
  "commit_timestamp":"2026-07-25T15:04:28.377Z"},"ids":[130398987]}
OK: postgres_changes received — Realtime delivered a row change over a Go-minted token.
```

**That is the gate, cleared.** A token HQ's Go backend minted, with no GoTrue
anywhere, opened a WebSocket to self-hosted Realtime and received a live row
change with the full record — including `_deleted` and `_modified`.

[`rtwatch/main.go`](rtwatch/main.go) does the Host-header trick with a custom
`http.Transport.DialContext` that ignores the URL's host and dials `-addr`
instead: the URL supplies the Host header, the flag supplies the socket. It
speaks Phoenix channels at `vsn=1.0.0` — the same protocol `supabase-js` speaks,
not a bespoke one. It uses `github.com/coder/websocket v1.8.14`, the exact
version `backend/go.mod` already lists as a direct dependency, so **nothing new
enters HQ's supply chain.**

### R3 — Realtime enforces RLS too, and the enrolment failure mode

Two more proofs, because "an event arrived" is as weak on its own as "HTTP 200".

**Does Realtime scope events per token?** Subscribe as **bob**, then insert one
alice-owned row and one bob-owned row in the same batch:

```bash
go run ./rtwatch -addr "127.0.0.1:$RT" -host realtime-dev.localhost -token "$BOB" -timeout 30s
# meanwhile:
docker exec spike-supabase-db-1 psql -U supabase_admin -d postgres \
  -c "insert into public.spike_notes (id,owner_id,body) values ('note-alice-rt-2','user-alice','alice row while bob is listening');" \
  -c "insert into public.spike_notes (id,owner_id,body) values ('note-bob-rt-1','user-bob','bob row while bob is listening');"
```

```
RECV event=system           payload={"message":"Subscribed to PostgreSQL","status":"ok",...}
RECV event=postgres_changes payload={"data":{"type":"INSERT","record":{...
                                     "id":"note-bob-rt-1","owner_id":"user-bob"}...}}
OK: postgres_changes received
```

**The alice row was inserted first and never arrived.** Bob's socket saw exactly
one of the two writes. Realtime evaluates the same RLS policies per subscriber,
so the token scopes the change stream, not just the REST reads. **This would
catch:** a design that assumes RLS covers REST and then leaks every row to every
connected device over the WebSocket.

**What happens if you forget `ALTER PUBLICATION`?** Build an identical table,
grant it, policy it, set replica identity — and skip only the publication step:

```bash
docker exec spike-supabase-db-1 psql -U supabase_admin -d postgres \
  -c "create table if not exists public.spike_unpublished (id text primary key, owner_id text not null, body text not null, _deleted boolean not null default false, _modified timestamptz not null default now());" \
  -c "alter table public.spike_unpublished enable row level security;" \
  -c "grant select, insert, update on public.spike_unpublished to authenticated;" \
  -c "create policy up_sel on public.spike_unpublished for select to authenticated using (owner_id = current_setting('request.jwt.claims', true)::json ->> 'sub');" \
  -c "alter table public.spike_unpublished replica identity full;"

go run ./rtwatch -addr "127.0.0.1:$RT" -host realtime-dev.localhost -token "$ALICE" \
                 -table spike_unpublished -timeout 30s
```

```
RECV event=phx_reply  payload={"status":"ok","response":{"postgres_changes":[{"id":97697523,
                               "event":"*","schema":"public","table":"spike_unpublished"}]}}
RECV event=system     payload={"message":"{:error, \"Unable to subscribe to changes with given
                               parameters. Please check Realtime is enabled for the given connect
                               parameters: [event: *, schema: public, table: spike_unpublished]\"}",
                               "status":"error","extension":"postgres_changes","channel":"spike"}
...
TIMEOUT/READ-ERR: failed to get reader: context deadline exceeded
```

Read that carefully, because it is the trap. **The `phx_join` reply says
`{"status":"ok"}`** and hands back a `postgres_changes` subscription id. A client
that checks only the join reply believes it is subscribed. The truth arrives
*afterwards*, in a separate `system` frame with `"status":"error"`. Then nothing
ever fires.

**This would catch:** an HQ Realtime wrapper that resolves its "subscribed"
promise on the join reply. It must treat the `system` / `status:"error"` frame as
a subscription failure, or a forgotten one-line `ALTER PUBLICATION` on one table
becomes a table that just quietly never syncs.

---

## Step 6 — the per-table contract

This is the number that sizes the next card. Enrolment is **per table**, and
there is no dashboard toggle in self-hosted:

```bash
docker exec spike-supabase-db-1 psql -U supabase_admin -d postgres \
  -c "select pubname, schemaname, tablename from pg_publication_tables order by 1,3;"
```

```
                pubname                 | schemaname |      tablename
----------------------------------------+------------+---------------------
 supabase_realtime                      | public     | spike_notes
 supabase_realtime_messages_publication  | realtime   | messages_2026_07_24
 ... (realtime's own internal partitions)
```

`spike_unpublished` is absent — it was created identically and simply never
added. **Six steps per table**, all observed:

| # | Step | Observed cost | What its absence does |
|---|------|---------------|-----------------------|
| 1 | `id text primary key` | column definition | client-generated offline ids cannot round-trip through a `bigserial` |
| 2 | `_deleted boolean not null default false` | column definition | hard `DELETE`s are invisible to a pull handler; replicas keep the row forever |
| 3 | `_modified timestamptz not null default now()` | column definition | no pull checkpoint cursor; every sync is a full sync |
| 4 | `BEFORE INSERT OR UPDATE` trigger stamping `_modified` | 1 function (shareable across tables) + **1 trigger per table** | client clock skew corrupts the cursor — silently re-pulling or silently missing rows (proof P5) |
| 5 | `ENABLE ROW LEVEL SECURITY` + `REVOKE ... FROM anon` + `GRANT ... TO authenticated` + policies | **per table**; the policy predicates repeat per table | grants without RLS = every user reads every row; RLS without revoke = unauthenticated hole (proofs P1–P3) |
| 6 | `ALTER PUBLICATION supabase_realtime ADD TABLE` + `REPLICA IDENTITY FULL` | **per table, manual, no UI** | subscription joins `ok` then never fires; `system` error frame is the only signal (proof R3) |

Steps 1–3 are ordinary column definitions. **Steps 4–6 are three
mechanically-repeated DDL statements per table plus a policy set** — the real
recurring cost. The good news is that they are uniform enough to generate: none
of the six required any per-table judgement in this spike. The `supabase_realtime`
publication itself already exists in a fresh stack (the image's own
`00000000000001-initial-schema.sql` runs `create publication supabase_realtime;`),
so it is only ever `ADD TABLE`, never `CREATE PUBLICATION`.

---

## Step 7 — teardown (a separate, deliberate act)

**The stack was left running on purpose.** Nothing tears it down for you. When
you are finished exploring:

```bash
docker compose -p spike-supabase -f docker-compose.supabase.yml down --volumes
```

`--volumes` is what makes it a real reset — without it the Postgres data volume
survives, and since all the schema work happens at first `initdb`, a re-`up`
against a stale volume will *not* re-run the init scripts. If a bring-up is
behaving strangely after you have edited anything under `initdb/`, `down
--volumes` first.

To reclaim the 3.3 GB of images as well:

```bash
docker rmi supabase/postgres:15.8.1.060 postgrest/postgrest:v12.2.12 supabase/realtime:v2.34.47
```

---

## Sharp edges

Everything below was hit for real during this spike.

**1. Setting `POSTGRES_USER` breaks the image's own bootstrap.** (Bring-up
attempt 1.) `supabase/postgres` defaults `POSTGRES_USER` to `supabase_admin` and
its post-`initdb` `migrate.sh` reconnects as that role to install the
`anon`/`authenticated`/`service_role`/`authenticator` set. Setting
`POSTGRES_USER=postgres` — the obvious, harmless-looking thing — makes `initdb`
create `postgres` as the superuser instead, and:

```
psql: error: FATAL:  password authentication failed for user "supabase_admin"
DETAIL:  Role "supabase_admin" does not exist.
```

The container exits `2` and both dependents never start. **Omit `POSTGRES_USER`
entirely.** Supabase's own self-hosted compose omits it too.

**2. The image creates `authenticator` with no password.** (Bring-up attempt 2.)
PostgREST connects as `authenticator`, so it loops on:

```
Failed to establish a connection. ... FATAL:  password authentication failed for user "authenticator"
postgrest: thread killed
```

and exits `1`. The role exists — `\du` shows it — it just has no password. This
is not in the image; Supabase ships it as a separate `volumes/db/roles.sql`.
Ours is [`initdb/99-roles.sql`](initdb/99-roles.sql), and the `99-` prefix is
load-bearing (the role does not exist until the image's own `00000000000003`
script has run).

**3. `pg_isready` is not a readiness check for this image.** It answers on the
temporary bootstrap server that runs *during* `initdb`, so dependents start
while `migrate.sh` is still installing roles. The compose healthcheck uses an
authenticated `psql -c 'select 1'` instead.

**4. `_realtime` schema must pre-exist.** Realtime's `/app/bin/migrate` connects
with `search_path = _realtime` and does not create the schema. Hence
[`initdb/00-realtime-schema.sql`](initdb/00-realtime-schema.sql).

**5. `docker-entrypoint.sh: ignoring /docker-entrypoint-initdb.d/init-scripts` is
a red herring.** The stock Postgres entrypoint prints it because the path is a
directory; the image's own `migrate.sh` then processes that directory anyway.
Nothing is wrong.

**6. Realtime tenant routing is by `Host` header** — see [R1](#r1--the-negative-control-wrong-tenant).
The failure is a bare `403` with no body.

**7. Tenant `jwt_secret` is encrypted at rest with `DB_ENC_KEY`**, which must be
**exactly 16 characters** (AES-128). This is why the compose file lets
`SEED_SELF_HOST=true` create the tenant row rather than `INSERT`ing it by hand:
a hand-written row needs the secret pre-encrypted with Realtime's own scheme.
Changing `DB_ENC_KEY` later orphans every existing tenant row. If you ever *do*
need to rename the tenant, `UPDATE _realtime.tenants SET external_id = ...`
preserves the already-encrypted secret — far easier than re-encrypting.

**8. `auth.jwt()` does not exist here, and `auth.uid()` will not work for HQ.**
This is the one that will bite the next card. Without GoTrue's migrations the
`auth` schema ships only three functions:

```
 proname
---------
 email
 role
 uid
```

and `auth.uid()` is defined as:

```sql
select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid;
```

Two problems. It reads the **legacy singular** GUC `request.jwt.claim.sub`,
which PostgREST only populates when `PGRST_DB_USE_LEGACY_GUCS=true` (this stack
sets it `false`). And it **casts to `uuid`** — HQ user ids are not necessarily
UUIDs, and a non-UUID `sub` makes it raise rather than return null. Every
copy-pasted policy from Supabase's hosted documentation uses `auth.uid()` or
`auth.jwt()`. **Use `current_setting('request.jwt.claims', true)::json ->> '<claim>'`
directly**, as `sql/spike-fixture.sql` does.

---

## Open question for the operator — NOT answered here

**Which HQ claims map to which grants is out of scope for this spike and was
deliberately left undecided.** The `owner_id = sub` policy in the fixture is a
proof device chosen because it is the simplest predicate that can be observed
either admitting or refusing a request. It is **not** a proposal for HQ's real
policy shape.

That design belongs to the card `sync-jwt-bridge-endpoint` and to the operator.
Concretely, the questions this spike surfaced but did not answer:

- HQ's Users-tab grants are **per-tab / per-feature**, not per-app. Does each
  grant become a claim in the token, or does the token carry only identity while
  the policies join against an HQ-side grants table?
- Row ownership in HQ is frequently **not** a single `owner_id` — a checklist
  submission belongs to a submitter *and* an approver. A single-subject
  predicate does not express that.
- `service_role` is a `BYPASSRLS` god-token. What, if anything, is ever allowed
  to mint one?

---

<!-- ======================================================================= -->
<!-- SEAM: HALF 2 (card W2, RxDB schema and replication) APPENDS BELOW HERE. -->
<!-- Half 1 above is append-only from half 2's side: extend it, do not edit  -->
<!-- it. Everything above stands alone and remains runnable on its own if    -->
<!-- half 2 is never written.                                               -->
<!-- ======================================================================= -->

# Supabase sync spike — runbook, half 2: does RxDB actually replicate over it?

> **This half assumes half 1 above.** It does not repeat the LOCAL-ONLY banner
> or the throwaway-credential warning at the top of this file — they apply to
> everything here too, unchanged. The stack, the fixture table and the Go token
> minter are all half 1's; half 2 adds no service and modifies
> `docker-compose.supabase.yml` by not one byte.

## What this half is

Half 1 proved the *substrate*: a self-hosted Supabase accepts a JWT that HQ's own
Go backend minted, on both PostgREST and Realtime, with RLS discriminating. That
is a proof about Supabase. It is **not** a proof about RxDB.

Night-crew card **W2 `sync-spike-rxdb-replication`** answers the next question:
**does RxDB itself actually replicate over that substrate**, in both directions,
and what does it do when two writers collide?

Every command below was actually run on 2026-07-25 against the stack half 1
leaves running, and every **fact** asserted under it was observed. Nothing here
says "should print". Where output is trimmed, the trim is marked.

**One block in this half is an abridged presentation rather than a verbatim
capture** — the `node smoke.js` output in Step 1a. It is listed with the five
in half 1 under [Integrity of the output blocks](#integrity-of-the-output-blocks),
which is the authoritative list for the whole document.

The artefacts live in [`rxdb/`](rxdb/) and are the thing you run.

---

## Step 0 — the isolated harness, and why it is isolated

```bash
cd .night-crew/qa/spike-supabase/rxdb
npm ci          # first time: npm i rxdb @supabase/supabase-js ws
```

Observed on the cold install:

```
added 78 packages, and audited 79 packages in 20s
found 0 vulnerabilities
```

Installed, and pinned in [`rxdb/package-lock.json`](rxdb/package-lock.json):

| package | version |
|---|---|
| `rxdb` | 17.4.0 |
| `@supabase/supabase-js` | 2.109.0 |
| `ws` | 8.21.1 |

**This directory has its own `package.json` and its own lockfile on purpose.**
The repo-root `package.json` is the Playwright environment every night-crew card
in the repo builds against; adding a dependency to it would change the test
environment for work that has nothing to do with sync. Never run `npm i` from
the repo root for this spike — always `cd` here first.

**Failure this catches:** a spike quietly widening the production dependency
surface. The root `package.json` and `package-lock.json` are byte-identical to
what they were before this card ran; you can check that yourself with
`git diff overnight-20260725..HEAD -- package.json package-lock.json`, which
prints nothing.

`ws` is needed only because Node 20 has no global `WebSocket`
(`node -e "console.log(typeof globalThis.WebSocket)"` → `undefined` on
v20.20.0). A browser needs no such shim.

---

## Step 1 — the gateway-less bridge, and why half 2 needs one

This is the first real finding of half 2, and it is worth understanding before
any proof output.

`@supabase/supabase-js` assumes **one origin fronted by Kong**. Its constructor
derives `<baseUrl>/rest/v1` and `<baseUrl>/realtime/v1` from a single URL and
freezes both:

```js
this.realtimeUrl = new URL("realtime/v1", baseUrl);
...
this.rest = new PostgrestClient(new URL("rest/v1", baseUrl).href, { ... });
```

(read out of `rxdb/node_modules/@supabase/supabase-js/dist/index.mjs`)

Half 1 deliberately did not deploy Kong. So in this stack PostgREST and Realtime
sit on two *different* Docker-assigned host ports and neither serves under those
path prefixes. A stock `createClient()` cannot reach either.

[`rxdb/spike-env.js`](rxdb/spike-env.js) bridges that with the two extension
points supabase-js already exposes — **no fork, no patch, no new service**:

- `global.fetch` — strips the `/rest/v1` prefix so requests land on PostgREST's root;
- `realtime.transport` — a `ws` subclass that re-points host:port at Realtime,
  rewrites the path to `/socket/websocket`, and sets `Host: realtime-dev.localhost`
  so Realtime's tenant lookup resolves (half 1, sharp edge 6).

**This shim is a spike artefact, not a recommendation.** See
["What the shim means for the real migration"](#what-the-shim-means-for-the-real-migration)
at the end of this half.

### Step 1a — prove the bridge before blaming RxDB

```bash
export PATH="/usr/local/go/bin:$PATH"   # spike-env.js shells out to `go run ./mintjwt`
node smoke.js
```

Real output, trimmed to the lines that matter:

```
# stack: rest=46233 realtime=46355 db=46011  run=r1784996960627
token len 184 segments 3
REST error: null
REST rows visible to user-alice: [
  { id: 'note-alice-1', owner_id: 'user-alice', body: 'alice seed row', ... },
  ... (remaining user-alice rows elided) ...
]
realtime status: SUBSCRIBED
FINAL realtime status: SUBSCRIBED
```

Two things are proven here and both matter. `REST error: null` with rows coming
back means PostgREST is reachable through the fetch shim — and **every row
returned is `user-alice`'s**; the `note-bob-1` seed row from
[`sql/spike-fixture.sql`](sql/spike-fixture.sql) is absent, so RLS is still
discriminating through supabase-js exactly as it did through `curl` in half 1.
`SUBSCRIBED` means the Realtime channel actually joined through the transport
shim.

**Failure this catches:** the single most expensive way to waste a night on a
sync spike — spending hours "debugging replication" that is really a client that
never reached the server at all. Run this first; if it does not print
`SUBSCRIBED`, nothing below can work and RxDB is not the reason.

Note the ports are resolved at runtime by `docker compose port`, never
hardcoded, because half 1's compose publishes bare container ports and Docker
reassigns the host side on every `up`.

---

## Step 2 — the collection

One collection, defined in [`rxdb/spike-env.js`](rxdb/spike-env.js), against the
one throwaway `spike_notes` table half 1 created:

```js
export const spikeNotesSchema = {
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id:       { type: 'string', maxLength: 100 },
        owner_id: { type: 'string', maxLength: 100 },
        body:     { type: 'string' }
    },
    required: ['id', 'owner_id', 'body']
};
```

**This is deliberately not HQ's checklist domain model.** W2 answers "does the
mechanism work", not "what is our schema" — modelling the real domain is the
card `sync-rxdb-schema-and-replication`. Using the real model here would have
mixed a mechanism failure and a modelling failure into one indistinguishable
result.

Two omissions from `properties` are load-bearing rather than accidental:

- **`_deleted` is not declared.** RxDB owns that field internally; the plugin
  maps the Postgres column onto it.
- **`_modified` is not declared either.** `replicateSupabase` only round-trips
  `_modified` into the document if the schema declares it, and its
  compare-and-swap (`addDocEqualityToQuery`) only includes `_modified` in the
  `WHERE` clause if the schema declares it. Leaving it out keeps `_modified`
  purely a **server-stamped pull cursor**, which is exactly what half 1's
  trigger makes it. Step 5 shows why that choice is visible in the conflict
  behaviour.

---

## Step 3 — PUSH: local RxDB write appears in Postgres

```bash
node proof-push.js
```

Real output (the RxDB dev-mode banner is elided):

```
# stack: rest=46233 realtime=46355 db=46011  run=r1784996962670
initial replication done
local insert  id=push-r1784996962670
awaitInSync resolved
postgrest verify HTTP 200
row in postgres: [
  {
    "id": "push-r1784996962670",
    "owner_id": "user-alice",
    "body": "written locally in RxDB",
    "_deleted": false,
    "_modified": "2026-07-25T16:29:23.295004+00:00"
  }
]
PUSH: PROVEN — the locally-created RxDB document exists as a Postgres row
```

The document was created with `collection.insert()` against local storage. It
reached Postgres with no HTTP call written by us.

Note `_modified` — nobody sent it. The client does not set it and the plugin
explicitly `delete`s it before every update. It was stamped by the trigger in
[`sql/spike-fixture.sql`](sql/spike-fixture.sql), i.e. **by the Postgres clock**.
That is the property that makes the pull cursor trustworthy across clients with
skewed clocks.

**Failure this catches:** RxDB reporting "in sync" when the push actually
failed. `awaitInSync()` only tells you RxDB has nothing queued — it is perfectly
consistent with a write that was rejected and dropped. That is why the
verification is a **separate HTTP request that does not go through RxDB at
all**. You can run the same check by hand:

```bash
export PATH="/usr/local/go/bin:$PATH"
cd .night-crew/qa/spike-supabase
TOKEN="$(go run ./mintjwt -secret 2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c -sub user-alice)"
REST=$(cd ../../.. && docker compose -p spike-supabase -f docker-compose.supabase.yml port rest 3000 | cut -d: -f2)
curl -s -w '\nHTTP %{http_code}\n' -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:$REST/spike_notes?id=eq.push-r1784996962670&select=*"
```

```
[{"id":"push-r1784996962670","owner_id":"user-alice","body":"written locally in RxDB","_deleted":false,"_modified":"2026-07-25T16:29:23.295004+00:00"}]
HTTP 200
```

---

## Step 4 — PULL: Postgres write reaches a RUNNING client, no restart

PUSH and PULL are two separate scripts on purpose. **A single script that starts
replication and then observes "the data is on both sides" cannot tell you which
direction carried it** — and a one-directional proof presented as bidirectional
is the most common way this class of spike fools itself.

`proof-pull.js` starts replication **once**, at the top, and from that point
never restarts the process, never re-creates the collection, and never calls
`reSync()`. Every write below is made over a raw `fetch` to PostgREST that RxDB
knows nothing about.

```bash
node proof-pull.js
```

```
# stack: rest=46233 realtime=46355 db=46011  run=r1784996964892
initial replication done; local doc count = 7
--- from here on the client is NEVER restarted and reSync() is NEVER called ---
postgrest INSERT HTTP 201
PULL/insert converged in 128 ms -> {"id":"pull-ins-r1784996964892","body":"born in Postgres"}
postgrest UPDATE HTTP 200
PULL/update converged in 129 ms -> {"id":"pull-ins-r1784996964892","body":"edited in Postgres, never touched locally"}
postgrest SOFT-DELETE HTTP 204
PULL/soft-delete converged in 121 ms -> findOne returns null
PULL: PROVEN — insert, update and soft-delete made in Postgres all reached the running RxDB client with no restart and no manual reSync
```

Three cases, because they exercise different code paths:

1. **remote INSERT** — a row the client has never held.
2. **remote UPDATE** — a row the client *does* hold. This is the one that
   actually exercises the checkpoint and the conflict path. **A pull
   implementation can pass case 1 and fail case 2**, which is why they are not
   collapsed into one.
3. **remote soft-delete** (`_deleted = true`) — RxDB replication is soft-delete
   only. A hard `DELETE` is invisible to a pull handler: the row simply stops
   appearing and every offline replica keeps it forever. `findOne` returning
   `null` is the proof the soft-delete propagated as a deletion.

The **121–129 ms** convergence times transcribed above are the evidence that
this came over the Realtime `postgres_changes` stream and not a retry (the
earlier run `r1784996802`, logged in
`.night-crew/runs/2026-07-25-autonomous/timings.log` but not transcribed here,
was faster still: insert 59 ms / update 45 ms / soft-delete 121 ms).
`replicateSupabase`'s `retryTime` defaults to 5000 ms; anything arriving two
orders of magnitude faster than that did not arrive by retry.

**Failure this catches:** a Realtime subscription that is not actually live.
Half 1's proof R3 established that a `phx_join` can reply `{"status":"ok"}` with
a `postgres_changes` id while the subscription has in fact **failed**, the real
error arriving later on a separate `system` frame. A client that only checks the
join reply believes it is subscribed and then silently never receives anything —
which in an offline-first PWA looks exactly like "the other person hasn't
saved yet". This script would hang on that; the 30 s `timeout()` on each case
turns the hang into a visible, attributable failure instead of a green run.

**Both directions are therefore proven, separately and explicitly.**

---

## Step 5 — CONFLICT: what actually happens is *not* last-write-wins

The explore session chose "last-write-wins, no custom conflict handler". This
step does not assume that. It constructs one concurrent-write case and records
what happens.

The case, and why it is shaped this way:

1. one document, agreed on both sides;
2. the client goes offline (replication cancelled);
3. **Postgres is edited first** (T1);
4. **RxDB is edited second** (T2 > T1) — the local write is *strictly later* in
   wall-clock time;
5. the client reconnects, reusing the **same `replicationIdentifier`** so this is
   a genuine reconnect of the same replica rather than a fresh client.

Step 4 is load-bearing. Under genuine last-write-wins the later write — the
local one — must survive.

`proof-lww.js` installs an **observing** conflict handler that delegates every
decision to RxDB's own `defaultConflictHandler` and only *prints* what it was
asked and what it answered. Behaviour is unchanged; making the handler nicer
would have destroyed the answer we were after.

```bash
node proof-lww.js
```

```
# stack: rest=46233 realtime=46355 db=46011  run=r1784996967135
1. agreed state       remote: {"id":"lww-r1784996967135","owner_id":"user-alice","body":"agreed-original","_deleted":false,"_modified":"2026-07-25T16:29:27.604558+00:00"}
2. replication cancelled — the client is now "offline"
3. remote edit HTTP 200 at T1 -> {"id":"lww-r1784996967135",...,"body":"REMOTE-EDIT (written first, T1)","_modified":"2026-07-25T16:29:27.63884+00:00"}
4. local edit at T2=2026-07-25T16:29:29.153Z -> local body now: LOCAL-EDIT (written second, T2)
5. reconnecting...

=========================== OBSERVED ===========================
local  body after reconnect : REMOTE-EDIT (written first, T1)
remote body after reconnect : REMOTE-EDIT (written first, T1)
remote _modified            : 2026-07-25T16:29:27.63884+00:00   (stamped by the Postgres trigger, i.e. the SERVER clock)
local write happened at     : 2026-07-25T16:29:29.153Z   (the CLIENT clock)
replication errors surfaced : 0 []
conflict handler invocations: 1
  - assumedMasterState.body: agreed-original
    newDocumentState.body  : LOCAL-EDIT (written second, T2)    <- the local (later) write
    realMasterState.body   : REMOTE-EDIT (written first, T1)    <- what the server actually held
    handler CHOSE          : REMOTE-EDIT (written first, T1)

VERDICT: the LATER (local) write WAS DISCARDED
VERDICT: the EARLIER (remote) write SURVIVED
NOT last-write-wins. The winner is the MASTER (server) state regardless of which write happened later; no timestamp participated in the decision.
the losing write was discarded SILENTLY — nothing was emitted on error$ for the app to react to.
================================================================
```

Reproduced identically on two independent runs (`r1784996803916` and
`r1784996967135`).

### Which clock decided: none

This is the part worth being precise about, because "last-write-wins" invites the
question "whose clock?" and the honest answer is that **no clock participated at
all**.

The mechanism, read out of
`rxdb/node_modules/rxdb/dist/esm/plugins/replication-supabase/index.js`, is
**optimistic concurrency, not timestamp comparison**. The push handler issues:

```
UPDATE spike_notes SET ... WHERE id = ... AND owner_id = ... AND body = <assumed master value> AND _deleted = ...
```

— a compare-and-swap against the state the client *believed* the server held.
The remote edit had already changed `body`, so zero rows matched, so the plugin
re-fetched the row and handed it to RxDB as a conflict. RxDB then applied its
default conflict handler, which is documented in its own source as:

```js
resolve(i) {
    /**
     * The default conflict handler will always
     * drop the fork state and use the master state instead.
     */
    return i.realMasterState;
}
```

So the resolution rule is **master (server) wins, unconditionally**. `_modified`
never entered the decision — with `_modified` absent from the collection schema
it is not even in the compare-and-swap; it is only the pull cursor. Making the
local clock later, or earlier, or skewed by an hour changes nothing.

### Why this is a finding and not a bug

Server-wins is a perfectly defensible policy. What makes it a finding is the gap
between it and what was assumed, and the fact that **the loss is silent by
default**: nothing is thrown, `error$` emitted nothing, and nothing whatsoever
reaches a crew member unless the application writes code to put it there. On
this configuration, out of the box, the offline edit simply never happened as
far as the user is concerned.

For HQ specifically: a crew member fills in a checklist on a phone with no
signal in the truck; a manager edits the same submission from the office; the
crew member's phone reconnects. **As configured, the crew member's work is
dropped without a word.** That is a product decision, not an implementation
detail, and it is **not fixed in this card** — it is recorded and routed to the
operator. See `.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md`.

#### But "silent" is not "unobservable" — `conflict$` carries the lost write

This distinction is load-bearing for sizing, so it is stated separately rather
than folded into the sentence above. **An earlier draft of this runbook claimed
there was no signal an application could subscribe to. That was wrong, and it
inflated the cost of one of the operator's options.** Corrected here on measured
evidence.

`error$` really does emit zero events — that is reproduced and stands. But
`error$` is not the only observable `replicateSupabase()` returns.
`RxReplicationState` also exposes **`conflict$`**, and in this exact scenario it
**emits one event carrying the discarded local write**.

From the shipped source,
`rxdb/node_modules/rxdb/dist/esm/plugins/replication/index.js`:

```js
// :44
conflict: new Subject() // all conflicts that are reported by the remote on pushes, together with the conflictHandler output
// :51
this.conflict$ = this.subjects.conflict.asObservable();
// :287-289
this.internalReplicationState.events.resolvedConflicts.subscribe(conflict => {
    this.subjects.conflict.next(conflict);
})
```

Measured, not read: re-running the scenario above with
`rep.conflict$.subscribe(...)` added alongside the existing `error$`
subscription in `startReplication()` — a one-line addition to `proof-lww.js`,
run as a throwaway probe and not committed — produced:

```
error$    emissions         : 0 []
conflict$ emissions         : 1
  [B] top-level keys              : [ 'input', 'output' ]
  [B] input.assumedMasterState.body: agreed-original
  [B] input.newDocumentState.body : LOCAL-EDIT (written second, T2)
  [B] input.realMasterState.body  : REMOTE-EDIT (written first, T1)
  [B] output.body                 : REMOTE-EDIT (written first, T1)
```

`input.newDocumentState` **is the discarded document, in full**. So an
application already has everything needed to tell the crew member *"the
temperature you recorded while offline was replaced by the office's edit — here
is what you had typed."* Surfacing a discarded write is therefore
**`replicationState.conflict$.subscribe(...)` plus UI — not new plumbing.**

Two precise caveats, so this is not over-read in the other direction:

- `conflict$` is fed from a plain `Subject`, **not** a `ReplaySubject`. A
  subscriber attached *after* the conflict has already resolved receives
  nothing. Subscribe where the replication state is constructed, as the probe
  did.
- The event fires **per replication state**, not per document write, and carries
  no user-facing text. Deciding *what* to show, *when*, and what the crew member
  can do about it is still real product work. What is cheap is the signal, not
  the experience.

RxDB does support a per-collection custom `conflictHandler` — the hook this
harness used as a read-only probe is the same hook a real policy would be
written into. That hook decides *which value wins*; `conflict$` is how the app
finds out a decision was made. They are complementary, and both are already
there. Sizing is in the design note.

**Failure this catches:** shipping offline-first on the belief that an offline
edit is safe until something newer overwrites it, when reconnecting can discard
it with no error.

---

## Step 6 — licensing and storage: what the real PWA would actually use

**This is a go/no-go input, not trivia**, so it was checked against RxDB's own
current pages rather than taken on anyone's word.

### The answer

**The free path is sufficient. HQ does not need an RxDB premium licence to ship
this.**

- The `rxdb` npm package ships **Apache-2.0** — verified locally, not inferred:
  `rxdb@17.4.0`'s `package.json` says `"license": "Apache-2.0"` and
  `node_modules/rxdb/LICENSE.txt` is the full Apache 2.0 text.
- **Dexie storage (`rxdb/plugins/storage-dexie`) is free**, and is the browser
  storage a real HQ PWA would use on the free tier.
- **IndexedDB storage is premium.** Source: <https://rxdb.info/rx-storage.html>,
  which marks it `👑 IndexedDB` and says to use it *"if you have 👑 premium
  access"*. The same page's own recommendation reads, in full: *"In the Browser:
  Use the LocalStorage storage for simple setup and small build size. For bigger
  datasets, use either the dexie.js storage (free) or the IndexedDB RxStorage if
  you have 👑 premium access which is a bit faster and has a smaller build
  size."* The trailing clause is quoted rather than trimmed because it is the
  page saying, in its own words, exactly what premium buys — speed and build
  size, not capability.
- Corroborated by <https://rxdb.info/premium/>, whose free tier lists *"Default
  RxStorage (Dexie, Memory, LokiJS)"* and whose paid tiers list RxStorage
  **OPFS**, **IndexedDB**, **SQLite**, **Filesystem**, **Worker**,
  **SharedWorker**, **Sharding**, **Memory-Mapped** and the **Localstorage Meta
  Optimizer**.

**The operator's reading was correct**: Dexie is the free browser path,
IndexedDB is premium. Premium buys *performance*, not *capability* — it is an
optimisation available later, not a gate on starting. Note that the free Dexie
storage is itself an IndexedDB-backed store (via dexie.js); the premium
`storage-indexeddb` is a faster direct implementation, not the only way to reach
IndexedDB.

You will see RxDB advertise this in your own console on every run — that banner
is dev-mode's, and it is worth reading once rather than filtering out forever:

```
🤗 Hint: To get the most out of RxDB, check out the Premium Plugins
to get access to faster storages and more professional features: https://rxdb.info/premium/
```

### Is the Supabase replication a supported plugin or an example we would maintain?

**It is a real, shipped plugin.** The BACKLOG's wording is correct. Verified
three ways, deliberately not just by reading the marketing page:

1. It is **in the npm tarball**: `rxdb@17.4.0` exports `./plugins/replication-supabase`
   from its `package.json`, and `dist/esm/plugins/replication-supabase/index.js`
   is 247 lines of real implementation exporting `replicateSupabase()`, with
   TypeScript types in `dist/types/plugins/replication-supabase/`.
2. `rxdb@17.4.0`'s own shipped `README.md` lists it among *"production-ready
   plugins to easily replicate with … Supabase …"*.
3. The docs page <https://rxdb.info/replication-supabase.html> documents the
   import as `import { replicateSupabase } from 'rxdb/plugins/replication-supabase';`
   with no "example", "community" or "beta" caveat.

**But size it as young, not mature.** From `rxdb`'s own shipped `CHANGELOG.md`:

```
### 16.19.0 (4 September 2025)
- ADD [Supabase Replication Plugin](https://rxdb.info/replication-supabase.html) (beta)
```

It entered **as beta in 16.19.0 (4 Sept 2025)**, roughly ten months before the
17.4.0 we are on (13 July 2026), and has been actively worked since — the
changelog also records `FIX(supabase-replication) push.modifier is not used`
(**16.21.0**, 25 Nov 2025) and `feat: replication-supabase querybuilder`
(**16.21.1**, 2 Dec 2025).

So: **we would not be maintaining the replication protocol ourselves** — that
cost is not in the sizing. What we *would* be is an early adopter of a
young plugin on a self-hosted stack its author almost certainly tests against
hosted Supabase. Budget for reading its source when something behaves oddly, as
this card had to. It is 247 readable lines; that is a small, bounded risk, not
an open-ended one.

---

## Step 7 — what a Node-side proof does NOT establish

Naming this limit is part of the deliverable, so it is stated plainly rather
than buried.

**This harness proves the replication protocol.** Specifically: that RxDB's
push handler, pull handler, checkpoint cursor, Realtime change stream and
conflict resolution all function correctly against a self-hosted,
GoTrue-less, Kong-less Supabase using an HQ-minted HS256 token.

**It does not prove, and must not be cited as proving:**

- **Browser storage behaviour.** These runs used `getRxStorageMemory()`. Nothing
  here exercises Dexie/IndexedDB, its quota limits, its eviction behaviour under
  storage pressure, Safari's stricter eviction, or private-browsing mode. A
  memory store cannot fail the way a browser store fails.
- **Persistence across reloads.** Memory storage starts empty every run. The
  fact that the pull proof re-reads everything on start is a property of the
  harness, not evidence about a real client's warm start.
- **Service-worker interaction.** HQ's `sw.js` is a Workbox-generated worker with
  network-first API handling and an offline JSON fallback. How that interacts
  with RxDB's own fetches and with a long-lived Realtime WebSocket is completely
  untested here, and is a genuine risk area — an offline fallback that answers a
  replication request with cached JSON is a plausible and nasty failure.
- **PWA offline semantics.** No airplane mode, no flaky network, no backgrounded
  tab, no iOS killing the page. Step 5's "offline" was `rep.cancel()` — a clean,
  cooperative pause, which is the *friendliest* possible version of going
  offline.
- **Leader election across tabs.** `waitForLeadership: false` was set precisely
  because the harness is one process. In a browser this defaults to `true` and
  multi-tab behaviour is a real, untested surface.
- **Anything about HQ's actual data model,** its volume, its relations, or how
  RLS should be written for it. The fixture is one flat table with a
  single-owner predicate.

The next card should carry a browser-side check for at least the storage and
service-worker items; they are the two most likely to produce a nasty surprise.

---

## What the shim means for the real migration

Step 1's `global.fetch` + `realtime.transport` shim exists only because half 1
deliberately omitted Kong. It is ~25 lines and it worked first try, so it is not
a problem — but it does surface a real decision the migration has to make:

**either** run the Supabase API gateway (Kong) so `supabase-js` sees the single
origin it expects, **or** keep the gateway-less stack and ship a small,
permanent client-construction helper in HQ that points supabase-js at two
separate origins.

Both are viable. The second is what this spike ran on and it is arguably
cleaner — one fewer service, one fewer thing to secure — but it is a **standing
piece of HQ code that tracks a supabase-js internal** (the derived
`rest/v1` / `realtime/v1` paths), and supabase-js could change that in a minor
release. This belongs to the card `sync-jwt-bridge-endpoint` / whichever card
owns client construction, and should be decided rather than inherited by
accident.

---

## Sharp edges, half 2

Numbered continuing from half 1's list.

**9. dev-mode refuses a storage with no schema validator (`DVM1`).** Enabling
`RxDBDevModePlugin` and then calling `createRxDatabase({ storage: getRxStorageMemory() })`
throws:

```
RxError (DVM1): When dev-mode is enabled, your storage must use one of the schema
validators at the top level.
```

The fix is to wrap the storage —
`wrappedValidateAjvStorage({ storage: getRxStorageMemory() })` — as
`spike-env.js` does. Measured, not guessed. Worth knowing before it eats twenty
minutes.

**10. `waitForLeadership` defaults to `true`.** In a browser that is correct: one
tab replicates and the others follow. In a single-process Node script it is
simply a way to make `awaitInitialReplication()` never resolve. All three proofs
set it to `false`, and that is a harness concession — **a real browser client
should leave it at the default.**

**11. `_modified` in the schema is a semantics switch, not a formality.**
Declaring it changes two behaviours at once: the plugin starts round-tripping the
server timestamp into the document, *and* `addDocEqualityToQuery` starts
including `_modified` in the compare-and-swap `WHERE`. The latter makes conflict
detection strictly tighter — any server-side touch, even a semantically
irrelevant one, becomes a conflict. Leaving it out (as here) keeps `_modified` a
pure pull cursor. Decide this deliberately in
`sync-rxdb-schema-and-replication`; do not let it be decided by whether someone
copied the field into the schema.

**12. Node 20 has no global `WebSocket`.** `typeof globalThis.WebSocket` is
`undefined` on v20.20.0, so `@supabase/realtime-js` needs an explicit
`realtime.transport`. This is a harness-only concern — browsers have one — but
it will reappear if HQ ever runs a replication client server-side.

**13. Re-running is safe, by construction and not by cleanup.** Every proof
writes ids prefixed with a fresh per-run token (`push-r<epoch-ms>`, and so on),
so no run can ever collide with a previous run's rows and **no reset step is
needed**. The cost is that `spike_notes` accumulates rows; that is deliberate —
half 1's stack is throwaway and its teardown drops the volume anyway. If you
want it empty, tear the stack down and bring it back up.

---

## Teardown — still a separate, deliberate act

Half 2 adds no service and no volume, so half 1's teardown is still the whole
teardown, and it is still **not run automatically by anything in this
document**. The stack is left running on purpose.

```bash
docker compose -p spike-supabase -f docker-compose.supabase.yml down --volumes
```

The harness's own `node_modules/` is gitignored and reinstallable with
`npm ci` from `rxdb/package-lock.json`; delete it whenever you like.

<!-- ======================================================================= -->
<!-- SEAM: HALF 3 (card C, sync-rxdb-browser-delivery-spike) APPENDS BELOW.  -->
<!-- Halves 1 and 2 above are append-only from half 3's side, with ONE       -->
<!-- sanctioned exception already applied: T-22 decision 53's integrity      -->
<!-- repair, which is confined to the two claim paragraphs and the new       -->
<!-- "Integrity of the output blocks" section. No finding above was revised. -->
<!-- ======================================================================= -->

# Supabase sync spike — runbook, half 3: does any of it work in a real browser?

> **This half assumes halves 1 and 2 above.** It does not repeat the LOCAL-ONLY
> banner or the throwaway-credential warning at the top of this file — they
> apply to everything here too, unchanged. The stack, the fixture table and the
> Go token minter are all half 1's; half 3 adds no service and modifies
> `docker-compose.supabase.yml` by not one byte.

## What this half is

Half 1 proved the **substrate**. Half 2 proved the **replication protocol** — in
Node, on `getRxStorageMemory()`, with `waitForLeadership: false`, and it said so
plainly. Half 2's own closing list of what a Node proof does *not* establish
(`sync-rxdb-feasibility-spike.md:512-537`) is the scope of this half:

1. **Delivery** — how does RxDB even *reach* a browser in a repo whose stated
   constraint is "Static only: No build step, no framework"?
2. **Dexie storage in a real browser** — not memory. Write, reload, is it there?
3. **Service-worker interaction** — HQ's Workbox `sw.js` is network-first for
   `/api/` with an offline JSON fallback. What does that do to replication?
4. **Multi-tab leader election** — at the browser default (`true`), never run.
5. **Token expiry across an offline period** — untouched by W1 and W2.

**Every command in this half was actually run on 2026-07-26**, against the stack
half 1 leaves running, and every **fact** asserted below was observed. **Two
blocks in this half are trimmed, and each says so at the point of use** — the
`build-vendor.sh` capture (npm's `EBADENGINE`/deprecated warnings removed) and
the Playwright summary (each test's spec-path-and-describe prefix abbreviated for
width). **Everything else in half 3 is a verbatim capture**, and no block
anywhere in this half composes output from more than one run.

*(That sentence is narrower than the one this half originally carried, which
claimed flatly that "nothing in half 3 is annotated or abridged" while two blocks
below already carried trim markings. Having narrowed W1's over-strong claim in
[Integrity of the output blocks](#integrity-of-the-output-blocks), writing an
over-strong one of my own a few hundred lines later would have reproduced the
exact defect the rider was invoked to fix.)*

The artefacts live in [`browser/`](browser/) and in [`../../../vendor/`](../../../vendor/).

---

## Step 0 — the delivery mechanism, and why it is a committed bundle

Before anything can be tested in a browser, RxDB has to get there. Three
candidates were considered at slating; the third was chosen.

| # | Candidate | Verdict |
|---|---|---|
| 1 | CDN ESM import (`https://esm.sh/rxdb@17.4.0`) — matches the SortableJS precedent (`workflows.html:172`) | **Rejected.** This is a food-truck PWA whose entire reason for adopting RxDB is offline correctness. Putting the offline data engine behind a CDN the truck cannot reach inverts the point. Workbox precaches **local content-hashed files**; a CDN URL needs runtime caching — a *second* offline story, which Engineering KR3 forbids. |
| 2 | A real build step (esbuild wired into `task sw` / the Dockerfile) | **Rejected for now.** Cleanest long-term, but it breaks a stated project constraint and rewrites the deploy path. **Operator decision, not a card's.** |
| 3 | **Vendored pre-built bundle** | ✅ **Chosen and proven.** |

```bash
bash vendor/build-vendor.sh    # hand-run, on upgrade only
node build-sw.js               # re-precache the new content hash
```

Verbatim, from a **cold** run with `vendor/node_modules` deleted first (npm's
`EBADENGINE` / `deprecated` warnings on the install are the only lines removed,
and this note is that trim being marked):

```
added 77 packages in 35s
==> rxdb              17.4.0
==> @supabase/supabase-js 2.109.0
==> esbuild           0.28.1 (via npx, not a devDependency)

  rxdb.bundle.js  495.0kb

⚡ Done in 1426ms
==> rxdb.bundle.js  506885 bytes (495.0 KiB raw, 145.1 KiB gzip)
==> done. Now run: node build-sw.js   (then commit vendor/ and sw.js together)
```

**The regeneration is deterministic**: that cold rebuild produced a
`vendor/rxdb.bundle.js` **byte-identical** to the committed one
(`git status --porcelain vendor/rxdb.bundle.js` printed nothing). That matters
more than it looks — it means a reviewer can verify the committed binary artefact
against its declared inputs instead of taking it on trust.

```
SW built: 23 files precached (1947.1 KB)
Frontend version: 1.1.0
```

**Four things about this are load-bearing and none are decoration.**

**(a) `npx esbuild@0.28.1`, not a devDependency.** The repo-root `package.json`
and `package-lock.json` are the Playwright environment for every night-crew card
in every worktree. `npx` resolves into npm's own cache and writes nothing into
the repo's dependency graph. The pin is what makes it reproducible without one.
The rxdb / supabase-js **sources** come from `vendor/package.json` +
`vendor/package-lock.json` — again, not the root.

**(b) The output is committed.** That is the whole mechanism. `task prod:deploy`
→ `git pull` → `task sw` → `docker build` never runs a bundler and never needs
npm. **The "no build step" constraint holds.** Regenerating is a deliberate,
hand-run act on upgrade — which is exactly where FORK 4's "smoke test that fails
loudly on upgrade" attaches: `build-vendor.sh` stamps the resolved lockfile
versions into an exported `VENDOR_BUILD` object, and
`specs/leg1-delivery.spec.js` asserts them against RxDB's own `RXDB_VERSION`.

**(c) The Workbox glob is `vendor/*.bundle.js`, NOT `vendor/**`.** The card
asked for `vendor/**`; that would have been a mistake and the measurement says
so. `globIgnores`' `'node_modules/**'` is a **top-level** pattern and does not
match `vendor/node_modules/**`. A bare `vendor/**` sweeps **8,919 files / 67 MB**
of build inputs into the precache manifest. The narrow glob also keeps
`package-lock.json`, `src/*.mjs` and `build-vendor.sh` off the phone.
`'vendor/node_modules/**'` was added to `globIgnores` as belt-and-braces.

**(d) `build-vendor.sh` fails loudly above 5 MiB.** `build-sw.js` sets
`maximumFileSizeToCacheInBytes` to 5 MiB, and Workbox **silently drops** a file
above it. A bundle that silently left the precache would look fine in
development and break offline on the truck. At 506,885 bytes there is ample
headroom; the check exists so the next upgrade cannot cross the line quietly.

### The finding that reshaped the bundle: RxDB dev-mode phones home

**`specs/leg1-delivery.spec.js` asserts the page reaches ZERO external hosts.**
It went red:

```
Error: the page reached ZERO external hosts — no CDN in the offline data path
- Array []
+ Array [
+   "rxdb.info",
+ ]
```

Chased down to
`node_modules/rxdb/dist/esm/plugins/dev-mode/dev-mode-tracking.js`.
`addRxPlugin(RxDBDevModePlugin)` calls `addDevModeTrackingIframe()`, which
appends a hidden 1×1 iframe pointing at
`https://rxdb.info/html/dev-mode-iframe.html?version=17.4.0` to `document.body`,
and hashes `location.host` to pick which marketing link it shows. Its own
comment, at line 36 of the shipped file, reads:

```
  /**
   * Only run this in browser AND localhost AND dev-mode.
   * Make sure this is never used in production by someone.
   */
  if (iframeShown || typeof window === 'undefined' || typeof location === 'undefined' || typeof document === 'undefined'
  // !isLocalHost()
  ) {
```

**The `!isLocalHost()` term of that guard is commented out in the shipped
17.4.0 build.** The only thing that suppresses the iframe is
`hasPremiumFlag()` — a paid flag.

So `dev-mode` and its mandatory `validate-ajv` wrapper were **removed from the
vendored entry entirely**. Measured consequences:

| | with dev-mode | without | delta |
|---|---|---|---|
| raw | 708,556 B | 506,885 B | −201,671 B |
| gzip | 205,765 B | 148,591 B | −57,174 B |
| third-party hosts contacted | 1 (`rxdb.info`) | **0** | — |

Two knock-on effects, both real and both worth knowing before
`sync-rxdb-schema-and-replication` starts:

- **W2 sharp edge 9 stops applying.** `wrappedValidateAjvStorage` was only
  required *because* dev-mode was on. Plain `getRxStorageDexie()` is used
  directly.
- **`ignoreDuplicate: true` becomes illegal.** RxDB throws `RxError (DB9)` —
  *"ignoreDuplicate is only allowed in dev-mode and must never be used in
  production"*. W2's `spike-env.js makeLocalDb` sets it and never hit this,
  because it had dev-mode on. A page that opens the same database twice must now
  hold its own handle instead.
- **Cost, stated honestly:** no schema/typing checks, and RxDB errors arrive as
  bare codes with an `rxdb.info` doc link instead of prose. A developer who
  wants them adds `rxdb/plugins/dev-mode` to `vendor/src/rxdb-hq-entry.mjs`,
  runs `build-vendor.sh`, and **must not commit the result**.

---

## Step 1 — run the browser harness

```bash
export PATH="/usr/local/go/bin:$PATH"    # the harness shells out to `go run ./mintjwt`
cd .night-crew/qa/spike-supabase/browser
npx playwright test -c playwright.spike.config.js
```

**No `npm install` step.** This directory has a `package.json` with **no
dependencies at all**: `@playwright/test` resolves up the tree to the repo-root
`node_modules`, and `serve.mjs` is deliberately zero-dependency. The root
`package.json` / `package-lock.json` are byte-untouched.

2026-07-26, 1-min load average **8.58** at start and **30.39** at end — this box
runs ~13 unrelated containers and a second night-crew card was executing
concurrently, so this is a **loaded**, not a quiet, measurement.

The block below is the real run with **one marked trim**: each test's
`specs/legN-….spec.js:NN:5 › <full describe title> ›` prefix is abbreviated to
`legN ›` for width. Nothing else is altered, and no timing or count is composed
from a different run.

```
Running 11 tests using 1 worker

  ✓   1 leg1 › the bundle loads via a plain <script type="module"> from a local path (577ms)
  ✓   2 leg1 › the bundle exports every symbol a real client would need (300ms)
  ✓   3 leg2 › write, reload, and the data is still there (645ms)
  ✓   4 leg2 › a second, independent page in a fresh context sees the same store (371ms)
  ✓   5 leg3 › 3a — the vendored bundle is served from the Workbox precache while OFFLINE (1.1s)
  ✓   6 leg3 › 3b — the /api/ offline fallback is real: 503 {"error":"offline"} (1.1s)
  ✓   7 leg3 › 3c — THE TRAP: a same-origin replication URL under /api/ is answered from STALE CACHE while offline (3.3s)
  ✓   8 leg3 › 3d — replication itself works with the SW controlling, over a non-/api/ same-origin path (2.1s)
  ✓   9 leg3 › 3e — a long-lived WebSocket UNDER /api/ is untouched by the service worker (2.5s)
  ✓  10 leg4 › exactly one of two tabs leads, and the survivor takes over when it closes (2.4s)
  ✓  11 leg5 › expiry while offline is recoverable by swapping the token, with no data loss (1.5m)

  11 passed (1.8m)
```

### Why the harness serves its own origin

`browser/serve.mjs` is a ~200-line zero-dependency static server that publishes
**the same PostgREST under two same-origin paths**:

```
  /rest/v1/*        -> PostgREST   (does NOT match HQ sw.js's /\/api\//)
  /api/v1/rest/v1/* -> PostgREST   (DOES match it)
```

and `supabase-js` chooses between them purely from the base URL it is handed,
because it derives `<url>/rest/v1` internally:

```js
createClient('http://127.0.0.1:PORT')          // -> /rest/v1/...
createClient('http://127.0.0.1:PORT/api/v1')   // -> /api/v1/rest/v1/...
```

This matters twice.

**First, it removes the shim.** W2's Node harness needed a `global.fetch`
override to strip supabase-js's `/rest/v1` prefix, because W1 deliberately
deployed no Kong (`spike-env.js` §3). **Fronting PostgREST on HQ's own origin
makes that shim unnecessary** — the client's assumed path *becomes* the real
path. That is a genuinely useful finding for the real migration: the
"gateway-less client shim" of FORK 4 shrinks to a base-URL choice on the REST
side.

**Second, and this is the point:** a cross-origin PostgREST on `127.0.0.1:46233`
never matches `/\/api\//` at all. A naive cross-origin harness would have
"passed" the service-worker leg **by never touching the trap.** That is exactly
the false green this half exists to avoid.

---

## Step 2 — leg 2: Dexie survives a reload

`specs/leg2-dexie-reload.spec.js`. Opens a real
`getRxStorageDexie()` database (asserted: `db.storage.name === 'dexie'`, so a
silent fall-back to memory cannot pass), inserts three documents, confirms the
browser's own `indexedDB.databases()` lists the store, then does a **real
`page.reload()`** — not `db.close()` — and re-opens.

**PROVEN.** All three documents come back with the right bodies, and a document
id that was never written still returns `null`. That negative half is not
ceremony: without it a leg that returned everything would look identical to a
leg that proved persistence.

One thing recorded rather than asserted: a **fresh Playwright browser context**
sees an empty store (`[]`). That measures Playwright's storage isolation, not
RxDB's durability, so it is logged and not turned into a pass/fail.

**What this does NOT establish, stated plainly:** quota limits, eviction under
storage pressure, Safari's stricter eviction, and private browsing are all
untested. Chromium's Dexie backend is not iOS Safari's. See the verdict.

---

## Step 3 — leg 3: the service worker, and the trap that is real

`specs/leg3-service-worker.spec.js`, run under
`browser/playwright.spike.config.js`.

> ### ⚠ The repo-wide `serviceWorkers: 'block'` was NOT touched
>
> `playwright.config.js:60` blocks service workers for the whole suite, and that
> setting is **load-bearing for 500+ existing tests** — HQ's Workbox SW
> precaches every page and would serve stale HTML into a suite that mutates
> state at every step. This leg is proved in a **separate config** that enables
> service workers for these specs and these specs only. Nothing in `task test`
> reads it.

Every sub-leg below registers HQ's **real** `/sw.js` and waits for it to
actually control the page (`navigator.serviceWorker.controller` non-null) before
measuring anything.

### 3a — the offline data engine is itself available offline ✅

`GET /vendor/rxdb.bundle.js` with the network cut returns **HTTP 200 and the
identical byte count** as online. This is the payoff of the vendored-bundle
decision: RxDB is a precached local asset, so it is there exactly when it is
needed. A CDN import could not do this without a second offline story.

### 3b — the fallback is real ✅

A never-before-fetched `/api/v1/spike-ping?never-cached=…` with the network cut:

```
status 503, content-type application/json, body {"error":"offline"}
```

Workbox reached `handlerDidError` because `NetworkFirst` had no cache entry to
fall back to. Confirmed, and it is the *benign* half of the story.

### 3c — 🛑 THE TRAP, and it is the nasty variant

The feasibility note warned about "an offline fallback answering a replication
request with cached JSON". The reality is worse than the 503, because
`NetworkFirst` tries **network → cache → `handlerDidError`, in that order**.

The leg: prime the cache with one successful, well-formed pull at a stable
replication-shaped URL under `/api/`; insert a new row **out of band**, straight
at PostgREST, so the browser cannot know; cut the network; ask for the same URL.

Verbatim:

```
[leg3c] under /api/ while offline: {"status":200,"contentType":"application/json; charset=utf-8",
  "isArray":true,"count":26,"bodyStart":"[{\"id\":\"note-alice-1\",\"owner_id\":\"user-alice\",
  \"body\":\"alice seed row\",\"_deleted\":false,\"_modified\":\"2026-07-25T15:03:07."}
[leg3c] NOT under /api/ while offline: {"threw":true,"error":"TypeError: Failed to fetch"}
[leg3c] VERDICT INPUT — silent-stale-200 under /api/ while offline: true
```

**Read that carefully.** Offline, under `/api/`, HQ's service worker answered a
replication pull with **HTTP 200, `application/json`, a well-formed array of
rows** — stale, missing the row written out of band, and **indistinguishable
from a fresh pull.** RxDB has no way to tell. It would accept the documents and
**advance `_modified`/`id` checkpoint past data it never saw**, and the missing
row would never be pulled again.

The contrast line is the other half of the finding: the **same query shape not
under `/api/`** fails honestly with `TypeError: Failed to fetch`, which RxDB
handles correctly by retrying.

**This is a hard constraint on the real migration, not a curiosity:**

> **The Supabase replication endpoint must NOT be mounted under a path matching
> HQ's `sw.js` runtime route (`/\/api\//`).** Mount it at `/rest/v1/*`,
> `/sync/*`, or cross-origin — anywhere the Workbox `NetworkFirst` route does
> not claim. If it ever must live under `/api/`, `build-sw.js` has to exclude
> that sub-path from the runtime route **explicitly**, and that exclusion needs
> a regression test, because the failure is silent.

### 3d — replication works fine with the SW in the middle ✅

Full RxDB → PostgREST push over the non-`/api/` same-origin path with the
service worker controlling: `awaitInitialReplication` → `ok`, `awaitInSync` →
`in-sync`, **0 replication errors**, and the row verified over an independent
request straight at PostgREST's own host port — not through the page, not
through the proxy, not through the SW, and not through RxDB's own opinion of
whether it is in sync.

**The service worker is not a problem for replication. The `/api/` route is.**

### 3e — a long-lived WebSocket under `/api/` is untouched ✅

To keep this a question about the *service worker* rather than about Realtime's
vhost handling, `serve.mjs` completes a real RFC 6455 handshake itself at
`/api/v1/spike-ws` — a path that matches `/\/api\//` exactly.

```
[leg3e] /api/-mounted WebSocket with SW controlling, online:  {"verdict":"open"}
[leg3e] /api/-mounted WebSocket with SW controlling, offline: {"verdict":"error"}
[leg3e] RECORDED (not asserted) — browser -> self-hosted Realtime vhost: {"verdict":"error"}
```

A WebSocket handshake is not a fetch the service worker can answer, so it opens
normally online and fails as a **transport error** offline. It can never become
the 503 JSON fallback and can never be served from `api-cache`.

**Note the asymmetry, because it is the operationally important part:** the
WebSocket half of replication fails **loudly** where the HTTP half (3c) fails
**silently**.

The third line is **recorded, not asserted**: a direct browser handshake to
self-hosted Realtime via the `realtime-dev.localhost` vhost did not connect in
Chromium. That is W1's substrate surface, not this card's — W1 already proved
Realtime over a Go-minted token with `rtwatch`, and a red here would attribute
their finding to us. **It is not established that a browser can reach
self-hosted Realtime directly**, and `sync-rxdb-schema-and-replication` should
budget for it. Nothing in this half depends on it.

---

## Step 4 — leg 4: multi-tab leader election at the browser default ✅

`specs/leg4-leader-election.spec.js`. Two pages in **one** browser context (two
tabs of one profile — two *contexts* would be two profiles and would prove
nothing), same Dexie database name, both starting replication with
`waitForLeadership` **left undefined** so `replicateSupabase`'s own default
(`true`) applies. That is the configuration nothing has ever run.

```
[leg4] leadership after both tabs started: A=true B=false
[leg4] follower replication state before handover: {"started":false,"errorCount":0,"kinds":[],"last":null}
[leg4] survivor took over: true after 47 ms
[leg4] survivor replication state 552 ms after handover: {"started":true,"errorCount":0,"kinds":[],"last":null}
```

Three facts, and the third is the one that mattered:

1. **Exactly one tab leads.** The follower's replication `started` is `false` —
   leader election is doing real work, not decorating. Two tabs do not
   double-push.
2. **Handover is fast.** 47 ms from closing the leader tab (65 ms and 87 ms on
   two earlier runs).
3. **The survivor does not just win the election, it actually begins
   replicating** — `started: true` 552 ms after handover. "Leader" and
   "replicating" are two different facts, and asserting only the first would
   have missed a crew member's remaining tab sitting there silently not syncing.

**W2's `waitForLeadership: false` was a harness concession and can be dropped.
The browser default is correct and works.**

---

## Step 5 — leg 5: token expiry across an offline period ✅ (with a sharp edge)

`specs/leg5-token-expiry-offline.spec.js`. A real 20 s TTL is elapsed, not
mocked.

### 🛑 First, the sharp edge that nearly produced a false green

The first version of this leg used a 20 s TTL and waited 28 s offline. **The
write pushed successfully.** That was not RxDB being clever — **PostgREST
v12.2.12 accepts a token past its `exp` for roughly 30 more seconds.**

Measured directly against W1's stack on 2026-07-26. Each probe uses a **fresh,
never-before-seen token**, so this is not a JWT-cache artefact:

```bash
for d in -5s -15s -25s -29s -30s -31s -35s -45s -60s; do
  T=$(go run ./mintjwt -secret $SEC -sub user-alice -ttl $d)
  curl -s -o /dev/null -w "exp offset $d -> HTTP %{http_code}\n" \
    -H "Authorization: Bearer $T" "http://127.0.0.1:$RESTP/spike_notes?limit=1"
done
```

```
exp offset -5s -> HTTP 200
exp offset -15s -> HTTP 200
exp offset -25s -> HTTP 200
exp offset -29s -> HTTP 401
exp offset -30s -> HTTP 401
exp offset -31s -> HTTP 401
exp offset -35s -> HTTP 401
exp offset -45s -> HTTP 401
exp offset -60s -> HTTP 401
```

and the rejection is the expected one:

```
{"code":"PGRST301","details":null,"hint":null,"message":"JWT expired"} | HTTP 401
```

**`exp` IS enforced — W1's finding stands — but with ~30 seconds of clock-skew
leeway.** Two consequences:

- **Any HQ token-refresh design that treats `exp` as a hard edge is wrong by
  half a minute.** In practice this is a small safety margin in HQ's favour, but
  it must be *known* rather than discovered.
- **Any test that waits less than ~30 s past `exp` is measuring the leeway, not
  the expiry.** The leg now waits 65 s offline against a 20 s TTL.

### The leg itself

Write locally with a valid token and sync. Go offline. Write again. Let the
token die out there. Come back online.

```
[leg5] replication state with an EXPIRED token:
  {"started":true,"errorCount":15,"kinds":["jwt-expired","network"],
   "last":{"kind":"jwt-expired","head":" RxDB Error-Code: RC_PUSH. …"}}
```

Then swap the token **without restarting replication** —
`supabase-js` wires `fetchWithAuth` to `_getAccessToken()` and calls the
`accessToken` callback on **every request** (`dist/index.mjs:673`), so a fresh
token can be handed in while the RxDB database stays open:

```
[leg5] awaitInSync after token swap: in-sync
[leg5] offline-written row in Postgres after recovery:
  [{"id":"tok-c1785031215539-offline","owner_id":"user-alice",
    "body":"written OFFLINE, after the token expired","_deleted":false,
    "_modified":"2026-07-26T02:01:45.632156+00:00"}]
```

**Four findings:**

1. **Replication does not die.** It retries indefinitely on `RC_PUSH`. There is
   no terminal state to recover *from*.
2. **The offline write is never lost.** It sits in Dexie through the whole
   expiry, through 15 failed pushes, and pushes successfully afterwards.
3. **Recovery is a token swap, not a re-login.** The RxDB database is never torn
   down, so nothing queued is at risk. **This is what the real client-construction
   helper (FORK 4 / T-22 decision 51) must expose**: a mutable token source, not
   a token baked into the client at construction.
4. **The error is classifiable and must be classified.** `jwt-expired` and
   `network` are different answers demanding different UI, and RxDB surfaces
   both through the same `error$` with the same `RC_PUSH` code. The harness
   pattern-matches `PGRST301|JWT expired` vs `Failed to fetch`; **HQ will need
   the same discrimination**, because "you are offline" and "please sign in
   again" are not interchangeable messages to a crew member on a truck.

> **Not established:** Realtime's own token lifecycle. `supabase-js` requires a
> separate `setAuth()` on the Realtime socket when the token changes, and the
> browser could not reach self-hosted Realtime directly here (3e). The
> **HTTP** half of replication recovers on a token swap; the **WebSocket** half
> is untested and `sync-rxdb-schema-and-replication` must budget for it.

---

## Sharp edges (half 3)

**1. `vendor/**` is the wrong Workbox glob.** `globIgnores`' `'node_modules/**'`
is top-level only. A bare `vendor/**` precaches 8,919 files / 67 MB. Use
`vendor/*.bundle.js`.

**2. RxDB dev-mode phones home to `rxdb.info` on ANY host.** The
`!isLocalHost()` guard is commented out in shipped 17.4.0. Keep dev-mode out of
any bundle that ships.

**3. Dropping dev-mode makes `ignoreDuplicate: true` throw DB9.** It is
"only allowed in dev-mode". W2's harness sets it.

**4. Without dev-mode, RxDB errors are bare codes.** `DB9`, `RC_PUSH`, `DVM1`
with a doc URL and no prose. Budget a few minutes per unfamiliar code, or
temporarily rebuild the bundle with dev-mode **without committing it**.

**5. HQ's `sw.js` will answer a replication pull with stale cached JSON** if the
endpoint sits under `/api/`. Silent. See 3c.

**6. PostgREST tolerates ~30 s of `exp` skew.** See leg 5.

**7. Service workers cannot be tested in the main Playwright suite.**
`playwright.config.js:60` blocks them and that is correct. Anything
SW-dependent needs its own config, as `browser/playwright.spike.config.js` is.

**8. A cross-origin spike harness would have missed the whole service-worker
finding.** Same-origin is not a convenience here; it is the test condition.

**9. `spike_notes` keeps accumulating.** Half 2's sharp edge 13 still applies,
and it is why leg 5 asserts **by document id** rather than by row count. A count
assertion in this harness tests the fixture's history, not the code.

---

## Teardown — still a separate, deliberate act

Half 3 adds no service and no volume, so half 1's teardown is still the whole
teardown, and it is still **not run automatically by anything in this
document**. The stack is left running on purpose.

```bash
docker compose -p spike-supabase -f docker-compose.supabase.yml down --volumes
```

`browser/` installs nothing; there is no `node_modules` under it to delete.
`vendor/node_modules/` is gitignored and reinstallable with `npm ci` from
`vendor/package-lock.json`; delete it whenever you like — the **committed
bundle does not need it**, which is the entire point.
