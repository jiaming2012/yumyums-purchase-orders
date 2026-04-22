// import-notion converts a Notion CSV catalog export into a purchase_items.yaml seed file.
//
// Usage:
//
//	go run ./cmd/import-notion/ \
//	  --csv "/path/to/All Items.csv" \
//	  --images "/path/to/All Items/" \
//	  --output "internal/inventory/fixtures/purchase_items.yaml"
//
// Environment variables for DO Spaces upload:
//
//	DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_ENDPOINT, DO_SPACES_REGION, DO_SPACES_BUCKET
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/yumyums/hq/internal/photos"
)

// categoryMapping maps Notion category values to existing/new group names (per D-01).
var categoryMapping = map[string]string{
	"Produce":          "Produce",
	"Bread":            "Bread",
	"Dairy":            "Dairy",
	"Frozen":           "Proteins",
	"Meat":             "Proteins",
	"Frozen / Meat":    "Proteins",
	"Drinks":           "Beverages",
	"Cleaning":         "Cleaning Supplies",
	"Cooking Supplies": "Packaging",
	"Seasoning":        "Seasoning",
	"Dry Groceries":    "Sauces",
	"Sauces":           "Sauces",
	"Dry Groceries / Sauces": "Sauces",
}

const defaultGroup = "Dry Goods"

// outputItem is the YAML structure written to purchase_items.yaml.
type outputItem struct {
	Description   string `yaml:"description"`
	FullName      string `yaml:"full_name,omitempty"`
	PhotoURL      string `yaml:"photo_url,omitempty"`
	StoreLocation string `yaml:"store_location,omitempty"`
}

type outputGroup struct {
	Group         string       `yaml:"group"`
	PurchaseItems []outputItem `yaml:"purchase_items"`
}

type outputFile struct {
	Items []outputGroup `yaml:"items"`
}

// titleCase normalizes item names matching normalizeItemName() in handler.go.
func titleCase(s string) string {
	return cases.Title(language.English).String(strings.ToLower(strings.TrimSpace(s)))
}

// toSlug converts a description to a URL-safe key (lowercase, hyphens, no special chars).
var nonAlphanumHyphen = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(s string) string {
	lower := strings.ToLower(s)
	slug := nonAlphanumHyphen.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func main() {
	csvPath := flag.String("csv", "", "Path to Notion CSV export file (required)")
	imagesDir := flag.String("images", "", "Path to 'All Items/' image directory (required)")
	outputPath := flag.String("output", "internal/inventory/fixtures/purchase_items.yaml", "Output YAML path")
	dryRun := flag.Bool("dry-run", false, "Skip DO Spaces upload, use placeholder URLs")
	flag.Parse()

	if *csvPath == "" || *imagesDir == "" {
		flag.Usage()
		log.Fatal("--csv and --images are required")
	}

	ctx := context.Background()

	// Set up S3 client for DO Spaces (only needed if not dry-run)
	var s3Client *s3.Client
	var spacesEndpoint, spacesBucket string

	if !*dryRun {
		key := os.Getenv("DO_SPACES_KEY")
		secret := os.Getenv("DO_SPACES_SECRET")
		endpoint := os.Getenv("DO_SPACES_ENDPOINT")
		region := os.Getenv("DO_SPACES_REGION")
		bucket := os.Getenv("DO_SPACES_BUCKET")

		if key == "" || secret == "" || endpoint == "" || region == "" || bucket == "" {
			log.Fatal("DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_ENDPOINT, DO_SPACES_REGION, DO_SPACES_BUCKET must be set (or use --dry-run)")
		}

		spacesEndpoint = endpoint
		spacesBucket = bucket

		s3Client = s3.New(s3.Options{
			Region:       region,
			Credentials:  credentials.NewStaticCredentialsProvider(key, secret, ""),
			BaseEndpoint: aws.String(endpoint),
			UsePathStyle: true,
		})
	}

	// Parse CSV
	f, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		log.Fatalf("read csv header: %v", err)
	}

	// Build column index map
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[strings.TrimSpace(h)] = i
	}

	// Verify expected columns exist
	for _, col := range []string{"Name", "Category", "Full Name", "Photo", "Store"} {
		if _, ok := colIdx[col]; !ok {
			log.Fatalf("CSV missing expected column %q. Found: %v", col, headers)
		}
	}

	// grouped maps group name -> list of items
	grouped := map[string][]outputItem{}
	groupOrder := []string{} // preserve insertion order
	seenGroups := map[string]bool{}

	totalItems := 0
	itemsWithPhotos := 0
	itemsWithoutPhotos := 0

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read csv row: %v", err)
		}

		get := func(col string) string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		name := get("Name")
		fullName := get("Full Name")
		category := get("Category")
		photoVal := get("Photo")
		store := get("Store")

		// D-09: If Name is empty, fall back to Full Name for description
		description := titleCase(name)
		if description == "" {
			description = titleCase(fullName)
		}
		if description == "" {
			// Skip rows with no usable name
			continue
		}

		// Normalize full_name with title case
		fullNameNorm := titleCase(fullName)

		// D-13: Store location
		storeLocation := titleCase(store)

		// D-01: Map category to group
		group := mapCategory(category)

		// Track group order
		if !seenGroups[group] {
			seenGroups[group] = true
			groupOrder = append(groupOrder, group)
		}

		// Handle photo upload
		photoURL := ""
		if photoVal != "" {
			// URL-decode the photo path (e.g. "All%20Items/Brisket%20Rolls/screenshot.png")
			decoded, err := url.QueryUnescape(photoVal)
			if err != nil {
				log.Printf("WARN: url-decode photo %q: %v", photoVal, err)
				decoded = photoVal
			}

			// Strip "All Items/" prefix — images dir IS the "All Items/" directory
			relativePath := strings.TrimPrefix(decoded, "All Items/")
			absImagePath := filepath.Join(*imagesDir, relativePath)

			if fileExists(absImagePath) {
				slug := toSlug(description)
				s3Key := fmt.Sprintf("items/%s.png", slug)

				if *dryRun {
					photoURL = fmt.Sprintf("https://placeholder.digitaloceanspaces.com/%s", s3Key)
					itemsWithPhotos++
				} else {
					uploadedURL, err := uploadToSpaces(ctx, s3Client, spacesEndpoint, spacesBucket, s3Key, absImagePath)
					if err != nil {
						log.Printf("WARN: upload %q: %v — skipping photo", description, err)
						itemsWithoutPhotos++
					} else {
						photoURL = uploadedURL
						itemsWithPhotos++
						log.Printf("  uploaded: %s -> %s", filepath.Base(absImagePath), s3Key)
					}
				}
			} else {
				log.Printf("WARN: image file not found: %s", absImagePath)
				itemsWithoutPhotos++
			}
		} else {
			itemsWithoutPhotos++
		}

		item := outputItem{
			Description:   description,
			FullName:      fullNameNorm,
			PhotoURL:      photoURL,
			StoreLocation: storeLocation,
		}

		grouped[group] = append(grouped[group], item)
		totalItems++
	}

	// Build output grouped by order of first appearance
	out := outputFile{}
	for _, group := range groupOrder {
		out.Items = append(out.Items, outputGroup{
			Group:         group,
			PurchaseItems: grouped[group],
		})
	}

	// Marshal YAML
	var buf bytes.Buffer
	buf.WriteString("# Generated from Notion CSV export. DO NOT EDIT.\n")
	buf.WriteString("# Re-run: go run ./cmd/import-notion/ --csv ... --images ... --output ...\n")

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		log.Fatalf("marshal yaml: %v", err)
	}

	// Write output file
	if err := os.WriteFile(*outputPath, buf.Bytes(), 0644); err != nil {
		log.Fatalf("write output: %v", err)
	}

	// Print summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total items:        %d\n", totalItems)
	fmt.Printf("  Items with photos:  %d\n", itemsWithPhotos)
	fmt.Printf("  Items without:      %d\n", itemsWithoutPhotos)
	fmt.Printf("  Groups:\n")
	for _, group := range groupOrder {
		fmt.Printf("    %-25s %d items\n", group+":", len(grouped[group]))
	}
	fmt.Printf("\nOutput written to: %s\n", *outputPath)
}

func mapCategory(category string) string {
	category = strings.TrimSpace(category)
	if group, ok := categoryMapping[category]; ok {
		return group
	}
	// Check for partial matches (e.g. "Frozen / Meat" combined variants)
	for notionCat, group := range categoryMapping {
		if strings.EqualFold(category, notionCat) {
			return group
		}
	}
	if category == "" {
		return defaultGroup
	}
	// Unknown category → default
	log.Printf("WARN: unknown category %q, assigning to %q", category, defaultGroup)
	return defaultGroup
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func uploadToSpaces(ctx context.Context, client *s3.Client, endpoint, bucket, key, filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filePath, err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("image/png"),
		ACL:         s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("put object %s: %w", key, err)
	}

	return photos.PublicURL(endpoint, bucket, key), nil
}
