# Decisions needed — overnight-20260712 (for the morning of 2026-07-12)

> Written for the operator at morning triage (`/nc-morning-triage`). These are **not
> parked cards** — every one of the 10 slate cards completed and passed G6. The items
> below are **forward operator forks** the run surfaced but (per the rules of engagement)
> does not decide: waivers and denominator scoping are operator sign-offs, like D-3
> (Inventory Trends/Cost) at the 2026-07-10 triage.

## D-5 (recommendation) — Waive Onboarding FR-16 + NFR-4 (video pipeline) as environment-gated?

**What:** Onboarding FR-16 (video presign→PUT→FFmpeg transcode/thumbnail) and NFR-4
(the `503 video_storage_not_configured` fallback + FFmpeg-missing path) are **fully
implemented** — `handler.go:540-640`, `video.go:22-206` (real `convertToMP4`,
`extractThumbnail`, `downloadFromSpaces`/`uploadToSpaces`, `processVideo`). They are
**not broken**; they are untestable in the E2E environment without DO Spaces creds + an
`ffmpeg` binary on the host. Both stay **UNPROVEN** after the confirm-absence sweep.

**The fork:** waive FR-16 + NFR-4 from the Engineering-KR "0 known-broken flows"
denominator as **unbuilt/untestable-in-env** (parallel to D-3's Trends/Cost waiver), so
they ship as-is and don't block the cycle gate — **OR** stand up a Spaces+ffmpeg test
fixture in Activity 4 and prove them.

**Run's recommendation:** waive (env-gated, not a shipped-feature defect). This matches
the PRD's own "likely waiver candidates" note and the exemplar's confirmed-only-BROKEN
discipline. Decide at triage; the run does not waive on its own authority.

**Contrast:** NFR-5 (video-led reopen no-op) is **NOT** a waiver — it is a confirmed
BROKEN in a shipped flow (real code, wrong behavior) → an Activity-4 fix-card (BACKLOG).

*(No other decisions open as of the cards completed so far. Appended to during the run if
anything parks.)*
