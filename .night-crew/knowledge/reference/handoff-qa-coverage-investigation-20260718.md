# Handoff — QA coverage investigation: why five escaped defects slipped, and whether our agents could have caught them

**Audience:** an investigating agent (fresh context).
**Status:** investigation brief. All five defects below are already **fixed and committed to `dev`** (see commit table). Your job is NOT to fix them — it is to explain the *coverage gap* and produce a capability-vs-gap conclusion.

---

## 0. Mission (what you must produce)

Produce a report, `reference/qa-coverage-findings-20260718.md`, that answers:

1. **Why did the night-crew QA process miss all five defects?** Ground this in what the process actually tests (Section 4), not speculation.
2. **Could the `ui-jury` pipeline (cartographer → driver → critics → verifier) have found them** — (a) as currently configured, and (b) in principle if reconfigured? Answer per-defect (Section 5-6).
3. **Capability vs gap conclusion.** The operator's thesis: *"there are many states in the app that are never looked at until I manually go through different roles, sub-lists, and combinations."* Confirm or refute this with evidence, and enumerate exactly which **state dimensions** no current agent tool reaches. This is the deliverable that matters most.
4. **Recommendations** for making the missing states auto-discoverable/auto-drivable by agents, ranked by leverage.

Do the investigation read-only. Do not modify product code or tests.

---

## 1. The five escaped defects (all found by manual operator play on `dev`, 2026-07-17→18)

| # | Commit | Symptom (operator-visible) | Root cause | State dimension it lived in |
|---|---|---|---|---|
| 1 | `5c423ac` | Editing a checkbox on device A didn't sync to device B (both logged in as the operator, a **superadmin**). | `ResolveEntityAccess` fanned live ops only to a checklist's **assignees**; a superadmin editing a checklist they can view but aren't assigned to was excluded. | **Cross-user access** (editor ∉ assignees) × **cross-device** |
| 2 | `93c74a4` | (a) A manager's **rejection reason** never reached the submitter's other device live; (b) an **observer's list count** stayed frozen at the pre-rejection number. | `applyOp` handled `REJECT_ITEM`/`APPROVE_ITEM` by refreshing only the *Approvals tab* (a no-op for non-approvers); the client kept rendering from a **stale `MY_SUBMISSIONS` cache**. | **Op-type** (submission-lifecycle ops, not field edits) × **cross-device** × **derived views** (banner, count) |
| 3 | `08ebb4f` | Rejecting a **sub-step** ("Cut the check → Do B") stored the comment + photo requirement but showed them **nowhere**. | Correction banner rendered only at the **parent** field level; the hydrate field-id filter (`tplFieldIds`) never included sub-step ids. | **Nesting** (sub-step vs top-level field) × **rejection state** |
| 4 | `9b1aa32` | A non-photo field (checkbox) rejected with **"require photo"** showed "Photo required" with **no way to attach a photo**, and the resubmit gate then blocked the crew (dead-end). Plus a latent **hard-block**: a sub-step require-photo made resubmit impossible. | The resubmit gate demanded the field's *value* be a photo URL — impossible for a checkbox. No slot existed for a correction photo separate from the answer. Sub-step ids matched the backend gate but were unsatisfiable. | **Field-type × rejection-state** combination; **nesting**; **new data-entry path** |
| 5 | `b224c79` | A rejected **sub-step** came back **still checked** (top-level rejected fields uncheck correctly). | The uncheck logic (`delete FIELD_RESPONSES[rej.field_id]`) only clears top-level state; a sub-step's checked-state lives in the *parent's* `sub_steps` map, which it never touched. | **Nesting** × **rejection state** |

Full narrative + prior QA-gap analysis: `reference/qa-gap-20260717-live-sync-access.md` (defects 1-2) and its **§ 2026-07-18 addendum** (defects 2-5). `BACKLOG.md` §"Escaped defect + QA gap" carries the graduated work-orders.

---

## 2. The common structure (the whole point)

Every one of these bugs lived in an **untested cell of a multi-dimensional state space**. The dimensions:

- **A. Actor role** — superadmin / admin / manager / team_member (config-superadmin is derived at auth; DB roles are `admin|manager|team_member`).
- **B. Assignment relationship** — is the actor an assignee? an approver? neither-but-can-view (admin)? 
- **C. Observer** — a *second* device/session watching while the actor acts (live propagation).
- **D. Op type** — field edit (`SET_FIELD`) vs submission-lifecycle (`SUBMIT`/`APPROVE_ITEM`/`REJECT_ITEM`).
- **E. Nesting depth** — top-level field vs **sub-step** (a checkbox with children; sub-steps are real `checklist_fields` rows with `parent_field_id`, but their answer-state lives inside the parent's response value, not their own).
- **F. Field type × rejection modifier** — checkbox/text/yes-no/temperature/photo × rejected-with-comment × rejected-with-require-photo.
- **G. Derived view** — the thing that must update: the correction banner, edit-vs-readonly mode, the My-Checklists progress **count**, the checkbox **checked-state**, the Approvals queue.

The existing automated suite covers **A(one), one actor, no observer, D(field-edit only), E(top-level mostly), F(field-type only), G(field value only)**. Every escaped bug is at least one axis outside that box. The operator found them precisely because manual play naturally varies A, C, E, F, G at once.

---

## 3. Reproduction & environment (so you can re-derive if needed)

- **Test stack:** Playwright E2E in `tests/`, Go units in `backend/internal/`. Config `playwright.config.js` defaults to `localhost:5432` — **do NOT use that**; on this box `:5432` is an unrelated always-on Postgres. Use an **isolated pg16**: container `yumyums-e2e-pg` (may still be running; `docker port yumyums-e2e-pg 5432/tcp` for its host port), and run with `CI=1 DB_HOST=127.0.0.1 DB_PORT=<port> TEST_PORT=8231 npx playwright test …`. Go units: `DB_TEST_URL=postgres://yumyums:yumyums@127.0.0.1:<port>/hq_test?sslmode=disable go test ./internal/workflow/`.
- **Regression tests that encode these bugs** (read them — they are the executable spec of each state combo):
  - `tests/sync.spec.js`: `FLD-LIVE-01/02`, `RJT-LIVE-01/02/03` (cross-device reject/approve), the **W-3 convergence matrix** (~line 1011-1300).
  - `tests/workflows.spec.js`: `APR-REPRO-0718`, `APR-DEADEND-0718`, `APR-SUBSTEP-0718`, `APR-SUBSTEP-UNCHECK`.
  - `tests/persistence.spec.js`: `FLD-CORRECTION-PHOTO`.
  - `backend/internal/sync/access_test.go`; `backend/internal/workflow/resubmit_photo_gate_test.go`.

---

## 4. What the night-crew QA process actually does (read before concluding)

Read these and characterize the *shape* of coverage:

- **`CLAUDE.md` → "Definition of Done"**: it *requires* a **State Enumeration Table** in every UI-SPEC (empty/loading/error/success + ≥2 edge rows), a **self-verification screenshot ritual**, and a **verifier subagent gate**. Investigate: the enumeration is authored **per-phase, by hand**, scoped to the phase's component. Nothing enumerates the **cross-cutting** combinatorial space (roles × lifecycle × nesting) that spans phases. That is the structural blind spot.
- **The W-3 convergence matrix** (`tests/sync.spec.js` ~1011+): note it is a **field-TYPE** matrix (checkbox/yes-no/temp/text/sub-step/photo each converge for *one* assignee editing their *own* checklist). It varies F but holds A, B, C, D fixed. The qa-gap addendum already recommends adding an **op-type axis** and a **cross-user access axis**.
- **`reference/qa-gap-20260717-live-sync-access.md`** (+ addendum): the prior root-cause of the QA miss — "every convergence test drives the assignee editing their own checklist," "the sync package had zero Go tests," "the matrix was field-type not access/op-type." Build on this; don't repeat it.

---

## 5. What `ui-jury` actually is, and how it's configured here (read before concluding)

`ui-jury` is a **visual/runtime review pipeline**, not a behavioral/state-coverage tool. Pipeline: **cartographer** (discover routes/SPA-nav via bounded BFS) → **seeder** (group states by fixture) → **driver** (capture per-state bundles: desktop+mobile screenshot/DOM/a11y, console, network, backend log) → **critics** (`hierarchy`, `mobile`, `correctness`) → **verifier** (score/keep-drop) → **annotator**.

Read:
- `docs/ui-jury.md` — the project's own run doc + **caveats** (quote these in your report).
- `.ui-jury/routes.template.yaml` — the **actual declared surface**.
- `.ui-jury/hooks.yaml` — fixtures.
- Agent defs in `~/.claude/agents/ui-jury-*.md` (esp. `ui-jury-cartographer.md`, `ui-jury-driver.md`, `ui-jury-critic-correctness.md`) — for the true capability of each stage, including the cartographer's **Step 2.5 multi-fixture impact observation** (it *can* run `db:reset`/`db:seed` hooks) and its BFS of buttons/tabs/menuitems.

**Current hq configuration (the crux of "as-configured"):**
- **7 pages, one `at-rest` state each.** No lifecycle states declared.
- **One identity** — `jamal@yumyums.kitchen` (a **superadmin**). No manager/team_member pass.
- **Default tab only** (docs caveat: `workflows.html` captured on "My Checklists"; Approvals/Builder not captured; deeper interactions deferred to "v2").
- **No `db_seed` map** (hooks.yaml declares only `db_reset`, which TRUNCATEs *inventory* tables and **preserves** users/templates). So **no submitted / rejected / approved / rejected-with-photo** state is ever seeded.
- **Single browser session** — the driver walks routes in one context; there is no second observer session.
- **Critics judge visual hierarchy, mobile layout, and runtime errors** (console/network/backend NDJSON) — not "this checklist *should* show a rejection comment and doesn't." There is no **behavioral/expected-content oracle**.

---

## 6. The central question — map each defect against each tool

Build this matrix (fill "reachable?" = could the tool *arrive at* the state; "detectable?" = could it *flag the defect* once there). Justify each cell from Sections 4-5.

| Defect | night-crew E2E matrix | ui-jury cartographer (discover) | ui-jury driver (capture, as-configured) | ui-jury critics (detect) |
|---|---|---|---|---|
| 1 cross-user live sync | ? | ? | ? | ? |
| 2 reject/approve stale cache | ? | ? | ? | ? |
| 3 sub-step banner missing | ? | ? | ? | ? |
| 4 require-photo dead-end | ? | ? | ? | ? |
| 5 sub-step stays checked | ? | ? | ? | ? |

Anchor your reasoning to these axes (state them explicitly as the gap taxonomy):

- **Multi-identity gap** — does *any* current tool assume more than one concurrent role/identity? (ui-jury: one; convergence matrix: one; cartographer crawl: one session.)
- **Cross-device / live-propagation gap** — bugs 1-2 are *only* visible with a **second session observing while the first acts**. A single-session capture is blind to op fan-out by construction.
- **Seeded-lifecycle gap** — bugs 2-5 require a checklist in a **submitted→rejected(→with photo)** state. With no `db_seed` fixtures, neither the driver nor a naive BFS crawl can *reach* those states (they aren't reachable by clicking from a clean DB in one session; they need an approver acting on a submitter's submission).
- **Nesting / sub-list gap** — bugs 3-5 are **sub-step** states. Does the cartographer's BFS descend into a checklist field's expanded sub-steps? Does anything enumerate "rejected sub-step"? (The operator's "sub-lists" concern.)
- **Oracle gap** — even if a bad state were captured, the critics look for *visual/runtime* problems. A silently-missing comment or a frozen count is a **semantic** defect with no console error and a plausible-looking screenshot. What oracle would catch it?

---

## 7. Desired conclusion shape (what "done" looks like for this investigation)

Your report should land on:

1. **A confirmed gap taxonomy** (the five axes above, or a better one), each with: which tools cover it today (likely none for cross-device/seeded-lifecycle/oracle), and which defect(s) prove the gap.
2. **A crisp verdict on ui-jury**: it is a *visual regression* tool over a *declared, single-actor, default-tab, unseeded* surface — so as-configured it could not have found any of the five; and even reconfigured, it lacks a **multi-session** capability and a **behavioral oracle**, so it could at best reach (not detect) defects 3-5, and could not reach 1-2 at all. Confirm or correct this hypothesis with evidence from the agent defs.
3. **Recommendations, ranked.** Candidate directions to evaluate (you decide which have leverage):
   - A **state-model matrix generator** for the checklist engine: enumerate {role} × {assignment} × {op-type} × {field-nesting} × {observer} × {derived-view} and emit a driveable plan — the automation of what the operator does by hand. (The BACKLOG already has a "cross-user × op-type live-sync matrix + sync-package unit coverage" WO; assess whether that is the right home.)
   - A **two-session / cross-device harness** as a first-class capability (the convergence tests already do this ad hoc with two Playwright contexts — could it be generalized so *every* mutation is auto-checked on a second observer?).
   - **`db_seed` lifecycle fixtures** for ui-jury (submitted, rejected-with-comment, rejected-with-photo, approved-with-feedback) × per-role login passes, so the driver *reaches* these states and the mobile/hierarchy critics at least see them.
   - **Behavioral oracles** (expected-content assertions) distinct from the visual critics — e.g., a "rejection feedback must render for a rejected field/sub-step" invariant.
   - **Cartographer depth**: sub-step expansion + multi-role crawl + `db_seed` impact observation, so route/state discovery itself surfaces the combinatorial surface.
4. **An honest "residual" note**: which classes of bug remain economically un-auto-findable even after the above, and why (so the operator knows where manual play is still the backstop).

---

## 8. Files to read (complete list)

- `reference/qa-gap-20260717-live-sync-access.md` (+ addendum) — prior root-cause of the miss.
- `BACKLOG.md` §"Escaped defect + QA gap" — the graduated WOs.
- `CLAUDE.md` — "Definition of Done", "State Enumeration Table", "Verifier subagent gate", the persistence rule, the bug-fix protocol.
- `docs/ui-jury.md`, `.ui-jury/routes.template.yaml`, `.ui-jury/hooks.yaml`, `.ui-jury/scripts/`.
- `~/.claude/agents/ui-jury-{cartographer,driver,seeder,critic-correctness,critic-hierarchy,critic-mobile,verifier,annotator}.md`.
- Product: `workflows.html` (the checklist engine — `hydrateFieldState`, `applyOp`, `renderRunnerField`, `renderApprovals`, the reject/approve handlers), `sync.js` (`applyOp`, `submitOp`), `backend/internal/sync/ops.go` (`ResolveEntityAccess`), `backend/internal/workflow/handler.go` (`validateResubmitPhoto`, `hasResubmitPhoto`).
- Tests listed in Section 3 — the executable spec of the fixed state-combos.

## 9. Guardrails

- Read-only investigation. No product/test edits.
- `git log --oneline 5c423ac~1..b224c79` is the exact set of fixes; `main` is untouched (`b89c202`), all five are on `dev`.
- The commits' own messages contain precise root-cause detail — use them.
- Don't re-litigate the fixes; the question is **coverage**, not correctness.
