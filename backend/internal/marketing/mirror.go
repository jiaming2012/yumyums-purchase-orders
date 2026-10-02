package marketing

// mirror.go — the scan_attempts mirror (card H3a, run 20261002).
//
// # Why a mirror exists at all
//
// public.scan_attempts is DEVICE-OWNED and PUSH-ONLY: Supabase RLS grants
// `authenticated` INSERT and nothing else, so a tablet can record what happened
// at the counter and can never read it back. Only `service_role` can SELECT
// (spike 03, 2026-10-01: service_role 200, device JWT 403 — the asymmetry
// Activity A established is intact). That means HQ, which is where a manager
// reconciles, had NO server-side copy of the counter's outcome. This poller is
// that copy, and its existence is what makes F4's scan_attempts-status
// acceptance bullet true (B-424; the full list of what becomes visible is in
// this card's merge-intent).
//
// # Why keyset and not an offset or a "since" timestamp
//
// Attempts arrive from devices in bursts after a sync, so two rows routinely
// share a `scanned_at` to the microsecond. An OFFSET page shifts under inserts;
// a `scanned_at > last` filter silently drops the second row of any tie. The
// cursor is therefore the PAIR (scanned_at, id) and the predicate is the strict
// lexicographic successor — spike 03 proved PostgREST accepts
// `order=scanned_at.asc,id.asc`, and this file's test proves the tie-break leg
// resumes correctly against the live substrate.
//
// # Where the cursor lives
//
// In scan_attempts_mirror itself: max (scanned_at, id) already mirrored. There
// is deliberately NO checkpoint table. A checkpoint that can disagree with the
// data it describes is a second source of truth, and the failure mode — a
// checkpoint ahead of the rows — is a permanent silent gap. Deriving it costs
// one indexed row read per tick (scan_attempts_mirror_keyset_idx) and cannot
// drift. A restart just re-derives it; an upsert makes a re-read harmless.
//
// # 🛑 A NAMED GAP IN THE KEYSET, STATED RATHER THAN SHIPPED QUIETLY
//
// `scanned_at` is WHEN THE CODE WAS ACCEPTED AT THE COUNTER, not when the row
// reached the substrate. An offline override syncs minutes or hours after the
// scan, so an attempt can ARRIVE upstream carrying a `scanned_at` EARLIER than a
// cursor this poller has already passed — and a strict (scanned_at, id) keyset
// will then never see it. The card binds the mechanism to `(scanned_at, id)`
// (slate H3a; spike 03 proved exactly that order), so that is what ships, but the
// consequence is real and belongs on the record rather than in a surprise later:
// late-arriving offline attempts older than the cursor are not mirrored.
//
// The fix, when someone wants it, is cheap and already safe here: resume from
// `cursor.scanned_at - <replay window>` rather than from the cursor itself. The
// upsert below is idempotent, so re-reading a bounded window costs round trips
// and nothing else. It is deliberately NOT done tonight because it would
// contradict the card's done_when ("the second poll resumes after the first's
// last (scanned_at, id)"), which is the behaviour mirror_test.go pins.
//
// # Substrate coordinates
//
// The SAME HQ_SYNC_REST_URL / HQ_SYNC_SERVICE_KEY pair H1's projection reads,
// through H1's own ProjectionConfig. This card adds no env var: the mirror and
// the projection talk to one substrate, and two names for one endpoint is how
// they drift apart.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MirrorInterval is the poll period. Five minutes is the card's figure and it
// is the right order of magnitude: the consumer is a human reconciling a queue,
// for whom "within five minutes of the counter" is indistinguishable from live,
// while a tighter loop would spend a service-role round trip per minute to
// discover nothing on a truck that is closed 18 hours a day.
const MirrorInterval = 5 * time.Minute

// MirrorPageSize is the PostgREST page size. 500 keeps a burst of a busy
// service inside two or three pages without asking the substrate for a
// multi-megabyte body.
const MirrorPageSize = 500

// MirrorMaxPages bounds ONE tick. A cold start against a long history drains
// over several ticks rather than holding a single poll open indefinitely —
// progress is durable after every page, because every page is upserted before
// the next is requested.
const MirrorMaxPages = 50

// mirrorRequestBudget caps a single page read. Generous relative to a 5-minute
// period, tight enough that a hung substrate cannot pin the goroutine past the
// next tick.
const mirrorRequestBudget = 30 * time.Second

// mirrorSelect is the upstream column list, named explicitly. A `select=*`
// would make the mirror's shape depend on whatever the substrate happens to
// have, which is how a column silently stops being copied.
//
// `campaign_id` is NOT here: upstream scan_attempts has no such column and
// code_id carries no FK PostgREST could embed through, so the mirror's
// campaign_id (which §4 defines) stays NULL. Resolving it needs a `codes` pull,
// which belongs to whoever needs it.
const mirrorSelect = "id,code_id,device_id,scanned_at,status,reason,offline_override," +
	"override_by,unverified_code,policy_unresolved,token_hash,pos_order_number," +
	"pos_business_date,redeemed_value,match_status"

// MirrorCursor is the keyset position: the last (scanned_at, id) mirrored.
type MirrorCursor struct {
	ScannedAt time.Time
	ID        string
}

// String renders the cursor for logs and test assertions.
func (c MirrorCursor) String() string {
	return c.ScannedAt.UTC().Format(mirrorTimeLayout) + "/" + c.ID
}

// mirrorTimeLayout is microsecond-precision RFC3339 in UTC. Microseconds
// because that is Postgres timestamptz's resolution — formatting nanoseconds we
// do not have would build a cursor strictly greater than the row it came from
// and skip its tie-mates. UTC because a `+` in an offset inside a PostgREST
// `or=(…)` group is one more quoting rule to get wrong.
const mirrorTimeLayout = "2006-01-02T15:04:05.999999Z"

// MirrorAttempt is one upstream row as decoded from PostgREST.
type MirrorAttempt struct {
	ID               string       `json:"id"`
	CodeID           *string      `json:"code_id"`
	DeviceID         string       `json:"device_id"`
	ScannedAt        time.Time    `json:"scanned_at"`
	Status           string       `json:"status"`
	Reason           *string      `json:"reason"`
	OfflineOverride  bool         `json:"offline_override"`
	OverrideBy       *string      `json:"override_by"`
	UnverifiedCode   bool         `json:"unverified_code"`
	PolicyUnresolved bool         `json:"policy_unresolved"`
	TokenHash        *string      `json:"token_hash"`
	PosOrderNumber   *string      `json:"pos_order_number"`
	PosBusinessDate  string       `json:"pos_business_date"`
	RedeemedValue    *json.Number `json:"redeemed_value"`
	MatchStatus      string       `json:"match_status"`
}

// MirrorResult is one poll's outcome.
type MirrorResult struct {
	// Fetched is how many upstream rows this poll read. On a resumed poll it
	// counts only rows AFTER the starting cursor — which is the whole claim the
	// keyset makes, and what TestScanAttemptsMirrorKeysetResumes asserts.
	Fetched int
	// Upserted is how many landed in scan_attempts_mirror. Equal to Fetched
	// unless the substrate handed back a row the mirror refused.
	Upserted int
	// Pages is how many PostgREST round trips this poll made.
	Pages int
	// From is the cursor the poll STARTED at (nil = from the beginning).
	From *MirrorCursor
	// To is the cursor the poll ENDED at (nil = nothing has ever been mirrored).
	To *MirrorCursor
}

// ErrMirrorUnconfigured is returned when the substrate coordinates are absent.
// It is a normal state for a box with no substrate, which is why MirrorStart
// logs it and idles rather than treating it as a failure.
var ErrMirrorUnconfigured = errors.New("marketing mirror: not configured")

var mirrorHTTPClient = &http.Client{Timeout: mirrorRequestBudget}

// MirrorStart launches the poller. It is the ONE call site this card adds to
// cmd/server/main.go, and it sits inside main.go's schedulersDisabled region so
// E2E_DISABLE_SCHEDULERS=1 turns it off with the other background pollers — an
// E2E stack must not have a goroutine reaching for a substrate mid-suite.
//
// Unconfigured is IDLE AND LOGGED, never a crash and never silent: a dev box
// with no substrate is the common case, and the log line is what tells an
// operator why their reconciliation queue is empty.
func MirrorStart(ctx context.Context, pool *pgxpool.Pool, cfg ProjectionConfig) {
	if !cfg.Configured() {
		slog.Info("scan-attempts mirror idle: substrate not configured; scan_attempts_mirror will stay empty",
			"missing", RESTURLEnv+" and/or "+ServiceKeyEnv, "interval", MirrorInterval)
		return
	}
	if pool == nil {
		slog.Error("scan-attempts mirror not started: nil pool")
		return
	}
	slog.Info("scan-attempts mirror starting", "interval", MirrorInterval, "page_size", MirrorPageSize)

	go func() {
		mirrorTick(ctx, pool, cfg)
		ticker := time.NewTicker(MirrorInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("scan-attempts mirror: shutting down")
				return
			case <-ticker.C:
				mirrorTick(ctx, pool, cfg)
			}
		}
	}()
}

// mirrorTick is one scheduled poll, logged. A failure is ERROR and the next
// tick retries from the same durable cursor — there is nothing to reset.
func mirrorTick(ctx context.Context, pool *pgxpool.Pool, cfg ProjectionConfig) {
	res, err := MirrorPollOnce(ctx, pool, cfg)
	if err != nil {
		slog.Error("scan-attempts mirror poll failed", "error", err,
			"fetched", res.Fetched, "upserted", res.Upserted, "pages", res.Pages)
		return
	}
	if res.Fetched == 0 {
		slog.Debug("scan-attempts mirror: nothing new", "cursor", mirrorCursorLog(res.To))
		return
	}
	slog.Info("scan-attempts mirror: copied attempts",
		"fetched", res.Fetched, "upserted", res.Upserted, "pages", res.Pages,
		"from", mirrorCursorLog(res.From), "to", mirrorCursorLog(res.To))
}

func mirrorCursorLog(c *MirrorCursor) string {
	if c == nil {
		return "(none)"
	}
	return c.String()
}

// MirrorPollOnce drains everything after the durable cursor, one page at a
// time, upserting each page before asking for the next so progress survives a
// mid-drain failure.
func MirrorPollOnce(ctx context.Context, pool *pgxpool.Pool, cfg ProjectionConfig) (MirrorResult, error) {
	var res MirrorResult
	if !cfg.Configured() {
		return res, fmt.Errorf("%w (%s and/or %s empty)", ErrMirrorUnconfigured, RESTURLEnv, ServiceKeyEnv)
	}
	if pool == nil {
		return res, fmt.Errorf("marketing mirror: nil pool")
	}

	cursor, err := MirrorReadCursor(ctx, pool)
	if err != nil {
		return res, err
	}
	res.From = cursor
	res.To = cursor

	for page := 0; page < MirrorMaxPages; page++ {
		rows, err := mirrorFetchPage(ctx, cfg, cursor)
		if err != nil {
			return res, err
		}
		res.Pages++
		if len(rows) == 0 {
			return res, nil
		}
		n, err := mirrorUpsert(ctx, pool, rows)
		res.Fetched += len(rows)
		res.Upserted += n
		if err != nil {
			return res, err
		}
		last := rows[len(rows)-1]
		cursor = &MirrorCursor{ScannedAt: last.ScannedAt, ID: last.ID}
		res.To = cursor
		if len(rows) < MirrorPageSize {
			return res, nil
		}
	}
	slog.Warn("scan-attempts mirror: page cap reached; the remainder drains on the next tick",
		"max_pages", MirrorMaxPages, "cursor", mirrorCursorLog(res.To))
	return res, nil
}

// MirrorReadCursor derives the keyset position from the mirror itself. nil means
// nothing has ever been mirrored, i.e. start from the beginning of history.
func MirrorReadCursor(ctx context.Context, pool *pgxpool.Pool) (*MirrorCursor, error) {
	var c MirrorCursor
	err := pool.QueryRow(ctx,
		`SELECT scanned_at, id::text FROM scan_attempts_mirror
		  ORDER BY scanned_at DESC, id DESC LIMIT 1`).Scan(&c.ScannedAt, &c.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("marketing mirror: read cursor: %w", err)
	}
	return &c, nil
}

// mirrorFetchPage is the service-role read. The keyset predicate is the strict
// lexicographic successor of the cursor:
//
//	scanned_at > c.scanned_at  OR  (scanned_at = c.scanned_at AND id > c.id)
//
// expressed in PostgREST's or=(…,and(…)) grammar. The cursor row itself is
// excluded; its tie-mates with a greater id are not.
func mirrorFetchPage(ctx context.Context, cfg ProjectionConfig, cursor *MirrorCursor) ([]MirrorAttempt, error) {
	q := url.Values{}
	q.Set("select", mirrorSelect)
	q.Set("order", "scanned_at.asc,id.asc")
	q.Set("limit", fmt.Sprint(MirrorPageSize))
	if cursor != nil {
		ts := cursor.ScannedAt.UTC().Format(mirrorTimeLayout)
		q.Set("or", fmt.Sprintf(`(scanned_at.gt."%s",and(scanned_at.eq."%s",id.gt.%s))`, ts, ts, cursor.ID))
	}
	endpoint := strings.TrimRight(cfg.RESTURL, "/") + "/scan_attempts?" + q.Encode()

	reqCtx, cancel := context.WithTimeout(ctx, mirrorRequestBudget)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("marketing mirror: build request: %w", err)
	}
	req.Header.Set("apikey", cfg.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceKey)
	req.Header.Set("Accept", "application/json")

	resp, err := mirrorHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marketing mirror: read scan_attempts: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Never echo the body — PostgREST errors can carry row data. The status
		// is the diagnosis: 401/403 means the service key is wrong (devices get
		// 403 here by design), 400 means the column list no longer matches the
		// substrate.
		return nil, fmt.Errorf("marketing mirror: scan_attempts read: postgrest status %d", resp.StatusCode)
	}
	var rows []MirrorAttempt
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("marketing mirror: decode scan_attempts: %w", err)
	}
	return rows, nil
}

// mirrorUpsert lands a page in one transaction, keyed on the upstream id.
//
// DO UPDATE and not DO NOTHING: `status`, `reason` and `match_status` CHANGE
// upstream after the row first appears (a pending attempt is arbitrated; an
// unmatched one is matched), so the mirror has to carry the latest value or the
// queue would show every attempt as it was first seen. mirrored_at records when
// HQ last agreed with the substrate.
func mirrorUpsert(ctx context.Context, pool *pgxpool.Pool, rows []MirrorAttempt) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("marketing mirror: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	for _, a := range rows {
		var redeemed any
		if a.RedeemedValue != nil {
			// Passed as its decimal STRING, not a float: the column is numeric
			// and money that round-trips through float64 is money that can
			// disagree with the substrate in the last cent.
			redeemed = a.RedeemedValue.String()
		}
		batch.Queue(`
			INSERT INTO scan_attempts_mirror
			    (id, code_id, campaign_id, device_id, scanned_at, status, reason,
			     offline_override, override_by, unverified_code, policy_unresolved,
			     token_hash, pos_order_number, pos_business_date, redeemed_value, match_status)
			VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT (id) DO UPDATE SET
			    code_id           = EXCLUDED.code_id,
			    device_id         = EXCLUDED.device_id,
			    scanned_at        = EXCLUDED.scanned_at,
			    status            = EXCLUDED.status,
			    reason            = EXCLUDED.reason,
			    offline_override  = EXCLUDED.offline_override,
			    override_by       = EXCLUDED.override_by,
			    unverified_code   = EXCLUDED.unverified_code,
			    policy_unresolved = EXCLUDED.policy_unresolved,
			    token_hash        = EXCLUDED.token_hash,
			    pos_order_number  = EXCLUDED.pos_order_number,
			    pos_business_date = EXCLUDED.pos_business_date,
			    redeemed_value    = EXCLUDED.redeemed_value,
			    match_status      = EXCLUDED.match_status,
			    mirrored_at       = now()`,
			a.ID, a.CodeID, a.DeviceID, a.ScannedAt, a.Status, a.Reason,
			a.OfflineOverride, a.OverrideBy, a.UnverifiedCode, a.PolicyUnresolved,
			a.TokenHash, a.PosOrderNumber, a.PosBusinessDate, redeemed, a.MatchStatus)
	}
	br := tx.SendBatch(ctx, batch)
	for i := range rows {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return 0, fmt.Errorf("marketing mirror: upsert attempt %s: %w", rows[i].ID, err)
		}
	}
	if err := br.Close(); err != nil {
		return 0, fmt.Errorf("marketing mirror: batch close: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("marketing mirror: commit: %w", err)
	}
	return len(rows), nil
}
