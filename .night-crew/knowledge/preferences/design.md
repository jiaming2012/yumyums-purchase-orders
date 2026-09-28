# Preferences — design

> How the product is shaped: defaults, naming, information architecture.
>
> Weighted opinions, not binding rules: a session may deviate, but must say why.
> Add entries as `P-1`, `P-2`, … in the shape of the commented example below —
> `night-crew preferences validate` checks that shape, never your judgment.

<!--
## P-1 · A short title naming the leaning

- **Preference:** what to do, stated so a session can act on it.
- **Why (operator):** the reasoning, in your own words — this is what gets cited back to you.
- **Weight:** strong — and then whatever qualifier you want (start with strong, moderate, or weak).
- **Evidence:** optional link to a research note or a past decision.
- **Recorded:** 2026-01-01
-->

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them, nothing validates them, and no
> command promotes them. To adopt one, renumber it to the next free `P-n`, move it up
> above this section, and delete what you don't want. To drop one, delete it.

## C-1 · Overlay controls stay out of native-chrome territory

- **Preference:** Custom controls overlaid on media or fullscreen surfaces avoid the regions native chrome owns — the corners over a `<video controls>` element (iOS draws fullscreen/PiP top-left, mute top-right), the status-bar strip, the notch. The exit affordance from a fullscreen surface is a labeled ≥44px pill ("✕ Close"), top-center for fullscreen media (the one region iOS video chrome never occupies), offset by `env(safe-area-inset-*)` in standalone PWA mode.
- **Why (operator):** requested after the 2026-08-28 walkthrough — the video player's 36px close circle sat directly under the native mute icon and was easy to miss; operator asked for "more prominent, or an entirely different UI element … top center … or between the center and right margin" and picked the top-center labeled pill from mockups.
- **Weight:** strong
- **Evidence:** close-pill fix, commit 353b423 (2026-08-28); docs/ui-design-rules.md UI-R1/UI-R2.
- **Recorded:** 2026-08-28
- **Offered at:** an attended session, 2026-08-28
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · Fail-open behind an entitlement when policy data hasn't arrived

- **Preference:** When a safety refusal depends on data the client cannot yet read, ship the permissive default gated behind an explicit per-user entitlement plus a confirmation, audit-flag every use, and require a named follow-up card that arms the real refusal before the feature meets real data — never a refuse-all default that deadens the product surface, and never an ungated permissive default.
- **Why (operator):** Operator ratified D-1 (run 20260905) at morning triage 2026-09-05: unknown→false stands as shipped — force-submit behind the marketing-offline-override entitlement + §13 confirmation, every attempt audit-flagged offline_override=true — with follow-up card requires-online-replication required before any real campaign is provisioned. No further reason stated.
- **Weight:** moderate
- **Operator:** jamal@Jamals-MacBook-Pro.local
- **Recorded:** 2026-09-05
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
