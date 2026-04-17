package sync

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrConflict is returned when an incoming op loses the LWW check.
var ErrConflict = errors.New("lww conflict: incoming op is stale")

// Op type constants.
const (
	OpSetField        = "SET_FIELD"
	OpSubmitChecklist = "SUBMIT_CHECKLIST"
	OpApproveItem     = "APPROVE_ITEM"
	OpRejectItem      = "REJECT_ITEM"
	OpSaveTemplate    = "SAVE_TEMPLATE"
	OpArchiveTemplate = "ARCHIVE_TEMPLATE"
)

// OpInput describes an operation to be recorded.
type OpInput struct {
	DeviceID   string          `json:"device_id"`
	UserID     string          `json:"user_id"`
	EntityID   string          `json:"entity_id"`
	EntityType string          `json:"entity_type"`
	OpType     string          `json:"op_type"`
	Payload    json.RawMessage `json:"payload"`
	LamportTS  int64           `json:"lamport_ts"`
}

// Op is a recorded operation row from the ops table.
type Op struct {
	ID         string          `json:"id"`
	DeviceID   string          `json:"device_id"`
	UserID     string          `json:"user_id"`
	EntityID   string          `json:"entity_id"`
	EntityType string          `json:"entity_type"`
	OpType     string          `json:"op_type"`
	Payload    json.RawMessage `json:"payload"`
	LamportTS  int64           `json:"lamport_ts"`
	ServerTS   time.Time       `json:"server_ts"`
	Applied    bool            `json:"applied"`
}

// ConflictResult is returned when an op loses the LWW check.
type ConflictResult struct {
	WinnerLamportTS int64           `json:"lamport_ts"`
	WinnerPayload   json.RawMessage `json:"payload"`
	WinnerDeviceID  string          `json:"device_id"`
}

// CheckLWW reads the current lamport_ts for the entity and compares it to the
// incoming op. Returns nil if the incoming op wins, or a ConflictResult if it
// is stale (D-10, D-12). The function runs inside the provided transaction so
// the read is part of the same write transaction.
func CheckLWW(ctx context.Context, tx pgx.Tx, entityID, entityType string, incomingTS int64, incomingDeviceID string) (*ConflictResult, error) {
	// Build the query based on entity type.
	var query string
	switch entityType {
	case "field_response":
		query = `SELECT lamport_ts FROM submission_responses WHERE field_id = $1 LIMIT 1`
	case "submission":
		query = `SELECT lamport_ts FROM checklist_submissions WHERE id = $1`
	case "template":
		query = `SELECT lamport_ts FROM checklist_templates WHERE id = $1`
	default:
		// Unknown entity type — allow the write (no guard possible).
		return nil, nil
	}

	var currentTS int64
	err := tx.QueryRow(ctx, query, entityID).Scan(&currentTS)
	if errors.Is(err, pgx.ErrNoRows) {
		// First write — no conflict.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Incoming wins outright.
	if incomingTS > currentTS {
		return nil, nil
	}

	// Equal timestamp — tiebreak by device_id lexicographic order.
	if incomingTS == currentTS {
		// We need the current winner's device_id to tiebreak.
		// Fetch from most recent op row for this entity at the tied timestamp.
		var winnerDeviceID string
		var winnerPayload json.RawMessage
		tieRow := tx.QueryRow(ctx,
			`SELECT payload, device_id FROM ops WHERE entity_id = $1 AND lamport_ts = $2 ORDER BY server_ts DESC LIMIT 1`,
			entityID, currentTS,
		)
		if scanErr := tieRow.Scan(&winnerPayload, &winnerDeviceID); scanErr != nil {
			// No prior op row for this entity at this timestamp — incoming wins.
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, scanErr
		}
		// Higher device_id wins.
		if incomingDeviceID > winnerDeviceID {
			return nil, nil
		}
		return &ConflictResult{
			WinnerLamportTS: currentTS,
			WinnerPayload:   winnerPayload,
			WinnerDeviceID:  winnerDeviceID,
		}, nil
	}

	// incomingTS < currentTS — stale, fetch winner state.
	var winnerPayload json.RawMessage
	var winnerDeviceID string
	row := tx.QueryRow(ctx,
		`SELECT payload, device_id FROM ops WHERE entity_id = $1 AND lamport_ts = $2 ORDER BY server_ts DESC LIMIT 1`,
		entityID, currentTS,
	)
	if scanErr := row.Scan(&winnerPayload, &winnerDeviceID); scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			// No prior op — let the write proceed (defensive fallback).
			return nil, nil
		}
		return nil, scanErr
	}
	return &ConflictResult{
		WinnerLamportTS: currentTS,
		WinnerPayload:   winnerPayload,
		WinnerDeviceID:  winnerDeviceID,
	}, nil
}

// updateEntityLamportTS updates the entity table lamport_ts column within the
// current transaction.
func updateEntityLamportTS(ctx context.Context, tx pgx.Tx, entityID, entityType string, lamportTS int64) error {
	var query string
	switch entityType {
	case "field_response":
		query = `UPDATE submission_responses SET lamport_ts = $1 WHERE field_id = $2`
	case "submission":
		query = `UPDATE checklist_submissions SET lamport_ts = $1 WHERE id = $2`
	case "template":
		query = `UPDATE checklist_templates SET lamport_ts = $1 WHERE id = $2`
	default:
		// Unknown type — skip silently (no entity table to update).
		return nil
	}
	_, err := tx.Exec(ctx, query, lamportTS, entityID)
	return err
}

// InsertOpAndNotify opens a transaction, runs the LWW guard, inserts the op
// row, updates the entity's lamport_ts, and fires pg_notify. Returns the new
// op ID on success, or ("", conflictResult, ErrConflict) if the guard fires.
func InsertOpAndNotify(ctx context.Context, pool *pgxpool.Pool, op OpInput) (string, *ConflictResult, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	conflict, err := CheckLWW(ctx, tx, op.EntityID, op.EntityType, op.LamportTS, op.DeviceID)
	if err != nil {
		return "", nil, err
	}
	if conflict != nil {
		return "", conflict, ErrConflict
	}

	// Update entity lamport_ts.
	if err := updateEntityLamportTS(ctx, tx, op.EntityID, op.EntityType, op.LamportTS); err != nil {
		return "", nil, err
	}

	// Insert op row.
	var opID string
	err = tx.QueryRow(ctx,
		`INSERT INTO ops (device_id, user_id, entity_id, entity_type, op_type, payload, lamport_ts)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		op.DeviceID, op.UserID, op.EntityID, op.EntityType, op.OpType, op.Payload, op.LamportTS,
	).Scan(&opID)
	if err != nil {
		return "", nil, err
	}

	// Fire pg_notify within the same transaction.
	notifyPayload, _ := json.Marshal(map[string]string{
		"op_id":       opID,
		"entity_id":   op.EntityID,
		"entity_type": op.EntityType,
		"user_id":     op.UserID,
	})
	if _, err := tx.Exec(ctx,
		`SELECT pg_notify('ops_channel', $1)`,
		string(notifyPayload),
	); err != nil {
		return "", nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", nil, err
	}
	return opID, nil, nil
}

// EmitOp fires InsertOpAndNotify in a goroutine with a 5-second timeout.
// Errors are logged but not propagated. Use this for server-side emission
// after a handler has already written successfully (no conflict expected).
func EmitOp(pool *pgxpool.Pool, op OpInput) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, conflict, err := InsertOpAndNotify(ctx, pool, op)
		if err != nil {
			if errors.Is(err, ErrConflict) {
				log.Printf("EmitOp: LWW conflict on entity %s (winner lamport_ts=%d device=%s)",
					op.EntityID, conflict.WinnerLamportTS, conflict.WinnerDeviceID)
				return
			}
			log.Printf("EmitOp: error inserting op for entity %s: %v", op.EntityID, err)
		}
	}()
}

// EmitOpWithConflictCheck is the synchronous variant of EmitOp. Used by
// handlers that need to return 409 to the client when an op is stale.
func EmitOpWithConflictCheck(ctx context.Context, pool *pgxpool.Pool, op OpInput) (string, *ConflictResult, error) {
	return InsertOpAndNotify(ctx, pool, op)
}

// OpsSince returns all ops accessible to userID with lamport_ts > lamportTS,
// ordered ascending. Access is resolved via template_assignments so that a
// user receives ops for any entity linked to templates they are assigned to
// (D-09).
func OpsSince(ctx context.Context, pool *pgxpool.Pool, userID string, lamportTS int64) ([]Op, error) {
	rows, err := pool.Query(ctx,
		`SELECT o.id, o.device_id, o.user_id, o.entity_id, o.entity_type,
		        o.op_type, o.payload, o.lamport_ts, o.server_ts, o.applied
		 FROM ops o
		 WHERE o.lamport_ts > $2
		   AND (
		         o.user_id = $1
		         OR o.entity_id IN (
		               SELECT template_id::uuid
		               FROM template_assignments
		               WHERE user_id = $1::uuid
		         )
		       )
		 ORDER BY o.lamport_ts ASC`,
		userID, lamportTS,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []Op
	for rows.Next() {
		var o Op
		if err := rows.Scan(
			&o.ID, &o.DeviceID, &o.UserID, &o.EntityID, &o.EntityType,
			&o.OpType, &o.Payload, &o.LamportTS, &o.ServerTS, &o.Applied,
		); err != nil {
			return nil, err
		}
		ops = append(ops, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if ops == nil {
		ops = []Op{}
	}
	return ops, nil
}

// GetOpByID fetches a single op by primary key. Used by the PG listener after
// receiving a NOTIFY with just the op_id.
func GetOpByID(ctx context.Context, pool *pgxpool.Pool, opID string) (*Op, error) {
	var o Op
	err := pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, entity_id, entity_type,
		        op_type, payload, lamport_ts, server_ts, applied
		 FROM ops WHERE id = $1`,
		opID,
	).Scan(
		&o.ID, &o.DeviceID, &o.UserID, &o.EntityID, &o.EntityType,
		&o.OpType, &o.Payload, &o.LamportTS, &o.ServerTS, &o.Applied,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ResolveEntityAccess returns the user IDs who should receive an op for the
// given entity via WebSocket. For field_response and submission entities the
// submission's template_id is looked up first, then template_assignments is
// queried. For template entities, template_assignments is queried directly.
func ResolveEntityAccess(ctx context.Context, pool *pgxpool.Pool, entityID, entityType string) ([]string, error) {
	var templateID string

	switch entityType {
	case "template":
		templateID = entityID

	case "field_response":
		// Look up submission_id from submission_responses, then template_id.
		var submissionID string
		err := pool.QueryRow(ctx,
			`SELECT submission_id FROM submission_responses WHERE field_id = $1 LIMIT 1`,
			entityID,
		).Scan(&submissionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return []string{}, nil
			}
			return nil, err
		}
		err = pool.QueryRow(ctx,
			`SELECT template_id FROM checklist_submissions WHERE id = $1`,
			submissionID,
		).Scan(&templateID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return []string{}, nil
			}
			return nil, err
		}

	case "submission":
		err := pool.QueryRow(ctx,
			`SELECT template_id FROM checklist_submissions WHERE id = $1`,
			entityID,
		).Scan(&templateID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return []string{}, nil
			}
			return nil, err
		}

	default:
		return []string{}, nil
	}

	rows, err := pool.Query(ctx,
		`SELECT DISTINCT user_id::text FROM template_assignments WHERE template_id = $1::uuid`,
		templateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		if _, ok := seen[uid]; !ok {
			seen[uid] = struct{}{}
			userIDs = append(userIDs, uid)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if userIDs == nil {
		userIDs = []string{}
	}
	return userIDs, nil
}
