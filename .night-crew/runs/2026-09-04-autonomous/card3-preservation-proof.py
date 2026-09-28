#!/usr/bin/env python3
"""card3-preservation-proof.py — Card 3 `backlog-machine-migration` (run 20260904).

Whole-document content-preservation proof (Q-KR3's mechanical form): token-multiset
CONTAINMENT over the entire BACKLOG.md. Every alphanumeric token occurrence present
in the PRE-reshape document must be present in the landed one — words may move or
be added (handles, canonical status heads, leads), never vanish.

Proof integrity (merge-intent rule 6): the BEFORE side is read FROM GIT — the
red-baseline commit that precedes every reshape commit on this branch — never from
a scratch copy the reshaper controlled both sides of.

Usage:  python3 card3-preservation-proof.py [<before-ref>] [<after-path>]
  before-ref  default: the red-baseline commit (subject-pinned lookup, falls back
              to the literal SHA recorded in the merge-intent)
  after-path  default: .night-crew/knowledge/BACKLOG.md in the working tree

Exit 0: containment holds. Exit 1: tokens were lost (each named). Exit 2: cannot run.
"""
import re
import subprocess
import sys
from collections import Counter

DOC = ".night-crew/knowledge/BACKLOG.md"
TOK = re.compile(r"[A-Za-z0-9]+")


def sh(*args):
    return subprocess.run(args, capture_output=True, text=True)


def main():
    before_ref = sys.argv[1] if len(sys.argv) > 1 else None
    after_path = sys.argv[2] if len(sys.argv) > 2 else DOC

    if before_ref is None:
        # The red-baseline commit is the first commit of this card's branch,
        # subject-pinned so a skeptic needs no hardcoded SHA.
        r = sh("git", "log", "--format=%H %s", "--all")
        if r.returncode != 0:
            print("cannot run: git log failed:", r.stderr.strip())
            return 2
        for line in r.stdout.splitlines():
            sha, _, subj = line.partition(" ")
            if "card3 merge-intent + red baseline" in subj:
                before_ref = sha
                break
        if before_ref is None:
            print("cannot run: red-baseline commit not found by subject")
            return 2

    r = sh("git", "show", f"{before_ref}:{DOC}")
    if r.returncode != 0:
        print(f"cannot run: git show {before_ref}:{DOC} failed:", r.stderr.strip())
        return 2
    before = r.stdout
    after = open(after_path, encoding="utf-8").read()

    b, a = Counter(TOK.findall(before)), Counter(TOK.findall(after))
    lost = b - a

    print("# whole-document token-multiset containment proof")
    print(f"BEFORE ref : {before_ref}:{DOC}")
    print(f"  sha256(content): {subprocess.run(['shasum','-a','256'],input=before,capture_output=True,text=True).stdout.split()[0]}")
    print(f"  tokens: {sum(b.values())} occurrences / {len(b)} distinct")
    print(f"AFTER path : {after_path} (working tree at the landing commit)")
    print(f"  sha256(content): {subprocess.run(['shasum','-a','256'],input=after,capture_output=True,text=True).stdout.split()[0]}")
    print(f"  tokens: {sum(a.values())} occurrences / {len(a)} distinct")
    print(f"before − after (lost token occurrences): {sum(lost.values())}")
    if lost:
        for t, k in list(lost.items())[:50]:
            print(f"  LOST: {t!r} × {k}")
        print("VERDICT: RED — containment FAILS")
        return 1
    print("VERDICT: GREEN — every token occurrence present before is present after")
    return 0


if __name__ == "__main__":
    sys.exit(main())
