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

type ChangePassword struct {
	users  repository.UserRepository
	hasher port.PasswordHasher
	audit  port.AuditPort
}

func NewChangePassword(users repository.UserRepository, hasher port.PasswordHasher, audit port.AuditPort) *ChangePassword {
	return &ChangePassword{users: users, hasher: hasher, audit: audit}
}

func (uc *ChangePassword) Execute(ctx context.Context, id uuid.UUID, in dto.ChangePasswordInput) error {
	if err := valueobject.ValidatePassword(in.NewPassword); err != nil {
		return err
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
	if err := uc.users.Save(ctx, user); err != nil {
		return err
	}

	uc.audit.Record(ctx, entity.NewAuditEvent(
		entity.AuditEventPasswordChanged,
		uuid.NullUUID{UUID: id, Valid: true},
		uuid.NullUUID{UUID: id, Valid: true},
		"password changed",
	))

	return nil
}
