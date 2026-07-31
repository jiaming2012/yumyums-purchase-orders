# SUMMARY — `sync-rxdb-conflict-notice-ui`

Run `overnight-20260801`, card C2, branch `card/c2-sync-rxdb-conflict-notice-ui`.
Written at the **fix round**, after both gates returned. Not before — CLAUDE.md orders SUMMARY.md
*after* the verifier gate, so its absence at G6 time was correct, and its presence now is due.

Four artefacts forward-reference this file, and until it existed those references were false:

| Artefact | Reference |
|---|---|
| `UI-SPEC.md:16`, `:32`, `:349` | the r2 revision row, and the A-3 redraw obligation |
| `mockup.html:43`, `:1019`, `:1145`, `:1210` | the plate captions |
| `.night-crew/runs/2026-08-01-autonomous/merge-intent-c2-sync-rxdb-conflict-notice-ui.md:58` | "noted here and in SUMMARY.md per CLAUDE.md's mockup rule" |
| `.night-crew/knowledge/roadmap.md:1305`, `:1334` | "Deviation from the signed plates, noted in SUMMARY.md" |

**Decision 98 makes this a condition of the operator's signature, not a formality.** Its wording is
that the UI card *"must redraw those plates and note the deviation in SUMMARY.md"*.

---

## 1 · The mockup deviation, explicitly

`mockup.html` **revision 2** was signed at morning triage 2026-07-29 (ledger T-28 **decision 98**) —
sixteen plates, walked one by one with the PNGs read back. **Three of the sixteen deviate from what
was signed.** All three deviations are the *same* one, and it is an obligation that came out of the
signing walk itself rather than a build-time drift:

| Plate | Signed as | Redrawn as | Under |
|---|---|---|---|
| `edge-removed` | the removed question's **raw field id**, muted monospace | the question's **frozen label**, struck through and read-only | **A-3 · decision 95** |
| `openq-count-a` | same | same | **A-3 · decision 95** |
| `openq-count-b` | same | same | **A-3 · decision 95** |

`edge-removed` deviates a **second** way beyond the label swap: it **gained a row** that draws
A-3's *fallback* — the raw field id, kept for a `template_snapshot` that carries no label for that
id. That row did not exist on the signed plate. It is there because A-3's label source is the
submission's own frozen snapshot, and **nothing validates `template_snapshot`** (B1's
recorded-not-fixed item **R-C**: declared `{type:'object'}` with no nested `properties`, no
dev-mode or validation plugin in the committed vendor bundle, nothing rejecting a malformed value).
A-3 therefore promoted R-C from "recorded" to a *dependency* of this card, and the fallback is what
a malformed snapshot renders. Drawing A-3 without drawing its fallback would have signed off a
design for the happy path only.

**The other thirteen plates are unchanged and ship as signed.**

`shoot.mjs`'s measurement 6 (the counting-arithmetic reconciler) was widened in the same change so
it classifies **both** renderings of a removed-field row — `.cf-q-struck` (the frozen label) and
`.cf-q-gone` (the raw-id fallback) — as the same *kind*. Keying only on `.cf-q-gone` would have
silently reclassified every redrawn row as an ordinary answer, and measurement 6 would have gone on
printing PASS against the wrong arithmetic.

### 1a · A SECOND deviation, opened by this fix round

**`a2-confirm-dark` no longer matches what ships.** Verifier finding **V-1** (section 5) required
the dark scheme's Cancel and Replace controls to read at equal weight. That was fixed in
`workflows.html` and **`mockup.html` was deliberately left alone** — redrawing a signed plate is the
operator's call, and decision 98's own procedure is to *note* the deviation. So:

> Production's dark-mode batch confirm now draws Cancel on a mid-grey tint and Replace on a muted
> red tint. The signed `a2-confirm-dark` plate still draws the old pairing — Cancel sunk into the
> card, Replace on a saturated `#7f1d1d` fill. **Light mode is identical in both.** Order, labels
> and sizes are unchanged everywhere. If the operator would rather re-sign the plate than carry the
> deviation, `mockup.html:203-204` is the two-line change and `shoot.mjs` re-shoots it.

---

## 2 · What was OBSERVED

CLAUDE.md's self-verification ritual: *report what was observed, not what was intended.* This
environment is headless, so every claim below came from reading a PNG back with the multimodal
Read tool, or from a printed measurement — never from reading code.

### 2a · The state screenshots

`tests/states-sync-rxdb-conflict-notice.spec.js` forces all 14 State-Enumeration-Table rows from a
seeded store and releases the shutter in **both colour schemes**, after a **population floor**
assertion (`assertPopulation`) that reds if the state rendered empty. 14 floors, 14 shot calls, 9 of
them double-framed (`<name>-banner-*.png` with the sheet closed, plus `<name>-*.png` of the sheet)
because the real layout is `position:fixed` over My Checklists and a single capture would hide the
banner behind the sheet.

**46 PNGs**, from these 14 states: `empty`, `loading`, `success`, `error`, `outcomes`,
`edge-novalue`, `edge-many`, `edge-confirm`, `a1-banner`, `edge-longvalue`, `edge-removed`,
`edge-removed-fallback`, `edge-storage`, `edge-cap`.

Observed in this fix round, by reading the PNGs back:

* **`edge-confirm-dark`, BEFORE V-1** — Cancel rendered as a near-black fill (`--bg` `#1a1a1a`) on
  a `#262626` card, visible only by its text; "Replace 3 answers" rendered as a saturated `#7f1d1d`
  fill with light-red text. The destructive control was, plainly, the most prominent element on the
  screen.
* **`edge-confirm-dark`, AFTER V-1** — Cancel now carries a visible mid-grey fill (~`#444`) with a
  light border and reads at the same weight as Replace; Replace is a muted red-tinted fill with a
  red border and `#fca5a5` text — still unmistakably the destructive one, still second in the row,
  same label, same size.
* **`edge-confirm-light`, AFTER V-1** — unchanged: pale grey Cancel, pale red Replace, both at equal
  weight on a white card. The light scheme was already balanced and was not touched.
* **`edge-cap-light`, AFTER V-2** — all **ten** visible group chips are distinct:
  `sub_cap100`, `sub_cap200`, `sub_cap900`, `sub_cap300`, `sub_cap100·2`, `sub_cap400`,
  `sub_cap110`, `sub_cap500`, `sub_cap120`, `sub_cap600`. Before the fix, "Checklist 1" and
  "Checklist 10" both read `sub_cap100`. Chip width is unchanged and the page does not scroll
  sideways.

### 2b · The mockup measurements

`node .planning/phases/sync-rxdb-conflict-notice/screenshots/shoot.mjs` — **exit 0**, 16 plates x 2
schemes, all seven measurements PASS in both schemes, **no floor lowered**:

```
tap targets:        64 measured (floor >=62), 0 under 44px
A-1 two figures:     8 banners (pinned ==8), 0 carrying only one
A-1 banner lines:   24 measured (floor >=24), 0 truncated
A-2 names the loss: 13 destructive controls (12 row/batch + 1 confirm), 0 silent
A-1 arithmetic:     10 counting plates reconciled, 0 disagreeing
no-restore exits:    7 .cf.unrec rows (floor >=6), 0 with no Dismiss
overflow:           scrollWidth 480 == clientWidth 480
```

The plate PNGs under `screenshots/` are byte-unchanged by this fix round (`git status` clean over
that directory), because `mockup.html` was not modified — see 1a.

---

## 3 · The evidence is NOT REACHABLE BY A MERGER

**The 46 self-verification PNGs live in `test-results/`, which is gitignored** (`.gitignore:2`).
They are **not in the diff, not in any commit, and not on the branch.** A reviewer or merger reading
this file, this card's commits, or the merge-intent **cannot open the screenshots these observations
were read off.** Every observation in 2a is, to a merger, an unverifiable claim.

Worse in kind: Playwright **wipes `outputDir` at the start of every run**, so the PNGs a previous
run produced are gone the moment anything else runs the suite. They were deleted and regenerated
twice during this fix round alone.

This is stated plainly rather than worked around, because the honest form of "I verified it" here is
"I verified it, and you cannot check that I did unless you re-run this":

```bash
# from the repo root, in this worktree, on branch card/c2-sync-rxdb-conflict-notice-ui
psql "postgres://yumyums:yumyums@localhost:5433/postgres?sslmode=disable" \
  -c "DROP DATABASE IF EXISTS hq_fix_c2_e2e;" -c "CREATE DATABASE hq_fix_c2_e2e;"
npx bddgen
TEST_DB_NAME=hq_fix_c2_e2e TEST_PORT=8360 \
  npx playwright test tests/states-sync-rxdb-conflict-notice.spec.js --retries=0 --project=chromium
# => 46 PNGs in test-results/states-sync-rxdb-conflict-notice/
```

* **Spec that generates them:** `tests/states-sync-rxdb-conflict-notice.spec.js` (`SHOT_DIR` at
  `:43`, `shot()` at `:244`).
* **Output directory:** `test-results/states-sync-rxdb-conflict-notice/` — gitignored, and wiped at
  the start of any Playwright run in this worktree.
* **Port / DB used in this round:** `TEST_PORT=8360`, `TEST_DB_NAME=hq_fix_c2_e2e`. Any free port
  and any scratch database will do; the spec seeds its own store and touches no server state.
* **The mockup plates are the exception** — those 32 PNGs under
  `.planning/phases/sync-rxdb-conflict-notice/screenshots/` **are committed** (`.planning/` is
  gitignored but force-added), so the *mockup* evidence is reachable and the *implementation*
  evidence is not.

**A judgement, not a complaint:** if the operator wants the implementation screenshots to survive
review, the change is to force-add a chosen subset into the phase directory the way the mockup
plates already are. That is a policy call and is not made here.

---

## 4 · F-4 — PLAN.md entered the repo AFTER the implementation commits

Stated in the open because the verifier gate's independence depends on it, and on the evidence
available it is not demonstrated.

```
55ecf47  2026-07-31 14:03:54 -0400  feat(sync-rxdb): the overwritten-answers MODEL ...
f9caa36  2026-07-31 14:04:10 -0400  feat(workflows): the overwritten-answers banner, sheet ...
48c7193  2026-07-31 14:10:05 -0400  docs(plates): A-3 redraw ...        <- PLAN.md enters here
```

`PLAN.md` carries the `done_when:` block (30 rows) and the State Enumeration Table (14 rows) that
**the verifier subagent grades against**, and per CLAUDE.md the verifier's inputs are *only* that
file, the diff and the screenshots — deliberately not the build conversation. That isolation is the
whole point of the gate, and it is worth exactly as much as the contract's independence from the
code.

**Commit order is not proof of authoring order.** `.planning/` is gitignored and force-added, so a
plan written first and `git add -f`-ed late is entirely ordinary in this repo, and the six-minute
gap is consistent with a routine batched add. **But the record cannot distinguish that from a
contract written to fit code that already existed**, and the gate's independence rests on the
contract predating the code. On the evidence available it does not demonstrably do so.

Recorded, not argued away. **Triage weighs it.** Two things that bear on the weighing:

* Nothing in the `done_when:` rows is scoped to an implementation detail in a way that would only be
  writable after the fact — they name observable behaviour and the check that proves it, and several
  (rows 1, 7, 8, 9, 21) name **how they fail**, which is the harder direction to write backwards.
* One row's wording has since drifted from the code by this fix round's own hand: the loading row
  says "exactly 2 `.sk`", and F-5 renamed that class to `.cn-sk`. **PLAN.md is deliberately NOT
  edited to match.** Amending a graded contract after the gate is precisely the thing this section
  is about. The floor itself is unchanged and still bites — `assertPopulation(..., {skeletons: 2})`
  now selects `.cn-sk`, and a rename that had missed the markup would have reported 0.

---

## 5 · The two gate verdicts, and what each found beyond the contract

### Verifier subagent (CLAUDE.md's build/SUMMARY gate) — **PASS, 30 of 30 `done_when:` rows**

No row failed and no row was waived. Beyond the contract it raised **two** issues, both real, both
fixed in this round:

* **V-1 · Dark mode: Cancel and Replace were not equal weight.** In `edge-confirm-dark`, Cancel was
  a dark fill on a dark card while "Replace 3 answers" was a saturated red fill — the *destructive*
  control was the most prominent element on screen. Light mode was balanced. **Row 9's machine check
  passes**, correctly: it tests order, text and size, and all three were right. A-2's stated intent
  is "Cancel at equal weight", and this is a confirm that destroys someone's answers, on a phone used
  at night in a truck. Fixed in `workflows.html` (dark-scheme weighting only; order, labels and
  sizes untouched). Observed in the re-shot PNG — section 2a.
* **V-2 · The document-id chip was not unique, and a committed screenshot proved it.** `shortDocId`
  sliced to six significant characters; in `edge-cap-light`, "Checklist 1" and "Checklist 10" both
  displayed `sub_cap100`. Grouping and counting were unaffected (display-only), and with real uuids
  a six-hex collision is ~1 in 16M per pair — but the chip exists so a crew member can tell groups
  apart and quote one to a manager, and the shipped plate demonstrated it failing at exactly that.
  Fixed as a **guarantee** (`assignDocChips`, sheet-wide, before the cap slice) rather than a longer
  slice: the two ids agree for 27 characters, so widening would have produced a 32-character chip
  that does not fit the 480px column it is drawn in. Observed — section 2a.

### G6 review — **APPROVE WITH FINDINGS**

G6 confirmed, and this round did not disturb: **no scope breach**; **no write path swapped**
(`autoSaveField` -> `submitOp('SET_FIELD')` -> `DRAFT_RESPONSES` -> `hydrateFieldState`
byte-untouched); `findFieldInSnapshot` **total beyond its own coverage**; **no `shoot.mjs` floor
lowered**; the **precache trap not re-armed**; the **version bump correct** (Frontend 1.3.0 ->
1.4.0, mirrored in `package.json` and `version.json`).

Its findings, and their disposition:

| | Finding | Disposition |
|---|---|---|
| **F-1** | `SUMMARY.md` does not exist, and four artefacts forward-reference it | **This file.** Correct at G6 time (CLAUDE.md orders it after the gate); due now that the gate has passed. |
| **F-2** | `formatValue`'s `_v` unwrap recursed with no depth bound and read the property with no `try` — on the same hot path as the parser that *is* total | **Fixed**, red-first. `MAX_VALUE_UNWRAP_DEPTH` beside `MAX_SNAPSHOT_DEPTH`, `try` around the accessor, degradation to the em-dash the renderer already draws. See the reachability note below. |
| **F-3** | `sync-rxdb-conflict.spec.js:752` claimed an assertion reddens if the threading is removed from `startHQReplication`; it does not | **Both halves done.** Comment corrected *and* a test added that genuinely drives `startHQReplication` and reds on the deletion. |
| **F-5** | non-array `records` -> `TypeError`; `btn()`'s label interpolated unescaped; `.more`/`.sk`/`.sk-l` un-namespaced | **All three fixed.** |
| — | `[LC-02]` and `tests/inventory.spec.js:2908` | **Filed as B-32, not attributed and not retired** — see section 6. |

#### F-2's reachability, measured rather than repeated

G6's finding says the deep `_v` chain *"is JSON-representable, so it survives a store round-trip"*.
Binary-searched on the box that ran this fix (node 20.20 / V8) **it does not**:

```
structuredClone (what IndexedDB uses)   overflows past ~1 954 deep
JSON.stringify                          overflows past ~4 174 deep
the OLD unbounded formatValue           overflowed past ~8 397 deep
```

A chain deep enough to break the renderer could not have come back out of the local store on this
engine. **The bound is kept anyway** — those margins are stack-budget and engine dependent and lie
within one order of magnitude of each other, `sync-hard-cutover` gives this function a network-fed
producer whose payload is *parsed* rather than cloned, and **the second half of F-2 — a throwing
accessor — is reachable today**, with nothing validating the stored value (R-C). Exposure today is
nil either way: the UI is dormant, no producer writes a record. This is recorded rather than quietly
dropped, because a finding half-right is worth more once it is exactly right.

---

## 6 · Two reds that are NOT this card's, filed as **B-32**

Both refusals are evidenced, not asserted, and **neither is retired**:

* **`[LC-02]`** (`tests/workflows.spec.js`, *"submitted checklist survives builder edit with
  assignment change"*) — the identical test reddened the same way on **2026-07-26**, on card B's
  leg, on a tree where `workflows.html` and `tests/workflows.spec.js` were proven **byte-identical to
  base** (`.night-crew/qa/spike-supabase/captures/gate-20260726-card-b.txt`, `.night-crew/runs/2026-07-26-autonomous/HANDOFF.md:166`). Five
  days before C2 existed; refused attribution then.
* **`tests/inventory.spec.js:2908`** (*Receipt sync button > reload mid-run shows Syncing...*) — 30 s
  `networkidle` timeout in G6's run. `inventory.html` contains **zero** references to
  `sync-rxdb`/`conflict-notice`, and neither it nor `tests/inventory.spec.js` appears anywhere in
  this card's diff.

Filed as **one** entry describing the family — load/scale-sensitive 30 s timeouts that redden at
whole-suite scale and green in isolation — alongside **B-27** and **B-30**, because the family is the
finding. `B-32` was confirmed free against the **merged run branch** (`overnight-20260801`), not
merely this worktree.

---

## 7 · Gate results for the fix round

Named runs, raw exit codes, on `TEST_PORT=8360` / `hq_fix_c2_e2e` / `hq_fix_c2_go` — no shared
database touched, and only databases created by this round were dropped.

| Gate | Command | Exit | Result |
|---|---|---|---|
| **G1** | `go build ./...` | `0` | — |
| **G1** | `go vet ./...` | `0` | — |
| **G2 (Go)** | `go test ./... -p 1 -count=1` | `0` | every package `ok`; DB live: **51 tables**, **goose 73** |
| **G2 (Playwright)** | full suite, `--retries=0` | see 7a | — |
| **G4** | `node build-sw.js` | `0` | **29 files**, 2111.1 KB; byte-identical on re-run; Frontend **1.4.0** = `package.json` = `version.json` |
| **states spec** | `states-sync-rxdb-conflict-notice.spec.js` | `0` | **14 population floors** all assert and all bite |
| **shoot.mjs** | `node .../screenshots/shoot.mjs` | `0` | no floor lowered — 64>=62 tap targets, 7>=6 unrec rows, 8==8 banners |

### 7a · Playwright full-suite result

**ONE run, ONE summary block, quoted here in full.** Command:

```
TEST_DB_NAME=hq_fix_c2_e2e TEST_PORT=8360 npx playwright test --retries=0 --reporter=line
```

```
Running 739 tests using 1 worker
...
  Slow test file: [chromium] > tests/inventory.spec.js (5.8m)
  Slow test file: [chromium] > tests/sync.spec.js (5.3m)
  6 skipped
  733 passed (25.9m)
```

**Raw exit code `0`. 733 passed, 6 skipped, 0 failed, 0 flaky.** Both Playwright projects ran —
`chromium` (`./tests`) and `bdd` (`.features-gen/`, generated by `npx bddgen` first, without which
the `bdd` project resolves to zero spec files and the suite reports success while a whole project
contributes nothing — B-09 / T-25 decision 73). The log contains exactly **one** `Running 739
tests` line and exactly **one** terminal summary, so this is a single valid run and not two
concatenated ones.

The 6 skips are the suite's ordinary conditional ones (two `zz-sw-manifest-*-probe.html` fixtures
"not in HEAD", plus PARKED cases needing live DO Spaces / photo plumbing) — none is new.

🛑 **`[LST-17]` did NOT fire in this run.** Both of its tests —
`tests/sync.spec.js:446` and `:1006`, *"list page progress decrements/updates when another device
un/completes a field"* — **passed**, at suite positions 568 and 579 of 739. This is recorded as an
observation, not as a disarm: the armed red is load- and timing-sensitive, which is precisely the
B-32 family's shape, and this project's own standing rule is that **one clean leg is not a disarm**.
It remains armed. Nothing was done to it.

Neither `[LC-02]` nor `tests/inventory.spec.js:2908` reddened in this run either — consistent with
B-32's "greens in isolation, and on a quieter box" characterisation, and equally not a retirement.
This leg ran with **one** Playwright stack on the box, against its own database and port; G6's run
did not.

---

## 8 · Commits in this fix round

```
81e12b2  fix(sync-rxdb): bound formatValue's _v unwrap — F-2, the other total walk
c393b84  test(sync-rxdb): actually DRIVE startHQReplication — F-3, and the false comment
428dabb  fix(sync-rxdb): the document chip is unique on the sheet — V-2, plus F-5's records guard
5a9e0c4  fix(workflows): Cancel and Replace at equal weight in dark mode — V-1
1ebed04  fix(sync-rxdb): escape btn's label and namespace the sheet's global classes — F-5
13813da  build(sw): regenerate after the fix round — 29 files, Frontend 1.4.0 unchanged
e8996b9  docs(nc): B-32 — the load/scale-sensitive 30 s timeout family, filed not attributed
```

Nothing was pushed, nothing tagged, nothing deployed. `HQ_SYNC_REST_URL` remains unset everywhere.
