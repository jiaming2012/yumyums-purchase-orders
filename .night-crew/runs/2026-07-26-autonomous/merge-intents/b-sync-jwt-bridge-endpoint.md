# Merge intent — card B `sync-jwt-bridge-endpoint` *(backend half only)*

Branch: `card/b-sync-jwt-bridge-endpoint` (cut from `overnight-20260726` at `c9dc440`)
Written BEFORE implementation, as the card's first commit, per §15ad.65.

## Card in one line

A Go endpoint that mints the Supabase-compatible **HS256** JWT from HQ's **existing**
session/bearer auth and **existing** grant data, plus an attack-variant suite that proves
the RLS policies written against that token **discriminate** — every variant captured
refusing *before* the policy that refuses it is written. **No GoTrue, no Supabase Auth, no
Kong, no new Go module.** The frontend client-construction helper and the
`@supabase/supabase-js` pin are **explicitly NOT this card's** — they moved to the client
layer at slating.

## Shared files touched

Files outside this card's own packages, each with one line of why:

- `.night-crew/runs/2026-07-26-autonomous/merge-intents/b-sync-jwt-bridge-endpoint.md` —
  this note. New file, unique to this card. No conflict surface.
- `.night-crew/runs/2026-07-26-autonomous/timings.log` — **append-only**, lines prefixed
  `B_`. Card C appends its own prefixed lines concurrently. On conflict, **union both
  sides**; never drop a side.
- `.night-crew/runs/2026-07-26-autonomous/DECISIONS-NEEDED.md` — **only if this card
  parks, or if it surfaces an operator question worth recording.** Appended as its own
  new `## B — …` section. It already carries D-1/D-2/D-3 from card A's review; those are
  **not** to be overwritten. Union on conflict.
- `.night-crew/knowledge/roadmap.md` — the `sync-jwt-bridge-endpoint` card status flip,
  required in the same change set as the work. **The flip text must say BACKEND HALF
  ONLY** and name where the client half went.
- `backend/cmd/server/main.go` — one route mount for the new endpoint, inside the existing
  cookie-auth group. This is the only production wiring file this card touches; it is a
  single-line-region addition, so a conflict here is an ordinary adjacent-hunk resolve.
- `.night-crew/qa/spike-supabase/sql/**` — **new SQL fixture files only.** W1's
  `spike-fixture.sql` is **read-only** for this card: card C is running against the live
  stack concurrently and W1's proof artifacts must keep re-verifying.

Everything else this card writes lives in its own packages:

- `backend/internal/sync/**` — the mint, the endpoint handler, the Go test files.
- `backend/internal/auth/**` — read-mostly; touched only if the grant read needs a shared
  helper.

### Late additions

_(filled in only if implementation forces a file outside the list above)_

- **Nothing here.** — updated at card end if that changes.

## What must survive any merge

1. **The mint is stdlib-only.** `crypto/hmac`, `crypto/sha256`, `encoding/base64`,
   `encoding/json`. W1 already proved this is accepted by both PostgREST and Realtime. A
   merge that "modernises" this by pulling in `github.com/golang-jwt/jwt` breaks HARD
   constraint 1 and destroys the card's central claim.
2. **The `auth.uid()` prohibition, and the policies that honour it.** Every RLS policy this
   card writes reads
   `current_setting('request.jwt.claims', true)::json ->> '<claim>'` — the **plural** GUC.
   `auth.uid()` / `auth.jwt()` are **wrong for this stack** (no GoTrue migrations; `uid`
   reads the legacy singular GUC with no plural fallback and casts to `uuid`). A merge that
   "simplifies" a policy back to `auth.uid()` reintroduces a defect that fails
   **non-obviously** — NULL, or `invalid input syntax for type uuid`. The comment block
   recording *why* must survive with the code.
3. **The discrimination proof, including the `service_role` BYPASSRLS control.** A 200
   proves nothing on its own. Every scoped result in the suite is paired with a
   `service_role` read of the same table showing the rows were there and RLS hid them.
   Dropping the control for brevity turns the whole suite vacuous — it becomes
   indistinguishable from "the table was empty."
4. **Red-first ordering as a fact of the history.** The attack-variant suite is committed,
   run, and captured **failing** in a commit that precedes the policy commit. Squashing
   this card's history into one commit erases the card's real gate. If a merge must squash,
   the red-capture excerpts in the commit message must be carried forward.
5. **`service_role` is never mintable by anything a client can reach.** The endpoint refuses
   to emit it, and there is a test that says so. This is W1's stated warning, promoted here
   into an enforced invariant.
6. **The grant→claim mapping maps only EXISTING HQ concepts.** No new grant or permission
   concept is introduced — that is this card's PARK trigger. Any merge that adds one is
   answering a product question the run has no authority to answer.
7. **The revocation semantics.** Whatever this card lands, a token replayed after its
   grant is revoked must be shown refused, and the mechanism that refuses it must survive
   intact. A stale claim inside a still-unexpired token is the sharpest edge in the whole
   bridge.

## What is safe to drop

- Prose, comment wording, section ordering, and test-name strings anywhere, so long as
  items 1–7 survive intact.
- The `timings.log` lines — a record, not a behaviour.
- Log-level and error-message phrasing in the handler.
- Anything in this note itself.

## Not done, deliberately

- **No frontend.** No client-construction helper, no `@supabase/supabase-js` pin, no
  upgrade smoke test, no `*.html` change. Those belong to the client-layer card
  (slate split table). Decision 51's substance is unchanged; only its address moved.
- **No Kong / gateway** — FORK 4 resolved gateway-less (decision 51).
- **No GoTrue, no Supabase Auth, no hosted Supabase project** — self-hosted only, the
  roadmap's standing rule.
- **No teardown or reconfiguration of the running spike stack.** It is shared with card C
  right now. New SQL fixtures are added; nothing W1 left running is disturbed.
- **No new Go module dependency**, for any reason.

## Four-HARD-constraints attestation

Each of these, if broken, changes the build or test environment for **every other card
tonight** — and card C is running **concurrently**, so a breach is not theoretical. All
four are untouched by this card; verifiable with
`git diff --stat overnight-20260726..HEAD -- <path>`.

1. **`backend/go.mod` — UNTOUCHED.** This is the constraint this card protects
   *specifically*. The mint is stdlib-only (`crypto/hmac`, `crypto/sha256`,
   `encoding/base64`, `encoding/json`). **No JWT library is added.** W1's `mintjwt/main.go`
   is the standing counter-argument to reaching for one: an HS256 JWT is a base64url-joined
   header, payload and HMAC-SHA256, and that is all it is. `backend/go.sum` likewise
   untouched.
2. **Root `package.json` and `package-lock.json` — UNTOUCHED.** They are the Playwright
   environment for every card and worktree tonight. Card C is live in another worktree; a
   new devDep here breaks concurrent dispatch for the whole night. This card is Go and SQL
   only and needs nothing from npm.
3. **`docker-compose.nc.yml` — UNTOUCHED.** The Supabase stack lives in the separate
   `docker-compose.supabase.yml` under project name `spike-supabase`, already running from
   W1. This card adds SQL fixtures to it, not compose changes.
4. **Root `Taskfile.yml` — UNTOUCHED.** No new task targets. The Go suite is run directly
   (`go test ./... -count=1 -p 1`), and the operator path stays W1's README of real
   commands.

## Run mechanics this card is bound by

- `export PATH="/usr/local/go/bin:$PATH"` before **any** Playwright or Go leg — the
  non-interactive shell does not carry Go, and Playwright's `webServer` dies with
  `go: not found` / exit 127, which **looks like a test failure and is not**.
- `CI=1` on every suite leg (forces `reuseExistingServer:false`).
- **`TEST_PORT=8399`**, DB names `hq_test_b1`, `hq_test_b2`, … Card C holds 8299; 8199 is
  the foreign-server latch. Neither is touched.
- **Never `git stash`** — worktrees share `refs/stash` and card C is live.
- `go test ./... -count=1 -p 1` — **`-p 1` is load-bearing.** Without it
  `backend/internal/workflow` reds with `checklist_templates_created_by_fkey` violations
  from cross-package `users` truncation on a shared `hq_test_go`. Known false red.
- Every suite result is reported with the **measured 1-min load average**, and with card
  C's concurrency stated explicitly — a green here bounds a **loaded** condition.
- Non-attribution: `tests/sync.spec.js:1198` (~16–20 % flake, exposure deliberately raised
  by card A) and `tests/purchasing.spec.js:1407`. If this card cannot test whether a red is
  its own, it **flags the correlation and refuses to attribute it**.
