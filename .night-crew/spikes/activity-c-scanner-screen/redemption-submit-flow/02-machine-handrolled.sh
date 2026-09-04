#!/usr/bin/env bash
# 02-machine-handrolled.sh — spike: the no-dependency candidate for the
# commissioned fork. The SAME 10-sequence conformance suite (js/conformance.mjs
# — byte-identical to the one the XState candidate runs) passes against
# js/machine-handrolled.mjs, a dependency-free parallel-region machine in
# plain ES. The owned-code price is enumerated (lines + bytes).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  all sequences held.   exit 1  a sequence failed.   exit 2  no node.
#
# No dependencies, no npm, no browser leg needed — plain ES is vanilla by
# construction. Touches nothing outside this directory.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"

echo "── the shared conformance suite ──"
node "$SCRIPT_DIR/js/run-handrolled.mjs" || fail "the hand-rolled machine failed the shared suite"

echo
echo "── the owned-code price, enumerated ──"
M="$SCRIPT_DIR/js/machine-handrolled.mjs"
LINES="$(wc -l < "$M" | tr -d ' ')"
BYTES="$(wc -c < "$M" | tr -d ' ')"
echo "  machine-handrolled.mjs: $LINES lines, $BYTES bytes, 0 dependencies"

printf '\n✅ VERDICT: GREEN — the hand-rolled parallel-region machine passes the identical suite at %s lines / %s bytes / 0 deps.\n' "$LINES" "$BYTES"
