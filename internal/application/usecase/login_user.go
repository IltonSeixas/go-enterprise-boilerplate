package usecase

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type LoginUser struct {
	users  repository.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenService
}

func NewLoginUser(
	users repository.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenService,
) *LoginUser {
	return &LoginUser{users: users, hasher: hasher, tokens: tokens}
}

func (uc *LoginUser) Execute(ctx context.Context, in dto.LoginInput) (dto.AuthOutput, error) {
	email, err := valueobject.NewEmail(in.Email)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return dto.AuthOutput{}, apperror.ErrAccountInactive
	}

	ok, err := uc.hasher.Verify(in.Password, user.PasswordHash())
	if err != nil || !ok {
		return dto.AuthOutput{}, apperror.ErrInvalidCredentials
	}

	pair, err := uc.tokens.GeneratePair(user.ID().UUID(), user.Role())
	if err != nil {
		return dto.AuthOutput{}, err
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
