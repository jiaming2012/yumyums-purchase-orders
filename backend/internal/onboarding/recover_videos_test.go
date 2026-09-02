package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// ── recover-videos: the B-172 class, videos edition ─────────────────────────
//
// ob_video_parts.url stores an absolute URL minted at upload time, so a
// storage move strands every prior upload (dead DO Spaces host, observed
// 2026-08-25/26: taps on the Pre-heat Procedure video did nothing and the
// training could not complete). The handler's finder classifies every part —
// ok / dead / never_uploaded — and a real run re-fetches dead public URLs and
// re-homes them on the current bucket.

const testStoragePrefix = "https://cur.example/hq-bucket/"

func recoverReq(t *testing.T, ctx context.Context, handler http.HandlerFunc, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/recoverVideos", strings.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func TestRecoverVideosRequiresManager(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	crew := obEnsureUser(t, "rv-crew@yumyums.kitchen", []string{"team_member"})
	h := RecoverVideosHandler(testPool, testStoragePrefix, func(ctx context.Context, key, ct string, body io.Reader) (string, error) {
		t.Fatal("uploader must not be called for a forbidden caller")
		return "", nil
	})
	rec, _ := recoverReq(t, obUserContext(crew, "Crew", "team_member"), h, `{"dry_run":true}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("crew caller = %d; want 403", rec.Code)
	}
}

func TestRecoverVideosDryRunClassifiesAndTouchesNothing(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	mgr := obEnsureUser(t, "rv-mgr@yumyums.kitchen", []string{"manager"})

	deadURL := "https://old-spaces.example/oldbucket/videos/onboarding/t1/p1/preheat.mov"
	okURL := testStoragePrefix + "videos/onboarding/t1/p2/tempcontrol.mp4"
	deadID := seedVideoPart(t, "RV Dry Run", "Pre-heat Procedure", deadURL)
	seedVideoPart(t, "RV Dry Run 2", "Temperature Control", okURL)
	seedVideoPart(t, "RV Dry Run 3", "Never Uploaded", "blob:http://localhost/xyz")

	h := RecoverVideosHandler(testPool, testStoragePrefix, func(ctx context.Context, key, ct string, body io.Reader) (string, error) {
		t.Fatal("uploader must not be called on a dry run")
		return "", nil
	})
	rec, out := recoverReq(t, obUserContext(mgr, "Mgr", "manager"), h, `{"dry_run":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("dry run = %d body=%s; want 200", rec.Code, rec.Body.String())
	}
	if got := out["dead"].(float64); got < 1 {
		t.Errorf("dead count = %v; want >= 1", got)
	}
	if got := out["never_uploaded"].(float64); got < 1 {
		t.Errorf("never_uploaded count = %v; want >= 1", got)
	}
	if got := out["ok"].(float64); got < 1 {
		t.Errorf("ok count = %v; want >= 1", got)
	}

	var url string
	if err := testPool.QueryRow(t.Context(), `SELECT url FROM ob_video_parts WHERE id = $1`, deadID).Scan(&url); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if url != deadURL {
		t.Errorf("dry run mutated the URL: %q", url)
	}
}

func TestRecoverVideosRecoversDeadPart(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	mgr := obEnsureUser(t, "rv-mgr2@yumyums.kitchen", []string{"manager"})

	var fetches int32
	videoBytes := []byte("MOV-BYTES-PREHEAT")
	oldHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fetches, 1)
		if r.URL.Path != "/oldbucket/videos/onboarding/t1/p1/preheat.mov" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "video/quicktime")
		_, _ = w.Write(videoBytes)
	}))
	defer oldHost.Close()

	deadID := seedVideoPart(t, "RV Recover", "Pre-heat Procedure",
		oldHost.URL+"/oldbucket/videos/onboarding/t1/p1/preheat.mov")
	seedVideoPart(t, "RV Recover 2", "Never Uploaded", "blob:http://localhost/abc")

	var gotKey, gotCT string
	var gotBody []byte
	h := RecoverVideosHandler(testPool, testStoragePrefix, func(ctx context.Context, key, ct string, body io.Reader) (string, error) {
		gotKey, gotCT = key, ct
		gotBody, _ = io.ReadAll(body)
		return testStoragePrefix + key, nil
	})
	rec, out := recoverReq(t, obUserContext(mgr, "Mgr", "manager"), h, ``)
	if rec.Code != http.StatusOK {
		t.Fatalf("real run = %d body=%s; want 200", rec.Code, rec.Body.String())
	}

	// The stranded key is reused on the current bucket, bytes intact.
	if gotKey != "videos/onboarding/t1/p1/preheat.mov" {
		t.Errorf("upload key = %q; want the original videos/... key", gotKey)
	}
	if gotCT != "video/quicktime" {
		t.Errorf("content type = %q; want video/quicktime", gotCT)
	}
	if !bytes.Equal(gotBody, videoBytes) {
		t.Errorf("uploaded bytes differ from the fetched object")
	}

	// The DB row is re-pointed at the current bucket.
	var url string
	if err := testPool.QueryRow(t.Context(), `SELECT url FROM ob_video_parts WHERE id = $1`, deadID).Scan(&url); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if want := testStoragePrefix + "videos/onboarding/t1/p1/preheat.mov"; url != want {
		t.Errorf("url = %q; want %q", url, want)
	}

	// A blob: part has nothing behind it — it must be reported, never fetched.
	if n := atomic.LoadInt32(&fetches); n != 1 {
		t.Errorf("old host saw %d fetches; want exactly 1 (never_uploaded parts must not be fetched)", n)
	}
	if recovered, ok := out["recovered"].([]any); !ok || len(recovered) != 1 {
		t.Errorf("recovered = %v; want exactly 1 entry", out["recovered"])
	}
}
