# Phase 11: Onboarding + Users Admin — Research

**Researched:** 2026-04-17
**Domain:** Go backend API (onboarding schema + user admin), vanilla JS frontend wiring, invite/accept token flow
**Confidence:** HIGH — all findings sourced from the codebase itself; no speculative library research needed

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Onboarding Schema + API**
- D-01: Separate `ob_*` tables (ob_templates, ob_sections, ob_items, ob_progress, ob_signoffs) — NOT reusing checklist_* tables
- D-02: Auto-save per item — each checkbox tap or video-watched mark POSTs to server immediately
- D-03: RPC-style API endpoints (POST /api/v1/onboarding/saveProgress, /signOff, etc.) matching workflows Phase 10 write pattern
- D-04: Builder save uses explicit Save button + full PUT replace; Discard button to abandon

**Sign-off Journal**
- D-05: Sign-off form requires notes (free text) AND rating (Ready / Needs Practice / Struggling)
- D-06: Sign-off display shows only name + timestamp inline on hire's checklist view; notes/rating stored but not shown to hire
- D-07: ob_signoffs columns: manager_id, section_id, hire_id, timestamp, notes, rating

**User Naming Model**
- D-08: users table gets `first_name`, `last_name`, `nickname` (optional) columns — replaces single `display_name` column
- D-09: display_name derived: `nickname` if set, else `"{first_name} {last_initial}."`
- D-10: Nickname uniqueness: must not collide with any other user's nickname OR any user's derived display_name — enforced server-side
- D-11: Nickname collision = 409 with message "'X' is already taken by Y." — inline error under nickname field

**Invite Acceptance Flow**
- D-12: Admin creates user in users.html (first name, last name, email, role); user appears in list with "Invited" badge immediately
- D-13: Invited users can be assigned to checklists/approvals BEFORE accepting invite; cannot log in until accepted
- D-14: Opaque token for Phase 11 — no Resend email integration; admin copies and shares manually
- D-15: After creating user, UI shows invite link + copy-to-clipboard button + "An email has been sent" text
- D-16: login.html detects `?token=` in URL → accept-invite mode: "Welcome, {name}" heading, password + confirm password fields, "Set Password & Log In" button; email field hidden
- D-17: On successful accept-invite: status → 'active', "Invited" badge disappears, redirect to index.html
- D-18: Expired/used token: inline error "This invite link has expired. Ask your manager to send a new one."

**Password Reset Flow**
- D-19: Password reset reuses invite token flow — admin triggers, gets link, shares; same set-password UX on login.html

**Session Revocation UI**
- D-20: Force Logout button in Edit user form, grouped with Reset Password and Delete User below separator
- D-21: Confirmation: "Force logout {name}? They will need to log in again." [Cancel] [Force Logout]
- D-22: Lazy 401 detection — no push; crew's next API call returns 401 → client redirects to login.html

**Users.html Migration**
- D-23: Big-bang mock→API swap — replace USERS, APPS, DEFAULT_PERMS, USER_GRANTS with API calls in one pass
- D-24: Editable after creation: role, nickname, first name, last name; email is read-only
- D-25: Edit form: role dropdown, first name, last name, nickname fields → separator → Reset Password, Force Logout, Delete User

**Onboarding Migration**
- D-26: Big-bang mock→API swap for onboarding.html — delete MOCK_OB_TEMPLATES and SECTION_STATES
- D-27: Existing 35 onboarding E2E tests rewritten against real Go server with test database

### Claude's Discretion
- ob_* table schema details (exact columns, indexes, constraints)
- API endpoint naming within the RPC pattern
- Onboarding API response shapes
- Error handling and loading states (follow Phase 10 patterns: skeleton screens D-24, inline error with retry D-25)
- Sign-off form UI layout and styling
- Invite link URL structure and token format
- Test seed data for onboarding templates

### Deferred Ideas (OUT OF SCOPE)
- Resend email integration for real invite emails
- Self-service "Forgot password?" on login.html
- Manager reporting dashboard for sign-off ratings
- Granular per-template builder permissions
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ONBD-01 | Onboarding templates, sections, items, FAQ Q&A persisted to Postgres | ob_* schema design; migrations 0017-0021 needed; seed data from MOCK_OB_TEMPLATES |
| ONBD-02 | Training progress (checked items, video parts watched) saved per hire | ob_progress table; POST /api/v1/onboarding/saveProgress endpoint; OB_CHECKS → API |
| ONBD-03 | Section sign-off journal entries persisted with manager, reason, timestamp | ob_signoffs table; POST /api/v1/onboarding/signOff endpoint; notes + rating fields |
| ONBD-04 | onboarding.html fetches data from API instead of hardcoded JS arrays | Big-bang swap: MOCK_OB_TEMPLATES + SECTION_STATES + MOCK_HIRES + OB_CHECKS → API |
| USER-01 | Admin can invite new users via API (email invite flow) | POST /api/v1/users/invite exists in api doc; invite_tokens table exists (migration 0003); opaque token, no email |
| USER-02 | Admin can manage user roles and app permissions via API | PATCH /api/v1/users/:id, PUT /api/v1/apps/:slug/permissions — new handlers needed |
| USER-03 | users.html wired to real admin API (replacing mock data) | Big-bang swap: USERS + APPS + DEFAULT_PERMS + USER_GRANTS → API calls |
</phase_requirements>

---

## Summary

Phase 11 is a large-surface API wiring phase with two distinct tracks: (1) the onboarding backend — designing and building the `ob_*` table set and RPC API from scratch — and (2) the users admin backend — wiring `users.html` and `login.html` to APIs that are partially designed but not yet implemented as handlers.

The key complexity is the **users table schema migration** required by D-08: the existing `users` table has a single `display_name TEXT NOT NULL` column (migration 0001) which must be replaced with `first_name`, `last_name`, and `nickname` columns. This is a destructive migration to an already-seeded table — it requires careful handling so superadmin data is preserved.

The **invite/accept-invite flow** reuses existing infrastructure (`invite_tokens` table from migration 0003, `GenerateToken()` from auth/service.go) but needs new handlers: `POST /api/v1/users/invite`, `POST /api/v1/auth/accept-invite`, `POST /api/v1/users/:id/reset-password`, and `POST /api/v1/users/:id/revoke`. The login.html accept-invite mode is a frontend-only extension to the existing form.

The **onboarding schema** must be designed from scratch — it is NOT in `docs/user-management-api.md`. The source of truth for what the schema must support is `onboarding.html`'s `MOCK_OB_TEMPLATES` data structure: templates have sections; sections have items of type `checkbox` or `video_series`; `video_series` items have `parts`; sections have a `requiresSignOff` flag and an `isFaq` flag.

**Primary recommendation:** Treat the schema migration (D-08, users table) as the first task in Wave 1 — it unlocks every downstream user-facing feature. Design ob_* tables in one migration batch before any handler work.

---

## Standard Stack

### Core — Already In Use (no new installs)

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| Go stdlib | 1.25.5 | HTTP handlers, JSON, crypto/rand | Already the server language |
| pgx/v5 | (existing) | Postgres driver + pool | All existing handlers use pgxpool |
| chi/v5 | (existing) | HTTP router + middleware groups | All routes registered in main.go |
| bcrypt (`golang.org/x/crypto`) | (existing) | Password hashing | Used in auth/service.go HashPassword() |
| goose | (existing) | SQL migrations | Migrations 0001-0016 already managed by goose embedded |

### No new dependencies needed

Phase 11 requires no new Go packages. All patterns — token generation, password hashing, session management, migration writing, chi route registration — are already in the codebase.

**Installation:** None required.

---

## Architecture Patterns

### Established Patterns (MUST follow exactly)

#### Pattern 1: RPC-Style Handler with Auth Check

All write handlers follow this structure (from workflow/handler.go):

```go
// Source: backend/internal/workflow/handler.go
func MyHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := auth.UserFromContext(r.Context())
        if user == nil {
            writeError(w, http.StatusUnauthorized, "unauthorized")
            return
        }
        // Role check if admin-only:
        if !isAdmin(user) {
            writeError(w, http.StatusForbidden, "forbidden")
            return
        }
        var input MyInput
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            writeError(w, http.StatusBadRequest, "invalid_body")
            return
        }
        // ... business logic ...
        writeJSON(w, http.StatusOK, result)
    }
}
```

`writeJSON` and `writeError` helpers are in workflow/handler.go — the new `onboarding` and `users` packages should define identical helpers (or share from a common package if preferred — Claude's discretion).

#### Pattern 2: Route Registration in main.go

New route groups added inside the `r.Route("/api/v1", ...)` block:

```go
// Source: backend/cmd/server/main.go lines 258-273
r.Group(func(r chi.Router) {
    r.Use(auth.Middleware(pool, superadmins))
    // Add new route blocks here:
    r.Route("/onboarding", func(r chi.Router) { ... })
    r.Route("/users", func(r chi.Router) { ... })
})
```

#### Pattern 3: Token Generation (Reuse auth.GenerateToken)

```go
// Source: backend/internal/auth/service.go
raw, hash, err := auth.GenerateToken()  // 32-byte random, SHA-256 hash for storage
// Store hash in invite_tokens.token_hash
// Return raw to client (in invite link URL)
```

Token lookup: compute `auth.HashToken(rawFromURL)` → query `invite_tokens WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()`.

#### Pattern 4: goose Migration Structure

```sql
-- Source: backend/internal/db/migrations/0001_users.sql
-- +goose Up
BEGIN;
CREATE TABLE ... ;
COMMIT;

-- +goose Down
DROP TABLE ...;
```

One logical change per file. Next available number is **0017**.

#### Pattern 5: Frontend api() Wrapper (Big-Bang Swap Reference)

The Phase 10 `workflows.html` introduced an `api()` wrapper. Phase 11 must replicate it in `onboarding.html` and `users.html`:

```javascript
// Pattern from workflows.html (established in Phase 10)
async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (res.status === 401) { window.location.href = '/login.html'; throw new Error('unauthorized'); }
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw data;
  return data;
}
```

#### Pattern 6: Skeleton Screen (Phase 10 D-24)

Per UI-SPEC: skeleton rows use `.skeleton-row` class with `background: var(--brd)`. Show on initial load, replace with real data when API resolves.

---

### Recommended Package Structure

```
backend/internal/
├── auth/           # Existing — reuse GenerateToken, HashToken, HashPassword
├── db/
│   └── migrations/
│       ├── 0017_users_naming.sql      # ADD first_name, last_name, nickname; migrate display_name; DROP display_name
│       ├── 0018_ob_templates.sql      # ob_templates table
│       ├── 0019_ob_sections.sql       # ob_sections table
│       ├── 0020_ob_items.sql          # ob_items table (checkbox + video_series + faq item types)
│       ├── 0021_ob_progress.sql       # ob_progress table (per hire, per item)
│       └── 0022_ob_signoffs.sql       # ob_signoffs table
├── onboarding/
│   ├── handler.go    # All /api/v1/onboarding/* handlers
│   └── db.go         # SQL queries for ob_* tables
└── users/
    ├── handler.go    # All /api/v1/users/* and /api/v1/apps/* handlers
    └── db.go         # SQL queries for users, invite_tokens, sessions, app_permissions
```

---

## Onboarding Schema Design

### Data Structures from MOCK_OB_TEMPLATES (source of truth)

The mock data in `onboarding.html` reveals the required schema:

**ob_templates:**
- id, name, role (nullable — some templates are role-agnostic, e.g. tpl_food_safety has `role: null`)
- created_by (UUID ref users), created_at, updated_at

**ob_sections:**
- id, template_id (FK), title, order, requires_sign_off (boolean), is_faq (boolean)

**ob_items:**
- id, section_id (FK), type (`checkbox` | `video_series` | `faq`), label/question, answer (for faq), order
- For `video_series`: the item holds child `ob_video_parts` (separate table)

**ob_video_parts:**
- id, item_id (FK), title, description, url, order

**ob_progress:**
- id, hire_id (FK users), item_id (FK ob_items OR ob_video_parts — see note below), checked_at

**ob_signoffs:**
- id, section_id (FK ob_sections), manager_id (FK users), hire_id (FK users), notes TEXT, rating TEXT CHECK ('ready','needs_practice','struggling'), signed_off_at TIMESTAMPTZ

### Key Schema Decision (Claude's Discretion)

Progress tracking granularity: item-level for checkboxes + video-part-level for video series. Two approaches:

**Option A (recommended):** Single `ob_progress` table with `item_id` that can reference either `ob_items` (for checkboxes/faq) or `ob_video_parts` (for video parts), with a `progress_type` discriminator column (`'item' | 'video_part'`). Simpler to query.

**Option B:** Two separate tables — `ob_item_progress` and `ob_video_part_progress`. More normalized but requires two queries to compute section completion.

Option A is recommended for consistency with the monolithic progress query needed for `getProgress()`.

### Template Assignment

Hires in the mock are assigned templates via `assignedTemplates` array. Need an `ob_template_assignments` table:
- id, hire_id (FK users), template_id (FK ob_templates), assigned_by, assigned_at

---

## Users Table Migration (D-08)

### Current State (migration 0001)
```sql
display_name TEXT NOT NULL
```

### Required State (D-08)
```sql
first_name TEXT NOT NULL,
last_name  TEXT NOT NULL,
nickname   TEXT  -- nullable
```

### Migration Strategy

The 0017 migration must:
1. ADD `first_name TEXT NOT NULL DEFAULT ''`, `last_name TEXT NOT NULL DEFAULT ''`, `nickname TEXT`
2. UPDATE rows to split existing `display_name` values (e.g., `"Jamal M."` → `first_name='Jamal'`, `last_name='M'`) — best effort, superadmin data only
3. DROP COLUMN `display_name`
4. ADD UNIQUE constraint logic for nickname (see D-10)

**Nickname uniqueness (D-10) enforcement:** Server-side check only — no simple SQL UNIQUE INDEX can enforce "nickname must not collide with another user's derived display_name" (which is computed). The uniqueness check requires a Go query before INSERT/UPDATE.

### auth.User struct must be updated

`auth.User` currently has `DisplayName string`. After migration:
- DB query in `LookupSession` and `AuthenticateUser` must compute display_name from `first_name`, `last_name`, `nickname`
- DisplayName field is kept in the struct as a derived value (computed in SQL or Go)
- All callers that currently set `display_name` in response JSON continue to work

**Recommended:** Add `CONCAT(COALESCE(NULLIF(nickname,''), first_name || ' ' || LEFT(last_name,1) || '.'))` computation in the `LookupSession` query as a SQL expression, stored as `display_name` in the User struct. This keeps callers unchanged.

---

## Invite Token Flow

### What Already Exists

- `invite_tokens` table: migration 0003 (id, user_id, token_hash, expires_at, used_at)
- `auth.GenerateToken()`: produces raw + hash pair
- `auth.HashToken(raw)`: for looking up from URL parameter

### What Needs to Be Built

**POST /api/v1/users/invite** (admin+):
1. Insert user with `status='invited'`, `password_hash=NULL`, `first_name`, `last_name`, `email`, `role`
2. Generate token: `raw, hash, _ := auth.GenerateToken()`
3. Insert `invite_tokens` row: `token_hash=hash, expires_at=now()+7days, used_at=NULL`
4. Return user + invite link: `https://[host]/login.html?token=[raw]`

**POST /api/v1/auth/accept-invite** (unauthenticated):
1. Look up token: `SELECT it.*, u.first_name FROM invite_tokens it JOIN users u ON it.user_id = u.id WHERE it.token_hash = $1 AND it.used_at IS NULL AND it.expires_at > now()`
2. Hash password: `auth.HashPassword(input.Password)`
3. Update user: `status='active', password_hash=hash, accepted_at=now()`
4. Mark token used: `UPDATE invite_tokens SET used_at=now() WHERE id=$1`
5. Create session: `auth.CreateSession(ctx, pool, userID)`
6. Set `hq_session` cookie (same as LoginHandler)
7. Return same shape as /auth/login response

**Token accept-invite also needs the user's `first_name`** for the "Welcome, {first_name}" heading — the frontend needs to call `GET /api/v1/auth/invite-info?token=[raw]` first (to show the name before password is set), OR the token lookup endpoint returns first_name in the 200 response body. Recommended: add a `GET /api/v1/auth/invite-info?token=` endpoint that returns `{first_name, status}` for the welcome heading on login.html.

**POST /api/v1/users/:id/reset-password** (admin+):
- Same as invite flow: generate new token, insert into invite_tokens, return link
- Does NOT invalidate existing sessions (force logout is separate)

**POST /api/v1/users/:id/revoke** (admin+):
- `DELETE FROM sessions WHERE user_id = $1`
- Returns 204

---

## Session Revocation Implementation

The lazy-401 pattern (D-22) is already implemented in `auth.Middleware` — it checks `token_hash` on every request. Deleting all sessions for a user from the DB is sufficient; no push mechanism needed.

```sql
-- Force logout implementation
DELETE FROM sessions WHERE user_id = $1;
```

The existing `DeleteSessionByHash` function deletes one session. Need a new `DeleteAllSessionsByUserID(ctx, pool, userID)` function in auth/service.go.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Secure random tokens | Custom random | `auth.GenerateToken()` | Already exists, uses crypto/rand correctly |
| Password hashing | Custom hash | `auth.HashPassword()` / `auth.VerifyPassword()` | bcrypt with DefaultCost, already in codebase |
| Cookie management | Custom cookie logic | Replicate `auth.LoginHandler` cookie pattern | SameSite, HttpOnly, Secure flags already handled |
| SQL migrations | Manual schema changes | goose migrations (0017+) | Goose handles ordering, rollback, embedded FS |
| Display name derivation | DB column | SQL COALESCE expression in query | Computed on read, not stored — avoids sync issues |
| Nickname collision check | SQL UNIQUE INDEX | Go server-side query check | Cross-column uniqueness (nickname vs derived names) cannot be expressed as a simple index |

---

## Common Pitfalls

### Pitfall 1: display_name Column References After Migration

**What goes wrong:** After migration 0017 drops `display_name`, any query that SELECTs `display_name` breaks at runtime.
**Why it happens:** `auth/service.go` LookupSession and AuthenticateUser both SELECT `u.display_name`. The `auth.User` struct has a `DisplayName` field. After the column is dropped, these queries will fail.
**How to avoid:** Update the SQL queries in `service.go` SIMULTANEOUSLY with writing migration 0017. Use a SQL expression: `COALESCE(NULLIF(u.nickname,''), u.first_name || ' ' || LEFT(u.last_name,1) || '.') AS display_name` — the struct field stays the same, callers stay the same.
**Warning signs:** Server fails to start after migration runs — check LookupSession query first.

### Pitfall 2: Invited User Login Attempt

**What goes wrong:** An invited user tries to log in before accepting. `AuthenticateUser` returns nil because `password_hash IS NULL`. The UI shows "Invalid credentials" which is confusing.
**Why it happens:** Current `AuthenticateUser` returns `nil, nil` when password_hash is nil (line 140-142 in service.go). This is correct behavior — but the error message is wrong.
**How to avoid:** The login flow can't distinguish "wrong password" from "not accepted yet". This is acceptable per D-13 (invited users can't log in until accepted). No change needed to auth flow — just accept this UX limitation.

### Pitfall 3: Token Expiry Race on accept-invite

**What goes wrong:** Token lookup succeeds but token expires between lookup and the UPDATE used_at transaction, or token gets used twice in concurrent requests.
**Why it happens:** Non-atomic check-then-update pattern.
**How to avoid:** Use `UPDATE invite_tokens SET used_at=now() WHERE id=$1 AND used_at IS NULL AND expires_at > now() RETURNING user_id` — this makes the mark-used step atomic. If 0 rows returned, return 400 token_expired/token_used.

### Pitfall 4: ob_progress Item ID Ambiguity

**What goes wrong:** ob_progress rows reference both ob_items IDs and ob_video_parts IDs. A JOIN on `item_id` to ob_items will miss video part progress.
**Why it happens:** Video series progress tracks at the part level, not the series item level.
**How to avoid:** Use the discriminator column approach (Option A above). Query progress in two steps: fetch all item-type progress + video-part-type progress in a UNION or separate queries. The frontend JS `isSectionComplete()` function reveals the query structure needed.

### Pitfall 5: Big-Bang Swap Timing — onboarding.html SECTION_STATES

**What goes wrong:** onboarding.html uses `SECTION_STATES` (in-memory dict) to track which sections are locked/active. After the swap, this must come from the API (ob_progress + ob_signoffs determine section state).
**Why it happens:** Section unlock logic is currently pure client-side: `isSectionComplete()` + `canUnlockNext()` + `tryAdvanceSections()`. These functions inspect `OB_CHECKS` (local) and `SECTION_STATES` (local).
**How to avoid:** The onboarding API must return section state as part of the hire's training data: `{ section_id, state: 'active'|'locked'|'complete'|'signed_off' }`. Server computes section state from ob_progress + ob_signoffs. Client renders from server-returned state, not derived locally.

### Pitfall 6: display_name Derivation for "Invited" Users

**What goes wrong:** An invited user has first_name and last_name set at invite time. The derived display_name (e.g., "Dev P.") works. But if last_name is empty string (left blank during invite), the display_name becomes "Dev .".
**Why it happens:** D-08 requires last_name NOT NULL but admin could submit empty string.
**How to avoid:** Add server-side validation: both first_name and last_name must be non-empty on invite. Return 422 validation_error if either is blank.

### Pitfall 7: Onboarding Tests Reference Mock Data

**What goes wrong:** All 35 existing `onboarding.spec.js` tests assume `MOCK_OB_TEMPLATES` and `MOCK_HIRES` exist in the page. After the big-bang swap, these tests will fail immediately.
**Why it happens:** Tests were written against the mock data frontend.
**How to avoid:** Per D-27, tests must be rewritten against the real API with a test database. The test seed data must include: at least one onboarding template with all item types (checkbox, video_series with parts, faq), at least one hire with assigned templates and partial progress. Create `config/ob_templates.yaml` (or equivalent seed) — same pattern as `config/templates.yaml` used by workflows.

---

## Code Examples

### Derived display_name SQL Expression

```sql
-- Source: pattern derived from D-08/D-09 requirements
COALESCE(
  NULLIF(u.nickname, ''),
  u.first_name || ' ' || LEFT(u.last_name, 1) || '.'
) AS display_name
```

Use this in every query that currently selects `u.display_name`:
- `auth.LookupSession` (service.go line 92)
- `auth.AuthenticateUser` (service.go line 120)

### Accept-Invite Atomic Token Claim

```go
// Source: pattern adapted from invite_tokens table (migration 0003)
var userID string
err := pool.QueryRow(ctx, `
    UPDATE invite_tokens
    SET used_at = now()
    WHERE token_hash = $1
      AND used_at IS NULL
      AND expires_at > now()
    RETURNING user_id
`, tokenHash).Scan(&userID)
if err != nil {
    // pgx.ErrNoRows → token not found, expired, or already used
    return nil, ErrTokenInvalid
}
```

### Section State Computation (server-side)

```sql
-- Source: derived from onboarding.html isSectionComplete() + canUnlockNext() logic
-- Returns progress count per section for a given hire
SELECT
  os.id AS section_id,
  os.order,
  os.requires_sign_off,
  COUNT(DISTINCT op.item_id) FILTER (WHERE op.progress_type = 'item') AS items_checked,
  (SELECT COUNT(*) FROM ob_items WHERE section_id = os.id AND type != 'faq') AS items_total,
  (SELECT id FROM ob_signoffs WHERE section_id = os.id AND hire_id = $2 LIMIT 1) AS signoff_id
FROM ob_sections os
LEFT JOIN ob_items oi ON oi.section_id = os.id
LEFT JOIN ob_progress op ON op.item_id = oi.id AND op.hire_id = $2
WHERE os.template_id = $1
GROUP BY os.id, os.order, os.requires_sign_off
ORDER BY os.order
```

### Force Logout (Delete All Sessions)

```go
// New function to add to backend/internal/auth/service.go
func DeleteAllSessionsByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) error {
    _, err := pool.Exec(ctx,
        `DELETE FROM sessions WHERE user_id = $1`,
        userID,
    )
    if err != nil {
        return fmt.Errorf("delete sessions for user %s: %w", userID, err)
    }
    return nil
}
```

### invite-info Endpoint (for login.html accept-invite mode)

```go
// GET /api/v1/auth/invite-info?token=<raw>
// Returns first_name so login.html can show "Welcome, {first_name}" before password is set
// Unauthenticated endpoint
func InviteInfoHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        raw := r.URL.Query().Get("token")
        if raw == "" {
            writeError(w, http.StatusBadRequest, "missing_token")
            return
        }
        tokenHash := auth.HashToken(raw)
        var firstName string
        err := pool.QueryRow(r.Context(), `
            SELECT u.first_name
            FROM invite_tokens it
            JOIN users u ON it.user_id = u.id
            WHERE it.token_hash = $1
              AND it.used_at IS NULL
              AND it.expires_at > now()
        `, tokenHash).Scan(&firstName)
        if err != nil {
            writeError(w, http.StatusBadRequest, "token_expired")
            return
        }
        writeJSON(w, http.StatusOK, map[string]string{"first_name": firstName})
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Mock data in JS arrays | API fetch with skeleton loading | Phase 10 | onboarding.html must replicate Phase 10 pattern |
| Single display_name column | first_name + last_name + nickname | Phase 11 (D-08) | Requires migration + query changes across auth package |
| Static user list | Dynamic API-loaded list with roles | Phase 11 | users.html USERS array replaced by GET /api/v1/users |
| Alert/confirm mocks | Real API calls with error handling | Phase 11 | sendInvite(), deleteUser(), resetPw() all get real endpoints |

**Deprecated in Phase 11:**
- `MOCK_OB_TEMPLATES`: deleted; replaced by GET /api/v1/onboarding/templates
- `MOCK_HIRES`: deleted; replaced by GET /api/v1/onboarding/myHires (manager view) and hire assignment from user data
- `SECTION_STATES`, `OB_CHECKS`, `FAQ_VIEWED`: deleted; replaced by server-returned progress state
- `USERS`, `APPS`, `DEFAULT_PERMS`, `USER_GRANTS` in users.html: deleted; replaced by API calls

---

## Open Questions

1. **Hire record vs user record: are they the same?**
   - What we know: `MOCK_HIRES` has `userId` field pointing to a user. A "hire" is a user who has been assigned onboarding templates.
   - What's unclear: Does Phase 11 need a separate `ob_hires` table, or is a "hire" just a user with ob_template_assignments?
   - Recommendation: No separate hires table. A "hire" is a user + their template assignments. `ob_template_assignments(hire_id, template_id)` table is sufficient. The Manager tab fetches users who have at least one ob_template_assignment.

2. **Onboarding seed data format**
   - What we know: workflows uses `config/templates.yaml` loaded by `workflow.LoadTemplateConfig()`
   - What's unclear: Whether to create `config/ob_templates.yaml` for Phase 11 seed data, or seed via SQL in migration
   - Recommendation: SQL seed data in a separate migration (`0023_ob_seed.sql`) using `INSERT ... ON CONFLICT DO NOTHING` — same idempotency pattern as `db.SeedHQApps()`. This avoids adding a new YAML loader.

3. **Invite link base URL**
   - What we know: Login.html is served from the same origin as the API. Token goes in `?token=` query param.
   - What's unclear: How the server knows its own public base URL to construct the invite link.
   - Recommendation: Accept `HOST` or `BASE_URL` env var in the invite handler. Default to `http://localhost:8080` for dev. Return only the path `/login.html?token=[raw]` in the response and let the frontend prepend `window.location.origin` — cleaner and avoids server needing to know its own URL.

4. **users.html Access tab with real data**
   - What we know: The Access tab currently shows DEFAULT_PERMS per app and USER_GRANTS per app. After migration it must call `GET /api/v1/apps/permissions` and `PUT /api/v1/apps/:slug/permissions`.
   - What's unclear: The Access tab in the existing UI is per-app, not per-user. It shows all apps with their role toggles. D-23 says big-bang swap — but no specific endpoint for this was called out in the user decisions.
   - Recommendation: Implement `GET /api/v1/apps/permissions` (returns all apps with role_grants and user_grants) and `PUT /api/v1/apps/:slug/permissions` as documented in `docs/user-management-api.md`. Access tab calls GET on load, calls PUT on each toggle change (debounced or on explicit Save — Claude's discretion, but follow Phase 10 auto-save pattern).

---

## Environment Availability

Step 2.6: SKIPPED — Phase 11 is backend Go + frontend JS changes only. No external tools, services, or CLI utilities beyond the existing Go/Node/Playwright setup are needed. All required tools confirmed available: Go 1.25.5, Node 22.5.0, Playwright 1.59.1.

---

## Validation Architecture

nyquist_validation is explicitly `false` in `.planning/config.json`. Validation Architecture section skipped.

---

## Project Constraints (from CLAUDE.md)

- Static HTML/CSS/JS — one build step: `node build-sw.js` (Workbox SW generation) — run after any HTML/JS changes
- Run `task sw` or `node build-sw.js` after changing HTML/JS files
- Playwright E2E tests: `task test` (headless, auto-rebuilds SW + creates test DB)
- Every user-entered value MUST follow: update FIELD_RESPONSES → `autoSaveField()` → `DRAFT_RESPONSES` (for workflows); for onboarding: `OB_CHECKS[id] = val` → `saveProgress(id, val)` → server
- **Required test for every new field type:** enter data → back → reopen → data still there (tests/persistence.spec.js)
- Event delegation pattern: ONE click + ONE input listener per container div, routes via `data-action` attributes
- `SCREAMING_SNAKE_CASE` for constants, `camelCase` for functions
- GSD Workflow Enforcement: use `/gsd:execute-phase` for planned phase work — no direct repo edits
- Bug fix protocol: write regression test FIRST, confirm it fails, apply fix, confirm it passes

---

## Sources

### Primary (HIGH confidence)

- `/Users/jamal/projects/yumyums/hq/backend/internal/auth/service.go` — Token generation, password hash, session CRUD, LookupSession query shape
- `/Users/jamal/projects/yumyums/hq/backend/internal/auth/handler.go` — LoginHandler cookie pattern, accept-invite should match
- `/Users/jamal/projects/yumyums/hq/backend/internal/auth/middleware.go` — Session validation; lazy-401 confirmed
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/` — All 16 existing migrations; next number is 0017
- `/Users/jamal/projects/yumyums/hq/backend/internal/workflow/handler.go` — RPC handler pattern; writeJSON/writeError helpers; auth check pattern
- `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go` — Route registration structure; chi group pattern
- `/Users/jamal/projects/yumyums/hq/docs/user-management-api.md` — User/session/invite_tokens schema; API contracts for users endpoints (NOTE: display_name column superseded by D-08)
- `/Users/jamal/projects/yumyums/hq/onboarding.html` — MOCK_OB_TEMPLATES data structures; section state logic; all item types
- `/Users/jamal/projects/yumyums/hq/users.html` — Mock data arrays; edit/access UI patterns to be wired
- `/Users/jamal/projects/yumyums/hq/login.html` — Existing form structure; accept-invite mode to extend
- `/Users/jamal/projects/yumyums/hq/.planning/phases/11-onboarding-users-admin/11-CONTEXT.md` — Locked decisions D-01 through D-27
- `/Users/jamal/projects/yumyums/hq/.planning/phases/11-onboarding-users-admin/11-UI-SPEC.md` — Component inventory; new components; interaction contracts; copywriting

### Secondary (MEDIUM confidence)
- None required — all findings are from the codebase

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Go + pgx + chi + goose all confirmed in use; no new dependencies
- Architecture: HIGH — all patterns lifted directly from existing Go handlers and JS in the codebase
- Pitfalls: HIGH — display_name migration pitfall is confirmed by reading service.go; token race condition is a known Go/SQL pattern
- ob_* schema design: MEDIUM — derived from mock data analysis; Claude's discretion per CONTEXT.md

**Research date:** 2026-04-17
**Valid until:** 2026-05-17 (stable stack — all dependencies are already locked in go.mod)
