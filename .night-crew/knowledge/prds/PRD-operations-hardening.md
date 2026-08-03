# PRD — Operations app hardening (exemplar)

> **Cycle:** HQ hardening — first night-crew guinea-pig run.
> **App:** Operations (aka Workflows) — `workflows.html`, `/api/v1/workflow/*`, Go + Postgres.
> **Depth:** *Enumerate + mark only.* This PRD is the honest flow map with per-flow
> status and a falsifiable definition of "working." It does **not** fix anything —
> each BROKEN/UNPROVEN flow becomes a work order after sign-off.
> **Role of this doc:** the **exemplar** the other four app PRDs copy.
> **Enumeration provenance (for the ≥90% recall KR):** flows were derived by reading
> `workflows.html` (3 tabs, ~1500 lines), `tests/workflows.spec.js` +
> `tests/persistence.spec.js` (28 + persistence tests), and the Go router under
> `backend/` (12 `/api/v1/workflow/*` endpoints enumerated in §Verification), then
> **cross-checked by a second independent pass (G5)** angled at backend-only logic,
> notifications, and audit views. Recall is measured post-build as
> `enumerated ÷ (enumerated + discovered-during-WO-build)`.
> **Scope note (resolves G2):** the OKR's phrase "critical flows" is interpreted here
> as **all** real user-facing flows — no sub-selection. A flow is in-scope if a real
> person (crew / manager / admin) triggers it. Dropping the "critical" filter removes
> an undefined term that made the recall metric gameable; completeness of the whole
> set + the G5 cross-check is what keeps recall honest.

## Objective

Produce a written, agreed enumeration of every end-to-end flow in the Operations
app — from a crew member opening a scheduled checklist to a manager approving or
rejecting the finished submission — with each flow honestly marked **WORKING**,
**UNPROVEN**, or **BROKEN**, and each backed by a falsifiable definition of what
"working" means. This is the first of the cycle's five app hardening PRDs and the
pattern the other four follow. It advances the **Product objective**: KR-1 (5/5
apps have a hardening PRD enumerating their E2E flows, delivered as an early
blocking gate) and KR-2 (enumeration recall ≥ 90%). Its status tally is the
denominator the **Engineering** (0 known-broken flows at cycle end) and **QA**
(every flow has a real, asserting test) objectives grade against.

## Operators & users

- **Crew member (team_member)** — opens today's scheduled checklists on a phone,
  fills fields (checkbox / yes-no / text / temperature / photo / sub-steps),
  records corrective action on a fail, submits. Primary user of Tab 1.
- **Manager** — reviews submitted checklists in Tab 2, approves or rejects
  individual items with required comments; accountability (who checked what).
- **Admin / owner** — authors and edits checklist templates in Tab 3 (field types,
  day-of-week scheduling, section visibility, skip logic); admin-only CRUD.
- **The overnight crew (indirect)** — consumes this enumeration + status as the
  work-order backlog; a flow marked BROKEN/UNPROVEN here is a candidate WO.

## Requirements

Status legend: **WORKING** = flow works E2E *and* a test drives it and asserts
observable DB/UI state · **UNPROVEN** = flow appears to work in code but no test
verifies the behavior it names (missing or vacuous test) · **BROKEN** = flow is
**confirmed** incomplete/stubbed by code inspection, or a test reveals it does not
do what it claims. Every requirement traces to an OKR key result.

**G1 resolution:** BROKEN is reserved for *confirmed* breakage only. A flow I
merely *suspect* is broken but have not confirmed absent stays **UNPROVEN**. Three
flows (FR-12, FR-18, NFR-3) are therefore UNPROVEN, not BROKEN — but they are the
**highest-priority** UNPROVEN: each carries a *confirm-absence* step, and graduates
to BROKEN (a code-fix WO) if that step confirms the behavior is missing/stubbed.

### Tab 1 — My Checklists

- **FR-1** — The system lists a crew member's checklists for today, filtered by
  their role and the template's day-of-week schedule, merged with any in-progress
  draft and prior submission. *(GET `/myChecklists?dow=`)* — **WORKING** — traces
  to Product KR-1, QA KR-2.
- **FR-2** — The system persists every field entry across all seven field types
  (checkbox, yes/no, text, temperature, sub-steps, fail-note text, fail severity)
  via the `autoSaveField → DRAFT_RESPONSES → hydrateFieldState` path, surviving
  back-to-list and reopen. *(POST `/saveResponse`)* — **WORKING** (14 persistence
  tests) — traces to Product KR-1, QA KR-2/KR-3.
  > 🛑 **Correction appended 2026-08-04 (B-65 / card A2, run `20260804`) — the requirement is
  > unchanged; its NAMES were wrong.** The path is
  > `debouncedSaveField → submitOp('SET_FIELD') → POST /ops → DRAFT_RESPONSES → hydrateFieldState`.
  > `autoSaveField` is defined nowhere in the tree and no frontend code posts to `/saveResponse`
  > (the endpoint exists on the backend; the op journal is the single write channel, D-08). This
  > PRD is the **exemplar the other four app PRDs copy** — the correction is here so the phantom is
  > not copied forward. The signed requirement text above is left intact as the record.
- **FR-3** — A temperature reading outside the template's min/max triggers a
  corrective-action card requiring a fail note (text + severity, optional photo)
  before submit. *(POST `/saveResponse`, `/submitChecklist` validation)* —
  **WORKING** (temperature path tested) — traces to Engineering KR-1, QA KR-2.
- **FR-4** — A "No" answer on a yes/no field triggers the same corrective-action
  card as FR-3. — **BROKEN** *(confirm-absence sweep 2026-07-11, G6-confirmed:
  enforcement absent — `evaluateFailTrigger` handles only `out_of_range`
  (`workflows.html:1656-1668`); yes/no fields are created with `fail_trigger:null`
  and the builder sets no trigger for them (`workflows.html:558,724`); submit
  validation short-circuits on `!f.fail_trigger` (`workflows.html:2398-2405`) and
  server `validateFailNotes` only checks `out_of_range` (`handler.go:80,101`). The
  "No" fail card renders (`workflows.html:2068`) but nothing blocks submit — a
  cosmetic card, not enforcement.)* — traces to QA KR-1, QA KR-2. **→ Activity-4
  fix-card (front+back enforcement + red-first AC-3 test).**
- **FR-5** — Sub-steps (nested checkboxes) persist per-sub-step attribution and
  auto-check their parent when all children are checked, surviving reopen. —
  **WORKING** (tested) — traces to Product KR-1, QA KR-2.
- **FR-6** — Submitting a checklist writes an idempotent submission record (UUID
  idempotency key, D-15) from the current draft; a duplicate submit does not
  create a second record. *(POST `/submitChecklist`)* — **UNPROVEN** (mechanism
  present; no test drives the duplicate-submit path to assert single record) —
  traces to Engineering KR-1, QA KR-1.
- **FR-7** — A permitted user can unsubmit a submission back to draft; a
  non-submitter is refused. *(POST `/unsubmitChecklist`)* — **UNPROVEN** (the
  authorization refusal path is untested) — traces to QA KR-1, Engineering KR-1.
- **FR-8** — The system shows a crew member their last 50 submissions in History.
  *(GET `/myHistory`)* — **UNPROVEN** (no asserting test) — traces to QA KR-2.

### Tab 2 — Approvals

- **FR-9** — The system lists submissions pending approval, filtered to
  managers/admins by role. *(GET `/pendingApprovals`)* — **WORKING** — traces to
  Product KR-1, QA KR-2.
- **FR-10** — A manager can flag an individual item for rejection, with an optional
  comment and (where the template requires) a photo. — **UNPROVEN** (flag test
  exists but does not verify a resulting rejection-status change) — traces to
  QA KR-1.
- **FR-11** — A manager can approve a submission, optionally attaching feedback
  comments. *(POST `/approveSubmission`)* — **WORKING** — traces to Product KR-1.
- **FR-12** — A manager can reject a submission carrying one or more flagged items,
  each flagged item requiring a comment; the submission returns to the crew member
  as corrective work. *(POST `/rejectItem`)* — **UNPROVEN** *(confirm-absence sweep
  2026-07-11, G6-confirmed NEGATIVE: the reject handler DOES flip
  `status='rejected'` + set reviewer in a tx (`repository.go:887-915`, esp.
  902-909) — behavior present, so no BROKEN graduation; drops from priority to a
  plain test-only WO.)* — traces to Engineering KR-1, QA KR-1.
- **FR-13** — A rejected-and-returned checklist shows the crew member the manager's
  rejection feedback on the corrected checklist. — **UNPROVEN** (feedback-retrieval
  assertion incomplete) — traces to QA KR-2.

### Tab 3 — Builder (admin only)

- **FR-14** — An admin can create, edit (full replace, D-09), and archive
  (soft-delete, D-07) checklist templates; non-admins are refused.
  *(POST `/createTemplate`, PUT `/updateTemplate/{id}`, DELETE
  `/archiveTemplate/{id}`)* — **WORKING** (template CRUD tested) — traces to
  Product KR-1, QA KR-2.
- **FR-15** — An admin can add each of the six builder field types (checkbox,
  yes/no, text, temperature, photo, sub-steps) to a template. — **WORKING**
  (builder field tests) — traces to Product KR-1.
- **FR-16** — A template's day-of-week schedule governs which days a checklist
  appears in a crew member's list (feeds FR-1). — **UNPROVEN** (scheduling logic
  has no asserting test) — traces to QA KR-1.
- **FR-17** — A template section's visibility condition shows/hides the section
  based on a preceding field's value. — **UNPROVEN** (no asserting test) — traces
  to QA KR-1.
- **FR-18** — Skip logic hides/shows a field based on the value of a preceding
  field. — **UNPROVEN** (confirm-absence step *resolved by the G5 cross-check*:
  skip logic **is** implemented — `isFieldVisible` at `workflows.html:1907-1929`,
  operators equals/not_equals/_notempty — so present-but-untested, not stubbed; a
  test-only WO) — traces to Engineering KR-1, QA KR-1.

### Cross-cutting (non-functional / platform guarantees)

- **NFR-1 (Persistence contract)** — Every user-entered value follows
  `autoSaveField → POST /saveResponse → DRAFT_RESPONSES → hydrateFieldState`; no
  field type bypasses it. Fail-note metadata (`{note, severity, photo}`) is bundled
  and restored. — **WORKING** for the seven canonical states — traces to Product
  KR-1, QA KR-2. *(Governs FR-2, FR-3, FR-4, FR-5.)*
  > 🛑 **Correction appended 2026-08-04 (B-65 / card A2, run `20260804`).** Two corrections, one
  > naming and one substantive. **(a)** The path is
  > `debouncedSaveField → submitOp('SET_FIELD') → POST /ops → DRAFT_RESPONSES → hydrateFieldState`;
  > `autoSaveField` does not exist and no frontend code posts to `/saveResponse`. **(b)** The
  > "**WORKING** for the seven canonical states" verdict was correct as scoped and *incomplete as
  > read*: `{note, severity, photo}` is named here as bundled and restored, but the **photo** leg
  > was NOT working — the fail-photo upload chain called the phantom and its own `.catch()`
  > swallowed the `ReferenceError`, so the thumbnail rendered and nothing persisted. Fixed by card
  > A2 and pinned by `tests/persistence.spec.js` `[FLD-16B]`. There are **nine** persisted states
  > today, not seven (add fail photo and correction photo). The requirement itself is unchanged.
- **NFR-2 (Photo pipeline)** — Photo fields presign against
  `/api/v1/photos/presign` and PUT to S3; the stored value is the public URL, and
  it round-trips on reopen. — **UNPROVEN** (upload + error-path untested; photo
  persistence not among the seven verified states) — traces to QA KR-1.
- **NFR-3 (Photo-required enforcement)** — Where a template requires a photo for
  corrective action / rejection, submit/resubmit is blocked until a photo is
  attached. — **BROKEN** *(confirm-absence sweep 2026-07-11, G6-confirmed:
  enforcement absent — frontend submit validation checks only note+severity, never
  a photo (`workflows.html:2397-2419`); the `fld-photo-required` toggle + reject
  `require_photo` only feed a render-time banner (`workflows.html:2024-2025`);
  backend `validateFailNotes` checks only note+severity (`handler.go:54-88`, esp.
  71); the submit handler runs exactly one validation with no photo gate
  (`handler.go:458`) and there is no separate resubmit endpoint. `PhotoURL` is
  storable (`model.go:92`) but never required.)* — traces to Engineering KR-1, QA
  KR-1. **→ Activity-4 fix-card (block submit/resubmit until required photo
  attached, front+back, + red-first test).**
- **NFR-4 (Authorization tiers)** — admin / manager / team_member tiers gate
  template CRUD (admin), approvals (manager+), and unsubmit (submitter) —
  D-11/D-23. — **WORKING** for list-filtering; **UNPROVEN** for the unsubmit
  refusal (see FR-7) — traces to Engineering KR-1, QA KR-1.
- **NFR-5 (Sync / offline / conflict)** — The sync framework queues writes offline
  (D-22), surfaces a save-status indicator (pending/saved/error), resolves
  concurrent-edit conflicts, and cleans up drafts after submit. — **UNPROVEN**
  (queueing, conflict resolution, save-status, and draft cleanup are all untested)
  — traces to QA KR-1, Engineering KR-1.

### Additional flows — G5 second-pass cross-check

A second, independent enumeration (angled at blind spots the first pass's three
files could not see) found **4 flows the first pass missed** — the empirical basis
for the recall note in §Success metrics. Each is UNPROVEN:

- **FR-19** — On submit, the checklist **freezes a snapshot of the template**
  (`checklist_submissions.template_snapshot` JSONB) so a later admin edit or
  archive does not alter an already-submitted checklist. *(POST `/submitChecklist`;
  `backend/internal/workflow/repository.go:445-476`)* — **UNPROVEN** — traces to
  Engineering KR-1, QA KR-1.
- **FR-20** — Saving a template with "requires approval" ON is refused unless at
  least one approver role is assigned (`hasApprover`). *(POST `/createTemplate`,
  PUT `/updateTemplate/{id}`; `handler.go:128-135, 195-197, 252-254`)* —
  **UNPROVEN** — traces to Engineering KR-1, QA KR-1.
- **NFR-6** — Submitting a checklist whose template was **archived while the user
  was offline** returns 409 and shows "archived while you were offline; not
  submitted" instead of failing silently. *(POST `/submitChecklist` → 409
  `template_archived`; `handler.go:443-487`, `workflows.html:2468-2477`)* —
  **UNPROVEN** — traces to Engineering KR-1, QA KR-1.
- **NFR-7** — A 401 on any API call mid-checklist redirects to `/login.html`;
  local drafts persist. *(`workflows.html:2791`)* — **UNPROVEN** — traces to QA
  KR-1.

## Acceptance criteria

Surface-anchored, Given/When/Then. These define "working" for representative
flows; every enumerated flow inherits the pattern of *drive-the-real-flow +
assert-observable-state* (the WORKING bar).

- **AC-1 (FR-2, persistence):** *Given* a checklist with one field of a given type,
  *When* the crew member enters a value, waits for auto-save (1500ms), taps back to
  the list, and reopens the checklist, *Then* the list shows correct progress and
  the entered value is still present, and no other field is affected. *(Per the
  CLAUDE.md required back-and-reopen test, one per field type.)*
- **AC-2 (FR-3, temperature fail):** *Given* a temperature field with min/max,
  *When* the crew member enters an out-of-range reading, *Then* a corrective-action
  card appears and `/submitChecklist` is rejected until a fail note (text +
  severity) is recorded.
- **AC-3 (FR-4, yes/no fail — currently UNPROVEN):** *Given* a yes/no field that
  fails on "No", *When* the crew member answers "No", *Then* the same
  corrective-action card as AC-2 appears and blocks submit until resolved. *(A
  red-first test must show this failing when the trigger is broken.)*
- **AC-4 (FR-6, idempotent submit — currently UNPROVEN):** *Given* a completed
  draft, *When* submit is invoked twice with the same idempotency key, *Then*
  exactly one submission record exists in the DB.
- **AC-5 (FR-12, reject E2E — UNPROVEN priority):** *Given* a submitted checklist
  with one item flagged and commented, *When* the manager rejects it, *Then* the
  submission's status changes to rejected in the DB and it reappears for the crew
  member with the manager's comment visible.
- **AC-6 (FR-18, skip logic — UNPROVEN priority):** *Given* a template where field B
  is shown only when field A = X, *When* the crew member sets A ≠ X, *Then* field B
  is not rendered and not required for submit; *When* A = X, *Then* B renders.
- **AC-7 (FR-14, admin gate):** *Given* a non-admin session, *When* it calls
  `POST /createTemplate`, *Then* the request is refused (403) and no template row
  is created.

## Verification plan

- **Environment:** localhost Postgres (`brew postgresql@16`) — the E2E suite
  requires a local DB; the remote Windows DB is too slow for the suite (N+1 ×
  50ms RTT). Playwright blocks service workers (`serviceWorkers: 'block'`).
- **Suites:** `tests/workflows.spec.js` (28 tests) + `tests/persistence.spec.js`.
  Run per-flow during iteration (`npx playwright test <file> -g "<name>"`), full
  suite (`task test`) at gate.
- **This PRD specifies the test each flow needs; it does not write them (resolves
  G4).** Writing/repairing a test is itself a work order — this doc names the
  assertion, the WO delivers it. No test-authoring is in scope here.
- **What each status turns into downstream:**
  - **WORKING** flows: a **test-audit WO** — spot-check the existing test is
    non-vacuous (no `test.skip`, no early `return` guard, an assertion that would
    fail if the feature broke). If vacuous, it drops to UNPROVEN.
  - **UNPROVEN** flows (all non-WORKING): a **test-only WO first (resolves G3)** —
    write a real seeded, red-first assertion (failing when the flow is broken,
    passing after) per the bug-fix protocol. The flow graduates to a **fix WO only
    if that test goes red** — i.e. we do not pre-judge an untested flow as needing
    code changes.
  - **UNPROVEN (priority)** flows (FR-12, FR-18, NFR-3): the test-only WO opens with
    a **confirm-absence step** (grep/inspect the handler or render path). If the
    behavior is confirmed missing/stubbed, the flow is re-marked **BROKEN** and the
    WO becomes a code-fix + regression-test.
- **Endpoints in scope (12 + presign):** GET `/templates`, POST `/createTemplate`,
  PUT `/updateTemplate/{id}`, DELETE `/archiveTemplate/{id}`, GET
  `/myChecklists?dow=`, GET `/myHistory`, POST `/saveResponse`, POST
  `/submitChecklist`, GET `/pendingApprovals`, POST `/approveSubmission`, POST
  `/rejectItem`, POST `/unsubmitChecklist`, POST `/api/v1/photos/presign`.

### Status tally (the denominator downstream objectives grade against)

Total requirements enumerated: **27** (20 FR + 7 NFR) — 23 first-pass + 4 from the
G5 cross-check.

**Updated by the Activity-2 confirm-absence sweep (2026-07-11, G6-passed).** The two
priority-UNPROVEN flows graduated to **BROKEN** (FR-4, NFR-3 — enforcement confirmed
absent at cited lines); FR-12's confirm-absence step came back **NEGATIVE** (reject
handler complete) so it stays UNPROVEN. Net: WORKING 10 · UNPROVEN 17 → 15 · BROKEN
0 → 2.

| Status | Count | Flows |
|---|---|---|
| **WORKING** | 10 | FR-1, FR-2, FR-3, FR-5, FR-9, FR-11, FR-14, FR-15, NFR-1, NFR-4 |
| **UNPROVEN** | 15 | FR-6, FR-7, FR-8, FR-10, FR-12, FR-13, FR-16, FR-17, FR-18, FR-19, FR-20, NFR-2, NFR-5, NFR-6, NFR-7 |
| **BROKEN** | 2 | **FR-4** (yes/no "No" corrective-action enforcement absent), **NFR-3** (photo-required-at-submit enforcement absent) — each → an Activity-4 fix-card |

*(15 UNPROVEN flows = the candidate test-only work-order backlog; the 2 BROKEN
(FR-4, NFR-3) each = an Activity-4 code-fix + regression-test card. Every one must
have a shipped WO by cycle end — Delivery KR-1 — and reach 0 known-broken —
Engineering KR-1. NFR-4 is WORKING for role-filtering but carries the FR-7
unsubmit-refusal gap, already counted under UNPROVEN. The BROKEN count of 2 enters
the Engineering-KR "0 known-broken" denominator.)*

### Activity-2 confirm-absence sweep record (2026-07-11)

Two-pass static audit (pass 1 UI-flow, pass 2 backend-only cross-check) of all 17
UNPROVEN flows against `workflows.html` + `backend/internal/workflow/*`; adversarial
G6 re-check of every citation at the cited line. **Pass 2 was decisive:** read from
the UI alone, FR-4's visible "No" fail card and NFR-3's "Photo required" banner look
like working enforcement — only the submit-validation / backend cross-check proved
the enforcement absent. Confirm-notes for the 15 flows kept UNPROVEN (all
present-but-untested, none stubbed):

| Flow | Present at | Confirm-note |
|---|---|---|
| FR-6 | `repository.go:469-475` | idempotency upsert `ON CONFLICT (idempotency_key) DO UPDATE` present |
| FR-7 | `repository.go:935-937` | unsubmit refuses non-submitter → 403 `not_submitter` present |
| FR-8 | `repository.go:698-707`; `workflows.html:2220-2232` | history LIMIT 50 + fetch/render present |
| FR-10 | `repository.go:894-909`; `workflows.html:2583-2589` | flag→reject wired with require_photo/comment |
| FR-12 | `repository.go:902-909` | reject flips `status='rejected'` in a tx (confirm-absence NEGATIVE) |
| FR-13 | `workflows.html:1404,1951-1956` | `FEEDBACK_NOTES` + `renderFeedbackNote` render manager comment |
| FR-16 | `repository.go:620`; `workflows.html:1909-1910` | DOW scheduling `= ANY(cs.active_days)` present |
| FR-17 | `workflows.html:1902-1904,1934-1935` | `isSectionVisible` present (day-based condition) |
| FR-18 | `workflows.html:1907-1929` | skip logic `isFieldVisible` (equals/not_equals/_notempty) present |
| FR-19 | `repository.go:455-459,470-475` | `template_snapshot` freeze on submit present |
| FR-20 | `handler.go:128-135,195-197,252-254` | `hasApprover` gate → 400 `requires_approver` present |
| NFR-2 | `workflows.html:1516-1573` | presign → PUT → public_url round-trip wired |
| NFR-5 | `handler.go:337-341`; `workflows.html:2468-2472` | draft cleanup + offline queue + save-status present |
| NFR-6 | `handler.go:452,465-467`; `workflows.html:2473-2477` | archived-template submit → 409 `template_archived` present |
| NFR-7 | `workflows.html:2791` | 401 → `/login.html` redirect present; drafts persist |

### Activity-3 test-audit sweep record (2026-07-11)

Two-pass static audit (pass 1 locate+read each WORKING flow's assertion, pass 2
subtle-vacuousness cross-check for swallowed catches / unentered-`if` / tautological
asserts) of all 10 WORKING flows against `tests/workflows.spec.js` +
`tests/persistence.spec.js`; adversarial G6 re-check of every cited assertion.
**Result: 0 drops — all 10 WORKING tests are non-vacuous** (each drives the real flow
and makes an `expect` on observable DB/UI state; none skipped/guarded/swallowed).
WORKING stays 10.

Two notes carried to the downstream Operations test-hardening WO (not
reclassifications — neither meets the G3 static-vacuousness drop bar):
- **FR-15 coverage gap (QA KR-2):** the six builder field types are exercised via
  `POST /createTemplate` API calls, **not** the builder Add-Field UI, and only 4/6
  types are covered (`type` census: checkbox, temperature, text, yes_no — **photo =
  0**, sub-steps thin). The flow the requirement literally names (adding each type
  *via the builder*) is under-proven. Add builder-UI Add-Field coverage for all six
  types incl. photo.
- **FR-10/FR-12 intel:** the `reject item with comment` test
  (`workflows.spec.js:485-508`) wraps its whole body in `if (await
  flagBtn.isVisible())` with **no `expect` at all** — genuinely vacuous, but it maps
  to FR-10/FR-12 (already UNPROVEN), so it drives no WORKING drop. Feed this to the
  FR-12 test-only WO as the starting point (the assertion to add).

## Out of scope

- The other four apps (Inventory, Onboarding, Users, Purchasing) — later passes.
- **Fixing** any flow — this PRD enumerates and marks; work orders fix.
- Any net-new feature or field type (hardening only, per the brief's constraints).
- Changing the build (static HTML + vanilla JS front end, Go + Postgres back end;
  no framework, no new dependency).
- The retry-parse / receipt pipeline and inventory endpoints (different app).

## Success metrics

- **Enumeration recall ≥ 90%** — `enumerated ÷ (enumerated + discovered-during-WO-
  build) ≥ 0.90`. Denominator: the **27** requirements above plus any flow the
  build surfaces. *(Product KR-2.)*
  - **Empirical finding (guinea-pig signal):** the first pass enumerated 23; the G5
    cross-check found 4 more → **single-pass recall ≈ 23/27 = 85%, under the 90%
    bar.** The two-pass total (27) clears it. Lesson for the other four app PRDs:
    **one enumeration pass is not enough for ≥90% — the cross-check is mandatory,
    not optional.** This is a process finding worth carrying to the roadmap.
- **5/5 apps gate** — this PRD is 1 of 5; the other four follow the same shape.
  *(Product KR-1.)*
- **0 known-broken flows** across Operations at cycle end — the 13 UNPROVEN (incl.
  any that graduate to BROKEN via a confirm-absence step) either reach WORKING or
  are explicitly waived. *(Engineering KR-1.)*
- **100% of UNPROVEN flows have a shipped WO** by cycle end. *(Delivery KR-1.)*
- **Every WORKING flow's test is non-vacuous** and every repaired flow carries a
  red-first proof. *(QA KR-1, KR-3.)*
