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
  `sync-hard-cutover` stays double-blocked — it also needs the "Cross-user checklist hydration
  divergence" backlog item routed through a `/nc-pm-session` (Product KR2). **Note for whoever
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

- **`pwa-cache-and-build-hygiene`** · **PLANNED — READY TO SLATE, NO DEPENDENCIES** (new card,
  authored at morning triage 2026-07-26 from ledger T-23 decisions 57, 58, 59) · 🛑 **Carries a live
  cross-tenant disclosure on shipped crew phones — this should not wait behind the sync work.**
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

- **`workflow-offline-double-submit`** · **PLANNED** (new card, authored at morning triage
  2026-07-26 from ledger T-23 decision 60; real, pre-existing, untouched by the status-vocabulary
  card) · Offline submit → reopen → submit again writes **two** `checklist_submissions` rows:
  `workflows.html:1656` mints a fresh `idempotency_key` per call and `:2778` handles `err.offline`
  by returning to the list without pushing into `MY_SUBMISSIONS`, so the checklist stays correctly
  editable and a second submit mints a second UUID past the `UNIQUE` guard. **Fix client-side —
  reuse the enqueued key on re-submit.** Explicitly NOT the server-side duplicate guard: that
  reopens decision 49 and trips the status card's park trigger for no added benefit. Footprint:
  `workflows.html`.

- **`sync-rxdb-schema-and-replication`** · **PLANNED** (depends on `sync-spike-stack-and-jwt-bridge`'s
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
  Footprint: `workflows.html`, new RxDB client layer.

- **`sync-rxdb-conflict-notice-ui`** · **PLANNED — ATTENDED-BLOCKED** (new card, fanned out of
  `sync-rxdb-schema-and-replication` 2026-07-26 at slating; depends on that card's
  `conflictHandler`) · The **user-visible half of decision 50**: when a same-field clash falls back
  to master-wins, `conflict$` must surface it to the crew member **with the discarded value
  recoverable** — not silently dropped. Cheap to *get hold of* (`conflict$` already emits the
  discarded document per-document with its id, verified); the expensive half is deciding what the
  crew member should see and how they recover the value. **Why this cannot run unattended:**
  CLAUDE.md requires a committed mockup at `.planning/.../<phase>/mockup.html` and an explicit human
  *"ok, build this"* before UI code on phases introducing new components, plus a State Enumeration
  Table and the verifier-subagent gate. **The operator owes a mockup sign-off before this card can
  ever be slated.** Footprint: `workflows.html`, the RxDB client layer.

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

- **`sync-hard-cutover`** · **PLANNED** (depends on schema+replication AND jwt-bridge) · Replace
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
