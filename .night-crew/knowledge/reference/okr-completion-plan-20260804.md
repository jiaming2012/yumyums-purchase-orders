# OKR completion plan — "Sync foundation" cycle

> **Written:** 2026-08-03 evening, before tonight's slate, at the operator's request.
> **Grading basis:** hand-graded. The mechanical grader is blind in this repo — `night-crew okr
> grade` returns *"no metrics.jsonl found under .night-crew/runs"* and `milestone export` returns
> *"no runs after 20260724"*, both the `<runid>` vs `2026-08-03-autonomous` directory-shape defect
> already filed as **B-33** / **B-77**. Every grade below cites a file and a line.
> **Status:** proposal. Nothing here is decided. The OKR rewords in Track B require operator
> sign-off before `okrs.md` is edited.

---

## 0. Current standing

**6 MET · 2 PARTIAL · 3 UNMET · 1 UNGRADEABLE.**

| KR | Grade | Basis |
|---|---|---|
| P-KR1 feasibility gate before schema/JWT WOs | ✅ MET | Verdict in `ledger.md` at triage 2026-07-25 (T-22); downstream WOs dispatched 07-26 and later |
| P-KR2 no cutover WO while hydration item reads `new` | ✅ MET | Item ruled 2026-07-26, decision 67 (`roadmap.md:2172`); cutover dispatched 08-03 |
| P-KR3 no-parallel-run constraint carried verbatim | ✅ MET | `roadmap.md:2010`; 0 build WOs proposed it — the operator chose that shape under an explicit waiver |
| D-KR1 0 of 3 downstream WOs dispatch before go | ✅ MET | Same timestamps as P-KR1 |
| D-KR2 prod parity + returning-client screenshots | ❌ UNMET | Never run |
| D-KR3 per-card timing for every card + median | ⚠️ PARTIAL | Median computed (N=11, 103m vs 94m); `card-actuals.md:898,267` records 7+ cards with legs untimed |
| E-KR1 0 of 2 fetch-storm items recur | ❌ UNMET | See §1 |
| E-KR2 0 attack variants bypass RLS | ✅ MET | `jwtbridge_rls_test.go` → `TestJWTBridgeRLS`: V1–V13 + 3 controls; caveat B-58 per decision 121 |
| E-KR3 1 owner per offline data class after cutover | ❌ UNMET | `designs/` holds 7 notes; none is it |
| Q-KR1 RxDB + JWT bridge ship tests in the same WO | ✅ MET | `jwtbridge_test.go` / `jwtbridge_handler_test.go` dated with the 20260726 run; 5 RxDB spec files dated with their cards |
| Q-KR2 100% of WOs carry red-run evidence in the WO record | 🛑 UNGRADEABLE | The named field does not exist in the record format — see §1 |
| Q-KR3 every WO states full-suite runtime + load-sensitivity | ⚠️ PARTIAL | Absent from `slate-20260801.md`; one partial mention in `slate-20260802.md` |

---

## 1. Why this happened — and why it is mostly not a delivery failure

The OKRs were authored **2026-07-25** against an architecture that was disproved **2026-08-03**.

Card S1b `sync-hard-cutover` parked on a measurement: RxDB replicates to the **self-hosted Supabase
substrate**, a second Postgres, and `0002_hq_fdw.sql`'s bridge runs **HQ → substrate, read-only,
carrying permissions rather than data**. Nothing carries a checklist row back. Ledger decision 49's
deciding argument — *"RxDB replicates rows straight from Postgres and there is no API boundary left
to translate at"* — is false as built. Decision **126** then retired the card and chose a different
shape: **RxDB serves reads, HQ's REST path keeps owning writes**, with P-KR3's parallel-run
prohibition waived for that shape only.

Four KRs were written to measure the retired architecture:

- **E-KR3** grades a state *"after cutover"* that no longer exists in that form.
- **E-KR1**'s two design notes argue entirely from *"`sync-hard-cutover` deletes `sync.js`"*
  (`fetchstorm-catchup-gate-superseded.md:8,64`). Verified tonight: `sync.js` is **799 lines, still
  in the tree**, and both fetch-storm mechanisms are live at the lines the notes cite —
  `sync.js:443-454` (catch-up replay) and `:475-479` (the `SAVE_TEMPLATE` gate). Decision 126 keeps
  the REST/ops write path, and **B-69** records `sync.js` also holds `api()` (12 call sites), the
  submit queue and `APP_TIMEZONE`. It is not being deleted.
- **D-KR2** reads *"once `sync-hard-cutover` ships"*. It never shipped; the precondition never fired.
- **Q-KR2** measures *"the red-first evidence **field** in each `.night-crew/runs/` WO record"*.
  Checked tonight: `night-crew-main/prompts/` contains **no merge-intent template** — the schema
  lives in the launch prompt each slate writes, and it has never carried that field. Of every
  merge-intent this cycle, exactly one (`merge-intent-prestep.md`, 20260802) has a `## Red-first`
  heading. The KR is unauditable, not failed.

🛑 **E-KR1 is the exception, and it must not be treated like the others.** Its subject did not move
under it — the bug class it names is genuinely still in the tree. That is a real miss.

---

## 2. The integrity test

Every reword below is checked against one rule:

> **A reword is honest when it changes what is *measured* while preserving what is *protected*.
> It is laundering when it lowers the bar on the thing the KR exists to guarantee.**

Three legitimate grounds, and one illegitimate one:

| Ground | Legitimate? | Applies to |
|---|---|---|
| The KR measures a system that an operator decision replaced | ✅ | E-KR3, D-KR2 |
| The KR over-specified the *evidence* and under-specified the *property* | ✅ | D-KR3 |
| The KR names an artifact the process never built | ✅ | Q-KR2 |
| The work was hard and did not get done | ❌ **that is a red KR** | E-KR1, Q-KR3 |

This extends the authoring lesson already recorded in `okrs.md:17-19` — *"write KR metrics that
cannot be failed by desirable behavior."* Three of these were failed by desirable behavior: parking
a card that would have silently destroyed data, and taking an architecture decision that avoided it.

---

## 3. Track A — build it

Cards that close a gap by changing code. Sizes are against the current class median of **103m**
end-to-end (`card-actuals.md`, N=11), not implement-only.

### A1 · `e2e-gate-database-isolation` — S–M, 75–120m — **closes no KR, gates all of their evidence**

> *As the owner, I want a failing test to mean the code is broken, so that I am not deciding whether
> to ship based on last week's leftover data.*

`night-crew.toml:33-34` sets `suite = "npx playwright test"` and
`subset = 'npx playwright test "{tags}"'`. The only `DROP DATABASE IF EXISTS hq_test_e2e` /
`CREATE DATABASE` lives in `Taskfile.yml:53-59` under `task test`, and `playwright.config.js` has
**no `globalSetup`**. So no night-crew gate leg — full or subset — has ever reset the e2e database.
This is B-76's mechanism, and decision **131** already reproduced its effect: the merged tree scored
777/6/1 on a fresh DB against the base's 758/6/4, and the shared DB now reds three onboarding tests
where it once red one.

Fix: move the reset where it cannot be bypassed (a `globalSetup` guarded to refuse any database
whose name is not a test database), or point `[e2e] suite`/`subset` at a resetting target. Plus a
test that proves the reset fires on the **subset** path, which is the one that gets used most.

🛑 Do not touch `playwright.config.js:60`'s repo-wide `serviceWorkers: 'block'` (B-15).

### A2 · `workflows-autosavefield-phantom` — S, 45–75m — **closes no KR, fixes shipped code**

> *As a crew member, I want the photo I attach to a failed check to actually save, so that evidence
> I took the trouble to capture is not thrown away.*

`workflows.html:2219` calls `autoSaveField`; **zero definitions repo-wide**. A live `ReferenceError`
on the fail-note-with-photo path today. The real write path is `debouncedSaveField` →
`submitOp('SET_FIELD')` → `POST /ops`. Red-first: a Playwright test on that path throws before the
fix. **B-65.**

Note the documentation blast radius — the card charter, `roadmap.md`, `sync-rxdb/bootstrap.js:9` and
`sync-rxdb/conflict-notice-ui.js:23` all name `autoSaveField` as the live write path. So does
`CLAUDE.md`'s own persistence rule. Those are wrong and should be corrected in the same change set.

### A3 · `gate-evidence-hygiene` — S, 60–90m — three named mechanisms, each small

> *As the owner, I want the checks that say "clean" to be capable of saying "dirty", so that a green
> result is information rather than a shrug.*

- **B-70** — `sync-rxdb/client.js` byte **50850** is a literal NUL inside `scopeFingerprint`'s
  template literal. `file` reports the whole 59 KB file as `data`, so plain `grep` returns nothing
  and **exits 0**. Several `done_when:` rows this milestone are of the form *"grep returns nothing."*
  Fix: `\u0000` escape — identical runtime string, file becomes text. Add a test asserting the tree
  holds no NUL bytes.
- **B-78** — `night-crew.toml:59-62` claims *"re-verified at landing"* that four tokens select
  exactly four spec files. Measured: `sync` matches **six**. Safe direction, false claim, and a
  future editor trusting it could narrow the tag into under-inclusion.
- **B-79** — S1a's unknown-`scope.mode` refusal is named a must-survive in its own merge-intent and
  is pinned by no test; `if (false)` leaves the suite 55/55 green.

### A4 · `offline-ownership-design-note` — S, 45–75m — **closes E-KR3 under the §4 reword**

> *As the owner, I want one written answer to "when a phone is offline, who owns each piece of what
> is on it," so that when data goes missing I know which system to open instead of guessing between
> two.*

🛑 **Under the reworded E-KR3 this is closeable without building the cutover.**

**The analysis below was done at planning time (2026-08-03 evening) and is the card's specified
content.** The implementer's job is to **re-verify every row at source and publish**, not to
rediscover it at 2am. Where verification contradicts a row here, the measurement wins and the
deviation is stated — this table is a starting position, not an authority.

Destination: `.night-crew/knowledge/designs/offline-ownership.md`.

#### Required content 1 — the class inventory

E-KR3's parenthetical names **two** classes (*"static assets → Workbox, checklist data → RxDB"*).
Measured, there are **eight**, across six physical stores. Publishing this table *is* the note's
main deliverable; the count is the finding.

| # | Class | Store | Owner | Dir. | Verify with |
|---|---|---|---|---|---|
| 1 | App shell — 31 files (HTML, JS, icons, `manifest.json`, `version.json`, `vendor/rxdb.bundle.js`, `sync-rxdb/*`) | Workbox precache (Cache API) | **Workbox** | read | precache manifest in `sw.js` |
| 2 | API responses for the four non-replicated apps (inventory, onboarding, users, purchasing) | Workbox runtime `api-cache` | **Workbox** | read | `runtimeCaching` in `build-sw.js:441` |
| 3 | 🛑 API responses carrying **checklist** data (`/api/v1/workflow/*`) | Workbox runtime `api-cache` | **Workbox** — contested with #8 | read | same route; see rule 2 |
| 4 | Identity partition key | `hq-identity` Cache | `sw.js` (machinery) | internal | `cacheKeyWillBeUsed` / `cacheWillUpdate` in `sw.js` |
| 5 | Cached grant list | `localStorage` `hq_apps` | `index.html` writes, `bootstrap.js` reads | read | `bootstrap.js` `cachedGrantSlugs()` |
| 6 | Pending checklist **submissions** | `hq_offline_v1` → `submitQueue` | `sync.js` | **write** | `sync.js:52`, `enqueueSubmission`, `drainQueue` |
| 7 | Lamport clock state | `hq_offline_v1` → `syncMeta` | `sync.js` (op-log machinery) | internal | `sync.js:55,116-132` |
| 8 | Replicated checklist rows (4 tables) | RxDB / Dexie IndexedDB | RxDB — **buffer only, see rule 2** | read (dark) | `HQ_SYNC_REST_URL` unset; `bootstrap.js` starts no replication |

Two splits the note must make explicitly, because both hide a class when collapsed:

- **`hq_offline_v1` is two classes, not one** (#6 and #7) — *with different fates*. Retire the
  op-log and `syncMeta` becomes dead weight while `submitQueue` survives untouched. A row reading
  "`hq_offline_v1` → `sync.js`" conceals that.
- **`api-cache` is two classes, not one** (#2 and #3) — split by whether RxDB replicates the
  underlying rows. #2 is uncontested forever; #3 is the only class that can ever become dual-owned.

#### Required content 2 — the four ownership rules

1. **Workbox owns delivery, unconditionally and permanently.** RxDB is *precached by* Workbox
   (`vendor/rxdb.bundle.js` is row #1). RxDB cannot bootstrap itself: with no service worker, an
   offline launch yields no document, so no script, so no RxDB — and its IndexedDB holds good data
   nothing can read. **There is no configuration in which RxDB replaces Workbox.** State this
   directly; "RxDB handles offline, so why Workbox" is the question this note exists to close.
2. **RxDB's IndexedDB is a replication buffer, NOT an offline read source.** Nothing may read from
   RxDB on a code path that can execute offline. Offline reads are Workbox's exclusively. **This
   rule is the entire reason class #3 is uncontested today** — and it is a rule, not an
   observation, so it must be written where a future implementer will hit it.
3. **The boundary is records vs. responses, not static vs. dynamic.** Workbox stores URL-keyed
   opaque responses and has no query layer — it cannot answer `?date=05` from cached `?date=03`
   and `?date=04`, cannot invalidate by row, and cannot update partially. RxDB stores queryable
   records. Assign each class by the granularity it needs. (This is also the structural account of
   the fetch-storm class: `sync.js:443-454` re-fetches an entire list per replayed op because a
   whole response is its only storage unit — see E-KR1.)
4. **Writes have exactly one owner, and it is never Workbox.** Workbox is read-side.
   `workbox-background-sync` is the only write-side option and it is **Chromium-only — Safari does
   not support it**, which is disqualifying on a crew running iPhones (T-21d). #6 owns pending
   submissions; RxDB push owns replicated-row writes when live.

#### Required content 3 — 🛑 the class with NO owner

Under decision 126, REST writes land in **HQ's Postgres** and RxDB push writes land in **the
substrate** (push is unconditional — `client.js:1194`, see §3 A5). **Nothing reconciles them.** If
one succeeds and the other fails, the two stores diverge silently and no code detects it.

**This class has no owner, and the note must say so in those words.** The reworded E-KR3 forbids an
*unstated* owner; it does not forbid an honest "nobody — here is the risk, here is what would
detect it." Recording it is what makes the KR meaningful rather than decorative. Suppressing it to
make the table look clean is the laundering case.

#### Required content 4 — current state, target state, and the trigger between them

| | Class #3 (`api-cache` checklist responses) | Class #8 (RxDB rows) | Contested classes |
|---|---|---|---|
| **Today** | owned by Workbox | buffer only (rule 2) | **0** |
| **Target — §8** | **removed from the Workbox route** | owns checklist data | **0** |

🛑 **The target state is not "after a real cutover" — it is the named architecture in §8, "Two
stores."** The note must cite §8 as the destination rather than gesturing at a future card, so the
next implementer aims at a stated endpoint instead of choosing one.

**Trigger:** the moment the page reads checklist data from RxDB on a code path that can run
offline. At that moment both cells must change **in the same change set** — narrow
`build-sw.js`'s `/\/api\//` pattern to exclude the RxDB-owned endpoints, and drop rule 2. Change
one without the other and class #3 becomes genuinely dual-owned, which is the failure E-KR3 exists
to prevent.

Record the rejected alternative: **a service worker answering `/api/v1/workflow/*` from RxDB's
IndexedDB** (one store, two access paths) is technically possible — a SW has IndexedDB access —
and is rejected because it would duplicate the API's response-shaping logic in a second place,
against RxDB's internal Dexie layout, producing a failure nobody can debug from a truck.

#### Required content 5 — one piece of good news, verified

The sharpest edge `roadmap.md:238-240` warned about — *"a fallback answering a replication request
with cached JSON"* — **cannot occur, and the note should say why so nobody re-litigates it.** The
sync proxy is mounted at **root**, outside `/api/v1` (`main.go:439-442`), so RxDB's traffic is
`<origin>/sync/rest/*` and `<origin>/sync/realtime/*`. Workbox's only runtime route is
`/\/api\//`. **No match.** The transport layers are already cleanly partitioned by URL namespace;
the residual overlap is data, not traffic.

#### `done_when:`

- The note exists at `designs/offline-ownership.md` and every one of the 8 rows names a store, an
  owner and a direction — **0 rows with an unstated or ambiguous owner** — each re-verified at the
  cited file, with any deviation from the table above stated.
- The no-owner reconciliation gap is stated explicitly, in those words.
- Rules 1–4 appear as rules, with rule 2's prohibition phrased so a future implementer reading only
  the note knows not to read RxDB on an offline path.
- The trigger is stated as a single condition with both required changes named together.
- Cross-checked against `build-sw.js`'s `runtimeCaching` block and `bootstrap.js`'s
  no-replication-started state (E-KR3's own "cross-checked against the `build-sw.js` / RxDB-init
  diff" clause) — cite both at the line.
- 🛑 **Do not claim RxDB "handles offline" generally, and do not describe Workbox as
  transitional.** Both are permanent; rule 1 says why.

### A6 · `app-version-badge` — S, 45–75m — **operator's proposal 2026-08-03; replaces D-KR2b's evidence method**

> *As the owner, I want the app to tell me which version it is actually running, so that I can
> confirm a deploy reached my phone without staging a photograph of it.*

> *As a crew member, I want to say which version I'm on when I report something broken, so that
> nobody spends an hour chasing a bug I already have the fix for.*

Replaces the screenshot ritual for proving **propagation**. Discreet version line in the UI
(`index.html` footer is the natural home — the launcher every user passes through).

🛑 **THE SOURCE IS THE DESIGN. Read the precached `version.json`, NEVER `/api/v1/health`.** The
server's `frontend_version` is always current — it was never wrong in T-21d — so a badge fed from
the API would have displayed the correct new version on a phone frozen on the old bundle. **It
would have hidden the defect rather than caught it.** `version.json` is precached
(`sw.js` manifest, revision `d324802…`) and served cache-first, so it reports what *this device's
installed bundle* is, which is the only value capable of being stale. That is precisely why it is
the right one.

**Most of this already exists.** `build-sw.js:291` writes the file, and its own comment states the
purpose: *"so the frontend can read its own version without hitting the API."* It currently holds
`{"frontend":"1.4.0"}`. No page displays it. The missing work is the display and its test.

**Make it self-checking** (recommended, small addition): show the cached frontend version beside the
server's `frontend_version` from `/api/v1/health`. Equal ⇒ the deploy propagated. Different ⇒ this
is the T-21d class, and the app has diagnosed itself with both numbers on screen. The check becomes
*open the app, glance at the corner* rather than *compare against a number you memorised*.

**Scope limit, stated so it is not overclaimed:** this proves a bundle propagated. It does **not**
prove a feature renders correctly — that belongs to each card's own `done_when:`. D-KR2 conflated
the two; separating them is the point.

**Effect on §4's D-KR2b reword: the N/A disappears.** D-KR2b was to grade N/A this cycle because no
client-visible surface shipped. A version badge is a client-visible surface **every** cycle, so the
KR becomes gradeable regardless of what shipped. That is a strictly better KR than the one §4
proposes, and §4's D-KR2b text is amended accordingly.

### A5 · `sync-live-fill-view` — L, 2h30m–4h30m — **not yet authored; recommended for the NEXT cycle**

> *As a crew member splitting a shift, I want the checklist I have open to update when my colleague
> ticks a box on their phone, so that we do not both fill the same one and I can trust what I am
> looking at.*

The decision-126 successor: RxDB backs the fill view's live read; both list tabs and Submit stay on
REST; both write paths stay live under the waiver. Carries eight open riders — **B-62** (Realtime
filter never proven against a live server), **B-63**, **B-64**, **B-66**, **B-67**, **B-68**,
**B-69**, **B-79**.

🛑 **PREMISE SETTLED 2026-08-03 evening, by reading the code — and the answer narrows this card.**

`startHQReplication` (`sync-rxdb/client.js:1132-1195`) configures **both** legs for all four
collections, unconditionally — `pull: {batchSize, queryBuilder}` and `push: {batchSize}`. There is
no read-only mode and no option to suppress push. So the double-write is not an inference about
intent: it is what the code does the moment `HQ_SYNC_REST_URL` is set, and push is the **only**
mechanism that would ever put a row in the substrate.

**That makes "RxDB serves reads" materially narrower than decision 126's wording implies.** The
substrate's four tables are populated *exclusively* by RxDB clients pushing to them — not by HQ's Go
backend, not by the ops log, not by any admin path, and there is no backfill. Therefore:

- every checklist row written **before** cutover is absent from the substrate, permanently;
- every row written by any path that is not a browser running RxDB is absent;
- a device pulling a checklist nobody has pushed gets **nothing**, and an empty pull is
  indistinguishable from "this checklist has no answers."

So this card does not RxDB-back the fill view's *read*. It adds **a live cross-device overlay of
edits made after cutover, between devices both running RxDB**, while REST continues to hydrate the
state the fill view renders. That is still the two-device live update the cycle was bought for — but
it is a smaller card with a different `done_when:`, and its central proof is **two devices**, never
one device reloading.

**Two consequences carried into this plan:**

1. Rename and re-scope before authoring. "Backs the fill view" is the wording that would have sent
   an implementer looking for a read path that cannot exist.
2. **A new divergence class, which A4's note must name:** REST writes to HQ, push writes to the
   substrate, and **nothing reconciles them**. If one succeeds and the other fails, the two stores
   disagree silently and no code detects it. This is the concrete "divergence risk" the reworded
   E-KR3 requires stated.

---

## 4. Track B — reword it

Proposed replacement text, with the protected property named for each. **These require operator
sign-off before `okrs.md` is edited.**

### D-KR2 — split, because half of it was never about the cutover

**Protected:** a deploy must be proven to reach people who never close the app. The T-21d lesson —
a fresh load hides staleness — is the whole point.

**Current:** *"once `sync-hard-cutover` ships, `task version` shows prod backend/frontend == local
`version.go` constants with 0 drift, AND 2/2 tab screenshots are verified on a returning client…"*

**Proposed — D-KR2a:** *"Prod parity is verified this cycle, independent of any cutover: `task
version` shows prod backend/frontend == local `version.go` constants with 0 drift. Measured by:
`task version` output recorded in the ledger."*

**Proposed — D-KR2b** *(amended 2026-08-03 after the operator proposed a version badge — see §3 A6;
this supersedes the screenshot-based text)*: *"After each deploy, a **returning** client — an
installed PWA that saw the previous version, never a fresh load, never a cleared cache, never
incognito (T-21d by name) — displays a frontend version read from the **precached** `version.json`
that matches the deployed `version.go` `Frontend` constant. Measured by: the version line read on
that client and recorded in the ledger. Reading the version from `/api/v1/health` does not satisfy
this KR and never will — the server's value cannot be stale, which is the entire property under
test."*

**Integrity:** the returning-client discipline is unchanged and still cannot be satisfied by a fresh
load — the KR now says so explicitly, and names the one shortcut that would silently defeat it.
**Two improvements over the screenshot form:** the evidence is unambiguous (a string matches or it
does not, where a screenshot needs a human to judge "does this look new?"), and **the N/A case
disappears** — a version line is a client-visible surface every cycle, so the KR is gradeable
regardless of what shipped. **Closes with an attended deploy, ~15 min, once A6 lands.**

### D-KR3 — measure the property, not the coverage

**Protected:** we know what a card costs, and cannot hide behind a cherry-picked sample.

**Current:** *"Per-card wall-clock timing is recorded for all 5 cards in this activity (4 sync + 1
independent fix), and a median is computed against the prior cycle's baseline (N=12 / 94m)."*

**Proposed:** *"A defensible per-card cycle-time median is computed against the prior cycle's
baseline (N=12 / 94m), with (a) N ≥ 8, (b) every excluded card listed with its reason, and (c) a
stated sensitivity check showing the median is robust to those exclusions. Measured by: the median
table, the Excluded-with-reasons section, and the Sensitivity section in
`reference/card-actuals.md`."*

**Integrity: this is stricter, not looser.** The current KR permits a 5-of-5 median with no
sensitivity analysis; the proposed one forbids silent exclusions. The activity grew from 5 cards to
24 by three recorded fan-outs, so "all 5" was already unmeetable as literally written — and
`card-actuals.md` **already contains all three required sections**, computed to a higher standard
than the KR asked for. **Grades MET on evidence already in the tree.**

### E-KR3 — one owner per class *and direction*

**Protected:** when data goes missing, the owner knows which store to open. Nothing is silently
owned by two systems.

**Current:** *"Exactly 1 documented owner exists per offline data class after cutover (static assets
→ Workbox, checklist data → RxDB) — 0 classes with dual or ambiguous ownership…"*

🛑 **As written this is unmeetable by construction under decision 126.** A deliberate parallel run
gives checklist data two stores. The KR would fail *because* the operator chose the shape that
avoided silent data loss — precisely the "failed by desirable behavior" trap `okrs.md:17-19` warns
against.

**Proposed:** *"Exactly 1 documented owner exists per offline data class **and direction** — static
assets → Workbox; checklist offline reads → one named owner; checklist offline writes → one named
owner — with **0 classes whose owner is unstated or ambiguous**. Any deliberate two-store split is
named as such, with its reason and its divergence risk stated. Measured by: 1 design note in
`.night-crew/knowledge/designs/`, reviewed at the cycle gate, cross-checked against the
`build-sw.js` / `sync-rxdb` init diff. Gradeable whether or not a cutover shipped."*

**Integrity:** it still forbids the actual hazard — an owner nobody wrote down. It permits a
documented split, which is what was chosen, and requires the split to state what it costs.
**Closes with A4.**

### Q-KR2 — name a field that will exist, and stop demanding red-first of doc cards

**Protected:** a test is proven to fail before the code that makes it pass.

**Current:** *"100% of this activity's WOs carry red-run evidence in the WO record… Measured by: the
'red-first' evidence field in each `.night-crew/runs/` WO record."*

**Proposed:** *"Every WO whose deliverable includes a code change carries red-run evidence — the
named test, the tree or commit it was captured red against, and the green after. Measured by: a
mandatory `## Red-first` section in each card's `merge-intent.md`, asserted by the run's launch
prompt. A WO whose deliverable is documentation, audit or a spike records `n/a — no code change`
explicitly, and an absent section grades the KR down."*

**Integrity:** unchanged for code cards, and the explicit `n/a` closes the hole where "no section"
and "not applicable" look alike. The carve-out is real — **P6 changed no code at all**
(`backend/` byte-identical to base); demanding red-first of it is the same anti-pattern.

**This cycle grades UNAUDITABLE, not MET** — the field did not exist, so no retroactive claim is
available. The launch prompt for the next run carries the section from tonight forward.

### Q-KR3 — per slate, not per WO; and no backfilling

**Protected:** nobody is surprised at 3am by what the gate costs.

**Current:** *"Every WO for this activity states its expected runtime under the full Playwright suite
… and explicitly flags `sync.spec.js` load-sensitivity risk before dispatch."*

**Proposed:** *"Every slate document carries, once, a **Gate cost** section stating the expected
full-suite runtime and the load-sensitivity risks in play for that night's cards. Measured by: that
section's presence in `reference/slate-<runid>.md`. **Backfilling a signed slate is prohibited** — a
missing section grades the KR down and is not repaired retroactively."*

**Integrity:** the anti-backfill clause is the point. Precedent: last night's orchestrator reverted a
correction to `slate-20260803.md:331` because *"backdating a correction into a signed plan of record
would erase the evidence that the discovery happened."* **This cycle grades PARTIAL and stays
PARTIAL.**

### E-KR1 — 🛑 no reword proposed

**Protected:** the migration's entire justification. The cycle exists because of a recurring
fetch-storm / stale-hydration class. If the class survives, the migration bought nothing.

There is no honest reword. `sync.js:443-454` and `:475-479` are in the tree tonight; decision 126
keeps them there; B-69 records three further reasons the file is not going away. Any reword that
lets this grade green is the laundering case.

**Proposed treatment:**

1. Grade **NOT MET**, stated plainly at close.
2. **Un-drop** the two backlog items — *replay-fetch-storm* and the *`sync.js` catch-up gate* — which
   were dropped 2026-07-25 as "superseded by the migration." They are not superseded; the migration
   that would have superseded them was retired.
3. Re-head both design notes as **analysis of an architecture that was not built**, with the retirement
   dated and decision 126 cited. They are good analysis of the wrong system — worth keeping, worth
   not mistaking for a live guarantee. (This is A3-sized, ~30m, and can ride any card.)
4. Carry the class into the next cycle's OKRs as a KR about the architecture that actually ships.

---

## 5. Recommendation

**Reword the four that measure the wrong system. Take the red on E-KR1. Build three small cards.
Leave the cutover to open the next cycle with a real roadmap behind it.**

| Do | Why |
|---|---|
| Reword **D-KR2, D-KR3, E-KR3, Q-KR2, Q-KR3** per §4 | Four measure a retired architecture or an artifact the process never built; one (D-KR3) becomes stricter |
| Build **A1** (gate isolation), **A2** (B-65), **A4** (ownership note), **A6** (version badge) | A1 makes every other grade mean something; A2 is a live user-facing defect; A4 closes reworded E-KR3 without the cutover; A6 closes D-KR2b cleanly and is mostly built already |
| Grade **E-KR1 NOT MET**, un-drop its two backlog items | The class genuinely survives |
| Grade **Q-KR2 UNAUDITABLE**, **Q-KR3 PARTIAL** | Honest, and neither is repairable retroactively |
| Attended: `task prod:deploy` → `task version` → read the version line on a returning client | Closes D-KR2a **and** D-KR2b, once A6 has landed. No N/A half, no screenshots |
| Defer **A5** (`sync-live-fill-view`) to the next cycle | Not authored, carries 8 riders, and rests on a premise decision 126 does not actually state (§3 A5) |

**Projected close: 9 MET · 1 PARTIAL · 1 NOT MET · 1 UNAUDITABLE**, from 6 MET today — without
building the cutover, and without a single grade that a reader could call generous. The headline
count is unchanged by A6, but **D-KR2 now closes whole instead of MET-with-an-N/A-half**, and the
KR stays gradeable in every future cycle rather than only in cycles that ship a visible tab.

**Cost:** one night (A1 + A2 + A4 + A6 ≈ 210–345m serial + 30m closeout — a full night, tighter
than the three-card version; **A6 is the stretch card** if the night runs long, since D-KR2b is the
only thing waiting on it), plus ~15 attended minutes for the deploy, plus a sign-off on the §4
reword text.

---

## 6. What this plan will not do

Recorded so it is not quietly done later:

- **Backfill `slate-20260801.md` or `slate-20260802.md`** to satisfy Q-KR3. Signed artifacts record
  what was believed at signing.
- **Reword E-KR1** to make a surviving bug class grade green.
- **Claim red-first evidence for past WOs** from HANDOFF prose. The KR names a field; the field did
  not exist.
- **Retire any armed red.** B-27 and the three other armed reds stay armed — passing retires nothing
  (decision 100 / T-31 decision 120).
- **Cite any full-suite figure from this milestone as stable** until A1 lands. Per decision 131, a
  green can be an artifact too.

---

## 7. Open questions for the operator

1. **Sign off on the §4 reword text?** Nothing in `okrs.md` changes until you do.
2. **D-KR2b — N/A or NOT MET?** The precondition never fired. Precedent for N/A exists: QA KR4 last
   cycle, *"no schema migration shipped"* (`ledger.md:510`).
3. ~~**Does the §3-A5 premise hold?**~~ **ANSWERED 2026-08-03 by reading `client.js:1132-1195`** —
   push is unconditional, and it is the only thing that populates the substrate. See §3 A5. The
   residual question is a product one and it is yours: **is a live overlay of post-cutover edits
   between two RxDB devices the feature you wanted**, given it does not serve historical rows and
   does not replace REST hydration? If yes, A5 is buildable at a smaller size. If you expected RxDB
   to serve the fill view's actual state, that needs a data plane and it is a milestone, not a card
   (decision 126 option (i)).
4. **Build tonight, or close first?** This plan assumes build-then-close. Closing first and folding
   A1/A2/A4 into the next cycle is equally defensible and costs one less night.
5. **Carry §8 ("Two stores") into the next roadmap round as an activity?** It is milestone-sized,
   not card-sized, and it is decision 126 option (i) — the shape triage assessed as *"the honest end
   state but a milestone rather than a card."* Nothing in this plan builds it. The ask here is only
   whether it becomes the next cycle's named destination.

---

## 8. Target architecture — two stores

> **Origin:** operator, 2026-08-03 evening, during review of this plan. **Status: a destination, not
> a proposal for tonight.** Nothing in §3 builds it. It is recorded so A4's note can cite a named
> endpoint, and so the next `/nc-roadmap-round` has a shaped candidate rather than a vague "finish
> the sync thing."

**The statement:** *RxDB is the definitive source for the Operations app **after it loads**.
Workbox owns everything else.*

Two things in that sentence are load-bearing and should survive any rewording:

- **"after it loads"** encodes ownership rule 1. Workbox still delivers the app — including
  `vendor/rxdb.bundle.js` — and RxDB still cannot bootstrap itself. This is not a step toward
  removing Workbox, and no future card may read it that way.
- **"the Operations app"** draws the line **per-app, not per-technology.** Inventory, Onboarding,
  Users and Purchasing stay wholly Workbox's; Operations (`workflows.html`) goes wholly to RxDB.
  A product-level boundary holds under pressure in a way a technology-level one does not — "is this
  static or dynamic?" erodes at the first ambiguous case; "is this the Operations app?" does not.

### Where the 8 classes land

| # | Class | Fate under §8 |
|---|---|---|
| 1 | App shell (31 files) | → **Workbox**, unchanged |
| 2 | API responses, 4 non-checklist apps | → **Workbox**, unchanged |
| 3 | Checklist API responses in `api-cache` | ❌ **eliminated** — RxDB is the source |
| 4 | Identity partition key | → Workbox machinery (reclassify; not a store) |
| 5 | `localStorage` `hq_apps` | ⚠️ **residual — see below** |
| 6 | Pending submissions (`submitQueue`) | → **RxDB** — a pending write is an unpushed document |
| 7 | Lamport clock (`syncMeta`) | ❌ **evaporates** with the op-log |
| 8 | Replicated checklist rows | → **RxDB** |

**Two stores, plus one documented exception.** Class #5 is a genuine chicken-and-egg:
`bootstrap.js` reads the cached grant list to decide *what to replicate*, so it must be readable
before RxDB exists. It could move to the Cache API; that buys almost nothing. The honest end state
is **two stores and one ~20-line exception, written down** — not a fiction of two.

### Benefits

1. **The fetch-storm class dies structurally rather than by gating.** `sync.js:443-454` re-fetches a
   whole list per replayed op *because a whole response is its only storage unit* (ownership rule 3).
   Records make partial update possible, so the storm has no mechanism left. The file goes and the
   class goes with it. 🛑 **This is the only path on which E-KR1 can ever grade green honestly** —
   see §4's no-reword ruling.
2. **One staleness rule instead of four.** Today: `api-cache` NetworkFirst/10s · RxDB checkpoints ·
   `submitQueue` drain-on-reconnect · `localStorage` never expires.
3. **The Lamport layer evaporates** — not only `syncMeta`, but `hub.go`, `listener.go`, the clock
   logic, and decision 92's territory of impossible same-transaction guarantees.
4. **One conflict rule.** Decision 113 (uncontested delete wins; the loss is reported and
   recoverable) becomes *the* rule, rather than one of three alongside `submitQueue`'s ad-hoc
   later-press-wins and `api-cache`'s no-concept-of-conflict.
5. **The sync banner becomes truthful.** `renderSyncBanner` counts only `submitQueue` and cannot see
   unsent field ops or unpushed RxDB documents. One store ⇒ one pending count the crew can trust.
6. **The identity-partitioning machinery simplifies.** `cacheKeyWillBeUsed` / `cacheWillUpdate`
   exist because *responses do not know who they belong to*. Rows do — RLS plus `answered_by`.
7. **E-KR3 grades trivially and permanently** — two classes, two owners, zero ambiguity, every cycle.
8. **The founding user story becomes answerable.** *"A crew member says their answers vanished"* goes
   from five places to look to one.

### What it costs — stated plainly, because this is the expensive shape

🛑 **This is decision 126 option (i), *"Substrate becomes the truth source."*** Triage assessed it
as *"the honest end state but a milestone rather than a card,"* and nothing here contradicts that.
Making RxDB definitive means `/submit`, `/myChecklists`, approvals, every report and the
sales-processor period-summary endpoint can no longer read HQ's Postgres for the four replicated
tables. That requires repointing HQ's Go read paths at the substrate, or building a substrate→HQ
data plane — **F-1 again, and its largest option.**

Three specific bills:

- **HQ cannot read checklists while the substrate is down.** `0002_hq_fdw.sql`'s header already
  accepts HQ sitting on the network path of every row check; this adds the converse.
- **B-68** — `backend/internal/sync/{ops,handler}.go` carry **all** workflow validation
  (`main.go:47-95` routes `SUBMIT_CHECKLIST` → fail-note → resubmit-photo → archived-template
  validators). Retiring them requires RLS or trigger equivalents, not deletion.
- **`submitQueue`'s earned logic** — per-call idempotency keys, `currentSubmitPeriod()` stamping,
  later-press-wins by `queuedAt` — must be re-expressed in RxDB's push, not assumed away. Each rule
  exists because something went wrong once.

### What is available before that milestone: effectively nothing

Checked class by class, and they are locked together: #3 cannot go until the page reads from RxDB;
#7 cannot go until the op-log retires; #6 cannot move while writes stay on REST (decision 126). The
only free move is **reclassifying #4** as Workbox machinery rather than a store — 8 classes → 7, by
accounting rather than by change.

**So §8's value tonight is not a reduction anyone can bank. It is that the architecture now has a
named destination** — which is what A4's target-state column cites, and what a roadmap round would
turn into an activity.
