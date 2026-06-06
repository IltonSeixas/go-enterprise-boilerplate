package security

import (
	"context"
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
	secret          []byte
	accessTTL       time.Duration
	refreshTTL      time.Duration
	redis           *redis.Client
}

func NewJWTService(secret string, accessTTL, refreshTTL time.Duration, redis *redis.Client) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		redis:      redis,
	}
}

func (s *JWTService) GeneratePair(userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
		Role: role,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	refreshToken := uuid.New().String()

	if err = s.redis.Set(
		context.Background(),
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
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
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

func (s *JWTService) RotateRefreshToken(oldToken string, userID uuid.UUID, role entity.Role) (port.TokenPair, error) {
	ctx := context.Background()
	stored, err := s.redis.Get(ctx, refreshKey(oldToken)).Result()
	if err != nil || stored != userID.String() {
		return port.TokenPair{}, apperror.ErrTokenInvalid
	}

	if err = s.redis.Del(ctx, refreshKey(oldToken)).Err(); err != nil {
		return port.TokenPair{}, apperror.ErrInternal
	}

	return s.GeneratePair(userID, role)
}

func (s *JWTService) RevokeRefreshToken(token string) error {
	return s.redis.Del(context.Background(), refreshKey(token)).Err()
}

var _ port.TokenService = (*JWTService)(nil)

func refreshKey(token string) string {
	return "refresh:" + token
}
