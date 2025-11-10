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
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/edc"
)

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	repo := edc.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate edc tables")
	}

	service := edc.NewService(repo)
	handler := edc.NewHandler(service)

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1/edc").Subrouter()
	handler.Register(api)

	address := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.EDCServicePort)
	server := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		logger.Log.WithField("addr", address).Info("EDC service listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start edc service")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down EDC service...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("EDC service forced to shutdown")
	}
	logger.Log.Info("EDC service stopped")
}
