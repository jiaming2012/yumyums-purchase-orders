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
// Only returns transactions that are "sent" and have attachments.
func FetchTransactions(ctx context.Context, apiKey string, startDate, endDate time.Time) ([]MercuryTransaction, error) {
	url := "https://api.mercury.com/api/v1/transactions"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("FetchTransactions: failed to create request: %w", err)
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
		return nil, fmt.Errorf("FetchTransactions: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FetchTransactions: non-200 response: %d", resp.StatusCode)
	}

	var envelope mercuryListTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("FetchTransactions: failed to decode response: %w", err)
	}

	if len(envelope.Transactions) >= 1000 {
		return nil, fmt.Errorf("FetchTransactions: response limit reached — implement pagination")
	}

	var out []MercuryTransaction
	for _, tx := range envelope.Transactions {
		if tx.Status != mercuryStatusSent {
			continue
		}
		if !isSupportedKind(tx.Kind) {
			continue
		}
		if len(tx.Attachments) > 0 {
			out = append(out, tx)
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
