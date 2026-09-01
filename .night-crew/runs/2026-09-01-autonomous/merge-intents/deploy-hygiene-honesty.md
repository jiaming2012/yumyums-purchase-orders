# Merge intent — `deploy-hygiene-honesty` (Card 11, Track C)

Closes **B-135** and **B-17**. Pure build-tooling hygiene: unify the two `version.json`
generators at the byte level, and correct one empirically-false comment.

## The two facts, measured

**B-135 — the byte divergence.** `version.json` has two shipping generators:
- `scripts/write-version-json.js:42` (used by `build-sw.js`, local dev, `task sw`, `task test`)
  writes `JSON.stringify(payload) + '\n'` → `{"frontend":"X.Y.Z"}\n` = **21 bytes**, md5 `a87612b…`.
- `backend/Dockerfile:64` regenerates it INSIDE the image with
  `printf '{"frontend":"%s"}' "$fe_ver"` → `{"frontend":"X.Y.Z"}` = **20 bytes**, md5 `93bef58…`.

The one-byte difference is the trailing newline. In prod the Docker copy (20 bytes) is what
the SW serves, but `sw.js`'s precache **revision** for `version.json` was computed by
`build-sw.js` against the Node-written 21-byte file — so the manifest revision does not
describe the bytes served. Fix: make the Node generator authoritative and have the Dockerfile
emit **byte-identical** output (add the trailing `\n`). Same artifacts ship, same paths, same
release-flow shape — only the Dockerfile's `printf` gains a `\n`. Byte-identical unification,
squarely the run's to make (NOT the PARK condition — nothing relocates or drops).

Note: `version.json` is git-IGNORED (`.gitignore:13`) — there is **no committed root or
embedded copy**. Both are generated: the served one by `build-sw.js`/`write-version-json.js`,
the embedded one (`backend/cmd/server/public/version.json`) by the Dockerfile at image-build
time. So "root and embedded byte-identical" is enforced by making the two GENERATORS agree,
verified by a red-first byte-diff unit test — not by comparing committed files (there are none).

**B-17 — the false comment.** `build-sw.js:46` claims `--name-only` C-quotes any path with
"a space or a non-ASCII byte". Measured in a throwaway repo: `git ls-tree -r --name-only HEAD`
returns `has space.txt` **unquoted** but `"caf\303\251.txt"` **quoted**. `core.quotePath`
(default true) escapes non-ASCII/control bytes, NOT spaces. The `-z` flag stays — it is still
correct and necessary (unquoted NUL-separated output sidesteps quoting entirely). Only the
stated *reason* is half-wrong. Fix the WORDING, keep the flag.

## Shared files touched

- `build-sw.js` — comment-only edit at line 46 (B-17). **No manifest/glob/count change.**
  🛑 Card 7 `sync-doc-honesty` ALSO touches `sw.js`/precache. The orchestrator re-runs
  `task sw` on the merged tree after the LAST of Cards 7 and 11 lands, so my committed `sw.js`
  is a checkpoint, not the final artifact — reconcile there.
- `sw.js` + `version.json`(committed? NO — ignored) — I regenerate `sw.js` via `task sw` and
  commit it in my own change set (the sw.js rule). `version.json` is gitignored, not committed.
- `backend/Dockerfile` — one-char change: `printf '{"frontend":"%s"}'` → `printf '{"frontend":"%s"}\n'`
  (B-135). Confined to line 64.
- `scripts/write-version-json.js` — unchanged in behavior; it is already authoritative. (I add
  a one-line comment noting the Dockerfile now mirrors it byte-for-byte, so the parity is stated.)
- `tests/version-json-parity.spec.js` — NEW red-first byte-diff test (my footprint).
- `.night-crew/knowledge/roadmap.md` — flip this card `PLANNED → DONE` in the final commit,
  and touch the B-17 mirror IF the live roadmap still carries the verbatim false claim.

## What must survive any merge

- The Dockerfile's `version.json` output must be **byte-identical** to `write-version-json.js`'s
  (`{"frontend":"<ver>"}\n`, trailing newline). A future Dockerfile edit must not drop the `\n`.
- `write-version-json.js` remains the single authoritative payload definition; the Dockerfile
  is a subordinate mirror (it reads `version.go` because the image has no `package.json` build
  context wired for it, but must produce the SAME bytes).
- The corrected B-17 comment: `--name-only` C-quotes non-ASCII/control bytes (`core.quotePath`),
  NOT spaces — which is why `-z` is required.
- The precache count stays **31**. No asset added or removed.

## What is safe to drop

- Exact comment wording (as long as it is factually correct about `core.quotePath`).
- The new test's filename / internal helper names — any test that reproduces the byte-diff
  (RED before, GREEN after) satisfies the RF requirement.

## B-17 mirror-location note (scope honesty)

The card names `roadmap.md:383` as a verbatim mirror of the false claim. As of this run the
**live** `.night-crew/knowledge/roadmap.md` `deploy-hygiene-honesty` card is a *paraphrase*
("justifies a load-bearing flag with an empirically false claim") — it does NOT carry the
false sentence verbatim; the `:383` line pointer is stale (Cards 1–10 reflowed the file). The
verbatim claim survives only in FROZEN historical artifacts:
`reference/roadmap-2026-08-05-sync-foundation.md:449` (a dated archived snapshot) and
`runs/2026-07-29-autonomous/merge-intent-a-precache-manifest-from-head.md:75` (a frozen run
artifact — do-not-edit per CLAUDE.md). Those are the historical record and are correctly left
as-is. So B-17's live-source correction is `build-sw.js:46`; the live roadmap card needs no
claim-correction (it is already honest). This is stated so a reviewer does not read the absent
`:383` edit as a missed location.

## 🛑 Env finding for the orchestrator — workbox version drift (READ before re-running `task sw`)

This worktree's **symlinked `node_modules` carries `workbox-build@7.3.0`**, but
`package-lock.json` pins **`7.4.1`**. Regenerating sw.js here (`node build-sw.js`) therefore
produces a **spurious workbox-runtime-chunk-hash-only** delta:
`define(["./workbox-0225851e"…])` → `define(["./workbox-d4a0f5c1"…])`, plus a new
`workbox-d4a0f5c1.js` chunk file. **Every precache `revision` hash — including
`version.json` (`226015341780c9f481a7d23133f4c77e`, 21 bytes, unchanged) — is byte-identical;
the count stays 31.** This drift is environmental, NOT a product of this card.

I did NOT commit that drift. I restored the committed sw.js. My source change (comment-only
in `build-sw.js`/`write-version-json.js`; one `\n` in `backend/Dockerfile`) touches **no
precached file** and does not change the version.json bytes, so sw.js needs **no regeneration**
for B-135/B-17 — the committed sw.js already correctly describes the 21-byte version.json the
authoritative generator writes, and post-fix the Docker generator writes those same bytes.

When the orchestrator re-runs `task sw` on the merged tree after Cards 7 and 11 land: if the
merge node_modules is still 7.3.0, sw.js will regenerate with the `d4a0f5c1` chunk — that is
the workbox drift, not a Card-11 or Card-7 change. If a version-parity or workbox-pin card is
wanted, that is a SEPARATE backlog item (worktree node_modules ≠ locked workbox); flag it, do
not fold it into this card's sw.js.

## PARK section

Not parked. The unification is byte-identical — same artifacts, same paths, same release-flow
shape; nothing is removed or relocated. Proceeding per the slate's explicit allowance.
