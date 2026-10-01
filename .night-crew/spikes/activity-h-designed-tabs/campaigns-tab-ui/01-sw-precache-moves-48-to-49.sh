#!/usr/bin/env bash
# 01-sw-precache-moves-48-to-49.sh — spike: the card's mechanical invariant.
# (a) baseline: build-sw.js on current HEAD exits 0 with EXACTLY 48 precached
#     files; (b) positive: a COMMITTED stub marketing/campaigns.js referenced
#     from marketing.html moves the precache to EXACTLY 49, including the new
#     module; (c) negative: marketing.html referencing an un-precached module
#     makes build-sw.js exit NON-zero and NAME marketing.html (B-37 guard).
# Same shape as activity-c/marketing-tile-and-page spike 01 (passed 2026-09-04),
# re-run because the count has moved since (31 → 48).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a)+(b)+(c) held.   exit 1  a leg failed.   exit 2  could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"
[ -d "$REPO_ROOT/node_modules/workbox-build" ] || cannot_run "workbox-build not installed"
STAMP="$(date +%s)"; BRANCH="spike-h2-sw-$STAMP"; WT="${TMPDIR:-/tmp}/spike-h2-sw-$STAMP"
cleanup() { cd "$REPO_ROOT"; git worktree remove --force "$WT" >/dev/null 2>&1 || true; git branch -D "$BRANCH" >/dev/null 2>&1 || true; }
trap cleanup EXIT
echo "# isolation: worktree=$WT branch=$BRANCH (throwaway; nothing pushed)"
git -C "$REPO_ROOT" worktree add -b "$BRANCH" "$WT" HEAD >/dev/null || cannot_run "git worktree add failed"
ln -s "$REPO_ROOT/node_modules" "$WT/node_modules"
count() { grep -o 'revision' "$WT/sw.js" | wc -l | tr -d ' '; }
( cd "$WT" && node build-sw.js >/tmp/spike-h2-a.$$ 2>&1 ) || fail "(a) baseline build-sw.js exited non-zero: $(cat /tmp/spike-h2-a.$$)"
A="$(count)"; echo "# (a) baseline precache = $A"; [ "$A" = "48" ] || fail "(a) baseline is $A, documented invariant is 48"
printf '// stub module for the spike\nexport const campaigns = true;\n' > "$WT/marketing/campaigns.js"
sed -i '' 's#<script type="module" src="marketing/submit-flow.js"></script>#<script type="module" src="marketing/submit-flow.js"></script>\n<script type="module" src="marketing/campaigns.js"></script>#' "$WT/marketing.html"
grep -q 'marketing/campaigns.js' "$WT/marketing.html" || cannot_run "could not inject the script tag"
( cd "$WT" && git add marketing/campaigns.js marketing.html && git -c user.email=spike@local -c user.name=spike commit -qm "spike: stub campaigns.js" ) || cannot_run "commit failed"
( cd "$WT" && node build-sw.js >/tmp/spike-h2-b.$$ 2>&1 ) || fail "(b) build-sw.js exited non-zero after adding the module: $(cat /tmp/spike-h2-b.$$)"
B="$(count)"; echo "# (b) precache = $B"; [ "$B" = "49" ] || fail "(b) precache is $B, expected 49"
grep -q 'marketing/campaigns.js' "$WT/sw.js" || fail "(b) sw.js manifest does not name marketing/campaigns.js"
sed -i '' 's#marketing/campaigns.js"></script>#marketing/campaigns.js"></script>\n<script type="module" src="marketing/missing-for-spike.js"></script>#' "$WT/marketing.html"
( cd "$WT" && git add marketing.html && git -c user.email=spike@local -c user.name=spike commit -qm "spike: dangling ref" ) || cannot_run "commit failed"
if ( cd "$WT" && node build-sw.js >/tmp/spike-h2-c.$$ 2>&1 ); then fail "(c) build-sw.js exited 0 with a dangling script reference — the B-37 guard did not fire"; fi
grep -q 'marketing.html' /tmp/spike-h2-c.$$ || fail "(c) guard fired but did not name marketing.html: $(cat /tmp/spike-h2-c.$$)"
echo "# (c) guard named the referrer: $(grep -m1 'marketing.html' /tmp/spike-h2-c.$$)"
rm -f /tmp/spike-h2-*.$$
echo "✅ GREEN — 48 → 49 on a committed module; dangling reference fails the build and names marketing.html"
