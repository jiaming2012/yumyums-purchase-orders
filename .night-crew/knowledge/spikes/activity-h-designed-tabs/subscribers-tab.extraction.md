# Extraction — subscribers-tab

Outcome: learned

Approach used: a read-only `--inspect` of the live Fluent Forms submissions
table (the website's Hostinger MySQL, via the existing
`website/scripts/count_customers.py` in a throwaway venv) to learn the real
field keys, plus an enumerated E.164 normalize / dedupe / last-4 mask set.
Two spikes; the inspect was red on run 1 and exit 0 on run 2 after a signed
correction; the phone set was exit 0 first run.

Confirmed: this machine reaches the form database read-only; eight spellings
of one phone number normalize to one E.164 value (dedupe key), short/empty
yield null, the list's mask is the last four of the normalized value.

Learned: the live form does NOT store `phone` or `source` under those names.
The newest submission carries `names` (an object, `{first_name}`), `email`,
`input_text` (the phone), and `checkbox` (an array of consent channels,
e.g. `['Phone','Email']`); the hidden ad-source field was absent. So (a) the
web_form adapter maps `input_text` → phone, `checkbox[]` → sms/email
consent, `names.first_name` → display name, and treats `source` as nullable
— first-touch then comes only from the QR landing's `q=` param; (b) the
website's own counting script uses the wrong defaults and is reporting zero
SMS opt-ins today (GAP-H5-1 in the ledger, website repo's to fix).

Plan change: the H5 card's adapter spec and fixture use the real key set
(handoff §6 H5 row and roadmap card updated 2026-10-01, operator-signed);
`consent_evidence` records which checkbox was ticked. No schema change —
`source_short` was already nullable.
