package marketing

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	qrcode "github.com/skip2/go-qrcode"
)

// PNGSizeLadder is the set of pixel sizes GET /codes/{id}.png will render.
//
// This card's call, and a LADDER rather than a range on purpose: these PNGs go
// to a printer or into a share sheet, and an off-ladder request is answered
// with a 400 that names the options rather than silently snapped. A print job
// that asked for 900px and got 1024 is a surprise at the printer.
//
//	256  — the in-app code sheet thumbnail
//	512  — a share-sheet image and a receipt footer
//	1024 — the default: the big code sheet, and a flyer at ~3.5in / 300dpi
//	2048 — a truck sign or a table tent at print resolution
var PNGSizeLadder = []int{256, 512, 1024, 2048}

// DefaultPNGSize is what a request with no ?size= gets.
const DefaultPNGSize = 1024

// CodePNGHandler is §5 row 7: image/png of the short URL at Medium error
// correction with a 4-module quiet zone, cached privately for an hour.
//
// Medium and the quiet zone are both from the signed build-fact (spike 01, exit
// 0): the 35-character payload encodes at QR version 3 — 29 modules a side — at
// Medium, and version 3 is the CEILING for a sign-scannable code. Raising the
// correction level or lengthening the payload tips it to version 4 (33 modules).
// go-qrcode's 4-module border is the default, so DisableBorder is left alone.
//
// Cache-Control is PRIVATE: a code can be re-pointed without being reprinted,
// so a shared cache must never serve one campaign's PNG to another's manager.
// An hour is short enough that a re-point shows up on the next sheet open.
func CodePNGHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		id := chi.URLParam(r, "id")

		size := DefaultPNGSize
		if raw := r.URL.Query().Get("size"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || !containsInt(PNGSizeLadder, n) {
				writeErrorWith(w, http.StatusBadRequest, "bad_size",
					map[string]any{"allowed": PNGSizeLadder})
				return
			}
			size = n
		}

		var short string
		err := d.Pool.QueryRow(r.Context(),
			`SELECT short FROM qr_codes WHERE id = $1`, id).Scan(&short)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "code_not_found")
			return
		}
		if err != nil {
			slog.Error("marketing: load code for png", "error", err, "code_id", id)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		png, err := qrcode.Encode(d.QRBaseURL+"/q/"+short, qrcode.Medium, size)
		if err != nil {
			slog.Error("marketing: encode qr png", "error", err, "code_id", id, "size", size)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		w.Header().Set("Content-Length", strconv.Itoa(len(png)))
		// Names the file in a Save-PNG / share flow instead of handing the
		// phone "png".
		w.Header().Set("Content-Disposition", `inline; filename="`+short+`.png"`)
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(png)
		}
	}
}

func containsInt(haystack []int, needle int) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
