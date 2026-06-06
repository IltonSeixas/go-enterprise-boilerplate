package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/memory"
)

func makeUser(t *testing.T, emailStr string) *entity.User {
	t.Helper()
	email, _ := valueobject.NewEmail(emailStr)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(email, hash, "Test", entity.RoleUser)
	require.NoError(t, err)
	return u
}

func TestSaveAndFindByID(t *testing.T) {
	repo := memory.NewUserRepository()
	u := makeUser(t, "a@b.com")
	require.NoError(t, repo.Save(context.Background(), u))
	found, err := repo.FindByID(context.Background(), u.ID().UUID())
	require.NoError(t, err)
	assert.Equal(t, u.ID().UUID(), found.ID().UUID())
}

func TestFindByEmail(t *testing.T) {
	repo := memory.NewUserRepository()
	u := makeUser(t, "find@example.com")
	require.NoError(t, repo.Save(context.Background(), u))
	email, _ := valueobject.NewEmail("find@example.com")
	found, err := repo.FindByEmail(context.Background(), email)
	require.NoError(t, err)
	assert.Equal(t, "find@example.com", found.Email().String())
}

func TestFindByID_NotFound(t *testing.T) {
	repo := memory.NewUserRepository()
	_, err := repo.FindByID(context.Background(), makeUser(t, "x@y.com").ID().UUID())
	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestCount(t *testing.T) {
	repo := memory.NewUserRepository()
	n, _ := repo.Count(context.Background())
	assert.Equal(t, int64(0), n)
	require.NoError(t, repo.Save(context.Background(), makeUser(t, "a@b.com")))
	require.NoError(t, repo.Save(context.Background(), makeUser(t, "c@d.com")))
	n, _ = repo.Count(context.Background())
	assert.Equal(t, int64(2), n)
}

func makeOwner(t *testing.T, emailStr string) *entity.User {
	t.Helper()
	email, _ := valueobject.NewEmail(emailStr)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(email, hash, "Owner", entity.RoleOwner)
	require.NoError(t, err)
	return u
}

func TestSaveFirstOwner_OnlyFirst(t *testing.T) {
	repo := memory.NewUserRepository()
	u1 := makeOwner(t, "owner@b.com")
	u2 := makeOwner(t, "second@b.com")

	claimed, err := repo.SaveFirstOwner(context.Background(), u1)
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = repo.SaveFirstOwner(context.Background(), u2)
	require.NoError(t, err)
	assert.False(t, claimed)
}

func TestDelete(t *testing.T) {
	repo := memory.NewUserRepository()
	u := makeUser(t, "del@b.com")
	require.NoError(t, repo.Save(context.Background(), u))
	require.NoError(t, repo.Delete(context.Background(), u.ID().UUID()))
	_, err := repo.FindByID(context.Background(), u.ID().UUID())
	assert.ErrorIs(t, err, apperror.ErrUserNotFound)
}
