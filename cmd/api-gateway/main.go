package main

import (
	"context"
	"errors"
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
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/edc"
	gatewayauth "github.com/synaptica-ai/platform/pkg/gateway/auth"
	"github.com/synaptica-ai/platform/pkg/gateway/httpclient"
	"github.com/synaptica-ai/platform/pkg/gateway/middleware"
	"github.com/synaptica-ai/platform/pkg/gateway/routes"
	"github.com/synaptica-ai/platform/pkg/identity"
	"github.com/synaptica-ai/platform/pkg/linkage"
	"github.com/synaptica-ai/platform/pkg/observability/metrics"
	"github.com/synaptica-ai/platform/pkg/storage"
)

func main() {
	logger.Init()
	cfg := config.Load()
	metrics.Init()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to postgres")
	}

	jwtManager, err := gatewayauth.NewJWTManager(cfg.AuthTokenSecret, cfg.AuthTokenIssuer, cfg.AuthTokenAudience, cfg.AuthTokenTTL)
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to configure JWT manager")
	}

	identityRepo := identity.NewRepository(db)
	if err := identityRepo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Warn("Failed to migrate identity tables")
	}
	identityService := identity.NewService(identityRepo)
	if cfg.AuthBootstrapOrgName != "" && cfg.AuthBootstrapOrgSlug != "" && cfg.AuthBootstrapAdminEmail != "" && cfg.AuthBootstrapAdminPassword != "" {
		_, _, bootstrapErr := identityService.Bootstrap(context.Background(), models.BootstrapRequest{
			OrganizationName: cfg.AuthBootstrapOrgName,
			OrganizationSlug: cfg.AuthBootstrapOrgSlug,
			AdminEmail:       cfg.AuthBootstrapAdminEmail,
			AdminName:        cfg.AuthBootstrapAdminName,
			AdminPassword:    cfg.AuthBootstrapAdminPassword,
		})
		if bootstrapErr != nil && !errors.Is(bootstrapErr, identity.ErrBootstrapNotAllowed) {
			logger.Log.WithError(bootstrapErr).Warn("automatic bootstrap failed")
		}
	}

	// Shared HTTP client for downstream calls
	client := httpclient.New(cfg.GatewayRequestTimeout)

	// Setup router
	router := mux.NewRouter()

	// Middleware
	router.Use(middleware.Logging)
	router.Use(middleware.Recovery)
	router.Use(middleware.AttachUserIfPresent(jwtManager))
	router.Use(middleware.RLS) // Row-Level Security
	router.Use(middleware.CORS)
	router.Use(middleware.RateLimit(cfg.GatewayRateLimitRPS, cfg.GatewayRateLimitBurst))
	router.Use(middleware.BodyLimit(cfg.MaxRequestBody))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w)
	}).Methods("GET")

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// TODO: add downstream checks (Kafka, etc.) once available
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authHandler := routes.NewAuthHandler(identityService, jwtManager)
	authHandler.Register(authRouter)

	ingestionProxy := &routes.IngestionProxy{Client: client, Cfg: cfg}
	routes.RegisterIngestionRoutes(apiRouter, ingestionProxy)

	metricsHandler := routes.NewMetricsHandler(db)
	metricsHandler.Register(apiRouter)

	alertsHandler := routes.NewAlertsHandler(db)
	alertsHandler.Register(apiRouter)

	redisClient := database.GetRedis()
	lakehouse := storage.NewLakehouseWriter(db)
	olap := storage.NewOLAPWriter(db)
	featureStore := storage.NewFeatureStore(db, redisClient, cfg.FeatureOnlinePrefix, cfg.FeatureCacheTTL)
	linkageRepo := linkage.NewRepository(db)
	templateRepo := cohort.NewTemplateRepository(db)
	if err := templateRepo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Warn("Failed to ensure cohort templates table")
	}
	materialRepo := cohort.NewMaterializationRepository(db)
	if err := materialRepo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Warn("Failed to ensure cohort materializations table")
	}
	cohortService := cohort.NewService(
		lakehouse,
		olap,
		cohort.WithFeatureStore(featureStore),
		cohort.WithLinkageRepository(linkageRepo),
		cohort.WithTemplateRepository(templateRepo),
		cohort.WithMaterializer(materialRepo, featureStore, cfg.FeatureMaterializeWorkers),
	)
	cohortHandler := routes.NewCohortHandler(cohortService)
	cohortHandler.Register(apiRouter)

	trainingProxy := routes.NewTrainingProxy(client, cfg)
	routes.RegisterTrainingRoutes(apiRouter, trainingProxy)

	edcRepo := edc.NewRepository(db)
	if err := edcRepo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Warn("Failed to ensure EDC tables")
	}
	edcService := edc.NewService(edcRepo)
	edcHandler := edc.NewHandler(edcService)
	edcRouter := apiRouter.PathPrefix("/edc").Subrouter()
	edcHandler.Register(edcRouter)

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
