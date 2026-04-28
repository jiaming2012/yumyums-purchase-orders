# Quick Task 260428-j2r: Photo upload on Inventory Setup - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Task Boundary

Add photo upload/change field to Inventory Setup item edit form. Photos already appear on PO item cards — this extends that to Setup so users can add/change photos when editing items.

</domain>

<decisions>
## Implementation Decisions

### Photo placement
- Thumbnail (48x48) above the Name/Group/Store Location fields in the edit form
- Shows current photo or a placeholder letter (matching PO card style)
- "Change photo" link next to thumbnail to trigger file picker

### Upload timing
- Immediate upload on file select (matches existing shopping photo flow)
- Uses existing `POST /api/v1/photos/upload` endpoint (DO Spaces)
- Thumbnail updates instantly after upload
- Save button stores the photo_url to the item record

### Backend changes needed
- Add `photo_url` field to `UpdateItemHandler` input struct
- Add `photo_url` to the UPDATE query in `UpdateItemHandler`
- Frontend sends photo_url in the existing PUT /api/v1/inventory/items call

### Claude's Discretion
- JPEG conversion quality/size (reuse existing `convertToJpeg` from purchasing.html)
- Path prefix for uploaded photos (use 'items' to distinguish from 'shopping')

</decisions>

<specifics>
## Specific Ideas

- Reuse `convertToJpeg(file, 1600, 0.85, callback)` from purchasing.html
- Reuse `POST /api/v1/photos/upload` with FormData
- Photo shows on item row in collapsed view as well (small thumb, like PO cards)

</specifics>
