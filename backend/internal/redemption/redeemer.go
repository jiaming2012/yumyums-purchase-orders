package redemption

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Env vars for the production Redeemer. RESTURLEnv is the SAME PostgREST base
// the sync proxy uses (internal/sync/proxy.go ProxyRESTURLEnv — redeclared
// here rather than imported so this package does not pull the proxy in).
// ServiceKeyEnv is NEW: the substrate's service_role key. The redeem() RPC's
// grant anticipates exactly this caller ("HQ's Go orchestration (Activity D)
// calls it as service_role" — supabase/migrations/20260904000200). Both unset
// or either unset ⇒ not configured ⇒ the endpoint fails closed with 503.
const (
	RESTURLEnv    = "HQ_SYNC_REST_URL"
	ServiceKeyEnv = "HQ_SYNC_SERVICE_KEY"
)

// RedeemerConfig is the production Redeemer's wiring.
type RedeemerConfig struct {
	RESTURL    string
	ServiceKey string
}

// LoadRedeemerConfig reads the config from the environment.
func LoadRedeemerConfig() RedeemerConfig {
	return RedeemerConfig{
		RESTURL:    os.Getenv(RESTURLEnv),
		ServiceKey: os.Getenv(ServiceKeyEnv),
	}
}

// Configured reports whether both halves are present.
func (c RedeemerConfig) Configured() bool {
	return c.RESTURL != "" && c.ServiceKey != ""
}

// RPCRedeemer is the production Redeemer: it calls the substrate's atomic
// redeem() RPC through PostgREST as service_role.
//
// Two steps, and only the second touches redemption state:
//
//  1. IDENTITY RESOLUTION — token_hash → codes.id. This reads which row the
//     token names, never whether it is redeemed; there is no redemption-state
//     filter in the query and nothing gates on one (§18 edge-case 1 — the
//     conditional UPDATE inside redeem() stays the only arbiter). A missing
//     row is the definitive not_found verdict (F2's unknown-code override
//     arrives here).
//  2. THE ATOMIC BURN — POST /rpc/redeem. Its (ok, reason) verdict is
//     returned verbatim as the §6 taxonomy; a malformed verdict (ok=false,
//     reason NULL — impossible per GAP-1, defended anyway) surfaces as an
//     empty status for the machine's E-KR2 fallback to fail loudly.
//
// Both requests honor ctx — gstate cancels it on state exit, and a hung call
// on a dropped hotspot must abort with it (§18 edge-case 4).
type RPCRedeemer struct {
	restURL string
	key     string
	client  *http.Client
}

// NewRPCRedeemer builds the production Redeemer. The client timeout is a
// transport backstop; per-arbitration budgets come from ctx.
func NewRPCRedeemer(cfg RedeemerConfig) *RPCRedeemer {
	return &RPCRedeemer{
		restURL: strings.TrimRight(cfg.RESTURL, "/"),
		key:     cfg.ServiceKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *RPCRedeemer) Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (string, error) {
	// 1. Identity resolution (no redemption-state read).
	q := url.Values{}
	q.Set("select", "id")
	q.Set("token_hash", "eq."+tokenHash)
	var codes []struct {
		ID string `json:"id"`
	}
	if err := r.do(ctx, http.MethodGet, "/codes?"+q.Encode(), nil, &codes); err != nil {
		return "", fmt.Errorf("resolve code: %w", err)
	}
	if len(codes) == 0 {
		return OutcomeNotFound, nil
	}

	// 2. The atomic burn.
	body, err := json.Marshal(map[string]string{
		"p_code":   codes[0].ID,
		"p_device": deviceID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal rpc body: %w", err)
	}
	var verdict []struct {
		OK     bool    `json:"ok"`
		Reason *string `json:"reason"`
	}
	if err := r.do(ctx, http.MethodPost, "/rpc/redeem", body, &verdict); err != nil {
		return "", fmt.Errorf("redeem rpc: %w", err)
	}
	if len(verdict) == 0 {
		return "", nil // malformed verdict → E-KR2 fallback fails loudly
	}
	if verdict[0].OK {
		return OutcomeRedeemed, nil
	}
	if verdict[0].Reason == nil {
		return "", nil // GAP-1 says impossible; E-KR2 fallback if it ever is
	}
	return *verdict[0].Reason, nil
}

func (r *RPCRedeemer) do(ctx context.Context, method, path string, body []byte, out any) error {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.restURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", r.key)
	req.Header.Set("Authorization", "Bearer "+r.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Never echo the response body wholesale — PostgREST errors can carry
		// row data. Status alone is enough to act on.
		return fmt.Errorf("postgrest %s %s: status %d", method, path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
