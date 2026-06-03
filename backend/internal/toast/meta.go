package toast

import (
	"encoding/json"
	"fmt"
	"time"
)

// Source values for the sidecar metadata file (D-03).
const (
	MetaSourceSFTP      = "sftp"      // written by live sync ticks
	MetaSourceMigration = "migration" // written by cmd/migrate-toast-archive
)

// MetaSidecar is the JSON payload written alongside each ItemSelectionDetails.csv
// at toast/YYYYMMDD/ItemSelectionDetails.meta.json (D-03).
//
// Field order = JSON output order. Keep snake_case to match the operational spec.
type MetaSidecar struct {
	OriginalFilename string `json:"original_filename"`
	DownloadedAtISO  string `json:"downloaded_at_iso"`
	Source           string `json:"source"` // "sftp" or "migration"
}

// NewMetaSidecar returns a MetaSidecar with DownloadedAtISO set to time.Now().UTC()
// in RFC3339 format. Callers fill OriginalFilename and Source.
func NewMetaSidecar(originalFilename, source string) MetaSidecar {
	return MetaSidecar{
		OriginalFilename: originalFilename,
		DownloadedAtISO:  time.Now().UTC().Format(time.RFC3339),
		Source:           source,
	}
}

// Bytes returns the JSON byte representation of the sidecar.
// Wraps json.Marshal so error returns include context.
func (m MetaSidecar) Bytes() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal meta sidecar: %w", err)
	}
	return b, nil
}
