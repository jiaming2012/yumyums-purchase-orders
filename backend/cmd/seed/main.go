package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Hash the test password "test123"
	hash, err := bcrypt.GenerateFromPassword([]byte("test123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Update superadmin user with password and active status
	tag, err := pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, status = 'active', accepted_at = now()
         WHERE email = 'jamal@yumyums.com'`,
		string(hash))
	if err != nil {
		log.Fatalf("Failed to update superadmin: %v", err)
	}
	if tag.RowsAffected() == 0 {
		log.Fatal("No user found with email jamal@yumyums.com -- did you run the server first (to trigger UpsertSuperadmins)?")
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

	fmt.Println("Superadmin jamal@yumyums.com seeded with password: test123")
}
