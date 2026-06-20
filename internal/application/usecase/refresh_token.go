package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type RefreshToken struct {
	users  repository.UserRepository
	tokens port.TokenService
	audit  port.AuditPort
}

func NewRefreshToken(users repository.UserRepository, tokens port.TokenService, audit port.AuditPort) *RefreshToken {
	return &RefreshToken{users: users, tokens: tokens, audit: audit}
}

func (uc *RefreshToken) Execute(ctx context.Context, in dto.RefreshInput) (dto.AuthOutput, error) {
	userID, found, err := uc.tokens.FindUserIDByRefreshToken(ctx, in.RefreshToken)
	if err != nil || !found {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrUserNotFound
	}

	if !user.IsActive() {
		// Best-effort revoke: the account is already being denied via ErrAccountInactive below,
		// so a revoke failure here must not change the response or leak storage details to the caller.
		_ = uc.tokens.RevokeRefreshToken(ctx, in.RefreshToken)
		return dto.AuthOutput{}, apperror.ErrAccountInactive
	}

	pair, err := uc.tokens.RotateRefreshToken(ctx, in.RefreshToken, userID, user.Role())
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	uc.audit.Record(ctx, entity.NewAuditEvent(
		entity.AuditEventTokenRefreshed,
		uuid.NullUUID{UUID: userID, Valid: true},
		uuid.NullUUID{},
		"refresh token rotated",
	))

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
