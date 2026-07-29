# Handoff — `overnight-20260729-2`

> **Run branch:** `overnight-20260729-2` (not merged to `dev` — that is morning triage's call)
> **Slate:** `slate-20260729-2.md`, signed by the operator 2026-07-28, 5 cards, CONCURRENT across
> three tracks.
> **Started** 20:00 EDT · **finished** 23:25 EDT · **3h25m of an 11h budget.**
>
> **3 cards landed and merged. 2 cards parked.** Neither park was a failure to build — both were
> refusals to decide something that is yours.

---

## The short version

**What you can use tomorrow:** the test harness no longer lies, and the sync milestone has its
schema foundation. Both merged, both adversarially reviewed.

**What needs you:** two product decisions, one of which touches payroll and spans two
repositories. Plus a mockup awaiting your signature — the only thing standing between
`sync-rxdb-conflict-notice-ui` and a slate.

**The night's shape, honestly:** every card came in at or under estimate, and the two parks both
happened for the same underlying reason — **tonight's work discovered facts that earlier decisions
were made without.** That is the system working, not failing.

---

## Per-card outcomes

| Card | Track | Outcome | Estimate | Actual | Review |
|---|---|---|---|---|---|
| **H1** `test-harness-fail-loud` | A | ✅ **MERGED** `526efd1` | 1h15m–2h | ~1h05m + repair | G6 APPROVE-WITH-NITS |
| **C1** `conflict-notice-mockup-amendments` | C | ✅ **MERGED** `e0f3247` | 45m–1h15m | ~26m + 2 repairs | Verifier PASS (3rd pass) + G6 APPROVE-WITH-NITS |
| **B1** `sync-rxdb-collections-and-table-contract` | B | ✅ **MERGED** `95a2657` | 2h15m–3h15m | ~53m + repair | G6 APPROVE-WITH-NITS |
| **B2** `sync-rxdb-row-visibility-rls` | B | 🅿️ **PARKED** (docs merged `4de3ca0`) | 2h45m–4h15m | ~15m to park | G6: **park CORRECT** |
| **A1** `app-timezone-unify-new-york` | A | 🅿️ **PARKED, not merged** | 2h15m–3h15m | ~1h18m built | G6: **REJECT** |

---

## What landed

**H1 — the harness stops lying.** Two mechanisms. `task test` was running **19 of 20 spec files**
in a fresh worktree and reporting success; `bdd:gen` now joins its dependency chain. And a
**dropped or unreachable database was reporting `ok`** across the whole Go tree — now, where
`DB_TEST_URL` is *set but unreachable*, the suite fails loud. Skip-on-**unset** is deliberately
preserved: a contributor without a database must still run the unit tests. The bug was the
*symmetry*, not the skip.

This card mattered more than its size. Its G6 settled the riskiest reasoning by execution rather
than reading — that fixing five `TestMain`s subsumes ~25 unedited per-test skip sites — and
confirmed every named site now fails loud with **zero DB-backed skips** remaining.

**B1 — the sync schema foundation.** Four replicated RxDB collections mirroring the Postgres domain
model, one local collection, and the self-hosted table contract as SQL. Schema only: no
replication, no policy. It proved the SQL by *executing* it twice against a scratch database —
idempotent, triggers stamping `_modified`, and **`SET ROLE authenticated` returning 0 rows where
the owner saw 1.** RLS is deny-all, which is the correct state until B2 lands.

**C1 — the revised conflict-notice plates.** 16 plates, 32 renders, 35 `done_when:` rows.
Amendments A-1 (the banner shows what happened *and* how many rows are still unhandled) and A-2
(the override states what it destroys, and confirms while showing the values about to be
overwritten). **This does not discharge the sign-off** — see next actions.

---

## What parked, and why it isn't failure

**B2 — the projection cannot be written where decision 61 says it must.** The projection and the
mutation are in **two different Postgres servers**; no transaction can contain both. Verified twice,
on both servers, by two independent agents: 0 of 5 supabase roles on HQ, all 5 on the substrate,
0 of 4 source tables on the substrate, `max_prepared_transactions = 0` at *both* ends, and `Sign()`
an allowlist that can only emit `authenticated`.

The card **deliberately committed no red**, and that judgment is the finding. It could have seeded
the projection by fixture and produced a green 16-variant matrix identical in shape to the JWT
bridge's — *a security proof about a table nothing in production writes.* On a run whose first card
exists because a suite that ran nothing reported `ok`, that would have been the same defect one
level up.

**A1 — HQ's timezone ruling collides with a published two-repo contract.** The card was built well
and reported DONE. Its G6 returned REJECT and I verified the decisive evidence myself before
parking it: `21-SALES-PROCESSOR-CONTRACT.md` pins `America/Chicago` and carries **A5 — "If the food
truck moves to a different TZ, both repos must update."** Ledger decision 83 never addresses that
contract.

I did **not** salvage the half that doesn't touch sales-processor, because the card forbids it:
*"One card, all sites… park it rather than land half."* And the open question decides whether the
money-path sites should move at all.

🛑 **Neither branch was deleted, reset, or renamed.** Both worktrees are preserved and most of A1's
work is reusable whichever way you answer.

---

## Gate evidence on the final merged tree

Run by the orchestrator on the merged branch, **not inherited from card reports**.

- **G1** — `go build ./...` **exit 0**, `go vet ./...` **exit 0**.
- **G2 (Go)** — **exit 0**, 9 test packages, real DB timings (inventory 22.4s, receipt 20.0s,
  recipes 21.7s, sync 18.5s, purchasing 7.3s, auth 2.9s, workflow 2.1s). **DB liveness proven, not
  assumed: 48 tables and goose version 71** in a freshly created `hq_test_go_final2`. Goose sits at
  **71, not 72, precisely because A1 did not merge** — migration `0072` is unclaimed.
- **Harness self-check** (`scripts/verify-test-harness.sh`, the card's own falsifiable gate) —
  **RAW EXIT 0**, all four checks: `bdd:gen` in the dependency chain; **597 tests in 21 files**
  against a floor of 20; unreachable DB ⇒ exit 1; unset DB ⇒ exit 0, skip preserved.
- **G2 (Playwright)** — **591 passed / 6 skipped of 597, 0 failed, 0 flaky. RAW EXIT 0, 25.7m.**
  Run at `--retries=0` (stricter than `task test`'s default `retries: 1`) after `npx bddgen`,
  against a freshly created `hq_test_e2e_final2` on port 8290. **21 spec files** — 20 under `tests/`
  plus the generated `[bdd]` spec, which ran as test **597** and passed in 3.5s, i.e. it genuinely
  executed rather than being collected. The `webServer` was confirmed up before the run
  ("connected to database", migrations applied) rather than assumed.
  **Neither armed red fired:** `sync.spec.js:446` [LST-17] passed, and `inventory.spec.js:883` —
  the known pre-existing cross-spec pollution — did not surface this run, consistent with its
  load/ordering sensitivity.
- **G4** — `node build-sw.js` **idempotent**: 22 files / 1468.9 KB precached, `git status` clean
  afterwards. `version.go` `Frontend = "1.2.2"` ≡ `package.json` `"version": "1.2.2"`;
  `Backend = "0.3.0"`. **No version moved** — no card asked, and bumping is `/save-project`'s job.
  `sw.js` and `version.json` show **zero diff against `dev`**, which is the expected and correct
  result: no merged card tonight touched a frontend file.

🛑 **On the exit-0-having-run-nothing hazard**, which burned this run's predecessor: every gate
above was run **without piping through `tail`**, with the raw exit code captured on its own line,
and each is reported with a **count** — 9 packages with timings, 48 tables, 597 tests in 21 files.
A number is the only thing that distinguishes "green" from "never ran."

---

## 🛑 Next actions, in order

1. **`/nc-morning-triage`** — review the run branch and merge to `dev`. **Seven open forks** in
   `DECISIONS-NEEDED.md`; **none blocks the merge**, because both parked cards left the tree in a
   safe state rather than a half-changed one.

2. **🛑 Answer D-2 before the next payroll run — it spans two repositories.** HQ publishes a
   contract pinning `America/Chicago` (with assumption **A5: "both repos must update"**), and
   ledger decision 83 ruled the app is New York without addressing it. Until you decide, HQ and
   sales-processor could disagree by one hour at each period edge on the completeness gate.
   **Nothing is broken right now** — A1 did not merge, so the tree still behaves as it did
   yesterday. This is a decision to make, not damage to repair.

3. **Answer D-1 — it is the milestone's critical path.** `sync-rxdb-row-visibility-rls` cannot be
   built until you choose how the projection is written, and three of the four remaining Activity 1
   cards sit behind it. The fork is laid out with real costs; **option (a) is one `CREATE EXTENSION`
   away and its only true blocker is that it reverses decision 61** — which is yours to reverse.

4. **Sign or refuse Card C1's revised plates — D-4.** This is the *only* thing that unblocks
   `sync-rxdb-conflict-notice-ui`. Walk the 16 plates, answer the two decisions deliberately left
   open inside them (the removed-field counting reading, and the retention window), then give or
   refuse *"ok, build this"* on revision 2.

5. **The attended two-device convergence check is NOT re-armed by this run.** The slate expected it
   to be, because Card A1 would have touched `sync.js` and `workflows.html` and regenerated `sw.js`.
   **A1 parked, so no frontend file changed and `sw.js` is byte-identical to `dev`'s.** Whatever its
   state was this morning, it is unchanged. Runbook, if you want it anyway:
   `.night-crew/knowledge/reference/attended-two-device-check.md`.

6. **Before any `task prod:deploy`** — re-run the `submission_fail_notes` duplicate check and record
   it in the deploy note. Migration `0071`'s unique index makes a duplicate arriving in the window a
   crashloop. **Nothing tonight deploys, and nothing tonight added a migration** (`0072` is
   unclaimed), but this re-arms before *every* deploy without exception.

   ```sql
   SELECT submission_id, field_id, count(*) FROM submission_fail_notes
   GROUP BY 1,2 HAVING count(*) > 1;
   ```

7. **Do NOT set `HQ_SYNC_REST_URL`.** The interlock is unchanged: `sync-proxy-endpoint` forwards
   every method with no row filtering, and B2 — the card that was to add filtering — parked. B1's
   four tables are deny-all, so the door is shut, but the proxy is still unguarded.

---

## Follow-ups carried out of the run, not fixed in it

Each was found by adversarial review, judged non-blocking, and deliberately left rather than
widening a signed card.

- **`sync.spec.js:1198` is a dead line anchor** and has been since 2026-07-24, with an unactioned
  migration item filed that same day. Tonight's slate still armed it. Every card told to "expect
  `:1198`" for five nights was told to expect something unobservable. **`:446` [LST-17] is live and
  correct.** → **D-5**.
- **`inventory.spec.js:883`** fails from cross-spec pollution, proven pre-existing by reproduction.
  The first-proposed mechanism was wrong and has been corrected in the record. **`retries: 1` in
  `playwright.config.js` normally masks it** — which is why the baseline reads green. → **D-7**.
- **A deliberate semantics change in H1:** with `DB_TEST_URL` set-but-unreachable, the five
  `TestMain` packages now exit *before* `m.Run()`, so **zero** tests execute — including hermetic
  ones that previously passed. Intended ("setting it is a statement of intent"), documented, and
  worth knowing: a broken DB now yields *no* hermetic signal from those packages rather than
  partial signal.
- **Nothing mechanically ties B1's JS schemas to its SQL DDL.** They match today (hand-diffed, 8/11/6/7
  columns), but "mirroring the Postgres domain model" is the card's contract and drift is
  undetected. Good backlog candidate.
- **`template_snapshot` is `{type:'object'}` with no nested `properties`.** The committed vendor
  bundle ships no dev-mode or validation plugin, so nothing rejects it today. The replication card
  inherits an open question rather than a hidden one.
- **`go test ./...` must be run with `-p 1`.** Parallel packages produce ~120 fabricated failures —
  deadlocks, FK violations on rows just inserted — because DB-coupled `TestMain`s truncate shared
  tables. `Taskfile.yml:108` already does this. Two agents hit it independently tonight. Pre-existing
  harness sharp edge, worth a backlog note.
- **`build-sw.js` silently no-ops outside a git repo** — it shells out to `git ls-tree`, so a bare
  `git archive` extraction produces an empty rebuild that looks successful. A trap for any future
  reviewer verifying `sw.js` from a scratch copy.

---

## The recurring pattern this run kept hitting, worth naming

**Four separate times tonight, a check passed because its population was empty or mis-scoped.**

1. C1's A-2 criterion selected controls by *label* (`/^Restore/i`), so it never measured
   `plate-error`'s `Retry` — the same destructive write, on the one plate where the user has already
   failed once. It also printed `0 Restore controls -> PASS` under rename.
2. C1's tap-target check had no population floor: deleting every `Undo` — "the only escape from a
   mis-tapped Restore" — passed green at exit 0.
3. H1's own verification script checked skip-on-unset for only 2 of the 7 converted packages,
   missing the five where the new ordering is delicate; the mutation that breaks them reported
   "harness OK".
4. H1's other check graded the *mechanism* (`bddgen` runs) rather than the *property* (the runner
   resolves every spec file).

All four are now fixed with population floors and behaviour-scoped predicates, each proven by
running the mutation. But the pattern is the point: **this repo's characteristic bug is a check
whose subject set can go empty.** It is the same shape as B-09 and B-16, one level up. Worth a
standing item in the reviewer prompt.

---

## Housekeeping

- **Conflict log:** `.night-crew/knowledge/reference/conflicts-20260729-2.md` — **5 entries for 4
  merges**, the fifth recording A1's *non*-merge, because a missing entry is indistinguishable from
  a merge never attempted.
- **Trailers (B-21):** every commit on this branch parses. **Including all five merge commits** —
  the previous run's merge commits carried no trailer at all, and its closeout's was split by a
  blank line. Verified with `git interpret-trailers --parse`, not by eye.
- **Databases:** every agent created uniquely-named databases and dropped only its own. `hq_test_go_h1`,
  `hq_test_e2e_h1`, `hq_test_go_b2` were deliberately left in place for reviewers. No repeat of the
  B-16(a) incident.
- **Worktrees preserved:** `a1-app-timezone-unify-new-york` and `b2-sync-rxdb-row-visibility-rls`.
- **No push, no tag, no deploy, `main` untouched.**
