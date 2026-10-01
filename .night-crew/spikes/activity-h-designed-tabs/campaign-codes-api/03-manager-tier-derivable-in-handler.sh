#!/usr/bin/env bash
# 03-manager-tier-derivable-in-handler.sh — spike: the §16 "enforce the
# manager tier inside the handler" rule has something to read. Premise: the
# authenticated user reaching a /marketing handler carries its roles in the
# request context (auth.UserFromContext → User.Roles), so a handler can answer
# 403 managers_only without a new middleware or a new DB read.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  both symbols exist and the auth package compiles.   exit 1  not.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
fail() { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
export PATH="/usr/local/go/bin:$PATH"
grep -q '^func UserFromContext(ctx context.Context) \*User' "$REPO_ROOT/backend/internal/auth/middleware.go" || fail "auth.UserFromContext not found"
grep -qE '^\s*Roles\s+\[\]string' "$REPO_ROOT/backend/internal/auth/service.go" || fail "auth.User has no Roles field"
echo "# auth.UserFromContext(ctx) *User exists; User.Roles []string exists"
( cd "$REPO_ROOT/backend" && go build ./internal/auth/ ) || fail "internal/auth does not build"
echo "✅ GREEN — the handler can read the caller's roles from context; no new middleware needed"
