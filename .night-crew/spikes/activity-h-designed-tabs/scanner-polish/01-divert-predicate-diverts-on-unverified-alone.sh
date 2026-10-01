#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname -- "${BASH_SOURCE[0]}")"
command -v node >/dev/null || { echo "⚠ COULD-NOT-RUN — node missing" >&2; exit 2; }
node ./01-divert-predicate-diverts-on-unverified-alone.mjs || { echo "🛑 VERDICT: RED" >&2; exit 1; }
