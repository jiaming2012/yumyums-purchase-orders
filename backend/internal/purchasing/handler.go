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

		if err := UpsertLineItems(r.Context(), pool, id, user.ID, req.Items); err != nil {
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
