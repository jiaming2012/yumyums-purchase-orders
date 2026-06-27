#!/bin/bash
# scripts/ui-jury/render-routes.sh — render routes.yaml from routes.template.yaml
#
# Reads HQ_TEST_PASSWORD from .ui-jury/.env (gitignored) and substitutes it
# into the ${HQ_TEST_PASSWORD} placeholder in routes.template.yaml, writing
# the result to routes.yaml (also gitignored).
#
# ui-jury v1 routes.yaml does not support env-var substitution at run time —
# this script does it at render time so the password never gets committed.

set -euo pipefail

cd "$(dirname "$0")/../.."

if [ ! -f .ui-jury/.env ]; then
  cat >&2 <<'ERR'
ERROR: .ui-jury/.env not found.

First-time setup:
  cp .ui-jury/.env.example .ui-jury/.env
  # then edit .ui-jury/.env and set HQ_TEST_PASSWORD

ERR
  exit 1
fi

# Source the .env file — supports lines like HQ_TEST_PASSWORD=value (no export
# needed; `set -a` marks all assignments for export).
# shellcheck disable=SC1091
set -a
. .ui-jury/.env
set +a

if [ -z "${HQ_TEST_PASSWORD:-}" ]; then
  echo "ERROR: HQ_TEST_PASSWORD is unset or empty in .ui-jury/.env" >&2
  exit 1
fi

if [ ! -f routes.template.yaml ]; then
  echo "ERROR: routes.template.yaml not found at repo root" >&2
  exit 1
fi

# Render via python3 — handles any special characters in the password safely
# (sed would need escaping for &, \, |, newlines; envsubst would expand every
# env var unless given an explicit allowlist; python3 string.replace is the
# narrowest substitution that does exactly one variable).
python3 -c '
import os, sys
with open("routes.template.yaml") as f:
    template = f.read()
sys.stdout.write(template.replace("${HQ_TEST_PASSWORD}", os.environ["HQ_TEST_PASSWORD"]))
' > routes.yaml

echo "Rendered routes.yaml from routes.template.yaml (HQ_TEST_PASSWORD substituted)"
