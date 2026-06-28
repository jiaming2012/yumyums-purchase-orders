#!/bin/bash
# .ui-jury/scripts/install-into.sh — scaffold .ui-jury/ into another project
#
# Usage:
#   ./.ui-jury/scripts/install-into.sh [target_dir]
#
# If target_dir is omitted, installs into the current working directory.
# Idempotent: existing files are NOT overwritten (script reports "[skip] …").
#
# What gets installed in TARGET:
#   .ui-jury/scripts/render-routes.sh   (verbatim copy of the renderer)
#   .ui-jury/scripts/db-reset.sh        (no-op stub — customize for your DB)
#   .ui-jury/routes.template.yaml       (starter template, single at-rest route)
#   .ui-jury/.env.example               (starter — declares EXAMPLE_VAR)
#   .ui-jury/hooks.yaml                 (declares db_reset path)
#   .gitignore                          (appends /routes.yaml + /.ui-jury/.env)
#
# After install, in TARGET:
#   1. cp .ui-jury/.env.example .ui-jury/.env  → edit the values
#   2. edit .ui-jury/routes.template.yaml → declare your routes
#   3. edit .ui-jury/scripts/db-reset.sh → wire to your real reset command
#   4. ./.ui-jury/scripts/render-routes.sh
#   5. /ui-jury <dev-url> --backend-log-path <path>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

TARGET="${1:-$(pwd)}"
if [ ! -d "$TARGET" ]; then
  echo "ERROR: target directory does not exist: $TARGET" >&2
  exit 1
fi
TARGET="$(cd "$TARGET" && pwd)"

echo "[install-ui-jury] Source: $SCRIPT_DIR"
echo "[install-ui-jury] Target: $TARGET"

if ! git -C "$TARGET" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "[install-ui-jury] WARNING: target is not a git repo (.gitignore will still be written)"
fi

mkdir -p "$TARGET/.ui-jury/scripts"

write_if_absent() {
  # $1 = destination path; reads body from stdin
  local dest="$1"
  if [ -e "$dest" ]; then
    echo "[skip] $dest already exists"
    cat >/dev/null  # drain stdin so heredocs don't error
    return 0
  fi
  cat > "$dest"
  echo "[ok]   $dest"
}

# --- 1. Copy the renderer verbatim from the source project --------------
DEST="$TARGET/.ui-jury/scripts/render-routes.sh"
if [ -e "$DEST" ]; then
  echo "[skip] $DEST already exists"
else
  cp "$SCRIPT_DIR/render-routes.sh" "$DEST"
  chmod 755 "$DEST"
  echo "[ok]   $DEST"
fi

# --- 2. Generate a no-op db-reset.sh stub -------------------------------
DEST="$TARGET/.ui-jury/scripts/db-reset.sh"
write_if_absent "$DEST" <<'BODY'
#!/bin/bash
# .ui-jury/scripts/db-reset.sh — ui-jury db_reset hook (STUB)
#
# Customize this to bring your project's DB to a known starting state.
# Contract: no args, idempotent, exit 0 = success. Anything else aborts
# the /ui-jury run with SETUP_HOOK_FAILED and the stderr is surfaced to
# the user verbatim — do NOT echo secrets.
#
# Static-site project? Leave the `exit 0` below as-is.
# Have a DB? Replace `exit 0` with the real command, e.g.:
#   exec task db-reset-test
#   psql "$DB_URL" -c 'TRUNCATE my_table CASCADE;'

set -euo pipefail
cd "$(dirname "$0")/../.."
exit 0
BODY
[ -e "$DEST" ] && chmod 755 "$DEST"

# --- 3. Generate starter .env.example -----------------------------------
DEST="$TARGET/.ui-jury/.env.example"
write_if_absent "$DEST" <<'BODY'
# .ui-jury/.env.example — copy to .ui-jury/.env and fill in real values.
#
#   cp .ui-jury/.env.example .ui-jury/.env
#   # then edit .ui-jury/.env
#
# .ui-jury/.env is gitignored — DO NOT commit it.
#
# Every VAR=value line below is auto-discovered by render-routes.sh and
# becomes substitutable in routes.template.yaml as ${VAR}. Add your own
# (passwords, API keys, anything the template needs). Variable names are
# free-form — no naming convention is enforced.

EXAMPLE_VAR=
BODY

# --- 4. Generate starter routes.template.yaml ---------------------------
DEST="$TARGET/.ui-jury/routes.template.yaml"
write_if_absent "$DEST" <<'BODY'
# .ui-jury/routes.template.yaml — ui-jury route declarations (TEMPLATE)
#
# Rendered → ./routes.yaml at repo root (gitignored) by:
#   ./.ui-jury/scripts/render-routes.sh
#
# Reference variables from .ui-jury/.env as ${VAR} — they're substituted
# at render time. ui-jury v1's routes.yaml schema does NOT expand env
# vars at run time, so substitution MUST happen at render time.
#
# Schema: https://ui-jury.dev/schema/routes.schema.json (v1)

version: 1

# Replace with the viewport you care about. 393×852 = iPhone 14.
viewport:
  width: 393
  height: 852

# Uncomment + customize if your app requires authentication. The
# Driver runs setup.steps[] after navigating to setup.url.
#
# setup:
#   description: Sign in once before all routes.
#   url: /login
#   steps:
#     - action: fill
#       locator: { role: textbox, name: "Email" }
#       value: "you@example.com"
#     - action: fill
#       locator: { role: textbox, name: "Password" }
#       value: "${EXAMPLE_VAR}"
#     - action: click
#       locator: { role: button, name: "Sign In" }

routes:
  - path: /
    states:
      - name: at-rest
BODY

# --- 5. Generate starter hooks.yaml -------------------------------------
DEST="$TARGET/.ui-jury/hooks.yaml"
write_if_absent "$DEST" <<'BODY'
# .ui-jury/hooks.yaml — ui-jury hook configuration
#
# db_reset is required. The orchestrator invokes the script with cwd =
# repo root, no args, no env. Idempotent, exit 0 = success.
#
# db_reset_timeout defaults to 60s — set explicitly only if your reset
# takes longer.

db_reset: ./.ui-jury/scripts/db-reset.sh
BODY

# --- 6. Update .gitignore (idempotent — append missing entries only) ----
GITIGNORE="$TARGET/.gitignore"
touch "$GITIGNORE"

added=0
for entry in "/routes.yaml" "/.ui-jury/.env"; do
  if grep -qxF "$entry" "$GITIGNORE"; then
    echo "[skip] $GITIGNORE already has $entry"
  else
    if [ "$added" -eq 0 ]; then
      printf "\n# ui-jury: rendered routes.yaml + secret env file\n" >> "$GITIGNORE"
    fi
    printf "%s\n" "$entry" >> "$GITIGNORE"
    echo "[ok]   $GITIGNORE += $entry"
    added=$((added + 1))
  fi
done

cat <<NEXT

[install-ui-jury] Done.

Next steps in $TARGET:
  1. cp .ui-jury/.env.example .ui-jury/.env
  2. Edit .ui-jury/.env with your real values
  3. Edit .ui-jury/routes.template.yaml — declare your routes + states
  4. Edit .ui-jury/scripts/db-reset.sh — wire to your real DB reset command
  5. ./.ui-jury/scripts/render-routes.sh
  6. /ui-jury <dev-url> --backend-log-path <path>
NEXT
