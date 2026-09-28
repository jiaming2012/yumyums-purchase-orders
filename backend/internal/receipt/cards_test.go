package receipt

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var errStubCards = errors.New("stub: Mercury cards lookup failed")

// ---------------------------------------------------------------------------
// Who swiped which card (Purchases tab: "Mercury: COGS · Jamal · 8478").
//
// RED-FIRST: written against the tree where MercuryTransaction has no CardID,
// FetchCards / resolveCard do not exist, and pending_purchases has no
// card_holder / card_last4 columns. Mercury's transaction JSON carries only
// `cardId`; the holder and last-4 live on GET /account/{id}/cards (verified
// against the live API 2026-09-28: two active cards, "Jamal Cole" ••8478 and
// "Latanya Mcgriff" ••0994, ~20 accounts to walk).
// ---------------------------------------------------------------------------

// mercuryCardsServer serves the two endpoints FetchCards walks: the accounts
// list and each account's cards. Card ids are unique across accounts in
// Mercury, so the map the walk produces is flat.
func mercuryCardsServer(t *testing.T, accountCards map[string][]MercuryCard) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer stub-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/accounts":
			accounts := []map[string]string{}
			for id := range accountCards {
				accounts = append(accounts, map[string]string{"id": id, "name": "Mercury Checking ••" + id})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"accounts": accounts})
		case strings.HasPrefix(r.URL.Path, "/api/v1/account/") && strings.HasSuffix(r.URL.Path, "/cards"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/account/"), "/cards")
			cards, ok := accountCards[id]
			if !ok {
				http.Error(w, "no such account", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"cards": cards})
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
}

func TestFetchCards_WalksEveryAccountAndKeysByCardID(t *testing.T) {
	srv := mercuryCardsServer(t, map[string][]MercuryCard{
		"4613": {
			{CardID: "card-jamal", NameOnCard: "Jamal Cole", LastFourDigits: "8478"},
			{CardID: "card-latanya", NameOnCard: "Latanya Mcgriff", LastFourDigits: "0994"},
		},
		"7818": {}, // savings — no cards
		"2368": {
			{CardID: "card-james", NameOnCard: "James Cole", LastFourDigits: "1189"},
		},
	})
	defer srv.Close()
	origTransport := http.DefaultTransport
	http.DefaultTransport = rewriteHostTransport(srv.URL)
	defer func() { http.DefaultTransport = origTransport }()

	cards, err := FetchCards(t.Context(), "stub-key")
	if err != nil {
		t.Fatalf("FetchCards: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("len(cards) = %d, want 3 (every account walked, empty ones tolerated): %+v", len(cards), cards)
	}
	if got := cards["card-jamal"]; got.NameOnCard != "Jamal Cole" || got.LastFourDigits != "8478" {
		t.Errorf("card-jamal = %+v, want Jamal Cole / 8478", got)
	}
	if got := cards["card-james"]; got.NameOnCard != "James Cole" || got.LastFourDigits != "1189" {
		t.Errorf("card-james = %+v, want James Cole / 1189", got)
	}
}

func TestMercuryTransaction_DecodesCardID(t *testing.T) {
	// The shape Mercury actually returns (live probe 2026-09-28): a flat
	// cardId plus the legacy details.debitCardInfo.id. We read the flat one.
	raw := `{"id":"tx-1","amount":-250.79,"status":"sent","kind":"debitCardTransaction",
	         "cardId":"6a623cba-df4c-11f0-b81f-1be9d4fac4aa",
	         "details":{"debitCardInfo":{"id":"6a623cba-df4c-11f0-b81f-1be9d4fac4aa"}}}`
	var tx MercuryTransaction
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tx.CardID != "6a623cba-df4c-11f0-b81f-1be9d4fac4aa" {
		t.Errorf("CardID = %q, want the flat cardId", tx.CardID)
	}
}

func TestResolveCard_SetsHolderAndLast4_UnknownCardLeavesEmpty(t *testing.T) {
	cards := map[string]MercuryCard{
		"card-jamal": {CardID: "card-jamal", NameOnCard: "Jamal Cole", LastFourDigits: "8478"},
	}
	tx := MercuryTransaction{ID: "t1", CardID: "card-jamal"}
	resolveCard(&tx, cards)
	if tx.CardHolder != "Jamal Cole" || tx.CardLast4 != "8478" {
		t.Errorf("resolved = %q / %q, want Jamal Cole / 8478", tx.CardHolder, tx.CardLast4)
	}

	// A card the walk did not return (cancelled + pruned, or a lookup that
	// failed and handed back an empty map) must NOT invent a label.
	unknown := MercuryTransaction{ID: "t2", CardID: "card-gone"}
	resolveCard(&unknown, cards)
	if unknown.CardHolder != "" || unknown.CardLast4 != "" {
		t.Errorf("unknown card resolved to %q / %q, want empty", unknown.CardHolder, unknown.CardLast4)
	}

	// Not a card swipe at all (ACH, wire): no cardId, nothing to resolve.
	ach := MercuryTransaction{ID: "t3"}
	resolveCard(&ach, cards)
	if ach.CardHolder != "" || ach.CardLast4 != "" {
		t.Errorf("no-card tx resolved to %q / %q, want empty", ach.CardHolder, ach.CardLast4)
	}
}

func TestInsertPendingPurchase_WritesCardHolderAndLast4(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:              "tx-card-1",
		Amount:          -113.97,
		BankDescription: "RESTAURANT DEPOT",
		CreatedAt:       "2026-09-27T10:00:00Z",
		CardID:          "card-jamal",
		CardHolder:      "Jamal Cole",
		CardLast4:       "8478",
	}
	if err := insertPendingPurchase(t.Context(), testPool, tx, nil, ReceiptSummary{}, "", nil, "no_attachment_on_bank_tx", ""); err != nil {
		t.Fatalf("insertPendingPurchase: %v", err)
	}
	var holder, last4 sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT card_holder, card_last4 FROM pending_purchases WHERE bank_tx_id = $1`, "tx-card-1",
	).Scan(&holder, &last4); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !holder.Valid || holder.String != "Jamal Cole" {
		t.Errorf("card_holder = %+v, want Jamal Cole", holder)
	}
	if !last4.Valid || last4.String != "8478" {
		t.Errorf("card_last4 = %+v, want 8478", last4)
	}

	// An unresolved card stores NULL, not "" — the UI keys on absence.
	none := MercuryTransaction{ID: "tx-card-2", Amount: -5, BankDescription: "X", CreatedAt: "2026-09-27T10:00:00Z"}
	if err := insertPendingPurchase(t.Context(), testPool, none, nil, ReceiptSummary{}, "", nil, "no_attachment_on_bank_tx", ""); err != nil {
		t.Fatalf("insertPendingPurchase (no card): %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT card_holder, card_last4 FROM pending_purchases WHERE bank_tx_id = $1`, "tx-card-2",
	).Scan(&holder, &last4); err != nil {
		t.Fatalf("select (no card): %v", err)
	}
	if holder.Valid || last4.Valid {
		t.Errorf("no-card row = %+v / %+v, want NULL / NULL", holder, last4)
	}
}

// The 59 rows already sitting in the queue (and every confirmed event inside
// the lookback window) pre-date the column: the worker's per-tx refresh must
// backfill them on the next poll, the way mercury_category self-heals.
func TestRunIngestCycle_BackfillsCardOnExistingRows(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
		VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb)`,
		"T-card-backfill", -113.97, "RESTAURANT DEPOT",
	); err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	var vendorID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO vendors (name) VALUES ('Restaurant Depot') RETURNING id`).Scan(&vendorID); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total)
		VALUES ($1, $2, '2026-09-20', 0, 50)`, vendorID, "T-card-event",
	); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{
			{ID: "T-card-backfill", Amount: -113.97, CreatedAt: "2026-09-27T10:00:00Z", CardID: "card-jamal"},
			{ID: "T-card-event", Amount: -50, CreatedAt: "2026-09-20T10:00:00Z", CardID: "card-latanya"},
		},
		cards: map[string]MercuryCard{
			"card-jamal":   {CardID: "card-jamal", NameOnCard: "Jamal Cole", LastFourDigits: "8478"},
			"card-latanya": {CardID: "card-latanya", NameOnCard: "Latanya Mcgriff", LastFourDigits: "0994"},
		},
	}
	installWorkerStubs(t, stubs)

	if _, err := runIngestCycle(t.Context(), WorkerConfig{MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14}); err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	var holder, last4 sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT card_holder, card_last4 FROM pending_purchases WHERE bank_tx_id = $1`, "T-card-backfill",
	).Scan(&holder, &last4); err != nil {
		t.Fatalf("select pending: %v", err)
	}
	if holder.String != "Jamal Cole" || last4.String != "8478" {
		t.Errorf("pending backfill = %+v / %+v, want Jamal Cole / 8478", holder, last4)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT card_holder, card_last4 FROM purchase_events WHERE bank_tx_id = $1`, "T-card-event",
	).Scan(&holder, &last4); err != nil {
		t.Fatalf("select event: %v", err)
	}
	if holder.String != "Latanya Mcgriff" || last4.String != "0994" {
		t.Errorf("event backfill = %+v / %+v, want Latanya Mcgriff / 0994", holder, last4)
	}
}

// A cards lookup that fails must not fail the ingest cycle: receipts still
// flow, the label is just absent until the next poll succeeds.
func TestRunIngestCycle_CardsLookupFailureIsNonFatal(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns:     []MercuryTransaction{{ID: "T-nocards", Amount: -9.99, CreatedAt: "2026-09-27T10:00:00Z", CardID: "card-jamal"}},
		cardsErr: errStubCards,
	}
	installWorkerStubs(t, stubs)

	res, err := runIngestCycle(t.Context(), WorkerConfig{MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}
	if res.PendingReview != 1 {
		t.Errorf("PendingReview = %d, want 1 — the swipe must still land in the queue", res.PendingReview)
	}
	var holder sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT card_holder FROM pending_purchases WHERE bank_tx_id = $1`, "T-nocards").Scan(&holder); err != nil {
		t.Fatalf("select: %v", err)
	}
	if holder.Valid {
		t.Errorf("card_holder = %q, want NULL when the lookup failed", holder.String)
	}
}
