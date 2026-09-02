package inventory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// aliasReq drives AddItemAliasHandler / DeleteItemAliasHandler directly.
func aliasReq(t *testing.T, method, itemID, alias string) int {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"purchase_item_id": itemID, "alias": alias})
	req := httptest.NewRequest(method, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	if method == http.MethodDelete {
		DeleteItemAliasHandler(testPool).ServeHTTP(rec, req)
	} else {
		AddItemAliasHandler(testPool).ServeHTTP(rec, req)
	}
	return rec.Code
}

// listAliases returns the aliases GET /items reports for one item.
func listAliases(t *testing.T, itemID string) []string {
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
			return it.Aliases
		}
	}
	t.Fatalf("item %s not in ListItems response", itemID)
	return nil
}

func insertItem(t *testing.T, desc string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ($1) RETURNING id::text`, desc).Scan(&id)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	return id
}

func TestItemAliases(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	t.Run("add then list", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey")
		if code := aliasReq(t, http.MethodPost, honey, "100% Cl Hny 24z Bram"); code != http.StatusCreated {
			t.Fatalf("add alias status = %d, want 201", code)
		}
		got := listAliases(t, honey)
		if len(got) != 1 || got[0] != "100% Cl Hny 24z Bram" {
			t.Errorf("aliases = %v, want [100%% Cl Hny 24z Bram]", got)
		}
	})

	t.Run("alias equal to own description is a no-op", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey")
		if code := aliasReq(t, http.MethodPost, honey, "honey"); code != http.StatusNoContent {
			t.Fatalf("self-alias status = %d, want 204", code)
		}
		if got := listAliases(t, honey); len(got) != 0 {
			t.Errorf("aliases = %v, want none", got)
		}
	})

	t.Run("re-adding re-points to the new item", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey")
		agave := insertItem(t, "Agave")
		if code := aliasReq(t, http.MethodPost, honey, "Sweet Stuff"); code != http.StatusCreated {
			t.Fatalf("first add status = %d", code)
		}
		// Same alias, different case — latest link wins.
		if code := aliasReq(t, http.MethodPost, agave, "SWEET STUFF"); code != http.StatusCreated {
			t.Fatalf("re-point status = %d", code)
		}
		if got := listAliases(t, honey); len(got) != 0 {
			t.Errorf("honey aliases = %v, want none after re-point", got)
		}
		if got := listAliases(t, agave); len(got) != 1 {
			t.Errorf("agave aliases = %v, want the re-pointed alias", got)
		}
	})

	t.Run("delete removes and 404s when absent", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey")
		aliasReq(t, http.MethodPost, honey, "Bee Juice")
		if code := aliasReq(t, http.MethodDelete, honey, "bee juice"); code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want 204", code)
		}
		if got := listAliases(t, honey); len(got) != 0 {
			t.Errorf("aliases = %v, want none", got)
		}
		if code := aliasReq(t, http.MethodDelete, honey, "bee juice"); code != http.StatusNotFound {
			t.Fatalf("second delete status = %d, want 404", code)
		}
	})

	t.Run("merge carries aliases and learns source name", func(t *testing.T) {
		resetFixtures(t)
		honey := insertItem(t, "Honey")
		bram := insertItem(t, "100% Cl Hny 24Z Bram")
		aliasReq(t, http.MethodPost, bram, "Clover Honey")

		body, _ := json.Marshal(map[string]string{"source_id": bram, "target_id": honey})
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		MergeItemsHandler(testPool).ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("merge status = %d, want 204", rec.Code)
		}

		got := listAliases(t, honey)
		want := map[string]bool{"Clover Honey": false, "100% Cl Hny 24Z Bram": false}
		for _, a := range got {
			want[a] = true
		}
		for a, seen := range want {
			if !seen {
				t.Errorf("alias %q missing after merge (got %v)", a, got)
			}
		}
		if len(got) != 2 {
			t.Errorf("aliases = %v, want exactly the carried alias + source name", got)
		}
	})
}
