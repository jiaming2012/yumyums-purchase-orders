# Cycle closeout — "Sync foundation" · closed 2026-08-05

> **Attended `/nc-milestone-close`, 2026-08-05.** Milestone: "Sync foundation" — replace
> `workflows.html`'s hand-rolled WebSocket + Postgres LISTEN/NOTIFY + Lamport-clock op-log sync
> layer with RxDB against a self-hosted Supabase stack. Opened 2026-07-24/25 (attended
> `/nc-roadmap-round`), closed 2026-08-05. Previous boundary: `hq-20260724` ("Prove & surface").
>
> 🛑 **This close was graded BY HAND. Every night-crew milestone verb failed against this repo** —
> see §5. The grades below rest on artifacts in this tree, each cited to a file and line. No
> number here was produced by `night-crew okr grade`, and none was invented to fill a gap.

---

## 0. 🛑 The headline: the milestone did not deliver what it is named for

**"Sync foundation" set out to replace `workflows.html`'s sync layer with RxDB. At close, RxDB
replicates nothing in production, and the layer it was meant to replace is still the live path.**

Verified at source during this close, not taken from any report:

- `createHQSyncDatabase()` and `startHQReplication()` appear in production code in **exactly one
  file** — `sync-rxdb/bootstrap.js`, as an import (`:49-50`) and two deferred re-exports
  (`:83`, `:90`). **Neither is ever called.** Every other occurrence in the tree is a test, a
  design note, a backlog entry or a ledger line.
- `bootstrap.js:18-28` says so in as many words: *"It also does NOT create the RxDB database and
  does NOT start replication, and both omissions are deliberate rather than unfinished."*
- **`window.HQSync.db` is never assigned anywhere**, so `workflows.html:3590`'s
  `if (db && db.conflict_records)` is dead by construction.
- The **495 KB** `vendor/rxdb.bundle.js` is precached to every crew phone (confirmed in `sw.js`'s
  manifest) and does nothing.
- `ledger.md:2663` already recorded it: *"`HQSync.surfaces` has no consumer and
  `startHQReplication` is never called from production code."*

**Nine overnight runs and 28 landed cards produced a tested library with zero call sites.**

🛑 **And every grade in §1 is still correct.** That is the finding, not a contradiction: all four
objectives assert delivery (*"the sync rewrite ships…"*), and **not one of the 13 key results
measures whether RxDB serves a single byte.** They measure sequencing, parity, cadence, coverage,
attack-variant counts — all legitimately MET-able with the objective undelivered. A close can be
arithmetically honest and substantively empty at the same time, and this one is.

**Why it went unnoticed by every close artifact:** `sync-hard-cutover` — the card that would have
made RxDB live — was retired by T-31 decision 126 on a premise measured false (RxDB replicates
from a *second* Postgres; nothing carries a row back to HQ's). **The retirement was correct.** Its
named successor, `sync-live-fill-view`, was **never authored** — and a card count cannot see a card
nobody wrote. So "cards still white: ZERO" was true and meaningless.

This is filed in the night-crew clone as **B-340** (objectives checked against nothing) and
**B-341** (retiring a card silently shrinks the deliverable set), from a parallel investigation on
2026-08-05. The operator's own framing, recorded the same day: *"this is the second time a
milestone has had to be created to cover a predecessor that was not dev complete"* — filed as
**B-344** (dev-complete as a close bar) and **B-345** (prove the route with spike scripts before
cutting cards).

**What is real and must not be rebuilt:** the substrate schema, RLS and write policies (59
`TestRowVisibilityRLS` subtests green), RxDB database creation and conflict-handler wiring proven
in a real browser, the JWT bridge, and the `HQ_SYNC_REST_URL` interlock that has kept all of it
inert exactly as designed. ~11,200 lines of spec. The next milestone's job is to **call** it.

---

## 1. Grades — 8 MET · 1 PARTIAL · 3 NOT MET · 1 UNAUDITABLE (N=13)

KRs are counted at **13**, not 12, because the 2026-08-03 amendment (ledger T-33, operator
sign-off) split D-KR2 into D-KR2a and D-KR2b. Both halves are graded separately below.

| KR | Grade | Basis |
|---|---|---|
| **P-KR1** feasibility gate before schema/JWT WOs | ✅ MET | Spike verdict reached `ledger.md` at triage 2026-07-25 (T-22); both downstream WOs dispatched 07-26 or later |
| **P-KR2** no cutover WO while the hydration item reads `new` | ✅ MET | Item ruled 2026-07-26 (T-24 decision 67, `roadmap.md:2172`); cutover dispatched 08-03 |
| **P-KR3** no-parallel-run constraint carried verbatim | ✅ MET | `roadmap.md:2010`; 0 build WOs proposed a parallel run |
| **D-KR1** 0 of 3 downstream WOs dispatch before the go verdict | ✅ MET | Same timestamp comparison as P-KR1 |
| **D-KR2a** prod parity — `task version` shows 0 drift | ❌ **NOT MET** | The deploy was never run. See §2 — this is a deferral, not a delivery failure |
| **D-KR2b** returning client displays the precached `version.json` frontend version | ❌ **NOT MET** | Card A6 (`app-version-badge`) landed the mechanism on run `20260804`; the returning-client read it exists to enable was never performed |
| **D-KR3** defensible per-card median, N≥8 + exclusions + sensitivity | ✅ MET | `card-actuals.md:864` N=11, median **103m** vs the 94m baseline; `:879` Sensitivity (103–105m, robust to both perturbations); `:890` Excluded-with-reasons. All three reword conditions satisfied |
| **E-KR1** 0 of 2 fetch-storm-class items recur | ❌ **NOT MET** | Deliberate, and deliberately **not reworded**. `sync.js` is still in the tree; both mechanisms live at `:443-454` and `:475-479`. Its two backlog items are **un-dropped** and the class carries forward |
| **E-KR2** 0 attack variants bypass RLS | ✅ MET | `jwtbridge_rls_test.go` → `TestJWTBridgeRLS`: V1–V13 plus 3 controls; caveat B-58 recorded per T-31 decision 121 |
| **E-KR3** 1 documented owner per offline data class **and direction** | ✅ MET | `designs/offline-ownership.md` — card A4, run `20260804`. Closes the reworded KR without a cutover |
| **Q-KR1** RxDB + JWT bridge ship tests in the same WO | ✅ MET | `jwtbridge_test.go` / `jwtbridge_handler_test.go` dated with the `20260726` run; 5 RxDB spec files dated with their own cards |
| **Q-KR2** every code-change WO carries red-run evidence | 🛑 **UNAUDITABLE** | The KR named a `## Red-first` field that **has never existed** in the merge-intent format — exactly one merge-intent this cycle carries the heading. Not repairable retroactively; the reword takes effect from the next run |
| **Q-KR3** every slate carries a Gate-cost section | ⚠️ **PARTIAL, permanently** | Absent from `slate-20260801.md`; one partial mention in `slate-20260802.md`. **Backfilling a signed slate is prohibited** — this stays PARTIAL |

### By team

| Team | MET | PARTIAL | NOT MET | UNAUDITABLE |
|---|---|---|---|---|
| Product | 3 | 0 | 0 | 0 |
| Delivery | 2 | 0 | 2 | 0 |
| Engineering | 2 | 0 | 1 | 0 |
| QA | 1 | 1 | 0 | 1 |
| **Total** | **8** | **1** | **3** | **1** |

**Against the plan.** `reference/okr-completion-plan-20260804.md` §0 recorded **6 MET** on
2026-08-03 and §5 projected **9 MET · 1 PARTIAL · 1 NOT MET · 1 UNAUDITABLE** if the night built
A1/A2/A4/A6 *and* the operator ran the deploy. The night delivered all four cards; the deploy did
not happen. **The entire gap between the projection and this close is that one 15-minute attended
action** — E-KR3 moved UNMET → MET and D-KR3 PARTIAL → MET exactly as planned, and D-KR2's two
halves are the only rows that missed their projection.

---

## 2. Why D-KR2a and D-KR2b are NOT MET, recorded accurately

**Operator decision, 2026-08-05 attended close:** close now rather than deploy first, on the
stated ground that *"there is nothing new feature-wise shipping to prod."*

🛑 **The record must not let that reason harden into a fact it isn't.** `dev` is **436 commits
ahead of `main`**; `main` sits at `32afb39` (2026-07-24, backend 0.2.2). `dev` carries backend
**0.3.0** / frontend **1.4.0**, migration `0072_app_timezone_new_york.sql`, and A6's user-visible
version badge. It is true that no *new feature tab* ships; it is not true that nothing ships.

So the honest grade is **NOT MET by deferral** — the evidence was reachable in ~15 minutes and the
operator chose not to spend them at this boundary. That is a legitimate call. It is graded as a
miss rather than N/A because the precondition **did** fire once A6 landed: the thing D-KR2b
measures became measurable and simply was not measured. (The N/A precedent — QA KR4 last cycle,
*"no schema migration shipped"* at `ledger.md:510` — does not apply, because here a migration is
precisely what is waiting.)

🛑 **A deploy tomorrow does not repair either grade.** The amendment's own rule forbids
backfilling a grade after the boundary. Both stay NOT MET in this cycle's record permanently, and
the deploy — when it happens — is the *next* cycle's evidence.

**Still owed to the sales-processor maintainer:** the migration `0072` changeover date, named
twice as the only outstanding item (T-35 decision 143, T-36). It is unblocked and gated only on
promoting `dev` to `main` and deploying. **This close does not discharge it.**

---

## 3. Delivery metrics (aggregate)

- **Window:** 9 runs — `20260725` · `20260726` · `20260727` · `20260729` · `20260729-2` ·
  `20260801` · `20260802` · `20260803` · `20260804`. All merged into `dev`, all triaged.
- **Cards:** 30 distinct roadmap cards under Activity 1 — **28 landed**, 1 dissolved
  (`sync-rxdb-schema-and-replication`, 2026-07-28), 1 parked-then-retired (`sync-hard-cutover`,
  parked on `20260803`, retired by T-31 decision 126 in favour of reads-on-RxDB / writes-on-REST).
- **Median card cycle:** **103m (N=11)** vs the prior cycle's **94m (N=12)** baseline — flat
  across a population that absorbed a Supabase stack, RLS, and a client-library migration.
  Sensitivity range 103–105m. **B-39 stands:** 11 of 18 candidate cards were countable; two
  consecutive cycles have now produced un-countable cards.
- **Final run (`20260804`):** 4 of 4 cards landed, 0 parked, median 71m56s (N=4).
- **Decisions recorded this cycle:** **99** (decisions 49–147), across ledger sections T-22 → T-36.
- **Open forks at close:** **0.**
- **Suite at close:** 🛑 **no citable figure.** The `20260804` closeout's `786 passed / exit 0` did
  **not** reproduce — an independent run on the same tree gave `785 passed / 1 failed / 6 skipped,
  exit 1` (B-131). Per T-31 decision 131, a green can be an artifact too.

### Armed reds carried out of this cycle — none retired

**B-27** plus three others, and **B-131** (`[RUN-10]`) added at triage 2026-08-03. An armed red
stays armed until someone **diagnoses** it, not until it passes once (decision 100; T-31 decision
120, where four passed and none retired). The attended `task sandbox:e2e` flag is likewise **still
unsatisfied and still armed**.

---

## 4. Unmet reflections — the target-vs-tool ladder

The ritual's step 3 walks each unmet entry. 🛑 **This repo has no `reflections.md`** — the file
`night-crew okr audit` maintains has never existed here, so there is no reflection set to flip.
The ladder is therefore applied to the three NOT MET grades directly.

| Unmet | Rung | Disposition |
|---|---|---|
| **D-KR2a / D-KR2b** | **1 — target-side** | An operator scheduling choice at this repo's own boundary. No night-crew behaviour caused it. Stays in hq: the deploy is the next cycle's first attended item |
| **E-KR1** | **1 — target-side** | The fetch-storm class survives because the cutover that would have removed it was retired on a measurement (T-31 decision 126), not because any tool misbehaved. Its two backlog items are un-dropped and carry to the next roadmap round |

**Tool-implicating (rung 2): one finding, and it is a repeat.** Every milestone verb is blind
against this target — see §5. It was found at the *previous* close, transferred then, and remains
open in the clone's backlog.

---

## 5. 🛑 The close ran no CLI step — all four verbs failed

Recorded in full, because a hand-graded close that does not say so reads as a tool-verified one.

| Step | Command | Result |
|---|---|---|
| 0 | `night-crew scorecard` | `No runs to show.` — a **vacuous** pass, not a clean one |
| 1 | `night-crew okr grade` | **exit 1** — `no metrics.jsonl found under .night-crew/runs` |
| 2 | `night-crew okr audit` | **exit 1** — same error |
| 4 | `night-crew milestone export -name hq` | **exit 1** — `no runs after "20260724" — nothing to export` |
| 7 | `night-crew milestone mark` | run — see §6 |

**Root cause, and it is not a flag or config problem** (`-repo .` changes nothing; `night-crew.toml`
declares no runs/metrics keys; the binary is current at **v3.2.0+6**, built from `f31fff2 (main)`,
and was **not** rebuilt from the dev clone for this ritual — memory `nc-tooling-tracks-main`).

hq's nights since 2026-07-24 were **hand-run slates, not `night-crew run` dispatches**.
`run-evidence check` names the five artifacts the CLI reads — `scorecard/<runid>.jsonl`,
`runs/<runid>/metrics.jsonl`, `runs/<runid>/summary.json`, `runs/<runid>/journal.jsonl`,
`reference/conflicts-<runid>.md`. **None exists for any run in this window.** hq's run
directories hold `HANDOFF.md`, `DECISIONS-NEEDED.md` and `merge-intents/` and nothing else.

Compounding it, **B-77**'s two path families are live: the CLI seeks `.night-crew/runs/20260804/`
while this repo names it `2026-08-04-autonomous`, and it seeks a root-level `reference/conflicts-*`
while this repo — being scaffolded — keeps them under `.night-crew/knowledge/reference/`.

**This is the second consecutive close to hit it** — and checking the first one turned up a
second defect.

`hq-20260724.md` in the clone opens with the same disclosure, claiming the blindness was *"filed
as B-105"* and that 3 tool-implicating findings (B-105 · B-106 · B-107) were transferred at that
close. 🛑 **Both claims are false.** The clone's **B-105** is an unrelated question about which
per-change discipline to adopt; **B-106** is a workflow-preflight reporting gap and **B-107** a
hardcoded grep in `nc-dev-close` — all three predate that close and carry origins from a roadmap
round and two G6 reviews. `grep 'milestone-close hq'` over the clone's `BACKLOG.md` returns
**nothing**: no entry has ever originated at an hq milestone close. **The CLI blindness was
never actually filed.** Offered properly at this close instead — see the ledger entry.

The nearest existing handle is the clone's **B-28** (*"This repo has no milestone-close ritual,
and its OKRs cannot be graded"*), but that is scoped to the night-crew clone closing **itself**
and was already promoted → `add-dev-milestone-close`, which shipped as `/nc-dev-close`. It does
not cover a **target** repo — hq — where `/nc-milestone-close` is the supported ritual and still
cannot run.

---

## 6. Boundary marked — with a defect, left visible

`night-crew milestone mark -name hq-20260805` exited 0 and appended the marker. **It is the one
ritual step that ran.** But it reported:

    marked milestone "hq-20260805" (runs through (no runs yet))

🛑 **The written record carries NO `last_run` field**, where the 2026-07-24 marker carries
`"last_run":"20260724"`. Same root cause as §5: the CLI cannot see this repo's runs, so it could
not name the window boundary and wrote the marker without one.

**Deliberately NOT hand-patched to `20260804`.** A hand-written `last_run` would be
indistinguishable to the next reader from one the tool established — which is precisely the
failure mode this entire close is documenting. The gap is recorded here and filed instead. It
costs nothing functionally today, because the next export cannot run either.

---

## 7. What carries into the next cycle

- **The deploy** — `task prod:deploy` → `task version` → returning-client version read. First
  attended item of the next cycle; also discharges the `0072` changeover date owed to the
  sales-processor maintainer.
- **E-KR1's two un-dropped backlog items** — the fetch-storm class is live and unaddressed.
- **B-89** (`cachedGrantSlugs()` returns `[]` unconditionally — a live bug in shipped code) and
  **B-132**, routed to "the next night" by T-34 decision 137 but never promoted to a card.
- **B-139** — `/period-summary` logs no successful request and no `ready` verdict, with the
  **B-81 / B-82 / B-86 / B-93** "a check cannot tell you what it actually did" cluster.
- **Five armed reds**, none retired, plus the armed attended `task sandbox:e2e` flag.
- **90 open `· new` backlog items** of 103 — the roadmap round's feedstock.
- **§8 of the completion plan ("Two stores")** — decision 126's option (i), assessed as *"the
  honest end state but a milestone rather than a card."* Whether it becomes the next cycle's named
  destination is an open question for the roadmap round.
