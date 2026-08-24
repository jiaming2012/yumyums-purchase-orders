#!/usr/bin/env bash
# vendor/build-vendor.sh — regenerate the committed browser bundle in this directory.
#
# ============================================================================
# THIS IS NOT A BUILD STEP. IT IS A HAND-RUN ACT ON UPGRADE.
# ============================================================================
#
# HQ's stated project constraint is "Static only: No build step, no framework"
# (CLAUDE.md). This script exists so that constraint keeps holding: the OUTPUT
# (rxdb.bundle.js) is COMMITTED to git, so the deploy path — `task sw` →
# `docker build` → restart — never runs a bundler and never needs npm.
#
# Nothing invokes this script automatically. It is deliberately NOT wired into
# `task sw`, the root Taskfile, or backend/Dockerfile. Wiring it in would BE the
# "real build step" option that was considered and rejected; that is an operator
# decision, not a script's.
#
# You run it exactly twice in a package's life: when you first vendor it, and
# when you upgrade it. Then you commit the diff of the bundle along with the
# lockfile that produced it.
#
# ---------------------------------------------------------------------------
# WHY npx AND NOT A devDependency
# ---------------------------------------------------------------------------
# esbuild is invoked as `npx esbuild@$ESBUILD_VERSION`, which resolves into
# npm's own cache and writes NOTHING into the repo's dependency graph. The root
# package.json / package-lock.json are the Playwright environment for every
# night-crew card in every worktree; adding a devDependency there for a bundler
# that runs twice a year would change the test environment for work that has
# nothing to do with sync. The pin is what makes this reproducible without it.
#
# The rxdb / @supabase/supabase-js SOURCES come from ./package.json +
# ./package-lock.json in THIS directory — again, not the root. ./node_modules
# here is gitignored and is an input, not an artifact.
#
# ---------------------------------------------------------------------------
# USAGE
# ---------------------------------------------------------------------------
#   cd <repo root>
#   bash vendor/build-vendor.sh
#
# Then:
#   node build-sw.js          # re-precache the new content hash
#   git add vendor/ sw.js && git commit
#
# Requires: node + npm on PATH, and network access on first run (npx fetches
# esbuild into the npm cache; subsequent runs are offline-capable).
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --------------------------------------------------------------------------
# PINS. Changing any of these three lines is the upgrade.
# --------------------------------------------------------------------------
ESBUILD_VERSION="0.28.1"
# rxdb and @supabase/supabase-js are pinned EXACTLY (no ^, no ~) in
# ./package.json and locked in ./package-lock.json. They are read back out
# below rather than duplicated here, so the two can never drift.

cd "$HERE"

# --------------------------------------------------------------------------
# 1. Install the SOURCES, from this directory's own lockfile.
# --------------------------------------------------------------------------
if [ -f package-lock.json ]; then
    npm ci --no-audit --no-fund
else
    npm install --no-audit --no-fund
fi

RXDB_VERSION="$(node -p "require('./node_modules/rxdb/package.json').version")"
SUPABASE_VERSION="$(node -p "require('./node_modules/@supabase/supabase-js/package.json').version")"

echo "==> rxdb              ${RXDB_VERSION}"
echo "==> @supabase/supabase-js ${SUPABASE_VERSION}"
echo "==> esbuild           ${ESBUILD_VERSION} (via npx, not a devDependency)"

# --------------------------------------------------------------------------
# 2. Bundle.
#
#    --format=esm      loaded with <script type="module" src="vendor/...">
#    --platform=browser + the two defines: RxDB and supabase-js both reach for
#                      `process.env.NODE_ENV` and for `global`; without these
#                      the bundle throws ReferenceError on first import in a
#                      browser. Measured, not guessed.
#    --minify          this ships to phones over a food-truck LTE connection.
#    NO sourcemap      the repo gitignores *.map, so a committed sourcemap
#                      would be silently dropped and the bundle would 404 for
#                      it. Omitted deliberately rather than accidentally.
# --------------------------------------------------------------------------
BANNER="/* GENERATED — DO NOT EDIT. Regenerate with: bash vendor/build-vendor.sh
   rxdb@${RXDB_VERSION} + @supabase/supabase-js@${SUPABASE_VERSION}, bundled by esbuild@${ESBUILD_VERSION}.
   Committed on purpose: HQ has no build step in its deploy path. See vendor/build-vendor.sh. */"

npx --yes "esbuild@${ESBUILD_VERSION}" \
    src/rxdb-hq-entry.mjs \
    --bundle \
    --format=esm \
    --platform=browser \
    --target=es2020 \
    --minify \
    --legal-comments=none \
    --define:process.env.NODE_ENV='"production"' \
    --define:global=globalThis \
    --define:__VENDOR_RXDB_VERSION__="\"${RXDB_VERSION}\"" \
    --define:__VENDOR_SUPABASE_VERSION__="\"${SUPABASE_VERSION}\"" \
    --define:__VENDOR_ESBUILD_VERSION__="\"${ESBUILD_VERSION}\"" \
    --banner:js="${BANNER}" \
    --outfile=rxdb.bundle.js

# --------------------------------------------------------------------------
# 3. Report the size, because it is a real constraint.
#
#    build-sw.js sets maximumFileSizeToCacheInBytes to 5 MiB. A bundle above
#    that is silently DROPPED from the Workbox precache — which would look like
#    it worked and then fail offline on the truck. Fail loudly here instead.
# --------------------------------------------------------------------------
BYTES="$(wc -c < rxdb.bundle.js)"
GZIP="$(gzip -9 -c rxdb.bundle.js | wc -c)"
LIMIT=$((5 * 1024 * 1024))
printf '==> rxdb.bundle.js  %s bytes (%.1f KiB raw, %.1f KiB gzip)\n' \
    "$BYTES" "$(echo "$BYTES/1024" | bc -l)" "$(echo "$GZIP/1024" | bc -l)"

if [ "$BYTES" -ge "$LIMIT" ]; then
    echo "!! BUNDLE EXCEEDS build-sw.js maximumFileSizeToCacheInBytes (${LIMIT}). Workbox would" >&2
    echo "!! DROP it from the precache silently and the PWA would break offline. Raise the" >&2
    echo "!! limit deliberately or split the bundle — do not ignore this." >&2
    exit 1
fi

echo "==> done. Now run: node build-sw.js   (then commit vendor/ and sw.js together)"
