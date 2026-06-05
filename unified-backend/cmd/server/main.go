package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"unified-backend/internal/config"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/migrations"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
	"unified-backend/internal/shlinkctl"
)

func main() {
	cfg := config.MustLoad()

	ctx := context.Background()

	// ── Postgres ─────────────────────────────────────────────────────────────
	db, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	if err := postgres.RunMigrations(ctx, db, migrations.FS); err != nil {
		slog.Error("postgres migrate", "err", err)
		os.Exit(1)
	}

	userRepo    := postgres.NewUserRepository(db)
	auditRepo   := postgres.NewAuditRepository(db)
	rolesRepo   := postgres.NewRolePermissionsRepository(db)
	ownerRepo   := postgres.NewURLOwnershipRepository(db)

	// ── Shlink client ────────────────────────────────────────────────────────
	shlinkClient := shlink.NewClient(cfg.ShlinkBaseURL)
	if err := shlinkClient.ValidateVersion(ctx, 3, 5, 3*time.Second); err != nil {
		slog.Error("shlink version check", "err", err)
		os.Exit(1)
	}

	// ── Services ─────────────────────────────────────────────────────────────
	permsCache := service.NewPermissionsCache(rolesRepo, cfg.AdminRole)
	if err := permsCache.Load(ctx); err != nil {
		slog.Error("permissions cache init", "err", err)
		os.Exit(1)
	}

	shlinkSvc := service.NewShlinkService(shlinkClient, cfg, permsCache)

	// ── Provisioner ──────────────────────────────────────────────────────────
	var runner shlinkctl.Runner
	if cfg.ShlinkRunnerMode == "native" {
		runner = shlinkctl.NewNativeRunner(cfg.ShlinkBin)
	} else {
		runner = shlinkctl.NewDockerRunner(cfg.ShlinkContainerName)
	}
	provisioner := shlinkctl.NewProvisioner(db, runner)

	// ── Handlers ─────────────────────────────────────────────────────────────
	meH        := handler.NewMeHandler(cfg, permsCache)
	dashH      := handler.NewDashboardHandler(shlinkSvc, userRepo)
	proxyH     := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo, ownerRepo, cfg)
	adminH     := handler.NewAdminHandler(userRepo, auditRepo, rolesRepo)
	rolesH     := handler.NewRolesHandler(permsCache, rolesRepo, cfg)
	urlDetailH := handler.NewURLDetailHandler(shlinkSvc)
	settingsH  := handler.NewSettingsHandler(cfg, shlinkSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public — health probes (both variants for docker/k8s compatibility)
	healthFn := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	r.Get("/health", healthFn)
	r.Get("/healthz", healthFn)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.ExtractIdentity(cfg.RoleGroups))
		r.Use(middleware.RequireActiveUser(userRepo, auditRepo, provisioner, cfg))

		r.Get("/api/me", meH.ServeHTTP)
		r.Get("/api/dashboard", dashH.GetDashboard)

		// Shlink proxy
		r.Get("/api/shlink/short-urls", proxyH.ListShortURLs)
		r.Post("/api/shlink/short-urls", proxyH.CreateShortURL)
		r.Patch("/api/shlink/short-urls/{shortCode}", proxyH.UpdateShortURL)
		r.Delete("/api/shlink/short-urls/{shortCode}", proxyH.DeleteShortURL)

		r.Get("/api/shlink/tags", proxyH.ListTags)
		r.Put("/api/shlink/tags/{tagId}", proxyH.RenameTag)
		r.Delete("/api/shlink/tags/{tagId}", proxyH.DeleteTag)

		// URL detail
		r.Get("/api/urls/{shortCode}/detail", urlDetailH.GetURLDetail)

		// Settings
		r.Get("/api/settings", settingsH.GetSettings)
		r.Patch("/api/settings", settingsH.PatchSettings)

		// Admin
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly(cfg.AdminRole, auditRepo))

			r.Get("/api/admin/users", adminH.ListUsers)
			r.Get("/api/admin/users/{sub}", adminH.GetUser)
			r.Put("/api/admin/users/{sub}", adminH.UpdateUser)
			r.Put("/api/admin/users/{sub}/apikey", adminH.UpdateAPIKey)
			r.Put("/api/admin/users/{sub}/prefix", adminH.UpdateSlugPrefix)
			r.Get("/api/admin/users/{sub}/links", adminH.GetUserLinks)
			r.Get("/api/admin/logs", adminH.ListLogs)

			r.Get("/api/admin/roles", rolesH.ListRoles)
			r.Get("/api/admin/roles/{role}", rolesH.GetRole)
			r.Put("/api/admin/roles/{role}/permissions", rolesH.UpsertRolePermissions)

			r.Get("/api/admin/settings", adminH.GetSettings)
			r.Patch("/api/admin/settings", adminH.PatchSettings)
		})
	})

	// ── Server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "err", err)
	}
	slog.Info("server stopped")
}
