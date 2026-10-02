# `fluentforms_submissions.json` — the committed fixture of the REAL shape

🛑 **The night never connects to the live Fluent Forms database.** This file is
what `sources.FluentForms`' tests read, and it is the only Fluent Forms input
any automated gate has ever seen. The first live import is an **ATTENDED**
operator act: set `FF_DB_HOST/USER/PASSWORD/NAME/PREFIX` and call
`POST /api/v1/marketing/subscribers/import/web-form` by hand.

## Where the shape comes from

Spike `fluentforms-shape-inspect`
(`.night-crew/spikes/activity-h-designed-tabs/subscribers-tab/01-fluentforms-shape-inspect.sh`)
read the newest live submission read-only on 2026-10-01. Run 1 was RED: the
premise — the keys `names, email, phone, source` the website's own
`count_customers.py` defaults assume — was false. The real key set, signed by
the operator in the batch sitting, is:

| key | type | maps to |
|---|---|---|
| `names` | object, `{first_name, middle_name, last_name}` | `display_name` ← `names.first_name` |
| `email` | string | `email` |
| `input_text` | string | `phone_e164` ← `NormalizeE164(input_text)` |
| `checkbox` | array of strings, e.g. `["Phone","Email"]` | `sms_consent` ← contains "Phone"; `email_consent` ← contains "Email" |
| `source` | string, **often ABSENT** | `source_short` (NULLABLE) |

No contact value in this file belongs to a real person: the names are invented
and every phone number is in the 555 exchange reserved for fiction. The spike
itself printed **key names only**, never values, for exactly this reason.

## What each row is here to exercise

| `id` | the case |
|---|---|
| 1184 | the happy row — both consent channels, a clean parenthesized phone |
| 1183 | email-only consent (`checkbox:["Email"]`) → `consent:"email_only"` |
| 1182 | `source` **PRESENT** → `source_short` set → campaign attribution |
| 1181 | a 7-digit phone → `NormalizeE164` yields nothing → `phone_e164` NULL |
| 1180 | the SAME human as 1184 in a different spelling (`17735554821`) → the phone dedupe collapses it onto 1184's row instead of minting a second subscriber |
