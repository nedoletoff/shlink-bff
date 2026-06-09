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
	"unified-backend/internal/controller"
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

	// ── Postgres ──────────────────────────────────────────────────────────────────────────────
	db, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	if err := postgres.RunMigrations(ctx, db, migrations.FS); err != nil {
		slog.Error("postgres migrate", "err", err)
		os.Exit(1)
	}

	userRepo     := postgres.NewUserRepository(db)
	auditRepo    := postgres.NewAuditRepository(db)
	ownerRepo    := postgres.NewURLOwnershipRepository(db)
	settingsRepo := postgres.NewServerSettingsRepository(db)

	// ── DB config: seed on first run, then always apply ─────────────────────────────────
	if err := settingsRepo.SeedFromEnv(ctx, cfg); err != nil {
		slog.Warn("settings: seed from env failed", "err", err)
	}
	if err := settingsRepo.ApplyAll(ctx, cfg); err != nil {
		slog.Warn("settings: apply from db failed", "err", err)
	}

	// ── Shlink client ─────────────────────────────────────────────────────────────────────────
	shlinkClient := shlink.NewClient(cfg.ShlinkBaseURL)
	if err := shlinkClient.ValidateVersion(ctx, 3, 5, 3*time.Second); err != nil {
		slog.Error("shlink version check", "err", err)
		os.Exit(1)
	}

	// ── Services ────────────────────────────────────────────────────────────────────────────
	shlinkSvc := service.NewShlinkService(shlinkClient, cfg)
	permSvc   := service.NewPermissionService(db)
	permCtrl  := controller.NewPermissionController(permSvc)

	// ── Provisioner ─────────────────────────────────────────────────────────────────────────
	var runner shlinkctl.Runner
	if cfg.ShlinkRunnerMode == "native" {
		runner = shlinkctl.NewNativeRunner(cfg.ShlinkBin)
	} else {
		runner = shlinkctl.NewDockerRunner(cfg.ShlinkContainerName)
	}
	provisioner := shlinkctl.NewProvisioner(db, runner)

	// ── Handlers ────────────────────────────────────────────────────────────────────────────
	meH        := handler.NewMeHandler(cfg, permSvc)
	dashH      := handler.NewDashboardHandler(shlinkSvc, userRepo, ownerRepo)
	proxyH     := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo, ownerRepo, cfg, permCtrl)
	userH      := handler.NewUserHandler(userRepo, auditRepo, permCtrl, permSvc)
	rolesH     := handler.NewRoleHandler(permCtrl)
	urlDetailH := handler.NewURLDetailHandler(shlinkSvc, ownerRepo, permCtrl)
	systemH    := handler.NewSystemHandler(cfg, shlinkSvc, settingsRepo, permCtrl)
	lifecycleH := handler.NewURLLifecycleHandler(shlinkSvc, ownerRepo, auditRepo, permCtrl)

	// ── Router ──────────────────────────────────────────────────────────────────────────────
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

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	// Все API-маршруты доступны авторизованным пользователям.
	// Проверка конкретных permissions выполняется внутри handler'ов через permCtrl.
	r.Group(func(r chi.Router) {
		r.Use(middleware.ExtractIdentity(cfg.RoleGroups))
		r.Use(middleware.RequireActiveUser(userRepo, auditRepo, provisioner, cfg))

		// ── Текущий пользователь ──────────────────────────────────────────────────────
		r.Get("/api/me", meH.ServeHTTP)

		// ── Дашборд ───────────────────────────────────────────────────────────────────────
		r.Get("/api/dashboard", dashH.GetDashboard)

		// ── Shlink proxy ─────────────────────────────────────────────────────────────────
		r.Get("/api/shlink/short-urls", proxyH.ListShortURLs)
		r.Post("/api/shlink/short-urls", proxyH.CreateShortURL)
		r.Patch("/api/shlink/short-urls/{shortCode}", proxyH.UpdateShortURL)
		r.Delete("/api/shlink/short-urls/{shortCode}", proxyH.DeleteShortURL)

		// ── Lifecycle ────────────────────────────────────────────────────────────────────
		r.Post("/api/shlink/short-urls/{shortCode}/deactivate", lifecycleH.DeactivateURL)
		r.Post("/api/shlink/short-urls/{shortCode}/activate", lifecycleH.ActivateURL)
		r.Delete("/api/shlink/short-urls/{shortCode}/permanent", lifecycleH.DeleteURLPermanently)

		// ── Теги ───────────────────────────────────────────────────────────────────────────
		r.Get("/api/shlink/tags", proxyH.ListTags)
		r.Post("/api/shlink/tags", proxyH.CreateTag)
		r.Put("/api/shlink/tags/{tagId}", proxyH.RenameTag)
		r.Delete("/api/shlink/tags/{tagId}", proxyH.DeleteTag)

		// ── URL detail ───────────────────────────────────────────────────────────────────
		r.Get("/api/urls/{shortCode}/detail", urlDetailH.GetURLDetail)

		// ── Настройки сервера (system.config) ────────────────────────────────────────────
		r.Get("/api/settings", systemH.GetSettings)
		r.Patch("/api/settings", systemH.PatchSettings)

		// ── Управление пользователями (users.*) ───────────────────────────────────────
		r.Get("/api/users", userH.ListUsers)
		r.Get("/api/users/{sub}", userH.GetUser)
		r.Put("/api/users/{sub}", userH.UpdateUser)
		r.Patch("/api/users/{sub}/role", userH.PatchUserRole)
		r.Get("/api/users/{sub}/links", userH.GetUserLinks)

		// ── Аудит (users.view) ───────────────────────────────────────────────────────────
		r.Get("/api/audit", userH.ListAudit)

		// ── Роли (roles.*) ──────────────────────────────────────────────────────────────────
		r.Get("/api/roles", rolesH.ListRoles)
		r.Post("/api/roles", rolesH.CreateRole)
		r.Get("/api/roles/{role}", rolesH.GetRole)
		r.Put("/api/roles/{role}/permissions", rolesH.UpsertRolePermissions)
		r.Delete("/api/roles/{role}", rolesH.DeleteRole)
	})

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
