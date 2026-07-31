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
| `sync-rxdb-conflict-notice-ui` | C2 | _(in flight at time of writing — see below)_ | — | — |

**Every card ran three phases, not one:** implementation → fresh-subagent G6 adversarial review →
fix round. **All three G6 reviews returned APPROVE WITH FINDINGS, and every one of them found at
least one blocking defect.** No card merged on its first submission. That is the gate working.

---

## Gate evidence on the FINAL MERGED TREE

Run by the orchestrator on `overnight-20260801` with **fresh isolated databases**
(`hq_final_go`, `hq_final_e2e`, `TEST_PORT=8290`) — **not inherited from card reports**.

- **G1** — `go build ./...` **exit 0**; `go vet ./...` **exit 0**.
- **G2 (Go)** — `go test -p 1 -count=1 ./...` **exit 0**. 9 packages `ok`, 0 failed.
  **DB liveness proven, not assumed: 51 tables, goose version 73** on a freshly created database —
  which is the real proof that A1's `0072` and B2's `0073` coexist and apply in order.
- **G2 (Playwright)** — `npx bddgen` **exit 0**; `npx playwright test --retries=0` **exit 1**.
  **657 passed / 6 skipped / 1 failed of 664**, 23.1m, **23 spec files** (22 static + 1 generated).
  **Exactly one summary block — the run is not VOID.**

  🛑 **The single failure is the ARMED RED**, by full title:
  `list page progress decrements when another device unchecks a field [LST-17]`
  (`tests/sync.spec.js:446`). Expected `0/1`, received `1/1 items` — i.e. the live bug it exists to
  hold open. **This is the expected state, and its firing is the run's own proof the suite is not
  silently green.** I am not calling G2 Playwright "green": it exits 1, and the sole failure is the
  test that is supposed to fail.

  Verified alongside it: the bare tag `[LST-17]` matches **two** tests, which is why the slate
  mandates the full title; and `tests/sync.spec.js:1198` is a bare `}` — **dead, and correctly not
  armed**.

  **Neither B-27 nor B-30 fired in this run.** Neither is retired by that.
- **G4** — `node build-sw.js` idempotent (tree clean on a second run), **27 files / 2027.3 KB**.
  Version parity: `version.go Frontend = 1.3.0` ≡ `package.json 1.3.0` ≡ `version.json 1.3.0`.
  `Backend` unchanged at `0.3.0`.
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

**B-28 was filed twice.** A1 and C1 independently picked the next free number without seeing each
other — the signature of concurrent dispatch. Both entries survive; C1's was renumbered to **B-31**.

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

1. **Review the run branch on its merits** and decide the two forks above. Fork 2 is the one with an
   external counterparty and a payroll consequence.
2. **The attended two-device convergence check has re-armed** — frontend files moved and `sw.js` was
   regenerated. Runbook: `reference/attended-two-device-check.md`.
3. **Coordinate with sales-processor** before anything deploys. Two notices, not one.
4. **Decide whether `purchasing.spec.js:1792` earns a backlog number.**
5. **Disarm `HQ_SYNC_REST_URL` only on evidence**, if you judge B2's suite sufficient — the run does
   not get to assert it.
