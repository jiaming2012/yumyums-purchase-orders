# HANDOFF — run `overnight-20260808`

**1 of 1 cards merged. Zero parks. Zero open forks. Spike E GREEN — Activity 2 is now
5 of 5 spikes closed (A, B, C, D, E all GREEN), and the checkpoint-pull question the
Activity 3–5 build cards were waiting on is answered: catch-up after a dark window CAN be
trusted — conditional on the carrier being the trigger-stamped substrate watermark (see
§"What is now true" for the exact condition).** Run window: ~08:53–09:57 EDT 2026-08-07
(attended-daytime launch immediately post-triage of `20260807-2`, same T-40-precedent
pattern as run `20260807`; branch dated for the morning after). Serial dispatch as signed —
one card. Final tree: closeout commit on `overnight-20260808`.

## Per-card outcomes

| Card | Outcome | Merge | G6 verdict | Planned → actual |
|---|---|---|---|---|
| E `spike-e-reconnect-catchup` | ✅ MERGED — **verdict GREEN, exit 0; UPDATE case EXERCISED and RECOVERED** (5th of 5 Activity-2 spikes; answers B-161) | `0ac5a20` | PASS (6 findings, none refuting; see §"G6 findings") | 60–120m → **51m** implementation (~26m of it the full Playwright suite); ~64m wall incl. G6 + merge |

Merge clean (all-additions plus two owned-file edits; run branch had not moved since the card
branched, so the merged tree is byte-identical to the gated card tip `cdc91c6` except the
conflict log). Conflict log: `.night-crew/knowledge/reference/conflicts-20260808.md` §1 —
clean merge logged per §15ad.66.

## What is now true

- **Spike E GREEN — a dark-window client recovers everything on reconnect via checkpoint
  pull, including an in-place UPDATE to a row it already held.** All three dark-window
  changes (INSERT / in-place UPDATE / INSERT) written through the real path (`cmd/server`,
  real login, `POST /api/v1/workflow/saveResponse` → 204) arrived in 1 ms of a 20 000 ms
  bound. The reconnect was **observed to be a checkpoint pull, not a full re-read**: the
  first post-reconnect pull was handed the sever-time checkpoint verbatim, captured through
  the RxDB supabase plugin's own `queryBuilder`, and the script refuses a null-checkpoint
  first pull by construction.
- **Why the UPDATE recovered — and the condition the green carries.** The pull checkpoints on
  the **substrate's** `_modified`, trigger-stamped at the projection boundary
  (`sql/hq-bridge-fixture.sql:79-90`) with a strict `_modified > m OR (_modified = m AND
  id > id)` — a `gt` with an id tie-breaker, minted independently of any HQ business
  timestamp. **The feared `gte.<iso>` weak spot does not exist on this path.**
  🛑 **Carried risk for the build cards:** the green is conditional on the carrier
  re-projecting on every HQ change. It holds because the carrier is a row-level
  `AFTER INSERT OR UPDATE` trigger. **A future relay that polls HQ on a business watermark
  instead of NOTIFY reintroduces the miss exactly** — see the next finding.
- **`submitted_at` semantics, measured (the B-161 question):**
  `checklist_submissions.submitted_at` **never advances after INSERT** — column default
  `now()`, zero user triggers, and the only UPDATEs HQ issues (`repository.go:1186` approve,
  `:1232` reject) leave it alone. It is a creation timestamp wearing a watermark's name —
  the same unreliable-watermark shape that disqualified `answered_at` for the polling-relay
  candidate. `submission_responses.answered_at` DOES advance on every save
  (`repository.go:829`, measured end-to-end this run). Neither is what the pull checkpoints
  on.
- **Meaning for Activity 3–5:** build cards can be designed against **trusted checkpoint
  catch-up with no explicit resync step**, PROVIDED the relay stays trigger/NOTIFY-driven
  (spike C's mechanism). If any build card contemplates a polling relay on a business
  watermark, an explicit resync step comes back as a requirement. State this in the next
  slate's card designs.
- **RF (red-first) held:** realtime-only recovery (`--no-pull`) exited **1** — all three
  dark-window changes MISSED with the liveness control ARRIVED (so the red is attributable
  to the absent pull, not a dead client) — committed before the green work.
  Log: `card-e-rf-red.log`.
- **Substrate discipline held:** A–D artifacts consumed strictly read-only (only `spike-e-*`
  siblings added — G6-verified zero edits); teardown snapshot/restore **byte-verified on both
  the red and green paths** (B-148's standard); scratch Postgres rode ephemeral ports
  (50459 red / 50299 green) with runtime refusal of 5432/5433/5434; `--fresh` /
  `--fresh-substrate` never passed — the flag does not exist on spike E's script (B-159
  honoured by absence).

## Gate evidence on the final tree

The merged tree is **byte-identical to the gated card tip `cdc91c6`** (`git diff cdc91c6
HEAD` empty but for the conflict log, which is outside every gate's subject set), so the
card-tip gate runs are the final tree's evidence; G4 was additionally re-run on the merge
commit itself.

| Gate | Result | Log |
|---|---|---|
| G1 | `go build` + `go vet` from `backend/`, both exit 0 | `gate-g1-card-e.log` |
| G2 (Go) | exit 0; **9 packages, 456 tests** (top of the 439–456 band); `internal/workflow` = **35**; `internal/sync` ok with its 59-subtest self-assertion; `DB_TEST_URL` → localhost:**5434**/`hq_test_go` role `hqtest`; `HQ_SYNC_SUBSTRATE_OPTIONAL` + `HQ_SYNC_GATE_CHILD` attested UNSET in the log | `gate-g2-go-card-e.log` |
| G2 (Playwright) | `npx bddgen` then full suite `--retries=0`: **exactly one summary block — 789 passed / 3 failed / 6 skipped, 26.1m**. All 3 reds carry armed signatures (§ below). B-156/FR-11 **passed** this run (recorded as evidence, not retirement — the standing rule) | `gate-g2-playwright-card-e.log` |
| G4 | Idempotent (twice at card tip + once on merge commit `0ac5a20`, all exit 0, tree clean); **precache 31 (2169.0 KB) unchanged**; parity 1.4.0 three ways, no bump | `gate-g4-card-e.log` |
| RF | red leg exit 1 committed before green; merge-intent `## Red-first` section carries it; G6 re-verified commit order | `card-e-rf-red.log` |
| G6 | **PASS** — adversarial, fresh context, inputs = slate entry + diff + evidence only | (report summarised below) |

**The 3 Playwright reds vs the armed baseline:** `tests/inventory.spec.js:883` = B-27's
recorded signature verbatim; `tests/sync.spec.js:446` = `[LST-17]`, the named armed red —
NOTE for triage: it **failed again tonight** after passing twice on `20260807-2`, which
settles last run's follow-up 2 in favour of "still armed" (flipping = still flaky; retire
question moot); `tests/receipt-carousel.spec.js:123` = B-32 family **shape** (30s
`networkidle`, whole-suite position, spec untouched — the card's diff carries zero
frontend/spec/Go files, so it provably is not the card's regression) but it is **not a named
member of any armed entry** — see G6 finding 1.

**G4 discipline greps: N/A-VACUOUS — neither package exists in this repo (B-14).**

## 🛑 Standing-rule-1 near-miss — disclosed, nothing touched

While provisioning the Go gate the implementer ran a **bare `task backend:db-test`**, whose
defaults point at **:5433**. The task's own guard refused (`refusing: localhost:5433 also
serves the dev DB 'yumyums'`, exit 1); the only contact was the guard's read-only
`SELECT 1 FROM pg_database`. **Nothing was created, modified, or dropped**, and the leg was
immediately re-run with explicit :5434 coordinates. Standing rule 1 forbids any :5433
contact including read-only, so this is reported as a breach, not narrated as a skip. The
run therefore CANNOT claim "nothing resolved a name against :5433 all night" — it can claim
one guarded, read-only, refused contact, zero writes. **Lesson for every future leg: never
invoke a `backend:*` task bare — its defaults are the production cluster; `task
test:targets` is the read-only way to get coordinates.** Triage may want a slate-flag or a
task-level guard hardening filed.

## G6 findings (PASS overall; severity-ordered; none refute the verdict)

1. **Unattributed Playwright red** — `tests/receipt-carousel.spec.js:123`, B-32 family shape,
   on no armed list, noted nowhere by the card. Per the B-156/FR-11 precedent: triage should
   file it as a candidate B-32 family member (or new B-nnn) rather than absorb it silently.
2. **Latent exit-1/exit-2 conflation, client leg** — an uncaught exception outside the
   step-1/step-3 try blocks in `rxdb/spike-e-reconnect.js` (e.g. a transient PostgREST
   hiccup mid-poll) exits Node 1, which the shell maps to VERDICT RED. Did not fire in the
   recorded runs. Spike-script hardening, next-touch.
3. **Latent exit-1 conflation, shell** — unguarded `$(srcpsql …)` assignments under `set -e`
   (`spike-e-reconnect.sh:411`, `:463-464`, `:552-554`) would report an infra failure as
   RED, not exit-2. Same class, next-touch.
4. **Vacuity-detection maps to RED (1), not 2** — `spike-e-reconnect.sh:564`: CLIENT_RC=0
   with B_ROWS≠1 calls `red`, which would falsely instruct Activity 3 to add a resync step;
   the honest code for "green was vacuous, no verdict" is 2. Conservative direction,
   semantically wrong per the card's own contract. Next-touch.
5. **Evidence provenance cites unreachable SHAs** — gate-log HEAD attestations name
   pre-rewrite twins (`ef801cb`/`d65273a`/`4ef8fb7`) from the trailer-normalising
   `filter-branch`; G6 verified each content-identical (empty diffs) to the reachable
   commits `ff778b9`/`aada295`/`8c0b942`. A fresh clone cannot resolve the cited SHAs; the
   RF red log carries no HEAD line at all. Record-keeping nit; the trees are proven
   identical.
6. **Minor:** Playwright wall clock 26.1m sits marginally at/over the 21–26m band (one run,
   not a re-arm); and the slate's "UPDATED without `submitted_at` advancing" question was
   answered by construction (the checkpoint rides the always-advancing substrate
   `_modified`) rather than by a watermark-stationary update — disclosed in the
   merge-intent's own finding 3, and the carried-risk paragraph above is the honest form of
   the claim.

## Follow-ups the run leaves (none blocking)

1. **File the receipt-carousel red** (G6 finding 1) under the flake protocol at triage.
2. **LST-17 stays armed** — tonight's fail after two passes is the evidence; record in the
   B-147/B-148-adjacent trail (rulings T-41 anticipated exactly this shape).
3. **Spike-E script hardening** (G6 findings 2–4), one small change each, fresh branch off
   `dev` whenever the script is next touched — the recorded verdict does not depend on them.
4. **Near-miss hardening** (§ above): consider a repo-level guard or slate flag for bare
   `backend:*` task invocation.
5. **`TEST_DB_NAME` deviation, recorded:** the launch prompt's literal `hqtest_e_reconnect`
   is refused by `scripts/reset-e2e-db.js:115`'s `TEST_DB_NAME_PATTERN`
   (`/^hq_test(?:_[a-z0-9]+)*$/`); the leg used the conforming `hq_test_e_reconnect`. Future
   launch prompts should mint isolation names in the pattern's shape.

## Attended work still waiting (unchanged by tonight; standing rule 1 kept the run out of all of it)

- **A3 re-gate** (`gate-rls-fixture-ownership`) — attended by ruling (decision 155); branch
  `card/a3-rls-fixture-own` + worktree preserved and untouched tonight (verified at launch:
  the only stranded work anywhere, excluded by name).
- **Decision-156 Mercury backfill** (attended).
- **B-146 SFTP key fix** (attended).
- **Decision-159 `archive_mode` enablement residual** (attended).
- **Decision-158 sales-processor message** (attended).
- **B-145 recovery Phase 1** (attended queue).

## Run hygiene

- One suite in flight all night (serial, single card; G6 ran no suites). All test
  coordinates :5434; the single :5433 event is the disclosed near-miss above (guard refusal,
  read-only, zero writes). Isolation: `TEST_PORT=4823` / `TEST_DB_NAME=hq_test_e_reconnect` /
  `HQ_RLS_TEST_DB=hqtest_e_reconnect_rls` (implementer), G6 ran read-only inspection only.
- Run git ops from the dedicated run worktree `../hq-worktrees/run-20260808` (decision 160).
- Stale-worker check at closeout: **queues `night-crew` and `night-crew-env` clear, no
  pollers**, checked 2026-08-07T09:57:04-04:00 (poller-TTL caveat noted; nothing was
  dispatched via Temporal tonight).
- Card worktree `e-reconnect-catchup` removed after merge; branch
  `card/spike-e-reconnect-catchup` retained.

## Next actions

1. `/nc-morning-triage` on this branch — audit the conflict log, rule on follow-ups 1–2,
   note 3–5, merge to `dev`.
2. Next slate: cut the Activity 3–5 build cards with spike E's verdict in hand — designed
   against **trusted checkpoint catch-up, no explicit resync step, conditional on a
   trigger/NOTIFY-driven relay** (the condition is load-bearing; put it in the cards' text).
