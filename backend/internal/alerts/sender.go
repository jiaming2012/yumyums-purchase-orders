package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

// SendZohoCliq delivers a plain-text message via Zoho Cliq incoming webhook.
// Returns nil without sending if webhookURL is empty (graceful no-op for dev).
func SendZohoCliq(webhookURL, message string) error {
	if webhookURL == "" {
		log.Printf("alerts: zoho_cliq webhook not configured — skipping delivery")
		return nil
	}

	payload := map[string]string{"text": message}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("SendZohoCliq: marshal: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("SendZohoCliq: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("SendZohoCliq: unexpected status %d", resp.StatusCode)
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
