package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func seedUsers(t *testing.T, repo *testutil.StubUserRepo, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		u := activeUser(t, fmt.Sprintf("user%d@example.com", i))
		require.NoError(t, repo.Save(context.Background(), u))
	}
}

func TestListUsers_ReturnsItemsAndPaginationMetadata(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	seedUsers(t, repo, 5)

	uc := usecase.NewListUsers(repo)
	out, err := uc.Execute(context.Background(), entity.RoleOwner, dto.ListUsersInput{Page: 2, PageSize: 2})
	require.NoError(t, err)

	assert.Len(t, out.Items, 2)
	assert.Equal(t, int32(2), out.Pagination.Page)
	assert.Equal(t, int32(2), out.Pagination.PageSize)
	assert.Equal(t, int64(5), out.Pagination.TotalItems)
	assert.Equal(t, int32(3), out.Pagination.TotalPages)
}

func TestListUsers_DefaultsPageAndPageSizeWhenAbsent(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewListUsers(repo)
	out, err := uc.Execute(context.Background(), entity.RoleOwner, dto.ListUsersInput{})
	require.NoError(t, err)

	assert.Equal(t, int32(1), out.Pagination.Page)
	assert.Equal(t, int32(20), out.Pagination.PageSize)
	assert.Equal(t, int32(0), out.Pagination.TotalPages)
}

func TestListUsers_ClampsPageSizeToMaximum(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewListUsers(repo)
	out, err := uc.Execute(context.Background(), entity.RoleOwner, dto.ListUsersInput{Page: 1, PageSize: 1000})
	require.NoError(t, err)

	assert.Equal(t, int32(100), out.Pagination.PageSize)
}

func TestListUsers_ClampsPageToMinimumOfOne(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewListUsers(repo)
	out, err := uc.Execute(context.Background(), entity.RoleOwner, dto.ListUsersInput{Page: 0})
	require.NoError(t, err)

	assert.Equal(t, int32(1), out.Pagination.Page)
}

func TestListUsers_RejectsCallerWithoutManageRolesPermission(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	uc := usecase.NewListUsers(repo)
	_, err := uc.Execute(context.Background(), entity.RoleUser, dto.ListUsersInput{})
	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}
