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
	_, err := pool.Exec(ctx, `
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
		  ('inventory-cost', 'Inventory · Cost', '💵')
		ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("seed hq_apps: %w", err)
	}
	return nil
}
