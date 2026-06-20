package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessTokenClaims struct {
	UserID uuid.UUID
	Role   entity.Role
}

type TokenService interface {
	GeneratePair(ctx context.Context, userID uuid.UUID, role entity.Role) (TokenPair, error)
	ValidateAccessToken(token string) (AccessTokenClaims, error)
	FindUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, bool, error)
	RotateRefreshToken(ctx context.Context, oldToken string, userID uuid.UUID, role entity.Role) (TokenPair, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}
