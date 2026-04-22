package photos

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// UploadHandler handles POST /api/v1/photos/upload.
// Accepts multipart form with fields: path_prefix, id, filename, and file.
// Uploads the file to DO Spaces server-side (no CORS issues from the browser).
func UploadHandler(client *s3.Client, bucket, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, `{"error":"photo storage not configured"}`, http.StatusServiceUnavailable)
			return
		}

		const maxSize = 10 << 20 // 10MB
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
		if err := r.ParseMultipartForm(maxSize); err != nil {
			http.Error(w, `{"error":"file too large or invalid form"}`, http.StatusBadRequest)
			return
		}

		pathPrefix := r.FormValue("path_prefix")
		id := r.FormValue("id")
		filename := r.FormValue("filename")
		if pathPrefix == "" || id == "" || filename == "" {
			http.Error(w, `{"error":"path_prefix, id, and filename are required"}`, http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"file field is required"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		key := pathPrefix + "/" + id + "/" + filename

		_, err = client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        file,
			ContentType: aws.String("image/jpeg"),
			ACL:         "public-read",
		})
		if err != nil {
			log.Printf("UploadHandler PutObject %s: %v", key, err)
			http.Error(w, `{"error":"upload failed"}`, http.StatusInternalServerError)
			return
		}

		pubURL := PublicURL(endpoint, bucket, key)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"public_url": pubURL}) //nolint:errcheck
	}
}
