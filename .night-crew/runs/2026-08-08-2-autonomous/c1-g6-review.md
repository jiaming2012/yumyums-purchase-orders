# G6 Review — Card C1 `skeleton-offline-ownership-honesty` (run 20260808-2)

Fresh-context adversarial reviewer; inputs were the slate entry, the diff on
`card/c1-skeleton-offline-ownership-honesty`, and the committed evidence only.
Reviewer left the card worktree byte-identical (probe edits reverted, artifacts removed).

## VERDICT: PASS-with-findings

The card does what it claims: the three source-text assertions are gone, the replacement
asserts against a real running page, the guard reds on a violation, gates check out
arithmetically, and B-88's lead is followed. But the committed red-first evidence proved
the guard against a violation shape that cannot physically occur, and the assertion that
actually gates the realistic violation is not the one the test is named for.

## Findings, ranked

**F1 — MEDIUM (evidence quality):** RF Probe B injected `HQSync.db` synchronously before
`window.HQSync` assignment — a shape real code cannot produce, since
`createHQSyncDatabase` is async (`sync-rxdb/client.js:1089`) and a real database only
arrives via a resolved promise after the test's `waitForFunction` has fired. G6 ran the
realistic violation (unconditional `createHQSyncDatabase().then(db => { HQSync.db = db })`
injected into bootstrap.js), 3 runs: **red 3/3 — every failure from the second assertion**
(`rxdbIndexedDbNames` caught `rxdb-dexie-hq_sync--0--*`), never from
`expect(state.dbUndefined).toBe(true)`, which samples before the promise resolves. The
acceptance criterion holds (guard gates, verified by execution), but the headline
assertion is timing-blind to every violation C2 could actually write; the IndexedDB
defense-in-depth check is the real gate.

**F2 — MINOR:** The load-bearing check is storage-scoped: `indexedDB.databases()`
filtered by `/rxdb|hq_sync/i`. Complete for every storage the vendored bundle can
construct today (Dexie only; memory storage not exported), but a future vendored memory
storage would evade both assertions. Carry as a line in C2's charter.

**F3 — MINOR:** "The sync flag" names nothing that exists — no flag identifier,
mechanism, or default is defined anywhere in the tree. The contract is still satisfiable
because it is stated behaviorally and the test is the contract; C2 must define the
mechanism.

**F4 — TRIVIAL (evidentiary):** G4's "tree clean both runs" was asserted, not evidenced
in the g4-*.log files. G6 re-ran `node build-sw.js` on the branch tip: byte-identical
sw.js, 31 precached, git status clean. Claim true; evidence incomplete.

## Re-ran vs audited

Re-ran (isolated env TEST_PORT=4321 / hq_test_c1g6): baseline B-88 test (1 passed, exit
0); realistic async-violation injection (1 failed × 3); build-sw.js idempotency (31
precached, clean). Audited from committed logs: G1, both G2 legs, G4, RF probes,
rerun-suspect-reds. Did not re-run the 22.5m full suite or anything touching :5433.

## Numbered answers (abridged)

1. RF Probe A verified (old guard blind to `HQSync.db`, exit 1). Probe B verified as far
   as it goes but proved the wrong violation shape (F1); realistic case re-proved by G6.
2. Asserts against the RUNNING page (login → goto workflows.html → waitForFunction →
   evaluate; also ran in the full suite, test 487).
3. "No RxDB collection reachable" enforced by proxy (db undefined + IndexedDB name scan);
   complete for current storages (F2 residual).
4. Flag-off contract behaviorally precise and enforceable; flag itself defined nowhere
   (F3).
5. No silent weakening from trimming: the two kept positives survive meaningfully; the
   three dropped negatives are compensated by the object-level check.
6. All counts verified: G2 Go 9 ok / 454 PASS / workflow=35 exactly; env attestation
   stated and uncontradicted; G2 PW exactly one summary block, 791/2/6, armed baseline
   (inventory:883, receipt-carousel:123, sync:446) all passed at the claimed log lines.
   Flake ruling on the two non-armed reds (inventory:3124, sync:1327) supported by
   rerun-suspect-reds.log (2 passed isolated, --retries=0, exit 0) and a disjoint diff.
7. G4 precache 31 in all four logs, final pair post-commit, idempotency re-verified.

## Orchestrator disposition

Merged without a fix round: acceptance met and independently re-proven by execution.
F1–F3 carried verbatim into C2's dispatch prompt; F2/F3 are C2 charter lines. Recorded
in conflicts-20260808-2.md.
