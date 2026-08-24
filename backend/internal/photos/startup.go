package photos

import (
	"fmt"
	"strings"
)

// StorageEnv is the raw five-variable object-storage configuration read from
// the environment at startup. An empty string means the variable is unset.
type StorageEnv struct {
	Key      string // STORAGE_KEY
	Secret   string // STORAGE_SECRET
	Bucket   string // STORAGE_BUCKET
	Endpoint string // STORAGE_ENDPOINT
	Region   string // STORAGE_REGION
}

// StorageDecision is the boot-time verdict for object-storage configuration.
type StorageDecision struct {
	// Configured is true only when all five variables are present, meaning the
	// caller should build the S3 client and presigner.
	Configured bool
	// Fatal is true when the process must refuse to start.
	Fatal bool
	// Reason explains the verdict for the startup log line. Set for the Fatal
	// and soft-unconfigured cases; empty when Configured.
	Reason string
}

// DecideStorageStartup applies the object-storage boot policy (B-172).
//
// The canceled-DO-Spaces incident was silent because a missing store only
// surfaces at upload time (a 503). This centralizes the boot decision:
//
//   - all five variables set → Configured; boot normally.
//   - all five empty         → not configured. A deliberate dev/test state
//     (uploads 503, /api/v1/health reports "unconfigured"). Fatal only when
//     required is true — prod sets STORAGE_REQUIRED=1 so a forgotten .env.prod
//     crashes loudly instead of silently degrading.
//   - partially set          → Fatal regardless of required. A partial config
//     is never intentional and cannot presign.
func DecideStorageStartup(env StorageEnv, required bool) StorageDecision {
	missing := env.missing()
	switch len(missing) {
	case 0:
		return StorageDecision{Configured: true}
	case 5:
		if required {
			return StorageDecision{Fatal: true, Reason: "STORAGE_REQUIRED=1 but no object-storage variables are set"}
		}
		return StorageDecision{Reason: "object-storage variables not set (uploads will 503)"}
	default:
		return StorageDecision{Fatal: true, Reason: fmt.Sprintf("object-storage config is partial; missing: %s", strings.Join(missing, ", "))}
	}
}

// missing returns the names of the unset variables, in a stable order.
func (e StorageEnv) missing() []string {
	var out []string
	for _, v := range []struct {
		name, val string
	}{
		{"STORAGE_KEY", e.Key},
		{"STORAGE_SECRET", e.Secret},
		{"STORAGE_BUCKET", e.Bucket},
		{"STORAGE_ENDPOINT", e.Endpoint},
		{"STORAGE_REGION", e.Region},
	} {
		if strings.TrimSpace(v.val) == "" {
			out = append(out, v.name)
		}
	}
	return out
}
