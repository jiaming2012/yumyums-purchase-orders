# HANDOFF — overnight run `20261006` (night of 2026-10-05 → morning 2026-10-06)

**Branch:** `overnight-20261006`, cut from `dev` at `bf89a26`. Nothing pushed, `main` untouched, no
deploy, no message sent to anyone, no provider called. **Slate:** `reference/slate-20261006.md`,
signed 2026-10-05 — 2 cards, Activity E, serial. Launched 19:22, run submitted 19:38, both cards
parked by 21:06, closeout written ~22:00 America/New_York.

## 🛑 Read these five things first

1. **Nothing landed. 0 of 2 cards merged; both are parked on one decision that is yours.**
   `DECISIONS-NEEDED.md` carries it in full.
2. **Why: the tablet's "Scan from photo" does not read about one customer code in seven** — the
   same codes every time, never a wrong customer. Card 1 (the code and its QR image) was built and
   proven to its spec; its own session then measured this and stopped rather than land it or
   rewrite its gates to go green. An independent re-check confirmed the number (87% vs 85%) and
   found the cause is the tablet page's reader, not what the code carries. Turning the picture and
   retrying read 600 of 600 on a scratch copy.
3. **Two things signed at the 2026-10-05 sitting were measured to have no effect:** the lighter
   error-correction level and widening the scan area to 640. The spike behind them drew one sample
   per variant.
4. **The single most useful thing you can do before deciding is on the truck's tablet:** photo-scan
   a few of the failing sample codes. Whether that tablet has a built-in barcode detector was not
   measured, and it decides how urgent this is.
5. **This was the first night on night-crew's run loop, and the loop carried it.** `loop check`
   passed (exit 0, both cards accounted for on the loop, both declared parked). It ran on its own
   lane because another night was already running on the shared one — see "How the night ran".

## Per-card outcomes

| # | Card | Outcome | Where the work is |
|---|---|---|---|
| 1 | `identity-code-and-qr` | **PARKED** by its build session after 86 min — built as specified, not committed, not merged | commit `fe45768` (preserve ref) + `.night-crew/runs/20261006/identity-code-and-qr/changes.patch` |
| 2 | `mms-send-on-signup` | **PARKED with Card 1** after 1 min — nothing built, nothing run | — |

Roadmap Activity E lines stay `PLANNED`. `night-crew.toml`, `sw.js` and the precache count (51)
are unchanged — nothing reached the run branch but documents.

## What Card 1's session ran and observed (un-landed; from its result report and logs)

- Red first: Go build failure on the new tests; 3 Playwright reds on the pre-change tree (`/m/…png`
  answered 404).
- Green: its five named Go tests plus four more, 9/9, with no egress-allowlist change; build + vet
  exit 0; full Go suite 463 pass / 0 fail / 3 skip, 15 packages, under the lock.
- Marketing browser subset, no retries: 131 of 132 pass. The one failure is `[IC-03]`, the signed
  negative control ("the stronger-correction code must not read") — it does read, as often as the
  card's own code. `[IC-01]`/`[IC-02]` pass only when the freshly minted code happens to be a
  readable one; with the config's one retry they would mostly pass and hide the problem.
- **Not run, by anyone:** the full browser suite on a merged tree, `sw.js` regeneration, and the
  separate adversarial review — none applies until something is merged.

## Base measurement (unmodified `dev@bf89a26`, config's own retries = 1)

- Go: `EXIT_TEST=0` — 456 pass / 0 fail / 3 skip, 15 packages.
- Browser suite: `EXIT_PW=1` — 26 failed / 2 flaky / 6 skipped / 1045 passed, 57.1 min. List in
  `logs/base-reds.txt`.
- Against run 20261005's base: three more reds (`onboarding.spec.js:2233`, `inventory.spec.js:1469`,
  `inventory.spec.js:2919` — flaky last night), one gone green (`inventory.spec.js:2931`). None is
  in the marketing area. Not isolated tonight — there was no final tree to compare against.

## How the night ran

- **Own lane (your choice at launch).** The night-crew clone's own run, also numbered `20261006`,
  was already executing on the shared scheduler lane with its own worker. hq ran in a separate
  Temporal namespace, `night-crew-hq`, with its own worker built clean from the same tool revision
  as the installed binary (`f56da33`). The other night was not touched. The namespace still
  exists; hq's worker was stopped at 21:47.
- **One `night-crew run`, both cards, width 1.** Records: `.night-crew/runs/20261006/` (summary,
  journal, metrics, plan, per-card session logs). Scorecard record committed
  (`scorecard/20261006.jsonl`); `night-crew scorecard --repo .` exit 0.
- **Queues at closeout** (`logs/workers-check-close.log`, 21:5x): hq's lane — no pollers. Shared
  lane — the night-crew clone's worker (pid 45477) still polling, which is that night's, not a
  leftover of this one.
- The 45-minute watchdog never fired; every ten-minute snapshot showed progress.

## Findings about the tooling and the slate (for triage to file or drop)

- **A slate written for the loop needs runnable gate lines.** The loop executes only fenced
  commands in a card's Gates; prose gates run nothing. Tonight's loop gate was compile + vet.
- **The loop shows a build session only Intent, Spec and Context** — the Gates text had to be
  copied into Context for the session to see what it must prove.
- **`night-crew run`'s default deadline is 2 hours**; the launch prompt's command names none.
- **The loop commits a card as one commit and the session cannot regenerate `sw.js`** (it is built
  from committed state), so "atomic commits", the run trailer and the `sw.js` step in the launch
  prompt have to move to the orchestrator's merge.
- **Two nights can collide on one scheduler lane and one run number** with nothing but
  `workers check` to notice; it noticed only because it was re-run after the other night started.
- **B-486 is one directory short:** the browser suite rewrites committed images under
  `runs/2026-10-02-autonomous/logs/h5/states/` as well as `h4/states/`. Both were restored on every
  leg tonight; `launch-20261007.md` names only `h4`.
- **`night-crew decisions check`** again flagged "not a user story / no evidence" on a question
  that stated role, want, benefit and evidence. Asked as drafted.

## Left in place, deliberately (nothing was deleted)

- Loop branches `night-crew/run/20261006`, `night-crew/20261006/identity-code-and-qr`,
  `night-crew/20261006/mms-send-on-signup`, and their three worktrees under
  `.night-crew/runs/20261006/` (both card worktrees are clean at `efcd970`).
- Scratch worktrees `hq-worktrees/base-20261006` (at `dev`) and `hq-worktrees/verify-e1-20261006`
  (at `fe45768`), both clean.
- Test databases on :5434: `hq_test_go_base`, `hq_test_e2e_base_20261006`,
  `hq_test_e2e_verify_20261006`, and Card 1's own. :5433 was never touched.
- **Your two uncommitted preference files** (`preferences/process.md`, `preferences/ux.md` — the
  adoptions you made at the terminal on 2026-10-05) were uncommitted when the branch was cut and
  are still uncommitted and unmodified by the night.

## Next actions

1. `/nc-morning-triage` — answer Decision 1. There is little to merge: `overnight-20261006` carries
   only run documents and the scorecard record.
2. **Slate `20261007` should not launch as written.** Its prompt requires `20261006` to have run
   and been triaged (it will have), but it also assumes Activity E's code is on `dev`; it is not.
   Its four cards do not depend on Activity E, so this is a wording check at triage, not a blocker.
3. The milestone tally is unchanged: Activity E's two cards remain white, plus
   `external-accounts-provision`. The attended acts (SignalWire provisioning, the test send to your
   phone, the real redemption) still wait behind Activity E.

## Standing flags after morning triage 2026-10-06 (ledger T-70)

- **Decision 1 (unreadable customer codes) — OPEN, waiting on the tablet check** (decision 210).
  Kit and instructions: `tablet-check/README.md`. Clears when the operator reports what the
  truck's tablet showed and picks a fix. Activity E's two cards stay PLANNED until then.
- **Card 1's built work is on no branch** — `fe45768` and `changes.patch`. Armed until it is
  re-dispatched or dropped; do not delete `hq-worktrees/verify-e1-20261006` or the preserve ref
  before then.
- **Corrections to this document from the triage reviewer** (evidence: `logs/triage-adversarial/`):
  "600 of 600" is not every code (B-489); Low vs Medium does differ; a new picture of the same
  code does NOT fail the same way; 2 of 48 failures showed "Not a Yumyums code" (B-490).
- **Slate `20261007` may launch as written** (decision 211) — its four cards do not touch Activity
  E. Launch notes are in `reference/triage-20261006.md`.
- **Temporal namespace `night-crew-hq` still exists** (B-494). Not removed by triage.
- **Carried unchanged:** `card/a3-rls-fixture-own`, `card/s2-demo-sync-target` (left in place by
  ruling, B-442); B-474 production count before the `0086` deploy; decision 203 deploy note.
