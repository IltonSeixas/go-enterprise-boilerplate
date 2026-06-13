package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func TestRefreshToken_RejectsUnknownRefreshToken(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	tokens := testutil.NewStubTokenServiceWithClaims("access-stub", port.AccessTokenClaims{})

	uc := usecase.NewRefreshToken(repo, tokens)
	_, err := uc.Execute(context.Background(), dto.RefreshInput{RefreshToken: "unknown-token"})

	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

func TestRefreshToken_RejectsWhenUserNoLongerExists(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	tokens := testutil.NewStubTokenServiceWithClaims("access-stub", port.AccessTokenClaims{
		UserID: uuid.New(),
		Role:   entity.RoleUser,
	})

	uc := usecase.NewRefreshToken(repo, tokens)
	_, err := uc.Execute(context.Background(), dto.RefreshInput{RefreshToken: tokens.ValidRefreshToken})

	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestRefreshToken_RevokesTokenAndRejectsInactiveAccount(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := activeUser(t, "refresh@example.com")
	user.Deactivate()
	require.NoError(t, repo.Save(context.Background(), user))

	tokens := testutil.NewStubTokenServiceWithClaims("access-stub", port.AccessTokenClaims{
		UserID: user.ID().UUID(),
		Role:   user.Role(),
	})

	uc := usecase.NewRefreshToken(repo, tokens)
	_, err := uc.Execute(context.Background(), dto.RefreshInput{RefreshToken: tokens.ValidRefreshToken})

	assert.ErrorIs(t, err, apperror.ErrAccountInactive)
}

func TestRefreshToken_RotatesTokenPairOnSuccess(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := activeUser(t, "refresh@example.com")
	require.NoError(t, repo.Save(context.Background(), user))

	tokens := testutil.NewStubTokenServiceWithClaims("access-stub", port.AccessTokenClaims{
		UserID: user.ID().UUID(),
		Role:   user.Role(),
	})

	uc := usecase.NewRefreshToken(repo, tokens)
	out, err := uc.Execute(context.Background(), dto.RefreshInput{RefreshToken: tokens.ValidRefreshToken})

	require.NoError(t, err)
	assert.Equal(t, "access-stub", out.AccessToken)
	assert.Equal(t, "refresh-new", out.RefreshToken)
}
