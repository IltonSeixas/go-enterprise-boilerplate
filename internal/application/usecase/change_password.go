package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type ChangePassword struct {
	users  repository.UserRepository
	hasher port.PasswordHasher
}

func NewChangePassword(users repository.UserRepository, hasher port.PasswordHasher) *ChangePassword {
	return &ChangePassword{users: users, hasher: hasher}
}

func (uc *ChangePassword) Execute(ctx context.Context, id uuid.UUID, in dto.ChangePasswordInput) error {
	if len(in.NewPassword) < 12 || len(in.NewPassword) > 128 {
		return apperror.ErrInvalidPassword
	}

	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return err
	}

	ok, err := uc.hasher.Verify(in.CurrentPassword, user.PasswordHash())
	if err != nil || !ok {
		return apperror.ErrInvalidCredentials
	}

	newHash, err := uc.hasher.Hash(in.NewPassword)
	if err != nil {
		return err
	}

	user.UpdatePassword(newHash)
	return uc.users.Save(ctx, user)
}
