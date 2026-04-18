package photos

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// presignUploadRequest is the JSON body for POST /photos/presign.
type presignUploadRequest struct {
	PathPrefix string `json:"path_prefix"` // "checklists" or "receipts"
	ID         string `json:"id"`          // submission_id or event_id
	Filename   string `json:"filename"`    // e.g. "field_abc123.jpg"
}

// presignUploadResponse is the JSON response for POST /photos/presign.
type presignUploadResponse struct {
	URL       string `json:"url"`        // short-lived PUT URL
	ObjectKey string `json:"object_key"` // full object key in bucket
	PublicURL string `json:"public_url"` // permanent public URL
}

// presignGetResponse is the JSON response for GET /photos/presign?key=...
type presignGetResponse struct {
	URL string `json:"url"` // short-lived GET URL
}

// PresignUploadHandler handles POST /api/v1/photos/presign.
// It accepts a path_prefix, id, and filename and returns a presigned PUT URL
// along with the object key and permanent public URL.
// If presigner is nil (env vars not configured), returns 503.
func PresignUploadHandler(presigner *s3.PresignClient, bucket, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if presigner == nil {
			http.Error(w, `{"error":"photo storage not configured"}`, http.StatusServiceUnavailable)
			return
		}

		var req presignUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
			return
		}
		if req.PathPrefix == "" || req.ID == "" || req.Filename == "" {
			http.Error(w, `{"error":"path_prefix, id, and filename are required"}`, http.StatusBadRequest)
			return
		}

		key := req.PathPrefix + "/" + req.ID + "/" + req.Filename

		putURL, err := GeneratePresignedPutURL(r.Context(), presigner, bucket, key, "image/jpeg", 15*time.Minute)
		if err != nil {
			http.Error(w, `{"error":"failed to generate presigned URL"}`, http.StatusInternalServerError)
			return
		}

		pubURL := PublicURL(endpoint, bucket, key)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(presignUploadResponse{ //nolint:errcheck
			URL:       putURL,
			ObjectKey: key,
			PublicURL: pubURL,
		})
	}
}

// PresignGetHandler handles GET /api/v1/photos/presign?key=...
// It returns a short-lived GET URL for a private object in DO Spaces.
// If presigner is nil (env vars not configured), returns 503.
func PresignGetHandler(presigner *s3.PresignClient, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if presigner == nil {
			http.Error(w, `{"error":"photo storage not configured"}`, http.StatusServiceUnavailable)
			return
		}

		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, `{"error":"key query param is required"}`, http.StatusBadRequest)
			return
		}

		getURL, err := GeneratePresignedGetURL(r.Context(), presigner, bucket, key, time.Hour)
		if err != nil {
			http.Error(w, `{"error":"failed to generate presigned URL"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(presignGetResponse{URL: getURL}) //nolint:errcheck
	}
}
