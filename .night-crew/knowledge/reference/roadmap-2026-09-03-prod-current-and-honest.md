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
    - **Notice FINALISED 2026-09-01** (ledger T-48) → `docs/contracts/NOTICE-sales-processor-2026-09-01.md`;
      §3 decided (`menu_item_name` stays, doc is the spec); both contract docs carry owner lines;
      **B-29 + B-137 closed.** Card stays PLANNED for its one remaining thread: record migration
      `0072`'s changeover date in the notice's §4 immediately after `task prod:deploy` (Activity 2).
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

- **`toast-ingest-resurrection`** · **DONE** (dev-provable half; prod proof is attended
  post-deploy) · Closes **B-146** (mechanism half). Run `20260901`, Card 3. Shipped the
  SFTP-key delivery mechanism the way `.env.prod` secrets already survive the hard reset:
  `docker-compose.prod.yml` now bind-mounts an operator-placed, git-ignored `id_rsa`
  (`./id_rsa:/app/id_rsa:ro`), pins `TOAST_SFTP_KEY_PATH=/app/id_rsa` and
  `TOAST_SYNC_INTERVAL=12h` to resurrect the daily worker (a missing key is now a LOUD boot
  failure via config.go's os.Stat guard + Card 2's fail-loud health field). `id_rsa` was
  already git-ignored; **no real key entered the run** — the operator places it, attended, on
  the prod box (attended-steps note in
  `reference/toast-archive-gap-20260901.md`). Archive gap enumerated day-by-day: 38-day gap
  (2026-07-25 → 2026-08-31); **10 aged-out** (2026-07-25 → 2026-08-03, permanently lost),
  **28 recoverable** (2026-08-04 → 2026-08-31) as of tonight against Toast's ~27-day
  retention — recovery of the >7-day slice needs sales-processor's local archive via
  `migrate-toast-archive` (no SFTP range-backfill CLI exists). **PROD PROOF (attended,
  post-deploy):** a current date-directory from prod's next scheduled sync + `toast_sync`
  health flipping to `ok`. Footprint: prod infra + toast worker/config.

- **`pipeline-fail-loud`** · **DONE** · Closes **B-139** and B-146's silent-death class. Two
  mechanisms, one class: (a) the Toast sync loop gets a fail-loud path — a sync that cannot
  open SFTP or auth surfaces in `/api/v1/health` and the Cliq alert (the drift-check alert
  pattern already exists — Phase 999.2), never log-and-continue; (b) `/period-summary` logs
  successful requests and a `ready` verdict, so a blocked payroll pull is visible from HQ's
  side. This card builds what the close bar's **kill-drill** proves. Footprint: receipt/toast
  worker + inventory endpoints. **Both halves landed on run `20260901`.**
  - **Half (b) `period-summary-visibility` — DONE, run `20260901` (Card 9, Track C).** Closed
    the B-139 half: `PeriodSummaryHandler` now emits one `slog.Info "period-summary served"`
    at the end of the success path (keys `from`, `to`, `ready`, `pending_review_count`,
    `unlinked_line_item_count`); the `/menu-cogs` sibling gained an analogous
    `slog.Info "menu-cogs served"` (keys `from`, `to`, `menu_item_count`, `breakdown`).
    Red-first tests in `internal/inventory` + `internal/recipes`.
  - **Half (a) `toast-sync-fail-loud` — DONE, run `20260901` (Card 2, Track A).** Closed the
    B-146 fail-loud half. A dial/auth failure in `SyncDate` used to be downgraded to
    `ErrSFTPMiss` (silent, "expected miss") — a dead transport was invisible. Now dial/auth
    failure returns the new `ErrSFTPUnavailable` sentinel; the worker routes it to a loud
    path via `handleSyncOutcome` — sets `toast.SyncStatus` to `failing` AND enqueues a Cliq
    alert immediately (reusing the existing `alerts.Queue`). `/api/v1/health` gained a
    `toast_sync` field: `{status: ok|failing|stale|unknown, last_success, last_error,
    last_error_summary}`. Genuine date-not-found still returns `ErrSFTPMiss` and stays silent
    (D-05). Red-first tests in `internal/toast` (fake failing dialer + fake alert sink). The
    SFTP key itself ships separately (Card 3).

- **`receipt-worker-correctness`** · **DONE** · Closes **B-28**, **B-175**. Two measured
  defects, one file family: `parseEventDate` stamps COGS periods from server-local time
  instead of the business timezone (B-28 — a period-boundary purchase lands in the wrong
  week); a failed download leaves the next attachment misindexed — ext/filename taken from
  the wrong entry (B-175). Red-first each. Footprint: receipt/toast worker.

- **`deploy-hygiene-honesty`** · **DONE** (run `20260901`) · Closed **B-135**, **B-17**.
  **B-135:** the byte divergence was the trailing newline — `scripts/write-version-json.js`
  (authoritative) writes `JSON.stringify+'\n'` (`{"frontend":"X"}\n`); `backend/Dockerfile`'s
  `printf '{"frontend":"%s"}'` dropped it, so the image served 20 bytes while sw.js's precache
  revision hashed the 21-byte served/local copy. Fix: added `\n` to the Dockerfile printf so
  the embedded copy is byte-identical to the authoritative one; the Dockerfile is now a
  subordinate byte-for-byte mirror. `version.json` is git-ignored — both copies are generated,
  not committed — so parity is enforced at the two generators and pinned by the red-first
  `tests/version-json-parity.spec.js` (RED at 20≠21 bytes / two md5s → GREEN identical).
  **B-17:** corrected `build-sw.js`'s comment — plain `--name-only` C-quotes NON-ASCII/control
  bytes (`core.quotePath`), NOT spaces (a spaced path comes back bare); `-z` stays load-bearing
  because it sidesteps quoting entirely. The `:383` verbatim mirror the card named no longer
  exists in this live roadmap (the card was already a correct paraphrase; line pointer was
  stale after Cards 1–10 reflowed the file). The verbatim claim survives only in frozen
  historical artifacts (`reference/roadmap-2026-08-05-sync-foundation.md:449`,
  `runs/2026-07-29-autonomous/merge-intent-a-precache-manifest-from-head.md:75`) — left as-is
  by the do-not-edit-frozen-artifacts rule. **B-17 residual (BACKLOG-recorded, out of scope):**
  `ls-tree HEAD` reads local HEAD while the image builds from `origin/main`, so the 404 class is
  tighter but not fully closed by `precache-manifest-from-head`. Precache count unchanged at 31;
  version parity 1.5.0. **Env note for orchestrator:** this worktree's symlinked node_modules
  carries `workbox-build@7.3.0` while `package-lock.json` pins `7.4.1`; regenerating sw.js here
  produced a spurious workbox-chunk-hash-only delta (`workbox-0225851e`→`d4a0f5c1`, every
  precache revision byte-identical). My source change touches no precached file and does not
  change the version.json bytes, so sw.js needs NO regeneration for this card — committed sw.js
  left untouched. Footprint: build tooling.

- **`media-recovery`** · **PLANNED** · Closes **B-173**. The onboarding videos and
  fail/correction photos lost with the DO Spaces account: establish what survives (local
  copies, phone originals), re-upload to the current store, and re-point rows. Attended-assist
  — the operator holds the originals; the card builds the re-upload path and does the
  mechanical half. If nothing survives, the card records that as the outcome and closes the
  entry honestly. Footprint: prod infra + onboarding data.

## Activity 4 — Sync hardened past its demo

> **Trace:** Engineering objective. Overnight-parallel with Activity 3 — disjoint footprints.
> T-43(b) (the My Checklists read path) remains **OPEN by ruling**; nothing here decides it.

- **`client-guard-coverage`** · **DONE** · Closes **B-149**, **B-10**; carries the
  **B-154 rider** (first sync-touching card: add the `night-crew.toml [e2e.seams]` row for
  `sync-rxdb/` paths, authored deliberately against the sync footprint — until it exists every
  sync card de-confines to the full suite at ~22 min/leg). The uid-mismatch half of the B-89
  fix has zero tests (`cachedGrantSlugs()` envelope verification); the `await` on
  `clearApiCache()` has zero coverage — dropping the `await` leaves the suite green. Red-first
  both. Footprint: sync client.

- **`cdc-single-fire`** · **DONE** (run 20260901, Card 5) · Closes **B-157**. One
  `/saveResponse` call fired any CDC trigger on `submission_responses` twice — the save
  INSERT plus `EmitOp`'s separate `UPDATE ... SET lamport_ts`. Fixed by folding the
  lamport_ts stamp into the save's own upsert (`saveResponse` gained a `stampLamportTS`
  param; handler emits the op row via the new `opsync.EmitOpForStampedEntity`, no second
  row write). Red-first trigger-count test: 2 fires pre-change → 1 after
  (`internal/workflow/cdc_single_fire_test.go`). The `/ops` path is deliberately left
  unchanged (stamp=0) — its stamp happens after CheckLWW and is out of B-157's scope.
  Footprint: backend workflow (+ isolated additions to backend sync).

- **`app-slug-association`** · **DONE** (run 20260901, Card 8) · Closes **B-160** — the open
  question spike B handed the cutover: HQ stored **no template→app association**, so `app_slug`
  was a constant in the sync projection writer. Association home decided under standing
  authority (E-KR4, recorded in the merge-intent): a nullable FK
  **`checklist_templates.app_id` → `hq_apps(id)` ON DELETE SET NULL** (migration 0076), backfilled
  to the `operations` app (the checklist engine IS Operations — reproduces the constant, changes
  no projected row). `spikec_relay.go` now resolves each row's `app_slug` per-row via
  `appSlugForField` (field → section → template → app.slug); the `AppSlug` config field, its
  guard, and the `SPIKE_C_APP_SLUG` read are gone — **0 hardcoded `app_slug` constants remain**
  in the writer (grep-provable). Red-first: an AST structural test naming the 4 constant sources
  (RED→GREEN) + a per-template resolution test (ops→operations, inv→inventory, distinct).
  Not a product fork (schema-shape choice, invisible to the operator). Footprint: backend sync +
  migration 0076.

- **`sync-doc-honesty`** · **DONE** (run 20260901, Card 7) · Closes **B-140**, **B-18**,
  **B-167**. Retired **five** stale `row-visibility-rls` activation gates (the four B-140 named
  — `workflows.html:329` + `:3946` [the latter spelled the slug; precached → `sw.js` regen],
  `sync-rxdb/conflict-notice-ui.js:26`, `tests/states-sync-rxdb-conflict-notice.spec.js:30` —
  plus a fifth found beyond them in `sync-schema/sql/0001_sync_tables.sql`, gated via the
  slate-ID "B2"), each restated against the genuinely-open precondition: the **cutover** (no
  page calls `startHQReplication`), preserving the substrate-must-carry-RLS safety point where
  it still holds. Corrected the two `sync/proxy.go` comment fictions (B-18a RawPath, B-18b the
  `EscapedPath` log-launder documented with the code fix deferred to a code-footprint card) and
  the ACTIVATION-ORDER banner (substrate-state, not card-landing, precondition). Broadened
  `tests/repo-hygiene.spec.js` case 3 from slug-matching to **fact-matching** — whole-tree scan,
  DONE-slug set read from all roadmaps, asserts no live comment names a DONE card as a future
  precondition; red-first proven (a paraphrased gate on a DONE card is caught, removed→green).
  B-167's six carried observations discharged-by-conversion into per-item checklist entries
  homed to the next flag/fill/lifecycle sync cards. RF `n/a — no code change` for the comment
  retirements; the spec broadening is red-first. Footprint: backend sync + sync client
  (comments only) + `workflows.html` + `sw.js` + `tests/*`.

- **`sync-dev-one-command`** · **DONE** (run 20260901) · Closes **B-171** (parked pending a
  credential boundary — the boundary now exists in shape: dev targets carry the 4 `HQ_SYNC_*`
  vars, prod compose does not, and the standing credential-isolation preference says capability,
  not guards). One command (`task sync:dev`) brings up data plane + dev server together for the
  operator's daily use — it composes `sync:dev:up` (carries the untouched B-164 :5433 refusal)
  → `backend:dev:tailscale` (carries the 4 `HQ_SYNC_*` vars), adding no new logic and no
  coordinate-guard machinery. Also discharged the B-170 bare-`npx playwright` hardening in
  `spike-f-browser-live.sh` (the one Playwright-running script in footprint). Footprint: prod
  infra (Taskfile) + spike scripts.

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
