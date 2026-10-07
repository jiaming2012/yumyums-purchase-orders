#!/usr/bin/env bash
# K1 mutation legs on the FINAL tree worktree (port 8393). One guard mutated at a time in
# inventory.html, the named [IS-xx] test run with --retries=0, file restored. :5434 only.
set -u
W=/home/jcole/projects/hq-worktrees/triage-adv-20261007
L=/home/jcole/projects/hq/.night-crew/runs/2026-10-07-autonomous/logs/triage-adversarial
PW=$L/pw-leg.sh
cd $W
mut() { # name, python-replace-old, python-replace-new, grep pattern
  local name=$1 old=$2 new=$3 pat=$4
  python3 - "$old" "$new" <<'EOF'
import sys
old,new=sys.argv[1],sys.argv[2]
p='inventory.html'; s=open(p).read()
assert s.count(old)==1, ('mutation target not unique/found', s.count(old))
open(p,'w').write(s.replace(old,new,1))
EOF
  echo "## $name"; git diff --stat -- inventory.html
  git diff -- inventory.html > $L/k1-$name.diff
  bash $PW $W 8393 hq_test_triage_adv_20261007_e2e k1-$name tests/inventory.spec.js -g "$pat" --retries=0
  git checkout -- inventory.html
}

mut m1-no-seq-guard "if(seq<ITEMS_APPLIED){" "if(false&&seq<ITEMS_APPLIED){" "IS-02"
mut m2-dead-photo-rerenders "  if(img.classList.contains('item-photo-thumb')){
    // Editor thumb" "  renderItemsList();if(false){
    // Editor thumb" "IS-04"
mut m3-no-empty-name-alert "if(!name){alert('Enter an item name first.');return;}" "if(!name)return;" "IS-03"
mut m4-no-draft-carryover "function captureItemEditDraft(body){
  if(!ITEM_EDIT_ID)return null;" "function captureItemEditDraft(body){
  return null;if(!ITEM_EDIT_ID)return null;" "IS-04"
# (d) review finding: selected group NOT kept when groups land — does [IS-01] stay green?
mut m5-group-reset-on-groups-land "  sel.value=picked;
  if(sel.value!==picked)sel.value='';" "  sel.value='';" "IS-01"
git status --porcelain
