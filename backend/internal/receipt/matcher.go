package receipt

import (
	"strings"
	"unicode"
)

// tokenMatchThreshold is the minimum token-overlap score required for a
// confident token-based catalog match. Score = (overlapping catalog tokens) /
// (total catalog-name tokens).
//
// 0.5 was chosen because real catalog entries are typically 1–3 content words
// ("Lemonade Mix", "Hoagie Containers", "Chicken Tenders") and SKU-style
// receipt names include the key product noun but not every descriptor word.
// Requiring 50%+ overlap means the PRIMARY noun of the catalog entry must
// appear in the receipt name — which is a meaningful signal — while rejecting
// completely unrelated pairings. Empirically, 0.7 produced 0/45 matches on a
// live Restaurant Depot receipt because the catalog entries have 2 tokens and
// only 1 overlaps with the SKU (0.5 score). AI fallback handles ambiguous
// cases (< 0.5) anyway, so false positives here are limited.
const tokenMatchThreshold = 0.5

// noiseTokens are short or meaningless words stripped before scoring so they
// don't inflate overlap counts or penalize mismatches.
var noiseTokens = map[string]bool{
	"oz": true, "qt": true, "lb": true, "lbs": true, "ct": true, "ea": true,
	"pk": true, "pack": true, "the": true, "and": true, "for": true,
}

// tokenizeForMatch normalizes a name to a set of meaningful tokens: lowercase,
// strip punctuation (replace with spaces), drop short (<3 char), entirely
// numeric, or noise tokens.
func tokenizeForMatch(name string) []string {
	name = strings.ToLower(name)
	// Replace non-alphanumeric, non-space runes with spaces.
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			return r
		}
		return ' '
	}, name)

	rawTokens := strings.Fields(name)
	var out []string
	for _, tok := range rawTokens {
		if len(tok) < 3 {
			continue
		}
		if noiseTokens[tok] {
			continue
		}
		// Skip tokens that are entirely digits (e.g. "35", "12", "99").
		allDigit := true
		for _, r := range tok {
			if !unicode.IsDigit(r) {
				allDigit = false
				break
			}
		}
		if allDigit {
			continue
		}
		out = append(out, tok)
	}
	return out
}

// matchByTokens scores rawName against every entry in existingMap using token
// overlap. Score = (tokens in catalog-entry that appear in rawName) /
// (total meaningful tokens in catalog entry). Returns the best match's catalog
// ID and description when score >= threshold; otherwise returns "", "", score.
func matchByTokens(rawName string, existingMap map[string]string, threshold float64) (id string, name string, score float64) {
	rawTokenSet := make(map[string]bool)
	for _, t := range tokenizeForMatch(rawName) {
		rawTokenSet[t] = true
	}
	if len(rawTokenSet) == 0 {
		return "", "", 0
	}

	var bestID, bestName string
	var bestScore float64

	for desc, itemID := range existingMap {
		catTokens := tokenizeForMatch(desc)
		if len(catTokens) == 0 {
			continue
		}
		overlap := 0
		for _, t := range catTokens {
			if rawTokenSet[t] {
				overlap++
			}
		}
		s := float64(overlap) / float64(len(catTokens))
		if s > bestScore {
			bestScore = s
			bestID = itemID
			bestName = desc
		}
	}

	if bestScore >= threshold {
		return bestID, bestName, bestScore
	}
	return "", "", bestScore
}
