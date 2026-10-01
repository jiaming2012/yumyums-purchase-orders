#!/usr/bin/env bash
# wrapper — the verdict is node's exit status (see 01-slices-reconcile.mjs)
set -euo pipefail
cd "$(dirname -- "${BASH_SOURCE[0]}")"
command -v node >/dev/null || { echo "⚠ COULD-NOT-RUN — node missing" >&2; exit 2; }
node ./01-slices-reconcile.mjs || { echo "🛑 VERDICT: RED — a slice did not reconcile" >&2; exit 1; }
