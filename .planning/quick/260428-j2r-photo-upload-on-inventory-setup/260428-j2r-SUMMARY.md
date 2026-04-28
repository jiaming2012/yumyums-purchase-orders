---
phase: quick
plan: 260428-j2r
status: complete
---

# Quick Task 260428-j2r: Photo upload on Inventory Setup

## Result

All 2 tasks completed successfully.

## Commits

| # | Hash | Message |
|---|------|---------|
| 1 | a2d41d6 | feat(quick-260428-j2r): add photo_url to ListItems SELECT and UpdateItem handler |
| 2 | cd544eb | feat(quick-260428-j2r): add photo thumbnail and upload UI to inventory Setup edit form |

## Changes

- **backend/internal/inventory/types.go**: Added `PhotoURL *string` field to `PurchaseItem` struct
- **backend/internal/inventory/handler.go**: Added `photo_url` to ListItems SELECT/Scan and UpdateItem input/UPDATE query
- **inventory.html**: Photo thumbnail (48x48) above edit form fields, "Add photo"/"Change photo" link, `convertToJpeg` function, immediate upload via `/api/v1/photos/upload`, 24px thumbnails on collapsed rows, photo_url included in save payload
- **tests/inventory.spec.js**: E2E test verifying photo area and change photo link exist in edit form

## Verification

- Backend compiles cleanly
- New photo UI test passes
- All existing inventory tests unaffected
