package inventory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setDisplay drives SetItemAliasDisplayHandler. An empty alias clears.
func setDisplay(t *testing.T, itemID, alias string) (int, map[string]string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"purchase_item_id": itemID, "alias": alias})
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	SetItemAliasDisplayHandler(testPool).ServeHTTP(rec, req)
	out := map[string]string{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	return rec.Code, out
}

// itemFromList returns the full PurchaseItem GET /items reports.
func itemFromList(t *testing.T, itemID string) PurchaseItem {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListItemsHandler(testPool).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListItems status = %d", rec.Code)
	}
	var items []PurchaseItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	for _, it := range items {
		if it.ID == itemID {
			return it
		}
	}
	t.Fatalf("item %s not in ListItems response", itemID)
	return PurchaseItem{}
}

func TestItemDisplayName(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	t.Run("unpromoted item displays its description", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		aliasReq(t, http.MethodPost, id, "HNY BEAR 24OZ")
		got := itemFromList(t, id)
		if got.DisplayName != "100% Cl Hny 24Z Bram" {
			t.Errorf("display_name = %q, want the description", got.DisplayName)
		}
		if got.DisplayAlias != "" {
			t.Errorf("display_alias = %q, want empty", got.DisplayAlias)
		}
	})

	// The bug this whole feature exists to prevent: with no promoted alias,
	// "just show an alias" renders the raw receipt string.
	t.Run("promoting beats the raw receipt alias", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		aliasReq(t, http.MethodPost, id, "100% CL HNY 24Z BRAM XX") // machine-learned
		aliasReq(t, http.MethodPost, id, "Honey")                   // human-typed
		if code, _ := setDisplay(t, id, "Honey"); code != http.StatusOK {
			t.Fatalf("set display status = %d, want 200", code)
		}
		got := itemFromList(t, id)
		if got.DisplayName != "Honey" {
			t.Errorf("display_name = %q, want Honey", got.DisplayName)
		}
		if got.DisplayAlias != "Honey" {
			t.Errorf("display_alias = %q, want Honey", got.DisplayAlias)
		}
		if got.Description != "100% Cl Hny 24Z Bram" {
			t.Errorf("description = %q — promoting must not touch the catalog identity", got.Description)
		}
	})

	t.Run("promotion is a radio, not a checkbox", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "Sugar Item")
		aliasReq(t, http.MethodPost, id, "Sugar")
		aliasReq(t, http.MethodPost, id, "White Sugar")
		setDisplay(t, id, "Sugar")
		if code, _ := setDisplay(t, id, "White Sugar"); code != http.StatusOK {
			t.Fatalf("second promote status = %d, want 200", code)
		}
		var n int
		if err := testPool.QueryRow(t.Context(),
			`SELECT count(*) FROM item_aliases WHERE purchase_item_id = $1 AND is_display`, id).Scan(&n); err != nil {
			t.Fatalf("count display aliases: %v", err)
		}
		if n != 1 {
			t.Fatalf("promoted alias count = %d, want exactly 1", n)
		}
		if got := itemFromList(t, id); got.DisplayName != "White Sugar" {
			t.Errorf("display_name = %q, want White Sugar", got.DisplayName)
		}
	})

	t.Run("clearing restores the description without losing the alias", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "Catalog Name")
		aliasReq(t, http.MethodPost, id, "Nickname")
		setDisplay(t, id, "Nickname")
		if code, _ := setDisplay(t, id, ""); code != http.StatusOK {
			t.Fatalf("clear status = %d, want 200", code)
		}
		got := itemFromList(t, id)
		if got.DisplayName != "Catalog Name" {
			t.Errorf("display_name = %q, want the description back", got.DisplayName)
		}
		if len(got.Aliases) != 1 || got.Aliases[0] != "Nickname" {
			t.Errorf("aliases = %v — clearing must demote, not delete", got.Aliases)
		}
	})

	t.Run("promoting an alias the item does not have 404s", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "Lonely Item")
		if code, _ := setDisplay(t, id, "Not An Alias"); code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", code)
		}
	})

	t.Run("deleting the promoted alias falls back to the description", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "Catalog Name")
		aliasReq(t, http.MethodPost, id, "Nickname")
		setDisplay(t, id, "Nickname")
		if code := aliasReq(t, http.MethodDelete, id, "Nickname"); code != http.StatusNoContent {
			t.Fatalf("delete status = %d", code)
		}
		if got := itemFromList(t, id); got.DisplayName != "Catalog Name" {
			t.Errorf("display_name = %q, want the description", got.DisplayName)
		}
	})

	// Both of these would abort on item_aliases_one_display_per_item if the
	// handlers did not demote first. They are the reason that index is safe.
	t.Run("re-pointing a promoted alias to an item that already has one", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey Catalog")
		agave := insertItem(t, "Agave Catalog")
		aliasReq(t, http.MethodPost, honey, "Sweet Stuff")
		setDisplay(t, honey, "Sweet Stuff")
		aliasReq(t, http.MethodPost, agave, "Agave Nectar")
		setDisplay(t, agave, "Agave Nectar")

		if code := aliasReq(t, http.MethodPost, agave, "SWEET STUFF"); code != http.StatusCreated {
			t.Fatalf("re-point status = %d, want 201", code)
		}
		got := itemFromList(t, agave)
		if got.DisplayName != "Agave Nectar" {
			t.Errorf("display_name = %q — a re-pointed alias must arrive unpromoted", got.DisplayName)
		}
	})

	t.Run("merging two items that each have a promoted alias", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey Catalog")
		bram := insertItem(t, "100% Cl Hny 24Z Bram")
		aliasReq(t, http.MethodPost, honey, "Honey")
		setDisplay(t, honey, "Honey")
		aliasReq(t, http.MethodPost, bram, "Clover Honey")
		setDisplay(t, bram, "Clover Honey")

		body, _ := json.Marshal(map[string]string{"source_id": bram, "target_id": honey})
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		MergeItemsHandler(testPool).ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("merge status = %d, want 204", rec.Code)
		}
		got := itemFromList(t, honey)
		if got.DisplayName != "Honey" {
			t.Errorf("display_name = %q — the target keeps the name it had", got.DisplayName)
		}
		var n int
		if err := testPool.QueryRow(t.Context(),
			`SELECT count(*) FROM item_aliases WHERE purchase_item_id = $1 AND is_display`, honey).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 1 {
			t.Errorf("promoted alias count after merge = %d, want 1", n)
		}
	})
}
