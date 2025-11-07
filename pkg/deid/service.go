package deid

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type Service struct {
	repo *Repository
	salt string
}

func NewService(repo *Repository, salt string) *Service {
	return &Service{repo: repo, salt: salt}
}

func (s *Service) Tokenize(ctx context.Context, data map[string]interface{}, detection models.PHIDetectionResult) (map[string]interface{}, map[string]string, string, error) {
	tokenized := make(map[string]interface{}, len(data))
	vault := make(map[string]string)
	targets := make(map[string]struct{})
	for _, pos := range detection.Positions {
		if pos.Value != "" {
			targets[pos.Value] = struct{}{}
		}
	}

	for key, value := range data {
		tokenized[key] = s.tokenizeValue(ctx, value, vault, targets)
	}

	return tokenized, vault, DefaultAnonymity, nil
}

func (s *Service) tokenizeValue(ctx context.Context, value interface{}, vault map[string]string, targets map[string]struct{}) interface{} {
	switch v := value.(type) {
	case string:
		if !shouldTokenize(v, targets) {
			return v
		}
		token := s.generateToken(v)
		vault[token] = v
		_ = s.repo.Save(ctx, token, v)
		return token
	case map[string]interface{}:
		res := make(map[string]interface{}, len(v))
		for k, nested := range v {
			res[k] = s.tokenizeValue(ctx, nested, vault, targets)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, nested := range v {
			res[i] = s.tokenizeValue(ctx, nested, vault, targets)
		}
		return res
	default:
		return value
	}
}

func shouldTokenize(value string, targets map[string]struct{}) bool {
	if len(targets) == 0 {
		return false
	}
	for match := range targets {
		if match != "" && strings.Contains(value, match) {
			return true
		}
	}
	return false
}

func (s *Service) generateToken(value string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", s.salt, value, uuid.New().String())))
	return "token_" + strings.ToLower(hex.EncodeToString(hash[:8]))
}
