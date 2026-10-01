# Spikes — stats-tab-ui

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> Hand-run convention (see `campaign-codes-api.md` header). No substrate
> involved — the premise here is arithmetic.

## The goal, and which legs need a spike

The card (H4): the Stats tab — overview with revenue / discount / net, three
slices (campaign / channel / item) with the same money columns, the
reconciliation queue, decline-with-note, the declined bucket. The one premise
that would make the tab wrong on day one is that the three slices and the
overview can disagree: the metric definitions in handoff §5 must make them
sum identically under first-touch attribution, with the implied→actual
discount rule (decision 190) and declines in the orphan rate.

## Spike: slices-reconcile

- proves: on one fixture (codes with channel + item, subscribers with a
  first-touch `source_short`, accepted attempts with matched / unmatched /
  declined / duplicate-scan outcomes, one direct redemption with no
  first-touch), the §5 definitions make Σ(by campaign) == Σ(by channel) ==
  Σ(by item) == overview for scans, signups, redeemed, revenue, discount, net;
  the orphan rate counts declines except `duplicate_scan`; `per_dollar` is
  null at zero discount.
- plan: a Node ESM script implementing the definitions; `assert.strict` on
  every column per slice.
- script: .night-crew/spikes/activity-h-designed-tabs/stats-tab-ui/01-slices-reconcile.sh

## Verdict (hand-run 2026-10-01)

- **slices-reconcile: failed (run 1) → passed (run 2)** — run 1 exited 1:
  `net` Σ(by campaign) = −1100 vs overview −1150. Cause: the first draft
  chose the discount basis per *group* (actual when every row in the group is
  matched, otherwise implied), so a fully-matched slice used actual while the
  partly-matched overview used implied — the gap is exactly one row's
  actual−implied difference (250 vs 300). Run 2 after the correction below:
  exit 0; all three slices Σ scans=12 signups=5 redeemed=7 revenue=4350
  discount=5450 net=−1100 == overview; orphan rate 2/7 with duplicate_scan
  excluded; per_dollar null at zero discount.

## Corrections

- **discount basis is per ROW, never per group** — the agents changed
  `money()` to sum, row by row, the matched order's actual discount where
  there is one and the campaign's face value where there is not, on every
  slice and the overview alike; `basis` became a label (`actual` / `implied`
  / `mixed`) instead of a choice of arithmetic, and the implied total is still
  carried so the UI can render "implied −$X · actual −$Y" where they differ.
  **This refines decision 190's wording** ("implied until actual; both render
  when they differ"): the *totals* use the per-row rule; "both render" is
  about display. Queued for the operator's batch review; on sign-off the
  handoff §3 row 190 and §5 metric definitions get the per-row sentence.

## Comebacks

- none recorded — the gap was in the spike's own premise and is closed by the correction above

## Review

- signed: operator, 2026-10-01 — covers 1 correction(s) (batch sitting, presented as user stories at the operator's request; "Sign all three")
