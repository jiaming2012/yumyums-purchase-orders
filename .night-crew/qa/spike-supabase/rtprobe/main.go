// Command rtprobe is SPIKE D's live-Realtime observer.
//
// ═══════════════════════════════════════════════════════════════════════════
// WHAT IT IS FOR, AND WHY IT IS NOT rtwatch
//
// `rtwatch` (the sibling directory) answers spike A's question — "does
// self-hosted Realtime accept a Go-minted token and deliver a row change over
// it at all" — and it answers it by subscribing to ONE table with NO filter and
// exiting on the FIRST postgres_changes frame. That shape is load-bearing for
// spike A's committed README proofs (R3 in particular), so it is left untouched.
//
// Spike D (card `spike-d-realtime-live`, closing B-62) needs the opposite
// shape on all three axes:
//
//   1. MANY BINDINGS AT ONCE, each with its OWN filter, on ONE socket. The
//      whole question is comparative — the same row must be shown ARRIVING on
//      an unfiltered channel and NOT arriving on a filtered one, at the same
//      instant, over the same connection. Two processes cannot testify to that;
//      a difference between them could always be a difference in timing.
//
//   2. OBSERVE A WINDOW, NEVER EXIT ON FIRST EVENT. "The out-of-scope row did
//      not arrive" is only meaningful after a bounded wait during which the
//      in-scope row DID arrive. Exiting on the first frame would make the
//      negative leg unobservable by construction.
//
//   3. REPORT, DO NOT JUDGE. Every verdict belongs to spike-d-realtime.sh's
//      exit status. This binary emits machine-readable lines and exits 0 for
//      "I observed the window", 3 for "I could not run". It has no notion of
//      pass or fail and must never grow one.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 A JOIN THAT REPLIES "ok" IS NOT A SUBSCRIPTION.
//
// Self-hosted Realtime acks `phx_join` with `{"status":"ok"}` and a
// postgres_changes id BEFORE it has established the CDC subscription. If the
// subscription then fails — table not in the publication, a filter the
// `realtime.subscription_check_filters` trigger rejects, a column the claimed
// role cannot SELECT — the failure arrives AFTERWARDS as a separate `system`
// frame:
//
//	{"status":"error","extension":"postgres_changes",
//	 "message":"Unable to subscribe to changes with given parameters..."}
//
// A client that only checks the join reply believes it is subscribed. That is
// spike A's measured finding (sql/spike-fixture.sql §3, README proof R3), and
// for THIS card it is the difference between a red verdict and a vacuous one:
// a rejected filter and an honoured filter both produce "the out-of-scope row
// did not arrive". So rtprobe joins, then SETTLES for a fixed interval
// collecting `system` frames, and reports JOIN-OK / SYS-ERR per label before it
// prints READY. The script refuses to assert anything about a label that
// carried a SYS-ERR.
//
// ═══════════════════════════════════════════════════════════════════════════
// TENANT ROUTING IS BY Host HEADER — inherited verbatim from rtwatch, because
// it is the single sharpest edge in this stack. Self-hosted Realtime is
// multi-tenant even with one tenant and takes the FIRST dot-separated label of
// the HTTP Host header as the tenant's external_id. Dial `ws://localhost:PORT/`
// and the tenant is "localhost", which does not exist, and you get a bare 403
// with no explanation. `-host` sets the virtual host used in the Host header
// while `-addr` says where to actually connect the TCP socket.
//
// Wire protocol: Phoenix channels, vsn=1.0.0, object form.
// Dependency: github.com/coder/websocket v1.8.14 — the exact version rtwatch
// uses and backend/go.mod already lists directly. Nothing new enters the repo.
//
//	go run ./rtprobe -addr 127.0.0.1:50807 -host realtime-dev.localhost \
//	  -token "$TOKEN" -window 15s \
//	  -bind 'f-gte|public|spike_d_submissions|submitted_at=gte.2026-08-06T00:00:00Z' \
//	  -bind 'u-sub|public|spike_d_submissions|'
//
// EXIT STATUS
//
//	0   the observation window completed. Says NOTHING about pass/fail.
//	3   could not run — dial/handshake failed, or the socket died before the
//	    window completed. Infrastructure, not a verdict.
//	64  usage error.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const (
	exitOK        = 0
	exitCannotRun = 3
	exitUsage     = 64
)

type phxMsg struct {
	Topic   string          `json:"topic"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
	Ref     string          `json:"ref"`
	JoinRef string          `json:"join_ref,omitempty"`
}

// binding is one `label|schema|table|filter` spec. An EMPTY filter means the
// binding is deliberately unfiltered — that is a control, not a default, and
// the pipe-delimited form makes the empty case explicit rather than implied by
// an omitted field.
//
// 🛑 The delimiter is `|` and not `:` because filter values carry colons:
// `submitted_at=gte.2026-08-07T01:02:03Z` is exactly the clause this card
// exists to drive.
type binding struct {
	label, schema, table, filter, topic string
	joined                              bool
	joinErr                             string
	sysErr                              string
	events                              int
}

type bindList []*binding

func (b *bindList) String() string { return "" }

func (b *bindList) Set(v string) error {
	parts := strings.SplitN(v, "|", 4)
	if len(parts) != 4 {
		return fmt.Errorf("want label|schema|table|filter (4 pipe-delimited fields, filter may be empty), got %q", v)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return fmt.Errorf("label, schema and table are all required: %q", v)
	}
	*b = append(*b, &binding{
		label: parts[0], schema: parts[1], table: parts[2], filter: parts[3],
		topic: "realtime:" + parts[0],
	})
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	var binds bindList
	addr := flag.String("addr", "", "real TCP host:port of the realtime container's published port")
	vhost := flag.String("host", "realtime-dev.localhost", "virtual host; its FIRST label must equal the tenant external_id")
	token := flag.String("token", "", "HS256 JWT minted by ./mintjwt with the stack's JWT_SECRET")
	joinWait := flag.Duration("joinwait", 15*time.Second, "give up waiting for phx_reply after this long")
	settle := flag.Duration("settle", 4*time.Second, "after the joins reply, collect `system` frames for this long before declaring READY")
	window := flag.Duration("window", 15*time.Second, "observe postgres_changes for this long after READY")
	flag.Var(&binds, "bind", "repeatable: label|schema|table|filter (filter may be empty)")
	flag.Parse()

	if *addr == "" || *token == "" || len(binds) == 0 {
		fmt.Fprintln(os.Stderr, "rtprobe: -addr, -token and at least one -bind are required")
		return exitUsage
	}

	// One overall deadline so a wedged socket cannot hang the script that owns
	// the verdict. Generous: the phases police themselves.
	ctx, cancel := context.WithTimeout(context.Background(), *joinWait+*settle+*window+30*time.Second)
	defer cancel()

	// Connect the socket to -addr but present -host in the Host header, so
	// Realtime's tenant lookup sees the tenant while TCP goes to 127.0.0.1.
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
			fmt.Fprintf(os.Stderr, "rtprobe: handshake failed: %v (HTTP %s)\n", err, resp.Status)
		} else {
			fmt.Fprintln(os.Stderr, "rtprobe: handshake failed:", err)
		}
		return exitCannotRun
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")
	// Realtime frames carry the full row; 1 MiB is far more than any spike row
	// and still bounded.
	conn.SetReadLimit(1 << 20)
	fmt.Printf("RTP CONNECT addr=%s host=%s status=%s\n", *addr, *vhost, resp.Status)

	byTopic := map[string]*binding{}
	for i, b := range binds {
		byTopic[b.topic] = b
		fl := b.filter
		if fl == "" {
			fl = "-"
		}
		fmt.Printf("RTP BIND label=%s topic=%s entity=%s.%s filter=%s\n", b.label, b.topic, b.schema, b.table, fl)

		pgc := map[string]any{"event": "*", "schema": b.schema, "table": b.table}
		// 🛑 The key is OMITTED, not set to "", when there is no filter. An
		// empty string is a filter Realtime would try to parse.
		if b.filter != "" {
			pgc["filter"] = b.filter
		}
		payload, _ := json.Marshal(map[string]any{
			"config": map[string]any{
				"broadcast":        map[string]any{"ack": false, "self": false},
				"presence":         map[string]any{"key": ""},
				"private":          false,
				"postgres_changes": []any{pgc},
			},
			"access_token": *token,
		})
		ref := fmt.Sprint(i + 1)
		if err := writeJSON(ctx, conn, phxMsg{
			Topic: b.topic, Event: "phx_join", Ref: ref, JoinRef: ref, Payload: payload,
		}); err != nil {
			fmt.Fprintln(os.Stderr, "rtprobe: join write failed:", err)
			return exitCannotRun
		}
	}

	// Phoenix drops a socket that stops heart-beating.
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		n := 1000
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

	// 🛑 ONE reader goroutine on the PARENT context. coder/websocket CLOSES the
	// connection when a Read's context is cancelled, so per-phase read deadlines
	// would tear down the socket at the first phase boundary. Phases are timers
	// in the select below; the read never gets a deadline of its own.
	type frame struct {
		data []byte
		err  error
	}
	frames := make(chan frame, 64)
	go func() {
		defer close(frames)
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				select {
				case frames <- frame{err: err}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case frames <- frame{data: data}:
			case <-ctx.Done():
				return
			}
		}
	}()

	handle := func(data []byte) {
		var m phxMsg
		if err := json.Unmarshal(data, &m); err != nil {
			fmt.Printf("RTP RAW-UNPARSED %s\n", truncate(string(data), 400))
			return
		}
		b := byTopic[m.Topic]
		switch m.Event {
		case "phx_reply":
			if b == nil {
				return
			}
			var pr struct {
				Status   string          `json:"status"`
				Response json.RawMessage `json:"response"`
			}
			_ = json.Unmarshal(m.Payload, &pr)
			if pr.Status == "ok" {
				b.joined = true
				fmt.Printf("RTP JOIN-OK label=%s response=%s\n", b.label, truncate(string(pr.Response), 300))
			} else {
				b.joinErr = truncate(string(m.Payload), 400)
				fmt.Printf("RTP JOIN-ERR label=%s payload=%s\n", b.label, b.joinErr)
			}
		case "system":
			var sp struct {
				Status    string `json:"status"`
				Extension string `json:"extension"`
				Message   string `json:"message"`
			}
			_ = json.Unmarshal(m.Payload, &sp)
			label := "?"
			if b != nil {
				label = b.label
			}
			if sp.Status == "error" {
				if b != nil {
					b.sysErr = sp.Message
				}
				// 🛑 THE FRAME THAT MATTERS. See the header block.
				fmt.Printf("RTP SYS-ERR label=%s extension=%s message=%s\n", label, sp.Extension, truncate(sp.Message, 400))
			} else {
				fmt.Printf("RTP SYS label=%s status=%s extension=%s message=%s\n", label, sp.Status, sp.Extension, truncate(sp.Message, 200))
			}
		case "postgres_changes":
			var ev struct {
				Data struct {
					Schema    string         `json:"schema"`
					Table     string         `json:"table"`
					Type      string         `json:"type"`
					EventType string         `json:"eventType"`
					Record    map[string]any `json:"record"`
					New       map[string]any `json:"new"`
				} `json:"data"`
			}
			_ = json.Unmarshal(m.Payload, &ev)
			kind := ev.Data.Type
			if kind == "" {
				kind = ev.Data.EventType
			}
			rec := ev.Data.Record
			if rec == nil {
				rec = ev.Data.New
			}
			id, _ := rec["id"].(string)
			label := "?"
			if b != nil {
				label = b.label
				b.events++
			}
			fmt.Printf("RTP EVENT label=%s type=%s table=%s.%s id=%s\n", label, kind, ev.Data.Schema, ev.Data.Table, id)
		default:
			fmt.Printf("RTP OTHER event=%s topic=%s payload=%s\n", m.Event, m.Topic, truncate(string(m.Payload), 200))
		}
	}

	allReplied := func() bool {
		for _, b := range binds {
			if !b.joined && b.joinErr == "" {
				return false
			}
		}
		return true
	}

	// ---- phase 1: joins -----------------------------------------------------
	joinTimer := time.NewTimer(*joinWait)
	defer joinTimer.Stop()
joinPhase:
	for !allReplied() {
		select {
		case f, ok := <-frames:
			if !ok {
				fmt.Fprintln(os.Stderr, "rtprobe: socket closed during the join phase")
				return exitCannotRun
			}
			if f.err != nil {
				fmt.Fprintln(os.Stderr, "rtprobe: read error during the join phase:", f.err)
				return exitCannotRun
			}
			handle(f.data)
		case <-joinTimer.C:
			fmt.Println("RTP JOIN-TIMEOUT some bindings never replied")
			break joinPhase
		}
	}

	// ---- phase 2: settle — collect the `system` frames the joins do not carry
	settleTimer := time.NewTimer(*settle)
	defer settleTimer.Stop()
settlePhase:
	for {
		select {
		case f, ok := <-frames:
			if !ok || f.err != nil {
				fmt.Fprintln(os.Stderr, "rtprobe: socket died during the settle phase")
				return exitCannotRun
			}
			handle(f.data)
		case <-settleTimer.C:
			break settlePhase
		}
	}

	joined, failed := 0, 0
	for _, b := range binds {
		if b.joined && b.sysErr == "" {
			joined++
		} else {
			failed++
		}
		state := "SUBSCRIBED"
		switch {
		case b.sysErr != "":
			state = "SYS-ERR"
		case b.joinErr != "":
			state = "JOIN-ERR"
		case !b.joined:
			state = "NO-REPLY"
		}
		fmt.Printf("RTP STATE label=%s state=%s\n", b.label, state)
	}
	// 🛑 READY is the handshake with the shell: it does not insert until it sees
	// this line, so no row can be written before every subscription exists.
	fmt.Printf("RTP READY subscribed=%d failed=%d window=%s\n", joined, failed, *window)

	// ---- phase 3: observation window ---------------------------------------
	winTimer := time.NewTimer(*window)
	defer winTimer.Stop()
	total := 0
windowPhase:
	for {
		select {
		case f, ok := <-frames:
			if !ok || (f.err != nil && !errors.Is(f.err, context.Canceled)) {
				if f.err != nil {
					fmt.Fprintln(os.Stderr, "rtprobe: socket died during the observation window:", f.err)
				} else {
					fmt.Fprintln(os.Stderr, "rtprobe: socket closed during the observation window")
				}
				return exitCannotRun
			}
			handle(f.data)
			total++
		case <-winTimer.C:
			break windowPhase
		}
	}

	for _, b := range binds {
		fmt.Printf("RTP COUNT label=%s events=%d\n", b.label, b.events)
	}
	fmt.Printf("RTP DONE frames=%d\n", total)
	return exitOK
}

func writeJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
