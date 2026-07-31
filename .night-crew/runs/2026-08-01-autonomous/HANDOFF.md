# HANDOFF — run 20260801

**Branch:** `overnight-20260801` (not merged to `dev`; nothing pushed; nothing deployed)
**Slate:** `.night-crew/knowledge/reference/slate-20260801.md`, signed 2026-07-31, 4 cards, 3 tracks, CONCURRENT
**Dispatch:** Track A, B and C concurrently, one in-flight card per track.

🛑 **NOTHING DEPLOYED, and Track A's work must not deploy alone** — it is half of a two-repo
agreement with sales-processor. Until both repos land, one of them is wrong.

---

## Per-card outcomes

| Card | Track | Outcome | G6 | Merge |
|---|---|---|---|---|
| `app-timezone-unify-new-york` | A | **LANDED** (resume of the 07-29 park) | APPROVE WITH FINDINGS → fixed | clean |
| `sync-rxdb-row-visibility-rls` | B | **LANDED** (resume of the 07-29 park) | APPROVE WITH FINDINGS → fixed | clean |
| `sync-rxdb-replication-and-conflict-handler` | C1 | **LANDED** | APPROVE WITH FINDINGS → fixed | 3 conflicts, all pre-resolved |
| `sync-rxdb-conflict-notice-ui` | C2 | **LANDED** | verifier **PASS 30/30** + G6 APPROVE WITH FINDINGS → fixed | clean |

**4 of 4 landed. Nothing parked.**

**Every card ran three phases, not one:** implementation → fresh-subagent G6 adversarial review →
fix round. C2 ran four, adding the CLAUDE.md verifier gate. **All four G6 reviews returned APPROVE
WITH FINDINGS, and every one found at least one blocking defect.** No card merged on its first
submission. That is the gate working, not the cards failing.

**The Track-C serialization paid for itself.** C2 was cut *after* C1 merged, so it developed against
the merged state — and the two cards sharing the RxDB client layer and `workflows.html` most heavily
produced **zero conflicts**. C2 also checked its backlog number against the merged run branch rather
than its own worktree, which is exactly the check A1 and C1 both skipped when they collided on B-28.

---

## Gate evidence on the FINAL MERGED TREE (all four cards)

Run by the orchestrator on `overnight-20260801` with **fresh isolated databases**
(`hq_final2_go`, `hq_final2_e2e`, `TEST_PORT=8292`) — **not inherited from card reports**.

- **G1** — `go build ./...` **exit 0**; `go vet ./...` **exit 0**.
- **G2 (Go)** — `go test -p 1 -count=1 ./...` **exit 0**. 9 packages `ok`, 0 failed.
  **DB liveness proven, not assumed: 51 tables, goose version 73** on a freshly created database —
  the real proof that A1's `0072` and B2's `0073` coexist and apply in order.
- **G2 (Playwright)** — `npx bddgen` **exit 0**; `npx playwright test --retries=0` **exit 0**.
  **733 passed / 6 skipped / 0 failed of 739**, 22.9m. **Exactly one summary block — not VOID.**
- **G4** — `node build-sw.js` idempotent (tree clean on a second run), **29 files / 2111.1 KB**.
  Parity: `version.go Frontend = 1.4.0` ≡ `package.json 1.4.0` ≡ `version.json 1.4.0`.
  `Backend` unchanged at `0.3.0`.

### 🛑 The armed red is itself intermittent, and that changes what it proves

An earlier gate run on the **three-card** tree exited **1** with a single failure: the armed red
`list page progress decrements when another device unchecks a field [LST-17]`
(`tests/sync.spec.js:446`), expected `0/1`, received `1/1 items`. On the **four-card** tree it
**passed**.

**This was checked rather than assumed, because the first check was inconclusive.** The gate script
piped Playwright through `tail -35`, which truncated the armed red's line out of the captured log —
so "passed" and "skipped" were indistinguishable in the evidence. That is the same defect class this
run was armed against, produced by the orchestrator's own instrumentation. Resolved directly: the
test is a plain `test(...)` with **no `.skip`/`.fixme`**, and a targeted run on the final tree
(`-g "list page progress decrements when another device unchecks a field"`, fresh DB, port 8294)
reported **`1 passed (2.4m)`** — so it genuinely **executed and passed**.

**Across tonight's Playwright legs `[LST-17]` fired in roughly three and passed in five.** It is not
skipped and it is not disarmed — but it is **load/order-sensitive, in the same family as B-27, B-30
and B-32**. The consequence is worth stating plainly:

> **"The armed red fired" is not reliable proof the suite is not silently green, and "the armed red
> did not fire" is not a disarm.** A tripwire that only trips sometimes cannot carry the evidential
> weight the slate assigns it. **Recommend triage consider folding `[LST-17]`'s flakiness into
> B-32**, or replacing it with a deterministic tripwire.

Verified alongside it: the bare tag `[LST-17]` matches **two** tests, which is why the slate mandates
the full title; and `tests/sync.spec.js:1198` is a bare `}` — **dead, and correctly not armed**.

**B-27, B-30 and B-32's members did not fire in the final run. None is retired by that.**
- **G4 discipline greps** — `internal/journal` and `internal/workorder` **do not exist in this
  repo**. Recorded **N/A-VACUOUS, not clean** (**B-14**, now the fourth consecutive triage).
- **G3** — **N/A**. Workflow preflight verdict is `openspec: absent`; no scaffolding was created.

---

## 🛑 Open forks — `DECISIONS-NEEDED.md`. Both are yours, not the run's.

**Fork 1 — does decision 92 survive the measured per-row fdw cost?**
G6 measured the RLS path at **~23 ms per row, linear** (5 rows → 177 ms; 205 rows → 4,698 ms).
Sign-off accepted *"HQ's Postgres is on the network path of every RLS row check"* — true, and it
does not convey per-row cost. RxDB initial replication pulls whole collections.
**This is the card's own named PARK trigger.** The run carried the work onto the branch with the
fork recorded instead of parking, reasoning that the branch is reviewed before it reaches `dev`.
**Overrule by rejecting Track B at triage if you disagree** — nothing has touched `dev`.

**Fork 2 — what is sales-processor told, and when? It is now TWO notices, not one.**
G6 **refuted at source** the card's claim that the `pending_review_ids` `COALESCE` "has been there
since Phase 21". Phase 21's contract was *accurate when written*; `COALESCE` entered **2026-06-06**
via quick task `260606-0gh` (`cf959bd`), which **changed which rows the completeness gate returns
and never updated the contract**. Consequence: under the published expression a late-discovered
receipt did **not** block payroll; under the shipped code it **does**. **Sales-processor may have
been receiving an undocumented `ready:false` since June 2026.** So there are two notices — the
timezone move, and this. Filed as **B-29**. **Fork stays OPEN.**

---

## New backlog items filed tonight

| ID | Card | Substance |
|---|---|---|
| **B-28** | A1 | `receipt/worker.go`'s `parseEventDate` stamps a COGS period from **server-local** time — the one path exempt from A1's zone unification is the path producing the value that wins the `COALESCE`. Silent when it happens. |
| **B-29** | A1 | The undisclosed 2026-06-06 completeness-gate drift (Fork 2). **An operator decision, not a code task.** |
| **B-30** | A1 | `[A1-TZ-02]` reds under whole-suite load, greens in isolation. Filed with its mechanism recorded rather than left as folklore. |
| **B-31** | C1 | `index.html`'s launcher **hides tools a user can reach**: it gates tiles on literal slugs, but per-tab grants are not literal slugs. A user holding only `inventory-trends` reaches the tab and the API and sees **no Inventory tile**. Pre-existing; belongs to whichever card owns the launcher. |
| **B-32** | C2 | The **load/scale-sensitive 30 s timeout family** — tests that redden at whole-suite scale and green in isolation. Files `[LC-02]` and `inventory.spec.js:2908` alongside B-27 and B-30. **Both refused attribution on evidence, neither retired.** |

**B-28 was filed twice.** A1 and C1 independently picked the next free number without seeing each
other — the signature of concurrent dispatch. Both entries survive; C1's was renumbered to **B-31**.

**`[LC-02]` was refused attribution on hard evidence, not on a hunch.** The same test reddened as a
30 s timeout on **2026-07-26**, on a different card's leg, on a tree where `workflows.html` and
`tests/workflows.spec.js` were **proven byte-identical to base** — five days before C2 existed. Its
recorded signature there was *"element was detached from the DOM, retrying"*. The reviewer's own run
then reproduced the class on `inventory.spec.js:2908`, a spec C2's diff does not touch at all.

---

## Standing flags

| Flag | State after this run |
|---|---|
| 🔵 **Attended two-device convergence check** | **RE-ARMED.** A1 and C1 both moved frontend files and `sw.js` was regenerated (22 → 27 precached files). Runbook: `reference/attended-two-device-check.md`. Automating it is B-15. **This is a morning attended follow-up, not a run task.** |
| 🔴 **`submission_fail_notes` duplicate check** | **ARMED** — re-arms before every `task prod:deploy`. Tonight added **two** migrations (`0072`, `0073`), so it is armed regardless of outcome. |
| 🔴 **`HQ_SYNC_REST_URL` must NOT be set** | **STILL ARMED.** B2 landed the row filtering, and B2's own review supplies evidence — but **the flag disarms at triage on evidence, never by the run asserting it.** Verified set nowhere in the tree by three separate agents. |
| 🔴 **Guard integrity B-22/B-23/B-24** | **Still armed, and vindicated tonight** — see below. |
| 🟡 **`go test -p 1`** | Held. Every agent ran it; no parallel-package failures. |

---

## What tonight actually proved about the guard-integrity flag

B-22/B-23/B-24 said a guard printing PASS is not evidence until its subject set is shown non-empty.
Tonight produced three independent confirmations:

1. **B2's own suite.** With the FDW repointed at a **migrated-but-empty** database, `count(*)`
   returned `0` **with no error**, and **12 of 19 numbered attack variants still passed**. Every
   variant that caught it caught it on its *positive* half. A refusal-only suite would have printed
   green. Reproduced independently by G6, character-for-character.
2. **A1's parity guard.** Renaming a constant so the regex missed it produced *"no app-timezone
   literal found — the parity check's subject set has gone empty, which is a failure, not a pass."*
   The guard bites on emptiness, by construction.
3. **The merge procedure itself.** Regenerating `sw.js` mid-merge emitted a **24-file** manifest
   instead of 27 — silently dropping `vendor/rxdb.bundle.js` — printed `skipped (not in HEAD)` and
   **exited 0**. `build-sw.js` reads **git HEAD**, not the working tree. That is Obligation 5's exact
   production-outage mode, produced by the *orchestration*, not by any card. Recorded in the conflict
   log; the regeneration must follow the merge commit.

---

## Known reds — none retired

- **B-27** (`inventory.spec.js:883`, cross-spec pollution): passed in four runs tonight and
  **failed once** at `--retries=0` in a G6 reviewer's run — on a **quiet** box (load 3.67), while
  the runs under heavy load passed. That is backwards from the load-sensitivity story B-27 carries,
  and worth a look. **Not retired.**
- **B-30** (`[A1-TZ-02]`): new tonight, reds under whole-suite load, greens 5/5 in isolation.
- **`purchasing.spec.js:1792`** (FR-23 repurchase-reset): failed once during B2's fix round with a
  **640 ms backwards jump between two server-side `now()` reads**, then passed in isolation.
  Observed, flagged, **not retired** and not yet filed — decide at triage whether it earns a number.

---

## What the concurrency cost

Three concurrent Playwright suites drove load to **17 on 8 cores**. Measured consequences:
- C1's first full run was **invalidated** — a killed Playwright survived `pkill` and raced its
  replacement on the same port and database, producing **two summary blocks under one header**. The
  card discarded it and re-ran rather than quoting the noise.
- Both resume cards found the shared `hq_test_go` / `hq_test_e2e` databases at a goose version the
  *other* card had set, and `hq_test_e2e` refused to boot. Each worked around it with its own
  databases rather than dropping databases it did not create (**B-16**).
- Every G6 review and fix round was therefore given a distinct database prefix and `TEST_PORT` by
  the orchestrator.

**Worth weighing at the next slate:** concurrent dispatch bought wall-clock and cost one invalidated
gate run, two database collisions, and a duplicate backlog number.

---

## Run-level notes

- **B-26 confirmed again.** The launch prompt inherits gates G1–G6 from
  `reference/overnight-run-plan-20260707.md`, **which does not exist in this repo** — it lives only
  in the night-crew clone, and the ladder there is night-crew's own (`go build`, `openspec validate`,
  greps for `internal/journal`). HQ's actual ladder was reconstructed from the 07-29-2 handoff's
  gate-evidence block and pasted into every agent prompt. This is **B-26**, already filed — recorded
  here as another instance, not as a discovery.
- **Stale `night-crew` binary.** The installed `~/go/bin/night-crew` (built 2026-07-23) has no
  `launch-prompt`, `run-evidence`, `worktrees` or `workers` verb. All four exist on **`main` @
  `258d723` (v3.0.2)**, so every guard was run from a binary built out of a pinned `main` worktree
  into scratch. **Nothing was built from `dev` and the operator's install was not modified.**
  Refresh with `task nc:update`.
- **Trailer drift, and it was the orchestrator's fault.** The run branch's seed commit used
  `Card:`/`Run:` rather than the repo's `Night-Crew-Card:`/`Night-Crew-Run:`, and A1's ten fix
  commits inherited it before it was caught. Corrected from that point on. Existing commits were
  **not** rewritten — amending would rewrite verified history for a trailer carrying no attribution
  the parents don't already have, and triage declined the identical trade in T-26 decision 86.

---

## Next actions — for morning triage

1. **Review the run branch on its merits** and decide the **three forks** in `DECISIONS-NEEDED.md`.
   **Fork 2 is the one to read first** — it has an external counterparty and a payroll consequence.
   Fork 3 is explicitly non-blocking; if you do nothing, C2 ships as signed and is correct.
2. **The attended two-device convergence check has RE-ARMED** — frontend files moved and `sw.js` was
   regenerated twice (22 → 27 → 29 precached files). Runbook:
   `reference/attended-two-device-check.md`. **This is an attended follow-up, not a run task.**
3. **Coordinate with sales-processor before anything deploys. Two notices, not one** — the timezone
   move, and the undisclosed June 2026 completeness-gate drift.
4. **Decide whether `purchasing.spec.js:1792` earns a backlog number** (640 ms backwards jump between
   two server-side `now()` reads; passed in isolation).
5. **Disarm `HQ_SYNC_REST_URL` only on evidence**, if you judge B2's suite sufficient. The run does
   not get to assert it, and did not.
6. **Consider what to do about `[LST-17]`.** It is intermittent (see the gate section), so it can no
   longer carry the evidential weight the slate assigns it. Fold into B-32, or replace it with a
   deterministic tripwire.
7. **Two C2 items that are yours, not the card's:** the **new mockup deviation** (production's dark
   confirm no longer matches the signed `a2-confirm-dark` plate, because V-1 was fixed in
   `workflows.html` and the signed plate deliberately left alone — SUMMARY.md §1a identifies the
   two-line change if you would rather re-sign it), and **F-4** — `PLAN.md` landed *after* the
   implementation, and it carries the contract the verifier gate grades against. Recorded in
   SUMMARY.md §4, not argued away.
8. **A version convention question, not a defect.** C1 took `1.2.2 → 1.3.0` and C2 took
   `1.3.0 → 1.4.0`, walking the frontend two minors in one night for one feature delivered in halves.
   Both bumps are individually correct.

## What is NOT done

- **Nothing is deployed, nothing is pushed, nothing is tagged, `main` is untouched.**
- **`sync-hard-cutover` inherits three notes**, all recorded rather than improvised: the `_deleted`
  delete-vs-edit product question (**severity corrected — the uncheck-vs-edit collision is the most
  likely conflict on this schema, not a corner case**), `makeSyncFetch` losing method and body on a
  `Request` argument, and `formatValue`'s now-bounded unwrap needing care once a network-fed producer
  exists.
- **RxDB push replication is still refused** — B2 landed SELECT policies only; writes remain
  deny-all until a follow-up card writes `WITH CHECK` policies.
