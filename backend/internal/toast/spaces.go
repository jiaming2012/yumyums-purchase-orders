package toast

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Key + path builders (Phase 22.1 D-02, D-03, D-12).
// dateDir is a YYYYMMDD string produced by time.Time.Format("20060102").

// SpacesCSVKey returns the canonical Spaces key for a date's ItemSelectionDetails.csv.
// Format: toast/YYYYMMDD/ItemSelectionDetails.csv
func SpacesCSVKey(dateDir string) string {
	return fmt.Sprintf("toast/%s/ItemSelectionDetails.csv", dateDir)
}

// SpacesMetaKey returns the Spaces key for the sidecar metadata JSON.
// Format: toast/YYYYMMDD/ItemSelectionDetails.meta.json
func SpacesMetaKey(dateDir string) string {
	return fmt.Sprintf("toast/%s/ItemSelectionDetails.meta.json", dateDir)
}

// CacheCSVPath returns the local cache filesystem path for a date's CSV.
// cacheDir is typically "backend/cache/toast" (Config.CacheDir).
func CacheCSVPath(cacheDir, dateDir string) string {
	return filepath.Join(cacheDir, dateDir, "ItemSelectionDetails.csv")
}

// CacheMetaPath returns the local cache filesystem path for the sidecar JSON.
func CacheMetaPath(cacheDir, dateDir string) string {
	return filepath.Join(cacheDir, dateDir, "ItemSelectionDetails.meta.json")
}

// SpacesKeyExists performs a HeadObject and returns true if the key exists.
// Returns (false, nil) on 404 (NotFound). Returns (false, err) on any other error.
// Used by the migration CLI (D-09) for idempotency.
func SpacesKeyExists(ctx context.Context, client *s3.Client, bucket, key string) (bool, error) {
	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
