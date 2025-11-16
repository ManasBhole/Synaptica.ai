package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrEmailAlreadyExists   = errors.New("email already registered")
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type OrganizationModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string
	Slug      string `gorm:"uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (OrganizationModel) TableName() string {
	return "organizations"
}

type UserModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrganizationID uuid.UUID `gorm:"type:uuid;index"`
	Email          string    `gorm:"uniqueIndex"`
	Name           string
	Role           string `gorm:"index"`
	PasswordHash   string
	AvatarURL      string
	Metadata       datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Organization OrganizationModel `gorm:"foreignKey:OrganizationID"`
}

func (UserModel) TableName() string {
	return "users"
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&OrganizationModel{}, &UserModel{})
}

type CreateOrganizationInput struct {
	Name string
	Slug string
}

func (r *Repository) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (models.Organization, error) {
	org := OrganizationModel{
		ID:        uuid.New(),
		Name:      input.Name,
		Slug:      strings.ToLower(input.Slug),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(&org).Error; err != nil {
		return models.Organization{}, err
	}

	return models.Organization{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}

type CreateUserInput struct {
	OrganizationID uuid.UUID
	Email          string
	Name           string
	Role           string
	PasswordHash   string
	AvatarURL      string
	Metadata       map[string]interface{}
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput) (models.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))

	var existing int64
	if err := r.db.WithContext(ctx).Model(&UserModel{}).Where("email = ?", normalizedEmail).Count(&existing).Error; err != nil {
		return models.User{}, err
	}
	if existing > 0 {
		return models.User{}, ErrEmailAlreadyExists
	}

	user := UserModel{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		Email:          normalizedEmail,
		Name:           input.Name,
		Role:           input.Role,
		PasswordHash:   input.PasswordHash,
		AvatarURL:      input.AvatarURL,
		Metadata:       datatypes.JSONMap(input.Metadata),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return models.User{}, err
	}

	return mapUserModel(user), nil
}

func (r *Repository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash": passwordHash,
		"updated_at":    time.Now().UTC(),
	}).Error
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user UserModel
	err := r.db.WithContext(ctx).Where("email = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return mapUserModel(user), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user UserModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return mapUserModel(user), nil
}

func (r *Repository) GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var user UserModel
	err := r.db.WithContext(ctx).Select("password_hash").Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return user.PasswordHash, nil
}

func (r *Repository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&count).Error
	return count, err
}

func mapUserModel(user UserModel) models.User {
	return models.User{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		Name:           user.Name,
		Role:           user.Role,
		AvatarURL:      user.AvatarURL,
		Metadata:       map[string]interface{}(user.Metadata),
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}
