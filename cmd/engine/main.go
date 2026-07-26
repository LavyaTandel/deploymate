package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"deploymate/internal/auth"
	"deploymate/internal/cache"
	"deploymate/internal/config"
	"deploymate/internal/handler"
	"deploymate/internal/policy"
	"deploymate/internal/preview"
	"deploymate/internal/store"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	// PostgreSQL
	pgStore, err := store.NewStore(context.Background(), cfg.Database.URL())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pgStore.Close()
	log.Info().Msg("database connected")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	log.Info().Msg("redis connected")

	desiredStateCache := cache.NewRedisCache(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)

	// OIDC
	oidcValidator := auth.NewOIDCValidator(cfg)
	log.Info().Str("issuer", cfg.OIDC.Issuer).Msg("oidc validator initialized")

	// Policy engine
	policyEngine := policy.NewEngine(cfg.Policy.CacheTTL)
	_ = policyEngine
	log.Info().Dur("ttl", cfg.Policy.CacheTTL).Msg("policy engine initialized")

	// Preview manager
	previewMgr := preview.NewManager()

	// Handlers
	deploymentHandler := handler.NewDesiredStateHandler(desiredStateCache, pgStore)
	webhookHandler := handler.NewWebhookHandler("", previewMgr)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)

	// Health check (no auth)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// API routes with auth
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireOIDC(oidcValidator))

		r.Get("/api/v1/deployments/{id}/desired-state", deploymentHandler.GetDesiredState)
		r.Post("/api/v1/deployments/{id}/rollback", deploymentHandler.Rollback)
	})

	// Webhook (HMAC auth, no OIDC)
	r.Post("/api/v1/webhooks/github", webhookHandler.HandleGitHubWebhook)

	// SSE events (no auth for agent polling)
	r.Get("/api/v1/events", deploymentHandler.Events)

	// Server
	port, _ := strconv.Atoi(cfg.Server.Port)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
		log.Info().Msg("server stopped")
	}()

	log.Info().Int("port", port).Msg("engine starting")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Org-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}
