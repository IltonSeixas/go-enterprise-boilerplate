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

func TestUpdateProfile_Success(t *testing.T) {
	user := activeUser(t, "profile@example.com")

	repo := testutil.NewStubUserRepo()
	require.NoError(t, repo.Save(context.Background(), user))

	uc := usecase.NewUpdateProfile(repo)
	out, err := uc.Execute(context.Background(), user.ID().UUID(), dto.UpdateProfileInput{
		Name: "Updated Name",
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", out.Name)
	assert.Equal(t, user.ID().UUID(), out.ID)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewUpdateProfile(repo)
	_, err := uc.Execute(context.Background(), uuid.New(), dto.UpdateProfileInput{
		Name: "Any Name",
	})

	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}
