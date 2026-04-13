# User Management API — Backend Design

## Overview

This document defines the data model, API contracts, configuration format, and integration plan for the Yumyums HQ user management backend. The current `users.html` is a static mockup; this doc is the implementation target for the Go + Postgres backend.

---

## Data Model

### Table: `users`

```sql
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT UNIQUE NOT NULL,
  display_name  TEXT NOT NULL,
  password_hash TEXT,                         -- NULL until invite accepted
  role          TEXT NOT NULL CHECK (role IN ('admin', 'manager', 'team_member')),
  status        TEXT NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active')),
  invited_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at   TIMESTAMPTZ
);
```

**Notes:**
- Superadmins are NOT stored in this table. They are bootstrapped from `config/superadmins.yaml` on server startup and held in memory (or a read-only cache).
- `password_hash` is NULL for invited users who have not yet accepted.
- `role` covers non-superadmin roles only. Superadmin is a config-level grant.

---

### Table: `hq_apps`

```sql
CREATE TABLE hq_apps (
  id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug    TEXT UNIQUE NOT NULL,   -- e.g. 'purchasing', 'users'
  name    TEXT NOT NULL,          -- e.g. 'Purchasing'
  icon    TEXT NOT NULL,          -- emoji, e.g. '🛒'
  enabled BOOLEAN NOT NULL DEFAULT true
);
```

**Seed data:**

| slug        | name        | icon |
|-------------|-------------|------|
| purchasing  | Purchasing  | 🛒   |
| payroll     | Payroll     | 💰   |
| scheduling  | Scheduling  | 📅   |
| hiring      | Hiring      | 👥   |
| bi          | BI          | 📊   |
| users       | Users       | 🔐   |

---

### Table: `app_permissions`

```sql
CREATE TABLE app_permissions (
  id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_id  UUID NOT NULL REFERENCES hq_apps(id) ON DELETE CASCADE,
  role    TEXT CHECK (role IN ('admin', 'manager', 'team_member')),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  -- Exactly one of role or user_id must be set (XOR constraint)
  CONSTRAINT role_or_user CHECK (
    (role IS NOT NULL AND user_id IS NULL) OR
    (role IS NULL AND user_id IS NOT NULL)
  )
);

CREATE UNIQUE INDEX app_permissions_role_idx ON app_permissions(app_id, role) WHERE role IS NOT NULL;
CREATE UNIQUE INDEX app_permissions_user_idx ON app_permissions(app_id, user_id) WHERE user_id IS NOT NULL;
```

**Semantics:**
- A row with `role = 'admin'` grants all users with role=admin access to that app.
- A row with `user_id = X` grants a specific user access regardless of their role.
- Superadmins always have access to all apps (bypassed at the API layer, not stored here).

---

### Table: `sessions`

```sql
CREATE TABLE sessions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token_hash  TEXT UNIQUE NOT NULL,  -- SHA-256 of the bearer token
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_user_idx ON sessions(user_id);
```

**Notes:**
- Sessions expire after 8 hours by default (configurable).
- Token is a random 32-byte value, sent as a Bearer token in the Authorization header.
- `token_hash` is stored (not the token itself) to limit exposure if DB is compromised.

---

### Table: `invite_tokens`

```sql
CREATE TABLE invite_tokens (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ
);
```

---

## Superadmin Configuration

**File:** `config/superadmins.yaml`

```yaml
# Superadmins are bootstrapped on server startup from this file.
# They cannot be assigned or revoked via the API.
# Add or remove emails here and restart the server to update.

superadmins:
  - email: jamal@yumyums.com
    display_name: Jamal M.
```

**Bootstrap behavior:**
- On startup, the server reads this file.
- For each email, it upserts a synthetic user object held in memory (not persisted to the `users` table).
- Login works normally (password_hash for superadmins is stored externally or in a separate config secret).
- Superadmin identity is checked at the API layer by comparing the authenticated email against the in-memory set.

---

## API Contracts

### Base URL

All endpoints are under `/api/v1`. Authentication uses `Authorization: Bearer <token>` for all protected routes.

---

### Auth

#### POST /api/v1/auth/login

Request:
```json
{
  "email": "jamal@yumyums.com",
  "password": "secret"
}
```

Response 200:
```json
{
  "token": "<bearer-token>",
  "expires_at": "2026-04-12T08:00:00Z",
  "user": {
    "id": "uuid",
    "email": "jamal@yumyums.com",
    "display_name": "Jamal M.",
    "role": "superadmin"
  }
}
```

Response 401:
```json
{ "error": "invalid_credentials" }
```

---

#### POST /api/v1/auth/logout

Authenticated. Invalidates the current session token.

Response 204: No content.

---

### Users

All users endpoints require `admin` or `superadmin` role.

#### GET /api/v1/users

Returns all users (including invited).

Response 200:
```json
[
  {
    "id": "uuid",
    "email": "sarah@yumyums.com",
    "display_name": "Sarah K.",
    "role": "admin",
    "status": "active",
    "invited_at": "2026-03-01T00:00:00Z",
    "accepted_at": "2026-03-02T00:00:00Z"
  }
]
```

---

#### POST /api/v1/users/invite

Creates a new user in `invited` status and sends an invitation email.

Request:
```json
{
  "email": "newuser@yumyums.com",
  "display_name": "New User",
  "role": "team_member"
}
```

Response 201:
```json
{
  "id": "uuid",
  "email": "newuser@yumyums.com",
  "display_name": "New User",
  "role": "team_member",
  "status": "invited",
  "invited_at": "2026-04-12T00:00:00Z"
}
```

Response 409: `{ "error": "email_already_exists" }`

---

#### PATCH /api/v1/users/:id

Update a user's role or display_name. Cannot update superadmin users (403).

Request:
```json
{
  "role": "manager",
  "display_name": "Sarah K."
}
```

Response 200: Updated user object.

---

#### DELETE /api/v1/users/:id

Delete a user and all their sessions. Cannot delete superadmin users (403).

Response 204: No content.

---

#### POST /api/v1/users/:id/reset-password

Triggers a password reset email for the specified user. Any admin+ can reset another user's password (per business rule: "anyone with Users app access can reset another user's password").

Response 204: No content.

---

### Invite Acceptance Flow

1. Admin invites user via `POST /api/v1/users/invite`.
2. Server creates user with `status=invited`, generates a secure invite token, stores `token_hash` in `invite_tokens`.
3. Server sends email: `https://hq.yumyums.com/accept-invite?token=<raw-token>`
4. User clicks link, lands on `login.html` in accept-invite mode.
5. Browser POSTs to:

   **POST /api/v1/auth/accept-invite**

   Request:
   ```json
   {
     "token": "<raw-token-from-email>",
     "password": "chosen-password"
   }
   ```

   Response 200: Session token (same shape as `/auth/login` response).
   Response 400: `{ "error": "token_expired" }` or `{ "error": "token_used" }`.

6. Server marks `invite_tokens.used_at`, sets `users.status = 'active'`, sets `users.password_hash`, and creates a session.

---

### App Permissions

All endpoints require `admin` or `superadmin` role.

#### GET /api/v1/apps/permissions

Returns current permission state for all apps.

Response 200:
```json
[
  {
    "app_slug": "purchasing",
    "role_grants": ["admin", "manager"],
    "user_grants": ["uuid-of-specific-user"]
  }
]
```

---

#### PUT /api/v1/apps/:slug/permissions

Replaces the full permission set for an app.

Request:
```json
{
  "role_grants": ["admin", "manager", "team_member"],
  "user_grants": []
}
```

Response 200: Updated permission object.

---

#### POST /api/v1/apps/:slug/permissions/users/:user_id

Grants a specific user access to an app (individual grant).

Response 201: Updated permission object.

---

#### DELETE /api/v1/apps/:slug/permissions/users/:user_id

Revokes an individual user grant.

Response 204: No content.

---

### Current User

#### GET /api/v1/me

Returns the authenticated user's profile.

Response 200:
```json
{
  "id": "uuid",
  "email": "sarah@yumyums.com",
  "display_name": "Sarah K.",
  "role": "admin",
  "status": "active"
}
```

---

#### GET /api/v1/me/apps

Returns the list of HQ apps the current user has access to. This drives PWA tile filtering.

Response 200:
```json
[
  { "slug": "purchasing", "name": "Purchasing", "icon": "🛒" },
  { "slug": "users", "name": "Users", "icon": "🔐" }
]
```

**Access rules evaluated server-side:**
1. Superadmin → all enabled apps.
2. Admin → all apps where `app_permissions` has a role grant for `admin` OR a user_id grant for this user.
3. Manager / team_member → same logic for their respective roles.

---

## PWA Integration Plan

When the backend goes live, `index.html` will be updated to:

1. On load, call `GET /api/v1/me/apps` with the stored Bearer token.
2. Replace the static tile grid with tiles from the API response.
3. Apps not in the response are hidden (not shown as "Soon").
4. If the request fails with 401, redirect to `login.html`.
5. `login.html` stores the token in `localStorage` as `hq_token` on successful login, then redirects to `index.html`.

**Fallback for offline PWA:** Cache the last known app list in `localStorage` as `hq_apps_cache`. On load, show cached tiles immediately, then refresh in background.

---

## Error Codes

| Code                  | HTTP | Meaning                              |
|-----------------------|------|--------------------------------------|
| `invalid_credentials` | 401  | Wrong email or password              |
| `unauthorized`        | 401  | No valid session token               |
| `forbidden`           | 403  | Authenticated but insufficient role  |
| `not_found`           | 404  | Resource does not exist              |
| `email_already_exists`| 409  | Invite email already in users table  |
| `token_expired`       | 400  | Invite token past expiry             |
| `token_used`          | 400  | Invite token already consumed        |
| `validation_error`    | 422  | Request body failed validation       |
