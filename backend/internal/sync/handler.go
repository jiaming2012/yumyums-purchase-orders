package sync

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// WsHandler upgrades HTTP connections to WebSocket, registers the client in
// the hub, and runs the read/write pumps for the connection lifetime.
func WsHandler(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Allow same-origin connections from any host (dev: localhost, LAN IP, Tailscale).
			// Auth middleware already validates the session cookie — origin check is redundant.
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws: accept error for user %s: %v", user.ID, err)
			return
		}

		client := &Client{
			UserID: user.ID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		hub.register <- client
		defer func() { hub.unregister <- client }()

		// Write pump — reads from client.Send channel and writes to WebSocket.
		go func() {
			ctx := context.Background()
			for msg := range client.Send {
				if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
					log.Printf("ws: write error for user %s: %v", user.ID, err)
					conn.Close(websocket.StatusInternalError, "write error")
					return
				}
			}
			// Channel closed by hub — send normal close.
			conn.Close(websocket.StatusNormalClosure, "")
		}()

		// Read loop — keeps the connection alive and handles client messages.
		// When the request context is cancelled (client disconnects), Read returns
		// an error which triggers cleanup via the deferred unregister.
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				// Normal disconnect or protocol error — stop reading.
				return
			}
			// Future: handle ack / protocol messages from client here.
		}
	}
}

// OpRequest is the JSON body for POST /ops.
type OpRequest struct {
	OpType     string          `json:"op_type"`
	EntityID   string          `json:"entity_id"`
	EntityType string          `json:"entity_type"`
	Payload    json.RawMessage `json:"payload"`
	LamportTS  int64           `json:"lamport_ts"`
	DeviceID   string          `json:"device_id"`
}

// writeJSONError writes a JSON error response with the given status code and message.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

// RouteOpResult is the result of routing an op through business logic.
// EntityID may be updated by the router (e.g. SUBMIT_CHECKLIST returns a new ID).
type RouteOpResult struct {
	EntityID string // may differ from request EntityID (e.g. new submission ID)
}

// OpRouter is a function that applies the business logic for an op.
// It is called by OpHandler before InsertOpAndNotify. The router is responsible
// for all validation, idempotency checks, and DB writes. It returns the
// (possibly updated) entity ID, or an OpRouterError if the op should be rejected.
// Returning nil means the op was applied successfully.
type OpRouter func(ctx context.Context, userID string, req OpRequest) (*RouteOpResult, error)

// OpRouterError is an error returned by OpRouter with an HTTP status code.
type OpRouterError struct {
	Status  int
	Message string
}

func (e *OpRouterError) Error() string { return e.Message }

// OpHandler receives client ops, routes by op_type to existing business logic
// via the provided router, then records the op via InsertOpAndNotify (D-11).
// Business logic (fail note validation, idempotency, archive checks) is
// preserved inside the router per D-14. The router is injected to avoid a
// circular import between the sync and workflow packages.
func OpHandler(pool *pgxpool.Pool, router OpRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req OpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_body"}`, http.StatusBadRequest)
			return
		}

		if req.OpType == "" || req.DeviceID == "" {
			http.Error(w, `{"error":"missing op_type or device_id"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// Route by op_type to existing business logic (D-11).
		result, err := router(ctx, user.ID, req)
		if err != nil {
			var routerErr *OpRouterError
			if errors.As(err, &routerErr) {
				writeJSONError(w, routerErr.Status, routerErr.Message)
				return
			}
			log.Printf("OpHandler router error: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if result != nil && result.EntityID != "" {
			req.EntityID = result.EntityID
		}

		// Record op and notify (D-11) — use client's device_id and lamport_ts.
		opInput := OpInput{
			DeviceID:   req.DeviceID,
			UserID:     user.ID,
			EntityID:   req.EntityID,
			EntityType: req.EntityType,
			OpType:     req.OpType,
			Payload:    req.Payload,
			LamportTS:  req.LamportTS,
		}

		opID, conflict, err := EmitOpWithConflictCheck(ctx, pool, opInput)
		if err != nil {
			if errors.Is(err, ErrConflict) && conflict != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
					"_conflict": true,
					"winner": map[string]any{
						"lamport_ts": conflict.WinnerLamportTS,
						"payload":    conflict.WinnerPayload,
						"device_id":  conflict.WinnerDeviceID,
					},
				})
				return
			}
			// Business logic succeeded but op recording failed — log and return success.
			// The entity tables are consistent; only the op log missed an entry.
			log.Printf("OpHandler InsertOpAndNotify error: %v", err)
			opID = ""
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"op_id": opID}) //nolint:errcheck
	}
}

// OpsSinceHandler returns ops accessible to the authenticated user with
// lamport_ts greater than the provided query parameter.
func OpsSinceHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var lamportTS int64
		if raw := r.URL.Query().Get("lamport_ts"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err == nil {
				lamportTS = parsed
			}
		}

		ops, err := OpsSince(r.Context(), pool, user.ID, lamportTS)
		if err != nil {
			log.Printf("OpsSince error for user %s: %v", user.ID, err)
			http.Error(w, `{"error":"internal_error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ops); err != nil {
			log.Printf("OpsSince encode error: %v", err)
		}
	}
}
