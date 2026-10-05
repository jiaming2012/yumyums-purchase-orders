# Spikes — inventory-setup-races

Activity: Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> Tool-run (`night-crew spikes run`). One Playwright spike in a THROWAWAY worktree off `dev`
> (`../hq-worktrees/spike-k1-20261007`) on a spike-owned e2e stack
> (`hq_test_spike_k1_20261007`, `TEST_PORT=8341`, `:5434`). Never `:5433`.

## The goal, and which legs need a spike

The card (K1 — B-459, B-478; triage T-66 named them the obvious next small card): `inventory.html`'s
`ALL_ITEMS` has three unsequenced writers, so a stale response can overwrite a fresh one and
revert a nickname the server kept (B-459); and `loadItems()`'s late re-render of the Setup add
bar empties a name already typed, after which the create click does nothing (B-478). Two premises
a script can settle now, by execution through the SHIPPED page with only the network timing
altered: (1) with `GET /inventory/groups` delayed 600 ms, a name typed right after opening Setup
is gone when the response lands and the create click sends no POST; (2) with the Setup tab's
first `GET /inventory/items` held until after an alias POST and the handler's own refetch, the
chip count reverts from 2 to 1 when the stale response is released — and the server still holds
both aliases (the view lies).

## Spike: late-responses-wipe-the-add-bar-and-revert-the-chips

- proves: (a) B-478 — delay `GET /api/v1/inventory/groups` 600 ms, open Setup, type a name
  immediately: after the response lands `#new-item-name` is EMPTY and a create click sends zero
  `POST /api/v1/inventory/items`; control with no delay keeps the typed name; (b) B-459 — hold
  the first `GET /api/v1/inventory/items` the Setup tab fires, add a nickname through the UI
  (its POST and refetch pass), see 2 chips, release the held response: the chips revert to 1
  while `GET /items` from the server still lists both aliases.
  **Amended after runs 2 and 3 (both exit 1, below — FINDINGS, not script bugs):** premise (b)
  did NOT reproduce. Run 2: chips 2 while held, 2 after release, the view shows the typed
  nickname and the server holds both. Run 3, identical timing: chips 1 while held, 1 after,
  and the server holds ONLY the seeded alias — the add never reached the server. In neither run
  did the view disagree with the server. Leg (b) now records the invariant both runs showed —
  **the view and the server AGREE** — and logs `addDropped`; a red on it would mean the filed
  "view lies" mechanism reproduced after all.
- plan: worktree, warm build, copy a three-spec Playwright file into the worktree's `tests/`,
  run on the spike stack, read the outcomes from the JSON reporter.
- script: .night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/inventory-setup-races/01-late-responses-wipe-the-add-bar-and-revert-the-chips.sh

### Runs

- 2026-10-05T15:07:25Z · exit 1 · failed
- 2026-10-05T15:11:41Z · exit 1 · failed
- 2026-10-05T15:15:19Z · exit 1 · failed
- 2026-10-05T15:19:06Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **late-responses-wipe-the-add-bar-and-revert-the-chips: passed** on its fourth run (exit 0,
  2.7 m): (a) B-478 — groups delayed 600 ms → `{"before":"Spike Typed Item","after":"","posts":0,
  "editForms":0}`: the late re-render wiped the typed name and the create click sent no POST;
  control (typed after networkidle) kept `"Spike Control Item"`; (b) B-459 — three measured runs
  under the filed timing (the Setup tab's first `GET /items` held until after the add, then
  released): run 2 `{chipsWhileHeld:2, chipsAfterRelease:2, viewHasTyped:true, server:[both]}`;
  run 3 `{1, 1, false, server:[seeded only]}`; run 4 `{1, 1, false, server:[seeded only],
  addDropped:true}`. **In no run did the view disagree with the server** — the filed "stale
  response reverts the chips while the server keeps both" mechanism did not reproduce. **In 2 of
  3 runs the add never reached the server at all** — a dropped click/POST while the first items
  fetch is in flight, a different and nondeterministic failure that produces the SAME symptom
  the suite sees (`toHaveCount(2)` → 1). The four runs' lines: run 1 — a sitting-side control
  bug (typed before the page settled under three concurrent stacks); runs 2 and 3 — the premise
  being corrected by what was observed (named above); run 4 — green on the corrected premise.

## Corrections

- **One, agent-reached, carried into the card:** B-459's diagnosed mechanism is wrong. The
  card's B-459 half becomes (i) a diagnosis-and-fix of the DROPPED ADD using this spike's leg 3
  as the reproduction (2 of 3 runs), with the night naming the mechanism it finds, (ii) request
  sequencing on `ALL_ITEMS`/`ITEM_GROUPS` as hardening, and (iii) the 10×/5× measurement that
  alone may retire `inventory.spec.js:2186`'s red. `[IS-04]` added to the card's done_when.
  B-478 is unchanged — reproduced exactly as filed.

## Comebacks

- Sitting-side: a control that types "after the page settled" must wait for network idle, not
  a fixed delay — under three concurrent Playwright stacks the item list took longer than 800 ms.
- For the backlog (B-459's entry): the filed mechanism is superseded by this spike's
  observation; the symptom stands.

## Review

- signed: operator, 2026-10-05 — covers 1 correction(s) (reviewed at the slate sitting of
  2026-10-05, slate-20261007 §4 batch sign-off: the B-459 correction and `[IS-04]` on screen).
