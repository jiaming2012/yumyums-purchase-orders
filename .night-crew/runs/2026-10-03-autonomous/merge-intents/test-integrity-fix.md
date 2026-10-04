# Merge intent — card `test-integrity-fix` (Card 1, roadmap I1, run `20261003`)

Branch `card/i1-test-integrity-fix`, cut from `overnight-20261003` at `ec2830c`. Track A, first.
Three false test gates made honest — **B-462** (the campaigns harness asserts a hand-copied
`failClosed`), **B-465** (`[SP-03b]` never consults the predicate; the codes-replica schema
omits `campaign_id` from `required`), **B-466** (`TestNothingInThisPackageSends` scans two
filenames). Carries the three decision-191 amendment riders (ledger T-62).

Contract: `.night-crew/knowledge/reference/slate-20261003.md` §"Card 1". Goal ledger:
`.night-crew/knowledge/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/test-integrity-fix.md`
(2/2 passed; build-facts binding).

**This card changes what the TESTS measure, not what the phone does.** The `requires_online =
true` refusal, the B-432 fail-closed predicate and decisions 166/199 are untouched.

---

## Shared files touched, and why each

| File | In footprint? | Why |
|---|---|---|
| `marketing/submit-flow.js` | yes (exports only) | **B-462**: `failClosed`, `policyFor` (and the `namesNoCampaign` helper both read) become module-level `export`s. 🛑 **A build-fact the slate did not have:** they are not "module-internal", they are *closures inside `async function boot()`* — `export` cannot be added in place. They are HOISTED to module scope with their bodies byte-for-byte unchanged; `policyFor` gains the policy source as an explicit first parameter (it used to close over `CAMPAIGN_POLICY`), and its two call sites pass `CAMPAIGN_POLICY`. No predicate body changes; the spike's one-line `failClosed` inversion recipe still applies to the hoisted function verbatim. Card 2 (`scan-time-verify`) edits this file after this card — serial, same track. |
| `marketing/sync/harness/campaigns-harness.mjs` | yes | **B-462**: leg 3 imports the shipped `failClosed` / `policyFor`, the mirror copy is deleted, and the harness asserts the import (the functions came from `submit-flow.js` and the harness source declares no local copy). |
| `marketing/sync/replicas.js` | yes | **B-465**: `MARKETING_REPLICA_SCHEMA` → `version: 1`, `campaign_id` in `required`, `MARKETING_MIGRATION_STRATEGIES = {1: d => d}` carried by `marketingCollectionSpec()` for BOTH `codes` and `offers`. |
| `marketing/sync/harness/push-harness.mjs` | **no — outside the slate's footprint** | A dependent keyed on the version: it builds the codes collection from the bare `MARKETING_REPLICA_SCHEMA` (`{ schema }`, no strategies). At version 1 that is refused by RxDB (a strategy is missing). It takes the collection entry from `marketingCollectionSpec()` instead. No leg, assertion or verdict changes. |
| `marketing/sync/harness/replica-schema-harness.mjs` (**new**) | **no — new file under `marketing/sync/harness/`** | The node half of the `[SP-03b]` replacement: spike 02's recipe as a committed gate — a v0 store holding rows in `codes` and `offers` reopens through the SHIPPED `marketingCollectionSpec()` at v1 with its rows; a row without `campaign_id` is refused with `VD2`. Node + the vendored RxDB, memory storage + ajv; no substrate, no database. Not precached (the harness directory never is). |
| `tests/marketing.spec.js` | yes | `[SP-03b]` retired; replaced by tests that run the schema gate above and that reopen a v0 store at v1 in the page. Card 2 appends its own `describe` later — union at merge. |
| `backend/internal/marketing/subscribers_test.go` | yes | **B-466**: `TestNothingInThisPackageSends` walks every non-test `.go` file in `internal/marketing` and `internal/marketing/sources`, with an explicit allowlist (`projection.go`, `mirror.go`), each with its reason in the test. One function. |
| `sw.js` | yes | Regenerated after the precached edits are committed. `replicas.js` + `submit-flow.js` change revision; **no file added or removed — count stays 51.** The orchestrator regenerates at the merged HEAD; never hand-merge. |
| `.night-crew/knowledge/BACKLOG.md`, `roadmap.md` | yes (bookkeeping) | B-462 / B-465 / B-466 → `landed → test-integrity-fix`; roadmap card `PLANNED` → `LANDED`. |
| `.night-crew/runs/2026-10-03-autonomous/logs/test-integrity-fix/` | evidence | Red-first and gate logs. |

`night-crew.toml`: **not touched.** No new key, no new token.

## Other mirrors under `marketing/sync/harness/*` (the B-462 sweep — READ only, not fixed here)

* 🛑 **`refusal-harness.mjs:183` — a second hand-copied `policyFor`, and it is STALE.** Its
  no-source arm and its catch arm both `return false` — the pre-B-436 shape. The shipped
  `policyFor` has failed CLOSED there since decision 191 (run `20261002`). The harness comment
  says "verbatim in behaviour"; it is not. Whether any leg of that harness reaches the stale arm
  was not measured by this card. Not fixed here (the slate scopes the fix to the campaigns
  harness); it is the same defect class as B-462 and wants its own BACKLOG row at triage.
* `refusal-harness.mjs:170-176` (`mapMirror` / `preCardPolicy`) — a deliberate reconstruction of
  the PRE-card policy for that harness's red mode, not a mirror of shipped code. Not a defect.
* `f2-harness.mjs:102` — composes "the F2 shape `submit-flow.js:240` composes for an unknown
  code" by hand (a literal attempt row, with a stale line anchor). A fixture shape, not a
  predicate; a mirror in the weak sense. Listed, not fixed.
* `harness.mjs`, `clock-harness.mjs`, `push-harness.mjs`, `recovery-clear-harness.mjs` — no
  reimplementation of shipped predicates found (`naiveHandler` in `harness.mjs` and the "spike
  §6 handler" in `push-harness.mjs` are deliberate RED-mode strawmen).

---

## What must survive any merge

1. 🛑 **The three-part schema shape, for `codes` AND `offers`:** `MARKETING_REPLICA_SCHEMA`
   `version: 1` + `MARKETING_MIGRATION_STRATEGIES` passed by `marketingCollectionSpec()` on both
   entries + `RxDBMigrationSchemaPlugin` registered by every builder. Drop the strategy and
   `addCollections` rejects; the crew's phone shows "Scanner failed to start". A merge that keeps
   `campaign_id` in `required` but reverts the version is the `DB6` brick spike 02 measured.
2. 🛑 **The predicate bodies in `submit-flow.js`** — `failClosed`, `policyFor`,
   `namesNoCampaign` — byte-identical to the pre-card tree apart from their position in the file
   and `policyFor`'s explicit first parameter. Card 2 must not change them either.
3. **The harness's import and its import assertion.** A merge that restores a local
   `failClosed` / `policyFor` in `campaigns-harness.mjs` re-opens B-462; the harness now reds on
   exactly that.
4. **The egress allowlist is two files with reasons.** Adding a third is a deliberate act in
   the test, not a merge resolution.
5. Card 2's edits to `submit-flow.js` / `tests/marketing.spec.js` are additive to this card's.

## What is safe to drop

* The comments. The in-page reopen test (`[TI-02]`) is beyond the slate's stated replacement
  and can be dropped without breaking the done_when — at the cost of the only test that runs the
  migration on the storage the phone actually uses.
* Evidence logs under `logs/test-integrity-fix/` (they are evidence, not behaviour).

## Stubs and fixtures, per done_when clause

* **Inverted `failClosed` → `campaigns-run.sh` exits 1:** no stub. The harness imports the
  shipped module; in node it supplies a `window` whose `MarketingScan.ready` never settles so
  `submit-flow.js`'s page boot stays parked — the two predicates are pure and are the only thing
  the harness uses from it. That shim is the one thing standing in for the browser; it stands in
  for the PAGE, not for the predicate.
* **`[SP-03]` reds, no spec green by never consulting the predicate:** no stub. Real page, real
  `submit-flow.js`.
* **`spikeEgress()` in `campaigns.go` → `TestNothingInThisPackageSends` reds:** no stub; the
  test reads the real source files.
* **v1 store opens with v0 rows intact:** the v0 schema is a frozen literal in the test/harness
  (it must be — the shipped schema is v1 now); everything on the v1 side is the shipped
  `marketingCollectionSpec()`. Memory storage in node; the Dexie path is stated separately in the
  Red-first section below, as observed.

## Red-first

All logs: `.night-crew/runs/2026-10-03-autonomous/logs/test-integrity-fix/`. Every mutation
ran in a throwaway worktree of this card's own (`i1-prechange-mut` at `ec2830c`,
`i1-postchange-mut` at `982bed3`) or was reverted by `git checkout --` before any commit; the
card branch never carried one. The mutation is the spike's: one line, `failClosed` returns
`namesNoCampaign(campaignId)` instead of its negation; and `spikeEgress()` (an `http.Post`)
appended to `campaigns.go`.

| done_when clause | Before this card | After this card |
|---|---|---|
| inverted `failClosed` → `campaigns-run.sh` exits 1 | `red-harness-prechange-inverted.log` — `VERDICT: GREEN`, **`EXIT=0`** (the false gate) | `red-harness-postchange-inverted.log` — 3 disagreements in leg 3, **`EXIT=1`**; unmutated `green-harness-postchange.log` **`EXIT=0`** with `import asserted` printed |
| inverted `failClosed` → `[SP-03]` reds, no spec green by never consulting the predicate | `red-pw-prechange-inverted.log` / `.specs.txt` — whole `tests/marketing.spec.js`: **1 failed (`[SP-03]`), 52 passed, `EXIT=1`** — `[SP-03b]` among the 52 (the false gate) | `red-pw-postchange-inverted.log` / `.specs.txt` — **1 failed (`[SP-03]`), 53 passed, `EXIT=1`**; `[SP-03b]` no longer exists. Unmutated: `green-pw-postchange.log` **54 passed, `EXIT=0`** |
| `spikeEgress()` in `campaigns.go` → `TestNothingInThisPackageSends` reds | `red-go-prechange-egress.log` — `--- PASS`, **`EXIT=0`** (the false gate) | `red-go-postchange-egress.log` — `--- FAIL` naming `campaigns.go`, **`EXIT=1`**; mutation removed, `green-go-postchange.log` `--- PASS`, **`EXIT=0`** |
| schema-v1 store opens with v0 rows intact | `red-schema-harness-prechange.log` — the new gate against the pre-change `replicas.js`: `RED: expected version 1, got 0`, **`EXIT=1`**; `red-pw-ti-prechange-replicas.log` — `[TI-01]` and `[TI-02]` both red, **`EXIT=1`** | `green-schema-harness.log` **`EXIT=0`** (codes 2/2, offers 2/2 rows at v1, VD2 ×2, DB6 and COL12 controls); `[TI-01]` + `[TI-02]` green inside the 54 |

**Read honestly — what "no spec stays green by never consulting the predicate" does and does
not mean here.** Under the inversion exactly ONE spec in the file reds, before and after. The
other 53 are green because they do not reach the no-source arm (their policy source is healthy,
or they are about other things) — none of them CLAIMS to guard it. The one spec that claimed it
and did not test it, `[SP-03b]`, is gone. The predicate's decision-166 half ("names no campaign →
not fail-closed") is now asserted on the shipped function by the harness, and reds under the
same inversion (line `✗ decision 166: …` in `red-harness-postchange-inverted.log`). In the
Playwright file the no-source arm is still guarded by one spec, `[SP-03]`.

**Every dependent of the schema version, executed** (`substrate-legs-summary.log`, one hold of
the suite lock, 22:52–22:56): `push-run.sh`, `run.sh`, `clock-run.sh`, `f2-run.sh`,
`refusal-run.sh`, `recovery-clear-run.sh` — all **`EXIT=0`**, `VERDICT: GREEN`, default (green)
mode, building the v1 collections against the live substrate.

**Which storage each schema test ran on.** `[TI-01]` / the harness: RxDB memory storage + ajv,
in node. `[TI-02]`: Chromium's IndexedDB through Dexie, in the page, through the shipped bundle
and the shipped `marketingCollectionSpec()` — two v0 collections with two rows each reopened at
v1 with rows equal. That is one browser engine, headless, on a fresh database; it is not a
phone, and B-441 (the `scan_attempts` Dexie path) is not touched or retired by it.

**A limit, measured, that the BACKLOG row for B-465 did not anticipate**
(`probe-browser-required.log`, a throwaway spec, deleted): on the post-change tree the PAGE
accepts and stores a `codes` row with no `campaign_id` — `{"accepted":true,"stored":true,
"version":1}`. `scan-page.js` wraps no validator around Dexie, so on the phone `required` is a
declared shape. "The invariant is a schema fact" is true under a validating storage (every node
harness) and is NOT enforced in the browser; there the enforcement is the server's NOT NULL on
both tables. So the branch `policyFor(…, null)` via `offerReady` is unreachable from pulled data
and still reachable from a locally written row. Wrapping the browser storage in a validator
would change what the phone does on a bad row — not this card's call.

## Gates (observed)

* **G1** `g1.log` — `go build ./...` `EXIT_BUILD=0`, `go vet ./...` `EXIT_VET=0` (from `backend/`).
* **G2-Go** `g2-go.log` — `go test -p 1 -count=1 -v ./...` under the suite lock, `EXIT_TEST=0`:
  15 packages `ok`, 447 top-level tests — 444 pass / 0 fail / 3 skip (identical to tonight's
  base); `TestRowVisibilityRLS` 59 subtests pass; `HQ_SYNC_SUBSTRATE_OPTIONAL` and
  `HQ_SYNC_GATE_CHILD` unset; `DB_TEST_URL` = `:5434/hq_test_go_i1`.
* **G2-Playwright** `g2-pw-full.log` / `g2-pw-full.specs.txt` — full suite under the suite lock,
  hand-provisioned stack (`NIGHTCREW_ENV_URL`, `:8211`, `hq_test_e2e_i1_20261003` reset), one
  summary block: **28 failed / 7 skipped / 1028 passed (32.9m), `EXIT=1`**, 1063 tests (base 1062:
  −`[SP-03b]` +`[TI-01]` +`[TI-02]`). Every `tests/marketing.spec.js` test passed.
  Base tonight: 25 failed. **All 25 base reds are red here; 3 + 1 more:**
  `inventory.spec.js:2931` (in the 24-red set, green in tonight's one base sample),
  `onboarding.spec.js:2233` / `:2268` (the named pair in the 29-red set), and
  **`sync.spec.js:1327` "checkbox answer converges (live + catch-up)" — in neither measured set.**
  Attribution: (a) mechanical — the card's diff outside `.night-crew/` is seven files under
  `marketing/` + `tests/marketing.spec.js` + one Go test + `sw.js`; only `marketing.html` loads
  the changed modules and service workers are blocked in the suite, so none of the four specs can
  execute a changed line; (b) `iso-extra-reds.log` — the four, ×3 each, confined on the card tree:
  **12 passed, `EXIT=0`**. NOT done: the same isolation on the base tree (the base worktree is the
  orchestrator's). So: order/load-dependent at whole-suite position, not reproduced in isolation,
  not attributable to this diff; `sync.spec.js:1327` is new to the named lists and wants a
  `bugs.md` line at triage.
* **G4** `g4-sw.log` — `node build-sw.js` `EXIT=0`, **51 files precached** (51 before); a second
  run at the `sw.js` commit left the tree clean (idempotent).
* The full suite rewrote 16 tracked PNGs under `.night-crew/runs/2026-10-02-autonomous/logs/`
  (B-467); restored with `git checkout --`, none committed.

## Fix round (G6: APPROVE-WITH-FINDINGS, one P2)

**P2 — `MARKETING_MIGRATION_STRATEGIES` was documented "Total and lossless"; it is not total.**
Reproduced: a v0 store holding a row WITHOUT `campaign_id`, reopened through the shipped
`marketingCollectionSpec()` — under an ajv-wrapped storage `addCollections` throws **DM4**; under
a non-validating storage (and in Chromium/Dexie) the row is carried to v1 unchanged and the store
opens. No committed fixture had that row shape.

Fixed as a comment + test change. **The strategy body, the schema and everything the phone does
are unchanged** (`git diff` on `replicas.js` for this round is comment lines only). The phone
runs a non-validating storage and no pulled row can lack `campaign_id`, so the strategy is safe
as shipped; it stops being safe the day the browser storage is wrapped in a validator, and the
comment now says that first.

| | Evidence | Exit |
|---|---|---|
| RED — a leg asserting the comment's "total" claim, tree `73521b1` | `fixround-red-total-claim.log` — `the reopen was REFUSED with DM4` | `EXIT=1` |
| GREEN — leg 5 pins the observed behaviour on both storages | `fixround-green-schema-harness.log` — 5(a) validating → DM4; 5(b) non-validating → codes 2/2, offers 2/2 carried, campaign-less row unchanged | `EXIT=0` |
| MUTATION — strategy changed to drop the row (`d => d.campaign_id ? d : null`), then reverted | `fixround-red-strategy-mutated.log` — `5(a): the reopen SUCCEEDED under a validating storage` | `EXIT=1` |
| Confined gate — whole `tests/marketing.spec.js`, unmutated | `fixround-green-pw-marketing.log` — **54 passed (1.1m)** | `EXIT=0` |

What turns red on which future change: changing the strategy (drop or default the row) → harness
leg 5(a) or 5(b), and so `[TI-01]`; wrapping the BROWSER storage in a validator → `[TI-02]`, whose
v0 fixture now carries a campaign-less row in both `codes` and `offers` and requires it back
unchanged. 5(b) runs in a child process of the harness because the dev-mode plugin refuses a
non-validating storage (DVM1). Not re-run this round, by instruction: the full Go and Playwright
suites, and the substrate harnesses (none of their files changed).
