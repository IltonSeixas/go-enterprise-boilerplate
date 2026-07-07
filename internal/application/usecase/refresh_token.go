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
	redemption, err := uc.tokens.RedeemRefreshToken(ctx, in.RefreshToken)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	switch redemption.Outcome {
	case port.RedemptionReused:
		_ = uc.tokens.RevokeAllRefreshTokens(ctx, redemption.UserID)
		uc.audit.Record(ctx, entity.NewAuditEvent(
			entity.AuditEventRefreshTokenReuseDetected,
			uuid.NullUUID{UUID: redemption.UserID, Valid: true},
			uuid.NullUUID{},
			"refresh token reuse detected; all sessions revoked",
		))
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	case port.RedemptionInvalid:
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	userID := redemption.UserID

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrUserNotFound
	}

	if !user.IsActive() {
		// Redemption already revoked the token atomically above — no separate
		// revoke call is needed here.
		return dto.AuthOutput{}, apperror.ErrAccountInactive
	}

	pair, err := uc.tokens.GeneratePair(ctx, userID, user.Role())
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
