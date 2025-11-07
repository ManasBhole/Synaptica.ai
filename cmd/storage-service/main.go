package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/pipeline"
	"github.com/synaptica-ai/platform/pkg/storage"
	"gorm.io/datatypes"
)

type StorageApp struct {
	lakehouse *storage.LakehouseWriter
	olap      *storage.OLAPWriter
	features  *storage.FeatureStore
	consumer  *kafka.Consumer
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	lakehouseWriter := storage.NewLakehouseWriter(db)
	if err := lakehouseWriter.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate lakehouse table")
	}

	olapWriter := storage.NewOLAPWriter(db)
	if err := olapWriter.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate OLAP table")
	}

	redisClient := database.GetRedis()
	featureStore := storage.NewFeatureStore(db, redisClient, cfg.FeatureOnlinePrefix, cfg.FeatureCacheTTL)
	if err := featureStore.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate feature store table")
	}

	app := &StorageApp{
		lakehouse: lakehouseWriter,
		olap:      olapWriter,
		features:  featureStore,
	}
	app.consumer = kafka.NewConsumer(cfg.LinkageOutputTopic, "storage-service")
	defer app.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.consumer.Consume(ctx, app.handleEvent); err != nil {
			logger.Log.WithError(err).Fatal("consumer error")
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8086"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8086",
		}).Info("Storage Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Storage Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("Storage Service stopped")
}

func (a *StorageApp) handleEvent(ctx context.Context, event models.Event) error {
	record, linkageResult, err := parseLinkagePayload(event.Data)
	if err != nil {
		return err
	}

	fact := &storage.LakehouseFact{
		ID:           record.ID,
		MasterID:     linkageResult.MasterPatientID,
		PatientID:    record.PatientID,
		ResourceType: record.ResourceType,
		Canonical:    datatypes.JSONMap(record.Canonical),
		Codes:        toJSONMap(record.Codes),
		Timestamp:    record.Timestamp,
	}
	if err := a.lakehouse.Write(ctx, fact); err != nil {
		logger.Log.WithError(err).Error("failed to write lakehouse fact")
	}

	rollup := &storage.Rollup{
		ID:        fmt.Sprintf("%s:%s:%d", linkageResult.MasterPatientID, record.ResourceType, time.Now().UnixNano()),
		MasterID:  linkageResult.MasterPatientID,
		PatientID: record.PatientID,
		Metric:    strings.ToLower(record.ResourceType),
		Value:     datatypes.JSONMap(record.Canonical),
		EventTime: record.Timestamp,
	}
	if err := a.olap.Write(ctx, rollup); err != nil {
		logger.Log.WithError(err).Error("failed to write OLAP rollup")
	}

	features := pipeline.ExtractFeatures(record, linkageResult)
	if err := a.features.BuildFeatures(ctx, linkageResult.MasterPatientID, features, 1); err != nil {
		logger.Log.WithError(err).Error("failed to store offline features")
	}
	if err := a.features.MaterializeHotFeatures(ctx, linkageResult.MasterPatientID, features); err != nil {
		logger.Log.WithError(err).Error("failed to cache online features")
	}

	return nil
}

func parseLinkagePayload(data map[string]interface{}) (*models.NormalizedRecord, *models.LinkageResult, error) {
	linkageMap, ok := data["linkage"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("linkage payload missing")
	}
	linkageBytes, err := json.Marshal(linkageMap)
	if err != nil {
		return nil, nil, err
	}
	var linkageResult models.LinkageResult
	if err := json.Unmarshal(linkageBytes, &linkageResult); err != nil {
		return nil, nil, err
	}

	recordMap, ok := data["canonical"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("canonical payload missing")
	}
	recordBytes, err := json.Marshal(recordMap)
	if err != nil {
		return nil, nil, err
	}
	var record models.NormalizedRecord
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return nil, nil, err
	}

	return &record, &linkageResult, nil
}

func toJSONMap(in map[string]string) datatypes.JSONMap {
	out := make(datatypes.JSONMap, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
