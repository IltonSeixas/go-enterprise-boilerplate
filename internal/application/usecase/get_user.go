package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type GetUser struct {
	users repository.UserRepository
}

func NewGetUser(users repository.UserRepository) *GetUser {
	return &GetUser{users: users}
}

func (uc *GetUser) Execute(ctx context.Context, callerID uuid.UUID, callerRole entity.Role, targetID uuid.UUID) (dto.UserOutput, error) {
	if callerID != targetID && !callerRole.CanManageRoles() {
		return dto.UserOutput{}, apperror.ErrInsufficientPerms
	}

	user, err := uc.users.FindByID(ctx, targetID)
	if err != nil {
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
