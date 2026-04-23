package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/alerts"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/config"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/inventory"
	"github.com/yumyums/hq/internal/me"
	"github.com/yumyums/hq/internal/onboarding"
	"github.com/yumyums/hq/internal/purchasing"
	"github.com/yumyums/hq/internal/photos"
	"github.com/yumyums/hq/internal/receipt"
	opsync "github.com/yumyums/hq/internal/sync"
	"github.com/yumyums/hq/internal/users"
	"github.com/yumyums/hq/internal/workflow"
)

// workflowOpRouter implements opsync.OpRouter by routing ops to existing
// workflow business logic. Defined here to avoid a circular import between
// the sync and workflow packages.
func workflowOpRouter(pool *pgxpool.Pool) opsync.OpRouter {
	return func(ctx context.Context, userID string, req opsync.OpRequest) (*opsync.RouteOpResult, error) {
		routerErr := func(status int, msg string) error {
			return &opsync.OpRouterError{Status: status, Message: msg}
		}

		switch req.OpType {
		case opsync.OpSetField:
			var p struct {
				FieldID string          `json:"field_id"`
				Value   json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(req.Payload, &p); err != nil {
				return nil, routerErr(http.StatusBadRequest, "invalid_payload")
			}
			if err := workflow.SaveResponseFunc(ctx, pool, p.FieldID, p.Value, userID); err != nil {
				log.Printf("OpRouter SET_FIELD error: %v", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}

		case opsync.OpSubmitChecklist:
			var input workflow.SubmitChecklistInput
			if err := json.Unmarshal(req.Payload, &input); err != nil {
				return nil, routerErr(http.StatusBadRequest, "invalid_payload")
			}
			if err := workflow.ValidateFailNotesFunc(ctx, pool, input); err != nil {
				return nil, routerErr(http.StatusBadRequest, err.Error())
			}
			id, err := workflow.SubmitChecklistFunc(ctx, pool, input, userID)
			if err != nil {
				if errors.Is(err, workflow.ErrTemplateArchived) {
					return nil, routerErr(http.StatusConflict, "template_archived")
				}
				log.Printf("OpRouter SUBMIT_CHECKLIST error: %v", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}
			return &opsync.RouteOpResult{EntityID: id}, nil

		case opsync.OpApproveItem:
			var body struct {
				SubmissionID string `json:"submission_id"`
			}
			if err := json.Unmarshal(req.Payload, &body); err != nil || body.SubmissionID == "" {
				return nil, routerErr(http.StatusBadRequest, "invalid_payload")
			}
			if err := workflow.ApproveSubmissionFunc(ctx, pool, body.SubmissionID, userID); err != nil {
				log.Printf("OpRouter APPROVE_ITEM error: %v", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}

		case opsync.OpRejectItem:
			var input workflow.RejectItemInput
			if err := json.Unmarshal(req.Payload, &input); err != nil {
				return nil, routerErr(http.StatusBadRequest, "invalid_payload")
			}
			if err := workflow.RejectItemFunc(ctx, pool, input, userID); err != nil {
				log.Printf("OpRouter REJECT_ITEM error: %v", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}

		case opsync.OpSaveTemplate:
			var peek struct {
				ID string `json:"id"`
			}
			json.Unmarshal(req.Payload, &peek) //nolint:errcheck
			if peek.ID != "" {
				var input workflow.TemplateInput
				if err := json.Unmarshal(req.Payload, &input); err != nil {
					return nil, routerErr(http.StatusBadRequest, "invalid_payload")
				}
				if err := workflow.UpdateTemplateFunc(ctx, pool, peek.ID, input); err != nil {
					log.Printf("OpRouter SAVE_TEMPLATE update error: %v", err)
					return nil, routerErr(http.StatusInternalServerError, "internal_error")
				}
			} else {
				var input workflow.TemplateInput
				if err := json.Unmarshal(req.Payload, &input); err != nil {
					return nil, routerErr(http.StatusBadRequest, "invalid_payload")
				}
				id, err := workflow.CreateTemplateFunc(ctx, pool, input, userID)
				if err != nil {
					log.Printf("OpRouter SAVE_TEMPLATE create error: %v", err)
					return nil, routerErr(http.StatusInternalServerError, "internal_error")
				}
				return &opsync.RouteOpResult{EntityID: id}, nil
			}

		case opsync.OpArchiveTemplate:
			if err := workflow.ArchiveTemplateFunc(ctx, pool, req.EntityID); err != nil {
				log.Printf("OpRouter ARCHIVE_TEMPLATE error: %v", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}

		default:
			return nil, routerErr(http.StatusBadRequest, "unknown_op_type")
		}

		return nil, nil
	}
}

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

	// Seed onboarding templates
	if err := onboarding.SeedOnboardingTemplates(ctx, pool); err != nil {
		log.Fatalf("Failed to seed onboarding templates: %v", err)
	}

	// Seed inventory fixtures (vendors, item groups, tags, purchase items)
	if err := inventory.SeedInventoryFixtures(ctx, pool); err != nil {
		log.Fatalf("Failed to seed inventory fixtures: %v", err)
	}

	// Initialize DO Spaces presigner + client (optional — graceful degradation if env vars missing)
	var spacesPresigner *s3.PresignClient
	var spacesClient *s3.Client
	spacesEndpoint := os.Getenv("DO_SPACES_ENDPOINT")
	spacesBucket := os.Getenv("DO_SPACES_BUCKET")
	spacesRegion := os.Getenv("DO_SPACES_REGION")
	if spacesEndpoint == "" && spacesRegion != "" {
		spacesEndpoint = "https://" + spacesRegion + ".digitaloceanspaces.com"
	}
	if os.Getenv("DO_SPACES_KEY") != "" && os.Getenv("DO_SPACES_SECRET") != "" && spacesBucket != "" && spacesEndpoint != "" {
		spacesCfg := photos.SpacesConfig{
			AccessKey: os.Getenv("DO_SPACES_KEY"),
			SecretKey: os.Getenv("DO_SPACES_SECRET"),
			Endpoint:  spacesEndpoint,
			Region:    spacesRegion,
			Bucket:    spacesBucket,
		}
		spacesClient = photos.NewSpacesClient(spacesCfg)
		p, err := photos.NewSpacesPresigner(spacesCfg)
		if err != nil {
			log.Printf("WARNING: Failed to initialize DO Spaces presigner: %v — photo and video upload endpoints will return 503", err)
		} else {
			spacesPresigner = p
			log.Printf("DO Spaces presigner initialized (bucket: %s, endpoint: %s)", spacesBucket, spacesEndpoint)
		}
	} else {
		log.Println("WARNING: DO Spaces env vars not set (DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_BUCKET, DO_SPACES_REGION) — photo and video upload endpoints will return 503")
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
		r.Post("/logs", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Level   string `json:"level"`
				Message string `json:"message"`
				URL     string `json:"url"`
				UA      string `json:"ua"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// Best-effort user identification from session cookie
			userInfo := "anonymous"
			if cookie, err := r.Cookie("hq_session"); err == nil && cookie.Value != "" {
				tokenHash := auth.HashToken(cookie.Value)
				if user, err := auth.LookupSession(r.Context(), pool, tokenHash, superadmins); err == nil && user != nil {
					userInfo = user.DisplayName + " (" + user.Email + ")"
				}
			}
			log.Printf("[CLIENT %s] %s | user=%s | page=%s | ua=%s", strings.ToUpper(body.Level), body.Message, userInfo, body.URL, body.UA)
			w.WriteHeader(http.StatusNoContent)
		})
		r.Post("/auth/login", auth.LoginHandler(pool, superadmins, secureCookie))
		r.Get("/auth/invite-info", users.InviteInfoHandler(pool))
		r.Post("/auth/accept-invite", users.AcceptInviteHandler(pool, secureCookie))

		// Protected — auth middleware applied to this group
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(pool, superadmins))
			r.Post("/auth/logout", auth.LogoutHandler(pool))
			r.Get("/me", me.MeHandler())
			r.Get("/me/apps", me.MeAppsHandler(pool))

			// User admin endpoints — admin only
			r.Route("/users", func(r chi.Router) {
				r.Get("/", users.ListUsersHandler(pool))
				r.Post("/invite", users.InviteHandler(pool))
				r.Patch("/{id}", users.UpdateUserHandler(pool))
				r.Post("/{id}/reset-password", users.ResetPasswordHandler(pool))
				r.Post("/{id}/revoke", users.RevokeHandler(pool))
				r.Delete("/{id}", users.DeleteUserHandler(pool))
			})

			// App permissions endpoints — admin only
			r.Route("/apps", func(r chi.Router) {
				r.Get("/permissions", users.GetAppPermissionsHandler(pool))
				r.Put("/{slug}/permissions", users.SetAppPermissionsHandler(pool))
			})

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
				r.Post("/ops", opsync.OpHandler(pool, workflowOpRouter(pool)))
			})

			// Photos endpoints — presigned URL generation for DO Spaces
			r.Route("/photos", func(r chi.Router) {
				r.Post("/presign", photos.PresignUploadHandler(spacesPresigner, spacesBucket, spacesEndpoint))
				r.Get("/presign", photos.PresignGetHandler(spacesPresigner, spacesBucket))
				r.Post("/upload", photos.UploadHandler(spacesClient, spacesBucket, spacesEndpoint))
			})

			// Video endpoints — presigned upload URL + FFmpeg processing trigger
			r.Route("/videos", func(r chi.Router) {
				r.Post("/presign", onboarding.VideoPresignHandler(spacesPresigner, spacesBucket, spacesEndpoint))
				r.Post("/process", onboarding.VideoProcessHandler(spacesPresigner, spacesBucket, spacesEndpoint, pool))
			})

			// Inventory endpoints — all authenticated
			r.Route("/inventory", func(r chi.Router) {
				r.Get("/vendors", inventory.ListVendorsHandler(pool))
				r.Post("/vendors", inventory.CreateVendorHandler(pool))
				r.Put("/vendors", inventory.UpdateVendorHandler(pool))
				r.Post("/vendors/merge", inventory.MergeVendorsHandler(pool))
				r.Get("/purchases", inventory.ListPurchaseEventsHandler(pool))
				r.Post("/purchases", inventory.CreatePurchaseEventHandler(pool))
				r.Get("/purchases/pending", inventory.ListPendingPurchasesHandler(pool))
				r.Post("/purchases/confirm", inventory.ConfirmPendingPurchaseHandler(pool))
				r.Post("/purchases/discard", inventory.DiscardPendingPurchaseHandler(pool))
				r.Put("/purchases/pending-items", inventory.UpdatePendingItemsHandler(pool))
				r.Post("/purchases/pending-seed", inventory.SeedPendingPurchaseHandler(pool))
				r.Get("/stock", inventory.GetStockHandler(pool))
				r.Post("/stock/count", inventory.UpdateStockCountHandler(pool))
				r.Get("/items", inventory.ListItemsHandler(pool))
				r.Post("/items", inventory.CreateItemHandler(pool))
				r.Put("/items", inventory.UpdateItemHandler(pool))
				r.Post("/items/merge", inventory.MergeItemsHandler(pool))
				r.Get("/groups", inventory.ListGroupsHandler(pool))
				r.Post("/groups", inventory.CreateGroupHandler(pool))
				r.Put("/groups", inventory.UpdateGroupHandler(pool))
				r.Get("/tags", inventory.ListTagsHandler(pool))
			})

			// Purchasing endpoints — all authenticated
			r.Route("/purchasing", func(r chi.Router) {
				// Cutoff config (admin-only for PUT)
				r.Get("/cutoff", purchasing.GetCutoffConfigHandler(pool))
				r.Put("/cutoff", purchasing.UpsertCutoffConfigHandler(pool))

				// Simulate cutoff (admin-only)
				r.Post("/simulate-cutoff", purchasing.SimulateCutoffHandler(pool))

				// GET /orders?status=locked — must be before POST /orders and before /{id} wildcard
				r.Get("/orders", purchasing.GetOrdersByStatusHandler(pool))
				r.Post("/orders", purchasing.GetOrCreateOrderHandler(pool))
				r.Get("/orders/{id}", purchasing.GetOrderHandler(pool))
				r.Put("/orders/{id}/items", purchasing.UpsertLineItemsHandler(pool))
				r.Get("/orders/{id}/suggestions", purchasing.GetSuggestionsHandler(pool))

				// PO state machine (admin-only)
				r.Post("/orders/{id}/lock", purchasing.LockPOHandler(pool))
				r.Post("/orders/{id}/unlock", purchasing.UnlockPOHandler(pool))
				r.Post("/orders/{id}/approve", purchasing.ApprovePOHandler(pool))

				// Shopping list routes — static paths before wildcard {id}
				r.Get("/shopping/active", purchasing.GetActiveShoppingListHandler(pool))
				r.Get("/shopping/history", purchasing.GetShoppingListHistoryHandler(pool))
				r.Get("/shopping/{id}", purchasing.GetShoppingListHandler(pool))
				r.Post("/shopping/{id}/check", purchasing.CheckShoppingItemHandler(pool))
				r.Put("/shopping/{id}/items/{itemId}/location", purchasing.UpdateShoppingItemLocationHandler(pool))
				r.Put("/shopping/{id}/items/{itemId}/photo", purchasing.UpdateShoppingItemPhotoHandler(pool))
				r.Post("/shopping/{id}/vendors/{vendorSectionId}/complete", purchasing.CompleteVendorSectionHandler(pool))
			})

			// Onboarding endpoints — all authenticated
			r.Route("/onboarding", func(r chi.Router) {
				r.Get("/templates", onboarding.ListTemplatesHandler(pool))
				r.Get("/templates/{id}", onboarding.GetTemplateHandler(pool))
				r.Get("/myTrainings", onboarding.MyTrainingsHandler(pool))
				r.Get("/hireTraining/{hireId}", onboarding.HireTrainingHandler(pool))
				r.Get("/managerHires", onboarding.ManagerHiresHandler(pool))
				r.Post("/saveProgress", onboarding.SaveProgressHandler(pool))
				r.Post("/signOff", onboarding.SignOffHandler(pool))
				r.Post("/createTemplate", onboarding.CreateTemplateHandler(pool))
				r.Put("/updateTemplate/{id}", onboarding.UpdateTemplateHandler(pool))
				r.Delete("/deleteTemplate/{id}", onboarding.DeleteTemplateHandler(pool))
				r.Post("/assignTemplate", onboarding.AssignTemplateHandler(pool))
				r.Post("/unassignTemplate", onboarding.UnassignTemplateHandler(pool))
			})
		})
	})

	r.Handle("/*", http.FileServerFS(staticFS))

	// Start receipt ingestion background worker
	// Gracefully skips if MERCURY_API_KEY or ANTHROPIC_API_KEY is not set.
	{
		workerInterval := 6 * time.Hour
		if intervalStr := os.Getenv("RECEIPT_WORKER_INTERVAL"); intervalStr != "" {
			if d, err := time.ParseDuration(intervalStr); err == nil {
				workerInterval = d
			} else {
				log.Printf("WARNING: invalid RECEIPT_WORKER_INTERVAL %q — using 6h default", intervalStr)
			}
		}

		lookbackDays := 14
		if lbStr := os.Getenv("MERCURY_LOOKBACK_DAYS"); lbStr != "" {
			if n, err := strconv.Atoi(lbStr); err == nil && n > 0 {
				lookbackDays = n
			} else {
				log.Printf("WARNING: invalid MERCURY_LOOKBACK_DAYS %q — using 14 default", lbStr)
			}
		}

		receiptCfg := receipt.WorkerConfig{
			MercuryAPIKey:   os.Getenv("MERCURY_API_KEY"),
			AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
			Pool:            pool,
			SpacesPresigner: spacesPresigner,
			SpacesEndpoint:  spacesEndpoint,
			SpacesBucket:    spacesBucket,
			Interval:        workerInterval,
			LookbackDays:    lookbackDays,
		}
		receipt.StartWorker(ctx, receiptCfg)
	}

	// Initialize and start alert queue — gracefully no-ops when env vars are not set
	alertCfg := alerts.LoadConfig()
	alertQ := alerts.NewQueue(alertCfg)
	alertQ.Start(ctx)
	purchasing.SetAlertQueue(alertQ)
	log.Println("Alert queue started")

	// Start cutoff scheduler — polls every 15m to auto-lock POs and send reminders
	purchasing.StartScheduler(ctx, pool)

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
