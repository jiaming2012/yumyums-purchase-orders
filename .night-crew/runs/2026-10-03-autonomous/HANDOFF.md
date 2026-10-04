# HANDOFF — overnight run `20261003` (night of 2026-10-03 → morning 2026-10-04)

**Branch:** `overnight-20261003`, cut from `dev` at `ec2830c`. Nothing pushed, `main` untouched, no
deploy. **Slate:** `reference/slate-20261003.md`, signed 2026-10-02. **4 cards, Activity I.**
Launched 21:41, closeout written ~02:55 America/New_York (~5 h 15 m — the slate's mid projection).

## 🛑 Read these five things first

1. **3 of 4 cards landed; 1 is parked for you.** Landed: `dish-merge-and-erasure-backstop` (I3),
   `test-integrity-fix` (I1), `scan-time-verify` (I2). Parked: `atomic-scan-dedupe` (I4) — built and
   review-approved, but its migration **permanently deletes past scan rows on first deploy**. That is
   **D-1** in `DECISIONS-NEEDED.md`, the one decision waiting for you.
2. **The final full Playwright suite on the complete tree is `EXIT_PW=1` with 34 reds against
   tonight's base of 25 — and one of the extra ten is NOT cleared.** All 63 marketing tests pass.
   Nine of the ten extra reds pass 3/3 alone; **`inventory.spec.js:2931` is red 3 of 5 alone on the
   merged tree and 1 of 5 alone on the base tree**, with an earlier-failing step seen only on the
   merged tree. Five runs per tree neither attributes it to tonight nor clears it. See "Reds".
3. **The night was NOT run through night-crew's run loop, and the end-of-night `loop check` REFUSED**
   (`no run summary at .night-crew/runs/20261003/summary.json`). You chose "as the saved prompt says"
   at launch; this is that choice's stated consequence, not a clean pass. The scorecard record was
   written by hand as on every previous night here.
4. **Two fix rounds were not re-reviewed.** Cards I1 and I2 each got a fix round for their review's
   P2 findings; the orchestrator checked each fix round's diff shape and confined tests, but neither
   went back through a second fresh-context review.
5. **Before the deploy that carries migration `0086`:** it silently blanks any `qr_scans.subscriber_id`
   that names no existing subscriber, one-way. Whether production holds such rows was not measured
   (B-474). Count them first.

## Per-card outcomes

| # | Card | Merge | Review | Fix round | Notes |
|---|---|---|---|---|---|
| 3 | `dish-merge-and-erasure-backstop` (I3) | `ed4f568` | APPROVE-WITH-FINDINGS, 0 P1 / 0 P2 / 5 P3 | — | Reviewer ran the deletes: subscriber delete clears the timeline and blanks scans; un-scanned code delete blanks first-touch; scanned code delete refused (`23503 qr_scans_short_fkey`); merge re-points campaign + code. The merge was exercised through the repository function only, not the HTTP endpoint. |
| 1 | `test-integrity-fix` (I1) | `8a7065d` | APPROVE-WITH-FINDINGS, 0 P1 / 1 P2 / 6 P3 | ✅ red-first | All three false gates red under their mutations when the reviewer re-ran them. P2: the v0→v1 migration is not "total" — a v0 row without `campaign_id` makes a VALIDATING store refuse to open (`DM4`); the phone is unaffected only because its storage wraps no validator. Fixed as a pinned test + corrected comment; the migration is unchanged. |
| 2 | `scan-time-verify` (I2) | `3986161` (+ `fb3a35d` `sw.js`) | APPROVE-WITH-FINDINGS, 0 P1 / 2 P2 / 4 P3 | ✅ red-first | Reviewer drove expired / redeemed / not-yet-replicated / null-campaign / 401 / 500 / malformed rows: no unverified code got a discount path beyond what the phone offered before. P2s fixed: a test now guards "ask the server only when the code is on neither local list"; the result area shows "Checking with the server…" during the wait. |
| 4 | `atomic-scan-dedupe` (I4) | **NOT MERGED — PARKED (D-1)** | APPROVE-WITH-FINDINGS, 0 P1 / 1 P2 / 3 P3 | — | 12 simultaneous taps → 1 row, 5/5. Branch `card/i4-atomic-scan-dedupe` @ `8251ac9`, worktree preserved. Full Go green on the branch; **Playwright not run** on it. |

Every card got a fresh-context review on contract + diff + evidence. All four returned findings the
card had not found.

## What changed for the crew (landed cards only)

- **Scanner, online, code not yet on the phone:** the phone now asks the server before showing the
  offer. Live → the offer; already redeemed → "already used", no discount prompt; server unreachable →
  today's message plus "couldn't check the server", inside 3.5 s; while waiting → "Checking with the
  server…". **Offline: zero network calls, output byte-identical to before** (fixture-compared).
- **Dish merge (API-only — no screen calls it):** now moves the dish's campaigns and QR codes to the
  surviving dish instead of failing with a 500. `CLAUDE.md`'s Merge line was corrected to say so.
- **Deleting a subscriber** removes their timeline and blanks their scans; **deleting an un-scanned
  code** blanks the subscriber's first-touch; **a scanned code still cannot be deleted** (by design).
- Nothing else is crew-visible: I1 changed what the tests measure, not what the phone does.

## Gate evidence on the FINAL tree `fb3a35d` (I3 + I1 + I2)

| Gate | Result | Log |
|---|---|---|
| **G1** | `EXIT_BUILD=0`, `EXIT_VET=0` | `logs/final-G1-G2go.log` |
| **G2 (Go)** | **`EXIT_TEST=0` — 449 pass / 0 fail / 3 skip, 15 packages**, counts checked; `TestRowVisibilityRLS` 59 subtests; `HQ_SYNC_SUBSTRATE_OPTIONAL` + `HQ_SYNC_GATE_CHILD` unset in-log. Base 444/0/3; +5 = I3's tests | `logs/final-G1-G2go.log` |
| **G2 (Playwright)** | **`EXIT_PW=1` — 34 failed / 7 skipped / 1031 passed (57.7 m)**, exactly one summary block. Marketing 63/63 green | `logs/final-pw.log`, `logs/final-reds.txt` |
| **G3** | N/A — `openspec: absent`, re-confirmed by `workflow preflight` at launch | — |
| **G4** | **51 precached**, committed file reproduces on two runs, reachability 38 / 67 / 0 outside. 51 → 51 at every merge, as the slate predicted | `logs/final-G4-sw.log` |
| **RF** | every code-changing card showed its red first; both fix rounds ran a second red-first round | per-card `logs/<card>/` |
| **G6** | 4 of 4 reviewed in fresh context; 4 of 4 found something | this file |

There is no G5. The G4 discipline greps are N/A-VACUOUS in this repo (B-14).

## Reds — what is and is not known

**Base, measured tonight on `dev@ec2830c`:** Go clean (444/0/3). Playwright 25 failed / 7 skipped /
1030 passed, all 25 inside the previously measured sets (`logs/base-reds.txt`). The first base attempt
was invalid (webServer 60 s timeout, zero tests) and is kept as
`logs/base-pw-invalid-1-webserver-timeout.log`. The base ran with 7 skips where last night ran 6; the
extra skip was not identified.

**Final tree:** 24 of the 25 base reds still red; `inventory.spec.js:1469` went green; **10 outside base.**
Each of the ten was run 3× alone, database reset and server restarted before every run
(`logs/final-isolation.log`):

| Test | In the full suite | Alone |
|---|---|---|
| `inventory.spec.js:2931` creating item opens edit form | `.item-edit-form` not found | **merged: 3 red / 2 green · base: 1 red / 4 green** |
| `onboarding.spec.js:2233`, `:2268` | 30 s timeouts | 3/3 green each |
| `sync.spec.js:836`, `:1691`, `:1756`, `:2509` | timeouts; `:1756` expected "LOSER", got "WINNER" | 3/3 green each |
| `workflows.spec.js:2824`, `:3528`, `:4214` | timeouts; canvas never removed | 3/3 green each |

- **`inventory.spec.js:2931` is open.** It is red alone on both trees, so it is not exclusive to the
  merge — but on the merged tree three of the five failures were at an EARLIER step
  (`#new-item-name` never visible/editable, `:2936`/`:2938`), a message never seen on the base tree.
  No landed card changes `inventory.html` or its JS; I3 changes `backend/internal/recipes/repository.go`
  and adds migration `0086`. Not attributed, not cleared. It belongs to the B-459 Inventory family in
  `bugs.md` by title; that is a description, not an attribution.
- **The nine that pass alone** failed only inside a 57.7-minute single-worker run. None was re-run in
  suite context and none was isolated on the base tree, so "load" is the leading explanation and not
  a measured one. The seven `sync` / `workflows` reds appeared on no card's own suite tonight.
- **Per-card suites, for comparison** (each `EXIT=1`): I3 28 failed, I1 28 failed, I2 27 failed —
  3–4 outside base each, all passing alone on the card tree. I3's full Go run had one red
  (`TestRVClaimFixtureDatabase_RefusesExisting`, a 32 s timeout dropping a probe database while other
  suites shared the cluster); alone it passed in 0.54 s, and the final tree's full Go run is clean.

## Decisions waiting

**D-1 (only one):** what happens to past scan rows when the double-tap fix goes live — see
`DECISIONS-NEEDED.md`. Routed through `night-crew decisions log` → `verdict: park` (medium severity,
citation floor unmet). The record's phrasing notes said the question was not written as a user story;
D-1 as written for you is.

## Review findings that were NOT fixed (all filed)

- **B-469** — a dish merge deletes the source dish's sales history (`daily_menu_sales` cascades; the
  merge does not re-point it). Predates tonight; feeds menu-COGS. Read from the catalog, not run.
- **B-470** — `refusal-harness.mjs:183` is a second stale copy of the refusal predicate.
- **B-471** — the egress guard misses an aliased `net/http` import, a new subdirectory, `net.Dial`,
  `exec.Command("curl")`, a hand-built request, and a send added inside an allowlisted file.
- **B-472** — the "names no campaign → stays overridable" half of the refusal has no automatic gate
  (only `campaigns-run.sh`, which is not in the gate list); and `campaign_id` required is NOT enforced
  on the phone (no validator wrapped).
- **B-473** — no direct retry on the "couldn't check the server" card; a test hook can double-ask.
- **B-474** — migration `0086`'s silent blanking of dangling scan references; no index on
  `qr_scans.subscriber_id`.

## Not verified by anyone tonight

- The scan-time server read through the REAL door / JWT mint / PostgREST. The e2e stack has no
  substrate, so the server's row is stubbed in `[SV-01]`/`[SV-02]` (the slate's one permitted stub).
- The schema v0→v1 migration on a real phone or Safari (B-441 stays open); whether codes/offers
  re-pull from scratch after the version bump.
- The dish merge through `POST /inventory/recipes/merge` with a campaign attached.
- How many production rows migrations `0086` and (parked) `0087` would touch — the run may not read `:5433`.
- The photo-scan path during a hung server lookup.

## Process notes

- **Commit trailers on I3 and on I1's first eight commits are text, not git trailers** — the
  orchestrator's brief put a blank line between `Night-Crew-Run` and `Co-Authored-By`, so
  `git log --format='%(trailers:key=Night-Crew-Run)'` returns empty for them. Corrected in the brief
  for I4, I2 and both fix rounds. The orchestrator's defect, not the cards'.
- **The suite lock was the night's pacing item.** A full Playwright run took 35–58 minutes (the slate
  priced 37). Reviews were started as soon as a card's code was committed, in parallel with its
  queued suite; I4 was dispatched off I3's unmerged tip and I2 off I1's. Both follow-on merges were
  three-way, never squashed.
- **The full suite rewrites 16–18 tracked screenshots under `.night-crew/runs/2026-10-02-autonomous/`**
  (`h4/states`, `h5/states`). Every session restored them and committed none; this closeout touches
  none of them (B-467).
- **Post-merge gates at merges 1 and 2 were confined** (build, vet, the touched Go packages, `sw.js`),
  not full suites; the final full run is the combined gate.
- Scratch databases left on `:5434`: `hq_test_go_{base,i1,i2,i3,i4,g6i1,g6i3,g6i4,final}`,
  `hq_test_e2e_{base,i1,i2,i3,g6i1,g6i2,final}_20261003`, `hq_rls_*_20261003`. Substrate still up.
- Worktrees left: `i1-test-integrity-fix`, `i2-scan-time-verify`, `i3-dish-merge-and-erasure-backstop`
  (all merged), `i4-atomic-scan-dedupe` (PARKED — keep). The review, base and final worktrees the
  orchestrator created were removed.
- `card/a3-rls-fixture-own` and `card/s2-demo-sync-target` still hold un-landed work (B-442, your
  2026-09-05 ruling: leave in place). Untouched.

## Next actions

1. `/nc-morning-triage` — review and merge `overnight-20261003`, rule on D-1.
2. Decide what to do about `inventory.spec.js:2931` before trusting the merge as regression-free.
3. Count dangling `qr_scans.subscriber_id` rows on production before any deploy carrying `0086`.

`git log --oneline dev..overnight-20261003` is the full record of the night.
