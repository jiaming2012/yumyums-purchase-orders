package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	mercuryStatusSent           = "sent"
	mercuryKindCreditCard       = "creditCardTransaction"
	mercuryKindDebitCard        = "debitCardTransaction"
	mercuryKindCreditCardCredit = "creditCardCredit"
	mercuryKindDebitCardCredit  = "debitCardCredit"

	// mercuryPageLimit is the maximum page size accepted by Mercury's list
	// endpoint (both the default and the maximum). We always request this many
	// records per page so pagination terminates as quickly as possible.
	mercuryPageLimit = 500
)

// FetchTransactions fetches Mercury transactions for the given date range.
// Returns every "sent" supported transaction regardless of attachment count;
// classifying attached vs. unattached rows is the worker's job (see worker.go)
// so the completeness gate can fail on unreceipted card spend.
//
// Pagination: Mercury caps each response at 500 records and its
// /api/v1/transactions endpoint is CURSOR-paginated — it accepts
// `start_after` (exclusive transaction-ID cursor) and silently IGNORES an
// `offset` parameter. Offset pagination therefore returns the identical
// first page forever once a window holds >500 raw transactions (verified
// against the live API on 2026-08-06, during the B-145 backfill: offsets 0
// and 500 returned the same first ID). Windows under 500 fit in one page,
// which is why the ordinary 14-day lookback never surfaced this. We request
// order=asc and cursor forward from each page's last raw transaction until
// a short page arrives.
func FetchTransactions(ctx context.Context, apiKey string, startDate, endDate time.Time) ([]MercuryTransaction, error) {
	var all []MercuryTransaction
	startAfter := ""
	pages := 0
	for {
		filtered, rawCount, lastID, err := fetchTransactionsPage(ctx, apiKey, startDate, endDate, mercuryPageLimit, startAfter)
		if err != nil {
			return nil, err
		}
		all = append(all, filtered...)
		// Use the RAW page size to decide whether to continue — not the filtered
		// count. A page of 500 raw txs where 460 are filtered would otherwise
		// appear as a 40-tx "short page" and incorrectly stop pagination early.
		if rawCount < mercuryPageLimit {
			break
		}
		if lastID == "" || lastID == startAfter {
			// A full page whose cursor did not advance can only loop forever.
			return nil, fmt.Errorf("Mercury FetchTransactions: pagination cursor did not advance (start_after=%q) — bailing", startAfter)
		}
		startAfter = lastID
		pages++
		if pages > 100 {
			// Defensive ceiling: 100 pages = 50 000 raw transactions, far past
			// any plausible volume for this business.
			return nil, fmt.Errorf("Mercury FetchTransactions: exceeded 100 pages — bailing")
		}
	}
	return all, nil
}

// fetchTransactionsPage fetches a single page of transactions from the Mercury
// list endpoint with the given limit, cursoring forward from startAfter (the
// exclusive transaction-ID cursor; empty for the first page). It applies the
// status/kind filters so callers only see supported, sent transactions.
//
// Returns the filtered slice, the raw count of transactions Mercury returned
// (before filtering), and the ID of the LAST raw transaction on the page —
// the caller's next cursor. The caller must use rawCount — not
// len(filtered) — to decide whether to continue paginating, because a page
// of e.g. 500 raw txs where 460 are unsupported would look like a 40-tx
// "short page" if the caller used the filtered length, terminating the loop
// prematurely. lastID likewise comes from the raw page, not the filtered
// slice, or a fully-filtered page would stall the cursor.
func fetchTransactionsPage(ctx context.Context, apiKey string, startDate, endDate time.Time, limit int, startAfter string) (filtered []MercuryTransaction, rawCount int, lastID string, err error) {
	const mercuryURL = "https://api.mercury.com/api/v1/transactions"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mercuryURL, nil)
	if err != nil {
		return nil, 0, "", fmt.Errorf("Mercury FetchTransactions: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "application/json;charset=utf-8")

	q := req.URL.Query()
	q.Add("start", startDate.Format("2006-01-02"))
	q.Add("end", endDate.Format("2006-01-02"))
	q.Add("limit", fmt.Sprintf("%d", limit))
	// asc so the cursor walks oldest→newest deterministically; if a long
	// backfill is interrupted, the oldest data has already been ingested.
	q.Add("order", "asc")
	if startAfter != "" {
		q.Add("start_after", startAfter)
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("Mercury FetchTransactions: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, "", fmt.Errorf("Mercury FetchTransactions: non-200 response: %d", resp.StatusCode)
	}

	var envelope mercuryListTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, 0, "", fmt.Errorf("Mercury FetchTransactions: failed to decode response: %w", err)
	}

	rawCount = len(envelope.Transactions)
	if rawCount > 0 {
		lastID = envelope.Transactions[rawCount-1].ID
	}

	var out []MercuryTransaction
	for _, tx := range envelope.Transactions {
		if tx.Status != mercuryStatusSent {
			continue
		}
		if !isSupportedKind(tx.Kind) {
			continue
		}
		// Attachment-or-not classification is the worker's job now —
		// see worker.go. Both branches need to be tracked so the
		// completeness gate can fail on unreceipted spend.
		out = append(out, tx)
	}

	return out, rawCount, lastID, nil
}

// FetchTransactionsByIDs fetches Mercury transactions in the caller-supplied
// [since, until] window and returns a map keyed by tx ID for the requested IDs
// only. The window is explicit (rather than a hard-coded lookback) so callers
// like the receipt recovery path can derive it from the affected rows' event
// dates and never silently miss older transactions.
//
// Returns ONLY the txs whose ID is in the requested set (filters out others).
// IDs in the request set that Mercury doesn't return are simply absent from
// the result map — callers should treat that as "not found in Mercury".
//
// FetchTransactions paginates automatically, so wide windows with many
// transactions are handled correctly.
func FetchTransactionsByIDs(ctx context.Context, apiKey string, requestedIDs []string, since, until time.Time) (map[string]MercuryTransaction, error) {
	if len(requestedIDs) == 0 {
		return nil, nil
	}
	txs, err := fetchTransactions(ctx, apiKey, since, until)
	if err != nil {
		return nil, fmt.Errorf("FetchTransactionsByIDs: %w", err)
	}

	want := make(map[string]bool, len(requestedIDs))
	for _, id := range requestedIDs {
		want[id] = true
	}

	out := make(map[string]MercuryTransaction, len(requestedIDs))
	for _, tx := range txs {
		if want[tx.ID] {
			out[tx.ID] = tx
		}
	}
	return out, nil
}

func isSupportedKind(kind string) bool {
	switch kind {
	case mercuryKindCreditCard, mercuryKindDebitCard,
		mercuryKindCreditCardCredit, mercuryKindDebitCardCredit:
		return true
	}
	return false
}

// downloadReceiptFile downloads a receipt attachment from the given URL and
// returns the raw bytes and detected content type.
func downloadReceiptFile(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("downloadReceiptFile: failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("downloadReceiptFile: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("downloadReceiptFile: non-200 response: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("downloadReceiptFile: failed to read body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}
