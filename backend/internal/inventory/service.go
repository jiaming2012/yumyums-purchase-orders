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
			 ON CONFLICT DO NOTHING
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

	return nil
}
