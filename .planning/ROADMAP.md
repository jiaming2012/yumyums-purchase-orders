# Roadmap: Yumyums HQ

## Milestones

- ✅ **v1.0 Operations Console MVP** — Phases 1-5 (shipped 2026-04-14) — [Archive](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1 Inventory App** — Phases 6-8 (shipped 2026-04-14) — [Archive](milestones/v1.1-ROADMAP.md)
- ✅ **v2.0 Backend** — Phases 9-13 (shipped 2026-04-19) — [Archive](milestones/v2.0-ROADMAP.md)
- 🚧 **v2.1 Onboarding Video Upgrade** — Phase 1 (in progress)

## Phases

<details>
<summary>✅ v1.0 Operations Console MVP (Phases 1-5) — SHIPPED 2026-04-14</summary>

- [x] Phase 1: Template Builder (3/3 plans) — completed 2026-04-13
- [x] Phase 2: Fill-Out and Conditional Logic (2/2 plans) — completed 2026-04-13
- [x] Phase 3: Photo, Approval, and Integration (2/2 plans) — completed 2026-04-13
- [x] Phase 4: Onboarding App (3/3 plans) — completed 2026-04-14
- [x] Phase 5: Onboarding Builder (2/2 plans) — completed 2026-04-14

</details>

<details>
<summary>✅ v1.1 Inventory App (Phases 6-8) — SHIPPED 2026-04-14</summary>

- [x] Phase 6: Foundation and History Tab (2/2 plans) — completed 2026-04-14
- [x] Phase 7: Stock and Reorder Tab (2/2 plans) — completed 2026-04-14
- [x] Phase 8: Trends and Cost Intelligence Tabs (2/2 plans) — completed 2026-04-14

</details>

<details>
<summary>✅ v2.0 Backend (Phases 9-13) — SHIPPED 2026-04-19</summary>

- [x] Phase 9: Foundation + Auth (4/4 plans) — completed 2026-04-15
- [x] Phase 10: Workflows API (5/5 plans) — completed 2026-04-15
- [x] Phase 10.1: Cross-Device State Sync (5/5 plans) — completed 2026-04-17
- [x] Phase 10.2: Reactive Sync Framework (3/3 plans) — completed 2026-04-17
- [x] Phase 11: Onboarding + Users Admin (6/6 plans) — completed 2026-04-18
- [x] Phase 12: Inventory + Photos + Tile Permissions (6/6 plans) — completed 2026-04-18
- [x] Phase 13: Integration Fixes (2/2 plans) — completed 2026-04-19

</details>

_Full phase details archived to [milestones/v2.0-ROADMAP.md](milestones/v2.0-ROADMAP.md)_

### v2.1 Onboarding Video Upgrade (In Progress)

**Milestone Goal:** Replace placeholder video links with a real video experience — upload to DO Spaces, inline playback, watch enforcement, and thumbnails.

- [ ] **Phase 1: Onboarding Video Upgrade** — DO Spaces video upload, inline player, watch-to-complete enforcement, thumbnails

## Phase Details

### Phase 1: Onboarding Video Upgrade
**Goal**: Training videos can be uploaded to DO Spaces or linked by URL, play inline with poster thumbnails, and the watched checkbox only checks after the full video is played
**Depends on**: v2.0 (existing presign infrastructure)
**Requirements**: SC-01 (Builder upload/URL), SC-02 (DO Spaces storage), SC-03 (thumbnails), SC-04 (inline player), SC-05 (watch enforcement)
**Success Criteria** (what must be TRUE):
  1. Builder: admin can enter a video URL or upload a video file from their device for each video part
  2. Uploaded videos are stored in DO Spaces with an organized path (e.g. `videos/onboarding/{template_id}/{part_id}.mp4`)
  3. Video thumbnails display as poster images in the training runner
  4. Videos play in an inline `<video>` player (not an external link)
  5. A video part checkbox only checks after the video `ended` event fires — no manual override
**Plans**: 3 plans

Plans:
- [ ] 01-01-PLAN.md — Backend: migration, video presign, FFmpeg processing, watch progress API
- [ ] 01-02-PLAN.md — Builder UI: upload/URL toggle, XHR progress bar, thumbnail override
- [ ] 01-03-PLAN.md — Runner UI: inline video player, seeking restriction, watch enforcement
