package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

type ChangeUserRole struct {
	users repository.UserRepository
}

func NewChangeUserRole(users repository.UserRepository) *ChangeUserRole {
	return &ChangeUserRole{users: users}
}

func (uc *ChangeUserRole) Execute(ctx context.Context, actorID, targetID uuid.UUID, in dto.ChangeRoleInput) (dto.UserOutput, error) {
	actor, err := uc.users.FindByID(ctx, actorID)
	if err != nil {
		return dto.UserOutput{}, err
	}

	target, err := uc.users.FindByID(ctx, targetID)
	if err != nil {
		return dto.UserOutput{}, err
	}

	if err = target.ChangeRole(in.Role, actor); err != nil {
		return dto.UserOutput{}, err
	}

	if err = uc.users.Save(ctx, target); err != nil {
		return dto.UserOutput{}, err
	}

	return dto.UserOutput{
		ID:        target.ID().UUID(),
		Email:     target.Email().String(),
		Name:      target.Name(),
		Role:      target.Role(),
		IsActive:  target.IsActive(),
		CreatedAt: target.CreatedAt(),
		UpdatedAt: target.UpdatedAt(),
	}, nil
}
