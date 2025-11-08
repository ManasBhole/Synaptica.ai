package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/analytics/cohort"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/gateway/auth"
	"github.com/synaptica-ai/platform/pkg/gateway/httpclient"
	"github.com/synaptica-ai/platform/pkg/gateway/middleware"
	"github.com/synaptica-ai/platform/pkg/gateway/routes"
	"github.com/synaptica-ai/platform/pkg/storage"
)

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to postgres")
	}

	// Initialize OIDC authenticator
	oidcAuth, err := auth.NewOIDCAuthenticator(cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret)
	if err != nil {
		logger.Log.WithError(err).Warn("OIDC authentication not configured, running without auth")
	}

	// Shared HTTP client for downstream calls
	client := httpclient.New(cfg.GatewayRequestTimeout)

	// Setup router
	router := mux.NewRouter()

	// Middleware
	router.Use(middleware.Logging)
	router.Use(middleware.Recovery)
	if oidcAuth != nil {
		router.Use(middleware.Authenticate(oidcAuth))
	}
	router.Use(middleware.RLS) // Row-Level Security
	router.Use(middleware.CORS)
	router.Use(middleware.RateLimit(cfg.GatewayRateLimitRPS, cfg.GatewayRateLimitBurst))
	router.Use(middleware.BodyLimit(cfg.MaxRequestBody))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// TODO: add downstream checks (Kafka, etc.) once available
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	ingestionProxy := &routes.IngestionProxy{Client: client, Cfg: cfg}
	routes.RegisterIngestionRoutes(apiRouter, ingestionProxy)

	metricsHandler := routes.NewMetricsHandler(db)
	metricsHandler.Register(apiRouter)

	alertsHandler := routes.NewAlertsHandler(db)
	alertsHandler.Register(apiRouter)

	lakehouse := storage.NewLakehouseWriter(db)
	olap := storage.NewOLAPWriter(db)
	cohortService := cohort.NewService(lakehouse, olap)
	cohortHandler := routes.NewCohortHandler(cohortService)
	cohortHandler.Register(apiRouter)

	// Server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": cfg.ServerPort,
		}).Info("API Gateway started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("API Gateway stopped")
}
