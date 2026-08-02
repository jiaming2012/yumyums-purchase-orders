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
  **Card AUTHORED 2026-07-31 evening (`/nc-slate-plan`, §1 fork gate — ledger T-30 decision 111):**
  **`sync-rxdb-write-policies`**. Not a split — a **missing** card.
  `sync-rxdb-row-visibility-rls` shipped SELECT policies only and deferred writes to "a follow-up
  card"; no such card existed, so `sync-hard-cutover` — which makes RxDB the single write path —
  was blocked on nothing at all. Verified at source before authoring: `0003_rls_policies.sql` has
  zero `WITH CHECK`. Its permission semantic was an open operator decision, resolved inline at slate
  planning rather than discovered at 3am. The denominator moves again.
  **Fan-out 2026-08-02 evening (`/nc-slate-plan`, §1 split rule — slate `20260803`):**
  `sync-hard-cutover` split into **`sync-cutover-list-scope`** + **`sync-hard-cutover`**. The trigger
  was an operator product decision taken inline at planning (B-43): the two list views — **My
  Checklists** and **Approvals** — are lists over MANY submissions and cannot name the single
  `checklistId` that `sync-replication-scope-per-checklist` made mandatory, and the operator chose
  **lists stay live**, i.e. a recorded C-2 widening rather than a fill-view-only cutover. That made
  the card bundle a scope model, a write-path swap and a retirement — three mechanisms each rivalling
  a normal card. The scope half is separable, independently provable against the existing 54-subtest
  RLS suite, and de-risks the swap completely. 🛑 **This is a SPLIT, not a parallel run** —
  `autoSaveField` → `/saveResponse` stays the live path until the cutover swaps it, and the cutover
  swaps and retires in one change set, so P-KR3 is unviolated. The name `sync-hard-cutover` stays
  with the card that does the hard swap so P-KR3 still names the WO it was written about. Activity 1
  holds **25** card bullets; the Delivery per-card denominator moves again.
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

> **Merge status, 2026-07-31 morning triage.** All four `overnight-20260801` cards
> (`app-timezone-unify-new-york`, `sync-rxdb-row-visibility-rls`,
> `sync-rxdb-replication-and-conflict-handler`, `sync-rxdb-conflict-notice-ui`) are now **MERGED
> to `dev`** (`--no-ff`), which upgrades their bullets from the run's build-time DONE convention
> to an actual merge. **Still deployed to nothing** — nothing pushed to prod, nothing tagged,
> `main` untouched, and the two-repo sales-processor agreement is unmet, so no deploy is
> authorized. Reviewed by adversarial re-execution: G1/G2-Go/G4 green, **G2 Playwright exit 1 on
> B-27** (pre-existing cross-spec pollution, no card's doing) — the closeout's "gates green" claim
> was refuted at triage. Ledger **T-29**, decisions 104–110.
>
> 🛑 **Standing rule now binding every remaining sync card (T-29 decision 105):** replication
> scope is **per-open-checklist, never all collections at once.** No card may widen it without a
> recorded decision.

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

- **`app-timezone-unify-new-york`** · **DONE — every site, both contracts, one card** (2026-08-01,
  run `overnight-20260801`, Track A card A1, branch `card/a1-app-timezone-unify-new-york`) ·
  **🛑 BUILT AND HANDED TO THE ORCHESTRATOR FOR MERGE — DEPLOYED TO NOTHING.** Flipping this
  bullet DONE ahead of the merge is run convention; asserting "MERGED" would not be, and this
  card's whole subject is records that read as evidence. As written, the branch is complete and
  gate-clean; the orchestrator merges it. This card does not authorize a deploy. Until
  sales-processor ships its matching change, one repo is wrong — the disagreement is one hour at
  each period edge, and it can only move a `pending_purchases` row with **no extracted
  `event_date`** (an extracted `event_date` wins the `COALESCE`, so it is never exposed to the
  zone). Confirmed pendings and `purchase_events` cannot move at all.
  **🛑 CHANGEOVER DATE: THE FIRST DEPLOY AFTER THIS MERGE — TBD, AND NOT YET SCHEDULED.** An
  earlier draft of this bullet stamped **2026-07-29** for the sites the parked attempt moved and
  **2026-08-01** for `trends.go`'s window and the badge-reset client. Both are wrong and were
  corrected in the code comments and the migration header: `overnight-20260729-2` **PARKED** this
  card and merged nothing, and nothing deploys on 2026-08-01 either. No boundary moved on either
  day. The changeover is the deploy that runs migration `0072`; recover its date afterwards from
  `goose_db_version.tstamp` for version 72. **FIX FORWARD ONLY** — weekly COGS and payroll figures
  produced before that deploy were already acted on and are **NOT** restated.
  **The 07-29 park was correct and was resolved, not overridden.** The parked branch was resumed
  intact at `8da3ded`; nothing was rebuilt. Three additions landed on top of it, exactly as
  decisions 93 + 94 scoped them:
  **(1) Both contract documents and assumption A5.** `21-SALES-PROCESSOR-CONTRACT.md:27`, `:67`,
  `:319` and `999.2-SALES-PROCESSOR-CONTRACT.md:30` + a new A10. A5 is rewritten from "confirm
  this" into the coordinated-release instruction it now is — what is changing, what HQ has BUILT
  (and pointedly has **not** deployed), what sales-processor must do, the sequencing, and the
  bounded blast radius. `:67` also gained a
  correction the zone edit surfaced, and the card's first attempt at it got the provenance exactly
  backwards. The **published** `pending_review_ids` expression had never stated the
  `COALESCE(event_date, …)` — but that is **not** because Phase 21 forgot to write down what it
  shipped. Phase 21 published what it shipped, accurately. **HQ changed what `pending_review_ids`
  returns on 2026-06-06** — commit `cf959bd`, quick task `260606-0gh`, a task separate from Phase 21
  — **and never published it.** Under the published expression a late-discovered receipt did not
  block payroll; under the shipped code it does, so sales-processor may have been receiving an
  undocumented `ready:false` since June 2026. That is a **second** undisclosed change to tell the
  counterparty about, not a footnote to a timezone card, and it is filed as **B-29**.
  **(2) `trends.go`'s `trendsWindow`** — a bare `time.Now()` with no `LoadLocation`, missed by the
  parked attempt, eleven lines above a site it did edit. Red first: `trendsWindow(2026-07-27T01:30Z)`
  answered `{from 2026-05-11, to 2026-07-27}` against `{from 2026-05-04, to 2026-07-26}` — a full
  week off, and a "today" naming tomorrow. A third control subtest ("midday, both zones agree")
  passed before AND after, proving the fix is a zone conversion and not a day shift.
  **(3) Decision 94 — badge reset follows the app zone.** `tests/inventory.spec.js`'s
  *"badge reset saves with browser timezone, not hardcoded value"* was **rewritten, not worked
  around**: it is now parameterised over `Asia/Tokyo` (catches follow-the-device) and
  `America/Chicago` (catches the zone this card removes, which follow-the-device also produces on a
  Central phone — the one wrong answer that could look right), with a fixture-liveness check so it
  cannot pass while testing nothing. Both reds observed at `America/New_York` vs the device zone.
  This one mattered structurally: the config upsert is DELETE+INSERT with an explicit zone, so the
  next badge-reset save would have **overwritten migration `0072`'s UPDATE**. Migrating the row
  without fixing the client would have fixed nothing.
  **Three park-note items absorbed.** The money path finally has a boundary test —
  `pendingPeriodDateExpr`, the card's headline site, was covered by no red at all, and the nearest
  existing case used a timestamp resolving to the same date in both zones. The new test places rows
  in the one hour where the zones disagree at **opposite** period edges so no single-zone answer
  satisfies both; proven red by injecting the old Chicago literal, which produced exactly the
  predicted inversion. Eight stale Chicago comments in that file cleared, with the one
  non-distinguishing case now saying so in its own comment. `APP_TIMEZONE` parity is now
  **mechanical** — `[A1-TZ-PARITY]` reads `users.DefaultTimezone` out of the Go source as the
  authority and asserts all three frontend literals plus the live `window.APP_TIMEZONE` agree; its
  subject set is asserted non-empty and pinned to a count (B-22/23/24), and it was proven to bite by
  injection rather than assumed. `appDateString`'s silent UTC fallback now warns, naming the
  consequence. **One item filed rather than fixed:** `receipt/worker.go`'s `parseEventDate` stamps a
  COGS period from server-local time — `backend/internal/receipt` is outside the footprint, so it is
  **B-28** in BACKLOG.md, deliberately carried out rather than missed.
  **Original card text follows.** (was: **PLANNED — HIGH, medium/large (RESCOPED), RESUMABLE**,
  **🅿️ PARKED on `overnight-20260729-2` and NOT merged** — G6 returned REJECT and the orchestrator
  verified the decisive evidence before parking. Branch
  **`card/a1-app-timezone-unify-new-york` @ `8da3ded` is preserved, unmerged, worktree intact**;
  migration `0072` is unclaimed and most of the work is reusable. **✅ FORK RESOLVED 2026-07-29
  (ledger T-28 decisions 93 + 94).** The park was CORRECT: HQ **publishes** a contract to
  sales-processor pinning `America/Chicago` — `21-SALES-PROCESSOR-CONTRACT.md:27`, `:67` (the exact
  expression A1 replaced), **`:319` assumption A5 "If the food truck moves to a different TZ, both
  repos must update"**, and `999.2-SALES-PROCESSOR-CONTRACT.md:30` — which decision 83 never
  addressed. A1 reported *"Nothing parked — no site turned out to be deliberately Chicago"*, false
  against the repo's own artifacts. **Operator ruling: both repos move to `America/New_York` in a
  coordinated release.** So the card is now **wider than A1 built it** and carries three additions:
  (1) **edit both contract documents and assumption A5** — this is half of a two-repo agreement, not
  a refactor; (2) **`trends.go:89-98`**, which A1 missed and which would otherwise leave two 12-week
  COGS windows on two different zones (the park note lists two further sites to absorb); (3)
  **decision 94 — the Setup-tab Badge Reset (`inventory.html:2713`) must follow the app zone, not the
  browser's**, and `tests/inventory.spec.js:2022` is **asserting the defect** and must be rewritten
  rather than worked around. 🛑 **Nothing ships to prod until sales-processor's matching change is
  ready** — until both land one repo is wrong, and the disagreement is one hour at each period edge on
  rows with no extracted `event_date`. Nothing is broken today: A1 did not merge.) **Original card text
  follows.** (new card, morning triage
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

- **`sync-rxdb-row-visibility-rls`** · **DONE — built on `overnight-20260801` (B2), resumed from the
  2026-07-29 park.** Shipped: migration **`0073_sync_fdw_views.sql`** (HQ-side — three least-privilege
  read-through views + the `hq_sync_fdw` role, created NOLOGIN with no committed password),
  **`sync-schema/sql/0002_hq_fdw.sql`** (substrate-side `postgres_fdw` server, mapping and three
  foreign tables, with the tables revoked from `anon`/`authenticated`/`public` so PostgREST cannot
  serve HQ's role map), **`sync-schema/sql/0003_rls_policies.sql`** (`hq_can_see_template` /
  `hq_can_see_field` and SELECT policies on three of the four replicated tables), and a **27-subtest
  attack suite** (19 numbered variants V1-V19, 4 positives, 2 population floors, 2 `service_role`
  controls) at `backend/internal/sync/rowvisibility_rls_test.go`, **red-first** — captures at
  `.night-crew/qa/spike-supabase/captures/{red,green}-20260801-row-visibility.txt` (red: 16 FAIL /
  11 PASS with policies withheld; green: 27/27). Both inherited properties are preserved **and asserted**
  — a mutation adding `assignment_role = 'assignee'` turns `POSITIVE/alice` red, and deleting the
  admin arm turns exactly `POSITIVE/carol`, `V12`, `V14` red. **Two things triage must rule on:**
  (1) the card ships **SELECT policies only** — INSERT/UPDATE stay policy-less (deny-all) because
  `ResolveEntityAccess` is a fan-out resolver and extending it to writes would invent a permission
  semantic, so **RxDB push replication is refused until a follow-up card writes `WITH CHECK`
  policies**; (2) `submission_rejections` likewise keeps no policy — the resolver has no case for it,
  so a policy there would be an extension, not a port. `HQ_SYNC_REST_URL` is **still not set by this
  branch** and disarms only at triage, on evidence. *Original card text follows.* · **FORK
  RESOLVED 2026-07-29 (ledger T-28 decision 92) — the projection is fed by `postgres_fdw` from the
  substrate to HQ, and decision 61 is REVERSED.** Card B1 settled the topology on
  `overnight-20260729-2` in the direction that makes decision 61's contract impossible: the
  projection and the mutation are in **two different Postgres servers**, `max_prepared_transactions`
  is `0` at both ends, and `Sign()` is an allowlist that can only emit `authenticated`, so no
  transaction can contain both and no restructuring of the mutation changes that. B2 parked on this
  rather than seeding a projection by fixture and producing a green matrix — *a security proof about
  a table nothing in production writes.* **What the resuming card builds instead:** foreign tables
  from the substrate onto HQ's live `template_assignments ⋈ users`, so **there is no projection to
  write** and "same transaction" is vacuous — zero stale-permissive window. The extension is proven
  installable at both ends by executing the C symbol. **Accepted standing cost: HQ's Postgres is on
  the network path of every RLS row check.** The full port of `ResolveEntityAccess` into
  `hq_can_see_template()` is **already written out in
  `runs/2026-07-29-2-autonomous/park-b2-sync-rxdb-row-visibility-rls.md`**, verified by G6 as a
  character-for-character faithful transposition with both inherited properties preserved — do not
  redo it. Decision 99: decision 61 applied by analogy (it named `app_permissions`; this card needs
  `template_assignments`), and decision 92 supersedes the question by removing the projection. If
  `sync-hard-cutover` later co-locates the databases the fdw becomes vestigial and decision 61 comes
  true structurally. **Original card text follows.** (fanned out of
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

- **`sync-rxdb-replication-and-conflict-handler`** · **DONE — landed on `overnight-20260801`
  (Track C, card C1), 2026-07-31.** Four new files under `sync-rxdb/`
  (`conflict-handler.js` — zero imports, the field-level three-way merge;
  `client.js` — the permanent same-origin helper; `bootstrap.js` — the one module a page
  loads; `package.json` — `"type":"module"`, zero dependencies), two new spec files
  (`tests/sync-rxdb-conflict.spec.js` 35 tests at HEAD,
  `tests/sync-rxdb-client.spec.js` 24 tests), and edits to `workflows.html` (ONE
  `<script type="module">` tag), `build-sw.js`, `backend/Dockerfile` and
  `tests/sw-manifest.spec.js`.
  **Red-first figure, corrected at G6:** the conflict spec was **33 tests with 29
  failing** at the red-first commit `ab53478` (RAW EXIT 1), measured by the reviewer.
  An earlier draft of this bullet said "35 tests with 29 failing" — **35 is the count
  at HEAD, not at `ab53478`**; the 29 is exact. Of the 4 passing at `ab53478`, three
  are the master-wins reproduction proper and the fourth (`the case table is non-empty
  and names exactly the decided rules`) is an anti-vacuity check, not part of the
  reproduction.
  **Obligation 5 decision: the glob WAS re-added and `vendor/` WAS added to the
  Dockerfile, in one commit** — precache 22 → 27 files, 1.50 → 1.97 MB — plus a
  mechanical guard that simulates the Dockerfile's staging and asserts every precached
  URL survives it, with a companion test proving the guard can print FAIL.
  **The `assumedMasterState`-absent fallback: master wins on every field and every
  differing field is recorded** — decision 50's own same-field-clash branch applied to
  the whole document, so the winner is byte-identical to RxDB's default while every
  discarded value stays recoverable. Not a product call; no park.
  **Deliberately NOT done here:** no RxDB database is created, no replication is started,
  and no write path is rerouted. Those are `sync-hard-cutover`, and `HQ_SYNC_REST_URL`
  remains unset everywhere. Original card text follows.
  ~~**PLANNED — SLATE-READY**~~ (fanned out of
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

- **`sync-rxdb-write-policies`** · ✅ **DONE — run `20260802`, Track A (A2), branch
  `card/a2-sync-rxdb-write-policies`.** Decision 111's four rows shipped as
  `sync-schema/sql/0004_write_policies.sql`; **the PARK trigger did not fire — no fifth predicate
  was needed.** · **Red-first, and there are TWO reds because one of them is useless here.** The
  familiar `SYNC_RLS_SKIP_POLICIES=1` (RLS torn down) reds **30** subtests — every write REFUSAL —
  but passes every write POSITIVE, and this card's starting state was **deny-all**, which passes
  every refusal in the file. So the card added a second red mode,
  `SYNC_RLS_SKIP_WRITE_POLICIES=1` (0003 kept, 0004 withheld), which reds **12** — all ten write
  positives, the rewritten V18, and the positive halves of W9/W14. Both were captured before the
  policies existed and both reproduce on demand. **GREEN: 52/52 subtests, all executed**, up from
  the shipped 27. · **What landed, per row.** `checklist_templates`: **no write policy, and the
  absence is the answer** — the builder keeps the REST path, no phone writes a template definition,
  and W1/W2/W3 assert it including **as an admin**, so it is a decision with evidence rather than a
  gap that looks like an oversight next to three tables that all got one.
  `checklist_submissions`: `with check (hq_can_see_template(template_id))`, which closes the lie
  `0003_rls_policies.sql:243` names by hand. `submission_responses`:
  `with check (hq_can_see_field(field_id))` — **field-scoped**, and WP3 pushes a row with
  `submission_id: null` because a submission-scoped write predicate would refuse every offline
  draft *while passing every attack variant*. `submission_rejections`: approver-only write **and
  the SELECT policy it never had**. · 🛑 **EVERY UPDATE POLICY CARRIES BOTH `using` AND
  `with check`.** Postgres requires only the first and warns about neither, and `using` alone
  passes every cross-user attack in the suite while still letting an attacker **move a row they
  legitimately own into a scope they cannot see** — W5 re-parents a submission under the orphan
  template, W7 repoints a response at another user's field. Neither is visible in the status code
  (200 `[]` for a `using` exclusion, 403 for a `with check` rejection), so **every write assertion
  verifies through the pool and never the response.** · **The asymmetry is real and is asserted in
  both directions.** Reads keep INHERITED PROPERTY 1 byte-for-byte; only approvals filter on
  `assignment_role`, through a **separate relation** (`hq_template_approvers`, 0002 §3d) so neither
  half can drift into the other. Loosen writes to match reads → every crew member signs off their
  own checklist (**W9** reds). Tighten reads to match writes → every approver goes blind (**WP5**
  and `POSITIVE/alice` red). · 🛑 **ONE DEVIATION FROM THE AUTHORED PARK TRIGGERS, TAKEN
  DELIBERATELY:** the bullet as authored said the card "must not need a `backend/migrations/`
  file". It needed one. `assignment_role` does not cross the FDW — 0002 §3a keeps the column off
  the wire *on purpose* — so decision 111's own `hq_can_approve_template` is not evaluable on the
  substrate without it. **Migration `0074_sync_fdw_approver_view.sql` creates ONE read-only VIEW
  and ONE grant; it alters no table, no column, no constraint**, and the view contains only
  approver rows so it cannot widen a read. The signed slate narrowed this card's park conditions to
  exactly one — a write predicate beyond the four rows — and that one did not fire. Recorded as a
  deviation rather than folded away. **It adds `template_assignments.assignment_role` to 0073's
  "may no longer be retyped" list.** · **B-36 FIXED in the same change set, and both arms proved.**
  `resolveSpikeConfig` turned ANY `docker compose … port` failure into `t.Skip`: measured at
  `ok … 0.014s`, exit 0, with `--- SKIP` on both attack suites — forty-nine subtests and the same
  `ok` line the package prints when they all pass. The opt-out is now typed:
  `HQ_SYNC_SUBSTRATE_OPTIONAL=1` is the only door to a skip; anything else `t.Fatal`s. Demonstrated
  live in all three arms (unreachable+unset → FAIL exit 1; unreachable+`=1` → SKIP; reachable →
  RUN, ~~27 subtests executed~~ **54 subtests in `TestRowVisibilityRLS` and 16 in
  `TestJWTBridgeRLS`** — the 27 was wrong, corrected at the G6 fix round, finding F7; it matched
  neither suite, and the row-visibility count rose from 52 to 54 when that round added WP8/W16).
  Pinned by `spikeResolution`, a pure function `resolveSpikeConfig`
  switches on directly, whose test enumerates all 8 combinations **and asserts exactly one is a
  skip** — so an implementation that skipped everywhere cannot pass it (B-22/B-23/B-24).
  🛑 ~~**Ledger T-29 decision 108's reporting rule can now be retired: the package `ok` line means
  something again.**~~ **STRUCK — FINDING F6, and decision 108's rule is KEPT.** B-36 closes ONE
  road to a silent skip; it does not close the road. Verbatim, on the fixed tree:
  `HQ_SYNC_SUBSTRATE_OPTIONAL=1` still produces `--- SKIP: TestJWTBridgeRLS` / `--- SKIP:
  TestRowVisibilityRLS` / `PASS` / `ok  github.com/yumyums/hq/internal/sync  0.019s` — exit 0, zero
  attack variants run, **and the `ok` line does not say the variable was set.** `-run` filtering is
  a second road: `-run TestSpikeGate` prints `ok` having run no variant at all. **The amendment: an
  `internal/sync` result is reportable only when it cites `-run TestRowVisibilityRLS -v` with the
  subtests EXECUTED, and states that `HQ_SYNC_SUBSTRATE_OPTIONAL` was unset.** A bare `ok` line
  from this package remains unreportable. · **Still open, deliberately:** `HQ_SYNC_REST_URL` remains **ARMED AND
  UNSET** — making push *possible* is not making it *live*, and this card set, referenced and
  implied it nowhere. · Discoveries routed under scope freeze: **B-49** (rejections just became the
  fourth collection B-42's unscoped live leg applies to — filed as B-46, renumbered at triage when
  the B1 leg merged first and kept that number), **B-47** (`sync-schema/sql/` has no
  applier or manifest), **B-48** (no static guard on the templates deny-all — **re-specified at the
  G6 fix round**: the original lead proposed guarding a non-defect, see F3), **B-51** (the
  "two independent gates refuse a DELETE" claim was false; `authenticated` holds DELETE and
  TRUNCATE on all four tables, so there is one gate — finding F4).

  · 🛑 **G6 ADVERSARIAL REVIEW RETURNED THREE BLOCKING DEFECTS, ALL IN THE SUITE, NONE IN THE
  POLICIES — AND THE SUITE IS THIS CARD'S DELIVERABLE.** The shipped `0004` refused every attack
  the reviewer constructed, `Prefer: return=minimal` included. What it could not do was prove it:
  **3 of 5 mutations to the shipped write predicates survived fully green, and 2 of the 3 were
  mutations the file itself names as guarded.** Root cause **F1**: `spikeStack.do` set
  `Prefer: return=representation` unconditionally, so every write went through PostgREST with
  RETURNING and Postgres applied **0003's SELECT policy to the new row** — and since the read
  predicates are identical to (rows 2, 3) or broader than (row 4) the write predicates, **the read
  policy was silently enforcing the write half.** W8's rejection arm was reading a 403 issued by
  `hq_can_see_field`. Fixed by `rvPushRefused`: every write refusal now goes out under BOTH Prefer
  headers. **F2**: `submission_rejections_update`'s `with check` — the ONE predicate in the file
  narrower than its table's SELECT policy, hence the only one Postgres will not enforce for us —
  had no variant of either sign; the matched pair **WP8/W16** was added on the approver-UPDATE axis
  (a new fixture field `fldApprover2` makes the pair run the identical PATCH probe). **F3**: §5's
  "single most load-bearing sentence" was factually wrong about PostgreSQL in both halves —
  Postgres substitutes `using` for an omitted `with check`, so on rows 2 and 3 that omission is a
  **non-defect**, and no test was invented to pretend otherwise. Corrected in place with a
  four-probe isolated experiment. **Mutation set re-run after the fix: M1 survives (correctly, a
  non-defect), M1b survives (correctly), M1c → W16, M1d → W4/W6/W8/W13/W14, M2 → WP3/WP4/W14, M3 →
  W3, M4 → W8.** Every reachable mutation is now caught.

  · **The card AS AUTHORED follows, kept verbatim rather than rewritten, so the contract this was
  built against stays readable next to what was built.** Authored at slate planning 2026-07-31
  evening (ledger T-30 decision 111).
  🛑 **BINDS: the replication-scope rule (ledger T-29 decision 105).**
  · **Why it exists:** `sync-rxdb-row-visibility-rls` shipped **SELECT policies only** and deferred
  writes to "a follow-up card that writes `WITH CHECK` policies". **No such card existed.** Verified
  at source before authoring: `sync-schema/sql/0003_rls_policies.sql` contains zero `WITH CHECK`,
  zero `FOR INSERT`, zero `FOR UPDATE` — RxDB **push** replication is deny-all today, and
  `sync-hard-cutover` makes RxDB the single write path. The cutover was blocked on a card nobody had
  written. Authored here rather than folded into the cutover for the same reason
  `sync-replication-scope-per-checklist` was: it carries a permission semantic, and hiding a new
  permission semantic inside a write-path swap is how both get reviewed badly.
  · **The contract is decided, not open — decision 111, per table:**
  `checklist_templates` INSERT/UPDATE stay **deny-all** (the builder keeps the REST path; no phone
  writes a template definition) and a refusal variant asserts it rather than leaving it an absence;
  `checklist_submissions` gets `WITH CHECK (public.hq_can_see_template(template_id))`, which closes
  the lie `0003_rls_policies.sql:243` already names by hand; `submission_responses` gets
  `WITH CHECK (public.hq_can_see_field(field_id))` — **field-scoped, never submission-scoped**,
  because `submission_id` is nullable for offline drafts and that is the whole offline story;
  `submission_rejections` gets an **approver-only** write and, necessarily, **a SELECT policy it
  does not have today**.
  · 🛑 **Two consequences that are spec lines, not discoveries.** (1) **Approvals must become
  readable.** `submission_rejections` is RLS-enabled with zero policies — deny-all both ways, and
  variant V17 asserts it. A device that can write a row it cannot read back breaks replication, so
  the card adds `for select using (public.hq_can_see_field(field_id))`, matching
  `submission_responses`: the assignee whose field was rejected must be able to read their own
  feedback. (2) **This is the FIRST place `assignment_role` is filtered on.** `hq_can_see_template`
  deliberately never filters on it — an `'approver'` sees what an `'assignee'` sees, recorded as
  "knowing, not accidental", with the standing note that changing it is a separate card. Decision
  111 changes it **on the write side only**: a new `public.hq_can_approve_template(tid)` =
  `EXISTS` an assignment with `assignment_role = 'approver'` **OR** the unconditional
  `roles && ARRAY['admin','superadmin']` admin arm. Reads keep the old property byte-for-byte. That
  asymmetry is the decision; do not "tidy" it by making reads match.
  · **Its gate is an attack-variant suite, red-first**, extending the shipped 27-subtest
  `backend/internal/sync/rowvisibility_rls_test.go` rather than starting a new file — write variants
  first (push a submission claiming a template you cannot see; push a response on a field you cannot
  see; push a template edit as an assignee; write an approval as a plain assignee; write an approval
  as an approver — positive; admin writes everything — positive), each captured **refusing before
  the policy that refuses it is written**. A 200 alone proves nothing; keep the `service_role`
  BYPASSRLS control and the population floors that rule out "the table was empty."
  · **🛑 ALSO FIXES B-36, and that is not a scope breach — it is this card's own gate.** `internal/sync`
  prints `ok` and exits 0 while `t.Skip`-ing the entire RLS suite whenever `resolveSpikeConfig`'s
  `docker compose … port` shell-out fails for any reason, so the package `ok` line carries **zero**
  information about whether the security suite ran. This card doubles that suite's size; shipping a
  second attack suite into a package whose gate cannot prove it executed is building on a foundation
  that needs rip-out. Gate `t.Skip` on an explicit `HQ_SYNC_SUBSTRATE_OPTIONAL=1` opt-out and
  `t.Fatal` otherwise. Ledger T-29 decision 108's reporting rule stands until this lands.
  · **PARK triggers:** a write predicate the operator has not decided (anything beyond the four
  rows of decision 111's table); a schema change to HQ's own tables (this card is substrate-side
  policies plus Go tests — it must not need a `backend/migrations/` file); `submission_responses`
  turning out to need submission-scoped writes after all, which would reopen the offline-draft story.
  · Footprint: `sync-schema/sql/0003_rls_policies.sql` (or a new `0004_write_policies.sql`),
  `backend/internal/sync/rowvisibility_rls_test.go`, `backend/internal/sync` (the `resolveSpikeConfig`
  skip path, B-36).

- **`sync-cache-and-identity-hygiene`** · **DONE — one mechanism, three call sites, obligation 8
  folded** (2026-08-02, run `overnight-20260802`, Track B; landed on
  `card/b1-sync-cache-and-identity-hygiene`) · **The cross-tenant disclosure was SHOWN before it was
  closed**, end-to-end through a real service worker in the suite's only
  `serviceWorkers:'allow'` spec (`tests/sw-api-cache-partition.spec.js`): a `team_member` with no
  Users grant, offline on a device user A had used, was handed **HTTP 200 and the entire team
  roster** — every colleague's email, role and employee number. Post-fix the same request returns
  `503 {"error":"offline"}`. `api-cache` is **partitioned**, not retired: `cacheKeyWillBeUsed`
  writes `__hq_id=<uuid>` into every cache key and `cacheWillUpdate` refuses to write at all when no
  identity is established. The token lives in a `hq-identity` CacheStorage bucket — the only store
  the page and the worker can both reach (a worker cannot see `localStorage`; the session cookie is
  `HttpOnly` and is attached after the fetch event). The same token drives the purge at all three
  call sites: `logout()`, `login.html`'s `signIn()`/`acceptInvite()` (obligation 7b — the identity
  change that never runs `logout()`), and `establishIdentity()` on a verified `/api/v1/me`, which
  also prunes any foreign partition. `hq_apps` became an identity-stamped `{uid,apps}` envelope and
  a legacy bare array is **discarded, not migrated** (obligation 7a). Obligation 8's stale comment
  at `tests/sync.spec.js:1584` is corrected — `'submitted'` → `'completed'`
  (`repository.go:715-716`); no test in that file was touched and **[LST-17] stays armed**. Precache
  count unchanged at **29 files**. 🛑 The merge-intent note
  (`.night-crew/runs/2026-08-02-autonomous/merge-intent-b1-sync-cache-and-identity-hygiene.md` §2)
  is the contract P1 and S1 are held to when they edit `build-sw.js` after this card.
  *(Original re-specification, kept as the record of why the card exists:)*
  **RE-SPECIFIED 2026-07-31 evening (ledger T-30
  decision 112): the `api-cache` retirement is STRUCK, and the card is now per-identity cache
  partitioning.** 🛑 **The retirement premise was false, and verified false at source:**
  `build-sw.js:149` registers `urlPattern: /\/api\//` — a NetworkFirst route over **every** endpoint
  in the app, all five tools. RxDB replicates **four** collections, all of them `workflow`. Retiring
  the route would take offline API reads away from Inventory, Users, Onboarding and Purchasing,
  which RxDB has never covered; decision 105's per-open-checklist scope narrows RxDB's coverage
  further, so the argument is weaker now than when it was made. **What obligation 3 actually names
  is a cross-tenant read** — a URL-only cache key with no `Vary`, so user B's device serves user A's
  cached API responses. **The card's mechanism is therefore: key the cache by identity and purge it
  when identity changes** — which is the SAME mechanism obligations 7(a) and 7(b) need, so the card
  collapses from four errands into one mechanism with three call sites. **Red-first is mandatory:**
  the red is a test in which user A's cached API response is served to user B on the same device.
  · **BINDS: the replication-scope rule (ledger T-29 decision 105)** — but note the rule no longer
  *blocks* this card, it only explains why the retirement is wrong. (fanned out of
  `sync-rxdb-schema-and-replication` obligations 3, 7 and 8 at the 2026-07-28 dissolution; ~~depends
  on `sync-rxdb-replication-and-conflict-handler`~~ — **that dependency is DISCHARGED and, under
  decision 112, was never load-bearing anyway: it existed only to make the retirement correct, and
  there is no retirement**) · **The split is a gain here, and the dissolved card
  said why:** it carried these under an *"accepted cost: if this card slips, they slip with it."*
  They no longer ride the biggest card in the cycle, and three of the four items are cross-tenant
  disclosures.
  **Obligation 3 — `api-cache`, decision 57, structural half. ~~Retire the route entirely.~~
  SUPERSEDED by decision 112 — partition it by identity instead.** The URL-only cache key with no
  `Vary` is a cross-tenant read; the immediate mitigation already shipped in
  `pwa-cache-and-build-hygiene`, and this card owns the structural fix. *(The struck text read:
  "the expected answer is to retire the route entirely — once RxDB replicates, offline data comes
  from IndexedDB and `api-cache` is obsolete." Kept as the record of why the obligation exists.
  It is wrong for the reason recorded above: the route covers all five tools and RxDB covers four
  `workflow` collections.)* Note also that the cited anchor `build-sw.js:60-78` has **moved** — the
  route now registers at `:149`; find it by `cacheName: 'api-cache'`, not by line (B-25).
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

- **`sync-rxdb-conflict-notice-ui`** · **DONE** (2026-08-01, run `overnight-20260801`, card branch
  `card/c2-sync-rxdb-conflict-notice-ui`) · The user-visible half of decision 50 is built:
  `sync-rxdb/conflict-notice.js` (model — counting, retention, cap, the A-3 label read) and
  `sync-rxdb/conflict-notice-ui.js` (banner, sheet, A-2 confirm), mounted from `workflows.html`.
  **A-1, A-2 and A-3 are all implemented as obligations, not as the struck rules.**
  **🛑 DORMANT BY CONSTRUCTION, said plainly rather than discovered later:** the sheet reads the
  durable local record, the record is written when `conflict$` fires, `conflict$` fires when
  replication runs, and replication is not started (`HQ_SYNC_REST_URL` unset everywhere). So in
  production today there are zero records and the banner never appears. **No write path was
  swapped** — that is `sync-hard-cutover`, which switches the producer on and replaces the injected
  `applyRestore`. Every state is therefore verified from a **seeded store**, through the same seam
  the cutover card will hand a real RxDB collection to.
  **A-3's redraw landed** — `edge-removed`, `openq-count-a`, `openq-count-b`, with `edge-removed`
  now drawing BOTH the struck-through label and the raw-id fallback, because B1's item **R-C**
  (nothing validates `template_snapshot`) means the fallback is where every malformed snapshot
  lands. **Deviation from the signed plates, noted in SUMMARY.md.**
  **Carried from C1's G6:** `describeConflict` now threads the handler's own merge opts, so the
  sheet cannot report a clash list the handler never produced. Red-first, 4/4.
  **One real bug caught by the guard, worth not repeating:** a CSS comment written
  `/* --ok-*/ … */` closed itself at the `*/` inside `--ok-*` and swallowed the whole `:root`
  variable block, leaving **every colour on `workflows.html`** unresolved. The storage plate's
  contrast assertion is what reddened.
  Contract at `.planning/phases/sync-rxdb-conflict-notice/PLAN.md` (30 `done_when:` rows, 14 State
  Enumeration rows, each with a population floor); 46 PNGs at
  `test-results/states-sync-rxdb-conflict-notice/`.
  *(Original card text follows, for the record.)* · **✅ PLANNED — SIGNED OFF, SLATE-READY** (revision 2 signed by
  the operator at morning triage 2026-07-29; ledger T-28 decision 98) · **🛑 NO LONGER
  ATTENDED-BLOCKED.** All 16 plates were walked attended — read back as PNGs, light renders, not
  described from the spec — and amendments **A-1** and **A-2** hold at the worst case rather than the
  easiest: A-1's three banner lines coexist at 480px with no truncation on the
  four-answers/two-handled/two-unidentifiable plate, and A-2's confirm names the loss in its title
  (*"Replace 3 of Dana M.'s answers?"*), lists all three server values struck through with who saved
  each and when, and labels its primary button **Replace** rather than Restore.
  **Three operator decisions came out of the walk and are build obligations:**
  **A-3 (decision 95) — a removed question keeps its label, struck through and read-only.** The
  operator's words: *"show the deleted question crossed out and read only so that the user isnt
  confused."* This **supersedes both drawn readings of open decision (i)**. The committed plates
  `edge-removed`, `openq-count-a` and `openq-count-b` draw the raw field id `fld_prep_sink_temp` in
  muted monospace on the grounds that "the template no longer holds a label for it" — true of the
  template, **false of the submission**: `template_snapshot` is `json.Marshal(tmpl)` of the whole
  template (`repository.go:695`) and `Field.Label` is on it (`model.go:44-57`), so the discarded
  document carries its own frozen label. Buildable with no new schema requirement, but it makes the
  snapshot's *shape* load-bearing — B1's recorded-not-fixed item R-C (`template_snapshot` is
  `{type:'object'}`, nothing rejects a malformed value) becomes a dependency, not an open question.
  🛑 **Those three plates must be redrawn and the deviation noted in SUMMARY.md** per CLAUDE.md's
  mockup rule. Counting follows Reading A (the headline counts what was taken from the crew member;
  the `+N` line keeps meaning only "we couldn't identify") — recorded as triage's inference from the
  rider, not as operator words.
  **Decision 96 — retention stays 30 days.** Decision 80 stands; read from one named constant, no
  surface restates the literal. Accepted costs are the ones the plate names: a longer list, and a
  promise the device may not keep (local-only, per-device, evictable — hence the storage-error plate).
  **Decision 97 — the sheet caps at 10 groups with an "and N more" line; no date filter.** Rows below
  the line are not dropped and the banner still reports the true total.
  **Inherited hard requirement:** A-2 hardens who-and-when on the row from a graceful degradation into
  a **hard requirement on `sync-rxdb-schema-and-replication`** — if that card declines to carry it,
  the confirm plate cannot be built as drawn. **Triage's own reservation, recorded not blocking:**
  `a1-banner` puts four figures on one screen plus a batch button reading a fifth — all consistent,
  all documented, but a lot of counting at 6am with wet hands. Watch it in the built UI.
  **Superseded status line and original card text follow.** · **🛑 WAS: PLANNED — ATTENDED-BLOCKED
  AGAIN (reverted from
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
  ~~**🛑 THIS CARD IS STILL ATTENDED-BLOCKED.** A revised mockup existing is not a sign-off, and the
  fan-out card did not and could not discharge it. **What is owed: the operator walks the 16 plates,
  answers open decisions (i) and (ii), and gives (or refuses) an explicit *"ok, build this"* on
  revision 2.** A *no* remains a successful outcome.~~
  **🛑 STRUCK 2026-08-01 by the card that ran. This block was TRUE when written and FALSE from
  morning triage 2026-07-29 onward** — the operator walked all 16 plates, gave the signature
  (decision 98) and answered both open decisions (95, 96) plus the cap (97). It is struck rather
  than deleted because it is the record of what was owed, but it was left standing in the
  imperative below a header that already said SIGNED, which is exactly the shape of trap the r1
  sign-off block further down documents: **an unattended reader going top-down could take it as
  binding and park a signed card.** The card's live status is the DONE block at the head of this
  bullet.

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

- **`sync-replication-scope-per-checklist`** · **DONE — built on `overnight-20260802` (Track A,
  card A1), 2026-08-01, branch `card/a1-sync-replication-scope-per-checklist`.**
  **🛑 BUILT AND HANDED TO THE ORCHESTRATOR FOR MERGE — DEPLOYED TO NOTHING, and
  `HQ_SYNC_REST_URL` REMAINS ARMED AND UNSET.** Flipping this bullet DONE ahead of the merge is
  run convention. Nothing on this card authorizes a deploy, and this card specifically must not
  be read as evidence that the env var can be set: it narrows what replication asks for, it does
  not open the door. · Authored at morning triage 2026-07-31 (ledger T-29 decision 105), sized
  and sequenced by triage rather than put to the operator: it is a bounded change to code that
  already landed, and it gates `sync-hard-cutover`, so it went **before** the cutover rather than
  inside it. · **What it did:** every replicated collection now carries a `pull.queryBuilder` —
  `templates` `id.eq.<templateId>` (else `archived_at.is.null`), `checklists`
  `id.eq.<checklistId>`, `approvals` `submission_id.eq.<checklistId>`, and `responses`
  `or(submission_id.eq.<checklistId>, and(submission_id.is.null, field_id.in.(<the open
  checklist's own fields>)))`. `startHQReplication` now **THROWS** without an `opts.scope`
  carrying a `checklistId`: a default that fell back to the whole collection would widen
  preference `architecture/C-2` silently, and C-2 requires a recorded decision to widen. ·
  **Red first, as the card required:** `[SCOPE-01]` at commit `d5d2e4d` was **3 failed / 3
  passed** — the failing three include `an offline DRAFT on the open checklist still replicates`
  receiving `["rsp-1","rsp-2","rsp-3","rsp-4","rsp-5"]`, i.e. the whole table including two rows
  belonging to a checklist the device never opened. The three that passed at the red are the
  guards (non-empty subject set per B-22/B-23/B-24, the vendored-seam tripwire, the batch-size
  check) and passed on purpose. **6/6 green** after the fix at `80b2149`. The harness does not
  assert on an options object: it reproduces the vendored plugin's own pull construction — read
  out of `vendor/rxdb.bundle.js`, where `pull.queryBuilder` runs BEFORE the checkpoint `.or()`
  and PostgREST ANDs the two — and EVALUATES the emitted PostgREST filters over a two-checklist
  fixture. · **NO SCHEMA CHANGE, NO POLICY CHANGE — the PARK trigger did not fire.** Every key the
  scope uses was already declared by B1; `sync-schema/collections.js` and `sync-schema/sql/` are
  byte-unchanged. The roadmap's *"`sync-schema/collections.js` if the scope needs a queryable
  key"* proviso was not needed. · **Three scoping edges, all decided by C-2 rather than parked:**
  (1) a draft response has `submission_id IS NULL` until submit and drafts are the offline case
  this layer exists for, so the scope is the open checklist's submitted rows OR a draft on one of
  its **own** field ids — not "all my drafts everywhere", which is a different unbounded set (and
  it matches what the RLS predicate already does: `TestRowVisibilityRLS/V13` is literally *DRAFT
  responses are scoped by FIELD, not by submission*); (2) `templates` is scoped `id.eq.<templateId>`
  and `templateId` is REQUIRED — there is no "caller omitted it" fallback, because a widening
  triggered by a forgotten argument is not a recorded decision; (3) an unscoped call is refused
  loudly, and the refusal is raised while BUILDING the pull rather than inside `pull.handler`,
  which the plugin wraps in an unbounded silent retry. · **The `replicationIdentifier` CARRIES THE
  SCOPE — `hq-sync-<table>-<fingerprint of that collection's own filter>`.** 🛑 **This reverses
  what this bullet said when the card was first flipped DONE, and the reversal is the point:** the
  original text argued the scope must stay OUT of the identifier because folding it in would mint a
  blank checkpoint on every checklist switch. Measured against `vendor/rxdb.bundle.js`, that was
  backwards. RxDB keys its checkpoint meta store by `hash([collection.name, replicationIdentifier])`
  and **by nothing else** — the scope is not in the key — and the pull's returned checkpoint is the
  last row of the *scoped* result set, which the next pull ANDs `_modified > C` against. One
  identifier across scopes therefore meant: open today's checklist, checkpoint advances to today;
  open **yesterday's**, every one of its rows is `<= C`, **zero rows, permanently**. The cost the
  old reasoning was protecting against had already been removed by this same card: before it, a
  blank checkpoint meant re-pulling all history (20 pages × 50 rows); after it, it means re-pulling
  **one checklist** — about one batch. Identical scope still resumes; only a scope change mints a
  new checkpoint. Found by the G6 adversarial review as BLOCKING finding F-1 and fixed in the same
  card; pinned by `[SCOPE-02]`, which runs the plugin's pull construction twice through a meta
  store keyed the way RxDB keys its own. · 🛑 **STILL OPEN,
  and it is `sync-hard-cutover`'s: `B-42 SYNC-REALTIME-SCOPE`.** The PULL is scoped; the plugin's
  live `postgres_changes` subscription is not, and has no seam — `pull.modifier` reaches stream
  documents but cannot drop one (no null filter on the downstream path), and Realtime's own
  `filter` takes a single clause, which cannot express the `responses` branch. Bounded: the pull
  was the unbounded leg (all history, every page load) and is now scoped; the stream is bounded by
  what other people change while the crew member is looking. Stated in `sync-rxdb/client.js`'s
  `REPLICATION SCOPE` header, not only in the backlog. · 🛑 **UNBOUNDED PHONE STORAGE IS IMPROVED,
  NOT FIXED — reworded after G6 finding F-6, which caught this bullet claiming otherwise.** RxDB's
  downstream only ADDS to the local store; nothing evicts, and there is no retention sweep for any
  of the four replicated collections. A phone still accumulates every checklist it has **opened**
  — the bound moved from *all history* to *opened checklists*, which is a large improvement and is
  not the same thing as bounded. Whoever owns retention owns it after `sync-hard-cutover`. · 🛑
  **ALSO FIXED IN THE G6 ROUND, same card:** scope values are validated against a strict
  whitelist and quoted into PostgREST's logic-tree grammar (F-4 — an unescaped value could
  otherwise rewrite the predicate to match every row, *through* the thing this card calls a gate);
  and the `[SCOPE-01]` fixture gained a second submission of the OPEN checklist's OWN template plus
  approval/response rows on OPEN field ids under a different submission (F-2 — without them the
  fixture could not tell per-checklist scoping from per-template or per-field scoping, and the
  reviewer's two mutations both survived 6/6 green). · 🛑 **STILL REQUIRED AND NOT DONE BY THIS
  CARD: re-measure the ~23 ms/row constant on production-like topology.** It was measured through
  Docker loopback NAT, which production does not have — the linear *shape* is structural, the
  *constant* is not, and no card should rely on the specific number until it is re-taken. This
  card removed the multiplier (20 pages × 50 rows), not the constant. · Footprint as built:
  `sync-rxdb/client.js`, `tests/sync-rxdb-client.spec.js`, plus one line of doc in
  `sync-rxdb/bootstrap.js` and a scope argument in `tests/sync-rxdb-conflict.spec.js`'s driven
  threading test, which genuinely calls `startHQReplication` and would otherwise red on a change
  it is not about.

- **`sync-cutover-list-scope`** · **PLANNED — SLATED 2026-08-02 evening as a COMMITTED card on
  `overnight-20260803`, first in Track A** (`reference/slate-20260803.md`) · *(½ of the fanned-out
  `sync-hard-cutover`; authored at slate planning under the §1 split rule — see the fan-out note
  above)* · 🛑 **Exists because of an operator product decision taken inline at planning, 2026-08-02:
  B-43, "lists stay live — widen the scope."** `sync-replication-scope-per-checklist` (run
  `20260802`, A1) made `scope.checklistId` **mandatory and singular**, which is exactly right for the
  checklist-fill view `architecture/C-2` names and is **not the view a crew member lands on**:
  `workflows.html` opens on **My Checklists** (every submission assigned to this user) and its second
  tab is **Approvals** (every rejection awaiting this user). Both are lists over MANY submissions.
  The operator chose to keep the live-list behaviour `sync.js`'s WebSocket provides today rather than
  trade it for a smaller card. 🛑 **This slate IS the recorded decision that amends ledger T-29
  decision 105** — the rule is not repealed, it is amended to: *per-open-checklist for the fill
  collections; per-user-with-a-date-floor for the two list collections; never all history, never all
  users.* · Give `normalizeScope` / `scopeFilterFor` (`sync-rxdb/client.js`) a **list scope** beside
  the fill scope — `checklists: assigned_to.eq.<userId>` plus a **mandatory date floor**, and the
  approver-side collections scoped to this approver. 🛑 **The date floor is not optional and is a
  `done_when:` row**: B-42 already recorded that nothing evicts, so the per-phone bound only moved
  from *all history* to *opened checklists* and a per-user list scope widens it again. Apply **B-42
  option (i)** — Realtime's single-clause `filter` on `checklists`, `approvals` and
  `submission_rejections`; `responses` stays unfiltered on the live leg with the residual recorded at
  the call site (its predicate needs `or(submission_id.eq.X, and(submission_id.is.null,
  field_id.in.(…)))`, which one clause cannot express). **B-49 is a line item on that fix, not a
  separate one.** Folds **B-58**: add the one discriminating subtest for
  `submission_rejections_update`'s `USING` clause — substituting `hq_can_see_field` for
  `hq_can_approve_field` today leaves all 54 subtests green, which is the clause T-31 decision 121
  named as E-KR2's stated caveat — and correct the three comments that assert a guard measurement
  says is not one. **Rejected framings, recorded so they are not re-derived:** *fill-view-only*
  (cheapest, matches C-2 literally, but the crew loses live lists they have today and the offline
  story splits into two owner classes E-KR3 is graded against) and *poll-on-focus* (recovers most of
  the behaviour without a widening, but adds a second refresh mechanism alongside RxDB's own, which
  is the dual-offline-story E-KR3 forbids). **PARK if the list scope needs a SCHEMA change** (a
  queryable key on the row, a new column, a view) — that is B-42 option (ii), explicitly a different
  card's; or if a policy change would widen SELECT beyond the four rows decision 111 authorises.
  Footprint: `sync-rxdb/client.js`, `sync-schema/sql/**` (a `0005_*.sql` only if the scope needs a
  policy it lacks — **not** a table, column or role), `backend/internal/sync/rowvisibility_rls_test.go`,
  `.night-crew/knowledge/designs/`. 🛑 **`sync-rxdb/bootstrap.js` and `workflows.html` HARD-untouched**
  — the cutover owns the wiring, and this card changes **no** write path.

- **`sync-hard-cutover`** · **PLANNED — LAST · SLATED 2026-08-02 evening as a COMMITTED card on
  `overnight-20260803`, second in Track A, cut AFTER `sync-cutover-list-scope` merges**
  (`reference/slate-20260803.md`; supersedes the 2026-07-31 stretch slating on
  `overnight-20260801-2`, which the budget never reached). · *(½ of the fanned-out card — it keeps
  the name so P-KR3's "the no-parallel-run constraint carried into the WO verbatim" still names the
  WO it was written about)* ·
  🛑 **FOOTPRINT CORRECTION, verified at source 2026-08-02 — the bullet below said
  "`backend/internal/sync` (deleted)", which was true when written and is now FALSE.** That package
  today holds the op-log (`hub.go`, `listener.go`, `ops.go`, `handler.go` — **delete these, this is
  what the cutover retires**) *and* `jwtbridge.go` + `jwtbridge_handler.go` (**KEEP** — RxDB cannot
  authenticate without the bridge) *and* `proxy.go` (**KEEP** — it is the door RxDB talks to) *and*
  `rowvisibility_rls_test.go` (**KEEP** — 54 subtests, E-KR2's entire evidence) *and* `access_test.go`
  / `spikestack_gate_test.go` (**KEEP** — substrate gates). **A card that deleted the package would
  delete the bridge, the door and the milestone's own proof.** This is a `done_when:` row, not a
  footnote. ·
  ✅ **ITS INHERITED PRODUCT FORK IS RESOLVED — ledger T-30 decision 113, so this card is
  fork-free and slateable.** `sync-rxdb/conflict-handler.js:105-160`'s `🛑 OPEN QUESTION INHERITED
  BY sync-hard-cutover` — *"should an intentional delete beat a concurrent edit?"* — is answered:
  **the uncontested delete still wins, and the annihilated edit is REPORTED and RECOVERABLE.** Who
  wins does not change; the silence does. The card MUST emit a conflict record carrying the
  discarded edit so it lands on C2's existing conflict sheet with the Restore affordance — **the
  same surface, no new component** (a new component would need mockup sign-off an unattended run
  cannot obtain). 🛑 **EXTEND the pinned test *`_deleted` participates in the merge — an UNCONTESTED
  local delete survives*, do not replace it** — it asserts the winner and the winner is unchanged;
  the new assertion is that the loss is reported. Rejected framings, recorded so they are not
  re-derived: *edit-blocks-delete* (a crew member on the truck observed the item was not done and
  should not be silently overruled by someone who was not there; a box that re-checks itself is
  worse than a reported loss) and *last-write-wins* (depends on device clocks agreeing — the exact
  assumption class this migration exists to escape). **Why it is not an edge case:** HQ hard-deletes
  three of the four mirrored tables from live paths — `saveResponse` (`repository.go:811`) DELETEs a
  response whenever a value goes null, **which is unchecking a checkbox**, the tool's
  highest-frequency write. · **Two further inherited notes, engineering-level, not forks:**
  `makeSyncFetch` loses method and body when handed a `Request` argument, and `formatValue`'s
  now-bounded unwrap needs care once a network-fed producer exists. · 🛑 **BINDS: the replication-scope rule (ledger
  T-29 decision 105).** Replication scope is **per-open-checklist, never all collections at once**.
  This card owns the write path and is the moment the scope becomes load-bearing: `startHQReplication`
  today loops all four collections with `pull:{batchSize:50}` and **no selector**, which is what
  produced Fork 1's ~23 s figure and what makes every phone hold every response ever taken. The
  cutover must land the pull filter, not inherit the full-collection pull. A card may not widen this
  scope without a recorded decision. (dependency restated at the 2026-07-28 dissolution:
  depends on **all three** of `sync-rxdb-collections-and-table-contract`,
  `sync-rxdb-row-visibility-rls` and `sync-rxdb-replication-and-conflict-handler`, plus jwt-bridge.
  🛑 **AND, added 2026-07-31 evening: `sync-rxdb-write-policies` and
  `sync-replication-scope-per-checklist`.** The write-policies dependency is **hard and was
  previously invisible** — RxDB push is deny-all until `WITH CHECK` policies exist, so this card
  cannot make RxDB the single write path without it. That is now five dependencies, not three.
  🛑 **Row-visibility RLS is not optional for this card specifically** — the cutover makes RxDB the
  single write path, which means it is also the moment `HQ_SYNC_REST_URL` gets set in a real
  deploy) · Replace
  BOTH current write paths in `workflows.html` — `autoSaveField`→`POST /saveResponse` and
  `sync.js`'s WebSocket/ops-log broadcast — with the RxDB store as the single write path. Retire
  `sync.js`, `backend/internal/sync/`, and `/saveResponse` entirely. Hard swap, no parallel run
  (per the explore session — no need to keep the old system live during cutover). Reconcile the
  existing Workbox service-worker offline caching against RxDB's own local persistence so there
  is exactly one offline story, not two (Workbox keeps owning static-asset caching; RxDB owns
  data). ~~Footprint: `workflows.html`, `sync.js` (deleted), `backend/internal/sync` (deleted),
  `backend/internal/workflow` (`/saveResponse` removed).~~ **Footprint superseded — see the
  correction at the head of this bullet.** Actual: `workflows.html`; `sync-rxdb/bootstrap.js`;
  `sync.js` (**deleted**); `backend/internal/sync/{hub,listener,ops,handler}.go` (**deleted — and
  nothing else in that package**); `backend/internal/workflow` (`/saveResponse` removed);
  `backend/cmd/server/main.go`; `build-sw.js`; `sw.js`; `tests/**`; `version.go` + `package.json`.
  · **Riders added at slate planning 2026-08-02, each with its reason:**
  **(a) B-54 — write the precache pin.** T-31 decision 123 already settled B-54's (a)-or-(b) fork in
  favour of (a), having watched the count go 31 → 32 in silence within an hour of B-54 being filed.
  This is the next card to edit `build-sw.js` and the first that may legitimately move the number, so
  it pins the count in `tests/sw-manifest.spec.js` with the justification in the same comment **and
  re-bases the pin in the same commit**.
  **(b) B-20 — carry the banner scoping forward, do not fix it in place.** `sync.js:671`'s
  document-wide `[data-template-id]` selector paints the Queued badge onto Builder rows; this card
  **deletes that file**, so P3 was excluded from the slate as wasted work. Whatever replaces
  `renderSyncBanner` must be scoped to the checklist list, and **this card's report gives B-20 its
  disposition** — fixed forward, or evaporated with the file.
  **(c) Revisit `bootstrap.js`'s fail-soft — its own header instructs this card to.** It currently
  records a construction failure on `window.HQSync.error` rather than throwing, which is correct
  while nothing depends on it. This card makes the page **depend** on it, and a page whose checklists
  silently stop persisting because the sync layer swallowed an error is worse than a loud failure.
  State the decision either way; do not inherit it.
  **(d) `build-sw.js:29`'s stale Taskfile citation** — one line, the same false claim card P1 fixed
  elsewhere. Drive-by.
  **(e) E-KR3 is closed here or not at all:** exactly **one** design note in
  `.night-crew/knowledge/designs/` naming each offline data class and its single owner, cross-checked
  against the `build-sw.js` / RxDB-init diff, **0 classes with dual or ambiguous ownership.**
  🛑 **Do NOT change `playwright.config.js:60`'s repo-wide `serviceWorkers: 'block'`** (B-15) — every
  prior card has held that line. 🛑 **PARK triggers:** a new terminal status or any lifecycle value
  the schema does not carry; retiring `/saveResponse` reopening ledger **decision 49** (the
  server-side duplicate guard — a recorded fork, not a judgement call); or the two-device list proof
  being unconstructible at all — a cutover whose central claim cannot be proven is a park with
  evidence, not a merge with a caveat.

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
| Cross-user checklist hydration divergence (approved-vs-rejected ghost state) | ~~left `new` — needs a product ruling on desired cross-user semantics; routed to the next `/nc-pm-session` intake, not resolved this round~~ → **RULED 2026-07-26, operator-decided, ledger T-24 decision 67** (given directly in answer to the question put plainly — no PM session was run; the ceremony was heavier than the single question). *"A new cycle starts fresh for every user — 0/2 for both A and B."* B's rejected submission does **not** resurrect as current state: it is archived and stays visible as **history**. A's fresh 0/2 **must accept clicks** — *"the silent no-op is a bug, not intended behavior."* The convergence matrix still needs its missing asymmetric approved-for-A / rejected-for-B cell seeded, asserting 0/2 for both **and** that A's clicks POST. Rationale: *"rejection means redo."* The ruling defines the target state, not the work — the build is slated as Night B card **P4** (`reference/slate-20260802.md`) and is not yet authored. Product KR2's half of `sync-hard-cutover`'s double block is **cleared** (see `:110-125` above); that card stays blocked on `sync-rxdb-schema-and-replication` only. |

All other `· new` backlog items (security/infra hygiene, grants follow-ups, test/run-mechanics
hygiene, product/display nuance, money-precision) were grouped for this round's walk but not
yet walked — left untouched, still `new`.
