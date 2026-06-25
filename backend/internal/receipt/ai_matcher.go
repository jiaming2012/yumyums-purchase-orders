package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ItemMatch represents one AI-resolved mapping from a receipt name to a
// catalog entry.
type ItemMatch struct {
	RawName     string `json:"raw_name"`
	CatalogName string `json:"catalog_name"`
	Confidence  string `json:"confidence"` // "high" | "medium" | "low" | "none"
}

// matchItemsWithAI is a package-level seam var so tests can inject a stub
// without changing the exported function signature.
//
// Production code calls MatchItemsWithAI directly via this var. Tests save
// the original, replace with a stub in t.Cleanup.
var matchItemsWithAI = MatchItemsWithAI

// MatchItemsWithAI sends unmatched receipt item names + the full catalog list
// to Claude Haiku and returns a map of raw_name → catalog_item_id for matches
// marked confidence == "high". Lower-confidence matches are dropped — the
// human will hand-match in the FE. Errors are non-fatal; the caller logs and
// proceeds with partial matches.
func MatchItemsWithAI(ctx context.Context, apiKey string, unmatchedNames []string, catalog map[string]string) (map[string]string, error) {
	if len(unmatchedNames) == 0 {
		return nil, nil
	}
	if apiKey == "" {
		return nil, nil
	}

	// Build sorted catalog list for reproducible prompts (map iteration is
	// random in Go — determinism matters so Claude sees a stable input).
	catalogNames := make([]string, 0, len(catalog))
	for name := range catalog {
		catalogNames = append(catalogNames, name)
	}
	sort.Strings(catalogNames)

	systemPrompt := `You are matching receipt item names (often vendor SKUs or abbreviations) to a catalog of pantry/supply items.

Return JSON with one entry per receipt name showing the best catalog match.
Mark confidence "high" ONLY when you are certain the receipt item and catalog item are the same product.
Use "none" if there is no plausible match.
Do not invent catalog names — use only names from the provided catalog list.

Response format (raw JSON, no markdown fences):
{"matches":[{"raw_name":"...","catalog_name":"...","confidence":"high|medium|low|none"}]}`

	userMsg := fmt.Sprintf("Receipt item names to match:\n%s\n\nCatalog (use exact names from this list):\n%s",
		strings.Join(unmatchedNames, "\n"),
		strings.Join(catalogNames, "\n"),
	)

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			{Role: "user", Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(userMsg),
			}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("MatchItemsWithAI: API call failed: %w", err)
	}

	var rawText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawText += block.Text
		}
	}
	if rawText == "" {
		return nil, fmt.Errorf("MatchItemsWithAI: empty response from API")
	}

	// Strip optional markdown code fences — same pattern as parseJSONBody.
	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	var response struct {
		Matches []ItemMatch `json:"matches"`
	}
	if err := json.Unmarshal([]byte(rawText), &response); err != nil {
		return nil, fmt.Errorf("MatchItemsWithAI: unmarshal failed: %w (text: %.200s)", err, rawText)
	}

	// Build result: only high-confidence matches, using catalog to look up ID.
	result := make(map[string]string)
	for _, m := range response.Matches {
		if m.Confidence != "high" {
			continue
		}
		id, ok := catalog[m.CatalogName]
		if !ok {
			// AI returned a name not in the catalog — skip rather than corrupt.
			log.Printf("receipt worker: AI matcher returned unknown catalog name %q for %q (skipping)", m.CatalogName, m.RawName)
			continue
		}
		result[m.RawName] = id
	}
	return result, nil
}
