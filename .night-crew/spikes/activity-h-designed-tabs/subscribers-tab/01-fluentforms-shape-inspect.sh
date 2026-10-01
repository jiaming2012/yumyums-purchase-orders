#!/usr/bin/env bash
# 01-fluentforms-shape-inspect.sh — spike: the Fluent Forms submissions the
# Subscribers tab imports carry the field keys the reader will map
# (names, email, phone, source) — the premise behind H5's web_form adapter.
# READ-ONLY: runs the repo-adjacent website/scripts/count_customers.py --inspect
# (a SELECT of the newest submission's response JSON) with that script's own
# .env. The DB is the live WordPress MySQL on Hostinger; nothing is written.
# pymysql/python-dotenv are installed into a THROWAWAY venv under TMPDIR.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the four keys are present in the newest submission.
#   exit 1  connected, inspected, keys differ → the adapter maps different names.
#   exit 2  could not connect (remote MySQL not allowing this IP, no .env).
set -euo pipefail
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
WS="${WEBSITE_SCRIPTS:-$HOME/projects/yumyums/website/scripts}"
[ -f "$WS/count_customers.py" ] || cannot_run "count_customers.py not found at $WS"
[ -f "$WS/.env" ] || cannot_run "$WS/.env missing (DB creds live there; never copied here)"
V="${TMPDIR:-/tmp}/spike-h5-venv-$$"; trap 'rm -rf "$V"' EXIT
python3 -m venv "$V" >/dev/null 2>&1 || cannot_run "venv creation failed"
"$V/bin/pip" -q install pymysql python-dotenv >/dev/null 2>&1 || cannot_run "pip install pymysql failed (network?)"
echo "# target: the Fluent Forms MySQL named in $WS/.env — READ-ONLY --inspect of the newest submission"
OUT="$(cd "$WS" && timeout 40 "$V/bin/python" count_customers.py --inspect 2>&1)" || cannot_run "inspect did not run: $(printf %s "$OUT" | tail -3)"
printf '%s\n' "$OUT" | grep -oE "'[a-z_0-9]+':" | sort -u | tr '\n' ' '; echo   # keys only — values are a real person's contact details and stay out of the record
# CORRECTION (run 1 → red): the live form does NOT use `phone`/`source` as
# the script's defaults assume. The newest submission carries: `names`
# (an OBJECT, {first_name}), `email`, `input_text` (the phone number), and
# `checkbox` (an ARRAY of consent channels, e.g. ['Phone','Email']); the
# hidden `source` field is absent on this submission (nullable in the model).
for k in names email input_text checkbox; do printf '%s' "$OUT" | grep -q "'$k'" || fail "field key '$k' not present in the newest submission"; done
printf '%s' "$OUT" | grep -q "'source'" && echo "# source: present" || echo "# source: ABSENT on this submission — the adapter treats it as nullable (first-touch comes from the qr landing's q= param when present)"
echo "✅ GREEN — the adapter maps names.first_name → display_name, email → email, input_text → phone, checkbox[] → sms/email consent, source → source_short (nullable)"
