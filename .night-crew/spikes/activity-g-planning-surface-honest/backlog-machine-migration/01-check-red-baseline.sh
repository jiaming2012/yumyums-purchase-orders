#!/usr/bin/env bash
# 01-check-red-baseline.sh — spike: the migration's measuring instruments work and
# the red they measure is real, enumerated, and pinned — READ-ONLY on the document.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all four legs held:
#            (1) `night-crew backlog check` exits NON-zero on the real document
#                and its "N issue(s) across M entries" line parses — the red the
#                card retires exists and is countable
#            (2) `night-crew backlog list` exits 0 and emits a countable set —
#                the card's done_when comparator (list count == doc entry count)
#                has a working left-hand side
#            (3) both counts are printed side by side (the enumerated baseline)
#            (4) the document's sha256 is byte-identical before and after —
#                the instruments are read-only
#   exit 1   a leg failed (including: check unexpectedly PASSES — then the
#            premise "there is a migration to do" is dead and the card is moot).
#   exit 2   could not run (CLI missing).
#
# A finding is a set, not a sample (B-216): the counts come from the CLI's own
# enumeration, and the command that enumerates is printed with its output.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
DOC="$REPO_ROOT/.night-crew/knowledge/BACKLOG.md"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

command -v night-crew >/dev/null 2>&1 || cannot_run "night-crew CLI not on PATH"
[ -f "$DOC" ] || cannot_run "document missing: $DOC"

echo "# target coordinates (read-only statement — this script writes NOTHING):"
echo "#   document : $DOC (the real one; sha-verified untouched at exit)"
echo "#   binary   : $(command -v night-crew) ($(night-crew version 2>/dev/null | sed -n 1p || echo '?'))"
SHA_BEFORE="$(shasum -a 256 "$DOC" | awk '{print $1}')"
echo "#   sha256   : $SHA_BEFORE"

echo
echo "── leg 1: night-crew backlog check -repo $REPO_ROOT ──"
set +e
CHECK_OUT="$(night-crew backlog check -repo "$REPO_ROOT" 2>&1)"
CHECK_RC=$?
set -e
echo "  exit=$CHECK_RC"
echo "$CHECK_OUT" | tail -3 | sed 's/^/  | /'
[ "$CHECK_RC" -ne 0 ] || fail "backlog check exits 0 — the document is already valid and the migration premise is dead; re-scope the card before slating it"

SUMMARY="$(echo "$CHECK_OUT" | grep -oE 'backlog invalid: [0-9]+ issue\(s\) across [0-9]+ entries' | tail -1)"
[ -n "$SUMMARY" ] || fail "check failed but its summary line did not parse — the instrument's output shape moved; the card's done_when needs re-anchoring"
ISSUES="$(echo "$SUMMARY" | grep -oE '[0-9]+' | sed -n 1p)"
ENTRIES="$(echo "$SUMMARY" | grep -oE '[0-9]+' | sed -n 2p)"
echo "  parsed: issues=$ISSUES entries=$ENTRIES"
[ "$ISSUES" -gt 0 ] && [ "$ENTRIES" -gt 0 ] || fail "parsed a zero from the summary line: '$SUMMARY'"

echo
echo "── leg 2: night-crew backlog list -repo $REPO_ROOT (countable set) ──"
set +e
LIST_OUT="$(night-crew backlog list -repo "$REPO_ROOT" 2>&1)"
LIST_RC=$?
set -e
echo "  exit=$LIST_RC"
[ "$LIST_RC" -eq 0 ] || fail "backlog list exited $LIST_RC — the done_when comparator has no working left-hand side: $(echo "$LIST_OUT" | head -2)"
LIST_N="$(echo "$LIST_OUT" | grep -c '^' || true)"
LIST_B="$(echo "$LIST_OUT" | grep -cE '\[B-[0-9]+\]|^B-[0-9]+' || true)"
echo "  list lines=$LIST_N  lines carrying a B-handle=$LIST_B"
# sed -n '1,3p' reads its whole input — `head -3` here SIGPIPEs the echo on the
# document's multi-KB single-line entries and pipefail turns that into exit 141.
printf '%s\n' "$LIST_OUT" | sed -n '1,3p' | cut -c1-160 | sed 's/^/  | /'

echo
echo "── leg 3: the enumerated baseline, side by side ──"
echo "  check:  $ISSUES issue(s) across $ENTRIES entries (exit $CHECK_RC)"
echo "  list:   $LIST_N lines / $LIST_B handle-bearing (exit $LIST_RC)"
echo "  (the card's done_when: check exit 0 AND list count == document entry count)"

echo
echo "── leg 4: the document is untouched ──"
SHA_AFTER="$(shasum -a 256 "$DOC" | awk '{print $1}')"
echo "  sha256 after: $SHA_AFTER"
[ "$SHA_BEFORE" = "$SHA_AFTER" ] || fail "the document's bytes moved during a read-only baseline — stop and look"

printf '\n✅ VERDICT: GREEN — the red is real (%s issues / %s entries), both instruments run, document untouched\n' "$ISSUES" "$ENTRIES"
exit 0
