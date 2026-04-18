package receipt

import (
	"strings"

	"github.com/antzucaro/matchr"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const jaroWinklerThreshold = 0.85

// DerivePurchaseItemID finds the best matching purchase item ID for rawName
// using exact case-insensitive match first, then Jaro-Winkler fuzzy match
// at 0.85 threshold. If no match is found, returns a title-cased version
// of rawName as the new item name with isNew=true.
//
// existingMap is keyed by description (the stored item name), value is the
// item's UUID in the DB.
func DerivePurchaseItemID(rawName string, existingMap map[string]string) (id string, name string, isNew bool) {
	rawLower := strings.ToLower(rawName)

	// Step 1: exact case-insensitive match
	for desc, itemID := range existingMap {
		if strings.ToLower(desc) == rawLower {
			return itemID, desc, false
		}
	}

	// Step 2: Jaro-Winkler fuzzy match at 0.85 threshold
	var bestMatch string
	var bestID string
	var highestScore float64

	for desc, itemID := range existingMap {
		score := matchr.JaroWinkler(rawLower, strings.ToLower(desc), true)
		if score > highestScore {
			highestScore = score
			bestMatch = desc
			bestID = itemID
		}
	}

	if highestScore >= jaroWinklerThreshold {
		return bestID, bestMatch, false
	}

	// Step 3: no match — title-case the raw name for the new entry
	titleName := cases.Title(language.English).String(rawName)
	return "", titleName, true
}
