package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		slog.Error("DB_URL environment variable is required")
		os.Exit(1)
	}

	superadminPath := os.Getenv("SUPERADMIN_CONFIG")
	if superadminPath == "" {
		superadminPath = "config/superadmins.yaml"
	}
	superadmins, err := config.LoadSuperadmins(superadminPath)
	if err != nil {
		slog.Error("failed to load superadmins config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Seed each superadmin that has a dev_password set
	seeded := 0
	for email, entry := range superadmins {
		if entry.DevPassword == "" {
			slog.Info("skipping user (no dev_password set)", "email", email)
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(entry.DevPassword), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password", "email", email, "error", err)
			os.Exit(1)
		}

		tag, err := pool.Exec(ctx,
			`UPDATE users SET password_hash = $1, status = 'active', accepted_at = now()
			 WHERE email = $2`,
			string(hash), email)
		if err != nil {
			slog.Error("failed to update user", "email", email, "error", err)
			os.Exit(1)
		}
		if tag.RowsAffected() == 0 {
			slog.Warn("no user found -- did you run the server first?", "email", email)
		} else {
			slog.Info("seeded user with dev password", "email", email)
			seeded++
		}
	}

	// Seed hq_apps if empty
	var appCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM hq_apps").Scan(&appCount)
	if err != nil {
		slog.Error("failed to count hq_apps", "error", err)
		os.Exit(1)
	}
	if appCount == 0 {
		_, err = pool.Exec(ctx, `
            INSERT INTO hq_apps (slug, name, icon) VALUES
              ('purchasing', 'Purchase Orders', '🛒'),
              ('payroll', 'Payroll', '💰'),
              ('scheduling', 'Scheduling', '📅'),
              ('hiring', 'Hiring', '👥'),
              ('bi', 'BI', '📊'),
              ('users', 'Users', '🔐'),
              ('operations', 'Operations', '📋')
            ON CONFLICT (slug) DO NOTHING`)
		if err != nil {
			slog.Error("failed to seed hq_apps", "error", err)
			os.Exit(1)
		}
		fmt.Println("Seeded 7 hq_apps rows")
	} else {
		fmt.Printf("hq_apps already has %d rows, skipping seed\n", appCount)
	}

	fmt.Printf("Done. Seeded %d superadmin(s).\n", seeded)
}
