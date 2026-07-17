# DECISIONS-NEEDED — overnight-20260719 (for the operator, morning of 2026-07-19)

> **RESOLVED 2026-07-19 — recorded in `ledger.md` §"Morning-triage resolutions (2026-07-19)" (T-16).**
> Run merged to `dev` `--no-ff` (`a8854c3`). No open forks: §A empty (no PARK), §C resolved at run
> close (operator chose graduate → BACKLOG `suite-isolation-approved-checklist`), §B (the 2 PENDING
> prod KRs) **deferred** to the attended Activity-7 ship step (operator, 2026-07-19). Kept as the
> run's analysis record.

> Forks/exceptions from the cycle-gate closeout that need an operator decision or attended action.
> **This run never decides — it executes** (launch contract). The gate PASSED attested; no card was
> parked (the PARK trigger did not fire). What follows is **not** an improvised fork — it is the
> **planned attended follow-up** the 2026-07-17 "Gate now, prod KRs pending" resolution deferred to
> Activity 7, plus one carried test-hardening call.

## §A — Uncategorized suite reds (Card 1 PARK trigger)

_**none.** Card 1 ran the deterministic stack on an isolated pg16: **Go units exit-0** (all 7 pkgs)
+ **Playwright 450 pass · 1 fail · 0 flaky · 6 skip**. The single red
(`workflows.spec.js › approved checklist … [LST-08 RUN-08]`) is **cross-test DB-pollution** — an
isolation re-run on a fresh single-test DB (`--retries=0`) **greened it (1 passed)**. 0 uncategorized
reds → no PARK. Convergence suite proven zero-flake (39/39 × 3 under `--retries=0`). Evidence:
`reference/cycle-closeout-20260719.md` §1._

## §B — Activity 7 (prod ops) — the attended ship step that flips the 2 PENDING KRs

The gate scored 2 KRs **PENDING** because they cannot attest in a read-only, dev-only gate — they
require the operator-run prod deploy + prod DB mutation. Per the resolved fork, **Activity 7 runs
attended AFTER this gate**, in order (attest-green-before-ship): deploy the attested build, run the
ghost-item cleanup, then a short parity-confirm flips both KRs and **formally closes the milestone**.

### 1. `prod-deploy-parity` → flips **Delivery KR3 (prod parity)**

- **Action (operator-run, never automated):**
  ```
  task prod:deploy      # SSH → git pull → task sw → docker build → restart yumyums-hq
  ```
- **Verify → flip to PASS when true:**
  ```
  task version          # prod backend/frontend == local backend/internal/version/version.go
  ```
  Assert prod `Backend` **and** `Frontend` equal the local `version.go` constants (must include
  `42eeb39` + everything this cycle shipped). If prod is behind, re-run `task prod:deploy`.

### 2. `prod-ghost-item-rename` → flips **QA KR3 (prod ghost catalog item)**

- **Action (prod DB mutation — runs attended or with explicit operator go).** Operator-chosen
  handling (2026-07-16): rename the empty-description item `''` → `(Unnamed — needs review)`,
  **KEEP** its line-item links.
- **Verify → flip to PASS when both hold (run against prod):**
  ```
  SELECT count(*) FROM purchase_items WHERE trim(description) = '';          -- must be 0
  ```
  AND the previously-linked `purchase_line_items` count is **unchanged** (links preserved, not
  orphaned). Both conditions → PASS.

> Recommendation (non-binding): run Activity 7 as the next attended step so the milestone closes
> with a complete scorecard. Once both KRs flip, record the flip in `ledger.md` and the closeout §3.

## §C — Carried test-hardening call — waiver #1's last mile (Eng KR5)

> **✅ RESOLVED 2026-07-19 (operator, attended at run close): chose (a) — GRADUATE the
> test-hardening WO to formally close waiver #1.** Graduated to `BACKLOG.md` as
> **`suite-isolation-approved-checklist`** (§"Waiver-#1 last mile", marked `new` → next-cycle
> `/nc-okr-session` feedstock). It stays a next-cycle scope item; it did not hold this gate. Eng
> KR5 remains PARTIAL for the 2026-07-19 gate of record; it flips to PASS when the WO lands literal
> `task test` exit-0.

**What.** Waiver #1 (`task test` exit-0) is **substantially retired (38 reds → 1)** but **not
formally**: the one remaining Playwright red — `workflows.spec.js › approved checklist shows Approved
badge and cannot be resubmitted [LST-08 RUN-08]` — is a **cross-test DB-pollution** artifact (it
`#toast`-asserts, greens in isolation, fails only in the full shared-`hq_test` suite). It is a
**test-isolation defect, not a product defect**.

**Decision for the operator (next-cycle scope — does NOT hold this gate):**
- **(a) Graduate a small test-hardening WO** — isolate/repair the `approved checklist` test's state
  dependency (some earlier spec leaves approval/`#toast` state in shared `hq_test`) so `task test`
  reaches **literal exit-0** and **formally retires waiver #1**. *(Recommended — cheap, well-scoped,
  and it closes the waiver for good.)*
- **(b) Accept the substitute and carry** — the substitute criterion "0 new uncategorized reds vs
  baseline" is met; log the pollution red and keep waiver #1 carried, reduced.

Either way: this does **not** block the gate, the merge, or Activity 7.

---

_No other forks. No operator-only questions arose during execution. The two invalid suite attempts
(foreign-server reuse; unmigrated DB) were harness-provisioning defects the run corrected and
documented (`suite-logs/attempts/`) — not decisions._
