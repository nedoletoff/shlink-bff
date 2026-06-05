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

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/migrations"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
	"unified-backend/internal/shlinkctl"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Миграции — запускаем до любого обращения к схеме.
	if err := postgres.RunMigrations(ctx, pool, migrations.FS); err != nil {
		slog.Error("database migration failed", "err", err)
		os.Exit(1)
	}

	// Репозитории
	userRepo := postgres.NewUserRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)
	permsRepo := postgres.NewRolePermissionsRepository(pool)

	// Кеш permissions — загружаем при старте
	permsCache := service.NewPermissionsCache(permsRepo, cfg.AdminRole)
	if err := permsCache.Load(ctx); err != nil {
		// Не фатально: кеш вернёт дефолты для известных ролей
		slog.Warn("permissions cache load failed, using defaults", "err", err)
	}

	// Shlink
	shlinkClient := shlink.NewClient(cfg.ShlinkURL)
	shlinkSvc := service.NewShlinkService(shlinkClient, cfg, permsCache)

	if err := shlinkClient.ValidateVersion(ctx, 5, 10, 3*time.Second); err != nil {
		slog.Error("shlink version validation failed", "err", err)
		os.Exit(1)
	}

	// CLI provisioner — генерация per-user API-ключей
	var runner shlinkctl.Runner
	if cfg.ShlinkRunnerMode == "native" {
		runner = shlinkctl.NewNativeRunner(cfg.ShlinkBin)
		slog.Info("shlinkctl: using native runner", "bin", cfg.ShlinkBin)
	} else {
		runner = shlinkctl.NewDockerRunner(cfg.ShlinkContainerName)
		slog.Info("shlinkctl: using docker runner", "container", cfg.ShlinkContainerName)
	}
	provisioner := shlinkctl.NewProvisioner(pool, runner)

	// Хендлеры
	meH := handler.NewMeHandler(cfg, permsCache)
	dashH := handler.NewDashboardHandler(shlinkSvc)
	proxyH := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo)
	adminH := handler.NewAdminHandler(userRepo, auditRepo)
	rolesH := handler.NewRolesHandler(permsCache, permsRepo)
	urlDetailH := handler.NewURLDetailHandler(shlinkSvc)
	settingsH := handler.NewSettingsHandler(cfg, shlinkSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"unified-backend"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.ExtractIdentity(cfg.RoleGroups))
		r.Use(middleware.RequestLogger)
		r.Use(middleware.RequireActiveUser(userRepo, auditRepo, provisioner, cfg))

		r.Get("/api/me", meH.ServeHTTP)

		// Dashboard — 4 отдельных endpointа (соответствуют вызовам с фронтенда)
		r.Get("/api/dashboard/overview", dashH.GetOverview)
		r.Get("/api/dashboard/users", dashH.GetUsersActivity)
		r.Get("/api/dashboard/urls", dashH.GetUrlsStats)
		r.Get("/api/dashboard/devices", dashH.GetDevicesStats)

		// URL detail — детальная статистика отдельной ссылки
		r.Get("/api/urls/{shortCode}/detail", urlDetailH.GetURLDetail)

		// Shlink proxy — enforcement внутри хендлеров через PermissionsCache
		r.Get("/api/shlink/short-urls", proxyH.ListShortURLs)
		r.Post("/api/shlink/short-urls", proxyH.CreateShortURL)
		r.Patch("/api/shlink/short-urls/{shortCode}", proxyH.UpdateShortURL)
		r.Delete("/api/shlink/short-urls/{shortCode}", proxyH.DeleteShortURL)

		r.Get("/api/shlink/tags", proxyH.ListTags)
		r.Put("/api/shlink/tags/{tagId}", proxyH.RenameTag)
		r.Delete("/api/shlink/tags/{tagId}", proxyH.DeleteTag)

		// Admin-only: управление пользователями, аудит, роли, настройки
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly(cfg.AdminRole, auditRepo))

			r.Get("/api/admin/users", adminH.ListUsers)
			r.Get("/api/admin/users/{sub}", adminH.GetUser)
			r.Put("/api/admin/users/{sub}", adminH.UpdateUser)
			r.Put("/api/admin/users/{sub}/apikey", adminH.UpdateAPIKey)
			r.Put("/api/admin/users/{sub}/prefix", adminH.UpdateSlugPrefix)
			r.Get("/api/admin/users/{sub}/links", adminH.GetUserLinks)
			r.Get("/api/admin/logs", adminH.ListLogs)

			// Settings — чтение/обновление runtime-настроек
			r.Get("/api/admin/settings", settingsH.GetSettings)
			r.Patch("/api/admin/settings", settingsH.PatchSettings)

			// Управление permissions ролей — только роли с can_manage_roles
			r.With(
				middleware.RequirePermission(permsCache,
					func(p domain.RolePermissions) bool { return p.CanManageRoles },
					auditRepo,
				),
			).Get("/api/admin/roles", rolesH.ListRoles)
			r.With(
				middleware.RequirePermission(permsCache,
					func(p domain.RolePermissions) bool { return p.CanManageRoles },
					auditRepo,
				),
			).Get("/api/admin/roles/{role}", rolesH.GetRole)
			r.With(
				middleware.RequirePermission(permsCache,
					func(p domain.RolePermissions) bool { return p.CanManageRoles },
					auditRepo,
				),
			).Put("/api/admin/roles/{role}/permissions", rolesH.UpsertRolePermissions)
		})
	})

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("unified-backend starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
