# Roadmap — "Nothing silently lost" cycle (checklist data integrity)

> **Cycle:** Nothing silently lost — every way the checklist engine can silently lose
> crew-entered work is enumerated, fixed, and made structurally impossible.
> **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA,
> signed 2026-07-16). **Produced:** 2026-07-16 attended `/nc-okr-session` (the roadmap's
> guaranteed producer, DESIGN §15u). Previous cycle's roadmap archived at
> `reference/roadmap-2026-07-09-hq-hardening.md`.
> **Trigger:** operator-reported, same-day-reproduced P0 — template edits silently discard
> open-device crew work (repro: `tests/repro-cut-task.spec.js`, untracked until stage 1
> flips it green; root cause: `replaceTemplate` field-ID churn + FK-dropped silent dead-id
> writes; backlog entry has full cites).

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan` sizes
  to a night. Cards carry a **module footprint** (for parallel tracks) and a **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** Cards-per-night is the planner's call
  against the night budget + quality bar (budget is a floor, not a ceiling).
- **Sequencing rule (operator-ratified 2026-07-16):** stages 1–2 ship BEFORE the
  versioning build — interim protection deploys to prod without waiting on a schema
  migration. The `replaceTemplate` double-touch (stage-1 upsert now, versioning rework
  later) is accepted cost.
- **Red-first is mandatory on every fix card** (QA KR2): the test fails before the fix,
  recorded in the WO record.

## Module footprints (independent → parallelizable)

| Track | Frontend | Backend | Tests |
|---|---|---|---|
| Workflow engine | `workflows.html`, `sync.js` | `backend/internal/workflow` | `workflows.spec.js`, `sync.spec.js`, `persistence.spec.js`, `repro-cut-task.spec.js` |
| Users | `users.html` | `backend/internal/users` | `users.spec.js` |
| Inventory (prod ops) | — | `backend/internal/inventory` | `inventory.spec.js` |
| Test-debt | — | — | all spec files (audit-scoped) |

---

## Activity 1 — PRD gate · *blocking, first*

> The stages 1–3 data-integrity PRD. No build WO dispatches before this lands.
> Produced by the evening `/nc-pm-session` + `/nc-pm-grill-back`.

- **`prd-data-integrity`** · PLANNED · The PRD enumerating every silent-loss mode with a
  requirement→(reproduced failure | named invariant) trace table; the operator-signed
  mid-run edit semantic ("crews finish the run they started; edits take effect next run");
  the 8-item backlog routing record. → Product KR1, KR2, KR3. *(attended evening session)*

## Activity 2 — Stop the bleeding (stage 1 + carried small fixes) · *after Activity 1; parallel tracks*

- **`stage1-field-id-preservation`** · PLANNED · `replaceTemplate` upserts by the field IDs
  the Builder already sends (diff: update kept / insert new / delete removed; conditions
  remap for new fields only) instead of delete+reinsert. Flips `repro-cut-task.spec.js`
  red→green and commits it to the suite. Footprint: workflow engine. → Eng KR "Stage 1a",
  Delivery KR "Stage 1 ships first".
- **`stage1-dead-id-reject`** · PLANNED · Draft saves (`submission_id IS NULL`) naming an
  unknown field ID rejected with a distinct error envelope (app-level existence check —
  NOT a restored FK; submitted responses reference snapshot IDs by design). Red-first Go
  test. Footprint: workflow engine (backend only). → Eng KR "Stage 1b".
- **`ops-nfr3-resubmit-photo-gate`** · PLANNED · Carried fix-card: plumb rejection context
  into submit validation so a rejected-with-`require_photo` field blocks direct-API
  resubmit server-side; red-first. Footprint: workflow engine (backend). → QA KR2
  (red-first denominator), carried from hardening cycle (ledger T-10).
- **`users-s3-orphan-cleanup`** · PLANNED · Trivial carried card: remove dead
  `<div id="s3">` at `users.html:122`. Footprint: Users (zero contention — free
  parallelism). → hygiene; no KR.

## Activity 3 — Stage 2: template-updated broadcast · *after Activity 2*

- **`stage2-template-updated-broadcast`** · PLANNED · Handle `SAVE_TEMPLATE` ops in
  `applyOp` (they already flow through live WS + `wsCatchUp` — clients just ignore them):
  re-fetch template, re-render the open checklist, preserve in-progress input, and stay
  silent on catch-up replay (the `42eeb39` no-toast rule). Red-first E2E for the mixed
  old/new-device case. Footprint: workflow engine (`sync.js` + `workflows.html`).
  → Delivery KR "Stage 2 ships".

## Activity 4 — Versioning design gate · *attended; blocks Activity 5*

- **`versioning-openspec-design`** · PLANNED · The OpenSpec change for immutable
  run-pinned template versions: stable field UUIDs honored forever; edits create versions
  with "the template" a head pointer; runs pin the version current at run start; responses
  key on (run, field-UUID). Extends submit-time `template_snapshot` upstream to edit-time.
  **Operator sign-off on the design is the gate — 0 build WOs dispatch before it**
  (auditable from ledger timestamps). → Delivery KR "versioning design signed before build".

## Activity 5 — Versioning build · *serialized after Activity 4 sign-off*

- **`versioning-schema-migration`** · PLANNED · `template_versions` schema + head pointer +
  run pinning columns; all existing templates/drafts migrate intact; down-migration proven
  by an up→down→up cycle recorded in the WO. Footprint: workflow engine (migrations).
  → Eng KR "Stage 3 built", QA KR "down-migration + backup".
- **`versioning-backend-runtime`** · PLANNED · Runs pin their version; responses key on
  (run, field-UUID); `replaceTemplate` becomes create-new-version. Footprint: workflow
  engine (backend). → Eng KR "Stage 3 built".
- **`versioning-runner-frontend`** · PLANNED · Runner loads the run's pinned version;
  Builder edits never mutate an in-flight run's shape. Footprint: workflow engine
  (frontend). → Eng KR "Stage 3 built".
- **`versioning-e2e-semantics`** · PLANNED · The ≥ 2 acceptance tests encoding the signed
  semantic: mid-run edit leaves the in-flight run untouched; the next run reflects the new
  shape. Extends `repro-cut-task.spec.js`. → Product KR "mid-run edit semantic".

## Activity 6 — Test-debt retirement · *independent parallel track (any time)*

- **`vacuous-tests-18-to-0`** · PLANNED · Each remaining conditional `test.skip()` / silent
  guard-return becomes a real seeded assertion or is deleted (denominator = the audit that
  produced the 18). Footprint: test-debt (audit-scoped). → QA KR1, retires carried
  waiver #2.
- **`carried-fix-wos-sweep`** · PLANNED · The carried prove-sweep fix-WO with no harness
  dependency: `WO-cron-clock-seam` (a `now` seam in the 4 `run*Check` funcs + real
  cron-decision unit tests — unblocks P-6, Purchasing FR-19/20/21/22). The other two
  prove-sweep PARKs (photo-S3 harness, offline-IndexedDB harness) are deferred this cycle
  — harness-infrastructure class, see routing record. Red-first. → QA KR2.

## Activity 7 — Prod ops · *operator-gated*

- **`prod-ghost-item-rename`** · PLANNED · Operator-chosen handling (2026-07-16): rename
  `''` → `(Unnamed — needs review)`, KEEP line-item links. Verify: empty-description count
  = 0 in prod AND previously-linked `purchase_line_items` count unchanged. **Prod data
  mutation — runs attended or with explicit operator go.** Footprint: Inventory (prod DB).
  → QA KR3.
- **`prod-deploy-parity`** · PLANNED · Operator runs `task prod:deploy` (never automated);
  card verifies `task version` shows prod == local `version.go` constants (includes
  `42eeb39`). → Delivery KR "prod parity".

## Activity 8 — Cycle gate · *last, serialized*

- **`cycle-gate`** · PLANNED · Suite-green attestation on the deterministic stack
  (`task test` exit 0 — formally retires carried waiver #1); median WO cycle time vs the
  recorded baseline (first cycle with a pass/fail target); per-KR scorecard; closeout doc.
  → Eng KR "task test exits 0", Delivery KR "median WO cycle".

---

## Backlog routing record (Product KR3 — 15/15 `new` items routed 2026-07-16)

| Backlog item | Door | Destination |
|---|---|---|
| Ops P0 template-edit data loss (stage 1) | promoted | `stage1-field-id-preservation` + `stage1-dead-id-reject` |
| Ops stage 2 broadcast | promoted | `stage2-template-updated-broadcast` |
| Ops stage 3 immutable versions | promoted | Activities 4–5 (`versioning-*`) |
| Inventory prod ghost item | promoted | `prod-ghost-item-rename` |
| Ops NFR-3 resubmit photo gate | promoted | `ops-nfr3-resubmit-photo-gate` |
| Users `#s3` orphan cleanup | promoted | `users-s3-orphan-cleanup` |
| Onboarding video-pipeline fixture | deferred | off-theme for this cycle; needs operator Spaces creds; revisit next cycle (operator 2026-07-16) |
| `/nc-status` non-determinism | deferred | framework/tooling outside this repo; stays in backlog (operator 2026-07-16) |
| Ops FR-15 builder-UI coverage gap | promoted | rides `vacuous-tests-18-to-0` |
| Ops FR-10/FR-12 vacuous reject test | promoted | rides `vacuous-tests-18-to-0` |
| Onboarding 6 conditional-skip guards | promoted | rides `vacuous-tests-18-to-0` |
| Inventory ~40 data-dependent guards | promoted | rides `vacuous-tests-18-to-0` |
| WO-cron-clock-seam | promoted | `carried-fix-wos-sweep` (pure code seam, no harness dependency) |
| WO-photo-s3-harness | deferred | needs mock-S3/test-bucket harness + creds — same class as the video fixture; revisit next cycle |
| WO-offline-indexeddb-harness | deferred | needs a new SW+IndexedDB Playwright project; off-theme this cycle |
