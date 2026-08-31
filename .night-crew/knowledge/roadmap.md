# Roadmap — "Prod current and honest" cycle (ship what dev proved, and make prod unable to lie)

> **Cycle:** Prod current and honest — production runs today's code, ingests sales again, and
> tells its consumers the truth when something moves. **Traces to:**
> `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA, authored in the same
> sitting per DESIGN §15j.42). **Produced:** 2026-08-28 attended `/nc-roadmap-round`, at the
> milestone boundary. Previous cycle ("Sync, dev complete") archived at
> `reference/roadmap-2026-08-28-sync-dev-complete.md` +
> `reference/okrs-2026-08-28-sync-dev-complete.md`; its close at ledger T-45 (close line,
> marker `hq-20260828`); its retro at `reference/retro-sync-dev-complete.md`.
>
> **Trigger — stated plainly.** The last cycle ended with the sync capability attested by the
> operator's own hands **in dev** — and production meanwhile running code from before two
> months of fixes: `dev` is 436+ commits ahead of `main`, the deploy was explicitly delayed at
> the close, **prod's Toast sales ingest has been silently dead since the 2026-07-28 image
> rebuild** (B-146 — the SFTP key never ships in the image; caught only because B-145's
> recovery happened to look), and the `/period-summary` contract drifted on 2026-06-06 with
> the payroll consumer never told (B-29). The business's weekly numbers ride on a pipeline
> that is stale, partially dead, and silent about both.

## The operator's acceptance criterion

> *As the owner whose payroll runs on numbers HQ serves, I want production running today's
> code, ingesting sales again, and telling its consumers the truth when something moves, so
> that the weekly numbers are computed from real data — and what I attested in dev is what
> the crew actually runs.*

**The close bar, chosen at this round — three legs, all operator-verifiable:**

1. **Parity:** `task version` shows prod backend/frontend == local `version.go` constants,
   0 drift, recorded in `ledger.md` (retires D-KR4's inherited debt from two cycles).
2. **Ingest alive:** the operator sees Toast ingest current in prod — last successful sync
   within 48h of the check, from the app or health surface, not from a shell on the box.
3. **The kill-drill:** with the pipeline deliberately broken (key unreadable or SFTP
   unreachable), the failure **announces itself** through the alert path within one scheduled
   cycle — observed by the operator, then restored. A pipeline that can die silently has not
   left the class this milestone exists to retire.

🛑 **The milestone may not close until the operator has personally seen all three.** No KR
grade, card count or closeout substitutes. (Per the standing "dev complete means the operator
ran it" rule — decision 161's class, `process/C-3` pending.)

## How this roadmap works

- **Activity-level cards**, WO-sized, each carrying a module footprint and a KR trace.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Build order is load-bearing:** Activity 1 (the counterparty notice) **gates** Activity 2
  (the deploy) — migration `0072` changes what sales-processor sees and the changeover date is
  owed to them (T-37 decision 149, decision 158). Activity 2 gates Activity 3's fixes
  *reaching prod* (everything lands via dev→main). Activities 4–5 are overnight-parallel with
  3 — disjoint footprints.
- **Red-first (gate RF)** per decision 153: every code card records the named test red on the
  pre-change tree, then green. Docs/audit cards record `n/a — no code change` explicitly.
- **Spike-class sizing note (retro):** spike estimates ran at/under the low band 4 consecutive
  times last cycle — size spike cards one band down this cycle.
- 🛑 **Tests run on `:5434` (`yumyums-test-pg`), never `:5433`** — standing rule, decision 155.
  The Group 3 coordinate-safety backlog items were walked this round and deliberately left
  `new`; the structural guard is the test cluster itself. A mistyped coordinate is still
  possible — stay deliberate.

## Module footprints (independent → parallelizable)

| Footprint | Files |
|---|---|
| **prod infra** | `docker-compose.prod.yml`, prod-clone secrets pattern (`.env.prod`-adjacent), `backend/Dockerfile`, root `Taskfile.yml` |
| **receipt/toast worker** | `backend/internal/receipt/**` |
| **inventory endpoints** | `backend/internal/inventory/**`, `docs/contracts/*` |
| **build tooling** | `build-sw.js`, `backend/internal/version/**`, `package.json` |
| **sync client** | `sync-rxdb/*.js`, `tests/sync-*.spec.js`, `night-crew.toml` |
| **backend workflow** | `backend/internal/workflow/**` |
| **backend sync** | `backend/internal/sync/**`, `backend/internal/db/migrations/**` |
| **planning docs** | `.night-crew/knowledge/BACKLOG.md`, slate/closeout templates under `reference/` |

---

## Activity 1 — Tell the truth first (the counterparty notice)

> **Why first:** migration `0072` and the 2026-06-06 `/period-summary` gate change both alter
> what sales-processor sees. Decision 158 already ruled these become **one combined notice**;
> two cycles have carried it. Nothing honest ships to prod over an untold consumer.
> **Trace:** Product objective.

- **`counterparty-combined-notice`** · **PLANNED** · Mostly **attended** (sending is
  outward-facing — the operator's act; drafting and contract-doc updates are preparable).
  One combined notice to the sales-processor maintainer covering: the `/period-summary`
  completeness-gate change of 2026-06-06 (B-29), migration `0072`'s changeover date (D-KR4's
  carried dependency), and anything else `docs/contracts/inventory-period-summary.md` +
  `inventory-menu-cogs.md` say that is no longer true. Update both contract docs in the same
  change so the docs of record match what prod will serve after Activity 2. Records the
  counterparty-process lesson (B-137) in the ledger: sales-processor is a peer with a contract
  of record, not an external afterthought. Closes **B-29**, **B-137**.
  Footprint: inventory endpoints (docs only) + the outbound notice. **Gates Activity 2.**
  - **Prep half `counterparty-notice-prep` — DONE, run `20260901` (Card 10, Track C).**
    The **preparable** half is complete: both contract docs re-verified against the
    pre-deploy tree (Cards 1+9) — the 2026-08-03 corrections still hold byte-for-byte,
    no response-shape changed; added the B-137 owner line to both doc headers and a
    2026-09-01 re-verification addendum to each §0 (noting Card 9's additive
    `slog.Info` success-log line, which is server-side only, not a wire change). The
    combined first-person notice draft (changelog + pre-deploy checklist, per B-137's
    fix shape) lands at `.night-crew/knowledge/reference/counterparty-notice-20260901-draft.md`,
    covering B-29 (the 2026-06-06 gate change), migration `0072`'s changeover, every
    corrected contract statement, and Card 9's new visibility.
  - **Send half — awaiting the operator (attended).** The card flips to DONE and
    **B-29 / B-137 close only when the operator SENDS the notice** (P-KR3), which must
    precede the `0072`-carrying deploy. The B-137 counterparty-process lesson line is
    captured in the draft's footer for the ledger, to be recorded on send. Card stays
    **PLANNED** until then.

## Activity 2 — Ship (dev→main, deploy, parity)

> **Trace:** Delivery objective. Attended — the release flow is TTY-guarded and the deploy is
> the operator's own command.

- **`ship-dev-to-main`** · **PLANNED** · **Attended.** Merge `dev` → `main`, push,
  `task prod:deploy` (prod clone hard-resets to `origin/main`; the **committed** `sw.js` is
  what ships — B-13's rule), then `task version` parity check: prod backend/frontend ==
  local constants, **0 drift**, recorded as a dated ledger line. Rollback path stated before
  the deploy (`task prod:rollback`). This deploy carries ~2 months of fixes including the
  0072 changeover — hence gated on Activity 1. Retires the D-KR4 debt inherited through two
  cycles. Footprint: release flow only — no code change rides this card.

## Activity 3 — Prod ingest lives, and can't die silently again

> **Trace:** Product + QA objectives. The Toast resurrection is the business half; the
> observability cards are the class-retiring half — B-146's real lesson is not "the key was
> missing" but "it died and nothing said so for 9 days."

- **`toast-ingest-resurrection`** · **PLANNED** · Closes **B-146**. Ship the SFTP key into
  prod the way `.env.prod` secrets already survive the hard reset (untracked file beside it,
  bind-mount or COPY — B-146's recorded fix shape), resurrect the daily sync, and verify the
  archive is still gapless (Toast retains ~27 days; the gap was inside the window when
  measured — re-verify at execution, and if days have aged out, record exactly which, per the
  backfill-horizon discipline of decision 156). Prove: prod's next scheduled sync writes a
  current date-directory. Footprint: prod infra + receipt/toast worker.

- **`pipeline-fail-loud`** · **PLANNED (split — half (b) landed)** · Closes **B-139** and
  B-146's silent-death class. Two mechanisms, one class: (a) the Toast sync loop gets a
  fail-loud path — a sync that cannot open SFTP or auth surfaces in `/api/v1/health` and the
  Cliq alert (the drift-check alert pattern already exists — Phase 999.2), never
  log-and-continue; (b) `/period-summary` logs successful requests and a `ready` verdict, so a
  blocked payroll pull is visible from HQ's side. This card builds what the close bar's
  **kill-drill** proves. Footprint: receipt/toast worker + inventory endpoints.
  - **Half (b) `period-summary-visibility` — DONE, run `20260901` (Card 9, Track C).** Closed
    the B-139 half: `PeriodSummaryHandler` now emits one `slog.Info "period-summary served"`
    at the end of the success path (keys `from`, `to`, `ready`, `pending_review_count`,
    `unlinked_line_item_count`); the `/menu-cogs` sibling gained an analogous
    `slog.Info "menu-cogs served"` (keys `from`, `to`, `menu_item_count`, `breakdown`).
    Red-first tests in `internal/inventory` + `internal/recipes`.
  - **Half (a) `toast-sync-fail-loud` — awaiting (Track A).** The card flips to DONE only when
    both halves land; the orchestrator does the flip after (a) merges.

- **`receipt-worker-correctness`** · **DONE** · Closes **B-28**, **B-175**. Two measured
  defects, one file family: `parseEventDate` stamps COGS periods from server-local time
  instead of the business timezone (B-28 — a period-boundary purchase lands in the wrong
  week); a failed download leaves the next attachment misindexed — ext/filename taken from
  the wrong entry (B-175). Red-first each. Footprint: receipt/toast worker.

- **`deploy-hygiene-honesty`** · **PLANNED** · Closes **B-135**, **B-17**. `version.json`'s
  two shipping generators disagree at the byte level (so prod's precache manifest can carry a
  stale version artifact); `build-sw.js` justifies a load-bearing flag with an empirically
  false claim. Make one generator authoritative and delete or subordinate the other; correct
  the claim or the flag. Footprint: build tooling.

- **`media-recovery`** · **PLANNED** · Closes **B-173**. The onboarding videos and
  fail/correction photos lost with the DO Spaces account: establish what survives (local
  copies, phone originals), re-upload to the current store, and re-point rows. Attended-assist
  — the operator holds the originals; the card builds the re-upload path and does the
  mechanical half. If nothing survives, the card records that as the outcome and closes the
  entry honestly. Footprint: prod infra + onboarding data.

## Activity 4 — Sync hardened past its demo

> **Trace:** Engineering objective. Overnight-parallel with Activity 3 — disjoint footprints.
> T-43(b) (the My Checklists read path) remains **OPEN by ruling**; nothing here decides it.

- **`client-guard-coverage`** · **PLANNED** · Closes **B-149**, **B-10**; carries the
  **B-154 rider** (first sync-touching card: add the `night-crew.toml [e2e.seams]` row for
  `sync-rxdb/` paths, authored deliberately against the sync footprint — until it exists every
  sync card de-confines to the full suite at ~22 min/leg). The uid-mismatch half of the B-89
  fix has zero tests (`cachedGrantSlugs()` envelope verification); the `await` on
  `clearApiCache()` has zero coverage — dropping the `await` leaves the suite green. Red-first
  both. Footprint: sync client.

- **`cdc-single-fire`** · **PLANNED** · Closes **B-157**. One `/saveResponse` call fires any
  CDC trigger on `submission_responses` twice (`workflow/repository.go`) — today's relay
  tolerates it; any future carrier that counts or bills per event will not, and double-NOTIFY
  is double load on the relay for nothing. Red-first: a trigger-count assertion. Footprint:
  backend workflow.

- **`app-slug-association`** · **PLANNED** · Closes **B-160** — the open question spike B
  handed the cutover: HQ stores **no template→app association**, so `app_slug` is a constant
  in every sync projection writer. Decide where the association lives (engineer/PM-level call
  — decide and record in the merge-intent under standing authority; park only if it turns
  product-fork), migrate, and populate the projections from it. Footprint: backend sync.

- **`sync-doc-honesty`** · **PLANNED** · Closes **B-140**, **B-18**, **B-167**. The stale
  `sync-rxdb-row-visibility-rls` activation gate survives at four sites five days after the
  card merged; two `sync/proxy.go` comments describe code that does not exist or launder the
  evidence they record; B-167's bundled G6 carried observations get discharged or converted to
  entries. Docs/comments diff — RF `n/a — no code change` recorded explicitly. Footprint:
  backend sync + sync client (comments only).

- **`sync-dev-one-command`** · **PLANNED** · Closes **B-171** (parked pending a credential
  boundary — the boundary now exists in shape: dev targets carry the 4 `HQ_SYNC_*` vars, prod
  compose does not, and the standing credential-isolation preference says capability, not
  guards). One command (`task sync:dev`) brings up data plane + dev server together for the
  operator's daily use. Footprint: prod infra (Taskfile) + spike scripts.

## Activity 5 — The planning surface is honest

> **Trace:** QA objective. The round that authored this roadmap could see only 63 of ~125
> open backlog items; four candidates could not be rendered onto the triage page at all.

- **`backlog-machine-migration`** · **PLANNED** · Closes **B-02**, **B-168**, **B-12**,
  **B-133** (four filings of one defect — B-38's channel-gap shape happening to the backlog
  itself). Reshape the 193 legacy-shape entries to the canonical B-NN form until
  `night-crew backlog check` exits 0, with **content preservation proven** (stripped-text
  diff: every entry body present before is present after; handles assigned above the current
  max — collisions have happened, B-39→B-44). Then **arm the triage §4.5 gate** so the
  document cannot drift back. done_when is mechanical: `check` exit 0, and `backlog list`
  count == document entry count. Footprint: planning docs.

- **`team-records-from-hand-runs`** · **PLANNED** · The scorecard sees no rostered role on
  this target — hand-run nights write no team records, so every close renders `—` for all
  four teams (retro §3). Scope: emit the per-run scorecard files the CLI already reads, from
  this repo's hand-run slate/closeout ritual (template + ritual step). If that provably
  requires CLI changes, the card records the finding, files it clone-side, and closes with
  the target-side half done. Footprint: planning docs.

---

## Backlog dispositions this round

**Walked:** all 63 CLI-visible open items, in 8 groups. **Not walked:** the ~62 document-only
legacy entries (invisible to `backlog list` — the exact gap `backlog-machine-migration`
closes; they become walkable next round). Group labels were this round's scaffolding and are
deliberately **not** written into `BACKLOG.md`; only each entry's status is.

| Group | Handles | Disposition |
|---|---|---|
| Live prod defects | B-146, B-29, B-137, B-175, B-28, B-173, B-135, B-17 (+B-139 folded in from Observability) | **promoted** → Activities 1–3 |
| Ship it / sync hardening | B-171, B-160, B-149, B-157, B-140, B-18, B-10, B-167 (+B-154 as rider) | **promoted** → Activities 2, 4 |
| Gate coordinate safety | B-169, B-164, B-151, B-152, B-134, B-150, B-170, B-165, B-158, B-159 (+B-50 aggravation) | **left `new`** — walked, and waits. The retro's candidate goes deliberately unfunded; the structural guard (test cluster, decision 155) stands; next round re-asks from scratch |
| Observability siblings | B-81, B-82, B-86, B-93 (document-only) | **left `new`** — invisible to the tooling until the migration lands; B-139 promoted separately |
| Armed reds / load-flake family | B-27, B-30, B-32, B-162, B-156, B-131, B-166, B-174, B-34, B-04 | **left `new`, stay armed** — retired by diagnosis, never by passing once (decision 100) |
| UI polish | B-31, B-05 | **left `new`** — waits for a UI-themed round; becomes crew-visible once Activity 2 deploys, knowingly |
| Measurement debts | B-15, B-07, B-153, B-24, B-23, B-144, B-155 | **left `new`, named knowingly** — B-23's vacuous spec-file floor and B-15's attended two-device cost stated out loud at the walk |
| Planning hygiene | B-02, B-168, B-12, B-133 | **promoted** → `backlog-machine-migration` |
| Planning hygiene (small items) | B-41, B-44, B-25, B-38, B-39, B-40, B-08, B-11, B-21, B-03, B-01 | **left `new`** |
| Tracked elsewhere | B-33 | **dropped** — night-crew tooling defect, tracked clone-side as B-346/B-347 |

**Round decisions recorded at this sitting** (ledger entry accompanies this commit): the PM
rating is a weighted composite — **0.7 × forks-prevented + 0.3 × PRD-to-KR traceability**,
with a not-computable component rendered as N/A and the rating computed from the remainder,
labelled — never silently zero or full marks (§15bk.188 choice, made on the retro's real
numbers). Scorecard role-visibility: worth a small card (`team-records-from-hand-runs`), not
a recorded absence.
