// spikec_relay.go — 🛑 SPIKE CODE. NOT PRODUCTION. NOT A PROPOSAL FOR HQ.
//
// Night-crew card C `spike-c-round-trip` (Spike C), run 20260807-2. This file is
// the mechanism half of a SPIKE whose entire purpose is to answer one question
// by script: does a row written through HQ's real write path
// (`POST /api/v1/workflow/saveResponse`) reach an RxDB-served read at all, and
// by what mechanism? Decision 126 measured the "rows flow back from the
// substrate" premise FALSE on night nine of nine, after ~11,200 spec lines had
// been built on it. This is that premise being measured on purpose, up front.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 NOTHING IN cmd/server REFERENCES THIS FILE, AND NOTHING SHOULD.
//
//	grep -rn RunSpikeCRelay backend/cmd/server   ->   no matches
//
// It is exercised only by `backend/cmd/spikec-relay` (a ~40-line runner that is
// itself marked spike code) which is launched only by
// `.night-crew/qa/spike-supabase/spike-c-roundtrip.sh`. It registers no route,
// starts no goroutine at boot, and is not wired into main.go's Hub/Listener.
//
// 🛑 END OF LIFE. The Activity 3 card `skeleton-one-row-end-to-end` either
// adopts this mechanism properly — in which case it REPLACES this file with
// something that has tests, retries, backpressure and an ownership story — or
// the spike verdict retires it. Do not grow features here. If you are adding a
// second table, a queue, or a metric to this file, you are building the
// production relay inside a spike artefact.
// ═══════════════════════════════════════════════════════════════════════════
//
// THE MECHANISM
//
//	HQ Postgres  --NOTIFY 'spike_c_relay'-->  this relay
//	             --HTTP POST (PostgREST, service identity)-->  substrate
//	             --Realtime/pull-->  RxDB
//
// Deliberate properties, each one a finding the cutover card inherits:
//
//  1. It is OUTSIDE the write path. `saveResponse` is not edited by this card;
//     the trigger (sql/spike-c-relay-trigger.sql) observes the write from the
//     database side. A substrate outage cannot red a crew member's checkbox.
//
//  2. It re-reads the row by id rather than trusting the notification payload —
//     the same shape `listener.go`'s handler already uses (GetOpByID), and for
//     the same reason plus one more: pg_notify's payload is capped at 8000
//     bytes and `submission_responses.value` is unbounded JSONB.
//
//  3. It writes through PostgREST with a SERVICE IDENTITY, not a per-user
//     token. Spike B measured that a bulk lane CANNOT run on per-user tokens:
//     `hq_sync_checklists_insert`'s WITH CHECK refuses a row whose owner holds
//     no live grant on its app, and real datasets contain exactly such rows.
//     This relay inherits that finding rather than re-discovering it.
//
//  4. app_slug is a CONSTANT here, and that is spike B's finding #1 surfacing
//     again, not laziness: HQ stores NO template->app association, so there is
//     nothing to populate the sync contract's app_slug from. Hardcoding it in a
//     spike and labelling it is honest; hardcoding it in production would be
//     the bug. Where that association should live is an open question the
//     cutover card inherits.
//
// The dependency set is exactly what internal/sync already imports: pgx/v5,
// pgxpool and pgxlisten are all DIRECT dependencies of backend/go.mod because
// listener.go already uses every one of them. This file adds nothing to HQ's
// supply chain.
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
)

// SpikeCRelayChannel is the NOTIFY channel sql/spike-c-relay-trigger.sql fires
// on. Deliberately NOT `ops_channel`: sharing the production channel would make
// a spike's traffic indistinguishable from the op journal's on any listener
// that happens to be up.
const SpikeCRelayChannel = "spike_c_relay"

// SpikeCRelayConfig is everything the relay needs. Every field is required; the
// relay refuses to start with any of them empty rather than degrading into a
// no-op, which is the exact failure shape this card exists to avoid.
type SpikeCRelayConfig struct {
	// HQConnStr is the RAW connection string for the SCRATCH HQ-shaped Postgres
	// (project `spike-c-hq`, Docker-assigned ephemeral port). A dedicated
	// connection is opened for LISTEN, exactly as StartListener does.
	//
	// 🛑 The caller is responsible for this never being :5433 (PRODUCTION) or
	// :5434 (yumyums-test-pg). spike-c-roundtrip.sh asserts it twice.
	HQConnStr string

	// RESTBase is spike A's PostgREST origin, e.g. http://127.0.0.1:46233.
	RESTBase string

	// ServiceToken is an HS256 JWT with role=service_role, minted by the spike's
	// own Go minter. See property 3 in the file banner for why a per-user token
	// cannot do this job.
	ServiceToken string

	// SyncTable is the substrate table (spike A's `hq_sync_checklists`).
	SyncTable string

	// AppSlug is the constant app_slug written into every projected row. See
	// property 4 in the file banner — this is spike B's finding #1, not a
	// default worth trusting.
	AppSlug string

	// Ready, if non-nil, is closed once the LISTEN connection is established and
	// the relay is genuinely able to receive. The shell waits on the process's
	// stdout line rather than a race on "the process exists" — a relay that has
	// not finished connecting drops the very first notification, and a spike
	// that misses the only write it makes reds for the wrong reason.
	Ready func()
}

// spikeCNotify is the JSON shape sql/spike-c-relay-trigger.sql emits.
type spikeCNotify struct {
	ResponseID   string  `json:"response_id"`
	FieldID      string  `json:"field_id"`
	AnsweredBy   string  `json:"answered_by"`
	SubmissionID *string `json:"submission_id"`
	Op           string  `json:"op"`
}

// RunSpikeCRelay blocks until ctx is cancelled, relaying every
// submission_responses write it is told about into the substrate.
//
// 🛑 IT FAILS LOUD AND IT NEVER SWALLOWS. Contrast StartListener, which logs
// and returns nil on every error so a bad notification cannot kill the server:
// that is right for production and WRONG for a spike, where a swallowed error
// is a silent no-op and a silent no-op is indistinguishable from "the mechanism
// does not exist". Every failure here is returned, and the runner exits
// non-zero on it.
func RunSpikeCRelay(ctx context.Context, cfg SpikeCRelayConfig) error {
	switch {
	case cfg.HQConnStr == "":
		return fmt.Errorf("spike-c relay: HQConnStr is empty")
	case cfg.RESTBase == "":
		return fmt.Errorf("spike-c relay: RESTBase is empty")
	case cfg.ServiceToken == "":
		return fmt.Errorf("spike-c relay: ServiceToken is empty")
	case cfg.SyncTable == "":
		return fmt.Errorf("spike-c relay: SyncTable is empty")
	case cfg.AppSlug == "":
		return fmt.Errorf("spike-c relay: AppSlug is empty")
	}

	pool, err := pgxpool.New(ctx, cfg.HQConnStr)
	if err != nil {
		return fmt.Errorf("spike-c relay: pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("spike-c relay: ping HQ postgres: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	errCh := make(chan error, 1)

	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			return pgx.Connect(ctx, cfg.HQConnStr)
		},
		ReconnectDelay: time.Second,
		LogError: func(ctx context.Context, err error) {
			// Surfaced, not swallowed: a LISTEN that keeps failing to reconnect
			// would otherwise present as a mechanism that simply never fires.
			slog.Error("spike-c relay listen error", "error", err)
		},
	}

	listener.Handle(SpikeCRelayChannel, pgxlisten.HandlerFunc(
		func(ctx context.Context, n *pgconn.Notification, _ *pgx.Conn) error {
			if err := relayOne(ctx, pool, client, cfg, n.Payload); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return err
			}
			return nil
		}))

	go func() {
		// pgxlisten's Listen establishes the connection and issues LISTEN before
		// it starts waiting. There is no callback for "connected", so readiness
		// is announced by the caller after a successful Ping + a short settle;
		// the shell additionally polls for the relay's stdout READY line.
		if err := listener.Listen(ctx); err != nil && ctx.Err() == nil {
			select {
			case errCh <- fmt.Errorf("spike-c relay: listener exited: %w", err):
			default:
			}
		}
	}()

	// Give LISTEN a moment to be established before announcing readiness. The
	// alternative — announcing immediately — loses the first NOTIFY, and this
	// spike makes exactly one write.
	select {
	case <-time.After(750 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
	if cfg.Ready != nil {
		cfg.Ready()
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// relayOne is the whole transform, in one readable function on purpose.
func relayOne(ctx context.Context, pool *pgxpool.Pool, client *http.Client, cfg SpikeCRelayConfig, payload string) error {
	var n spikeCNotify
	if err := json.Unmarshal([]byte(payload), &n); err != nil {
		return fmt.Errorf("spike-c relay: bad notify payload %q: %w", payload, err)
	}

	// Re-read by id. See property 2 in the file banner.
	//
	// The JOIN out to checklist_fields is what makes the projected body
	// recognisable to a human reading the RxDB document, and it is also the
	// cheapest available proof that the relay read HQ's REAL schema rather than
	// a table it invented: `checklist_fields` exists only because HQ's own
	// migrations created it in this database.
	var (
		respID     string
		fieldID    string
		fieldLabel string
		valueJSON  []byte
		answeredBy string
		answeredAt time.Time
	)
	err := pool.QueryRow(ctx, `
		SELECT sr.id::text, sr.field_id::text, COALESCE(f.label, ''),
		       sr.value, sr.answered_by::text, sr.answered_at
		FROM submission_responses sr
		LEFT JOIN checklist_fields f ON f.id = sr.field_id
		WHERE sr.id = $1`, n.ResponseID,
	).Scan(&respID, &fieldID, &fieldLabel, &valueJSON, &answeredBy, &answeredAt)
	if err != nil {
		return fmt.Errorf("spike-c relay: re-read response %s: %w", n.ResponseID, err)
	}

	body, err := json.Marshal(map[string]any{
		"kind":        "submission_response",
		"response_id": respID,
		"field_id":    fieldID,
		"field_label": fieldLabel,
		"value":       json.RawMessage(valueJSON),
		"answered_at": answeredAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("spike-c relay: marshal body: %w", err)
	}

	// The sync contract's primary key is TEXT and HQ's keys are uuid — the same
	// impedance mismatch spike B crossed with ::text. The prefix keeps a
	// projected row visibly distinguishable from spike A's fixture rows in a
	// shared table, which is what makes this spike's teardown able to prove it
	// restored the substrate exactly.
	row := map[string]any{
		"id":       "spikec-" + respID,
		"owner_id": answeredBy,
		"app_slug": cfg.AppSlug,
		"body":     string(body),
	}
	buf, err := json.Marshal([]any{row})
	if err != nil {
		return fmt.Errorf("spike-c relay: marshal row: %w", err)
	}

	url := fmt.Sprintf("%s/%s", cfg.RESTBase, cfg.SyncTable)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("spike-c relay: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceToken)
	// merge-duplicates makes this an UPSERT: `/saveResponse` is an upsert too
	// (ON CONFLICT (field_id, answered_by) DO UPDATE), so the second save of the
	// same field must project as an UPDATE, not a 409. An update is also the
	// harder RxDB pull case — an insert can arrive via a full re-read, whereas
	// an update to a document the client already holds exercises the checkpoint
	// path (spike A README half 2, PULL case 2).
	req.Header.Set("Prefer", "resolution=merge-duplicates,return=minimal")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("spike-c relay: POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("spike-c relay: POST %s -> HTTP %d: %s", url, resp.StatusCode, string(respBody))
	}

	slog.Info("spike-c relay projected row",
		"id", row["id"], "owner_id", answeredBy, "app_slug", cfg.AppSlug, "http", resp.StatusCode)
	return nil
}
