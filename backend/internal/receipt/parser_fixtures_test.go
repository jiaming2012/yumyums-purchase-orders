package receipt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseJSONBody_Fixtures drives parseJSONBody against captured LLM
// responses stored under testdata/llm_responses/. Each .txt file is a
// verbatim LLM response body; its sibling file determines the expected
// outcome:
//
//	<name>.expected.json    → parseJSONBody must succeed; output deep-
//	                          equal to the expected JSON
//	<name>.expected-err.txt → parseJSONBody must err; err.Error() must
//	                          contain the first non-empty line of the
//	                          .expected-err.txt file as a substring
//
// No live Anthropic calls — these tests are deterministic, fast, free.
// Future Anthropic schema drift surfaces here, not in production.
//
// Phase 260607-k1n: the 05_with_leading_prose fixture locks in current
// behavior — parseJSONBody only strips markdown fences via jsonFenceRe;
// bare leading prose fails json.Unmarshal. Improving the parser's
// leading-prose robustness is explicitly out of scope (L4+).
func TestParseJSONBody_Fixtures(t *testing.T) {
	const dir = "testdata/llm_responses"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read testdata dir: %v", err)
	}

	var ran int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		// Skip sibling .expected-err.txt files — only iterate the input files.
		if strings.HasSuffix(e.Name(), ".expected-err.txt") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".txt")
		inputPath := filepath.Join(dir, e.Name())
		okPath := filepath.Join(dir, name+".expected.json")
		errPath := filepath.Join(dir, name+".expected-err.txt")

		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read input %s: %v", inputPath, err)
			}

			items, summary, parseErr := parseJSONBody(string(body))

			if expBytes, err := os.ReadFile(okPath); err == nil {
				// Success case.
				if parseErr != nil {
					t.Fatalf("parseJSONBody returned error (expected success): %v", parseErr)
				}
				var expected struct {
					Items   []ReceiptItem  `json:"items"`
					Summary ReceiptSummary `json:"summary"`
				}
				if err := json.Unmarshal(expBytes, &expected); err != nil {
					t.Fatalf("parse expected %s: %v", okPath, err)
				}
				// Round-trip comparison via JSON — robust to float
				// representation differences (e.g. 3.5 vs 3.50).
				gotJSON, _ := json.Marshal(map[string]interface{}{
					"items": items, "summary": summary,
				})
				wantJSON, _ := json.Marshal(map[string]interface{}{
					"items": expected.Items, "summary": expected.Summary,
				})
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("parsed result mismatch\n got:  %s\n want: %s", gotJSON, wantJSON)
				}
				ran++
				return
			}

			if errBytes, err := os.ReadFile(errPath); err == nil {
				// Error case.
				needle := firstNonEmptyLine(errBytes)
				if parseErr == nil {
					t.Fatalf("parseJSONBody returned nil error (expected error containing %q)", needle)
				}
				if needle == "" {
					t.Fatalf("%s is empty — must contain expected-error substring on line 1", errPath)
				}
				if !strings.Contains(parseErr.Error(), needle) {
					t.Errorf("error %q does not contain %q", parseErr.Error(), needle)
				}
				ran++
				return
			}

			t.Fatalf("fixture %s has neither %s nor %s sibling — add one",
				inputPath, okPath, errPath)
		})
	}

	if ran < 5 {
		t.Errorf("ran %d fixtures, expected at least 5 (corpus shrunk unexpectedly)", ran)
	}
}

func firstNonEmptyLine(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
