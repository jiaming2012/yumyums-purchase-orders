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
// line items and summary. Thin wrapper around parseReceiptWithModel so the
// Sonnet fallback (ParseReceiptWithSonnet) can share the same logic with a
// different model + max_tokens. Error wrapping is done by parseReceiptWithModel
// using the supplied label so call-site logs distinguish Haiku vs Sonnet
// failures (the seam in worker.go logs both error strings on double-fail).
func ParseReceipt(ctx context.Context, apiKey string, fileBytes []byte, contentType string) ([]ReceiptItem, ReceiptSummary, error) {
	return parseReceiptWithModel(ctx, apiKey, fileBytes, contentType, anthropic.ModelClaudeHaiku4_5, 2048, "ParseReceipt")
}

// ParseReceiptWithSonnet is the Sonnet fallback used by the worker when
// Haiku (via ParseReceipt) fails on a receipt. Uses claude-sonnet-4-6 with
// MaxTokens=4096 — Sonnet handles complex multi-page or low-quality PDF
// receipts that Haiku chokes on. Same return shape as ParseReceipt so the
// caller can fall through to ValidateReceiptData on success.
func ParseReceiptWithSonnet(ctx context.Context, apiKey string, fileBytes []byte, contentType string) ([]ReceiptItem, ReceiptSummary, error) {
	return parseReceiptWithModel(ctx, apiKey, fileBytes, contentType, anthropic.ModelClaudeSonnet4_6, 4096, "ParseReceiptWithSonnet")
}

// parseReceiptWithModel is the shared implementation that both ParseReceipt
// (Haiku) and ParseReceiptWithSonnet (Sonnet) call into. label is used to
// prefix wrapped error messages so worker logs can attribute failures to
// the correct model.
func parseReceiptWithModel(ctx context.Context, apiKey string, fileBytes []byte, contentType string, model anthropic.Model, maxTokens int64, label string) ([]ReceiptItem, ReceiptSummary, error) {
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
		Model:     model,
		MaxTokens: maxTokens,
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
