# HQ's verification ladder — the repo-local source of truth

> **This file exists because of B-26, and B-26 is a recurrence, not a one-off.**
>
> Slates kept inheriting G1–G6 from `reference/overnight-run-plan-20260707.md`, a file that **has
> never existed in this repo** (`git log --all` finds no trace) — it lives only in the night-crew
> clone, and the ladder in it is night-crew's own, not HQ's. `slate-20260803.md` caught this and
> wrote the ladder out in full under §"HQ's verification ladder"; `slate-20260804.md` then
> **regressed to the dangling pointer**, and the 20260804 orchestrator reconstructed it by hand for
> the fourth time.
>
> Inlining the ladder in each slate is what regressed. A repo-local file cannot: a slate now
> references **this path**, in **this repo**, and a broken reference fails visibly at authoring time
> rather than silently at 3am.
>
> **Decided at morning triage 2026-08-03 (ledger §T-34, decision 138).** Slates and launch prompts
> for this repo cite `.night-crew/knowledge/reference/gate-ladder.md` and nothing in the night-crew
> clone.

---

## The gates

| Gate | Command | Pass condition |
|---|---|---|
| **G1** | `go build ./...` · `go vet ./...` (from `backend/`, the module root) | both exit 0. 🛑 Run from `backend/` — `./...` from the repo root matches **no module** and, piped into `tail`, prints a false green (`card-actuals.md`, and again at triage 2026-08-03) |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` | exit 0, **and counts checked, not `ok`**. 🛑 `DB_TEST_URL` must be set or the suite exits 0 while skipping every DB-coupled test — `internal/workflow` runs **zero** tests and still prints `ok`. Expect 9 packages and ~439 tests; `internal/workflow` should run **35**. For `internal/sync` the package `ok` line is **not** evidence — cite `-run TestRowVisibilityRLS -v` showing subtests **ran** (59), and state `HQ_SYNC_SUBSTRATE_OPTIONAL` was **unset** (B-36 / T-29 decision 108, as amended by T-31 decision 116) |
| **G2 (Playwright)** | `npx bddgen` · `npx playwright test --retries=0` | **Exactly one summary block.** Two blocks under one header = an invalidated run; discard and re-run. Judged against the armed-reds baseline, never against green |
| **G3** | — | **N/A.** Preflight verdict `openspec: absent`; ledger §T-34 decision 140 keeps it that way. Create no OpenSpec scaffolding |
| **G4** | `node build-sw.js` (or `task sw`) | Idempotent — tree clean on a second run. Precache count **31**; if it moves without an asset being deliberately added or removed, that is B-37's silent drop returning. Version parity `version.go Frontend` ≡ `package.json` ≡ `version.json`. 🛑 **Reads git HEAD, not the working tree** — regenerate **after** the merge commit, never mid-merge (B-37) |
| **G6** | fresh-context adversarial subagent | Inputs are the card's slate entry, the diff and the evidence — **never** the implementer's reasoning. On run 20260804 all four cards' G6 found something the card had not; the slot earns itself |

## 🛑 Capture the whole log, then read the file

**Never pipe a gate through `tail`.** This repo has been bitten three times: a `go build` false green
(`card-actuals.md`), the 20260804 orchestrator's own final Playwright gate (**B-93**), and the triage
that found B-93 doing it again to `backlog check` an hour later.

    npx playwright test --retries=0 > gate.log 2>&1; echo "EXIT=$?"; tail -30 gate.log

The exit code must come from the command, not from `tail`. **G2 (Playwright)'s pass condition —
"exactly one summary block" — cannot be evaluated on a tail at all**; count `N passed (` lines over
the complete log. Committing the log under `.night-crew/runs/<date>-autonomous/` lets morning triage
audit the figures instead of inheriting them.

## Mandatory per-leg isolation — not a suggestion

Every implementer, G6 reviewer and fix round gets a **unique `TEST_PORT`, a unique `TEST_DB_NAME`, a
unique `HQ_RLS_TEST_DB`, and a unique scratchpad directory.**

🛑 **`TEST_DB_NAME` became load-bearing on 2026-08-04 and the launch prompt did not know it.** Since
card A1 landed, `playwright.config.js`'s `webServer.command` **DROPs the database it is pointed at as
its first act**, on every leg including the `night-crew.toml` subset path. Two legs differing only in
`TEST_PORT` no longer merely collide — **the later destroys the earlier mid-suite** (G6 demonstrated
it: an in-flight subset collapsed to 3 failed / 27 did not run). An unqualified leg is now
destructive, not just noisy. **B-80**; `launch-20260804.md` carried no isolation stanza at all (zero
hits for `TEST_PORT`, `TEST_DB_NAME`, `HQ_RLS_TEST_DB`, `unique`), and every value that night came
from the orchestrator ad hoc.

B-35 and B-16 bit on 20260801; on 20260802 shared scratchpads clobbered logs and produced a
retracted contamination report. A stale binary another session left in a shared scratchpad produces
false findings (B-50).

## 🛑 Playwright's path filter matches the ABSOLUTE path (B-87)

A positional filter is a regex against the **full path**, so a worktree whose directory name contains
a spec token silently turns a confined subset into the **full suite**. This bit twice on 20260804:
A2's "confined" subset ran all 787 tests because its worktree was `a2-workflows-…`, and an A6
fix-round command would have done the same from `a6-app-version-badge`.

Always use the `tests/`-anchored form:

    npx playwright test tests/persistence.spec.js -g "FLD-16C"     # ✅
    npx playwright test "persistence"                              # 🛑 may select everything

The CLI filters are OR'd, so the failure mode is **over-running, never running the wrong specs** —
G6 refuted the stronger claim at source. Over-running is still a mis-measurement: a card that reports
a confined gate actually paid for a full suite.

## Environment facts that cost time to rediscover

- `export PATH="/usr/local/go/bin:$PATH"` before **any** Go or Playwright leg. The non-interactive
  shell does not carry Go; Playwright's `webServer` dies with `go: not found` / exit 127, which
  **looks like a test failure and is not**.
- Postgres is on **:5433**, not 5432. Credentials on this box are **`yumyums:yumyums`**, not
  `postgres:postgres` — the wrong one costs a 15-minute run (it fails loud, correctly).
- `-p 1` is load-bearing (`Taskfile.yml:108` already does it): packages share one test DB and each
  `TestMain` truncates. Without it six packages red on cross-package interference — **not** a
  production defect.
- `npx bddgen` is **not optional**. `playwright.config.js` defines two projects; without generation
  the `bdd` project resolves to **zero** spec files, and the suite reports success (B-09).
- `serviceWorkers: 'block'` is repo-wide (B-15). No test observes Workbox serving from the precache;
  assert against the committed `sw.js` **manifest** instead, and parse it rather than string-matching
  — a `runtimeCaching` mention satisfies a naive regex while the file sits outside the precache.

## Check the footprint before you claim a subset

`night-crew.toml`'s `[e2e.seams]` maps changed paths → the spec subset a card must run. A narrow or
missing entry is exactly how a regression escapes a green gate — **this has happened**. Anything
touching a shared or undeclared file de-confines the card to the full suite.
