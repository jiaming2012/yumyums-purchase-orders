# Spikes — sync-coordinates-provisioning

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> This target repo has no `usm/roadmap.txt` story map (the layout the
> `night-crew spikes gate/run` verbs read), so those verbs cannot drive here —
> the established hand-run convention (activity-5, Activities A + G, and this
> activity's five sibling ledgers). This ledger and the runnable scripts that
> ARE the verdict (B-345) are authored to the skill's paths anyway.
>
> **Substrate:** the committed LOCAL `spike-supabase` compose project only
> (reconcile mode, never `--fresh`), against the **built** artifacts —
> `supabase/migrations/*.sql` + `supabase/seed.sql` applied by the committed
> `supabase/verify/lib.sh` helpers. **Plus, new for this goal:** the real Go
> backend (`go run ./cmd/server`, the shipped `main.go` wiring — real
> `auth.Middleware`, real `TokenHandler`, real `ProxyHandler`) on a
> spike-owned test database `hq_test_spike_prov` on **:5434** (`hqtest`, the
> TEST-ONLY container — the same coordinates every Playwright run uses, via
> `scripts/reset-e2e-db.js` and its name guard; a spike-owned TEST_DB_NAME so
> a concurrent `task test` on `hq_test_e2e` cannot collide). Never :5433, no
> hosted project. This is the first Activity B spike that runs the HQ door
> itself rather than talking straight to PostgREST — because this card's whole
> premise IS the door.

## The goal, and which legs need a spike

The card (roadmap Activity B, authored at the attended sitting of 2026-09-06,
ledger decision 180 — the operator's "land provisioning first" call): wire the
two coordinates that arm sync on a real device, so the replicas actually
start. `SYNC_KEY` (`hq_marketing_sync_v1`) has exactly one occurrence in the
tree — its own declaration at `scan-page.js:27` — so `readJson(SYNC_KEY)` is
always null, `startSync` never runs, `campaignPolicy.attach()` never runs,
every local collection stays empty, and close-bar leg 3 / Q-KR1 cannot be
attested. done_when: (1) a provisioned device holds real codes offline,
provisioned through the shipped path; (2) `campaignPolicy.attached()` true +
`size()` non-zero against the real source; (3) B-438 discharged —
`policy_unresolved` records what its DDL comment says, no `setCampaignPolicy`
stub; (4) B-439 discharged — `unresolved()` returns to `false` after a
failure + full recovery.

The roadmap guessed this card "likely thin, since the mechanism is threaded
and both endpoints are landed", and named two open questions: bearer
lifetime/refresh, and where provisioning triggers from (page init vs login).
Recon for this ledger found the thinness guess is **wrong in two places the
slate must price**, and both became spike legs:

- **The client bearer is inert at the door.** `internal/sync/proxy.go`
  DISCARDS the caller's `Authorization` and substitutes a token minted
  per-request for the SESSION user (`poolMinter` → `MintForUser`,
  `DefaultTokenTTL` 15 min). If that holds when run, "bearer
  lifetime/refresh" dissolves: the credential that arms sync is the
  `hq_session` cookie (whose `sessions` row is inserted with NO `expires_at`
  — `auth/service.go:73` — and whose middleware treats NULL as never-expires),
  and no client-side refresh machinery exists to build. Spikes 01 + 02
  measure this rather than trust the comment.
- **The push replica is ALSO unwired, and its device identity is
  constrained.** `startScanAttemptsReplica` (push-replication.js:445) has
  zero callers outside its own module — `startSync` starts only the three
  pull replicas. And RLS on `scan_attempts`
  (`20260904000100_qr_attribution_spine.sql`) requires
  `device_id = JWT sub` — which through the door is the **session user's id**
  (the substituted token's sub), not anything the client picks. A card that
  "arms sync" without starting the push, or that invents a device id, either
  strands every attempt locally forever or lands in the F-2 throw-retry
  poison class. Spike 03 measures both edges; the scope call is recorded
  under What the spikes decided once run.

What is already proven and NOT re-spiked here: the pull mechanism itself with
the GAP-1 keyset checkpoint (sibling ledgers `rxdb-pull-replica`,
`requires-online-replication`), the fail-closed policy seam and the
`policy_unresolved` discriminator landing post-migration (sibling
`refusal-holds-before-sync`, GAP-1 validated), the window observability on
the shipped campaigns handle *pre-ready* (sibling spike 01 — this goal's
spike 04 measures the POST-ready recovery edge, which that spike did not),
and the proxy's own room-level behavior against live Realtime/PostgREST
(`internal/sync/proxy_live_test.go`, G6-verified) — but those tests mint
their own spike JWTs and never run the shipped *client* modules through the
door, which is exactly the seam this card lives on.

Also stated, not spiked: **the unprovisioned-deploy degenerate case stays.**
On a deploy where `/sync/token` answers 503 (secret unset) or the user has no
session, no sync starts, `attached()` stays false, and a genuinely-unknown
override still files `policy_unresolved=false` — that is B-436-adjacent
territory (no policy source at all), outside this card's four clauses, and
carrying it needs no new measurement. Likewise the browser's
`credentials: 'same-origin'` default (the shipped `fetchImpl` is bare
`fetch`, so the cookie rides along in a real browser) is a documented
platform behavior the card's own Playwright e2e will exercise natively; the
Node spikes attach the cookie header explicitly and say so.

## Spike: coordinates-through-the-door

- proves: the shipped pull replicas, given `restUrl = <origin>/sync/rest` and
  ANY truthy bearer, deliver the seeded rows end-to-end through the real
  backend with only the session cookie as credential — and the client bearer
  is provably inert at that door. Enumerates, as a set and not a sample
  (B-216), the auth matrix: (a) valid cookie + garbage bearer → every pull
  200 AND row counts non-zero for codes, offers, campaigns (the silent-empty
  failure mode — 200 with RLS-filtered nothing — is asserted against, not
  assumed away); (b) no cookie + a freshly-minted VALID bridge token in
  `Authorization` → 401 (identity is the session only; the door never honors
  a client bearer); (c) no cookie + no bearer → 401; (d) control, direct to
  PostgREST with the garbage bearer → non-200 (PostgREST does validate, so
  (a)'s 200 can only be the proxy's substitution). Then the full startSync
  shape: all three shipped replicas (`startCodesReplica`, `startOffersReplica`,
  `startCampaignsReplica` with `createCampaignPolicySource` attached in
  scan-page order) reach `awaitInitialReplication`, and the policy source
  reports `attached() === true`, `size() > 0`, `unresolved() === false` —
  done_when clause 2's premise, against the real source, nothing stubbed.
  Falsified if the URL composition breaks (`buildPullUrl`'s
  `${restUrl}/${table}` vs the proxy's `/sync/rest` strip), if RLS filters
  authenticated pulls to empty, or if any matrix row disagrees.
- plan: source `supabase/verify/lib.sh`; substrate up (reconcile);
  `reset_bare` + `apply_all`. Reset `hq_test_spike_prov` via
  `scripts/reset-e2e-db.js` (TEST_DB_NAME guard-checked), start the real
  server (`go run ./cmd/server`) with `HQ_SYNC_JWT_SECRET` = the compose
  file's committed throwaway secret and `HQ_SYNC_REST_URL` = the substrate's
  PostgREST, login as the bootstrapped superadmin
  (`POST /api/v1/auth/login`), then a Node client importing the SHIPPED
  `marketing/sync/replicas.js` runs the matrix + the three-replica leg with a
  fetchImpl that attaches the cookie header.
- script: .night-crew/spikes/activity-b-offline-first-replica/sync-coordinates-provisioning/01-coordinates-through-the-door.sh

## Spike: mint-supplies-the-coordinates

- proves: everything provisioning must write is derivable at **page init**
  from the session alone — settling the card's trigger-point question by
  measurement. With only the cookie: `POST /api/v1/sync/token` → 200
  `{token, expires_at, sub, role, grants}`, `sub` equal to the logged-in
  user's id (the push replica's `deviceId` coordinate — spike 03's premise),
  `exp − iat` = exactly the 15-minute `DefaultTokenTTL`, and the minted token
  accepted DIRECTLY by PostgREST (the coordinate is a real substrate
  credential, not just a truthy placeholder — it keeps the direct-substrate
  and future-Realtime paths open even though spike 01 proves it inert at the
  door). Fail-closed degradations enumerated as a set: no cookie → 401; a
  second server instance with `HQ_SYNC_JWT_SECRET` unset → token endpoint
  503 `sync_bridge_not_configured`; with `HQ_SYNC_REST_URL` unset → the
  `/sync/rest` room refuses (status enumerated) rather than guessing an
  upstream. Longevity run-proven, not code-read: the spike session's
  `sessions` row shows `expires_at IS NULL`, and after aging `created_at` by
  30 days via psql the same cookie still mints — so a provisioned device
  holds as long as its session row exists, and re-visits re-provision
  idempotently (`syncHandles` guard). Also enumerates the `SYNC_KEY` writer
  set in the tree (expected: exactly the one declaration — nothing conflicts
  with the card's writer). Falsified if any envelope field is missing/wrong,
  if the TTL differs from the constant, if an aged session stops minting, or
  if a degradation path fails open.
- plan: same stack as spike 01 (substrate + server + login). curl + node for
  the envelope and JWT decode; psql (via the test container's `hqtest` role)
  for the NULL check and the aging UPDATE — scoped `WHERE token_hash =` the
  spike's own session, on the spike-owned database only; a second
  `go run ./cmd/server` on another port with the two env vars unset for the
  fail-closed legs.
- script: .night-crew/spikes/activity-b-offline-first-replica/sync-coordinates-provisioning/02-mint-supplies-the-coordinates.sh

## Spike: push-lands-as-the-session-user

- proves: the shipped push replica works through the door under the identity
  constraint the door imposes — and what getting the identity wrong costs.
  (a) Enumerates the caller set of `startScanAttemptsReplica` in the tree
  (expected: zero outside its own module — the scope finding priced into the
  slate, not discovered on card night). (b) With `deviceId` = the mint
  envelope's `sub`, the shipped `startScanAttemptsReplica` +
  `enqueueAttempt` drain a legitimate redeem attempt on a freshly-inserted
  live code (build-fact 5 of the sibling ledger: never reuse a seeded code)
  through `<origin>/sync/rest` — the RPC redeem call composes with the
  prefix strip, the attempt lands `accepted`, and the code reads back
  redeemed. (c) With `deviceId` = anything else (`rogue-device`), the same
  path draws the RLS with-check refusal — status enumerated — and the
  shipped handler THROWS on it, i.e. the F-2 throw-retry head-of-line poison
  class, so "derive deviceId from the mint sub" is a correctness constraint,
  not a style choice. Falsified if the legit-identity leg cannot land
  through the door, or if the rogue leg somehow lands (RLS not actually
  binding through the proxy would be a security finding, escalated
  immediately).
- plan: same stack as spike 01. psql inserts the fresh live code (spike-local
  data on the throwaway substrate, committed migrations untouched); Node
  client imports the SHIPPED `push-replication.js`, runs leg (b) with the
  cookie-attaching fetchImpl and `deviceId` from spike 02's mint, then leg
  (c) on a second in-memory db instance; psql reads the landed rows back.
- script: .night-crew/spikes/activity-b-offline-first-replica/sync-coordinates-provisioning/03-push-lands-as-the-session-user.sh

## Spike: recovery-edge-for-b439

- proves: the B-439 fix has a real edge to key on. done_when clause 4 wants
  `unresolved()` back to `false` after a pull failure + full recovery — the
  shipped latch (`replicas.js` `createCampaignPolicySource`) clears
  `lastError` exactly once, at first-ready, and the sibling spike measured
  only the PRE-ready window. What has never been run: after `ready = true`,
  a 503 blip, then recovery — which shipped observable on the handle marks
  the SUCCESSFUL post-ready pull cycle so the latch can clear? Enumerates
  the candidate set (`error$`, `active$`, `remoteEvents$`, `awaitInSync()`
  re-resolution) through both recovery shapes: recovery-with-docs (a row
  changed while erroring — something to deliver) and **recovery-empty** (the
  worst case for an observable-based fix: the recovered pull returns zero
  new rows — if nothing observable fires there, the fix must move to the
  pull-handler seam `replicas.js` itself constructs, a different card shape
  with its own cost). Each candidate is measured in BOTH phases — fires on
  success, and does NOT fire (or is distinguishable) while erroring — and
  the verdict names which edges are usable per shape. Falsified if no
  shipped signal distinguishes a successful post-ready cycle in the
  recovery-empty shape AND the handler-seam tap (measured in the same run as
  the fallback shape) also fails to observe it.
- plan: substrate as spike 01 (no HQ server needed — this is a
  client-observable question; direct PostgREST with a mintjwt device token,
  the sibling spike's exact harness pattern). Node client: shipped campaigns
  replica + policy source on memory storage, gated fetch 200→503→200,
  `reSync()` as the nudge the page owns, one leg with a psql row touch
  mid-error, one leg without; every emission timestamped.
- script: .night-crew/spikes/activity-b-offline-first-replica/sync-coordinates-provisioning/04-recovery-edge-for-b439.sh

## Verdicts (run 2026-09-06, hand-run per the no-story-map convention)

- **coordinates-through-the-door: passed** — exit 0 on the second attempt;
  the first attempt COULD-NOT-RUN on a harness defect (correction 1), zero
  premise changes.
- **mint-supplies-the-coordinates: passed** — exit 0, first run, no
  corrections.
- **push-lands-as-the-session-user: passed** — exit 0 on the third attempt;
  two harness defects (corrections 2 and 3), zero premise changes.
- **recovery-edge-for-b439: passed** — exit 0, first run, no corrections.

### What the spikes decided

**1 — the session cookie is the credential; the client bearer is inert; the
bearer-lifetime/refresh question DISSOLVES.** The auth matrix, measured
against the real server: cookie + garbage bearer → 200 with rows; no cookie +
a freshly-minted VALID bridge token → 401; no cookie + nothing → 401; the
same garbage bearer direct to PostgREST → 401 `PGRST301` (the control — the
door's 200 can only be per-request substitution). All three shipped replicas
delivered through `<origin>/sync/rest` (codes 4, offers 4, campaigns 2 —
non-zero asserted, the silent-empty mode did not occur), `buildPullUrl`'s
`${restUrl}/${table}` composes with the proxy's prefix strip unmodified, the
policy source reached `attached=true size=2 unresolved=false` (done_when
clause 2's premise, nothing stubbed), and the §5.1 clock calibrated through
the proxy hop (Date header survives). **There is no client refresh problem to
build**: the proxy re-mints per request from the session, and the session row
is minted with `expires_at IS NULL` — measured, a 30-day-aged session still
mints. The card's "bearer" coordinate exists to satisfy `startSync`'s
truthiness check and to keep the direct-substrate/future-Realtime path open;
it authorizes nothing at the door.

**2 — provisioning triggers at PAGE INIT, and this is now a decided
engineering call, not an open question.** Everything the card must write is
derivable with only the session cookie, which every logged-in page load
already has: `restUrl` is the constant `/sync/rest` (same-origin, decision 69),
`bearer` comes from the mint envelope, and `deviceId` — the coordinate the
roadmap did not name — comes from the same envelope's `sub` (measured equal
to `users.id`). `exp − iat` = 900 s exactly (`DefaultTokenTTL`). Login-time
provisioning would add a cross-page write for nothing. Degradations all fail
closed and are survivable by the shipped `if (sc) startSync(sc)` guard:
anonymous mint 401, secret-less deploy 503 `sync_bridge_not_configured`,
upstream-less door 503. The tree has exactly one `hq_marketing_sync_v1`
occurrence (the declaration), so the card adds the only writer.

**3 — the deviceId constraint is a correctness cliff, measured both ways.**
With `deviceId` = the mint `sub`, the shipped `startScanAttemptsReplica` +
`makePushHandler` drained a redeem attempt live through the door — request
log `redeem → land`, attempt `accepted/landed/burn_ok`, and the burn recorded
the session user's id as the device. With `deviceId = 'rogue-device'`: the
landing insert drew **HTTP 403** from RLS (with-check `device_id = sub`
evaluated against the SUBSTITUTED token), the shipped handler THREW — the
F-2 throw-retry head-of-line poison class — **and the redeem RPC had already
burned the code before the refusal**, so the burn can never record
(`codes.redeemed_by='rogue-device'`, zero `scan_attempts` rows). A wrong
deviceId is not a degraded mode; it is a stranded burn plus a poisoned queue.

**4 — the push replica is UNWIRED, and wiring it is in this card's scope.**
`startScanAttemptsReplica` has zero callers outside its own module (leg (a),
enumerated). "Provisioning arms sync" without it ships a scanner that queues
attempts forever; F-2's own rationale ("so Card 3's push handler can't
400/retry-poison the queue **when provisioning arms sync**") already assumed
it runs. The card starts it inside `startSync` with `deviceId` from the mint
envelope — stated here as the decided scope, priced by leg (b)'s green.

**5 — the B-439 recovery edge exists, and it is exactly one signal.** The
post-ready latch bug reproduced on shipped surface: after ready, one 503
latches `lastError` and `unresolved()` stays `true` through TWO full
recoveries. The candidate set, measured in both recovery shapes:

| candidate | recovery-with-docs | recovery-EMPTY | silent while erroring |
|---|---|---|---|
| `clock.captures` | fires (+2) | fires (+2) | **yes (+0)** — USABLE |
| `remoteEvents$` | fires | fires | no (fires on 503 cycles too) |
| `active$` transitions | fires | fires | no (fires on 503 cycles too) |

`clock.captures` increments on every HTTP-200 pull (the §5.1 seam scan-page
already injects) and ONLY on success — including the zero-new-rows recovery,
the shape that would defeat any docs-based signal. It is a polled counter,
not an event: the card chooses the mechanism (compare captures across the
latch's own error$ emissions, or extend the same pull-handler seam
`replicas.js` constructs with a success callback beside `clock`). The edge is
real either way; falsification did not occur.

### Build-facts the card inherits

1. **The provisioning writer is small and fully specified:** on page init
   with a session, `POST /api/v1/sync/token` → write
   `{restUrl: '/sync/rest', bearer: envelope.token}` to `SYNC_KEY` → call the
   already-exported `startSync` — plus `deviceId = envelope.sub` for the push
   side. On 401/503, write nothing; the shipped guard keeps today's no-sync
   behavior.
2. **The push wiring must derive `deviceId` from the mint envelope's `sub`,
   never from any device-local identity** — decided by measurement (fact 3
   above): the alternative is a stranded burn + poisoned queue, discovered
   only when provisioning arms sync in production.
3. **`SCAN_STATE.synced` stays honest for free**: initial replication through
   the door resolved for all three replicas; no new readiness machinery is
   owed by this card.
4. **B-438's discharge mechanics are proven end-to-end**: attach happens
   inside `startSync` (spike 01 exercised the exact scan-page order through
   the door), and the discriminator's landed honesty was proven by the
   sibling ledger; what B-438 waited on was only that `startSync` runs — this
   card's whole point. The clause-3 e2e must fail the replica for REAL
   (kill/block the campaigns upstream at the network layer, not
   `setCampaignPolicy` — ledger T-55 decision 174's rule).
5. **The B-439 fix keys on successful-pull evidence** (fact 5): clear
   `lastError` when a pull completes with HTTP 200 — `clock.captures` is the
   shipped witness; pin it with the error→recover→assert-false test the
   backlog names, in BOTH recovery shapes (with-docs and empty).
6. **Test-stack fact for the card's e2e:** the Playwright suite has no sync
   substrate. The e2e that "provisions through the shipped path" needs the
   spike's stack shape (real server + `HQ_SYNC_*` env + PostgREST) or a
   route-intercepted `/sync/rest/*`; intercepting at the network layer is
   consistent with decision 174 (it fails the transport, not the seam).
   The card states which it uses; if it cannot have a substrate in CI, the
   provisioning e2e's sync half runs against intercepted routes and the
   full-stack proof lives in the harness the way `refusal-run.sh` does.

## Corrections

Three agent-reached corrections, all harness repairs — no premise moved:

1. **lib-hq.sh test-pg bring-up fought the Taskfile's compose project.**
   Spike 01's first attempt COULD-NOT-RUN: a bare
   `docker compose -f docker-compose.test.yml up` uses project `hq`, but the
   running `yumyums-test-pg` belongs to project `yumyums-test` (the
   Taskfile's `test:db:up`), so create conflicted on the container name.
   Fixed: reuse a running container; otherwise invoke with
   `-p yumyums-test`, the Taskfile's own shape.
2. **door-push.mjs lacked `RxDBMigrationSchemaPlugin`.** `scan_attempts` is
   schema v1 with a device migration (the discriminator card's version
   bump); the page loads that plugin (`scan-page.js`) and the spike must
   too. Spike 03's first attempt crashed at `addCollections`; fixed by
   loading the same plugin the shipped page loads.
3. **Spike 03's readback SQL named a column that does not exist and
   under-reported the rogue leg.** `order by created_at` →
   `order by scanned_at` (the schema's actual column), and the readback now
   prints fresh code B's `redeemed_by='rogue-device'` so the
   burn-that-can-never-record is reported as a measurement, not implied.

## Review

- signed: operator, 2026-09-06 — covers 3 correction(s)

## Comebacks

- gap: **B-438 + B-439 carried as this card's done_when clauses 3 and 4** —
  not new gaps found by these spikes, but the spikes are now their
  operational definition: spike 01 is the attach-through-the-door mechanics
  (B-438's missing precondition) and spike 04 reproduces the B-439 latch
  post-ready on shipped surface. The card that ships the fixes owes, in the
  same sitting: **one re-run of spike 04 against the shipped clear** (the
  latch must go `unresolved()=false` in BOTH recovery shapes where this run
  measured it stuck `true`), and its own clause-3 e2e per build-fact 4. The
  `refusal-run.sh` precedent: wrapper-free re-execution of the spike against
  the shipped code.
  - validated: spike-04 re-run — run `20260907`, card
    `sync-coordinates-provisioning` (branch
    `card/sync-coordinates-provisioning`). The clear shipped on the
    pull-handler seam `replicas.js` itself constructs: `makePullHandler`
    fires an `onSuccess` on every HTTP-200 pull (the `clock.captures` edge —
    fact 5's verdict), `startReplica` exposes it as `onPullSuccess(fn)` on
    the handle, and `createCampaignPolicySource.attach` clears `lastError`
    on it — so the spike's own construction (`startCampaignsReplica` +
    `attach`) inherits the clear with zero wiring. The owed re-execution is
    `marketing/sync/harness/recovery-clear-run.sh` (spike 04's phases, live
    substrate, committed migrations + seed): **GREEN, EXIT=0** — the latch
    goes `unresolved()=false` in BOTH shapes (C1 with-docs; C2
    recovery-EMPTY, zero rows delivered, captures still advancing) and STILL
    latches during both error phases (the clear is an edge, not a disarm).
    Pre-fix the same script exited 1 at C1, stuck `true` — the spike
    measurement reproduced
    (`.night-crew/runs/2026-09-07-autonomous/c1-red-recovery-clear.log`).
    The UNMODIFIED spike script, re-run post-fix, exits 1 at ITS OWN C1
    ghost-check — "unresolved() self-cleared after recovery — then B-439 is
    already fixed and this spike is measuring a ghost"
    (`c1-spike04-rerun-postfix.log`; that script's outer wrapper prints the
    seam-moves prose for ANY node exit 1 — the RED line above it names the
    real cause). The clause-3 e2e per build-fact 4 landed as the
    network-layer-killed campaigns leg in `tests/marketing.spec.js`
    (describe "Sync provisioning"), no `setCampaignPolicy` anywhere in it.

