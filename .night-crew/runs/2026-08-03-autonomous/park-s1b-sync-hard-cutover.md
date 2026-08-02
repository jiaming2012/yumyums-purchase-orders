# 🛑 PARK — Card S1b `sync-hard-cutover`

Run `overnight-20260803`, Track A, branch `card/s1b-sync-hard-cutover`
(worktree preserved at `/home/jcole/projects/hq-worktrees/s1b-sync-hard-cutover`).

**A recorded PARK trigger fired. No production code, SQL, policy, migration or test was
written. `HQ_SYNC_REST_URL` was NOT set and nothing was deployed — the standing interlock
(decision 119) is unchanged and still in force.**

Every claim below is first-hand measurement on this branch at `overnight-20260803`
(`68aeda6`). Provenance is in §7.

---

## 1. The trigger, verbatim, and why it fired

> 🛑 **PARK if retiring `/saveResponse` turns out to reopen ledger decision 49** (the
> server-side duplicate guard) — that is a recorded fork, not a judgement call.

It fired — and, as with B2's park, for a reason **stronger than the trigger anticipated.**

Decision 49 does not merely *touch* the cutover. Its decisive argument **is a claim about
the cutover**, and that claim is false as built. Verbatim (`ledger.md:1100-1110`):

> The decisive argument is one the fork document did not carry: Activity 1 ends in
> `sync-hard-cutover`, where RxDB replicates rows **straight from Postgres** and **there is
> no API boundary left to translate at** — so (c) is a translation layer with a known
> expiry date…

Measured: **RxDB does not replicate from HQ's Postgres.** It replicates from a *second,
different* Postgres — the self-hosted Supabase substrate — and **there is no mechanism of
any kind carrying a row from that substrate back into HQ's Postgres.** The API boundary is
not removed by the cutover. It is the only thing that connects the two databases, and the
cutover as specified deletes it.

Decision 49 chose option (a) *because* the boundary was going away. It is not going away.
That is decision 49 reopened, on measurement, exactly as the trigger contemplates.

---

## 2. The measurement — two databases, one direction

`sync-schema/sql/0001_sync_tables.sql` creates the four replicated tables. Its own header,
and 0002's, state the topology in capitals (`0002_hq_fdw.sql:6-13`):

> ⚠  **THIS FILE DOES NOT RUN AGAINST HQ's POSTGRES.** … This targets the SELF-HOSTED
> SUPABASE Postgres (`docker-compose.supabase.yml`'s `db` service).

So there are two `public.submission_responses` tables:

| | HQ Postgres (`:5433`, db `yumyums`) | Substrate (Supabase `db` service) |
|---|---|---|
| created by | `backend/internal/db/migrations/0012_submission_responses.sql` | `sync-schema/sql/0001_sync_tables.sql:202` |
| `id` | `UUID PRIMARY KEY DEFAULT gen_random_uuid()` | `text primary key`, no default |
| FKs | to `checklist_fields`, `users` | none |
| extra cols | `lamport_ts` | `_deleted`, `_modified` (+ trigger) |
| written by | `saveResponse` (`repository.go:794`) | RxDB push → PostgREST |
| **read by `/myChecklists`, `/myDrafts`, `/submit`, approvals, reports** | **YES** | **NO** |

The FDW bridge is **one-directional and carries permissions, not data.** It imports exactly
four foreign tables into the substrate (`0002_hq_fdw.sql:194-261`):

```
hq_template_assignees   hq_user_roles   hq_field_templates   hq_template_approvers
```

— HQ-side views from migrations `0073_sync_fdw_views.sql` / `0074_sync_fdw_approver_view.sql`,
existing so the substrate's RLS predicates (`hq_can_see_template`, `hq_can_see_field`,
`hq_can_approve_field`) can resolve against live HQ permissions.

**Nothing goes the other way.** Repo-wide search for a reverse FDW, a writeback worker, a
backfill, a reconcile job or an endpoint returns nothing. The single mention is a note that
one would have to be written (`0001_sync_tables.sql:147`, *"NOTE for B2 and for whoever
writes the backfill"*) — future work, not code.

### What that means for the card's first `done_when:` row

> *A field answer entered in the runner reaches Postgres **via RxDB**, with `/saveResponse`
> unmounted — verified by a Playwright test that asserts the value survives back-to-list and
> reopen.*

Under the built topology this row is **not satisfiable, and worse, it is misleading if
forced.** A field answer pushed through RxDB lands in the *substrate's*
`submission_responses`. The value would indeed "survive back-to-list and reopen" — served
from local IndexedDB and the substrate — while HQ's Postgres never saw it. Meanwhile
`hydrateFieldState` reads `DRAFT_RESPONSES` from `GET /myDrafts` (HQ), and `SUBMIT_CHECKLIST`
builds the submission from HQ's `submission_responses`.

So unmounting `/saveResponse` does not move the write. **It silently detaches the answers
from submission.** A crew member would fill a checklist, see every answer persist, press
Submit, and submit an empty one. The card's own test would pass while that happened — which
is precisely the failure class this milestone keeps arming itself against.

**This is not a gap I can close inside the card's no-parallel-run constraint.** Bridging the
two servers is the thing ledger **decision 92** already ruled out in the obvious direction:
`max_prepared_transactions` is 0 at both ends, no transaction can contain both, and B2
parked on exactly this wall. *"The park was correct."* It is the same wall.

---

## 3. Second finding — `backend/internal/sync` is load-bearing for the WORKFLOW package

The slate already carries one footprint correction for this package (`jwtbridge`, `proxy`,
`rowvisibility_rls_test.go` must be KEPT). **There is a second one, of the same kind, and it
lands on the files the card says to DELETE.**

`backend/cmd/server/main.go:47-95` — `workflowOpRouter` — is the router behind `POST /ops`:

```go
case opsync.OpSetField:
    if err := workflow.SaveResponseFunc(ctx, pool, p.FieldID, p.Value, userID); err != nil {
        if errors.Is(err, workflow.ErrUnknownField) { … 422 "unknown_field" }
case opsync.OpSubmitChecklist:
    workflow.ValidateFailNotesFunc(...)
    workflow.ValidateResubmitPhotoFunc(...)
    workflow.SubmitChecklistFunc(...)
```

`ops.go` / `handler.go` are not "the Lamport-clock layer". They are **the transport for all
server-side workflow validation** — fail-note validation, resubmit-photo validation, archived
template rejection, submission construction. RxDB pushes rows to PostgREST and **cannot run
any Go validator.**

Some of that is genuinely covered on the substrate side and I want to be precise rather than
alarmist: `saveResponse`'s app-level existence check (`ErrUnknownField`, the *"churn-driven
Friday P0"* guard at `repository.go:795-807`) **does** have a substrate equivalent — the
`submission_responses_insert` policy's `with check (hq_can_see_field(field_id))`, which
resolves an unknown field to false. `TestRowVisibilityRLS/W8` pins exactly that. **That guard
survives.**

What has **no** substrate equivalent is the `SUBMIT_CHECKLIST` half: `ValidateFailNotesFunc`,
`ValidateResubmitPhotoFunc`, `SubmitChecklistFunc` and the `ErrTemplateArchived` path. Those
are Go business logic with no RLS counterpart, and deleting `handler.go`/`ops.go` deletes
their only transport. Filed **B-68**.

---

## 4. Third finding — the card's client-side premise is factually wrong

The card, the roadmap bullet, `bootstrap.js:9` and `conflict-notice-ui.js:23` all describe
the live write path as `autoSaveField` → `POST /saveResponse`, and the card's scope names it
as one of **two** write paths to replace.

**`autoSaveField` does not exist.** It is defined nowhere in the tree — no `function
autoSaveField`, no `autoSaveField =`, no `autoSaveField:`. It is *called* once, at
`workflows.html:2219`, on the fail-note-photo path, where it is a live `ReferenceError`.
Filed **B-65**.

The actual live path is a single one, and it runs **through the op-log the card intends to
delete**:

```
debouncedSaveField (workflows.html:389)
  → submitOp('SET_FIELD', …) (workflows.html:404, sync.js:781)
    → POST /api/v1/workflow/ops
      → workflowOpRouter (main.go:62) → workflow.SaveResponseFunc → HQ Postgres
```

`POST /saveResponse` (`main.go:558`) is mounted, but the runner does not use it — it is
reached by `tests/persistence.spec.js` and `tests/ops-authz-coverage.spec.js`. So "retire
both write paths" is really "retire the one write path, which is the op-log", and the op-log
is §3's problem.

---

## 5. Fourth finding — `sync.js` is not separable as "the op-log" either

The card's footprint says `sync.js` (**deleted**), with the `done_when:` grep
`grep -rn 'sync\.js' *.html` empty. Measured, `sync.js`'s 799 lines are three things, and
only one of them is the op-log:

| Bucket | Symbols | RxDB replaces it? |
|---|---|---|
| **OP-LOG / WS** | `LamportClock`, `LAMPORT_CLOCK`, `wsConnect`, `applyOp`, `submitOp`, `flashField`, `enqueueSyncToast`, `showSyncToast`, `renderFieldResponse`, `rerenderOpenChecklistAfterSave` | yes — this is the cutover's target |
| **SUBMISSION QUEUE** | `enqueueSubmission`, `drainQueue`, `renderSyncBanner`, `currentSubmitPeriod`, `idbGetAll`/`idbGet`/`idbPut`/`idbDelete` | **no** — whole-checklist submit with `idempotency_key`, guarded by decision 60 |
| **GENERAL INFRA** | **`api()` — 12 call sites in `workflows.html`**, `generateUUID`, `APP_TIMEZONE`, `appDateString`, `store`, `SAVE_DEBOUNCE`, `updateSaveStatus` | **no** — unrelated to sync |

Deleting the file deletes `workflows.html`'s only fetch wrapper, its offline submission
queue, and the app timezone that card A1-TZ standardised. Filed **B-69** as a footprint
correction exactly analogous to the backend one the slate already carries.

This is *not* on its own a park reason — the honest resolution is to split the file, delete
the op-log half outright and move the rest under a truthful name, which satisfies the grep
without a parallel run. I am recording it so the successor card sizes itself correctly
rather than discovering it mid-swap.

---

## 6. What else is not ready (measured, not inferred)

- **`sync-schema/sql/0001`–`0004` are applied by nothing outside the Go test harness.**
  `rowvisibility_rls_test.go:442-522` reads them off disk. `docker-compose.supabase.yml`
  mounts only `.night-crew/qa/spike-supabase/initdb/*`. No Taskfile target, no migration
  runner, no compose mount. The substrate schema is not deployed anywhere. Filed **B-67**.
- **No page creates the RxDB database or starts replication.** `bootstrap.js` is
  construction-only by its own header; `HQSync.startReplication` is exposed and nothing calls
  it.
- **The vendored push handler is real** (`vendor/rxdb.bundle.js`, `replicateSupabase`) — not
  a stub — but its 23505 recovery re-fetches by **primary key**, while the draft uniqueness
  is the partial index `(field_id, answered_by) where submission_id is null`. Two devices
  pushing the same draft under different client-minted `id`s would 23505 and then throw
  `doc not found`. Filed **B-66**.
- **No test anywhere writes through RxDB and asserts the row reached a Postgres.** The
  closest is Go and does not go through RxDB: `TestRowVisibilityRLS/WP3` raw-POSTs a draft
  through PostgREST. That proves the *door* accepts a write; it proves nothing about the
  store, the plugin, or the substrate→HQ direction.

---

## 7. Provenance of every first-hand claim

| Claim | How measured |
|---|---|
| E-KR2 evidence intact — **59 subtests**, `HQ_SYNC_SUBSTRATE_OPTIONAL` **unset** | `go test -count=1 -run TestRowVisibilityRLS -v ./internal/sync/` on this branch, `HQ_RLS_TEST_DB=hq_rls_s1b_impl`; `ok … 15.800s`, subtest lines counted = 59 |
| G1 clean at baseline | `go build ./...` + `go vet ./...` from `backend/`, both exit 0 |
| Two Postgres servers, no reverse path | `0001_sync_tables.sql:202-229`, `0002_hq_fdw.sql:6-13,194-261`, `0012_submission_responses.sql`, `0073`/`0074`; repo-wide grep for writeback/backfill/reverse-FDW → nothing |
| `/ops` routes all workflow validation | `backend/cmd/server/main.go:47-95` read in full |
| `autoSaveField` undefined | grep for `function autoSaveField` / `autoSaveField =` / `autoSaveField:` → zero; sole call at `workflows.html:2219` |
| `sync.js` bucket table | symbol-by-symbol reference map across `workflows.html`, other `*.html`, `tests/**`, `sync-rxdb/**` |
| Write RLS policies exist | `0004_write_policies.sql:388,393,417,422,486,491,496`; `checklist_templates` deliberately write-deny-all at `:361` |
| `sync-schema/sql` applied nowhere but tests | grep for the filenames → only `rowvisibility_rls_test.go:442-522` |
| Worktree unmodified | `git diff overnight-20260803..HEAD -- . ':!.night-crew'` empty |

---

## 8. What the successor card needs, and what it must not do

The blocker is **one operator decision**, routed in `DECISIONS-NEEDED.md` as **F-1**. Three
shapes, none of which a card may pick for itself:

- **(i) The substrate becomes the truth source for the four tables.** HQ's Go read paths
  (`/myChecklists`, `/myDrafts`, `/submit`, approvals, reports) repoint at it. Largest, and
  the only one that actually delivers "one write path". Note the standing cost already
  accepted in `0002`'s header — HQ is on the network path of every RLS row check, so this
  makes HQ and the substrate mutually dependent.
- **(ii) A substrate→HQ propagation path** — logical replication, a reconcile worker, or a
  writable reverse FDW. Re-opens decision 92's territory; the same-transaction version is
  already proven impossible, so any version of this is eventually-consistent by construction
  and needs its own conflict rule.
- **(iii) Narrow the cutover.** RxDB owns *reads* (the two list views and the fill view),
  HQ's REST path keeps owning *writes*. **This is the one that must be named explicitly as
  violating the card's constraint** — it is a parallel run, which P-KR3 forbids the build WOs
  from proposing. It is listed because it is a real option **for the operator**, not one a
  card may design into.

🛑 **A successor must not "just wire it up and see".** Starting replication against the
current topology produces a runner where every answer appears to save and none of them
submit — silent, and indistinguishable from working until a crew member's shift is lost.

---

## 9. Disposition of the six riders addressed to this card

| Rider | Disposition |
|---|---|
| **B-61** (list narrower at cutover) | **Escalated, not shipped.** Routed to `DECISIONS-NEEDED.md` as **F-2**. It is an operator product call by S1a's own filing and by decision 111's four-row bound; the cutover that would have made it visible did not happen, so nothing narrowed. Destination re-pointed to the successor card. |
| **B-62** (Realtime filter never proved live) | **Not provable tonight, and the reason is now firmer than when filed.** It is testable only once `HQ_SYNC_REST_URL` is set against a live substrate; that did not happen and must not, since F-1 is unresolved. Stays open, destination = successor card. |
| **B-63** (two concurrent replications) | **Not answered.** It is a page-lifecycle decision that only has meaning once the page depends on replication. Answering it now would be designing against a topology F-1 may change. Stays open, destination = successor card. |
| **B-64** (stale `bootstrap.js` banner) | **NOT fixed — deliberately.** The card owns the file, but B-64's own lead says *"do this in the same commit as B-63's decision; the comment cannot be correct until the lifecycle question it describes is answered."* B-63 is unanswered, so a fix would substitute one wrong banner for another. Stays open, destination = successor card. |
| **B-20** (`renderSyncBanner` badges Builder rows) | **Neither fixed-forward nor evaporated — it SURVIVES.** `sync.js:671` was not deleted, so the defect is exactly as filed. BACKLOG entry updated to record that S1b was its destination and parked without touching it; destination re-pointed to P3 `sync-banner-builder-tab-scope`, which is where it was already promoted and which does not depend on the cutover. |
| **B-54** (precache pin) | **NOT written.** The rider is "write the pin and re-base it in the same commit **if S1b moves the number**". S1b moved nothing — `build-sw.js` and `sw.js` are byte-unchanged and the count is still 31. Writing the pin here would be a real improvement but is unrelated to a parked cutover and would put a production-test edit in a park. Destination re-pointed to the successor card, decision 123's option (a) unchanged. |

## 10. KR consequences, stated plainly

- **P-KR3 — MET.** The no-parallel-run constraint is carried into this WO record verbatim
  (§11), and **0 build WOs proposed a parallel-run alternative**: the one parallel-run shape
  that exists is listed in §8 as an *operator* option, explicitly flagged as violating the
  constraint, and it is routed as a fork rather than designed into.
- **E-KR3 — NOT MET, and cannot be met from here.** The KR asks for one design note naming
  each offline data class and its single owner *after cutover*, cross-checked against the
  `build-sw.js`/RxDB-init diff. There is no cutover and no diff, so any note written tonight
  would describe an intention. Writing it would be the "narrated skip reads as a clean pass"
  failure. **E-KR3 goes to the successor card.**
- **E-KR2 — unaffected and re-measured green:** 59 subtests, substrate gate unset.
- **D-KR2** (deploy + returning-client screenshots) depended on S1b as its precondition. That
  precondition is unmet.

---

## 11. The P-KR3 constraint, carried verbatim into this WO record

> *"Hard swap, no parallel run — there is no need to keep the old system live during cutover."*
