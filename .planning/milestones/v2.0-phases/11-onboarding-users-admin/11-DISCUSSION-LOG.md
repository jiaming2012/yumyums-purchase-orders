# Phase 11: Onboarding + Users Admin - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 11-onboarding-users-admin
**Areas discussed:** Onboarding schema + API, Sign-off journal, Invite acceptance UX, Session revocation UI, Users.html migration scope, Onboarding API migration, Password reset flow, Nickname collision UX

---

## Onboarding Schema + API

### Save model

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-save per item | Each checkbox tap or video-watched mark auto-saves to the server immediately. Matches workflows pattern. | ✓ |
| Explicit submit per section | Crew completes a section, then taps Submit/Done. Simpler API but risks losing progress. | |
| Auto-save with section submit | Hybrid: auto-save items + explicit "Complete Section" action. | |

**User's choice:** Auto-save per item (Recommended)
**Notes:** Consistent UX across both tools.

### Schema

| Option | Description | Selected |
|--------|-------------|----------|
| Separate ob_* tables | Own tables: ob_templates, ob_sections, ob_items, ob_progress, ob_signoffs. Cleaner separation. | ✓ |
| Extend checklist_* tables | Add onboarding columns to existing checklist tables. Shares infrastructure but muddies schema. | |

**User's choice:** Separate ob_* tables (Recommended)
**Notes:** None

### API style

| Option | Description | Selected |
|--------|-------------|----------|
| RESTful resources | GET/POST/PUT/DELETE on /api/v1/onboarding/*. Standard REST for CRUD. | |
| RPC-style (match workflows) | POST /api/v1/onboarding/saveProgress, /signOff, etc. Consistent with Phase 10. | ✓ |

**User's choice:** RPC-style (match workflows)
**Notes:** None

### Builder save

| Option | Description | Selected |
|--------|-------------|----------|
| Same pattern — explicit Save + full PUT | Matches workflows builder (Phase 10 D-08/D-09). | ✓ |
| Auto-save builder edits | Every builder change auto-saves. More modern but inconsistent. | |

**User's choice:** Same pattern — explicit Save + full PUT (Recommended)
**Notes:** None

---

## Sign-off Journal

### Sign-off data

| Option | Description | Selected |
|--------|-------------|----------|
| Name + timestamp only | Minimal. Clean, sufficient for accountability. | |
| Name + timestamp + notes | Optional free-text notes field. More context but more friction. | |
| Name + timestamp + notes + rating | Notes plus rating (Ready / Needs Practice / Struggling). | ✓ (modified) |

**User's choice:** Custom — Name + timestamp in UX, but notes and ratings stored in backend.
**Notes:** Backend captures rich data (notes + rating), but crew-facing UX only shows name + timestamp.

### Journal view

| Option | Description | Selected |
|--------|-------------|----------|
| Inline on hire's checklist | Sign-off details under each section header. Both manager and hire see it. | ✓ |
| Separate journal tab/page | Dedicated journal view, chronological. | |
| Both — inline + journal | Most complete but most UI work. | |

**User's choice:** Inline on hire's checklist (Recommended)
**Notes:** None

### Sign-off UX

| Option | Description | Selected |
|--------|-------------|----------|
| Tap-to-approve, no form | Instant sign-off, no modal. Notes/rating NULL for now. | |
| Optional expand form | Expandable form with optional notes + rating. | |
| Required notes + rating | Manager must enter notes and select rating before sign-off. | ✓ |

**User's choice:** Required notes + rating
**Notes:** Manager sees form with notes (text) and rating (Ready / Needs Practice / Struggling) before confirming.

---

## Invite Acceptance UX

### Invite mode

| Option | Description | Selected |
|--------|-------------|----------|
| Same page, mode switch | login.html detects ?token= in URL. Shows set-password form. | ✓ (modified) |
| Separate accept-invite.html page | Dedicated page. Cleaner separation but more files. | |

**User's choice:** Custom — Detailed flow description
**Notes:** Admin creates user in Users app (first name, last name, email, role). User shows up immediately with Invited badge. Managers can assign them to checklists before acceptance. Email sent (or opaque token for testing). User clicks link → set password → auto-login → home page. Invited badge disappears.

### Token errors

| Option | Description | Selected |
|--------|-------------|----------|
| Inline error + contact admin | Friendly error: "This invite link has expired. Ask your manager to send a new one." | ✓ |
| Error + request new invite button | Error with "Request New Invite" button that emails admin. | |

**User's choice:** Inline error + contact admin (Recommended)
**Notes:** None

### Email requirement

| Option | Description | Selected |
|--------|-------------|----------|
| Opaque token only for now | Admin copies link and shares manually. No Resend needed yet. | ✓ |
| Real email required | Resend integration from day one. | |

**User's choice:** Opaque token only for now (Recommended)
**Notes:** User also wants "An email has been sent" text shown (future-proofing).

### Link display

| Option | Description | Selected |
|--------|-------------|----------|
| Show link + copy button | After creating user, show invite link with copy-to-clipboard. | ✓ |
| Email only, no link shown | Requires email integration to be working. | |

**User's choice:** Show link + copy button (Recommended)
**Notes:** Also notify admin that an email has been sent.

### Naming model

**User's choice:** Custom — unprompted decision
**Notes:** display_name is a derived function: first_name + last_initial (e.g., "Jamal C."). Optional nickname field overrides display_name. Nickname must not collide with any other user's nickname or derived display_name.

---

## Session Revocation UI

### Placement

| Option | Description | Selected |
|--------|-------------|----------|
| In the Edit user form | Edit form shows role/name fields plus Force Logout button at bottom. | ✓ |
| On the user list card | Quick action on user card. Clutters list. | |
| Separate "Sessions" section | New section showing active sessions. More complex. | |

**User's choice:** In the Edit user form (Recommended)
**Notes:** None

### After revoke

| Option | Description | Selected |
|--------|-------------|----------|
| Next API call gets 401 → redirect to login | Lazy detection. Simple, no WebSocket needed. | ✓ |
| Immediate redirect via polling | Frontend polls session-check endpoint. More responsive but overhead. | |

**User's choice:** Next API call gets 401 → redirect to login (Recommended)
**Notes:** None

### Confirmation

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, confirmation required | "Force logout {name}? They will need to log in again." | ✓ |
| No confirmation | Tap → immediate logout. | |

**User's choice:** Yes, confirmation required (Recommended)
**Notes:** Same pattern as Delete User confirmation.

---

## Users.html Migration Scope

### Swap style

| Option | Description | Selected |
|--------|-------------|----------|
| Big-bang swap | Replace all mock arrays with API calls in one pass. | ✓ |
| Incremental migration | Wire one section at a time. | |

**User's choice:** Big-bang swap (Recommended)
**Notes:** Consistent with workflows pattern.

### Editable fields

| Option | Description | Selected |
|--------|-------------|----------|
| Role + nickname editable | First/last name read-only after creation. | |
| Everything editable | Admin can update any field. | ✓ (modified) |

**User's choice:** Custom — Role, nickname, first name, last name editable. Email is read-only (set at invite time, never changes).
**Notes:** None

---

## Onboarding API Migration

### OB swap

| Option | Description | Selected |
|--------|-------------|----------|
| Big-bang swap | Replace all mock data with API calls in one pass. | ✓ |
| Incremental — read-only first | Wire read paths first, then writes separately. | |

**User's choice:** Big-bang swap (Recommended)
**Notes:** None

### Tests

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite against real server | Existing 35 tests rewritten with real API + test DB. | ✓ |
| Keep mock tests, add API tests | Two test suites. More coverage but harder to maintain. | |

**User's choice:** Rewrite against real server (Recommended)
**Notes:** Consistent with workflows test migration.

---

## Password Reset Flow

### Reset flow

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse invite flow | Admin triggers reset → server generates token → link shown with copy button → same set-password screen. | ✓ |
| Self-service forgot password | "Forgot password?" on login.html. Requires Resend. | |
| Admin sets temporary password | Admin enters temp password. Crew logs in and must change. | |

**User's choice:** Reuse invite flow (Recommended)
**Notes:** 100% reuse of invite UX. No new screens.

---

## Nickname Collision UX

### Collision UX

| Option | Description | Selected |
|--------|-------------|----------|
| Error on save | Server returns 409: "'X' is already taken by Y." Inline error. | ✓ |
| Inline validation on blur | Async check on field blur. Real-time but adds endpoint. | |
| Server validation + suggestions | 409 plus alternative suggestions. Friendlier but complex. | |

**User's choice:** Error on save (Recommended)
**Notes:** None

---

## Claude's Discretion

- ob_* table schema details
- API endpoint naming within RPC pattern
- Onboarding API response shapes
- Error handling / loading states (follow Phase 10 patterns)
- Sign-off form UI layout
- Invite link URL structure and token format
- Test seed data

## Deferred Ideas

- Resend email integration — future phase
- Self-service forgot password — future phase
- Manager reporting dashboard for sign-off ratings — future phase
- Granular per-template builder permissions — future phase
