// Package delivery — THROWAWAY spike copy for E2-01. Copied by the spike script into the
// worktree as backend/internal/delivery/signalwire.go. It is the SHAPE the card builds — a
// Sender interface the marketing package may import, and a SignalWire implementation that
// speaks the Twilio-compatible LaML REST surface — not the card's code.
//
// Why a separate package: internal/marketing's TestNothingInThisPackageSends reds any outbound
// HTTP call in that package and any import path containing a vendor name. So the sender lives
// HERE, the package path carries no vendor name, and marketing imports only the interface.
package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Sender is the one seam internal/marketing imports. A mocked Sender is what every automated
// gate runs against (D-KR2); the live one is wired from env at boot.
type Sender interface {
	SendMMS(ctx context.Context, to, mediaURL, body string) (sid string, err error)
}

// SignalWire sends through the LaML (Twilio-compatible) Messages resource:
//   POST {BaseURL}/api/laml/2010-04-01/Accounts/{ProjectID}/Messages.json
//   basic auth ProjectID:APIToken; form-encoded From, To, Body, MediaUrl.
// BaseURL is https://<space>.signalwire.com in production and an httptest.Server in tests.
type SignalWire struct {
	BaseURL   string
	ProjectID string
	APIToken  string
	From      string
	HTTP      *http.Client
}

type messageReply struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    any    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func (s SignalWire) SendMMS(ctx context.Context, to, mediaURL, body string) (string, error) {
	form := url.Values{}
	form.Set("From", s.From)
	form.Set("To", to)
	form.Set("Body", body)
	form.Set("MediaUrl", mediaURL)
	endpoint := strings.TrimRight(s.BaseURL, "/") + "/api/laml/2010-04-01/Accounts/" + s.ProjectID + "/Messages.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(s.ProjectID, s.APIToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("signalwire: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var r messageReply
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("signalwire: decode reply: %w", err)
	}
	if r.SID == "" {
		return "", fmt.Errorf("signalwire: reply carries no sid: %s", strings.TrimSpace(string(raw)))
	}
	return r.SID, nil
}
