package training

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/ml/linear"
	"github.com/synaptica-ai/platform/pkg/storage"
	"gorm.io/datatypes"
)

type Service struct {
	repo         *Repository
	lakehouse    *storage.LakehouseWriter
	featureStore *storage.FeatureStore
	artifactDir  string
	workerSem    chan struct{}
	delay        time.Duration
}

func NewService(repo *Repository, lakehouse *storage.LakehouseWriter, featureStore *storage.FeatureStore, artifactDir string, maxWorkers int, delay time.Duration) (*Service, error) {
	s := &Service{
		repo:         repo,
		lakehouse:    lakehouse,
		featureStore: featureStore,
		artifactDir:  artifactDir,
		delay:        delay,
	}
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	s.workerSem = make(chan struct{}, maxWorkers)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Create(ctx context.Context, input CreateJobInput) (models.TrainingJob, error) {
	jobID := uuid.New()
	job := &JobModel{
		ID:        jobID,
		ModelType: input.ModelType,
		Config:    datatypes.JSONMap(input.Config),
		Filters:   datatypes.JSONMap(input.Filters),
		Status:    StatusQueued,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, job); err != nil {
		return models.TrainingJob{}, err
	}
	go s.run(jobID, input)
	return toDomain(job), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (models.TrainingJob, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return models.TrainingJob{}, err
	}
	return toDomain(job), nil
}

func (s *Service) List(ctx context.Context, limit int) ([]models.TrainingJob, error) {
	jobs, err := s.repo.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	results := make([]models.TrainingJob, 0, len(jobs))
	for _, job := range jobs {
		copy := job
		results = append(results, toDomain(&copy))
	}
	return results, nil
}

func (s *Service) GetArtifact(id uuid.UUID) (Artifact, error) {
	job, err := s.repo.Get(context.Background(), id)
	if err != nil {
		return Artifact{}, err
	}
	metrics := map[string]interface{}{}
	if job.Metrics != nil {
		metrics = map[string]interface{}(job.Metrics)
	}
	return Artifact{JobID: job.ID, Path: job.ArtifactPath, Metrics: metrics}, nil
}

func (s *Service) run(jobID uuid.UUID, input CreateJobInput) {
	s.workerSem <- struct{}{}
	defer func() { <-s.workerSem }()

	ctx := context.Background()
	start := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, jobID, StatusRunning, nil, "", ""); err != nil {
		logger.Log.WithError(err).Error("failed to mark job running")
	}
	if err := s.repo.SetTimestamps(ctx, jobID, &start, nil); err != nil {
		logger.Log.WithError(err).Error("failed to set start timestamp")
	}

	trainingData, err := s.lakehouse.GetTrainingData(ctx, input.Filters)
	if err != nil {
		s.failJob(ctx, jobID, fmt.Errorf("lakehouse query failed: %w", err))
		return
	}

	featureViews, err := s.featureStore.GetFeatureViews(ctx, extractFeatureNames(input.Config))
	if err != nil {
		s.failJob(ctx, jobID, fmt.Errorf("feature store query failed: %w", err))
		return
	}

	time.Sleep(s.delay)

	samples, labels, featureNames, buildErr := buildDataset(trainingData, input.Config)
	if buildErr != nil {
		s.failJob(ctx, jobID, buildErr)
		return
	}

	opts := linear.Options{
		Epochs:       intFromConfig(input.Config, "epochs", 200),
		LearningRate: floatFromConfig(input.Config, "learning_rate", 0.01),
	}
	weights, trainMetrics := linear.TrainLogistic(samples, labels, opts)

	metrics := map[string]interface{}{
		"training_samples": len(samples),
		"feature_views":    len(featureViews),
		"epochs":           opts.Epochs,
		"loss":             trainMetrics.Loss,
		"accuracy":         trainMetrics.Accuracy,
		"duration_seconds": s.delay.Seconds(),
		"timestamp":        time.Now().UTC(),
		"threshold":        floatFromConfig(input.Config, "threshold", 120),
		"weights": map[string]interface{}{
			"bias":         weights.Bias,
			"coefficients": weights.Coefficients,
		},
	}

	artifactPath, err := s.writeArtifact(jobID, input, metrics, weights, featureNames)
	if err != nil {
		s.failJob(ctx, jobID, fmt.Errorf("artifact write failed: %w", err))
		return
	}

	if err := s.repo.UpdateStatus(ctx, jobID, StatusCompleted, metrics, artifactPath, ""); err != nil {
		logger.Log.WithError(err).Error("failed to mark job complete")
	}
	completed := time.Now().UTC()
	if err := s.repo.SetTimestamps(ctx, jobID, nil, &completed); err != nil {
		logger.Log.WithError(err).Error("failed to set completion timestamp")
	}
}

func (s *Service) failJob(ctx context.Context, jobID uuid.UUID, err error) {
	logger.Log.WithError(err).Error("training job failed")
	_ = s.repo.UpdateStatus(ctx, jobID, StatusFailed, nil, "", err.Error())
	completed := time.Now().UTC()
	_ = s.repo.SetTimestamps(ctx, jobID, nil, &completed)
}

func (s *Service) writeArtifact(jobID uuid.UUID, input CreateJobInput, metrics map[string]interface{}, weights linear.Weights, featureNames []string) (string, error) {
	artifact := map[string]interface{}{
		"job_id": jobID.String(),
		"model": map[string]interface{}{
			"type":          input.ModelType,
			"algorithm":     "logistic_regression",
			"feature_names": featureNames,
			"weights": map[string]interface{}{
				"bias":         weights.Bias,
				"coefficients": weights.Coefficients,
			},
		},
		"config":     input.Config,
		"filters":    input.Filters,
		"metrics":    metrics,
		"created_at": time.Now().UTC(),
	}
	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.artifactDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(s.artifactDir, fmt.Sprintf("%s.json", jobID.String()))
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", err
	}
	latestPath := filepath.Join(s.artifactDir, fmt.Sprintf("%s_latest.json", input.ModelType))
	if err := os.WriteFile(latestPath, payload, 0o644); err != nil {
		logger.Log.WithError(err).Warn("failed to update latest artifact pointer")
	}
	return path, nil
}

func toDomain(job *JobModel) models.TrainingJob {
	result := models.TrainingJob{
		ID:           job.ID,
		ModelType:    job.ModelType,
		Status:       job.Status,
		CreatedAt:    job.CreatedAt,
		StartedAt:    job.StartedAt,
		CompletedAt:  job.CompletedAt,
		ArtifactPath: job.ArtifactPath,
		ErrorMessage: job.ErrorMessage,
	}
	if job.Config != nil {
		result.Config = map[string]interface{}(job.Config)
	}
	if job.Metrics != nil {
		result.Metrics = map[string]interface{}(job.Metrics)
	}
	return result
}

func extractFeatureNames(config map[string]interface{}) []string {
	if config == nil {
		return nil
	}
	if views, ok := config["feature_views"].([]interface{}); ok {
		var names []string
		for _, v := range views {
			if name, ok := v.(string); ok {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func floatFromConfig(config map[string]interface{}, key string, defaultVal float64) float64 {
	if config == nil {
		return defaultVal
	}
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
	}
	return defaultVal
}

func intFromConfig(config map[string]interface{}, key string, defaultVal int) int {
	if config == nil {
		return defaultVal
	}
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultVal
}

func buildDataset(trainingData []map[string]interface{}, config map[string]interface{}) ([][]float64, []float64, []string, error) {
	threshold := floatFromConfig(config, "threshold", 120)
	var samples [][]float64
	var labels []float64
	featureNames := []string{"value"}
	for _, record := range trainingData {
		canonical, ok := record["canonical"].(map[string]interface{})
		if !ok {
			continue
		}
		value, err := numericFromAny(canonical["value"])
		if err != nil {
			continue
		}
		samples = append(samples, []float64{value})
		if value >= threshold {
			labels = append(labels, 1)
		} else {
			labels = append(labels, 0)
		}
	}
	if len(samples) == 0 {
		return nil, nil, nil, errors.New("no numeric data available for training")
	}
	return samples, labels, featureNames, nil
}

func numericFromAny(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconvParse(v)
	default:
		return 0, fmt.Errorf("unsupported value type %T", value)
	}
}

func strconvParse(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseFloat(s, 64)
}
