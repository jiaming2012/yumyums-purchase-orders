# Merge intent — card 1 · `sync-coordinates-provisioning` (run 20260907)

Branch: `card/sync-coordinates-provisioning`, off `dev` @ `db4bcd6`.
Slate: slate-20260907 (1 card). Goal ledger:
`.night-crew/knowledge/spikes/activity-b-offline-first-replica/sync-coordinates-provisioning.md`
(six build-facts, BINDING).

## What this card does

Wires the missing caller: `marketing/scan-page.js` gains the tree's ONLY
`SYNC_KEY` (`hq_marketing_sync_v1`) writer — on page init with a session,
`POST /api/v1/sync/token` → 200 → write `{restUrl: '/sync/rest', bearer:
envelope.token, deviceId: envelope.sub}` → call the already-exported
`startSync`. On 401/503/network failure it writes NOTHING and the shipped
`if (sc) startSync(sc)` guard keeps today's no-sync behavior. `startSync`
additionally starts the push replica (`startScanAttemptsReplica`, first caller
in the tree — spike fact 4's decided scope) with `deviceId` taken ONLY from
the stored mint `sub`; if the stored coordinates carry no `deviceId`, the push
does NOT start (a guessed identity is a stranded burn + poisoned queue —
spike fact 3, measured both ways). The B-439 fix clears the campaign policy
source's `lastError` on successful-pull evidence: `makePullHandler` gains an
optional `onSuccess` callback fired on every HTTP-200 pull (the same edge
`clock.captures` witnesses — fires in BOTH recovery shapes including
recovery-EMPTY, never on a failed cycle), `startReplica` exposes it as
`onPullSuccess(fn)` on the replication handle, and
`createCampaignPolicySource.attach` subscribes to it. B-438 discharges as a
consequence: `attach()` already sits inside `startSync`; it now actually runs.

## Claims, stated only in the window they hold

- Provisioning arms sync **only when** (a) the page loads with a live
  `hq_session` and (b) the deploy carries `HQ_SYNC_JWT_SECRET`. A secret-less
  deploy answers 503 `sync_bridge_not_configured` → nothing is written, no
  sync starts, `attached()` stays false, and a genuinely-unknown override
  files `policy_unresolved=false` — that is B-436-adjacent territory, outside
  this card's four clauses, deliberately untouched.
- B-436 itself (the no-source device; `!CAMPAIGN_POLICY → false` in
  submit-flow) is untouched, per the slate's PARK line.
- The bearer written to `SYNC_KEY` is inert at the door (spike fact 1 — the
  proxy substitutes a per-request session mint); it exists to satisfy
  `startSync`'s truthiness check and keep the direct-substrate path open.
  There is no client refresh machinery, because there is nothing to refresh.

## Shared files touched (outside `marketing/` + `marketing/sync/`)

| File | Why (one line) |
|---|---|
| `tests/marketing.spec.js` | additive: one new `describe` block for the card's four done_when clauses; zero existing tests modified |
| `sw.js` | committed artifact — regenerated after the client-file commits (content hashes moved; precache count stays 43, no file added/removed) |
| `.night-crew/knowledge/roadmap.md` | the card flips its own line PLANNED → LANDED |
| `.night-crew/knowledge/BACKLOG.md` | B-438 / B-439 dispositions (status field carries the status, narrative in the body — decision 177) |
| `.night-crew/knowledge/spikes/…/sync-coordinates-provisioning.md` | the owed `validated:` line appended to Comebacks |
| `.night-crew/runs/2026-09-07-autonomous/` | merge-intent + gate logs |

`backend/` is expected untouched and IS untouched — no Go suite leg owed
(G1 still runs). `supabase/migrations/` untouched. `night-crew.toml`
untouched. `package.json` / version constants untouched (frontend-only
behavior change rides the run's normal versioning at release, not this card).
New files: `marketing/sync/harness/recovery-clear-run.sh` +
`recovery-clear-harness.mjs` (the owed spike-04 re-execution, refusal-run.sh
precedent) — under `marketing/sync/harness/`, which the sw precache globs
deliberately exclude (single-level `marketing/sync/*.js`).

## What must survive any merge

- The shipped `if (sc) startSync(sc)` guard and the write-nothing degradation
  on 401/503/network failure (the B-436-adjacent degenerate case stays).
- `createCampaignPolicySource`'s B-432 fail-closed predicate — UNTOUCHED
  (a KNOWN code whose campaign is unresolved refuses). The B-439 clear keys
  ONLY on successful-pull evidence, never on `error$`/`active$`/
  `remoteEvents$` (all disqualified by spike-04 measurement).
- The two adjacent, unawaited statements in `startSync`
  (`startCampaignsReplica` → `campaignPolicy.attach`) — still adjacent, still
  unawaited (error$ does not replay).
- Push `deviceId` = the mint envelope's `sub`, never any device-local
  identity; no push starts without it.
- `marketing/submit-flow.js` / `submit-machine.js`: zero new (state,event)
  pairs; `policyUnresolvedFor`'s three arms unchanged. (Neither file is
  touched.)
- GAP-1's two belts and the F-2 divert path in `push-replication.js`.
  (File not touched.)
- Existing `tests/marketing.spec.js` tests byte-identical: the Playwright
  webServer carries no `HQ_SYNC_JWT_SECRET`, so every existing test's page
  load takes the 503 → write-nothing path — today's behavior, now exercised
  on every load.

## Safe to drop in a conflict

- Gate logs under `.night-crew/runs/2026-09-07-autonomous/` (re-runnable).
- Nothing else — the code diff is one logical unit.

## The build-fact-6 test-stack call (decided by this card)

**Route interception at the network layer**, not a CI substrate. The clause-1
e2e provisions through the shipped path with `page.route` serving the two
transports: `POST /api/v1/sync/token` answers a mint envelope (the Playwright
webServer deliberately has no `HQ_SYNC_JWT_SECRET`, so the REAL endpoint
answers 503 there — the shipped degradation, which every existing test now
exercises), and `/sync/rest/*` answers PostgREST-shaped rows. This is
decision-174-consistent: it fails/serves the TRANSPORT, not the seam — the
shipped writer, `startSync`, both replica directions, the pull URL
composition and the policy source all run for real; the test never touches
`SYNC_KEY`, never calls `setCampaignPolicy`, never injects a policy. The
full-stack half (real server + real mint + real PostgREST through the real
door) lives in the harness the way `refusal-run.sh` does: spike 01/02/03
already run the shipped modules through the shipped door, and this card adds
`marketing/sync/harness/recovery-clear-run.sh` (spike-04 re-executed against
the shipped clear, live substrate, both recovery shapes) — graded on its exit
code.

## Red-first

**Playwright (pre-change tree, commit `768a173` + tests only):**
`npx playwright test tests/marketing.spec.js -g "sync-coordinates-provisioning" --retries=0`
→ **EXIT=1, 6 failed / 6 run** —
log `.night-crew/runs/2026-09-07-autonomous/c1-red-provisioning.log`.
- Clause-1 structural red, exactly as the slate names it: the poll on
  `localStorage.getItem('hq_marketing_sync_v1')` receives `null` after 15 s —
  no code path writes `SYNC_KEY` (`c1-red-provisioning.log`, failure 1).
- Clause-2 red: `attached() && ready && size() > 0` never true (failure 2).
- B-438 reds: `lastError` never latches (attach never runs) in the unhealthy
  leg; `attached` stays false in the healthy control (failures 3–4).
- B-439 reds: both recovery-shape tests die at phase A (`attached` false —
  the pre-change tree cannot even reach the latch) (failures 5–6).

**Harness (pre-fix tree, same commit set):**
`bash marketing/sync/harness/recovery-clear-run.sh` → **EXIT=1** at phase C1:
`captures` advanced 1→3 (successful pulls happening) while
`unresolved()` stuck `true` after a with-docs recovery — the spike-04
measurement reproduced on the pre-fix shipped surface.
Log `.night-crew/runs/2026-09-07-autonomous/c1-red-recovery-clear.log`.

**Green, after the fix (commit `74f27f1`):**
- The same six tests: **EXIT=0, 6 passed** (scratchpad green log; re-run
  inside the standalone marketing leg below).
- `marketing/sync/harness/recovery-clear-run.sh` → **GREEN, EXIT=0** — the
  latch clears in BOTH recovery shapes (C1 with-docs, C2 recovery-EMPTY with
  zero rows delivered) and still latches during both error phases. Log:
  `c1-recovery-clear-green.log`.
- The UNMODIFIED spike-04 script, post-fix → **EXIT=1 at its own C1
  ghost-check**: "unresolved() self-cleared after recovery — then B-439 is
  already fixed and this spike is measuring a ghost"
  (`c1-spike04-rerun-postfix.log`). Note the script's outer wrapper prints
  its exit-1 prose ("no shipped signal survives…") unconditionally; the RED
  line above it names the real cause. The committed
  `recovery-clear-run.sh` is the both-shapes verdict.

## Gates (final tree)

| Gate | Result |
|---|---|
| G1 | `go build ./...` EXIT=0, `go vet ./...` EXIT=0 (from `backend/`; zero .go changes) — `c1-g1-build.log`, `c1-g1-vet.log` |
| G2 Go | **not owed** — `backend/` untouched (verified by diff-stat below) |
| G2 Playwright (marketing standalone) | `npx playwright test tests/marketing.spec.js --retries=0` → **EXIT=0, 39 passed** (33 pre-existing + 6 new), exactly one summary block — `c1-g2-marketing.log` |
| G2 Playwright (full) | `npx playwright test --retries=0` → **EXIT=1, 866 passed / 4 failed / 6 skipped (29.2m), exactly ONE summary block** — `c1-g2-full.log`. The 4 reds vs the four-red baseline: `sync.spec.js:2976 [SYNC-FC-01]`, `:3062 [SYNC-RF-01]`, `:3136 [SYNC-RF-02]` = baseline; **`B1-XT-01` PASSED this run**, and the fourth red is **OUTSIDE baseline**: `tests/sync-fill-view.spec.js:451 [FILL-04]` (poll for `window.HQSync.db` answered −1 for 60 s — db handle never constructed under full-suite load). Isolation evidence: the whole spec standalone on the same tree + coordinates → **EXIT=0, 9 passed** (`c1-fill04-isolation.log`). Same one-off shape as run 20260906-2 (B1-XT-01 passing + a single foreign spec red once → B-437 precedent). No seam from this card reaches it: the diff touches `marketing/*`, `tests/marketing.spec.js`, `sw.js` only. Reported, not hidden. |
| G3 | N/A — `openspec: absent` (decision 140) |
| G4 | `task sw` idempotent (second run: tree clean, sw.js unmodified), **43 precached** (count unchanged; no asset added/removed — new harness files sit under `marketing/sync/harness/`, which the single-level globs exclude by design), version parity `1.6.2` ≡ `1.6.2` ≡ `1.6.2` — `c1-g4-sw-idempotent.log` |
| RF | this section |

Environment stated: every leg ran with `HQ_SYNC_SUBSTRATE_OPTIONAL` and
`HQ_SYNC_GATE_CHILD` **unset** (explicitly `env -u` on the full-suite leg);
test DB on `:5434` (`yumyums-test-pg`, role `hqtest`) only.

**Deviation, stated:** the launch prompt's isolation value
`TEST_DB_NAME=hq_c1_impl_0907` fails `scripts/reset-e2e-db.js`'s name guard
(`/^hq_test(?:_[a-z0-9]+)*$/` — the guard that keeps a mistyped name from
DROPping a non-test database). Used `hq_test_c1impl0907` instead — unique to
this card's legs, guard-conforming; the guard was not widened.
