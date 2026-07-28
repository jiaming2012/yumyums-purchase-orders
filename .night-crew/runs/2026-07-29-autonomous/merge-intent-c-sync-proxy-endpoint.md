# Merge intent — Card C `sync-proxy-endpoint`

Branch: `card/c-sync-proxy-endpoint` (cut from `overnight-20260729` @ `25fbc16`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

Build the **same-origin door** decision 69 chose: a `/sync/*` `httputil.ReverseProxy` in the
existing Go backend fronting `rest:3000` (PostgREST) and `realtime:4000` (Phoenix/Realtime),
**including the WebSocket upgrade path** — auth reused from the existing session middleware, and
the bridge JWT the backend already mints (`sync-jwt-bridge-endpoint`, 07-26) being what the
proxied services accept. Fanned out of obligation 6 of
`sync-rxdb-schema-and-replication`; backend-only, fork-free. This card builds the door; the RxDB
client knocks on it in a later card.

## Shared files touched

- `backend/internal/sync/proxy.go` — **new, the card's core.** No other card on tonight's slate
  names it. No conflict surface.
- `backend/internal/sync/proxy_test.go` — **new, the card's core.** Red-first tests for the plain
  HTTP path and the upgrade path. No conflict surface.
- `backend/internal/sync/proxy_live_test.go` — **new, the card's core.** The proof against the
  live `spike-supabase-realtime-1` container. ~~`t.Skip`ped when the container is not reachable so
  the suite stays green on a machine without it.~~ **STRUCK at G6 repair (R4, refined at F-4).** It
  is now gated on `HQ_SYNC_SPIKE_LIVE`, **asymmetrically**: flag unset *or falsy* (`0`/`false`/
  `no`/`off`) → skip; flag truthy and port up → run; **flag truthy and port dead → FAIL, not
  skip.** The struck version's "stays green on a machine without it"
  was exactly the problem — with the containers down, `go test` printed `ok ... 1.513s`,
  indistinguishable in a non-verbose log from a run that proved the live upgrade. No conflict
  surface.
- `backend/cmd/server/main.go` — ⚠️ **THE ONE FILE CARD B AND I ARE BOTH LIKELY TO TOUCH.**
  See the dedicated section below.
- `backend/internal/version/version.go` — **shared file, every card touches it.** This card edits
  the **`Backend`** constant ONLY (`0.2.2` → `0.3.0`; a new endpoint is a minor). **`Frontend` is
  NOT touched** — Card A already moved it to `1.2.1` and Card B may move it again. A merge that has
  to pick must take the **higher** semver on each constant *independently*, and must NOT let a
  `Backend` conflict drag `Frontend` backwards.
- `package.json` — **NOT touched.** This card changes zero frontend files, so the
  `Frontend` ↔ `package.json` mirror is not this card's business. If a merge shows this card
  touching `package.json`, that is wrong — drop it.
- `sw.js` / `version.json` — **NOT touched, not regenerated.** No HTML/JS file changes here, so
  `node build-sw.js` has nothing to do. If a merge shows an `sw.js` diff attributed to this card,
  take the other side.
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip, in the same
  change set as the work, matching the convention Card A's flip at `:359` uses. Single-card edit
  in the `sync-rxdb-schema-and-replication` / fan-out region (~`:495`). Every card tonight edits
  its own card in this file; conflicts are per-card and **both sides should be kept**.
- `.night-crew/runs/2026-07-29-autonomous/merge-intent-c-sync-proxy-endpoint.md` — this note. New
  file, unique to this card. No conflict surface.

## ⚠️ `backend/cmd/server/main.go` — the shared router, and what my registration must look like

Card B also edits `backend/internal/workflow` and adds a migration; route registration in
`main.go` is the surface we can collide on. Resolve any conflict **against intent, not text** —
here is my intent, stated so it can be reapplied by hand:

My registration is a **new top-level route group**, a sibling of the existing `/ws` group at
`main.go:410-416`, mounted **outside** the `r.Route("/api/v1", ...)` block:

```go
// Sync substrate proxy — the same-origin door (decision 69).
r.Group(func(r chi.Router) {
    r.Use(auth.Middleware(pool, superadmins))
    r.Handle("/sync/*", opsync.ProxyHandler(pool, opsync.LoadProxyConfig()))
})
```

Three properties of that snippet are load-bearing and must survive however the text merges:

1. **It is at the root, at `/sync/*` — NOT inside `/api/v1`.** `/api/v1/sync/token` already exists
   (`main.go:555`, the JWT bridge) and mounting the proxy under the same prefix would put a
   catch-all wildcard in front of it. The two are different things at different paths and must
   stay that way.
2. **It sits behind `auth.Middleware(pool, superadmins)` and nothing else.** That is the *existing*
   bearer/session middleware — this card invents no second auth path. It is deliberately **not**
   behind `auth.RequirePermission`, for the same reason `/api/v1/sync/token` is not: it is
   access-resolution plumbing, and the real per-row authorization is RLS inside the proxied
   services reading the live grant projection. Picking a grant to gate it behind would be
   inventing a permission concept — the parent card's park trigger, not mine.
3. **It uses `r.Handle("/sync/*", ...)` and not `r.Mount`.** The handler does its own
   `/sync/rest/...` vs `/sync/realtime/...` sub-routing internally, because the upgrade path needs
   the raw `*http.Request` and a chi sub-router buys nothing here.

If Card B's changes and mine both land in `main.go`, **keep both hunks** — they are in different
regions (mine near `:410`, Card B's presumably inside the `/api/v1/workflow` route block near
`:526-540`) and are not semantically related.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean)_

Closed out after the gates. **The footprint held exactly as declared** — the three new
`proxy*.go` files, `backend/cmd/server/main.go`, `backend/internal/version/version.go`, the roadmap
flip, and this note. **No file outside the list was edited.** ~~The whole note was re-read; **no
line above is contradicted, so nothing is struck.**~~ **STRUCK after the G6 repair round** — that
sentence was true when written and is not any more.

**Re-read history — this note has now been read IN FULL three times** (first close-out, then per
B-11 at each of the two G6 repair rounds). **Five lines are struck across the file:** the
`proxy_live_test.go` skip description above, this sentence, the "safe to drop" entry for that
file, constraint 3's closing clause, and — added at repair round 2 — the original wording of
must-survive item 8, which was wrong in both directions at once. The footprint claim itself still
holds through both rounds: only files already on the list were edited, and `main.go` and
`version.go` were touched by neither.
Four things a merge should know that the note did not anticipate:

1. **The registration snippet in the section below is EXACT — it is what shipped, verbatim.**
   `main.go:418-444` (comment block plus the four-line group). Re-check it against that section if
   a merge has to reapply it by hand.
2. **`ProxyHandler(pool, cfg)` keeps the signature the note promised, but delegates.** The
   implementation added an injection seam — `type TokenMinter func(ctx, *auth.User, sid) (string,
   error)` and an unexported `newProxyHandler(mint, cfg)` — so the proxy's own behaviour is
   testable without Postgres. `ProxyHandler` is a one-line wrapper around `newProxyHandler(
   poolMinter(pool), cfg)`. **`main.go`'s call site is unchanged from what the note declared.**
3. **`proxy_test.go` imports `github.com/go-chi/chi/v5` and `.../middleware`** for
   `TestProxy_SurvivesTheRealChiRouterAndMiddlewareStack`, which rebuilds `main.go`'s actual router
   shape and drives an upgrade through it. Both are already direct dependencies —
   **`backend/go.mod` is still UNTOUCHED**, constraint 2 holds.
4. **The card's premise was corrected by the red, and the roadmap entry records it.** A naive
   `httputil.NewSingleHostReverseProxy` baseline was stood up purely to take a behavioural red, and
   it **passed the 101 and the bidirectional echo** — Go's stdlib `ReverseProxy` handles the
   protocol switch correctly on its own. What a naive proxy actually gets wrong here is the Host
   header (Realtime's tenant routing), the prefix strip, and credential injection. That baseline
   was **not committed**. *(G6 independently rebuilt this baseline and confirmed the correction.)*

### G6 repair round 1 — what changed after the first close-out

G6 adversarial review returned **APPROVE-WITH-NITS**. Five items were repaired in two commits
(`7760252`, `0ce6b76`). **No file outside the declared footprint was touched.**

*(This list was numbered 5-8, which collided with the must-survive list's own item 8. Renumbered
R-a…R-d at the second repair round — G6 F-5. Cross-references below point at MUST-SURVIVE item
numbers and are unaffected.)*

- **R-a. R1 added a security-relevant behaviour to `proxy.go` — it belongs on the must-survive
  list, and it is MUST-SURVIVE ITEM 8 below.** Path traversal: the room remainder was forwarded
  un-normalised, `%2f` became a real separator, and `/sync/realtime/../rest/spike_notes` reached
  the *other* upstream carrying the minted JWT. Now `400 sync_path_rejected` via
  `unsafeRequestPath`, checked **before the room is chosen** — the ordering is load-bearing,
  because the room is chosen from the decoded path and an encoded separator forges that decision.
- **R-b. R2 added an ACTIVATION-ORDER CONSTRAINT that is now recorded in three places** —
  `proxy.go`'s env-var block, the parent card's obligation-6 annotation, and the DONE card. It is
  MUST-SURVIVE ITEM 9 below and it is the one thing in this whole card an operator can get hurt
  by. A merge that keeps the code and drops all three copies of the warning ships a loaded gun.
- **R-c. `main.go` was NOT touched by the repair round.** The registration snippet in the section
  above is still exact. The Card B collision surface is unchanged.
- **R-d. `version.go` was NOT bumped again.** `Backend` stays `0.3.0`; these were repairs to
  unreleased code in the same change set, not a second increment.

### G6 repair round 2 (final) — what changed after that

G6 reviewed the repair delta and returned **APPROVE-WITH-NITS** again, having run a 38-vector
attack matrix and verified the "not over-broad" claim against the LIVE containers (24 PostgREST +
4 Realtime calls through the door, zero false rejections, every non-200 originating from PostgREST
itself). Four items, repaired in commit `c9b8cdd`. **No file outside the declared footprint was
touched; `main.go` and `version.go` were again NOT touched.**

- **R2-a. F-1 — the `%2f` check was BYPASSABLE, and two places in this repo claimed it was not.**
  `unsafeRequestPath` read only `u.EscapedPath()`, which discards `RawPath` and re-escapes the
  decoded `Path` whenever RawPath holds a byte Go's `encodePath` validator rejects
  (`{ } | ^ \ " < >`). Go's escaper does not escape `/`, so the `%2f` vanished before the check:
  `GET /sync/rest%2fadmin{` → `200 OK`, upstream saw `GET /admin%7B`. **Not exploitable** — the
  dot-segment loop reads the decoded `Path`, so `..` was still caught, and G6 could not get a `..`
  or a `%2f` onto the wire by any route. Fixed anyway because `proxy.go` claimed it "rejected
  EVERY encoded separator" and must-survive item 8 below claimed "anywhere in the request path",
  and **the right move on a false durable claim is to make it true, not to soften it.** Now tests
  `u.RawPath` (the untouched request target, always populated when `%2f` is present) alongside
  `EscapedPath()`.
- **R2-b. F-2 — "any dot segment" was an overstatement the other way.** See the strike on
  must-survive item 8.
- **R2-c. F-4 — `HQ_SYNC_SPIKE_LIVE=0` opted IN.** The gate tested `os.Getenv(...) == ""`, so the
  obvious spelling of "off" produced a hard failure. Now `""`/`0`/`false`/`no`/`off` are off, with
  `TestSpikeLiveRequested` pinning the table.
- **R2-d. F-6 — a "see point 3 below" pointer aimed at a point above it.** Corrected.

**Deferred to the follow-up list by the coordinator, NOT fixed here:** `main.go`'s registration
comment reads as an all-clear with no "don't configure it yet" (deliberately left alone to protect
the Card B collision surface — but see must-survive item 9, which is where that warning lives);
`q.Encode()`'s alphabetisation of the outbound query (pre-existing, harmless); and the DB-drop
silent-green hazard in this package's OTHER test files (`access_test.go:29,33`,
`jwtbridge_test.go:169,173` skip on connect *and* ping failure — `pgxpool.New` is lazy, so a
missing database surfaces at `Ping` as a skip and non-verbose output is a bare `ok`). The R4
asymmetry in `proxy_live_test.go` is the shape that fixes it; applying it there is someone else's
card.

## What must survive any merge

1. **The proxy exists as a same-origin `/sync/*` door in the HQ backend.** Decision 69 chose it
   over a second origin precisely so there is no CORS, no second hostname in the Cloudflare Tunnel,
   and no second origin for the service worker to reason about. Anything that reintroduces a
   cross-origin shape reopens decision 62, which is closed.
2. **The WebSocket upgrade path.** A `/sync/realtime/...` request carrying `Connection: Upgrade` /
   `Upgrade: websocket` must reach the Realtime backend as a real upgrade, get its `101` back to
   the client, and pass bytes in **both** directions afterwards. This is the half a naive
   `ReverseProxy` gets wrong and the half the slate says is worth a test. Losing it silently is
   worse than not shipping it, because the failure only appears under a live Realtime container.
3. **The client never sees an internal service URL or a bridge token in a proxy error path.**
   Upstream failures produce a generic JSON envelope; the target's host:port is logged
   server-side, never echoed. (G4.)
4. **A client-supplied `Authorization` header or `apikey` query parameter is STRIPPED and replaced
   by a server-minted token.** The proxy mints for the *context* user via the existing
   `MintForUser`, so a caller cannot smuggle a token minted for someone else — the same
   impersonation invariant `TokenHandler` carries, applied at the door.
5. **Fail-closed when unconfigured.** Unset upstream URL ⇒ `503 {"error":"sync_proxy_not_configured"}`,
   mirroring `sync_bridge_not_configured` and `auth.ServiceTokenMiddleware`. A misconfigured deploy
   must never proxy somewhere unintended.
6. **`Backend` in `version.go` is bumped to at least `0.3.0`.**
7. **The red-first tests.** Both of them — the plain-HTTP request AND the upgrade request. A merge
   that keeps `proxy.go` and drops `proxy_test.go` keeps a claim without its proof.
8. **The path-traversal rejection, and its POSITION.** *(Added at repair round 1, R1; corrected at
   repair round 2, F-1/F-2.)* ~~Any dot segment (`.`, `..`) or encoded separator (`%2f`, `%5c`,
   literal `\`) anywhere in the request path ⇒ `400 sync_path_rejected`~~ — **STRUCK, and replaced
   with what is actually true**, because this sentence was wrong in BOTH directions when written:
   - **Too weak on encoded separators, now made TRUE.** "Anywhere in the request path" did not hold
     — `EscapedPath()` dropped the `%2f` for eight suffix bytes (`{ } | ^ \ " < >`). The check now
     reads `u.RawPath` as well, so the claim holds as stated. **A merge that reverts to an
     `EscapedPath()`-only check re-opens the bypass and re-falsifies this line.**
   - **Too strong on dot segments.** The rule is an EXACT match on `.` / `..` against **Go's own
     decoded segmentation** — it is not a universal dot-segment rule. `..;/`, `....//`, `..%00/`,
     `..%c0%af..` and `%252e%252e` all pass and reach the upstream verbatim. That is deliberate
     (G6 confirmed none traverses against nginx or Kong), but it stops being safe behind a
     Tomcat/Jetty-class parser, which treats `..;` as a dot segment, or anything that decodes
     twice.

   So: **encoded separator (`%2f`, `%5c`, literal `\`) anywhere in the request path, OR an exact
   `.`/`..` segment in Go's decoding ⇒ `400 sync_path_rejected`** — and **the check runs before the
   room is resolved**. Position is
   not stylistic: the room is chosen from the DECODED path, so `/sync/rest%2f..%2f..%2fadmin`
   selects the REST room with separators the caller forged. It must **reject, not `path.Clean`** —
   cleaning silently proxies a different request than the one that was made.
   `TestProxy_RejectsPathTraversal` and `TestProxy_LegitimatePathsStillPass` are a pair: the second
   is what stops a future over-broad rule (a dot IN a segment is legal, a dot SEGMENT is not).
9. **The activation-order constraint, in all three places it is written.** *(Added at the G6 repair
   round, R2.)* **Do not set `HQ_SYNC_REST_URL` in any deploy until row-visibility RLS (obligation
   1) lands.** The door forwards every method with a `role: authenticated` token and does no row
   filtering of its own — by design, since filtering is the parent card's. Activating REST before
   RLS gives every logged-in crew member full read AND write on the whole exposed schema.
   `HQ_SYNC_REALTIME_URL` is the safe half to adopt first (read-only). The three copies are in
   `proxy.go`'s env-var block, the parent card's obligation-6 annotation, and the DONE card; a
   merge should keep all three, and must keep at least the `proxy.go` one.

## What is safe to drop

- **Comment wording** anywhere in the new files, and test names. **Two exceptions, added at the G6
  repair round:** the activation-order block and the Realtime-logging caveat in `proxy.go`'s
  env-var comment are *comments whose content is load-bearing* — they are the only place the
  operational constraint lives in code. Reword them freely; do not drop them.
- **The env-var spellings** (`HQ_SYNC_REST_URL`, `HQ_SYNC_REALTIME_URL`,
  `HQ_SYNC_REALTIME_HOST`) — the *behaviour* matters (configured upstreams, fail-closed when
  unset, a Host override for Realtime's tenant routing); the exact names do not, as long as
  nothing else in the repo already reads them.
- **The exact sub-path spelling** `/sync/rest/*` and `/sync/realtime/*`. What matters is that one
  same-origin prefix reaches both upstreams and that they are distinguishable.
- **`proxy_live_test.go` entirely.** ~~It is skipped without the spike container and proves nothing
  a merge conflict is qualified to adjudicate.~~ **The reasoning is STRUCK at the G6 repair round
  (R4)** — the file no longer merely skips; it FAILS when `HQ_SYNC_SPIKE_LIVE=1` is set and the
  port is dead, which is the whole point of it. The *conclusion* still stands: it depends on a
  container a merge cannot adjudicate, and the hermetic upgrade test is the one that must survive.
  But dropping it drops the only evidence that Realtime accepts what comes out of the door.
- **The roadmap card's prose.** The **status flip** matters; the wording does not.
- **Anything in this note itself.**

## Not done, deliberately

- **RLS predicates (obligation 1).** `ResolveEntityAccess` porting, `template_assignments ⋈ users`
  projection, `EXISTS` predicates — all of it belongs to the parent card
  `sync-rxdb-schema-and-replication`. Not in scope, per the slate.
- **RxDB client code.** No `vendor/` glob re-add, no `globPatterns` change, no replication state
  machine, no conflict handler. The door only.
- **`workflows.html`.** Untouched. Zero frontend files, hence no `sw.js` regeneration and no
  `package.json` version mirror.
- **No new `docker-compose` service and no infra provisioning.** The spike stack
  (`docker-compose.supabase.yml`, project `spike-supabase`) is *read* by the live test if it
  happens to be up; nothing is brought up, torn down, or added.
- **No `openspec/` directory or OpenSpec mechanics.** `night-crew workflow preflight` reports
  openspec ABSENT for this repo. Universal per-change discipline only (red-first, atomic commits,
  `Night-Crew-Card:` trailer, roadmap flip).
- **`tests/sync.spec.js:1584`'s stale comment (B-06) is NOT folded in.** It belongs to the parent
  card, not tonight's slate.
- **No second auth path.** No API-key header, no service token, no bespoke query-param session.
  `auth.Middleware` or nothing.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **BOTH UNTOUCHED.** This card changes no
   frontend file, so the `Frontend` ↔ `package.json` mirror is not engaged. No devDependency, no
   script, no lockfile edit. This is the Playwright environment shared by every card tonight.
2. **`backend/go.mod`** — **UNTOUCHED.** `net/http/httputil` is stdlib and
   `github.com/coder/websocket v1.8.14` is already a direct dependency (used by `WsHandler` and by
   the 07-25 spike's `rtwatch`), so the upgrade test needs no new third-party surface.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** No compose service added, renamed, or re-ported.
   `docker-compose.supabase.yml` is likewise **not edited**; the live test only dials a container
   that is already running ~~and skips when it is not~~ — **STRUCK at the G6 repair round (R4):**
   it skips only when `HQ_SYNC_SPIKE_LIVE` is unset, and **fails** when the flag is set and the
   container is not there. Still no compose file read, written, or brought up by any test.
4. **Root `Taskfile.yml`** — **UNTOUCHED.** Go tests are invoked directly with an explicit
   `DB_TEST_URL` pointing at this card's own database (`hq_test_go_c`), because a card is running
   concurrently and the shared `hq_test_go` would be truncated out from under it. No task added,
   no var default changed.
