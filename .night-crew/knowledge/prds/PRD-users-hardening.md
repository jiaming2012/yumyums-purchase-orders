# PRD — Users app hardening

> **Cycle:** HQ hardening — first night-crew guinea-pig run.
> **App:** Users — `users.html`, `/api/v1/users/*` + `/api/v1/apps/*` + `/api/v1/auth/{invite-info,accept-invite}`, Go + Postgres.
> **Depth:** *Enumerate + mark only.* This PRD is the honest flow map with per-flow
> status and a falsifiable definition of "working." It does **not** fix anything —
> each BROKEN/UNPROVEN flow becomes a work order after sign-off.
> **Role of this doc:** one of the four app PRDs copying the shape of the signed
> `PRD-operations-hardening.md` exemplar.
> **Enumeration provenance (for the ≥90% recall KR):** flows were derived by reading
> `users.html` (2-tab layout — Users / Access — ~665 lines), `tests/users.spec.js`
> (~18 tests) + `tests/multi-role.spec.js` (~9 tests), and the Go layer under
> `backend/internal/users/` (`db.go` + `handler.go`) plus the router
> (`backend/cmd/server/main.go:419-469`), then **cross-checked by a second
> independent pass** angled at **auth/permission enforcement** — invite-token
> lifecycle, role-tier refusals, grant persistence — the backend-only behaviors the
> UI-first pass cannot see. Recall is measured post-build as
> `enumerated ÷ (enumerated + discovered-during-WO-build)`.
> **Scope note:** the card scoped Users as "4 endpoints" (`/users`, `/users/`,
> `/users/invite`, `/users/{id}`). That is the UI-visible subset. Pass-2 endpoint
> reading surfaces **12 Users-owned endpoints** (§Verification). A flow is in-scope
> if a real person (admin/owner, invited crew member, or self) triggers it, OR if an
> auth tier gates it. No sub-selection on "critical" — the whole set + the
> enforcement cross-check is what keeps recall honest.

## Objective

Produce a written, agreed enumeration of every end-to-end flow in the Users app —
from an admin listing the team, inviting a crew member and watching the invite-token
lifecycle, through editing roles and per-app grants, resetting passwords, forcing
logout, and deleting users, down to the permission tiers that gate each of those —
with every flow honestly marked **WORKING**, **UNPROVEN**, or **BROKEN** and backed
by a falsifiable definition of "working." This advances the **Product objective**:
KR-1 (5/5 apps have a hardening PRD enumerating their E2E flows) and KR-2
(enumeration recall ≥ 90%). Its status tally is the denominator the **Engineering**
(0 known-broken flows at cycle end) and **QA** (every flow has a real, asserting
test; vacuous tests → 0) objectives grade against.

## Operators & users

- **Admin / owner (admin or superadmin)** — the only tier the Users tool is built
  for. Lists the team, invites crew (name + email + roles + employee type +
  salary), edits a user's roles / nickname / alert channels / timezone, resets
  passwords, force-logs-out sessions, deletes users, and manages per-app role +
  individual-user access grants in the Access tab. Every `/users/*` and `/apps/*`
  admin endpoint refuses non-admins with 403 (`isAdmin` at `handler.go:32-34`).
- **Invited crew member (status='invited', unauthenticated at accept time)** —
  receives an invite link (`/login.html?token=…`), the accept-invite page
  personalizes with their first name, and they set a password to activate. Also the
  target of reset-password and reinvite.
- **Self (any authenticated user)** — the notification-preference endpoints are the
  one Users surface gated **admin-or-self** (`handler.go:508`, `:536`), not
  admin-only — a user may read/set their own alert channels.
- **The overnight crew (indirect)** — consumes this enumeration + status as the
  work-order backlog; a flow marked BROKEN/UNPROVEN here is a candidate WO.

## Requirements

Status legend: **WORKING** = flow works E2E *and* a test drives it and asserts
observable DB/UI state · **UNPROVEN** = flow appears to work in code but no test
verifies the behavior it names (missing or vacuous test) · **BROKEN** = flow is
**confirmed** incomplete/stubbed by code inspection, or a test reveals it does not do
what it claims. Every requirement traces to an OKR key result.

**BROKEN is reserved for *confirmed* breakage only** — a broken *flow*, or a test
that reveals a feature defect. A flow merely *suspected* broken, or one whose only
failing test can't even run, stays **UNPROVEN**. So there are **0 BROKEN** here: the
Access-tab tests (FR-16, FR-17) target a DOM (`#t3`/`#s3`) that the 2-tab refactor
removed, so they can't run to the flow and reveal no defect — the feature itself
renders correctly into `#s2`. Both are UNPROVEN (stale test), and their repair is a
test-only WO.

### Tab 1 — Users (list + invite + edit + destructive actions)

- **FR-1 (list team)** — The system lists all users ordered by display name, each
  row showing display name + a role/status pill (Invited / Super Admin / Admin /
  Manager / Team Member), with a "Reinvite" button on invited users and an empty
  state ("No team members yet") when the list is empty. *(GET `/api/v1/users`;
  `handler.go:37`, `db.go:90`, `users.html:192-215`)* — **WORKING** (`User List`
  tests drive load + assert a known user renders; `multi-role` asserts `roles` is an
  array per row) — traces to Product KR-1, QA KR-2.
- **FR-2 (skeleton + error states)** — The list shows skeleton rows while loading and
  an inline "Could not load team." error on fetch failure.
  *(`users.html:153-163, 208-214`)* — **WORKING** for skeleton (a test delays the
  route and asserts skeletons appear then get replaced); **UNPROVEN** for the error
  branch (no test forces a failed `/users` fetch) — traces to QA KR-1, QA KR-2.
- **FR-3 (invite crew member)** — An admin fills the invite form (first, last, email,
  roles, employee type W2/1099, hourly salary), submits, and the system creates a
  `status='invited'` user, mints an opaque invite token (7-day expiry), returns an
  `invite_path` (`/login.html?token=…`), and shows the invite-link panel. *(POST
  `/api/v1/users/invite`; `handler.go:60`, `db.go:245`, `db.go:360`,
  `users.html:265-295`)* — **WORKING** (invite tests drive the form and assert the
  panel + `/login.html?token=` link + the user appearing "Invited"; `multi-role`
  asserts multi-role round-trips) — traces to Product KR-1, QA KR-2.
- **FR-4 (invite validation + duplicate email)** — Empty first/last/email is refused
  (422 `validation_error`); a duplicate email is refused (409 `email_already_exists`).
  *(`handler.go:80-83, 94-98`; frontend guard `users.html:276`)* — **UNPROVEN**
  (neither the 422 nor the 409 path is asserted by any test) — traces to
  Engineering KR-1, QA KR-1.
- **FR-5 (invite-link panel + copy)** — The invite-link panel shows a prominent
  "email has been sent" notice *above* the link textarea, and a Copy button that
  copies the URL and flips to "Copied!". *(`users.html:297-307, 559-568`)* —
  **WORKING** (a test asserts the copy button flips text; another asserts notice
  DOM-order precedes the textarea) — traces to Product KR-1, QA KR-2.
- **FR-6 (reinvite from list)** — Tapping "Reinvite" on an invited-user row issues a
  fresh reset-password token and shows a "resent" toast, caching the URL on the user
  object. *(POST `/api/v1/users/{id}/reset-password`; `users.html:574-592`)* —
  **UNPROVEN** (no test taps the row-level Reinvite button or asserts the toast /
  new token) — traces to QA KR-1.
- **FR-7 (edit profile — roles + fields)** — An admin opens a user, edits first /
  last / nickname / roles (multi-select chips) / alert channels / timezone, saves
  via PATCH, and the list reflects the change. *(PATCH `/api/v1/users/{id}`;
  `handler.go:151`, `db.go:266`, `users.html:392-421`)* — **WORKING** (`Edit User`
  drives a name edit and asserts the updated name in the list; `multi-role` asserts
  a role PATCH returns the updated array; a test asserts role chips render 3 options)
  — traces to Product KR-1, QA KR-2.
- **FR-8 (nickname collision)** — Saving a nickname that collides with another user's
  nickname or derived display name is refused (409 `nickname_taken`) and shows an
  inline field error. *(`handler.go:174-189`, `db.go:339`, `users.html:414-418`)* —
  **WORKING** (a test seeds a nickname on user1, tries it on user2, asserts the
  "taken" inline error) — traces to Engineering KR-1, QA KR-2.
- **FR-9 (email is immutable)** — The edit form renders the email field read-only
  ("Email cannot be changed"); the PATCH handler has no email field. *(`users.html:360`,
  `handler.go:161-168`)* — **UNPROVEN** (readonly attribute present in markup, but no
  test asserts email cannot be changed, and no server-side rejection path exists to
  test — it is simply absent from the update surface) — traces to QA KR-1.
- **FR-10 (alert-channel defaults + validation)** — A new user defaults to both
  channels (`zoho_cliq` + `email`); the edit form requires ≥1 channel selected
  (client `alert` + server 400 on empty / invalid channel). *(`users.html:398-400`,
  `db.go:171-188, 292-305`)* — **WORKING** for the default (a test asserts the API
  returns both channels and both chips render "on"; a test toggles chips);
  **UNPROVEN** for the server-side empty/invalid-channel 400 (no test posts an empty
  channel set) — traces to Product KR-1, QA KR-1.
- **FR-11 (timezone edit)** — The edit form offers a timezone select; PATCH validates
  the value against IANA (`time.LoadLocation`) and 400s an invalid zone.
  *(`users.html:370-381`, `db.go:306-314`)* — **UNPROVEN** (no test changes timezone
  or asserts the invalid-zone 400) — traces to QA KR-1.
- **FR-12 (reset password)** — An admin taps "Reset Password" on a user, which mints
  a fresh 7-day token and shows the "Password Reset Link" panel; the email subject
  differs for invited vs active users. *(POST `/api/v1/users/{id}/reset-password`;
  `handler.go:221`, `users.html:423-435`)* — **WORKING** (a test drives the button
  and asserts the "Password Reset Link" panel + `/login.html?token=` link) — traces
  to Product KR-1, QA KR-2.
- **FR-13 (force logout / revoke sessions)** — An admin force-logs-out a user, which
  deletes all their sessions (204) and shows a "revoked" toast. *(POST
  `/api/v1/users/{id}/revoke`; `handler.go:280`, `users.html:437-448`)* —
  **WORKING** (a test confirms the dialog, clicks force-logout, asserts the "revoked"
  toast) — traces to Engineering KR-1, QA KR-2.
- **FR-14 (delete user)** — An admin deletes a user (confirm dialog → DELETE → 204,
  sessions cascade); the row disappears from the list. *(DELETE `/api/v1/users/{id}`;
  `handler.go:302`, `db.go:422`, `users.html:450-463`)* — **WORKING** (a test
  confirms the dialog, deletes, asserts the specific row is gone;
  `multi-role` deletes via API as cleanup) — traces to Product KR-1, QA KR-2.
- **FR-15 (get-invite-link inline)** — On an invited user with no cached URL, "Get
  Invite Link" mints a reset token and renders the link inline in the edit card.
  *(POST `/api/v1/users/{id}/reset-password`; `users.html:593-610`)* — **UNPROVEN**
  (no test exercises the inline get-invite-link path) — traces to QA KR-1.

### Tab 2 — Access (per-app role + user grants)

- **FR-16 (view app permissions)** — The Access tab lists every enabled app with
  role toggles (Admin / Manager / Team Member), the set of individually-granted
  users as removable chips, and a picker to add more. *(GET
  `/api/v1/apps/permissions`; `handler.go:432`, `db.go:431`,
  `users.html:465-496`)* — **UNPROVEN (stale test — no assertion reaches the flow)** —
  the code path renders correctly into `#s2` (tab 2) via `renderAccess()`
  (`users.html:466`; `#t2`/`#s2` are the live Access tab, line 103/121), **but the two
  Access-tab tests target a stale DOM**: they `page.click('#t3')` and wait for `#s3`
  to hold `.access-card` (`users.spec.js:464-467, 484-488`). In the current
  `users.html` there is **no `#t3` button** (only `#t1`/`#t2`, line 102-104) and `#s3`
  is an empty hidden div (line 122). These tests exercise a layout that was refactored
  out — they cannot even run to the toggle, so they reveal *no* feature defect (per
  the legend, that keeps this UNPROVEN, not BROKEN). Test-repair pointer: repoint the
  navigation to `#t2`/`#s2` and assert the rendered app cards + toggles, then this can
  reach WORKING. — traces to QA KR-1, Engineering KR-2.
- **FR-17 (toggle role grant)** — Toggling a role switch on an app immediately
  persists the full grant set via PUT and the toggle reflects the new state.
  *(PUT `/api/v1/apps/{slug}/permissions`; `handler.go:455`, `db.go:463`,
  `users.html:619-628, 498-506`)* — **UNPROVEN (stale test — no assertion reaches the
  flow)** — same root cause as FR-16: the "can toggle a role permission" test
  navigates via `#t3` and `#s3` (`users.spec.js:484-488`), which no longer exist. The
  persistence path (optimistic push → `savePermissions` → PUT → `SetAppPermissions`
  transaction) is real; the stale test can't reach the toggle, so it reveals no
  feature defect — UNPROVEN, not BROKEN. Test-repair pointer: repoint to `#t2`/`#s2`,
  toggle a role, and assert the PUT-persisted state survives a reload. — traces to
  QA KR-1, Engineering KR-2.
- **FR-18 (add individual user grant)** — Selecting a user in the app's picker and
  tapping Add appends them to `user_grants`, persists via PUT, and re-renders the
  card with the new chip. *(`users.html:629-639`, `db.go:463-506`)* — **UNPROVEN**
  (no test adds an individual grant or asserts the persisted chip; also unreachable
  via the broken Access-tab navigation) — traces to QA KR-1.
- **FR-19 (remove individual user grant)** — Tapping the × on a granted-user chip
  removes them from `user_grants`, persists via PUT, and re-renders.
  *(`users.html:640-648`)* — **UNPROVEN** (no test removes a grant or asserts the
  removal persists) — traces to QA KR-1.

### Invite-token lifecycle (accept-invite — the non-admin surface)

- **FR-20 (invite-info personalization)** — The accept-invite page fetches the first
  name for a valid token so it can greet the invitee ("Welcome, Welcome"); an
  expired/used token returns 400 `token_expired` and the page shows an error.
  *(GET `/api/v1/auth/invite-info?token=`; `handler.go:324`, `db.go:510`,
  `login.html`)* — **WORKING** (a test hits the accept-invite URL and asserts the
  welcome heading; another asserts an invalid token shows an "expired" error) —
  traces to Engineering KR-1, QA KR-2.
- **FR-21 (accept invite → activate + session)** — The invitee sets a password;
  the server atomically claims the token (marks `used_at`, checks unexpired),
  activates the user (`status='active'`, stores password hash + accepted_at + optional
  nickname/pos/cashapp/phone), creates a session cookie, and redirects to index.
  *(POST `/api/v1/auth/accept-invite`; `handler.go:350`, `db.go:373` (`ClaimInviteToken`),
  `db.go:403` (`ActivateUser`))* — **WORKING** (a test sets a password on a real
  invite and asserts redirect to index) — traces to Engineering KR-1, QA KR-2.

### Cross-cutting (auth/permission enforcement — the second-pass blind spot)

- **NFR-1 (admin-only tier on all admin endpoints)** — Every `/users/*` (except
  notification-preference) and both `/apps/*` endpoints refuse a non-admin caller
  with 403 `forbidden`. *(`isAdmin` guard repeated at `handler.go:40, 63, 154, 224,
  283, 305, 435, 458`)* — **UNPROVEN (priority)** — *no test drives any admin
  endpoint as a non-admin and asserts 403.* Confirm-absence step: grep the two spec
  files for a non-admin session — none exists (both log in as the superadmin only).
  This is the single largest untested surface: 8 handlers share one guard and none
  is negatively tested. — traces to Engineering KR-1, QA KR-1.
- **NFR-2 (invite-token 7-day expiry + single-use)** — Invite/reset tokens expire
  after 7 days (`InsertInviteToken(…, 7)`) and are single-use: `ClaimInviteToken`
  only claims a token where `used_at IS NULL AND expires_at > now()`, atomically
  setting `used_at`. A second claim of the same token, or a claim after expiry,
  returns `ErrTokenInvalid` → 400 `token_expired`. *(`db.go:360-390`,
  `handler.go:368-377`)* — **UNPROVEN (priority)** — the *invalid-token* path is
  tested (FR-20's expired-token test), but the **single-use** guarantee (accept
  twice → second refused) and the **expiry boundary** (token past `expires_at`) have
  no asserting test. Confirm-absence step: grep for a double-accept or an
  expiry-boundary seed — none found. — traces to Engineering KR-1, QA KR-1.
- **NFR-3 (grant persistence across tools)** — Role + user grants written via PUT
  `/apps/{slug}/permissions` are what `me/apps` (and the launcher gate) reads to
  decide which tools a user sees; `SetAppPermissions` replaces the set in one
  transaction (DELETE-all + INSERT). — **UNPROVEN** (the round-trip "grant here →
  tool appears/disappears for that user in `/me/apps`" is never tested; the toggle
  tests that would even touch the write are themselves stale — FR-17) — traces to
  Engineering KR-1, QA KR-1.
- **NFR-4 (notification-preference admin-or-self tier)** — The GET/PUT
  notification-preference endpoints are the one Users surface gated *admin-or-self*
  (`!isAdmin && caller.ID != targetID → 403`), not admin-only, and validate ≥1 valid
  channel. *(`handler.go:500-564`)* — **UNPROVEN** (neither the self-allow nor the
  other-user-refuse branch is tested; the frontend does not even call these — it uses
  the PATCH `notification_pref` field instead, so this endpoint pair is
  backend-only and entirely unexercised) — traces to Engineering KR-1, QA KR-1.
- **NFR-5 (401 → login redirect)** — Any API call returning 401 mid-session
  redirects to `/login.html`. *(`users.html:139`)* — **UNPROVEN** (no test forces a
  401 and asserts the redirect) — traces to QA KR-1.

## Additional flows — second-pass enforcement cross-check

The independent second pass (angled at auth/permission enforcement — the
backend-only behaviors the UI-first pass could not see) added the flows the first
pass missed. The empirical basis for the recall note in §Success metrics:

- **FR-20, FR-21** (invite-token lifecycle on the *unauthenticated* accept surface) —
  the UI-first pass, reading `users.html`, sees only the admin side of inviting;
  the accept-invite handler lives in the same Go package but a different page
  (`login.html`). Second pass surfaced both.
- **NFR-1** (403 refusals across all 8 admin handlers) — invisible to a UI-first
  pass that only ever logs in as an admin.
- **NFR-2** (token single-use + expiry boundary) — a DB-transaction guarantee in
  `ClaimInviteToken`, not a UI behavior.
- **NFR-4** (notification-preference admin-or-self tier + the fact the frontend
  never calls it) — a backend-only endpoint pair the UI-first pass has no path to.
- The **endpoint undercount** itself: the card scoped "4 endpoints"; the second-pass
  router read (`main.go:453-469`) surfaced **12 Users-owned endpoints** (§Verification).

## Acceptance criteria

Surface-anchored, Given/When/Then. These define "working" for representative flows;
every enumerated flow inherits the pattern of *drive-the-real-flow +
assert-observable-state* (the WORKING bar).

- **AC-1 (FR-3, invite):** *Given* an admin on the Users tab, *When* they fill the
  invite form and submit, *Then* the invite-link panel shows a `/login.html?token=`
  URL, a `status='invited'` user row exists in the DB, and the row renders "Invited"
  in the list.
- **AC-2 (FR-4, duplicate email — UNPROVEN):** *Given* an existing user with email X,
  *When* an admin invites another user with email X, *Then* the request returns 409
  `email_already_exists` and no second row is created.
- **AC-3 (FR-8, nickname collision):** *Given* user A with nickname "Jay", *When* an
  admin sets user B's nickname to "Jay", *Then* PATCH returns 409 `nickname_taken`
  and the edit form shows the inline "taken" error.
- **AC-4 (FR-14, delete):** *Given* a user in the list, *When* an admin confirms
  delete, *Then* DELETE returns 204, the user's sessions are gone, and the row no
  longer renders.
- **AC-5 (FR-16/FR-17, Access tab — stale-test UNPROVEN):** *Given* an admin on the Access
  tab, *When* they toggle a role switch on an app, *Then* the toggle persists via PUT
  and reflects the new state on reload. *(The existing tests navigate via `#t3`/`#s3`
  — which do not exist in the current 2-tab layout — so they cannot reach the toggle;
  the WO must repoint them to `#t2`/`#s2` before this can be proven.)*
- **AC-6 (NFR-1, admin gate — UNPROVEN priority):** *Given* a non-admin
  (team_member) session, *When* it calls `GET /api/v1/users` (or any admin endpoint),
  *Then* the response is 403 `forbidden` and no data is returned. *(A red-first test
  must show the refusal is real, not incidental.)*
- **AC-7 (NFR-2, token single-use — UNPROVEN priority):** *Given* a valid invite
  token, *When* accept-invite is called twice with it, *Then* the first activates the
  user and the second returns 400 `token_expired`, and only one activation occurred.
- **AC-8 (FR-21, accept invite):** *Given* a valid invite link, *When* the invitee
  sets a password, *Then* the user is `status='active'` with a stored hash + session
  cookie, and the browser lands on index.

## Verification plan

- **Environment:** localhost Postgres (`brew postgresql@16`) — the E2E suite
  requires a local DB; the remote Windows DB is too slow. Playwright blocks service
  workers (`serviceWorkers: 'block'`). All existing tests log in as the superadmin
  (`jamal@yumyums.kitchen`) — **no non-admin fixture exists**, which is precisely why
  NFR-1 is untested.
- **Suites:** `tests/users.spec.js` (User List, Invite, Accept Invite, Edit User,
  Destructive, Password Reset, Invite Panel, Access, Last Name, Alert Channels) +
  `tests/multi-role.spec.js` (roles-as-array round-trips). Run per-flow during
  iteration (`npx playwright test <file> -g "<name>"`), full suite (`task test`) at
  gate.
- **This PRD specifies the test each flow needs; it does not write them.** Writing or
  *repairing* a test is itself a work order — this doc names the assertion, the WO
  delivers it.
- **What each status turns into downstream:**
  - **WORKING** flows: a **test-audit WO** — spot-check the existing test is
    non-vacuous. If vacuous, it drops to UNPROVEN.
  - **UNPROVEN** flows: a **test-only WO first** — write a real seeded, red-first
    assertion per the bug-fix protocol. Graduates to a **fix WO only if the test goes
    red**.
  - **UNPROVEN (priority)** flows (NFR-1, NFR-2): the test-only WO opens with a
    **confirm-absence step** (grep the specs for a non-admin fixture / a
    double-accept). Both confirm-absence steps have *already been run for this PRD*
    and confirm the assertions are absent — so these are the highest-priority
    test-only WOs.
  - **Stale-test UNPROVEN** flows (FR-16, FR-17): a **test-repair WO** — repoint the
    Access-tab tests from `#t3`/`#s3` to `#t2`/`#s2` so they exercise the real render +
    toggle path, then assert the PUT persists. These are UNPROVEN, not BROKEN: the
    tests reference DOM (`#t3`, `#s3`) the 2-tab refactor removed
    (`users.html:102-104, 121-122`) and so can't run — revealing no feature defect. The
    feature renders correctly into `#s2`; the repair makes it provable.
- **Endpoints in scope (12 Users-owned):**
  - Admin-only: GET `/api/v1/users` · POST `/api/v1/users/invite` · PATCH
    `/api/v1/users/{id}` · POST `/api/v1/users/{id}/reset-password` · POST
    `/api/v1/users/{id}/revoke` · DELETE `/api/v1/users/{id}` · GET
    `/api/v1/apps/permissions` · PUT `/api/v1/apps/{slug}/permissions`.
  - Admin-or-self: GET `/api/v1/users/{id}/notification-preference` · PUT
    `/api/v1/users/{id}/notification-preference`.
  - Unauthenticated (invite-token lifecycle): GET `/api/v1/auth/invite-info?token=` ·
    POST `/api/v1/auth/accept-invite`.
  - *(Adjacent, not owned by Users but read by it: GET `/api/v1/me` — init gate;
    GET `/api/v1/me/apps` — the grant consumer for NFR-3.)*

### Status tally (the denominator downstream objectives grade against)

Total requirements enumerated: **26** (21 FR + 5 NFR).

| Status | Count | Flows |
|---|---|---|
| **WORKING** | 10 | FR-1, FR-3, FR-5, FR-7, FR-8, FR-12, FR-13, FR-14, FR-20, FR-21 |
| **UNPROVEN** | 16 | FR-2, FR-4, FR-6, FR-9, FR-10, FR-11, FR-15, FR-16, FR-17, FR-18, FR-19, NFR-3, NFR-4, NFR-5 · **priority:** NFR-1, NFR-2 |
| **BROKEN** | 0 | *(none confirmed — the Access-tab issue is a stale test that can't run, revealing no feature defect; per the legend it stays UNPROVEN (FR-16, FR-17))* |

**Total: 10 WORKING + 16 UNPROVEN + 0 BROKEN = 26.** Each row is counted exactly
once (no overlap). FR-2 is counted once, as UNPROVEN: its skeleton branch is tested
(WORKING) but its error branch is not, and an app flow with any unproven branch is
UNPROVEN until the gap is closed. The **16 UNPROVEN flows are the candidate WO
backlog**. Two of them (FR-16, FR-17) are **stale-test** WOs — test-repair only, since
the features render correctly into `#s2` and the tests merely point at dead DOM
(`#t3`/`#s3`). The 2 priority-UNPROVEN (NFR-1 admin-gate refusals, NFR-2 token
single-use) are the highest-value new assertions because they guard security
invariants shared across all 8 admin handlers.*

### Activity-2 confirm-absence sweep record (2026-07-11, G6-passed)

Two-pass static audit (pass 1 UI-flow, pass 2 auth-enforcement cross-check) of all 16
UNPROVEN flows against `users.html` + `backend/internal/users/*` + the router;
adversarial G6 re-check of every citation at the cited line. **Result: 0
graduations — all 16 stay UNPROVEN** (every behavior is present-but-untested; no
missing handler, dead route, stub return, never-firing validation, or
render-to-dead-node). Tally unchanged: WORKING 10 · UNPROVEN 16 · BROKEN 0.

Two G3 subtleties confirmed: **FR-9** (email immutable) — the PATCH handler having no
email field IS the immutability feature, not a broken flow. **FR-16/FR-17** — the
render path (`renderAccess`, `users.html:466`) writes to the **live** `#s2` node (the
variable is confusingly *named* `s3` — a holdover from the old 3-tab layout that
likely seeded the stale-test bug); the toggle→PUT persistence chain is real, so these
are stale-test (test-repair) UNPROVEN, not BROKEN.

| Flow | Present at | Confirm-note |
|---|---|---|
| FR-2 | `users.html:159-163,212-213` | `renderError` + load catch "Could not load team." present |
| FR-4 | `handler.go:80-83,94-98` | 422 `validation_error` + 409 `email_already_exists` present |
| FR-6 | `users.html:574-592` | reinvite → POST reset-password → toast + URL cache present |
| FR-9 | `users.html:360-361`; `handler.go:161-168` | email readonly + PATCH omits email = immutability by design (not broken) |
| FR-10 | `users.html:366-367,400`; `db.go:292-305` | default-both-channels + client ≥1 guard + server 400 present |
| FR-11 | `users.html:370-381`; `db.go:306-314` | tz select + `time.LoadLocation` invalid-zone 400 present |
| FR-15 | `users.html:593-610` | inline get-invite-link → POST reset-password → render present |
| FR-16 | `users.html:465-496` (renders into live `#s2`) | stale-test: feature renders; test targets dead `#t3`/`#s3` |
| FR-17 | `users.html:619-628,498-506`; `handler.go:455`; `db.go:463-506` | toggle → savePermissions → PUT → `SetAppPermissions` txn present (stale test) |
| FR-18 | `users.html:629-639`; `db.go:494-500` | add-grant → push → PUT → re-render present |
| FR-19 | `users.html:640-648` | remove-grant → filter → PUT → re-render present |
| NFR-1 | `handler.go:32-34` invoked at `40,63,154,224,283,305,435,458` | `isAdmin` 403 guard live in all 8 admin handlers (priority; no non-admin test) |
| NFR-2 | `db.go:373-390,360-364`; `handler.go:368-377` | `ClaimInviteToken` single-use+expiry predicate + 7-day mint present (priority) |
| NFR-3 | `db.go:463-506,431` | `SetAppPermissions` txn + `GetAppPermissions` feeding `/me/apps` present |
| NFR-4 | `handler.go:500-564` (508,536); `main.go:461-462` | notification-pref admin-or-self + ≥1-channel present; frontend-orphaned (users.html:407 uses PATCH) |
| NFR-5 | `users.html:139` | 401 → `/login.html` redirect present |

### Activity-3 test-audit sweep record (2026-07-11, G6-passed)

Two-pass static audit (pass 1 locate+read each WORKING flow's assertion, pass 2
tautology/subtle-vacuousness cross-check) of all 10 WORKING flows against
`tests/users.spec.js` + `tests/multi-role.spec.js`; adversarial G6 re-check of every
cited assertion incl. the hollow-visibility probes. **Result: 0 drops — all 10
WORKING tests are non-vacuous.** WORKING stays 10.

The visibility/text assertions most likely to be tautological were verified against
`users.html` to require the real code path, not a static node:
- **FR-8** `#nick-err` `toBeVisible()` — node is `display:none` by default
  (`users.html:49`), `.show` added only on the 409 `nickname_taken` branch
  (`users.html:416`) and removed at save start (`:402`) → requires a real 409.
- **FR-13** `#toast` `toContainText('revoked')` — toast empty by default
  (`users.html:127`), text "Session revoked" set only on revoke success (`:444`).
- **FR-14** delete `.not.toBeVisible()` — targets the specific freshly-created
  `data-user-id` row (proven to exist first), not an always-absent selector.
- **FR-3/FR-5/FR-12** `/login.html?token=` — the token substring appears only when
  the server mint succeeds (panel renders inside the success `try`,
  `users.html:284-285`).

**Stale-test fold-in confirmed (BACKLOG / FR-16-17):** the two Access-tab tests
(`shows all apps…` `users.spec.js:464-467`, `can toggle a role…` `:484-488`) click a
non-existent `#t3` and wait on the empty `#s3` div — dead DOM. They map to FR-16/FR-17
(already UNPROVEN stale-test) and back **none** of the 10 WORKING flows. Activity-4
test-repair pointer: `#t3`/`#s3` → `#t2`/`#s2`.

## Out of scope

- The other four apps (Operations, Inventory, Onboarding, Purchasing) — separate PRDs.
- **Fixing** any flow — this PRD enumerates and marks; work orders fix (including
  the Access-tab test repair for FR-16/FR-17).
- Any net-new feature (hardening only). No new role tier, no new grant type.
- Changing the build (static HTML + vanilla JS front end, Go + Postgres back end; no
  framework, no new dependency).
- `login.html` itself and the auth/session machinery (`auth` package) beyond the
  invite-token lifecycle the Users package owns (invite-info / accept-invite).
- `/api/v1/me` and `/api/v1/me/apps` internals — the launcher gate — except as the
  downstream consumer that makes NFR-3's grant round-trip observable.

## Success metrics

- **Enumeration recall ≥ 90%** — `enumerated ÷ (enumerated + discovered-during-WO-
  build) ≥ 0.90`. Denominator: the **26** requirements above plus any flow the build
  surfaces. *(Product KR-2.)*
  - **Recall note (guinea-pig signal):** Pass 1 (UI-first read of `users.html` + the
    two spec files) enumerated **19** flows — the admin-visible surface (list, invite,
    edit, reset, revoke, delete, the Access tab, alert channels). Pass 2 (the
    enforcement cross-check) **added 7**: FR-20, FR-21 (accept-invite lifecycle),
    NFR-1 (403 refusals across 8 handlers), NFR-2 (token single-use + expiry),
    NFR-3 (grant→`me/apps` round-trip), NFR-4 (notification-preference admin-or-self,
    frontend-orphaned), NFR-5 (401 redirect). **Single-pass recall = 19 / 26 ≈ 73%,
    well under the 90% bar.** The two-pass total (26) is the honest set. This is a
    *stronger* signal than Operations' 85%: the Users app's untested mass is almost
    entirely backend enforcement (auth tiers, token lifecycle, grant persistence)
    that a UI-first pass structurally cannot see — the enforcement-angled cross-check
    is not optional here, it is where two-thirds of the risk lives.
- **5/5 apps gate** — this PRD is 1 of 5; same shape as the exemplar. *(Product KR-1.)*
- **Stale-test finding (feeds QA KR-1: vacuous tests 23 → 0):** two Users tests are
  **stale — they target dead DOM and can't run to the flow they name** —
  `shows all apps without needing to select a user` and
  `can toggle a role permission on an app` (`users.spec.js:459-503`) navigate through
  `#t3` and wait on `#s3`, both removed by the 2-tab refactor. They do not silently
  pass; they fail/hang before reaching the toggle, so they assert nothing about the
  feature (which itself renders correctly into `#s2`). That's why FR-16/FR-17 are
  UNPROVEN (stale test), not BROKEN — a test that can't run reveals no defect. One
  additional *soft* vacuity: the alert-channel toggle tests restore state without
  asserting the restored value survives a reload (they assert only the in-DOM class
  flip). No `test.skip` guards exist in the Users specs except the onboarding-template
  fallback in `multi-role.spec.js:196` (not a Users flow).
- **0 known-broken flows** across Users at cycle end — the 16 UNPROVEN (incl. the two
  stale-test repairs, FR-16/FR-17) either reach WORKING or are explicitly waived.
  *(Engineering KR-1.)*
- **100% of non-WORKING flows have a shipped WO** by cycle end. *(Delivery KR-1.)*
- **Every WORKING flow's test is non-vacuous** and every repaired flow carries a
  red-first proof — most urgently the NFR-1 admin-gate and NFR-2 token-single-use
  security invariants. *(QA KR-1, KR-3.)*
