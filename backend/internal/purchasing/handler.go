package purchasing

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GetOrCreateOrderHandler returns (or creates) the draft PO for the current week.
// POST /api/v1/purchasing/orders
func GetOrCreateOrderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		po, err := GetOrCreateOrder(r.Context(), pool)
		if err != nil {
			log.Printf("GetOrCreateOrder: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, po)
	}
}

// GetOrderHandler returns a single PO with its line items.
// GET /api/v1/purchasing/orders/{id}
func GetOrderHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			log.Printf("GetOrderByID: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if po == nil {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSON(w, http.StatusOK, po)
	}
}

// UpsertLineItemsHandler replaces line items on a draft PO.
// PUT /api/v1/purchasing/orders/{id}/items
func UpsertLineItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		id := chi.URLParam(r, "id")

		var req UpsertLineItemsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		// require_draft=true: Order tab sends this to prevent editing locked POs
		// even for admin (admin edits locked POs via PO tab without this param)
		requireDraft := r.URL.Query().Get("require_draft") == "true"
		allowLocked := isAdmin(user) && !requireDraft

		if err := UpsertLineItems(r.Context(), pool, id, user.ID, req.Items, allowLocked); err != nil {
			if errors.Is(err, ErrPOLockedAdminOnly) {
				writeError(w, http.StatusForbidden, "po_locked_admin_only")
				return
			}
			if errors.Is(err, ErrPONotDraft) {
				writeError(w, http.StatusConflict, "po_not_draft")
				return
			}
			log.Printf("UpsertLineItems: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			log.Printf("GetOrderByID after upsert: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, po)
	}
}

// GetSuggestionsHandler returns items below their group threshold not already on this PO.
// GET /api/v1/purchasing/orders/{id}/suggestions
func GetSuggestionsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		suggestions, err := GetSuggestions(r.Context(), pool, id)
		if err != nil {
			log.Printf("GetSuggestions: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if suggestions == nil {
			suggestions = []OrderSuggestion{}
		}
		writeJSON(w, http.StatusOK, suggestions)
	}
}

// GetActiveShoppingListHandler returns the active shopping list or 404 if none.
// GET /api/v1/purchasing/shopping/active
func GetActiveShoppingListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		sl, err := GetActiveShoppingList(r.Context(), pool)
		if err != nil {
			log.Printf("GetActiveShoppingList: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if sl == nil {
			writeError(w, http.StatusNotFound, "no_active_shopping_list")
			return
		}
		writeJSON(w, http.StatusOK, sl)
	}
}

// GetShoppingListHistoryHandler returns past completed shopping lists.
// GET /api/v1/purchasing/shopping/history
func GetShoppingListHistoryHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		lists, err := GetShoppingListHistory(r.Context(), pool)
		if err != nil {
			log.Printf("GetShoppingListHistory: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if lists == nil {
			lists = []ShoppingList{}
		}
		writeJSON(w, http.StatusOK, lists)
	}
}

// GetShoppingListHandler returns a specific shopping list with items.
// GET /api/v1/purchasing/shopping/{id}
func GetShoppingListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		id := chi.URLParam(r, "id")
		sl, err := GetShoppingListByID(r.Context(), pool, id)
		if err != nil {
			log.Printf("GetShoppingListByID: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if sl == nil {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSON(w, http.StatusOK, sl)
	}
}

// CheckShoppingItemHandler toggles the checked state on a shopping list item.
// POST /api/v1/purchasing/shopping/{id}/check
func CheckShoppingItemHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req struct {
			ItemID  string `json:"item_id"`
			Checked bool   `json:"checked"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == "" {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := CheckShoppingItem(r.Context(), pool, req.ItemID, req.Checked, user.ID); err != nil {
			log.Printf("CheckShoppingItem: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// UpdateShoppingItemLocationHandler updates store_location on a shopping list item (and its catalog item).
// PUT /api/v1/purchasing/shopping/{id}/items/{itemId}/location
func UpdateShoppingItemLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		itemID := chi.URLParam(r, "itemId")
		var req struct {
			StoreLocation string `json:"store_location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := UpdateShoppingItemLocation(r.Context(), pool, itemID, req.StoreLocation); err != nil {
			log.Printf("UpdateShoppingItemLocation: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// UpdateShoppingItemPhotoHandler updates photo_url on a shopping list item (and its catalog item).
// PUT /api/v1/purchasing/shopping/{id}/items/{itemId}/photo
func UpdateShoppingItemPhotoHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		itemID := chi.URLParam(r, "itemId")
		var req struct {
			PhotoURL string `json:"photo_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhotoURL == "" {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		if err := UpdateShoppingItemPhoto(r.Context(), pool, itemID, req.PhotoURL); err != nil {
			log.Printf("UpdateShoppingItemPhoto: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

// CompleteVendorSectionHandler marks a vendor section as completed and cascades if all sections done.
// POST /api/v1/purchasing/shopping/{id}/vendors/{vendorSectionId}/complete
func CompleteVendorSectionHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		vendorSectionID := chi.URLParam(r, "vendorSectionId")

		listCompleted, err := CompleteVendorSection(r.Context(), pool, vendorSectionID, user.ID)
		if err != nil {
			log.Printf("CompleteVendorSection: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"completed":      true,
			"list_completed": listCompleted,
		})
	}
}

// GetCutoffConfigHandler returns the current cutoff config or an empty object.
// GET /api/v1/purchasing/cutoff
func GetCutoffConfigHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		cfg, err := GetCutoffConfig(r.Context(), pool)
		if err != nil {
			log.Printf("GetCutoffConfig: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if cfg == nil {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

// UpsertCutoffConfigHandler saves the cutoff config (admin-only, per D-01).
// PUT /api/v1/purchasing/cutoff
func UpsertCutoffConfigHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		var req struct {
			DayOfWeek  int    `json:"day_of_week"`
			CutoffTime string `json:"cutoff_time"`
			Timezone   string `json:"timezone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if req.CutoffTime == "" || req.Timezone == "" {
			writeError(w, http.StatusBadRequest, "cutoff_time and timezone required")
			return
		}

		cfg, err := UpsertCutoffConfig(r.Context(), pool, req.DayOfWeek, req.CutoffTime, req.Timezone)
		if err != nil {
			log.Printf("UpsertCutoffConfig: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

// SimulateCutoffHandler immediately locks the current draft PO (admin-only, per D-04/D-05/D-06).
// POST /api/v1/purchasing/simulate-cutoff
func SimulateCutoffHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		// Block if there's already a locked PO awaiting approval
		lockedPO, err := GetOrdersByStatus(r.Context(), pool, "locked")
		if err != nil {
			log.Printf("SimulateCutoff: check locked: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if lockedPO != nil {
			writeError(w, http.StatusConflict, "locked_po_pending_approval")
			return
		}

		// Find current draft PO
		po, err := GetOrCreateOrder(r.Context(), pool)
		if err != nil {
			log.Printf("SimulateCutoff: GetOrCreateOrder: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if po.Status != "draft" {
			writeError(w, http.StatusConflict, "po_not_draft")
			return
		}

		if err := LockPO(r.Context(), pool, po.ID); err != nil {
			if errors.Is(err, ErrPONotDraft) {
				writeError(w, http.StatusConflict, "po_not_draft")
				return
			}
			log.Printf("SimulateCutoff: LockPO: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Return the now-locked PO
		locked, err := GetOrderByID(r.Context(), pool, po.ID)
		if err != nil {
			log.Printf("SimulateCutoff: GetOrderByID: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, locked)
	}
}

// GetOrdersByStatusHandler returns the most recent PO with the given status.
// GET /api/v1/purchasing/orders?status=locked
func GetOrdersByStatusHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		status := r.URL.Query().Get("status")
		if status == "" {
			writeError(w, http.StatusBadRequest, "status query param required")
			return
		}

		po, err := GetOrdersByStatus(r.Context(), pool, status)
		if err != nil {
			log.Printf("GetOrdersByStatus: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		// Return null (not 404) when no PO with this status exists — avoids console errors in frontend
		writeJSON(w, http.StatusOK, po)
	}
}

// LockPOHandler locks a draft PO (admin-only).
// POST /api/v1/purchasing/orders/{id}/lock
func LockPOHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		id := chi.URLParam(r, "id")
		if err := LockPO(r.Context(), pool, id); err != nil {
			if errors.Is(err, ErrPONotDraft) {
				writeError(w, http.StatusConflict, "po_not_draft")
				return
			}
			log.Printf("LockPO: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			log.Printf("LockPOHandler: GetOrderByID: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, po)
	}
}

// UnlockPOHandler unlocks a locked PO (admin-only, blocked after approval per D-13).
// POST /api/v1/purchasing/orders/{id}/unlock
func UnlockPOHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		id := chi.URLParam(r, "id")
		if err := UnlockPO(r.Context(), pool, id); err != nil {
			if errors.Is(err, ErrPONotLocked) {
				writeError(w, http.StatusConflict, "po_not_locked")
				return
			}
			if errors.Is(err, ErrUnlockAfterApproval) {
				writeError(w, http.StatusConflict, "cannot_unlock_after_approval")
				return
			}
			log.Printf("UnlockPO: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			log.Printf("UnlockPOHandler: GetOrderByID: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, po)
	}
}

// ApprovePOHandler approves a locked PO and creates a shopping list snapshot (admin-only, per D-10).
// POST /api/v1/purchasing/orders/{id}/approve
func ApprovePOHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		id := chi.URLParam(r, "id")
		listID, err := ApprovePO(r.Context(), pool, id, user.ID)
		if err != nil {
			if errors.Is(err, ErrPONotLocked) {
				writeError(w, http.StatusConflict, "po_not_locked")
				return
			}
			if errors.Is(err, ErrActiveShoppingListExists) {
				writeError(w, http.StatusConflict, "active_shopping_list_exists")
				return
			}
			log.Printf("ApprovePO: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"shopping_list_id": listID})
	}
}
