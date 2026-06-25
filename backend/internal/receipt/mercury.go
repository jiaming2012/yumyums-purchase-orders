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
	mercuryStatusSent             = "sent"
	mercuryKindCreditCard         = "creditCardTransaction"
	mercuryKindDebitCard          = "debitCardTransaction"
	mercuryKindCreditCardCredit   = "creditCardCredit"
	mercuryKindDebitCardCredit    = "debitCardCredit"
)

// FetchTransactions fetches Mercury transactions for the given date range.
// Returns every "sent" supported transaction regardless of attachment count;
// classifying attached vs. unattached rows is the worker's job (see worker.go)
// so the completeness gate can fail on unreceipted card spend.
func FetchTransactions(ctx context.Context, apiKey string, startDate, endDate time.Time) ([]MercuryTransaction, error) {
	url := "https://api.mercury.com/api/v1/transactions"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactions: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "application/json;charset=utf-8")

	q := req.URL.Query()
	q.Add("start", startDate.Format("2006-01-02"))
	q.Add("end", endDate.Format("2006-01-02"))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactions: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mercury FetchTransactions: non-200 response: %d", resp.StatusCode)
	}

	var envelope mercuryListTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactions: failed to decode response: %w", err)
	}

	if len(envelope.Transactions) >= 1000 {
		return nil, fmt.Errorf("Mercury FetchTransactions: response limit reached — implement pagination")
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

	return out, nil
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
// Errors out if Mercury returns >= 1000 transactions (existing pagination
// limit — see FetchTransactions). A 1-year window for a small business
// should never hit this; if it does, the operator must shorten the window
// or implement pagination.
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
