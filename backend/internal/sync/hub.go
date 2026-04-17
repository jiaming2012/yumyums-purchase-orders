package sync

import (
	"log"

	"github.com/coder/websocket"
)

// Client represents a connected WebSocket client.
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

// BroadcastMsg targets a message at a set of user IDs.
type BroadcastMsg struct {
	UserIDs []string
	Data    []byte
}

// Hub maintains the set of active clients and broadcasts messages to them.
// A single goroutine owns the client map — no mutex required (D-06, D-09).
type Hub struct {
	// clients maps userID -> slice of active connections for that user.
	clients    map[string][]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan BroadcastMsg
}

// NewHub creates a new Hub ready to Run.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string][]*Client),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan BroadcastMsg, 256),
	}
}

// Run processes register, unregister, and broadcast events. Must be called in
// a goroutine — it blocks until the process exits.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c.UserID] = append(h.clients[c.UserID], c)

		case c := <-h.unregister:
			conns := h.clients[c.UserID]
			updated := conns[:0]
			for _, existing := range conns {
				if existing != c {
					updated = append(updated, existing)
				}
			}
			if len(updated) == 0 {
				delete(h.clients, c.UserID)
			} else {
				h.clients[c.UserID] = updated
			}
			close(c.Send)

		case msg := <-h.broadcast:
			for _, uid := range msg.UserIDs {
				conns, ok := h.clients[uid]
				if !ok {
					continue
				}
				active := conns[:0]
				for _, c := range conns {
					select {
					case c.Send <- msg.Data:
						active = append(active, c)
					default:
						// Send buffer full — disconnect this client.
						log.Printf("hub: dropping slow client for user %s", uid)
						close(c.Send)
					}
				}
				if len(active) == 0 {
					delete(h.clients, uid)
				} else {
					h.clients[uid] = active
				}
			}
		}
	}
}
