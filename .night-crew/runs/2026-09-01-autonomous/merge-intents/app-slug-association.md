# Merge intent — `app-slug-association` (Card 8, Track B)

Closes **B-160** (E-KR4). Branch `wo-app-slug-association` off `overnight-20260901`.

## What the card is

HQ stores NO template→app association, so `app_slug` is a hardcoded CONSTANT in
the one sync projection writer that exists — `backend/internal/sync/spikec_relay.go`
(`cfg.AppSlug`, banner finding #4 = spike B's finding #1). Every projected row
claims the same app. The card gives the run standing authority to decide where the
association lives and to populate the writer from it.

## E-KR4 — the ASSOCIATION-HOME DECISION (recorded here under standing authority)

**Decision: a nullable FK column `checklist_templates.app_id UUID REFERENCES hq_apps(id) ON DELETE SET NULL`.**

**Where:** a column on `checklist_templates` (migration `0006_checklist_templates.sql`),
referencing `hq_apps(id)` — the same FK target `app_permissions.app_id` already uses
(`0005_app_permissions.sql:5`).

**Why this and not the alternatives:**

- **A template belongs to exactly one app.** Checklist templates ARE the Operations
  tool's data — the workflows/checklist engine is the `operations` app (`hq_apps` slug
  `operations`, seeded in `backend/internal/db/db.go:73`). One template → one app is a
  plain 1:N, so a column on the child (`checklist_templates`) is the normal-form home.
  A dedicated mapping table would model a many-to-many the product does not have, and
  would add a join every projection writer must remember — the exact kind of drift this
  card exists to remove.
- **`app_id` (FK), not `app_slug` (denormalised text).** `hq_apps.slug` is `UNIQUE NOT NULL`
  but mutable text; the id is the stable key every other app-scoped table already joins on
  (`app_permissions.app_id`, the `hq_apps(id)` references in `0005`/`0024`). Storing the id
  and resolving the slug at read time keeps one source of truth for the slug and means a
  future slug rename does not have to rewrite `checklist_templates`. The sync contract's
  wire field is still `app_slug`; the writer resolves `app_id → hq_apps.slug` at projection
  time (one extra join, mirroring how `spikec_relay.go` already joins out to
  `checklist_fields` for the field label).
- **`ON DELETE SET NULL`, nullable.** Deleting an app should not cascade-delete its
  checklist templates (they carry crew-entered submissions); nulling the association is the
  safe failure. Nullable also lets the column exist before every template is guaranteed to
  have an app in every environment.

**Backfill:** every existing `checklist_templates` row → the `operations` app
(`UPDATE ... SET app_id = (SELECT id FROM hq_apps WHERE slug = 'operations')`). Rationale:
the checklist engine is the Operations tool, so every template that exists today is an
Operations template — this is exactly the constant (`operations`) all five spike shell
scripts already hardcode (`SPIKE_C_APP_SLUG=operations` / `sync-dev-up.sh:66`), so the
backfill reproduces today's behaviour rather than changing any projected row. If the
`operations` app row is somehow absent (a bare test DB that skipped the `db.go` seed), the
subquery yields NULL and the UPDATE is a harmless no-op — the FK is nullable by design.

## NOT a product fork — why this is the run's call, not the operator's

The home decision is a schema-shape choice between a column and a table, both invisible to
the operator: no new operator-facing concept, no UI to assign templates to apps (templates
are already, implicitly, Operations-only), no template-belongs-to-multiple-apps requirement.
The projected `app_slug` value is unchanged for every row that exists today (`operations`),
so nothing the operator sees changes. This is the engineer/PM-level home decision the
roadmap text authorises, not a park.

## Implementation

1. **Migration 0076** (`0076_checklist_template_app.sql`, goose `BEGIN;/COMMIT;` style like
   `0075`): add `checklist_templates.app_id UUID REFERENCES hq_apps(id) ON DELETE SET NULL`,
   an index on it, backfill existing rows to `operations`.
2. **`spikec_relay.go`**: extend `relayOne`'s re-read to resolve the projected row's
   `app_slug` from the association —
   `submission_responses.field_id → checklist_fields.section_id → checklist_sections.template_id → checklist_templates.app_id → hq_apps.slug`
   — and write that per-row value into the projected row. Remove the `AppSlug` field from
   `SpikeCRelayConfig`, its empty-guard, and its two other uses (row map + log line). 0
   hardcoded `app_slug` constants remain in the writer (grep-provable).
3. **`cmd/spikec-relay/main.go`**: drop the `AppSlug: os.Getenv("SPIKE_C_APP_SLUG")` line.
   The shell scripts that still export `SPIKE_C_APP_SLUG=operations` keep working — the env
   var is simply no longer read, and the derived value is `operations` for every seeded
   template anyway.
4. **Red-first**: (a) a structural test that greps the projection writer for hardcoded
   `app_slug` string/`cfg.AppSlug` constants — RED on the pre-change tree, GREEN after; and
   (b) a resolver-query unit test seeding two templates on two apps and asserting each
   template's response resolves to its own app's slug (RED against a constant, GREEN against
   the association-driven query).

## Shared files — conflict story

- `internal/sync/spikec_relay.go` — MY change. Disjoint from tonight's other sync edits:
  Card 5 added symbols to `internal/sync/ops.go` (a DIFFERENT file); Card 7 fixed comments in
  `internal/sync/proxy.go` (a DIFFERENT file). No card tonight touches `spikec_relay.go`.
  My edit is confined to `relayOne`, the config struct, and the guard — keep both sides on
  any surprise, but none is expected.
- `cmd/spikec-relay/main.go` — MY change (one line removed). No other card touches it.
- migrations — I add **0076** only (latest was 0075). No renumber, no edit to existing.

## What must survive / what is safe to drop

- SURVIVE: every projected row's `app_slug` value is still `operations` for today's data
  (backfill guarantees it) — no behaviour change, only its SOURCE changes (association, not
  constant). SURVIVE: the spike shell harness keeps running (env var unread, not removed).
- SAFE TO DROP: `SpikeCRelayConfig.AppSlug`, its guard, `SPIKE_C_APP_SLUG` read in main.go.
  Nothing in production references any of them (the whole relay is spike-derived, unwired at
  boot — banner + `grep -rn RunSpikeCRelay backend/cmd/server` = no matches).

## Note for G6 / orchestrator

- The production sync path is **read-through FDW views** (decision 92, migration 0073), NOT
  a written projection — nothing in `cmd/server` runs the relay. This card still discharges
  B-160/E-KR4 because it puts the template→app association in the SCHEMA (0076) and makes the
  one existing projection writer read it, so when the Activity-3 cutover card replaces the
  spike with a tested production relay, the association is already there and grep-clean.
- Footprint is Go + SQL only (`internal/sync`, migrations). No HTML/JS — Playwright N/A by
  footprint.
