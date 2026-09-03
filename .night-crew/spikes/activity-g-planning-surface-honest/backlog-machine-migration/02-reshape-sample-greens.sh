#!/usr/bin/env bash
# 02-reshape-sample-greens.sh — spike: the migration's WORKING LOOP is viable —
# reshape a copy, re-check the copy, prove content preservation — with the real
# document never touched.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all four legs held:
#            (1) isolation — `backlog check --file <copy>` demonstrably reads
#                the COPY (an entry appended to the copy moves ITS counts; the
#                real document's sha never changes)
#            (2) reshape — the sample legacy entry (B-90: an extra
#                `**destination: ...**` segment where the checker wants
#                origin · status · lead) is mechanically reshaped, and its
#                issues DISAPPEAR from the copy's check output while the total
#                issue count strictly drops
#            (3) preservation — every alphanumeric token of the original entry
#                is still present in the reshaped entry (multiset containment:
#                reshaping may move words, never delete them)
#            (4) the real document's sha256 is unchanged end to end
#   exit 1   a leg failed — the card's loop premise is not proven.
#   exit 2   could not run.
#
# The sample is ONE entry by design: this spike proves the loop, not the
# migration. The card does the other ~200 entries with this exact loop.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
DOC="$REPO_ROOT/.night-crew/knowledge/BACKLOG.md"
SAMPLE="B-90"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

command -v night-crew >/dev/null 2>&1 || cannot_run "night-crew CLI not on PATH"
command -v python3   >/dev/null 2>&1 || cannot_run "python3 not on PATH (the reshape surgery uses it)"
[ -f "$DOC" ] || cannot_run "document missing: $DOC"

echo "# target coordinates (read-only on the real document; all writes go to a scratch copy):"
echo "#   document : $DOC"
SHA_BEFORE="$(shasum -a 256 "$DOC" | awk '{print $1}')"
echo "#   sha256   : $SHA_BEFORE"

WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
COPY="$WORK/BACKLOG.md"
cp "$DOC" "$COPY"
echo "#   scratch  : $COPY"

check_file() { # $1 file → prints "RC|ISSUES|ENTRIES"
  local out rc issues entries summary
  set +e; out="$(night-crew backlog check --file "$1" 2>&1)"; rc=$?; set -e
  if [ "$rc" -eq 0 ]; then
    entries="$(echo "$out" | grep -oE 'valid — [0-9]+' | grep -oE '[0-9]+' | head -1)"
    echo "$rc|0|${entries:-0}"
  else
    summary="$(echo "$out" | grep -oE 'backlog invalid: [0-9]+ issue\(s\) across [0-9]+ entries' | tail -1)"
    issues="$(echo "$summary" | grep -oE '[0-9]+' | sed -n 1p)"
    entries="$(echo "$summary" | grep -oE '[0-9]+' | sed -n 2p)"
    echo "$rc|${issues:-?}|${entries:-?}"
  fi
}

echo
echo "── leg 1: the instrument reads the COPY, not the real document ──"
BASE="$(check_file "$COPY")"
echo "  copy baseline:            rc|issues|entries = $BASE"
echo "$BASE" | grep -qE '^[0-9]+\|[0-9]+\|[0-9]+$' || fail "the copy's check output did not parse: $BASE"
printf -- '- **B-9999 · spike isolation probe** — a deliberately appended entry; if the instrument reads the copy, the entry count moves. · _spike 02, this run_ · new · lead: none, this line is deleted with the scratch dir\n' >> "$COPY"
MOVED="$(check_file "$COPY")"
echo "  after appending B-9999:   rc|issues|entries = $MOVED"
E0="$(echo "$BASE"  | cut -d'|' -f3)"; E1="$(echo "$MOVED" | cut -d'|' -f3)"
[ "$E1" -gt "$E0" ] || fail "appending an entry to the copy did not move its entry count ($E0 → $E1) — the instrument is not reading the copy"
SHA_MID="$(shasum -a 256 "$DOC" | awk '{print $1}')"
[ "$SHA_MID" = "$SHA_BEFORE" ] || fail "the REAL document changed while only the copy was written — stop and look"
cp "$DOC" "$COPY"   # reset the copy for the reshape leg
echo "  ✓ instrument reads the copy; real document untouched"

echo
echo "── leg 2: reshape the sample entry ($SAMPLE) on the copy ──"
BEFORE_CHECK="$(night-crew backlog check --file "$COPY" 2>&1 || true)"
# sed -n '1,3p' not `head -3`: head would SIGPIPE grep on multi-KB lines (pipefail → 141)
echo "$BEFORE_CHECK" | grep -F "[$SAMPLE]" | sed -n '1,3p' | cut -c1-160 | sed 's/^/  | /'
echo "$BEFORE_CHECK" | grep -qF "[$SAMPLE]" || fail "the sample $SAMPLE raises no issue in the current document — pick a new sample before trusting this spike"
I_BEFORE="$(echo "$BEFORE_CHECK" | grep -oE 'backlog invalid: [0-9]+' | grep -oE '[0-9]+')"

python3 - "$COPY" "$SAMPLE" <<'PY' || fail "the reshape surgery failed"
import re, sys
path, handle = sys.argv[1], sys.argv[2]
text = open(path, encoding="utf-8").read()
# The entry is one '- **B-NN · ...' bullet line. Find it.
lines = text.split("\n")
idx = next((i for i, l in enumerate(lines) if l.startswith(f"- **{handle} ·")), None)
assert idx is not None, f"{handle} bullet not found"
line = lines[idx]
# The defect: an extra ' · **destination: ...**' segment sitting between the
# status and the lead, where the rubric wants origin · status · lead. Fold the
# destination text INTO the lead (parenthesized), deleting no words.
m = re.search(r" · \*\*(destination: [^*]+)\*\* · lead: ", line)
assert m is not None, f"{handle} does not carry the expected ' · **destination: ...** · lead: ' shape"
dest = m.group(1)
line = line[:m.start()] + f" · lead: ({dest}) " + line[m.end():]
lines[idx] = line
open(path, "w", encoding="utf-8").write("\n".join(lines))
print(f"  reshaped {handle}: folded the destination segment into the lead")
PY

AFTER_CHECK="$(night-crew backlog check --file "$COPY" 2>&1 || true)"
I_AFTER="$(echo "$AFTER_CHECK" | grep -oE 'backlog invalid: [0-9]+' | grep -oE '[0-9]+' || echo 0)"
echo "  issues: $I_BEFORE → ${I_AFTER:-0}"
if echo "$AFTER_CHECK" | grep -qF "[$SAMPLE]"; then
  echo "$AFTER_CHECK" | grep -F "[$SAMPLE]" | sed 's/^/  | /'
  fail "$SAMPLE still raises issues after the reshape"
fi
[ "${I_AFTER:-0}" -lt "$I_BEFORE" ] || fail "the total issue count did not drop ($I_BEFORE → $I_AFTER)"
echo "  ✓ $SAMPLE greens under the real checker"

echo
echo "── leg 3: content preservation (token multiset containment) ──"
python3 - "$DOC" "$COPY" "$SAMPLE" <<'PY' || fail "content was lost in the reshape"
import re, sys
from collections import Counter
orig_path, new_path, handle = sys.argv[1], sys.argv[2], sys.argv[3]
def entry_line(path):
    for l in open(path, encoding="utf-8"):
        if l.startswith(f"- **{handle} ·"):
            return l
    raise AssertionError(f"{handle} not found in {path}")
toks = lambda s: Counter(re.findall(r"[A-Za-z0-9]+", s))
before, after = toks(entry_line(orig_path)), toks(entry_line(new_path))
missing = before - after
assert not missing, f"tokens lost in reshape: {dict(missing)}"
print(f"  ✓ all {sum(before.values())} tokens of the original entry survive the reshape")
PY

echo
echo "── leg 4: the real document is untouched end to end ──"
SHA_AFTER="$(shasum -a 256 "$DOC" | awk '{print $1}')"
echo "  sha256 after: $SHA_AFTER"
[ "$SHA_AFTER" = "$SHA_BEFORE" ] || fail "the real document's bytes moved — stop and look"

printf '\n✅ VERDICT: GREEN — the reshape loop works on a scratch copy: sample greens, no token lost, real document untouched\n'
exit 0
