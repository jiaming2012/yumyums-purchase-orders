#!/usr/bin/env bash
# 01-machine-xstate.sh — spike: the XState candidate for the commissioned fork.
# (a) the §19.1 parallel-region scanner modeled in XState v5 passes the SHARED
#     10-sequence conformance suite (js/conformance.mjs — transcribed from the
#     §19.4 acceptance criteria, identical for both candidates);
# (b) the no-build premise: a single-file XState artifact exists, is vendored
#     as ONE file, loads in REAL Chromium from a plain page, and drives a
#     parallel machine there. Dist artifacts + bytes enumerated.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a)+(b) held.
#   exit 1  a sequence failed, or no single-file browser artifact exists —
#           the XState door closes on evidence.
#   exit 2  could not run (npm/network, browser missing).
#
# Touches nothing outside this directory. No substrate, no DB.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"

echo "── spike-local dep (xstate; nothing at repo root) ──"
(cd "$SCRIPT_DIR/js" && npm install --no-audit --no-fund --silent) || cannot_run "npm install failed — network?"
XV="$(node -e "console.log(require('$SCRIPT_DIR/js/node_modules/xstate/package.json').version)")"
echo "  xstate $XV"

echo
echo "── leg (a): the shared conformance suite ──"
node "$SCRIPT_DIR/js/run-xstate.mjs" || fail "the XState model failed the shared suite"

echo
echo "── leg (b): single-file browser artifact — enumerate, vendor, load ──"
DIST="$SCRIPT_DIR/js/node_modules/xstate/dist"
[ -d "$DIST" ] || fail "xstate ships no dist/ directory"
echo "  dist artifacts:"
(cd "$DIST" && ls -la *.js 2>/dev/null | awk '{print "    "$5"\t"$9}') || true
# The UMD build is the convention-fit artifact: HQ loads third-party JS as
# classic <script> files (SortableJS; html5-qrcode in the sibling spike), and
# Chromium CORS-blocks ES-module imports from file:// anyway (first run red).
UMD="$DIST/xstate.umd.min.js"
[ -f "$UMD" ] || fail "no single-file UMD build (dist/xstate.umd.min.js) — the no-build premise fails for XState"
cp "$UMD" "$SCRIPT_DIR/web/xstate.js"
BYTES="$(wc -c < "$SCRIPT_DIR/web/xstate.js" | tr -d ' ')"
echo "  vendored xstate.js: $BYTES bytes (the dependency weight the card would inherit)"

cd "$REPO_ROOT"
npx --no-install playwright test -c "$SCRIPT_DIR/playwright.config.js" \
  || fail "the vendored file did not drive a parallel machine in Chromium"

printf '\n✅ VERDICT: GREEN — XState v%s passes the shared suite and loads as ONE vendored file (%s bytes) in a plain page.\n' "$XV" "$BYTES"
