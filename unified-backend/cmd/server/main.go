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
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

func main() {
	cfg := config.MustLoad()

	// ── Postgres ─────────────────────────────────────────────────────────────
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	if err := postgres.Migrate(db); err != nil {
		slog.Error("postgres migrate", "err", err)
		os.Exit(1)
	}

	userRepo := postgres.NewUserRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	rolesRepo := postgres.NewRolePermissionsRepository(db)

	// ── Shlink client ────────────────────────────────────────────────────────
	shlinkClient := shlink.NewClient(cfg.ShlinkBaseURL)
	if err := shlinkClient.ValidateVersion(context.Background(), 3, 5, 3*time.Second); err != nil {
		slog.Error("shlink version check", "err", err)
		os.Exit(1)
	}

	// ── Services ─────────────────────────────────────────────────────────────
	permsCache, err := service.NewPermissionsCache(context.Background(), userRepo)
	if err != nil {
		slog.Error("permissions cache init", "err", err)
		os.Exit(1)
	}

	shlinkSvc := service.NewShlinkService(shlinkClient, cfg, permsCache)

	// ── Handlers ─────────────────────────────────────────────────────────────
	meH := handler.NewMeHandler(userRepo, shlinkSvc)
	dashH := handler.NewDashboardHandler(shlinkSvc, userRepo)
	proxyH := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo)
	adminH := handler.NewAdminHandler(userRepo, auditRepo, rolesRepo)

	urlDetailH := handler.NewURLDetailHandler(shlinkSvc)
	settingsH := handler.NewSettingsHandler(cfg, shlinkSvc)

	// ── OIDC middleware ───────────────────────────────────────────────────────
	oidcMW, err := middleware.NewOIDCMiddleware(cfg)
	if err != nil {
		slog.Error("oidc middleware init", "err", err)
		os.Exit(1)
	}

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

	// Public
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(oidcMW.Authenticate)
		r.Use(middleware.EnsureUser(userRepo, cfg))

		r.Get("/api/me", meH.GetMe)
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
		r.Patch("/api/settings", settingsH.UpdateSettings)

		// Admin
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Get("/api/admin/users", adminH.ListUsers)
			r.Get("/api/admin/users/{sub}", adminH.GetUser)
			r.Put("/api/admin/users/{sub}", adminH.UpdateUser)
			r.Put("/api/admin/users/{sub}/apikey", adminH.UpdateAPIKey)
			r.Put("/api/admin/users/{sub}/prefix", adminH.UpdateSlugPrefix)
			r.Get("/api/admin/users/{sub}/links", adminH.GetUserLinks)
			r.Get("/api/admin/logs", adminH.ListLogs)

			r.Get("/api/admin/roles", adminH.ListRoles)
			r.Get("/api/admin/roles/{role}", adminH.GetRole)
			r.Put("/api/admin/roles/{role}/permissions", adminH.UpdateRolePermissions)

			r.Get("/api/admin/settings", adminH.GetSettings)
			r.Patch("/api/admin/settings", adminH.PatchSettings)
		})
	})

	// ── Server ────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
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
