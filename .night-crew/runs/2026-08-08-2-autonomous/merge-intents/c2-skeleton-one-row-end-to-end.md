# Merge intent — C2 · `skeleton-one-row-end-to-end`

Run `20260808-2` · branch `card/c2-skeleton-one-row-end-to-end` · Activity 3, the milestone's spine.
Base: `fdfd867` (run branch with C1 merged beneath).

---

## Scope

Thread ONE checklist row from HQ's real write path (`POST /api/v1/workflow/saveResponse`) to an
RxDB-served read on `/workflows.html`, behind an explicit flag that is **OFF by default**. This is
**the first production call site of `createHQSyncDatabase()` and `startHQReplication()` in this
repo's history** — until this card, both were exported, documented and driven only by tests.

Both list views (My Checklists, Approvals) and the fill view are **untouched and still on REST**.
The RxDB-served read is an isolated dev surface (`#sync-one-row`), reachable only with the flag
explicitly on and only when the URL names a scope.

### Decision 126 — cited verbatim, at the call site

> **Decision 126 (ledger T-32, 2026-08-02): the cutover splits reads from writes. RxDB serves
> reads; HQ's REST path keeps owning writes.** `POST /saveResponse` and `POST /submitChecklist`
> keep owning ALL writes.

This card is a **build** card and therefore may not propose that split; it carries it. The comment
block at the call site in `sync-rxdb/bootstrap.js` states it in those words and names the decision
number, so a future reader meets the constraint at the code rather than in a planning file.
`debouncedSaveField → submitOp('SET_FIELD') → POST /ops` is byte-untouched. There is no
`autoSaveField` (B-65). **This card changes READS only.**

### Decision 105 — the scope is per-open-checklist, never all collections at once

`openSyncScope()` refuses without a scope (`normalizeScope` throws) and there is no unscoped call
path. No widening: the two shapes `normalizeScope` already accepts are the two shapes this call
site accepts.

### Spike E's condition (ledger T-42) — carried verbatim, at the call site

> Trusted checkpoint catch-up with NO explicit resync step is valid **ONLY while the relay stays
> trigger/NOTIFY-driven** (spike C's mechanism). A polling relay on a business watermark
> reintroduces the missed-UPDATE hazard — `submitted_at` never advances after INSERT. If a
> polling relay is ever adopted, an explicit resync step comes back as a requirement.

This card contemplates **no polling**. It starts no watermark loop, adds no interval, and adds no
resync step — because the condition holds. The condition is written into the call site's comment so
that whoever changes the relay meets it.

---

## The flag mechanism (G6-F3: "the sync flag" named NOTHING before this card)

| | |
|---|---|
| **Name** | `hq_sync_read` — one string, exported as `SYNC_READ_FLAG` from `sync-rxdb/bootstrap.js` |
| **Location** | `sync-rxdb/bootstrap.js` (`SYNC_READ_FLAG`, `resolveSyncReadFlag()`); re-exposed on `window.HQSync.SYNC_READ_FLAG` / `window.HQSync.readEnabled` |
| **Storage** | `localStorage['hq_sync_read']`, value exactly `'on'` |
| **Override** | URL query `?hq_sync_read=on` turns it on **and persists**; `?hq_sync_read=off` clears it. So it is drivable from a phone with no devtools, which is what "explicit and discoverable" has to mean for a crew device |
| **Default** | **OFF.** Key absent ⇒ off. Any value other than the literal `'on'` ⇒ off. There is no environment, build or deploy in which it defaults on |
| **Resolved** | Once, synchronously, at module load — before anything can call the call site |

### How the flag-OFF path satisfies C1's guard

C1's guard (`tests/sync-rxdb-client.spec.js`, test *"nothing reads from RxDB on a code path that
can execute offline — window.HQSync.db is undefined at end of load (B-88)"*) requires that with the
flag off, page load leaves `window.HQSync.db` `undefined` **and** no `/rxdb|hq_sync/i` IndexedDB
database exists.

**`openSyncScope()` refuses SYNCHRONOUSLY when the flag is off — it throws before returning a
promise, before `createHQSyncDatabase()` is referenced, before any `await`.** This is deliberately
stronger than "the promise rejects late", and it is written to G6-F1's finding: C1's headline
`dbUndefined` assertion samples early and is timing-blind to async database creation, so the
flag-off path must never *begin* async database creation, not merely fail to finish it. There is no
code path in the tree on which the database's creation starts and is then abandoned.

Three independent things have to all be true before a database exists:
1. `HQSync.readEnabled === true` (the flag), checked first and synchronously;
2. `HQSync.client` constructed (it is, on every load — that part is C1-of-20260801's, unchanged);
3. someone calls `openSyncScope(scope)` with a scope `normalizeScope` accepts.

`workflows.html`'s dev surface calls it only when the flag is on **and** the URL carries
`hq_sync_checklist` + `hq_sync_template` + `hq_sync_field`. Nothing else in the tree calls it.

G6-F2 (the guard's IndexedDB scan only catches Dexie-backed storage) is honoured by omission:
**no memory-backed RxDB instance is introduced on any path.** `createHQSyncDatabase()` is called
with no `storage` override, so it uses `getRxStorageDexie()` — the storage the guard can see.

---

## Shared files touched

| File | Why |
|---|---|
| `sync-rxdb/bootstrap.js` | **Owned.** The flag (`SYNC_READ_FLAG`, `resolveSyncReadFlag`) and the call site (`openSyncScope`, `ensureDatabase`, the scope registry). |
| `workflows.html` | **Owned.** One `<div id="sync-one-row" hidden>` before `</body>` and one `<script type="module">` that mounts it. No existing element, handler or render function is modified. |
| `tests/sync-one-row.spec.js` | **New, owned.** The end-to-end test, the flag-off vacuity check, and the concurrent-scope call-site contract. |
| `night-crew.toml` | **Comment only.** The `[e2e.seams]` roll-call line — the machine-checked list of spec files the four Operations tokens select — gains `sync-one-row.spec.js`. **No new key, no new token, no value changed.** See the note below. |
| `tests/repo-hygiene.spec.js` | The hard-coded `expect(actual.length).toBe(9)` that guards that roll-call becomes `10`. |
| `.night-crew/knowledge/roadmap.md` | Card status flip for `skeleton-one-row-end-to-end`, same change set (universal mechanics). |

`sync-rxdb/client.js` was **read** (`normalizeScope`, `startHQReplication`, `createHQSyncDatabase`,
`scopePlanFor`) and **not edited** — the call site needed no new seam. The slate listed it as
"touches"; it turned out not to need touching, which is stated rather than left to a reader's
inference.

Nothing else. No backend Go file, no other HTML tool page, no `docker-compose*.yml`, no
`Taskfile.yml`, no `sync-schema/`, no `vendor/`, no version bump (not this card's job).

### 🛑 The `night-crew.toml` edit is NOT the PARK trigger, and here is why

This card's PARK note fires on "a new `night-crew.toml` key". **No key was added, no token was
added, no token's value changed.** What changed is a `#   selects:` comment line, and changing it
is the *designed* response to what happened: `tests/repo-hygiene.spec.js` asserts that comment
against `ls tests/` and states its own intent — *"adding a spec whose name contains a seam token
reds this test rather than quietly inflating a 'confined' gate."* The new spec's name contains
`sync`, so the guard fired exactly as built, and updating the roll-call is what it asks for.

The alternative — naming the spec so it matches no seam token — was rejected: it would hide a spec
that genuinely guards `workflows.html` + `sync-rxdb/*` from the seam that selects them, which is
the coverage hole the seam exists to prevent.

**Cost, stated because the slate prices legs off this list:** an Operations-confined card now pays
for **10** spec files, not 9. Per B-87 the CLI filters are OR'd, so this is over-selection (cost),
never mis-selection (coverage).

---

## What must survive any merge

- **The flag's name and default.** `hq_sync_read`, off unless the value is exactly `'on'`. A merge
  that changes the spelling must change `tests/sync-one-row.spec.js`'s `FLAG` constant with it —
  the constant is spelled out in the test on purpose so a rename reds rather than silently
  disabling the skeleton.
- **The synchronous refusal.** `openSyncScope()` must keep throwing *before* any `await` when the
  flag is off. Making it `async` with the check inside the body would still reject, and would still
  pass a naive reading of C1's guard, and would be wrong (G6-F1).
- **`HQSync.db` is assigned only inside `ensureDatabase()`**, which only `openSyncScope()` reaches.
- **The call-site shape C3 builds on** (below) — the scope registry, the per-scope handle, and
  `cancel()`.
- **The three tests in `tests/sync-one-row.spec.js`.** In particular the flag-off test is the
  vacuity check for C1's guard and must not be folded away as a duplicate of it: C1 asserts the OFF
  state, this file asserts OFF *and* ON on one tree, which is what makes OFF meaningful.
- **The decision-126 / decision-105 / spike-E citations at the call site.** They are the card's
  contract, not decoration.

## What C3 (`activate-fill-view-reads`) may build on

`window.HQSync.openSyncScope(scope)` — and it is shaped for T-43(c) (*crew members work multiple
checklists concurrently; multiple live per-checklist fill replications at once ARE the design*):

```js
const handle = await window.HQSync.openSyncScope({ checklistId, templateId, fieldIds });
// handle = { key, scope, db, states, cancel() }
```

- **Multiple DIFFERENT scopes are live simultaneously.** Each gets its own entry in the registry,
  its own `startHQReplication` states and — because `replicationIdentifier` already carries the
  scope fingerprint — its own per-identifier checkpoint. Asserted pairwise-distinct in the third
  test.
- **The SAME scope twice returns the SAME handle.** One replication per shape, not two
  subscriptions on one topic. This is the mechanical half of B-63/B-64's *"cancel before re-scoping
  THE SAME shape"* — S1 restates the banner; this makes re-opening the same shape harmless.
- **ONE shared database** across all scopes (`ensureDatabase()` memoises the promise, so concurrent
  first calls do not race two `createRxDatabase` calls). `handle.db` is that database.
- **`handle.cancel()`** cancels that scope's replication states and drops it from the registry;
  other scopes keep running. `window.HQSync.openScopeKeys()` is the inspection hook.

C3 does **not** need to change this call site to add the fill view. It needs to decide *when* to
open and cancel — the page lifecycle — which is C3's job and deliberately not done here.

## What is safe to drop

- Every prose comment explaining *why* (informative, not load-bearing) — except the three citations
  named above.
- The dev surface's inline CSS.
- The gate logs under `.night-crew/runs/2026-08-08-2-autonomous/c2-gates/` — evidence only,
  regenerable from the commands recorded in them.

---

## Red-first

The card's own new end-to-end test, run against the **pre-change tree** (HEAD `fdfd867`, production
files byte-unmodified — only `tests/sync-one-row.spec.js` present in the working tree).

```
TEST_PORT=4331 TEST_DB_NAME=hq_test_c2impl HQ_RLS_TEST_DB=hq_test_c2impl_rls \
  npx playwright test tests/sync-one-row.spec.js --project=chromium --retries=0
...
  3 failed
EXIT=1
```

Log: `.night-crew/runs/2026-08-08-2-autonomous/c2-gates/rf-red-prechange.log`.

**The red is in the right place, and that is the point.** Test 1 ran the entire harness green and
failed only on the missing production surface:

```
1) tests/sync-one-row.spec.js:179 › a value written by POST /saveResponse is served to the page
   out of RxDB, behind the flag
   Error: expect(locator).toBeVisible() failed
   Locator: locator('#sync-one-row')
   Expected: visible
   Error: element(s) not found
     > 220 |     await expect(surface).toBeVisible({ timeout: 30000 });
```

Everything before line 220 passed on the unmodified tree — the template creation, the real
`POST /saveResponse`, the submit that moves the draft onto a submission, the `myChecklists`
read-back, **and the psql read of `submission_responses` asserting exactly one row whose `value` is
`true` and whose `submission_id` is the real checklist id.** So the harness's own legs are proven
independently of the fix; what was missing was only the thing this card builds.

Tests 2 and 3 red for the two other halves:

```
2) ... the flag is OFF by default ...
   expect(received).toBe(expected)   Expected: false   Received: undefined
     > 268 |     expect(state.readEnabled).toBe(false);

3) ... the call site holds multiple concurrent per-checklist scopes ...
   TimeoutError: page.waitForFunction: Timeout 15000ms exceeded.
     > 285 |     await page.waitForFunction(() => window.HQSync.readEnabled === true, ...)
```

`window.HQSync.readEnabled` is `undefined` on the pre-change tree because **no flag existed**
(G6-F3: "the sync flag" named nothing in this repo).

**GREEN after the fix**, same command, final tree (`ba464c1`):

```
TEST_PORT=4331 TEST_DB_NAME=hq_test_c2impl HQ_RLS_TEST_DB=hq_test_c2impl_rls \
  npx playwright test tests/sync-one-row.spec.js --project=chromium --retries=0
  ✓ a value written by POST /saveResponse is served to the page out of RxDB, behind the flag
  ✓ the flag is OFF by default — no database, no IndexedDB, and the surface stays absent
  ✓ the call site holds multiple concurrent per-checklist scopes, and is idempotent per scope
  3 passed (11.9s)
EXIT=0
```

Log: `c2-gates/rf-green-postchange.log`.

---

## 🛑 THE FIRST FULL SUITE WENT RED ON THIS CARD'S OWN DEFECT — B-70, recurring

Named here rather than folded into a "green" line, because it is the more useful fact about this
card than any of the gates that passed.

Leg 1 of G2 (Playwright), against `dc6e43a`, came back **1 failed** and the failure was mine:

```
✘  357 [chromium] › tests/repo-hygiene.spec.js:41:1 ›
       no source file under sync-rxdb/ contains a NUL byte
```

`scopeKey()` in `sync-rxdb/bootstrap.js` had been written with two **literal `U+0000` bytes** as
its field delimiter, at offsets 14032 and 14123. That is exactly **B-70**, in a new file: a raw NUL
puts GNU grep into binary mode over the whole file, so `grep -n 'export' <file>` prints nothing and
exits 1 on a file full of matches — which is what makes every `done_when: "grep returns nothing"`
criterion unreliable **in the passing direction**. `client.js`'s `scopeFingerprint` carried the same
byte and was fixed the same way on `repo-hygiene-preconditions` (run 20260806).

Fixed in `42e547c` to the **escape sequence `\0`** — the same byte at runtime, so no scope key and
no `replicationIdentifier` changes, and the file is 7-bit clean. A comment at `scopeKey()` names the
guard and says not to tidy it back.

**The guard earned itself on the milestone's spine card, on the first suite that could catch it.**

---

## What this card does NOT prove, stated so nobody inherits a false claim

The **substrate half** of the chain — HQ Postgres → LISTEN/NOTIFY relay → Supabase substrate →
PostgREST — is spike C's mechanism, proven by
`.night-crew/qa/spike-supabase/spike-c-roundtrip.sh` against real containers, and it is **not
production code**. `HQ_SYNC_REST_URL` remains unset in every environment and the door still answers
503, exactly as before this card.

`tests/sync-one-row.spec.js` therefore stubs that leg at the **transport only** (`page.route` over
`/sync/rest/**`). The stub is narrow and the row it serves is **not invented**: it is read back out
of HQ's own `submission_responses` with psql after `/saveResponse` wrote it, and re-shaped into the
sync wire contract (`_modified`, `_deleted`, flat body) exactly as `sql/spike-c-relay-trigger.sql`
does. Everything downstream of the stub is unmocked production code — the vendored supabase-js
client, `makeSyncFetch`'s rewriting and credential stripping, `startHQReplication`'s scope plan and
identifier, RxDB's pull handler, the Dexie/IndexedDB write, and the page's read.

**Running the real three-service stack end to end is `demo-sync-target` (S2) — `task demo:sync`,
the milestone's close bar.** That is by design: S2's roadmap entry names C2's skeleton as its
subject. This card ships the skeleton S2 drives.

🛑 The test's psql read uses `resolveE2eDb()` from `scripts/reset-e2e-db.js` — the one place e2e
Postgres coordinates are computed — so it resolves **:5434 / `hqtest` / `yumyums-test-pg`**. It
issues `SELECT` only: no DDL, no writes. Nothing in this card, at any point, contacted :5433.

---

## Gate results

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **exit 0**, `go vet ./...` **exit 0**, both from `backend/` (the module root — never the repo root, which matches no module and prints a false green). Log: `c2-gates/g1-build-vet.log` |
| **G2 (Go)** | **exit 0.** 9 packages `ok`, **0 `FAIL`**, **454 `--- PASS:` lines**, 0 `--- FAIL:`, 2 `--- SKIP:` (`TestProxyLive_RealtimeUpgrade`, `TestProxyLive_RESTRequest` — the two live-proof skips). `internal/workflow` = **exactly 35**. Per package: alerts 3, auth 18, inventory 72, purchasing 25, receipt 73, recipes 61, sync 156, toast 11, workflow 35. 🛑 **`HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` were both UNSET** — attested by `env \| grep '^HQ_SYNC_'` printing nothing at the head of the log, not by "I didn't set it". DB `hq_test_c2impl_go` on **:5434** (`yumyums-test-pg`, role `hqtest`). Log: `c2-gates/g2-go.log` |
| **G2 (Playwright)** | De-confined to the **full suite** (this card touches `workflows.html`). `npx bddgen` first, exit 0 (B-165). **Two legs, ONE summary block each** (`grep -c '^Running '` = 1 per log), 802 tests each — 799 before this card plus its 3. **Leg 1 (`dc6e43a`): 795 passed / 1 failed / 6 skipped, 23.0m** — the failure is the B-70 NUL defect above. **Leg 2, final tree (`ba464c1`): 795 passed / 1 failed / 6 skipped, 22.4m.** See the red accounting below |
| **G3** | **N/A** — `openspec: absent`, ledger T-34 decision 140. No scaffolding created |
| **G4** | `node build-sw.js` **exit 0**, run **after** each commit (it reads git HEAD). **Idempotent — byte-identical `sw.js` across consecutive runs** (`md5sum` 7b3cc7d1… both times), tree clean on the repeat. **Precache count 31 — UNCHANGED, and deliberately so:** this card adds no precached asset (the dev surface is an inline `<script type="module">` in `workflows.html`; `sync-rxdb/bootstrap.js` was already precached), only their content hashes moved. Verified in the artifact itself, not just the tool's own line: 31 `url:` keys and 31 `revision:` keys in `sw.js`. Import reachability: 18 precached files parsed, 31 local references resolved, **0 outside the precache**. Log: `c2-gates/g4-build-sw.log` |
| **RF** | Red before, green after — above. Logs: `c2-gates/rf-red-prechange.log`, `c2-gates/rf-green-postchange.log` |

### The armed-reds baseline: all three PASSED, on BOTH legs

`tests/inventory.spec.js:883` (B-27), `tests/sync.spec.js:446` (LST-17) and
`tests/receipt-carousel.spec.js:123` (B-162) were **`✓` in both full-suite logs** (leg 1 lines 229 /
1317 / 724; leg 2 lines 226 / 1305 / 707). None of them occurred, so none of them is doing any work
to explain either leg's failure. **They stay armed** — an armed red is retired by diagnosis, never
by passing (decision 100; T-31 decision 120; T-42 reaffirmed all three after the same flip-flop).

### The one non-armed red on the final leg — investigated, did not reproduce

```
✘  416 [chromium] › tests/sw-api-cache-partition.spec.js:92:1 ›
       offline, user B is served user A cached /api/v1/users roster from api-cache [B1-XT-01]

    expect(received).toBe(expected)   Expected: true   Received: false
    > 118 |   expect(keysAfterA.some(u => u.includes('/api/v1/users'))).toBe(true);
```

**Could this card's diff plausibly cause it?** No, on three independent grounds.

1. **It PASSED on leg 1**, at the identical position (416), on a tree differing only by the `\0`
   delimiter fix — a change with no runtime effect at all (same byte). Log line 998 of
   `c2-gates/g2-pw-full.log`: `✓ 416 … [B1-XT-01] (6.4s)`.
2. **Subject matter is disjoint.** The test drives `users.html` and the service worker's *runtime*
   `api-cache` NetworkFirst route. This card changed `sync-rxdb/bootstrap.js`, the bottom of
   `workflows.html`, tests, a `night-crew.toml` comment, and `sw.js`'s **precache hashes** — the
   runtime api-cache route is not derived from the precache manifest and is byte-unchanged.
3. **The assertion that failed is a timing one, not a content one.** Line 118 asserts the roster
   URL is *already present* in the cache; the SW's runtime cache write had not landed yet. Nothing
   about the cached content was wrong.

**Reproduced?** No. Re-run isolated, `tests/`-anchored (B-87), same isolation env, `--retries=0`,
**twice**:

```
npx playwright test tests/sw-api-cache-partition.spec.js --project=chromium --retries=0
  3 passed (23.7s)   RUN1_EXIT=0
  3 passed (23.4s)   RUN2_EXIT=0
```

Log: `c2-gates/rerun-suspect-red.log`. **Ruled flake-trail evidence, not a card failure** — and
named here rather than dropped, so it can be re-litigated. It is a **new** entry in the flake trail
(not B-27/B-30/B-32/B-162's family, which are inventory/receipt-side); if it recurs it should be
filed and armed by diagnosis, per decision 100.

## Nothing else

No PARK condition was hit — no scope widening beyond decisions 105/126, no new `night-crew.toml`
key, and **nothing here decides the My Checklists read path** (operator kept it OPEN, T-43(b)).
The RxDB-served read is a dev surface with its own element id, its own script block and its own URL
scope; My Checklists renders exactly what it rendered before this card, from REST. The Approvals
tab stays on re-fetch, per T-43(a).

No backend Go change. No API contract change. No `docker-compose*.yml`, no `Taskfile.yml`, no
version bump. `main`, `dev`, and every other card's worktree/branch — in particular
`card/a3-rls-fixture-own`, untouchable per decision 155 — are untouched.
