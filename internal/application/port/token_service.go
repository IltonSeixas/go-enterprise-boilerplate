package port

import (
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
	GeneratePair(userID uuid.UUID, role entity.Role) (TokenPair, error)
	ValidateAccessToken(token string) (AccessTokenClaims, error)
	FindUserIDByRefreshToken(token string) (uuid.UUID, bool, error)
	RotateRefreshToken(oldToken string, userID uuid.UUID, role entity.Role) (TokenPair, error)
	RevokeRefreshToken(token string) error
}
