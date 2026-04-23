package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"sync"
	"time"
)

// zohoToken caches the OAuth access token with thread-safe refresh.
type zohoToken struct {
	mu           sync.Mutex
	accessToken  string
	expiresAt    time.Time
	clientID     string
	clientSecret string
	refreshToken string
}

// getAccessToken returns a valid access token, refreshing if expired.
func (t *zohoToken) getAccessToken() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.accessToken != "" && time.Now().Before(t.expiresAt) {
		return t.accessToken, nil
	}

	// Refresh the token
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {t.clientID},
		"client_secret": {t.clientSecret},
		"refresh_token": {t.refreshToken},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://accounts.zoho.com/oauth/v2/token", data)
	if err != nil {
		return "", fmt.Errorf("zoho token refresh: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("zoho token refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"` // seconds
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("zoho token refresh: parse: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("zoho token refresh: empty access_token in response: %s", string(body))
	}

	t.accessToken = result.AccessToken
	// Refresh 5 minutes early to avoid edge-case expiry
	t.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)
	log.Printf("alerts: zoho access token refreshed (expires in %ds)", result.ExpiresIn)

	return t.accessToken, nil
}

// package-level token cache (initialized on first use)
var zohoTokenCache *zohoToken
var zohoTokenOnce sync.Once

func getZohoToken(cfg Config) *zohoToken {
	zohoTokenOnce.Do(func() {
		zohoTokenCache = &zohoToken{
			clientID:     cfg.ZohoCliqClientID,
			clientSecret: cfg.ZohoCliqClientSecret,
			refreshToken: cfg.ZohoCliqRefreshToken,
		}
	})
	return zohoTokenCache
}

// SendZohoCliq delivers a plain-text message to a Zoho Cliq channel via OAuth.
// Returns nil without sending if credentials are not configured (graceful no-op for dev).
func SendZohoCliq(cfg Config, message string) error {
	if cfg.ZohoCliqClientID == "" || cfg.ZohoCliqRefreshToken == "" {
		log.Printf("alerts: zoho_cliq not configured — skipping delivery")
		return nil
	}
	if cfg.ZohoCliqChannel == "" {
		log.Printf("alerts: zoho_cliq channel not set — skipping delivery")
		return nil
	}

	token := getZohoToken(cfg)
	accessToken, err := token.getAccessToken()
	if err != nil {
		return fmt.Errorf("SendZohoCliq: %w", err)
	}

	apiURL := fmt.Sprintf("https://cliq.zoho.com/api/v2/channelsbyname/%s/message", cfg.ZohoCliqChannel)
	payload := map[string]string{"text": message}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("SendZohoCliq: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("SendZohoCliq: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SendZohoCliq: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SendZohoCliq: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SendEmail delivers an alert via SMTP.
// Returns nil without sending if smtpAddr or from is empty (graceful no-op for dev).
func SendEmail(smtpAddr, username, password, from, to, subject, body string) error {
	if smtpAddr == "" || from == "" {
		log.Printf("alerts: SMTP not configured — skipping email to %s", to)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)

	var auth smtp.Auth
	if username != "" && password != "" {
		host, _, _ := splitHostPort(smtpAddr)
		auth = smtp.PlainAuth("", username, password, host)
	}

	if err := smtp.SendMail(smtpAddr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("SendEmail to %s: %w", to, err)
	}
	return nil
}

// splitHostPort extracts the host from "host:port" or returns the input unchanged.
func splitHostPort(addr string) (host, port string, err error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return addr, "", nil
}
