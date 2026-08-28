# UI Design Rules — distilled from operator walkthrough fixes

> **Purpose.** Reusable UI rules extracted from bugs the operator hit while walking the
> Onboarding, Operations and Users apps on a phone (2026-08-26 → 2026-08-28, branch
> `fix/remote-failcard-reconcile`). Written rule-first so review tooling can consume them
> as a checklist: every rule carries a **Check** that a critic, reviewer, or test can
> execute. Cite rules by id (UI-R1…UI-R7).
>
> **Consumed by:**
> - **Every Claude session** — pointer in `CLAUDE.md` §Conventions.
> - **night-crew** — candidate preferences recorded with operator consent 2026-08-28 in
>   `.night-crew/knowledge/preferences/` (`ux.md` C-2/C-3, `design.md` C-1,
>   `process.md` C-2), each citing its rule here.
> - **/ui-jury** — no repo-level conventions input exists today (the skill's calibration
>   fixture anchors verifier *scoring*, not design rules). These rules are written in
>   critic-consumable shape; when ui-jury grows a project-conventions input, point it at
>   this file. Until then, cite rule ids by hand in review prompts.
>
> **Predecessor:** `docs/ui-bug-post-mortem.md` (2026-04-26). Its patterns #2 and #3
> recurred verbatim in the FAQ bug below — a narrative post-mortem nobody re-reads does
> not prevent recurrence. Wiring rules into guaranteed consumers is the fix attempt.

---

## UI-R1 · Overlay controls stay out of native-chrome territory

- **Rule:** A custom control overlaid on media or fullscreen surfaces must not sit in
  regions the platform owns: the corners over a `<video controls>` element (iOS draws
  fullscreen/PiP top-left and mute top-right), the status-bar strip, or the notch.
  Top-center is the one region iOS video chrome never occupies. In standalone PWA mode,
  offset with `env(safe-area-inset-*)`.
- **Incident:** the crew video player's close was a 36px circle at `top:12px;right:16px`
  — directly under the native mute icon, small and easy to miss (operator screenshots,
  2026-08-28). Fixed as a top-center pill in `353b423`.
- **Check:** screenshot each overlay at 393×852 with native controls visible; no custom
  control's box may intersect the native-chrome corner regions or the status bar.

## UI-R2 · The way out is labeled, 44px, and obvious

- **Rule:** The escape action from any fullscreen or blocking surface (video player,
  modal, picker) is a **labeled** control ("✕ Close"), minimum 44px touch target,
  high-contrast against its background (ring/border when floating on media), positioned
  per UI-R1. A bare glyph is not sufficient prominence for the only way out.
- **Incident:** same close-button fix; operator: "should be more prominent, or an
  entirely different UI element."
- **Check:** for every fullscreen surface, locate the exit control; assert it has a text
  label, a ≥44px target, and visible contrast in a screenshot over real content.

## UI-R3 · Blank is never a valid render

- **Rule:** `undefined`/missing data in a user-visible slot must never silently coerce
  to an empty string. Escape/format helpers (`escapeOBAttr`, `titleCase`, …) render what
  they are handed — the call site must not hand them `undefined`. Missing required
  display data renders a visible placeholder or a loud error. A row whose primary text
  region is empty is a defect by definition, not a styling state.
- **Incident:** FAQ rows rendered `escapeOBAttr(faqItem.question)` where the API serves
  the question as `label` — three blank bars showing only disclosure triangles
  (`2e078d3`). The blankness *looked designed*, which is why it survived a walkthrough
  until the operator squinted.
- **Check:** in screenshot review, any card/row/header whose primary text region is
  empty fails. In code review, every field referenced in a render template must exist in
  the API response it renders from.

## UI-R4 · The frontend speaks the backend's field names

- **Rule:** Before rendering or persisting a field, verify the exact JSON key against
  the Go struct tag — the struct is the contract of record, not the frontend's naming
  taste. Go's `json.Unmarshal` silently drops unknown keys, so a wrong key produces no
  error anywhere: it surfaces as blank UI (read path) or a silently discarded edit
  (write path). Trace the full round trip: frontend state → API payload → Go struct →
  SQL → response → hydrate.
- **Incident:** the frontend invented `question` in four places; the backend's contract
  is `Item.Label` (`db.go:84`). Blank crew rows, blank Builder inputs, question edits
  black-holed on decode, and type-less new items broke template saves outright
  (`2e078d3`). **This is a recurrence of `ui-bug-post-mortem.md` patterns #2 and #3.**
- **Check:** every new render/persist site names the struct field + json tag it binds
  to; every new data concept ships with the CLAUDE.md persistence regression test
  (create → save → reload → assert still there).

## UI-R5 · Assert content, not containers

- **Rule:** An E2E assertion guarding user-visible text must assert the **text itself**
  (`toContainText` / `toHaveValue`, with the expectation sourced from the API or seed),
  never just element visibility. A container with decoration (icons, triangles,
  borders) is "visible" while its content is blank — the assertion passes on the
  strength of the decoration alone.
- **Incident:** `expect(page.locator('.faq-q').first()).toBeVisible()` passed for the
  entire life of the blank-question bug; the disclosure triangle satisfied it. The
  strengthened test (question text asserted from the API's `label`) went red instantly
  on the unfixed code (`2e078d3`).
- **Check:** in spec review, flag `toBeVisible()` on any element whose purpose is text
  content; require a companion text assertion whose expected value comes from data, not
  a hardcoded copy of the template.

## UI-R6 · Failures are loud, retryable, and overridable

- **Rule:** Every async failure a crew member can hit (media load, data fetch, upload)
  has three parts: a **visible error state** naming what happened in crew language, a
  **Retry** affordance, and an **authority override** path that unblocks the flow when
  retry cannot (e.g. manager marks a step watched). Nobody gets silently stranded
  mid-task on a phone in a truck.
- **Incident:** onboarding video failures were silent — the player just never played.
  `7d9efcc` added the loud "Video unavailable" overlay + Retry + manager mark-watched
  override + the recovery tool for moved/missing files.
- **Check:** the phase's State Enumeration Table must contain the failure row for every
  async dependency; force the failure in a spec, screenshot it, assert the error text
  and Retry control exist, and name the override path in the table row.

## UI-R7 · A new state transition updates every projection of that state

- **Rule:** When a state machine gains a transition (reject, unsubmit, redo), enumerate
  every place that state is **projected** — count chips, badges, list rows, cards, and
  live-sync views on other devices — and update each one. A correct transition with a
  stale projection reads as data loss or phantom progress to the crew.
- **Incident:** rejected sub-steps kept their checkmarks and inflated the progress count
  until `0b9752d` reset them to NULL and deducted; fail-card rejection flags didn't
  reconcile when a redo op arrived live until `27aa079`.
- **Check:** any PR changing a state transition lists the projections it touched; the
  footprint's spec subset covers each (the count, the reopened view, and the live peer
  where sync applies).
