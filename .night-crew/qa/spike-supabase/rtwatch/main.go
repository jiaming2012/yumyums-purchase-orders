// Command rtwatch subscribes to a self-hosted Supabase Realtime channel with a
// Go-minted HS256 token and prints every frame it receives until it sees a
// postgres_changes event (or times out).
//
// It exists to answer one question the spike is gated on: does self-hosted
// Realtime accept a token HQ's Go backend minted itself, with no GoTrue in the
// picture, and actually deliver a row change over it?
//
// Two things about this client are worth understanding before reading the code:
//
//  1. TENANT ROUTING IS BY Host HEADER. Self-hosted Realtime is multi-tenant
//     even with one tenant. It takes the FIRST dot-separated label of the HTTP
//     Host header as the tenant's external_id. Dial `ws://localhost:PORT/...`
//     and the tenant is "localhost", which does not exist, and you get a bare
//     403 with no explanation. The -host flag sets the virtual host used in the
//     Host header and TLS/HTTP routing while -addr says where to actually
//     connect the TCP socket — that split is the whole trick, and it is done
//     with a custom http.Transport.DialContext below.
//
//  2. The wire protocol is Phoenix channels, vsn=1.0.0 (the object form the
//     supabase-js client uses), not a bespoke Supabase protocol.
//
// The websocket library is github.com/coder/websocket v1.8.14 — the exact
// version HQ's backend/go.mod already lists as a direct dependency. No new
// third-party surface enters the repo.
//
//	go run ./rtwatch -addr 127.0.0.1:46355 -host realtime-dev.localhost \
//	                 -token "$TOKEN" -table spike_notes
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
)

type phxMsg struct {
	Topic   string          `json:"topic"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
	Ref     string          `json:"ref"`
	JoinRef string          `json:"join_ref,omitempty"`
}

func main() {
	addr := flag.String("addr", "", "real TCP host:port of the realtime container's published port")
	vhost := flag.String("host", "realtime-dev.localhost", "virtual host; its FIRST label must equal the tenant external_id")
	token := flag.String("token", "", "HS256 JWT minted by ./mintjwt with the stack's JWT_SECRET")
	table := flag.String("table", "spike_notes", "table to subscribe to")
	schema := flag.String("schema", "public", "schema to subscribe to")
	timeout := flag.Duration("timeout", 60*time.Second, "give up after this long")
	flag.Parse()

	if *addr == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "rtwatch: -addr and -token are required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Connect the socket to -addr but present -host in the Host header, so
	// Realtime's tenant lookup sees "realtime-dev" while TCP goes to 127.0.0.1.
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, *addr)
			},
		},
	}

	url := fmt.Sprintf("ws://%s/socket/websocket?apikey=%s&vsn=1.0.0", *vhost, *token)
	conn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: httpClient})
	if err != nil {
		if resp != nil {
			fmt.Fprintf(os.Stderr, "rtwatch: handshake failed: %v (HTTP %s)\n", err, resp.Status)
		} else {
			fmt.Fprintln(os.Stderr, "rtwatch: handshake failed:", err)
		}
		os.Exit(1)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")
	fmt.Printf("CONNECTED ws://%s -> %s (HTTP %s)\n", *vhost, *addr, resp.Status)

	topic := "realtime:spike"
	join := phxMsg{
		Topic: topic,
		Event: "phx_join",
		Ref:   "1", JoinRef: "1",
		Payload: json.RawMessage(fmt.Sprintf(`{
			"config": {
				"broadcast": {"ack": false, "self": false},
				"presence": {"key": ""},
				"private": false,
				"postgres_changes": [{"event": "*", "schema": %q, "table": %q}]
			},
			"access_token": %q
		}`, *schema, *table, *token)),
	}
	if err := writeJSON(ctx, conn, join); err != nil {
		fmt.Fprintln(os.Stderr, "rtwatch: join write failed:", err)
		os.Exit(1)
	}
	fmt.Printf("SENT phx_join topic=%s postgres_changes=%s.%s\n", topic, *schema, *table)

	// Phoenix drops a socket that stops heart-beating.
	go func() {
		t := time.NewTicker(25 * time.Second)
		defer t.Stop()
		n := 100
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n++
				_ = writeJSON(ctx, conn, phxMsg{
					Topic: "phoenix", Event: "heartbeat",
					Payload: json.RawMessage(`{}`), Ref: fmt.Sprint(n),
				})
			}
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "TIMEOUT/READ-ERR:", err)
			os.Exit(1)
		}
		var m phxMsg
		if err := json.Unmarshal(data, &m); err != nil {
			fmt.Println("RECV (unparsed):", string(data))
			continue
		}
		fmt.Printf("RECV event=%-18s topic=%-16s payload=%s\n", m.Event, m.Topic, string(m.Payload))
		if m.Event == "postgres_changes" {
			fmt.Println("OK: postgres_changes received — Realtime delivered a row change over a Go-minted token.")
			return
		}
	}
}

func writeJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}
