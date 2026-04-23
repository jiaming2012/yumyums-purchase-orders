-- +goose Up
BEGIN;

-- Auto-incrementing employee number starting at 130
CREATE SEQUENCE employee_number_seq START WITH 130;

ALTER TABLE users
  ADD COLUMN employee_number INT UNIQUE DEFAULT nextval('employee_number_seq'),
  ADD COLUMN employee_type TEXT CHECK (employee_type IN ('W2', '1099')),
  ADD COLUMN starting_salary NUMERIC(10,2),
  ADD COLUMN toast_pos_number TEXT,
  ADD COLUMN cash_app_id TEXT,
  ADD COLUMN phone_number TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE users
  DROP COLUMN IF EXISTS phone_number,
  DROP COLUMN IF EXISTS cash_app_id,
  DROP COLUMN IF EXISTS toast_pos_number,
  DROP COLUMN IF EXISTS starting_salary,
  DROP COLUMN IF EXISTS employee_type,
  DROP COLUMN IF EXISTS employee_number;

DROP SEQUENCE IF EXISTS employee_number_seq;

COMMIT;
