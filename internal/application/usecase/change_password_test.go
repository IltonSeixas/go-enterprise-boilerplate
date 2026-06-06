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
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func TestChangePassword_Success(t *testing.T) {
	user := activeUser(t, "chpass@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), user))

	uc := usecase.NewChangePassword(repo, testutil.NewStubHasher())
	err := uc.Execute(context.Background(), user.ID().UUID(), dto.ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123456",
	})

	assert.NoError(t, err)
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	user := activeUser(t, "chpass2@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), user))

	uc := usecase.NewChangePassword(repo, testutil.NewStubHasherRejectAll())
	err := uc.Execute(context.Background(), user.ID().UUID(), dto.ChangePasswordInput{
		CurrentPassword: "wrongcurrent",
		NewPassword:     "newpassword123456",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewChangePassword(repo, testutil.NewStubHasher())
	err := uc.Execute(context.Background(), uuid.New(), dto.ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123456",
	})

	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}
