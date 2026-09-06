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

(filled with evidence as produced; exit codes recorded)

- [ ] Clause-1 structural red: the provisioning e2e on the pre-change tree —
  no code path writes `SYNC_KEY`, every scan resolves `unknownCode`.
- [ ] Clause-2 red: `attached()` never true on the pre-change tree.
- [ ] B-438 red: no shipped path lands `policy_unresolved=true`.
- [ ] B-439 red: the latch stays `unresolved()===true` through both recovery
  shapes (Playwright, and the harness against the pre-fix tree).
