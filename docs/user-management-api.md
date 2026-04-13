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
| operations  | Operations  | 📋   |

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

### Workflow Checklists

#### Data Model — Checklists

```sql
CREATE TABLE checklist_templates (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name             TEXT NOT NULL,
  active_days      INTEGER[],           -- JS Date.getDay() values (0=Sun, 1=Mon, ..., 6=Sat); NULL = every day
  requires_approval BOOLEAN NOT NULL DEFAULT false,
  created_by       UUID REFERENCES users(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  version          INTEGER NOT NULL DEFAULT 1  -- bumped on edit; completions reference version
);

CREATE TABLE checklist_sections (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  "order"     INTEGER NOT NULL,
  condition   JSONB  -- { "days": [1,2,3,4,5] } or NULL
);

CREATE TABLE checklist_fields (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  section_id     UUID NOT NULL REFERENCES checklist_sections(id) ON DELETE CASCADE,
  type           TEXT NOT NULL CHECK (type IN ('checkbox', 'yes_no', 'text', 'temperature', 'photo')),
  label          TEXT NOT NULL,
  required       BOOLEAN NOT NULL DEFAULT false,
  "order"        INTEGER NOT NULL,
  config         JSONB,   -- type-specific: { "unit": "F" } for temperature
  fail_trigger   JSONB,   -- { "type": "out_of_range", "min": 350, "max": 500 } or NULL
  condition      JSONB    -- { "field_id": "uuid", "operator": "equals", "value": "true", "days": [1,2,3] } or NULL
);

CREATE TABLE checklist_submissions (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id  UUID NOT NULL REFERENCES checklist_templates(id),
  version      INTEGER NOT NULL,              -- template version at time of submission
  submitted_by UUID NOT NULL REFERENCES users(id),
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'completed')),
  -- 'pending' = awaiting manager approval (only when template.requires_approval = true)
  -- 'approved' = manager approved
  -- 'rejected' = manager rejected (items sent back for correction)
  -- 'completed' = submitted without approval requirement, or re-approved after correction
  reviewed_by  UUID REFERENCES users(id),
  reviewed_at  TIMESTAMPTZ
);

CREATE TABLE submission_responses (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  value         JSONB NOT NULL,               -- true, false, "text", 42, "blob-url" depending on type
  answered_by   UUID NOT NULL REFERENCES users(id),
  answered_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE submission_fail_notes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  note          TEXT NOT NULL DEFAULT '',
  severity      TEXT CHECK (severity IN ('minor', 'major', 'critical')),
  photo_url     TEXT                           -- S3/Spaces URL or NULL
);

CREATE TABLE submission_rejections (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),  -- which item was flagged
  comment       TEXT NOT NULL,                 -- manager's reason for flagging this item
  require_photo BOOLEAN NOT NULL DEFAULT false,-- manager requires photo evidence on re-submit
  rejected_by   UUID NOT NULL REFERENCES users(id),
  rejected_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Notes:**
- `checklist_templates` are immutable once a submission exists against a version. Edits create a new version.
- `submission_rejections` stores per-item rejection details. When a submission is rejected, the status changes to `rejected` and the crew sees which specific items need correction.
- On re-submission after rejection, a new `checklist_submissions` row is created (not an update to the old one). The old rejected submission is kept for audit trail.
- Photos are stored in Digital Ocean Spaces (or S3). The `value` field in `submission_responses` stores the Spaces URL for photo fields.

---

#### API — Checklist Templates (admin/manager only)

##### GET /api/v1/checklists/templates

Returns all templates the user has permission to manage.

Response 200:
```json
[
  {
    "id": "uuid",
    "name": "Setup Checklist",
    "active_days": [1,2,3,4,5,6,0],
    "requires_approval": true,
    "sections": [
      {
        "id": "uuid",
        "title": "Equipment",
        "order": 0,
        "condition": null,
        "fields": [
          {
            "id": "uuid",
            "type": "yes_no",
            "label": "All equipment powered on?",
            "required": false,
            "order": 0,
            "config": null,
            "fail_trigger": null,
            "condition": null
          }
        ]
      }
    ]
  }
]
```

##### POST /api/v1/checklists/templates

Create a new template. Request body matches the response shape above (minus `id` fields — server generates them).

Response 201: Created template with IDs.

##### PUT /api/v1/checklists/templates/:id

Update a template. Bumps `version`. If submissions exist against the current version, the old version is frozen and a new version is created.

Response 200: Updated template.

##### DELETE /api/v1/checklists/templates/:id

Soft-delete (sets `enabled = false`). Existing submissions are preserved.

Response 204: No content.

---

#### API — Checklist Fill-Out (all roles)

##### GET /api/v1/checklists/today

Returns templates active today for the current user (filtered by `active_days` and role-based access).

Response 200:
```json
[
  {
    "id": "uuid",
    "name": "Setup Checklist",
    "requires_approval": true,
    "sections": [ ... ],
    "existing_submission": null
  }
]
```

`existing_submission` is non-null if the user has an in-progress or rejected submission for today, allowing resume.

##### POST /api/v1/checklists/submissions

Submit a completed checklist.

Request:
```json
{
  "template_id": "uuid",
  "version": 1,
  "responses": [
    { "field_id": "uuid", "value": true },
    { "field_id": "uuid", "value": 42 },
    { "field_id": "uuid", "value": "spaces-url" }
  ],
  "fail_notes": [
    { "field_id": "uuid", "note": "Adjusted temp", "severity": "minor", "photo_url": null }
  ]
}
```

Response 201:
```json
{
  "id": "uuid",
  "status": "pending",
  "message": "Submitted for approval. Waiting for manager review."
}
```

If `requires_approval = false`, status is `completed` directly.

##### POST /api/v1/checklists/submissions/:id/upload-photo

Upload a photo for a field response. Multipart form data.

Request: `multipart/form-data` with `field_id` and `photo` file.

Response 200:
```json
{ "url": "https://spaces.digitalocean.com/yumyums/photos/uuid.jpg" }
```

---

#### API — Approval Flow (admin/manager only)

##### GET /api/v1/checklists/approvals

Returns all submissions with `status = 'pending'` that the current user can approve.

Response 200:
```json
[
  {
    "id": "uuid",
    "template_name": "Setup Checklist",
    "submitted_by": { "display_name": "Jamal M.", "initials": "JM" },
    "submitted_at": "2026-04-13T14:13:00Z",
    "sections": [
      {
        "title": "Equipment",
        "fields": [
          {
            "field_id": "uuid",
            "label": "All equipment powered on?",
            "type": "yes_no",
            "value": true,
            "answered_by": "Jamal M.",
            "answered_at": "2026-04-13T14:12:00Z"
          }
        ]
      }
    ],
    "fail_count": 0,
    "photo_count": 1
  }
]
```

##### POST /api/v1/checklists/approvals/:submission_id/approve

One-tap approval.

Response 200:
```json
{ "status": "approved", "reviewed_by": "Sarah K.", "reviewed_at": "2026-04-13T14:20:00Z" }
```

##### POST /api/v1/checklists/approvals/:submission_id/unapprove

Revoke a previous approval. Requires a reason. Returns the submission to `pending` status.

Request:
```json
{
  "reason": "Approved by accident — need to review temperature readings"
}
```

Response 200:
```json
{ "status": "pending", "unapproved_by": "Sarah K.", "unapproved_at": "2026-04-13T14:25:00Z" }
```

Response 400: `{ "error": "reason_required" }` — reason cannot be empty.
Response 404: Submission not found or not in `approved` status.

---

##### Audit Trail

All approval actions (approve, reject, unapprove) are recorded in a `submission_audit_log` table for compliance:

```sql
CREATE TABLE submission_audit_log (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  action        TEXT NOT NULL CHECK (action IN ('approved', 'rejected', 'unapproved')),
  performed_by  UUID NOT NULL REFERENCES users(id),
  performed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  reason        TEXT,                          -- required for reject and unapprove; NULL for approve
  details       JSONB                          -- rejected_items array for reject; NULL for approve/unapprove
);

CREATE INDEX audit_log_submission_idx ON submission_audit_log(submission_id);
CREATE INDEX audit_log_action_idx ON submission_audit_log(action);
```

**Query example — full audit trail for a submission:**
```sql
SELECT action, performed_by, performed_at, reason
FROM submission_audit_log
WHERE submission_id = $1
ORDER BY performed_at ASC;
```

---

##### POST /api/v1/checklists/approvals/:submission_id/reject

Item-level rejection. Manager flags specific items with comments and optional photo requirements.

Request:
```json
{
  "rejected_items": [
    {
      "field_id": "uuid",
      "comment": "Temperature was not in safe range — re-check and log corrective action",
      "require_photo": true
    },
    {
      "field_id": "uuid",
      "comment": "Photo was blurry — retake",
      "require_photo": false
    }
  ]
}
```

Response 200:
```json
{
  "status": "rejected",
  "rejected_items_count": 2,
  "message": "Crew must fix 2 item(s) and resubmit"
}
```

**Server-side behavior on rejection:**
1. Sets `checklist_submissions.status = 'rejected'`
2. Creates `submission_rejections` rows for each flagged item
3. The original submission responses are preserved (audit trail)
4. When the crew opens the checklist, `GET /api/v1/checklists/today` returns `existing_submission` with the rejected items flagged
5. The crew re-completes only the flagged items (unflagged items retain their responses)
6. On re-submit, a new `checklist_submissions` row is created referencing the same template version
7. The new submission goes back to `status = 'pending'` for re-approval

**Correction flow on the crew's device:**
1. Crew opens My Checklists → sees the checklist with a "Corrections needed" badge
2. Opens the checklist → rejected items are unchecked with a red banner showing the manager's comment
3. If `require_photo = true`, the field requires a photo before the item can be marked complete
4. Once all rejected items are re-completed, crew taps Submit → new submission enters `pending` state

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
