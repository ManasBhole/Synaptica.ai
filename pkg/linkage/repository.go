package linkage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNoMatch = errors.New("no linkage found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&MasterPatient{}, &PatientLink{})
}

func (r *Repository) FindMasterByDeterministicKey(ctx context.Context, key string) (string, error) {
	var link PatientLink
	result := r.db.WithContext(ctx).
		Where("deterministic_key = ?", key).
		Order("created_at DESC").
		First(&link)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", ErrNoMatch
	}
	return link.MasterID, result.Error
}

func (r *Repository) CreateMaster(ctx context.Context) (string, error) {
	id := uuid.New().String()
	master := MasterPatient{ID: id, CreatedAt: time.Now().UTC()}
	if err := r.db.WithContext(ctx).Create(&master).Error; err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) SaveLink(ctx context.Context, link *PatientLink) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	link.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *Repository) RecentLinks(ctx context.Context, limit int) ([]PatientLink, error) {
	var links []PatientLink
	if limit <= 0 {
		limit = 100
	}
	result := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&links)
	return links, result.Error
}

func (r *Repository) FindLinksByPatient(ctx context.Context, patientID string, limit int) ([]PatientLink, error) {
	if limit <= 0 {
		limit = 25
	}
	var links []PatientLink
	result := r.db.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Limit(limit).
		Find(&links)
	return links, result.Error
}
