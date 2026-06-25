package receipt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// makeTxPage returns a slice of n MercuryTransactions, each with a unique ID,
// Status="sent", and Kind=mercuryKindDebitCard so they pass the filter inside
// fetchTransactionsPage. The IDs are sequential: "tx-N".
func makeTxPage(startIdx, n int) []MercuryTransaction {
	txs := make([]MercuryTransaction, n)
	for i := range txs {
		txs[i] = MercuryTransaction{
			ID:     fmt.Sprintf("tx-%d", startIdx+i),
			Amount: -float64(startIdx + i + 1),
			Status: mercuryStatusSent,
			Kind:   mercuryKindDebitCard,
		}
	}
	return txs
}

// mercuryPageServer returns an httptest.Server that simulates the Mercury
// list endpoint's offset-based pagination. pages maps offset → []tx to
// return for that page. Calls to unmapped offsets return an empty list.
// The server increments *callCount on every request.
func mercuryPageServer(t *testing.T, pages map[int][]MercuryTransaction, callCount *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*callCount++

		offsetStr := r.URL.Query().Get("offset")
		offset := 0
		if offsetStr != "" {
			if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
				http.Error(w, "bad offset", http.StatusBadRequest)
				return
			}
		}

		txs := pages[offset] // nil/empty if offset not in map → last-page signal
		resp := mercuryListTransactionsResponse{
			Transactions: txs,
			Total:        len(txs),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("mercuryPageServer: encode: %v", err)
		}
	}))
}

// TestFetchTransactions_Paginates verifies that FetchTransactions issues
// multiple HTTP requests — one per page — and accumulates results across
// pages. The server returns:
//
//	offset=0   → 500 txs (full page → continue)
//	offset=500 → 500 txs (full page → continue)
//	offset=1000 → 200 txs (short page → stop)
//
// Expected: 1200 txs total, 3 HTTP calls.
func TestFetchTransactions_Paginates(t *testing.T) {
	callCount := 0
	pages := map[int][]MercuryTransaction{
		0:    makeTxPage(0, 500),
		500:  makeTxPage(500, 500),
		1000: makeTxPage(1000, 200),
	}

	srv := mercuryPageServer(t, pages, &callCount)
	defer srv.Close()

	// Patch the constant URL used by fetchTransactionsPage by overriding the
	// request URL. We accomplish this via a test-local wrapper that replaces
	// the host in the URL with the httptest server. Because fetchTransactionsPage
	// is unexported and builds the URL internally, we exercise it end-to-end
	// through FetchTransactions — but we need to point it at the test server.
	//
	// Strategy: temporarily override the package-level mercuryAPIBaseURL that
	// fetchTransactionsPage uses. Since the function hard-codes the URL as a
	// const, we instead use an httptest transport trick: wrap the default
	// http.DefaultTransport to rewrite the host.
	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	txs, err := FetchTransactions(t.Context(), "test-key", start, end)
	if err != nil {
		t.Fatalf("FetchTransactions: %v", err)
	}

	wantTotal := 1200
	if len(txs) != wantTotal {
		t.Errorf("len(txs) = %d, want %d", len(txs), wantTotal)
	}
	if callCount != 3 {
		t.Errorf("HTTP calls = %d, want 3 (one per page)", callCount)
	}
}

// TestFetchTransactions_StopsOnShortFirstPage verifies that when the first
// page returns fewer than mercuryPageLimit records, the loop terminates after
// exactly 1 HTTP call.
func TestFetchTransactions_StopsOnShortFirstPage(t *testing.T) {
	callCount := 0
	pages := map[int][]MercuryTransaction{
		0: makeTxPage(0, 200),
		// offset=200 would only be reached if the loop continued — it must not.
		200: makeTxPage(200, 999), // sentinel: if reached, test must fail
	}

	srv := mercuryPageServer(t, pages, &callCount)
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	txs, err := FetchTransactions(t.Context(), "test-key", start, end)
	if err != nil {
		t.Fatalf("FetchTransactions: %v", err)
	}

	if len(txs) != 200 {
		t.Errorf("len(txs) = %d, want 200", len(txs))
	}
	if callCount != 1 {
		t.Errorf("HTTP calls = %d, want 1 (short first page must stop loop)", callCount)
	}
}

// TestFetchTransactions_EmptyFirstPage verifies that an empty first-page
// response (e.g. no transactions in the date range) returns an empty slice
// with no error and exactly 1 HTTP call.
func TestFetchTransactions_EmptyFirstPage(t *testing.T) {
	callCount := 0
	pages := map[int][]MercuryTransaction{
		0: {}, // explicit empty page
	}

	srv := mercuryPageServer(t, pages, &callCount)
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	txs, err := FetchTransactions(t.Context(), "test-key", start, end)
	if err != nil {
		t.Fatalf("FetchTransactions: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("len(txs) = %d, want 0", len(txs))
	}
	if callCount != 1 {
		t.Errorf("HTTP calls = %d, want 1", callCount)
	}
}

// TestFetchTransactions_FiltersUnsupportedKinds verifies that the per-page
// filter inside fetchTransactionsPage drops transactions whose Kind or Status
// is not supported. The server returns a mix of supported and unsupported txs;
// FetchTransactions must return only the supported ones.
func TestFetchTransactions_FiltersUnsupportedKinds(t *testing.T) {
	callCount := 0
	pages := map[int][]MercuryTransaction{
		0: {
			{ID: "good-1", Status: mercuryStatusSent, Kind: mercuryKindDebitCard},
			{ID: "bad-pending", Status: "pending", Kind: mercuryKindDebitCard},
			{ID: "bad-kind", Status: mercuryStatusSent, Kind: "wireTransaction"},
			{ID: "good-2", Status: mercuryStatusSent, Kind: mercuryKindCreditCard},
			{ID: "good-3", Status: mercuryStatusSent, Kind: mercuryKindCreditCardCredit},
			{ID: "good-4", Status: mercuryStatusSent, Kind: mercuryKindDebitCardCredit},
		},
	}

	srv := mercuryPageServer(t, pages, &callCount)
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	txs, err := FetchTransactions(t.Context(), "test-key", time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("FetchTransactions: %v", err)
	}

	if len(txs) != 4 {
		t.Errorf("len(txs) = %d, want 4 (only supported+sent txs)", len(txs))
	}
	for _, tx := range txs {
		if tx.Status != mercuryStatusSent {
			t.Errorf("tx %q: Status = %q, want %q", tx.ID, tx.Status, mercuryStatusSent)
		}
		if !isSupportedKind(tx.Kind) {
			t.Errorf("tx %q: Kind = %q is not supported", tx.ID, tx.Kind)
		}
	}
}

// makeMixedTxPage returns a slice of (supported + unsupported) MercuryTransactions.
// The first `supported` entries are debitCard/sent (pass the filter); the
// remaining `unsupported` entries use kind="bookkeepingTransaction" so the
// filter drops them. IDs are globally unique via startIdx.
func makeMixedTxPage(startIdx, supported, unsupported int) []MercuryTransaction {
	txs := make([]MercuryTransaction, 0, supported+unsupported)
	for i := range supported {
		txs = append(txs, MercuryTransaction{
			ID:     fmt.Sprintf("tx-good-%d", startIdx+i),
			Amount: -float64(startIdx + i + 1),
			Status: mercuryStatusSent,
			Kind:   mercuryKindDebitCard,
		})
	}
	for i := range unsupported {
		txs = append(txs, MercuryTransaction{
			ID:     fmt.Sprintf("tx-bad-%d", startIdx+supported+i),
			Amount: -float64(startIdx + supported + i + 1),
			Status: mercuryStatusSent,
			Kind:   "bookkeepingTransaction",
		})
	}
	return txs
}

// TestFetchTransactions_PaginatesWhenFilterShrinksPage verifies that FetchTransactions
// continues paginating based on the RAW page size returned by Mercury — not the
// count of filtered (supported) transactions. Without the fix, a page of 500 raw txs
// where only 50 pass the filter looks like a 50-tx page, which is less than
// mercuryPageLimit (500), so the loop breaks early and never fetches page 2 or 3.
//
// Server behaviour:
//
//	offset=0    → 500 raw (50 supported, 450 unsupported) — full page, must continue
//	offset=500  → 500 raw (50 supported, 450 unsupported) — full page, must continue
//	offset=1000 → 100 raw (10 supported, 90 unsupported)  — short page, must stop
//
// Expected: 110 supported txs total, 3 HTTP calls.
// Failure mode without the fix: 50 txs, 1 HTTP call.
func TestFetchTransactions_PaginatesWhenFilterShrinksPage(t *testing.T) {
	callCount := 0
	pages := map[int][]MercuryTransaction{
		0:    makeMixedTxPage(0, 50, 450),
		500:  makeMixedTxPage(500, 50, 450),
		1000: makeMixedTxPage(1000, 10, 90),
	}

	srv := mercuryPageServer(t, pages, &callCount)
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	txs, err := FetchTransactions(t.Context(), "test-key", start, end)
	if err != nil {
		t.Fatalf("FetchTransactions: %v", err)
	}

	wantTxs := 110 // 50 + 50 + 10 supported
	if len(txs) != wantTxs {
		t.Errorf("len(txs) = %d, want %d (all supported txs across 3 pages)", len(txs), wantTxs)
	}
	if callCount != 3 {
		t.Errorf("HTTP calls = %d, want 3 (pagination must use raw page size, not filtered count)", callCount)
	}
	// Sanity: every returned tx must pass the filter.
	for _, tx := range txs {
		if tx.Status != mercuryStatusSent {
			t.Errorf("tx %q: Status = %q, want %q", tx.ID, tx.Status, mercuryStatusSent)
		}
		if !isSupportedKind(tx.Kind) {
			t.Errorf("tx %q: Kind = %q is not a supported kind", tx.ID, tx.Kind)
		}
	}
}

// rewriteHostTransport returns an http.RoundTripper that rewrites the host of
// every outbound request to the given base URL. This lets tests point
// FetchTransactions (which hard-codes the Mercury hostname) at an httptest.Server
// without changing production code.
type rewriteHostRoundTripper struct {
	baseURL string
	base    http.RoundTripper
}

func rewriteHostTransport(baseURL string) http.RoundTripper {
	return &rewriteHostRoundTripper{baseURL: baseURL, base: http.DefaultTransport}
}

func (r *rewriteHostRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request so we don't mutate the caller's copy.
	clone := req.Clone(req.Context())
	// Parse the test server URL and graft its scheme+host onto the request.
	clone.URL.Scheme = "http"
	// Extract host from baseURL ("http://127.0.0.1:PORT").
	host := r.baseURL
	if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}
	clone.URL.Host = host
	return r.base.RoundTrip(clone)
}
