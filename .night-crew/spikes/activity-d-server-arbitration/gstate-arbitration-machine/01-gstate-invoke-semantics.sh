#!/usr/bin/env bash
# 01-gstate-invoke-semantics.sh — spike: the §18 gstate machine's assumptions
# hold against the REAL github.com/floodfx/gstate library (which is NOT in
# backend/go.mod — the appendix was written against an imagined API).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   the library resolves, the §18-shaped machine compiles against its
#            actual API, and all seven legs hold: happy path, outcome routing,
#            unknown-result→failed (never silent expired), Invoke ctx
#            auto-cancel on state exit, no-token never invokes, After-retry
#            composes with Invoke.
#   exit 1   FAILED PREMISE — module not found, the API cannot express §18, or
#            a semantic leg red. This is a real verdict: the card re-prices
#            (fallback named in §18: qmuntal/stateless, async inverted).
#   exit 2   could not run — network/proxy unreachable, toolchain missing.
#
# Touches nothing outside its own directory. backend/go.mod is NOT touched;
# the spike module is self-contained. No substrate, no DB, no network calls
# beyond the Go module proxy.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

# Go PATH repair (run-mechanics rule — non-interactive shells drop it).
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH
command -v go >/dev/null 2>&1 || cannot_run "go not on PATH"

cd "$SCRIPT_DIR/go"

echo "── resolve the library (the existence premise) ──"
# Distinguish 'proxy unreachable' (could-not-run) from 'module not found' (RED).
if ! OUT="$(GOFLAGS=-mod=mod go mod download github.com/floodfx/gstate 2>&1)"; then
  echo "$OUT"
  case "$OUT" in
    *"no such host"*|*"connection refused"*|*"timeout"*|*"TLS handshake"*|*"proxyconnect"*)
      cannot_run "module proxy unreachable — network, not premise" ;;
    *) fail "github.com/floodfx/gstate did not resolve — the §18 library premise fails" ;;
  esac
fi
GOFLAGS=-mod=mod go mod tidy >/dev/null 2>&1 || fail "go mod tidy failed — dependency graph broken"
VERSION="$(go list -m github.com/floodfx/gstate)" || fail "resolved module not listable"
echo "  resolved: $VERSION"

echo
echo "── compile + run the seven legs (go test -v) ──"
if ! go vet ./... ; then
  fail "the §18-shaped machine does not compile against the real API — drift is a finding, record it"
fi
go test -v -count=1 -timeout 120s ./... || fail "a semantic leg red — see the test log above"

printf '\n✅ VERDICT: GREEN — %s carries the §18 machine: Invoke(ctx,snap,mutate)+auto-cancel, ordered Always fallback, After retry all hold.\n' "$VERSION"
