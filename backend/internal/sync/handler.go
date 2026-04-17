package sync

import (
	"context"
	"encoding/json"
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
			// InsecureSkipVerify is false — origin checks are enabled.
			// OriginPatterns can be set here for production hardening.
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
