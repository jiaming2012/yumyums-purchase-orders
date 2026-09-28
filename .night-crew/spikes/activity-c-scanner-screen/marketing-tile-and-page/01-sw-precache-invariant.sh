#!/usr/bin/env bash
# 01-sw-precache-invariant.sh — spike: the card's mechanical invariant.
# (a) baseline: build-sw.js on current HEAD exits 0 with EXACTLY 31 precached
#     files (the documented invariant holds before the card moves it);
# (b) positive: a committed stub marketing.html + index.html tile link moves
#     the precache to EXACTLY 32, including marketing.html;
# (c) negative: marketing.html referencing an un-precached script makes
#     build-sw.js exit NON-zero and NAME marketing.html — the B-37 guard
#     demonstrably guards the file this card adds.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a)+(b)+(c) held.   exit 1  a leg failed.   exit 2  could not run.
#
# Isolation: a THROWAWAY git worktree on a throwaway branch under the session
# scratchpad. The real working tree, dev, and every remote are untouched;
# nothing is pushed; worktree + branch are removed on exit (trap).

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"
[ -d "$REPO_ROOT/node_modules/workbox-build" ] || cannot_run "workbox-build not installed at repo root (npm ci first)"

STAMP="$(date +%s)"
BRANCH="spike-sw-invariant-$STAMP"
WT="${TMPDIR:-/tmp}/spike-sw-invariant-$STAMP"

cleanup() {
  cd "$REPO_ROOT"
  git worktree remove --force "$WT" >/dev/null 2>&1 || true
  git branch -D "$BRANCH" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "# isolation: worktree=$WT branch=$BRANCH (throwaway, removed on exit; nothing pushed)"
git -C "$REPO_ROOT" worktree add -b "$BRANCH" "$WT" HEAD >/dev/null || cannot_run "git worktree add failed"
ln -s "$REPO_ROOT/node_modules" "$WT/node_modules" || cannot_run "node_modules symlink failed"

count_precache() { grep -o 'url:' "$WT/sw.js" | wc -l | tr -d ' '; }
build() { (cd "$WT" && node build-sw.js); }

echo
echo "── leg (a): baseline on HEAD — exit 0, exactly 31 precached ──"
OUT="$(build 2>&1)" || { echo "$OUT"; fail "baseline build-sw.js exited non-zero on unmodified HEAD"; }
echo "$OUT" | grep -E 'SW built: [0-9]+ files precached' || true
N="$(count_precache)"
echo "  precache entries (url: pattern): $N"
[ "$N" = "31" ] || fail "baseline precache is $N, not the documented 31 — the invariant moved without this card (B-37 class)"
echo "$OUT" | grep -q 'SW built: 31 files precached' || fail "build-sw's own count line does not say 31"

echo
echo "── leg (b): committed stub marketing.html + tile link → exactly 32 ──"
cat > "$WT/marketing.html" <<'HTML'
<!doctype html><html><head><meta charset="utf-8"><title>Marketing (spike stub)</title></head>
<body><a href="index.html">← HQ</a><h1>Marketing</h1><script src="ptr.js"></script></body></html>
HTML
# the tile reference, as the card will make it (before </body> keeps it valid)
sed -i.bak 's|</body>|<a href="marketing.html" style="display:none">Marketing</a></body>|' "$WT/index.html" \
  && rm -f "$WT/index.html.bak"
git -C "$WT" add marketing.html index.html
git -C "$WT" -c user.name=spike -c user.email=spike@local commit -q -m "spike: stub marketing page (throwaway)" \
  || cannot_run "worktree commit failed"
OUT="$(build 2>&1)" || { echo "$OUT"; fail "build-sw.js exited non-zero with a valid committed marketing.html"; }
N="$(count_precache)"
echo "  precache entries: $N"
[ "$N" = "32" ] || fail "adding one page moved the precache to $N, not 32"
grep -q 'marketing\.html' "$WT/sw.js" || fail "marketing.html is NOT in the generated precache (B-37 silent drop)"
echo "  marketing.html present in the manifest"

echo
echo "── leg (c): un-precached reference → guard fires, names the referrer ──"
cat > "$WT/marketing.html" <<'HTML'
<!doctype html><html><head><meta charset="utf-8"><title>Marketing (spike stub)</title></head>
<body><a href="index.html">← HQ</a><script src="marketing-missing.js"></script></body></html>
HTML
git -C "$WT" add marketing.html
git -C "$WT" -c user.name=spike -c user.email=spike@local commit -q -m "spike: broken reference (throwaway)" \
  || cannot_run "worktree commit failed"
set +e
OUT="$(build 2>&1)"; RC=$?
set -e
echo "$OUT" | tail -8
[ "$RC" -ne 0 ] || fail "build-sw.js exited 0 while marketing.html references un-precached marketing-missing.js — the guard does not guard"
echo "$OUT" | grep -q 'marketing' || fail "the failure message does not name the referrer"
echo "  guard fired (exit $RC) and named the file"

printf '\n✅ VERDICT: GREEN — baseline 31 holds, the card moves it to exactly 32, and the reachability guard fires on the exact file the card adds.\n'
