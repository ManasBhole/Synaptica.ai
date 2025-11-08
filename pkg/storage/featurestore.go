package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OfflineFeature struct {
	ID        string            `gorm:"primaryKey;column:id"`
	PatientID string            `gorm:"column:patient_id"`
	Features  datatypes.JSONMap `gorm:"column:features"`
	Version   int               `gorm:"column:version"`
	CreatedAt time.Time         `gorm:"column:created_at"`
}

func (OfflineFeature) TableName() string {
	return "feature_offline_store"
}

type FeatureStore struct {
	offlineDB *gorm.DB
	online    *redis.Client
	cacheTTL  time.Duration
	keyPrefix string
}

func NewFeatureStore(db *gorm.DB, redisClient *redis.Client, prefix string, ttl time.Duration) *FeatureStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &FeatureStore{offlineDB: db, online: redisClient, cacheTTL: ttl, keyPrefix: prefix}
}

func (fs *FeatureStore) AutoMigrate() error {
	return fs.offlineDB.AutoMigrate(&OfflineFeature{})
}

func (fs *FeatureStore) BuildFeatures(ctx context.Context, patientID string, features map[string]interface{}, version int) error {
	if features == nil {
		features = map[string]interface{}{}
	}
	record := &OfflineFeature{
		ID:        fmt.Sprintf("%s:%d", patientID, version),
		PatientID: patientID,
		Features:  datatypes.JSONMap(features),
		Version:   version,
		CreatedAt: time.Now().UTC(),
	}
	return fs.offlineDB.WithContext(ctx).Create(record).Error
}

func (fs *FeatureStore) MaterializeHotFeatures(ctx context.Context, patientID string, features map[string]interface{}) error {
	if fs.online == nil {
		return nil
	}
	payload, err := json.Marshal(features)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", fs.keyPrefix, patientID)
	return fs.online.Set(ctx, key, payload, fs.cacheTTL).Err()
}

func (fs *FeatureStore) GetFeatures(ctx context.Context, patientID string) (map[string]interface{}, error) {
	if fs.online == nil {
		return map[string]interface{}{}, nil
	}
	key := fmt.Sprintf("%s%s", fs.keyPrefix, patientID)
	val, err := fs.online.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return map[string]interface{}{}, nil
		}
		return map[string]interface{}{}, err
	}
	var features map[string]interface{}
	if err := json.Unmarshal([]byte(val), &features); err != nil {
		return map[string]interface{}{}, err
	}
	return features, nil
}

func (fs *FeatureStore) GetFeatureViews(ctx context.Context, names []string) (map[string]interface{}, error) {
	var rows []OfflineFeature
	query := fs.offlineDB.WithContext(ctx).Order("created_at desc").Limit(200)
	if len(names) > 0 {
		query = query.Where("id IN ?", names)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]interface{}, len(rows))
	for _, row := range rows {
		result[row.ID] = map[string]interface{}(row.Features)
	}
	return result, nil
}

func (fs *FeatureStore) GetLatestOfflineFeatures(ctx context.Context, patientID string) (map[string]interface{}, error) {
	if fs.offlineDB == nil {
		return map[string]interface{}{}, nil
	}
	var feature OfflineFeature
	query := fs.offlineDB.WithContext(ctx).Where("patient_id = ?", patientID).Order("created_at desc").Limit(1)
	if err := query.First(&feature).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]interface{}{}, nil
		}
		return map[string]interface{}{}, err
	}
	return map[string]interface{}(feature.Features), nil
}
