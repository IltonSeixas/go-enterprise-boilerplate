package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type UpdateProfile struct {
	users repository.UserRepository
}

func NewUpdateProfile(users repository.UserRepository) *UpdateProfile {
	return &UpdateProfile{users: users}
}

func (uc *UpdateProfile) Execute(ctx context.Context, id uuid.UUID, in dto.UpdateProfileInput) (dto.UserOutput, error) {
	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return dto.UserOutput{}, err
	}

	if err = user.UpdateName(in.Name); err != nil {
		return dto.UserOutput{}, err
	}

	if err = uc.users.Save(ctx, user); err != nil {
		return dto.UserOutput{}, err
	}

	return dto.UserOutput{
		ID:        user.ID().UUID(),
		Email:     user.Email().String(),
		Name:      user.Name(),
		Role:      user.Role(),
		IsActive:  user.IsActive(),
		CreatedAt: user.CreatedAt(),
		UpdatedAt: user.UpdatedAt(),
	}, nil
}
