package inventory

import (
	"context"
	_ "embed"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

//go:embed fixtures/purchase_item_groups.yaml
var fixturesYAML []byte

//go:embed fixtures/purchase_items.yaml
var itemsYAML []byte

type fixturesFile struct {
	Vendors    []vendorFixture    `yaml:"vendors"`
	ItemGroups []itemGroupFixture `yaml:"item_groups"`
}

type vendorFixture struct {
	Name string `yaml:"name"`
}

type itemGroupFixture struct {
	Name          string                `yaml:"name"`
	ParDays       *int                  `yaml:"par_days,omitempty"`
	Tags          []string              `yaml:"tags"`
	PurchaseItems []purchaseItemFixture `yaml:"purchase_items"`
}

type purchaseItemFixture struct {
	Description string `yaml:"description"`
}

type itemsFile struct {
	Items []itemSeedGroup `yaml:"items"`
}

type itemSeedGroup struct {
	Group         string          `yaml:"group"`
	PurchaseItems []itemSeedEntry `yaml:"purchase_items"`
}

type itemSeedEntry struct {
	Description   string `yaml:"description"`
	FullName      string `yaml:"full_name,omitempty"`
	PhotoURL      string `yaml:"photo_url,omitempty"`
	StoreLocation string `yaml:"store_location,omitempty"`
}

// nilIfEmpty converts an empty string to nil, so optional fields become NULL in Postgres.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// SeedInventoryFixtures loads purchase_item_groups.yaml and inserts vendors,
// tags, item_groups, and purchase_items idempotently (ON CONFLICT DO NOTHING).
func SeedInventoryFixtures(ctx context.Context, pool *pgxpool.Pool) error {
	var f fixturesFile
	if err := yaml.Unmarshal(fixturesYAML, &f); err != nil {
		return fmt.Errorf("inventory: parse fixtures yaml: %w", err)
	}

	// Seed vendors
	for _, v := range f.Vendors {
		_, err := pool.Exec(ctx,
			`INSERT INTO vendors (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`,
			v.Name,
		)
		if err != nil {
			return fmt.Errorf("inventory: seed vendor %q: %w", v.Name, err)
		}
	}
	log.Printf("inventory: seeded %d vendor(s)", len(f.Vendors))

	// Collect all unique tags across all item groups
	tagSet := map[string]struct{}{}
	for _, g := range f.ItemGroups {
		for _, t := range g.Tags {
			tagSet[t] = struct{}{}
		}
	}
	for tag := range tagSet {
		_, err := pool.Exec(ctx,
			`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`,
			tag,
		)
		if err != nil {
			return fmt.Errorf("inventory: seed tag %q: %w", tag, err)
		}
	}
	log.Printf("inventory: seeded %d unique tag(s)", len(tagSet))

	// Seed item groups, group-tag links, and purchase items
	for _, g := range f.ItemGroups {
		var groupID string
		err := pool.QueryRow(ctx,
			`INSERT INTO item_groups (name, par_days) VALUES ($1, $2)
			 ON CONFLICT (name) DO NOTHING
			 RETURNING id`,
			g.Name, g.ParDays,
		).Scan(&groupID)
		if err != nil {
			// Row already exists — look it up
			err2 := pool.QueryRow(ctx,
				`SELECT id FROM item_groups WHERE name = $1`, g.Name,
			).Scan(&groupID)
			if err2 != nil {
				return fmt.Errorf("inventory: upsert item_group %q: %w", g.Name, err2)
			}
		}

		// Link tags to this group
		for _, tagName := range g.Tags {
			_, err := pool.Exec(ctx,
				`INSERT INTO item_group_tags (group_id, tag_id)
				 SELECT $1, id FROM tags WHERE name = $2
				 ON CONFLICT DO NOTHING`,
				groupID, tagName,
			)
			if err != nil {
				return fmt.Errorf("inventory: link tag %q to group %q: %w", tagName, g.Name, err)
			}
		}

		// Seed purchase items
		for _, item := range g.PurchaseItems {
			_, err := pool.Exec(ctx,
				`INSERT INTO purchase_items (description, group_id) VALUES ($1, $2)
				 ON CONFLICT (description) DO NOTHING`,
				item.Description, groupID,
			)
			if err != nil {
				return fmt.Errorf("inventory: seed purchase_item %q: %w", item.Description, err)
			}
		}
	}
	log.Printf("inventory: seeded %d item_group(s)", len(f.ItemGroups))

	// Seed purchase items from purchase_items.yaml
	var items itemsFile
	if err := yaml.Unmarshal(itemsYAML, &items); err != nil {
		return fmt.Errorf("inventory: parse items yaml: %w", err)
	}

	itemCount := 0
	for _, g := range items.Items {
		// Look up the group_id by name
		var groupID string
		err := pool.QueryRow(ctx,
			`SELECT id FROM item_groups WHERE name = $1`, g.Group,
		).Scan(&groupID)
		if err != nil {
			log.Printf("inventory: warning: group %q not found, skipping %d items", g.Group, len(g.PurchaseItems))
			continue
		}

		for _, item := range g.PurchaseItems {
			_, err := pool.Exec(ctx,
				`INSERT INTO purchase_items (description, full_name, photo_url, store_location, group_id)
				 VALUES ($1, $2, $3, $4, $5)
				 ON CONFLICT (description) DO UPDATE SET
				   full_name = COALESCE(EXCLUDED.full_name, purchase_items.full_name),
				   photo_url = COALESCE(EXCLUDED.photo_url, purchase_items.photo_url),
				   store_location = COALESCE(EXCLUDED.store_location, purchase_items.store_location),
				   group_id = EXCLUDED.group_id`,
				item.Description, nilIfEmpty(item.FullName), nilIfEmpty(item.PhotoURL), nilIfEmpty(item.StoreLocation), groupID,
			)
			if err != nil {
				return fmt.Errorf("inventory: seed item %q: %w", item.Description, err)
			}
			itemCount++
		}
	}
	log.Printf("inventory: seeded %d purchase item(s) from YAML", itemCount)

	return nil
}
