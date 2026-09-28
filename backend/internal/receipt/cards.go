package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Who swiped which card.
//
// Mercury's transaction JSON identifies a card swipe only by `cardId`. The
// holder's name and the last four digits live on a separate endpoint, per
// account: GET /api/v1/account/{id}/cards. FetchCards walks every account
// once and returns a flat map keyed by card id (card ids are unique across
// accounts), which the worker consults per transaction via resolveCard.
//
// Live shape (probe 2026-09-28): ~20 accounts, most with zero or only
// cancelled cards; two active ones in use. Cancelled cards are kept in the
// map on purpose — a swipe from a card cancelled last week still needs its
// name on the pending row.

const mercuryAPIBase = "https://api.mercury.com/api/v1"

type mercuryAccountsResponse struct {
	Accounts []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"accounts"`
}

type mercuryCardsResponse struct {
	Cards []MercuryCard `json:"cards"`
}

func mercuryGetJSON(ctx context.Context, apiKey, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mercuryAPIBase+path, nil)
	if err != nil {
		return fmt.Errorf("mercury GET %s: build request: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json;charset=utf-8")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mercury GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mercury GET %s: non-200 response: %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("mercury GET %s: decode: %w", path, err)
	}
	return nil
}

// FetchCards returns every card on every account, keyed by card id.
func FetchCards(ctx context.Context, apiKey string) (map[string]MercuryCard, error) {
	var accounts mercuryAccountsResponse
	if err := mercuryGetJSON(ctx, apiKey, "/accounts", &accounts); err != nil {
		return nil, fmt.Errorf("FetchCards: %w", err)
	}
	cards := make(map[string]MercuryCard)
	for _, acct := range accounts.Accounts {
		var page mercuryCardsResponse
		if err := mercuryGetJSON(ctx, apiKey, "/account/"+acct.ID+"/cards", &page); err != nil {
			return nil, fmt.Errorf("FetchCards: account %s: %w", acct.ID, err)
		}
		for _, c := range page.Cards {
			if c.CardID != "" {
				cards[c.CardID] = c
			}
		}
	}
	return cards, nil
}

// resolveCard fills tx.CardHolder / tx.CardLast4 from the map. A transaction
// with no cardId, or a cardId the walk did not return, is left empty — the
// row stores NULL and the UI omits the label rather than guessing.
func resolveCard(tx *MercuryTransaction, cards map[string]MercuryCard) {
	if tx.CardID == "" {
		return
	}
	c, ok := cards[tx.CardID]
	if !ok {
		return
	}
	tx.CardHolder = c.NameOnCard
	tx.CardLast4 = c.LastFourDigits
}

// loadCards is the worker's non-fatal wrapper: a failed lookup logs and hands
// back an empty map, so receipts keep flowing and the label self-heals on the
// next poll through refreshCardOnRows.
func loadCards(ctx context.Context, apiKey string) map[string]MercuryCard {
	cards, err := fetchCards(ctx, apiKey)
	if err != nil {
		slog.Warn("receipt worker: Mercury cards lookup failed — card holder/last4 will be NULL this cycle", "error", err)
		return map[string]MercuryCard{}
	}
	return cards
}
