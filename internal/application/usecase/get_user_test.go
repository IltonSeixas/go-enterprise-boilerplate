package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func TestGetUser_OwnerCanFetchAny(t *testing.T) {
	user := activeUser(t, "target@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), user))

	ownerID := uuid.New()
	uc := usecase.NewGetUser(repo)

	out, err := uc.Execute(context.Background(), ownerID, entity.RoleOwner, user.ID().UUID())
	require.NoError(t, err)
	assert.Equal(t, user.ID().UUID(), out.ID)
}

func TestGetUser_UserCanFetchSelf(t *testing.T) {
	user := activeUser(t, "self@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), user))

	uc := usecase.NewGetUser(repo)
	out, err := uc.Execute(context.Background(), user.ID().UUID(), entity.RoleUser, user.ID().UUID())
	require.NoError(t, err)
	assert.Equal(t, user.ID().UUID(), out.ID)
}

func TestGetUser_UserCannotFetchOther(t *testing.T) {
	target := activeUser(t, "other@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), target))

	callerID := uuid.New()
	uc := usecase.NewGetUser(repo)

	_, err := uc.Execute(context.Background(), callerID, entity.RoleUser, target.ID().UUID())
	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}

func TestGetUser_NotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	uc := usecase.NewGetUser(repo)

	nonexistent := uuid.New()
	_, err := uc.Execute(context.Background(), nonexistent, entity.RoleOwner, nonexistent)
	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}
