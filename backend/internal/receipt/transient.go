package receipt

import (
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// Transient Anthropic failures: the receipt is fine, the API was not.
//
// The worker parks a receipt (stores parse_error, never re-parses it) after
// BOTH parse attempts fail, so a receipt the models genuinely cannot read does
// not burn tokens forever. A billing wall, a rate limit or an outage is not
// that — the next poll after the condition clears would succeed. Those land
// with a `transient: <reason>` marker instead: the card shows the reason in
// words, and classifyExistingTx treats the row as retryable.

// transientMarker prefixes pending_purchases.parse_error for retryable rows.
const transientMarker = "transient: "

// isTransientAPIError reports whether err is an Anthropic failure that will
// clear on its own, with a short human reason for the pending card.
func isTransientAPIError(err error) (bool, string) {
	if err == nil {
		return false, ""
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 429:
			return true, "Anthropic rate limit hit"
		case apiErr.StatusCode == 529:
			return true, "Anthropic API overloaded"
		case apiErr.StatusCode >= 500:
			return true, "Anthropic API error"
		case apiErr.StatusCode == 400 && isCreditBalanceText(apiErr.RawJSON()):
			return true, "Anthropic account out of credits"
		}
		return false, ""
	}
	// Untyped (wrapped or stubbed): fall back to the message text. The SDK's
	// own Error() embeds the response body, so the billing sentence survives
	// every fmt.Errorf("%w") layer the worker adds.
	if isCreditBalanceText(err.Error()) {
		return true, "Anthropic account out of credits"
	}
	return false, ""
}

func isCreditBalanceText(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "credit balance") || strings.Contains(s, "billing_error")
}

// transientParseMarker returns the marker to store when a parse attempt pair
// failed, or "" when neither failure was transient (store the raw errors).
// Either attempt being transient is enough: if the retry hit a billing wall,
// nothing is known about the receipt yet.
func transientParseMarker(primaryErr, retryErr error) string {
	if ok, reason := isTransientAPIError(retryErr); ok {
		return transientMarker + reason
	}
	if ok, reason := isTransientAPIError(primaryErr); ok {
		return transientMarker + reason
	}
	return ""
}
