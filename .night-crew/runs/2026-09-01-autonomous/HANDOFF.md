# HANDOFF — run `20260901`

**Slate:** `reference/slate-20260901.md` (signed 2026-08-31, operator "Yes — sign it"). **11 cards.**
**Run branch:** `overnight-20260901` (off `dev` @ `55aa6f8`). **NOT merged to `dev`** — that is morning triage's act.
**Executed:** evening 2026-08-31, ~16:00→~19:15 EDT (~3h15m wall). **First CONCURRENT 3-track night in this repo.**
**Dispatch:** orchestrator + fresh subagent-per-card in worktrees; a separate fresh-context G6 per card; orchestrator alone merged, in landing order. Tracks A/B/C dispatched concurrently, one in-flight card per track.

---

## 🟢 OUTCOME: ALL 11 CARDS LANDED. 0 parked. 0 operator forks.

The build-everything-the-deploy-will-carry night delivered every unattended-buildable card in Activities 1, 3 and 4. The morning is now: **review + SEND the counterparty notice → merge dev→main → `task prod:deploy`**, and the deploy carries two months of fixes plus tonight's, with the kill-drill mechanism already in the tree.

---

## 🛑 DO THESE FIRST (in this order)

1. **Review + SEND the counterparty notice — BEFORE the deploy (P-KR3).**
   Draft: `.night-crew/knowledge/reference/counterparty-notice-20260901-draft.md`.
   It tells the sales-processor maintainer about: the 2026-06-06 `/period-summary` completeness-gate change (B-29), migration `0072`'s Chicago→NY changeover (gated on the deploy that runs it; `/menu-cogs` does NOT move), the long-undocumented additive `/period-summary` fields, the four never-true `/menu-cogs` field names + cost-formula corrections, and tonight's new server-side visibility. Both contract docs (`docs/contracts/inventory-period-summary.md`, `inventory-menu-cogs.md`) were re-audited against tonight's tree and are accurate. **The ordering (notice before deploy) is the KR — do not deploy first.**

2. **Triage this run branch**, merge `overnight-20260901` → `dev` (this is triage's act, not the run's).

3. **After the deploy — place the Toast SFTP key (attended).** Card 3 built the mechanism (bind-mount `./id_rsa:/app/id_rsa:ro` in `docker-compose.prod.yml`, `TOAST_SYNC_INTERVAL=12h`); the attended steps are in `.night-crew/knowledge/reference/toast-archive-gap-20260901.md` — place the key beside the prod compose (`chmod 600`), redeploy, verify `task health:prod` shows `toast_sync: ok`, then confirm a current `toast/YYYYMMDD/` dir after the next sync. **No prod credential entered the run.**

4. **Attended, post-deploy:** the kill-drill (prove the fail-loud path fires) + toast prod-proof. These are the close bar's legs; they cannot be slated.

---

## Per-card outcomes (all merged to `overnight-20260901`)

| # | Card | Track | Merge | G6 | Notes |
|---|---|---|---|---|---|
| 1 | `receipt-worker-correctness` | A | `0e988ce` | PASS-W-ISSUES | B-28 unparseable fallback → business tz + WARN; B-175 misindex via `blobWithAttachment`. **G6's parseable-path boundary finding WAIVED → filed B-177** (out of B-28 scope, entangled with the 0072 changeover). |
| 2 | `toast-sync-fail-loud` | A | `d2120c8` | PASS | SFTP-transport-death → `/health` `toast_sync:failing` + Cliq alert; genuine date-not-found stays silent (no alert fatigue). Race-clean. |
| 3 | `toast-ingest-resurrection` | A | `4bbb7dc` | PASS | Key-ship mechanism (bind-mount, `.env.prod` pattern) + sync resurrection + day-by-day archive gap (10 aged-out / 28 recoverable) + attended-steps note. **NO-KEY safety check clean.** |
| 4 | `client-guard-coverage` | B | `5d96274` | PASS | B-149 + B-10 guard tests (red-first-proven); B-154 `[e2e.seams]` sync-rxdb rider (confines Cards 5-8). Full suite 822/7 (baseline only). |
| 5 | `cdc-single-fire` | B | `3359dee` | PASS | B-157 — one `/saveResponse` = one row write (trigger 2→1). `/ops` path correctly left unchanged (CheckLWW ordering). |
| 6 | `sync-dev-one-command` | B | `3b90709` | PASS | `task sync:dev` = data plane + dev server; B-164 :5433 refusal preserved; B-170 bare-npx hardened. |
| 7 | `sync-doc-honesty` | B | `29793e9` | PASS | Retired **5** stale row-visibility-rls gates (0 live remaining); proxy.go fictions corrected; repo-hygiene case 3 broadened slug→fact (red-first); sw.js regen (precache 31). B-167's 6 items re-homed. |
| 8 | `app-slug-association` | B | `809e15d` | PASS | B-160 — `checklist_templates.app_id` FK (migration 0076); per-row app_slug resolution; **0 hardcoded constants**. G6 confirmed the relay is the only writer (FDW views project no app_slug). |
| 9 | `period-summary-visibility` | C | `4c387cb` | PASS | B-139 — `slog.Info` visibility on `/period-summary` + `/menu-cogs` success (from/to/ready/counts). Split (b) of `pipeline-fail-loud`. |
| 10 | `counterparty-notice-prep` | C | `4c8f431` | PASS | Combined notice drafted (see DO-FIRST #1); both contract docs re-audited (accurate); B-137 lesson captured. `counterparty-combined-notice` stays PLANNED (closes on SEND). |
| 11 | `deploy-hygiene-honesty` | C | `8dcf506` | PASS | B-135 — version.json generators byte-identical (Dockerfile printf `\n`); B-17 false `--name-only` claim corrected. Precache 31. |

**Roadmap flips:** all 11 cards DONE; `pipeline-fail-loud` DONE (both split halves landed).

---

## Gate evidence — on the final merged tree (`overnight-20260901` @ `809e15d`)

- **Per-card:** every card was G6-reviewed (fresh context, diff + red-first, never the implementer's reasoning) and control-loop re-gated on the merged tree at its merge (G1 + footprint package(s); Playwright cards' subsets/full suite as noted). Per-merge gate evidence is in `reference/conflicts-20260901.md` (Merges 1–11, one entry per merge).
- **Authoritative full-tree gate (on `809e15d`, closeout):**
  - **G1:** `go build ./...` exit 0; `go vet ./...` exit 0.
  - **G2-Go:** `go test -p 1 -count=1 ./...`, `DB_TEST_URL=hq_test_go`, `HQ_SYNC_SUBSTRATE_OPTIONAL`+`HQ_SYNC_GATE_CHILD` **unset** — **every test-bearing package `ok`** (alerts, auth, inventory, onboarding, photos, purchasing, receipt, recipes, toast, workflow) **EXCEPT `internal/sync`** whose only failure is `TestJWTBridgeRLS` = the B-178 relay-contamination env red (below).
  - **G2-Playwright:** `npx bddgen` then `npx playwright test --retries=0` — **exactly ONE summary block: 823 passed / 8 failed / 6 skipped (25.3m)**. Of the 8 failures, **4 are the documented deterministic baseline** (B-174 ×3 `sw-api-cache-partition` B1-XT-01/-02/-05; B-176 ×1 `workflows.spec.js` DBL-05) and **4 are load-flakes proven GREEN in an isolated re-run** (all 4 passed, exit 0): `inventory.spec.js:3692` FR-11, `sync.spec.js:2976` SYNC-FC-01, `sync.spec.js:3136` SYNC-RF-02, `workflows.spec.js:2824` RUN-10 (cross-device WS-propagation + seed-timing races under cumulative load). **No real new red.** Logs: `scratchpad/closeout-{go,pw}.log`, isolation re-run `scratchpad/iso-recheck.log`.
- **G4 (sw.js):** reconciled at Card 7's merge — `node build-sw.js` on the merged tree is **idempotent (zero delta), precache 31**. Card 8 touched no precached asset, so no further regen needed.
- **RF:** every code-changing card showed a red before its fix, re-verified by G6. Docs/infra cards recorded `n/a — no code change` with reason (Card 10; parts of Cards 3, 7).
- **G4 discipline greps:** **N/A-VACUOUS** — neither `internal/journal` nor `internal/workorder` exists in this repo (B-14). (Not `clean`, not `PASS`.)

### 🛑 One environmental red, all night: `internal/sync/TestJWTBridgeRLS` (B-178)
The operator's **Spike C relay** (`.night-crew/qa/spike-supabase/.persistent-dev/spikec-relay`, pid 31802, connected to the live `:5433` substrate over Tailscale) leaks 12–13 `spikec-*` rows into the sync RLS fixture, so the CONTROL "service_role sees exactly the fixture rows" assertion fails. **Reproduced on base, zero-concurrency; no card touches the sync fixture.** NOT cleanable from an overnight run (rows trace to `:5433` — hard-forbidden — and stopping the relay would not remove already-written rows). Treated as a known-environmental red (same class as the armed Playwright baseline). Filed **B-178** — the durable fix is to isolate the RLS fixture from the live substrate (fold into `gate-rls-fixture-ownership`) and, attended, stop the relay + clean its `spikec-*` rows. **Every other Go package is green.** (Notably, Card 8's G6 found the leaked spikec rows carry a *correctly-resolved* `app_slug`, which incidentally re-proves Card 8's resolver.)

### Playwright baseline correction (carry forward)
The slate's NAMED armed reds (B-27 `inventory:883`, LST-17 `sync:446`, B-162 `receipt-carousel:123`) **all PASSED** this run — they are flaky-named. The real **deterministic** Playwright baseline on this tree is **B-174** (`sw-api-cache-partition` B1-XT-01/-02/-05) **+ B-176** (`workflows.spec.js` DBL-05). Both are documented pre-existing reds on clean `dev`. Update the armed-reds record.

---

## Findings filed this run (for triage)

- **B-177** — `parseEventDate`'s PARSEABLE path stamps `event_date` from the timestamp's own zone (UTC), not the business zone; a parseable boundary receipt lands in the wrong COGS week. Pre-existing, entangled with the 0072 changeover; **waived from Card 1** (out of B-28's defined scope). Scope alongside 0072.
- **B-178** — the sync RLS fixture is not isolated from the live Spike C relay's writes (above).
- **B-179** — `node_modules` `workbox-build@7.3.0` drifts from the lockfile's pinned `7.4.1`; a regen can re-emit the workbox runtime chunk under a different name. Idempotent on this box (committed sw.js reproduces), but a fresh `npm ci` clone would diverge. Decide the authoritative workbox version, then make node_modules + lockfile + committed sw.js agree.

**Also for triage (nice-to-haves surfaced by G6, no code impact):** Card 7's merge-intent prose overstates B-18(b) (says logging was added; it was correctly deferred as a comment); Card 8 flagged `internal/sync` has no TestMain (a review-env footnote) and `access_test.go`'s `TRUNCATE ops` coupling (pre-existing). B-18(b)/(c) log-field code fixes were deferred to a future code-footprint card (out of Card 7's comment-only footprint).

---

## Concurrent 3-track dispatch — the verdict (criterion 1 for recommending it again)

**It held up. Recommend it again.** All 11 cards landed clean, 0 parked, 0 merge conflicts (every merge was `ort`-clean — the disjoint-footprint planning held). Tracks A (3 cards) and C (3 cards) finished well ahead; **Track B (5 cards, serial-within-track) was the critical path**, exactly as the slate modeled. Wall-clock ~3h15m vs the slate's concurrent mid-estimate of ~5h45m — Opus implementer speed (cards landed in ~10–46m each vs 45–140m estimates) beat the estimates handily.

**Two coordination findings the concurrent model surfaced (both handled, both worth carrying):**
1. **`internal/sync`'s RLS/bridge tests use fixed, shared DB names on :5434** (`hq_test_b1`, `hq_test_b2_fdw`, `hq_test_go`, `hq_test_e2e`) — NOT isolated by per-card `DB_TEST_URL`. So two concurrent `go test ./...` legs collide there (this is what first *looked* like a Card-1 sync regression). **Resolution adopted mid-run:** implementers gate their OWN footprint package (isolated DB) + G1; the orchestrator runs the authoritative full-suite Go re-gate serially. Kept for the whole run; worked. (This is a milder cousin of B-178.)
2. **Playwright full-suite legs must not overlap wall-clock** (sync.spec.js load-sensitivity) — the slate anticipated this. In practice only Cards 4 and 7 ran heavy Playwright, and they never overlapped (Card 4 early, Card 7 late; Card 11's suite deferred to closeout as it changed no frontend asset). No flake attributable to overlap.

**Isolation model that worked:** per-leg unique Go DB (`hq_test_go_cNN`, created fresh, TestMain auto-migrates) and unique `TEST_DB_NAME`/`TEST_PORT` for Playwright; neutral worktree dir names (`c01`…`c08`) to dodge B-87 (Playwright's absolute-path filter). node_modules symlinked from the main checkout into Playwright/G4 worktrees.

---

## Context checkpoint
🔄 **Clear now (recommended).** `/nc-morning-triage` starts cold and reviews the run branch on its
merits; everything it needs is in this HANDOFF, DECISIONS-NEEDED.md, and `reference/conflicts-20260901.md`.
Nothing in the run session is load-bearing. Next command: **`/nc-morning-triage`**.

## Closeout housekeeping
- **Poller check at closeout:** `internal/sync`'s dispatch was subagent-per-card (Agent tool), never a
  Temporal queue — Temporal was down the whole run, so no poller could intercept work (holds by
  construction: a down server has zero pollers). The Spike C relay (pid 31802) is NOT a night-crew
  card-work poller — it is the operator's spike relay to `:5433`, left running and untouched (it is the
  source of the B-178 sync red; attended cleanup recommended).
- **Worktrees:** the 8 card worktrees under `/Users/jamal/gsd-workspaces/nc-20260901/` were removed at
  closeout; their branches (`wo-*`) remain until triage merges the run branch.
- This session is disposable — safe to clear after reading.
