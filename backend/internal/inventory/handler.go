package inventory

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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
			ORDER BY total_spend DESC`,
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
			_, err := tx.Exec(r.Context(), `
				INSERT INTO purchase_line_items
				(purchase_event_id, purchase_item_id, description, quantity, price, is_case)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				eventID, li.PurchaseItemID, li.Description, li.Quantity, li.Price, li.IsCase,
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

		// Fetch the pending purchase to get bank_tx_id
		var bankTxID string
		err = tx.QueryRow(r.Context(),
			`SELECT bank_tx_id FROM pending_purchases WHERE id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
			input.ID,
		).Scan(&bankTxID)
		if err != nil {
			writeError(w, http.StatusNotFound, "pending_purchase_not_found")
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
			_, err := tx.Exec(r.Context(), `
				INSERT INTO purchase_line_items
				(purchase_event_id, purchase_item_id, description, quantity, price, is_case)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				eventID, li.PurchaseItemID, li.Description, li.Quantity, li.Price, li.IsCase,
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
