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
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/gateway/auth"
	"github.com/synaptica-ai/platform/pkg/gateway/middleware"
	"github.com/synaptica-ai/platform/pkg/gateway/routes"
)

func main() {
	logger.Init()
	cfg := config.Load()

	// Initialize OIDC authenticator
	oidcAuth, err := auth.NewOIDCAuthenticator(cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret)
	if err != nil {
		logger.Log.WithError(err).Warn("OIDC authentication not configured, running without auth")
	}

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
	router.Use(middleware.RateLimit(50, 100)) // basic per-process limiter

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	routes.RegisterIngestionRoutes(apiRouter)

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

