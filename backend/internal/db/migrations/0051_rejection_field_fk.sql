-- +goose Up
-- Drop the FK constraint on submission_rejections.field_id because template
-- updates (replaceTemplate) delete and re-create fields with new UUIDs.
-- Rejections reference field IDs from the submission's template_snapshot,
-- which may no longer exist in checklist_fields.
ALTER TABLE submission_rejections DROP CONSTRAINT submission_rejections_field_id_fkey;

-- +goose Down
ALTER TABLE submission_rejections ADD CONSTRAINT submission_rejections_field_id_fkey FOREIGN KEY (field_id) REFERENCES checklist_fields(id);
