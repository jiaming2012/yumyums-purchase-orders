# Phase 10: Workflows API — Research

**Researched:** 2026-04-15
**Domain:** Go REST API (chi + pgx) + IndexedDB offline sync + Playwright E2E rewrite
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Schema Design**
- D-01: JSONB columns for conditions and field config on sections and fields tables
- D-02: Normalized submission + field_responses tables — one row per answered field
- D-03: Self-referencing fields table for sub-steps: nullable `parent_field_id UUID REFERENCES fields(id)`
- D-04: Separate schedules table for template scheduling (not array column on template)
- D-05: Junction table `template_assignments` for assignees and approvers: `assignee_type` ('role'|'user'), `assignee_id`, `assignment_role` ('assignee'|'approver')
- D-06: Approval/rejection is per-field. Rejection flags (comment, requirePhoto) live on field_responses. Submission status derived from field states.
- D-07: Soft delete for templates via `archived_at TIMESTAMP`

**Builder Save Flow**
- D-08: Explicit Save button — no auto-save for builder
- D-09: Full replace on save — PUT entire template JSON, server deletes old rows, inserts new in transaction
- D-10: Snapshot on submit — freeze template as JSONB in submission row at submit time
- D-11: Builder access restricted to admin + superadmin only for now

**Offline Sync Strategy**
- D-12: Submit queue only — browsing/starting checklists requires online
- D-13: Banner + per-item badge for pending-sync indicator
- D-14: Reject and notify on conflict — 409 if template archived while offline
- D-15: Client-generated UUID idempotency keys stored in IndexedDB queue
- D-16: Race condition tests required

**API Migration Approach**
- D-17: Big-bang swap — all mocks replaced with fetch calls in one pass
- D-18: Full RPC for writes, GET for reads (POST /api/v1/workflow/createTemplate, etc.)
- D-19: Rewrite existing 54 Playwright E2E tests against real Go server with test database
- D-20: Empty start — no pre-seeded templates; YAML-based seeding for dev/testing

**My Checklists Tab**
- D-21: Auto-save responses on each field tap/entry
- D-22: Today only + history link

**Approvals Tab**
- D-23: Pending approvals grouped by template

**Error Handling & Loading States**
- D-24: Skeleton screens for loading states (pulsing gray placeholder shapes)
- D-25: Inline error with retry button — no modals

**Photos**
- D-26: Photo fields render but show "Coming soon" placeholder — Phase 12

### Claude's Discretion
- IndexedDB schema design (table names, key structure)
- Debounce timing for auto-save field responses
- Error retry backoff strategy
- Skeleton screen CSS implementation details

### Deferred Ideas (OUT OF SCOPE)
- Photo storage and camera integration — Phase 12
- Granular per-template Builder permissions — Phase 11 (Users app)
- Complex scheduling (per-user, time-of-day, recurrence) — future milestone
- Template versioning with diff/rollback — future milestone
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WKFL-01 | Templates, sections, and fields persisted to Postgres (replacing MOCK_TEMPLATES) | Schema design, migration pattern, PUT full-replace transaction |
| WKFL-02 | Checklist submissions saved with field responses, user attribution, timestamps | submissions + field_responses tables, snapshot JSONB, auto-save debounce |
| WKFL-03 | Approval flow persisted — pending, approved, rejected states with manager notes | field-level rejection tables, status machine, per-field rejection API |
| WKFL-04 | workflows.html fetches data from API instead of hardcoded JS arrays | Big-bang mock swap, fetch wrapper pattern, skeleton screens |
| SYNC-01 | Checklist completions queued in IndexedDB when offline | IndexedDB schema, queue structure, UUID idempotency keys |
| SYNC-02 | Queue replays on `online` event with idempotency keys preventing duplicates | `online` event listener, sequential drain, server-side UNIQUE constraint |
| SYNC-03 | User sees visual indicator of pending offline submissions | Banner + badge pattern, queue count display |
</phase_requirements>

---

## Summary

Phase 10 replaces all mock data in `workflows.html` (MOCK_TEMPLATES, MOCK_RESPONSES, FAIL_NOTES, REJECTION_FLAGS, PENDING_APPROVALS, APPROVED_SUBMISSIONS) with a real Go + Postgres backend. The work spans three layers: (1) a new `workflow` package inside the existing chi server, (2) six goose migrations for the checklist schema, and (3) a full rewrite of the 1939-line `workflows.html` data layer — replacing in-memory objects with `fetch` calls against the new API.

The schema is already well-designed in `docs/user-management-api.md` and the CONTEXT decisions. The primary risks are (a) the size of the JS rewrite — every mock mutation must map cleanly to an API call — and (b) the IndexedDB offline queue, which requires careful handling of the `online` event and idempotency on the server side. The Playwright test suite (54 tests, 736 lines) must be rewritten against a real test server — the mock data tests will be deleted entirely.

The architecture is a natural extension of the Phase 9 patterns: new `workflow` package alongside `auth` and `me`, new goose migrations numbered 0006+, and chi route groups registered under `/api/v1/workflow/`. The CONTEXT.md schema is authoritative — `docs/user-management-api.md` documents an earlier version of the schema design, and D-01 through D-07 decisions in CONTEXT.md supersede it where they differ (notably: `template_assignments` junction table replaces array columns, `archived_at` replaces `enabled` boolean, `parent_field_id` for sub-steps instead of inline `sub_steps` array).

**Primary recommendation:** Build the Go layer first (migrations + handlers + repository), verify with curl, then rewrite the JS layer pointing at real endpoints. Do the Playwright rewrite last once the API is stable.

---

## Standard Stack

### Core (already in go.mod — no new dependencies needed)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/go-chi/chi/v5` | v5.2.5 | HTTP routing, middleware groups | Already in project |
| `github.com/jackc/pgx/v5` | v5.9.1 | Postgres driver + pgxpool | Already in project |
| `github.com/pressly/goose/v3` | v3.27.0 | SQL migrations with embed.FS | Already in project |
| `golang.org/x/crypto` | v0.50.0 | bcrypt for password hashing | Already in project |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML template seed config | Already in project |

### Frontend (already loaded)

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| SortableJS | 1.15.7 | Drag-to-reorder in Builder | Already in workflows.html via CDN |
| IndexedDB (browser API) | n/a | Offline submission queue | Built-in; no library needed per D-12 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Raw IndexedDB API | idb library (Jake Archibald) | idb is a thin promise wrapper; raw API is fine given the simple queue schema — saves a CDN dep |
| Background Sync API | `online` event + manual drain | Background Sync has zero iOS Safari support (confirmed in STATE.md decision log) |
| `database/sql` | pgx directly | pgx is already used — no reason to add sql layer |

**No new Go dependencies required.** All needed libraries are already in `go.mod`.

---

## Architecture Patterns

### New Package Structure

```
backend/
├── cmd/server/main.go          # Add: r.Route("/api/v1/workflow", ...)
├── internal/
│   ├── auth/                   # Existing — unchanged
│   ├── config/                 # Existing — extend for template YAML seed
│   ├── db/
│   │   └── migrations/
│   │       ├── 0006_checklist_templates.sql
│   │       ├── 0007_checklist_sections.sql
│   │       ├── 0008_checklist_fields.sql
│   │       ├── 0009_checklist_submissions.sql
│   │       ├── 0010_submission_responses.sql
│   │       └── 0011_submission_rejections.sql
│   ├── me/                     # Existing — unchanged
│   └── workflow/               # NEW
│       ├── handler.go          # HTTP handlers (createTemplate, submitChecklist, etc.)
│       ├── repository.go       # DB queries (pgxpool.Pool)
│       ├── model.go            # Go structs for Template, Submission, etc.
│       └── seed.go             # YAML template seeding (like superadmins pattern)
```

### Pattern 1: Handler → Repository separation (established in Phase 9)

The `me` package shows the pattern: handlers decode JSON + call repository functions + encode response. Handlers do NOT embed SQL.

```go
// Source: backend/internal/me/handler.go
func MeAppsHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := auth.UserFromContext(r.Context())
        rows, err := queryUserApps(r, pool, user)
        // ... encode response
    }
}
```

Apply the same pattern in `workflow/handler.go` — each handler is a closure over `*pgxpool.Pool`.

### Pattern 2: Goose migrations (one logical change per file, BEGIN/COMMIT)

```sql
-- Source: backend/internal/db/migrations/0001_users.sql
-- +goose Up
BEGIN;
CREATE TABLE ...;
COMMIT;

-- +goose Down
DROP TABLE ...;
```

All six new migration files follow this pattern. Each file = one table.

### Pattern 3: YAML seed config (reuse LoadSuperadmins pattern)

```go
// Source: backend/internal/config/config.go
type SuperadminsConfig struct {
    Superadmins []SuperadminEntry `yaml:"superadmins"`
}
func LoadSuperadmins(path string) (map[string]SuperadminEntry, error) { ... }
```

For template seeding, create a parallel `TemplatesConfig` struct with the same YAML loading pattern. Seed file: `config/templates.yaml`. Called from `main.go` startup sequence after migrations (same as `SeedHQApps`).

### Pattern 4: Chi route group for workflow (protected)

```go
// In main.go, inside r.Route("/api/v1", ...)
r.Group(func(r chi.Router) {
    r.Use(auth.Middleware(pool, superadmins))
    // ... existing protected routes ...

    // Workflow — all authenticated
    r.Route("/workflow", func(r chi.Router) {
        r.Get("/templates",          workflow.ListTemplatesHandler(pool))
        r.Post("/createTemplate",    workflow.CreateTemplateHandler(pool))
        r.Put("/updateTemplate/{id}", workflow.UpdateTemplateHandler(pool))
        r.Delete("/archiveTemplate/{id}", workflow.ArchiveTemplateHandler(pool))
        r.Get("/myChecklists",       workflow.MyChecklistsHandler(pool))
        r.Post("/saveResponse",      workflow.SaveResponseHandler(pool))
        r.Post("/submitChecklist",   workflow.SubmitChecklistHandler(pool))
        r.Get("/pendingApprovals",   workflow.PendingApprovalsHandler(pool))
        r.Post("/approveSubmission", workflow.ApproveSubmissionHandler(pool))
        r.Post("/rejectSubmission",  workflow.RejectSubmissionHandler(pool))
    })
})
```

### Pattern 5: Frontend fetch wrapper (new in this phase)

Replace mock data reads with a thin `api(method, path, body)` wrapper. Consistent error handling, JSON encode/decode, cookie auth (httpOnly session cookie is sent automatically by browser).

```javascript
// Source: design decision D-17/D-18
async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch('/api/v1/workflow/' + path, opts);
  if (res.status === 401) { window.location = '/login.html'; return null; }
  if (!res.ok) throw await res.json();
  if (res.status === 204) return null;
  return res.json();
}
```

### Pattern 6: IndexedDB offline queue (Claude's discretion)

Simple queue table — one object store named `submitQueue`. Each entry: `{ id: uuid (key), templateId, version, responses, failNotes, queuedAt }`.

```javascript
// Open / create store
const DB_NAME = 'hq_offline', DB_VERSION = 1;
function openDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = e => e.target.result.createObjectStore('submitQueue', { keyPath: 'id' });
    req.onsuccess = e => resolve(e.target.result);
    req.onerror = e => reject(e.target.error);
  });
}

// Drain on online event
window.addEventListener('online', drainQueue);
async function drainQueue() {
  const db = await openDB();
  const tx = db.transaction('submitQueue', 'readonly');
  const entries = await getAllFromStore(tx, 'submitQueue');
  for (const entry of entries) {
    try {
      await api('POST', 'submitChecklist', entry); // server uses entry.id as idempotency key
      await deleteFromStore('submitQueue', entry.id);
    } catch (err) {
      if (err.code === 'ALREADY_SUBMITTED') await deleteFromStore('submitQueue', entry.id);
      else break; // network still down — stop draining
    }
  }
  renderSyncBanner();
}
```

### Pattern 7: Auto-save responses (D-21)

Debounce each field input 400ms before calling `POST /api/v1/workflow/saveResponse`. Per-field error indicator (red border) on failure with tap-to-retry. Do not block the UI while save is in flight.

```javascript
// Recommended debounce timing: 400ms (long enough to not spam, short enough to feel instant)
const saveDebounce = {};
function autoSaveField(fieldId, value) {
  clearTimeout(saveDebounce[fieldId]);
  saveDebounce[fieldId] = setTimeout(async () => {
    try {
      await api('POST', 'saveResponse', { field_id: fieldId, value });
      clearFieldError(fieldId);
    } catch (e) {
      showFieldError(fieldId); // tap-to-retry per D-25
    }
  }, 400);
}
```

### Pattern 8: Skeleton screens (D-24, Claude's discretion)

```css
/* Pulsing gray placeholder shapes */
.skeleton {
  background: linear-gradient(90deg, var(--brd) 25%, var(--bg) 50%, var(--brd) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.2s infinite;
  border-radius: 6px;
}
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

Render skeleton shapes matching checklist card layout while `GET /workflow/myChecklists` is in flight. Replace with real content on response.

### Anti-Patterns to Avoid

- **Fetching templates on every field tap:** Cache the active template in JS state after the initial `GET /myChecklists`. Only re-fetch on tab switch or explicit refresh.
- **Storing session tokens in localStorage:** Already an established project decision — httpOnly cookies handle auth automatically; `fetch()` sends them without any JS involvement.
- **Mutating IndexedDB entries on retry:** Use `delete + re-add` or `put` with same key rather than partial updates.
- **Running goose migrations outside server startup:** Keep `db.Migrate(pool)` in `main.go` — never a separate CLI command.

---

## Schema Design (Authoritative — from D-01 through D-07)

The CONTEXT.md decisions supersede the earlier `docs/user-management-api.md` schema in the following ways:

| Topic | docs/user-management-api.md | CONTEXT.md decision (authoritative) |
|-------|------------------------------|--------------------------------------|
| Assignees/approvers | Array columns on `checklist_templates` | D-05: `template_assignments` junction table |
| Soft delete | `enabled BOOLEAN` | D-07: `archived_at TIMESTAMPTZ` |
| Sub-steps | Inline `sub_steps` JSONB array | D-03: `parent_field_id UUID` self-reference on fields table |
| Schedules | `active_days INTEGER[]` on template | D-04: Separate `schedules` table |

### Migration Sequence (0006–0011 + schedules and assignments)

```sql
-- 0006_checklist_templates.sql
CREATE TABLE checklist_templates (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name              TEXT NOT NULL,
  requires_approval BOOLEAN NOT NULL DEFAULT false,
  created_by        UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  archived_at       TIMESTAMPTZ  -- D-07: soft delete
);

-- 0007_checklist_schedules.sql
CREATE TABLE checklist_schedules (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  active_days INTEGER[],  -- NULL = every day; D-04
  -- Designed for future per-user and time-of-day expansion
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 0008_template_assignments.sql
CREATE TABLE template_assignments (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id     UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  assignee_type   TEXT NOT NULL CHECK (assignee_type IN ('role', 'user')),
  assignee_id     TEXT NOT NULL,  -- role name OR user UUID
  assignment_role TEXT NOT NULL CHECK (assignment_role IN ('assignee', 'approver'))
  -- D-05: replaces assigned_to_roles[], assigned_to_users[], approver_roles[], approver_users[]
);
CREATE INDEX template_assignments_template_idx ON template_assignments(template_id);

-- 0009_checklist_sections.sql
CREATE TABLE checklist_sections (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  "order"     INTEGER NOT NULL,
  condition   JSONB  -- D-01: { "days": [1,2,3] } or NULL
);

-- 0010_checklist_fields.sql
CREATE TABLE checklist_fields (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  section_id       UUID NOT NULL REFERENCES checklist_sections(id) ON DELETE CASCADE,
  parent_field_id  UUID REFERENCES checklist_fields(id),  -- D-03: sub-steps
  type             TEXT NOT NULL CHECK (type IN ('checkbox','yes_no','text','temperature','photo')),
  label            TEXT NOT NULL,
  required         BOOLEAN NOT NULL DEFAULT false,
  "order"          INTEGER NOT NULL,
  config           JSONB,  -- D-01: { "unit": "F", "min": 300, "max": 500 }
  fail_trigger     JSONB,  -- D-01: { "type": "out_of_range", "min": 300, "max": 500 }
  condition        JSONB   -- D-01: { "field_id": "uuid", "operator": "equals", "value": "true", "days": [1,2,3] }
);

-- 0011_checklist_submissions.sql
CREATE TABLE checklist_submissions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id       UUID NOT NULL REFERENCES checklist_templates(id),
  template_snapshot JSONB NOT NULL,  -- D-10: freeze template at submit time
  submitted_by      UUID NOT NULL REFERENCES users(id),
  submitted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  status            TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','rejected','completed')),
  reviewed_by       UUID REFERENCES users(id),
  reviewed_at       TIMESTAMPTZ,
  idempotency_key   UUID UNIQUE  -- D-15: client-generated UUID, prevents double-submit
);

-- 0012_submission_responses.sql
CREATE TABLE submission_responses (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  value         JSONB NOT NULL,
  answered_by   UUID NOT NULL REFERENCES users(id),
  answered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- For in-progress (pre-submit) auto-saves: submission_id is NULL
  -- session-scoped draft keyed by (field_id, answered_by)
  UNIQUE (submission_id, field_id)  -- one response per field per submission
);
-- Allow NULL submission_id for draft responses:
CREATE UNIQUE INDEX submission_responses_draft_idx
  ON submission_responses(field_id, answered_by)
  WHERE submission_id IS NULL;

-- 0013_submission_fail_notes.sql
CREATE TABLE submission_fail_notes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  note          TEXT NOT NULL DEFAULT '',
  severity      TEXT CHECK (severity IN ('minor','major','critical')),
  photo_url     TEXT  -- Phase 12
);

-- 0014_submission_rejections.sql
CREATE TABLE submission_rejections (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  comment       TEXT NOT NULL,
  require_photo BOOLEAN NOT NULL DEFAULT false,  -- D-06
  rejected_by   UUID NOT NULL REFERENCES users(id),
  rejected_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Note on draft auto-saves (D-21):** The `submission_responses` table doubles as draft storage when `submission_id IS NULL`. Draft rows are keyed by `(field_id, answered_by)`. On submit, a `checklist_submissions` row is created and draft rows are linked by setting `submission_id`. This avoids a separate `draft_responses` table.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| UUID generation (Go) | Custom ID generator | `gen_random_uuid()` in Postgres + standard `uuid` package | Postgres already does this; pgx returns string UUIDs natively |
| UUID generation (JS, idempotency keys) | timestamp-based IDs | `crypto.randomUUID()` | Web Crypto API available in all modern browsers; truly random v4 UUID |
| Request body parsing | Manual JSON decode | `json.NewDecoder(r.Body).Decode(&body)` | Established pattern in every Phase 9 handler |
| Auth check in handlers | Manual cookie read | `auth.UserFromContext(r.Context())` | Middleware already puts user in context (Phase 9) |
| Migration tracking | Custom version table | goose (already in project) | Handles the `goose_db_version` table automatically |
| Duplicate submit prevention | Timestamp window check | `idempotency_key UUID UNIQUE` constraint | DB-level uniqueness is atomic and race-safe |
| Offline detection | Polling navigator.onLine | `window.addEventListener('online', ...)` | Event-driven; navigator.onLine polls are unreliable |

**Key insight:** Every primitive needed for this phase already exists in the project. The work is wiring, not invention.

---

## Common Pitfalls

### Pitfall 1: NULL submission_id draft rows leaking across sessions

**What goes wrong:** Auto-save (D-21) stores draft responses with `submission_id = NULL`. If a user never submits, drafts persist indefinitely and pollute future checklist loads.
**Why it happens:** Draft storage mixed with final responses without a TTL.
**How to avoid:** Add `draft_started_at TIMESTAMPTZ` to draft rows. Cron or on-load cleanup: delete drafts older than 24h. Alternatively, keep drafts only for `submitted_at = today` — query with `WHERE answered_at >= current_date`.
**Warning signs:** `GET /workflow/myChecklists` returns partially filled checklists from previous days.

### Pitfall 2: PUT full-replace (D-09) deleting fields that have draft responses

**What goes wrong:** Admin edits a template and saves. Server deletes old `checklist_fields` rows. Draft responses referencing those `field_id` values become orphaned (FK violation if ON DELETE CASCADE, or silent garbage if not).
**Why it happens:** Full replace on save is simple but destructive.
**How to avoid:** The `checklist_fields` FK on `submission_responses` must be `ON DELETE CASCADE`. Alternatively, check for in-progress submissions before allowing save. Simplest for Phase 10 scope: warn in UI if there are pending submissions before saving.
**Warning signs:** 500 errors from FK constraint violations on template save.

### Pitfall 3: Race condition on `online` event drain (D-16)

**What goes wrong:** Two `online` events fire close together (network flap), starting two concurrent drain loops that both try to submit the same queue entries.
**Why it happens:** Event listeners can fire multiple times; drain is async.
**How to avoid:** Use a module-level `isDraining` flag. Set to `true` at start of `drainQueue()`, reset to `false` when done. Skip if already draining. Server-side `UNIQUE(idempotency_key)` is the final safety net.
**Warning signs:** Duplicate submissions appearing in the Approvals tab.

### Pitfall 4: `template_snapshot` JSONB too large

**What goes wrong:** Long templates with many fields produce large JSONB blobs stored in every submission row. At scale this inflates storage significantly.
**Why it happens:** D-10 requires snapshotting the full template at submit time.
**How to avoid:** For Phase 10 (food truck scale), this is a non-issue — templates have at most 20-30 fields. Include a size assertion in tests: snapshot should be < 100KB. Document the trade-off so Phase N can switch to a `template_versions` table if scale requires it.
**Warning signs:** Submission rows > 100KB.

### Pitfall 5: Builder permission check (D-11) not enforced at API layer

**What goes wrong:** Builder tab is hidden in UI for non-admins but the API endpoint is callable by any authenticated user.
**Why it happens:** Client-side permission hiding is not security.
**How to avoid:** `createTemplate`, `updateTemplate`, `archiveTemplate` handlers must check `user.Role == "admin" || user.IsSuperadmin`. Return 403 for others. This is the same pattern as the existing `MeAppsHandler` superadmin check.
**Warning signs:** Team member able to create templates via direct API call.

### Pitfall 6: Playwright test rewrite scope creep

**What goes wrong:** 54 existing tests target static mock data behavior that will be deleted. Attempting to preserve mock-data tests while also adding real-API tests creates conflicting test infrastructure.
**Why it happens:** D-19 requires rewriting against real server, but existing tests are a significant asset.
**How to avoid:** Delete mock-data tests first. Write new tests against a real test server (started in `beforeAll` with a test DB). Use `playwright.config.js` `webServer` option to start the Go test server automatically. Seed test data via SQL or API calls in `beforeEach`.
**Warning signs:** Test file imports mock data constants or stubs `fetch`.

### Pitfall 7: SW fetch partition and the `/api/` path prefix

**What goes wrong:** `sw.js` is currently cache-first for all routes. API responses get cached and stale data is served.
**Why it happens:** `sw.js` caches all assets on install; fetch handler has no API-vs-static distinction yet.
**How to avoid:** INFRA-04 (Phase 9) must be complete before Phase 10 ships. The SW fetch handler must be partitioned: network-first for `/api/*`, cache-first for static. This is a Phase 9 dependency. Confirm it is done before running Phase 10 E2E tests. The CONTEXT.md notes: "SW fetch partition — `/api/*` already network-first" — verify this is true in the actual `sw.js` before writing the test plan.
**Warning signs:** API calls return stale cached responses; 304 responses without network hit.

---

## Code Examples

### Template full-replace transaction (D-09)

```go
// Source: design from D-09 + pgx transaction pattern
func replaceTemplate(ctx context.Context, pool *pgxpool.Pool, tplID string, tpl TemplateInput) error {
    tx, err := pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    // Delete old sections (cascades to fields via ON DELETE CASCADE)
    if _, err := tx.Exec(ctx,
        `DELETE FROM checklist_sections WHERE template_id = $1`, tplID); err != nil {
        return err
    }

    // Insert new sections + fields
    for i, sec := range tpl.Sections {
        var secID string
        err := tx.QueryRow(ctx,
            `INSERT INTO checklist_sections (template_id, title, "order", condition)
             VALUES ($1, $2, $3, $4) RETURNING id`,
            tplID, sec.Title, i, sec.Condition).Scan(&secID)
        if err != nil { return err }

        for j, fld := range sec.Fields {
            _, err = tx.Exec(ctx,
                `INSERT INTO checklist_fields
                 (section_id, type, label, required, "order", config, fail_trigger, condition)
                 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
                secID, fld.Type, fld.Label, fld.Required, j,
                fld.Config, fld.FailTrigger, fld.Condition)
            if err != nil { return err }
        }
    }

    // Update template metadata + bump updated_at
    _, err = tx.Exec(ctx,
        `UPDATE checklist_templates SET name=$1, requires_approval=$2, updated_at=now()
         WHERE id=$3`,
        tpl.Name, tpl.RequiresApproval, tplID)
    if err != nil { return err }

    return tx.Commit(ctx)
}
```

### Idempotency key enforcement on submit (D-15)

```go
// Source: design from D-15
// INSERT ... ON CONFLICT (idempotency_key) DO NOTHING RETURNING id
var submissionID string
err := pool.QueryRow(ctx, `
    INSERT INTO checklist_submissions
        (template_id, template_snapshot, submitted_by, idempotency_key, status)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
    RETURNING id`,
    templateID, snapshotJSON, userID, idempotencyKey, initialStatus,
).Scan(&submissionID)
// If err is a unique violation on a different column, that's a real error.
// ON CONFLICT on idempotency_key returns the existing row — safe to return 200/201.
```

### IndexedDB queue (offline, D-12 through D-16)

```javascript
// Source: design from D-12/D-15
const HQ_DB = 'hq_offline_v1';
let dbPromise = null;

function getDB() {
  if (!dbPromise) dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(HQ_DB, 1);
    req.onupgradeneeded = e => {
      e.target.result.createObjectStore('submitQueue', { keyPath: 'id' });
    };
    req.onsuccess = e => resolve(e.target.result);
    req.onerror = e => reject(e.target.error);
  });
  return dbPromise;
}

async function enqueue(payload) {
  // payload.id is a client-generated UUID (crypto.randomUUID())
  const db = await getDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('submitQueue', 'readwrite');
    tx.objectStore('submitQueue').put(payload);
    tx.oncomplete = resolve;
    tx.onerror = e => reject(e.target.error);
  });
}

let _draining = false;
async function drainQueue() {
  if (_draining || !navigator.onLine) return;
  _draining = true;
  try {
    const db = await getDB();
    const entries = await idbGetAll(db, 'submitQueue');
    for (const entry of entries) {
      try {
        await api('POST', 'submitChecklist', entry);
        await idbDelete(db, 'submitQueue', entry.id);
      } catch (err) {
        if (err && err.error === 'duplicate_submission') {
          await idbDelete(db, 'submitQueue', entry.id);
        } else {
          break; // network failure — stop draining
        }
      }
    }
  } finally {
    _draining = false;
    renderSyncBanner(); // update banner + badge count
  }
}
window.addEventListener('online', drainQueue);
```

### Sync banner (D-13)

```javascript
// Persistent banner at top of My Checklists tab
async function renderSyncBanner() {
  const db = await getDB();
  const entries = await idbGetAll(db, 'submitQueue');
  const banner = document.getElementById('sync-banner');
  if (!banner) return;
  if (entries.length === 0) { banner.style.display = 'none'; return; }
  banner.style.display = 'block';
  banner.textContent = entries.length + ' submission' + (entries.length > 1 ? 's' : '') + ' pending sync';
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `MOCK_TEMPLATES` in-memory array | `GET /api/v1/workflow/templates` | Phase 10 | Templates survive page reload |
| `MOCK_RESPONSES` dict | Auto-save to `submission_responses` | Phase 10 | Progress persists across devices |
| In-memory `PENDING_APPROVALS` | `checklist_submissions` table | Phase 10 | Approval state survives server restart |
| Mock submit (`SUBMITTED_TEMPLATES[id] = true`) | `POST /api/v1/workflow/submitChecklist` | Phase 10 | Real audit trail and user attribution |
| No offline support | IndexedDB queue + `online` drain | Phase 10 | Field work survives connection loss |

**Deprecated/outdated after Phase 10:**
- `MOCK_TEMPLATES`: deleted from workflows.html
- `MOCK_RESPONSES`, `FAIL_NOTES`, `SUBMITTED_TEMPLATES`, `PENDING_APPROVALS`, `APPROVED_SUBMISSIONS`, `REJECTED_SUBMISSIONS`, `REJECTION_FLAGS`, `WAS_REJECTED`: all deleted
- `MOCK_CURRENT_USER`, `MOCK_USERS`: replaced by `/api/v1/me` and `/api/v1/me/apps`
- All mock functions (`submitChecklist` in JS, `setYesNo`, etc.): replaced by `api()` calls

---

## Open Questions

1. **Draft response storage on the server (D-21)**
   - What we know: D-21 requires auto-save per field tap. The proposed approach (NULL `submission_id` draft rows) handles it without a separate table.
   - What's unclear: Whether a TTL/cleanup mechanism is needed in Phase 10 or can be deferred. Draft rows from incomplete sessions will accumulate.
   - Recommendation: Add `draft_started_at TIMESTAMPTZ DEFAULT now()` to `submission_responses`. In the `myChecklists` GET handler, only return draft responses from today. A cleanup job can be added in Phase 11+. Mark this as a known gap.

2. **Playwright test server setup (D-19)**
   - What we know: 54 existing tests hit static files. D-19 requires rewriting against a real Go server. `playwright.config.js` has a `webServer` option.
   - What's unclear: Whether a separate test DB URL is already configured or needs to be set up. The Taskfile has `db-start` but no `db-test` target.
   - Recommendation: Add a `db-test` Taskfile target that spins up a separate Postgres container on port 5433. Playwright `webServer` starts the Go server with `DB_URL` pointing at the test DB. `beforeEach` fixture seeds and tears down data via direct SQL using pgx.

3. **`docs/user-management-api.md` schema misalignment**
   - What we know: The doc has array columns for assignees/approvers; CONTEXT.md D-05 mandates a junction table. The doc has `enabled BOOLEAN`; D-07 uses `archived_at`.
   - What's unclear: The sub-steps design in the doc uses inline JSONB; D-03 uses `parent_field_id`. The doc is pre-discussion and may confuse implementers.
   - Recommendation: The planner should note that CONTEXT.md decisions are authoritative. The migration SQL in RESEARCH.md supersedes the table definitions in `docs/user-management-api.md`. Do not update the doc during Phase 10 (out of scope).

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25 | Backend compilation | Verify at run time | See go.mod | — |
| PostgreSQL | All DB operations | Assumed (Phase 9 running) | 13+ (Taskfile default) | — |
| Docker | `task db-start` | Assumed (Phase 9 running) | — | Manual Postgres install |
| Playwright | E2E test rewrite (D-19) | Assumed (existing npm test works) | See package.json | — |
| `crypto.randomUUID()` | IndexedDB idempotency keys | Chrome 92+, Safari 15.4+, Firefox 95+ | Browser built-in | `Math.random()` fallback (lower entropy, acceptable for food truck scale) |

**Missing dependencies with no fallback:** None identified. All Phase 9 infrastructure is assumed running.

---

## Sources

### Primary (HIGH confidence)

- Direct code inspection: `backend/internal/auth/handler.go` — handler closure pattern, JSON encode/decode
- Direct code inspection: `backend/internal/me/handler.go` — repository query pattern, pgxpool.Query
- Direct code inspection: `backend/internal/db/db.go` — goose migration pattern, SeedHQApps startup pattern
- Direct code inspection: `backend/internal/db/migrations/0001_users.sql` — migration file format
- Direct code inspection: `backend/internal/config/config.go` — YAML config loading pattern
- Direct code inspection: `backend/cmd/server/main.go` — chi router setup, route group pattern
- Direct code inspection: `workflows.html` — all mock data structures and their exact shapes
- Direct code inspection: `docs/user-management-api.md` — existing API design, checklist schema first draft
- Direct code inspection: `.planning/phases/10-workflows-api/10-CONTEXT.md` — all locked decisions D-01 through D-26
- Direct code inspection: `.planning/STATE.md` — Background Sync API ruled out (zero iOS Safari support)

### Secondary (MEDIUM confidence)

- MDN Web API documentation (from training knowledge): IndexedDB API, `online` event, `crypto.randomUUID()` — browser compatibility ranges verified against project's known iOS Safari target
- pgx v5 docs (from training knowledge): `pool.Begin()`, `tx.QueryRow()`, `ON CONFLICT ... DO UPDATE RETURNING` pattern

### Tertiary (LOW confidence, flagged)

- Playwright `webServer` config option for auto-starting Go test server — not verified against current Playwright version in project; check `package.json` for version before planning the test infrastructure tasks

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod; no new dependencies
- Schema design: HIGH — locked in CONTEXT.md D-01 through D-07; canonical `docs/user-management-api.md` exists as reference
- Architecture patterns: HIGH — directly derived from existing Phase 9 code
- IndexedDB pattern: MEDIUM — based on MDN API + project constraint (no library, per CLAUDE.md static-only convention)
- Pitfalls: HIGH — derived from direct code inspection of constraints and cross-referencing decisions

**Research date:** 2026-04-15
**Valid until:** 2026-05-15 (stable Go/pgx/goose ecosystem; IndexedDB browser APIs are stable)
