# Feature Research: Go Backend for Yumyums HQ PWA

**Domain:** Food truck operations backend — REST API, auth, data persistence, offline sync, receipt ingestion, food cost calculation
**Researched:** 2026-04-15
**Confidence:** HIGH — based on existing API design doc (`docs/user-management-api.md`), PROJECT.md active requirements, and Go/PWA ecosystem research. Feature categories are well-grounded; complexity estimates rely on codebase inspection + research.

---

## Context: What This Milestone Is

v2.0 replaces all hardcoded JS mock data in the existing PWA with a real Go + Postgres backend. The frontend UI is already fully built and validated (v1.0 + v1.1). This milestone is purely a backend and integration milestone — no new UI features.

Six work streams:

1. **Auth flow** — bearer token sessions, login.html wired to real API, invite acceptance
2. **Workflows persistence** — checklist completions, approvals, audit trail, correction loop
3. **Onboarding persistence** — training progress, section sign-off, manager journal entries
4. **Inventory/Purchasing pipeline** — receipt ingestion, real purchase events, food cost intelligence
5. **PWA offline mode** — IndexedDB queuing, service worker sync, conflict resolution
6. **Dev deployment** — Tailscale access to Hetzner box running Go server

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features the backend must provide for the app to function at all. Without these, the app is broken for real users.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Login / logout (bearer token auth) | App is useless without identity — every protected endpoint requires a valid session | LOW | `POST /api/v1/auth/login` + `POST /api/v1/auth/logout`; SHA-256 token hash in `sessions` table; 8-hour expiry; fully designed in `docs/user-management-api.md` |
| Invite flow (token-based invite acceptance) | Owner needs to add crew members who don't have passwords yet | MEDIUM | `POST /api/v1/users/invite` → email with one-time token → `POST /api/v1/auth/accept-invite`; `invite_tokens` table with `used_at` guard; email delivery is new complexity |
| Session validation middleware | Every protected endpoint needs auth checking | LOW | Go middleware reads `Authorization: Bearer <token>`, SHA-256s it, looks up `sessions`; attach user to request context; standard Go middleware chain pattern |
| Current user endpoint (`GET /api/v1/me`) | Frontend needs to know who is logged in — role, display name, permissions | LOW | Returns user from session context; drives tile filtering and role-based UI |
| App permissions endpoint (`GET /api/v1/me/apps`) | PWA `index.html` calls this on load to filter which tiles to show | LOW | Evaluates superadmin/role/user_id grant hierarchy against `app_permissions` table; returns only accessible app slugs |
| User CRUD (list, invite, update role, delete) | Owner/admin must manage team — otherwise the app cannot be onboarded for real use | MEDIUM | 5 endpoints; admin-only; superadmin guarded at API layer (not in DB); `PATCH /api/v1/users/:id` for role/display_name updates |
| Password reset trigger | Crew forgets passwords — owner must be able to reset without manual DB access | LOW | `POST /api/v1/users/:id/reset-password` sends email with new invite-style token; same token infrastructure as invite |
| Checklist template persistence (create, read, update, delete) | Builder tab currently saves nothing — crew sees only hardcoded templates on reload | MEDIUM | 4 endpoints; version bumping on edit when submissions exist; soft-delete (`enabled = false`) to preserve audit trail; 3-table write (template + sections + fields) must be atomic |
| Checklist fill-out submission (`POST /api/v1/checklists/submissions`) | Core crew workflow — completions must be saved or accountability is fiction | MEDIUM | Writes `checklist_submissions` + `submission_responses` + `submission_fail_notes` atomically; handles requires_approval routing; returns status |
| Today's checklists for current user (`GET /api/v1/checklists/today`) | Crew tab shows only checklists assigned to them and active today | LOW | Filter by `active_days` (day-of-week), `assigned_to_roles`, `assigned_to_users`; include `existing_submission` for resume |
| Approval actions (approve, reject, unapprove) | Approval flow is a core feature — without persistence the Approvals tab is a stub | MEDIUM | 3 endpoints; all write to `submission_audit_log`; reject creates `submission_rejections` rows; unapprove returns to `pending` |
| Photo upload to object storage | Photo fields in checklists require a real upload endpoint — data URLs are not an option at scale | MEDIUM | `POST /api/v1/checklists/submissions/:id/upload-photo`; multipart form; write to DO Spaces; return URL for storage in `submission_responses.value` |
| Onboarding template persistence | Onboarding builder saves nothing currently — same gap as workflow builder | MEDIUM | Analogous to checklist templates: templates, sections (with video fields, FAQ fields), CRUD endpoints |
| Onboarding progress persistence | Training progress per crew member must be saved | MEDIUM | Mark sections complete, record sign-off by manager, store journal entries |
| Inventory purchase event persistence | Inventory tab currently uses hardcoded mock data — real operations need real purchase history | MEDIUM | CRUD for purchase events and line items; drives all inventory analytics; mirrors Baserow schema from v1.1 mock |

---

### Differentiators (Competitive Advantage)

Features that go beyond basic persistence and provide specific value for food truck operations at this scale.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Receipt ingestion pipeline (photo → structured purchase event) | Owner photographs a receipt after a shopping trip; the backend extracts line items, totals, and vendor — eliminates manual data entry | HIGH | Multipart upload endpoint → image preprocessing → OCR API (Mindee or Veryfi for structured receipt parsing) or GPT-4o vision → structured output mapped to purchase event schema → human review/confirm UI; the OCR/LLM call is the uncertain piece |
| Food cost calculation (spending per ingredient category as % of revenue) | Owner can see if beef costs are creeping up as a fraction of sales — connects purchasing to margins | HIGH | Requires revenue data (not yet in scope) OR operates purely on spending trends as proxy; AI-assisted ingredient ratio derivation means categorizing line items by spend category and computing running percentages; may need a menu/recipe BOM to be meaningful |
| Offline-first checklist submission queue | Crew members on the truck may have no signal; submissions must queue locally and sync when back online | HIGH | IndexedDB queue in vanilla JS (`idb-keyval` or hand-rolled); service worker `sync` event registration; idempotency key (client UUID) on each submission to prevent doubles; conflict resolution: last-write-wins on field-level responses since single user fills each checklist |
| Superadmin config bootstrap from YAML | Superadmin identity is managed in `config/superadmins.yaml`, not the DB — prevents accidental DB grants to owner account | LOW | Go startup reads YAML, holds in-memory set; API layer checks email against set before consulting DB role; clean separation between bootstrap config and runtime data |
| Tailscale dev deployment | Backend accessible to developer from any device over Tailscale — enables real PWA testing on actual phones without public IP | LOW | Tailscale installed on Hetzner box + developer machine; Caddy serves Go binary over HTTPS; no firewall changes needed; standard Tailscale pattern |
| Template versioning with frozen history | When a template is edited, existing submissions reference the frozen version — audit trail stays coherent even as templates evolve | MEDIUM | `version` integer on template; bump on edit; if submissions exist against current version, freeze it and continue with new version; query joins submission to template version at submission time |
| Item-level correction loop persistence | Rejected items must be visible on the crew's device with manager comments and photo requirements — this is the accountability payoff | MEDIUM | `submission_rejections` table rows returned with `GET /api/v1/checklists/today` `existing_submission`; frontend already knows how to render rejected items (UI validated in v1.0) |
| Email delivery for invites and password resets | Without email, invite flow and password reset are dead ends | MEDIUM | Single `POST /email` call via SMTP or transactional email API (Resend or Postmark — both have straightforward Go clients); email templates are plain text for v1; HTML later |

---

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| JWT access + refresh tokens | "Stateless auth is better" sounds correct | JWT refresh token rotation adds complexity (rotate on use, invalidation on logout requires a denylist, short-lived AT + long-lived RT means two token types to manage) — for a 1-5 person internal app this is over-engineering; the DB lookup cost of opaque token validation is negligible | Opaque bearer token stored with SHA-256 hash in `sessions` table; 8-hour session; logout deletes row; `sessions` table indexed on `token_hash` — O(1) lookup, simple to reason about |
| Real-time push notifications for checklist approvals | "Alert me when it's approved" is natural | Background Push requires a service worker push subscription, VAPID key pair, push payload encryption, and browser permission prompt — Safari PWA push support is inconsistent on older iOS versions | Optimistic polling on the Approvals tab (`GET /api/v1/checklists/approvals` on tab focus) — adequate for a crew of 1-5 with no high-frequency approvals |
| Full-text workflow search across submissions | "Search for a specific past submission" | Adds query complexity and full-text index to an already complex schema; submissions are time-ordered and user-scoped — timeline browsing is sufficient | Filter submissions by date range and user on `GET /api/v1/checklists/submissions`; indexed on `submitted_by` + `submitted_at` |
| Webhook events for external integrations | "Pipe approval events to Slack" | Adds outbound delivery queue, retry logic, signature verification — scope expansion not justified at 1-5 person scale | Manual Tailscale-accessible API is sufficient; integrations can poll if needed |
| Multi-tenant namespace separation | "Build it right for the future" | Single food truck operation; tenant scoping adds a `tenant_id` column to every table, every query, and every index — complexity with zero current benefit | Single-tenant DB; namespace at the Postgres level (separate DB per future tenant) if multi-tenancy becomes real |
| Real-time collaborative checklist completion | "Two crew members fill the same checklist at once" | Requires websocket or SSE, optimistic locking, and last-write-wins at the field level in real time — far more complex than the actual workflow; each checklist is assigned to one person | Submissions are single-owner; if a manager needs to help, they do a new submission; the UI already models this correctly |
| Recipe BOM and menu-level food cost | "Connect purchasing to recipes" | Requires a separate `menu_items`, `recipes`, and `ingredients` data model with unit-of-measure conversions — doubles the scope of the food cost feature | Spending-by-category-as-%-of-revenue gives directional food cost insight without BOM; BOM is a future milestone if the owner asks for it |
| Barcode scanning for purchase entry | "Scan items when receiving" | Requires camera permission flow, barcode library, and a food item database with UPCs — large scope for a 1-5 person truck; existing Baserow data entry is good enough | Receipt ingestion pipeline (photo of receipt) is a better leverage point — one photo per shopping trip vs. scanning every item |
| Soft delete with restore UI for all entities | "Don't lose data" | Soft delete is appropriate for templates (preserve submissions), but exposing restore UI for users and submissions adds admin surface area — YAGNI for this scale | Soft delete on templates only; hard delete on users (sessions cascade); keep `submission_audit_log` for immutable history |

---

## Feature Dependencies

```
Auth (login / sessions)
    └──required-by──> Every protected endpoint

User CRUD
    └──requires──> Auth

App Permissions
    └──requires──> User CRUD (need users to grant permissions to)
    └──required-by──> PWA tile filtering (GET /me/apps)

Checklist Template CRUD
    └──requires──> Auth
    └──required-by──> Checklist fill-out (must have templates to fill)
    └──required-by──> Today's checklists endpoint

Checklist Submission
    └──requires──> Checklist Template CRUD (template + version must exist)
    └──requires──> Photo Upload (for photo-type fields)
    └──required-by──> Approval flow

Photo Upload (object storage)
    └──requires──> Auth
    └──required-by──> Checklist Submission (photo field values)
    └──required-by──> Receipt Ingestion (receipt photos)

Approval Flow (approve/reject/unapprove)
    └──requires──> Checklist Submission (submissions must exist)
    └──requires──> Audit Log (all actions recorded)

Item-Level Correction Loop
    └──requires──> Approval Flow (rejection creates correction_items)

Onboarding Template CRUD
    └──requires──> Auth (parallel to Checklist Template CRUD)

Onboarding Progress
    └──requires──> Onboarding Template CRUD

Receipt Ingestion Pipeline
    └──requires──> Photo Upload (infrastructure)
    └──requires──> Purchase Event CRUD (writes structured output)
    └──depends-on──> External OCR/LLM API (Mindee, Veryfi, or GPT-4o)

Food Cost Calculation
    └──requires──> Purchase Event CRUD (purchase history data)
    └──enhanced-by──> Receipt Ingestion (real data vs manual entry)
    └──depends-on──> Revenue data (not yet in scope; without it, only spending % trends available)

PWA Offline Queue (IndexedDB + SW sync)
    └──requires──> Checklist Submission (the endpoint being queued against)
    └──requires──> Idempotency key support on submission endpoint
    └──enhanced-by──> Background Sync API (Chrome/Edge only; iOS fallback is online listener)

Email Delivery
    └──required-by──> Invite Flow (can't complete invite without email link)
    └──required-by──> Password Reset (same)

Tailscale Dev Deployment
    └──requires──> Go server running on Hetzner (deployment target)
    └──enables──> Real phone testing of PWA against real API
```

### Dependency Notes

- **Auth is the absolute prerequisite.** Nothing else can be built or tested meaningfully without working sessions. Auth goes in Phase 1.
- **Template CRUD before Submission.** Submission POSTs reference a template UUID and version; templates must exist in the DB first. Both go in the same phase but template endpoints must be implemented first.
- **Photo Upload before full Checklist Submission.** Photo fields require a URL already in storage; the upload endpoint must exist before photo-type field submission works end-to-end.
- **Receipt Ingestion is high-value but high-uncertainty.** It depends on an external OCR API whose structured output quality is not guaranteed. It should be its own phase with a fallback (manual entry UI) if OCR reliability is poor.
- **Food Cost depends on what "cost" means.** Without revenue data, food cost % is not calculable; only spending-by-category (already in v1.1 mock) is available. Phase-gate food cost behind a decision on revenue data source.
- **Offline sync conflicts are low-risk for this domain.** Each checklist is single-owner; two users never fill the same submission concurrently. Conflict resolution is last-write-wins on field responses, keyed by idempotency UUID. No CRDTs needed.

---

## MVP Definition

### Launch With (v2.0 — first working backend)

Minimum backend for real crew use. Hardcoded mock data is replaced everywhere that matters operationally.

- [ ] **Auth (login, logout, session middleware)** — without this, nothing else is accessible
- [ ] **Invite flow + email delivery** — owner cannot add crew without it; email is the critical path
- [ ] **User CRUD** — owner must manage the team
- [ ] **App permissions** — PWA tile filtering requires this; users.html Access tab must work
- [ ] **Checklist template CRUD** — Builder tab must persist templates
- [ ] **Checklist fill-out + submission** — My Checklists must save completions
- [ ] **Today's checklists for current user** — crew must see their assigned checklists
- [ ] **Approval flow (approve/reject/unapprove + audit log)** — Approvals tab must work
- [ ] **Item-level correction loop** — rejection + re-submit cycle; this is the accountability core
- [ ] **Photo upload to DO Spaces** — photo fields are already in the UI; must work
- [ ] **Onboarding template + progress persistence** — My Trainings tab must save progress
- [ ] **Purchase event + line item CRUD** — Inventory History tab must show real data
- [ ] **Tailscale dev deployment** — required to test real PWA on real phone before declaring done

### Add After Validation (v2.x)

Add once the core backend is working and the owner is using it on a real device.

- [ ] **PWA offline queue (IndexedDB + SW sync)** — trigger: crew reports losing a submission while on the truck (no signal)
- [ ] **Receipt ingestion pipeline** — trigger: owner asks to stop manually entering purchases; validate OCR quality on real receipts first
- [ ] **Password reset flow** — trigger: first time a crew member forgets their password (low urgency for small team where owner can assist)
- [ ] **Stock estimation and reorder suggestions backed by real data** — trigger: once 30+ days of real purchase events exist

### Future Consideration (v2+)

- [ ] **Food cost calculation (spending as % of revenue)** — requires revenue data source decision (POS integration or manual entry)
- [ ] **Recipe BOM / menu-level food cost** — requires menu data model; separate milestone
- [ ] **Push notifications for approval events** — requires VAPID key infrastructure; not justified until crew size grows
- [ ] **AI-assisted food cost analysis** — defer until real data pipeline (receipt ingestion) is stable for 60+ days

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Auth (login/logout/sessions) | HIGH | LOW | P1 |
| Session middleware | HIGH | LOW | P1 |
| Invite flow + email | HIGH | MEDIUM | P1 |
| User CRUD | HIGH | MEDIUM | P1 |
| App permissions | HIGH | LOW | P1 |
| Checklist template CRUD | HIGH | MEDIUM | P1 |
| Checklist submission | HIGH | MEDIUM | P1 |
| Today's checklists endpoint | HIGH | LOW | P1 |
| Approval flow (approve/reject/unapprove) | HIGH | MEDIUM | P1 |
| Item-level correction loop | HIGH | MEDIUM | P1 |
| Photo upload (DO Spaces) | HIGH | MEDIUM | P1 |
| Onboarding template CRUD | HIGH | MEDIUM | P1 |
| Onboarding progress persistence | HIGH | MEDIUM | P1 |
| Purchase event CRUD | HIGH | LOW | P1 |
| Tailscale dev deployment | HIGH | LOW | P1 |
| Audit log (submission_audit_log) | MEDIUM | LOW | P1 (embedded in approval endpoints) |
| Password reset flow | MEDIUM | LOW | P2 |
| PWA offline queue (IndexedDB) | MEDIUM | HIGH | P2 |
| Receipt ingestion pipeline | HIGH | HIGH | P2 |
| Stock estimation (real data) | MEDIUM | LOW | P2 |
| Food cost % calculation | HIGH | HIGH | P3 |
| Recipe BOM | LOW | HIGH | P3 |
| Push notifications | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for v2.0 launch — app unusable without it
- P2: Add post-launch when triggered by real usage
- P3: Future milestone; requires additional data model or infrastructure decisions

---

## Auth Flow — Detailed Spec

This is the entry point for every user. The design is already specified in `docs/user-management-api.md` but is worth summarizing here for clarity:

### Normal login
1. `POST /api/v1/auth/login` with email + password
2. Server finds user by email, bcrypt-compares password_hash
3. Generates 32-byte random token, stores SHA-256 hash in `sessions`
4. Returns raw token + expiry + user object
5. Frontend stores token in `localStorage` as `hq_token`
6. All subsequent requests: `Authorization: Bearer <token>` header

### Invite acceptance
1. Admin calls `POST /api/v1/users/invite` → server creates `invited` user + `invite_tokens` row
2. Server emails `https://hq.yumyums.com/accept-invite?token=<raw>` link
3. User lands on `login.html` in accept-invite mode (token in query string)
4. Frontend calls `POST /api/v1/auth/accept-invite` with token + chosen password
5. Server marks token used, sets user active + password_hash, creates session
6. Returns same shape as login — frontend stores token, redirects to index.html

### Session expiry
- Sessions expire after 8 hours (configurable)
- Frontend receives 401 → redirects to `login.html`
- Tokens are NOT refreshed automatically — user must re-login

### Superadmin
- Email exists in `config/superadmins.yaml` (YAML, read at startup)
- Server holds an in-memory set of superadmin emails
- Login works via same flow (password stored in YAML or env secret)
- API layer checks in-memory set before consulting DB role — superadmin bypasses role checks

---

## Checklist Submission — State Machine

The submission lifecycle is a state machine. Understanding it is critical for backend correctness:

```
(crew submits)
     |
     v
  [pending]  ←── (unapprove)
     |
  requires_approval=true
     |
  manager reviews
     |
  ┌──┴──────────────────┐
  |                     |
[approved]          [rejected]
                        |
                   crew corrects
                        |
                   new submission
                        |
                     [pending]
                        |
                   manager re-approves
                        |
                    [approved]

If requires_approval=false:
(crew submits) → [completed] directly
```

Backend rules:
- `approve` valid on `pending` only
- `reject` valid on `pending` only
- `unapprove` valid on `approved` only; returns to `pending`
- On rejection: create `submission_rejections` rows; original submission rows preserved (audit)
- On correction: new `checklist_submissions` row (not UPDATE); old rejected row stays
- All transitions write to `submission_audit_log`

---

## Offline Sync — Conflict Resolution Strategy

Conflict risk for this domain is LOW because:
- Each submission is created by one user (no collaborative editing)
- A checklist can only be submitted once per user per day (today's template + assignee)
- If a user submits offline and then offline again before sync, it's the same submission — idempotency key prevents duplicates

**Recommended approach: last-write-wins with idempotency key**

1. Frontend generates UUID (`crypto.randomUUID()`) for each submission attempt
2. UUID sent as `Idempotency-Key` header on `POST /api/v1/checklists/submissions`
3. Server stores idempotency key in `checklist_submissions` table (indexed)
4. If a duplicate request arrives with the same key, server returns the existing submission (201 → 200)
5. IndexedDB queue removes entry only on 200/201 from server
6. If offline, queue retries on `online` event (Safari-safe) AND Background Sync event (Chrome/Edge bonus)

**iOS Safari limitation:** Background Sync API is not supported. The fallback is a `window.addEventListener('online', retryQueue)` handler — adequate because iOS users typically have signal before opening the checklist app.

---

## Receipt Ingestion — Pipeline Design

The owner photographs a receipt. The backend turns it into a purchase event.

### Pipeline stages
1. **Upload** — owner taps "Add Receipt" in Inventory tab → multipart POST to `/api/v1/receipts/upload`
2. **Preprocessing** — server validates image (MIME type, max size), resizes if needed
3. **OCR extraction** — call external API: Mindee (`receipt` document type) or Veryfi; both return structured JSON with vendor, date, total, line items
4. **Mapping** — server maps OCR output to purchase event schema; line items to `Purchase` rows; vendor name fuzzy-matched to `Vendor` table
5. **Review** — server returns draft purchase event to frontend for human confirmation/correction (not auto-committed)
6. **Commit** — owner confirms or edits → `POST /api/v1/purchases` creates real purchase event

### Uncertainty flags
- OCR accuracy on handwritten or thermal-faded receipts is poor — always require human review before commit (never auto-commit)
- Vendor name matching is heuristic — fuzzy match against known vendors, present top candidate, let owner confirm
- Line item names may not match `PurchaseItemGroup` names — require owner to map or create new item groups

### Recommended OCR API: Mindee
- Dedicated receipt document type with structured line item extraction
- Simple REST API with Go client library
- Pricing: free tier (250 pages/month) is adequate for a single food truck
- Alternative: Veryfi (better line item accuracy, higher price); GPT-4o vision (flexible but less structured output)

---

## Food Cost Calculation — What's Realistic at This Stage

True food cost % = (food spend / revenue) × 100. Without revenue data, this is not calculable.

**What IS calculable from purchase data alone:**
- Spending by category as absolute dollars (already in v1.1 mock)
- Month-over-month spending trend per category (already in v1.1 mock)
- Average price per item across purchase events (detects supplier price increases)
- Spending ratio between categories (e.g., "beef is 62% of total food spend")

**Recommendation for v2.0:** Implement category spending ratios and price trend detection. These are high-value and calculable from purchase data alone. Label them "Spending Insights" not "Food Cost %" to avoid misleading the owner. Food cost % requires a revenue data source decision — defer to v3.

---

## Phase-Specific Complexity Flags

| Phase Topic | Complexity Driver | Note |
|-------------|-----------------|------|
| Auth + email delivery | Email is the blocking dependency | SMTP or transactional API must be configured before invite flow can be tested end-to-end; test with a real email address in dev |
| Photo upload | DO Spaces CORS config | Spaces bucket must allow the PWA origin; multipart form parsing in Go is standard but Spaces SDK setup takes time |
| Checklist template CRUD | Atomic 3-table write | Template + sections + fields must be written in a single transaction; partial writes corrupt the template |
| Offline sync | iOS Safari Background Sync absence | Must implement `online` event fallback; do not rely solely on Background Sync |
| Receipt ingestion | External API latency and accuracy | OCR calls take 1-3 seconds; must show loading state; human review step is non-negotiable |
| Onboarding persistence | Data model not yet designed | `docs/user-management-api.md` covers workflows but not onboarding; onboarding tables need to be designed before Phase work begins |

---

## Dependencies on Existing HQ Features

| Existing Feature | How Backend Depends On It |
|-----------------|--------------------------|
| `workflows.html` submission UI | Backend submission endpoint must match the exact JSON shape the frontend already sends (field types, `fail_notes` structure, `value` format per field type) |
| Approval UI (approve/reject/unapprove buttons) | Backend approval endpoints must return the submission status object the frontend expects |
| `users.html` Access tab | App permissions API must return the same `role_grants` / `user_grants` shape the UI uses |
| `inventory.html` History tab | Purchase event API must return data that maps to the mock `PURCHASE_EVENTS` + `PURCHASES` structure |
| `onboarding.html` My Trainings tab | Onboarding progress API must return section-level completion state matching the frontend's section model |
| SW + localStorage (`hq_token`) | Auth flow must write token to `localStorage.hq_token`; 401 must redirect to `login.html` |

---

## Sources

- `docs/user-management-api.md` — existing API design covering auth, users, app permissions, checklist submission, approval flow (HIGH confidence — written for this project)
- `.planning/PROJECT.md` — v2.0 active requirements list (HIGH confidence)
- [Mindee Receipt OCR API](https://developers.mindee.com/docs/receipt-ocr) — structured receipt extraction API, free tier 250 pages/month
- [Veryfi Receipt OCR](https://www.veryfi.com/receipt-ocr-api/) — alternative OCR with better line-item accuracy
- [Receipt OCR Pipeline patterns — ExtractBill](https://www.extractbill.com/blog/receipt-ocr-api-automated-expense-reports) — pipeline stage patterns (MEDIUM confidence)
- [IndexedDB offline queue — Medium](https://medium.com/@11.sahil.kmr/offline-first-by-design-pwa-indexed-db-and-a-reliable-queue-775605b3d76c) — queue + idempotency patterns (MEDIUM confidence)
- [Background Sync browser support 2025 — LogRocket](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/) — confirmed Safari does not support Background Sync (HIGH confidence, 2025)
- [Food cost % benchmarks — Supy](https://supy.io/blog/how-to-calculate-food-cost-percentage) — restaurant food cost target ranges 25-35% (MEDIUM confidence)
- [opaque bearer tokens vs JWT — Go REST API patterns — Medium](https://medium.com/@janishar.ali/how-to-architecture-good-go-backend-rest-api-services-14cc4730c05b) — session table lookup pattern preferred over JWT for simplicity (MEDIUM confidence)
- [PWA offline sync data synchronization strategies — GTCSys](https://gtcsys.com/comprehensive-faqs-guide-data-synchronization-in-pwas-offline-first-strategies-and-conflict-resolution/) — conflict resolution strategy overview (MEDIUM confidence)

---
*Feature research for: Yumyums HQ v2.0 Go Backend*
*Researched: 2026-04-15*
