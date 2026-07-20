# DECISIONS-NEEDED — overnight-20260721

> **RESOLVED 2026-07-20 — recorded as `ledger.md` §T-18.** Run merged to `dev` `--no-ff`
> (`e1d22ad`). **B1:** design SIGNED — A4 = Option (i) two slugs · D2 = Ungrouped · rider (a)
> kept · rider (b) REWRITTEN to umbrella semantics (operator: "App grant = All tabs granted.
> They should not be considered separate objects.") · rider (c) signed expected → Activity 4
> UNBLOCKED. **B2+B3:** promoted as one card, `replay-fetchstorm-gate` (operator: "promote it").
> **B4:** ratified WIDER than drafted (operator: "Everyone should see live ops" — fan-out =
> everyone with entity access). **B5:** folded into `inventory-tab-gating`. Kept as the run's
> analysis record.

> Read with HANDOFF.md. §A = parked work (none tonight). §B = operator decisions the run
> surfaced but — per the standing rule — did not make. Resolve via `/nc-morning-triage`.

## §A — PARKed cards / cells

**None.** 4/4 cards DONE, 0 matrix cells parked, 0 footprint breaches, no worktrees preserved.

## §B — Operator forks surfaced (the run never decides)

### B1 — Activity-2 design sign-off: A4 + D2 + three riders *(the attended gate)*

The draft (`.night-crew/knowledge/designs/prove-surface-gating-and-endpoints.md`) is your signable
input. It does not choose. Open items it hands you, with the draft's advisory position in italics:

- **A4 (PRD FR-6a, queued 2026-07-19):** per-tab `app_permissions` representation —
  **(i) two dedicated slugs** (`inventory-trends`, `inventory-cost`; no migration) vs
  **(ii) a per-tab sub-permission column** (migration → down-migration + up→down→up proof +
  pre-deploy backup under NFR-3/INV-E). *Draft recommends (i).* Checkboxes in doc §5.
- **D2 (new, discovered by the draft):** linked-but-groupless lines — the `'(no itemized receipt)'`
  sentinel is seeded with `group_id NULL` (migration 0064), so it is neither "unlinked" nor
  groupable, breaking FR-6b's dichotomy. *Draft recommends an "Ungrouped" pseudo-group.* §5, second
  checkbox.
- **Riders (LOW, from C1's G6 — sign-aware, strike if unwanted):** (a) §2.2's per-week `unlinked`
  array elaborates AC-1's literal single "Unlinked $X" total; (b) Option (ii) bakes in
  "whole-app grant does NOT imply tab grants"; (c) §1.6 records "tab grant without app grant →
  tile hidden but direct URL works" as expected behavior — wants a deliberate yes/no.

### B2 — Successor intermittent red: `sync.spec.js:1198 › temperature answer converges (live + catch-up)`

Waiver #1 (LST-08) is **formally retired** — its fix landed and literal `task test` exit-0 was
achieved by the implementer and reproduced verbatim by G6. But G6's *first* independent full run
exited 201 with this one test red (both attempts; 12s `waitForResponse(POST /ops)` timeout), and it
also reddened in one `--retries=0` pair leg on the **pre-fix baseline** file — i.e. it is a
**pre-existing, load/order-sensitive** intermittent, not caused by tonight's card, sitting directly
downstream of the same catch-up fetch storm (§B3). Observed red 2-of-3 G6 legs that included it;
green in the other two runs including the exit-0 runs.

**Fork:** exit-0 is *achieved-and-reproduced, not deterministic*. Options: (a) graduate a
hardening/isolation card for this test next cycle (recommended — pairs with B3, same class);
(b) accept as a known intermittent and watch; (c) open a new narrow waiver. The run refused to
silently claim deterministic exit-0 (07-19 honesty pattern).

### B3 — Production defect class: ungated `SUBMIT_CHECKLIST` replay re-fetch (`sync.js:443`)

Root cause of LST-08 and siblings. On every fresh-context page load, `wsCatchUp` replays the full
ops journal and the `SUBMIT_CHECKLIST` branch fires `loadMyChecklists()` **per replayed op** with no
`silent` gate — the exact "needless fetch storm" sync.js's own `APPROVE_ITEM`/`SAVE_TEMPLATE`
comments warn against (those branches ARE gated). Consequences beyond tests: on a real phone
reconnecting after a long offline stretch this is a per-op fetch flood plus a mid-fill clobber
window (the ESC-class the operator already hit on 2026-07-18 for APPROVE_ITEM). The fix pattern
exists in-file: gate on `(runner open) || !silent`.

**Both the A2 implementer and its G6 recommend a production backlog card; G6 upgrades the urgency**
(and once it lands, A2's test-side `checkAllWithRepair` loop should become dead code and can revert
to plain clicks). B2's intermittent is plausibly the same defect's second symptom. **Fork:** add to
BACKLOG now vs fold into next cycle's slate. (Test-only footprint tonight forbade touching it.)

### B4 — Contract question: approvers in the live-sync fan-out (from B1's coverage)

The recipient query (`ops.go:521-530`) does not filter `template_assignments.assignment_role` —
**approvers receive live ops exactly like assignees**, while the stated FR-7 contract reads
"admins/superadmins ∪ assignees". `TestResolveEntityAccess_ApproverIncluded_CurrentBehavior`
(access_test.go:402-425) pins today's behavior with an explicit reviewer NOTE rather than
enshrining either reading. **Fork:** ratify approver-inclusion as intended (update the contract
wording) or schedule a production change to exclude them (out of tonight's footprint).

### B5 — Advisory (MEDIUM, pre-existing, from A1's G6): approve/reject endpoints have no authz gate

`ApproveSubmissionHandler`/`RejectItemHandler` accept any authenticated user of any role
(`backend/internal/workflow/handler.go:728-753, 793+`). Tonight's ADM matrix cells depend on this
looseness, which is how it surfaced. Candidate for the permission-granularity track (fits the
per-tab-grants go-forward convention). **Fork:** backlog it now vs fold into the gating design work.

## §C — Deferred observations (no decision needed, recorded for the ledger)

- `checklist_submissions.status` defaults `'pending'` and `submitChecklist` never sets it, so
  `requires_approval:false` submissions still read `'pending'` server-side (A1 impl observation,
  pre-existing, unchanged).
- Rejected-field UX quirk: answering a rejected field then reloading visually clears the new answer
  until resubmission (`workflows.html:1544` hydrate branch; A1 G6, LOW, device-local).
- Suite-teardown `loadMyChecklists error: Failed to fetch` log flood from closing contexts (A1 G6,
  LOW-noise; a fetch-abort guard in `sync.js api()` would silence it — would ride B3's card).
- `task test` surfaces Playwright failure as go-task exit **201**, not 1 (A2 G6, INFO — grep the
  Playwright summary line in gate logs, not the exit code name).
- Run-mechanics for future briefs: subagents must never background long runs (A1 lost ~25–30m);
  the compose postgres publishes no host port — scratchpad override with `ports:` is the working
  pattern (used by 5 of 6 agents tonight; consider adding a commented-out ports stanza or a
  documented override file next cycle).
