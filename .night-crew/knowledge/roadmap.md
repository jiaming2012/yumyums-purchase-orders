# Roadmap — "Prove & surface" cycle (trust the sync · surface the numbers)

> **Cycle:** Prove & surface — a MIXED cycle. (1) TRUST: close the live-sync convergence QA
> hole so the escaped-defect class stops reaching operator play, and retire the last carried
> test-debt. (2) SURFACE: ship the Inventory **Trends** (weekly spend by group) and **Cost**
> (per-menu-item margin + movers) tabs, gated through the Users app.
> **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA,
> validated 2026-07-19). **Produced:** 2026-07-19 attended `/nc-okr-session` (the roadmap's
> guaranteed producer, DESIGN §15u). Previous cycle archived at
> `reference/roadmap-2026-07-16-nothing-silently-lost.md`.
> **Trigger:** the just-closed cycle's own close note — "fold in the QA-coverage findings"
> (ledger T-17) — plus 3 operator-play escaped live-sync defects and 2 carried PARTIALs
> (Eng "task test exits 0"; Delivery "median WO cycle"). Feature scope operator-chosen at the
> OKR session: Trends = weekly-spend-by-group; Cost = margin table + movers; inline SVG/CSS
> charts (no new dep); gating = Users `app_permissions`; Cost-in-prod = accept sparse.

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan` sizes to a
  night. Cards carry a **module footprint** (for parallel tracks) and a **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** Cards-per-night is the planner's call against
  the night budget + quality bar (budget is a floor, not a ceiling).
- **Two gates, then two parallel tracks.** Activity 1 (PRD) blocks all build; Activity 2
  (design sign-off) blocks the **Feature** build track only. The **Trust** track (Activity 3)
  has no design-gate dependency and may start as soon as the PRD lands. Feature (Activity 4)
  is serialized after the Activity-2 sign-off.
- **Red-first is mandatory on every fix card** (QA KR2): the test fails before the fix,
  recorded in the WO record.
- **Per-card wall-clock timing is a standing output** on every build card this cycle
  (Delivery KR3 — the T-14 median baseline is N=23 / 22m28s).

## Module footprints (independent → parallelizable)

| Track | Frontend | Backend | Tests |
|---|---|---|---|
| Sync trust | `sync.js`, `workflows.html` | `backend/internal/sync` | `sync.spec.js`, `repro-cut-task.spec.js`, `broadcast-rerender.spec.js`, `workflows.spec.js` (waiver #1) |
| Inventory feature | `inventory.html` | `backend/internal/inventory`, `backend/internal/recipes` | `inventory.spec.js`, `recipes.spec.js`, `states-trends.spec.js`, `states-cost.spec.js` |
| Gating | `inventory.html`, `users.html` | `backend/internal/users`, `backend/internal/me`, `backend/internal/auth` (middleware) | `users.spec.js`, gating with/without-grant specs |
| Prod ops / process | — | deploy tooling, alert queue | run-mechanics (per-card timing, Cliq watch) |

---

## Activity 1 — PRD gate · *blocking, first*

> The "Prove & surface" PRD. No build WO dispatches before this lands.
> Produced by the evening `/nc-pm-session` + `/nc-pm-grill-back`.

- **`prd-prove-and-surface`** · **DONE** (signed 2026-07-19, `runs/2026-07-19-attended/sign-off.md`) · A single cycle PRD covering BOTH halves:
  (a) **Feature** — Trends/Cost per-tab State-Enumeration tables + `done_when` observable
  behaviors (Trends = weekly-spend-by-group chart+table; Cost = per-menu-item margin table +
  top/bottom movers; inline SVG/CSS; accept-sparse-prod for Cost margin);
  (b) **Convergence coverage contract** — the {viewer}×{editor}×{op-type}×{derived-view} matrix,
  with the escaped-defect→cell mapping (each of the 3 escaped defects maps to ≥1 would-have-caught
  cell); (c) the **gating decision** (tab-level gate via Users `app_permissions`; slug-vs-
  sub-permission modeling — operator-delegated, to be ratified at design); (d) the routing
  record for all 12 `· new` backlog items. `prd validate` green; requirement→(reproduced escape |
  named user-outcome) trace table. → Product KR1, KR2, KR3, KR4. *(attended evening session)*

## Activity 2 — Design gate · *attended; blocks the Feature build track (Activity 4)*

- **`prove-surface-openspec-design`** · **DONE — SIGNED 2026-07-20** (draft `08e81e1` merged `3d5fc17` overnight-20260721, G6 PASS; signed at morning triage, ledger T-18: **A4 = Option (i)** two per-tab slugs · **D2 = "Ungrouped" pseudo-group** · rider (b) rewritten to **umbrella semantics** (whole-app grant implies its tabs; per-tab grants for narrower access) · rider (c) signed expected · **B5 fold-in:** the gating WO also gates approve/reject endpoints. `designs/prove-surface-gating-and-endpoints.md` §8 is the signature record. **Activity 4 UNBLOCKED**) · The OpenSpec change for: (a) the **gating
  model** — how a tab-level grant is represented in `app_permissions` (dedicated slug vs a
  sub-permission column) AND the net-new **enforcement path** (a `RequirePermission`-style check
  on the Trends/Cost data endpoints — there is no permission middleware today, only
  logged-in-vs-not); the `/me`-style resolver extension that drives tab visibility; the Users-app
  admin-UI surface for the grant. (b) The **two aggregation endpoints** — by-week×by-group spend
  (tax-proration consistent with `period-summary`; NULL-`purchase_item_id` handling rule) and the
  **margin join** (`gross_amount − ingredient_cost_total`, `food_cost_%`, movers ordering).
  (c) The **convergence coverage contract** shape (which matrix cells, which derived views).
  **Operator sign-off on the design is the gate — 0 Feature build WOs dispatch before it**
  (auditable from ledger timestamps). → Delivery KR1 ("design signed before build").

## Activity 3 — Trust track · *parallel; independent modules; may start once the PRD lands (no design-gate dependency)*

- **`waiver1-isolation-fix`** · **DONE** (overnight-20260721 impl `544e68b`+`08c1bef`, G6 PASS, merged `24358f8`; root cause = ops-journal replay fetch storm, ungated `SUBMIT_CHECKLIST` re-fetch `sync.js:443`; literal `task test` exit-0 achieved + independently reproduced (473·0·6) — **waiver #1 formally retired**; NOTE: successor intermittent `sync.spec.js:1198` observed 1-of-2 G6 full runs → exit-0 not asserted deterministic, fork in DECISIONS-NEEDED) · Fix the 1 isolation-confirmed cross-test
  DB-pollution red (`tests/workflows.spec.js › approved checklist shows Approved badge and cannot
  be resubmitted` LST-08 — passes in isolation, `#toast` hidden in the full suite) by isolating
  its state dependency, then re-run the full suite to confirm **literal `task test` exit-0** —
  FORMALLY retires carried waiver #1 (Eng KR5 PARTIAL → PASS). Red-first: full-suite red is the
  baseline. Footprint: workflow-engine tests (test-only, no production change). → Eng KR5.
  *(carried from BACKLOG "Waiver-#1 last mile"; operator chose graduate 2026-07-19)*
- **`sync-pkg-unit-coverage`** · **DONE** (overnight-20260721 impl `0ebc81d`, G6 PASS, merged `38f2060`; 10-combo cartesian + entity-branch/dedup/negative coverage; superadmin N/A per `users_roles_check`; ESC-1 regression unweakened; approver-inclusion contract question surfaced → DECISIONS-NEEDED) · The `sync` package (0 Go tests today) gets
  `ResolveEntityAccess` coverage across all {role}×{assignment} combos, asserting recipient
  resolution unions admins + author + assignees; the escaped cross-user access defect carries a
  red-first unit test on the pre-fix code. Footprint: `backend/internal/sync` (+ `sync/access_test.go`).
  → QA KR1. *(from BACKLOG "Cross-user live-sync access matrix + sync-package unit coverage")*
- **`convergence-matrix-systematic`** · **DONE** (overnight-20260721 impl `c7b4ccd`, G6 PASS, merged `8249209`; 32-cell matrix: 24 covered / 8 N/A-with-reason; 13 new `MTX-*` cells; 65/65 ×3 fresh-DB `--retries=0` + independent G6 re-run; test-only, 0 cells PARKed) · Extend the convergence E2E matrix from
  SET_FIELD-only to {op-type ∈ field / submit / approve / reject} × {editor ∈ assignee /
  non-assignee-admin} × {derived-view ∈ field-value / correction-banner / readonly-mode / list
  progress-count}; each cell red-first then green across ≥2 devices; the 3 escaped defects each
  have a cell that reddens on the pre-fix build. 0 cells red at cycle end. Footprint: sync trust
  (`sync.spec.js`, `repro-cut-task.spec.js`, `broadcast-rerender.spec.js`; `sync.js`/`workflows.html`
  only if a determinism seam is needed). → Eng KR4, Product KR2, QA. *(from BACKLOG "Live
  approval-state convergence coverage" — the QA hole both escaped defects widened)*
- **`percard-timing-instrumentation`** · **DONE** (flipped at triage 2026-07-20, ledger T-18 — overnight-20260721's harness-measured per-card table (impl/G6/merge legs, epoch-stamped `timings.log`) IS the standing output; future build runs keep producing it and the cycle gate computes the KR3 median from `reference/card-actuals.md`) · Make per-card wall-clock timing the
  invariant build-run output for every build card this cycle (the `-0718` harness-measured table
  as standing practice) so the cycle gate can compute a real median vs the T-14 baseline
  (N=23 / 22m28s). Footprint: run-mechanics / process (no product code). → Delivery KR3.
  *(from BACKLOG "Per-card wall-clock instrumentation as a standing build-run output")*
- **`replay-fetchstorm-gate`** · **DONE** (2026-07-22, triaged T-19 — gate landed; de-flake tail + unclosed storm class graduated to BACKLOG) (promoted at triage 2026-07-20, ledger T-18 — operator: "promote it"; pairs DECISIONS-NEEDED §B2+§B3, same root cause) · Gate the ungated
  `SUBMIT_CHECKLIST` replay re-fetch (`sync.js:443`) with the proven in-file pattern
  (`(runner open) ∨ !silent`, as the `APPROVE_ITEM`/`SAVE_TEMPLATE` branches already do) so a
  fresh-context catch-up no longer fires `loadMyChecklists()` per replayed op — kills the
  reconnect fetch flood + mid-fill clobber window on real phones. Same card: harden the successor
  intermittent `sync.spec.js:1198 › temperature answer converges` (pre-existing, load-sensitive,
  downstream of the storm), revert A2's test-side `checkAllWithRepair` to plain clicks, optionally
  add the fetch-abort guard in `sync.js api()` (silences suite-teardown noise, §C), and flip the
  approver-inclusion pin comment (`access_test.go:402-425`) from reviewer-NOTE to contract per the
  signed B4 rider ("Everyone should see live ops" — fan-out = everyone with entity access).
  Red-first; production `sync.js` change → `task sw` + the attended two-device convergence flag
  re-arms on land. Footprint: sync trust (`sync.js`, `sync.spec.js`, `workflows.spec.js`,
  `access_test.go` comment). → Eng KR5 (determinism), QA. *(from DECISIONS-NEEDED-20260721 §B2/§B3)*
- **`syncspec-deflake`** · **DONE** (S1 shipped overnight-20260724, G6 verdict APPROVE;
  **test-side only — the two-device check did NOT re-arm.** `:1198` and `:525` FLD-LIVE-02
  KILLED with proven reds; LST-17 + GATE-04 HARDENED (not reproduced in 3 targeted contention
  legs each, labeled honestly). Determinism ARMED: 12/12 `--retries=0` contention legs + 5/5
  measured-quiet legs (loads 1.59–1.93, 0 foreign procs) + fresh-DB full-suite `--retries=0`
  green 541/0/6, independently replicated by G6 incl. a mutation probe proving the re-dispatch
  fails honestly on a real autosave break. `cycle-gate`'s no-retry gate is now ELIGIBLE. Line
  anchors dead — locate by title, e.g. `-g "temperature answer converges"` (now ~:1372).)
  (was: PLANNED — RE-AIMED at T-20 (2026-07-21); the original premise was
  wrong in its aim, right in its conclusion.) D1 PARKED this card 2026-07-20 and its evidence was
  re-derived twice. Net position: **the test IS flaky (16% overall; 20% under a concurrent
  Playwright suite — 4 red / 25 `--retries=0` legs), but NOT for the reason the card assumed.**
  Every red is a 12s timeout at **`sync.spec.js:1119`**, the `POST /ops` COMMIT wait — *not*
  `CONVERGE_TIMEOUT`, not a convergence assert. Debounce is 400ms (a 30× margin) and the journal
  gains **+3 rows on a red vs +5 on a green**: no `SET_FIELD` row, so **the op never fired. It was
  not slow. No timeout increase can fix this.** Mechanism: the prior de-flake *relocated* the
  clobber race rather than closing it — a stray WS catch-up `loadMyChecklists` re-render detaches
  the input between `fill()` and `dispatchEvent('change')`, so the change handler never arms
  `debouncedSaveField`. **Fix is TEST-SIDE** (settle the runner before entry; re-assert and
  re-dispatch when no POST is observed) — **no production change needed**, so this does NOT re-arm
  the attended two-device check. Evidence: `reference/1198-flake-reproduction-20260721.md`.
  **Scope widened at T-20 to three named tests** — `:1198`, `:525 FLD-LIVE-02`, and two new
  observations from triage: `sync.spec.js` **LST-17** (cross-device list-progress decrement) and
  `workflows.spec.js` **GATE-04** (required-photo submit gate), both passed-on-retry, neither
  previously recorded. `cycle-gate`'s no-retry premise **STANDS** — it genuinely cannot pass at
  ~1-in-5. Note `:525`'s disposition is now decided by preference P1 (clean DB): it fails only
  against a carried-over journal, so a gate that resets may never see it · Also covers
  `:525 FLD-LIVE-02`, which fails 3/3 in isolation *and at the pre-gate baseline* — a pre-existing
  order-dependent test that must be fixed as order-dependence, not reordered around. Deliverable is
  a `--retries=0` streak proof on a quiet box, not merely a green run. Footprint: sync trust
  (`tests/sync.spec.js`; `sync.js`/`workflows.html` only if a determinism seam is unavoidable —
  which would re-arm the attended two-device check). → Eng KR5 (determinism), QA, unblocks
  `cycle-gate`. *(from BACKLOG "sync.spec.js de-flake: `:1198` + `:525`")*
- **`prod-alert-dup-guard`** · **DONE** (T-21b decision 46, 2026-07-24: last duplicate observed
  2026-07-21 with test-fixture names, root-caused to dev-side senders holding live creds; remedy
  = one side disabled — outbound delivery now opt-in via `ALERTS_ENABLED=1`, prod compose only,
  red-first tested; rides the next deploy. Delivery KR4 settles.) · With Mercury receipt worker / alert queue / Zoho
  Cliq now live in prod against the SAME external accounts as dev, guard against duplicate alerts:
  observe the Cliq channel over the cycle and either confirm 0 duplicates OR disable one side.
  0 duplicate-alert incidents left unhandled (recorded in the ledger). Footprint: prod ops /
  alert queue. → Delivery KR4. *(from ledger T-17 standing note "prod integrations now live")*

## Activity 4 — Feature build track · *serialized after Activity 2 sign-off*

- **`trends-spend-by-group-endpoint`** · **DONE** (F1 re-dispatch shipped 2026-07-20 against design §2.2 **as amended**, T-19 decisions 29–31; the 2026-07-22 PARK is closed — parked work on `parked/f1-trends-endpoint-20260720b` @ 88cab9d was re-derived, not resumed) · `GET /api/v1/inventory/trends`: spend
  bucketed by `date_trunc('week', event_date)` × `purchase_items.group_id` over the fixed
  12-week window, **COGS-allowlist filtered** (Amendment 1) and at **face value with no tax
  proration** (Amendment 3), with the signed NULL-`purchase_item_id`
  (unlinked line item) handling rule + the signed **D2 rule** (linked-but-groupless lines →
  explicit "Ungrouped" pseudo-group) + per-week `unlinked` array (rider (a), signed).
  Unreviewed receipts and the per-event unitemized remainder are surfaced in a `completeness`
  object (Amendments 2/3). The reconciliation identity
  `Σcells + Σunlinked + pending_total == period_summary.cogs_excl_tax` is asserted in
  `trends_test.go` by calling `PeriodSummaryHandler` itself on the same window — never against
  a constant — over a fixture carrying all five of G6's breakers.
  Red-first Go test: every week×group cell = SUM of matching
  line-item spend on a ≥8-week/≥2-group seeded fixture. Footprint: `backend/internal/inventory`.
  → Eng KR1, Delivery KR2.
- **`trends-tab-frontend`** · **DONE** (F3 shipped 2026-07-20, merged `be5ffb0`, triaged T-20; **G6 PASS on the first pass — the only card of the run needing no revision**. Reconciles on `completeness.reconciles_to_cogs_excl_tax`, the payroll-facing figure, NOT the sum of penny-rounded display cells, and discloses the drift when they diverge. Net-zero cells render as absent (`—`), never as a measured `$0.00`; `unitemized_remainder` labelled **(net)** with an open disclosure. Three defects were caught by reading the screenshots back, not from code — a price splitting mid-number at 390px, an axis choice flattening week-to-week variation, and a `$0.00` total on empty weeks) · Build the Trends tab (`#s5`, replacing the
  `renderTrends` stub at `inventory.html:993-995`): inline SVG/CSS weekly-spend-by-group chart +
  table, ~8–12 week window. Ships with `tests/states-trends.spec.js` (State-Enumeration: empty /
  loading / error / populated + no-data + ungated edges, screenshots read back). Footprint:
  `inventory.html`. → Delivery KR2, QA KR3.
- **`cost-margin-endpoint`** · **DONE** (2026-07-22, triaged T-19; residual gap resolved by decision 33, 0%-food-cost left as an open investigation) · The margin join: extend `menu-cogs` (or a new
  endpoint) to also select `SUM(gross_amount)` (revenue) and compute `margin = gross_amount −
  ingredient_cost_total` and `food_cost_% = ingredient_cost_total / gross_amount`, plus top/bottom
  movers ordering. Red-first Go test matching a hand-computed fixture to the cent. Footprint:
  `backend/internal/recipes` (+ `inventory`). → Eng KR2, Delivery KR2.
- **`cost-tab-frontend`** · **DONE** (2026-07-22, triaged T-19; red-negative left as an open investigation — blocked on discount data) · Build the Cost tab (`#s6`, replacing the cost render
  stub at `inventory.html:997-998`): sortable per-menu-item food-cost table (units / revenue /
  ingredient cost / margin / food-cost %) + a top/bottom movers highlight; inline SVG/CSS bars;
  honest empty/low-data state where Toast sales are absent (accept-sparse-prod). Ships with
  `tests/states-cost.spec.js`. Footprint: `inventory.html`. → Delivery KR2, QA KR3.
- **`inventory-tab-gating`** · **DONE** (F5 shipped 2026-07-20, merged `c1a2393`, triaged T-20. **Its G6 caught a live authentication bypass the card would otherwise have shipped:** `POST /workflow/ops` routed `APPROVE_ITEM`/`REJECT_ITEM` to the same mutations from the same cookie-auth group with no check, so a `team_member` with zero assignments forged an approval that broadcast over the sync hub as legitimate. Fixed by moving `requireReviewAuthz` INSIDE `approveSubmission`/`rejectItem` (`8c71022`) so all four call sites inherit it — closing the class, not the instance. Tab gate held across 13 attack variants incl. path normalization, disabled rows, orphaned/wrong-app grants, service-token crossover; fails closed on DB error. **NOTE — this card shipped the per-tab MECHANISM, not coverage: only 2 of the 11 Users-tab grants are enforced anywhere. See `grant-enforcement-parity`, decision 36**) (design signed 2026-07-20: **Option (i)** two per-tab
  slugs via `SeedHQApps` — NO migration, QA-KR4 down-migration clause is N/A; **umbrella
  semantics** — `RequirePermission` passes on (tab slug ∨ whole-app `inventory` slug ∨ superadmin);
  **+ B5 fold-in:** also gate `ApproveSubmissionHandler`/`RejectItemHandler`
  (`workflow/handler.go:728-753, 793+`), role rule specified at slate time) · Server-enforced
  tab-level gate via the Users `app_permissions` model (per the signed design): a session user
  WITHOUT the grant gets a distinct 403 from the Trends/Cost data endpoints AND the tabs do not
  render; a granted user gets 200 + tabs. Wires the grant through the Users admin UI + the
  `/me`-style resolver + a `RequirePermission`-style check. Red-first with-grant/without-grant test
  pair incl. the mixed Trends-only case; 0 logged-in-only bypass. Footprint: gating (`users.html`,
  `backend/internal/users`, `me`, `auth` middleware, `inventory.html` tab render,
  `backend/internal/workflow` handlers). → Eng KR3, Product KR3, QA KR4.

## Activity 5 — Cycle gate · *last, serialized*

- **`grant-enforcement-parity`** · **DONE** (G1 shipped overnight-20260724, merged to run branch
  `4bb8649`, G6 verdict APPROVE as-is; all five live app surfaces now behind `RequirePermission` —
  parity spec covers 11/11 slugs (7 enforced, 4 N/A-with-reason). **Both riders RESOLVED at T-21
  (2026-07-23):** `/photos/*` stays authenticated-only as the documented exception, durable
  key-binding fix backlogged — decision 42; the `GET /inventory/items` (inventory ∨ purchasing)
  READ is RATIFIED — decision 43.) (was: PLANNED — URGENT, BLOCKS
  GO-LIVE; new card, T-20 decision 36,
  2026-07-21) · **The Users tab offers 11 grants; the backend enforces 2.** Every `RequirePermission`
  call in the server is `inventory-trends` and `inventory-cost` (both shipped by F5). The other nine
  — `inventory` itself, `operations`, `purchasing`, `onboarding`, `users`, plus the four placeholder
  slugs — are checked nowhere. `isAdmin`/`manager` **role** checks protect some endpoints, but roles
  are a different axis from grants: revoking a grant today removes launcher tiles and hides tabs,
  and changes nothing about what the holder can read from the API with a cookie. Concretely, an
  ungranted logged-in user reads `/inventory/purchases` (what was paid), `/items`, `/stock`,
  `/recipes`, `/menu-items` at 200.
  **Operator ruling (verbatim):** *"If an employee does not have access to the app (or access to the
  app's tab), then they should NOT be able to access the view / tab / data."* The grant model is a
  DATA boundary, not a UI convenience.
  **Not an F5 defect** — F5 built the mechanism the signed design scoped it to, and built it well
  (umbrella semantics, fail-closed on DB error, 13 attack variants held). Nothing migrated the
  pre-existing surface onto it.
  **No live exposure today** (no non-admin crew hold accounts) — **but this must be fixed before
  go-live**, which makes it the largest open correctness item on the roadmap.
  **First task is to prove it live:** the evidence is source enumeration of every `RequirePermission`
  call, NOT a grant-less curl. Start by reproducing with a real ungranted account, then migrate the
  surface onto `RequirePermission` app-by-app. Expect the `EXCEPTIONS`-style parity guard from
  `tests/ops-authz-coverage.spec.js` to be the right shape for a standing regression test — one that
  derives the app list from `SeedHQApps` and asserts every slug is enforced somewhere.
  → Eng KR1, QA KR3.

- **`cycle-gate`** · **PLANNED** (computable legs RAN overnight-20260724 — scorecard
  `runs/2026-07-24-autonomous/scorecard-20260724.md`: 11 MET · 2 PARTIAL (P4, E5) · 2 PENDING
  (D2, D4) · 1 N/A (Q4); **no-retry attestation GRANTED at T-21 with the LST-17 waiver, decision
  44** — E5's evidence is settled; remaining attended close-out: P4 interpretation, D2 prod ship,
  D4 Cliq-dup confirmation, E2 0%-food-cost note) · Milestone boundary for "Prove & surface". Per-KR scorecard;
  suite-green attestation on the isolated deterministic stack (with the no-retry hard gate,
  now proven-eligible); computed **median WO cycle time vs the T-14 baseline** (N=23 / 22m28s —
  Delivery KR3); prod-parity ship of both tabs (`task version` prod == local, 2/2 tabs
  screenshot-verified behind the gate on `https://hq.yumyums.kitchen` — Delivery KR2); closeout
  doc. → all teams' summary KRs.

---

## Backlog routing record (Product KR4 — 12/12 `· new` items routed 2026-07-19)

> Authored at the OKR session as the intended routing; **ratified/adjusted at the evening
> `/nc-pm-session` + grill-back** when the PRD lands (a routing door per item — folded / promoted /
> deferred-with-reason). Recorded here so the roadmap card names resolve; the PRD §Routing is the
> authoritative record once signed.

| Backlog item (`· new`) | Intended door | Destination |
|---|---|---|
| Cross-user live-sync access matrix + `sync` unit coverage | promoted | `sync-pkg-unit-coverage` (Activity 3) |
| Live approval-state convergence coverage | promoted | `convergence-matrix-systematic` (Activity 3) |
| `suite-isolation-approved-checklist` (retire waiver #1) | promoted | `waiver1-isolation-fix` (Activity 3) |
| Per-card wall-clock instrumentation | promoted | `percard-timing-instrumentation` (Activity 3) |
| Gate run-mechanics: `CI=1` + explicit pre-migration | folded | rides `percard-timing-instrumentation` / cycle-gate run-mechanics |
| Transactional op emission for Create/Archive (INV-1 parity) | deferred | small editprop tidy-up; PM recommendation kept as BACKLOG, not a KR — revisit if it rides a sync card |
| Fail-note conflict live-render on `applyOp`/409 path | deferred | out-of-footprint (needs `_fail_note` unpack on apply path); BACKLOG tidy-up, revisit next cycle |
| Atomic approval + feedback (`approveSubmission` tx) | deferred | small editprop tidy-up; BACKLOG, not a KR this cycle |
| Onboarding persistence tests: `waitForResponse` over fixed wait | deferred | low-priority test-hardening; BACKLOG |
| Runner — failed photo upload leaves a partial saved value | deferred | stale-state hygiene, off-theme; BACKLOG |
| Offline submit idempotency under IndexedDB failure (suspected) | deferred | needs the offline harness (not built this cycle); BACKLOG |
| Lamport clock corruption → catch-up gap (suspected) | deferred | same offline-harness dependency; BACKLOG |
