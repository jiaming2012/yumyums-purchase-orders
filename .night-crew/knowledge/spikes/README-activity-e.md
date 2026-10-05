# Activity E — Customer delivery (one identity code → QR → image)

> Ledgers live in `activity-e-customer-delivery-one-identity-code-qr-image/` (the slug the spike
> gate derives from the activity title); this README sits beside that directory because the gate
> reads every `.md` inside it as a goal ledger.

Two goal ledgers, one per card, authored at the slate sitting of 2026-10-05 (the evening after
morning triage T-66; the operator's instruction that this loop clears the milestone, ledger T-68).
Runs are tool-recorded (`night-crew spikes run --activity … --goal …`, binary v3.6.1+4), so the
ledgers' `### Runs` lines are the verdict the gate reads. Scripts live under
`.night-crew/spikes/activity-e-customer-delivery-one-identity-code-qr-image/<goal>/`.

What the spikes settle, and what they deliberately do not: the scanner side of
`identity-code-and-qr`'s done_when is ALREADY covered by shipped tests (`tests/marketing.spec.js`
— offline embedded offer, online/replica offer list), so E1's spikes prove the SERVER half (a
server-generated hybrid QR the shipped reader decodes; an identity-code row the tablet's offers
pull can see). E2's spikes prove the sender's home (outside `internal/marketing`, by the egress
guard's own refusal) and the provider call's mockability at the HTTP layer, and record the
consent / STOP baseline. **No spike talks to SignalWire** — live acceptance is D-KR1's attended leg.

Substrate discipline: `:5434` (`yumyums-test-pg`, role `hqtest`) for every Postgres write, in
spike-owned databases `hq_test_spike_e<n>_20261006` / `hq_test_spike_e<n>_go` (the reset guard's
pattern); mutations happen in THROWAWAY git worktrees under `../hq-worktrees/spike-e<n>-20261006`
that the scripts remove on every exit path. E1 spike 02 writes ONE row to the LOCAL
`spike-supabase` substrate (`public.codes`, id `e1000000-…-000000000001`) and deletes it on exit.
Never `:5433`.
