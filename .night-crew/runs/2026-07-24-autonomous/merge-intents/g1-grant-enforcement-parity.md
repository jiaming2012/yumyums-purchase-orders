# Merge intent — G1 grant-enforcement-parity (run 2026-07-24-autonomous)

Branch: `card/g1-grant-enforcement-parity`

## Red-first evidence part 1 — source enumeration of every `RequirePermission` call site (BEFORE this card)

Enumerated 2026-07-24 from the worktree at branch point (fcea6f8), via
`grep -rn "RequirePermission" backend --include="*.go"`. Excluding the middleware's
own definition (`backend/internal/auth/permission.go`) and its unit tests
(`permission_test.go`), the ONLY production mounts are:

    backend/cmd/server/main.go:583   r.Use(auth.RequirePermission(pool, "inventory-trends", "inventory"))
    backend/cmd/server/main.go:593   r.Use(auth.RequirePermission(pool, "inventory-cost",   "inventory"))

i.e. exactly the two F5 tab gates. `SeedHQApps` (backend/internal/db/db.go)
registers 11 slugs: purchasing, payroll, scheduling, hiring, bi, users,
operations, onboarding, inventory, inventory-trends, inventory-cost.
Backend enforcement before this card: 2 of 11. The card's claim is confirmed
at source level; the failing-test half of the evidence is the without-grant
tests in `tests/grant-enforcement-parity.spec.js` (committed red before the fix).

## Shared files touched (each outside this card's owned wiring, one line of why)

- `tests/workflows.spec.js` — beforeAll baseline: grant `operations` role_grants so its invited non-superadmin users still reach /workflow after gating.
- `tests/sync.spec.js` — same `operations` baseline (sync flows run as invited users over /ws + /workflow/ops).
- `tests/ops-authz-coverage.spec.js` — same `operations` baseline: its team_member fixture must remain "app-granted but unprivileged" so the file keeps guarding the /ops↔REST parity invariant, not the new app gate.
- `tests/ops-save-template-validation.spec.js` — same `operations` baseline, same reason.
- `tests/purchasing.spec.js` — `purchasing` baseline for its invited team_member shopping-flow users.
- `tests/onboarding.spec.js` — `onboarding` baseline for its invited trainee/manager users.
- `tests/users.spec.js` — NFR-3 now establishes its clean-purchasing-perms premise with the same full-replace PUT shape the test already uses (the purchasing.spec baseline had polluted its "fresh team_member sees no purchasing" assert); no assertion weakened.
- `tests/onboarding.spec.js` (second touch) — prove-progress `inviteUser` issues each new user an individual `onboarding` user_grant: roles-less hires (`inviteUser(page, [])`) fall outside any role baseline.
- `features/steps/user-invite-onboarding.js` — Device A setup resets `team_member` out of all role_grants before the scenario enables its two apps; the scenario asserts User B sees EXACTLY 2 tiles and chromium-project baselines had leaked a third.
- `backend/cmd/server/main.go` — the card's owned wiring lives here (handler registration), listed for completeness since main.go is a shared hotspot.

New files (no merge conflict expected): `tests/grant-enforcement-parity.spec.js`,
this note.

## What must survive any merge

- The per-app `RequirePermission` groups in `backend/cmd/server/main.go`
  (operations on /workflow + /ws, inventory on base /inventory + /inventory/recipes +
  /inventory/menu-items, purchasing on /purchasing, onboarding on /onboarding +
  /videos, users on /users admin surface + /apps) — dropping any one silently
  reopens that app's data to ungranted sessions.
- The nesting shape inside `Route("/inventory")`: base endpoints in their own
  gated group, trends/cost groups UNMODIFIED — flattening a `r.Use` to the Route
  level would break F5's tab-grant-only umbrella semantics.
- The GET /inventory/items group mounting `RequirePermission("inventory",
  "purchasing")` — the one deliberate cross-app READ. purchasing.html's init()
  builds the order form from the item catalog with no catch; dropping the
  purchasing umbrella breaks the Purchasing view for purchasing-only crew.
  Item WRITES stay inventory-only.
- The two notification-preference routes staying OUTSIDE the users gate
  (admin-or-self stays handler-internal; NFR-4 depends on self-access).
- The service-token group (period-summary, menu-cogs) staying untouched.
- The baseline-grant beforeAll blocks in the six spec files above — without them
  those suites' invited users are ungranted and the suites redden.
- `tests/grant-enforcement-parity.spec.js` in full (parity guard + per-app pairs).
- `requireReviewAuthz` remaining inside approveSubmission/rejectItem (8c71022) — this
  card layers the app gate on top and must not regress that.

## Safe to drop on merge

- Nothing here is decorative. If a conflict forces a choice, the test baselines
  can be re-derived mechanically (they are one helper + one beforeAll per file),
  but the main.go wiring and the parity spec must land intact.

## Parked (see final report)

- `/api/v1/photos/*` (presign GET/POST, upload) — cross-app utility called by
  workflows.html, purchasing.html, inventory.html, onboarding.html. Which grant
  governs it is genuinely ambiguous (park trigger i). Left at
  authenticated-only (status quo), recorded as a documented exception in the
  parity spec. Operator question in the final report.
