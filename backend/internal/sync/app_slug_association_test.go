package sync

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ════════════════════════════════════════════════════════════════════════════
// Card `app-slug-association` (Card 8, run 20260901) — closes B-160 / E-KR4.
//
// HQ used to store NO template→app association, so `app_slug` was a hardcoded
// CONSTANT in the one sync projection writer (spikec_relay.go, banner finding
// #4). Every projected row claimed the same app. This card puts the association
// in the schema (migration 0076: checklist_templates.app_id → hq_apps) and makes
// the writer resolve each projected row's app_slug from its template's real app.
//
// The two tests below are the card's RED-FIRST evidence:
//
//   1. TestProjectionWriterHasNoHardcodedAppSlug — a STRUCTURAL test. It parses
//      spikec_relay.go and asserts there is no `AppSlug` field on the relay
//      config and no `.AppSlug` selector left in the writer. On the pre-change
//      tree the config field and its three uses exist, so this test is RED;
//      after the change they are gone and it is GREEN. This is the grep-provable
//      "0 hardcoded app_slug constants remain" guard.
//
//   2. TestAppSlugResolvesPerTemplateAssociation — a BEHAVIOURAL test. It seeds
//      two templates on two different apps and asserts each template's response
//      resolves to ITS OWN app's slug via appSlugForField. Against a constant
//      the two would resolve identically; against the association they differ.
// ════════════════════════════════════════════════════════════════════════════

// TestProjectionWriterHasNoHardcodedAppSlug fails if the relay config still
// carries an AppSlug field or the writer still reads a .AppSlug selector. It
// reads the source, not the compiled package, so it is a true structural guard:
// a future edit that reintroduces a constant app_slug into the writer reds it.
func TestProjectionWriterHasNoHardcodedAppSlug(t *testing.T) {
	const src = "spikec_relay.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, src, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}

	var findings []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		// A `Foo.AppSlug` selector anywhere in NON-comment code is a hardcoded
		// per-writer app_slug source. (Comments are not part of the AST, so the
		// banner's prose mentions of AppSlug do not trip this.)
		case *ast.SelectorExpr:
			if node.Sel != nil && node.Sel.Name == "AppSlug" {
				findings = append(findings, fset.Position(node.Pos()).String()+": selector .AppSlug")
			}
		// An `AppSlug` field declared on the relay config struct.
		case *ast.Field:
			for _, name := range node.Names {
				if name.Name == "AppSlug" {
					findings = append(findings, fset.Position(name.Pos()).String()+": struct field AppSlug")
				}
			}
		}
		return true
	})

	if len(findings) > 0 {
		t.Fatalf("projection writer %s still carries a hardcoded app_slug source; "+
			"it must be resolved per-row from the template→app association (B-160):\n  %s",
			src, strings.Join(findings, "\n  "))
	}
}

// TestAppSlugResolvesPerTemplateAssociation seeds two templates on two apps and
// asserts appSlugForField returns each template's own app slug — the projection
// writer's per-row source. RED while the value is a constant / the resolver does
// not exist; GREEN once the writer resolves through the 0076 association.
func TestAppSlugResolvesPerTemplateAssociation(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	// db.Migrate does not seed hq_apps (db.go's startup upsert does), so seed
	// the two slugs this test needs directly.
	for _, slug := range []string{"operations", "inventory"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO hq_apps (slug, name, icon) VALUES ($1, $1, '🧪')
			 ON CONFLICT (slug) DO NOTHING`, slug); err != nil {
			t.Fatalf("seed app %q: %v", slug, err)
		}
	}

	fieldOps := seedTemplateFieldOnApp(t, pool, "operations", "Ops Template")
	fieldInv := seedTemplateFieldOnApp(t, pool, "inventory", "Inv Template")

	gotOps, err := appSlugForField(ctx, pool, fieldOps)
	if err != nil {
		t.Fatalf("appSlugForField(ops): %v", err)
	}
	gotInv, err := appSlugForField(ctx, pool, fieldInv)
	if err != nil {
		t.Fatalf("appSlugForField(inv): %v", err)
	}

	if gotOps != "operations" {
		t.Errorf("ops template resolved app_slug = %q, want %q", gotOps, "operations")
	}
	if gotInv != "inventory" {
		t.Errorf("inv template resolved app_slug = %q, want %q", gotInv, "inventory")
	}
	if gotOps == gotInv {
		t.Errorf("both templates resolved to the SAME app_slug %q — the writer is not "+
			"reading the per-template association (this is exactly the B-160 constant bug)", gotOps)
	}
}

// seedTemplateFieldOnApp creates a template bound to the given app slug, with one
// section + one field, and returns the field id. Mirrors seedTemplateWithField
// (access_test.go) but also sets checklist_templates.app_id from the app slug.
func seedTemplateFieldOnApp(t *testing.T, pool *pgxpool.Pool, appSlug, name string) string {
	t.Helper()
	ctx := context.Background()
	var tmplID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_templates (name, app_id)
		 VALUES ($1, (SELECT id FROM hq_apps WHERE slug = $2))
		 RETURNING id::text`, name, appSlug,
	).Scan(&tmplID); err != nil {
		t.Fatalf("seed template %q on app %q: %v", name, appSlug, err)
	}
	var secID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_sections (template_id, title, "order") VALUES ($1, 'S', 0) RETURNING id::text`,
		tmplID,
	).Scan(&secID); err != nil {
		t.Fatalf("seed section: %v", err)
	}
	var fieldID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_fields (section_id, type, label, "order") VALUES ($1, 'checkbox', 'Do C', 0) RETURNING id::text`,
		secID,
	).Scan(&fieldID); err != nil {
		t.Fatalf("seed field: %v", err)
	}
	return fieldID
}
