package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminRole = "owner"
	defaultUserRole  = "member"
)

var (
	ErrBootstrapNotAllowed = errors.New("platform already bootstrapped")
	ErrInvalidCredentials  = errors.New("invalid credentials")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Bootstrap(ctx context.Context, req models.BootstrapRequest) (models.Organization, models.User, error) {
	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return models.Organization{}, models.User{}, err
	}
	if count > 0 {
		return models.Organization{}, models.User{}, ErrBootstrapNotAllowed
	}
	if req.OrganizationName == "" || req.OrganizationSlug == "" {
		return models.Organization{}, models.User{}, fmt.Errorf("organization name and slug required")
	}
	if req.AdminEmail == "" || req.AdminPassword == "" {
		return models.Organization{}, models.User{}, fmt.Errorf("admin email and password required")
	}

	org, err := s.repo.CreateOrganization(ctx, CreateOrganizationInput{
		Name: strings.TrimSpace(req.OrganizationName),
		Slug: strings.TrimSpace(req.OrganizationSlug),
	})
	if err != nil {
		return models.Organization{}, models.User{}, err
	}

	user, err := s.createUser(ctx, createUserParams{
		OrganizationID: org.ID,
		Email:          req.AdminEmail,
		Name:           req.AdminName,
		Role:           defaultAdminRole,
		Password:       req.AdminPassword,
		AvatarURL:      req.AdminAvatarURL,
		Metadata:       req.AdminMetadata,
	})
	if err != nil {
		return models.Organization{}, models.User{}, err
	}

	return org, user, nil
}

type createUserParams struct {
	OrganizationID uuid.UUID
	Email          string
	Name           string
	Role           string
	Password       string
	AvatarURL      string
	Metadata       map[string]interface{}
}

func (s *Service) createUser(ctx context.Context, params createUserParams) (models.User, error) {
	if params.Password == "" {
		return models.User{}, fmt.Errorf("password required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}

	return s.repo.CreateUser(ctx, CreateUserInput{
		OrganizationID: params.OrganizationID,
		Email:          params.Email,
		Name:           params.Name,
		Role:           params.Role,
		PasswordHash:   string(hash),
		AvatarURL:      params.AvatarURL,
		Metadata:       params.Metadata,
	})
}

func (s *Service) RegisterUser(ctx context.Context, actor models.User, req models.RegisterUserRequest) (models.User, error) {
	if actor.Role != defaultAdminRole && actor.Role != "admin" {
		return models.User{}, fmt.Errorf("insufficient permissions")
	}
	if req.OrganizationID == uuid.Nil {
		req.OrganizationID = actor.OrganizationID
	}
	role := req.Role
	if role == "" {
		role = defaultUserRole
	}
	return s.createUser(ctx, createUserParams{
		OrganizationID: req.OrganizationID,
		Email:          req.Email,
		Name:           req.Name,
		Role:           role,
		Password:       req.Password,
		AvatarURL:      req.AvatarURL,
		Metadata:       req.Metadata,
	})
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return models.User{}, ErrInvalidCredentials
		}
		return models.User{}, err
	}
	if password == "" {
		return models.User{}, ErrInvalidCredentials
	}

	hash, err := s.repo.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return models.User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return models.User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}
