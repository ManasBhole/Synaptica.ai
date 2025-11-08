package cohort

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/gorm"
)

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) AutoMigrate() error {
	type CohortTemplate struct {
		ID          string    `gorm:"primaryKey;column:id"`
		TenantID    *string   `gorm:"column:tenant_id"`
		Name        string    `gorm:"column:name"`
		Description string    `gorm:"column:description"`
		DSL         string    `gorm:"column:dsl"`
		Tags        []string  `gorm:"type:text[];column:tags"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	return r.db.AutoMigrate(&CohortTemplate{})
}

func (r *TemplateRepository) List(ctx context.Context, tenantID string, limit int) ([]models.CohortTemplate, error) {
	if limit <= 0 {
		limit = 25
	}
	var rows []struct {
		ID          string    `gorm:"column:id"`
		TenantID    *string   `gorm:"column:tenant_id"`
		Name        string    `gorm:"column:name"`
		Description string    `gorm:"column:description"`
		DSL         string    `gorm:"column:dsl"`
		Tags        []string  `gorm:"column:tags"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	query := r.db.WithContext(ctx).Table("cohort_templates").Order("created_at DESC").Limit(limit)
	if tenantID != "" {
		query = query.Where("tenant_id IS NULL OR tenant_id = ?", tenantID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	templates := make([]models.CohortTemplate, 0, len(rows))
	for _, row := range rows {
		template := models.CohortTemplate{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			DSL:         row.DSL,
			Tags:        row.Tags,
			CreatedAt:   row.CreatedAt,
		}
		if row.TenantID != nil {
			template.TenantID = *row.TenantID
		}
		templates = append(templates, template)
	}
	return templates, nil
}

func (r *TemplateRepository) Create(ctx context.Context, tmpl models.CohortTemplate) (models.CohortTemplate, error) {
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}
	record := map[string]interface{}{
		"id":          tmpl.ID,
		"tenant_id":   nullableString(tmpl.TenantID),
		"name":        tmpl.Name,
		"description": tmpl.Description,
		"dsl":         tmpl.DSL,
		"tags":        tmpl.Tags,
	}
	if err := r.db.WithContext(ctx).Table("cohort_templates").Create(&record).Error; err != nil {
		return models.CohortTemplate{}, err
	}
	var row struct {
		ID          string    `gorm:"column:id"`
		TenantID    *string   `gorm:"column:tenant_id"`
		Name        string    `gorm:"column:name"`
		Description string    `gorm:"column:description"`
		DSL         string    `gorm:"column:dsl"`
		Tags        []string  `gorm:"column:tags"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}
	if err := r.db.WithContext(ctx).Table("cohort_templates").Where("id = ?", tmpl.ID).First(&row).Error; err != nil {
		return models.CohortTemplate{}, err
	}
	tmpl = models.CohortTemplate{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		DSL:         row.DSL,
		Tags:        row.Tags,
		CreatedAt:   row.CreatedAt,
	}
	if row.TenantID != nil {
		tmpl.TenantID = *row.TenantID
	}
	return tmpl, nil
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
