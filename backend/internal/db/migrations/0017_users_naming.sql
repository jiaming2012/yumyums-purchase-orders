-- +goose Up
BEGIN;
ALTER TABLE users ADD COLUMN first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN last_name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN nickname TEXT;

-- Best-effort migration of existing display_name data.
-- Pattern: "Jamal M." -> first_name='Jamal', last_name='M.'
-- Pattern: "Jamal" -> first_name='Jamal', last_name=''
UPDATE users SET
  first_name = CASE
    WHEN position(' ' IN display_name) > 0
      THEN left(display_name, position(' ' IN display_name) - 1)
    ELSE display_name
  END,
  last_name = CASE
    WHEN position(' ' IN display_name) > 0
      THEN substring(display_name FROM position(' ' IN display_name) + 1)
    ELSE ''
  END
WHERE display_name IS NOT NULL AND display_name != '';

ALTER TABLE users DROP COLUMN display_name;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
UPDATE users SET display_name = COALESCE(NULLIF(nickname, ''), first_name || ' ' || LEFT(last_name, 1) || '.');
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
ALTER TABLE users DROP COLUMN nickname;
COMMIT;
