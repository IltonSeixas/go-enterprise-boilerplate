package usecase_test

import (
	"context"
	"errors"
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

func TestRegisterUser_ShortPassword(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	hasher := testutil.NewStubHasher()
	tokens := testutil.NewStubTokenService()

	uc := usecase.NewRegisterUser(repo, hasher, tokens)
	_, err := uc.Execute(context.Background(), dto.RegisterInput{
		Email: "a@b.com", Password: "short", Name: "Test",
	})
	assert.ErrorIs(t, err, apperror.ErrInvalidPassword)
}

func TestRegisterUser_DuplicateEmail(t *testing.T) {
	email, _ := valueobject.NewEmail("a@b.com")
	hash := valueobject.NewPasswordHashFromPHC("x")
	existing, _ := entity.NewUser(email, hash, "Existing", entity.RoleUser)

	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(existing, nil)

	uc := usecase.NewRegisterUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	_, err := uc.Execute(context.Background(), dto.RegisterInput{
		Email: "a@b.com", Password: "validpassword123", Name: "Test",
	})
	assert.ErrorIs(t, err, apperror.ErrEmailAlreadyExists)
}

func TestRegisterUser_FirstUserBecomesOwner(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	repo.SetSaveFirstOwnerResult(true, nil)

	uc := usecase.NewRegisterUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	out, err := uc.Execute(context.Background(), dto.RegisterInput{
		Email: "owner@b.com", Password: "validpassword123", Name: "Owner",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.RoleOwner, out.User.Role)
}

func TestRegisterUser_SecondUserBecomesRegularUser(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	repo.SetSaveFirstOwnerResult(false, nil)

	uc := usecase.NewRegisterUser(repo, testutil.NewStubHasher(), testutil.NewStubTokenService())
	out, err := uc.Execute(context.Background(), dto.RegisterInput{
		Email: "user@b.com", Password: "validpassword123", Name: "User",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.RoleUser, out.User.Role)
}

func TestRegisterUser_InvalidEmail(t *testing.T) {
	uc := usecase.NewRegisterUser(
		testutil.NewStubUserRepo(),
		testutil.NewStubHasher(),
		testutil.NewStubTokenService(),
	)
	_, err := uc.Execute(context.Background(), dto.RegisterInput{
		Email: "notanemail", Password: "validpassword123", Name: "Test",
	})
	assert.True(t, errors.Is(err, apperror.ErrInvalidEmail))
}
