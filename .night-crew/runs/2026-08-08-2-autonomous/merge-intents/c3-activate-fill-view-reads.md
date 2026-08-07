# Merge intent — C3 · `activate-fill-view-reads`

Run `20260808-2`, Activity 4. Branch `card/c3-activate-fill-view-reads`, based on the
run branch with C1 and C2 merged (`04c6703`).

Written and committed **before** implementation, per §15ad.65.

## Scope

The checklist **FILL view** — an open checklist's fields — reads from RxDB, scoped
per-open-checklist (T-29 decision 105). Plus the two C2 G6 findings this card inherited
as requirements (`c2-g6-review.md` F-1 and F-2).

🛑 **NOT the list views.** T-43(b) is binding: the My Checklists read path is
DELIBERATELY OPEN and no card may decide it. `renderMyChecklists()` /
`renderChecklistList()` / `loadMyChecklists()` / the Approvals tab are **byte-untouched**
by this card and all stay on REST. Nothing here writes into `MY_CHECKLISTS`,
`MY_SUBMISSIONS` or `PENDING`.

🛑 **Reads only** — decision 126 (ledger T-32). `debouncedSaveField` →
`submitOp('SET_FIELD')` → `POST /ops` is byte-untouched; there is no `autoSaveField`
(B-65). `POST /saveResponse` + `POST /submitChecklist` keep owning ALL writes.

🛑 **Spike E's condition (ledger T-42) holds unchanged.** This card starts no interval,
watches no business watermark and adds no explicit resync step. The checkpoint pulls on
the substrate's trigger-stamped `_modified`; the relay is spike C's LISTEN/NOTIFY one.
If the relay ever becomes a poller on a business watermark, an explicit resync step
comes back as a requirement of the fill view's call site too.

## Shared files touched — one line why each

| File | Why |
|---|---|
| `sync-rxdb/client.js` | **F-2.** `normalizeScope`'s FILL branch gains a REQUIRED `userId`, and `scopeIdentity()` carries it into the fingerprint → `replicationIdentifier` → RxDB's checkpoint key. No filter clause changes. |
| `sync-rxdb/bootstrap.js` | **F-1** (rejection eviction in `ensureDatabase()` / `openSyncScope()`, plus a `HQSync.createDatabase` seam so the failure can be forced) and **F-2** (`scopeKey()` gains the fill scope's `userId`). |
| `workflows.html` | The fill view's read path + open/cancel lifecycle (`HQFillSync`), the `RXDB_FILL_RESPONSES` overlay layer in `hydrateFieldState`, and the `applyResponseRow` extraction the overlay shares with the draft loop. C2's dev panel gains the now-required `hq_sync_user` scope param. |
| `tests/sync-rxdb-client.spec.js` | The two-concurrent-fill regression test (the card's named hard requirement) in the existing recorder, plus the F-2 red-first tests. Fill-scope fixtures gain `userId`. |
| `tests/sync-fill-view.spec.js` | **NEW.** Browser-side: F-1's eviction, the fill view's RxDB-served read and its flag-off vacuity pair, and the page-level concurrent-scope lifecycle. |
| `tests/sync-one-row.spec.js` | **Outside the declared footprint, unavoidable.** C2's three tests drive fill scopes; F-2 makes `userId` required, so they must pass one. Mechanical. |
| `night-crew.toml` + `tests/repo-hygiene.spec.js` | The Operations-seam roll-call goes **10 → 11** for the new spec. Same coupled guard C2 answered at 9 → 10; the spec drives `workflows.html` + `sync-rxdb/*`, so it belongs to the seam and naming it to dodge the `sync` token would hide it. |
| `.night-crew/knowledge/roadmap.md` | Activity 4 card flip, same change set. |

## What must survive any merge

1. **`sync-rxdb/client.js` — `userId` REQUIRED on the FILL scope, and in `scopeIdentity`.**
   Dropping it re-opens F-2: crew member B on a shared truck phone resumes crew member
   A's `_modified` cursor and sleeps through B's own older draft rows. This is the exact
   hazard SCOPE-03 already guards for the LIST scope; the fill scope now uses the same
   convention. It is a **narrowing** (more distinct checkpoints, each over a subset), not
   a widening — decision 105 is satisfied, not amended.
2. **`sync-rxdb/bootstrap.js` — the rejection eviction in `ensureDatabase()` and in
   `openSyncScope()`.** Without it one transient IndexedDB failure bricks sync for the
   page's lifetime and `openScopeKeys()` reports a dead scope as live (F-1).
3. **`workflows.html` — the flag gate on every fill-sync entry point.** `hq_sync_read`
   is OFF by default in every environment; with it off `RXDB_FILL_RESPONSES` stays empty,
   `hydrateFieldState`'s overlay loop iterates nothing, and no scope is ever opened. C1's
   B-88 guard (`window.HQSync.db` undefined + no rxdb IndexedDB at load with the flag
   off) MUST stay green.
4. **`workflows.html` — `hydrateFieldState`'s layer order and the rejection skip.** The
   RxDB overlay runs LAST but skips any field carrying a `REJECTION_FLAGS` entry, so a
   bounced-back field stays cleared and the crew still has to redo it.
5. **The persistence rule.** `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops`
   untouched; `tests/persistence.spec.js`'s back-and-reopen tests green.

### 🛑 Against S1's banner edits (`list-views-decision-recording`)

S1 is sequenced AFTER this card and owns `sync-rxdb/bootstrap.js`'s **B-64 scope banner**
and a `sync-rxdb/client.js` **cancel-rule banner**. This card deliberately does NOT
absorb either. If S1 lands after me and rewrites them:

**Banner lines that matter to me — S1 must keep their substance:**
- `bootstrap.js` — the **SHAPE block** (`ONE database, shared` / `ONE registry entry per
  SCOPE` / `handle.cancel()` cancels that scope only). S1's corrected wording *"cancel
  before re-scoping THE SAME shape"* is compatible with it and is the wording my
  lifecycle implements; what must not come back is the pre-B-63 *"cancel before
  re-scoping"* full stop, which would read as "the fill view may hold only one scope"
  and contradicts T-43(c).
- `bootstrap.js` — the **F-1 eviction comments** on `ensureDatabase()` / `openSyncScope()`
  and the **F-2 sentence** on `scopeKey()`. These are findings-of-record, not banner prose.
- `client.js` — `startHQReplication`'s docblock line *"🛑 CANCEL BEFORE RE-SCOPING"* is
  S1's to restate; the **FILL shape line listing `{userId, checklistId, templateId,
  fieldIds}`** is mine and must survive verbatim or F-2 silently reopens in the docs.

**`workflows.html` hunks that matter to me** (S1's footprint says it does not own this
file, so a conflict here is a mistake, not a merge):
- the `applyResponseRow` extraction + the layer-3 overlay inside `hydrateFieldState`;
- the `HQFillSync` controller block and its `FILL_SYNC_SCOPES` registry;
- the four call sites that open/close a fill scope (checklist open; `#fill-back`; the
  post-submit exits; `show(1)`);
- C2's dev panel's new `hq_sync_user` param — dropping it makes C2's own e2e red.

## What is safe to drop

- Every prose/comment block in `workflows.html`, `client.js` and `bootstrap.js` that is
  narrative rather than a finding-of-record — keep the 🛑 lines, the rest can be
  reflowed.
- `tests/sync-fill-view.spec.js`'s *page-level* concurrency test (the client-level one in
  `tests/sync-rxdb-client.spec.js` is the card's named requirement and is the one that
  must not be dropped).
- The `data-*` attributes on the fill view used only for test anchoring, if a later card
  finds a better anchor.

## Red-first

🛑 **SAME SPEC REVISION, BOTH LEGS.** The tests were committed FIRST, alone, at
`5929f0f` (`test(sync): RED-FIRST — the fill view's reads, F-1's eviction, F-2's user
identity`) with **no production code in that commit**. The red leg ran against that
commit's working tree; the green leg ran the byte-identical spec files with only
production code added on top (`61d13d0`). `git diff 5929f0f 61d13d0 -- tests/` shows
only the two mechanical `userId` additions to `tests/sync-one-row.spec.js` and
`tests/sync-rxdb-conflict.spec.js` that F-2's refusal forced — **neither file carries a
red-first assertion**. C2 was dinged for red-on-revision-A / green-on-revision-B; this is
the answer to that finding.

**RED — pre-change tree, at `5929f0f`:**

```
export PATH="/usr/local/go/bin:$PATH"
export TEST_PORT=4351 TEST_DB_NAME=hq_test_c3impl HQ_RLS_TEST_DB=hq_test_c3impl_rls
npx bddgen                                                   # BDDGEN_EXIT=0
npx playwright test tests/sync-rxdb-client.spec.js \
                    tests/sync-fill-view.spec.js \
                    tests/repo-hygiene.spec.js --retries=0
=> 8 failed, 63 passed (5.0m)   EXIT=1
```

Log: `c3-gates/rf-red.log` (whole log, one summary block). The eight:

| # | Red | The defect it shows |
|---|---|---|
| 1 | `[SCOPE-05] a SECOND crew member on the same phone gets their own identifiers` | **F-2.** `scopeIdentity()` was `fill:${checklistId}`; two crew members on one phone hashed to the same 8 identifiers instead of 16. |
| 2 | `[SCOPE-05] the FILL scope REFUSES a missing userId` | **F-2.** `normalizeScope` accepted a fill scope with no user. |
| 3 | `[SCOPE-05] the fill scope's userId is IDENTITY, not a filter clause` | **F-2.** Same fingerprint for two users — the assertion that proves the narrowing is real. |
| 4 | `[FILL-03] a rejected createDatabase is EVICTED` | **F-1.** The failure could not even be FORCED — no `createDatabase` seam — which is exactly why G6 shipped F-1 PLAUSIBLE rather than CONFIRMED. |
| 5 | `[FILL-01] a response that exists ONLY in RxDB answers the field in the runner` | The card's deliverable: the fill view had no RxDB read path at all. |
| 6 | `[FILL-01] with the flag OFF ... no scope, no database, no IndexedDB` | The vacuity half — `window.HQFillSync` did not exist. |
| 7 | `[FILL-02] setup + food prep live at once ... cancelling one leaves the other` | The page-level concurrent lifecycle did not exist. |
| 8 | `[FILL-02] the fill scope carries the crew member` | **F-2**, at the page level. |

🛑 **Stated honestly rather than dressed up:** the card's *named* hard requirement — the
two-concurrent-fill regression test in the existing recorder, identifiers pairwise
distinct — **PASSED on the pre-change tree**, as did its "neither starves the other"
sibling. That is correct and is not a hole: two concurrent fill scopes for ONE crew
member already minted distinct identifiers (SCOPE-02 landed that), so the test is a
**regression guard** for a property the operator's ruling now makes load-bearing, not a
defect red. The red half of the same requirement is red #1 — the shared-phone case,
which is where concurrency and F-2 meet and where the identifiers really did collide.

**GREEN — post-change tree, at `61d13d0`, same spec revision:**

```
npx bddgen                                                   # BDDGEN_EXIT=0
npx playwright test tests/sync-rxdb-client.spec.js \
                    tests/sync-fill-view.spec.js \
                    tests/sync-one-row.spec.js \
                    tests/sync-rxdb-conflict.spec.js \
                    tests/repo-hygiene.spec.js --retries=0
=> 114 passed (33.9s)   EXIT=0
```

Log: `c3-gates/rf-green.log`. The green leg deliberately ADDS C2's `sync-one-row.spec.js`
and `sync-rxdb-conflict.spec.js` — the two specs F-2's new refusal forced a change in —
so their passing is evidence in the same run rather than a claim.

### FIX ROUND — G6 findings F-A (TOP, CONFIRMED), F-B, F-C, F-D

Same discipline: tests committed **alone** at `276068b`, no production code in that
commit; fix at `3e4397d`. `git diff 276068b -- tests/sync-fill-view.spec.js` = **0 lines**.

**RED — pre-fix tree, at `276068b`:**

```
npx bddgen                                                   # BDDGEN_EXIT=0
npx playwright test tests/sync-fill-view.spec.js --retries=0
=> 3 failed, 6 passed (43.6s)   EXIT=1
```

Log: `c3-gates/fix-rf-red.log`. Measured, not argued:

| Red | Observed |
|---|---|
| `[FILL-04]` foreign-submission row + foreign-user draft | runner rendered **`2 of 2 items complete`** on a **blank** two-field checklist (expected `0 of 2`) — G6's F-A, reproduced |
| `[FILL-05]` tab switch closes the scope `[F-B]` | `HQFillSync.openIds().length` **1**, expected 0, after `#t2` |
| `[FILL-05]` fast reopen keeps a live replication `[F-C]` | `HQSync.openScopeKeys().length` **0** while `HQFillSync.openIds()` reported the scope live — the two surfaces disagreeing, exactly the predicted symptom |

**GREEN — post-fix tree, at `3e4397d`, byte-identical spec:**

```
npx playwright test tests/sync-fill-view.spec.js \
                    tests/sync-one-row.spec.js \
                    tests/sync-rxdb-client.spec.js --retries=0
=> 74 passed (34.2s)   EXIT=0
```

Log: `c3-gates/fix-rf-green.log`.

🛑 **F-C was testable without flaky timing** — the assertion is on the two inspection
surfaces agreeing, not on a sleep deciding the outcome. No timing waiver needed.

🛑 **The card's original "stated bound" is corrected in place, not replaced**
(`workflows.html`, above `FILL_SYNC_SCOPES`): it reasoned about which scopes are open
CONCURRENTLY, and the defect was SEQUENTIAL — the Dexie DB is persistent and `cancel()`
purges nothing (B-42), so yesterday's rows for the same daily recurring checklist are
still resident under the same field ids. **A bound on which scopes are live says nothing
about which rows are resident.**

Gate logs under `.night-crew/runs/2026-08-08-2-autonomous/c3-gates/`.

## Parks

None at the time of writing. The PARK trigger is narrow: concurrent-fill needing a
**substrate schema/policy change** (decision 111 authorises four rows; a fifth is the
operator's). F-2's resolution adds **no** column, **no** policy, **no** filter clause and
**no** table — `userId` appears in the scope's IDENTITY only, exactly as SCOPE-03 already
does for the LIST scope — so the trigger does not fire. If that changes mid-card, this
section is rewritten and the card stops.
