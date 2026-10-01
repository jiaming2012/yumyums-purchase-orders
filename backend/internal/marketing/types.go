package marketing

import "time"

// The wire shapes are handoff §5's, field-for-field. UI code reads these json
// tags verbatim (docs/ui-design-rules.md UI-R: "frontend uses the Go struct's
// json tags verbatim"), so renaming a tag here is a UI change.

// itemRef is the `item:{id,name}|null` shape on a campaign.
type itemRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// funnelDTO is scans → signups → redeemed.
//
// Only Scans is real tonight. Signups needs `subscribers` (migration 0085, card
// H5) and Redeemed needs `scan_attempts_mirror` (migration 0084, card H3a);
// both are shipped as 0 and stated in the merge-intent rather than omitted, so
// H2 can render the final shape today.
type funnelDTO struct {
	Scans    int `json:"scans"`
	Signups  int `json:"signups"`
	Redeemed int `json:"redeemed"`
}

// moneyDTO is decision 190's money block.
//
// 🛑 THE ARITHMETIC IS CARD H3b's. This card ships the SHAPE as the zero value
// the slate specifies — {0, 0, "implied", 0, null} — because revenue needs
// toast_orders and discount needs scan_attempts_mirror, neither of which exists
// until migration 0084. DiscountBasis is a LABEL, never a choice of arithmetic
// (decision 190), and "implied" is the honest label for a period with no
// matched orders at all.
type moneyDTO struct {
	RevenueCents  int      `json:"revenue_cents"`
	DiscountCents int      `json:"discount_cents"`
	DiscountBasis string   `json:"discount_basis"`
	NetCents      int      `json:"net_cents"`
	PerDollar     *float64 `json:"per_dollar"`

	// Detail-route-only additions (§5 row 3). Null for the same reason.
	AvgOrderCentsWith    *int `json:"avg_order_cents_with,omitempty"`
	AvgOrderCentsWithout *int `json:"avg_order_cents_without,omitempty"`
}

// zeroMoney is the H3b placeholder, in one place so the list and detail routes
// cannot drift into two different zero shapes.
func zeroMoney() moneyDTO {
	return moneyDTO{DiscountBasis: "implied", PerDollar: nil}
}

// codeDTO is one qr_codes row.
type codeDTO struct {
	ID           string  `json:"id"`
	Short        string  `json:"short"`
	CampaignID   string  `json:"campaign_id"`
	Channel      string  `json:"channel"`
	ChannelLabel *string `json:"channel_label"`
	ItemID       *string `json:"item_id"`
	Placement    *string `json:"placement"`
	Variant      *string `json:"variant"`
	Landing      *string `json:"landing"`
	Active       bool    `json:"active"`
	V            int     `json:"v"`
	Scans        int     `json:"scans"`
	Signups      int     `json:"signups"`
	// PayloadURL is what the QR encodes, so the create sheet can render the
	// payload preview without rebuilding the host on the client.
	PayloadURL string `json:"payload_url"`
	PNGURL     string `json:"png_url"`
}

// campaignDTO is one campaigns_admin row with its funnel, money and codes.
type campaignDTO struct {
	ID             string     `json:"id"`
	Slug           string     `json:"slug"`
	Name           string     `json:"name"`
	OfferText      string     `json:"offer_text"`
	FaceValueCents int        `json:"face_value_cents"`
	RequiresOnline bool       `json:"requires_online"`
	Item           *itemRef   `json:"item"`
	Landing        string     `json:"landing"`
	Status         string     `json:"status"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         time.Time  `json:"ends_at"`
	ProjectedAt    *time.Time `json:"projected_at"`
	Funnel         funnelDTO  `json:"funnel"`
	Money          moneyDTO   `json:"money"`
	Codes          []codeDTO  `json:"codes"`
}

type listCampaignsResponse struct {
	Campaigns []campaignDTO `json:"campaigns"`
}

// createCampaignResponse is §5 row 2's `201 {campaign, codes:[…]}`, plus the
// warnings array decision 187 requires. Warnings is omitted when empty so a
// fully-projected create carries no noise.
type createCampaignResponse struct {
	Campaign campaignDTO `json:"campaign"`
	Codes    []codeDTO   `json:"codes"`
	Warnings []string    `json:"warnings,omitempty"`
}
