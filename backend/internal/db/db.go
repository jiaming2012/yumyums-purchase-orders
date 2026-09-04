package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	config.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

func Migrate(pool *pgxpool.Pool) error {
	// goose requires *sql.DB -- use stdlib wrapper over pgxpool
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("database migrations applied successfully")
	return nil
}

func SeedHQApps(ctx context.Context, pool *pgxpool.Pool) error {
	// Upsert all apps — runs every startup so new apps get added to existing databases
	//
	// NAMING CONVENTION — `<app>-<tab>` rows are GATED TABS, not launcher apps
	// (design `prove-surface-gating-and-endpoints.md` §1.4, Option (i)).
	//
	// Registering a gated tab as an hq_apps row is what lets it reuse every
	// existing station unchanged: /me/apps reports it, the Users Access list
	// renders standard toggles for it, and auth.RequirePermission queries it with
	// the same EXISTS shape as any app grant. No migration is needed — this
	// upsert runs on every startup, so the rows appear in dev and prod on deploy.
	//
	// The cost of the convention is that hq_apps now means "apps AND gated tabs".
	// The guard is that index.html's filterTilesByPermissions only toggles
	// EXISTING static tiles, so a tab slug renders no launcher tile. Do not add a
	// static tile whose id matches an `<app>-<tab>` slug, or the tab will start
	// appearing on the home grid as if it were an app.
	//
	// `marketing-offline-override` is a third kind of row: an ENTITLEMENT
	// surface, not a tab (QR-redemption design §13/§16, fork #12 — resolved by
	// the operator, slate-20260905). It rides the same convention so it reuses
	// every station (access editor, /me/apps, RequirePermission) unchanged, BUT
	// its eventual enforcement must be mounted NARROW —
	// RequirePermission(pool, "marketing-offline-override") with NO umbrella
	// slug — because holding the `marketing` app grant must never imply the
	// offline override. The umbrella rider ("App grant = All tabs granted") is
	// about tabs; this is not one. Pinned by
	// internal/auth/marketing_seed_test.go.
	//
	// The grant CTE below seeds initial grants on FIRST REGISTRATION ONLY
	// (rows the INSERT actually inserted): this function runs on every server
	// startup, and an unconditional grant upsert would resurrect grants an
	// operator revoked. Seeded grants (§16 permissions table): the `marketing`
	// APP (tab access — scan/redeem, min role team_member) to all three roles;
	// the offline_override entitlement to admin only ("seeded true for admin
	// users" — managers and team members get it by explicit grant, never by
	// role implication). Campaigns/stats view+create are manager-tier gates
	// enforced INSIDE their handlers when those endpoints land (§16: middleware
	// grants tab access; the tab is not the create/stats gate) — recorded
	// N/A-with-reason in tests/grant-enforcement-parity.spec.js until then.
	_, err := pool.Exec(ctx, `
		WITH seeded AS (
			INSERT INTO hq_apps (slug, name, icon) VALUES
			  ('purchasing', 'Purchase Orders', '🛒'),
			  ('payroll', 'Payroll', '💰'),
			  ('scheduling', 'Scheduling', '📅'),
			  ('hiring', 'Hiring', '👥'),
			  ('bi', 'BI', '📊'),
			  ('users', 'Users', '🔐'),
			  ('operations', 'Operations', '📋'),
			  ('onboarding', 'Onboarding', '🎓'),
			  ('inventory', 'Inventory', '📦'),
			  -- gated tabs of the inventory app (see convention note above)
			  ('inventory-trends', 'Inventory · Trends', '📈'),
			  ('inventory-cost', 'Inventory · Cost', '💵'),
			  ('marketing', 'Marketing', '📢'),
			  -- entitlement surface of the marketing app (see note above)
			  ('marketing-offline-override', 'Marketing · Offline Override', '🔓')
			ON CONFLICT (slug) DO NOTHING
			RETURNING id, slug
		)
		INSERT INTO app_permissions (app_id, role)
		SELECT s.id, g.role
		FROM seeded s
		JOIN (VALUES
		  ('marketing', 'admin'),
		  ('marketing', 'manager'),
		  ('marketing', 'team_member'),
		  ('marketing-offline-override', 'admin')
		) AS g(slug, role) ON g.slug = s.slug`)
	if err != nil {
		return fmt.Errorf("seed hq_apps: %w", err)
	}
	return nil
}
