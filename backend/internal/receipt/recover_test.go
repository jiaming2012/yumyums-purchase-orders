package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/photos"
)

// Recovery tests (B-172). The finder/rewriter predicate is structural — any
// URL off the configured {endpoint}/{bucket}/ prefix is dead — so fixtures
// cover DO Spaces URLs, Mercury-fallback URLs, on-prefix URLs, and empties.

const testStoragePrefix = "https://s3.us-test-000.backblazeb2.com/hq-test/"

const (
	deadDOURL      = "https://nyc3.digitaloceanspaces.com/hq.yumyums/receipts/tx/0.jpg"
	deadMercuryURL = "https://mercury-attachments.example.com/signed/receipt.pdf?expires=1"
)

func liveURL(txID string, i int, ext string) string {
	return fmt.Sprintf("%sreceipts/%s/%d%s", testStoragePrefix, txID, i, ext)
}

func recoverTestCfg() WorkerConfig {
	presigner, _ := photos.NewSpacesPresigner(photos.SpacesConfig{
		AccessKey: "test-key", SecretKey: "test-secret",
		Endpoint: "https://s3.us-test-000.backblazeb2.com", Region: "us-test-000", Bucket: "hq-test",
	})
	return WorkerConfig{
		MercuryAPIKey:   "test-mercury-key",
		Pool:            testPool,
		SpacesPresigner: presigner,
		SpacesEndpoint:  "https://s3.us-test-000.backblazeb2.com",
		SpacesBucket:    "hq-test",
	}
}

func seedRecoveryVendor(t *testing.T) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO vendors (name) VALUES ('Recover Test Vendor') RETURNING id::text`,
	).Scan(&id); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	return id
}

func seedEventWithURLs(t *testing.T, vendorID, txID, eventDate, receiptURL string, receiptURLs []string) {
	t.Helper()
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, total, receipt_url, receipt_urls)
		 VALUES ($1, $2, $3, 10.00, NULLIF($4, ''), $5)`,
		vendorID, txID, eventDate, receiptURL, receiptURLsJSON(receiptURLs),
	); err != nil {
		t.Fatalf("seed purchase_event %s: %v", txID, err)
	}
}

func seedPendingWithURLs(t *testing.T, txID, eventDate, receiptURL string, receiptURLs []string, discarded bool) {
	t.Helper()
	discardedAt := "NULL"
	if discarded {
		discardedAt = "now()"
	}
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, receipt_url, receipt_urls, discarded_at)
		 VALUES ($1, -10.00, 'V', '[]'::jsonb, NULLIF($2, '')::date, NULLIF($3, ''), $4, `+discardedAt+`)`,
		txID, eventDate, receiptURL, receiptURLsJSON(receiptURLs),
	); err != nil {
		t.Fatalf("seed pending_purchase %s: %v", txID, err)
	}
}

func eventURLs(t *testing.T, txID string) (string, []string) {
	t.Helper()
	return rowURLs(t, `SELECT COALESCE(receipt_url, ''), COALESCE(receipt_urls::text, '') FROM purchase_events WHERE bank_tx_id = $1`, txID)
}

func pendingURLs(t *testing.T, txID string) (string, []string) {
	t.Helper()
	return rowURLs(t, `SELECT COALESCE(receipt_url, ''), COALESCE(receipt_urls::text, '') FROM pending_purchases WHERE bank_tx_id = $1`, txID)
}

func rowURLs(t *testing.T, query, txID string) (string, []string) {
	t.Helper()
	var single, listJSON string
	if err := testPool.QueryRow(t.Context(), query, txID).Scan(&single, &listJSON); err != nil {
		t.Fatalf("read URLs for %s: %v", txID, err)
	}
	var list []string
	if listJSON != "" {
		if err := json.Unmarshal([]byte(listJSON), &list); err != nil {
			t.Fatalf("parse receipt_urls for %s: %v", txID, err)
		}
	}
	return single, list
}

func stubRecoverySeams(t *testing.T, txs []MercuryTransaction, downloadErr map[string]error) {
	t.Helper()
	origFetch := fetchTransactions
	fetchTransactions = func(_ context.Context, _ string, _, _ time.Time) ([]MercuryTransaction, error) {
		return txs, nil
	}
	t.Cleanup(func() { fetchTransactions = origFetch })

	origDL := downloadReceiptFileFn
	downloadReceiptFileFn = func(_ context.Context, url string) ([]byte, string, error) {
		if err := downloadErr[url]; err != nil {
			return nil, "", err
		}
		return []byte("receipt-bytes"), "image/jpeg", nil
	}
	t.Cleanup(func() { downloadReceiptFileFn = origDL })

	origUpload := uploadReceiptSlotFn
	uploadReceiptSlotFn = func(_ context.Context, _ WorkerConfig, txID string, i int, _ FileBlob, fileName string) (string, error) {
		ext := ".jpg"
		if fileName == "b.pdf" {
			ext = ".pdf"
		}
		return liveURL(txID, i, ext), nil
	}
	t.Cleanup(func() { uploadReceiptSlotFn = origUpload })
}

func TestFindDeadReceiptRows_MatchesOnlyOffPrefixURLs(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	vendorID := seedRecoveryVendor(t)

	// Dead: DO URL in receipt_url only.
	seedEventWithURLs(t, vendorID, "tx-do-single", "2026-04-10", deadDOURL, nil)
	// Dead: Mercury-fallback URL (structural predicate catches non-DO hosts too).
	seedEventWithURLs(t, vendorID, "tx-mercury", "2026-05-01", deadMercuryURL, nil)
	// Live: on-prefix URL must NOT match.
	seedEventWithURLs(t, vendorID, "tx-live", "2026-06-01", liveURL("tx-live", 0, ".jpg"), []string{liveURL("tx-live", 0, ".jpg")})
	// Dead: DO URL hiding inside receipt_urls JSONB only.
	seedPendingWithURLs(t, "tx-do-jsonb", "2026-03-15", "", []string{deadDOURL}, false)
	// Excluded: discarded pending row with a dead URL.
	seedPendingWithURLs(t, "tx-discarded", "2026-03-01", deadDOURL, nil, true)
	// Excluded: empty receipt_url (no-attachment rows).
	seedPendingWithURLs(t, "tx-empty", "2026-03-20", "", nil, false)

	dead, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("FindDeadReceiptRows: %v", err)
	}
	if dead.EventRows != 2 {
		t.Errorf("EventRows = %d, want 2", dead.EventRows)
	}
	if dead.PendingRows != 1 {
		t.Errorf("PendingRows = %d, want 1", dead.PendingRows)
	}
	wantIDs := map[string]bool{"tx-do-single": true, "tx-mercury": true, "tx-do-jsonb": true}
	if len(dead.TxIDs) != len(wantIDs) {
		t.Errorf("TxIDs = %v, want keys of %v", dead.TxIDs, wantIDs)
	}
	for _, id := range dead.TxIDs {
		if !wantIDs[id] {
			t.Errorf("unexpected tx id %q in %v", id, dead.TxIDs)
		}
	}
	// Window: earliest affected event_date (2026-03-15) minus 30d buffer.
	wantSince := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
	if !dead.Since.Equal(wantSince) {
		t.Errorf("Since = %s, want %s", dead.Since, wantSince)
	}

	// limit caps tx ids but row counts stay complete.
	capped, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 1)
	if err != nil {
		t.Fatalf("FindDeadReceiptRows limit=1: %v", err)
	}
	if len(capped.TxIDs) != 1 {
		t.Errorf("limit=1 TxIDs = %v, want exactly 1", capped.TxIDs)
	}
	if capped.EventRows != 2 || capped.PendingRows != 1 {
		t.Errorf("limit=1 row counts = (%d,%d), want full (2,1)", capped.EventRows, capped.PendingRows)
	}
}

func TestRecoverDeadReceiptURLs_RewritesBothTables(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	vendorID := seedRecoveryVendor(t)

	const txID = "tx-recover-1"
	seedEventWithURLs(t, vendorID, txID, "2026-04-10", deadDOURL, []string{deadDOURL})
	seedPendingWithURLs(t, txID, "2026-04-10", deadDOURL, nil, false)

	stubRecoverySeams(t, []MercuryTransaction{{
		ID: txID,
		Attachments: []Attachment{
			{URL: "https://mercury.example/dl/1", FileName: "a.jpg"},
			{URL: "https://mercury.example/dl/2", FileName: "b.pdf"},
		},
	}}, nil)

	dead, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("finder: %v", err)
	}
	res, err := RecoverDeadReceiptURLs(t.Context(), recoverTestCfg(), dead)
	if err != nil {
		t.Fatalf("RecoverDeadReceiptURLs: %v", err)
	}
	if res.Examined != 1 || res.Recovered != 1 || res.MissingAtMercury != 0 || res.Failed != 0 {
		t.Errorf("result = %+v, want 1 examined / 1 recovered", res)
	}

	wantURLs := []string{liveURL(txID, 0, ".jpg"), liveURL(txID, 1, ".pdf")}
	for name, get := range map[string]func(*testing.T, string) (string, []string){
		"purchase_events": eventURLs, "pending_purchases": pendingURLs,
	} {
		single, list := get(t, txID)
		if single != wantURLs[0] {
			t.Errorf("%s receipt_url = %q, want %q", name, single, wantURLs[0])
		}
		if len(list) != 2 || list[0] != wantURLs[0] || list[1] != wantURLs[1] {
			t.Errorf("%s receipt_urls = %v, want %v", name, list, wantURLs)
		}
	}
}

func TestRecoverDeadReceiptURLs_MissingAtMercury_LeavesURLsUntouched(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	vendorID := seedRecoveryVendor(t)

	seedEventWithURLs(t, vendorID, "tx-gone", "2026-04-10", deadDOURL, nil)
	seedEventWithURLs(t, vendorID, "tx-no-atts", "2026-04-11", deadDOURL, nil)

	// tx-gone absent from Mercury entirely; tx-no-atts present but attachment-less.
	stubRecoverySeams(t, []MercuryTransaction{{ID: "tx-no-atts"}}, nil)

	dead, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("finder: %v", err)
	}
	res, err := RecoverDeadReceiptURLs(t.Context(), recoverTestCfg(), dead)
	if err != nil {
		t.Fatalf("RecoverDeadReceiptURLs: %v", err)
	}
	if res.Examined != 2 || res.MissingAtMercury != 2 || res.Recovered != 0 || res.Failed != 0 {
		t.Errorf("result = %+v, want 2 examined / 2 missing", res)
	}
	if len(res.MissingTxIDs) != 2 {
		t.Errorf("MissingTxIDs = %v, want both tx ids", res.MissingTxIDs)
	}
	for _, txID := range []string{"tx-gone", "tx-no-atts"} {
		if single, _ := eventURLs(t, txID); single != deadDOURL {
			t.Errorf("%s receipt_url = %q, want untouched %q", txID, single, deadDOURL)
		}
	}
}

func TestRecoverDeadReceiptURLs_DownloadFailure_AtomicPerTx(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	vendorID := seedRecoveryVendor(t)

	const txID = "tx-dl-fail"
	seedEventWithURLs(t, vendorID, txID, "2026-04-10", deadDOURL, nil)

	stubRecoverySeams(t, []MercuryTransaction{{
		ID: txID,
		Attachments: []Attachment{
			{URL: "https://mercury.example/dl/ok", FileName: "a.jpg"},
			{URL: "https://mercury.example/dl/broken", FileName: "b.pdf"},
		},
	}}, map[string]error{"https://mercury.example/dl/broken": fmt.Errorf("boom")})

	dead, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("finder: %v", err)
	}
	res, err := RecoverDeadReceiptURLs(t.Context(), recoverTestCfg(), dead)
	if err != nil {
		t.Fatalf("RecoverDeadReceiptURLs: %v", err)
	}
	if res.Failed != 1 || res.Recovered != 0 {
		t.Errorf("result = %+v, want 1 failed / 0 recovered", res)
	}
	if len(res.FailedTxIDs) != 1 || res.FailedTxIDs[0] != txID {
		t.Errorf("FailedTxIDs = %v, want [%s]", res.FailedTxIDs, txID)
	}
	if single, list := eventURLs(t, txID); single != deadDOURL || len(list) != 0 {
		t.Errorf("row touched on failure: receipt_url=%q receipt_urls=%v", single, list)
	}
}

func TestRecoverDeadReceiptURLs_SecondRunIsNoOp(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	vendorID := seedRecoveryVendor(t)

	const txID = "tx-idempotent"
	seedEventWithURLs(t, vendorID, txID, "2026-04-10", deadDOURL, nil)

	stubRecoverySeams(t, []MercuryTransaction{{
		ID:          txID,
		Attachments: []Attachment{{URL: "https://mercury.example/dl/1", FileName: "a.jpg"}},
	}}, nil)

	dead, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("finder: %v", err)
	}
	if _, err := RecoverDeadReceiptURLs(t.Context(), recoverTestCfg(), dead); err != nil {
		t.Fatalf("first run: %v", err)
	}

	again, err := FindDeadReceiptRows(t.Context(), testPool, testStoragePrefix, 0)
	if err != nil {
		t.Fatalf("finder after recovery: %v", err)
	}
	if len(again.TxIDs) != 0 || again.EventRows != 0 || again.PendingRows != 0 {
		t.Errorf("second finder run = %+v, want empty", again)
	}
}

// TestUploadReceiptSlot_NoACLHeader is the direct B-172 regression test: the
// PUT against the presigned URL must NOT carry x-amz-acl (unsigned header →
// signature mismatch on any S3; B2 additionally rejects object ACLs outright).
func TestUploadReceiptSlot_NoACLHeader(t *testing.T) {
	var gotACL, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotACL = r.Header.Get("x-amz-acl")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	presigner, err := photos.NewSpacesPresigner(photos.SpacesConfig{
		AccessKey: "k", SecretKey: "s", Endpoint: srv.URL, Region: "us-test-000", Bucket: "hq-test",
	})
	if err != nil {
		t.Fatalf("NewSpacesPresigner: %v", err)
	}
	cfg := WorkerConfig{SpacesPresigner: presigner, SpacesEndpoint: srv.URL, SpacesBucket: "hq-test"}

	url, err := uploadReceiptSlot(t.Context(), cfg, "tx-acl", 0, FileBlob{Bytes: []byte("x"), ContentType: "image/jpeg"}, "a.jpg")
	if err != nil {
		t.Fatalf("uploadReceiptSlot: %v", err)
	}
	if gotACL != "" {
		t.Errorf("PUT carried x-amz-acl=%q, want no ACL header", gotACL)
	}
	if gotContentType != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", gotContentType)
	}
	want := photos.PublicURL(srv.URL, "hq-test", "receipts/tx-acl/0.jpg")
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}
}
