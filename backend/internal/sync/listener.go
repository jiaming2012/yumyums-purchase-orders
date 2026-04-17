package sync

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
)

// notifyPayload is the JSON shape emitted by pg_notify inside InsertOpAndNotify.
type notifyPayload struct {
	OpID       string `json:"op_id"`
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	UserID     string `json:"user_id"`
}

// StartListener opens a dedicated pgx connection for LISTEN, subscribes to
// ops_channel, and fans out incoming ops to connected WebSocket clients.
// The function starts the listener in a goroutine and returns immediately.
// connStr must be the raw DATABASE_URL connection string (not a pool URL).
func StartListener(ctx context.Context, connStr string, hub *Hub, pool *pgxpool.Pool) {
	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			return pgx.Connect(ctx, connStr)
		},
		ReconnectDelay: 5 * time.Second,
		LogError: func(ctx context.Context, err error) {
			log.Printf("pgxlisten: %v", err)
		},
	}

	listener.Handle("ops_channel", pgxlisten.HandlerFunc(func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
		var n notifyPayload
		if err := json.Unmarshal([]byte(notification.Payload), &n); err != nil {
			log.Printf("listener: bad notify payload: %v", err)
			return nil
		}

		// Fetch full op row from the pool (not the LISTEN-dedicated conn).
		op, err := GetOpByID(ctx, pool, n.OpID)
		if err != nil {
			log.Printf("listener: GetOpByID %s: %v", n.OpID, err)
			return nil
		}

		// Resolve which users should receive this op.
		userIDs, err := ResolveEntityAccess(ctx, pool, n.EntityID, n.EntityType)
		if err != nil {
			log.Printf("listener: ResolveEntityAccess entity=%s type=%s: %v", n.EntityID, n.EntityType, err)
			return nil
		}

		if len(userIDs) == 0 {
			// Always include the op author even if no template_assignments found.
			userIDs = []string{op.UserID}
		}

		data, err := json.Marshal(op)
		if err != nil {
			log.Printf("listener: marshal op %s: %v", op.ID, err)
			return nil
		}

		hub.broadcast <- BroadcastMsg{
			UserIDs: userIDs,
			Data:    data,
		}
		return nil
	}))

	go func() {
		if err := listener.Listen(ctx); err != nil && ctx.Err() == nil {
			log.Printf("pgxlisten exited unexpectedly: %v", err)
		}
	}()
}
