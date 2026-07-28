# HANDOFF — `overnight-20260729`

> **Run branch:** `overnight-20260729` (unmerged, unpushed) · **Slate:** `slate-20260729.md`,
> signed by the operator 2026-07-28, 4 cards, dispatch CONCURRENT.
> **Outcome: all 4 cards landed. Zero parked. Zero reverted.**
> Started 13:08, cards merged by 16:26 — ~3h20m against a serial estimate of ~8h15m.

---

## Per-card outcomes

| Card | Track | Outcome | Impl | Reviews | Merge |
|---|---|---|---|---|---|
| **A** `precache-manifest-from-head` | Wave 0 | ✅ DONE | ~75m | 1 × G6 → APPROVE-WITH-NITS | `25fbc16` clean |
| **C** `sync-proxy-endpoint` | B | ✅ DONE | ~25m + 2 repair rounds (~40m) | 3 × G6 → all APPROVE-WITH-NITS | `d73580d` clean |
| **B** `workflow-queue-period-and-failnote-upsert` | A | ✅ DONE | ~105m | 1 × G6 → APPROVE-WITH-NITS | `79fa7cd` **1 conflict resolved** |
| **D** `sync-rxdb-conflict-notice-mockup` | B | ✅ DONE | ~12m + 1 repair round (~25m) | 2 × restricted-input verifier → PASS-WITH-ISSUES, then sign-off-can-proceed | `4f75de2` clean |

**Merge order was A → C → B → D, not slate order.** Card B ran ~105 minutes (three full suites)
while Card C finished in ~25; holding C's merge would have serialised the two tracks the prompt told
me to run concurrently. Recorded in `conflicts-20260729.md` entry 3 rather than left to be inferred.

---

## What actually changed

**Card A — the precache manifest now reads the commit, not the index.** `build-sw.js` moved from
`git ls-files` to `git ls-tree -r --name-only -z HEAD`, so a staged-but-uncommitted file can no
longer enter the service-worker precache. This mattered because a precached URL that 404s fails the
**entire** SW install for every returning client, and `task prod:deploy` ships the *committed*
`sw.js` without regenerating it. `tests/sw-manifest.spec.js` test 1 moved off `ls-files` in the same
change — left on the index it would have kept agreeing with the bug.

**Card B — two data-integrity defects on the workflows path.** Fail notes now upsert instead of
appending (matching `ON CONFLICT` + migration `0071`'s unique index), so an approver no longer sees
the same corrective-action note twice. Queued-key reuse is bounded to the current period, closing a
path where a persistently-failing server let a stale key adopt a **later day's** submit. The
`workflows.html` comment claiming "the server upserts only the fields present" is now true rather
than edited away. And the "Pending sync" collision is resolved — **"Queued"** for a whole checklist
in the queue, **"Unsaved"** for a single field answer that hasn't reached the server.

**Card C — the same-origin `/sync/*` door.** A reverse proxy fronting PostgREST and Realtime,
including the WebSocket upgrade path, reusing the existing bearer/session middleware. **Inert in
every current deploy** — see D-2 before configuring it.

**Card D — the conflict-notice mockup.** 11 plates, 22 screenshots, all 20 `done_when:` rows
passing. This unblocks `sync-rxdb-conflict-notice-ui`, which stays ATTENDED-BLOCKED until you sign.

---

## Gate evidence on the final tree

Run by the orchestrator on the merged branch, not inherited from card reports:

- **G1** — `go build ./...` + `go vet ./...` **green**.
- **G2 (Go)** — **9/9 packages green** with real DB timings (inventory 22.7s, receipt 20.7s,
  recipes 21.9s, sync 19.2s) — *not* the silent-skip trap described in B-16 below.
- **G2 (Playwright)** — **563 passed / 0 failed / 0 flaky / 6 skipped of 569, 23.3m, exit 0.**
  Run on the final merged tree at `--retries=0`, against a **freshly DROP/CREATEd**
  `hq_test_e2e_final` on port 8210, after `npx bddgen`. **20 spec files** confirmed (B-09 count
  stated). Both known-armed reds (`sync.spec.js:446` LST-17 and `:1198`) **passed**.

  🛑 **The first attempt at this run was a false green, and it is worth recording as a third
  instance of the night's recurring hazard.** It reported **exit code 0 having executed zero
  tests**: Playwright's `webServer` could not start (`go: not found` — the background shell did not
  inherit Go's path), and piping the command through `tail` returned *tail's* exit status rather
  than Playwright's. Taken at face value it would have put a green suite in this document on the
  evidence of nothing. Same class as B-09 (`task test` silently running 19 of 20 files) and B-16 (a
  dropped database reading as a pass) — this time in the orchestrator's own verification step.
  The number above is from the corrected re-run.
- **G4** — `node build-sw.js` idempotent at the merge commit (22 files / 1468.9 KB, tree clean
  after). `version.go` `Frontend` ≡ `package.json` `"version"` = **1.2.2**. No manual SW cache bump.
- **Versions shipped:** Backend **0.2.2 → 0.3.0**, Frontend **1.2.0 → 1.2.2**.

---

## 🛑 Next actions, in order

1. **`/nc-morning-triage`** — review the run branch and merge to `dev`. Four open forks in
   `DECISIONS-NEEDED.md`; none blocks the merge.
2. **Attended two-device convergence check** — 🔴 **ARMED, and this run re-armed it twice.**
   Card A regenerated `sw.js`; Card B touched the submit path *and* renamed a user-visible badge
   class. Runbook: `.night-crew/knowledge/reference/attended-two-device-check.md`. Operator-owed;
   no run can discharge it (see B-15 — `serviceWorkers: 'block'` is repo-wide and must stay).
3. **Card D mockup sign-off** — D-3. The artifact is ready and three decisions inside it are yours
   to reject. `sync-rxdb-conflict-notice-ui` cannot be slated until you answer.
4. **Before any `task prod:deploy`** — re-run the duplicate check and record it in the deploy note:
   ```sql
   SELECT submission_id, field_id, count(*) FROM submission_fail_notes
   GROUP BY 1,2 HAVING count(*) > 1;
   ```
   Migration `0071` creates a unique index. If a duplicate reaches prod before the deploy, the
   migration fails → `os.Exit(1)` → `restart: unless-stopped` ⇒ **crashloop, prod down**. G6 proved
   the failure is *clean* (rollback, no index, goose stays at 70, rows intact), so recovery is a
   dedup plus a restart — but nothing tells the operator to look **before**. Probability is near
   zero (prod has had 2 submissions ever, 0 fail notes) and the bare index is the *instructed*
   behaviour under the card's PARK-over-improvise contract.
5. **Do NOT set `HQ_SYNC_REST_URL`** until row-visibility RLS lands — D-2.

---

## Follow-ups carried out of the run, not fixed in it

Each was found by adversarial review, judged non-blocking, and deliberately left rather than
widening a signed card. Filed as **B-16 … B-18** in `.night-crew/knowledge/BACKLOG.md`.

- **`build-sw.js:39` states something false** — the comment claims `--name-only` C-quotes paths
  containing a space; empirically it only escapes non-ASCII. `-z` is still correct and necessary;
  only the stated *reason* is wrong. Mirrored into `roadmap.md:383`, so the false claim is in two
  places.
- **A residual gap Card A cannot close** — `ls-tree HEAD` reads *local* HEAD while the image builds
  from `origin/main`. Commit a file locally, regenerate `sw.js`, push only the `sw.js` commit, and
  the 404 class returns. Strictly tighter than `ls-files` and correctly out of scope — but the DONE
  record shouldn't read as if the class is fully closed.
- **`proxy.go:257-264` describes code that isn't there** — a comment explains an
  `out.URL.RawPath = ""` assignment that does not exist. Harmless today; the hazard is the obvious
  "fix" (trimming RawPath in parallel with Path), which would make `EscapedPath()` return the
  caller's spliced bytes and hand the wire path to an attacker.
- **`proxy.go:203-205` logs the laundered path** — the rejection WARN records
  `r.URL.EscapedPath()`, i.e. the very function proven to launder `%2f`. `reason=encoded_slash` is
  the only surviving forensic signal.
- **`main.go:436-438` reads as an all-clear** — "inert until a deploy configures it", inches from
  the env-var names, with no "and do not configure it yet". Deliberately left alone to protect the
  Card B merge surface.
- **"10 packages ok" is 9** — nine `internal/` packages have tests. An inherited miscount (Card A's
  roadmap entry carries it too), immaterial to correctness.

---

## Incident: a reviewer destroyed a live card's test database, and it looked like a pass

Card C's first G6 reviewer **dropped `hq_test_go_c` during cleanup** while the implementer was still
using it. The implementer caught it only because it questioned a suspiciously fast green — the
DB-backed tests were **silently skipping**, not failing.

G6 later characterised the mechanism precisely: `internal/sync/access_test.go:29,33` and
`jwtbridge_test.go:169,173` `t.Skipf` on both connect *and* ping failure; `pgxpool.New` is lazy so a
missing database surfaces at `Ping` — as a **skip**; and non-verbose output is a bare
`ok ... 2.948s`, indistinguishable from a full run.

**This is the B-09 silent-green class one layer down: destroying the environment reads as passing.**
Every subsequent reviewer this run was given an explicit prohibition on dropping databases it did
not create, and all six card databases were verified present afterwards. Filed as **B-16**.

Housekeeping: the host now carries ~30 `hq_test_*` databases from tonight's and prior runs'
reviewers. Harmless, but per-agent database naming without cleanup is becoming a habit.

---

## What the reviews actually caught (why the G6 budget was worth it)

Three findings that would have shipped without adversarial review:

1. **Card C's signed premise was wrong.** The slate called the WebSocket upgrade "the part a naive
   `ReverseProxy` gets wrong". Card C built that baseline and found Go's stdlib handles the 101 *and*
   the bidirectional byte pump unaided; a naive proxy fails at **tenant lookup** (Realtime routes by
   the first dot-label of the Host header), not at the protocol switch. G6 reproduced this
   independently. The correction is recorded in the DONE card.
2. **A path-traversal hole, then a bypass of its own fix.** `..` segments reached the upstream
   un-normalised, including cross-room (`/sync/realtime/../rest/...`) carrying HQ's minted JWT — no
   live impact while both upstreams are path-less, but it activates the moment `HQ_SYNC_REST_URL`
   points at a gateway with a path prefix (`http://kong:8000/rest/v1`, the standard self-hosted
   Supabase shape). The fix was then itself found bypassable via Go's `EscapedPath()` fallback.
   Final state: 31 attack vectors all 400 with zero upstream hits, **and** 31 legitimate vectors
   through live PostgREST + Realtime with zero false rejections.
3. **Card D's mockup made a promise the mechanism can't keep.** The empty state rendered
   *"Nothing was overwritten."* — a flat guarantee — on the screen a crew member sees 99% of the
   time, in an app whose own limits panel says a non-leader tab, a missing subscription, or evicted
   storage all produce that identical screen. Its own contract forbade exactly that reading.

Two `done_when:` criteria were also caught being written so they **could not fail** — the 44px
touch-target criterion enumerated only classes already known to pass, which excluded `Undo` at
35×16, the sole escape from a mis-tapped Restore.

---

## Known-armed reds — status

Both **passed** in every clean full-suite run tonight (Card A's, Card B's, and the final-tree run):

- `tests/sync.spec.js:446` [LST-17] — load-sensitive; documented standing waiver (decision 44).
- `tests/sync.spec.js:1198` — ~16-20% flake.

`tests/sync.spec.js:1584`'s stale comment (B-06) was **not** folded in, as instructed — it belongs
to `sync-rxdb-schema-and-replication`, which is not in this slate.

---

## Milestone position

Activity 1 (Sync foundation) is **3 cards short**, and those 3 are the milestone's substance:

1. `sync-rxdb-schema-and-replication` — the re-scoped parent (8-12h). Now depends on Card C, which
   landed, so its obligation 6 is discharged.
2. `sync-rxdb-conflict-notice-ui` — ATTENDED-BLOCKED on D-3.
3. `sync-hard-cutover` — blocked on 1.

Tonight bought the door (C), the artifact that unblocks the UI (D), and two correctness debts
(A, B) that would otherwise have been paid inside the harder cards.
