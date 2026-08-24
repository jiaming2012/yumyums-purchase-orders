# G6 Review — Card S1 `list-views-decision-recording` (run 20260808-2)

Fresh-context adversarial reviewer; inputs were the slate entry, the diff on
`card/s1-list-views-decision-recording` (base 09aaa0e), and committed evidence only.
Documentation-honesty card → the review was a truth-audit: every banner sentence
checked against code reality and against T-43 as recorded in the ledger. All probes
read-only; nothing re-run; nothing touched any port.

## VERDICT: PASS-with-findings (all minor; none blocked merge)

- Truth audit held on every sentence: both list views proven fetch-rendered
  (renderMyChecklists ← GET myChecklists, renderApprovals ← GET pendingApprovals,
  neither touches RxDB); "Approvals BY RULING T-43(a)" matches the ledger verbatim;
  T-43(b) stated OPEN everywhere with no predicted outcome — the banner quotes the
  operator's deferral verbatim.
- Corrected cancel wording ("cancel before re-scoping THE SAME shape") matches what
  C3 implements (same scope → same handle; different scopes concurrent; close on tab
  switch/exit). The pre-B-63 full-stop wording survives only inside quotations.
- C3's five must-survive contracts all intact (verified line-by-line).
- B-64 closure legitimate (stale content gone, found-by-content account matches the
  diff); B-63 closure earned ([SCOPE-05] + [FILL-02] are exactly B-63's lead).
- RF "n/a — non-code deliverable" verified TRUE mechanically: stripping comment lines
  from the sync-rxdb/ diff leaves zero changed lines; sw.js diff is two revision
  hashes for exactly the two edited files, count 31.
- Gates: G1 0/0; G2 Go 9 ok / 454 PASS recounted / workflow=35 / env attested UNSET /
  :5434; G2 PW one summary block, 811/0/6 EXIT=0, armed trio ✓ at cited lines; G4
  idempotent, precache 31, version parity 1.4.0.

## Findings

- **F-1 (minor):** the "WHAT IS LIVE TODAY" fill-view bullet omits the flag-gate
  clause the C2 bullet carries — imprecise, not false (module-level rule at
  bootstrap.js:26 covers it). Fixed by the merger in the post-merge docs commit.
- **F-2 (minor, observation):** "many live at once" is the lifecycle's contract; the
  shipped tap-path serializes (single fillState slot; close-before-open). True at the
  layer the banner names ([FILL-02] proves concurrency live). No change.
- **F-3 (minor, convention):** B-63/B-64 closures lacked the commit SHA the file's
  own convention requires (the SHA ships inside the commit it would cite). Fixed by
  the merger: 35cb917 appended to both entries post-merge.
- **F-4 (negligible):** roadmap "docs-only diff" enumeration omits the regenerated
  sw.js. No change.

## Orchestrator disposition

Merged. F-1 + F-3 applied by the orchestrator in a docs commit immediately after the
merge (comments-only; sw.js regenerated post-commit per B-37). F-2/F-4 ride this
record.
