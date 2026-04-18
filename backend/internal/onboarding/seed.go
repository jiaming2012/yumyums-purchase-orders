package onboarding

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedOnboardingTemplates inserts seed onboarding templates if they do not exist.
// Idempotent: skips any template whose name already exists in ob_templates.
func SeedOnboardingTemplates(ctx context.Context, pool *pgxpool.Pool) error {
	kitchenBasics := CreateTemplateInput{
		Name:  "Kitchen Basics Training",
		Roles: nil,
		Sections: []CreateSectionInput{
			{
				Title:           "Safety & Hygiene",
				SortOrder:       1,
				RequiresSignOff: true,
				IsFaq:           false,
				Items: []CreateItemInput{
					{Type: "checkbox", Label: "Wash hands for 20 seconds before handling food", SortOrder: 1},
					{Type: "checkbox", Label: "Wear gloves when handling ready-to-eat food", SortOrder: 2},
					{Type: "checkbox", Label: "Keep raw meat separate from ready-to-eat food", SortOrder: 3},
					{Type: "checkbox", Label: "Check food temperatures with thermometer", SortOrder: 4},
				},
			},
			{
				Title:           "Equipment Training",
				SortOrder:       2,
				RequiresSignOff: true,
				IsFaq:           false,
				Items: []CreateItemInput{
					{
						Type:      "video_series",
						Label:     "Grill Operation",
						SortOrder: 1,
						VideoParts: []CreateVideoPartInput{
							{Title: "Pre-heat Procedure", Description: "How to safely pre-heat the grill", URL: "https://placeholder.example/grill-preheat", SortOrder: 1},
							{Title: "Temperature Control", Description: "Maintaining proper cooking temperatures", URL: "https://placeholder.example/grill-temp", SortOrder: 2},
							{Title: "Cleaning & Maintenance", Description: "Post-service grill cleaning procedures", URL: "https://placeholder.example/grill-clean", SortOrder: 3},
						},
					},
					{Type: "checkbox", Label: "Can identify all grill safety features", SortOrder: 2},
					{Type: "checkbox", Label: "Knows emergency shutoff procedure", SortOrder: 3},
				},
			},
			{
				Title:           "Menu Knowledge",
				SortOrder:       3,
				RequiresSignOff: false,
				IsFaq:           false,
				Items: []CreateItemInput{
					{Type: "checkbox", Label: "Can describe all menu items to customers", SortOrder: 1},
					{Type: "checkbox", Label: "Knows common allergens in each dish", SortOrder: 2},
					{Type: "checkbox", Label: "Understands daily specials rotation", SortOrder: 3},
				},
			},
			{
				Title:           "FAQ",
				SortOrder:       4,
				RequiresSignOff: false,
				IsFaq:           true,
				Items: []CreateItemInput{
					{
						Type:      "faq",
						Label:     "What do I do if I cut myself?",
						Answer:    strPtr("Immediately stop work, apply first aid from the kit under the service window, and notify your supervisor. Do not handle food until the wound is fully covered."),
						SortOrder: 1,
					},
					{
						Type:      "faq",
						Label:     "Where are the first aid supplies?",
						Answer:    strPtr("Under the service window in the labeled red kit. Check that it is restocked at the start of every shift."),
						SortOrder: 2,
					},
					{
						Type:      "faq",
						Label:     "What if a customer has an allergic reaction?",
						Answer:    strPtr("Alert the manager immediately. Do not leave the customer alone. If severe, call 911. Document the incident using the Yumyums HQ app."),
						SortOrder: 3,
					},
				},
			},
		},
	}

	if err := seedTemplate(ctx, pool, kitchenBasics); err != nil {
		return fmt.Errorf("seed Kitchen Basics Training: %w", err)
	}

	return nil
}

// seedTemplate inserts a single onboarding template if it does not already exist.
func seedTemplate(ctx context.Context, pool *pgxpool.Pool, input CreateTemplateInput) error {
	var existing int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ob_templates WHERE name = $1`,
		input.Name,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("check existing ob_template: %w", err)
	}
	if existing > 0 {
		log.Printf("Onboarding template %q already exists, skipping", input.Name)
		return nil
	}

	id, err := CreateTemplate(ctx, pool, input)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	log.Printf("Seeded onboarding template %q (id=%s)", input.Name, id)
	return nil
}

// strPtr returns a pointer to the given string — helper for optional string fields.
func strPtr(s string) *string {
	return &s
}
