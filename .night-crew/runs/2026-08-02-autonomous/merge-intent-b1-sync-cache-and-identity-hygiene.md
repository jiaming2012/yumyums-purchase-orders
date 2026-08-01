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

---

## 1. Shared files touched

| File | Shared with | Why this card touches it |
|---|---|---|
| `build-sw.js` | **P1 tonight** (import-reachability check), **S1 tomorrow** (hard cutover) | The `api-cache` runtime route lives here. Decision 112's mechanism *is* two new plugin hooks on that route. **Nothing in the precache/glob/manifest half is touched** — no glob added, no glob removed, `globIgnores` untouched, `committedOnlyTransform` untouched, `GENERATED_BUT_SHIPPED` untouched. The diff is confined to the `runtimeCaching[0].options.plugins` array and the comment block above it. **Precached file count is unchanged: 29 files / 2111.1 KB before and after.** |
| `sw.js` | P1, S1 (both regenerate it) | Generated artifact, committed by contract (Taskfile ships the committed `sw.js`; `build-sw.js` reads git HEAD). Regenerated AFTER the source commit, per G4. Any later card that edits `build-sw.js` must regenerate and re-commit `sw.js` in the same change set or the partition silently does not ship. |
| `index.html` | nobody tonight | Obligation 3 + 7(a): owns `checkAuth()`, `logout()` and the `hq_apps` read. Writes the identity token the service worker reads. |
| `login.html` | nobody tonight | Obligation 7(b): the identity change that never runs `logout()`. |
| `tests/sync.spec.js` | **P3 tonight** | Obligation 8 ONLY — a one-word comment correction at `:1584`. **No test in this file is added, removed, retitled or re-asserted.** |
| `tests/sw-api-cache-partition.spec.js` | new file, nobody | The red. The only spec in the suite that runs a REAL service worker. |
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
- **If a later card moves this token anywhere else, it must move BOTH ends in the
  same commit.** A `build-sw.js` that reads `hq-identity` while `index.html` writes
  somewhere else silently degrades every request to the `anon` partition — which
  under 2.3 means **nothing is cached at all** and the PWA stops working offline.
  It fails quiet, not loud.

### 2.2 `cacheKeyWillBeUsed` on the `api-cache` route — THE partition

```js
cacheKeyWillBeUsed: async ({ request }) => {
  ... reads hq-identity ... u.searchParams.set('__hq_id', id) ... return new Request(u.href)
}
```

- **The cache key carries `__hq_id=<uuid>` as a query parameter.** That parameter
  IS the partition. Two users on one phone produce two disjoint key spaces for the
  same URL.
- **Deleting this hook re-opens the cross-tenant read in full**, with no other
  symptom: the suite still installs, `build-sw.js` still exits 0, the precache count
  is still 29. The only thing that notices is
  `tests/sw-api-cache-partition.spec.js`.
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
- Removing it does not break any test in the suite except
  `[B1-XT-02]`. It fails silently otherwise.

### 2.4 The three purge call sites

| Call site | What it must do |
|---|---|
| `index.html` `logout()` | delete `api-cache`, delete `hq-identity`, remove `localStorage['hq_apps']` — **awaited before `window.location.href`**, because the navigation tears the page down and a lost race leaves the previous user's rows on the phone. |
| `login.html` `signIn()` **and** `acceptInvite()` | the same purge, **awaited before the redirect**. This is obligation 7(b) — B logging in while A's session is live never calls `logout()`. |
| `index.html` `establishIdentity(id)` | on a verified `/api/v1/me`: if the stored token differs from `id`, drop `hq_apps`; write the token; then **prune every `api-cache` entry whose `__hq_id` is not `id`**. The prune is the belt to 2.2's braces — it also clears anything a bypassed login left behind. |

**All three, not two.** The partition (2.2) makes a stale entry unreadable; the
purges make it absent. A later card that removes the purges leaves the disclosure
recoverable by anyone who can influence the token.

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

- **Expected precache count: 29 files / 2111.1 KB.** Derived by running
  `node build-sw.js` on the base commit before any edit, and again after. This card
  adds no file to the precache and removes none. **P1's B-37 work is the card that
  changes this number; if it changes under any other card, that is the silent-drop
  bug.**
- No new shared `.js` file was created for the identity helpers — they are inline in
  `index.html` and `login.html`. A new root `.js` would need a `globPatterns` entry
  **and** a `backend/Dockerfile` copy (`tests/sw-manifest.spec.js` asserts the
  pairing), and would have collided with P1 on the same lines. Deliberate.

---

## 3. What is safe to drop

- **The console `console.log` lines in `tests/sw-api-cache-partition.spec.js`.**
  They exist to make the leaked payload visible in the red output (guard-integrity
  bar B-22/B-23/B-24). They are evidence, not assertions.
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
