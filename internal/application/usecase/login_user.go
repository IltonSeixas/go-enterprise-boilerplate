package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type LoginUser struct {
	users  repository.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenService
	audit  port.AuditPort
}

func NewLoginUser(
	users repository.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenService,
	audit port.AuditPort,
) *LoginUser {
	return &LoginUser{users: users, hasher: hasher, tokens: tokens, audit: audit}
}

func (uc *LoginUser) Execute(ctx context.Context, in dto.LoginInput) (dto.AuthOutput, error) {
	email, err := valueobject.NewEmail(in.Email)
	if err != nil {
		uc.audit.Record(ctx, entity.NewAuditEvent(
			entity.AuditEventLoginFailed, uuid.NullUUID{}, uuid.NullUUID{}, "malformed email",
		))
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		uc.audit.Record(ctx, entity.NewAuditEvent(
			entity.AuditEventLoginFailed, uuid.NullUUID{}, uuid.NullUUID{}, "no account for email",
		))
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	if !user.IsActive() {
		uc.audit.Record(ctx, entity.NewAuditEvent(
			entity.AuditEventLoginFailed,
			uuid.NullUUID{UUID: user.ID().UUID(), Valid: true},
			uuid.NullUUID{},
			"account inactive",
		))
		return dto.AuthOutput{}, apperror.ErrAccountInactive
	}

	ok, err := uc.hasher.Verify(in.Password, user.PasswordHash())
	if err != nil || !ok {
		uc.audit.Record(ctx, entity.NewAuditEvent(
			entity.AuditEventLoginFailed,
			uuid.NullUUID{UUID: user.ID().UUID(), Valid: true},
			uuid.NullUUID{},
			"invalid password",
		))
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	pair, err := uc.tokens.GeneratePair(ctx, user.ID().UUID(), user.Role())
	if err != nil {
		return dto.AuthOutput{}, err
	}

	uc.audit.Record(ctx, entity.NewAuditEvent(
		entity.AuditEventLoginSucceeded,
		uuid.NullUUID{UUID: user.ID().UUID(), Valid: true},
		uuid.NullUUID{},
		"login succeeded",
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
