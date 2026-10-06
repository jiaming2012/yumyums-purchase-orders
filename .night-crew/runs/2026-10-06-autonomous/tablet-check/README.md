# Tablet check — does the truck's tablet read the codes the lab could not?

**Why:** decision 210 (ledger T-70, morning triage 2026-10-06). In the lab, on a browser with no
built-in barcode detector, "Scan from photo" missed about one customer code in seven. Whether the
truck's tablet behaves the same decides how much of a scanner fix is needed before customer codes
go out.

**What to do (about 10 minutes):**

1. Get the 16 images in this folder onto the truck's tablet's photo library, unchanged (AirDrop,
   a shared album at full quality, or a file share — not a screenshot of them, and not a messaging
   app that recompresses).
2. Open HQ → Marketing → the scanner, and use **Scan from photo** on each image in turn.
3. For each, write down what the screen showed: an **offer** (it read), **"No QR code found"**,
   or **"Not a Yumyums code"**. These are test codes, so an offer shows as "Unverified" and some
   as expired — that still counts as read.
4. Optional but worth it: show two or three of the "No QR code found" ones on another phone and
   point the tablet's **live camera** at them. Nobody has measured the live camera.

**The key — what the lab saw for each image:**

| Image | Lab result (fallback reader) | Tablet showed |
|---|---|---|
| code-01.png | lab: No QR code found | |
| code-02.png | lab: No QR code found | |
| code-03.png | lab: READ (control) | |
| code-04.png | lab: No QR code found | |
| code-05.png | lab: READ (control) | |
| code-06.png | lab: No QR code found | |
| code-07.png | lab: never read, even turned | |
| code-08.png | lab: No QR code found | |
| code-09.png | lab: Not a Yumyums code (misread as a barcode) | |
| code-10.png | lab: READ (control) | |
| code-11.png | lab: No QR code found | |
| code-12.png | lab: READ (control) | |
| code-13.png | lab: No QR code found | |
| code-14.png | lab: never read, even turned | |
| code-15.png | lab: No QR code found | |
| code-16.png | lab: Not a Yumyums code (misread as a barcode) | |

**How to read the result:**

- Tablet reads the 12 the lab could not (all or nearly all) → the tablet has its own detector; the
  problem is confined to other devices and the scanner fix is a safety net, not a blocker.
- Tablet fails the same 12 → the lab figure holds on the truck; fix the photo scan before any
  customer is sent a code.
- Tablet fails any of the 4 controls → something else is wrong; stop and report that first.

Images are fresh test codes drawn by the triage reviewer with the parked card's own code
(`fe45768`); they belong to no real customer.
