# OKRs

<!-- Cycle: "Nothing silently lost" — checklist data integrity. Opened 2026-07-16 after the
HQ-hardening cycle closed (gate PASS, attest & waive). Trigger: operator-reported, same-day
reproduced P0 — editing a template while crew devices have the checklist open silently
discards their work (field-ID churn in replaceTemplate + FK-dropped silent dead-id writes).
Theme: every way the checklist engine can silently lose crew-entered work is enumerated,
fixed, and made structurally impossible (immutable run-pinned template versions — operator
chose design+build this cycle, 2026-07-16). AMENDED at the evening PM grill-back
2026-07-16: straight to versioning — stages 1–2 deleted. Premise correction: production
has no active Operations users, so there is no interim-protection urgency; the operator
accepts wipe/reseed of prod Operations (workflow) data at the versioning deploy. RE-AMENDED
later the same grill-back: the semantic itself was put head-to-head — operator delegated
the choice to the PM with a hard UX bar (multi-device sync always convergent: all field
types, sub-steps, submit/unsubmit, progress bars); PM chose FROZEN-AT-SUBMIT (edits appear
live on unsubmitted checklists; submit freezes the record; rejection reopens live). The
versioning schema is deleted — stable field identity + loud rejection + edit broadcast
(old stages 1–2, revived as the permanent architecture) + transactional op emission +
a device-convergence matrix deliver the theme with no schema migration (the wipe is moot).
Carried from last cycle: retire waiver #1
(suite green — substantially paid by 42eeb39: 425/6/0 deterministic) and waiver #2
(vacuous-test remainder ~18). Deferred by operator decision 2026-07-16: onboarding
video-pipeline fixture (off-theme, needs creds); /nc-status framework WO (stays in backlog,
outside this repo). -->

## Product

### Objective: Every silent-loss mode in the checklist engine is enumerated, specified, and has a falsifiable acceptance test before build starts.
- A data-integrity PRD (frozen-at-submit edit propagation + engine-trust fixes) is delivered as a blocking gate before WO build, with 100% of its requirements tracing to either a reproduced failure mode or a named invariant (trace table in the PRD; audited at cycle gate).
- The edit semantic ("an unsubmitted checklist always shows the current template on every device; submit freezes the record forever; rejection reopens it live") is recorded as 1 operator-delegated, sign-off-ratified decision in the PRD and encoded as ≥ 2 passing acceptance tests: a mid-run edit re-renders every open device to the new shape with surviving answers intact, and a submitted checklist is unaffected by any later edit.
- 15/15 backlog items marked `new` at cycle open (2026-07-16: 8 main-list + 4 test-hardening notes + 3 prove-sweep PARK fix-WOs) are routed through a door — folded into the PRD, promoted to a roadmap card, or deferred with a written reason in BACKLOG.md; 0 `· new` markers remain (auditable via `grep -c '· new' BACKLOG.md`).

## Delivery

### Objective: The structural fix ships behind a signed design and reaches production, not just dev.
- The edit-propagation design (OpenSpec change: frozen-at-submit semantic, rejection rules, cut-field discard warning, convergence contract) is operator-signed BEFORE any build card is dispatched — 0 build WOs start ahead of the signed design (auditable from ledger timestamps).
- `tests/repro-cut-task.spec.js` is rewritten to assert the frozen-at-submit semantic and lands in the suite with exactly 1 recorded red baseline run (against the unfixed build) and 1 green run (post-fix) in the WO record — 0 edit-propagation WOs close without this pair.
- Prod parity at cycle end: `task version` shows prod backend/frontend equal to local `version.go` constants (includes today's 42eeb39 fixes plus everything this cycle ships).
- Median WO cycle time over this cycle's WOs ≤ the baseline recorded last cycle (first cycle with a real target; measured from the run records).

## Engineering

### Objective: Field identity is stable across template edits, edits propagate live, and multi-device state always converges — frozen-at-submit.
- Stable identity: after any template edit, 100% of surviving fields keep their ID — `updateTemplate` upserts by the IDs the Builder already sends instead of delete+reinsert (asserted by the rewritten repro spec's post-edit checks).
- Loud rejection: 100% of writes naming a field absent from the current template are rejected server-side with a distinct error envelope, and 0 return 200 (red-first Go test) — app-level existence check, NOT a restored FK.
- Edit propagation: a template edit re-renders every open device to the new shape with surviving answers intact and stays silent on catch-up replay; op emission is transactional with the write it describes — 0 accepted writes whose op is not durably queued for other devices (red-first test on the forced-failure path).
- Convergence matrix: all 7 field types + sub-steps + submit/unsubmit transitions + list-view progress indicators converge across ≥ 2 devices in a red-first E2E matrix — 0 matrix cells red at cycle end.
- `task test` exits 0 at cycle end on the deterministic stack — formally retires carried waiver #1 ("0 pre-existing reds", waived 2026-07-15, substantially paid by 42eeb39's 425 passed / 6 skipped / 0 failed).

## QA

### Objective: Carried test-debt is retired, every fix lands red-first, and prod data mutations are reversible.
- Vacuous-test remainder: 18 → 0 — each remaining conditional `test.skip()` / silent guard-return becomes a real seeded assertion or is deleted (counted by the same audit that produced the 18).
- 100% of this cycle's fix-WOs carry red-run evidence — the test failed before the fix, recorded in the WO record (bug-fix protocol; denominator = all WOs classified fix).
- Prod ghost catalog item resolved with the operator-chosen handling (rename `''` → `(Unnamed — needs review)`, keep line-item links): post-WO, `SELECT count(*) FROM purchase_items WHERE trim(description)=''` returns 0 in prod AND the previously-linked purchase_line_items count is unchanged.
- 100% of schema migrations shipped this cycle have a down-migration proven by an up→down→up cycle run green in the WO record, and 100% of prod deploys that include a migration record 1 pre-deploy DB backup artifact — 0 irreversible schema changes reach prod.
