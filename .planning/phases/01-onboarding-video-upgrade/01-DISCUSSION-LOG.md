# Phase 1: Onboarding Video Upgrade - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-19
**Phase:** 01-onboarding-video-upgrade
**Areas discussed:** Video upload experience, Inline player behavior, Watch enforcement strictness, Thumbnail strategy

---

## Video Upload Experience

| Option | Description | Selected |
|--------|-------------|----------|
| Radio toggle (Upload/URL) | Switch between file picker and URL text field | ✓ |
| Smart detect | Single input that accepts both pasted URL or file drop | |

**User's choice:** Radio toggle — Upload as first/default option, URL second
**Notes:** User wants Upload to be the primary choice since crew will mostly upload from device

### File Formats

| Option | Description | Selected |
|--------|-------------|----------|
| MP4 only | Universal browser support | |
| MP4 + MOV | Covers iPhone recordings | |
| MP4 + MOV + WebM | Broadest coverage | ✓ |

**User's choice:** MP4 + MOV + WebM accepted, server converts non-MP4 to MP4 via FFmpeg
**Notes:** User asked about client-side conversion — advised against (heavy on phones). Server-side FFmpeg chosen. API rejects formats other than these three.

### File Size

| Option | Description | Selected |
|--------|-------------|----------|
| 50 MB | Fast uploads on mobile data | |
| 100 MB | Allows longer training videos | |
| 200 MB | Generous, covers most use cases | ✓ |

**User's choice:** 200 MB

### Progress Indicator

| Option | Description | Selected |
|--------|-------------|----------|
| Inline progress bar | Shows percentage during upload | ✓ |
| Simple spinner | No percentage, just "Uploading..." | |

**User's choice:** Inline progress bar

---

## Inline Player Behavior

### Playback Mode

| Option | Description | Selected |
|--------|-------------|----------|
| Inline with full-screen option | Plays within section, optional full-screen button | |
| Always full-screen | Tapping play goes full-screen immediately | ✓ |

**User's choice:** Always full-screen

### Play Trigger

| Option | Description | Selected |
|--------|-------------|----------|
| Thumbnail + play button | Two taps: expand row → tap play | ✓ |
| Tap to play immediately | One tap starts fullscreen playback | |

**User's choice:** Thumbnail + play button (two-tap flow)

---

## Watch Enforcement Strictness

### Completion Threshold

| Option | Description | Selected |
|--------|-------------|----------|
| 100% — must reach ended event | No shortcuts | |
| 95%+ threshold | Accounts for browser imprecision | ✓ |

**User's choice:** 95%+ playback required

### Seeking Policy

| Option | Description | Selected |
|--------|-------------|----------|
| No seeking allowed | Must watch linearly | |
| Allow seeking to already-watched positions | Can rewind, can't skip ahead | ✓ |
| Allow free seeking | Trust crew, still need 95% time | |

**User's choice:** Allow seeking but only to already-watched positions

### Progress Persistence

| Option | Description | Selected |
|--------|-------------|----------|
| Save progress — resume from where left off | Good for longer videos, mobile interruptions | ✓ |
| Reset — must start over | Simpler, ensures full watch | |

**User's choice:** Save progress — resume from where they left off

---

## Thumbnail Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-extract from video | FFmpeg grabs frame at ~2s, zero admin effort | |
| Admin uploads custom thumbnail | Separate image upload, full control | |
| Auto-extract with optional override | Auto-generate default, admin can replace | ✓ |

**User's choice:** Auto-extract with optional override

---

## Claude's Discretion

- FFmpeg invocation strategy
- Thumbnail extraction timing
- Video player CSS/styling
- Error handling for failed conversions
- maxWatchedTime storage approach

## Deferred Ideas

None
