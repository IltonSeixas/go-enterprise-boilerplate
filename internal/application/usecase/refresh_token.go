package usecase

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type RefreshToken struct {
	users  repository.UserRepository
	tokens port.TokenService
}

func NewRefreshToken(users repository.UserRepository, tokens port.TokenService) *RefreshToken {
	return &RefreshToken{users: users, tokens: tokens}
}

func (uc *RefreshToken) Execute(ctx context.Context, in dto.RefreshInput) (dto.AuthOutput, error) {
	userID, found, err := uc.tokens.FindUserIDByRefreshToken(in.RefreshToken)
	if err != nil || !found {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrUserNotFound
	}

	if !user.IsActive() {
		_ = uc.tokens.RevokeRefreshToken(in.RefreshToken)
		return dto.AuthOutput{}, apperror.ErrAccountInactive
	}

	pair, err := uc.tokens.RotateRefreshToken(in.RefreshToken, userID, user.Role())
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	return dto.AuthOutput{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User: dto.UserSummary{
			ID:    user.ID().UUID(),
			Email: user.Email().String(),
			Name:  user.Name(),
			Role:  user.Role(),
		},
	}, nil
}
