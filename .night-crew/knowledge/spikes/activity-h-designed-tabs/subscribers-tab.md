# Spikes — subscribers-tab

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> Hand-run convention (see `campaign-codes-api.md` header). The first spike
> reads the LIVE WordPress MySQL behind the website's Fluent Forms signup,
> read-only (`--inspect` is one SELECT of the newest submission), with the
> credentials that already sit in `website/scripts/.env`; nothing is written
> and no contact values are copied into this record.

## The goal, and which legs need a spike

The card (H5): the Subscribers tab — `subscribers` + `subscriber_events`,
three source adapters (Fluent Forms web form, Toast guest CSV, QR signup
join), the read-only list and detail sheet, `Resend QR` as a recorded
request. Two premises are falsifiable now: the web-form adapter's field
mapping (what the live form actually stores), and the phone handling every
row depends on.

## Spike: fluentforms-shape-inspect

- proves: the newest live Fluent Forms submission carries the keys the
  web_form adapter will map — stated by the premise as `names`, `email`,
  `phone`, `source` (the defaults in `website/scripts/count_customers.py`).
- plan: throwaway venv under TMPDIR with pymysql + python-dotenv; run
  `count_customers.py --inspect` from `website/scripts` with its own `.env`;
  print the KEY set only; assert the mapped keys are present.
- script: .night-crew/spikes/activity-h-designed-tabs/subscribers-tab/01-fluentforms-shape-inspect.sh

## Spike: e164-normalize-mask

- proves: the phone handling is a closed, enumerated set — eight spellings of
  one number normalize to one E.164 value (so dedupe-on-normalized is the
  right unique key), short/empty/null yield null, and the list's masked
  render is the last four of the normalized value.
- plan: python assertions over the enumerated cases.
- script: .night-crew/spikes/activity-h-designed-tabs/subscribers-tab/02-e164-normalize-mask.sh

## Verdict (hand-run 2026-10-01)

- **fluentforms-shape-inspect: failed (run 1) → passed (run 2)** — run 1
  connected (the Hostinger MySQL accepted this machine) and exited 1: the
  newest submission's keys are `names` (an OBJECT, `{first_name}`), `email`,
  `input_text` (the phone number), `checkbox` (an ARRAY of consent channels,
  e.g. `['Phone','Email']`) plus form metadata — there is **no `phone` key
  and no `source` key** on it. The premise as stated was false. Run 2 after
  the correction below: exit 0; key set enumerated; `source` reported ABSENT.
- **e164-normalize-mask: passed** — exit 0, first run. 8 cases → one E.164
  value; masked render `•••• 4821`.

## Corrections

- **the web_form adapter maps the form's real keys, not the script's
  defaults** — `names.first_name` → `display_name`, `email` → `email`,
  `input_text` → `phone` (E.164-normalized), `checkbox[]` → `sms_consent`
  (contains "Phone") and `email_consent` (contains "Email"), `source` →
  `source_short` **nullable** (absent on this submission; first-touch then
  comes from the QR landing's `q=` param when the signup URL carried one).
  The spike's assertion was changed to the enumerated real keys and to report
  `source` presence rather than require it. The card's schema already had
  `source_short` nullable; `consent_evidence` records which checkbox was
  ticked. Queued for the operator's batch review.

## Comebacks

- gap: GAP-H5-1 — `website/scripts/count_customers.py`'s default field names
  (`FF_PHONE_FIELD=phone`, `FF_SOURCE_FIELD=source`) do not match the live
  form, so its SMS-opt-in and ad-source breakdowns are counting nothing
  today. Not this card's to fix (website repo); recorded so the H5 adapter is
  not built from that script's assumptions. Validates when the website script
  is re-pointed and its report shows non-zero SMS opt-ins.

## Review

- signed: operator, 2026-10-01 — covers 1 correction(s) (batch sitting, presented as user stories at the operator's request; "Sign all three")
