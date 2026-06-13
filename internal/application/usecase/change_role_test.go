package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func userWithRole(t *testing.T, email string, role entity.Role) *entity.User {
	t.Helper()
	e, err := valueobject.NewEmail(email)
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(e, hash, "Test User", role)
	require.NoError(t, err)
	return u
}

func TestChangeUserRole_RejectsWhenActorNotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	target := userWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), target))

	uc := usecase.NewChangeUserRole(repo)
	_, err := uc.Execute(context.Background(), uuid.New(), target.ID().UUID(), dto.ChangeRoleInput{Role: entity.RoleAdmin})

	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestChangeUserRole_RejectsWhenTargetNotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	actor := userWithRole(t, "actor@example.com", entity.RoleOwner)
	require.NoError(t, repo.Save(context.Background(), actor))

	uc := usecase.NewChangeUserRole(repo)
	_, err := uc.Execute(context.Background(), actor.ID().UUID(), uuid.New(), dto.ChangeRoleInput{Role: entity.RoleAdmin})

	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestChangeUserRole_RejectsWhenActorLacksPermission(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	actor := userWithRole(t, "actor@example.com", entity.RoleUser)
	target := userWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), actor))
	require.NoError(t, repo.Save(context.Background(), target))

	uc := usecase.NewChangeUserRole(repo)
	_, err := uc.Execute(context.Background(), actor.ID().UUID(), target.ID().UUID(), dto.ChangeRoleInput{Role: entity.RoleAdmin})

	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}

func TestChangeUserRole_RejectsWhenActorChangesOwnRole(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := userWithRole(t, "owner@example.com", entity.RoleOwner)
	require.NoError(t, repo.Save(context.Background(), owner))

	uc := usecase.NewChangeUserRole(repo)
	_, err := uc.Execute(context.Background(), owner.ID().UUID(), owner.ID().UUID(), dto.ChangeRoleInput{Role: entity.RoleAdmin})

	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}

func TestChangeUserRole_OwnerPromotesUserToAdminAndPersists(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	actor := userWithRole(t, "owner@example.com", entity.RoleOwner)
	target := userWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), actor))
	require.NoError(t, repo.Save(context.Background(), target))

	uc := usecase.NewChangeUserRole(repo)
	out, err := uc.Execute(context.Background(), actor.ID().UUID(), target.ID().UUID(), dto.ChangeRoleInput{Role: entity.RoleAdmin})
	require.NoError(t, err)
	assert.Equal(t, entity.RoleAdmin, out.Role)

	persisted, err := repo.FindByID(context.Background(), target.ID().UUID())
	require.NoError(t, err)
	assert.Equal(t, entity.RoleAdmin, persisted.Role())
}
