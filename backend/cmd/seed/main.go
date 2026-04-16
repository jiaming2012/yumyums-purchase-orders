package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	superadminPath := os.Getenv("SUPERADMIN_CONFIG")
	if superadminPath == "" {
		superadminPath = "config/superadmins.yaml"
	}
	superadmins, err := config.LoadSuperadmins(superadminPath)
	if err != nil {
		log.Fatalf("Failed to load superadmins config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Seed each superadmin that has a dev_password set
	seeded := 0
	for email, entry := range superadmins {
		if entry.DevPassword == "" {
			log.Printf("Skipping %s (no dev_password set)", email)
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(entry.DevPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password for %s: %v", email, err)
		}

		tag, err := pool.Exec(ctx,
			`UPDATE users SET password_hash = $1, status = 'active', accepted_at = now()
			 WHERE email = $2`,
			string(hash), email)
		if err != nil {
			log.Fatalf("Failed to update %s: %v", email, err)
		}
		if tag.RowsAffected() == 0 {
			log.Printf("Warning: no user found for %s -- did you run the server first?", email)
		} else {
			log.Printf("Seeded %s with dev password", email)
			seeded++
		}
	}

	// Seed hq_apps if empty
	var appCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM hq_apps").Scan(&appCount)
	if err != nil {
		log.Fatalf("Failed to count hq_apps: %v", err)
	}
	if appCount == 0 {
		_, err = pool.Exec(ctx, `
            INSERT INTO hq_apps (slug, name, icon) VALUES
              ('purchasing', 'Purchasing', '🛒'),
              ('payroll', 'Payroll', '💰'),
              ('scheduling', 'Scheduling', '📅'),
              ('hiring', 'Hiring', '👥'),
              ('bi', 'BI', '📊'),
              ('users', 'Users', '🔐'),
              ('operations', 'Operations', '📋')
            ON CONFLICT (slug) DO NOTHING`)
		if err != nil {
			log.Fatalf("Failed to seed hq_apps: %v", err)
		}
		fmt.Println("Seeded 7 hq_apps rows")
	} else {
		fmt.Printf("hq_apps already has %d rows, skipping seed\n", appCount)
	}

	fmt.Printf("Done. Seeded %d superadmin(s).\n", seeded)
}
