# Roadmap — "Sync foundation" cycle (replace the op-log/WebSocket sync layer with RxDB + self-hosted Supabase)

> **Cycle:** Sync foundation — replace `workflows.html`'s hand-rolled WebSocket + Postgres
> LISTEN/NOTIFY + Lamport-clock op-log sync layer with RxDB (client-side replication) against a
> self-hosted Supabase stack (Realtime + PostgREST, alongside the existing Postgres on the
> Windows box), unifying the field-autosave (`/saveResponse`) and live-broadcast (`sync.js`)
> write paths into one store. Root cause: a recurring fetch-storm/stale-hydration failure class
> diagnosed across the just-closed "Prove & surface" cycle (T-18 fetch-storm bug class, the
> retired-but-recurring `sync.spec.js` flakiness, cross-user hydration divergence) — the
> mechanism RxDB's checkpoint-based replication is built not to have.
> **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA,
> authored alongside this roadmap). **Produced:** 2026-07-24/25 attended `/nc-roadmap-round`,
> scoped to this activity at the operator's direction. Previous cycle archived at
> `reference/roadmap-2026-07-24-prove-and-surface.md`.
> **Trigger:** BACKLOG.md entry "`workflows.html` sync: migrate to RxDB + self-hosted Supabase"
> (operator explore session 2026-07-24, decisions on conflict resolution/scope/auth-bridge/
> cutover recorded there) plus the untracked handoff brief
> `reference/handoff-supabase-rxdb-20260724.md` (written from the night-crew development clone,
> asking for exactly this assessment — its "milestone boundary is the correct home" reading is
> now the actual state, since the milestone closed the same day).
> **Scope note:** this round walked ONE backlog group (sync fragility) at the operator's
> request. ~19 other `· new` backlog items across security/infra hygiene, grants follow-ups,
> test/run-mechanics hygiene, product/display nuance, and money-precision were proposed as
> groups but **not walked this round** — they remain `new` in BACKLOG.md, available for the next
> roadmap-round pass (either continuing this same round or a future one), not dropped.

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan` sizes to a
  night. Cards carry a **module footprint** (for parallel tracks) and a **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** Cards-per-night is the planner's call against
  the night budget + quality bar (budget is a floor, not a ceiling).
- **Build order within Activity 1.** The feasibility spike gates everything else — self-hosted
  Supabase + the JWT bridge must be proven before schema/replication and the auth endpoint are
  sized. Schema+replication and the JWT-bridge endpoint have disjoint footprints (frontend RxDB
  vs. backend auth) and may run in parallel once the spike clears. Hard cutover is serialized
  last — it depends on both.
  **Fan-out 2026-07-25 (`/nc-slate-plan`, §1 split rule):** the feasibility spike was split into
  **two** cards before slating, because the operator directed it to (a) exercise RxDB itself, not
  only the stack beneath it, and (b) leave a runbook the operator can run by hand. Those are two
  mechanisms of normal-card size — Docker/JWT infra and an RxDB replication client — with
  different failure modes and park triggers, so the card is `sync-spike-stack-and-jwt-bridge` +
  `sync-spike-rxdb-replication`. **The cycle's gate is the stack card alone**; the RxDB card
  deepens the verdict and de-risks `sync-rxdb-schema-and-replication`, but blocks nothing.
  Activity 1 therefore holds **6** cards, not 5 — the Delivery per-card-timing KR denominator
  moves with it.
  **Fan-out 2026-07-26 (`/nc-slate-plan`, §1 split rule):** `sync-rxdb-schema-and-replication`
  bundled **four** mechanisms, each rivalling a normal card. Fanned out before slating into
  (a) **`sync-rxdb-browser-delivery-spike`** — how RxDB reaches a no-build-step browser, plus Dexie
  storage, service-worker interaction, multi-tab leader election and token-expiry-offline (the
  feasibility verdict's own instruction, `sync-rxdb-feasibility-spike.md:505-508`: *"budget a real
  browser spike inside that card"*); (b) the schema/replication/`_modified`/`conflictHandler` work,
  which keeps the original name; and (c) **`sync-rxdb-conflict-notice-ui`** — the user-visible half
  of decision 50, split out because CLAUDE.md requires **mockup sign-off before UI code on phases
  introducing new components**, which an unattended run cannot obtain. Separately,
  `sync-jwt-bridge-endpoint` was **narrowed, not split**: its client-construction helper and
  `@supabase/supabase-js` pin move to the client layer where the client is actually constructed
  (decision 51's substance unchanged, only its address). Activity 1 now holds **9** cards; the
  Delivery per-card-timing KR denominator moves again.
- **Red-first is mandatory on every fix card.**
- **Per-card wall-clock timing is a standing output** on every build card, continuing the
  "Prove & surface" cycle's practice (T-14 baseline N=23 / 22m28s; last-measured median 94m
  N=12, population shift noted at that cycle's close).
- **night-crew CI constraint (from the handoff brief, verify before slating):** this repo's
  `[e2e.seams]` config leaves `sync` unmapped for subset optimization — touching it de-confines
  a card to the full Playwright suite. Every card in this activity touches the sync seam by
  construction, so every card here pays full-suite, not a subset. Additionally, night-crew's
  per-test gate granularity is Go-only; Playwright collapses to a single failure atom, so the
  gate cannot distinguish "this card broke it" from "already red" — price this into slating,
  especially given the known `sync.spec.js` load-sensitivity history.
- **Self-hosted only for unattended work.** night-crew's overnight runs prohibit production DB/
  deploy/push/infrastructure provisioning. Every card here builds against the self-hosted
  Supabase in the repo's own Docker compose — never a hosted Supabase project. Cutting over real
  prod data is the operator's own attended act, scoped out of every work order explicitly.

## Module footprints (independent → parallelizable)

| Track | Frontend | Backend | Tests |
|---|---|---|---|
| Sync rewrite | `sync.js`, `workflows.html`, new RxDB client layer | `backend/internal/sync` (retired by cutover), new self-hosted Supabase compose service, new JWT-bridge endpoint | `sync.spec.js` and successors, new RxDB replication tests |
| Independent fix | — | `backend/internal/workflow` (`checklist_submissions.status` default) | `workflow` package tests |

---

## Activity 1 — Sync foundation: RxDB + self-hosted Supabase

- **`sync-spike-stack-and-jwt-bridge`** · **DONE — verdict GO** (2026-07-25, run
  `overnight-20260725`, merged `51d0c02`; G6 PASS-WITH-FINDINGS, all non-blocking). Self-hosted
  Supabase (postgres + postgrest + realtime; **Kong/Studio/GoTrue proved unnecessary**) accepts a
  stdlib-only Go-minted HS256 token on both PostgREST and Realtime, with RLS **demonstrably
  discriminating** — verified twice, once by the card and once independently by G6 against the
  live stack, with a `service_role` BYPASSRLS control ruling out the empty-table explanation.
  Verdict at `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`; runbook half 1 at
  `.night-crew/qa/spike-supabase/README.md`; stack left running deliberately.
  ✅ **The verdict REACHED `ledger.md` at morning triage 2026-07-25 (T-22)** — Product KR1 /
  Delivery KR1 measure the ledger timestamp against `.night-crew/runs/` dispatch timestamps, and
  the ledger entry now predates any downstream dispatch. **`sync-rxdb-schema-and-replication` and
  `sync-jwt-bridge-endpoint` are UNBLOCKED and may be slated together** (disjoint footprints).
  `sync-hard-cutover` was double-blocked — it also needed the "Cross-user checklist hydration
  divergence" backlog item routed to a product decision (Product KR2). **That half is CLEARED as of
  2026-07-26: ledger T-24 decision 67, operator-decided — a new cycle starts fresh for every user
  (0/2 for both); a rejected submission archives to history rather than resurrecting as current
  state, and the fresh state must accept clicks. `sync-hard-cutover` is now blocked ONLY on
  `sync-rxdb-schema-and-replication` landing.** **Note for whoever
  reads the runbook: T-22 decision 53 — six blocks of its "captured output" are hand-composed
  presentation; the underlying facts all re-verify, but the document's own integrity claim
  (`README.md:32-34`, `:724-727`) is currently false and wants repair.** · *(½ of the fanned-out
  `sync-rxdb-feasibility-spike` — the cycle's Wave-0 gate)* · Stand up self-hosted Supabase
  (Realtime + PostgREST, via Docker) in a **new, separate `docker-compose.supabase.yml`** —
  never by extending `docker-compose.nc.yml`, which would boot Supabase for every night-crew run
  in this repo. Prove the Go backend can mint its own HS256 JWTs (`role: authenticated`, `sub`,
  `exp`, signed with Supabase's configured `JWT_SECRET`) using **stdlib only** (`crypto/hmac`;
  no new module dependency) and have self-hosted PostgREST/Realtime accept them for RLS —
  without GoTrue/Supabase Auth. The PostgREST proof must show the policy **discriminating** (an
  authorized read succeeds *and* an unauthorized one is refused); a 200 alone proves nothing.
  Realtime is proven over `github.com/coder/websocket`, already a direct dependency; self-hosted
  Realtime needs a **tenant row**, expected to be the sharpest edge. Confirm the
  self-hosted-specific table contract: text PK, `_deleted` boolean, `_modified` trigger, RLS
  enabled, and manual `ALTER PUBLICATION supabase_realtime ADD TABLE` (no dashboard toggle in
  self-hosted) — note the per-table cost, which is what sizes the schema card. Output: a written
  **GO or NO-GO** at `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` (a NO-GO is a
  completed card, not a failed one) plus **half 1 of the operator runbook**. Footprint: new
  compose file + `.night-crew/qa/spike-supabase/**`; **no production code, and `go.mod` / root
  `package.json` / `docker-compose.nc.yml` / `Taskfile.yml` HARD-untouched.** *(from BACKLOG
  "`workflows.html` sync: migrate to RxDB + self-hosted Supabase")*

- **`sync-spike-rxdb-replication`** · **DONE — verdict GO on RxDB, with one signed assumption
  DISPROVEN** (2026-07-25, run `overnight-20260725`, merged `ba22744`; G6 PASS-WITH-FINDINGS →
  1 blocking, revised and re-checked). `rxdb@17.4.0`'s `replicateSupabase` replicates **both
  directions** over W1's stack — push verified over an independent request, pull converging in
  ~90–130 ms into a client that was never restarted. **Apache-2.0, no paid dependency** (Dexie =
  free browser path, IndexedDB = premium; premium buys speed, not capability), and a **real shipped
  plugin**, not an example — though introduced as *beta* in 16.19.0, so size it as young.
  🛑 **The finding: conflict resolution is master-wins, NOT the last-write-wins the 2026-07-24
  explore session signed.** A strictly-later local write is discarded and **no clock participates**
  — compare-and-swap plus `defaultConflictHandler` returning `realMasterState`; `_modified` is only
  the pull cursor. Silent by default, but **observable via `conflict$`**. Reproduced 3×. **This gates
  the conflict-policy half of `sync-rxdb-schema-and-replication`** — see
  `runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` FORK 3 (and FORK 4, Kong vs. a client shim).
  A Node-side proof establishes the replication protocol only — **not** browser storage, service-worker
  interaction, or PWA offline semantics. · *(½ of the
  fanned-out `sync-rxdb-feasibility-spike`; operator ask 2026-07-25 — the spike must exercise
  RxDB itself, and leave something runnable)* · Drive an actual RxDB collection against the
  stack card's Supabase, from an **isolated Node harness** at `.night-crew/qa/spike-supabase/rxdb/`
  with its **own** `package.json` (the repo-root one is the Playwright environment for every card
  and stays HARD-untouched). Prove replication in **both directions separately** — local RxDB
  write visible in Postgres via PostgREST, and a direct Postgres write converging into RxDB
  without a client restart; a one-directional proof is how this class of spike fools itself.
  **Observe last-write-wins rather than assuming it**: construct one concurrent-write case and
  record what actually happens, including which clock decides and whether a write is silently
  lost — a divergence from the assumed LWW is a finding to route, not something to correct in
  code. **Answer two go/no-go inputs:** (a) which RxDB storage the real PWA would use in a browser
  and whether it is free or premium-licensed under RxDB's current terms (verify against the
  license page; a paid dependency is a cost the operator must know before four cards are built on
  it), and (b) whether Supabase replication is a supported plugin or a documented example built on
  `replicateRxCollection` that we would maintain ourselves. Uses **one throwaway table, not the
  real checklist domain** — modelling that is the schema card's job. States plainly what a
  Node-side proof does **not** establish (browser storage, service-worker interaction, PWA offline
  semantics). Output: **half 2 of the operator runbook** (append-only) + the RxDB half of the
  verdict, sizing `sync-rxdb-schema-and-replication`. Footprint:
  `.night-crew/qa/spike-supabase/rxdb/**` only.

- **`sync-rxdb-browser-delivery-spike`** · **DONE — verdict GO** (2026-07-26, run
  `overnight-20260726`) · *(the client track's Wave-0 gate — everything client-side waited on its
  verdict; it is now cleared)* · **All five legs PROVED in a real Chromium**, 11/11 harness tests
  green. Delivery: `vendor/rxdb.bundle.js`, 506,885 B raw / 148,591 B gzip, generated by a hand-run
  `npx esbuild@0.28.1` and loaded with a plain `<script type="module">` — **no build step in the
  deploy path, root `package.json`/`package-lock.json` byte-untouched.** Dexie survives a real
  reload. The Workbox SW does not interfere with replication and does not touch WebSockets. Leader
  election works at the browser default with ~50 ms handover and the survivor genuinely resuming.
  Token expiry across an offline period is recoverable by a **token swap**, with no data loss.
  **Three findings, all in the verdict:** 🛑 **(a) the replication endpoint must NOT be mounted
  under `/api/`** — Workbox `NetworkFirst` falls back to its CACHE before its error handler and
  answered an offline pull with a stale well-formed 200, which RxDB cannot distinguish from a fresh
  pull and would checkpoint past (silent data loss, measured); **(b) RxDB 17.4.0's dev-mode plugin
  appends a hidden `rxdb.info` iframe on ANY host** (its `!isLocalHost()` guard is commented out in
  the shipped build) — dropped from the bundle, −201,671 B and zero third-party hosts; **(c)
  PostgREST tolerates ~30 s of `exp` skew**, which nearly produced a false green in the token leg.
  **Red-first: N/A**, stated explicitly in the verdict, the merge-intent note and here — the
  deliverable is a verdict, so there is no production behaviour to hold red. **Rider T-22 decision
  53 landed:** the runbook's integrity claim was **narrowed** (not the presentation repaired), with
  a new 6-row "Integrity of the output blocks" inventory; a re-capture today cannot be byte-exact
  because `spike_notes` accumulates by design. Verdict at
  `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`; runbook half 3 at
  `.night-crew/qa/spike-supabase/README.md`; harness at `.night-crew/qa/spike-supabase/browser/`.
  **Not established:** Safari/iOS (Chromium only — quota, eviction, private browsing all untested),
  a browser reaching self-hosted Realtime directly, and Realtime's own token lifecycle.
  **No production page imports the bundle** — the path is proven, not adopted.

  *Original card text, for the record:* Prove in a **real browser**, against W1's
  self-hosted stack, everything the Node-side proof deliberately did not establish
  (`sync-rxdb-feasibility-spike.md:512-537`). **(1) Delivery** — HQ is *"static only, no build
  step"* with zero runtime deps and one CDN `<script>` (SortableJS, `workflows.html:172`), so how a
  large ESM library reaches the browser is load-bearing and was previously assumed away. **Planner
  decision, operator-accepted at sign-off: a committed, vendored pre-built bundle**
  (`vendor/rxdb.bundle.js`, generated by a committed script via `npx esbuild@<pinned>`, precached by
  Workbox like any local asset) — chosen over a CDN import (a food-truck PWA must not put its
  offline engine behind a CDN it cannot reach, and a CDN URL needs runtime caching, i.e. a *second*
  offline story, which Engineering KR3 forbids) and over a real build step (breaks a stated
  constraint and rewrites the deploy path — an operator decision, not a card's).
  **Root `package.json`/`package-lock.json` HARD-untouched** — that is why `npx`, not a devDep.
  **(2) Dexie storage in a real browser**, not `getRxStorageMemory()` — write, reload, assert
  survival. **(3) 🛑 Service-worker interaction, the sharpest edge:** HQ's Workbox `sw.js` is
  network-first for API calls with an offline JSON fallback, and a fallback answering a
  *replication* request with cached JSON is a plausible, nasty failure — and
  `playwright.config.js:60` sets `serviceWorkers: 'block'` **repo-wide**, so this leg cannot be
  proven inside the normal suite and needs its own harness. **Never change the repo-wide setting.**
  **(4) Multi-tab leader election** (harness set `waitForLeadership:false`; browsers default true).
  **(5) Token expiry across an offline period** — untouched by W1 and W2. Output: **GO or NO-GO**
  appended to `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` (a NO-GO is a completed
  card) + **half 3 of the operator runbook**. **Rider — T-22 decision 53:** this is the next card to
  touch the runbook, so it repairs W1's false integrity claim (six blocks of hand-composed
  presentation; the underlying facts all re-verify and W1's GO stands — repair the presentation or
  narrow the claim). Footprint: `vendor/**` (new), `build-sw.js`, `.night-crew/qa/spike-supabase/**`,
  the feasibility design doc. **No production `workflows.html` change** — it proves a delivery path,
  it does not adopt one.

- **`pwa-cache-and-build-hygiene`** · **DONE — all three** (2026-07-27, run `overnight-20260727`,
  Wave 0 alone; card authored at morning triage 2026-07-26 from ledger T-23 decisions 57, 58, 59)
  · The live cross-tenant disclosure is closed on the client side. **Five tests captured RED first
  on the unfixed tree at `221e59d`, zero production lines changed** (four genuinely red; the
  fifth — "every URL in the committed manifest is a tracked file" — passes vacuously on a clean
  worktree and is a stale-`sw.js` guard, not evidence; the direct decision-58 reproduction is the
  third `sw-manifest` test, which puts an untracked file in the repo root and rebuilds).
  `logout()` now **awaits** `caches.delete('api-cache')` before the redirect — unawaited loses the
  race against `window.location.href` and leaves the previous user's rows on the phone.
  `checkAuth()` fails closed on identity via **two load-bearing mechanisms**: it evicts every
  cached `/api/v1/me` response *before* probing, and probes a cache-busted URL the URL-keyed cache
  can never answer (eviction alone loses to a concurrent SW write; busting alone leaves the stale
  bare-URL entry in place). An unverified probe paints no name — **deliberately not a redirect**,
  because the launcher must stay reachable offline on a truck that loses LTE routinely.
  **Two claims narrowed at G6 review, on executed counter-evidence — recorded rather than
  quietly dropped.** (a) The `removeUserHeader()` call in the fail-closed branch is **defence in
  depth, not a live fix, and it is uncovered**: `renderUserHeader` has exactly one call site (the
  `res.ok` arm), `checkAuth()` runs once at parse time, and there is no `pageshow`/
  `visibilitychange` re-entry, so a `.user-bar` cannot exist when that line runs. G6 deleted the
  line and all three new `index.spec.js` tests still passed, and an A/B with `/api/v1/me` aborted
  gave `user-bar=0 greeting=(none)` on **both** base and HEAD — the pre-fix `catch(e){}` already
  painted no name. The line is kept for a future second render path; the earlier phrasing
  ("strips any name already on screen") asserted behaviour that cannot fire and is withdrawn.
  (b) "Fail closed" means **closed on failure, never closed on a wrong identity**: `identityVerified`
  is set by any 200 and the client cannot tell whose 200 it is — G6 demonstrated a stale 200 still
  renders `Hi, Ghost Of User A` on both trees. The disclosure decision 57 actually names is closed
  by the eviction and the cache-buster, which are the halves that carry tests. Precache went
  **23 files / 1947.1 KB (`dev`) → 22 / 1455.6 KB (at the merge `c14865b`)**; the final merged tree
  is 22 / 1459.8 KB after Card B's regeneration.
  > **Corrected at morning triage 2026-07-27 (T-25 decision 76).** This card previously read
  > "23 files / 1949.7 KB → 22 / 1454.7 KB (−495.0 KB)". **1949.7 KB is unattainable at any
  > commit** — it is 1454.7 + the 495 KiB vendored bundle, i.e. back-computed, which is the
  > synthetic "before" figure G6 caught during the run. The HANDOFF carries the corrected pair;
  > this card was missed and kept the fake number. Manifest-derived actuals: `dev` 1947.1,
  > `77339ea` 1454.7, `c14865b` 1455.6, `c8f9733` 1457.7, final tree 1459.8.
  Decision 58 landed as a `manifestTransforms` filter against `git ls-files`, with
  dropped entries **logged, not swallowed**, and `version.json` allowlisted — it is git-ignored but
  `backend/Dockerfile:33-44` regenerates it into the image *precisely because* `sw.js` precaches it,
  a trap a naive tracked-set filter walks straight into. **The `runtimeCaching` block was NOT
  touched** — cache-key design is deferred by decision 57 to `sync-rxdb-schema-and-replication`, so
  the card's scope boundary held. `login.html` also untouched: an identity change without a logout
  is closed by the `checkAuth` eviction, not by the logout clear.
  Original card text:
  Three unrelated-in-cause but same-file changes:
  (1) **`caches.delete('api-cache')` on logout.** `logout()` (`index.html:141-145`) POSTs and
  redirects; there is no `caches.delete` anywhere in app code, so a shared truck phone serves the
  previous user's rows to the next user from a URL-only-keyed cache. Also fail `checkAuth`'s offline
  branch CLOSED on identity — `/api/v1/me` is on the same `NetworkFirst` route, so today user B can
  be shown user A's cached identity with no redirect. Bounded by cache being served only on network
  failure or >10s timeout, which on a food truck is routine, not exotic.
  (2) **`build-sw.js` globs the tracked set (`git ls-files`), not the working tree.** A Workbox
  precache entry that 404s fails the ENTIRE service-worker install; the symptom is "the PWA stops
  updating" with an invisible cause. `task sw` runs automatically under both `task test` and
  `task prod:deploy`, and `backlog-round.html` — untracked, disposed as FORK 2 in T-22 — is still in
  the repo root waiting to fire it.
  (3) **Drop the vendored bundle's `globPatterns` entry** (−495 KiB, −34% precache) until a page
  imports it; `sync-rxdb-schema-and-replication` re-adds it on adoption.
  Footprint: `index.html`, `build-sw.js`.

- **`workflow-offline-double-submit`** · **DONE** (2026-07-27, run `overnight-20260727`, serial
  after Wave 0; card authored at morning triage 2026-07-26 from ledger T-23 decision 60;
  **G6 REJECT on the first pass, repaired and re-verified — see "what G6 caught" below**) ·
  Fixed **client-side**, as decision 60 required: `submitChecklistToAPI` looks the template up in
  the durable IndexedDB `submitQueue` before building the payload and adopts that entry's
  **`idempotency_key`** instead of minting a fresh one. Both presses then land on the server's
  `ON CONFLICT (idempotency_key) DO UPDATE ... RETURNING id` upsert
  (`backend/internal/workflow/repository.go:722`) — a guard that was always correct and simply
  never reached, because the client handed it a UUID it had never seen. **Four tests captured RED
  first, each in its own commit on a tree with zero production lines changed** (`387eedd`,
  `cc03300`): DBL-02 is the original defect verbatim (`pendingApprovals` for the template: expected
  1, **received 2**).
  **The lookup reads IndexedDB, deliberately, not a module-level map** — a reload is an ordinary
  way for a crew member to produce the second press, and the queue is the only durable record that
  a submission is in flight. DBL-03 pins this by seeding an entry the page never enqueued.
  **🛑 What G6 caught, and it was a real data-loss blocker (F1).** The first pass also reused
  `payload.id`. That was **unauthorised scope** — decision 60 authorises reusing the KEY — and it
  was the half doing damage. `submitQueue` is keyed on `id` (`sync.js:52`), so reusing it turns
  `enqueueSubmission`'s `idbPut` from an append into a **REPLACE**. Offline, the queued payload is
  the **only durable copy** of what the crew member entered: `submitOp` (`sync.js:695`) does not
  queue and throws when offline, and `hydrateFieldState` (`workflows.html:1470-1476`) clears
  `FIELD_RESPONSES` and rebuilds from `DRAFT_RESPONSES` on every reopen. So press 2 built an
  **empty** payload and overwrote the answers — **one row, ZERO recorded responses, success toast**,
  inside this card's own headline scenario. Measured, shipped vs repaired:
  `server rows=1 payload=[0]` → `server rows=1 payload=[1]`. A food-safety checklist submitting
  with no answers is worse than the duplicate row the card set out to fix. **`id` reuse reverted;
  key-only reuse provably meets the goal on its own.** Two queue entries sharing one key is the
  **correct** shape — the server upserts only the fields present in each payload, so entry 1's
  answers survive and the empty entry 2 adds nothing. A merge that "tidies" this back to one entry
  reintroduces the loss.
  **The first pass's justification for the `id` half was factually FALSE and is corrected in both
  places it was written** (`workflows.html` comment and this card). It claimed key-only reuse would
  409 with `duplicate_submission` and be evicted at `sync.js:591-594`. Measured: the repeat POST
  returns **201 with the same submission id** and is evicted on the **success** path
  (`sync.js:583`). `duplicate_submission` appears **nowhere in `backend/`** — `sync.js:591` is dead
  code, now commented as such. It also called "submits the stale response set first" a downside;
  that is exactly the property that **saves** the data. The sign was inverted.
  **`sync.js` edited (G6 F5), outside the card's planned footprint, disclosed in the merge intent.**
  `drainQueue` now sorts by `queuedAt` before replaying. `idbGetAll` returns store-**key** order and
  the key is a random UUID; harmless while one template could only have one queued entry, but the
  key-reuse fix makes two entries upsert onto the same row, so replay order decides who wins a field
  both set. Entries predating `queuedAt` sort first (conservative).
  **Declared coverage limits.** DBL-04 (offline BEFORE the click) is the sole guard against the
  `id` half returning — re-adding it reds DBL-04 alone, and DBL-01/02/03 stay green, which is why
  the first pass shipped: **DBL-01/02 answer the field while ONLINE**, so a server-side draft
  repopulates press 2 and the loss is structurally invisible to them. Deleting the fix entirely
  reds all four. **UNCOVERED, declared not faked:** the `queuedAt` sort (it remedies a
  *non-deterministic* order, and the case this card produces — press 2 empty — is order-immune),
  and `findQueuedSubmission`'s IndexedDB-failure fallback (fail-safe by construction: returns null →
  caller mints a fresh key → pre-card behaviour).
  **KNOWINGLY ACCEPTED, recorded per decision 65 — not silent.** `findQueuedSubmission` matches on
  `template_id` with **no time bound**, so a queue entry that survives a failed drain across a day
  boundary is adopted by a later period's submit, upserting today's answers onto the older
  submission. **Reproduced by G6** (seeded `queuedAt 2026-07-20`, adopted by today's submit).
  **G6's ruling: defer to a follow-up card** bounding the lookup to the current period and ageing
  out stale entries — a `queuedAt` window is scope decision 60 does not authorise. Mitigation is
  real and G6 verified it: the row carries a "Pending sync" badge and the banner reads
  "1 submission pending sync", so the UI already presents the entry as *this checklist's* pending
  submission. Also accepted: the banner transiently reading "2 submissions pending sync" for one
  checklist.
  **Two behaviours were confirmed CORRECT and left alone,** per decision 60: the checklist staying
  editable after an offline submit, and the `err.offline` branch (now `workflows.html:2829`)
  returning to the list without pushing into `MY_SUBMISSIONS`. **`backend/internal/workflow` never
  opened** — the server-side duplicate guard reopens decision 49 and was the card's park trigger.
  **A finding this card first reported was REFUTED by G6, in the card's favour, and is withdrawn:**
  the claim that a submit *after* a successful drain would write a second row. G6 executed it — the
  submit button is **gone** after drain, because the live `SUBMIT_CHECKLIST` op drives
  `loadMyChecklists()` (`sync.js:442-458`) and the checklist flips read-only. It was reasoned from
  code, not run, and was wrong in the conservative direction.
  **Full suite, not a subset** (the slate required it, and the repair touched `sync.js`):
  **553 passed / 0 failed / 0 flaky / 6 skipped of 559 in 22.2m** — the 549/0/0/6-of-555 baseline plus
  exactly this card's four new tests, no regression. Go: `go build`, `go vet`, and
  `go test ./... -count=1 -p 1` all green (10 packages ok, `internal/workflow` among them).
  **G3 (openspec validate) DOES NOT APPLY** — hq has no `openspec/` tree and
  `night-crew workflow preflight` reports ABSENT.
  Original card text:
  Offline submit → reopen → submit again writes **two** `checklist_submissions` rows:
  `workflows.html:1656` mints a fresh `idempotency_key` per call and `:2778` handles `err.offline`
  by returning to the list without pushing into `MY_SUBMISSIONS`, so the checklist stays correctly
  editable and a second submit mints a second UUID past the `UNIQUE` guard. **Fix client-side —
  reuse the enqueued key on re-submit.** Explicitly NOT the server-side duplicate guard: that
  reopens decision 49 and trips the status card's park trigger for no added benefit. Footprint:
  `workflows.html`.

- **`precache-manifest-from-head`** · **DONE** (2026-07-28, run `overnight-20260729`, Wave 0;
  card authored at morning triage 2026-07-27 from ledger T-25 decision 67) · `build-sw.js` now
  globs **`git ls-tree -r --name-only -z HEAD`** — the commit — instead of `git ls-files`, which
  reads the **index**. `trackedFiles()`/`trackedOnlyTransform()` were renamed to
  `committedFiles()`/`committedOnlyTransform()` so the names cannot drift back to the weaker bar,
  and the dropped-entry warning now reads `skipped (not in HEAD)`.
  **One test captured RED first, on a tree with zero production lines changed** (`c64008b`): the
  staged-file reproduction returned
  `["zz-sw-manifest-staged-probe.html", "workflows.html", ...]` where the probe was asserted
  absent. It stages the probe and **only** stages it, and asserts both halves of the premise
  before building — probe present in `ls-files`, probe absent from `ls-tree HEAD` — so a green
  result cannot be vacuous the way a clean-worktree manifest assertion is.
  **`tests/sw-manifest.spec.js` test 1 moved to `ls-tree` too**, as the card required: on
  `git ls-files` it agreed with the bug, and an `sw.js` precaching a staged file would have passed
  it. Renamed to "every URL in the committed precache manifest is committed in HEAD".
  **`GENERATED_BUT_SHIPPED` was NOT dropped** and still holds exactly `version.json` — it is
  git-ignored, so it is in **neither** the index nor the commit, and `backend/Dockerfile:33-44`
  regenerates it into the image *precisely because* `sw.js` precaches it. A naive "read the commit"
  filter without the allowlist walks into the same 404 from the other side.
  **`-r` and `-z` are both load-bearing and are recorded as such:** `-r` recurses into trees
  (without it the manifest allow-set contains `icons`, not `icons/icon-96x96.png`), and `-z` gives
  NUL-separated **unquoted** paths — plain `--name-only` C-quotes any path with a space or a
  non-ASCII byte, which would silently drop a legitimately committed asset out of the precache.
  **The PARK trigger did not fire.** Checked rather than assumed: on a clean tree `git ls-files`
  and `git ls-tree -r HEAD` are byte-identical, and of the 22 manifest URLs exactly one
  (`version.json`, allowlisted) is absent from HEAD. Nothing the image ships is newly excluded, so
  no Dockerfile question arose. `prod:deploy` was re-read at implementation and matches the card
  verbatim — `git reset --hard origin/main` then `docker compose build`, no `task sw` on the box.
  **`globPatterns` and `runtimeCaching` were NOT touched**: the vendored RxDB bundle stays out
  (decision 59) and cache-key design stays deferred (decision 57). No manual SW cache-version bump
  — Workbox content-hashes every entry, and CLAUDE.md forbids hand-bumped keys. Precache held at
  **22 files / 1463.6 KB**; the only manifest delta is `version.json`'s revision hash
  (`0966d7cf` → `8270ae36`), which moved because the frontend semver did.
  **Full suite, not a subset: 561 passed / 0 failed / 0 flaky / 6 skipped of 567 in 24.7m**
  at `--retries=0`, **20 spec files** (`npx bddgen` run first — `task test` omits the `bdd:gen`
  dependency and would have silently run 19 of 20, B-09). Both known-armed pre-existing reds
  (`sync.spec.js:446` LST-17 and `:1198`) passed this run. Go: `go build`, `go vet`, and
  `task test:go` all green (10 packages ok). Frontend **1.2.0 → 1.2.1** (patch — build-correctness
  only, no user-visible frontend behaviour); backend unchanged at 0.2.2.
  **G3 (openspec validate) DOES NOT APPLY** — hq has no `openspec/` tree and
  `night-crew workflow preflight` reports ABSENT.
  Original card text:
  **`build-sw.js` globs `git ls-files`, which reads the git
  INDEX.** A staged-but-uncommitted file enters the precache manifest, and a precached URL that
  404s fails the **entire** service-worker install for every returning client. Reproduced end-to-end
  at triage: `git add zz-adv27-staged.html && node build-sw.js` → `23 files precached`, the file
  present in `sw.js`; `git ls-tree -r --name-only HEAD` excludes it. **The trigger path is
  complete:** `task prod:deploy` (`Taskfile.yml:174-210`) does **not** run `task sw` on the box — it
  `git reset --hard origin/main` then `docker compose build` — so the *committed* `sw.js` ships.
  **This AMENDS decision 58's literal text** ("the tracked set (`git ls-files`)"), serving that
  decision's intent against its own letter; the run correctly refused to make the change itself.
  **Red-first test is the point of the card:** stage a file, assert it is absent from the manifest.
  Note `tests/sw-manifest.spec.js` test 1 currently uses the same `git ls-files`, so it must move to
  `ls-tree` too or it will keep agreeing with the bug. **Do NOT drop the `GENERATED_BUT_SHIPPED`
  allowlist** — `version.json` is git-ignored but `backend/Dockerfile:33-44` regenerates it into the
  image precisely because `sw.js` precaches it. Footprint: `build-sw.js`, `tests/sw-manifest.spec.js`,
  regenerated `sw.js`.

- **`workflow-queue-period-and-failnote-upsert`** · **DONE** (2026-07-28, run `overnight-20260729`,
  concurrent track; card authored at morning triage 2026-07-27 from ledger T-25 decision 71) ·
  All four items landed; **the PARK trigger did NOT fire.**
  **(1) Fail-note duplicates.** Migration `0071_submission_fail_notes_unique.sql` adds
  `UNIQUE (submission_id, field_id)`, and `repository.go`'s fail-note insert gains the matching
  `ON CONFLICT ... DO UPDATE SET note = EXCLUDED.note, severity = EXCLUDED.severity`. **Both halves
  are load-bearing and the card says so in three places**: the index alone turns a silent duplicate
  into a hard 500 on the second POST, which is strictly worse than the bug. **`photo_url` is
  deliberately NOT in the SET list** — it is absent from the INSERT column list, so
  `EXCLUDED.photo_url` is NULL and writing it would erase an attached photo on every re-POST;
  nothing repopulates it (the correction photo travels on the RESPONSE value as
  `_correction_photo`). A third test pins that omission.
  **The PARK trigger was CHECKED, not assumed, BEFORE the migration was written.** Queried the live
  Postgres on `:5433`: the `production` schema (prod's, per `docker-compose.prod.yml:41`) and
  `public` (dev's) both hold **0 rows** in `submission_fail_notes` — hence **0**
  `(submission_id, field_id)` duplicates, and no dedup rule had to be chosen. The migration is bare
  `CREATE UNIQUE INDEX IF NOT EXISTS` and carries a comment naming the check, what it found, and
  what to do if some other environment ever fails on it: **the dedup rule is an operator decision,
  do not add a `DELETE` unattended.** Rows with `submission_id IS NULL` stay unconstrained (Postgres
  treats NULLs as distinct) — `unsubmitChecklist` detaches fail notes to NULL and nothing
  re-attaches them, since they carry no `answered_by`. **Those orphans are a real, separate,
  pre-existing defect this card does NOT fix** and the index neither fixes nor collides with them.
  **(2) Cross-period stale queue entry.** `enqueueSubmission` (`sync.js`) now stamps `period` beside
  `queuedAt`, from a new `currentSubmitPeriod()` — the SAME
  `new Date().toISOString().slice(0, 10)` the checklist list and the runner already use for "already
  submitted today", named once so the queue and the list cannot drift on what "today" means.
  `findQueuedSubmission` requires a new `isCurrentPeriodEntry` predicate in addition to the
  `template_id` match. **Aging out is RETIREMENT FROM KEY REUSE, never deletion — a judgment call
  made explicitly, not by omission.** Offline, a queued payload is the only durable copy of what the
  crew member entered that day (the same reason `id` is not reused), and a submission queued Monday
  that finally drains on Thursday is CORRECT, not garbage; a delete would be the silent data loss
  the DBL-04 comment exists to forbid. A stale entry keeps draining normally, it just stops being a
  source of keys. An entry with **no** `period` (queued by older code) is treated as NOT current —
  conservative in the direction that matters; the transient deploy-boundary cost is written into
  the code rather than left to be rediscovered.
  **(3) The durable falsehood was made TRUE, not edited away.** "The server upserts only the fields
  present in each payload" held for `submission_responses` and was false for
  `submission_fail_notes`; item 1 makes the sentence accurate, and the comment now records that both
  tables upsert and why `photo_url` is excluded.
  **(4) Vocabulary collision — decided inside the card, applied to BOTH files.**
  `"Queued"` (`.sync-badge`, sync.js) for a whole queued **submission**; `"Unsaved"`
  (`.unsaved-mark`, workflows.html — the class was renamed too, not just the string) for one unsent
  **field** answer; banner `"N submissions queued to send"`. Chosen so the SCOPE is readable without
  a legend, and because these two ARE on one screen at once — `#sync-banner` sits in the same `#s1`
  panel as `#fill-body`. Both files carry a 🛑 VOCABULARY block naming both states and pointing at
  the other file, so the collision cannot be reintroduced from one side.
  **Red-first, twice, each failing on the defect rather than on a setup step.** Go
  (`failnote_upsert_test.go`, commit `5122b0d`): `fail_note_rows=2` where 1 was wanted, with the
  premise — both POSTs return the SAME submission id — asserted first so a green cannot be vacuous.
  Playwright `[DBL-05]` (commit `56a913d`): a three-day-old entry lent its key to today's submit.
  **DBL-03 is the positive control** proving the period bound did not over-block; `[VOC-01]` drives
  both badge states onto one screen and asserts the collided string is gone from the app entirely.
  **Full suite, not a subset: 563 passed / 0 failed / 0 flaky / 6 skipped of 569 in 24.7m**
  at `--retries=0`, **20 spec files** (`npx bddgen` run FIRST — `task test` omits the `bdd:gen`
  dependency and would have silently run 19 of 20, B-09). 569, not 567: Card A's baseline plus
  exactly this card's two new tests. **Both known-armed pre-existing reds passed** this run.
  **Two earlier runs were discarded and why is recorded, because a green claim built on either
  would have been false.** Run 1 was aborted; run 2 inherited its residue — `hq_test_e2e_b` was
  never reset, and `task test` DROPs and RECREATEs its database precisely because this suite shares
  one DB across every spec file. It reported three failures
  (`inventory.spec.js:2357` cutoff draft, `:3541` vendor pagination, `onboarding.spec.js:1179` FAQ
  counts) — all order-sensitive, none reproducible on a fresh DB, none related to this card.
  **Isolated before attribution, not after**: `sync.spec.js:446` [LST-17] red under load with its
  known signature (`expected "0/1", received "1/1 items"` — a cross-device broadcast, nothing to do
  with the queue) and **1/1 green isolated**; and one genuine defect **in this card's own test**,
  not in the app — `[VOC-01]`'s badge locator matched two elements because `renderSyncBanner`
  selects `[data-template-id]` document-wide and the Builder tab renders those too. Both badges
  read "Queued"; the contract held, the locator was under-specified, and it is now scoped to
  `#checklist-list`. **That the queued badge also lands on a Builder row is a pre-existing cosmetic
  quirk of `renderSyncBanner`, is NOT this card's, and is left un-fixed and written down.**
  Go: `go build`, `go vet`, and
  `go test -p 1 ./...` against this card's own `hq_test_go_b` all green (10 packages ok,
  `internal/workflow` among them). Backend **0.2.2 → 0.2.3**, frontend **1.2.1 → 1.2.2** (both
  patch; frontend bumped from Card A's Wave 0 value, not 1.2.0), `package.json` mirrored exactly,
  `sw.js` regenerated — **22 files / 1468.9 KB**, no manual cache-version bump, `globPatterns` and
  `runtimeCaching` untouched.
  **G3 (openspec validate) DOES NOT APPLY** — hq has no `openspec/` tree and
  `night-crew workflow preflight` reports ABSENT.
  Original card text:
  (1) **`submission_fail_notes` duplicates.** Measured at triage, not reasoned: the same payload
  POSTed twice with one `idempotency_key` → `201`/`201` with an **identical submission id**, and
  `submission_rows=1 response_rows=1 fail_note_rows=2`. The table has no unique constraint
  (migration `0013`) and its insert is bare (`repository.go:760-767`) while the responses insert
  directly above it carries `ON CONFLICT (submission_id, field_id) DO UPDATE`. An approver sees the
  same note twice. Fix: matching `ON CONFLICT` + unique index.
  (2) **Cross-period stale queue entry.** `findQueuedSubmission` filters on `template_id` only,
  queue entries carry no period, and nothing ages them out — so a persistently-failing server lets a
  stale key adopt a **later day's** submit, upserting today's answers onto the older row. Bounded
  (any successful drain clears it) and visible (banner + "Pending sync" badge), which is why G6
  deferred rather than parked. Fix: bound the lookup to the current period, age out stale
  `submitQueue` entries.
  **🛑 Also folds the "Pending sync" vocabulary collision (added 2026-07-28, commit `bc8721e`).**
  `sync.js:642` renders a `.sync-badge` reading **"Pending sync"** for a queued whole-checklist
  *submission*; `workflows.html` now renders a `.pending-sync-mark` reading the **same two words**
  for a single field answer that has not reached the server. Two different states, one string, both
  reachable on the same screen. This card owns both files and both mechanisms, so it names them
  apart — decide the vocabulary, do not let two independent authors keep the collision.
  **🛑 This card also repairs a durable falsehood by making it TRUE.** `workflows.html:1781`
  (**anchor re-checked 2026-07-28 — was `:1694` before commit `bc8721e` moved it; `:1694` is now
  `setTextAnswer`**),
  Card B's merge-intent note and the `workflow-offline-double-submit` card all assert *"the server
  upserts only the fields present in each payload."* That holds for `submission_responses` and is
  **false for `submission_fail_notes`** — Card B's design made D-4's trigger the normal path inside
  the same comment that says it cannot happen. The comment was deliberately **not** edited at triage
  (touching `workflows.html` moves `sw.js` via Workbox's per-entry revision hash and would oblige a
  full suite re-run for a comment); this card makes the sentence accurate instead. Footprint:
  `backend/internal/workflow`, a migration, `workflows.html`, `sync.js`.

- **`test-harness-fail-loud`** · **DONE — landed on `overnight-20260729-2` (Track A, card H1)** ·
  **PROMOTED from BACKLOG B-09 + B-16(b) at the 2026-07-28 slate-planning
  session under the §15k architecture-blocking bar (ledger T-27 decision 90).** Make a broken test
  environment **fail** instead of **pass**. Two mechanisms, one theme, and the theme is the point:
  `overnight-20260729` produced **three** silent-greens in one run, which the closeout correctly
  called the shape of this repo's harness rather than a coincidence.
  (1) **`task test` runs 19 of 20 spec files.** Add `bdd:gen` to `test:`'s `deps`
  (`Taskfile.yml:28-30`), matching `bdd:` (`:78`) and the CI task (`:102`) which already carry it.
  (2) **A dropped or unreachable database reports `ok`.** Where `DB_TEST_URL` is **set but
  unreachable**, `t.Fatalf`; keep `t.Skip` only for **unset**. This is the asymmetric gate this repo
  already built and proved in-tree at `backend/internal/sync/proxy_live_test.go:107-118`
  (`HQ_SYNC_SPIKE_LIVE` set + port dead ⇒ FAIL) — **copy that pattern, do not invent a second one.**
  Sites measured 2026-07-28 and **repo-wide, which is broader than B-16 states**:
  `recipes/helpers_test.go:30,35,39`, `workflow/stable_identity_test.go:95,201,240`,
  `workflow/requires_approver_test.go:81,110,161`, `inventory/sync_receipts_test.go:20,118,176,225`,
  `receipt/worker_test.go` (×17), `sync/access_test.go:29,33`, `sync/jwtbridge_test.go:169,173`.
  **Red-first is the whole card** — (a) delete the generated spec files, run `task test`, assert it
  regenerates and reports 20 (before the fix: 19); (b) point `DB_TEST_URL` at a dead port, assert
  the Go suite **FAILS** (before the fix: `ok`). A card that cannot show these two reds has proven
  nothing, because "the suite is green" is precisely the claim under suspicion. **Do NOT convert
  the `unset` case** — skip-on-unset is deliberate so a contributor without a database can still run
  unit tests; the bug is the *symmetry*, not the skip. **Scope discipline:** this fixes the harness,
  not any test the harness newly reveals as failing — park those with evidence. **B-16(a)** (reviewer
  prompts must forbid dropping a database the reviewer did not create) is **not** in this card; it is
  standing G6 dispatch text. Footprint: `Taskfile.yml`, `backend/internal/*/**_test.go`.
  **Landed as planned, with both reds observed first at `82350cc`.** (a) `npx playwright test
  --list` went `Total: 568 tests in 19 files` → `Total: 569 tests in 20 files`; the 20th is
  `.features-gen/features/user-invite-onboarding.feature.spec.js`. (b) All seven DB-backed packages
  reported `ok` in **under 1.3s each** against a `DB_TEST_URL` naming a database that does not
  exist; they now exit 1 with the DSN, the stage (connect vs ping) and the reason in the message.
  Two deviations, both disclosed in the merge-intent note: the asymmetry lives in **one** new
  test-only package, `backend/internal/testdb` (outside the stated footprint) rather than in eight
  inline copies — eight copies would be eight patterns; and the per-test `t.Skip` lines the card
  enumerates in `requires_approver_test.go` / `sync_receipts_test.go` / `worker_test.go` are
  **subsumed, not edited**, because they all key off the package-level `testPool == nil` that the
  five converted `TestMain`s set. Reproducible via `scripts/verify-test-harness.sh`, whose third
  check (B2) guards the *other* direction: `DB_TEST_URL` unset must keep skipping.

- **`app-timezone-unify-new-york`** · **PLANNED — HIGH, small/medium** (new card, morning triage
  2026-07-28, ledger T-26 decision 83) · **The app is running two conflicting timezone regimes, and
  the operator ruled the app's timezone is `America/New_York`.** `users.DefaultTimezone` is already
  New York — as are the Users-tab picker, the purchasing handler and scheduler fallbacks, and
  `playwright.config.js` — while **`America/Chicago` is hardcoded in the money paths**:
  `inventory/handler.go` (×6 — the COGS period-summary window **and** the completeness gate feeding
  sales-processor's weekly payroll), `inventory/trends.go:240`, `purchasing/service.go:60`
  (`CurrentWeekStart`, the Monday every purchasing week hangs off), `recipes/cost.go:103`,
  `recipes/scheduler.go:52` (drift check fires Mon 09:00 Chicago), migrations `0037`/`0042` as column
  defaults, and `purchasing.html:295`, which **actively writes** `America/Chicago` into cutoff config
  the backend would otherwise default to New York. **Blast radius, stated precisely:** the COGS date
  filters are `COALESCE(event_date, created_at AT TIME ZONE 'America/Chicago')`, so only rows with no
  extracted `event_date` are exposed — bounded. But `CurrentWeekStart` and the recipe cost week are
  **unconditional** Chicago, so every weekly boundary is currently an hour off the operating day.
  **Scope: one card, all sites** — piecemeal leaves two boundaries disagreeing, which is exactly
  today's bug. Move every hardcoded zone to a shared constant mirroring `users.DefaultTimezone`, and
  fix `purchasing.html` writing Chicago. **Fix forward only** — past weekly COGS/payroll figures were
  already acted on and are NOT restated; the card must note the changeover date so a future reader
  knows why one boundary moves exactly once. **Also folds D-1's original finding:**
  `currentSubmitPeriod()` (`sync.js:565`) and `isCurrentPeriodEntry` (`workflows.html:1758-1762`) are
  UTC, plus three pre-existing "already submitted today" comparisons at `workflows.html:2274`,
  `:2308`, `:2674` — reproduced at triage under a frozen clock: a queue entry stamped 6:30pm CT
  reuses its key at 6:45pm and refuses at 7:30pm, same weekday throughout, so an offline double-press
  straddling the rollover yields two submission rows for one operational evening.
  Footprint: `backend/internal/{inventory,purchasing,recipes}`, `sync.js`, `workflows.html`,
  `purchasing.html`, a migration for the two column defaults.

- **`sync-rxdb-collections-and-table-contract`** · **DONE — landed on `overnight-20260729-2` (Track B, card B1), 2026-07-29.**
  Three new files, no npm dependency, nothing outside the footprint:
  `sync-schema/collections.js` (the four replicated collection schemas + the LOCAL
  conflict record + `CONFLICT_RECORD_RETENTION_DAYS`), `sync-schema/sql/0001_sync_tables.sql`
  (all six items of the self-hosted per-table contract, for four tables, **with no
  `CREATE POLICY` — RLS is enabled with zero policies, i.e. deny-all, until
  `sync-rxdb-row-visibility-rls` lands**), and `tests/sync-schema.spec.js` (28 tests,
  red-first at `701fb52` with 27 failing). A fourth file, `sync-schema/package.json`,
  scopes `"type": "module"` to that directory and declares no dependencies.
  The schemas are **plain data**, validated by tests that need no RxDB runtime — the
  `rxdb` and `@supabase/supabase-js` packages stay unpinned and belong to
  `sync-rxdb-replication-and-conflict-handler`. The mirror is one-for-one:
  `templates`→`checklist_templates`, `checklists`→`checklist_submissions`,
  `responses`→`submission_responses`, `approvals`→`submission_rejections`.
  Sections/fields/schedules/assignments/fail-notes are deliberately not mirrored —
  `checklists.template_snapshot` makes a filled checklist self-contained offline, and
  `template_assignments` is B2's RLS input rather than a replicated collection.
  `lamport_ts` is not carried across on either side. The PARK trigger did **not** fire:
  none of the four mirrored tables carries money, and the one number crossing the schema
  is a temperature reading inside `responses.value`.
  · **✅ FORK RESOLVED 2026-07-28 — the durable conflict record is a personal, per-device undo, stored local-only (ledger T-27 decision 89).**
  The question raised at morning triage 2026-07-28 was where the record of an overwritten answer
  lives. The product question put to the operator: an *audit trail a manager can see*, or a
  *personal undo for the person holding the phone*? **Operator answer: personal undo, per-device.**
  ⇒ **A local RxDB collection. No server table, no endpoint, no replication of the conflict record
  itself** — which keeps the signed mockup's contract literally true (UI-SPEC: *"no new sync
  plumbing … no server endpoint"*) rather than quietly widening it. **But the shape is declared
  replication-ready:** it carries `submission_id`, `field_id`, the discarded value, and the same
  who-and-when decision 79 already requires the replicated rows to carry, so promoting this to a
  cross-device audit trail later is *adding a table and a policy*, not a redesign.
  **The consequences the operator surfaced at triage stand and are accepted, not mitigated:** the
  record is **per-device** (a manager cannot see that a crew member's food-safety reading was
  overwritten), **evictable** (iOS storage pressure destroys it, which is why a storage-error plate
  exists), and lost on reinstall. **Retention** stays 30 days as a **local** sweep — the number
  itself is reopened and belongs to `sync-rxdb-conflict-notice-mockup-amendments`, so read it from
  **one named constant** rather than scattering `30` through the code. Decision 80 left this as
  "the UI card's own call"; it lands here instead, where the table contract is being written.
  (fanned out of
  `sync-rxdb-schema-and-replication` 2026-07-28; foundation, both siblings below depend on it) ·
  Define the RxDB collections for **checklists, templates, responses, approvals** mirroring the
  current Postgres domain model, each satisfying the self-hosted table contract recorded above.
  Schema only — this card wires no replication and writes no policy; it is what the other two
  cards stand on.
  **🛑 It owns two schema declarations that other cards read and must not re-litigate:**
  (a) **`_modified` is NOT declared — decision 78.** Keeping it out leaves it a pure pull cursor.
  Declaring it pulls `_modified` into `addDocEqualityToQuery`'s compare-and-swap so **any**
  server-side touch becomes a conflict including ones where no answer changed (W2 sharp edge 11),
  which routes ordinary bookkeeping into the conflict-notice UI's *"a change we couldn't identify"*
  row — the one row from which nothing can be recovered. With a field-level three-way merge doing
  the real work on the replication card, the tightened detection buys little and costs the worst
  screen in the set. Recorded so it is decided, not inherited by whether someone copied the field
  in — which is exactly what the dissolved card demanded.
  (b) **Rows CARRY who-and-when — decision 79.** UI-SPEC §"Explicitly NOT decided here" names this
  as the declaration that makes the conflict sheet's attribution line real or fictional: without
  it, *"Dana M., 6:12 PM"* degrades to *"someone else"*. The product's stated core value is
  **accountability — who checked what**, the signed mockup draws attribution on the *Now shows*
  line, and the cost is two columns. Carry them.
  (c) **The conflict record is a LOCAL collection — decision 89 (2026-07-28).** Declare a local
  RxDB collection for discarded values: `submission_id`, `field_id`, the discarded value, and the
  same who-and-when (b) requires. **No server table, no endpoint, no replication of this
  collection.** Retention is a **local** 30-day sweep read from **one named constant** — the number
  is reopened and belongs to `sync-rxdb-conflict-notice-mockup-amendments`, so do not scatter `30`
  through the code.
  Footprint: the sync DB schema (SQL) and the RxDB collection definitions. No `workflows.html`, no
  policies, no client construction.

- **`sync-rxdb-row-visibility-rls`** · **PLANNED — SLATE-READY** (fanned out of
  `sync-rxdb-schema-and-replication` obligation 1 at the 2026-07-28 dissolution; depends on
  `sync-rxdb-collections-and-table-contract` — there must be tables to protect) · **Obligation 1,
  decision 55, unchanged in substance.** **PORT `ResolveEntityAccess`
  (`backend/internal/sync/ops.go:474`) — do not invent a predicate:** project
  `template_assignments ⋈ users` into the sync DB the way `hq_grant_projection` projects grants,
  and express RLS as an `EXISTS` against it. **Two inherited properties are knowing, not
  accidental:** the resolver never filters on `assignment_role` (an `'approver'` sees what an
  `'assignee'` sees), and the `roles && ARRAY['admin','superadmin']` arm is unconditional (every
  admin sees every template). **Changing either is a SEPARATE card** — do not vary substrate and
  permission semantics in one night. The projection is written **push-on-grant-change, in the same
  transaction as the `app_permissions` mutation** (decision 61); reconcile reintroduces the exact
  replay window the projection exists to eliminate.
  **🛑 THIS IS THE CARD THAT UNLOCKS THE DOOR — and until it lands the door is unlocked and
  unguarded.** `sync-proxy-endpoint` forwards **every method** to PostgREST with a
  `role: authenticated` token and no row filtering of its own, deliberately, because filtering was
  always meant to be this card. **`HQ_SYNC_REST_URL` must not be set in any deploy until this card
  lands** — doing so gives every logged-in crew member full read AND write on the whole exposed
  schema, a dishwasher can `PATCH` a template, with nothing in between.
  `HQ_SYNC_REALTIME_URL` is the safe half to adopt first: Realtime is read-only, so a subscription
  without RLS leaks reads but authors nothing. Recorded identically in
  `backend/internal/sync/proxy.go`'s env-var comment.
  **Its gate is an attack-variant suite, red-first, modelled on the 16/16 `sync-jwt-bridge-endpoint`
  ran** — prove RLS **discriminating**, not merely responding: a 200 alone proves nothing, and a
  `service_role` BYPASSRLS control is what rules out *"the table was empty."* Reuse the two findings
  that card banked and do not rediscover them: `auth.uid()` is wrong for HQ (read the **plural** GUC
  via `public.hq_jwt_claim`), and `public.hq_uid_trap` re-proves it every run.
  Footprint: sync DB policies/SQL, `backend/internal/sync`, a migration if the projection needs one.

- **`sync-rxdb-replication-and-conflict-handler`** · **PLANNED — SLATE-READY** (fanned out of
  `sync-rxdb-schema-and-replication` 2026-07-28; depends on
  `sync-rxdb-collections-and-table-contract`. **Parallel-safe with `sync-rxdb-row-visibility-rls`** —
  disjoint footprints, one is frontend/client, the other SQL/backend) · Wire RxDB's Supabase
  replication plugin client-side against **`/sync/rest/*` and `/sync/realtime/*` on HQ's own
  origin** (decision 69, same-origin, the door landed 2026-07-28). Do **not** fetch
  `/api/v1/sync/token` and attach a bearer — the proxy mints per request and injects it, and a
  client-supplied `Authorization`/`apikey` is deliberately discarded. Realtime is reached at
  `/sync/realtime/socket/websocket?vsn=1.0.0`; **do not add an `apikey` parameter, the door sets
  it.** Inherits from `sync-jwt-bridge-endpoint` (decision 51, address moved only): the permanent
  **client-construction helper** using `global.fetch` + `realtime.transport`, stay gateway-less, no
  Kong — and the **`@supabase/supabase-js` pin plus a smoke test that fails loudly on upgrade**,
  since the coupling is to how the library derives `<baseUrl>/rest/v1`, not to a public extension
  point.
  **The `conflictHandler` is this card's real work (decision 50).** RxDB's default is unconditional
  **master-wins** — not the last-write-wins the explore session assumed — no clock participates, and
  a strictly-later local write is discarded silently (reproduced four times). The decided policy is
  a **field-level three-way merge**: different fields edited by different people all survive; only a
  genuine same-field clash falls back to master-wins, **and then `conflict$` must surface it with
  the discarded value recoverable**. Tractable because `assumedMasterState` is in
  `RxConflictHandlerInput` (`conflict-handling.d.ts:10`) — diff fork-vs-assumed and
  master-vs-assumed to know who changed what. It is **optional in the type**, so the rule needs a
  defined fallback when absent. `conflict$` fires **per document** and carries the document id
  (`upstream.js:333`). **Headless and testable** is a requirement, not a nicety — the sibling UI
  card's whole design assumes this handler's behaviour.
  **Obligation 4 — umbrella slugs, decision 56.** The client-construction helper must expand
  umbrella slugs so the launcher shows the per-tab surfaces the user can actually reach
  (`inventory` ⇒ `inventory-trends`, `inventory-cost`).
  **Obligation 5 — the vendored bundle, decision 59, AND ITS TRAP.** This is where a page finally
  imports `vendor/rxdb.bundle.js`, so this is where the `globPatterns` precache entry is re-added.
  **🛑 RE-ADDING THE GLOB ALONE BREAKS PRODUCTION, verified at source.** `backend/Dockerfile` COPY
  lines are `21`, `25`, `26` (`icons`), `27` (`lib`), `30` (`backend`) — **`vendor/` is never copied
  into the image.** Morning triage 2026-07-27 simulated the image staging independently: all 22
  current precache URLs present, `vendor/rxdb.bundle.js` **absent**. A precached URL that 404s fails
  the **entire** service-worker install for every returning client — the exact bug
  `pwa-cache-and-build-hygiene` fixed. **This card must add `vendor/` to the Dockerfile in the same
  change set, or not re-add the glob.**
  Footprint: new RxDB client layer, `workflows.html` (import + construction only — the write-path
  swap is `sync-hard-cutover`), `build-sw.js`, `backend/Dockerfile`, `vendor/`.

- **`sync-cache-and-identity-hygiene`** · **PLANNED** (fanned out of
  `sync-rxdb-schema-and-replication` obligations 3, 7 and 8 at the 2026-07-28 dissolution; depends
  on `sync-rxdb-replication-and-conflict-handler` — the `api-cache` retirement is only correct once
  offline data actually comes from IndexedDB) · **The split is a gain here, and the dissolved card
  said why:** it carried these under an *"accepted cost: if this card slips, they slip with it."*
  They no longer ride the biggest card in the cycle, and three of the four items are cross-tenant
  disclosures.
  **Obligation 3 — `api-cache`, decision 57, structural half.** The URL-only cache key with no
  `Vary` (`build-sw.js:60-78`) is a cross-tenant read. The immediate mitigation already shipped in
  `pwa-cache-and-build-hygiene`; this card owns the design, and **the expected answer is to retire
  the route entirely** — once RxDB replicates, offline data comes from IndexedDB and `api-cache` is
  obsolete.
  **Obligation 7 — two more `api-cache`-shaped disclosures, decision 70.** (a)
  `localStorage['hq_apps']` is never cleared on logout, and `index.html:224` still parses the
  previous user's cached slug list in the fail-closed branch — offline on a shared truck phone,
  user B sees user A's tiles. (b) An identity change *without* a logout (B logs in while A's session
  is live) never runs `logout()`, and `login.html` does no cache hygiene of its own. Both UI-only;
  server-side grants remain the real gate.
  **Obligation 8 — the stale comment at `tests/sync.spec.js:1584`** (decision 66), waiting for
  "the next card touching that file" since 2026-07-26.
  Footprint: `build-sw.js`, `index.html`, `login.html`, `tests/sync.spec.js`.

- **`sync-rxdb-schema-and-replication`** · **🛑 DISSOLVED 2026-07-28 — this card no longer exists as
  a slateable unit.** It had been fanned out twice already (browser delivery 2026-07-26,
  `sync-proxy-endpoint` 2026-07-28) and was **still** carrying four independent mechanisms plus
  eight obligations. The §1 split rule — *a card bundling multiple mechanisms is split before it
  enters the slate* — applies, and an unattended run must never discover mid-night that a card is
  four cards. **Nothing in its scope is dropped**; every obligation is re-homed below, and the
  original text is kept verbatim underneath as the record of why each one exists.

  | Was | Now | Depends on |
  |---|---|---|
  | Collections + per-table SQL contract | **`sync-rxdb-collections-and-table-contract`** | — (foundation) |
  | Obligation 1 — row visibility (decision 55) | **`sync-rxdb-row-visibility-rls`** | collections |
  | `replicateSupabase` wiring, `conflictHandler`, client helper, obligations 4 + 5 | **`sync-rxdb-replication-and-conflict-handler`** | collections |
  | Obligations 3 + 7 + 8 — `api-cache`, identity hygiene, stale comment | **`sync-cache-and-identity-hygiene`** | replication |

  **The activation-order interlock survives the split and now spans two cards, which makes it
  easier to get wrong, not harder.** `HQ_SYNC_REST_URL` must not be set in any deploy until
  **`sync-rxdb-row-visibility-rls`** lands — see that card. The `_modified` call the original text
  demanded be *decided rather than inherited* is **decided: not declared** (decision 78); it is a
  spec line on the replication card now, not an open question.

  *(Original card text follows, for the record.)* · **PLANNED** (depends on
  `sync-spike-stack-and-jwt-bridge`'s
  go-decision reaching `ledger.md` — **cleared**; and now on **`sync-rxdb-browser-delivery-spike`'s
  verdict**, which sizes the storage/service-worker half; sized by `sync-spike-rxdb-replication`
  where it ran) · **Scope reduced 2026-07-26 by the fan-out above:** browser delivery/storage/SW
  moved to the browser spike, and the user-visible conflict notice moved to
  `sync-rxdb-conflict-notice-ui`. What remains here: the collections, the per-table SQL contract,
  the `replicateSupabase` wiring, the **`_modified` semantics call** (declaring it makes the plugin
  round-trip the server timestamp *and* pulls `_modified` into the compare-and-swap, tightening
  conflict detection so any server-side touch is a conflict; leaving it out keeps it a pure pull
  cursor — decide it, do not let it be decided by whether someone copied the field in), and the
  **headless, testable `conflictHandler`**. Also inherits the client-construction helper and the
  `@supabase/supabase-js` pin + upgrade smoke test from `sync-jwt-bridge-endpoint` (decision 51
  substance unchanged; this is where the client is actually constructed). Define
  RxDB collections for checklists, templates, responses, and approvals (mirroring the current
  Postgres domain model), each satisfying the self-hosted table contract above. Wire RxDB's
  Supabase replication plugin client-side. **🛑 UNBLOCKED AND RE-SCOPED at morning triage
  2026-07-25 (ledger T-22 decision 50) — the conflict policy is this card's real work.** The
  explore session's *"last-write-wins, no custom conflict handler"* is struck: RxDB's default is
  unconditional **master-wins**, no clock participates, and a strictly-later local write is
  discarded silently (reproduced four times, most recently at triage). The decided policy is a
  **field-level three-way merge**: different fields edited by different people all survive; only a
  genuine same-field clash falls back to master-wins, **and then `conflict$` must surface it to
  the user with the discarded value recoverable**. Tractable because `assumedMasterState` is in
  `RxConflictHandlerInput` (`conflict-handling.d.ts:10`) — diff fork-vs-assumed and
  master-vs-assumed to know who changed what. It is **optional** in the type, so the rule needs a
  defined fallback when absent. `conflict$` fires **per document** and carries the document id
  (`upstream.js:333`), correcting W2's caveat.
  **🛑 Five obligations added at morning triage 2026-07-26 (ledger T-23):**
  (1) **Row visibility — decision 55.** PORT `ResolveEntityAccess`
  (`backend/internal/sync/ops.go:474`), do not invent a predicate: project
  `template_assignments ⋈ users` into the sync DB the way `hq_grant_projection` projects grants,
  and express RLS as an `EXISTS` against it. Two inherited properties are knowing, not accidental:
  the resolver never filters on `assignment_role` (an `'approver'` sees what an `'assignee'` sees),
  and the `roles && ARRAY['admin','superadmin']` arm is unconditional (every admin sees every
  template). Changing either is a SEPARATE card — do not vary substrate and permission semantics in
  one night.
  (2) **Origin shape — decision 62.** DECLARE same-origin-fronted vs cross-origin as the card's
  FIRST spec line, and **cost the reverse proxy** if same-origin. It is unbuilt: `browser/serve.mjs`
  invents it for the harness only. Cross-origin moots verdict item 7 and inverts item 1 (W2's
  `global.fetch` shim returns).
  (3) **`api-cache` — decision 57, structural half.** The URL-only cache key with no `Vary`
  (`build-sw.js:60-78`) is a cross-tenant read. The immediate mitigation ships separately (see
  `pwa-cache-and-build-hygiene`); this card owns the design, and the expected answer is to **retire
  the route entirely** — once RxDB replicates, offline data comes from IndexedDB and `api-cache` is
  obsolete.
  (4) **Umbrella slugs — decision 56.** The client-construction helper must expand umbrella slugs,
  so the launcher shows the per-tab surfaces the user can actually reach (`inventory` ⇒
  `inventory-trends`, `inventory-cost`). Closed by the standing per-tab-granularity convention.
  (5) **Vendored bundle — decision 59.** Re-add the `globPatterns` precache entry here, when a page
  actually imports the bundle. It was excluded meanwhile.
  **🛑 TRAP, verified at source — re-adding the glob ALONE BREAKS PRODUCTION.** `backend/Dockerfile`
  COPY lines are `21`, `25`, `26` (`icons`), `27` (`lib`), `30` (`backend`) — **`vendor/` is never
  copied into the image.** Morning triage 2026-07-27 simulated the image staging independently: all
  22 current precache URLs present, **`vendor/rxdb.bundle.js` absent**. A precached URL that 404s
  fails the **entire** service-worker install for every returning client — the exact bug
  `pwa-cache-and-build-hygiene` just fixed. **This card must add `vendor/` to the Dockerfile in the
  same change set, or not re-add the glob.** *This also means decision 59 was under-argued:* it was
  justified on bandwidth (−495 KiB) when the real justification was that the base tree was shipping
  a broken SW install.
  **Added at morning triage 2026-07-27 (ledger T-25):**
  (6) **Origin shape is DECIDED — decision 69: SAME-ORIGIN, proxied by the Go backend.** Obligation
  2 is answered; it is a spec line now, not a park trigger. A `/sync/*` `httputil.ReverseProxy`
  handler in the existing backend fronts `rest:3000` and `realtime:4000`. Cloudflare Tunnel config
  unchanged, no second hostname, no CORS, no second origin for the SW to reason about. **Costed:
  one handler plus its tests.** Chosen partly because obligation 1 is a row-visibility predicate the
  backend must be positioned to enforce.
  **🛑 FANNED OUT 2026-07-28 → `sync-proxy-endpoint`** (slate-20260729). This obligation is
  backend-only, fork-free since decision 69, and the roadmap had already costed it as one handler
  plus its tests — it does not need the rest of this card to exist, and this card does not need it
  to land first. It builds the door the client will later knock on. Removed from this card's scope;
  this card now *depends* on it rather than containing it.
  **✅ THE DOOR LANDED 2026-07-28** (`sync-proxy-endpoint`, DONE below). This card's client work
  targets `/sync/rest/*` and `/sync/realtime/*` on HQ's own origin; it does NOT need to fetch
  `/api/v1/sync/token` and attach a bearer itself — the proxy mints per request and injects it,
  and a client-supplied `Authorization`/`apikey` is deliberately discarded. Realtime is reached at
  `/sync/realtime/socket/websocket?vsn=1.0.0`; do not add an `apikey` parameter, the door sets it.
  **🛑 BUT THE DOOR IS DELIBERATELY UNLOCKED, AND THIS CARD HOLDS THE KEY — ACTIVATION ORDER.**
  The proxy forwards **every method** to PostgREST with a `role: authenticated` token and no row
  filtering of its own, because filtering is obligation 1 — *this* card's. So: **do not set
  `HQ_SYNC_REST_URL` in any deploy until row-visibility RLS lands.** Doing so before then gives
  every logged-in crew member full read AND write on the whole exposed schema — a dishwasher can
  `PATCH` a template — with nothing in between, because the thing in between was always meant to
  be RLS. `HQ_SYNC_REALTIME_URL` is the safe half to adopt first: Realtime is read-only, so a
  subscription without RLS leaks reads but authors nothing. Recorded identically in
  `backend/internal/sync/proxy.go`'s env-var comment and in the DONE card below; G6 flagged its
  absence (R2) because the landing note read as an all-clear.
  (7) **Two more `api-cache`-shaped disclosures are OWNED HERE — decision 70.**
  (a) `localStorage['hq_apps']` is never cleared on logout, and `index.html:224` still parses the
  previous user's cached slug list in the fail-closed branch — offline on a shared truck phone,
  user B sees user A's tiles. (b) An identity change *without* a logout (B logs in while A's session
  is live) never runs `logout()`, and `login.html` does no cache hygiene of its own. Both UI-only;
  server-side grants remain the real gate. They ride here because this card is expected to retire
  `api-cache` entirely. **Accepted cost: if this card slips, they slip with it.**
  (8) **Fold in the stale comment at `tests/sync.spec.js:1584`** (decision 66) — it has been waiting
  for "the next card touching that file" since 2026-07-26 and Card B did not touch it.
  Footprint: `workflows.html`, new RxDB client layer, `backend/` (the `/sync/*` proxy handler),
  `backend/Dockerfile` (if obligation 5 is taken).

- **`sync-proxy-endpoint`** · **DONE** (2026-07-28, run `overnight-20260729`, **Track B** — corrected
  at morning triage 2026-07-28; the DONE line originally read "Wave 0", which was Card A's wave; card fanned
  out of `sync-rxdb-schema-and-replication` obligation 6 at slate-20260729 planning) · The
  same-origin door decision 69 chose now exists: `backend/internal/sync/proxy.go`, mounted at
  **root-level `/sync/*`** in `main.go` behind `auth.Middleware` and nothing else. `/sync/rest/*`
  fronts PostgREST, `/sync/realtime/*` fronts Realtime, **WebSocket upgrade included**. No CORS,
  no second hostname, no Cloudflare Tunnel change. Registered at the ROOT, deliberately, because
  `/api/v1/sync/token` already occupies that prefix and a wildcard would shadow it.
  **The upgrade path is PROVEN TWICE, hermetically and live** — the slate's park trigger did not
  fire. (a) A hermetic test dials through the proxy into a local `coder/websocket` echo server,
  asserts `101` **and bytes in both directions afterwards** (a proxy that hands back 101 and then
  fails to pump the hijacked connection passes a status-only assertion). (b) A live test drives a
  real upgrade into the running `spike-supabase-realtime-1` container and gets
  `phx_reply {"status":"ok"}` back from a real Phoenix join — which additionally proves Realtime's
  tenant lookup resolved and Realtime VERIFIED the HS256 token the proxy injected, since both
  failures are a bare 403 *before* any 101. The live test `t.Skip`s loudly when the container is
  down; **a skip is not a pass** and the file says so.
  **The red was taken twice, and the second correction is the interesting one.** The compile-level
  red (`undefined: ProxyHandler`) was re-taken against a deliberately naive
  `httputil.NewSingleHostReverseProxy` baseline to prove the assertions bite behaviourally. That
  baseline **passed the 101 and the echo**: Go's stdlib `ReverseProxy` handles the protocol switch
  and the post-upgrade byte pump correctly on its own. So the card's premise —
  *"the part a naive `ReverseProxy` gets wrong"* — was right that a naive proxy fails, but wrong
  about **where**. The three real traps are (1) Realtime routes tenants by the FIRST dot-label of
  the **Host header**, and a stock `Director` forwards the browser's Host → bare 403 with nothing
  in the logs; (2) the `/sync/rest` prefix must be stripped or PostgREST looks up a table called
  `sync`; (3) **Realtime's socket connect reads `apikey` and IGNORES the `Authorization` header**,
  so the token has to go in `?apikey=` on the Realtime path — and must NOT on the REST path,
  because PostgREST reads unknown query params as column filters. *Trap 3 was mis-stated on first
  write-up as "a browser cannot set a header on a WebSocket handshake" (G6 R5a): true, but a
  client-side fact about a DIRECT browser→Realtime connection, and no constraint at all on a
  server-side proxy that builds the outbound handshake itself — this one does, and sets
  `Authorization: Bearer` as well, which Realtime disregards. G6's mutation established the real
  reason: delete the `apikey` injection and the live upgrade 403s with the header still present.*
  **Credential handling is the door's other half.** A caller's own `Authorization`, `apikey` and
  `Cookie` are DISCARDED and a token minted for the *context* user is substituted — `TokenHandler`'s
  impersonation invariant applied at the door, so the proxy can never become a
  bring-your-own-token relay into the substrate. Fails closed on every axis: unset upstream → 503
  `sync_proxy_not_configured`, unset `HQ_SYNC_JWT_SECRET` → 503 `sync_bridge_not_configured`
  (checked *before* any hop), anonymous → 401, unknown room → 404, upstream down → 502
  `sync_upstream_unavailable` with the internal host logged and never echoed.
  **G6 adversarial review returned APPROVE-WITH-NITS**, having planted 6 mutants (Host override,
  `apikey` injection, `Authorization` header, prefix strip, `Cookie` Del, REST `apikey` Del) — all
  6 went red — re-signed the live token with a wrong secret and confirmed the 403, and
  independently rebuilt the naive baseline to confirm the premise correction above. It found **one
  thing with teeth**, repaired in the same change set:
  **🛑 PATH TRAVERSAL (G6 R1), fixed red-first.** The remainder after the room prefix was forwarded
  un-normalised: `out.URL.RawPath = ""` re-derived the wire path from the DECODED `Path`, and Go's
  path escaper does not re-escape `/`, so `%2f` became a real separator. Four vectors reproduced
  against the booted binary — `/sync/rest/../../admin`, `/sync/rest/..%2f..%2fadmin`,
  `/sync/rest%2f..%2f..%2fadmin`, and `/sync/realtime/../rest/spike_notes` (which reached the
  *other* upstream carrying the minted JWT in the query string). The third is the sharp one: the
  **room selection** was made on a decoded path whose separators the caller forged. No live impact
  while both upstreams are path-less, but it becomes real the instant `HQ_SYNC_REST_URL` points at
  a gateway **with a path prefix** — the standard self-hosted Supabase shape,
  `http://kong:8000/rest/v1` — at which point `/sync/rest/../auth/v1/admin/users` escapes into a
  **sibling service** carrying HQ's bearer. Now `400 sync_path_rejected`, checked before the room
  is chosen. It **rejects rather than `path.Clean`s**: cleaning silently proxies a different
  request than the one that was made.
  **A SECOND G6 round (also APPROVE-WITH-NITS) ran a 38-vector attack matrix against the fix** —
  the traversal invariant survived all of it, including WebSocket-upgrade traversal (rejected
  before the hijack, upstream saw nothing) — and verified the "not over-broad" claim against the
  LIVE containers: 24 PostgREST + 4 Realtime calls through the door, **zero false rejections**,
  including `..` and `%2F` inside filter values, RxDB-shaped checkpoint pulls, embedded resources
  and dotted table names; every non-200 came from PostgREST itself. It found one more thing:
  **the `%2f` check was bypassable and two places said it wasn't.** `EscapedPath()` discards
  `RawPath` and re-escapes the decoded `Path` whenever RawPath holds a byte Go's `encodePath`
  validator rejects (`{ } | ^ \ " < >`), and Go's escaper does not escape `/` — so
  `GET /sync/rest%2fadmin{` returned 200 with the upstream seeing `/admin%7B`. **Not exploitable**
  (the dot-segment loop reads the decoded `Path`, so `..` was still caught, and no `..` or `%2f`
  reached the wire by any route) — fixed because the code comment claimed it rejected *every*
  encoded separator and the merge-intent note claimed *anywhere in the request path*. **The right
  move on a false durable claim is to make it true, not to soften it:** the check now reads
  `u.RawPath`, the untouched request target. The dot-segment rule's scope is now stated exactly
  too — an exact `.`/`..` match on **Go's decoded segmentation**, not a universal rule: `..;/`,
  `....//`, `..%00/`, `..%c0%af..` and `%252e%252e` pass deliberately (none traverses nginx or
  Kong) but would matter behind a Tomcat/Jetty-class parser or a double-decoding gateway.
  17 Go tests (15 hermetic + 2 live), including one that rebuilds `main.go`'s actual chi +
  `middleware.Logger` + `Group` + `/sync/*` stack and drives an upgrade through it — pinning the
  dependency property that the logger's `ResponseWriter` wrapper implements `http.Hijacker`, which
  every WebSocket on this router silently depends on. The live pair is gated on
  `HQ_SYNC_SPIKE_LIVE=1` and **fails rather than skips when the flag is set but the port is dead**
  (G6 R4): a skip prints nothing without `-v`, so an intended live run could otherwise degrade to
  hermetic-only coverage and still report `ok` — the B-09 suite-honesty rule applied to a test
  file. The flag's falsy spellings (`0`/`false`/`no`/`off`) are honoured, because the first version
  tested `!= ""` and made `HQ_SYNC_SPIKE_LIVE=0` opt **in** (G6 F-4).
  Backend `0.2.2` → `0.3.0`.
  **Inert until configured:** `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` are unset in every
  current deploy, so every `/sync/*` request answers 503 today. **🛑 AND THEY MUST STAY THAT WAY
  UNTIL RLS LANDS** — see the activation-order block on obligation 6 above: the door forwards every
  method with a `role: authenticated` token and no row filtering of its own, so setting
  `HQ_SYNC_REST_URL` before obligation 1 gives every logged-in crew member read AND write on the
  whole exposed schema. `HQ_SYNC_REALTIME_URL` is the safe half to adopt first (read-only).
  **Residual once activated (G6 R3):** the Realtime credential rides in the query string because
  Realtime ignores the `Authorization` header. It is injected server-side, so it never enters
  browser history, a `Referer`, or a client log — G6 audited HQ's logs, Realtime's logs and the 502
  path and found 0 JWT occurrences — but the HQ→Realtime hop is plaintext inside the compose
  network, so **do not place an access-logging L7 proxy between HQ and Realtime, and do not enable
  Realtime request logging.** Either writes a live bearer to disk in cleartext.
  **Not in scope and NOT done:** RLS predicates (obligation 1), any RxDB client code, any
  `workflows.html` change — all still the parent's. Footprint held:
  `backend/internal/sync/proxy*.go` (new), `backend/cmd/server/main.go` (route registration),
  `backend/internal/version/version.go`.
  Original card text:
  Build the same-origin door decision 69 chose, ahead of the client that will use it. A `/sync/*`
  `httputil.ReverseProxy` handler in the existing Go backend fronts `rest:3000` and
  `realtime:4000` — including the **WebSocket upgrade** path Realtime needs, which is the part a
  naive `ReverseProxy` gets wrong and the part worth a test. Auth: reuse the existing bearer/session
  middleware; the JWT the backend already mints (`sync-jwt-bridge-endpoint`, DONE — backend half)
  is what the proxied services accept. **Not in scope:** RLS predicates (obligation 1, still the
  parent's), any RxDB client code, any `workflows.html` change. **Fork-free** — decision 69 settled
  origin shape; nothing here waits on an operator. Red-first: a test that proves a plain HTTP request
  proxies AND that an upgrade request survives, both failing before the handler exists. Footprint:
  `backend/` (new proxy handler + its route registration + tests). Depended on by
  `sync-rxdb-schema-and-replication`.

- **`sync-rxdb-conflict-notice-mockup`** · **DONE** (2026-07-28 — corrected at morning triage from
  "2026-07-29"; the run merged every card on 07-28, run `overnight-20260729`, card
  branch `card/d-sync-rxdb-conflict-notice-mockup`; card authored at slate-20260729 planning,
  fanned out of `sync-rxdb-conflict-notice-ui`) · **The sign-off artifact exists.**
  `.planning/phases/sync-rxdb-conflict-notice/mockup.html` — **eleven plates** (nine as first
  drafted, two added by the repair round), mobile-first 480px on HQ's shared variable block with
  dark mode, modelled on `.planning/phases/f3-trends-tab/mockup.html` — plus `UI-SPEC.md` beside it
  carrying the State Enumeration Table (**four base rows and six edges**, including all three the
  slate named: **no discarded value available**, **several conflicts at once**, **conflict on a
  field since removed from the template**), the `done_when:` block, and the `conflict$` evidence the
  design rests on. **Zero production code**, as the card required.
  **THE OPERATOR NOW HAS SOMETHING TO SAY YES OR NO TO — that answer is what unblocks
  `sync-rxdb-conflict-notice-ui`, and it has NOT been given.** A *no* is a successful outcome for
  this card; redrawing a mockup is cheaper than redrawing `workflows.html`.
  **The recovery path — the point of the card — is `Restore mine`, and it is deliberately boring.**
  Master-wins discarded the fork because `replicateSupabase`'s push found the server value had moved
  under its compare-and-swap; nothing about that state is unrecoverable, only *stale*. Writing the
  crew member's value again **now, from the current master state**, is an ordinary local edit that
  pushes cleanly — no new sync plumbing, no `conflictHandler` special case, no server endpoint. If
  the server moves again in between, that write conflicts in turn and lands back in the same sheet,
  so the loop is closed. Two degradations behind it: **Copy value** when the field is gone from the
  template (nowhere to write it back, and a clipboard copy is a real recovery on a phone), and
  "open the checklist" when there is nothing showable at all.
  **🛑 THREE VERIFIED LIMITS OF `conflict$` DROVE THE DESIGN — read these before building the UI.**
  Re-derived from W2 (`proof-lww.js`, README half 2 step 5, FORK 3), not assumed:
  (a) **It carries whole documents, never a field name.** The app must diff `input.newDocumentState`
  against `output` itself, and that diff can come back holding nothing a crew member would
  recognise — which is exactly why the *no discarded value available* row exists. Declaring
  `_modified` in the schema makes this common rather than rare (W2 sharp edge 11: any server-side
  touch becomes a conflict), so that schema choice is now a UI-visible one.
  (b) **It is a plain RxJS `Subject` — no replay — and RxDB persists nothing about a resolved
  conflict.** So the design has one hard precondition: **the app must write the discarded value to
  durable local storage the instant the event arrives**, or a reload destroys the value the whole
  screen exists to recover. Where that record lives is the UI card's call; that it must exist is a
  contract. This is also why the banner must read the stored record and not a live subscription —
  `waitForLeadership` defaults to `true`, so only the leader tab's `conflict$` fires at all.
  (c) **It carries no author and no timestamp of its own.** The mockup's "Dana M., 6:12 PM"
  attribution is only as real as the replicated row; if the schema does not carry who-and-when it
  must degrade to "someone else" rather than be invented. **That is a requirement this spec places
  on `sync-rxdb-schema-and-replication`, not a decision this card made.**
  **Open question left for the operator, deliberately:** beyond ~10 conflict groups the sheet needs
  a cap or a date filter; not designed. Judge it against one long dead-zone shift with an active
  manager.
  **Self-verified per CLAUDE.md's headless ritual** — **22 PNGs (11 plates × light/dark at 480px)**
  under `screenshots/`, produced by the committed `shoot.mjs`, read back multimodally and compared
  row by row against each row's visual contract. **Two findings were fixed rather than reported
  around:** nested `<span>`s rendered the banner headline and every checklist name run together with
  their subtitles, and the several-at-once caption promised a group action "above the individual
  buttons" when the render collapses them and puts it at the foot — the caption was corrected to the
  render, not the reverse. **Red-first DOES NOT APPLY and is not silently omitted:** there is no code
  and no test in this card, and the screenshot ritual is the substitute discipline.
  **⚠️ VERIFIER GATE: PASS-WITH-ISSUES → REPAIRED (repair round, same branch).** A verifier whose
  inputs were restricted to the UI-SPEC, the `done_when:` block, the diff and the screenshots — not
  the author's reasoning — confirmed the recovery path is concrete and usable, and returned nine
  defects. All nine are fixed on this branch; the sign-off the operator gives is against the
  **repaired** artifact. The four that change what the operator is being asked to approve:
  (a) **The counting rule was never defined and two plates disagreed about it in opposite
  directions.** It is now stated once in `UI-SPEC.md` §"The counting rule" — banner = Σ recoverable
  rows, chip = that group's rows, **handling a row never changes a count** (only Dismiss or expiry
  does), unidentifiable changes counted separately as `+N`. Every plate obeys it. The operator can
  reject this: it means **the banner is not a to-do list**.
  (b) **Long content overflowed.** A 90-char unbroken token pushed the page to a **951 px
  `scrollWidth`** in a 480 px viewport, with no wrap and no ellipsis — and there was no long-content
  edge row, though CLAUDE.md names it as canonical and a free-text note is the field most likely to
  survive a conflict. New edge row + plate, CSS fixed (`min-width:0` on row and value,
  `overflow-wrap:anywhere`), re-measured at **480 = 480**.
  (c) **Three sub-44px controls** — Undo (35×16), Done (37×16), Review (53×15) — passed a
  `done_when:` row that only grepped the two classes already known to declare `min-height:44px`.
  Undo is the only escape from a mis-tapped Restore. All fixed; `shoot.mjs` now **measures** every
  interactive element's box in both schemes and exits non-zero.
  (d) **Two states claimed more than the mechanism supports.** The empty state said "Nothing was
  overwritten" — a flat guarantee, on the screen shown most often, that a non-leader tab or an
  evicted store makes false; it now reads "Nothing recorded in the last 30 days". The storage-error
  state reassured that the checklists are fine without saying the **record is permanently gone** —
  the one screen in the set where something really is unrecoverable. Both rewritten.
  Also: the two previously-undrawn outcomes (**Keep theirs** and **Undo**) now have a plate, three
  captions that disagreed with their own render were corrected, "Restore all N of mine" was
  restyled primary, and two `done_when:` criteria that could not fail were rewritten (criterion 6
  asked a static plate to show a transition; criterion 14 named only the classes it knew passed).
  **G3 (openspec validate) DOES NOT APPLY** — hq has no `openspec/` tree and
  `night-crew workflow preflight` reports ABSENT.
  **No version constant moved** (`Backend 0.3.0`, `Frontend 1.2.1` untouched) and **`sw.js` was not
  regenerated** — no shipped file changed, and `.planning/**` is in `build-sw.js`'s `globIgnores`
  so the precache manifest is unaffected. Footprint held exactly:
  `.planning/phases/sync-rxdb-conflict-notice/` (new), the roadmap flip, and the merge-intent note.
  Original card text:
  Draft the committed
  mockup that the attended sign-off consumes, so the blocked UI card becomes unblockable. The
  roadmap has now recorded twice that **drafting the mockup — not chasing the sign-off — is the next
  action**, and that drafting is unattended-safe by construction: CLAUDE.md gates *production code*
  behind the sign-off, and the mockup is the artifact that gate reads. Deliver
  `.planning/phases/sync-rxdb-conflict-notice/mockup.html` (model:
  `.planning/phases/f3-trends-tab/mockup.html`, the only existing one) showing what the crew member
  sees when a same-field clash falls back to master-wins **and how the discarded value is
  recovered** — plus the State Enumeration Table CLAUDE.md requires, including the edge rows (no
  discarded value available, several conflicts at once, conflict on a field since removed from the
  template). Input is what `conflict$` actually emits — per document, carrying the document id and
  the discarded value (verified at W2). **Independent of `sync-rxdb-schema-and-replication`** — the
  mockup needs the event's shape, not a built `conflictHandler`. **Zero production code**; the card
  is done when the mockup and its table are committed and the operator has something to say yes or
  no to. Footprint: `.planning/phases/sync-rxdb-conflict-notice/` only.

- **`sync-rxdb-conflict-notice-mockup-amendments`** · **DONE** (2026-07-29, run
  `overnight-20260729-2`, Track C card C1, branch
  `card/c1-sync-rxdb-conflict-notice-mockup-amendments`) · **The revised plates exist.**
  `.planning/phases/sync-rxdb-conflict-notice/mockup.html` is now **revision 2** — **16 plates,
  32 renders** (was 11 / 22), and `UI-SPEC.md` carries **35 `done_when:` rows** (was 20) with the
  State Enumeration Table extended to 12 state rows plus 4 non-state plates.
  **A-1 is drawn:** every one of the 8 banners carries both figures — what happened, and how many
  rows are still to review — and the new `a1-banner` plate proves the worst case, **four banner
  lines coexisting at 480 px in light and dark** (headline + `2 still to review · 2 handled` +
  `+ 2 changes we couldn't identify` + the cause line) with **no truncation**, measured per line by
  `shoot.mjs` rather than judged by eye. A **failed restore counts as still-to-review**, proved on
  the `error` plate.
  **A-2 is drawn:** every Restore control names what it replaces; the new `a2-confirm` plate shows
  the batch override **confirming before writing** and **listing the 3 server values about to be
  overwritten**, each struck through with its author and timestamp; the collapsed `edge-many` view
  now carries **name AND time** on all five rows (it carried a bare "Dana M.").
  **Both open decisions are left open and drawn as decidable:** `openq-count-a` / `openq-count-b`
  render **both readings over identical data** with neither recommended, and the retention window
  renders as the placeholder token `⟨30⟩` — never as prose — with `openq-retention` showing the one
  screen that prints it at two candidate values.
  **Red-first applied and mutation-tested:** three new machine checks in `shoot.mjs` were run
  against the un-amended r1 mockup first (**5 banners carrying one figure, 7 Restore controls
  silent about the loss → exit 1**) and each was then mutation-tested to prove it can fail rather
  than pass vacuously. Two defects were found by **reading the renders back**, not by intention:
  the open-decision captions did not say "NOT SETTLED" inside the plate itself, and the U+1F6D1
  marker rendered as a tofu box in the headless font stack. Both repaired and re-shot.
  **🛑 THE SIGN-OFF IS NOT DISCHARGED.** `sync-rxdb-conflict-notice-ui` stays **ATTENDED-BLOCKED**
  — that is the correct outcome of this card, not a failure. Zero production code; footprint was
  `.planning/phases/sync-rxdb-conflict-notice/` plus the merge-intent note and this flip.
  **Original card text (preserved):** **FANNED OUT of `sync-rxdb-conflict-notice-ui` at the
  2026-07-28 slate-planning session (ledger T-27 decision 91).** Produce **revised plates** implementing
  amendments **A-1** and **A-2** that ledger T-26 decision 82 requires, both already written into
  `.planning/phases/sync-rxdb-conflict-notice/UI-SPEC.md` (see the parent card below for both in
  full). **This card does NOT discharge the sign-off** — it produces the artifact the operator signs
  at morning triage, and the parent stays ATTENDED-BLOCKED when it is done. That is the correct
  outcome, not a failure.
  **Why it separates cleanly, and why it is unattended-safe:** what blocks the parent is not code —
  it is that the committed plates do not yet show A-1 and A-2, so there is nothing to sign. Drafting
  revised plates is safe by exactly the argument that produced the original mockup card: CLAUDE.md
  gates *production code* behind the sign-off, and the mockup is the artifact that gate consumes.
  **Two decisions are deliberately LEFT OPEN and the plates must make each visibly decidable rather
  than quietly settle it:** (i) whether a removed-field row counts in the chip base or moves to
  `+N` — it has no Restore, its recovery is *Copy value*, yet it is counted as "1 answer"; **draw
  both readings** so the operator picks. (ii) The retention window — 30 days was accepted in
  decision 80 and reopened at triage; draw the number as an **obvious placeholder**, not a settled
  fact.
  **Held to the same discipline the original mockup passed:** State Enumeration Table extended for
  the new plates, `done_when:` rows for each, self-verification renders at 480px in **light and
  dark** read back with the Read tool and compared row-by-row, and the **restricted-input verifier
  gate** before any SUMMARY. 🛑 The verifier caught two `done_when:` criteria last time that were
  written so they **could not fail** (the 44px touch-target row enumerated only classes already
  known to pass, excluding `Undo` at 35×16 — the sole escape from a mis-tapped Restore). Write
  criteria that can fail. **Do NOT touch production code.**
  Footprint: `.planning/phases/sync-rxdb-conflict-notice/` only.

- **`sync-rxdb-conflict-notice-ui`** · **🛑 PLANNED — ATTENDED-BLOCKED AGAIN (reverted from
  SIGNED OFF, SLATE-READY at morning triage 2026-07-28; ledger T-26 decision 82)** · The sign-off
  recorded as decision 80 **stands as the record of what was decided at 18:12** and is **superseded,
  not erased**. Walking the committed plates at triage surfaced a defect the sign-off could not have
  accounted for, and the operator directed two amendments, both written into
  `.planning/phases/sync-rxdb-conflict-notice/UI-SPEC.md`:
  **A-1 — the banner must show what happened AND how many rows are still unhandled.** The operator's
  question was *"when she finishes the second, why does it still say three?"* Rule 3's answer —
  "because three answers were overwritten and that stays true" — is literally correct and wrong on a
  phone: a number in a coloured banner reads as a badge, and badges count outstanding work. Rows
  still never leave except on Dismiss or expiry, so **Undo survives**; only the banner changes.
  A failed restore counts as still-to-review, and a plate must prove the two banner lines coexist
  with `+ N change(s) we couldn't identify` at 480px.
  **A-2 — the override must state what it destroys.** "Restore all 3 of mine" sits under three
  `YOURS`/`NOW SHOWS` pairs and never says it replaces them; the batch action must name what it
  replaces, **confirm before writing while showing the N server values about to be overwritten**,
  and carry the same attribution + timestamp the expanded view does (today the collapsed batch view
  shows "Dana M." with no time — the riskiest action carries the least information).
  **Two decisions deferred until the revised plates exist:** whether a removed-field row counts in
  the chip base or moves to `+N`, and the retention window (decision 80 accepted 30 days as drawn;
  triage reopened it). **Unblocks when the operator signs the revised plates** — not before.
  (new card, fanned out of
  `sync-rxdb-schema-and-replication` 2026-07-26 at slating; depends on
  `sync-rxdb-replication-and-conflict-handler`'s `conflictHandler`) · The **user-visible half of
  decision 50**: when a same-field clash falls back
  to master-wins, `conflict$` must surface it to the crew member **with the discarded value
  recoverable** — not silently dropped. Cheap to *get hold of* (`conflict$` already emits the
  discarded document per-document with its id, verified); the expensive half is deciding what the
  crew member should see and how they recover the value. **Why this cannot run unattended:**
  CLAUDE.md requires a committed mockup at `.planning/.../<phase>/mockup.html` and an explicit human
  *"ok, build this"* before UI code on phases introducing new components, plus a State Enumeration
  Table and the verifier-subagent gate. **The operator owes a mockup sign-off before this card can
  ever be slated.** Footprint: `workflows.html`, the RxDB client layer.

  **✅ THE MOCKUP LANDED 2026-07-29** (`sync-rxdb-conflict-notice-mockup`, DONE above).
  `.planning/phases/sync-rxdb-conflict-notice/mockup.html` + `UI-SPEC.md` (State Enumeration Table,
  `done_when:`, the `conflict$` evidence) + **22 self-verification renders** are committed, and the
  artifact has been through a **verifier gate (PASS-WITH-ISSUES → all nine defects repaired)**;
  read the gate summary on the mockup card above before slating, because two of the repairs are
  design decisions the operator can reject outright (the counting rule, and handled rows staying on
  the sheet until Dismiss). **The
  scheduling decision below is DISCHARGED — drafting the mockup was the next action, and it is
  done.**

  **✅ THE REVISED PLATES LANDED 2026-07-29** (`sync-rxdb-conflict-notice-mockup-amendments`, DONE
  above, run `overnight-20260729-2`). `mockup.html` is now **revision 2 — 16 plates, 32 renders**,
  and **A-1 and A-2 are both drawn**: every banner carries two figures, the worst-case four-line
  banner is proved at 480 px in both schemes without truncation, and the batch override confirms
  before writing while listing the server values it will overwrite with author and timestamp. **The
  two deferred decisions are still open and are now drawn as decidable** — both readings of the
  removed-field counting question, and the retention window as a visible placeholder.
  **🛑 THIS CARD IS STILL ATTENDED-BLOCKED.** A revised mockup existing is not a sign-off, and the
  fan-out card did not and could not discharge it. **What is owed: the operator walks the 16 plates,
  answers open decisions (i) and (ii), and gives (or refuses) an explicit *"ok, build this"* on
  revision 2.** A *no* remains a successful outcome.

  **🖊️ THE r1 SIGN-OFF — 2026-07-28, operator, verbatim *"Ok, build this."* — SUPERSEDED IN PART
  at morning triage the same day (ledger T-26 decision 82); it is the record of what was decided at
  18:12 and does not cover revision 2.**
  **🛑 THE FOUR CLAIMS STRUCK BELOW ARE NO LONGER TRUE. They are kept, struck, as the record of what
  was believed at 18:12 — read the block above, not this one, for the card's live status.** An
  unattended merger reading top-down was previously able to take them as binding, because the
  supersession preamble framed them as history while the sentences themselves stayed in the
  imperative. This completes that supersession.
  ~~The gate
  CLAUDE.md sets before UI code on a phase introducing new components is **satisfied**; this card
  is no longer ATTENDED-BLOCKED and may enter a slate.~~ **STRUCK — the card IS still
  ATTENDED-BLOCKED (see the 🛑 block above); decision 82 reopened it and revision 2 does not
  discharge it.** The sign-off was given **with the two
  rejectable design decisions in view and neither was rejected** — ~~so both are now settled and a
  run must implement them as drawn, not re-open them: **(a) the counting rule** as stated in
  UI-SPEC §"The counting rule" — the banner reports how many answers were overwritten in the
  retention window, **not** how many are still unhandled, so nothing a crew member does to a row
  changes any count and a count drops only when a record *leaves* the sheet (Dismiss, or ageing
  out)~~ **— (a) IS STRUCK. Amendment A-1 (decision 82) OVERTURNED exactly this: the banner must now
  carry BOTH figures — what was overwritten in the window AND how many rows are still to review. A
  run that implements the struck counting rule "as drawn" reinstates the defect A-1 was filed
  against.** The half of (a) that SURVIVES is that rows never leave the sheet except on Dismiss or
  expiry — A-1 changed what the banner PRINTS, not what the sheet KEEPS; **(b) handled rows stay on
  the sheet** — a restored row and a kept-theirs row both collapse
  to a confirmation and keep an **Undo** rather than disappearing, because a removed row cannot be
  undone — **(b) STANDS, unchanged, and A-1 explicitly depends on it.** The scope of the yes was the
  artifact **as it stood at 18:12**: ~~`mockup.html` + `UI-SPEC.md` at
  `.planning/phases/sync-rxdb-conflict-notice/` as of the repair round, all 11 plates and 22
  renders~~ **— STRUCK as a description of what is committed today: the tree now holds revision 2,
  16 plates and 32 renders, which the 18:12 yes does not cover.** It is **not** blanket authority
  over the items UI-SPEC §"Explicitly NOT decided here" names.
  **🛑 Still open, and three of them change what this UI can truthfully show** — read UI-SPEC
  §"Explicitly NOT decided here" before slating. The `conflictHandler`'s merge rule and whether the
  replicated schema carries who-and-when belong to
  `sync-rxdb-replication-and-conflict-handler` and `sync-rxdb-collections-and-table-contract`;
  **`_modified` is now DECIDED — not declared, decision 78 below**, which is the good outcome for
  this card (it keeps the *"a change we couldn't identify"* row rare rather than routine). The
  durable conflict record's home is settled (ledger T-27 decision 89 — a local-only RxDB collection,
  personal and per-device). ~~the retention window remains this card's own implementation call —
  the mockup's empty state says **30 days**, and the sign-off accepted that number as drawn.~~
  **STRUCK — the retention window was REOPENED at morning triage 2026-07-28 and is open decision
  (ii), the operator's, not the implementer's. Revision 2's `empty` plate renders it as the
  placeholder token `⟨30⟩` in a dashed box, never as prose, and `openq-retention` draws the same
  screen at two candidate values so the choice is decidable. A run that reads "30 days" out of this
  line and ships it settles an operator decision by omission.**
  **One operator question is still unanswered and is NOT blocking:** UI-SPEC §"Open question for
  the operator" — beyond roughly ten conflict groups the sheet needs a cap or a date filter, and it
  is not designed here. Judge it against one long dead-zone shift with an active manager. A slate
  may proceed without it; if it is still unanswered when this card runs, the run implements no cap
  and says so.

  **🛑 SCHEDULING DECISION 2026-07-26 — DISCHARGED 2026-07-29, kept as the record of why.** Checked at
  slate-20260727 planning: **no mockup exists for this card.** The only `mockup.html` in the repo is
  `.planning/phases/f3-trends-tab/mockup.html`. So the owed sign-off has nothing to review, and
  "get the operator to sign off" is NOT the next action — **drafting the mockup is.**
  Drafting is **unattended-safe by construction**: CLAUDE.md gates *production code* behind the
  sign-off, and the mockup is the artifact that gate consumes, so a run may produce it. It is also
  **independent of `sync-rxdb-schema-and-replication`** — the mockup needs to know what `conflict$`
  emits (already verified: per-document, carrying the document id and the discarded value), not
  that the `conflictHandler` is built. **A draft-the-mockup card was considered for
  slate-20260727 and deliberately NOT added**, because adding a fourth card would have reopened a
  signed slate on the night its largest card (Card C) runs. **Put it at the top of the next slate:**
  it is cheap, off every critical path, and it converts a permanently-blocked card into one the
  operator can unblock over morning coffee. Do not rediscover this — it has now been checked twice.

- **`sync-jwt-bridge-endpoint`** · **DONE — BACKEND HALF ONLY** (2026-07-26, run
  `overnight-20260726`, card branch `card/b-sync-jwt-bridge-endpoint`). 🛑 **This card is the
  SERVER side only. The card is NOT finished as originally roadmapped** — the frontend
  client-construction helper and the `@supabase/supabase-js` pin + upgrade smoke test moved to
  the client layer (`sync-rxdb-schema-and-replication`) at slating, and **remain outstanding
  there**. Do not read this DONE as "the bridge is complete end to end."
  **Delivered:** `POST /api/v1/sync/token` (`backend/internal/sync/jwtbridge.go`,
  `jwtbridge_handler.go`), mounted inside the cookie group and deliberately outside every
  `RequirePermission` gate — access-resolution plumbing in the same category as `/me`, recorded
  as an exception in `tests/grant-enforcement-parity.spec.js`. Stdlib-only HS256 mint
  (`crypto/hmac`), **`backend/go.mod` byte-untouched**. Claims map EXISTING HQ data only —
  `sub`←`users.id`, `role`≡`authenticated` (constant), `hq_roles`←`users.roles`,
  `hq_grants`←`app_permissions ⋈ hq_apps` (the same predicate `RequirePermission` enforces, with
  an anti-drift test asserting per-slug agreement), `hq_sid`←`sessions.token_hash`. **No new
  grant or permission concept was invented** — the park trigger did not fire.
  **The real gate passed:** 16/16 attack variants, red-first. Red captured at `de00401` with 9
  variants failing; green at `09ffa65`. Captures at
  `.night-crew/qa/spike-supabase/captures/{red,green}-20260726-attack-variants.txt`; the red is
  reproducible on the same database via `SPIKE_SKIP_POLICIES=1`.
  **Finding (1) honoured and made self-verifying:** policies read the **plural** GUC
  `current_setting('request.jwt.claims', true)::json ->> 'sub'` via a single accessor
  `public.hq_jwt_claim`, and `public.hq_uid_trap` carries a deliberately WRONG `auth.uid()`
  policy so variant V13 re-proves the finding every run instead of trusting a comment.
  **🛑 The card's own load-bearing design call, for whoever builds the client and the cutover:**
  the token's `hq_grants` claim is **ADVISORY, not the gate.** Claims freeze at mint, so a
  claim-trusting policy would leave a revocation replay window as long as the TTL. RLS instead
  joins `public.hq_grant_projection` — a **live** projection of `app_permissions ⋈ hq_apps` —
  on every row, which is what makes revocation immediate (variants V8/V9/V12 prove it). **Who
  writes that projection table — ~~push-on-grant-change, periodic reconcile, or `postgres_fdw`~~ —
  was inherited by `sync-hard-cutover` as an open contract and is now CLOSED: morning triage
  2026-07-26 (ledger T-23 decision 61) chose **push on grant change**, in the same transaction as
  the `app_permissions` mutation. Reconcile reintroduces the exact replay window the projection
  exists to eliminate; `fdw` couples the two databases in a way the cutover has not settled.**
  ~~Also open by design: the `owner_id = sub` predicate cannot express HQ's non-single-owner
  rows.~~ **CLOSED at morning triage 2026-07-26 (T-23 decision 55): this was never an open product
  question — `ResolveEntityAccess` (`backend/internal/sync/ops.go:474`) is HQ's shipped answer and
  the cutover PORTS it rather than inventing one.** See the schema card. Policies at
  `.night-crew/qa/spike-supabase/sql/hq-bridge-{fixture,policies}.sql`.
  *(Original card text follows, for the record.)* · **Narrowed 2026-07-26 at
  slating:** the frontend client-construction helper and the `@supabase/supabase-js` pin + upgrade
  smoke test move to `sync-rxdb-schema-and-replication`, where the client is actually constructed.
  Decision 51's substance is unchanged; only its address moved. This card owns the **server** side.
  **Its real gate is the attack-variant suite (Engineering KR2)** — a new Go test file modelled on
  `tests/grant-enforcement-parity.spec.js`'s 13-variant structure: invalid `role` claim, expired
  token, missing `sub`, wrong signature, **token replay after grant revocation**, forged owner
  write, anon. Prove RLS **discriminating**, not merely responding — a 200 alone proves nothing, and
  a `service_role` BYPASSRLS control is what rules out *"the table was empty."* Red-first REQUIRED:
  every variant captured refusing before the policy that refuses it is written. Reuse W1's ~10-line
  stdlib-only HS256 mint (`crypto/hmac`) — **no new module dependency.** Go backend endpoint that mints the
  Supabase-compatible JWT from the existing session/bearer-token auth and grant data, bridging
  existing permissions into the `role`/`sub` claims Supabase's RLS policies read — no adoption
  of Supabase Auth/GoTrue. **🛑 UNBLOCKED at morning triage 2026-07-25, and it carries two
  findings from W1/W2 it must not rediscover.** (1) **`auth.uid()` is WRONG for HQ and every
  copy-pasted hosted-Supabase policy will fail non-obviously** — without GoTrue's migrations the
  `auth` schema ships only `email`/`role`/`uid`, and `uid` reads the *legacy singular* GUC
  (`current_setting('request.jwt.claim.sub', true)`) with no plural fallback; under the stack's
  `PGRST_DB_USE_LEGACY_GUCS: "false"` it returns NULL, and a non-UUID `sub` raises `invalid input
  syntax for type uuid`. Verified at triage against the live catalog. (2) **This card owns FORK
  4's resolution (T-22 decision 51): stay gateway-less** — no Kong — with a small permanent
  client-construction helper in HQ using `global.fetch` and `realtime.transport`. **Rider: pin
  `@supabase/supabase-js` and add a smoke test that fails loudly on upgrade**, since the coupling
  is to how the library derives `<baseUrl>/rest/v1`, not to the public extension points.
  Footprint: `backend/internal/auth` (or a new package), `backend/internal/sync`.

- **`sync-hard-cutover`** · **PLANNED — LAST** (dependency restated at the 2026-07-28 dissolution:
  depends on **all three** of `sync-rxdb-collections-and-table-contract`,
  `sync-rxdb-row-visibility-rls` and `sync-rxdb-replication-and-conflict-handler`, plus jwt-bridge.
  🛑 **Row-visibility RLS is not optional for this card specifically** — the cutover makes RxDB the
  single write path, which means it is also the moment `HQ_SYNC_REST_URL` gets set in a real
  deploy) · Replace
  BOTH current write paths in `workflows.html` — `autoSaveField`→`POST /saveResponse` and
  `sync.js`'s WebSocket/ops-log broadcast — with the RxDB store as the single write path. Retire
  `sync.js`, `backend/internal/sync/`, and `/saveResponse` entirely. Hard swap, no parallel run
  (per the explore session — no need to keep the old system live during cutover). Reconcile the
  existing Workbox service-worker offline caching against RxDB's own local persistence so there
  is exactly one offline story, not two (Workbox keeps owning static-asset caching; RxDB owns
  data). Footprint: `workflows.html`, `sync.js` (deleted), `backend/internal/sync` (deleted),
  `backend/internal/workflow` (`/saveResponse` removed).

- **`workflow-submission-status-default`** · **DONE — server half** (2026-07-25, run
  `overnight-20260725`, merged `53e921d`). `submitChecklist` writes `status='completed'` for
  `requires_approval:false` (a value the 0011 CHECK already permitted and nothing used — no
  migration), and `pendingApprovals` gates on the submission's own frozen snapshot
  (`(s.template_snapshot->>'requires_approval')::boolean IS NOT FALSE`) rather than the live
  template flag, which closes the actual approvals leak. Go gates green; the two new tests were
  proven to genuinely guard the fix (reverting `repository.go` at triage makes exactly those two
  fail). The `IS NOT FALSE` NULL trap was probed adversarially and **is disarmed** —
  `Template.RequiresApproval` is tagged `json:"requires_approval"` with no `omitempty` and
  `template_snapshot` is `jsonb NOT NULL`, so the key is always written; no backfill needed.
  🛑 **The client half is a separate card — see below.** · (independent footprint, no dependency
  on the sync cards) · Footprint: `backend/internal/workflow`. *(from BACKLOG
  "`checklist_submissions.status` never set for `requires_approval:false` submissions")*

- **`workflow-submission-status-client-half`** · **DONE — client half** (2026-07-26, run
  `overnight-20260726`, Wave 0 alone on a quiet box). **Both named reds are GREEN**
  (`tests/repro-cut-task.spec.js:153`, `tests/sync.spec.js:1581`), captured RED first on the
  unfixed tree at `82d0053` before any production line changed. All seven call-site line numbers
  were confirmed unmoved from the slate's. The seven sites now route through ONE vocabulary block
  (`SUBMITTED_STATUSES` / `PENDING_STATUSES` + `isSubmittedStatus` / `isPendingStatus` /
  `isApprovedStatus` / `isFrozenStatus`) rather than seven inline string comparisons, so the next
  lifecycle value is a one-line change in one place. The optimistic pair now writes what a reload
  would actually fetch (`'completed'` / `'pending'`) instead of `'submitted'` / `'pending_approval'`
  — **neither of which the server has ever persisted**; that divergence is exactly what hid the bug,
  and closing it is why the new test asserts optimistic == server. `0b53d46` was cherry-picked
  verbatim (diff byte-identical; only a `Night-Crew-Card` trailer added). New render test `RUN-09c`
  lands the missing assertion — list badge AND runner, before AND after a reload, plus a
  one-submission-row count. The `night-crew.toml:50-51` seam fix landed on the two keys named
  (`backend/internal/workflow`, `workflows.html`). **`backend/internal/workflow` was NOT
  touched** — the server's vocabulary is correct as F1 landed it, so the card's PARK trigger
  (reopening decision 49) never fired. **`idempotency_key` deliberately left per-call** — see the
  card report; a stable key would break unsubmit→resubmit and rejected→resubmit, and the offline
  replay path already reuses the enqueued key, which is where the guarantee actually lives.
  *(Original diagnosis, kept for the record: dispatched strictly first and alone because (i) while
  `dev` carried these two reds, every other card's "no new reds" baseline was polluted, and (ii) its
  acceptance test IS two specs going green, one of them in the load-sensitive `sync.spec.js` — a
  green sampled under concurrent load would prove nothing (20260722's S1 lesson).)* `workflows.html`
  recognises only `submitted` / `pending_approval` / `pending` / `approved`, so the new
  `'completed'` falls through to the **else** branch at `:2099-2105`: a submitted no-approval
  checklist comes back **fully editable with a live `#submit-btn`, no badge, and
  `fillState.readonly` false** — and `:1656` mints a fresh `idempotency_key` per submit, so a
  second submit writes a **second submission row**. Two E2E specs are red on `dev` until this
  lands: `tests/repro-cut-task.spec.js:153` and `tests/sync.spec.js:1581`.
  **Chosen over reverting or mapping at the API boundary because `sync-hard-cutover` deletes the
  API boundary a translation layer would live in** — the client must learn the DB's vocabulary
  either way. **Surface is at least SEVEN call sites, not the four first named:** `:2065`,
  `:2066`, `:2067` (list-card badges), `:2093-2095` (the render gate), `:2411` (`getProgress`
  snapshot gate), `:2453` (hydration gate), `:2717` + `:2720` (the optimistic pair, which today
  write `'submitted'` — a value the server never persists). `sync.js` has zero submission-status
  comparisons; no other page reads it. **Must also:** keep `0b53d46` (the red-first Playwright
  test already written on `card/f1-workflow-submission-status-default`) as this card's test; add
  a test asserting the no-approval **submitted state renders**, because a triage sweep found only
  two tests in the entire suite exercise it and both are the red ones (`GATE-01/03/06` pass
  vacuously here); and land the `night-crew.toml:50-51` seam fix — `backend/internal/workflow`
  and `workflows.html` → `["workflows", "persistence", "sync", "repro-cut-task"]` — which this
  card is the first to depend on. Footprint: `workflows.html`, `tests/workflows.spec.js`,
  `night-crew.toml`.

---

## Backlog dispositions this round

| Backlog item (`· new`) | Disposition |
|---|---|
| `workflows.html` sync: migrate to RxDB + self-hosted Supabase | promoted → ~~`sync-rxdb-feasibility-spike`~~ → **`sync-spike-stack-and-jwt-bridge` + `sync-spike-rxdb-replication`** (fanned out 2026-07-25 at slating), `sync-rxdb-schema-and-replication`, `sync-jwt-bridge-endpoint`, `sync-hard-cutover` (Activity 1) |
| `checklist_submissions.status` never set for `requires_approval:false` submissions | promoted → `workflow-submission-status-default` (Activity 1) |
| Replay fetch-storm class is NOT fully closed | dropped — superseded by the RxDB/Supabase migration (symptom of the mechanism being replaced) |
| `sync.js` catch-up fetch-storm gate | dropped — superseded by the RxDB/Supabase migration |
| Rejected-field hydrate quirk: new answer visually clears on reload until resubmission | dropped — superseded by the RxDB/Supabase migration |
| Cross-user checklist hydration divergence (approved-vs-rejected ghost state) | left `new` — needs a product ruling on desired cross-user semantics; routed to the next `/nc-pm-session` intake, not resolved this round |

All other `· new` backlog items (security/infra hygiene, grants follow-ups, test/run-mechanics
hygiene, product/display nuance, money-precision) were grouped for this round's walk but not
yet walked — left untouched, still `new`.
