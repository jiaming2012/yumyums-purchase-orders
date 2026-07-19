-- +goose Up
BEGIN;

CREATE TABLE daily_menu_sales (
  menu_item_id  UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
  business_date DATE NOT NULL,
  units_sold    INTEGER NOT NULL,
  gross_amount  NUMERIC(10,2) NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (menu_item_id, business_date)
);

CREATE INDEX daily_menu_sales_business_date_idx ON daily_menu_sales(business_date DESC);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS daily_menu_sales;
COMMIT;
