package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/config"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/me"
	opsync "github.com/yumyums/hq/internal/sync"
	"github.com/yumyums/hq/internal/workflow"
)

//go:embed all:public
var embeddedFS embed.FS

func main() {
	var staticFS fs.FS
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		// Dev: serve from disk — no rebuild needed for frontend changes
		log.Printf("Serving static files from disk: %s", dir)
		staticFS = os.DirFS(dir)
	} else {
		// Prod: serve from embedded FS (files baked into binary)
		log.Println("Serving static files from embedded FS")
		sub, err := fs.Sub(embeddedFS, "public")
		if err != nil {
			log.Fatalf("Failed to access embedded public dir: %v", err)
		}
		staticFS = sub
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Load superadmin config
	superadminPath := os.Getenv("SUPERADMIN_CONFIG")
	if superadminPath == "" {
		superadminPath = "config/superadmins.yaml"
	}
	superadmins, err := config.LoadSuperadmins(superadminPath)
	if err != nil {
		log.Fatalf("Failed to load superadmins: %v", err)
	}
	log.Printf("Loaded %d superadmin(s)", len(superadmins))

	// Load template seed config (optional — skip if file missing)
	templatePath := os.Getenv("TEMPLATE_CONFIG")
	if templatePath == "" {
		templatePath = "config/templates.yaml"
	}
	templateInputs, err := workflow.LoadTemplateConfig(templatePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Failed to load template config: %v", err)
		}
		log.Println("No template seed config found — skipping")
	}

	// Connect to database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Run migrations
	if err := db.Migrate(pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Upsert superadmins to users table on startup
	if err := auth.UpsertSuperadmins(ctx, pool, superadmins); err != nil {
		log.Fatalf("Failed to upsert superadmins: %v", err)
	}

	// Seed hq_apps if empty
	if err := db.SeedHQApps(ctx, pool); err != nil {
		log.Fatalf("Failed to seed hq_apps: %v", err)
	}

	// Seed templates if config was loaded
	if len(templateInputs) > 0 {
		// Use first superadmin as template creator
		var creatorID string
		for _, sa := range superadmins {
			err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", sa.Email).Scan(&creatorID)
			if err == nil {
				break
			}
		}
		if creatorID != "" {
			if err := workflow.SeedTemplates(ctx, pool, templateInputs, creatorID); err != nil {
				log.Fatalf("Failed to seed templates: %v", err)
			}
			log.Printf("Seeded %d template(s)", len(templateInputs))
		}
	}

	// Start WebSocket hub and Postgres LISTEN/NOTIFY pipeline
	hub := opsync.NewHub()
	go hub.Run()
	opsync.StartListener(ctx, dbURL, hub, pool)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Secure cookies require HTTPS — disable for local dev
	secureCookie := os.Getenv("STATIC_DIR") == ""

	// WebSocket endpoint at /ws — behind auth middleware, outside /api/v1 prefix
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(pool, superadmins))
		r.Get("/ws", opsync.WsHandler(hub, pool))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Unauthenticated
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})
		r.Post("/auth/login", auth.LoginHandler(pool, superadmins, secureCookie))

		// Protected — auth middleware applied to this group
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(pool, superadmins))
			r.Post("/auth/logout", auth.LogoutHandler(pool))
			r.Get("/me", me.MeHandler())
			r.Get("/me/apps", me.MeAppsHandler(pool))

			// Workflow endpoints — all authenticated
			r.Route("/workflow", func(r chi.Router) {
				r.Get("/templates", workflow.ListTemplatesHandler(pool))
				r.Post("/createTemplate", workflow.CreateTemplateHandler(pool))
				r.Put("/updateTemplate/{id}", workflow.UpdateTemplateHandler(pool))
				r.Delete("/archiveTemplate/{id}", workflow.ArchiveTemplateHandler(pool))
				r.Get("/myChecklists", workflow.MyChecklistsHandler(pool))
				r.Get("/myHistory", workflow.MyHistoryHandler(pool))
				r.Post("/saveResponse", workflow.SaveResponseHandler(pool))
				r.Post("/submitChecklist", workflow.SubmitChecklistHandler(pool))
				r.Get("/pendingApprovals", workflow.PendingApprovalsHandler(pool))
				r.Post("/approveSubmission", workflow.ApproveSubmissionHandler(pool))
				r.Post("/rejectItem", workflow.RejectItemHandler(pool))
				r.Get("/ops/since", opsync.OpsSinceHandler(pool))
			})
		})
	})

	r.Handle("/*", http.FileServerFS(staticFS))

	log.Printf("Yumyums HQ server listening on :%s", port)
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				log.Printf("  → http://%s:%s", ipnet.IP, port)
			}
		}
	}
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
