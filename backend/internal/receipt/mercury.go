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
// Pagination: Mercury caps each response at 500 records. This function
// iterates with increasing offsets until a page shorter than the limit is
// returned (the standard "last page" sentinel). An offset safety ceiling of
// 50 000 prevents infinite loops if Mercury ever misbehaves.
func FetchTransactions(ctx context.Context, apiKey string, startDate, endDate time.Time) ([]MercuryTransaction, error) {
	var all []MercuryTransaction
	offset := 0
	for {
		filtered, rawCount, err := fetchTransactionsPage(ctx, apiKey, startDate, endDate, mercuryPageLimit, offset)
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
		offset += mercuryPageLimit
		if offset > 50000 {
			// Defensive: bail out if Mercury keeps returning full pages well
			// beyond any plausible transaction volume for this codebase.
			return nil, fmt.Errorf("Mercury FetchTransactions: offset exceeded 50000 — bailing")
		}
	}
	return all, nil
}

// fetchTransactionsPage fetches a single page of transactions from the Mercury
// list endpoint with the given limit and offset. It applies the status/kind
// filters so callers only see supported, sent transactions.
//
// Returns both the filtered slice and the raw count of transactions Mercury
// returned (before filtering). The caller must use rawCount — not
// len(filtered) — to decide whether to continue paginating, because a page
// of e.g. 500 raw txs where 460 are unsupported would look like a 40-tx
// "short page" if the caller used the filtered length, terminating the loop
// prematurely.
func fetchTransactionsPage(ctx context.Context, apiKey string, startDate, endDate time.Time, limit, offset int) (filtered []MercuryTransaction, rawCount int, err error) {
	const mercuryURL = "https://api.mercury.com/api/v1/transactions"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mercuryURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("Mercury FetchTransactions: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "application/json;charset=utf-8")

	q := req.URL.Query()
	q.Add("start", startDate.Format("2006-01-02"))
	q.Add("end", endDate.Format("2006-01-02"))
	q.Add("limit", fmt.Sprintf("%d", limit))
	q.Add("offset", fmt.Sprintf("%d", offset))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("Mercury FetchTransactions: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("Mercury FetchTransactions: non-200 response: %d", resp.StatusCode)
	}

	var envelope mercuryListTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, 0, fmt.Errorf("Mercury FetchTransactions: failed to decode response: %w", err)
	}

	rawCount = len(envelope.Transactions)

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

	return out, rawCount, nil
}

// FetchTransactionsByIDs fetches Mercury transactions in a wide date range
// (1 year back from now by default) and returns a map keyed by tx ID for
// the requested IDs only. Used by the reprocess pipeline to bypass the
// list endpoint's narrow lookback in the standard ingest path.
//
// Returns ONLY the txs whose ID is in the requested set (filters out others).
// IDs in the request set that Mercury doesn't return are simply absent from
// the result map — callers should treat that as "not found in Mercury".
//
// FetchTransactions now paginates automatically, so large 1-year windows
// with many transactions are handled correctly.
func FetchTransactionsByIDs(ctx context.Context, apiKey string, requestedIDs []string) (map[string]MercuryTransaction, error) {
	if len(requestedIDs) == 0 {
		return nil, nil
	}
	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0) // 1 year back

	txs, err := fetchTransactions(ctx, apiKey, startDate, endDate)
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
