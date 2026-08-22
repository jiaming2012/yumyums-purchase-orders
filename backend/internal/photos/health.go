package photos

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage status values surfaced as the "storage" field of /api/v1/health.
// B-172: a canceled storage account used to be invisible — the client is built
// offline from static creds, so only a real probe can tell a dead account from
// a live one.
const (
	StorageOK           = "ok"
	StorageUnreachable  = "unreachable"
	StorageUnconfigured = "unconfigured"
)

// probeTimeout bounds a single HeadBucket probe. Kept under the 2s --max-time
// that `task version` curls /api/v1/health with: a down endpoint fails in
// milliseconds (refused/DNS), and a hung one must not drag health past the
// caller's own timeout.
const probeTimeout = 1500 * time.Millisecond

// StorageHealth answers "is object storage actually reachable?" with a cached
// HeadBucket probe. Status transitions are logged loudly, so a dead account is
// explicit in the server log — not discovered one failed upload at a time.
type StorageHealth struct {
	client *s3.Client
	bucket string
	ttl    time.Duration

	mu         sync.Mutex
	lastStatus string
	lastCheck  time.Time
}

// NewStorageHealth wraps client for probing. A nil client means storage is not
// configured; the zero-value ttl re-probes on every call.
func NewStorageHealth(client *s3.Client, bucket string, ttl time.Duration) *StorageHealth {
	return &StorageHealth{client: client, bucket: bucket, ttl: ttl}
}

// Status returns the current storage status, re-probing at most once per TTL.
// Concurrent callers serialize on the mutex: the first probes, the rest get
// the freshly cached verdict.
func (h *StorageHealth) Status(ctx context.Context) string {
	if h == nil || h.client == nil {
		return StorageUnconfigured
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastStatus != "" && time.Since(h.lastCheck) < h.ttl {
		return h.lastStatus
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	_, err := h.client.HeadBucket(probeCtx, &s3.HeadBucketInput{Bucket: aws.String(h.bucket)})
	status := StorageOK
	if err != nil {
		status = StorageUnreachable
	}

	if status != h.lastStatus {
		switch status {
		case StorageUnreachable:
			slog.Error("object storage unreachable — photo/video uploads and stored media will fail",
				"bucket", h.bucket, "error", err)
		case StorageOK:
			if h.lastStatus == StorageUnreachable {
				slog.Info("object storage reachable again", "bucket", h.bucket)
			}
		}
	}
	h.lastStatus = status
	h.lastCheck = time.Now()
	return status
}
