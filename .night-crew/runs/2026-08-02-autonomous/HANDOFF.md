# HANDOFF — run 20260802 (Night A of a two-night milestone close)

**Branch:** `overnight-20260802` (cut off `dev` @ `812bf84`; **not merged to `dev`**; nothing pushed; nothing deployed)
**Slate:** `.night-crew/knowledge/reference/slate-20260802.md`, signed by the operator 2026-08-01, 6 cards, no stretch
**Dispatch:** CONCURRENT, 2 tracks, one in-flight card per track.

🛑 **Nothing deployed.** `task prod:deploy` was never run, by any leg, under any outcome.

---

## Per-card outcomes

| Card | Track | Outcome | G6 | Merge |
|---|---|---|---|---|
| *(pre-step — not a card)* | — | **LANDED** — P-KR2, D-KR3, E-KR1 closed | n/a (docs only) | clean |
| `sync-replication-scope-per-checklist` | **A1** | **LANDED** | APPROVE WITH FINDINGS — **2 BLOCKING** → fixed | 1 conflict |
| `sync-rxdb-write-policies` | **A2** | **LANDED** | APPROVE WITH FINDINGS — **3 BLOCKING** → fixed | 1 conflict |
| `sync-cache-and-identity-hygiene` | **B1** | **LANDED** | APPROVE WITH FINDINGS — **2 BLOCKING** → fixed | 2 conflicts |
| `build-deploy-manifest-integrity` | **P1** | **LANDED** | APPROVE WITH FINDINGS — **no blocking** → tightened anyway | 1 conflict |
| `workflow-unsubmit-failnote-reattach` | **P2** | **NOT STARTED** — budget | — | — |
| `sync-banner-builder-tab-scope` | **P3** | **NOT STARTED** — budget | — | — |

**4 of 6 cards landed. Nothing parked. No card raised an operator-only fork.**
**All three of Night A's milestone (white) cards landed:** A1, A2, B1.

**Every card ran implement → fresh-context G6 → fix round → merge.** Four G6 reviews, **three of
them found blocking defects**, and **no card merged on its first submission.** That is the gate
working, not the cards failing — and it matches 2026-08-01 exactly.

---

## Gate evidence on the FINAL MERGED TREE

Run by the orchestrator on `overnight-20260802` @ `a9e2018`, fresh isolated databases
(`hq_n802_fin2_*`, `TEST_PORT=8331`) — **not inherited from card reports.**

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **exit 0** · `go vet ./...` **exit 0** |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` **exit 0**, all **9 packages `ok`** |
| **G2 (`internal/sync`)** | `-run TestRowVisibilityRLS -v` → **54 subtests RUN, 54 PASS, 0 SKIP**, with **`HQ_SYNC_SUBSTRATE_OPTIONAL` unset** and `HQ_SYNC_REST_URL` unset. Cited this way per decision 108 **as amended tonight** — the package `ok` line is still not evidence (see DECISIONS-NEEDED **R3**). |
| **G3** | **N/A** — preflight `openspec: absent`. No `openspec/` created; verified. |
| **G4** | `node build-sw.js` after the merge commit (B-37 ordering): **31 files precached (2139.2 KB)**, `18 files parsed, 30 local references resolved, 0 outside the precache`. Idempotent, `sw.js` unchanged. Parity `version.go Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**. |
| **G2 (Playwright)** | `npx bddgen` exit 0 · `npx playwright test --retries=0` → **exactly ONE summary block: `1 failed · 6 skipped · 762 passed (24.5m)`** of 769. |

### 🛑 The baseline, reported honestly — and it is NOT "green except B-27"

**B-27 PASSED.** `tests/inventory.spec.js:883 › Inventory › item modal pre-fills search with
current line item text` — green in **every** run tonight, across five legs and seven-plus full
suites. **Per decision 100 this retires nothing; B-27 stays armed**, and no card claimed it fixed.

**All four armed reds passed**, matched by FULL TITLE (never line anchor):
- `list page progress decrements when another device unchecks a field [LST-17]` (`sync.spec.js:446`)
  — and its sibling `…updates when another device completes a field [LST-17]` (`:1006`); the bare
  tag does match two tests
- `a queued submission still lends its idempotency_key at 7:30pm CT [A1-TZ-02]`
- `submitted checklist survives builder edit with assignment change [LC-02]`

**The single survivor, by FULL TITLE:**
`tests/sync.spec.js:1343 › Convergence matrix (W-3): surviving answers converge across devices ›
yes/no answer converges (live + catch-up)` — failed at 14.8s in the suite, **passes in isolation at
4.1s.** It is a **sixth distinct title** in the rotating family, not a new deterministic red, and
**it is not laundered as "not flaky"** — see `B-45`.

### 🛑 The first final-gate run was contended, BY THE ORCHESTRATOR, and is discarded

The first attempt returned **7 failed / 6 skipped / 756 passed in 51.7m**. Six of the seven were
28–34s timeouts. Cause: I ran the full Go suite and the RLS suite **concurrently with it**, having
asserted the box was quiet from a single point-in-time check taken before those started.

Same tree, same commit: **24.5m / 1 failure quiet vs 51.7m / 7 failures contended.** That is the
most controlled data point yet for `B-45` and it sharpens the entry's claim — what moves this
suite's distribution is **CPU starvation, not test flakiness.**

A second attempt then **never ran at all** for ~2h20m: a wait loop's `pgrep -f 'go test|playwright
test'` matched **its own command line** and blocked forever. Recorded because a command that never
ran and a clean result are indistinguishable unless someone says which one they saw.

---

## What is NOT done — every bullet names its destination (B-38)

| Item | Destination |
|---|---|
| **P2 `workflow-unsubmit-failnote-reattach`** (B-19) — data loss in the accountability path; needs a back-and-reopen test in `tests/persistence.spec.js` per CLAUDE.md's persistence rule | **Night B slate** — to be sized at Night B's planning against S1/P4/P5/P6. If Night B cannot hold it, **next milestone**. Its `BACKLOG.md` entry still reads `· promoted → P2 (slate-20260802) ·` and is accurate: promoted, not started. |
| **P3 `sync-banner-builder-tab-scope`** (B-20) — Queued badge painted on Builder-tab rows | **Night B slate**, same condition. Smallest card on the slate (30–50m); the natural first thing to add if Night B has room. |
| **D-KR2's returning-client verification** — P1 fixes the manifest; parity on a *returning* client was **not observed**. Nothing was deployed, and `serviceWorkers: 'block'` is repo-wide (B-15), so no test in this repo installs a real worker for `log.js`/`tab.js` | **The attended hour after Night B** — a deploy plus 2/2 tab screenshots on a returning client. The slate already says milestone close contains a release. |
| **The ~23 ms/row constant** (A1) — not re-measured; needs production-like topology this box does not have | **Left open on A1's roadmap bullet under a 🛑**, not waived. Next milestone or the attended hour. |
| **`B-50` substrate isolation** — `HQ_RLS_TEST_DB` isolates only the HQ-side FDW database; the Supabase `public` schema and the single PostgREST have no isolation variable | **NEXT milestone (run mechanics).** Reddens no named KR. 🛑 Until it lands, a slate putting two substrate-touching cards on concurrent tracks buys a class of unattributable red. |
| **`B-54`** — precache count 31 is stated in prose and enforced by nothing | **`sync-hard-cutover`** — next card to edit `build-sw.js` and the first that may legitimately move the number. |
| **`B-42`, `B-43`, `B-49`** — Realtime stream unscoped; no scope shape for the two list entry points; `submission_rejections` now a fourth affected collection | **`sync-hard-cutover`** (Night B's S1). |
| **`B-39`, `B-40`, `B-41`, `B-44`, `B-45`, `B-46`, `B-47`, `B-48`, `B-51`, `B-52`, `B-53`, `B-55`, `B-56`, `B-57`** | **NEXT milestone**, per the scope freeze. Each carries its destination inline. |
| **`build-sw.js:29`** carries the same stale Taskfile citation P1 fixed elsewhere | **`sync-hard-cutover`** as a one-line drive-by, or folded into B-57. |
| **`card/d1-syncspec-deflake`** — holds a fix and its exact revert, provably net-zero (`git diff 8c71022 6ee45e0` empty) | **Triage** — a closeout note, not a blocker. Delete the branch or leave it; it strands no work. |
| **`hq-worktrees/b1-sync-rxdb-collections-and-table-contract`** — a worktree from an earlier run still present; `worktrees check` says its patches are in `dev` | **Triage** — removing another night's worktree is not a run action. It makes future guard output harder to read. |

**19 backlog entries filed tonight: `B-39` … `B-57`. Audited: no duplicate numbers.**

---

## What the night actually found — the four things worth the operator's time

1. **A1 was recording a data-loss bug as a settled decision.** It kept the replication scope *out*
   of `replicationIdentifier` and wrote that reasoning into the code comment **and** the roadmap
   bullet. RxDB keys its checkpoint by `[collection, replicationIdentifier]` alone, so one
   checkpoint spanned all scopes: open today's checklist, then yesterday's, and every row of the
   older one is `<= checkpoint` and filtered out **permanently**. Latent (nothing calls
   `startHQReplication` in production yet) — which is exactly why it had to be caught before S1
   wires it.
2. **A2's suite could not tell its own policies from mutants, and that IS E-KR2.** Every write went
   out `Prefer: return=representation`, so 0003's SELECT policy silently enforced the write half and
   `W8`'s 403 was the *read* predicate's. 3 of 5 mutations survived green, 2 of them mutations the
   file itself named as guarded. **The policies were always correct; the proof was not.** Fixed, and
   **E-KR2 is now met.**
3. **B1's mechanism was sound and its guards were theatre.** Deleting `cacheWillUpdate` left 13/13
   green; removing both `login.html` purge sites left 13/13 green while the test *named* for
   `login.html` measured `index.html`. Its G6 also attacked the threat model and proved the
   partition is **not a boundary against page script** — CacheStorage is same-origin readable and the
   token is forgeable. It defends the shared-device honest-user case, which is real and worth having;
   the in-code comment claimed more and was corrected.
4. **P1's guard found a live defect with no synthetic case.** `log.js` referenced by all 7 precached
   pages, `tab.js` by 5, **neither in `globPatterns`** — shipped online via `COPY *.html *.js`,
   broken offline forever. `tab.js` applies `#tab=N` before paint, so five of seven tools opened on a
   returning offline client with every section visible and no switching. That is **D-KR2's exact
   subject**, found by a card written to catch something else.

---

## Orchestrator errors, stated plainly

- **Two backlog-number collisions, both mine.** Three legs each claimed `B-39`; then I gave A2 and
  B1 overlapping allocations and both filed `B-46`. Cards cannot see each other's numbers —
  allocation is the control loop's job.
- **Legs shared one scratchpad directory** and clobbered each other's logs. A1's implementer reported
  contaminated Playwright numbers before catching it and retracting. Every leg after that was told to
  write logs inside its own worktree.
- **I dispatched A1's fix round into A1's worktree while its implementer was still alive**, which is
  what produced the "foreign suite running in my worktree" report.
- **I asserted a quiet box from a stale check, then created the contention myself** — and separately
  reported a gate as passing when its `webServer` had exited 1 and the harness still returned exit 0.
- **A wait loop deadlocked on its own `pgrep` pattern** and idled ~2h20m while I reported it queued.

---

## Next actions

**Lead: is Night B cuttable? YES.**

- **A2 landed**, so **S1 has policies to swap onto** — 0004's four write policies, the SELECT policy
  on `submission_rejections`, and migration `0074`'s approver view. `E-KR2` is met and evidenced.
- **B1's merge-intent states what its cache partition needs to survive**, corrected after its G6
  proved the original claims false. It names the exact hooks, the three literal spellings, and which
  test defends which. **S1 inherits a note that is now accurate rather than reassuring.**
- **P1 landed too**, which S1 did not require but benefits from: `build-sw.js` now fails the build
  when a precached file references something not precached, so S1 cannot silently drop an asset.
  🛑 **S1 must update the canary assertion in the same commit if it replaces a canary** — the spec
  now pins the count to 3 and the exact pairs, so a bare deletion reds.

**At triage:**
1. Review the run branch on its merits and merge to `dev`.
2. **Ratify or reverse the seven items in `DECISIONS-NEEDED.md`** — the two substantive ones are
   **R1** (B-13, plus four further false deploy claims in `CLAUDE.md` the slate never anticipated)
   and **R2** (P1's precache count 29 → 31 under a scope freeze).
3. **Decide P2/P3's home** before Night B is slated.
4. **`HQ_SYNC_REST_URL` stays armed.** It disarms only here, on evidence, never by a run asserting it.
