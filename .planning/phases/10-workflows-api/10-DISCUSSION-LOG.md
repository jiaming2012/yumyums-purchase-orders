# Phase 10: Workflows API - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-15
**Phase:** 10-workflows-api
**Areas discussed:** Schema design, Builder save flow, Offline sync strategy, API migration approach, My Checklists tab behavior, Approvals tab behavior, Error handling & loading states, Photo storage

---

## Schema Design

### Conditions Storage

| Option | Description | Selected |
|--------|-------------|----------|
| JSONB column | Store conditions as JSON on field/section rows | ✓ |
| Separate conditions table | Normalize into own table with type/target/value | |
| Fully normalized | Separate tables per condition type | |

**User's choice:** JSONB column
**Notes:** Simpler schema, flexible for new condition types, matches existing JS objects

### Submissions Storage

| Option | Description | Selected |
|--------|-------------|----------|
| Submission row + response rows | One submissions table + field_responses table | ✓ |
| Submission + JSONB responses | Single table with JSONB column for all responses | |

**User's choice:** Submission row + response rows
**Notes:** Enables per-field queries and item-level approval

### Sub-steps Model

| Option | Description | Selected |
|--------|-------------|----------|
| Self-referencing fields table | Nullable parent_field_id on fields table | ✓ |
| Separate sub_steps table | Dedicated table for sub-steps | |
| JSONB on parent field | Store sub-steps as JSON array | |

**User's choice:** Self-referencing fields table

### Template Scheduling

| Option | Description | Selected |
|--------|-------------|----------|
| Array column on template | TEXT[] or JSONB on templates table | |
| Separate schedules table | Dedicated table with template_id + day_of_week rows | ✓ |

**User's choice:** Separate schedules table
**Notes:** User asked for pros/cons comparison. Chose separate table for future expansion (per-user scheduling, one-off overrides, complex recurrence).

### Assignees/Approvers

| Option | Description | Selected |
|--------|-------------|----------|
| Junction table | template_assignments with type/id/role columns | ✓ |
| JSONB on template | Store as JSON arrays on template row | |

**User's choice:** Junction table

### Approval Model

**User's choice:** Per-field approval/rejection, matching current mock UI behavior
**Notes:** User said "Approval / rejection should mimic the flow with mock data currently"

---

## Builder Save Flow

### Save Trigger

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit Save button | User taps Save to persist | ✓ |
| Auto-save on edit | Every change immediately PATCHes server | |
| Save on navigate away | Auto-save when switching tabs | |

**User's choice:** Explicit Save button

### Save Method

| Option | Description | Selected |
|--------|-------------|----------|
| Full replace | PUT entire template JSON, server replaces in transaction | ✓ |
| Granular PATCH | Send only changed fields/sections | |

**User's choice:** Full replace

### Template Versioning

| Option | Description | Selected |
|--------|-------------|----------|
| Snapshot on submit | Freeze template JSON in submission row at submit time | ✓ |
| Live reference | Submissions FK to current template | |
| Template versioning table | Full version history with version FK | |

**User's choice:** Snapshot on submit
**Notes:** User asked for pros/cons of each, then storage growth comparison (snapshot vs versioning). At their scale (~5 templates, daily checklists), difference is negligible (single-digit MB/year). Chose snapshot for simplicity.

### Builder Access

**User's choice:** Admin + superadmin only for now
**Notes:** "Permissions should be part of the users app. For now we can do admin + superadmin can create only, and build with future changes in mind"

### Template Deletion

| Option | Description | Selected |
|--------|-------------|----------|
| Soft delete | archived_at column, can unarchive | ✓ |
| Hard delete | DELETE CASCADE | |

**User's choice:** Soft delete

---

## Offline Sync Strategy

### Offline Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Submit queue only | Queue submissions in IndexedDB, sync on online event | ✓ |
| Full offline (cache templates too) | Cache templates in IndexedDB for offline browsing | |
| Offline-first | Everything works offline by default | |

**User's choice:** Submit queue only
**Notes:** User added requirement for race condition tests

### Sync Indicator

| Option | Description | Selected |
|--------|-------------|----------|
| Banner + per-item badge | Persistent banner + badge per queued checklist | ✓ |
| Toast notification only | Brief toast, no persistent indicator | |
| Submit button state change | Button reflects status on detail page only | |

**User's choice:** Banner + per-item badge

### Conflict Resolution

| Option | Description | Selected |
|--------|-------------|----------|
| Accept and flag | Server accepts, flags for manager review | |
| Reject and notify | Server returns 409, client shows error | ✓ |

**User's choice:** Reject and notify

### Idempotency

| Option | Description | Selected |
|--------|-------------|----------|
| Client-generated UUID | UUID per submission, UNIQUE constraint on server | ✓ |
| Server-side dedup | Match template_id + user_id + time window | |

**User's choice:** Client-generated UUID

---

## API Migration Approach

### Migration Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Big-bang swap | Replace all mock data with fetch calls in one pass | ✓ |
| Incremental (tab by tab) | Migrate one tab at a time | |
| Feature flag | Toggle between mock and real data | |

**User's choice:** Big-bang swap

### API Style

| Option | Description | Selected |
|--------|-------------|----------|
| REST with action endpoints | REST CRUD + action endpoints for workflow transitions | |
| Pure REST (status via PATCH) | PATCH for status transitions | |
| Full RPC | All operations are POST with action names | ✓ |

**User's choice:** Full RPC for writes, GET for reads (hybrid)

### E2E Tests

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite against real API | Tests hit real Go server with test database | ✓ |
| Keep mock tests, add API tests | Two test suites in parallel | |

**User's choice:** Rewrite against real API

### Seed Data

| Option | Description | Selected |
|--------|-------------|----------|
| Seed on first run | Pre-seed existing templates | |
| Empty start | App launches with no templates | ✓ |

**User's choice:** Empty start with ability to seed from a YAML doc

---

## My Checklists Tab Behavior

### In-Progress Checklists

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-save on each field | Each tap/entry saves to server immediately | ✓ |
| Save on Submit only | Nothing saved until explicit Submit | |

**User's choice:** Auto-save on each field

### Time Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Today only + history link | Today's checklists + "View History" link | ✓ |
| Today + recent 7 days | Today at top + past week scrollable | |

**User's choice:** Today only + history link

---

## Approvals Tab Behavior

### Grouping

| Option | Description | Selected |
|--------|-------------|----------|
| Chronological, newest first | Simple reverse-chronological list | |
| Grouped by template | All submissions for each template together | ✓ |
| Grouped by crew member | All submissions by each person together | |

**User's choice:** Grouped by template

---

## Error Handling & Loading States

### Loading States

| Option | Description | Selected |
|--------|-------------|----------|
| Skeleton screens | Placeholder shapes matching content layout | ✓ |
| Centered spinner | Spinning indicator in content area | |
| You decide | Claude's discretion | |

**User's choice:** Skeleton screens

### Error Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Inline error with retry | Error message in content area with Retry button | ✓ |
| Toast notification | Brief error toast at bottom | |

**User's choice:** Inline error with retry

---

## Photo Storage

### Phase 10 Photo Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Skip photos this phase | Photo fields show "Coming soon" placeholder | ✓ |
| Base64 in DB (temporary) | Store as base64 strings in JSONB | |
| Local filesystem | Save to server directory | |

**User's choice:** Skip photos this phase (defer to Phase 12)

---

## Claude's Discretion

- IndexedDB schema design (table names, key structure)
- Debounce timing for auto-save field responses
- Error retry backoff strategy
- Skeleton screen CSS implementation details

## Deferred Ideas

- Photo storage and camera integration — Phase 12
- Granular per-template Builder permissions — Phase 11 (Users app)
- Complex scheduling (per-user, time-of-day, recurrence) — future milestone
- Template versioning with diff/rollback — future milestone
