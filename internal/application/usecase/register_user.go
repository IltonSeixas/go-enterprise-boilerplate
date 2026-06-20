package usecase

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type RegisterUser struct {
	users  repository.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenService
}

func NewRegisterUser(
	users repository.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenService,
) *RegisterUser {
	return &RegisterUser{users: users, hasher: hasher, tokens: tokens}
}

func (uc *RegisterUser) Execute(ctx context.Context, in dto.RegisterInput) (dto.AuthOutput, error) {
	if err := valueobject.ValidatePassword(in.Password); err != nil {
		return dto.AuthOutput{}, err
	}

	email, err := valueobject.NewEmail(in.Email)
	if err != nil {
		return dto.AuthOutput{}, err
	}

	existing, err := uc.users.FindByEmail(ctx, email)
	if err != nil && err != apperror.ErrUserNotFound {
		return dto.AuthOutput{}, err
	}
	if existing != nil {
		return dto.AuthOutput{}, apperror.ErrEmailAlreadyExists
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return dto.AuthOutput{}, err
	}

	// Try to claim Owner atomically. If another request already did, register as User.
	candidate, err := entity.NewUser(email, hash, in.Name, entity.RoleOwner)
	if err != nil {
		return dto.AuthOutput{}, err
	}

	claimed, err := uc.users.SaveFirstOwner(ctx, candidate)
	if err != nil {
		return dto.AuthOutput{}, err
	}

	user := candidate
	if !claimed {
		user, err = entity.NewUser(email, hash, in.Name, entity.RoleUser)
		if err != nil {
			return dto.AuthOutput{}, err
		}
		if err = uc.users.Save(ctx, user); err != nil {
			return dto.AuthOutput{}, err
		}
	}

	pair, err := uc.tokens.GeneratePair(ctx, user.ID().UUID(), user.Role())
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
