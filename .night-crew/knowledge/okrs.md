# OKRs

<!-- Cycle: "Nothing silently lost" — checklist data integrity. Opened 2026-07-16 after the
HQ-hardening cycle closed (gate PASS, attest & waive). Trigger: operator-reported, same-day
reproduced P0 — editing a template while crew devices have the checklist open silently
discards their work (field-ID churn in replaceTemplate + FK-dropped silent dead-id writes).
Theme: every way the checklist engine can silently lose crew-entered work is enumerated,
fixed, and made structurally impossible (immutable run-pinned template versions — operator
chose design+build this cycle, 2026-07-16). Carried from last cycle: retire waiver #1
(suite green — substantially paid by 42eeb39: 425/6/0 deterministic) and waiver #2
(vacuous-test remainder ~18). Deferred by operator decision 2026-07-16: onboarding
video-pipeline fixture (off-theme, needs creds); /nc-status framework WO (stays in backlog,
outside this repo). -->

## Product

### Objective: Every silent-loss mode in the checklist engine is enumerated, specified, and has a falsifiable acceptance test before build starts.
- A data-integrity PRD (template-edit stages 1–3) is delivered as a blocking gate before WO build, with 100% of its requirements tracing to either a reproduced failure mode or a named invariant (trace table in the PRD; audited at cycle gate).
- The mid-run edit semantic ("crews finish the run they started; edits take effect next run") is recorded as 1 operator-signed decision in the PRD and encoded as ≥ 2 passing acceptance tests: a mid-run edit leaves the in-flight run untouched, and the next run reflects the new shape.
- 8/8 backlog items marked `new` at cycle open (2026-07-16) are routed through a door — folded into the PRD, promoted to a roadmap card, or deferred with a written reason in BACKLOG.md (routing count auditable from BACKLOG.md statuses at cycle end).

## Delivery

### Objective: The data-integrity fixes ship in sequence — interim protection first, structural fix behind it — and reach production, not just dev.
- Stage 1 ships first: `tests/repro-cut-task.spec.js` flips red→green and is committed to the suite (repro currently fails on dev by design; its red run is the recorded baseline).
- Stage 2 ships with a red-first E2E covering the mixed old/new-device case (a device that opened the checklist after the edit syncs with one that opened before it).
- The versioning design (OpenSpec change) is operator-signed BEFORE any versioning build card is dispatched — 0 build WOs start ahead of the signed design (auditable from ledger timestamps).
- Prod parity at cycle end: `task version` shows prod backend/frontend equal to local `version.go` constants (includes today's 42eeb39 fixes plus everything this cycle ships).
- Median WO cycle time over this cycle's WOs ≤ the baseline recorded last cycle (first cycle with a real target; measured from the run records).

## Engineering

### Objective: Field identity is stable across template edits — first by preservation (stage 1), then by construction (immutable run-pinned template versions).
- Stage 1a: after any template edit, every surviving field keeps its ID — `updateTemplate` upserts by the IDs the Builder already sends instead of delete+reinsert (asserted by the repro spec's post-edit checks).
- Stage 1b: a draft save (`submission_id IS NULL`) naming an unknown field ID is rejected server-side with a distinct error envelope — app-level existence check, NOT a restored FK (submitted responses reference snapshot IDs by design); red-first Go test.
- Stage 3 built: template edits create immutable versions with "the template" as a head pointer; a checklist run pins the version current at run start; responses key on (run, field-UUID). Proven by E2E: edit mid-run → the in-flight run is unaffected and the next run shows the new shape; migration applied with all existing templates/drafts intact (`task test` green post-migration).
- `task test` exits 0 at cycle end on the deterministic stack — formally retires carried waiver #1 ("0 pre-existing reds", waived 2026-07-15, substantially paid by 42eeb39's 425 passed / 6 skipped / 0 failed).

## QA

### Objective: Carried test-debt is retired, every fix lands red-first, and prod data mutations are reversible.
- Vacuous-test remainder: 18 → 0 — each remaining conditional `test.skip()` / silent guard-return becomes a real seeded assertion or is deleted (counted by the same audit that produced the 18).
- 100% of this cycle's fix-WOs carry red-run evidence — the test failed before the fix, recorded in the WO record (bug-fix protocol; denominator = all WOs classified fix).
- Prod ghost catalog item resolved with the operator-chosen handling (rename `''` → `(Unnamed — needs review)`, keep line-item links): post-WO, `SELECT count(*) FROM purchase_items WHERE trim(description)=''` returns 0 in prod AND the previously-linked purchase_line_items count is unchanged.
- 100% of schema migrations shipped this cycle have a down-migration proven by an up→down→up cycle run green in the WO record, and 100% of prod deploys that include a migration record 1 pre-deploy DB backup artifact — 0 irreversible schema changes reach prod.
