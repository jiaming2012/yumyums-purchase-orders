# DECISIONS-NEEDED — overnight-20260718 (for the operator, morning of 2026-07-18)

> The run **never decides — it executes** (launch contract). One bounded coverage residual
> surfaced. It did **not** block the card: `editprop-convergence-cell-hardening` is **DONE**,
> both halves landed, the operator rider is retired. This is a *should-we-follow-up* call, not a
> fork the run improvised around.

## D-1 — 2 fail-note field types are outside the W-6 conflict-coverage reach (footprint-blocked)

**What.** Half 2 of the card extended the W-6 LWW-409/`applyOp` conflict coverage from text-only to
the remaining persisted types. **4 landed** (yes_no, temperature, sub-step, checkbox — each
red→green, G6-verified). **2 did NOT:** `fail-note text+severity` and `fail-note photo-URL`.

**Why they're blocked (verified, not a guess).** The fail-note payload is the `{_v, _fail_note}`
bundle. On the incoming-op render path, `applyOp`'s `SET_FIELD` branch (`sync.js:405–441`) unpacks
only `value` / `value.value` (+ `sub_steps`) — it has **no `_fail_note` unpack** (grep for
`_fail_note`/`fail_note` in `sync.js` returns none). The bundle is unpacked ONLY by
`hydrateFieldState` (workflows.html, ~:1480), which runs on load/reopen, **not** on a live incoming
op / 409 conflict render. So a conflict-branch cell for a fail-note type cannot today assert the
loser renders the winner's fail-note — the value would render as neither a plain value nor a fail
card. Covering these two types requires a **production change to the applyOp render path
(`sync.js` / `workflows.html`)** — outside this card's declared test-only footprint. The
implementer correctly **declined to breach footprint** and parked instead; G6 independently
confirmed the block is real, not laziness.

**Is this a latent product gap or just a test gap?** Worth the operator's eye. It means: *if two
devices conflict-write a fail-note (value + note + severity, or a photo URL) and one loses LWW, the
losing device may not live-re-render the winning fail-note over the 409 path* — it would reconcile
only on the next reopen/hydrate. Whether that's a real-world scenario worth hardening (crew rarely
both edit the same fail card's note simultaneously) is a product/priority call. The other 4 types
(the common answer types) ARE covered.

**Options for the operator:**
- **(a) Accept as-is** — the 4 common types are covered, the rider is retired, fail-note conflict
  live-render is a rare path; log the gap and move on.
- **(b) Graduate a follow-up card** — "extend `applyOp` to unpack `_fail_note` on the incoming-op /
  409 path, then add the 2 parked W-6b cells." This is a *production* change (applyOp render path),
  so it wants its own design/footprint + G6 — it is **not** a test-debt card. Candidate
  `/nc-okr-session` feedstock or BACKLOG, next cycle. Note the sibling BACKLOG item F-B
  (transactional Create/Archive op emission) also touches op emission — may be worth bundling the
  op-path work.

**Recommendation (non-binding — the run does not decide):** (a) for tonight's close; consider (b)
only if fail-note concurrent-edit is a real crew pattern. Either way this does **not** hold the
card or the rider retirement.

---

_No other forks. No operator-only questions arose during execution. Nothing else parked._
