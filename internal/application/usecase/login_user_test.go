package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func activeUser(t *testing.T, email string) *entity.User {
	t.Helper()
	e, err := valueobject.NewEmail(email)
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(e, hash, "Test User", entity.RoleUser)
	require.NoError(t, err)
	return u
}

func inactiveUser(t *testing.T, email string) *entity.User {
	t.Helper()
	u := activeUser(t, email)
	u.Deactivate()
	return u
}

func TestLoginUser_Success(t *testing.T) {
	user := activeUser(t, "user@example.com")

	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(user, nil)

	uc := usecase.NewLoginUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	out, err := uc.Execute(context.Background(), dto.LoginInput{
		Email: "user@example.com", Password: "validpassword123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
}

func TestLoginUser_EmailNotFound(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(nil, apperror.ErrUserNotFound)

	uc := usecase.NewLoginUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	_, err := uc.Execute(context.Background(), dto.LoginInput{
		Email: "ghost@example.com", Password: "validpassword123",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

func TestLoginUser_WrongPassword(t *testing.T) {
	user := activeUser(t, "user@example.com")

	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(user, nil)

	uc := usecase.NewLoginUser(repo, testutil.NewStubHasherRejectAll(), testutil.NewStubTokenService())
	_, err := uc.Execute(context.Background(), dto.LoginInput{
		Email: "user@example.com", Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

func TestLoginUser_InactiveAccount(t *testing.T) {
	user := inactiveUser(t, "inactive@example.com")

	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(user, nil)

	uc := usecase.NewLoginUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	_, err := uc.Execute(context.Background(), dto.LoginInput{
		Email: "inactive@example.com", Password: "validpassword123",
	})

	assert.ErrorIs(t, err, apperror.ErrAccountInactive)
}
