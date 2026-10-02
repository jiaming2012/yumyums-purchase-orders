# Spikes — test-integrity-fix

Activity: Activity I — Honest gates, then verify before the till (triage 20261002 follow-ups)

> Hand-run convention (see README.md in this directory). Spike 1 runs in a THROWAWAY git
> worktree off `dev` so the mutations never touch the main tree; its harness leg uses the
> LOCAL spike-supabase substrate (reconcile). Spike 2 is node-only against the vendored RxDB.

## The goal, and which legs need a spike

The card (I1, the operator's stated condition for merging run 20261002 — ledger T-62): three
gates that pass against inverted or widened production code, plus the three decision-191
amendment riders. **B-462** — `marketing/sync/harness/campaigns-harness.mjs` leg 3 asserts a
hand-copied mirror of `failClosed`, not `marketing/submit-flow.js`'s. **B-465** — `[SP-03b]`
(`tests/marketing.spec.js`) passes against an inverted `failClosed` because for
`kind='unknownCode'` the predicate is never consulted; the branch it claims to guard exists
only because `marketing/sync/replicas.js` `MARKETING_REPLICA_SCHEMA` omits `campaign_id` from
`required`, a row both databases forbid. **B-466** — `TestNothingInThisPackageSends`
(`backend/internal/marketing/subscribers_test.go`) scans two filenames, so outbound HTTP added
to `campaigns.go` passes. Riders (decision 191 as amended): add `campaign_id` to the replica's
`required`; leg 3 must IMPORT the shipped `failClosed`; the "negative assertion moves with it"
claim was never true and the harness must now assert the import.

Two premises a script can settle now: (1) the three false gates reproduce by execution on
today's `dev`, exactly as triage reported — the card's red-first baseline; (2) adding
`campaign_id` to `required` on a version-0 RxDB schema is refused by RxDB on a store that
already holds rows, so the rider is a replica-schema migration (the three-part shape
`replicas.js` already documents for v1 of the campaigns schema), not a one-line edit — which
is what sizes the card.

## Spike: false-gates-pass-against-inverted-code

- proves: on a throwaway worktree of `dev`, (a) with `failClosed` inverted in
  `submit-flow.js`, `[SP-03]` goes red and `[SP-03b]` stays green, and the unmutated control
  is 2 passed; (b) with the same inversion `campaigns-run.sh` exits 0 (leg 3 asserts its own
  copy); (c) with an `http.Post(...)` added to `campaigns.go`,
  `TestNothingInThisPackageSends` still passes. Three false gates, reproduced by execution.
- plan: `git worktree add --detach` off `dev`, symlink node_modules, mutate, run the three
  legs with their exit codes captured, restore, remove the worktree. Spike-owned
  `TEST_DB_NAME=hq_test_spike_i1_20261003`, `TEST_PORT=8311`, Go `hq_test_spike_i1_go`.
- script: .night-crew/spikes/activity-i-honest-gates-verify-first/test-integrity-fix/01-false-gates-pass-against-inverted-code.sh

## Spike: required-campaign-id-needs-a-replica-migration

- proves: a store created with the shipped `MARKETING_REPLICA_SCHEMA` (version 0) holding one
  row refuses to reopen with `campaign_id` added to `required` at the same version, and
  reopens cleanly at version 1 with a migration strategy — so the rider ships as a schema
  version bump with `migrationStrategies`, and `[SP-03b]` retires because the row it tests
  can no longer exist on the device.
- plan: node against the vendored RxDB (the harness's own imports), a persistent storage so
  the reopen is real, three opens: v0 insert → v0+required (expect the schema-mismatch error)
  → v1+required+strategy (expect the row present).
- script: .night-crew/spikes/activity-i-honest-gates-verify-first/test-integrity-fix/02-required-campaign-id-needs-a-replica-migration.sh

## Verdict (hand-run 2026-10-02)

- **false-gates-pass-against-inverted-code: passed** — exit 0 on the second clean-state run
  (the first went RED on the CONTROL leg from a script bug — `grep -qx "[SP-03]=…"` read the
  brackets as a character class; fixed with `-F`, worktree and DBs re-created, re-run), 5 m 20 s,
  in a throwaway worktree off `dev` with three `node_modules` symlinks (root, the harness, and
  `.night-crew/qa/spike-supabase/rxdb` — `env-up.sh`'s healthcheck resolves modules from the
  worktree's QA dir). Legs: (a) control `[SP-03]=pass [SP-03b]=pass`, exactly 2 specs; with
  `failClosed` inverted (1-line diff) `[SP-03]=FAIL [SP-03b]=pass` — **B-465 reproduced**; (b)
  `campaigns-run.sh` with the inversion in place → `✅ VERDICT: GREEN`, exit 0 — **B-462
  reproduced**, and the static read printed why: the harness has ZERO import lines naming
  `submit-flow.js` (its only submit-path import is `submit-machine.js`), so the mutated file is
  never loaded; (c) `TestNothingInThisPackageSends` control PASS; with `spikeEgress()` (an
  `http.Post`) appended to `campaigns.go` → still PASS — **B-466 reproduced**.
- **required-campaign-id-needs-a-replica-migration: passed** — exit 0, first run, ~3 s, node
  against the vendored RxDB 17.4.0 exactly as `campaigns-harness.mjs` builds its database
  (memory storage + ajv validation, dev-mode + migration-schema plugins, the shipped
  `MARKETING_REPLICA_SCHEMA` imported from `replicas.js`): (1) v0 insert, `close()`, reopen v0 →
  row present (the reopen is real — memory storage keeps collection state across `close()`,
  only `remove()` drops it); (2) reopen v0 with `campaign_id` added to `required` → **refused,
  `DB6`** ("another instance created this collection with a different schema", hash
  `9561dff2…` vs `4fec2bab…`); (3) reopen v1 + `required` + `migrationStrategies {1: d => d}`
  → row present at `schema.version=1`; (4) v1 insert without `campaign_id` → **refused,
  `VD2`**. The rider is a schema migration, as the ledger said.

## Corrections

- none — no agent-reached corrections to a premise. Five build-facts for the card, stated:
  (1) `campaigns-harness.mjs` is a zero-coupling reimplementation, not an inexact mirror — it
  imports nothing from `submit-flow.js`, and `failClosed` / `policyFor` are module-internal
  there today, so the import rider requires `submit-flow.js` to EXPORT them first; (2) the
  error codes to pin: same-version widen → `DB6`, v1 insert without `campaign_id` → `VD2`;
  (3) RxDB 17: `db.close()` (no `destroy()`), and `addCollections` awaits the auto-migration —
  no explicit `migratePromise()`; (4) the internal collection doc is keyed `name-version`,
  which is why same-version → `DB6` and v1 → migration — BOTH `codes` and `offers` use
  `MARKETING_REPLICA_SCHEMA`, so `marketingCollectionSpec()` carries the strategies for both,
  and every builder already registers `RxDBMigrationSchemaPlugin` (B-447's change) — verify
  per harness rather than assume; (5) a worktree run of the harness needs the third
  `node_modules` symlink (`.night-crew/qa/spike-supabase/rxdb`), or `env-up.sh` fails on module
  resolution and LOOKS like a substrate failure.

## Comebacks

- **B-441 is NOT retired by spike 02:** no `fake-indexeddb` is vendored, so the Dexie v0→v1
  path is still executed by no test; the `DB6` / migration checks sit above the storage layer
  (`rx-database.js:323` and the migration plugin), so the verdict is storage-agnostic, but
  "the Dexie migration path is unproven" stays filed. The card states which path its test
  exercises.

## Review

- signed: operator, 2026-10-02 — covers 0 correction(s) (reviewed at the slate sitting of 2026-10-02, §4 batch sign-off, with the sitting's call for each in view)
