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
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

func main() {
	// Структурированный JSON-логинг (slog, Go 1.21+)
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

	// Репозитории
	userRepo  := postgres.NewUserRepository(pool)
	auditRepo := postgres.NewAuditRepository(pool)

	// Shlink клиент и сервис
	shlinkClient := shlink.NewClient(cfg.ShlinkURL)
	shlinkSvc    := service.NewShlinkService(shlinkClient, cfg)

	// Хендлеры
	meH        := handler.NewMeHandler(cfg)
	dashH      := handler.NewDashboardHandler(shlinkSvc)
	proxyH     := handler.NewShlinkProxyHandler(shlinkSvc, auditRepo)
	adminH     := handler.NewAdminHandler(userRepo, auditRepo)

	r := chi.NewRouter()

	// Базовые middleware
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Публичный healthcheck (без auth)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"unified-backend"}`))
	})

	// Все /api/* — за identity extraction + request logging + active user check
	r.Group(func(r chi.Router) {
		r.Use(middleware.ExtractIdentity)
		r.Use(middleware.RequestLogger)
		r.Use(middleware.RequireActiveUser(userRepo, auditRepo))

		// Профиль текущего пользователя
		r.Get("/api/me", meH.ServeHTTP)

		// Dashboard
		r.Get("/api/dashboard", dashH.ServeHTTP)

		// Shlink proxy — доступен обоим ролям (изоляция внутри хендлеров)
		r.Get("/api/shlink/short-urls", proxyH.ListShortURLs)
		r.Post("/api/shlink/short-urls", proxyH.CreateShortURL)
		r.Patch("/api/shlink/short-urls/{shortCode}", proxyH.UpdateShortURL)
		r.Delete("/api/shlink/short-urls/{shortCode}", proxyH.DeleteShortURL)

		r.Get("/api/shlink/tags", proxyH.ListTags)
		r.Post("/api/shlink/tags", proxyH.CreateTag)
		r.Put("/api/shlink/tags/{tagId}", proxyH.RenameTag)
		r.Delete("/api/shlink/tags/{tagId}", proxyH.DeleteTag)

		// Admin-only маршруты
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnly(auditRepo))

			r.Get("/api/admin/users", adminH.ListUsers)
			r.Get("/api/admin/users/{sub}", adminH.GetUser)
			r.Put("/api/admin/users/{sub}", adminH.UpdateUser)
			r.Put("/api/admin/users/{sub}/apikey", adminH.UpdateAPIKey)
			r.Put("/api/admin/users/{sub}/prefix", adminH.UpdateSlugPrefix)
			r.Get("/api/admin/users/{sub}/links", adminH.GetUserLinks)
			r.Get("/api/admin/logs", adminH.ListLogs)
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
