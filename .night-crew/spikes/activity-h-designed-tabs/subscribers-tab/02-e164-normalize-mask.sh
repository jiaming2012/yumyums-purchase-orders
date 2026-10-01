#!/usr/bin/env bash
# 02-e164-normalize-mask.sh — spike: the phone handling every subscriber row
# depends on — normalize the formats the three sources emit to E.164, dedupe
# on the normalized value, and render the masked last-4 the list shows — is a
# closed set we can enumerate and assert, not a per-row guess.
# 🛑 exit 0 = every case held; exit 1 = a case failed.
set -euo pipefail
python3 - <<'PY' || { echo "🛑 VERDICT: RED" >&2; exit 1; }
import re
def e164(raw):
    d=re.sub(r'\D','',raw or '')
    if len(d)==11 and d.startswith('1'): d=d[1:]
    if len(d)!=10: return None
    return '+1'+d
cases={'(773) 555-4821':'+17735554821','773-555-4821':'+17735554821','7735554821':'+17735554821','+1 773 555 4821':'+17735554821','17735554821':'+17735554821','555-4821':None,'':None,None:None}
for raw,exp in cases.items():
    got=e164(raw); assert got==exp, (raw,got,exp)
norm={e164(r) for r in cases if e164(r)}
assert norm=={'+17735554821'}, norm            # five spellings → one subscriber
assert '+17735554821'[-4:]=='4821'
print('# cases (set):', len(cases), '→ one E.164 value; masked render: •••• 4821')
print('✅ GREEN — E.164 normalize + dedupe + last-4 mask hold on the enumerated set')
PY
