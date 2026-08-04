# HANDOFF — run `overnight-20260804`

> **Branch:** `overnight-20260804` (cut from `dev` at `008e3ad`)
> **Slate:** `.night-crew/knowledge/reference/slate-20260804.md` — batch sign-off 2026-08-03 evening
> **Scope:** SERIAL — A1 `e2e-gate-database-isolation` → A2 `workflows-autosavefield-phantom` →
> A4 `offline-ownership-design-note` → 🅢 A6 `app-version-badge` (stretch)
> **Result: 4 of 4 landed. Zero parks. Zero open forks.**
> **Wall clock:** 10:23:02 → 14:38:08 for the cards (4h15m), plus closeout.

---

## Read this first — the three things that actually matter

1. **A1 changed what a gate result means.** Until tonight, no night-crew gate leg — full or subset —
   had ever reset the e2e database. Every Playwright figure in this milestone before tonight was
   measured against an accumulating database. **Tonight's figures are the first that are not.** Do
   not compare them to earlier ones as though they were the same measurement.
2. **Every card's G6 found something the card had not.** Not stylistic notes — a refuted finding, a
   ninth data class the note denied existed, and a regression the card was introducing. The
   adversarial review earned its slot on all four cards, which is worth knowing when the gate is
   next under time pressure.
3. **You have two attended items waiting**, both below under *Next actions*. One of them
   (`task prod:deploy`) is the only way D-KR2a/D-KR2b close, and it takes ~15 minutes.

---

## Per-card outcomes

| Card | Outcome | End-to-end | G6 | Closes |
|---|---|---|---|---|
| **A1** `e2e-gate-database-isolation` | **MERGED** | 76m20s | APPROVE-WITH-NOTES | **B-76** |
| **A2** `workflows-autosavefield-phantom` | **MERGED** | 76m45s | APPROVE-WITH-NOTES | **B-65** |
| **A4** `offline-ownership-design-note` | **MERGED** | 31m07s | APPROVE-WITH-NOTES | reworded **E-KR3** |
| 🅢 **A6** `app-version-badge` | **MERGED** | 67m33s | APPROVE-WITH-NOTES | **D-KR2b**'s evidence method |

Median end-to-end **71m56s** (N=4, no exclusions). Estimates were 75–120m / 45–75m / 45–75m / 45–75m;
three landed inside, A2 went 1m45s over its high end.

**The A6 stretch gate was evaluated once**, at the moment A4's merge completed (13:30), against the
rule that it must not be re-evaluated optimistically later: A1/A2/A4 all clean, and A6's high end
plus the 30m closeout fit with over an hour of margin. It was taken.

---

## A1 · `e2e-gate-database-isolation` — MERGED, closes B-76

**What was wrong:** `night-crew.toml:33-34` ran `npx playwright test` directly for both `suite` and
`subset`. The only `DROP DATABASE` lived in `Taskfile.yml` under `task test`, and
`playwright.config.js` had no `globalSetup`. **No night-crew gate leg had ever reset the e2e
database.**

**The fix** puts the reset in `playwright.config.js`'s `webServer.command` — not a `globalSetup`, and
the reason is measured rather than assumed: Playwright's `tasks.js:100-110` runs the webServer plugin
**before** `config.globalSetups`, so a `globalSetup` would drop the database out from under a server
that had already migrated and seeded it. `scripts/reset-e2e-db.js` refuses any database whose name
is not a test database, and fails loud.

**G6 re-derived both halves independently** — the exact subset invocation `night-crew.toml:34`
expands to leaves rows behind on the base tree and does not on the card tree. It also could not make
the guard eat a non-test database, and could not make it fail quietly.

🛑 **Two things this changed for everyone else:**
- **Every Playwright invocation now DROPs the database it is pointed at, as its first act.** Two legs
  differing only in `TEST_PORT` no longer coexist — the later destroys the earlier mid-suite. G6
  demonstrated it (an in-flight subset collapsed to 3 failed / 27 did not run). **Filed B-80.** I
  issued a unique `TEST_DB_NAME` to every leg for the rest of the night; that needs to become
  permanent, and see the launch-prompt gap below.
- The reset banner was printing to stdout, which Playwright's `webServer` discards — so a gate log
  contained **zero evidence the reset happened**, which is the exact shape of the defect the card
  closes. Fixed in the fix round (**B-81**, resolved).

---

## A2 · `workflows-autosavefield-phantom` — MERGED, closes B-65

**What was wrong, and it was worse than the slate knew.** `workflows.html:2219` called
`autoSaveField` — zero definitions repo-wide, a live `ReferenceError` on the fail-note-with-photo
path in shipped code. **Why nobody noticed:** the upload chain's own `.catch()` swallowed it, and
`FAIL_NOTES[fldId].photo` was mutated one line *before* the throw. So the crew member saw the
thumbnail, saw "photo attached", and nothing persisted. No `pageerror`, no toast.

🛑 **The argument was independently wrong too.** It passed
`FIELD_RESPONSES[fldId].value || FIELD_RESPONSES[fldId]` — but a fail card exists precisely when the
answer is falsy, so `false || resp` evaluates to the whole response **object**. **A rename-only fix
would have replaced a loud crash with silent corruption.** Shipped as `resp ? resp.value : null`,
mirroring the correction-photo path that has been right all along.

**A second phantom rode along:** every doc site also named the transport as `POST /saveResponse`.
**No frontend code posts to it** — the op journal has been the single write channel since the sync
work landed. So the doc sites were two-thirds wrong, not half.

**Eight doc sites, not the slate's four:** `CLAUDE.md`, `sync-rxdb/bootstrap.js`,
`sync-rxdb/conflict-notice-ui.js`, `roadmap.md`, plus `docs/data-flow-audit.md`, `README.md`,
`.claude/skills/save-project/SKILL.md`, and `prds/PRD-operations-hardening.md`.

🛑 **`CLAUDE.md`'s persistence rule survived intact** — G6 checked this specifically, because it was
the card's PARK trigger. Only the false name, transport and call shape changed. The persisted-states
list went **7 → 9** (*widening* coverage), and step 2 of the add-a-field-type procedure got
**stricter**, not looser. The PRD was **appended to, not rewritten** — signed requirement text left
intact with dated annotations below it.

---

## A4 · `offline-ownership-design-note` — MERGED, closes reworded E-KR3

**Delivered:** `.night-crew/knowledge/designs/offline-ownership.md`.

🛑 **It publishes NINE classes across six stores, not the plan's eight.** G6 found
`LOCAL_COLLECTIONS.conflict_records` (`sync-schema/collections.js:277-284`) — `replicated: false`,
**no `table` key at all** by design, 30-day client-side sweep, with a shipped write
(`conflict-notice.js:788-796`) and a shipped read (`workflows.html:3588-3592`) — while the note's own
Rule 4 footnote actively denied a ninth class existed.

It is distinct by the note's own splitting logic: class #8 is a replication buffer with a server
copy; **#9 never leaves the phone and has no server row anywhere.** It is the durable local record of
a crew member's *overwritten* edit. Losing it loses the only copy, permanently — and the note is the
document someone opens when data goes missing.

**The omission was inherited, not invented.** The plan's §3 A4 table and its §8 mapping table
*independently* both have 8 rows, and `grep -c conflict_records` over the whole plan returns **0**.
The signed non-negotiable ("publish all 8 classes across 6 stores") is satisfied — all 8 are
published unchanged at the same 6 stores — under the card's own governing rule that measurement wins
over the plan's table.

**Seven deviations stated rather than silently patched**, including one where a claim was attributed
to ledger **T-21d** that T-21d does not make (T-21d ends `:1041`; the supporting text is **T-21e**).

**§8 described, not decided.** G6 checked specifically: the note states in as many words that
adopting §8 is a roadmap decision, not a card's. No product fork improvised.

---

## A6 · `app-version-badge` 🅢 — MERGED, closes D-KR2b's evidence method

A discreet version line in `index.html`'s footer, reading the **precached `version.json`** — never
`/api/v1/health`, and **with no fallback to it**. The early return is structural: if the file read
fails, the line shows `v—` / `data-state="unknown"` and the health fetch never happens.

**Why that constraint is the whole card:** the server's `frontend_version` is always current, so an
API-fed badge would print the right number on a phone frozen on last week's bundle — **hiding the
exact defect it exists to catch.**

🛑 **G6 proved the test catches a reroute rather than accepting the claim.** It wrote two forbidden
implementations and ran the spec against each: API-only → **3 failed**; `version.json`-primary
**with an API fallback** → **2 failed**, including the decisive test that aborts `version.json` while
health still answers. The fallback variant is the realistic regression, and it reds.

**What the test cannot prove, stated honestly:** `serviceWorkers: 'block'` (B-15) is repo-wide, so no
test here observes Workbox serving from the precache — under test the file arrives over plain HTTP.
The other half is covered by asserting `version.json` is in the committed `sw.js` **manifest**; the
fix round strengthened that from a string match to a real manifest parse, because a future `sw.js`
mentioning the file in a `runtimeCaching` pattern would have satisfied the regex while the file sat
outside the precache.

🛑 **The card introduced a regression and the fix round closed it rather than filing it.**
`version.json` is a git-ignored build artifact; the `npx playwright test` path never created it, so
the new spec red on any clean checkout — handing every future card leg a red unrelated to its change,
which partially undoes what A1 landed tonight. Now generated in `webServer.command` beside A1's
reset. Proven closed by the same fresh-archive experiment G6 used to prove it existed: **1 failed →
5 passed**. **B-92 resolved.**

---

## Findings filed tonight — B-80 through B-92

Thirteen new entries. **Four are already resolved**; nine are open with destinations.

| # | What | State |
|---|---|---|
| **B-80** | Every Playwright leg DROPs the shared default `hq_test_e2e`; cards are issued `TEST_PORT` but **not** `TEST_DB_NAME` | open |
| **B-81** | Reset banner printed to stdout, which `webServer` discards → no gate-log evidence | ✅ resolved |
| **B-82** | `tests/db-isolation.spec.js` never runs on a seam-confined subset | open |
| **B-83** | Reset guard is name-only — host/port/credentials unchecked | open |
| **B-84** | `NIGHTCREW_ENV_URL` set → no reset **and** the guard test skips rather than fails | open |
| **B-85** | `task bdd` sets `DB_PORT: '5432'`; now hangs on the reset instead of failing at connect | open |
| **B-86** | `verify-test-harness.sh` floor `MIN_SPEC_FILES=20` is stale — repo has 27 | open |
| **B-87** | Playwright CLI path filter matches the **absolute** path, so a worktree name containing a seam token silently turns a confined subset into the **full suite** | open |
| **B-88** | Rule 2's guard watches three symbols but `workflows.html` reads RxDB via `window.HQSync.db`, unguarded | open |
| **B-89** | 🛑 **live code bug** — `index.html` writes `hq_apps` as `{uid, apps}`; `bootstrap.js:62-71` `Array.isArray`-gates it, so `cachedGrantSlugs()` returns `[]` **unconditionally** | open |
| **B-90** | `build-sw.js:457` comment says "still 29 files"; it is 31 | open |
| **B-91** | Three sites still say `hq_offline_v1` holds one store; it holds two | open |
| **B-92** | `npx playwright test` path never generated `version.json` | ✅ resolved |

**Also resolved tonight:** **B-65** (A2), **B-76** (A1).

🛑 **B-89 deserves your eye.** It is a live bug in shipped code, found incidentally while A4 verified
a table row: the cached grant list that decides what to replicate is **always empty** on a real
client, and the test at `sync-rxdb-client.spec.js:1385` plants the *array* shape so nothing catches
it. That is B-65's failure mode exactly — a test passing on a shape the app never produces.

🛑 **B-87 bit twice tonight, in practice.** A2's "confined" subset silently ran the full 787-test
suite because its worktree was named `a2-workflows-…`. And a command in my own A6 fix-round brief
(`npx playwright test "version-badge"`) would have selected the full suite from
`a6-app-version-badge`; the fix round caught it and used the `tests/`-anchored form. **Any command
run from a card worktree is subject to this** — worth knowing at triage.

---

## Two process defects in the run's own governing documents

Neither cost the night, both cost time, and both will recur.

1. 🛑 **B-26 recurred. `reference/overnight-run-plan-20260707.md` does not exist in this repo** — it
   never has (`git log --all` finds no trace); it lives only in the night-crew clone. `slate-20260804.md`
   line 217 and the launch prompt both inherit G1–G6 from it. **Slate-20260803 had already caught
   this** and wrote HQ's ladder out in full under §"HQ's verification ladder"; tonight's slate
   regressed to the dangling pointer. I reconstructed the ladder from slate-20260803 verbatim and
   pasted it into every implementer and G6 prompt. **The fix is to stop referencing that path** —
   either inline the ladder in every slate, or give this repo its own copy.
2. 🛑 **`launch-20260804.md` carries no per-card isolation stanza at all.** Grep it for `TEST_PORT`,
   `TEST_DB_NAME`, `HQ_RLS_TEST_DB` or `unique`: zero hits. `launch-20260803.md:97-98` had one.
   Every isolation value used tonight came from me ad hoc. With B-80 now live — every Playwright run
   DROPs its target database — **an unqualified leg is no longer merely noisy, it is destructive.**
   The stanza needs restoring to the launch-prompt template, with `TEST_DB_NAME` added.

---

## 🛑 Final gate — taken by me on the FINAL MERGED TREE, not inherited from card reports

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **rc=0** · `go vet ./...` **rc=0** (from `backend/`, the module root) |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` → **exit 0**, all 9 packages `ok`. **Counts checked, not `ok`: 439 ran / 437 PASS / 0 FAIL / 2 SKIP.** `internal/workflow` ran **35** tests — the package CLAUDE.md warns can print `ok` on zero; it did not. `TestRowVisibilityRLS` **59 subtests ran, 59 passed**, `HQ_SYNC_SUBSTRATE_OPTIONAL` **UNSET**, `HQ_SYNC_REST_URL` **UNSET**. The 2 skips are the pre-existing `TestProxyLive_*` pair (`HQ_SYNC_SPIKE_LIVE` unset). |
| **G2 (Playwright)** | See the block below — full suite, run **alone on the box**. |
| **G3** | **N/A** — workflow preflight re-confirmed at launch: `openspec: absent`, `gsd: detected`. No scaffolding created. |
| **G4** | `node build-sw.js` **after** the merge commits (B-37) → **31 files precached** (2165.0 KB), reachability 18 parsed / 30 resolved / 0 outside. Second run left the tree **clean** ⇒ idempotent. Version parity three-way: `version.go` `Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**. Backend **0.3.0**. **No version bump this run.** |
| **G6** | Four reviews, four fresh contexts, inputs limited to the slate entry + diff + evidence. All four APPROVE-WITH-NOTES; all four produced a fix round. |

### G2 (Playwright) — full suite on the final merged tree

Run **alone on the box** (verified before starting: 0 other test processes), `--retries=0`, after
`npx bddgen`. Isolation `TEST_PORT=8299` / `TEST_DB_NAME=hq_test_e2e_final`. `HQ_SYNC_REST_URL`
**unset**. Started 14:42:57, ended 15:05:48.

```
  6 skipped
  786 passed (22.8m)
```

**Exit 0. Zero failures.** Total 792 = 786 + 6, and the final test index printed was **792**.

🛑 **All four armed reds RAN and PASSED — and per decision 100 (armed reds are named by test title, never by line number) and decision 120 (all four passed and none was retired) that retires
nothing.** Verified by full title that each is present in the suite and none is statically skipped,
so with 0 failures each necessarily ran green:

- `item modal pre-fills search with current line item text`
- `yes/no answer converges (live + catch-up)` — note this one **failed** on A2's leg earlier tonight
  and passed here. Both outcomes are expected of an armed red; **neither is evidence about any
  card**, and no card claimed a fix for it.
- `a queued submission still lends its idempotency_key at 7:30pm CT [A1-TZ-02]`
- `submitted checklist survives builder edit with assignment change [LC-02]`

**Measured against a FRESH database** — the first final gate in this milestone for which that is
true, because A1 landed earlier tonight. Do not compare 22.8m to pre-A1 figures as though they were
the same measurement. (A2's leg saw 21.2m and A6's 21.0m on the same fresh-DB basis; 22.8m is within
that spread, and the box was quiet.)

🛑 **One honesty note about this capture, and it is mine, not a card's.** I piped this gate through
`tail -30`, so the retained log holds only the last 30 lines. **That means I cannot demonstrate
"exactly one summary block" by counting blocks in a complete log** — the check the ladder actually
specifies. What I can establish, and did: the test indices run consecutively through **792**, the
summary reconciles exactly (`786 + 6 = 792`), and the exit code was 0 — arithmetic consistent only
with a single run. I judge the gate sound on that basis, but the direct check was not performed and
I am not going to report it as though it were. **Filed as B-93**, because this repo has already been
bitten once by a pipe into `tail` masking a result (`card-actuals.md`: a false-green `go build`), and
the orchestrator's own final-gate capture is exactly the wrong place to repeat it.

---

## Conflict log

`.night-crew/knowledge/reference/conflicts-20260804.md` — **four entries, one per merge, all CLEAN.**
No hunks resolved, no card parked at a merge.

That is the serial dispatch doing its job rather than luck: each card branched from a base that
already contained its predecessors, so the shared files (`BACKLOG.md`, `roadmap.md`, `sw.js`) were
reconciled by construction. A4's merge-intent flagged a plausible `roadmap.md` conflict; it did not
materialize.

---

## Next actions — for you, in the morning

### 1. Two attended items this slate creates

🛑 **(a) The P6 fix-forward checklist (B3/B4/B6–B10) — BEFORE the period-summary notice goes
anywhere.** ~10 minutes, attended. This ordering is the point: the notice must not be sent before
the checklist is done.

🛑 **(b) `task prod:deploy` → `task version` → read the version line on a RETURNING client.**
~15 minutes, attended. **This closes D-KR2a and D-KR2b, and A6 landed, so it is now unblocked.**
The third step is the one that matters and the one that is easy to skip: the line must be read on a
client that *already had the app installed*, not a fresh load — a fresh load fetches the new bundle
and would show the correct version whether or not the staleness mechanism works. That is the whole
evidence method A6 exists to provide.

### 2. Triage decisions waiting

- **Nothing is parked and no fork is open.** `DECISIONS-NEEDED.md` has an empty *Open* section, and
  that is a real result, not an unwritten one.
- **B-89** is a live shipped bug — decide whether it rides the next night or is fixed attended.
- **B-80 + the launch-prompt isolation gap** are one decision, not two: restoring the stanza is the
  remedy for both.
- **B-26** — decide where HQ's gate ladder lives, so the next slate stops inheriting from a file
  that does not exist.
- **B-105** remains open and untouched: which per-change discipline this repo adopts. No card
  answered it, deliberately.

### 3. Standing flags, unchanged by this run

| Flag | State |
|---|---|
| Attended `task sandbox:e2e` | 🛑 **Still unsatisfied, still armed.** No card touched the verify/merge path, and triage's own `[FLD-16C]` change touched only `tests/` and `docs/`. **Re-arms whenever the verify/merge path changes underneath it.** |
| B-27 + three other armed reds | 🛑 **STILL ARMED.** See the gate block — passing retires nothing — an armed red stays armed until someone diagnoses it, not until it goes green once (decision 100, armed reds named by title; decision 120, four passed and none retired). **Triage adds a fifth: `[RUN-10]`, now B-131** — it failed on an independent full run of this tree and is flaky on `dev` too, so a red there is expected and must stop reading as a regression (T-34 decision 135). |
| `HQ_SYNC_REST_URL` | Unset, and stayed unset. No card set it. |
| E-KR1 | Graded **NOT MET** (T-33 decision 132). No card addressed it; deliberate. |
| `run-evidence check` | 🛑 Blind in this repo (B-77) — **and triage confirmed the prediction below in the affirmative.** After a 4-of-4 night that closed, `run-evidence check --run 20260804` still reported `no-run-evidence`: it seeks `.night-crew/runs/**20260804**/` (this repo names them `2026-08-04-autonomous`) and a root-level `reference/conflicts-*` (this repo is scaffolded, so it lives under `.night-crew/knowledge/`). `reference/closeout-20260804.md` **exists** and the night closed. 🛑 The record was **not** back-dated or written to make the verdict green. |
| G2 (Playwright) figures | 🛑 **NEW FLAG, armed by triage.** The closeout's `exit 0 / 786 passed` did not reproduce — an independent run on the same tree gave `785 passed / 1 failed / 6 skipped, exit 1` (B-131, not branch-attributable). **Do not cite 786/0 as a baseline.** Re-arms until B-131 is diagnosed or listed as an armed red. Note the one genuinely *new* clearance: **B-93's own pass condition was performed for the first time** — exactly one summary block in the complete 4716-line log — so that check is now evidenced rather than inferred. |

### Cleared by the morning's evidence

- **B-65's coverage half — CLEARED.** A2 closed the naming half; triage found the correction-photo twin had zero executing coverage and that a literal `autoSaveField` mutation left the suite green (9 passed, rc=0). `[FLD-16C]` closes it, mutation-verified in both directions (commit `70ea466`, B-136). **Re-arms if any bundled-metadata write gains a new path** — two of the nine persisted states still have a test that bypasses the transport it means to cover.
- **The four merges' resolutions — nothing to re-read.** All four were clean by construction (serial dispatch, each card branched off a base containing its predecessors); no hunk was auto-resolved unattended, so there is no 3am judgment call to audit.
- **B-26 / B-80 — CLEARED as recurring process defects**, by moving both out of the slate text into `reference/gate-ladder.md` (T-34 decisions 138–139). Re-arms if a future slate cites a night-crew-clone path again.

### 4. Milestone position

**Cards still white: ZERO**, unchanged. This slate did not close the milestone on cards and no slate
can — the close rests on grading, which ledger T-33 set up. Report in **cards, not KRs**, per the
standing rule.
