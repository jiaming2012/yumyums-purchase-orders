package inventory

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// normalizeItemName title-cases receipt text: "DEER PARK 40PK" → "Deer Park 40Pk".
func normalizeItemName(s string) string {
	return cases.Title(language.English).String(strings.ToLower(strings.TrimSpace(s)))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ListVendorsHandler returns all vendors ordered by name.
func ListVendorsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(),
			`SELECT id, name, created_at FROM vendors ORDER BY name`,
		)
		if err != nil {
			log.Printf("ListVendors query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		vendors := []Vendor{}
		for rows.Next() {
			var v Vendor
			if err := rows.Scan(&v.ID, &v.Name, &v.CreatedAt); err != nil {
				log.Printf("ListVendors scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			vendors = append(vendors, v)
		}
		writeJSON(w, http.StatusOK, vendors)
	}
}

// CreateVendorHandler creates a new vendor.
func CreateVendorHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.Name == "" {
			writeError(w, http.StatusBadRequest, "name_required")
			return
		}
		input.Name = normalizeItemName(input.Name)
		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO vendors (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, input.Name,
		).Scan(&id)
		if err != nil {
			log.Printf("CreateVendor insert: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// UpdateVendorHandler updates a vendor's name.
func UpdateVendorHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" || input.Name == "" {
			writeError(w, http.StatusBadRequest, "id_and_name_required")
			return
		}
		input.Name = normalizeItemName(input.Name)
		tag, err := pool.Exec(r.Context(), `UPDATE vendors SET name = $1 WHERE id = $2`, input.Name, input.ID)
		if err != nil {
			log.Printf("UpdateVendor update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "vendor_not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// MergeVendorsHandler merges source vendor into target: re-points all purchase_events, then deletes source.
func MergeVendorsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.SourceID == "" || input.TargetID == "" {
			writeError(w, http.StatusBadRequest, "source_id_and_target_id_required")
			return
		}
		if input.SourceID == input.TargetID {
			writeError(w, http.StatusBadRequest, "cannot_merge_into_self")
			return
		}
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		// Re-point purchase_events from source to target
		_, err = tx.Exec(r.Context(), `UPDATE purchase_events SET vendor_id = $1 WHERE vendor_id = $2`, input.TargetID, input.SourceID)
		if err != nil {
			log.Printf("MergeVendors re-point events: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		// Delete source vendor
		tag, err := tx.Exec(r.Context(), `DELETE FROM vendors WHERE id = $1`, input.SourceID)
		if err != nil {
			log.Printf("MergeVendors delete source: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "source_vendor_not_found")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// MergeItemsHandler merges source item into target: re-points all purchase_line_items, then deletes source.
func MergeItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.SourceID == "" || input.TargetID == "" {
			writeError(w, http.StatusBadRequest, "source_id_and_target_id_required")
			return
		}
		if input.SourceID == input.TargetID {
			writeError(w, http.StatusBadRequest, "cannot_merge_into_self")
			return
		}
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		// Re-point purchase_line_items from source to target
		_, err = tx.Exec(r.Context(), `UPDATE purchase_line_items SET purchase_item_id = $1 WHERE purchase_item_id = $2`, input.TargetID, input.SourceID)
		if err != nil {
			log.Printf("MergeItems re-point line_items: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		// Also update description to match target
		var targetDesc string
		err = tx.QueryRow(r.Context(), `SELECT description FROM purchase_items WHERE id = $1`, input.TargetID).Scan(&targetDesc)
		if err != nil {
			writeError(w, http.StatusNotFound, "target_item_not_found")
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE purchase_line_items SET description = $1 WHERE purchase_item_id = $2`, targetDesc, input.TargetID)
		if err != nil {
			log.Printf("MergeItems update descriptions: %v", err)
		}
		// Delete source item
		tag, err := tx.Exec(r.Context(), `DELETE FROM purchase_items WHERE id = $1`, input.SourceID)
		if err != nil {
			log.Printf("MergeItems delete source: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "source_item_not_found")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdatePendingItemsHandler updates the items JSONB on a pending purchase (persists item selections).
func UpdatePendingItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID    string          `json:"id"`
			Items json.RawMessage `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" {
			writeError(w, http.StatusBadRequest, "id_required")
			return
		}
		tag, err := pool.Exec(r.Context(),
			`UPDATE pending_purchases SET items = $1 WHERE id = $2 AND confirmed_at IS NULL AND discarded_at IS NULL`,
			input.Items, input.ID,
		)
		if err != nil {
			log.Printf("UpdatePendingItems: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "pending_purchase_not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListPurchaseEventsHandler returns purchase events with nested line items.
// Accepts optional ?vendor_id and ?page query params (LIMIT 50 per page).
func ListPurchaseEventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vendorID := r.URL.Query().Get("vendor_id")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * 50

		var (
			rows pgx.Rows
			err  error
		)
		if vendorID != "" {
			rows, err = pool.Query(r.Context(), `
				SELECT pe.id, pe.vendor_id, v.name, pe.bank_tx_id,
				       pe.event_date::text, pe.tax, pe.total, pe.receipt_url, pe.created_at
				FROM purchase_events pe
				JOIN vendors v ON v.id = pe.vendor_id
				WHERE pe.vendor_id = $1
				ORDER BY pe.event_date DESC
				LIMIT 50 OFFSET $2`,
				vendorID, offset,
			)
		} else {
			rows, err = pool.Query(r.Context(), `
				SELECT pe.id, pe.vendor_id, v.name, pe.bank_tx_id,
				       pe.event_date::text, pe.tax, pe.total, pe.receipt_url, pe.created_at
				FROM purchase_events pe
				JOIN vendors v ON v.id = pe.vendor_id
				ORDER BY pe.event_date DESC
				LIMIT 50 OFFSET $1`,
				offset,
			)
		}
		if err != nil {
			log.Printf("ListPurchaseEvents query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		events := []PurchaseEvent{}
		for rows.Next() {
			var pe PurchaseEvent
			if err := rows.Scan(&pe.ID, &pe.VendorID, &pe.VendorName, &pe.BankTxID,
				&pe.EventDate, &pe.Tax, &pe.Total, &pe.ReceiptURL, &pe.CreatedAt); err != nil {
				log.Printf("ListPurchaseEvents scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			events = append(events, pe)
		}
		if err := rows.Err(); err != nil {
			log.Printf("ListPurchaseEvents rows err: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Load line items for each event
		for i := range events {
			liRows, err := pool.Query(r.Context(), `
				SELECT id, purchase_event_id, purchase_item_id,
				       description, quantity, price, is_case
				FROM purchase_line_items
				WHERE purchase_event_id = $1`,
				events[i].ID,
			)
			if err != nil {
				log.Printf("ListPurchaseEvents line_items query: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			defer liRows.Close()
			events[i].LineItems = []LineItem{}
			for liRows.Next() {
				var li LineItem
				if err := liRows.Scan(&li.ID, &li.PurchaseEventID, &li.PurchaseItemID,
					&li.Description, &li.Quantity, &li.Price, &li.IsCase); err != nil {
					log.Printf("ListPurchaseEvents line_item scan: %v", err)
					writeError(w, http.StatusInternalServerError, "internal_error")
					return
				}
				events[i].LineItems = append(events[i].LineItems, li)
			}
		}

		writeJSON(w, http.StatusOK, events)
	}
}

// GetStockHandler returns aggregated stock levels across all purchase items.
func GetStockHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT
				sub.description,
				sub.group_name,
				COALESCE(sco.quantity, sub.total_quantity) AS total_quantity,
				sub.total_spend,
				sub.avg_price,
				sub.last_purchase_date
			FROM (
				SELECT
					COALESCE(pi.description, pli.description) AS description,
					ig.name AS group_name,
					SUM(pli.quantity) AS total_quantity,
					SUM(pli.quantity * pli.price) AS total_spend,
					AVG(pli.price) AS avg_price,
					MAX(pe.event_date)::text AS last_purchase_date
				FROM purchase_line_items pli
				JOIN purchase_events pe ON pe.id = pli.purchase_event_id
				LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
				LEFT JOIN item_groups ig ON ig.id = pi.group_id
				GROUP BY COALESCE(pi.description, pli.description), ig.name
			) sub
			LEFT JOIN stock_count_overrides sco ON sco.item_description = sub.description
			ORDER BY sub.total_spend DESC`,
		)
		if err != nil {
			log.Printf("GetStock query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		items := []StockItem{}
		for rows.Next() {
			var s StockItem
			if err := rows.Scan(&s.Description, &s.GroupName,
				&s.TotalQuantity, &s.TotalSpend, &s.AvgPrice, &s.LastPurchaseDate); err != nil {
				log.Printf("GetStock scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			items = append(items, s)
		}
		writeJSON(w, http.StatusOK, items)
	}
}

// UpdateStockCountHandler upserts a stock count override for an item.
func UpdateStockCountHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ItemDescription string `json:"item_description"`
			Quantity        int    `json:"quantity"`
			Reason          string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ItemDescription == "" {
			writeError(w, http.StatusBadRequest, "item_description_required")
			return
		}
		if input.Quantity < 0 {
			writeError(w, http.StatusBadRequest, "quantity_must_be_positive")
			return
		}
		if strings.TrimSpace(input.Reason) == "" {
			writeError(w, http.StatusBadRequest, "reason_required")
			return
		}
		_, err := pool.Exec(r.Context(), `
			INSERT INTO stock_count_overrides (item_description, quantity, reason, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (item_description) DO UPDATE SET quantity = $2, reason = $3, updated_at = now()`,
			input.ItemDescription, input.Quantity, input.Reason,
		)
		if err != nil {
			log.Printf("UpdateStockCount: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// CreatePurchaseEventHandler creates a purchase event with its line items in a transaction.
func CreatePurchaseEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = auth.UserFromContext(r.Context()) // require auth

		var input CreatePurchaseEventInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.VendorID == "" || input.BankTxID == "" || input.EventDate == "" {
			writeError(w, http.StatusBadRequest, "missing_required_fields")
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			log.Printf("CreatePurchaseEvent begin tx: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		var eventID string
		err = tx.QueryRow(r.Context(), `
			INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, receipt_url)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			input.VendorID, input.BankTxID, input.EventDate, input.Tax, input.Total, input.ReceiptURL,
		).Scan(&eventID)
		if err != nil {
			log.Printf("CreatePurchaseEvent insert event: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		for _, li := range input.LineItems {
			desc := normalizeItemName(li.Description)
			_, err := tx.Exec(r.Context(), `
				INSERT INTO purchase_line_items
				(purchase_event_id, purchase_item_id, description, quantity, price, is_case)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				eventID, li.PurchaseItemID, desc, li.Quantity, li.Price, li.IsCase,
			)
			if err != nil {
				log.Printf("CreatePurchaseEvent insert line_item: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
		}

		if err := tx.Commit(r.Context()); err != nil {
			log.Printf("CreatePurchaseEvent commit: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": eventID})
	}
}

// ListPendingPurchasesHandler returns pending purchases that have not been confirmed or discarded.
func ListPendingPurchasesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT id, bank_tx_id, bank_total, vendor, event_date::text,
			       tax, total, total_units, total_cases, receipt_url,
			       reason, items, confirmed_at, confirmed_by, discarded_at, created_at
			FROM pending_purchases
			WHERE confirmed_at IS NULL AND discarded_at IS NULL
			ORDER BY created_at DESC`,
		)
		if err != nil {
			log.Printf("ListPendingPurchases query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		pending := []PendingPurchase{}
		for rows.Next() {
			var p PendingPurchase
			if err := rows.Scan(
				&p.ID, &p.BankTxID, &p.BankTotal, &p.Vendor, &p.EventDate,
				&p.Tax, &p.Total, &p.TotalUnits, &p.TotalCases, &p.ReceiptURL,
				&p.Reason, &p.Items, &p.ConfirmedAt, &p.ConfirmedBy, &p.DiscardedAt, &p.CreatedAt,
			); err != nil {
				log.Printf("ListPendingPurchases scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			pending = append(pending, p)
		}
		writeJSON(w, http.StatusOK, pending)
	}
}

// ConfirmPendingPurchaseHandler creates a real purchase event from a pending purchase.
func ConfirmPendingPurchaseHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())

		var input ConfirmPendingInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" || input.VendorName == "" || input.EventDate == "" {
			writeError(w, http.StatusBadRequest, "missing_required_fields")
			return
		}

		// Look up or create vendor
		var vendorID string
		err := pool.QueryRow(r.Context(),
			`INSERT INTO vendors (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
			input.VendorName,
		).Scan(&vendorID)
		if err != nil {
			log.Printf("ConfirmPendingPurchase upsert vendor: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			log.Printf("ConfirmPendingPurchase begin tx: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		// Fetch the pending purchase to get bank_tx_id and bank_total
		var bankTxID string
		var bankTotal float64
		err = tx.QueryRow(r.Context(),
			`SELECT bank_tx_id, bank_total FROM pending_purchases WHERE id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
			input.ID,
		).Scan(&bankTxID, &bankTotal)
		if err != nil {
			writeError(w, http.StatusNotFound, "pending_purchase_not_found")
			return
		}

		// Validate total matches bank transaction (bank_total is negative for debits)
		lineTotal := input.Tax
		for _, li := range input.LineItems {
			lineTotal += li.Price * float64(li.Quantity)
		}
		absBankTotal := bankTotal
		if absBankTotal < 0 {
			absBankTotal = -absBankTotal
		}
		if absBankTotal-lineTotal > 0.01 || lineTotal-absBankTotal > 0.01 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("total_mismatch: receipt total $%.2f does not match bank transaction $%.2f", lineTotal, absBankTotal))
			return
		}

		// Create the real purchase event
		var eventID string
		err = tx.QueryRow(r.Context(), `
			INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			vendorID, bankTxID, input.EventDate, input.Tax, input.Total,
		).Scan(&eventID)
		if err != nil {
			log.Printf("ConfirmPendingPurchase insert event: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		for _, li := range input.LineItems {
			desc := normalizeItemName(li.Description)
			_, err := tx.Exec(r.Context(), `
				INSERT INTO purchase_line_items
				(purchase_event_id, purchase_item_id, description, quantity, price, is_case)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				eventID, li.PurchaseItemID, desc, li.Quantity, li.Price, li.IsCase,
			)
			if err != nil {
				log.Printf("ConfirmPendingPurchase insert line_item: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
		}

		// Mark pending purchase as confirmed
		_, err = tx.Exec(r.Context(),
			`UPDATE pending_purchases SET confirmed_at = $1, confirmed_by = $2 WHERE id = $3`,
			time.Now().UTC(), user.ID, input.ID,
		)
		if err != nil {
			log.Printf("ConfirmPendingPurchase update confirmed_at: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			log.Printf("ConfirmPendingPurchase commit: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"id": eventID})
	}
}

// DiscardPendingPurchaseHandler marks a pending purchase as discarded.
func DiscardPendingPurchaseHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input DiscardPendingInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" {
			writeError(w, http.StatusBadRequest, "missing_id")
			return
		}

		tag, err := pool.Exec(r.Context(),
			`UPDATE pending_purchases SET discarded_at = $1
			 WHERE id = $2 AND confirmed_at IS NULL AND discarded_at IS NULL`,
			time.Now().UTC(), input.ID,
		)
		if err != nil {
			log.Printf("DiscardPendingPurchase update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "pending_purchase_not_found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SeedPendingPurchaseHandler inserts a pending purchase for testing.
func SeedPendingPurchaseHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			BankTxID   string          `json:"bank_tx_id"`
			BankTotal  float64         `json:"bank_total"`
			Vendor     string          `json:"vendor"`
			EventDate  string          `json:"event_date"`
			Reason     string          `json:"reason"`
			Items      json.RawMessage `json:"items"`
			ReceiptURL *string         `json:"receipt_url,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.Items == nil {
			input.Items = json.RawMessage(`[]`)
		}
		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, event_date, reason, items, receipt_url)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			input.BankTxID, input.BankTotal, input.Vendor, input.EventDate, input.Reason, input.Items, input.ReceiptURL,
		).Scan(&id)
		if err != nil {
			log.Printf("SeedPendingPurchase insert: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// ListItemsHandler returns all purchase items with their group name.
func ListItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT pi.id, pi.description, pi.group_id, ig.name
			FROM purchase_items pi
			LEFT JOIN item_groups ig ON ig.id = pi.group_id
			ORDER BY pi.description`)
		if err != nil {
			log.Printf("ListItems query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		items := []PurchaseItem{}
		for rows.Next() {
			var item PurchaseItem
			if err := rows.Scan(&item.ID, &item.Description, &item.GroupID, &item.GroupName); err != nil {
				log.Printf("ListItems scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, items)
	}
}

// CreateItemHandler creates a new purchase item.
func CreateItemHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Description string  `json:"description"`
			GroupID     *string `json:"group_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.Description == "" {
			writeError(w, http.StatusBadRequest, "description_required")
			return
		}
		if input.GroupID == nil || *input.GroupID == "" {
			writeError(w, http.StatusBadRequest, "group_required")
			return
		}
		input.Description = normalizeItemName(input.Description)
		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO purchase_items (description, group_id)
			VALUES ($1, $2)
			ON CONFLICT (description) DO UPDATE SET description = EXCLUDED.description
			RETURNING id`,
			input.Description, input.GroupID,
		).Scan(&id)
		if err != nil {
			log.Printf("CreateItem insert: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// UpdateItemHandler updates a purchase item's description or group.
func UpdateItemHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID          string  `json:"id"`
			Description string  `json:"description"`
			GroupID     *string `json:"group_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" || input.Description == "" {
			writeError(w, http.StatusBadRequest, "id_and_description_required")
			return
		}
		tag, err := pool.Exec(r.Context(), `
			UPDATE purchase_items SET description = $1, group_id = $2 WHERE id = $3`,
			input.Description, input.GroupID, input.ID,
		)
		if err != nil {
			log.Printf("UpdateItem update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "item_not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListGroupsHandler returns all item groups with their tags.
func ListGroupsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT id, name, par_days, low_threshold, high_threshold FROM item_groups ORDER BY name`)
		if err != nil {
			log.Printf("ListGroups query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		groups := []ItemGroup{}
		for rows.Next() {
			var g ItemGroup
			if err := rows.Scan(&g.ID, &g.Name, &g.ParDays, &g.LowThreshold, &g.HighThreshold); err != nil {
				log.Printf("ListGroups scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			groups = append(groups, g)
		}

		// Load tags for each group
		for i := range groups {
			tagRows, err := pool.Query(r.Context(), `
				SELECT t.id, t.name FROM tags t
				JOIN item_group_tags igt ON igt.tag_id = t.id
				WHERE igt.group_id = $1 ORDER BY t.name`, groups[i].ID)
			if err != nil {
				log.Printf("ListGroups tags query: %v", err)
				continue
			}
			for tagRows.Next() {
				var t Tag
				if err := tagRows.Scan(&t.ID, &t.Name); err != nil {
					continue
				}
				groups[i].Tags = append(groups[i].Tags, t)
			}
			tagRows.Close()
		}

		writeJSON(w, http.StatusOK, groups)
	}
}

// CreateGroupHandler creates a new item group.
func CreateGroupHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.Name == "" {
			writeError(w, http.StatusBadRequest, "name_required")
			return
		}
		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO item_groups (name) VALUES ($1)
			RETURNING id`, input.Name,
		).Scan(&id)
		if err != nil {
			log.Printf("CreateGroup insert: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// UpdateGroupHandler updates a group's thresholds.
func UpdateGroupHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID            string `json:"id"`
			LowThreshold  int    `json:"low_threshold"`
			HighThreshold int    `json:"high_threshold"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.ID == "" {
			writeError(w, http.StatusBadRequest, "id_required")
			return
		}
		if input.LowThreshold < 0 || input.HighThreshold < 0 {
			writeError(w, http.StatusBadRequest, "thresholds_must_be_positive")
			return
		}
		if input.LowThreshold >= input.HighThreshold {
			writeError(w, http.StatusBadRequest, "low_must_be_less_than_high")
			return
		}
		tag, err := pool.Exec(r.Context(), `
			UPDATE item_groups SET low_threshold = $1, high_threshold = $2 WHERE id = $3`,
			input.LowThreshold, input.HighThreshold, input.ID,
		)
		if err != nil {
			log.Printf("UpdateGroup update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "group_not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ListTagsHandler returns all tags.
func ListTagsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `SELECT id, name FROM tags ORDER BY name`)
		if err != nil {
			log.Printf("ListTags query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		tags := []Tag{}
		for rows.Next() {
			var t Tag
			if err := rows.Scan(&t.ID, &t.Name); err != nil {
				log.Printf("ListTags scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			tags = append(tags, t)
		}
		writeJSON(w, http.StatusOK, tags)
	}
}
