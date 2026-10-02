-- +goose Up
BEGIN;

-- ===========================================================================
-- 0085 — subscribers + subscriber_events (card H5, `subscribers-tab`).
--
-- The mailing list, and the timeline the detail sheet renders. Four sources
-- feed it (§4's CHECK): the website's Fluent Forms signup (`web_form`), a
-- Toast guest CSV (`toast_import`), an SMS keyword reply (`sms_keyword`), and a
-- campaign QR landing (`qr`).
--
-- `source_short` is the FIRST-TOUCH campaign code (decision 189): the first
-- short a subscriber scanned, and the FK target migration 0083 promised
-- (`qr_codes(short)`). It is NULLABLE and is often absent — the live Fluent
-- Forms submission the 2026-10-01 spike inspected carries no `source` field at
-- all, so when the website's signup URL did not carry `q=<short>` there is no
-- first-touch signal and the column stays NULL. A NULL here is the normal
-- case, not a defect.
--
-- `unique (source, external_ref)` is the IDEMPOTENCY key every adapter upserts
-- on: a Fluent Forms submission id or a Toast guest id. Postgres treats NULLs
-- as distinct in a unique constraint, which is deliberate — a row with no
-- external reference (an SMS keyword reply, a QR signup with no form id) is
-- deduped on `phone_e164` instead, which has its own UNIQUE and is the key
-- spike `e164-normalize-mask` proved closed (eight spellings of one number →
-- one E.164 value).
--
-- 🛑 `phone_e164` and `email` NEVER leave the server. §5's list and detail
-- shapes carry `phone_last4` and `email_masked`, computed in
-- internal/marketing/subscribers.go; the full values are not in any response
-- body. The columns are here because the adapters dedupe on them and Activity
-- E will send to them — not because the browser may read them.
--
-- DDL below is docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md
-- §4's H5 block, verbatim, with goose framing, the repo's BEGIN/COMMIT
-- convention, and the read-path indexes added at the bottom.
-- ===========================================================================

create table subscribers (
  id            uuid primary key default gen_random_uuid(),
  display_name  text null,
  phone_e164    text null unique,
  email         text null,
  source        text not null check (source in ('web_form','toast_import','sms_keyword','qr')),
  source_short  text null references qr_codes(short),  -- first-touch campaign code, when known
  sms_consent   boolean not null default false,
  email_consent boolean not null default false,
  consent_evidence text null,                        -- "form checkbox 2026-09-28" / "YES reply 2026-09-28"
  opted_out_at  timestamptz null,
  joined_at     timestamptz not null,
  external_ref  text null,                           -- Fluent Forms submission id / Toast guest id (idempotency)
  unique (source, external_ref)
);
create table subscriber_events (                     -- the detail sheet's timeline
  id bigserial primary key, subscriber_id uuid not null references subscribers(id),
  kind text not null check (kind in ('signed_up','code_sent','scanned','redeemed','resend_requested','opted_out')),
  at timestamptz not null default now(), ref jsonb null
);

-- Not in §4, and load-bearing for the two read paths this card ships.
--   * the list orders by joined_at desc and filters on source / consent;
--   * the detail sheet reads one subscriber's whole timeline, newest first;
--   * the campaign funnel's `signups` counts subscribers by source_short.
create index subscribers_joined_at_idx on subscribers (joined_at desc);
create index subscribers_source_short_idx on subscribers (source_short) where source_short is not null;
create index subscriber_events_subscriber_idx on subscriber_events (subscriber_id, at desc);

COMMIT;

-- +goose Down
BEGIN;
-- Reverse dependency order: subscriber_events -> subscribers. The indexes go
-- with their tables.
DROP TABLE IF EXISTS subscriber_events;
DROP TABLE IF EXISTS subscribers;
COMMIT;
