package receipt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const receiptParsePrompt = `Parse this receipt image or set of receipt images.

When multiple images are attached, treat them as ONE combined transaction.
If one image is a refund / credit memo / return (negative amounts, REFUND/RETURN/CREDIT headers), include its line items with NEGATIVE prices.
summary.total = signed sum of price*quantity across ALL images + summary.tax.
summary.tax = signed sum of taxes across ALL images.
summary.vendor = vendor from the purchase image.

Return ONLY raw JSON. No markdown code fences. No explanation.

Example:
{"items": [{"name": "...", "quantity": 1, "price": 0.00, "is_case": false}], "summary": {"vendor": "...", "total_units": 0, "total_cases": 0, "tax": 0.00, "total": 0.00}}`

// FileBlob holds raw bytes and MIME type for a single receipt file to be sent
// to Claude. Multiple FileBlobs can be bundled into one prompt so that a
// purchase + refund pair are parsed as a single combined receipt.
type FileBlob struct {
	Bytes       []byte
	ContentType string
}

// ParseReceipt sends receipt file(s) to Claude Haiku and returns the parsed
// line items and summary. Thin wrapper around parseReceiptWithModel so the
// Sonnet fallback (ParseReceiptWithSonnet) can share the same logic with a
// different model + max_tokens. Error wrapping is done by parseReceiptWithModel
// using the supplied label so call-site logs distinguish Haiku vs Sonnet
// failures (the seam in worker.go logs both error strings on double-fail).
func ParseReceipt(ctx context.Context, apiKey string, blobs []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
	return parseReceiptWithModel(ctx, apiKey, blobs, anthropic.ModelClaudeHaiku4_5, 4096, "ParseReceipt")
}

// ParseReceiptWithSonnet is the Sonnet fallback used by the worker when
// Haiku (via ParseReceipt) fails on a receipt. Uses claude-sonnet-4-6 with
// MaxTokens=4096 — Sonnet handles complex multi-page or low-quality PDF
// receipts that Haiku chokes on. Same return shape as ParseReceipt so the
// caller can fall through to ValidateReceiptData on success.
func ParseReceiptWithSonnet(ctx context.Context, apiKey string, blobs []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
	return parseReceiptWithModel(ctx, apiKey, blobs, anthropic.ModelClaudeSonnet4_6, 4096, "ParseReceiptWithSonnet")
}

// parseReceiptWithModel is the shared implementation that both ParseReceipt
// (Haiku) and ParseReceiptWithSonnet (Sonnet) call into. label is used to
// prefix wrapped error messages so worker logs can attribute failures to
// the correct model. blobs holds one entry per attachment; all blobs are
// sent in a single Claude message so multi-receipt transactions (e.g. a
// purchase + refund) are parsed together and produce a single combined total.
func parseReceiptWithModel(ctx context.Context, apiKey string, blobs []FileBlob, model anthropic.Model, maxTokens int64, label string) ([]ReceiptItem, ReceiptSummary, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// Build one content block per blob so all attachments land in the same
	// Claude message. A purchase + refund pair seen together yields a combined
	// net total that ValidateReceiptData can match against the bank amount.
	var contentBlocks []anthropic.ContentBlockParamUnion
	for _, blob := range blobs {
		encoded := base64.StdEncoding.EncodeToString(blob.Bytes)
		if strings.HasPrefix(blob.ContentType, "application/pdf") {
			contentBlocks = append(contentBlocks, anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
				Data: encoded,
			}))
		} else {
			mediaType := normalizeImageMediaType(blob.ContentType)
			contentBlocks = append(contentBlocks, anthropic.NewImageBlockBase64(mediaType, encoded))
		}
	}
	contentBlocks = append(contentBlocks, anthropic.NewTextBlock(receiptParsePrompt))

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			{
				Role:    "user",
				Content: contentBlocks,
			},
		},
	})
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("%s: API call failed: %w", label, err)
	}

	// Extract text from the response
	var rawText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawText += block.Text
		}
	}

	if rawText == "" {
		return nil, ReceiptSummary{}, fmt.Errorf("%s: empty response from API", label)
	}

	items, summary, err := parseJSONBody(rawText)
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("%s: failed to parse JSON body: %w", label, err)
	}

	return items, summary, nil
}

// parseJSONBody extracts and parses the structured receipt JSON from Claude's response.
// Handles both bare JSON and JSON wrapped in markdown code fences (including truncated
// output where the closing ``` is missing — e.g. Haiku at 2048 maxTokens).
func parseJSONBody(text string) ([]ReceiptItem, ReceiptSummary, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result struct {
		Items   []ReceiptItem  `json:"items"`
		Summary ReceiptSummary `json:"summary"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("parseJSONBody: failed to unmarshal: %w (text: %.200s)", err, text)
	}

	return result.Items, result.Summary, nil
}

// ParseReceiptWithFeedback sends the same receipt blobs back to Sonnet with a
// reconciliation prompt that embeds the bank's ground-truth amount so Claude can
// find whatever was missed on the first pass (e.g. a refund / credit memo in a
// second attachment that it ignored). Always uses Sonnet + maxTokens=4096.
//
// A package-level seam var parseReceiptWithFeedback (in worker.go) wraps this
// function so tests can inject a stub without changing this signature.
func ParseReceiptWithFeedback(ctx context.Context, apiKey string, blobs []FileBlob, prevTotal float64, bankAmount float64, validateReason string) ([]ReceiptItem, ReceiptSummary, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	diff := prevTotal - (-bankAmount)

	feedbackPrompt := fmt.Sprintf(`You previously parsed this receipt and returned summary.total = $%.2f.
The bank transaction amount is $%.2f. The receipt total must equal $%.2f.
Discrepancy: $%.2f.

Look at the attached image(s) again. Find what's missing or wrong:
- Is there a refund / credit memo / return whose negative items you skipped?
- If multiple images are attached, did you treat them as one combined transaction?
- Did you misread a digit or miss a discount line?

Re-emit the corrected JSON. summary.total must equal $%.2f.
Same JSON shape as before. No markdown code fences. No explanation.`,
		prevTotal, bankAmount, -bankAmount, diff, -bankAmount)

	var contentBlocks []anthropic.ContentBlockParamUnion
	for _, blob := range blobs {
		encoded := base64.StdEncoding.EncodeToString(blob.Bytes)
		if strings.HasPrefix(blob.ContentType, "application/pdf") {
			contentBlocks = append(contentBlocks, anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
				Data: encoded,
			}))
		} else {
			mediaType := normalizeImageMediaType(blob.ContentType)
			contentBlocks = append(contentBlocks, anthropic.NewImageBlockBase64(mediaType, encoded))
		}
	}
	contentBlocks = append(contentBlocks, anthropic.NewTextBlock(feedbackPrompt))

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			{
				Role:    "user",
				Content: contentBlocks,
			},
		},
	})
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceiptWithFeedback: API call failed: %w", err)
	}

	var rawText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawText += block.Text
		}
	}

	if rawText == "" {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceiptWithFeedback: empty response from API")
	}

	items, summary, err := parseJSONBody(rawText)
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceiptWithFeedback: failed to parse JSON body: %w", err)
	}

	return items, summary, nil
}

// normalizeImageMediaType maps a content type to the values Claude accepts.
func normalizeImageMediaType(contentType string) string {
	// Strip parameters like "; charset=utf-8"
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	switch strings.ToLower(contentType) {
	case "image/png":
		return "image/png"
	case "image/gif":
		return "image/gif"
	case "image/webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
