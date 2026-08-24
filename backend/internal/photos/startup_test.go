package photos

import (
	"strings"
	"testing"
)

func fullStorageEnv() StorageEnv {
	return StorageEnv{
		Key:      "k",
		Secret:   "s",
		Bucket:   "b",
		Endpoint: "https://s3.us-west-004.backblazeb2.com",
		Region:   "us-west-004",
	}
}

func TestDecideStorageStartup(t *testing.T) {
	tests := []struct {
		name           string
		env            StorageEnv
		required       bool
		wantConfigured bool
		wantFatal      bool
	}{
		{"all set, not required", fullStorageEnv(), false, true, false},
		{"all set, required (prod happy path)", fullStorageEnv(), true, true, false},
		{"all empty, not required (deliberate dev/test)", StorageEnv{}, false, false, false},
		// B-172 regression: prod sets STORAGE_REQUIRED=1. With no storage config
		// the process MUST refuse to boot — a forgotten .env.prod cannot be
		// allowed to degrade silently to upload-time 503s.
		{"all empty, required (prod gap — must be fatal)", StorageEnv{}, true, false, true},
		// A partial config is never intentional and cannot presign, so it is
		// fatal in every environment regardless of the required flag.
		{"partial, not required", StorageEnv{Key: "k"}, false, false, true},
		{"partial missing region, required", func() StorageEnv { e := fullStorageEnv(); e.Region = ""; return e }(), true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideStorageStartup(tt.env, tt.required)
			if got.Configured != tt.wantConfigured {
				t.Errorf("Configured = %v, want %v (reason=%q)", got.Configured, tt.wantConfigured, got.Reason)
			}
			if got.Fatal != tt.wantFatal {
				t.Errorf("Fatal = %v, want %v (reason=%q)", got.Fatal, tt.wantFatal, got.Reason)
			}
			if got.Fatal && got.Reason == "" {
				t.Error("a fatal decision must carry a non-empty Reason for the startup log")
			}
			if got.Configured && got.Fatal {
				t.Error("Configured and Fatal are mutually exclusive")
			}
		})
	}
}

// A partial config's reason must name the missing variables so the operator
// can act on the crash without guessing.
func TestDecideStorageStartupPartialNamesMissing(t *testing.T) {
	env := fullStorageEnv()
	env.Secret = ""
	env.Region = ""
	got := DecideStorageStartup(env, false)
	if !got.Fatal {
		t.Fatalf("partial config must be fatal, got %+v", got)
	}
	for _, want := range []string{"STORAGE_SECRET", "STORAGE_REGION"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("reason %q should name missing var %q", got.Reason, want)
		}
	}
}
