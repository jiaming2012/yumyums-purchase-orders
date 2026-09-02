-- +goose Up
BEGIN;

-- ===========================================================================
-- 0076 — the template→app association. Card `app-slug-association` (B-160,
-- E-KR4, run 20260901).
-- ===========================================================================
--
-- Before this migration HQ stored NO association between a checklist template
-- and the app it belongs to, so the sync projection writer
-- (backend/internal/sync/spikec_relay.go) had nothing to populate the sync
-- contract's `app_slug` from and hardcoded it as a CONSTANT — banner finding #4,
-- surfacing spike B's finding #1. Every projected row therefore claimed the same
-- app. This column is the home that finding named as open.
--
-- WHY A COLUMN ON checklist_templates (and not a mapping table): a template
-- belongs to exactly one app. The checklist/workflow engine IS the `operations`
-- app, so template → app is a plain 1:N and the child column is its normal-form
-- home. A mapping table would model a many-to-many the product does not have.
--
-- WHY app_id (FK) AND NOT app_slug (text): `hq_apps.slug` is UNIQUE but mutable
-- text; `hq_apps.id` is the stable key every other app-scoped table already
-- joins on (app_permissions.app_id, the hq_apps(id) references in 0005/0024).
-- The writer resolves app_id → hq_apps.slug at projection time, so the slug has
-- one source of truth and a future slug rename does not rewrite this table.
--
-- WHY nullable + ON DELETE SET NULL: deleting an app must NOT cascade-delete its
-- checklist templates (they carry crew-entered submissions); nulling the
-- association is the safe failure. Nullable also lets the column exist before
-- every environment has guaranteed an app for every template.
ALTER TABLE checklist_templates
  ADD COLUMN app_id UUID REFERENCES hq_apps(id) ON DELETE SET NULL;

CREATE INDEX checklist_templates_app_id_idx ON checklist_templates (app_id);

-- Backfill: every template that exists today belongs to the Operations app —
-- the checklist engine IS Operations, which is exactly the constant
-- ('operations') all the spike shell harnesses already hardcode. This
-- reproduces today's projected value; it changes no projected row.
--
-- If the `operations` hq_apps row is absent (a bare DB that skipped db.go's
-- startup seed), the subquery yields NULL and this UPDATE is a harmless no-op —
-- the column is nullable by design.
UPDATE checklist_templates
   SET app_id = (SELECT id FROM hq_apps WHERE slug = 'operations')
 WHERE app_id IS NULL;

COMMENT ON COLUMN checklist_templates.app_id IS
  'The hq_apps this template belongs to (card app-slug-association, B-160). '
  'Source for the sync projection writer''s app_slug — resolved app_id -> hq_apps.slug '
  'at projection time. Nullable; ON DELETE SET NULL. Backfilled to the operations app.';

COMMIT;

-- +goose Down
BEGIN;

DROP INDEX IF EXISTS checklist_templates_app_id_idx;
ALTER TABLE checklist_templates DROP COLUMN IF EXISTS app_id;

COMMIT;
