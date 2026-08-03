# Offline ownership — who owns each piece of what is on a phone

Card `offline-ownership-design-note` (A4), run `overnight-20260804`.
Closes the reworded **E-KR3** (`reference/okr-completion-plan-20260804.md` §4).

> **As the owner, I want one written answer to "when a phone is offline, who owns each piece of
> what is on it," so that when data goes missing I know which system to open instead of guessing.**

**Specified content:** `reference/okr-completion-plan-20260804.md` §3 A4 (lines 143–261). That
analysis was done at planning time, 2026-08-03 evening. This card's job was to **re-verify every
row at source and publish** — not to rediscover it. Where measurement contradicted the plan's
table, the measurement wins and the deviation is stated; see **§7 Deviations**. Every row below
carries the file and line it was verified at, on the tree at `2041477`.

---

## 0. The one-paragraph answer

There are **8 classes of data on a crew phone, in 6 named stores** — not the two E-KR3's
parenthetical names. Workbox owns everything read-side and delivery-side, permanently. `sync.js`
owns the one write-side store. RxDB owns nothing on a phone today, because it is never started.
And there is **one class with no owner at all**, named in §4 — that is the honest finding, and it
is the reason this note exists rather than a table that looks clean.

**If data went missing, open the store in the row that matches what went missing.** That is the
whole point of §1.

---

## 1. The class inventory — 8 classes, 6 stores

Six *named stores* across three storage technologies: **Cache API** ×3 (Workbox precache,
`api-cache`, `hq-identity`), **`localStorage`** ×1, **IndexedDB** ×2 (`hq_offline_v1`, RxDB/Dexie).

| # | Class | Store | Owner | Direction | Verified at |
|---|---|---|---|---|---|
| 1 | **App shell** — 31 precached files | Cache API — Workbox precache | **Workbox** | read | `sw.js` precache manifest (31 entries); built by `build-sw.js:305-390` |
| 2 | **API responses for the four non-replicated apps** — Inventory, Onboarding, Users, Purchasing | Cache API — `api-cache` | **Workbox** | read | `build-sw.js:441-447` — one `NetworkFirst` route, `urlPattern: /\/api\//`, `networkTimeoutSeconds: 10` |
| 3 | 🛑 **API responses carrying checklist data** (`/api/v1/workflow/*`) | Cache API — `api-cache` | **Workbox** — the only class that can ever become contested (see rule 2) | read | same route, `build-sw.js:443`; see §5 |
| 4 | **Identity partition key** (`/__hq_identity` → a user uuid) | Cache API — `hq-identity` | **`index.html` writes, the service worker reads** — machinery, not user data | internal | writer `index.html:142-143`; readers `build-sw.js:459-486` (`cacheKeyWillBeUsed` / `cacheWillUpdate`) |
| 5 | **Cached grant list** (which tiles/tools this user has) | `localStorage['hq_apps']` | **`index.html` writes and invalidates, `login.html` also invalidates, `sync-rxdb/bootstrap.js` reads** | read | `index.html:104,211-212`; `login.html:148`; `sync-rxdb/bootstrap.js:62-71` |
| 6 | **Pending checklist submissions** | IndexedDB `hq_offline_v1` → `submitQueue` | **`sync.js`** | 🛑 **write** | `sync.js:43,48,51-52`; `enqueueSubmission` `sync.js:620-630`; `drainQueue` `sync.js:632-681` |
| 7 | **Lamport clock state** (`{id:'clock', lamport_ts, device_id}`) | IndexedDB `hq_offline_v1` → `syncMeta` | **`sync.js`** — op-log machinery, not user data | internal | `sync.js:54-56`; `LamportClock` `sync.js:108-139`, persisted at `:116-132` |
| 8 | **Replicated checklist rows** (4 workflow collections) | IndexedDB — RxDB / Dexie | **RxDB — a replication buffer only** (rule 2), and **not created today** | read (not started) | `sync-rxdb/bootstrap.js:17-28,82-90`; pinned by `tests/sync-rxdb-client.spec.js:1468-1470` |

**Rows 6, 7 and 8 exist only on `workflows.html`.** It is the only page that loads `sync.js`
(`workflows.html:309`) or `sync-rxdb/bootstrap.js` (`workflows.html:334`). This is already the
per-app boundary §8 proposes to make architectural.

### 1a. The 31 precached files, enumerated

Because "the app shell" is where a reader's eye slides off, and because the count is a gate
(`build-sw.js` fails the run if a precached file references something not precached):

- **7 pages** — `index.html`, `workflows.html`, `inventory.html`, `onboarding.html`, `users.html`,
  `purchasing.html`, `login.html`
- **4 root scripts** — `ptr.js`, `sync.js`, `log.js`, `tab.js`
- **2 manifests** — `manifest.json`, `version.json`
- **11 icons** — `icons/logo-square.png` + `icons/icon-{48,72,96,128,144,152,192,256,384,512}x*.png`
- **1 vendored engine** — 🛑 **`vendor/rxdb.bundle.js`**
- **5 sync-rxdb modules** — `bootstrap.js`, `client.js`, `conflict-handler.js`,
  `conflict-notice.js`, `conflict-notice-ui.js`
- **1 schema module** — `sync-schema/collections.js`

7 + 4 + 2 + 11 + 1 + 5 + 1 = **31**. Verified: `node build-sw.js` prints
`SW built: 31 files precached (2162.2 KB)`, twice, with no tree diff.

### 1b. The two splits, and why collapsing either one hides a class

**`hq_offline_v1` is two classes, not one (#6 and #7) — because they have different fates.**
Retire the op-log and `syncMeta` becomes dead weight while `submitQueue` survives untouched. A row
reading "`hq_offline_v1` → `sync.js`" conceals that entirely.

🛑 **This collapse has already happened in this repo, twice, and is in the tree right now.**
`designs/offline-save-honesty.md:12-13` states *"IndexedDB `hq_offline_v1` holds one store,
`submitQueue` (`sync.js:51`)"*, and `workflows.html:384-386` repeats *"the IndexedDB queue that
today holds submissions only (`sync.js:51`)"*. Both were true when written and are false now: the
database opens at **version 2** (`sync.js:48`) and `onupgradeneeded` creates **two** object stores,
`submitQueue` at `sync.js:51-53` and `syncMeta` at `sync.js:54-56`. The split is not pedantry —
it is the difference between a correct inventory and the one already written down.

*(Same sentence in `offline-save-honesty.md:11` also cites `submitOp` at `sync.js:695` and
`drainQueue` at `sync.js:560`; both have moved — they are now `sync.js:781` and `sync.js:632`.
The mechanism it describes is unchanged and still correct.)*

**`api-cache` is two classes, not one (#2 and #3) — split by whether RxDB replicates the underlying
rows.** One Workbox route (`build-sw.js:443`, `urlPattern: /\/api\//`) covers both, so the *store*
is one; the *ownership question* is two. #2 is uncontested **forever** — RxDB has never covered
Inventory, Onboarding, Users or Purchasing and nothing proposes that it should. #3 is the only
class in this table that can ever become dual-owned. Collapsing them makes the permanently-safe
half and the one genuinely contestable half look identical.

🛑 **Corollary, stated because a reader will reach for it: the `/\/api\//` route is not retirable.**
`build-sw.js:415-421` records why — RxDB replicates four collections, all of them `workflow`, so
retiring the route takes offline API reads away from four tools that have no other offline story.
Decision 57's *"retire it once RxDB replicates"* was false when written; decision 112 replaced it.

---

## 2. The four ownership rules

These are **rules**, not observations. A future implementer who reads only this section must come
away knowing what they may not do.

### Rule 1 — Workbox owns delivery, unconditionally and permanently.

🛑 **There is no configuration in which RxDB replaces Workbox, and Workbox is not transitional.**

The mechanism, verified at source: **`vendor/rxdb.bundle.js` is row #1 — it is precached by
Workbox.** RxDB reaches the browser *because Workbox put it there*. With no service worker, an
offline launch yields no document, therefore no script, therefore no RxDB — and RxDB's IndexedDB
would hold good data that nothing on the device can read. `sync-rxdb/bootstrap.js:30-35` says the
same thing from the other side: a production page importing the bundle *"is what earns it the
`globPatterns` precache entry"*, and *"a precached URL that 404s fails the ENTIRE service-worker
install."*

🛑 **Do not write, or accept in review, the claim that "RxDB handles offline."** It does not handle
offline generally. It is one class of one app's data (row #8), delivered by Workbox, and today not
running at all. *"RxDB handles offline, so why do we still have Workbox"* is the exact question
this note exists to close, permanently.

### Rule 2 — RxDB's IndexedDB is a replication buffer, NOT an offline read source.

🛑 **Nothing may read from RxDB on a code path that can execute offline.** Offline reads are
Workbox's, exclusively.

This is a **prohibition on future code**, and it is the entire reason class #3 is uncontested
today. Break it without also doing what §6 requires in the same change set and class #3 becomes
genuinely dual-owned — which is the precise failure E-KR3 exists to prevent.

It is currently enforced structurally: `workflows.html` contains no call to `HQSync.createDatabase`,
`HQSync.startReplication` or `HQSync.client`, and `tests/sync-rxdb-client.spec.js:1468-1470` reds
if any of the three appears. That test is the rule's only enforcement — treat editing it as
editing this rule.

### Rule 3 — the boundary is records vs. responses, not static vs. dynamic.

Workbox stores **URL-keyed opaque responses** and has no query layer. It cannot answer `?date=05`
from cached `?date=03` and `?date=04`, cannot invalidate by row, and cannot update partially. RxDB
stores **queryable records**. **Assign each class by the granularity it needs**, and never by
whether the bytes "feel static."

This also explains, structurally, the fetch-storm class: `sync.js` reconciles a replayed
`SUBMIT_CHECKLIST` by re-fetching an *entire* list (`sync.js:455-457`, and the same shape at
`:480-482`) **because a whole response is its only storage unit.** There is no partial update
available to it. Records make one possible; responses never will. (See E-KR1 and §8's benefit 1 —
the class dies structurally under records, not by gating.)

### Rule 4 — writes have exactly one owner, and it is never Workbox.

**Workbox is read-side.** It has no write path in this repo and must not be given one.

The only write-side Workbox option is `workbox-background-sync`. It appears **nowhere in this
tree** (grep: zero hits), and it is not a candidate: the Background Sync API is a Chromium-only
web-platform feature that WebKit does not implement, and **the crew runs iPhones** — ledger T-21d
records prod verification on **Safari** as the client under test (`ledger.md:1028,1045`). A
write-side mechanism that is inert on the browser the crew actually uses is disqualified.

So: **class #6 owns pending submissions** (`sync.js`, `submitQueue`). **RxDB push owns
replicated-row writes — when live.** Nothing else on a phone owns a write.

🛑 **Footnote so nobody wonders where a field edit lives: an unsent field answer has no store at
all.** `submitOp` (`sync.js:781-796`) awaits `api('POST', 'ops', …)` with no queue and no offline
branch, so offline it simply rejects and the rejection propagates; `debouncedSaveField`
keeps the value in the in-memory `PENDING_VALUES` / `PENDING_SYNC` only (`workflows.html:383-388`),
so a reload or app kill while still offline loses it. That is a **known, deliberately-scoped gap**
(`designs/offline-save-honesty.md`), not a ninth class — a class needs a store, and this has none.
It is named here so that "where did my typed answer go?" resolves to *nowhere, by design, and here
is the note* rather than to a hunt through six stores.

---

## 3. Current state at a glance

| | Class #3 (`api-cache` checklist responses) | Class #8 (RxDB rows) | Contested classes |
|---|---|---|---|
| **Today** | owned by **Workbox** | **buffer only** — and not even created (rule 2) | **0** |
| **Target — §8** | 🛑 **removed from the Workbox route** | **owns checklist data** | **0** |

**Zero classes are dual-owned today, and zero have an unstated owner** — with the single, loud
exception of §4, which is not a *storage* class at all.

---

## 4. 🛑 The class with no owner

Under **decision 126** HQ runs a deliberate parallel: **REST writes land in HQ's Postgres, and RxDB
push writes land in the substrate.** Push is unconditional — `sync-rxdb/client.js:1194` configures
`push: { batchSize: … }` with no gate, in contrast to `pull` immediately above it at `:1190-1193`,
which is scoped and filtered. When replication is switched on, it pushes.

# **Nothing reconciles them.**

If one write succeeds and the other fails, the two stores diverge **silently**, and **no code in
this repo detects it.** There is no reconciler, no drift check, no alert, and no test.

**This class has no owner.** Not "an owner we haven't named" — **no owner.** The reworded E-KR3
forbids an *unstated* owner; it does not forbid an honest *"nobody — here is the risk, and here is
what would detect it."* Recording this is what makes the KR meaningful rather than decorative.
**Suppressing it to make the table in §1 look clean is the laundering case.**

**Scope of the exposure, stated so it is not over- or under-read:** it is **latent today**.
`HQ_SYNC_REST_URL` and `HQ_SYNC_REALTIME_URL` are unset in every environment, so the `/sync/*` door
answers 503 to everything (`backend/cmd/server/main.go:436-438`,
`backend/internal/sync/proxy.go:78,111,130-131`) and no push ever fires. **It arms the moment a
deploy sets those variables** — which is exactly why it is written down before that deploy rather
than after the first divergence.

**What would detect it** (none of this exists; recorded so the next card starts from a shape):

1. A periodic row-count and content diff across the four replicated tables, HQ Postgres vs.
   substrate, alerting on any mismatch — the same shape as the existing Recipes drift check
   (weekly `drift_check_results` + Cliq message), which is precedent that this repo can carry a
   drift detector without inventing anything.
2. A write-time correlation id stamped on both paths, so a divergence names the specific write.
3. Or — the structural answer — **§8**, which removes the second write path entirely and therefore
   removes the class rather than monitoring it.

---

## 5. One piece of good news, verified

The sharpest edge the feasibility spike warned about — *"a fallback answering a **replication**
request with cached JSON"* (`roadmap.md:239-242`) — **cannot occur.** Nobody should re-litigate it.

- The sync proxy is mounted at **root**, deliberately outside `/api/v1`:
  `backend/cmd/server/main.go:439-442` registers `r.Handle("/sync/*", …)` in its own group.
- Its two prefixes are `/sync/rest` and `/sync/realtime`
  (`backend/internal/sync/proxy.go:124-125`), so RxDB's traffic is `<origin>/sync/rest/*` and
  `<origin>/sync/realtime/*`.
- Workbox's **only** runtime route is `urlPattern: /\/api\//` (`build-sw.js:443`).

**No match.** `/sync/rest/...` contains no `/api/` segment, so the `NetworkFirst` handler — and
therefore the `handlerDidError` 503-JSON fallback at `build-sw.js:486-491` — is never reached by a
replication request.

**The transport layers are already cleanly partitioned by URL namespace. The residual overlap is
data, not traffic** — which is §4, and which no URL routing can fix.

---

## 6. The target state, and the single trigger between here and there

🛑 **The target is not "after a real cutover." It is the named architecture in
`reference/okr-completion-plan-20260804.md` §8, "Two stores."** Cited by name so the next
implementer aims at a stated endpoint instead of choosing one.

**§8's statement:** *RxDB is the definitive source for the Operations app **after it loads**.
Workbox owns everything else.* Two things in that sentence are load-bearing: **"after it loads"**
encodes rule 1 (Workbox still delivers the app, including `vendor/rxdb.bundle.js`); **"the
Operations app"** draws the line **per-app, not per-technology** — Inventory, Onboarding, Users and
Purchasing stay wholly Workbox's.

**Where the 8 classes land under §8** (§8's own table): #1 → Workbox, unchanged · #2 → Workbox,
unchanged · **#3 eliminated** · #4 → Workbox machinery · **#5 residual, the one documented
exception** · #6 → RxDB (a pending write is an unpushed document) · **#7 evaporates** with the
op-log · #8 → RxDB. **Two stores plus one ~20-line exception, written down** — #5 is a genuine
chicken-and-egg, because `bootstrap.js:62-71` reads the cached grant list to decide *what to
replicate*, so it must be readable before RxDB exists.

🛑 **This note describes §8. It does not adopt it.** §8 is **decision 126 option (i)**, which triage
assessed as *"the honest end state but a milestone rather than a card."* Adopting it is a roadmap
decision, not this note's and not a card's.

### The trigger — one condition, two changes, one change set

> **The trigger is the moment the page reads checklist data from RxDB on a code path that can run
> offline.**

🛑 **At that moment both of these must change together, in the same change set:**

1. **Narrow `build-sw.js`'s `urlPattern: /\/api\//` (`:443`) to exclude the RxDB-owned endpoints**,
   so class #3 stops being written to `api-cache`; and
2. **Drop rule 2** from this note, because it will no longer be true.

**Change one without the other and class #3 becomes genuinely dual-owned** — the same checklist
answer cached as a Workbox response *and* stored as an RxDB record, with no rule saying which one a
reader gets. That is the failure E-KR3 exists to prevent, and it is reachable by a one-line edit in
either file.

### The rejected alternative, recorded so it is not re-proposed

**A service worker answering `/api/v1/workflow/*` from RxDB's IndexedDB** — one store, two access
paths — is technically possible; a service worker does have IndexedDB access. **Rejected:** it
duplicates the API's response-shaping logic in a second place, written against RxDB's internal
Dexie layout, producing a failure mode nobody can debug from a truck.

---

## 7. Deviations from the plan's table

Six places where measurement contradicted or refined
`reference/okr-completion-plan-20260804.md` §3 A4. **Stated rather than silently patched**, so the
next reader can tell a verified row from a quietly-corrected one.

| # | The plan said | Measured | Settled at |
|---|---|---|---|
| **D1** | Row 4: verify `cacheKeyWillBeUsed` / `cacheWillUpdate` *"in `sw.js`"* | They are **authored in `build-sw.js`** and reach `sw.js` only as generated, minified output — `sw.js` is a 3-line artifact with no citable structure. The reviewable source is `build-sw.js:459-486`. | `build-sw.js:459,478`; `wc -l sw.js` = 3 |
| **D2** | Row 5: *"`index.html` writes, `bootstrap.js` reads"* | A **third participant**: `login.html:148` also removes `hq_apps`, on a *different* rule from `index.html`'s — it skips the purge when the same crew member re-authenticates, where logout is unconditional (`index.html:205-212`). Ownership is write + **two** invalidators + read. | `login.html:143,148`; `index.html:205-212` |
| **D3** | Row 8: *"read (dark)"* | **Stronger than dark — the store does not exist.** `bootstrap.js` deliberately does not call `createHQSyncDatabase()` (`:17-28`); it is exposed as a deferred handle only (`:82-83`), and no production page calls it. Class #8 has **zero bytes on any phone today**. "Dark" implies populated-but-unread; it is not created. | `sync-rxdb/bootstrap.js:17-28,82-90`; `tests/sync-rxdb-client.spec.js:1468-1470` |
| **D4** | Rule 3: *"`sync.js:443-454` re-fetches an entire list per replayed op"* | The branch carries the **T-18 catch-up gate** (`sync.js:455-457`): a *silent* replay with no runner open no longer re-fetches. "Per replayed op" describes the pre-gate behavior. **The structural claim rule 3 needs is unaffected** — a whole response is still the only storage unit, so a whole-list re-fetch is still the only reconcile available. Line numbers also shifted: the branch is `:442-458`, the second site `:475-483`. | `sync.js:442-458`, `:475-483` |
| **D5** | Rule 4: background sync is *"Chromium-only — Safari does not support it … (T-21d)"* | **T-21d does not say this.** It records a prod SW-update pipeline defect and Safari as the verified client (`ledger.md:1025-1046`) — it establishes *the crew is on Safari*, not the API's support matrix. The Chromium-only fact is a **web-platform fact not verifiable in this repo**. Rule 4 above now cites T-21d only for what it actually establishes. (`workbox-background-sync`: zero hits in the tree.) | `ledger.md:1028,1045` |
| **D6** | Row 1's enumeration: *"HTML, JS, icons, `manifest.json`, `version.json`, `vendor/rxdb.bundle.js`, `sync-rxdb/*`"* | **Count correct (31), enumeration incomplete**: it omits `sync-schema/collections.js`, and `sync-rxdb/*` is **5 of the 6** files in that directory (`package.json` is not precached). Full list in §1a. | `sw.js` manifest; §1a |

**Everything else in the plan's table re-verified exactly as written**, including the two citations
most likely to have drifted: `build-sw.js:441` (`runtimeCaching`), `sync.js:55,116-132` (the Lamport
clock's `syncMeta` writes), `sync-rxdb/client.js:1194` (unconditional `push`), and
`main.go:439-442` (the root-mounted sync proxy) — each landed on the exact cited line.

---

## 8. How to use this note when data goes missing

| The report | Open this |
|---|---|
| *"The app won't load at all offline"* | Class #1 — Workbox precache. Check `sw.js` installed; a precached URL that 404s fails the whole install. |
| *"Inventory / Onboarding / Users / Purchasing shows stale or empty data offline"* | Class #2 — `api-cache`. NetworkFirst, 10s timeout. |
| *"My checklists list is stale offline"* | Class #3 — `api-cache`, same route. |
| *"I see another crew member's data on this phone"* | Class #4 — `hq-identity`. The partition key is missing or wrong; the fallback is the `anon` partition. |
| *"My tiles are wrong / I see a tool I shouldn't"* | Class #5 — `localStorage['hq_apps']`, plus server-side grants. The cache is not the authority. |
| *"I pressed Submit and it never arrived"* | Class #6 — `submitQueue` in `hq_offline_v1`. `drainQueue` fires on `online`. |
| *"Ops are replaying out of order / from the beginning"* | Class #7 — `syncMeta` in `hq_offline_v1`. |
| *"A typed answer vanished after a reload while offline"* | **No store.** Rule 4's footnote — in-memory only, by design. See `designs/offline-save-honesty.md`. |
| *"HQ and the substrate disagree"* | 🛑 **§4 — no owner.** Nothing reconciles them and nothing detects it. Latent until `HQ_SYNC_REST_URL` is set. |
