#!/bin/bash
# .ui-jury/scripts/render-routes.sh — project-agnostic ui-jury template renderer
#
# Reads variables from .ui-jury/.env (gitignored), substitutes every ${VAR}
# placeholder in .ui-jury/routes.template.yaml whose VAR is declared in .env,
# and writes the rendered routes.yaml to the repo root.
#
# Why repo root: ui-jury's preflight reads ./routes.yaml from the project
# root (SKILL.md step 0d). The template + secrets + scripts live in .ui-jury/
# but the rendered artifact must land at repo root.
#
# Project-agnostic behaviour:
#   - Variables to substitute are auto-discovered from .ui-jury/.env (any
#     line matching VAR_NAME=value). No hard-coded variable names.
#   - To use in another project: drop this script + .ui-jury/.env(.example) +
#     .ui-jury/routes.template.yaml into that project's tree. Declare your
#     own variables in .env and reference them with ${VAR} in the template.
#
# Required: bash, python3 (both ship by default on macOS + most Linux).

set -euo pipefail

# Anchor to repo root — script lives at .ui-jury/scripts/<this-file>
cd "$(dirname "$0")/../.."

ENV_FILE=".ui-jury/.env"
TEMPLATE=".ui-jury/routes.template.yaml"
OUTPUT="routes.yaml"

if [ ! -f "$ENV_FILE" ]; then
  cat >&2 <<ERR
ERROR: $ENV_FILE not found.

First-time setup:
  cp .ui-jury/.env.example $ENV_FILE
  # then edit $ENV_FILE and fill in the values

ERR
  exit 1
fi

if [ ! -f "$TEMPLATE" ]; then
  echo "ERROR: $TEMPLATE not found" >&2
  exit 1
fi

# Export every assignment in .env so python3 can read them via os.environ.
# shellcheck disable=SC1090
set -a
. "$ENV_FILE"
set +a

# Hand off to python3 for the actual substitution. python3 is safer than
# sed/envsubst because:
#   - string.replace handles any byte in the value (sed needs delimiter
#     escaping; envsubst expands ALL env vars unless given an allowlist)
#   - we can validate "declared in .env" vs "present in environment" as
#     two separate things, surfacing the right error message
python3 <<'PY'
import os, re, sys

env_file = ".ui-jury/.env"
template_path = ".ui-jury/routes.template.yaml"
output_path = "routes.yaml"

# Discover declared variable names from .env (ignore comments + blanks).
declared = []
with open(env_file) as f:
    for line in f:
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        m = re.match(r"^([A-Za-z_][A-Za-z0-9_]*)=", line)
        if m:
            declared.append(m.group(1))

if not declared:
    sys.stderr.write(
        f"ERROR: {env_file} has no variable assignments.\n"
        f"\n"
        f"Add VAR=value lines — see .ui-jury/.env.example for the variables\n"
        f"this project's routes.template.yaml expects.\n"
    )
    sys.exit(1)

# Every declared var must resolve to a non-empty value once .env is sourced.
empty = [v for v in declared if not os.environ.get(v)]
if empty:
    sys.stderr.write(
        f"ERROR: variables declared in {env_file} are empty: {', '.join(empty)}\n"
    )
    sys.exit(1)

with open(template_path) as f:
    rendered = f.read()

substituted = []
for var in declared:
    placeholder = "${" + var + "}"
    if placeholder in rendered:
        rendered = rendered.replace(placeholder, os.environ[var])
        substituted.append(var)

with open(output_path, "w") as f:
    f.write(rendered)

if substituted:
    sys.stderr.write(
        f"Rendered {output_path} from {template_path} "
        f"({len(substituted)} substitution(s): {', '.join(substituted)})\n"
    )
else:
    sys.stderr.write(
        f"Rendered {output_path} from {template_path} (no ${{...}} placeholders "
        f"in template matched any .env variable — output is a verbatim copy)\n"
    )
PY
