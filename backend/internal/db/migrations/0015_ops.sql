-- +goose Up
BEGIN;
CREATE TABLE ops (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id    TEXT        NOT NULL,
  user_id      UUID        NOT NULL REFERENCES users(id),
  entity_id    UUID        NOT NULL,
  entity_type  TEXT        NOT NULL,
  op_type      TEXT        NOT NULL,
  payload      JSONB       NOT NULL DEFAULT '{}',
  lamport_ts   BIGINT      NOT NULL,
  server_ts    TIMESTAMPTZ NOT NULL DEFAULT now(),
  applied      BOOLEAN     NOT NULL DEFAULT true
);
CREATE INDEX ops_user_lamport_idx ON ops (user_id, lamport_ts);
CREATE INDEX ops_entity_idx ON ops (entity_id);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS ops;
