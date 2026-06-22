package purchasing

import (
	"encoding/json"
	"errors"
	"log/slog"
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
			slog.Error("GetOrCreateOrder failed", "error", err)
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
			slog.Error("GetOrderByID failed", "error", err)
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
			slog.Error("UpsertLineItems failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			slog.Error("GetOrderByID after upsert failed", "error", err)
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
			slog.Error("GetSuggestions failed", "error", err)
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
			slog.Error("GetActiveShoppingList failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		// Return null (not 404) when no active list — avoids console errors in frontend
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
			slog.Error("GetShoppingListHistory failed", "error", err)
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
			slog.Error("GetShoppingListByID failed", "error", err)
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
			slog.Error("CheckShoppingItem failed", "error", err)
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
			slog.Error("UpdateShoppingItemLocation failed", "error", err)
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
			slog.Error("UpdateShoppingItemPhoto failed", "error", err)
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
			slog.Error("CompleteVendorSection failed", "error", err)
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
			slog.Error("GetCutoffConfig failed", "error", err)
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
			slog.Error("UpsertCutoffConfig failed", "error", err)
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
			slog.Error("SimulateCutoff check locked failed", "error", err)
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
			slog.Error("SimulateCutoff GetOrCreateOrder failed", "error", err)
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
			slog.Error("SimulateCutoff LockPO failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Return the now-locked PO
		locked, err := GetOrderByID(r.Context(), pool, po.ID)
		if err != nil {
			slog.Error("SimulateCutoff GetOrderByID failed", "error", err)
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
			slog.Error("GetOrdersByStatus failed", "error", err)
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
			slog.Error("LockPO failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			slog.Error("LockPOHandler GetOrderByID failed", "error", err)
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
			slog.Error("UnlockPO failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		po, err := GetOrderByID(r.Context(), pool, id)
		if err != nil {
			slog.Error("UnlockPOHandler GetOrderByID failed", "error", err)
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
			slog.Error("ApprovePO failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"shopping_list_id": listID})
	}
}

// RepurchaseResetHandler manually triggers a repurchase badge reset (admin-only, D-17).
// POST /api/v1/purchasing/repurchase-reset
func RepurchaseResetHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
		if err := TriggerRepurchaseReset(r.Context(), pool); err != nil {
			slog.Error("RepurchaseResetHandler failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetRepurchaseResetConfigHandler returns the current badge reset schedule (admin-only).
// GET /api/v1/purchasing/repurchase-reset
func GetRepurchaseResetConfigHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
		cfg, err := GetRepurchaseResetConfig(r.Context(), pool)
		if err != nil {
			slog.Error("GetRepurchaseResetConfigHandler failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if cfg == nil {
			writeJSON(w, http.StatusOK, nil)
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

// UpsertRepurchaseResetConfigHandler sets the weekly badge reset schedule (admin-only).
// PUT /api/v1/purchasing/repurchase-reset/config
func UpsertRepurchaseResetConfigHandler(pool *pgxpool.Pool) http.HandlerFunc {
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
		var input struct {
			DayOfWeek int    `json:"day_of_week"`
			ResetTime string `json:"reset_time"` // HH:MM
			Timezone  string `json:"timezone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if input.DayOfWeek < 0 || input.DayOfWeek > 6 {
			writeError(w, http.StatusBadRequest, "invalid_day_of_week")
			return
		}
		if input.ResetTime == "" {
			input.ResetTime = "06:00"
		}
		if input.Timezone == "" {
			input.Timezone = "America/New_York"
		}
		cfg, err := UpsertRepurchaseResetConfig(r.Context(), pool, input.DayOfWeek, input.ResetTime, input.Timezone)
		if err != nil {
			slog.Error("UpsertRepurchaseResetConfigHandler failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}
