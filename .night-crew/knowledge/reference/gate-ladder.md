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
>
> **B-26's residual closed 2026-08-06** (card A4, run `20260806`): writing the file left two things
> open — **G5 was still undefined**, and B-26's sibling finding **B-14 was recorded nowhere the
> ladder's readers would see it**. Both are now in this file, under §"There is no G5" and §"The
> morning-triage G4 discipline greps are VACUOUS in this repo". A future slate inherits an answer,
> not a gap.

---

## The gates

| Gate | Command | Pass condition |
|---|---|---|
| **G1** | `go build ./...` · `go vet ./...` (from `backend/`, the module root) | both exit 0. 🛑 Run from `backend/` — `./...` from the repo root matches **no module** and, piped into `tail`, prints a false green (`card-actuals.md`, and again at triage 2026-08-03) |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` | exit 0, **and counts checked, not `ok`**. 🛑 `DB_TEST_URL` must be set or the suite exits 0 while skipping every DB-coupled test — `internal/workflow` runs **zero** tests and still prints `ok`. Expect 9 packages and ~439 tests; `internal/workflow` should run **35**. For `internal/sync` the package `ok` line is **not** evidence — **the 59-subtest count is now asserted by the suite itself, not eyeballed from a `-v` log** (see below). The human evidence line must still state that **`HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` were both unset** (B-36, the package that prints `ok` on zero tests; decision 108 made proving-the-suite-ran a standing evidence rule, and decision 116 kept it with amendments) |
| **G2 (Playwright)** | `npx bddgen` · `npx playwright test --retries=0` | **Exactly one summary block.** Two blocks under one header = an invalidated run; discard and re-run. Judged against the armed-reds baseline, never against green |
| **G3** | — | **N/A.** Preflight verdict `openspec: absent`; ledger §T-34 decision 140 keeps it that way. Create no OpenSpec scaffolding |
| **G4** | `node build-sw.js` (or `task sw`) | Idempotent — tree clean on a second run. Precache count **31**; if it moves without an asset being deliberately added or removed, that is B-37's silent drop returning. Version parity `version.go Frontend` ≡ `package.json` ≡ `version.json`. 🛑 **Reads git HEAD, not the working tree** — regenerate **after** the merge commit, never mid-merge (B-37) |
| **G6** | fresh-context adversarial subagent | Inputs are the card's slate entry, the diff and the evidence — **never** the implementer's reasoning. On run 20260804 all four cards' G6 found something the card had not; the slot earns itself |

### 🛑 There is no G5. The number is retired, not missing.

**The ladder is G1, G2 (Go), G2 (Playwright), G3, G4, G6. That is the whole ladder.** A card that
has run those has run every gate this repo has. **Do not invent a G5, do not renumber G6 to G5, and
do not treat the gap as an open TODO.**

Every slate from 07-15 onward cited "G1–G6", and the runs practised exactly the six above — G5 was
**never defined in any sense a run used**. It survives only in two unrelated places that are not
gates and never were: the hardening PRDs use "G5" for a *second independent enumeration pass* over
user flows (`PRD-operations-hardening.md`, `PRD-purchasing-hardening.md`,
`PRD-onboarding-hardening.md`), and `reference/qa-coverage-findings-20260718.md` uses "G5" as a
*coverage gap number* for a missing behavioural-oracle layer. Neither is a verification gate; neither
was ever run as one.

**Decided at morning triage 2026-07-29 (ledger §T-28, decision 101, fork D-6 → B-26).** Recorded as
retired-never-defined rather than renumbered, deliberately: *a retired number reads as history, a
renumbered ladder hides that a gate was cited for a month and never existed.*

**If a future night wants a real fifth gate**, that is an operator call — it adds a new obligation to
every card — and it gets a name, not the number 5. Write the case and park it; do not legislate one
inside a run.

### The `internal/sync` evidence line, since run 20260806

The 59-subtest count is **asserted by the suite**, not read off a log by a human. Card A1 of run
`20260806` landed `TestRowVisibilitySubtestCount_Structural` and
`TestRowVisibilitySubtestCount_Executed` in `backend/internal/sync/spikestack_gate_test.go`, against
the constant `wantRowVisibilitySubtests = 59`. Q-KR1 asked for exactly this: an **assertion** in
place of the **inference** a human drew from `-run TestRowVisibilityRLS -v`. Adding or removing a
variant now reds the package until the constant is bumped deliberately.

So the human-facing evidence line no longer has to transcribe subtest names. **What it must still do
is name the environment**, because the assertion can be disarmed from outside the code:

- **`HQ_SYNC_SUBSTRATE_OPTIONAL` unset** — set, the RLS attack suite legitimately skips (B-36's
  typed opt-out door).
- 🛑 **`HQ_SYNC_GATE_CHILD` unset** — this is the newer and nastier one. `_Executed` and
  `TestSubstrateGate_ExitCodeAsymmetry` re-invoke `go test` as a subprocess and mark the child with
  `HQ_SYNC_GATE_CHILD=1` so it does not fork forever; both therefore **skip** when they see it.
  A leaked value in the parent environment disarms **both** the count assertion and B-36's
  exit-code pin **at once**, and the package still prints `ok`. Verified by execution on 20260806:

      HQ_SYNC_GATE_CHILD=1 go test -count=1 \
        -run '^(TestRowVisibilitySubtestCount_Executed|TestSubstrateGate_ExitCodeAsymmetry)$' \
        ./internal/sync/
      => ok  github.com/yumyums/hq/internal/sync  0.008s   EXIT=0

  That is B-36's own defect class — a check whose subject set can go empty — recurring one layer up,
  in the guard built to close it. A1 hardened the variable in code; **requiring the gate evidence to
  state it unset is the belt to that braces.** State both, every time. "I didn't set it" is not the
  same claim as "it was unset" — a leg inherits its parent's environment.

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

## 🛑 The morning-triage G4 discipline greps are VACUOUS in this repo (B-14)

**Report them `N/A-VACUOUS`. Never `clean`, never `PASS`.**

The morning-triage ritual's G4 discipline step runs three greps over night-crew's own journal and
work-order vocabulary, each excluding the package that legitimately owns it:

    grep -rn 'Outcome: "'   --include="*.go" internal cmd | grep -v internal/journal
    grep -rn 'journal.Entry{' --include="*.go" internal cmd | grep -v internal/journal
    # + work-order status string literals outside internal/workorder

**None of that vocabulary exists in hq, and neither do the excluded packages.** This repo's Go tree
is `backend/{cmd,internal}`, and `backend/internal/` holds `alerts auth config db inventory me
onboarding photos purchasing receipt recipes sync testdb toast users version workflow` — no
`journal`, no `workorder`, no `orchestration`. They are **night-crew orchestrator** packages; they
were never here, and `find` over the whole tree confirms it.

**Verified by execution, 2026-08-06** (card A4, run `20260806`), both ways a triage might run them:

| Run from | Result |
|---|---|
| repo root — where the ritual's literal command lands | `ugrep: warning: internal: No such file or directory` / `cmd: No such file or directory`, **exit 2** — the paths themselves are absent |
| `backend/` — the module root, where a helpful triage would retry | **empty, exit 1** — clean-looking, and every bit as vacuous |

So the greps return empty **because there is nothing to find**, and an empty grep reads exactly like
a clean one. Any run or triage reporting them "clean" is reporting a vacuum — the same silent-green
class as B-09 and B-36, one layer up in the tooling rather than in the suite. Triages on 07-27,
07-29, 07-29-2 and 08-01 each carried this and each wrote `N/A-VACUOUS` by hand.

🛑 **The remedy does not live here, and an hq run branch must never apply it.** B-14's own lead:
*"this binds the night-crew clone, not hq — carry it there."*

**Clone-side destination, so it cannot be lost:** the night-crew clone's
`.claude/skills/nc-morning-triage/SKILL.md` — the G4 discipline step (the three greps above) and,
just as importantly, the **triage merge-commit template further down the same file, which emits the
literal string `G4 discipline greps clean.`** That template is what converts a vacuum into a written
PASS, and fixing the greps without fixing the template leaves the false sentence in every commit.
Two shapes were proposed: make the greps **assert their target packages exist** before grading them,
or make them **repo-conditional** so an inapplicable gate reports `N/A` rather than `PASS`. Either
closes it; both are clone-side changes, and both must reach the template too.

**Until it ships clone-side, this is a standing hq obligation:** every HANDOFF, closeout and triage
brief that mentions the G4 discipline greps writes `N/A-VACUOUS — neither package exists in this
repo (B-14)`. If you ever see one recorded as PASS, that report graded a vacuum.

## Check the footprint before you claim a subset

`night-crew.toml`'s `[e2e.seams]` maps changed paths → the spec subset a card must run. A narrow or
missing entry is exactly how a regression escapes a green gate — **this has happened**. Anything
touching a shared or undeclared file de-confines the card to the full suite.
