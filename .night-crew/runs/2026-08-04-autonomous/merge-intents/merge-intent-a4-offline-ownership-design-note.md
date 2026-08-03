# Merge intent — A4 `offline-ownership-design-note`

- **Run:** `overnight-20260804`
- **Branch:** `card/a4-offline-ownership-design-note`
- **Base commit:** `2041477` — every diff claim in this note is measured against
  `2041477`, never against run-branch HEAD (T-32 decision 130a).
- **Closes:** reworded **E-KR3** (`okr-completion-plan-20260804.md` §4).
- **Specified content:** `okr-completion-plan-20260804.md` §3 A4, lines 143–261.

Written BEFORE the deliverable (§15ad.65). Updated in place only for facts that changed.

🛑 **Updated 2026-08-04 for the G6 fix round (APPROVE-WITH-NOTES).** Three findings applied, four
backlog entries filed, **no code changed**. The largest is a **ninth class** — the note now publishes
**9 classes across 6 stores**, not 8. See §"The G6 fix round" at the bottom, and item 1 under "What
must survive any merge", which has been revised rather than left contradicting the deliverable.

---

## Red-first

n/a — no code change

*(Q-KR2 as reworded in `okr-completion-plan-20260804.md` §4, ledger T-33 decision 132. The
deliverable is a design note; the section is written explicitly so that "absent" and "not
applicable" are distinguishable at triage. No synthetic red was invented to fill it.)*

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `.night-crew/knowledge/designs/offline-ownership.md` | **New. The card's own deliverable.** The single written answer to "when a phone is offline, who owns each piece of what is on it." Nothing else in the repo carries this. |
| `.night-crew/knowledge/roadmap.md` | The card's status flip, required in the same change set. |
| `.night-crew/knowledge/BACKLOG.md` | **Added in the G6 fix round.** Four filings the round produced and deliberately did **not** fix: **B-88** (rule 2's guard is bypassable and already bypassed), **B-89** (🛑 a live code bug — `cachedGrantSlugs()` returns `[]` unconditionally), **B-90** (`build-sw.js:457` says the precache is 29; it is 31), **B-91** (three sites still say `hq_offline_v1` holds one store). Append-only; no existing entry edited. |
| `.night-crew/runs/2026-08-04-autonomous/merge-intents/` | This note. |

**Outside the slate's stated footprint:** `BACKLOG.md`, added in the G6 fix round. The slate's
footprint is `designs/offline-ownership.md` + `roadmap.md`; the merge-intent note is universal
mechanics, and filing findings to `BACKLOG.md` is the standing disposition for anything a card finds
and is forbidden to fix. **The append is additive only** — four new `B-` entries at end of file, no
existing entry touched — so it cannot conflict with another leg's filings except by adjacency.

**Explicitly NOT touched — this is a docs-only card.** No `backend/**`, no `*.html`, no `*.js`,
no `sw.js`, no `build-sw.js`, no `package.json`, no `backend/internal/version/version.go`, no
`night-crew.toml`, no `tests/**`. `git diff 2041477 --stat` names only files under
`.night-crew/`. **Re-verified at landing** rather than asserted — the stat output is quoted in
the card report.

🛑 **`sw.js` is byte-identical to base.** `node build-sw.js` was run twice as the G4 sanity leg
and produced no diff either time; neither file this card touches is precached, so the precache
count holding at **31** is the expected result and a move in it would have been a finding.

---

## What must survive any merge

1. **The count.** ~~The note publishes **8 classes across 6 named stores**.~~ 🛑 **REVISED in the G6
   fix round: the note publishes **9 classes across 6 named stores**.** The struck number is left
   visible because this note is the artifact triage reads, and a merge-intent that quietly changes
   its own number is exactly what item 7 forbids in the deliverable. **The signed non-negotiable
   ("publish all 8 classes across 6 stores") is satisfied, not violated** — all 8 planned classes are
   published unchanged, at the same 6 stores, and a ninth was found and added. The **store** count
   did not move: #9 lives in the *same* RxDB/Dexie `hq_sync` database as #8, as a separate
   collection. E-KR3's own parenthetical names two (*"static assets → Workbox, checklist data →
   RxDB"*); the count *is* the finding, which is precisely why it may go **up** but must never be
   collapsed back toward two. A merge that drops row #9 to restore the signed number destroys the
   deliverable in the same way, and would be the laundering §4 exists to forbid.
2. **All three splits.**
   - `hq_offline_v1` is **two** classes — `submitQueue` (#6) and `syncMeta` (#7) — because they
     have **different fates**: retire the op-log and `syncMeta` is dead weight while
     `submitQueue` survives untouched. A row reading "`hq_offline_v1` → `sync.js`" conceals that.
   - `api-cache` is **two** classes — #2 (non-replicated apps) and #3 (checklist responses) —
     split by whether RxDB replicates the underlying rows. #2 is uncontested forever; #3 is the
     only class that can ever become dual-owned.
   - 🛑 **The RxDB/Dexie database is two classes** — #8 (replicated buffer) and #9
     (`conflict_records`) — **split by whether a server copy exists at all.** #8 is a copy of rows
     that also live in the substrate and rebuilds on the next pull; #9 is `replicated: false`,
     `local: true`, **no `table` key at all**, swept client-side at 30 days with no server-side
     retention job *because there is no server table*. It is the durable local record of a crew
     member's **overwritten** edit — wipe the phone and the only copy is gone. Added in the G6 fix
     round; this is the split the first draft missed.
3. **Rules 1–4 stated as rules**, not as observations and not as prose. Rule 2 in particular is a
   **prohibition** a future implementer must hit: nothing may read from RxDB on a code path that
   can execute offline. 🛑 **Its *enforcement* paragraph was corrected in the G6 fix round and the
   correction must survive.** The note used to say the rule was *"enforced structurally"* by
   `tests/sync-rxdb-client.spec.js:1468-1470` and that *"that test is the rule's only enforcement."*
   Every clause was literally true and the conclusion was false: the test asserts on the **source
   text** of `workflows.html` for three symbols (`HQSync.createDatabase`, `HQSync.startReplication`,
   `HQSync.client`), and `workflows.html:3588-3592` already reads RxDB through **`window.HQSync.db`**
   — a fourth route the guard does not watch — on a page-load path mounted eagerly at `:3639` that
   runs offline. **Rule 2 is gated today by the database not existing, not by that test.** A reader
   following the old sentence would guard the wrong line. The rule itself is unchanged; only its
   enforcement story was wrong. Code-side gap filed as **B-88**, not fixed.
4. 🛑 **The class with no owner, in those words.** REST writes land in HQ's Postgres, RxDB push
   writes land in the substrate, **nothing reconciles them**. Suppressing this to make the table
   look clean is the laundering case the reworded E-KR3 exists to forbid.
5. 🛑 **Rule 1's two prohibitions.** The note must not claim RxDB "handles offline" generally and
   must not describe Workbox as transitional. Both are permanent, and the mechanism is at source:
   Workbox precaches `vendor/rxdb.bundle.js`, so RxDB cannot bootstrap itself.
6. **The target state cites §8 by name**, not "a future card". §8 is decision 126 option (i),
   which triage assessed as *"the honest end state but a milestone rather than a card."* The note
   **describes** that destination; it does not adopt it. That distinction must survive.
7. **The stated deviations.** ~~Six~~ **Seven** places where measurement contradicted or refined
   `okr-completion-plan-20260804.md` §3 A4's table are recorded in the note's own Deviations
   section, each with the file:line that settles it. A merge that quietly drops them leaves the
   next reader unable to tell a verified row from a patched one. 🛑 **D7 is the new one and the
   largest**: the ninth class. It is stated as an **inherited** omission and must stay stated that
   way — the plan's §3 A4 table has **8 rows**, the plan's §8 mapping table *independently* has
   **8 rows**, and `conflict_records` has **0 hits in the whole plan**. Both tables missed the same
   class; the implementer did not invent it and did not introduce the error.
8. 🛑 **The two in-place strikes.** Two wrong sentences are struck with `~~…~~` plus a dated
   correction block — the convention A1 and A2's fix rounds used — rather than rewritten: rule 4's
   footnote *"not a ninth class — a class needs a store, and this has none"*, and rule 2's
   enforcement paragraph. **Both must stay visible.** This note is the document someone opens when
   data goes missing; a record that quietly changes its story teaches nobody. Note also that the
   footnote's argument is preserved as *correct about in-memory `PENDING_VALUES`* — only the
   **absolute** was wrong, and the good half is said to be good.

---

## What is safe to drop

- Any wording, ordering or table formatting in `designs/offline-ownership.md`, provided items
  1–8 above survive in substance.
- The roadmap bullet's phrasing (not its existence).
- This note itself, after triage.

---

## What this card deliberately did NOT do

- **It did not decide the two-store architecture.** §8 is described and cited as the destination;
  adopting it is milestone-sized (decision 126 option (i)) and is an operator/roadmap call. The
  PARK trigger for "deciding a product fork" did not fire because the card describes rather than
  decides.
- **It did not run the full Playwright suite or the Go suite.** The diff is documentation under
  `.night-crew/`; neither suite can observe it. Gates run: **G4 only** (`node build-sw.js` twice,
  31 both times, clean tree both times, frontend 1.4.0 == `package.json` == `version.go`
  `Frontend`).
- **It did not create OpenSpec scaffolding.** Workflow preflight: `openspec: absent`. Which
  per-change discipline this repo adopts is operator question **B-105**, not this card's.

---

## The G6 fix round (2026-08-04) — APPROVE-WITH-NOTES

🛑 **Docs-only, as the card was. No `.html`, no `.js`, no `sw.js`, no `tests/**`, no `backend/**`.**
Two new commits on top of `cd6030f`; nothing rebased, amended or forced.

### 1. 🛑 The ninth class — published

`LOCAL_COLLECTIONS.conflict_records` (`sync-schema/collections.js:277-284`) exists on a phone, has a
store, and has **both** a shipped write path (`sync-rxdb/conflict-notice.js:788-796`,
`collection.upsert`) and a shipped read path (`workflows.html:3588-3592`, mounted eagerly at
`:3639`). It was **absent from the table** — while the note's own rule 4 footnote foreclosed the
question outright with *"not a ninth class — a class needs a store, and this has none."*

It is a **distinct class by the note's own splitting logic — different fates.** #8 is a replication
buffer *with* a server-side copy behind every row; #9 **never leaves the phone and has no server row
anywhere** (`replicated: false`, `local: true`, **no `table` key at all** — deliberately absent so
replication fails loudly; 30-day *client-side* sweep at `:269`; `:266-267`: *"The sweep is LOCAL …
There is no server-side retention job, because there is no server table."*). It is the durable local
record of a crew member's **overwritten** edit. Losing it loses the only copy, permanently.

**Published as row #9 in §1.** The **store count stays 6** — #9 is a separate collection in the same
`hq_sync` database as #8 (registered at `sync-rxdb/client.js:1096-1098`). §1b now carries **three**
splits; §3's glance table, §6's §8 mapping and §8's usage table all carry #9; §0 and every count
heading read **9 classes across 6 stores**.

**Why this mattered more than a missing row:** this note is the document someone opens when data
goes missing, and there was no row for *"the conflict banner said my edit was overwritten and now
the record is gone."* There is now, and it says to check that phone and no other.

**The `:167` absolute was struck IN PLACE** — `~~…~~` plus a dated correction block, the convention
A1 and A2's fix rounds used — not silently rewritten, and the correction credits G6. The footnote's
argument is preserved as **correct about in-memory `PENDING_VALUES` / `PENDING_SYNC`**; only the
**absolute** was wrong, and the note says so rather than discarding the good half.

**Stated as deviation D7, and stated as INHERITED.** The plan's §3 A4 table has 8 rows; the plan's
§8 mapping table *independently* has 8 rows; `grep -c conflict_records` over the whole plan returns
**0**. Both tables missed the same class. This is not the implementer's invention and not an error
introduced by this card, and D7 says that plainly.

🛑 **On the signed non-negotiable ("publish all 8 classes across 6 stores"): publishing 9 satisfies
it, it does not violate it.** All 8 planned classes are published, unchanged, at the same 6 stores;
a ninth was found and added. The card's governing rule is explicit — *where measurement contradicts
the plan's table, the measurement wins and the deviation is stated* — and **the count being the
finding is the note's whole point.** Dropping the ninth to preserve a signed number would be the
exact laundering §4 exists to forbid.

### 2. Rule 2's enforcement claim — corrected, rule unchanged

Struck in place and replaced. The old paragraph was literally true and misleading; the new one says
the test is a **partial guard, not the enforcement**, quotes `defaultStore()`, names **`window.HQSync.db`**
as the unguarded route, and states that **rule 2 is gated today by the database not existing.** The
first card that creates a database breaks the rule with no diff to anything the guard watches. Filed
as **B-88**; not fixed here.

### 3. Citation drift (G6 finding F3) — each re-verified at source before changing

| Was | Now | G6 correct? |
|---|---|---|
| `offline-save-honesty.md:12-13` | **`:13-14`** | yes |
| `offline-save-honesty.md:11` (submitOp/drainQueue) | **`:13-14`** | yes |
| "collapsed twice" | **three times** — third site named at `offline-save-honesty.md:142` | yes |
| `ledger.md:1045` attributed to **T-21d** | **T-21e** — T-21d is `:1025-1041`, T-21e begins `:1043` | yes |
| D5's span `ledger.md:1025-1046` | **`:1025-1041`** | yes (same finding) |
| D4's second site `sync.js:475-483` (mid-comment) | **`:480-482`**, agreeing with rule 3 | yes |
| `submitOp` `sync.js:781-796` | **`:781-797`** | yes |
| `build-sw.js:305-390` (`globPatterns`) | **`:307-389`** | **partly** — G6 said `:307-388`; the array's closing `],` is on **389**, so the file wins and the range runs to 389 |
| Row 4 writer `index.html:142-143` (two `const`s) | **`:187`**, the actual `cache.put`, inside `establishIdentity()` `:179-196` | yes |
| Row 4 readers `build-sw.js:459-486` | **`:459-485`** — `:486` is `handlerDidError`, a different hook | not flagged by G6; tightened while in the row |

**D5's underlying claim is unaffected and stands** — T-21d genuinely does not say Background Sync is
Chromium-only. Only the attribution of the Safari line moved.

### 4. Four backlog entries filed — filings only

**B-88** rule 2's guard is bypassable and already bypassed · **B-89** 🛑 a **live code bug** —
`index.html:232-235` writes `hq_apps` as `{uid, apps}` and `sync-rxdb/bootstrap.js:62-71` still
`Array.isArray`-gates, so `cachedGrantSlugs()` returns `[]` unconditionally on a real client, with
`tests/sync-rxdb-client.spec.js:1385` planting the bare-array shape so nothing reds (sibling of B-65
in kind) · **B-90** `build-sw.js:457` says the precache is 29; it is 31, inside the block arguing
what a gate would *not* notice (same block as B-59) · **B-91** three sites still say `hq_offline_v1`
holds one store.

🛑 **None of the four was fixed.** B-89 in particular is a **code** bug and this round is docs-only.

### 5. Two places the code disagreed with the review brief — the file won

- The `29 files` literal is at **`build-sw.js:457`**, not `:456`. B-90 cites `:457`.
- The 30-day retention constant is at **`sync-schema/collections.js:269`**, not `:268` (`:268` is a
  comment rule). Row #9 and the usage table cite `:269`.

### 6. Gates

`node build-sw.js` run **twice**: `SW built: 31 files precached (2162.2 KB)` both times, tree clean
on the second run. **`sw.js` byte-identical to base `2041477`** — verified by hash, not asserted.
`git diff 2041477 --stat` names only files under `.night-crew/`. Playwright and Go suites **not
run**, deliberately: this diff cannot affect either, and running one would risk colliding with
another leg.

### 7. Unchanged by this round

**`## Red-first` still reads `n/a — no code change`, and that is correct** — the fix round changed
no code either. No synthetic red was invented to fill it, in this round any more than the first.
