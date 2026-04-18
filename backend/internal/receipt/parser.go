package receipt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const receiptParsePrompt = `Parse this receipt. Return ONLY a JSON object, no markdown, no explanation: {"items": [{"name": "...", "quantity": 1, "price": 0.00, "is_case": false}], "summary": {"vendor": "...", "total_units": 0, "total_cases": 0, "tax": 0.00, "total": 0.00}}`

var jsonFenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

// ParseReceipt sends a receipt file to Claude Haiku and returns the parsed
// line items and summary.
func ParseReceipt(ctx context.Context, apiKey string, fileBytes []byte, contentType string) ([]ReceiptItem, ReceiptSummary, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	// Build the content block for the receipt file
	var contentBlock anthropic.ContentBlockParamUnion
	encoded := base64.StdEncoding.EncodeToString(fileBytes)

	if strings.HasPrefix(contentType, "application/pdf") {
		contentBlock = anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
			Data: encoded,
		})
	} else {
		// Treat as image — use content type as media type
		mediaType := normalizeImageMediaType(contentType)
		contentBlock = anthropic.NewImageBlockBase64(mediaType, encoded)
	}

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			{
				Role: "user",
				Content: []anthropic.ContentBlockParamUnion{
					contentBlock,
					anthropic.NewTextBlock(receiptParsePrompt),
				},
			},
		},
	})
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceipt: API call failed: %w", err)
	}

	// Extract text from the response
	var rawText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawText += block.Text
		}
	}

	if rawText == "" {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceipt: empty response from API")
	}

	items, summary, err := parseJSONBody(rawText)
	if err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("ParseReceipt: failed to parse JSON body: %w", err)
	}

	return items, summary, nil
}

// parseJSONBody extracts and parses the structured receipt JSON from Claude's response.
// Handles both bare JSON and JSON wrapped in markdown code fences.
func parseJSONBody(text string) ([]ReceiptItem, ReceiptSummary, error) {
	text = strings.TrimSpace(text)

	// Try to extract JSON from markdown code fence first
	if matches := jsonFenceRe.FindStringSubmatch(text); len(matches) >= 2 {
		text = strings.TrimSpace(matches[1])
	}

	var result struct {
		Items   []ReceiptItem  `json:"items"`
		Summary ReceiptSummary `json:"summary"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, ReceiptSummary{}, fmt.Errorf("parseJSONBody: failed to unmarshal: %w (text: %.200s)", err, text)
	}

	return result.Items, result.Summary, nil
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
