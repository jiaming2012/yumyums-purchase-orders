> **RESOLVED 2026-09-05 — recorded as `ledger.md` §T-54 (decisions 170–172).**
> The run's own statement below is confirmed: nothing was parked, no delegated decision awaited
> ratification, and `night-crew decisions ratify` / `preferences ratchet` both returned empty at
> triage. The one operator call this triage put was **not** a fork the run left open — it was
> raised BY the triage's adversarial review: the must-verify-online refusal fails open while the
> campaigns replica lags (B-432). Answered *"merge, and block close-bar leg 3 on the fix"*
> (decision 170). Kept as the analysis record.

# DECISIONS-NEEDED — overnight run 20260906

**Nothing parked, and no delegated decisions await ratification.** Stated explicitly so
an empty file and logging-that-never-ran don't look alike:

- No card parked — the night's one card merged (G6 APPROVE).
- No `night-crew decisions log` record was created this run: no gray area reached the
  resolver. The slate pre-delegated the mechanics calls (replica-vs-RESYNC-fan-out,
  migration shape, constraint form, harness design) to the night, and both narrow PARK
  conditions were checked and NOT met — the §9/§19 re-read (implementer and G6
  independently) found `status='accepted'` + flags fits the existing attempt taxonomy
  with no new terminal status, and the `requires_online=true` refusal was strengthened,
  never weakened.
- Operator forks: none open. D-1 was already ratified pre-run (decision 166, ledger
  T-53); tonight's card is its rider, now discharged.
