package alerts

import "os"

// Config holds alert delivery configuration loaded from environment variables.
// Both delivery channels gracefully no-op when their config is empty,
// so the server starts without alerts configured during development.
type Config struct {
	// Enabled gates ALL outbound delivery from the queue (T-21b decision 46).
	// Only ALERTS_ENABLED=1 — set by docker-compose.prod.yml alone — turns
	// delivery on. Default-off exists because dev servers started from
	// backend/.env hold the SAME live Zoho/SMTP creds as prod and became a
	// second sender to the real Cliq channel (the 7/21 duplicate incident).
	// Transactional email (user invite / password reset in internal/users) is
	// deliberately NOT behind this flag: it is admin-initiated, never
	// duplicated by dev-vs-prod racing the same event, and already no-ops
	// when SMTP config is blank.
	Enabled bool

	// Zoho Cliq OAuth credentials (Self Client flow).
	// If ClientID or RefreshToken is empty, Zoho Cliq delivery is skipped.
	// Future channels can add their own env vars (e.g., ZOHO_CLIQ_OPERATIONS_*).
	ZohoCliqClientID     string
	ZohoCliqClientSecret string
	ZohoCliqRefreshToken string
	ZohoCliqChannel      string // channel name (e.g., "purchaseandinventory")

	// SMTPAddr is the SMTP server address (host:port, e.g. "smtp.sendgrid.net:587").
	// If empty, email delivery is skipped.
	SMTPAddr string

	// SMTPFrom is the sender address for outgoing alert emails.
	SMTPFrom string

	// SMTPUsername and SMTPPassword for SMTP AUTH (optional — some servers use IP allow-list).
	SMTPUsername string
	SMTPPassword string
}

// LoadConfig reads alert configuration from environment variables.
func LoadConfig() Config {
	return Config{
		Enabled:              os.Getenv("ALERTS_ENABLED") == "1",
		ZohoCliqClientID:     os.Getenv("ZOHO_CLIQ_CLIENT_ID"),
		ZohoCliqClientSecret: os.Getenv("ZOHO_CLIQ_CLIENT_SECRET"),
		ZohoCliqRefreshToken: os.Getenv("ZOHO_CLIQ_REFRESH_TOKEN"),
		ZohoCliqChannel:      os.Getenv("ZOHO_CLIQ_PURCHASE_AND_INVENTORY_CHANNEL"),
		SMTPAddr:             os.Getenv("SMTP_ADDR"),
		SMTPFrom:             os.Getenv("SMTP_FROM"),
		SMTPUsername:          os.Getenv("SMTP_USERNAME"),
		SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
	}
}
