# OKRs

Milestone: Prod current and honest — production runs today's code, ingests sales again, and tells its consumers the truth when something moves.

Grading modes: declared

<!-- Authored 2026-08-28, attended `/nc-roadmap-round`, in the same sitting as
`.night-crew/knowledge/roadmap.md` (DESIGN §15j.42). Previous cycle's page archived at
`reference/okrs-2026-08-28-sync-dev-complete.md`; its close at ledger T-45 (13 MET · 1 PARTIAL
(D-KR3) · 1 NOT MET (Q-KR4)).

Authoring rules carried from the last two closes:
  1. Each objective has at least one KR that measures its own claim directly — where an
     objective says a thing works, a KR asserts the thing was RUN and seen.
  2. Every KR names its `(measured by: …)` artifact in the parenthesized form the validator
     parses (prose `Measured by:` silently fails — probed at the 2026-08-05 round), and the
     artifact exists today or the KR is declared disclosed-deferred.
  3. Every KR declares its grading mode at authoring — derived · attested · disclosed-deferred
     — because the last close of this class graded 15 of 16 by hand, four weeks after anyone
     remembered what was meant.
  4. A KR must not be failable by desirable behaviour.
  5. Denominators are named by rule, not literal count, where the population can grow
     (B-40's lesson).

Honesty note on modes: this target's nights are hand-run and write no computed metrics, so
most verdicts are evidence-plus-judgment — declared `attested`, which is the honest common
answer, not a failure to declare. The two `derived` KRs name a command whose exit code or
output IS the verdict. -->

## Product

### Objective: The business's weekly numbers are computed from real, current data — prod runs today's code and its sales ingest is alive.

- **P-KR1 — the operator personally observes all 3 close-bar legs** — parity at 0 drift, Toast ingest current within 48h, and the kill-drill alert arriving — recorded as exactly 1 dated ledger line naming what was seen (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 The whole point of the cycle; no card status, KR grade or closeout substitutes for this line, and the milestone may not close without it.
- **P-KR2 — the Toast ingest gap is enumerated and dispositioned day-by-day**: every date-directory missing since 2026-07-28 is either recovered into the archive or written off by name with the retention evidence, 0 days silently absent (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 "Gapless" was true at measurement only because B-146 was caught inside Toast's ~27-day window — this KR makes the final state explicit rather than assumed.
- **P-KR3 — the combined counterparty notice is sent BEFORE the 0072-carrying deploy runs**, recorded as a dated ledger line naming what it disclosed, and 100% of statements in the two contract docs match what prod actually serves at close (graded: attested · measured by: `docs/contracts/inventory-period-summary.md`) 🛑 Ordering is the KR: a notice sent after the deploy grades this NOT MET even if sent.

## Delivery

### Objective: What dev proved is what prod runs — shipped through the release flow, parity-verified, with the cycle's cost honestly countable.

- **D-KR1 — prod parity verified at least 1 time with 0 drift**: `task version` shows prod backend/frontend == the local `version.go` constants, recorded as a dated ledger line (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 Inherited through two cycles (T-37 decision 149); this cycle retires it or explains itself.
- **D-KR2 — 100% of merged cards carry all three timestamps** (implementer-return, G6-return, fix-return where a fix round ran), so the close computes a per-card cycle-time median with N = every merged card and 0 excluded-without-reason (graded: attested · measured by: `.night-crew/knowledge/reference/card-actuals.md`) 🛑 The producing step is named per B-377's lesson: the slate ritual's per-card stamp block in `card-actuals.md`, maintained at each leg's return — not computed retroactively at close.
- **D-KR3 — the rollback path is real at the moment it matters**: after the deploy, the `:rollback` image tag's existence and target are verified and recorded in the same ledger line as the parity check, 1 time minimum (graded: attested · measured by: `.night-crew/knowledge/ledger.md`).

## Engineering

### Objective: The sync surface survives hardening — its guards are tested, its writes single-fire, and every comment in it tells the truth.

- **E-KR1 — the 2 unguarded behaviors gain red-first regression tests**: the uid-mismatch envelope check and the awaited `clearApiCache()` each pinned by 1 test recorded red on the pre-change tree, then green (graded: attested · measured by: `tests/`) 🛑 Red-first evidence lives in the cards' merge-intents under gate RF (decision 153); a green-only test grades this down.
- **E-KR2 — exactly 1 CDC fire per `/saveResponse` call**, proven by a trigger-count assertion that reds on the pre-change tree (where it counts 2) (graded: attested · measured by: `backend/internal/workflow`).
- **E-KR3 — 0 comment sites in the sync surface still gate on merged or retired cards at close**: the 4 stale activation-gate sites (B-140) and the 2 `proxy.go` fictions (B-18) corrected, with the sweep's site list named in the card's merge-intent so "0" is a checked claim, not a grep hope (graded: attested · measured by: `sync-rxdb`).
- **E-KR4 — 100% of sync projection writers populate `app_slug` from a stored association**, 0 constants remaining, with the association's home recorded as a decision in the card's merge-intent (graded: attested · measured by: `backend/internal/sync`) 🛑 Spike B's carried open question (B-160), decided rather than re-carried.

## QA

### Objective: Nothing in this pipeline can die silently — failures announce themselves, and the planning surface the rounds read is machine-true.

- **Q-KR1 — the kill-drill passes**: with the SFTP path deliberately broken, a failure alert arrives through the Cliq path within 1 scheduled sync cycle, observed by the operator, then the pipeline is restored and observed green — both halves recorded (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 This KR is failable only by silence — an alert that fires is MET, an alert that doesn't is NOT MET, and "the drill wasn't run" is NOT MET, never N/A.
- **Q-KR2 — `night-crew backlog check` exits 0 on the migrated BACKLOG.md** and `night-crew backlog list` emits every entry the document holds (list count == document entry count), with a content-preservation diff recorded in the card's merge-intent (graded: derived · measured by: `.night-crew/knowledge/BACKLOG.md`) 🛑 The exit code and the count equality ARE the verdict — run at close, no judgment involved.
- **Q-KR3 — the scorecard sees the four roles**: at close, `night-crew scorecard` renders at least 1 record-backed row per rostered team from this cycle's hand-run nights, OR the mechanical blocker is recorded with its clone-side handle and the target-side half is done (graded: derived · measured by: `.night-crew/knowledge/scorecard`) 🛑 The scorecard's own output is the verdict; the OR-arm exists because the CLI may own half the fix, and discovering that is a finding, not a miss.
- **Q-KR4 — 100% of this cycle's slate documents carry a `Gate cost` section exactly once**, present at signing, never backfilled (graded: attested · measured by: `.night-crew/knowledge/reference/`) 🛑 Last cycle's only NOT MET; the template fix is clone-side (B-376) but carrying the section is this repo's slates' own discipline, template or no template.
