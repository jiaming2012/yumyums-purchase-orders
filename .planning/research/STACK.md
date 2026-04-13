# Technology Stack

**Project:** Yumyums HQ — Workflow/Checklist Engine
**Researched:** 2026-04-12
**Scope:** Frontend (current milestone) + future Go + Postgres backend

---

## Context and Constraints

The existing app is plain HTML/CSS/vanilla JS with no build step. This milestone adds a two-tab
workflow feature (fill-out + template builder) to that same codebase. No framework can be
introduced. All recommendations here must work as either:

- a CDN `<script>` tag, or
- a hand-rolled vanilla JS module

The future backend is a separate project (Go + Postgres, already API-designed in
`docs/user-management-api.md`). The backend stack is included here to inform roadmap sequencing.

---

## Recommended Stack

### Frontend — Core (No Build Step)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Vanilla JS (ES2020+) | native | All interactivity | Matches existing codebase; no framework allowed |
| CSS custom properties | native | Theming and dark mode | Already established in the app; extend, don't replace |
| HTML5 file input (`capture="camera"`) | native | Photo capture field type | Works on iOS and Android in installed PWA mode without getUserMedia complexity |
| Service Worker (existing) | native | PWA offline shell | Already in repo via `ptr.js` patterns; extend for new pages |

**Confidence:** HIGH — these are the existing conventions, not new choices.

### Frontend — Drag-and-Drop Reorder (Template Builder)

| Technology | Version | CDN Available | Why |
|------------|---------|--------------|-----|
| SortableJS | 1.15.7 | Yes (unpkg, cdnjs) | Framework-agnostic, no jQuery, touch-native, 3,358+ dependent packages on npm, actively maintained. Handles reordering checklist items and sections in the builder. |

**Why not FormKit Drag and Drop:** ~5kb but requires npm/bundler setup. Defeats the no-build-step constraint.

**Why not Formeo:** Opinionated full form-builder with its own data model and UI chrome — too heavy for a custom builder that needs to match the existing HQ design system.

**Why not native HTML5 Drag and Drop API:** Does not handle touch events on iOS without a polyfill. SortableJS wraps this correctly.

**Confidence:** HIGH — SortableJS is the canonical answer for touch-friendly drag reorder in vanilla JS.

CDN:
```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/Sortable/1.15.7/Sortable.min.js"></script>
```

### Frontend — Conditional Logic Engine

| Technology | Version | Size | Why |
|------------|---------|------|-----|
| JsonLogic (JS) | latest (npm) / CDN | ~2kb min+gzip | Stores conditional rules as JSON, evaluable on both frontend and future Go backend. Language-portable: JS and Go implementations exist. Perfect for "if temperature > 40 then show corrective action" patterns. |

**Why JsonLogic over custom logic:** Conditional rules must be stored in the template definition
(JSON in Postgres). A custom evaluator would need to be rebuilt identically in Go when the backend
arrives. JsonLogic rules are data — they serialize, store, and travel between frontend and backend
without translation.

**Why not a full rules engine (json-rules-engine, Drools):** Overkill. JsonLogic covers:
- Field value comparisons (`>`, `<`, `==`, `!=`)
- Boolean combinations (`and`, `or`, `!`)
- Conditional branching (`if`)

That covers 100% of the required use cases: fail triggers, skip logic, day-of-week conditions.

**Confidence:** MEDIUM — JsonLogic is widely used but the specific CDN version was not verified.
Verify at https://cdnjs.com/libraries/json-logic-js before embedding.

### Frontend — No Additional Libraries Needed

| Rejected Library | Why Rejected |
|-----------------|-------------|
| SurveyJS | 900kb+ bundle, React/Angular/Vue only for its builder, would require a bundler |
| Form.io | React-coupled, complex infrastructure for a 4-person food truck team |
| Alpine.js | Tempting but introduces a framework pattern inconsistent with the existing codebase |
| Any CSS framework (Bootstrap, Tailwind) | Existing design system uses CSS custom properties; adding a utility framework creates conflicts |

---

## Template Data Structure

This is the central design decision for the workflow engine. Based on research into Lumiform-style
apps, food safety checklist platforms (GoAudits, BuildArray, Forms On Fire), and the project's
own requirements.

### Template Schema (stored as JSON in Postgres `jsonb` column)

```json
{
  "id": "uuid",
  "version": 1,
  "name": "Opening Checklist",
  "conditions": {
    "days_of_week": ["mon", "tue", "wed", "thu", "fri"]
  },
  "sections": [
    {
      "id": "section-uuid",
      "title": "Equipment",
      "order": 0,
      "items": [
        {
          "id": "item-uuid",
          "type": "checkbox",
          "label": "Fryer temperature checked",
          "required": true,
          "order": 0,
          "conditional_show": null,
          "on_fail": null
        },
        {
          "id": "item-uuid-2",
          "type": "temperature",
          "label": "Walk-in cooler temp (°F)",
          "required": true,
          "order": 1,
          "validation": { "min": 32, "max": 40 },
          "conditional_show": null,
          "on_fail": {
            "action": "show_corrective",
            "corrective_item_id": "item-uuid-3"
          }
        },
        {
          "id": "item-uuid-3",
          "type": "text",
          "label": "Describe corrective action taken",
          "required": false,
          "order": 2,
          "conditional_show": {
            "rule": { "if": [{ ">": [{"var": "item-uuid-2"}, 40] }, true, false] }
          }
        }
      ]
    }
  ]
}
```

**Field types to implement:** `checkbox`, `text`, `yes_no`, `temperature`, `photo`, `timestamp`

**Key design decisions:**
1. `conditional_show.rule` stores a JsonLogic expression — evaluable identically on frontend and backend
2. `on_fail` is a first-class property, not a conditional — fail triggers are separate from skip logic
3. `version` on the template is an integer monotone counter; completed responses store a snapshot of `template_version` so historical submissions stay readable even after template edits
4. Day-of-week conditions live at the template level (which days this template is active), not the item level

### Completion Record Schema

```json
{
  "id": "uuid",
  "template_id": "uuid",
  "template_version": 1,
  "completed_by": "user_id",
  "started_at": "ISO8601",
  "submitted_at": "ISO8601",
  "responses": {
    "item-uuid": { "value": true, "completed_by": "user_id", "at": "ISO8601" },
    "item-uuid-2": { "value": 38.5, "completed_by": "user_id", "at": "ISO8601" }
  }
}
```

**Immutability principle:** Completed responses are write-once. Once submitted, a completion record
is never mutated — corrections require a new completion. This mirrors food safety audit best
practices and makes the audit log trustworthy.

---

## Future Backend Stack (Go + Postgres)

These are design-ready choices for when the backend is built. The API contract already exists in
`docs/user-management-api.md`.

### Core Services

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.22+ | API server | Already chosen; consistent with Temporal infrastructure in `infra/` repo |
| go-chi/chi | v5.2.5 | HTTP router | Zero dependencies (stdlib only), idiomatic Go, active (Feb 2025 release), no magic. Chosen over Gin: Gin adds middleware overhead and magic that isn't needed for a small API |
| jackc/pgx | v5.9.1 | Postgres driver | Fastest Go Postgres driver (300% faster than sqlx in bulk ops), native pgx interface preferred over database/sql for JSONB queries, active (Mar 2026 release) |
| sqlc | 1.30.0 | Type-safe query generation | Write SQL, get Go functions. No ORM magic. Works with pgx/v5. Enforces query correctness at compile time — critical for JSONB template queries |
| golang-migrate/migrate | v4 | Schema migrations | SQL-file based migrations, sequential versioning, CLI + programmatic API, standard choice in the Go ecosystem |
| PostgreSQL | 16 | Persistence | Chosen for `infra/` Temporal stack already; JSONB column for template definitions, standard tables for completions and users |

### Why Not GORM

GORM hides SQL complexity but the workflow data model has JSONB columns and needs precise control
over how template definitions are queried and updated. sqlc + pgx gives full SQL control with
type safety. GORM's JSONB support requires workarounds and the abstractions leak.

### API Design Principles (for future implementation)

- Templates: `GET /templates`, `POST /templates`, `GET /templates/:id`, `PUT /templates/:id`
- Completions: `POST /completions`, `GET /completions?template_id=&date=`
- Template versions are immutable once a completion references them — edits create a new version number
- JSONB `template_snapshot` in the completions table preserves the exact template used at submission time

### Photo Storage

For the photo field type: store photos as uploads to Digital Ocean Spaces (S3-compatible). The
completion record stores the Spaces object URL, not the blob. Do not store images in Postgres.

**Confidence for backend stack:** MEDIUM — versions verified but no backend code exists yet;
implementation may surface pgx/sqlc integration quirks that require adjustment.

---

## PWA Considerations

### Photo Capture (current milestone, mocks only)

Use `<input type="file" accept="image/*" capture="environment">` — this opens the rear camera
directly on iOS and Android in installed PWA mode. No getUserMedia needed for basic capture.
For advanced use (real-time preview, annotations), getUserMedia is the upgrade path but adds
significant complexity.

### Offline Behavior (out of scope now, future)

When offline sync is needed: IndexedDB as the write-ahead log, Background Sync API for queuing
(note: Firefox and Safari do not support Background Sync as of 2025). The fallback for
non-Chromium browsers is polling on reconnect. Last Write Wins (timestamp-based) is sufficient
conflict resolution for single-user checklist data.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Drag reorder | SortableJS 1.15.7 | Native HTML5 DnD | Broken on iOS touch without polyfill |
| Conditional logic | JsonLogic | Custom evaluator | Custom evaluator must be reimplemented in Go; JsonLogic has Go port |
| Conditional logic | JsonLogic | json-rules-engine | Heavier, npm-only, no Go equivalent |
| Photo capture | `<input capture>` | getUserMedia | capture input is simpler, matches mobile idiom, no JS required |
| Go router | chi v5 | Gin | Gin adds magic/overhead; chi is stdlib-compatible |
| Go DB layer | sqlc + pgx/v5 | GORM | GORM hides JSONB control; sqlc enforces SQL correctness at build |
| Migrations | golang-migrate | Atlas, Flyway | golang-migrate is Go-native, no JVM dependency, standard ecosystem choice |

---

## Sources

- [SortableJS npm — v1.15.7, last published 2 months ago](https://www.npmjs.com/package/sortablejs)
- [SortableJS GitHub — no jQuery, touch devices, modern browsers](https://github.com/SortableJS/Sortable)
- [JsonLogic — framework-agnostic, Go + JS implementations](https://jsonlogic.com/)
- [go-chi/chi v5.2.5 — Feb 5, 2025 release](https://github.com/go-chi/chi/releases)
- [jackc/pgx v5.9.1 — Mar 22, 2026](https://github.com/jackc/pgx/tags)
- [sqlc 1.30.0 — pgx/v5 support confirmed](https://docs.sqlc.dev/en/stable/reference/changelog.html)
- [golang-migrate/migrate v4 — active, Postgres support](https://github.com/golang-migrate/migrate)
- [HTML5 camera input for PWA — capture="environment" pattern](https://simicart.com/blog/pwa-camera-access/)
- [IndexedDB + Background Sync offline patterns 2025](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/)
- [pgx vs sqlx performance — 300% faster in bulk inserts](https://dasroot.net/posts/2025/12/go-database-patterns-gorm-sqlx-pgx-compared/)
- [Food safety checklist structure — fail triggers, corrective actions](https://goaudits.com/food/)
- [JsonLogic rules as JSON data — storable + shareable between front/back](https://jsonlogic.com/)
