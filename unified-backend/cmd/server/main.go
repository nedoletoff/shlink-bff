package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unified-backend/internal/config"
	"unified-backend/internal/controller"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/migrations"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
	"unified-backend/internal/shlinkctl"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()

	// ── Postgres ──────────────────────────────────────────────────────────────
	db, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	if err := postgres.RunMigrations(ctx, db, migrations.FS); err != nil {
		slog.Error("postgres migrate", "err", err)
		os.Exit(1)
	}

	// ── Репозитории ──────────────────────────────────────────────────────────
	userRepo := postgres.NewUserRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	ownerRepo := postgres.NewURLOwnershipRepository(db)
	settingsRepo := postgres.NewServerSettingsRepository(db)
	roleRepo := postgres.NewRoleRepository(db)
	permRepo := postgres.NewPermissionRepository(db)

	// ── Настройки из БД ─────────────────────────────────────────────────────
	if err := settingsRepo.SeedFromEnv(ctx, cfg); err != nil {
		slog.Warn("settings: seed from env failed", "err", err)
	}
	if err := settingsRepo.ApplyAll(ctx, cfg); err != nil {
		slog.Warn("settings: apply from db failed", "err", err)
	}

	// ── Shlink клиент ───────────────────────────────────────────────────────
	shlinkClient := shlink.NewClient(cfg.ShlinkInternalURL)
	if err := shlinkClient.ValidateVersion(ctx, 3, 5, 3*time.Second); err != nil {
		slog.Error("shlink version check", "err", err)
		os.Exit(1)
	}

	// ── Сервисы и контроллеры ────────────────────────────────────────────────
	shlinkSvc := service.NewShlinkService(shlinkClient, cfg)
	permSvc := service.NewPermissionService(db)
	permCtrl := controller.NewPermissionController(permSvc)

	// ── Провижинер API-ключей ────────────────────────────────────────────────
	var runner shlinkctl.Runner
	if cfg.ShlinkRunnerMode == "native" {
		runner = shlinkctl.NewNativeRunner(cfg.ShlinkBin)
	} else {
		runner = shlinkctl.NewDockerRunner(cfg.ShlinkContainerName)
	}
	provisioner := shlinkctl.NewProvisioner(db, runner)

	// ── Хендлеры ────────────────────────────────────────────────────────────
	meH := handler.NewMeHandler(permSvc)
	dashH := handler.NewDashboardHandler(shlinkSvc, ownerRepo, permCtrl)
	proxyH := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo, ownerRepo, cfg, permCtrl)
	userH := handler.NewUserHandler(userRepo, auditRepo, permCtrl, permSvc)
	rolesH := handler.NewRoleHandler(roleRepo, permRepo, permCtrl, permSvc)
	urlDetailH := handler.NewURLDetailHandler(shlinkSvc, ownerRepo, permCtrl)
	systemH := handler.NewSystemHandler(cfg, shlinkSvc, settingsRepo, permCtrl)
	lifecycleH := handler.NewURLLifecycleHandler(shlinkSvc, ownerRepo, auditRepo, permCtrl)
	permsH := handler.NewPermissionsHandler()

	// ── Роутер ──────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   cfg.CORSAllowedMethods,
		AllowedHeaders:   cfg.CORSAllowedHeaders,
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	// Все API-маршруты проходят через аутентификацию и провизионирование
	r.Group(func(r chi.Router) {
		r.Use(middleware.ExtractIdentity(cfg.RoleGroups, cfg.TrustedHeaderSecret, cfg.DefaultRole))
		r.Use(middleware.RequireActiveUser(userRepo, auditRepo, provisioner, cfg, permSvc))

		// ── Текущий пользователь ────────────────────────────────────────────
		r.Get("/api/me", meH.ServeHTTP)

		// ── Дашборд ─────────────────────────────────────────────────────────
		r.Get("/api/dashboard", dashH.ServeHTTP)

		// ── Shlink proxy ───────────────────────────────────────────────────
		r.Get("/api/shlink/short-urls", proxyH.ListShortURLs)
		r.Post("/api/shlink/short-urls", proxyH.CreateShortURL)
		r.Patch("/api/shlink/short-urls/{shortCode}", proxyH.UpdateShortURL)
		r.Delete("/api/shlink/short-urls/{shortCode}", proxyH.DeleteShortURL)

		// ── Lifecycle (деактивация, активация, полное удаление) ─────────────
		r.Post("/api/shlink/short-urls/{shortCode}/deactivate", lifecycleH.DeactivateURL)
		r.Post("/api/shlink/short-urls/{shortCode}/activate", lifecycleH.ActivateURL)
		r.Delete("/api/shlink/short-urls/{shortCode}/permanent", lifecycleH.DeleteURLPermanently)

		// ── Теги ───────────────────────────────────────────────────────────
		r.Get("/api/shlink/tags", proxyH.ListTags)
		r.Post("/api/shlink/tags", proxyH.CreateTag)
		r.Put("/api/shlink/tags/{tagId}", proxyH.RenameTag)
		r.Delete("/api/shlink/tags/{tagId}", proxyH.DeleteTag)

		// ── Детали ссылки (статистика) ─────────────────────────────────────
		r.Get("/api/urls/{shortCode}/detail", urlDetailH.GetURLDetail)

		// ── Системные настройки ────────────────────────────────────────────
		r.Get("/api/settings", systemH.GetSettings)
		r.Patch("/api/settings", systemH.PatchSettings)

		// ── Управление пользователями ──────────────────────────────────────
		r.Get("/api/users", userH.ListUsers)
		r.Get("/api/users/{sub}", userH.GetUser)
		r.Put("/api/users/{sub}", userH.UpdateUser)
		r.Patch("/api/users/{sub}/role", userH.PatchUserRole)
		r.Get("/api/users/{sub}/links", userH.GetUserLinks)

		// ── Аудит ──────────────────────────────────────────────────────────
		r.Get("/api/audit", userH.ListAudit)

		// ── Роли и разрешения ──────────────────────────────────────────────
		r.Get("/api/roles", rolesH.ListRoles)
		r.Post("/api/roles", rolesH.CreateRole)
		r.Get("/api/roles/{role}", rolesH.GetRole)
		r.Put("/api/roles/{role}/permissions", rolesH.UpsertRolePermissions)
		r.Delete("/api/roles/{role}", rolesH.DeleteRole)

		// ── Реестр разрешений (для UI) ─────────────────────────────────────
		r.Get("/api/permissions", permsH.ListPermissions)
	})

	// ── HTTP сервер ─────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
	slog.Info("server stopped")
}

