package alerts

import "os"

// Config holds alert delivery configuration loaded from environment variables.
// Both delivery channels gracefully no-op when their config is empty,
// so the server starts without alerts configured during development.
type Config struct {
	// ZohoCliqWebhookURL is the incoming webhook URL for the Zoho Cliq channel.
	// If empty, Zoho Cliq delivery is skipped.
	ZohoCliqWebhookURL string

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
		ZohoCliqWebhookURL: os.Getenv("ZOHO_CLIQ_WEBHOOK_URL"),
		SMTPAddr:           os.Getenv("SMTP_ADDR"),
		SMTPFrom:           os.Getenv("SMTP_FROM"),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
	}
}
