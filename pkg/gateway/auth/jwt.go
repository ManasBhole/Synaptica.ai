package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type JWTManager struct {
	signingKey []byte
	issuer     string
	audience   string
	ttl        time.Duration
	nowFunc    func() time.Time
}

func NewJWTManager(secret, issuer, audience string, ttl time.Duration) (*JWTManager, error) {
	if len(secret) < 16 {
		return nil, errors.New("jwt secret must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &JWTManager{
		signingKey: []byte(secret),
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
		nowFunc:    time.Now,
	}, nil
}

type Claims struct {
	ID             string    `json:"jti"`
	Issuer         string    `json:"iss"`
	Subject        string    `json:"sub"`
	Audience       string    `json:"aud"`
	IssuedAt       int64     `json:"iat"`
	NotBefore      int64     `json:"nbf"`
	ExpiresAt      int64     `json:"exp"`
	UserID         uuid.UUID `json:"uid"`
	OrganizationID uuid.UUID `json:"oid"`
	Role           string    `json:"role"`
	Email          string    `json:"email"`
	Impersonated   bool      `json:"impersonated,omitempty"`
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func (m *JWTManager) IssueToken(user models.User) (string, error) {
	now := m.nowFunc()
	header := tokenHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}
	claims := Claims{
		ID:             uuid.NewString(),
		Issuer:         m.issuer,
		Subject:        user.ID.String(),
		Audience:       m.audience,
		IssuedAt:       now.Unix(),
		NotBefore:      now.Unix(),
		ExpiresAt:      now.Add(m.ttl).Unix(),
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Role:           user.Role,
		Email:          user.Email,
	}

	headerSegment, err := encodeSegment(header)
	if err != nil {
		return "", err
	}
	payloadSegment, err := encodeSegment(claims)
	if err != nil {
		return "", err
	}

	signature := signSegments(m.signingKey, headerSegment, payloadSegment)
	return strings.Join([]string{headerSegment, payloadSegment, signature}, "."), nil
}

func (m *JWTManager) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token empty")
	}
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	expectedSig := signSegments(m.signingKey, parts[0], parts[1])
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errors.New("invalid token signature")
	}

	var claims Claims
	if err := decodeSegment(parts[1], &claims); err != nil {
		return nil, err
	}

	now := m.nowFunc().Unix()
	if claims.Issuer != m.issuer {
		return nil, errors.New("invalid issuer")
	}
	if claims.Audience != m.audience {
		return nil, errors.New("invalid audience")
	}
	if now < claims.NotBefore {
		return nil, errors.New("token not yet valid")
	}
	if now > claims.ExpiresAt {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func encodeSegment(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func decodeSegment(segment string, dst interface{}) error {
	data, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func signSegments(secret []byte, header, payload string) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(header))
	h.Write([]byte("."))
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
