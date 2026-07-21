package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	"github.com/yumyums/hq/internal/photos"
	"github.com/yumyums/hq/internal/purchasing"
	"github.com/yumyums/hq/internal/receipt"
	"github.com/yumyums/hq/internal/recipes"
	opsync "github.com/yumyums/hq/internal/sync"
	"github.com/yumyums/hq/internal/toast"
	"github.com/yumyums/hq/internal/users"
	"github.com/yumyums/hq/internal/version"
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
				if errors.Is(err, workflow.ErrUnknownField) {
					// Field was cut from the template (FR-3, INV-4). Reject loudly so
					// the runner rolls back the optimistic checkmark instead of writing
					// under a dead field id.
					return nil, routerErr(http.StatusUnprocessableEntity, "unknown_field")
				}
				slog.Error("OpRouter SET_FIELD", "error", err)
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
			if err := workflow.ValidateResubmitPhotoFunc(ctx, pool, input, userID); err != nil {
				return nil, routerErr(http.StatusBadRequest, err.Error())
			}
			id, err := workflow.SubmitChecklistFunc(ctx, pool, input, userID)
			if err != nil {
				if errors.Is(err, workflow.ErrTemplateArchived) {
					return nil, routerErr(http.StatusConflict, "template_archived")
				}
				slog.Error("OpRouter SUBMIT_CHECKLIST", "error", err)
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
				slog.Error("OpRouter APPROVE_ITEM", "error", err)
				return nil, routerErr(http.StatusInternalServerError, "internal_error")
			}

		case opsync.OpRejectItem:
			var input workflow.RejectItemInput
			if err := json.Unmarshal(req.Payload, &input); err != nil {
				return nil, routerErr(http.StatusBadRequest, "invalid_payload")
			}
			if err := workflow.RejectItemFunc(ctx, pool, input, userID); err != nil {
				slog.Error("OpRouter REJECT_ITEM", "error", err)
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
					slog.Error("OpRouter SAVE_TEMPLATE update", "error", err)
					return nil, routerErr(http.StatusInternalServerError, "internal_error")
				}
			} else {
				var input workflow.TemplateInput
				if err := json.Unmarshal(req.Payload, &input); err != nil {
					return nil, routerErr(http.StatusBadRequest, "invalid_payload")
				}
				id, err := workflow.CreateTemplateFunc(ctx, pool, input, userID)
				if err != nil {
					slog.Error("OpRouter SAVE_TEMPLATE create", "error", err)
					return nil, routerErr(http.StatusInternalServerError, "internal_error")
				}
				return &opsync.RouteOpResult{EntityID: id}, nil
			}

		case opsync.OpArchiveTemplate:
			if err := workflow.ArchiveTemplateFunc(ctx, pool, req.EntityID); err != nil {
				slog.Error("OpRouter ARCHIVE_TEMPLATE", "error", err)
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
	// Configure structured JSON logging (NDJSON).
	// LOG_TO_FILE=1 redirects all log output to a timestamped file in logs/
	logOutput := os.Stderr
	var logFile *os.File
	if os.Getenv("LOG_TO_FILE") == "1" {
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			os.Exit(1)
		}
		randBytes := make([]byte, 4)
		rand.Read(randBytes)
		logName := fmt.Sprintf("hq_%s_%s.log",
			time.Now().Format("20060102_150405"),
			hex.EncodeToString(randBytes))
		logPath := filepath.Join(logDir, logName)
		var err error
		logFile, err = os.Create(logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logOutput = logFile
		fmt.Println(logPath)
	}
	logger := slog.New(slog.NewJSONHandler(logOutput, nil))
	slog.SetDefault(logger)
	// Bridge standard log (used by chi middleware) to slog JSON output
	log.SetFlags(0)
	log.SetOutput(&slogBridge{logger: logger})

	var staticFS fs.FS
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		// Dev: serve from disk — no rebuild needed for frontend changes
		slog.Info("serving static files from disk", "dir", dir)
		staticFS = os.DirFS(dir)
	} else {
		// Prod: serve from embedded FS (files baked into binary)
		slog.Info("serving static files from embedded FS")
		sub, err := fs.Sub(embeddedFS, "public")
		if err != nil {
			slog.Error("failed to access embedded public dir", "error", err)
			os.Exit(1)
		}
		staticFS = sub
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8484"
	}

	// Load superadmin config
	superadminPath := os.Getenv("SUPERADMIN_CONFIG")
	if superadminPath == "" {
		superadminPath = "config/superadmins.yaml"
	}
	superadmins, err := config.LoadSuperadmins(superadminPath)
	if err != nil {
		slog.Error("failed to load superadmins", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded superadmins", "count", len(superadmins))

	// Load template seed config (optional — skip if file missing)
	templatePath := os.Getenv("TEMPLATE_CONFIG")
	if templatePath == "" {
		templatePath = "config/templates.yaml"
	}
	templateInputs, err := workflow.LoadTemplateConfig(templatePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("failed to load template config", "error", err)
			os.Exit(1)
		}
		slog.Info("no template seed config found, skipping")
	}

	// Connect to database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		slog.Error("DB_URL environment variable is required")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to database")

	// Run migrations
	if err := db.Migrate(pool); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Upsert superadmins to users table on startup
	if err := auth.UpsertSuperadmins(ctx, pool, superadmins); err != nil {
		slog.Error("failed to upsert superadmins", "error", err)
		os.Exit(1)
	}

	// Seed hq_apps if empty
	if err := db.SeedHQApps(ctx, pool); err != nil {
		slog.Error("failed to seed hq_apps", "error", err)
		os.Exit(1)
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
				slog.Error("failed to seed templates", "error", err)
				os.Exit(1)
			}
			slog.Info("seeded templates", "count", len(templateInputs))
		}
	}

	// Seed onboarding templates
	if err := onboarding.SeedOnboardingTemplates(ctx, pool); err != nil {
		slog.Error("failed to seed onboarding templates", "error", err)
		os.Exit(1)
	}

	// Seed inventory fixtures (vendors, item groups, tags, purchase items)
	if err := inventory.SeedInventoryFixtures(ctx, pool); err != nil {
		slog.Error("failed to seed inventory fixtures", "error", err)
		os.Exit(1)
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
			slog.Warn("failed to initialize DO Spaces presigner, photo and video upload endpoints will return 503", "error", err)
		} else {
			spacesPresigner = p
			slog.Info("DO Spaces presigner initialized", "bucket", spacesBucket, "endpoint", spacesEndpoint)
		}
	} else {
		slog.Warn("DO Spaces env vars not set, photo and video upload endpoints will return 503", "required", "DO_SPACES_KEY, DO_SPACES_SECRET, DO_SPACES_BUCKET, DO_SPACES_REGION")
	}

	// Service-to-service token for sales-processor → /api/v1/inventory/period-summary
	// and /api/v1/inventory/menu-cogs (Phase 999.2).
	// Empty value = endpoint returns 503 (fail-closed); see auth.ServiceTokenMiddleware.
	serviceToken := os.Getenv("HQ_INVENTORY_SERVICE_TOKEN")
	if serviceToken == "" {
		slog.Warn("HQ_INVENTORY_SERVICE_TOKEN not set, /api/v1/inventory/period-summary AND /api/v1/inventory/menu-cogs will return 503")
	}

	// Start WebSocket hub and Postgres LISTEN/NOTIFY pipeline
	hub := opsync.NewHub()
	go hub.Run()
	opsync.StartListener(ctx, dbURL, hub, pool)

	// Receipt worker config — hoisted above route registration so the
	// on-demand sync handler (POST /inventory/sync-receipts) can close over it.
	// The background worker.StartWorker call below reuses the same config.
	workerInterval := 6 * time.Hour
	if intervalStr := os.Getenv("RECEIPT_WORKER_INTERVAL"); intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			workerInterval = d
		} else {
			slog.Warn("invalid RECEIPT_WORKER_INTERVAL, using 6h default", "value", intervalStr)
		}
	}
	lookbackDays := 14
	if lbStr := os.Getenv("MERCURY_LOOKBACK_DAYS"); lbStr != "" {
		if n, err := strconv.Atoi(lbStr); err == nil && n > 0 {
			lookbackDays = n
		} else {
			slog.Warn("invalid MERCURY_LOOKBACK_DAYS, using 14 default", "value", lbStr)
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Secure cookies require HTTPS — disable for local dev
	secureCookie := os.Getenv("STATIC_DIR") == ""

	// Load alert config early so handlers can send emails
	alertCfg := alerts.LoadConfig()

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
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":           "ok",
				"backend_version":  version.Backend,
				"frontend_version": version.Frontend,
				"git_sha":          version.GitSHA,
				"built_at":         version.BuiltAt,
			})
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
			slog.Info("client log", "level", strings.ToUpper(body.Level), "message", body.Message, "user", userInfo, "page", body.URL, "ua", body.UA)
			w.WriteHeader(http.StatusNoContent)
		})
		r.Post("/auth/login", auth.LoginHandler(pool, superadmins, secureCookie))
		r.Get("/auth/invite-info", users.InviteInfoHandler(pool))
		r.Post("/auth/accept-invite", users.AcceptInviteHandler(pool, secureCookie))

		// Service-to-service (no cookie session) — inventory period summary for
		// sales-processor weekly payroll flow. Lives in its OWN group with
		// service-token middleware; NOT under auth.Middleware (no cookie).
		//
		// HQ_COGS_CATEGORY_ALLOWLIST configures which Mercury categoryData.name
		// values count toward the COGS aggregate. Comma-separated, default "COGS".
		// Non-allowlisted (and NULL) events stay in purchase_events for
		// bookkeeping but don't roll up into food-cost numbers.
		rawAllow := envOrDefault("HQ_COGS_CATEGORY_ALLOWLIST", "COGS")
		cogsAllowlist := make([]string, 0)
		for _, s := range strings.Split(rawAllow, ",") {
			if t := strings.TrimSpace(s); t != "" {
				cogsAllowlist = append(cogsAllowlist, t)
			}
		}
		slog.Info("inventory COGS category allowlist", "allowlist", cogsAllowlist)

		r.Group(func(r chi.Router) {
			r.Use(auth.ServiceTokenMiddleware(serviceToken))
			r.Get("/inventory/period-summary", inventory.PeriodSummaryHandler(pool, cogsAllowlist))
			r.Get("/inventory/menu-cogs", recipes.MenuCogsHandler(pool)) // Phase 999.2
		})

		// Protected — auth middleware applied to this group
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(pool, superadmins))
			r.Post("/auth/logout", auth.LogoutHandler(pool))
			r.Get("/me", me.MeHandler())
			r.Get("/me/apps", me.MeAppsHandler(pool))

			// User admin endpoints — admin only
			r.Route("/users", func(r chi.Router) {
				r.Get("/", users.ListUsersHandler(pool))
				r.Post("/invite", users.InviteHandler(pool, alertCfg))
				r.Patch("/{id}", users.UpdateUserHandler(pool))
				r.Post("/{id}/reset-password", users.ResetPasswordHandler(pool, alertCfg))
				r.Post("/{id}/revoke", users.RevokeHandler(pool))
				r.Delete("/{id}", users.DeleteUserHandler(pool))
				// Notification preference — admin or self
				r.Get("/{id}/notification-preference", users.GetNotificationPreferenceHandler(pool))
				r.Put("/{id}/notification-preference", users.UpdateNotificationPreferenceHandler(pool))
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
				r.Get("/draftHolderCount", workflow.DraftHolderCountHandler(pool))
				r.Get("/myChecklists", workflow.MyChecklistsHandler(pool))
				r.Get("/myHistory", workflow.MyHistoryHandler(pool))
				r.Post("/saveResponse", workflow.SaveResponseHandler(pool))
				r.Post("/submitChecklist", workflow.SubmitChecklistHandler(pool))
				r.Get("/pendingApprovals", workflow.PendingApprovalsHandler(pool))
				r.Post("/approveSubmission", workflow.ApproveSubmissionHandler(pool))
				r.Post("/rejectItem", workflow.RejectItemHandler(pool))
				r.Post("/unsubmitChecklist", workflow.UnsubmitHandler(pool))
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
				// On-demand Mercury receipt sync (260607-bir): durable single-flight
				// runner backed by receipt_sync_runs. closes over receiptCfg above.
				r.Post("/sync-receipts", inventory.SyncReceiptsHandler(pool, func(ctx context.Context) (receipt.IngestResult, error) {
					return receipt.RunIngestCycle(ctx, receiptCfg)
				}))
				r.Get("/sync-receipts/status", inventory.SyncReceiptsStatusHandler(pool, receiptCfg.LookbackDays))
				r.Post("/purchases/reprocess-all", inventory.ReprocessAllPendingHandler(pool, func(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error) {
					return receipt.BatchReprocessFromSpaces(ctx, receiptCfg, rows)
				}))
				r.Post("/purchases/confirm", inventory.ConfirmPendingPurchaseHandler(pool))
				r.Post("/purchases/discard", inventory.DiscardPendingPurchaseHandler(pool))
				r.Post("/purchases/pending/{id}/retry-parse", inventory.RetryParsePendingPurchaseHandler(pool))
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
				// Toast menu items + this-week aggregate (Phase 22). Cookie-auth, not service-token.
				r.Get("/menu-items", toast.ListMenuItemsHandler(pool))

				// ── PER-TAB GATED SURFACES (design §1.3 station 1) ──────────
				//
				// These two are the ONLY inventory routes behind a grant. Every
				// route above is reachable by any logged-in user, unchanged.
				//
				// Each sits in its own r.Group so RequirePermission applies to
				// exactly one route: chi middleware is scoped to its group, and
				// a Use() at this Route's level would gate the whole tab set.
				//
				// The umbrella argument is the operator's signed rider (§8
				// amendment 1) — a whole-app `inventory` grant opens both tabs.
				r.Group(func(r chi.Router) {
					r.Use(auth.RequirePermission(pool, "inventory-trends", "inventory"))
					// design §2.2 as amended (decisions 29/30/31) — spend by ISO
					// week × item group. Cookie-auth, but it MUST be filtered by
					// the same cogsAllowlist the service-token period-summary is
					// constructed with (Amendment 1), or Trends over-reports
					// against payroll. The allowlist is built once above both
					// router groups — do not build a second copy here.
					r.Get("/trends", inventory.TrendsHandler(pool, cogsAllowlist))
				})
				r.Group(func(r chi.Router) {
					r.Use(auth.RequirePermission(pool, "inventory-cost", "inventory"))
					r.Get("/cost", recipes.CostHandler(pool)) // design §2.3 — cost/margin/food-cost-%
				})
			})

			// Phase 999.2 — recipes CRUD (cookie-auth; any authenticated user can edit).
			// The menu-cogs endpoint sits in the service-token group above (line ~343).
			// /drift is read by Plan 05's Recipes-tab banner (D-22 self-healing).
			r.Route("/inventory/recipes", func(r chi.Router) {
				r.Get("/", recipes.ListRecipesHandler(pool))
				r.Post("/", recipes.CreateRecipeHandler(pool))
				r.Put("/{id}", recipes.UpdateRecipeHandler(pool))
				r.Delete("/{id}", recipes.DeleteRecipeHandler(pool))
				r.Post("/merge", recipes.MergeMenuItemHandler(pool))
				r.Get("/drift", recipes.DriftBannerHandler(pool)) // Phase 999.2 Plan 04
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

				// Repurchase badge reset (admin-only)
				r.Get("/repurchase-reset", purchasing.GetRepurchaseResetConfigHandler(pool))
				r.Post("/repurchase-reset", purchasing.RepurchaseResetHandler(pool))
				r.Put("/repurchase-reset/config", purchasing.UpsertRepurchaseResetConfigHandler(pool))
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
				r.Post("/rejectSection", onboarding.RejectSectionHandler(pool))
				r.Post("/reopenSection", onboarding.ReopenSectionHandler(pool))
				r.Post("/createTemplate", onboarding.CreateTemplateHandler(pool))
				r.Put("/updateTemplate/{id}", onboarding.UpdateTemplateHandler(pool))
				r.Delete("/deleteTemplate/{id}", onboarding.DeleteTemplateHandler(pool))
				r.Post("/assignTemplate", onboarding.AssignTemplateHandler(pool))
				r.Post("/unassignTemplate", onboarding.UnassignTemplateHandler(pool))
			})
		})
	})

	r.Handle("/*", http.FileServerFS(staticFS))

	// E2E_DISABLE_SCHEDULERS=1 turns off the background pollers (receipt
	// ingest, cutoff auto-lock, drift check). All run an immediate check on
	// start; the cutoff check auto-locks the current draft PO once the day's
	// cutoff time has passed, which races E2E tests that exercise cutoff
	// manually via /simulate-cutoff (they'd intermittently get 409
	// locked_po_pending_approval depending on the wall-clock time of the run).
	// The receipt worker is worse: with real MERCURY_API_KEY/ANTHROPIC_API_KEY
	// leaked into a test stack (Taskfile dotenv loads backend/.env), it ingests
	// LIVE Mercury transactions into the test database mid-suite. Mirrors
	// TOAST_SYNC_INTERVAL=0. Prod leaves this unset, so scheduler behavior is
	// unchanged.
	schedulersDisabled := os.Getenv("E2E_DISABLE_SCHEDULERS") == "1"

	// Start receipt ingestion background worker.
	// Gracefully skips if MERCURY_API_KEY or ANTHROPIC_API_KEY is not set.
	// receiptCfg was constructed above (hoisted so the sync handler can close over it).
	if schedulersDisabled {
		slog.Info("receipt worker disabled", "reason", "E2E_DISABLE_SCHEDULERS=1")
	} else {
		receipt.StartWorker(ctx, receiptCfg)
	}

	// Initialize and start alert queue — gracefully no-ops when env vars are not set
	alertQ := alerts.NewQueue(alertCfg)
	alertQ.Start(ctx)
	purchasing.SetAlertQueue(alertQ)
	slog.Info("alert queue started")

	// Start cutoff scheduler — polls every 15m to auto-lock POs and send reminders
	if schedulersDisabled {
		slog.Info("cutoff scheduler disabled", "reason", "E2E_DISABLE_SCHEDULERS=1")
	} else {
		purchasing.StartScheduler(ctx, pool)
	}

	// Phase 999.2 Plan 04 — recipes drift scheduler.
	// Polls every 15m; runs the actual drift check once on Monday 09:00 Chicago.
	// SetAlertQueue MUST be called BEFORE StartDriftScheduler (mirrors toast pattern).
	recipes.SetAlertQueue(alertQ)
	if !schedulersDisabled {
		recipes.StartDriftScheduler(ctx, pool)
	}

	// Toast ingest — Phase 22.1.
	// Combined sync+ingest worker: SFTP→Spaces+cache per date, then Spaces→DB.
	// D-12 preserved: LoadConfigFromEnv fails fast on missing/unreadable TOAST_SFTP_KEY_PATH.
	// D-06: Spaces unreachable does NOT crash the server — worker logs WARNING and skips ticks;
	//       Cliq alert fires to purchaseandinventory channel after 3 consecutive failures.
	// TOAST_SYNC_INTERVAL=0 disables the in-process worker (cmd/sync-toast still available).
	{
		toastCfg, err := toast.LoadConfigFromEnv()
		if err != nil {
			slog.Error("toast worker config", "error", err)
			os.Exit(1)
		}
		toastCfg.Pool = pool
		toastCfg.SpacesClient = spacesClient
		toastCfg.SpacesBucket = spacesBucket
		toastCfg.SpacesEndpoint = spacesEndpoint
		toastCfg.CacheDir = envOrDefault("TOAST_CACHE_DIR", "backend/cache/toast")

		// Wire the alert queue so Plan 04's degraded-Spaces alert can dispatch.
		toast.SetAlertQueue(alertQ)

		if toastCfg.Interval == 0 {
			slog.Info("toast worker disabled, cmd/sync-toast remains available", "reason", "TOAST_SYNC_INTERVAL=0")
		} else {
			toast.StartWorker(ctx, toastCfg)
		}
	}

	// Bind the listener synchronously so the "ready" sentinel below is accurate
	// (the kernel buffers incoming connections in the backlog until http.Serve
	// picks them up). Avoids racing with background goroutine logs.
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("listen failed", "port", port, "error", err)
		os.Exit(1)
	}
	slog.Info("server listening", "port", port)
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				slog.Info("reachable at", "url", fmt.Sprintf("http://%s:%s", ipnet.IP, port))
			}
		}
	}
	slog.Info("Yumyums HQ ready, accepting connections")
	if err := http.Serve(ln, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// slogBridge adapts slog as an io.Writer so that standard log output
// (e.g., chi middleware.Logger) is emitted as NDJSON.
type slogBridge struct{ logger *slog.Logger }

func (b *slogBridge) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	b.logger.Info(msg, "source", "stdlog")
	return len(p), nil
}

// envOrDefault returns os.Getenv(k) if non-empty, else d. Used for optional
// env overrides where a sensible default exists (e.g., TOAST_CACHE_DIR).
func envOrDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
