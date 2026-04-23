package alerts

// Channel constants match the database CHECK constraint values.
const (
	ChannelZohoCliq = "zoho_cliq"
	ChannelEmail    = "email"
)

// AlertType identifies the kind of notification being sent.
const (
	TypeCutoffReminder    = "cutoff_reminder"
	TypeShoppingComplete  = "shopping_complete"
)

// Alert is the payload dispatched to the async queue.
// Channel and RecipientEmail are resolved by the caller before enqueue.
type Alert struct {
	// Channel is the delivery method for this alert (ChannelZohoCliq or ChannelEmail).
	Channel string

	// RecipientEmail is used when Channel == ChannelEmail.
	RecipientEmail string

	// Subject is used as the email subject line (ignored for Zoho Cliq).
	Subject string

	// Message is the plain-text body for both channels.
	Message string
}
