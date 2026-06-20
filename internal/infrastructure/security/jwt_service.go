package security

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Role entity.Role `json:"role"`
}

type JWTService struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
	redis      *redis.Client
}

// privateKeyPEM and publicKeyPEM must be PKCS#8 PEM-encoded Ed25519 keys,
// e.g. generated via `openssl genpkey -algorithm ed25519`.
func NewJWTService(privateKeyPEM, publicKeyPEM []byte, accessTTL, refreshTTL time.Duration, redis *redis.Client) (*JWTService, error) {
	privateKey, err := jwt.ParseEdPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT private key: %w", err)
	}

	publicKey, err := jwt.ParseEdPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}

	return &JWTService{
		privateKey: privateKey.(ed25519.PrivateKey),
		publicKey:  publicKey.(ed25519.PublicKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		redis:      redis,
	}, nil
}

func (s *JWTService) GeneratePair(ctx context.Context, userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
		Role: role,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(s.privateKey)
	if err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	refreshToken := uuid.New().String()

	if err = s.redis.Set(
		ctx,
		refreshKey(refreshToken),
		userID.String(),
		s.refreshTTL,
	).Err(); err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	return port.TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *JWTService) ValidateAccessToken(token string) (port.AccessTokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.publicKey, nil
	})
	if err != nil || !parsed.Valid {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	claims, ok := parsed.Claims.(*jwtClaims)
	if !ok {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return port.AccessTokenClaims{}, apperror.ErrTokenInvalid
	}

	return port.AccessTokenClaims{UserID: userID, Role: claims.Role}, nil
}

func (s *JWTService) FindUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, bool, error) {
	stored, err := s.redis.Get(ctx, refreshKey(token)).Result()
	if err == redis.Nil {
		return uuid.UUID{}, false, nil
	}
	if err != nil {
		return uuid.UUID{}, false, apperror.ErrInternal
	}

	userID, err := uuid.Parse(stored)
	if err != nil {
		return uuid.UUID{}, false, apperror.ErrTokenInvalid
	}

	return userID, true, nil
}

func (s *JWTService) RotateRefreshToken(ctx context.Context, oldToken string, userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	stored, err := s.redis.Get(ctx, refreshKey(oldToken)).Result()
	if err != nil || stored != userID.String() {
		return port.TokenPair{}, apperror.ErrTokenInvalid
	}

	if err = s.redis.Del(ctx, refreshKey(oldToken)).Err(); err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	return s.GeneratePair(ctx, userID, role)
}

func (s *JWTService) RevokeRefreshToken(ctx context.Context, token string) error {
	return s.redis.Del(ctx, refreshKey(token)).Err()
}

var _ port.TokenService = (*JWTService)(nil)

func refreshKey(token string) string {
	return "refresh:" + token
}
