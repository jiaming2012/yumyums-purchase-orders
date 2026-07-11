# PRD — Onboarding app hardening

> **Cycle:** HQ hardening — first night-crew guinea-pig run.
> **App:** Onboarding — `onboarding.html`, `/api/v1/onboarding/*` + `/api/v1/videos/*`, Go + Postgres.
> **Depth:** *Enumerate + mark only.* This PRD is the honest flow map with per-flow
> status and a falsifiable definition of "working." It does **not** fix anything —
> each BROKEN/UNPROVEN flow becomes a work order after sign-off.
> **Role of this doc:** the third of the cycle's five app hardening PRDs; copies the
> shape of the signed **Operations** exemplar (`PRD-operations-hardening.md`).
> **Enumeration provenance (for the ≥90% recall KR):** flows were derived by reading
> `onboarding.html` (3 tabs — My Trainings / Manager / Builder, ~2140 lines, 41
> functions), `tests/onboarding.spec.js` (44 tests across 7 describe blocks), the
> onboarding Go package (`handler.go`, `db.go`, `video.go`, `seed.go`), and the chi
> router in `backend/cmd/server/main.go:496-603` — **16 endpoints enumerated in
> §Verification** (14 under `/onboarding`, 2 under `/videos`), then **cross-checked
> by a second independent pass (G5)** angled at the sign→reject→reopen state machine,
> the video presign→process→FFmpeg pipeline, and manager-vs-hire role gating —
> backend-only status transitions the UI-first pass could not see. Recall is measured
> post-build as `enumerated ÷ (enumerated + discovered-during-WO-build)`.
> **Scope note (resolves G2):** "critical flows" is interpreted as **all** real
> user-facing flows — no sub-selection. A flow is in-scope if a real person (crew
> hire / manager / admin) triggers it, or if a background job (video processing)
> materially changes what the user sees. Dropping the "critical" filter removes an
> undefined term that made the recall metric gameable.

## Objective

Produce a written, agreed enumeration of every end-to-end flow in the Onboarding
app — from an admin authoring a training template, through a crew hire watching
videos and checking off tasks, to a manager signing off, rejecting, or reopening a
section — with each flow honestly marked **WORKING**, **UNPROVEN**, or **BROKEN**,
each backed by a falsifiable definition of "working." This advances the **Product
objective**: KR-1 (5/5 apps have a hardening PRD enumerating their E2E flows,
delivered as an early blocking gate) and KR-2 (enumeration recall ≥ 90%). Its status
tally is the denominator the **Engineering** (0 known-broken flows at cycle end) and
**QA** (every flow has a real, asserting test) objectives grade against.

## Operators & users

- **Crew hire (team_member)** — sees their assigned + role-auto-assigned trainings in
  Tab 1 (My Trainings), works through sections top-to-bottom (locked until the prior
  section completes), checks off tasks/sub-items, watches video parts, expands FAQ
  answers, uploads proof photos where required, and reopens (unsubmits) their own
  completed-but-not-signed-off section to re-edit. Primary user of Tab 1.
- **Manager / admin** — reviews hires and per-template progress in Tab 2 (Manager),
  opens a hire's training, signs off a completed section (notes optional, readiness
  rating required), and rejects/reopens a section (whether signed-off or just
  complete) to send it back to the crew as corrective work.
- **Admin / manager (author)** — creates, edits (ID-preserving full replace), and
  soft-deletes templates in Tab 3 (Builder): sections, checkbox items with sub-items,
  video_series with parts, FAQ items, sign-off requirements + role restrictions,
  reference photos, and proof-photo requirements. Uploads and processes training
  videos via the presign→PUT→FFmpeg pipeline.
- **The overnight crew (indirect)** — consumes this enumeration + status as the
  work-order backlog; a flow marked BROKEN/UNPROVEN here is a candidate WO.

## Requirements

Status legend: **WORKING** = flow works E2E *and* a test drives it and asserts
observable DB/UI state · **UNPROVEN** = flow appears to work in code but no test
verifies the behavior it names (missing or vacuous test) · **BROKEN** = flow is
**confirmed** incomplete/stubbed by code inspection, or a test reveals it does not do
what it claims. Every requirement traces to an OKR key result.

**G1 resolution:** BROKEN is reserved for *confirmed* breakage only. A flow I merely
*suspect* is broken but have not confirmed absent stays **UNPROVEN**. Three flows
(FR-16 video processing, NFR-3 auth-tier 403, NFR-5 reopen-a-video-led-section) are the
**highest-priority** UNPROVEN: each carries a *confirm-absence* step and graduates to
BROKEN if that step confirms
the behavior is missing/stubbed. Nothing here is marked BROKEN — no flow was
confirmed broken by inspection or a red test.

### Tab 1 — My Trainings (crew hire)

- **FR-1** — The system lists a hire's trainings, merging explicitly assigned
  templates (`ob_template_assignments`) with role-auto-assigned ones (template.roles
  overlaps user.roles), each showing section-level progress (`X of Y sections
  complete`, not item-level). *(GET `/myTrainings`; `db.go:687-749`)* — **WORKING**
  (empty-state + assigned-after-assignment + section-level-progress tests) — traces
  to Product KR-1, QA KR-2.
- **FR-2** — A hire opens a training and sees each section computed as
  locked / active / complete / signed_off, with a section locked until every prior
  section is complete or signed off. *(GET `/hireTraining/{hireId}?templateId=`;
  `db.go:436-606`, `canActivateSection` 672-683)* — **WORKING** (section-unlocks-
  after-completing-previous test drives the real gate) — traces to Product KR-1, QA KR-2.
- **FR-3** — Checking a checkbox item persists progress and survives page reload; the
  section recomputes as complete when all items are checked. *(POST `/saveProgress`
  with `progress_type:item`; `db.go:968-986`)* — **WORKING** (checkbox-persists-after-
  reload test) — traces to Product KR-1, QA KR-2.
- **FR-4** — A checkbox item with sub-items derives its checked state from all
  sub-items; the section completes only when every sub-item is checked, surviving
  reopen. *(POST `/saveProgress` with `progress_type:sub_item`; `db.go:564-577`,
  636-670)* — **WORKING** (sub-items-persist + completing-all-sub-items-marks-section-
  complete + manager-shows-correct-sub-item-completion tests) — traces to Product KR-1,
  QA KR-2.
- **FR-5** — Watching a video part marks it complete and persists across reload;
  `max_watched_time` is tracked via `GREATEST(...)` so scrub-back does not lose
  progress. *(POST `/saveProgress` with `progress_type:video_part` /
  `video_watch_position`; `db.go:970-978`, 482-505)* — **WORKING** (video-part-watched-
  persists + video-part-checked-returned-by-API tests) — traces to Product KR-1, QA KR-2.
- **FR-6** — A video part with a URL shows a play button; opening plays it in a modal;
  the close button dismisses the player and saves watch position. *(client
  `initVideoPlayer`, `markVideoPartWatched`, `saveVideoWatchProgress`;
  `onboarding.html:1234-1322`)* — **WORKING** (play-button-shows + modal-close-dismisses
  tests) — traces to Product KR-1, QA KR-2.
- **FR-7** — An FAQ item persists a "viewed" state when expanded; an FAQ section
  completes only when every question has been expanded, and the last expanded question
  stays open. *(POST `/saveProgress` with `progress_type:faq`; `db.go:557-558`,
  640-644)* — **WORKING** (FAQ-shows-Q&A + FAQ-viewed-count-completes + FAQ-last-
  question-stays-expanded tests) — traces to Product KR-1, QA KR-2.
- **FR-8** — Editing a hire's progress is **blocked** once their section is complete
  and awaiting sign-off: unchecking returns `400 section_awaiting_signoff`.
  *(POST `/saveProgress` → `IsSectionLockedForEdits`; `handler.go:199-209`,
  `db.go:900-966`)* — **WORKING** (backend-rejects-progress-updates-for-completed-
  sections test asserts the 400 + error body) — traces to Engineering KR-1, QA KR-2.
- **FR-9** — A hire can **reopen (unsubmit)** their own completed-but-unsigned section
  to re-edit: the sign-off (if any) is deleted and the first item's progress is
  removed, reverting the section to active. *(POST `/reopenSection`;
  `handler.go:474-505`, `ReopenSection` `db.go:998-1048`)* — **WORKING** (crew-can-
  unsubmit-a-completed-section-and-re-edit test drives the button + asserts items
  become interactive) — traces to Engineering KR-1, QA KR-2.
- **FR-10** — A hire uploads a proof photo on an item that requires one; the photo URL
  persists into `ob_progress.value` and round-trips on reopen. *(POST `/saveProgress`
  carrying `value`; `db.go:560-563`, 970-978; client `handleOBPhotoCaptureClick`
  `onboarding.html:256`)* — **UNPROVEN** (a test asserts the proof-photo *button*
  appears on a reopened item, but no test drives an actual photo capture → persist →
  reopen round-trip of the URL) — traces to QA KR-1.

### Tab 2 — Manager (manager / admin)

- **FR-11** — The manager sees the list of active hires with aggregate progress and
  per-template summaries (progress %, pending-sign-off flag, explicit-vs-role
  assignment). *(GET `/managerHires`; `db.go:751-898`)* — **WORKING** (manager-sees-
  hire-with-assigned-training + manager-shows-role-auto-assigned + manager-shows-
  correct-sub-item-completion tests) — traces to Product KR-1, QA KR-2.
- **FR-12** — A hire with a complete section awaiting sign-off is surfaced to the
  manager as "pending sign-off" and stays in the Active sub-tab (not Completed) until
  signed off. *(GET `/managerHires` `pending_signoff`; `db.go:856-869`; client
  `mgrState.subView`)* — **WORKING** (hire-with-pending-sign-off-stays-in-Active +
  manager-navigates-directly-to-Manager-tab-when-pending tests) — traces to Product
  KR-1, QA KR-2.
- **FR-13** — A manager signs off a completed section: readiness rating required
  (`ready` / `needs_practice` / `struggling`), notes optional; the sign-off is
  idempotent (`ON CONFLICT (section_id,hire_id) DO NOTHING`) and records manager
  attribution ("By {manager} @ {datetime}"). *(POST `/signOff`; `handler.go:220-298`,
  `SignOff` `db.go:988-996`)* — **WORKING** (rating-required + succeeds-with-rating-
  only + rejects-missing-rating + records-attribution + attribution-on-hire-view
  tests) — traces to Engineering KR-1, QA KR-2.
- **FR-14** — A section's `sign_off_roles` restricts who may sign off: a user whose
  roles don't intersect (and who isn't superadmin) is refused
  `403 sign_off_role_required`, and the sign-off button is hidden from them in the UI.
  *(POST `/signOff`; `handler.go:260-282`; client gate `onboarding.html:532-534`)* —
  **UNPROVEN** (one test asserts the *button is hidden* for a team_member — real UI
  role gating — but no test drives a restricted user POSTing `/signOff` to assert the
  `403 sign_off_role_required` refusal at the API) — traces to QA KR-1, Engineering KR-1.
- **FR-15** — A manager **rejects/reopens** a section — whether already signed off or
  merely complete — reverting it to active so the crew re-edits: the sign-off is
  deleted and the first item's progress removed (shared `ReopenSection`).
  *(POST `/rejectSection`; `handler.go:436-472`, `db.go:998-1048`)* — **WORKING**
  (manager-can-reject-a-signed-off-section + manager-can-reject-before-sign-off tests
  drive the button and assert the section header no longer shows "Signed Off" /
  "Waiting for Sign-Off") — traces to Engineering KR-1, QA KR-2.

### Video pipeline (Builder authoring — presign / process)

- **FR-16** — Uploading a training video presigns a PUT URL against DO Spaces
  (content-type restricted to mp4 / quicktime / webm), the browser PUTs the file,
  then a process call runs FFmpeg (H.264/AAC transcode for .mov/.webm), extracts a
  thumbnail at 2s, uploads both back, and updates `ob_video_parts.url` +
  `thumbnail_url`. *(POST `/videos/presign` → PUT S3 → POST `/videos/process`;
  `handler.go:537-640`, `video.go:22-206`; client `uploadVideoFile`
  `onboarding.html:1086-1191`)* — **UNPROVEN (priority)** (no test touches
  `/videos/presign` or `/videos/process` — the only video tests exercise the
  *save-for-later* download fallback, not the real presign→PUT→FFmpeg path;
  confirm-absence step: this needs S3 creds + an FFmpeg binary, so it may be
  **waived** as environment-gated rather than fixed) — traces to Engineering KR-1,
  QA KR-1.
- **FR-17** — When an upload fails, the part enters a "save-for-later" state offering
  a local download named after the (sanitized, required) part title. *(client
  `uploadVideoFile` error branch, `save-video-local` action;
  `onboarding.html:1120-1191`)* — **WORKING** (save-video-for-later-requires-part-title
  + save-video-for-later-uses-part-title-in-filename tests) — traces to QA KR-2.
- **FR-18** — A manager uploads a custom thumbnail for a video part in place of the
  FFmpeg-extracted one. *(client `uploadCustomThumbnail`, presign against
  `/api/v1/photos/presign`; `onboarding.html:1192-1233`)* — **UNPROVEN** (no test
  drives custom-thumbnail upload) — traces to QA KR-1.

### Tab 3 — Builder (template authoring, manager / admin)

- **FR-19** — An author creates a new template with sections, checkbox items
  (+ sub-items), video_series (+ parts), and FAQ items. *(POST `/createTemplate`;
  `handler.go:300-332`, `CreateTemplate` `db.go:1050-1076`, `insertSectionsTx`
  1265-1311)* — **WORKING** (can-create-a-new-template-via-Builder + shows-existing-
  seed-template tests) — traces to Product KR-1, QA KR-2.
- **FR-20** — An author edits a template via **ID-preserving full replace**: existing
  section/item/video_part/sub_item UUIDs are updated in place (so `ob_progress` rows
  referencing item IDs are not orphaned), and removed children are deleted deepest-
  first. *(PUT `/updateTemplate/{id}`; `UpdateTemplate` `db.go:1078-1235`)* —
  **WORKING** (progress-survives-template-edit — adding a new item preserves existing
  progress; complemented by the negative progress-lost-when-item-removed test) —
  traces to Engineering KR-1, QA KR-2.
- **FR-21** — An author soft-deletes (archives) a template by setting `archived_at`;
  archived templates drop out of all list/assignment queries but keep their rows.
  *(DELETE `/deleteTemplate/{id}`; `handler.go:507-535`)* — **UNPROVEN** (used only as
  test cleanup — 7 call sites — with no test asserting the archived template
  disappears from `/templates` or `/myTrainings` while its rows survive) — traces to
  QA KR-1.
- **FR-22** — An author sets a section's sign-off requirement and role restriction:
  the sign-off role picker appears only when "Require Sign-off" is on, and selected
  roles persist through save + reopen. *(client `renderOBSection` role chips
  `onboarding.html:961-1021`; persisted via `sign_off_roles`)* — **WORKING** (role-
  picker-shows-when-enabled + disappears-when-disabled + selected-roles-persist-after-
  save-and-reopen tests) — traces to Product KR-1, QA KR-2.
- **FR-23** — An author attaches a reference photo and/or a proof-photo requirement to
  items and sub-items; the reference photo URL persists through save and displays in
  the hire's training. *(fields `reference_photo_url`, `require_proof_photo` on items
  + sub_items; `db.go:225-235`, 1129-1189)* — **WORKING** (reference-photo-URL-
  persists-through-save-and-displays test) — traces to Product KR-1, QA KR-2.
- **FR-24** — Leaving the Builder (HQ back link or tab switch) with unsaved changes
  prompts a confirmation. *(client `obBuilderHasUnsavedChanges`, `beforeunload` +
  `show()` guard; `onboarding.html:141-151`, 299-300, 2103-2114)* — **WORKING** (HQ-
  back-link-prompts-when-builder-has-unsaved-changes test) — traces to QA KR-2.

### Assignment (manager / admin)

- **FR-25** — A manager assigns a template to a hire (idempotent — `ON CONFLICT
  (hire_id,template_id) DO NOTHING`). *(POST `/assignTemplate`; `handler.go:364-398`,
  `AssignTemplate` `db.go:1313-1321`)* — **WORKING** (assignment drives 24 test setups,
  incl. shows-assigned-template-after-assignment asserting the training then appears) —
  traces to Product KR-1, QA KR-2.
- **FR-26** — A manager unassigns a template from a hire, removing the explicit
  assignment row (a role-auto-assigned template still shows). *(POST
  `/unassignTemplate`; `handler.go:400-434`, `UnassignTemplate` `db.go:1323-1330`)* —
  **UNPROVEN** (no test calls `/unassignTemplate` at all — the unassign flow, incl.
  the role-auto-assign-still-shows edge, is entirely unverified) — traces to QA KR-1.
- **FR-27** — A single template can be fetched with its full nested structure for the
  Builder editor and test setup. *(GET `/templates/{id}`; `handler.go:70-89`,
  `GetTemplate` `db.go:285-305`)* — **WORKING** (drives most Builder + reject/reopen
  test setups; structure asserted indirectly across many tests) — traces to Product
  KR-1, QA KR-2.

### Cross-cutting (non-functional / platform guarantees)

- **NFR-1 (Progress persistence contract)** — Every crew-entered progress value —
  checkbox item, sub-item, video part, FAQ view, proof-photo URL — flows through
  `POST /saveProgress` → `ob_progress` (upsert on check, delete on uncheck) and
  round-trips on `GET /hireTraining`. — **WORKING** for the five canonical progress
  types (checkbox / sub-item / video / FAQ tested; proof-photo URL round-trip is the
  FR-10 gap) — traces to Product KR-1, QA KR-2.
- **NFR-2 (Section state machine)** — A section moves
  locked → active → complete → signed_off, and back to active via reopen/reject; the
  awaiting-sign-off state locks edits (FR-8) and gates the next section (FR-2). — **WORKING**
  (the transition set is covered by FR-2/FR-8/FR-9/FR-13/FR-15 tests) — traces to
  Engineering KR-1, QA KR-2.
- **NFR-3 (Authorization tiers — API 403)** — Manager/admin-only endpoints
  (`/managerHires`, `/signOff`, `/rejectSection`, template CRUD, assign/unassign,
  `/videos/*`) return `403 forbidden` to a team_member; `sign_off_roles` further
  restricts `/signOff` to `403 sign_off_role_required`. *(guards `isManagerOrAdmin`
  in `handler.go`; role check 260-282)* — **UNPROVEN (priority)** (the guards are
  present in code and the *frontend* hides t2/t3 for non-managers, but **no test posts
  to a manager endpoint as a team_member to assert a 403** — every test runs as
  admin/superadmin; confirm-absence step: add a team_member session and assert 403 on
  a manager endpoint) — traces to Engineering KR-1, QA KR-1.
- **NFR-4 (Video storage / FFmpeg availability)** — The video presign/process
  endpoints return `503 video_storage_not_configured` when Spaces isn't wired, and
  processing depends on an `ffmpeg` binary on the host. *(`handler.go:548-551`,
  603-606; `video.go` `exec.CommandContext`)* — **UNPROVEN** (the 503 fallback and the
  FFmpeg-missing failure path are untested) — traces to Engineering KR-1, QA KR-1.

### Additional flows — G5 second-pass cross-check

A second, independent pass (angled at the sign→reject→reopen state machine, the video
pipeline, and manager-vs-hire role gating) found **3 flows the first UI-first pass
missed** — the empirical basis for the recall note in §Success metrics. Each is UNPROVEN:

- **FR-28** — Seeding: on boot the server idempotently inserts the "Kitchen Basics
  Training" template (skip-if-name-exists, refresh roles on existing), so a fresh DB
  always has one working template. *(POST-boot `SeedOnboardingTemplates`;
  `seed.go:13-124`; `main.go:283-285`)* — **UNPROVEN** (tests *depend* on the seed
  template existing but none asserts the idempotent re-seed / role-refresh behavior) —
  traces to Engineering KR-1, QA KR-1.
- **FR-29** — Manager-hire aggregate progress is computed by a distinct, heavier query
  path than the hire's own `/myTrainings` (CROSS JOIN users, per-hire pending-signoff
  subqueries) — the two must agree on completion %. *(GET `/managerHires`
  `db.go:751-898` vs GET `/myTrainings` `db.go:687-749`)* — **UNPROVEN** (each side is
  tested in isolation; no test asserts the two progress numbers agree for the same
  hire/template) — traces to Engineering KR-1, QA KR-1.
- **NFR-5 (Reopen a video-led section is a silent no-op)** — reopen/reject of a section
  whose first item is a `video_series` does NOT revert it to active: the section stays
  `complete`/`signed_off` and the handler still returns `{"ok":"true"}`, masking it. —
  **BROKEN** *(confirm-absence sweep 2026-07-11, G6-confirmed at every causal link:
  `ReopenSection` selects the first `ob_items` row with no type filter
  (`db.go:1015-1017`), which per schema `0019_ob_items.sql:6` can be `video_series`;
  it deletes progress by the parent `ob_items.id` (`db.go:1040`), but video progress is
  keyed by `ob_video_parts.id` (written `db.go:970-978`, required for completeness at
  `db.go:645-651`), so the DELETE matches zero video_part rows and `isSectionComplete`
  still returns true. No `ob_video_parts` delete exists anywhere in `ReopenSection`
  (1000-1048). FAQ-led sections are SAFE — faq progress is keyed by the faq item's own
  `ob_items.id`, so the parent-keyed DELETE removes it. Shared defect: both
  `/reopenSection` (FR-9, crew) and `/rejectSection` (FR-15, manager) call the same
  `ReopenSection`.)* — traces to Engineering KR-1, QA KR-1. **→ Activity-4 fix-card
  (resolve the first checkable unit by item type — video_part / faq / sub_item / item —
  and delete its progress; red-first test on the seed's video-led Equipment Training §).**

## Acceptance criteria

Surface-anchored, Given/When/Then. These define "working" for representative flows;
every enumerated flow inherits the pattern of *drive-the-real-flow + assert-observable-
state* (the WORKING bar).

- **AC-1 (FR-3, checkbox persistence):** *Given* an assigned training with a checkbox
  item, *When* the hire checks it and reloads the page, *Then* the item is still
  checked and the section progress reflects it.
- **AC-2 (FR-2, section gating):** *Given* a template with two ordered sections,
  *When* the hire has not completed section 1, *Then* section 2 renders `locked`;
  *When* section 1 completes, *Then* section 2 becomes `active`.
- **AC-3 (FR-8, edit lock):** *Given* a complete section that requires sign-off,
  *When* the hire POSTs `/saveProgress` to uncheck an item, *Then* the API returns
  `400 section_awaiting_signoff` and the DB row is unchanged.
- **AC-4 (FR-15, manager reject):** *Given* a signed-off section, *When* the manager
  clicks "Reject & Reopen" and confirms, *Then* the `ob_signoffs` row is deleted, the
  first item's progress is removed, and the section header no longer shows "Signed
  Off."
- **AC-5 (FR-13, sign-off rating gate):** *Given* a complete section, *When* the
  manager POSTs `/signOff` with an empty rating, *Then* the API returns
  `400 invalid_rating`; *When* the rating is `ready` (notes empty), *Then* it succeeds
  and records manager attribution.
- **AC-6 (FR-16, video pipeline — UNPROVEN priority):** *Given* an author with a raw
  .mov file, *When* they upload it (presign → PUT → process), *Then*
  `ob_video_parts.url` points at a transcoded .mp4 and `thumbnail_url` at an extracted
  JPEG. *(May be waived as environment-gated — needs S3 creds + FFmpeg.)*
- **AC-7 (NFR-3, auth tier — UNPROVEN priority):** *Given* a team_member session,
  *When* it POSTs `/signOff` (or any manager endpoint), *Then* the API returns `403`
  and no row is written. *(A red-first test must show this failing if the guard is
  removed.)*
- **AC-8 (NFR-5, reopen a video-led section — UNPROVEN priority):** *Given* a complete
  section whose first item is a video_series, *When* it is reopened, *Then* the section
  reverts to active. *(Confirm-absence: current code only unchecks a checkbox/sub-item
  first item.)*

## Verification plan

- **Environment:** localhost Postgres (`brew postgresql@16`) — the E2E suite requires
  a local DB; the remote Windows DB is too slow (N+1 × 50ms RTT). Playwright blocks
  service workers (`serviceWorkers: 'block'`). FR-16/NFR-4 additionally need DO Spaces
  creds + an `ffmpeg` binary and are candidates for an explicit waiver.
- **Suite:** `tests/onboarding.spec.js` (44 tests, 7 describe blocks). Run per-flow
  during iteration (`npx playwright test tests/onboarding.spec.js -g "<name>"`), full
  suite (`task test`) at gate.
- **This PRD specifies the test each flow needs; it does not write them (resolves G4).**
  Writing/repairing a test is itself a work order — this doc names the assertion, the
  WO delivers it.
- **What each status turns into downstream:**
  - **WORKING** flows: a **test-audit WO** — spot-check the existing test is
    non-vacuous (no `test.skip`, no early `return` guard, an assertion that would fail
    if the feature broke). If vacuous, it drops to UNPROVEN.
  - **UNPROVEN** flows: a **test-only WO first (resolves G3)** — write a real seeded,
    red-first assertion. The flow graduates to a **fix WO only if that test goes red**.
  - **UNPROVEN (priority)** flows (FR-16, NFR-3, NFR-5): the test-only WO opens with a
    **confirm-absence step** (drive the presign/process endpoints; post as a
    team_member; reopen a video-led section). If the behavior is confirmed
    missing/stubbed, the flow is re-marked **BROKEN**; FR-16 may instead be **waived**
    as environment-gated.
- **Vacuous/stale note:** one early return-guard exists — `tests/onboarding.spec.js:991`
  (`if (signOffSections.length < 2) return;`) silently skips the notes-optional sign-off
  assertion if the seed has fewer than two sign-off sections. It is not a false-green on
  the named behavior (the assertion still runs in the seeded case), but it should be
  replaced by a self-seeded template in the WO so it can never silently skip.
- **Endpoints in scope (16):** GET `/onboarding/templates`, GET
  `/onboarding/templates/{id}`, GET `/onboarding/myTrainings`, GET
  `/onboarding/hireTraining/{hireId}?templateId=`, GET `/onboarding/managerHires`,
  POST `/onboarding/saveProgress`, POST `/onboarding/signOff`, POST
  `/onboarding/rejectSection`, POST `/onboarding/reopenSection`, POST
  `/onboarding/createTemplate`, PUT `/onboarding/updateTemplate/{id}`, DELETE
  `/onboarding/deleteTemplate/{id}`, POST `/onboarding/assignTemplate`, POST
  `/onboarding/unassignTemplate`, POST `/videos/presign`, POST `/videos/process`.

### Status tally (the denominator downstream objectives grade against)

Total requirements enumerated: **34** (29 FR + 5 NFR) — 31 first-pass + 3 from the
G5 cross-check.

**Updated by the Activity-2 confirm-absence sweep (2026-07-11, G6-passed).** The
priority-UNPROVEN **NFR-5 graduated to BROKEN** (reopen/reject of a video-led section is
a confirmed silent no-op — every causal link verified at cited lines). FR-16 and NFR-3
confirmed present-but-untested (FR-16 → recommended WAIVER, env-gated on S3+FFmpeg; NFR-3
guards invoked at 12 call sites). Net: WORKING 23 · UNPROVEN 11 → 10 · BROKEN 0 → 1.

| Status | Count | Flows |
|---|---|---|
| **WORKING** | 23 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-11, FR-12, FR-13, FR-15, FR-17, FR-19, FR-20, FR-22, FR-23, FR-24, FR-25, FR-27, NFR-1, NFR-2 |
| **UNPROVEN** | 10 | FR-10, FR-14, FR-18, FR-21, FR-26, FR-28, FR-29, NFR-4 · **priority:** FR-16 (waiver candidate), NFR-3 |
| **BROKEN** | 1 | **NFR-5** (reopen/reject of a video-led section is a silent no-op — `ReopenSection` deletes progress by parent `ob_items.id`, never `ob_video_parts.id`) → an Activity-4 fix-card |

*(Verify: 23 WORKING + 10 UNPROVEN + 1 BROKEN = 34 total. The 10 UNPROVEN + 1 BROKEN =
the candidate WO backlog. NFR-5 (BROKEN) = an Activity-4 code-fix + regression-test card
and enters the Engineering-KR "0 known-broken" denominator. FR-16 is flagged for an
explicit operator WAIVER decision at triage (environment-gated, like Inventory
Trends/Cost D-3) — its pipeline is fully implemented (`video.go`), just untestable
without S3 creds + an ffmpeg binary. Every UNPROVEN must have a shipped WO by cycle end —
Delivery KR-1 — and reach 0 known-broken — Engineering KR-1.)*

### Activity-2 confirm-absence sweep record (2026-07-11, G6-passed)

Two-pass static audit (pass 1 UI-flow, pass 2 state-machine / video / seed cross-check)
of all 11 UNPROVEN flows against `onboarding.html` + `backend/internal/onboarding/*`;
adversarial G6 re-check of every citation. **Pass 2 was decisive on NFR-5** — the break
is a DB-key mismatch (`ob_items.id` vs `ob_video_parts.id`) invisible to a UI-first pass.

| Flow | Present at / verdict | Confirm-note |
|---|---|---|
| FR-10 | `onboarding.html:277`; `db.go:973-978` | proof-photo capture→upload→`saveProgress(value)` persists; present, untested |
| FR-14 | `handler.go:260-282` | 403 `sign_off_role_required` guard + UI gate present; API refusal untested |
| FR-16 | `handler.go:540-640`; `video.go:22-206` | **WAIVER candidate** — full presign→PUT→FFmpeg pipeline present; env-gated (S3+ffmpeg), not broken |
| FR-18 | `onboarding.html:1192-1221` | custom-thumbnail presign→PUT→`part.thumbnail_url` present; untested |
| FR-21 | `handler.go:527`; `db.go:252,723,870` | soft-delete `archived_at=now()` + all queries filter `archived_at IS NULL`; untested |
| FR-26 | `db.go:1324-1330` | unassign deletes `ob_template_assignments` row; present, no test calls it |
| FR-28 | `seed.go:98-124` | seed skip-if-name-exists + role-refresh present; re-seed untested |
| FR-29 | `db.go:687-749` vs `751-898` | both progress-query paths present; agreement untested |
| NFR-3 | `handler.go:39` invoked at `52,151,230,309,343,373,409,446,516,558,613` | `isManagerOrAdmin` wired on every manager endpoint; no team_member 403 test |
| NFR-4 | `handler.go:548-551,603-606` | `presigner==nil → 503 video_storage_not_configured` present; untested |
| **NFR-5** | **→ BROKEN** `db.go:1015-1017,1040` vs `645-651` | reopen deletes by parent `ob_items.id`, never `ob_video_parts.id` → video-led section stays complete |

## Out of scope

- The other four apps (Operations, Inventory, Users, Purchasing) — separate PRDs.
- **Fixing** any flow — this PRD enumerates and marks; work orders fix.
- Any net-new feature, field type, or section state (hardening only).
- Changing the build (static HTML + vanilla JS front end, Go + Postgres back end, chi
  router; no framework, no new dependency).
- The shared photo pipeline internals (`/api/v1/photos/*`, `photos.GeneratePresigned*`)
  beyond how onboarding consumes them.
- The **video pipeline as a standalone product** — see the PARK note: if it grows past
  presign/process, it warrants its own PRD split; this doc enumerates it at flow level
  (FR-16/FR-17/FR-18/NFR-4) but does not deep-map FFmpeg encoding options.

## Success metrics

- **Enumeration recall ≥ 90%** — `enumerated ÷ (enumerated + discovered-during-WO-
  build) ≥ 0.90`. Denominator: the **34** requirements above plus any flow the build
  surfaces. *(Product KR-2.)*
  - **Empirical finding (guinea-pig signal):** the first (UI-first + endpoint) pass
    enumerated 31; the G5 cross-check (state machine / video / role gating) found 3
    more (FR-28 seeding, FR-29 dual progress-query agreement, NFR-5 reopen-first-item
    limitation) → **single-pass recall ≈ 31/34 = 91.2%**, just over the bar. The
    cross-check still surfaced the highest-value gap (NFR-5, a probable latent bug),
    reinforcing the exemplar's lesson: **the second pass is mandatory, not optional.**
- **5/5 apps gate** — this PRD is 3 of 5; all follow the same shape. *(Product KR-1.)*
- **0 known-broken flows** across Onboarding at cycle end — the 11 UNPROVEN (incl. any
  that graduate to BROKEN via a confirm-absence step) either reach WORKING or are
  explicitly waived (FR-16/NFR-4 are the likely waiver candidates — environment-gated).
  *(Engineering KR-1.)*
- **100% of UNPROVEN flows have a shipped WO** by cycle end. *(Delivery KR-1.)*
- **Every WORKING flow's test is non-vacuous** — including replacing the line-991
  early-return guard — and every repaired flow carries a red-first proof. *(QA KR-1, KR-3.)*
