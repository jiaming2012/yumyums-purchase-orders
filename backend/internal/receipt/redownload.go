package receipt

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RedownloadFromMercury re-fetches ONE bank transaction from Mercury and runs
// the standard download/parse/persist pipeline on whatever attachments it
// carries NOW. This is the answer when the operator replaced the receipt on
// Mercury — the wrong file was attached, or a purchase has since been paired
// with its refund — and the row in HQ still holds the old download. A
// storage re-read (ReprocessFromSpaces) would only re-parse the stale files;
// only Mercury has the new ones.
//
// Every attachment on the transaction is taken, so a purchase + refund pair
// reaches the parser together (see processSingleTx's download loop), and the
// stored receipt_url/receipt_urls are rewritten by updatePendingPurchase.
//
// Returns one of "auto_created", "pending_review", "cached", "errored" (from
// processSingleTx), or two outcomes of its own that the caller must report as
// failures, because nothing was changed:
//   - "missing_at_mercury": Mercury did not return the transaction in
//     [since, now]
//   - "no_attachments":     it came back with no receipt attached
func RedownloadFromMercury(ctx context.Context, cfg WorkerConfig, bankTxID string, since time.Time) (string, error) {
	if cfg.MercuryAPIKey == "" {
		return "errored", fmt.Errorf("RedownloadFromMercury: MERCURY_API_KEY not set")
	}
	txMap, err := FetchTransactionsByIDs(ctx, cfg.MercuryAPIKey, []string{bankTxID}, since, time.Now())
	if err != nil {
		return "errored", fmt.Errorf("RedownloadFromMercury: %w", err)
	}
	tx, ok := txMap[bankTxID]
	if !ok {
		slog.Info(fmt.Sprintf("receipt worker: redownload tx %s — not returned by Mercury since %s", bankTxID, since.Format("2006-01-02")))
		return "missing_at_mercury", nil
	}
	if len(tx.Attachments) == 0 {
		slog.Info(fmt.Sprintf("receipt worker: redownload tx %s — no attachments at Mercury", bankTxID))
		return "no_attachments", nil
	}
	slog.Info(fmt.Sprintf("receipt worker: redownload tx %s — %d attachment(s) at Mercury, re-reading", bankTxID, len(tx.Attachments)))
	return processSingleTx(ctx, cfg, tx, true)
}
