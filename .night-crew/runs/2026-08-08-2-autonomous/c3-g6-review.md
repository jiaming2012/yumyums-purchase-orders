# G6 Review — Card C3 `activate-fill-view-reads` (run 20260808-2)

Fresh-context adversarial reviewer; inputs were the slate entry, the diff on
`card/c3-activate-fill-view-reads` (base 04c6703), and committed evidence only. Two
rounds: initial review, then re-verification of the orchestrator-ordered fix round by
the same reviewer re-running its own probe. Worktree left byte-identical both times;
nothing touched :5433.

## Round 1 VERDICT: PASS-with-findings (one CONFIRMED TOP defect → fix round ordered)

Binding constraints all held under attack: T-43(b) (both list views REST, flag on and
off — FILL-01 asserts it), T-43(c) (two live fill scopes, pairwise-distinct identifiers
carrying the user), decision 105 (per-open-checklist; F-2's userId is a narrowing),
decision 111 (no substrate change — no SQL, policy, column, or filter clause in the
diff), decision 126 + persistence rule (no write path touched; B-88 guard green), spike
E (no polling). C2's G6 findings both discharged: F-1 rejection eviction (tested via the
`HQSync.createDatabase` seam, FILL-03), F-2 userId required on the fill scope
(SCOPE-05: byte-equal filters, distinct fingerprints/checkpoints per user; missing
userId REFUSED, not defaulted).

### Round-1 findings

**F-A (TOP, CONFIRMED by runtime probe):** the fill overlay applied response rows
belonging to a DIFFERENT submission. Subscription selected on `field_id` alone;
`applyRxdbFillDocs` never compared `doc.submission_id` to the open checklist; the
Dexie DB is persistent and `cancel()` purges nothing; field ids are per-template —
shared by every submission of that template. Yesterday's rows, legitimately resident,
marked today's blank recurring checklist answered with yesterday's values. Probe:
planted a resident row with foreign `submission_id` → runner rendered "1 of 1 items
complete" on a blank checklist. The card's own "stated bound" analyzed only the
concurrent same-template case; the sequential resident-row case is the default daily
path. **F-B (minor):** `show(2)`/`show(3)` did not close the active fill scope.
**F-C (minor, PLAUSIBLE):** close-then-fast-reopen race could strand a dead
replication reported live (deferred cancel killing the handle a reopen now holds).
**F-D (observation):** "RLS is the gate" prose oversold — `submission_responses_select`
gates field visibility (`hq_can_see_field`), not draft authorship; foreign users'
NULL-submission drafts do replicate.

## Fix round (276068b test-first / 3e4397d fix / a4d1ccc docs)

`acceptedFillDocs(docs, checklistId, userId)` admits only `submission_id ===
checklistId` OR (`submission_id == null && answered_by === me`) — mirroring the REST
hydrate's semantics; overlay slice rebuilt per emission (entry.sig guard) so a
row that stops being accepted cannot leave a stale answer; scope context passed
per-scope, not read off fillState. `closeActiveFillScope()` moved to the top of
`show()` — every tab switch closes, placed after the unsaved-changes confirm so a
cancelled navigation keeps the scope. Deferred cancel guarded by handle identity (not
checklist id — a different-scope reopen still gets the old replication cancelled).
Prose corrected in client.js and SCOPE-05 header. Red-first: [FILL-04]/[FILL-05]
committed alone, 3 failed pre-fix (log carries "2 of 2 items complete" on a blank
two-field checklist), 0-diff attested across legs. Full suite on final tree:
**811 passed / 0 failed / 6 skipped, EXIT=0** — armed trio passing; GLB-01
(workflows:3909, round 1's single unarmed red, re-run 4/4 green then) passed, retired
as flake.

## Re-verification VERDICT: PASS

- PROBE-1 (the original F-A attack, byte-for-byte): runner now shows "0 of 1 items
  complete", overlay empty. The fix holds against the exact attack that confirmed the
  finding.
- PROBE-2 (REST parity, untested by the card's own suite): substrate row with the OPEN
  submission's id and a foreign `answered_by` → renders. Only foreign submissions and
  foreign NULL-drafts are excluded, as required by shared-submission semantics.
- Full card surface re-run in G6's isolated env: 73/73 passed, exit 0 (FILL-01..05,
  SCOPE-01..05, B-88 guard, no-write-path-rerouted test).
- Hole hunt on the fix itself: sig collisions can only cause spurious rebuilds, never a
  missed change; slice bounded to the scope's own ids (product scopes disjoint);
  handle-identity guard has no different-scope hole; emission-in-flight-after-close
  early-returns. Nothing found.
- Observations only (no card): O-1 `overlayKeys()` returns a snapshot object, not keys
  — sane surface, misleading name; O-2 FILL-04's positive half doesn't cover the
  teammate-answered case (PROBE-2 does; fold in later).

## Orchestrator disposition

Fix round ordered on round 1 (F-A was an acceptance-level defect of this card's own
surface, unlike C2's C3-facing findings). Merged after re-verification PASS. O-1/O-2
ride this record.
