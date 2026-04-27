-- +goose Up
ALTER TABLE submission_fail_notes DROP CONSTRAINT submission_fail_notes_field_id_fkey;

-- +goose Down
ALTER TABLE submission_fail_notes ADD CONSTRAINT submission_fail_notes_field_id_fkey FOREIGN KEY (field_id) REFERENCES checklist_fields(id);
