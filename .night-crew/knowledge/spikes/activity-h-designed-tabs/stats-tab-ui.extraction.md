# Extraction — stats-tab-ui

Outcome: learned

Approach used: the handoff §5 metric definitions executed over a fixture
(codes with channel + item, first-touch `source_short` on subscribers,
matched / unmatched / declined / duplicate-scan attempts, one direct
redemption) by a Node script asserting Σ(by campaign) == Σ(by channel) ==
Σ(by item) == overview on every money and funnel column. One spike, red on
run 1, exit 0 on run 2 after a signed correction.

Confirmed: under first-touch attribution the three slices reconcile to the
overview for scans, signups, redeemed, revenue, discount, implied and net;
the orphan rate counts declines except `duplicate_scan`; `per_dollar` is
null at zero discount.

Learned: the discount basis cannot be chosen per table. Choosing "actual if
fully matched, else implied" per group made a fully-matched slice disagree
with a partly-matched overview by exactly one row's actual−implied gap. The
rule that reconciles is PER ROW: a matched attempt contributes the Toast
order's actual discount, an unmatched one contributes the campaign's face
value, summed identically on every slice and the overview; `basis` is a
display label (actual / implied / mixed).

Plan change: decision 190 refined to the per-row rule (handoff §3 row 190
and §5 metric definitions updated 2026-10-01, operator-signed). The card's
`GET /stats/by` and `GET /stats/overview` must compute discount per row and
carry both `discount_cents` and `implied_cents` so the UI can render the
pair where they differ.
