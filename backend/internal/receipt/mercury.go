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

// FetchTransactionByID fetches a single Mercury transaction by its ID using
// the GET /api/v1/transactions/{id} endpoint. This bypasses the list
// endpoint's date-range filter, so it can recover transactions older than
// the worker's normal 14-day lookback window.
//
// Returns (nil, nil) when Mercury returns 404 (transaction not found or
// deleted). All other non-2xx responses return a non-nil error.
func FetchTransactionByID(ctx context.Context, apiKey string, txID string) (*MercuryTransaction, error) {
	url := fmt.Sprintf("https://api.mercury.com/api/v1/transactions/%s", txID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactionByID: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "application/json;charset=utf-8")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactionByID: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Transaction not found — caller skips the row.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mercury FetchTransactionByID: non-200 response: %d", resp.StatusCode)
	}

	var tx MercuryTransaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, fmt.Errorf("Mercury FetchTransactionByID: failed to decode response: %w", err)
	}

	return &tx, nil
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
