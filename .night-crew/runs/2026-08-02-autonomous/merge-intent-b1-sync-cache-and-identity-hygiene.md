# Merge intent — Card B1 `sync-cache-and-identity-hygiene`

Run: `overnight-20260802`
Branch: `card/b1-sync-cache-and-identity-hygiene`
Base: `overnight-20260802`

**What this card does, in one line:** implements ledger T-30 decision 112 — the
`api-cache` NetworkFirst route is **partitioned by identity** (it is NOT retired),
and the same identity token drives the purge of `localStorage['hq_apps']` and the
login-without-logout path. One mechanism, three call sites.

> 🛑 **This note is load-bearing tonight and on Night B.** `P1
> build-deploy-manifest-integrity` edits `build-sw.js` after this card TONIGHT, and
> `S1 sync-hard-cutover` edits it again TOMORROW. Neither has this context.
> Section 2 is the contract they are held to.

> 🔴 **CORRECTED BY THE FIX ROUND (same run, after G6 adversarial review).** The
> G6 reviewer returned APPROVE WITH FINDINGS and proved by mutation that **three
> claims in this note were false**: §2.2 misattributed which test defends the
> partition, §2.3 claimed a guard that does not exist, §2.4 claimed guards on the
> `login.html` half that do not exist, and §1's table under-reported the diff by
> two files. **The wrong sentences are struck in place, not deleted** — a reader
> on Night B needs to see that they were corrected rather than find them quietly
> gone. Everything struck below was wrong when written; everything beside it was
> verified by re-applying the reviewer's mutations and watching the new tests go
> red. See §6 for the mutation table.

---

## 1. Shared files touched

> 🔴 **CORRECTED.** ~~This table listed 8 files.~~ The diff touches **10**.
> `tests/index.spec.js` (+101 lines, [B1-XT-03] and [B1-XT-04]) and this note
> itself were missing. **A file the note does not name is a file a later merge
> can clobber unnoticed** — which is the whole job of this table. The fix round
> adds no new files; it edits `login.html`, `index.html`,
> `tests/sw-api-cache-partition.spec.js`, `tests/index.spec.js`, `sw.js`,
> `BACKLOG.md` and this note.

| File | Shared with | Why this card touches it |
|---|---|---|
| `build-sw.js` | **P1 tonight** (import-reachability check), **S1 tomorrow** (hard cutover) | The `api-cache` runtime route lives here. Decision 112's mechanism *is* two new plugin hooks on that route. **Nothing in the precache/glob/manifest half is touched** — no glob added, no glob removed, `globIgnores` untouched, `committedOnlyTransform` untouched, `GENERATED_BUT_SHIPPED` untouched. The diff is confined to the `runtimeCaching[0].options.plugins` array and the comment block above it. **Precached file COUNT is unchanged: 29, before and after. The KB moves (2111.1 -> 2116.2 -> 2118.2); the count is the invariant — see §2.6.** The fix round does not touch this file at all. |
| `sw.js` | P1, S1 (both regenerate it) | Generated artifact, committed by contract (Taskfile ships the committed `sw.js`; `build-sw.js` reads git HEAD). Regenerated AFTER the source commit, per G4. Any later card that edits `build-sw.js` must regenerate and re-commit `sw.js` in the same change set or the partition silently does not ship. |
| `index.html` | nobody tonight | Obligation 3 + 7(a): owns `checkAuth()`, `logout()` and the `hq_apps` read. Writes the identity token the service worker reads. 🔴 Fix round renamed `clearApiCache()` → `purgeDeviceIdentity()` (§2.4). |
| `login.html` | nobody tonight | Obligation 7(b): the identity change that never runs `logout()`. 🔴 Fix round made the purge conditional on an identity CHANGE (§2.4) and gated it with `[B1-XT-06]`/`[B1-XT-07]`/`[B1-XT-08]`. |
| `tests/sync.spec.js` | **P3 tonight** | Obligation 8 ONLY — a one-word comment correction at `:1584`. **No test in this file is added, removed, retitled or re-asserted.** |
| `tests/sw-api-cache-partition.spec.js` | new file, nobody | The red. The only spec in the suite that runs a REAL service worker. Holds `[B1-XT-01]`, `[B1-XT-02]` and (fix round) `[B1-XT-05]`. |
| `tests/index.spec.js` | **nobody tonight** | 🔴 **MISSING FROM THE ORIGINAL TABLE.** +101 lines at card time (`[B1-XT-03]`, `[B1-XT-04]`), +~110 more at the fix round (`[B1-XT-06]`, `[B1-XT-07]`, `[B1-XT-08]` and the shared `readDeviceState` / `seedPreviousUserDeviceState` / `freezeRedirectTarget` helpers). The page-code half of the mechanism is gated ENTIRELY from this file — a later card that reverts it loses every guard on `login.html`. Nothing pre-existing in this file is retitled or re-asserted **except `[B1-XT-03]`**, which the fix round renamed and un-vacuumed (see §2.4). |
| `.night-crew/knowledge/roadmap.md` | every card tonight | The milestone status flip for **this card's bullet only**. |
| `BACKLOG.md` | every card tonight | One discovery filed with a handle (§ scope freeze). |

### `tests/sync.spec.js` — what this card does NOT do

🛑 **`list page progress decrements when another device unchecks a field [LST-17]`
STAYS ARMED** (T-29 decision 109). It is not disarmed, not deleted, not retitled,
not "fixed". Note the bare tag `[LST-17]` matches **two** tests; neither is
touched. The only edit to this file is the word `'submitted'` → `'completed'` in a
`//` comment.

---

## 2. What must survive any merge

This is the section a later diff is checked against. Each item names the thing and
what its removal costs.

### 2.1 The identity token lives in a CacheStorage bucket, not localStorage

- **Bucket name `hq-identity`, entry URL `/__hq_identity`, body = the user's UUID
  string.** Written by `index.html`, read by BOTH `index.html` and **the service
  worker**.
- **Why it is not localStorage:** a service worker cannot see `localStorage`. It is
  not the session cookie either: the cookie is `HttpOnly`
  (`backend/internal/auth/handler.go:61`) and the `Cookie` header is attached by
  the network stack *after* the `fetch` event, so it is not on the request the
  plugin sees. `CacheStorage` is the only store both contexts can reach without a
  new IndexedDB module in the precache.
- **THREE files name this bucket, and a rename must change all three in one
  commit.** There is no shared constant — the pages are inline-script by
  convention and a new root `.js` would have collided with P1 (see 2.6):
  1. `build-sw.js` — literals `"hq-identity"` / `"/__hq_identity"` inside both
     plugin hooks. Also lands in the generated `sw.js`, which must be regenerated.
  2. `index.html` — `IDENTITY_CACHE_NAME` / `IDENTITY_TOKEN_URL` consts; the only
     WRITER.
  3. `login.html` — `purgeDeviceIdentity()`, **literal strings**, not the consts.
- **A partial rename fails SILENT, not loud.** A `build-sw.js` reading
  `hq-identity` while `index.html` writes somewhere else degrades every request to
  the `anon` partition — which under 2.3 means **nothing is cached at all** and the
  PWA quietly stops working offline. A `login.html` left behind stops purging and
  the disclosure comes back. 🔴 **CORRECTED:** ~~Neither reddens anything except
  `tests/sw-api-cache-partition.spec.js` and `[B1-XT-03]`.~~ `[B1-XT-03]` does
  **not** red on a stranded `login.html` — that was the same misattribution §2.4
  records. The guards that red are `[B1-XT-06]` / `[B1-XT-07]` (the `login.html`
  end) and `tests/sw-api-cache-partition.spec.js` (`[B1-XT-02]` on a key-shape
  mismatch, `[B1-XT-05]` on the degraded `anon` partition).

### 2.2 `cacheKeyWillBeUsed` on the `api-cache` route — THE partition

```js
cacheKeyWillBeUsed: async ({ request }) => {
  ... reads hq-identity ... u.searchParams.set('__hq_id', id) ... return new Request(u.href)
}
```

- **The cache key carries `__hq_id=<uuid>` as a query parameter.** That parameter
  IS the partition. Two users on one phone produce two disjoint key spaces for the
  same URL.
- 🔴 **CORRECTED — WHICH TEST DEFENDS THIS.** ~~Deleting this hook re-opens the
  cross-tenant read in full, with no other symptom … The only thing that notices
  is `tests/sw-api-cache-partition.spec.js`.~~ **Measured, not assumed:** the G6
  reviewer deleted `cacheKeyWillBeUsed` (mutation M1) and **`[B1-XT-01]` still
  passed at 503**. It has to: the §2.4 purges make A's entries *absent* before B
  can miss them, so the end-to-end disclosure test cannot see the partition go
  away. The hook **is** defended — by **`[B1-XT-02]`'s key-shape assertion,
  alone**. 🛑 **A later card that deletes this hook and sees `[B1-XT-01]` green
  may reasonably conclude the disclosure is still closed. It is not.** What is
  lost is the structural half: with the purges as the only mechanism, the
  disclosure is merely *absent*, recoverable by anything that skips or races a
  purge, instead of *unreadable by construction*. `[B1-XT-02]` is the single test
  standing between this card and that regression — treat it accordingly.
- **It must stay on the SAME route object as `cacheName: 'api-cache'`.** Moving it
  to a second route, or adding a second `/\/api\//` route above it, un-partitions
  whatever the new route matches first.
- `cacheKeyWillBeUsed` affects the **cache key only** — the network request still
  goes to the original URL. A later card must not "clean up" the `__hq_id`
  parameter thinking it is sent to the server. It is not.

### 2.3 `cacheWillUpdate` on the same route — no identity, no write

```js
cacheWillUpdate: async ({ response }) => (await identity()) ? response : null
```

- **An API response fetched while the device cannot name its user is not cached at
  all.** This closes the boot window between page load and the `/api/v1/me` answer,
  during which the token does not yet exist. Without it, that window writes an
  `anon` partition that every subsequent user shares — a smaller version of the
  same bug.
- 🔴 **CORRECTED — THIS WAS THE WORST CLAIM IN THE NOTE.** ~~Removing it does not
  break any test in the suite except `[B1-XT-02]`. It fails silently otherwise.~~
  **That was false. Removing it broke NOTHING.** The G6 reviewer deleted the
  entire hook (mutation M2), regenerated `sw.js`, and the suite went **13/13
  green**. `[B1-XT-02]` cannot catch it *structurally*: it waits for
  `hq-identity` to appear, **then** `caches.delete('api-cache')`, **then**
  fetches — so by construction every write it observes happens with an identity
  already present, and the boot window this hook exists to close is never
  entered. **This note asserted a guard that did not exist, on the load-bearing
  artifact P1 and S1 are held to.**
- ✅ **FIXED (fix round).** `[B1-XT-05]` —
  `with no device identity the worker writes NOTHING to api-cache`, in
  `tests/sw-api-cache-partition.spec.js` — deletes `hq-identity`, fetches
  `/api/v1/users`, and asserts no key carrying `__hq_id=anon` lands in
  `api-cache`. It runs a **positive control in the same test first** (with
  identity present, a write is observable in ~7–15 ms) so that "nothing is
  cached" is evidence rather than a write that has not landed yet, and it
  re-asserts that the token is still absent at assert time so the window is
  proven entered. **Re-applying M2 turns it red**, printing the leaked key:
  `["http://localhost:8314/api/v1/users?__hq_id=anon"]`. `[B1-XT-02]` stays green
  under M2 — the two are not substitutes.

### 2.4 The three purge call sites

| Call site | What it must do |
|---|---|
| `index.html` `purgeDeviceIdentity()` in `logout()` | delete `api-cache`, delete `hq-identity`, remove `localStorage['hq_apps']` — **awaited before `window.location.href`**, because the navigation tears the page down and a lost race leaves the previous user's rows on the phone. **Unconditional**: logout hands the device back with no owner, so there is no incoming identity to compare against. 🔴 Named `clearApiCache()` at card time — renamed by the fix round, see below. |
| `login.html` `purgeDeviceIdentity(nextId)` in `signIn()` **and** `acceptInvite()` | the same purge, **awaited before the redirect**. This is obligation 7(b) — B logging in while A's session is live never calls `logout()`. 🔴 **Conditional as of the fix round**: `nextId` is the `user.id` the auth endpoint returns, and an EXACT match against the stored token skips the purge (same crew member re-authenticating after a session expiry). Every other state — no `nextId`, no stored token, unreadable, different — purges. Fail closed. |
| `index.html` `establishIdentity(id)` | on a verified `/api/v1/me`: if the stored token differs from `id`, drop `hq_apps`; write the token; then **prune every `api-cache` entry whose `__hq_id` is not `id`**. The prune is the belt to 2.2's braces — it also clears anything a bypassed login left behind. |

**All three, not two.** The partition (2.2) makes a stale entry unreadable; the
purges make it absent. A later card that removes the purges leaves the disclosure
recoverable by anyone who can influence the token.

- 🔴 **CORRECTED — "all three" was a statement of INTENT, not of coverage.** The
  sentence above is still true about the mechanism, and it is what a later card
  must honour. What was false is the implication that the suite enforced it.
  ~~The `login.html` half is covered.~~ **It was covered by ZERO tests.** The G6
  reviewer removed `purgeDeviceIdentity()` from `signIn()` (M4) → **13/13
  green**; removed it from **both** call sites (M4b) → **13/13 green**.
  `[B1-XT-03]`, titled `signing in on login.html drops the previous user cached
  tile list`, **passed with `login.html` doing nothing at all**, because
  `index.html`'s `establishIdentity()` sees `prev !== id` and drops `hq_apps`
  itself. **The test was named for `login.html` and measured `index.html`.** It
  was *additionally* conditionally vacuous: its only assertion sat inside
  `if (afterRaw) { … }`, so an absent `hq_apps` asserted nothing and passed —
  the B-22/B-23/B-24 shape this spec file is careful to avoid everywhere else.
- ✅ **FIXED (fix round), four changes:**
  1. **`[B1-XT-06]`** `login.html signIn() purges the previous user device state
     before the redirect lands` — seeds a foreign identity token, one `api-cache`
     entry keyed to it and an `hq_apps` envelope; asserts that subject set is
     non-empty; **freezes the redirect target** (`page.route` on `/` returns an
     inert script-less document) so `index.html` CANNOT run; signs in; asserts
     `api-cache` and `hq-identity` are **absent** and `hq_apps` is `null`.
     Re-applying **M4** turns it red. Re-applying **M4b** turns it red.
  2. **`[B1-XT-07]`** the same, for `acceptInvite()` and its `/index.html`
     redirect. Needed because `[B1-XT-06]` alone does not red under a mutation
     that removes **only** the invite call site. **M4b** turns it red.
  3. **`[B1-XT-03]` renamed and un-vacuumed** →
     `a second user signing in never inherits the previous user cached tile
     list`. It is an honest end-to-end outcome test for the `index.html` half; it
     is **not** a guard on `login.html` and no longer claims to be. The
     `if (afterRaw)` wrapper is gone: the envelope must exist (the `.grid` wait
     only completes after `writeCachedApps`), must not be a bare array, and must
     carry B's uid.
  4. **`[B1-XT-08]`** `the same crew member re-authenticating keeps their offline
     dataset` — the guard on the new conditional. Breaking the comparison **shut**
     (always purge) reds `[B1-XT-08]`; breaking it **open** (never purge) reds
     `[B1-XT-06]`. The pair brackets it from both sides, both proven by mutation.
- 🛑 **WHAT THIS MECHANISM DOES AND DOES NOT DEFEND.** The G6 reviewer ran a
  probe and **all three attacks succeeded**: (i) the victim's uuid is *in the
  cache key*, enumerable by any page script via `caches.open('api-cache').keys()`;
  (ii) `match(<that key>)` returns another partition's full body directly —
  **CacheStorage is same-origin and JS-readable, so the partition is not a
  boundary against page script at all**; (iii) page JS can `put()` a victim uuid
  into `/__hq_identity` and the worker will serve that partition at 200.
  `workflows.html` loads SortableJS from a **CDN**, so "a script on the page" is
  not hypothetical here. **What this card defends is the shared-device,
  honest-user case** — B, using the app as intended on a phone A held earlier, is
  never served A's rows by the offline fallback. That is obligation 7 and it is
  the whole of it. XSS and hostile dependencies are a different threat with a
  different mitigation (CSP, self-hosting the CDN) and this card does not claim
  them. The in-code comment at `tests/sw-api-cache-partition.spec.js` used to
  overclaim exactly this ("cannot be read back under another one") and was
  rewritten by the fix round to match this paragraph.
- 🔴 **`clearApiCache()` → `purgeDeviceIdentity()` (`index.html`).** The old name
  described one of its **three** subjects; it also deletes `hq-identity` and
  `localStorage['hq_apps']`, and a later reader could reasonably have added a
  fourth key without touching it. Both files now use the same name, so
  `grep -rn purgeDeviceIdentity` finds **every** site that clears device state.
  The arity differs on purpose (see the table above).

### 2.5 `localStorage['hq_apps']` is an identity-stamped envelope

- Shape is now `{"uid":"<user uuid>","apps":[…]}` — **not** a bare array.
- A bare-array value (anything written by a build before this card) is **discarded,
  not migrated**: its owner is unknown, and obligation 7(a) is exactly "the previous
  user's cached slug list". Fail closed.
- It is applied only when `envelope.uid` equals the identity established on this
  device. Offline with no prior verified load ⇒ no cached tiles. That is
  deliberate, and it is what makes `index.html`'s fail-closed branch actually
  closed.

### 2.6 The precache half of `build-sw.js` is untouched — and must stay measurable

- **Expected precache count: 29 files, before AND after this card.** Derived by
  running `node build-sw.js` on the base commit before any edit (29 files /
  **2111.1 KB**) and again after the fix commit (29 files / **2116.2 KB**), and
  once more after the FIX ROUND (29 files / **2118.2 KB**). The count is the
  invariant; the 7.1 KB total is `index.html` and `login.html` growing by the
  identity helpers and the fix round's conditional purge, and nothing else. **No
  file is added or removed by either commit** — that is why the count must not
  move.
- 🛑 **The number to check is the COUNT, not the exit code.** `build-sw.js` exits 0
  while silently dropping assets — that is B-37, and it shipped a 24-file manifest
  for real on 2026-08-01. **P1 is the card that legitimately changes this
  number** (it adds the import-reachability check). If 29 moves under any other
  card, that is the silent-drop bug, not an improvement.
- No new shared `.js` file was created for the identity helpers — they are inline in
  `index.html` and `login.html`. A new root `.js` would need a `globPatterns` entry
  **and** a `backend/Dockerfile` copy (`tests/sw-manifest.spec.js` asserts the
  pairing), and would have collided with P1 on the same lines. Deliberate.

---

## 3. What is safe to drop

- **The `console.log` lines in `tests/sw-api-cache-partition.spec.js` and in
  `[B1-XT-06]`/`[B1-XT-07]`/`[B1-XT-08]`.** They exist to make the leaked payload
  and the surviving device state visible in the red output (guard-integrity bar
  B-22/B-23/B-24). They are evidence, not assertions. 🛑 **The positive control
  inside `[B1-XT-05]` is NOT in this category** — delete it and the test's
  negative assertion becomes indistinguishable from a write that has not landed
  yet, which is the vacuity the whole guard exists to avoid.
- **The `hq-identity` cache name itself** — any name works, as long as 2.1's "both
  ends move together" rule holds.
- **The `__hq_id` parameter NAME** — any name works, same rule. The *presence* of a
  per-identity discriminator in the cache key is what must survive; the spelling is
  not.
- **The comment correction in `tests/sync.spec.js`** (obligation 8) — if it
  conflicts with P3, take P3's version of the file and re-apply the one word. It
  carries no behaviour.

## 4. What is NOT in this card

- **`api-cache` is NOT retired.** Decision 112 struck that; the route covers all
  five tools and RxDB covers four `workflow` collections. A later card that deletes
  the route is reversing a signed decision.
- **No `openspec/` scaffolding.** The preflight verdict is `openspec: absent`.
  Nothing here creates it.
- **`HQ_SYNC_REST_URL` is not set anywhere by this card.** It stays armed.
- **No backend change.** Zero files under `backend/` are edited. The identity comes
  from the existing `GET /api/v1/me` payload's `id` field
  (`backend/internal/me/handler.go:25`), which already ships.
- **No deploy.** `task prod:deploy` was not run and is not an available action.

## 5. Empty fields

- **Migrations:** nothing here.
- **New dependencies:** nothing here.
- **Schema changes:** nothing here.
- **Operator forks raised:** nothing here — the card did not park.

---

## 6. The mutation table — what is actually defended, measured

Every row was run: mutation applied to the tree, `node build-sw.js` re-run where
`build-sw.js` was touched, suite executed, mutation reverted. **The "before"
column is the G6 reviewer's result on the card's tree; the "after" column is this
fix round's result on the same mutations.**

| # | Mutation | Before (G6) | After (fix round) |
|---|---|---|---|
| **M1** | delete `cacheKeyWillBeUsed` | `[B1-XT-02]` red; **`[B1-XT-01]` still green at 503** | unchanged — and §2.2 now says so instead of claiming the disclosure re-opens "in full" |
| **M2** | delete `cacheWillUpdate` whole | 🛑 **13/13 GREEN — nothing noticed** | **`[B1-XT-05]` RED**, printing `__hq_id=anon`; `[B1-XT-02]` stays green |
| **M4** | remove the purge from `signIn()` | 🛑 **13/13 GREEN** | **`[B1-XT-06]` RED**; `[B1-XT-03]` still green, which is exactly why `[B1-XT-06]` had to exist |
| **M4b** | remove **both** `login.html` purge call sites | 🛑 **13/13 GREEN** | **`[B1-XT-06]` and `[B1-XT-07]` both RED** |
| **M7** | remove **all three** purge sites | `[B1-XT-01]` still refuses at 503 — vindicates 2.2 as load-bearing on its own | unchanged; this is the card's central claim and it survived adversarial review |
| **M-NB7** | break the new same-identity check **shut** (always purge) | n/a — mechanism did not exist | **`[B1-XT-08]` RED**, `[B1-XT-06]` green: the pair brackets the comparison |

🛑 **The rule this table encodes, for P1 and S1:** a claim in this note that a
test defends a mechanism is only worth what a re-applied mutation says. Three of
this note's original claims did not survive that test. If you change any of the
four mechanism pieces, run the mutation, not the suite.

## 7. Filed, not built

- **`B-46` — the blank-launcher degradation.** With no token, `readCachedApps`
  returns `null`, `filterTilesByPermissions` never runs, and `.grid` stays at its
  inline `visibility:hidden`: a **permanently blank launcher, no message, no
  spinner, no affordance**. It self-heals on the next online `index.html` load,
  which is why it is not blocking. **Deliberately not built** — a UI for the
  fail-closed identity branch is a design decision under a UI-SPEC, not a fix
  round's call under the run's scope freeze. Destination: **next milestone**.
- **Backlog renumbering.** Three legs of this run each independently claimed
  `B-39`. The pre-step's `B-39`/`B-40`/`B-41` merged first and keep those
  numbers; A1 took `B-42`/`B-43`. This card's entries moved: the
  `tests/repro-cut-task.spec.js:169` sibling rot **`B-39` → `B-44`**, the
  Playwright baseline instability **`B-40` → `B-45`**, and the blank launcher is
  new as **`B-46`**. Each renumber carries its reason inline in `BACKLOG.md`.
- 🛑 **UNFILED, needs a number from the orchestrator — the Supabase substrate is
  NOT isolated by `HQ_RLS_TEST_DB`, and a concurrent leg's schema change reds a
  card that touched no Go at all.** This card's G2 (Go) leg failed on
  `TestRowVisibilityRLS/V18/submission_rejections_is_deny-all,_for_admins_too`.
  V18 asserts the **ABSENCE** of a select policy on `submission_rejections`
  (`sync-schema/sql/0003_rls_policies.sql` §2(b) — a recorded decision, not a
  gap). At assert time `pg_policies` on the shared `spike-supabase-db-1` showed
  `submission_rejections_select`, `_insert` and `_update` on `{authenticated}`,
  **none of which exists anywhere in this tree** (`grep -rn
  submission_rejections_select` → not present). They were created by the
  concurrently-running leg `a2-sync-rxdb-write-policies`, which is precisely the
  card B-38 identified as owning WITH CHECK policies for this table. **B1 touches
  ZERO files under `backend/` or `sync-schema/`** across the whole card
  (`git diff --name-only 5aca9cd~1..HEAD`), so it cannot be the cause. 26 of the
  27 `TestRowVisibilityRLS` subtests pass; the one failure is the absence
  assertion. **This is B-35's failure mode one layer up:** `HQ_RLS_TEST_DB`
  isolates only the HQ **FDW source** database. The Supabase side — the shared
  Postgres `public` schema where the policies and seeded rows live, plus the
  single shared PostgREST — has **no isolation variable at all**, and
  `tearDownRowVisRLS` only drops the three policies that existed when it was
  written. Concurrent dispatch is this project's normal shape, so this is the
  expected case, not the unlucky one. **Not filed with a number** because this
  leg was instructed to use only B-44/B-45/B-46; it needs one. Sibling of B-35
  and B-36.
- **`B-45` strengthened to a third corroboration.** B1's G6 produced a *third
  distinct* single-failure title — `tests/inventory.spec.js:2994 › Receipt sync
  button › manual sync chip shows Synced from {date} using lookback_days`, a 30s
  `networkidle` timeout that passes in 1.4s in isolation — with B-27 green. Three
  runs of one tree, three different single failures, none of them the documented
  baseline. **The baseline is a distribution, not a list**, and `B-45` now says
  so, with all three titles.
