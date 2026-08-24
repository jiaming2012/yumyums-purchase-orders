# G6 Review — Card C2 `skeleton-one-row-end-to-end` (run 20260808-2)

Fresh-context adversarial reviewer; inputs were the slate entry, the diff on
`card/c2-skeleton-one-row-end-to-end` (base fdfd867), and the committed evidence only.
Worktree left byte-identical; probe DBs dropped; nothing touched :5433.

## VERDICT: PASS-with-findings

No binding-constraint violation found. The flag is genuinely default-off on every path
constructed, the flag-off refusal is genuinely synchronous, decision 126/105 and spike
E's condition are carried verbatim and cited at the call site, T-43(b) is respected, and
the gate evidence survived audit including deliberate re-counting. Four findings, none
TOP; two are C3-facing and C3 must read them before building on this call site.

## Findings, ranked

**F-1 (moderate, C3-facing, PLAUSIBLE — code-level, could not force at runtime):**
`ensureDatabase()` memoises `databasePromise` and `openSyncScope()` registers the handle
promise in `openScopes` before first await, with no eviction on rejection in either
place. If `createHQSyncDatabase()` ever rejects (IndexedDB quota, corrupt Dexie store),
the rejection is permanent for the page's lifetime: every later `openSyncScope()` returns
the same stale rejection, `openScopeKeys()` reports a dead scope as live, and
`handle.cancel()` is unreachable. Nothing for the dev skeleton; for C3's fill view one
transient storage failure bricks sync until reload. Forcing attempt (undefine
window.indexedDB) resolved anyway — Dexie holds its own reference — hence PLAUSIBLE,
not CONFIRMED.

**F-2 (moderate, C3-facing, PLAUSIBLE — pre-existing shape made reachable by C2):**
The LIST scope requires `userId` ("two crew members on one truck phone do not inherit
each other's checkpoint", SCOPE-03, tests/sync-rxdb-client.spec.js:965). The FILL scope
(`{checklistId, templateId, fieldIds}`, sync-rxdb/client.js:612-684) has NO user
dimension; its draft branch relies on server-side RLS to mean "MY draft". The persisted
Dexie checkpoint therefore has no user in its key: crew member B on a device A used
resumes from A's `_modified` cursor and can sleep through B's own older draft rows —
the exact hazard SCOPE-03 guards for LIST. `normalizeScope` is unchanged by this diff,
but C2 is the first production call site persisting a fill checkpoint. C3 must add the
user to the fill scope, clear sync state on identity change, or record a decision that
shared-device fill is out of scope.

**F-3 (minor, evidence honesty):** The RF red leg ran a ~7-line-earlier revision of the
spec than committed (line numbers 179/248/280 in the log vs 186/255/287 committed) —
red(revision A)/green(revision B), stated nowhere. Fully mitigated in practice: the
committed spec passed on the final tree in full-suite leg 2 and in G6's own re-run.

**F-4 (observation, deliberate and documented):** `?hq_sync_read=on` persists via
localStorage until an explicit `=off` visit — the accidental-enable vector the day the
sync door opens. Fail-safe bias is right: any non-`'on'` value clears (probe-verified).

## Re-ran vs audited

Re-ran (TEST_PORT=4341, hq_test_c2g6 on :5434): bddgen + sync-one-row.spec.js +
sync-rxdb-client.spec.js in one run — 59/59 passed exit 0 (includes all 3 C2 tests,
C1's B-88 guard, SCOPE-01..04); adversarial probe spec (synchronous flag-off throw with
no IndexedDB after 1.5s grace; hostile flag values all resolve OFF; F-1 forcing attempt);
NUL forensics (parent 2 raw NULs, HEAD 0, `'\0'` byte-identical); night-crew.toml
mechanical diff (0 non-comment lines). Audited: both full-suite logs, g2-go.log,
g1-build-vet.log, both G4 logs, rerun-suspect-red.log, both RF logs, merge-intent.

## Answers (abridged)

1. Flag default-off on every constructed path; resolved once synchronously at module
   load; flag-off throws synchronously before any async db creation; B-88 guard green.
2. Decision 126 held: no write path moved; 126 cited verbatim at the call site; push
   replication carries nothing.
3. T-43(b) respected: workflows.html diff purely additive at end of file; all three
   product views byte-untouched and on REST.
4. Spike E: no polling anywhere; checkpoint pulls on trigger-stamped `_modified`;
   condition carried verbatim at the call site.
5. End-to-end honest: real POST /saveResponse with real session; psql read-back from
   submission_responses on :5434 feeds a transport-only stub of /sync/rest/**;
   production code below the stub; RF red only at the missing surface (caveat F-3).
6. night-crew.toml comment-only, mechanically verified; repo-hygiene count 9→10 is the
   coupled guard working as designed.
7. B-70 fix byte-identical at runtime; scope keys/replication identifiers unchanged.
8. All gate-log claims verified: G2 Go 9 ok / 454 PASS / workflow=35 / env attested
   UNSET; G2 PW one summary block per leg, 795/1/6 both legs, armed baseline passing at
   the cited lines; leg-1 red was the card's own NUL defect (fixed 42e547c); leg-2 red
   (sw-api-cache-partition:92) supported as flake by rerun-suspect-red.log (3/3 × 2,
   --retries=0); G4 precache 31, idempotent, md5-equal.
9. Concurrency proven by committed test 3 (re-run green): two scopes live
   simultaneously, same scope → same handle (promise identity), cancel isolates. C3
   inherits F-1 and F-2.

## Orchestrator disposition

Merged without a fix round — no acceptance defect. F-1 and F-2 carried verbatim into
C3's dispatch prompt as requirements; F-2's escalation path (substrate schema/policy
change needed → PARK per decision 111) stated there. Recorded in
conflicts-20260808-2.md.
