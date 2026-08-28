package receipt

import (
	"testing"
)

// TestLoadPurchaseItemsMap_IncludesAliases asserts that learned aliases enter
// the worker's catalog map as exact-match keys, that description keys always
// win a case-insensitive collision with an alias, and that an alias key makes
// previously human-linked receipt text an exact match in DerivePurchaseItemID.
func TestLoadPurchaseItemsMap_IncludesAliases(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	var honeyID, agaveID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Honey') RETURNING id::text`).Scan(&honeyID); err != nil {
		t.Fatalf("insert Honey: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Agave') RETURNING id::text`).Scan(&agaveID); err != nil {
		t.Fatalf("insert Agave: %v", err)
	}
	// A learned receipt name, and an alias colliding with Agave's description.
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO item_aliases (purchase_item_id, alias) VALUES ($1, '100% CL HNY 24Z BRAM'), ($1, 'AGAVE')`,
		honeyID); err != nil {
		t.Fatalf("insert aliases: %v", err)
	}

	m, err := loadPurchaseItemsMap(t.Context(), testPool)
	if err != nil {
		t.Fatalf("loadPurchaseItemsMap: %v", err)
	}

	if got := m["100% CL HNY 24Z BRAM"]; got != honeyID {
		t.Errorf("alias key -> %q, want Honey %q", got, honeyID)
	}
	// The 'AGAVE' alias case-insensitively collides with the Agave
	// description — the description must win.
	if got := m["Agave"]; got != agaveID {
		t.Errorf("m[Agave] = %q, want the item's own id %q", got, agaveID)
	}
	if _, shadowed := m["AGAVE"]; shadowed {
		t.Errorf("alias 'AGAVE' entered the map despite colliding with a description")
	}

	// End-to-end through the exact-match stage: the raw receipt text resolves
	// to Honey without fuzzy help.
	id, _, isNew := DerivePurchaseItemID("100% cl hny 24z bram", m)
	if isNew || id != honeyID {
		t.Errorf("DerivePurchaseItemID via alias = (%q, isNew=%v), want (%q, false)", id, isNew, honeyID)
	}
}
