package marketing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// The projection reuses the SAME substrate coordinates the redemption arbiter
// and the sync proxy do. Redeclared here rather than imported so this package
// does not pull internal/redemption in — the precedent is
// redemption/redeemer.go doing exactly that against internal/sync.
const (
	RESTURLEnv    = "HQ_SYNC_REST_URL"
	ServiceKeyEnv = "HQ_SYNC_SERVICE_KEY"
)

// DefaultRequiresOnlineThresholdCents is the fallback for #5's
// marketing_settings.requires_online_threshold_cents, and it is deliberately
// the same value the substrate seeds ($20.00 —
// supabase/migrations/20260904000100_qr_attribution_spine.sql).
//
// WHY THERE IS A FALLBACK AT ALL: marketing_settings is a SUPABASE table with
// RLS on, no policies and no client grants. HQ Postgres has no copy of it and
// this card does not add one (that would be a second writer for an
// operator-owned knob). So the threshold is read over PostgREST as service_role
// when the projection is configured, and falls back to this constant — logged
// at WARN — when it is not. Creating a campaign is never blocked by an
// unreachable substrate.
const DefaultRequiresOnlineThresholdCents = 2000

// thresholdBudget caps the settings read. It sits on the create path, so a
// hung substrate must not hold a manager's save open.
const thresholdBudget = 3 * time.Second

// ProjectionConfig is the substrate half of Deps.
type ProjectionConfig struct {
	RESTURL    string
	ServiceKey string
}

// LoadProjectionConfig reads the config from the environment.
func LoadProjectionConfig() ProjectionConfig {
	return ProjectionConfig{
		RESTURL:    os.Getenv(RESTURLEnv),
		ServiceKey: os.Getenv(ServiceKeyEnv),
	}
}

// Configured reports whether both halves are present. Either half missing is
// "unconfigured", which is the normal state of a box with no substrate.
func (c ProjectionConfig) Configured() bool {
	return c.RESTURL != "" && c.ServiceKey != ""
}

// ProjectionRow is the FOUR tablet columns, and only those four.
//
// Supabase public.campaigns is (id, name, face_value, requires_online,
// updated_at) — there is no expires_at on it (expiry lives on `codes`), so
// decision 187's "expires_at-equivalent" has no column to land in and the
// projection is the four columns spike 02 proved. Widening this struct widens
// the replica and the RLS surface, which is the thing decision 187 exists to
// avoid.
type ProjectionRow struct {
	ID             string
	Name           string
	FaceValueCents int
	RequiresOnline bool
}

// ProjectCampaign upserts the four columns into Supabase `campaigns` over
// PostgREST as service_role, keyed by id.
//
// This is the Go port of
// .night-crew/spikes/activity-h-designed-tabs/campaign-codes-api/02-projection-upsert-over-postgrest.sh
// (exit 0, 2026-10-01): POST the row with
// `Prefer: resolution=merge-duplicates` so the same id updates in place.
// face_value is numeric DOLLARS upstream, so cents are divided by 100 here —
// the one unit conversion in this package, and the only place it happens.
//
// It is called AFTER the local transaction commits. A failure here is NEVER a
// rollback: the campaign is saved, projected_at stays NULL and the caller
// surfaces warnings:["not_projected"].
func ProjectCampaign(ctx context.Context, cfg ProjectionConfig, row ProjectionRow) error {
	if !cfg.Configured() {
		return fmt.Errorf("projection not configured (%s and/or %s empty)", RESTURLEnv, ServiceKeyEnv)
	}
	body, err := json.Marshal([]map[string]any{{
		"id":              row.ID,
		"name":            row.Name,
		"face_value":      float64(row.FaceValueCents) / 100.0,
		"requires_online": row.RequiresOnline,
	}})
	if err != nil {
		return fmt.Errorf("marshal projection row: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(cfg.RESTURL, "/")+"/campaigns", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build projection request: %w", err)
	}
	req.Header.Set("apikey", cfg.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "resolution=merge-duplicates")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("projection upsert: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Never echo the response body — PostgREST errors can carry row data.
		return fmt.Errorf("projection upsert: postgrest status %d", resp.StatusCode)
	}
	return nil
}

// thresholdCents resolves #5's requires_online threshold.
//
// Configured substrate -> read marketing_settings.requires_online_threshold_cents.
// Anything else        -> DefaultRequiresOnlineThresholdCents, logged at WARN.
//
// The WARN matters: an operator who raised the threshold and whose substrate is
// unreachable is getting the default, and that must be visible in the log
// rather than inferred from a campaign that came out with the wrong policy flag.
func thresholdCents(ctx context.Context, cfg ProjectionConfig) int {
	if !cfg.Configured() {
		return DefaultRequiresOnlineThresholdCents
	}
	ctx, cancel := context.WithTimeout(ctx, thresholdBudget)
	defer cancel()

	url := strings.TrimRight(cfg.RESTURL, "/") +
		"/marketing_settings?id=eq.1&select=requires_online_threshold_cents"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Warn("marketing: build threshold request failed; using the default",
			"error", err, "default_cents", DefaultRequiresOnlineThresholdCents)
		return DefaultRequiresOnlineThresholdCents
	}
	req.Header.Set("apikey", cfg.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceKey)

	resp, err := (&http.Client{Timeout: thresholdBudget}).Do(req)
	if err != nil {
		slog.Warn("marketing: marketing_settings unreachable; using the default requires_online threshold",
			"error", err, "default_cents", DefaultRequiresOnlineThresholdCents)
		return DefaultRequiresOnlineThresholdCents
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("marketing: marketing_settings read refused; using the default requires_online threshold",
			"status", resp.StatusCode, "default_cents", DefaultRequiresOnlineThresholdCents)
		return DefaultRequiresOnlineThresholdCents
	}
	var rows []struct {
		Cents *int `json:"requires_online_threshold_cents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil || len(rows) == 0 || rows[0].Cents == nil {
		slog.Warn("marketing: marketing_settings row absent or malformed; using the default requires_online threshold",
			"error", err, "rows", len(rows), "default_cents", DefaultRequiresOnlineThresholdCents)
		return DefaultRequiresOnlineThresholdCents
	}
	return *rows[0].Cents
}
