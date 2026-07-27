# DECISIONS-NEEDED — run `overnight-20260727`

Nothing here blocked a card. **No card parked.** Every item below is either an operator-only
question the run correctly refused to answer, or a finding a reviewer generated that outruns the
card it was found in.

Items are numbered `D-<n>` continuing this run's own sequence; ledger decision numbers referenced
are from `.night-crew/knowledge/ledger.md`.

---

## D-1 — `build-sw.js` globs the git INDEX, not the commit. Operator call.

**Raised by:** Card A G6, finding F3. **Reproduced by execution.**

Decision 58's literal text says `build-sw.js` should glob "the tracked set (`git ls-files`)", and
that is exactly what shipped. But `git ls-files` reads the **index**, not `HEAD`. The reviewer
staged a file it never committed:

    git add zz-staged-only.html && node build-sw.js
    → 23 files precached, and `zz-staged-only.html` is in sw.js

`backend/Dockerfile:25` copies `*.html` from a clone pinned to `origin/main`, so such a file still
404s in production and still fails the **entire** service-worker install — the precise failure
decision 58 exists to prevent. `git ls-tree -r --name-only HEAD` would close it.

**Why the run did not just fix it:** changing it varies the literal text of a recorded operator
decision. The run executes decisions; it does not edit them. **Your call.** No test covers the
staged-file case either way.

---

## D-2 — Two more cross-tenant disclosures of the same shape as decision 57, both unowned.

**Raised by:** Card A G6, findings F6 and (from the implementer, outside its card) the login path.

Card A closed the `api-cache` disclosure on **logout**. Two siblings remain:

1. **`localStorage['hq_apps']` is never cleared on logout.** `logout()` deletes `api-cache` but
   leaves the previous user's permitted-app slug list, and the fail-closed branch still runs
   `filterTilesByPermissions(JSON.parse(cached))` with it (`index.html:208-209`). On a shared truck
   phone, offline, user B sees user A's tile set. UI-only — server-side grants remain the real gate
   — but it is the same disclosure class. One line (`localStorage.removeItem(APPS_CACHE_KEY)`)
   closes the logout half.
2. **An identity change without a logout is not covered by the logout clear.** If B logs in via
   `login.html` while A's session is live, `logout()` never runs. `checkAuth`'s eviction + buster do
   cover it, but `login.html` performs no cache hygiene of its own.

**Decision needed:** do these fold into `sync-rxdb-schema-and-replication` (which is expected to
retire `api-cache` entirely), into the grant-enforcement card, or into their own small card?

---

## D-3 — "Fail closed" means closed-on-FAILURE, never closed-on-WRONG-IDENTITY.

**Raised by:** Card A G6, finding F1. **Repaired in-card by narrowing the claim, not the code** —
recorded here because it is a real bound on what shipped, not a defect.

`identityVerified` is set by **any** 200. The client cannot tell whose 200 it is. G6 measured a
stale 200 rendering `Hi, Ghost Of User A` on **both** the pre- and post-fix trees. What Card A
actually closed is the *cache* path (eviction + cache-buster, both tested); the `removeUserHeader()`
branch is defense-in-depth against a future second render path and **can never fire today** — the
reviewer deleted the line and all three new tests still passed.

**No action required tonight.** Flagged so that nobody reads the roadmap card as "cross-tenant
identity is solved." It is not; it is bounded.

---

## D-4 — Card B's key reuse produces duplicate `submission_fail_notes` rows. Needs the parked file.

**Raised by:** Card B re-review, finding F-N7. **CONFIRMED by measurement**, not reasoned:

    RR05 rows=1 responses=[1] failNotes=[2]

`submission_fail_notes` has no unique constraint (migration `0013`) and its insert has no
`ON CONFLICT` — unlike the responses insert directly above it, which does. With key reuse, two
queue entries land on one `submission_id` and the fail note accumulates. An approver sees the same
note twice.

**Severity: LOW.** Duplication, not loss. Reachable by: enter a fail note **online**, go offline,
press Submit twice.

**Why the run did not fix it:** the fix is `ON CONFLICT (submission_id, field_id) DO UPDATE` plus a
unique index in `backend/internal/workflow` — **Card B's explicit park trigger**. Correctly out of
reach. It is a cost of the authorized key reuse, and the alternative (`id` reuse) is the data-loss
blocker G6 rejected, so this is the right trade — just an undocumented one until now.

---

## D-5 — Cross-period stale queue entry. Deliberate, disclosed, deferred by G6.

**Raised by:** Card B implementer (self-disclosed), adjudicated by Card B G6. **Reproduced.**

If `drainQueue` keeps failing with a non-409, non-network error it `break`s and leaves the entry.
Post-fix, a submit for that template on a **later day** adopts the stale key, so the server upserts
today's responses onto the older submission instead of creating today's row. Pre-fix that produced
two (correct) rows.

G6's ruling: **defer.** It is bounded — a successful drain (fires on `online`, on WS open, on
`visibilitychange`) clears it, so the unbounded case needs a *persistently* failing server. And the
state is visible: G6 verified the banner reads "1 submission pending sync" and the row carries a
"Pending sync" badge.

**Recommended follow-up card:** bound `findQueuedSubmission` to the current period and age out
stale `submitQueue` entries. **Fold D-4 into the same card** — both are consequences of key reuse.

---

## D-6 — `sync-rxdb-schema-and-replication` was NOT started. Deliberate. Two findings bind it.

See the HANDOFF for the reasoning. Two things must enter its slate entry before it is dispatched:

1. **🛑 Re-adding the vendored bundle to the precache will BREAK PRODUCTION unless the Dockerfile
   changes too.** Card A G6 finding F4, verified at source: `backend/Dockerfile` COPY lines are
   `21`, `25`, `26` (`icons`), `27` (`lib`), `30` (`backend`) — **`vendor/` is never copied into the
   image.** So Card C's obligation 5 ("re-add the `globPatterns` entry on adoption") re-creates
   exactly the bug Card A just fixed: a precached URL that 404s in prod, failing the whole
   service-worker install for every returning client. Card C must add `vendor/` to the Dockerfile in
   the same change set, or not re-add the glob.
   *This also means decision 59 was under-argued.* It was justified on bandwidth (−495 KiB); the
   real justification is that the base tree was shipping a broken SW install.
2. **`tests/sync.spec.js:1584` still carries the stale comment** ledger decision 66 folds into "the
   next card touching that file." Card B did not touch it. Still open.

---

## D-7 — Process: the orchestrator and the cards collide on `timings.log`. Durable fix wanted.

Both merges hit this — merge 1 as an untracked-file overwrite, merge 2 as a content conflict. Adding
an `ORCH ` line prefix after merge 1 kept the two families *distinguishable* but did not stop them
landing on the same offset in an append-only file. Both were resolved by union with nothing
discarded, but a third night should not pay for it again.

**Suggested:** the orchestrator writes `runs/<date>/timings-orchestrator.log` and the closeout
concatenates. Cheap, and it removes a guaranteed conflict from every future run.

---

## D-8 — B-105 remains open: hq has no per-change discipline.

Unchanged from previous runs, restated because both cards had to be told what to assert. The
`night-crew workflow preflight` verdict is **ABSENT** (the subcommand does not exist in the deployed
binary at `~/go/bin/night-crew`) and hq has no `openspec/` tree. Both cards correctly asserted only
the universal mechanics (red-first, atomic commits, a `Night-Crew-Card:` trailer, roadmap flip) and
neither created an `openspec/` directory.

**Note for the record:** `main`'s *source* now carries `cmd/nightcrew/workflowcmd.go`, so the
subcommand exists on main but the installed binary predates it. The verdict is ABSENT either way,
and no mid-ritual rebuild was performed.
