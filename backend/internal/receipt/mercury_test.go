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

// mercuryCursorServer returns an httptest.Server that simulates the Mercury
// list endpoint's REAL pagination contract: cursor-based via `start_after`
// (the exclusive ID of the last transaction of the previous page). The
// server holds the full ordered transaction list and serves the next
// `limit`-sized slice after the cursor. It records each request's
// start_after value into *cursorsSeen and increments *callCount. A request
// carrying an `offset` parameter fails the test — the live endpoint ignores
// that parameter, so sending it means the pagination regression is back.
func mercuryCursorServer(t *testing.T, all []MercuryTransaction, callCount *int, cursorsSeen *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*callCount++

		if r.URL.Query().Has("offset") {
			t.Errorf("request sent `offset` — Mercury ignores it; pagination must use start_after")
		}

		limit := mercuryPageLimit
		if ls := r.URL.Query().Get("limit"); ls != "" {
			if _, err := fmt.Sscanf(ls, "%d", &limit); err != nil {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
		}

		startAfter := r.URL.Query().Get("start_after")
		if cursorsSeen != nil {
			*cursorsSeen = append(*cursorsSeen, startAfter)
		}
		from := 0
		if startAfter != "" {
			from = len(all) // unknown cursor → empty page
			for i, tx := range all {
				if tx.ID == startAfter {
					from = i + 1
					break
				}
			}
		}
		to := min(from+limit, len(all))

		resp := mercuryListTransactionsResponse{
			Transactions: all[from:to],
			Total:        to - from,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("mercuryCursorServer: encode: %v", err)
		}
	}))
}

// TestFetchTransactions_Paginates verifies that FetchTransactions issues
// multiple HTTP requests — one per page — cursoring with start_after and
// accumulating results. The server holds 1200 txs:
//
//	page 1 (start_after absent)  → tx-0..tx-499  (full page → continue)
//	page 2 (start_after=tx-499)  → tx-500..tx-999 (full page → continue)
//	page 3 (start_after=tx-999)  → tx-1000..tx-1199 (short page → stop)
//
// Expected: 1200 txs total, 3 HTTP calls, cursors ["", "tx-499", "tx-999"].
func TestFetchTransactions_Paginates(t *testing.T) {
	callCount := 0
	var cursors []string
	all := makeTxPage(0, 1200)

	srv := mercuryCursorServer(t, all, &callCount, &cursors)
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
	wantCursors := []string{"", "tx-499", "tx-999"}
	if fmt.Sprint(cursors) != fmt.Sprint(wantCursors) {
		t.Errorf("start_after cursors = %v, want %v", cursors, wantCursors)
	}
}

// TestFetchTransactions_StopsOnShortFirstPage verifies that when the first
// page returns fewer than mercuryPageLimit records, the loop terminates after
// exactly 1 HTTP call.
func TestFetchTransactions_StopsOnShortFirstPage(t *testing.T) {
	callCount := 0
	srv := mercuryCursorServer(t, makeTxPage(0, 200), &callCount, nil)
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
	srv := mercuryCursorServer(t, nil, &callCount, nil)
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
	all := []MercuryTransaction{
		{ID: "good-1", Status: mercuryStatusSent, Kind: mercuryKindDebitCard},
		{ID: "bad-pending", Status: "pending", Kind: mercuryKindDebitCard},
		{ID: "bad-kind", Status: mercuryStatusSent, Kind: "wireTransaction"},
		{ID: "good-2", Status: mercuryStatusSent, Kind: mercuryKindCreditCard},
		{ID: "good-3", Status: mercuryStatusSent, Kind: mercuryKindCreditCardCredit},
		{ID: "good-4", Status: mercuryStatusSent, Kind: mercuryKindDebitCardCredit},
	}

	srv := mercuryCursorServer(t, all, &callCount, nil)
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
//	page 1 → 500 raw (50 supported, 450 unsupported) — full page, must continue
//	page 2 → 500 raw (50 supported, 450 unsupported) — full page, must continue
//	page 3 → 100 raw (10 supported, 90 unsupported)  — short page, must stop
//
// Expected: 110 supported txs total, 3 HTTP calls.
// Failure mode without the fix: 50 txs, 1 HTTP call.
func TestFetchTransactions_PaginatesWhenFilterShrinksPage(t *testing.T) {
	callCount := 0
	var all []MercuryTransaction
	all = append(all, makeMixedTxPage(0, 50, 450)...)
	all = append(all, makeMixedTxPage(500, 50, 450)...)
	all = append(all, makeMixedTxPage(1000, 10, 90)...)

	srv := mercuryCursorServer(t, all, &callCount, nil)
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

// TestFetchTransactions_BailsWhenCursorIgnored reproduces the 2026-08-06
// B-145 backfill failure shape: a server that returns the SAME full page for
// every request regardless of pagination parameters (exactly what the live
// endpoint does when paginated by `offset`, which it silently ignores).
// FetchTransactions must fail fast with a cursor-did-not-advance error —
// not loop to a ceiling, and not return duplicated transactions.
func TestFetchTransactions_BailsWhenCursorIgnored(t *testing.T) {
	callCount := 0
	page := makeTxPage(0, 500)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := mercuryListTransactionsResponse{Transactions: page, Total: len(page)}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	_, err := FetchTransactions(t.Context(), "test-key", time.Now().AddDate(0, -1, 0), time.Now())
	if err == nil {
		t.Fatal("FetchTransactions: want cursor-did-not-advance error, got nil")
	}
	if callCount > 2 {
		t.Errorf("HTTP calls = %d, want ≤2 (must bail on the first repeated page)", callCount)
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
