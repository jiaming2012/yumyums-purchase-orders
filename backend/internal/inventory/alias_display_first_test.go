package inventory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// addAliasOpts drives AddItemAliasHandler with the display_if_first flag the
// Setup editor sends. Returns the status and decoded body.
func addAliasOpts(t *testing.T, itemID, alias string, displayIfFirst bool) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"purchase_item_id": itemID, "alias": alias, "display_if_first": displayIfFirst})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	AddItemAliasHandler(testPool).ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	return rec.Code, out
}

// TestAddAlias_DisplayIfFirst — a nickname typed in Setup for an item that has
// no nicknames yet becomes the displayed name by default; once any nickname
// exists (starred or not) a new one arrives unpromoted. The receipt-link
// auto-learn path never sends the flag, so raw receipt text cannot promote
// itself.
func TestAddAlias_DisplayIfFirst(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	t.Run("the first nickname is promoted", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		code, body := addAliasOpts(t, id, "Honey", true)
		if code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", code)
		}
		if body["display_alias"] != "Honey" {
			t.Errorf("response display_alias = %v, want Honey", body["display_alias"])
		}
		got := itemFromList(t, id)
		if got.DisplayAlias != "Honey" || got.DisplayName != "Honey" {
			t.Errorf("display_alias=%q display_name=%q, want Honey/Honey", got.DisplayAlias, got.DisplayName)
		}
	})

	t.Run("a second nickname is not promoted, and the first keeps the star", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		addAliasOpts(t, id, "Honey", true)
		code, body := addAliasOpts(t, id, "Bear Honey", true)
		if code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", code)
		}
		if body["display_alias"] != "" {
			t.Errorf("response display_alias = %v, want empty (not promoted)", body["display_alias"])
		}
		got := itemFromList(t, id)
		if got.DisplayAlias != "Honey" {
			t.Errorf("display_alias = %q, want Honey (first keeps the star)", got.DisplayAlias)
		}
		if len(got.Aliases) != 2 {
			t.Errorf("aliases = %v, want both", got.Aliases)
		}
	})

	t.Run("an existing UNSTARRED nickname still blocks the default", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		addAliasOpts(t, id, "Honey", true)
		setDisplay(t, id, "") // operator cleared the star
		code, _ := addAliasOpts(t, id, "Bear Honey", true)
		if code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", code)
		}
		got := itemFromList(t, id)
		if got.DisplayAlias != "" || got.DisplayName != "100% Cl Hny 24Z Bram" {
			t.Errorf("display_alias=%q display_name=%q, want none / the description", got.DisplayAlias, got.DisplayName)
		}
	})

	t.Run("without the flag (auto-learn) the first nickname is NOT promoted", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "100% Cl Hny 24Z Bram")
		if code := aliasReq(t, http.MethodPost, id, "HNY BEAR 24OZ"); code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", code)
		}
		got := itemFromList(t, id)
		if got.DisplayAlias != "" {
			t.Errorf("display_alias = %q, want empty — receipt text must never self-promote", got.DisplayAlias)
		}
	})

	t.Run("a nickname equal to the description is still skipped, never promoted", func(t *testing.T) {
		resetFixtures(t)
		id := insertItem(t, "Honey")
		code, _ := addAliasOpts(t, id, "honey", true)
		if code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", code)
		}
		got := itemFromList(t, id)
		if len(got.Aliases) != 0 || got.DisplayAlias != "" {
			t.Errorf("aliases=%v display_alias=%q, want none", got.Aliases, got.DisplayAlias)
		}
	})

	t.Run("re-pointing an alias onto an item with no nicknames promotes it there", func(t *testing.T) {
		resetFixtures(t)
		a := insertItem(t, "Item A")
		b := insertItem(t, "Item B")
		addAliasOpts(t, a, "Honey", true)
		addAliasOpts(t, a, "Other", true) // A now has two; Honey starred
		code, _ := addAliasOpts(t, b, "honey", true)
		if code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", code)
		}
		gotA, gotB := itemFromList(t, a), itemFromList(t, b)
		if gotB.DisplayAlias == "" {
			t.Errorf("B display_alias empty, want the re-pointed nickname (B had none)")
		}
		if gotA.DisplayAlias != "" {
			t.Errorf("A display_alias = %q, want empty — the starred alias left A", gotA.DisplayAlias)
		}
	})
}
