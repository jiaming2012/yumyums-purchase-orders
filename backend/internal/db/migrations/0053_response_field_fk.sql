-- +goose Up
-- Drop FK on submission_responses.field_id so submitted responses survive
-- template updates (replaceTemplate deletes and re-creates fields with new UUIDs).
-- Submitted responses keep their old field_id which matches the template_snapshot.
ALTER TABLE submission_responses DROP CONSTRAINT submission_responses_field_id_fkey;

-- +goose Down
-- Best-effort restore; may fail if orphaned field_ids exist
ALTER TABLE submission_responses ADD CONSTRAINT submission_responses_field_id_fkey FOREIGN KEY (field_id) REFERENCES checklist_fields(id);
