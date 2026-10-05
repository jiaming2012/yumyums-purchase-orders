package recipes

// zz_spike_k2_test.go — THROWAWAY spike test for K2 (dish-merge-shapes-and-backstop-tests;
// B-485 shapes and B-487 backstop). Copied by the spike script beside repository_test.go in a
// throwaway worktree and run there; never part of the tree. PASS means the baseline holds:
//   (a1) a merge whose SOURCE names no dish answers 200 {"rows_re_pointed":0}   — wrong shape
//   (a2) a merge whose target is not a uuid answers 500                           — wrong shape
//   (c)  deleting a dish a campaign and a code name leaves both rows with item_id NULL — the
//        backstop works, so the card's behavioural test has a known green.

import (
	"context"
	"net/http"
	"testing"
)

func TestSpikeK2ShapesAndBackstop(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	target := seedMenuItem(t, pool, "Spike Target Bowl")

	mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
	recA1 := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
		"source_menu_item_id": mergeMissingTargetID, "target_menu_item_id": target,
	})
	recA2 := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
		"source_menu_item_id": target, "target_menu_item_id": "not-a-uuid",
	})
	t.Logf("SPIKE-K2-a: missing-source=%d body=%s | non-uuid-target=%d body=%s",
		recA1.Code, recA1.Body.String(), recA2.Code, recA2.Body.String())
	if recA1.Code != http.StatusOK {
		t.Errorf("(a1) missing source answered %d; B-485 says 200 on today's tree", recA1.Code)
	}
	if recA2.Code != http.StatusInternalServerError {
		t.Errorf("(a2) non-uuid target answered %d; B-485 says 500 on today's tree", recA2.Code)
	}

	// (c) the deploy backstop, by behaviour.
	dish := seedMenuItem(t, pool, "Spike Doomed Dish")
	var userID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ('spike-k2-' || gen_random_uuid()::text || '@test.invalid', 'Spike', 'K2', ARRAY['manager'], 'active')
		 RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	var campaignID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, item_id, ends_at, created_by)
		 VALUES (gen_random_uuid(), 'spike-k2-' || substr(gen_random_uuid()::text, 1, 8), 'Spike K2', '$2 off', 200, false, $1, now() + interval '30 days', $2)
		 RETURNING id::text`, dish, userID).Scan(&campaignID); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	// qr_codes.short must match ^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$: hex digits 0/1 are
	// translated to 2/3 so the random suffix always satisfies the check.
	var codeID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO qr_codes (short, campaign_id, channel, item_id, created_by)
		 VALUES ('SK2' || upper(translate(substr(md5(gen_random_uuid()::text), 1, 3), '01', '23')), $1, 'flyer', $2, $3)
		 RETURNING id::text`, campaignID, dish, userID).Scan(&codeID); err != nil {
		t.Fatalf("seed code: %v", err)
	}
	_ = codeID
	if _, err := pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, dish); err != nil {
		t.Fatalf("(c) DELETE dish: %v — the backstop should let a referenced dish go", err)
	}
	var campItem, codeItem *string
	var campRows, codeRows int
	_ = pool.QueryRow(ctx, `SELECT count(*), max(item_id::text) FROM campaigns_admin WHERE id = $1`, campaignID).Scan(&campRows, &campItem)
	_ = pool.QueryRow(ctx, `SELECT count(*), max(item_id::text) FROM qr_codes WHERE campaign_id = $1`, campaignID).Scan(&codeRows, &codeItem)
	t.Logf("SPIKE-K2-c: campaign rows=%d item_id=%v | code rows=%d item_id=%v", campRows, campItem, codeRows, codeItem)
	if campRows != 1 || campItem != nil {
		t.Errorf("(c) campaign did not survive with item_id NULL (rows=%d item_id=%v)", campRows, campItem)
	}
	if codeRows != 1 || codeItem != nil {
		t.Errorf("(c) code did not survive with item_id NULL (rows=%d item_id=%v)", codeRows, codeItem)
	}
}
