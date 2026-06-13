package entity_test

import (
	"strings"
	"testing"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeUser(t *testing.T, role entity.Role) *entity.User {
	t.Helper()
	email, _ := valueobject.NewEmail("test@example.com")
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$v=19$m=65536,t=3,p=4$...")
	u, err := entity.NewUser(email, hash, "Test User", role)
	require.NoError(t, err)
	return u
}

func TestNewUser_ActiveByDefault(t *testing.T) {
	u := makeUser(t, entity.RoleUser)
	assert.True(t, u.IsActive())
}

func TestNewUser_EmptyName(t *testing.T) {
	email, _ := valueobject.NewEmail("a@b.com")
	hash := valueobject.NewPasswordHashFromPHC("x")
	_, err := entity.NewUser(email, hash, "   ", entity.RoleUser)
	assert.ErrorIs(t, err, apperror.ErrInvalidName)
}

func TestNewUser_NameTooLong(t *testing.T) {
	email, _ := valueobject.NewEmail("a@b.com")
	hash := valueobject.NewPasswordHashFromPHC("x")
	_, err := entity.NewUser(email, hash, strings.Repeat("a", 101), entity.RoleUser)
	assert.ErrorIs(t, err, apperror.ErrInvalidName)
}

func TestDeactivate(t *testing.T) {
	u := makeUser(t, entity.RoleUser)
	u.Deactivate()
	assert.False(t, u.IsActive())
}

func TestChangeRole_AdminCannotChangeRoles(t *testing.T) {
	target := makeUser(t, entity.RoleUser)
	actor := makeUser(t, entity.RoleAdmin)
	err := target.ChangeRole(entity.RoleAdmin, actor)
	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}

func TestChangeRole_OwnerCanChangeAnotherUsersRole(t *testing.T) {
	target := makeUser(t, entity.RoleUser)
	actor := makeUser(t, entity.RoleOwner)
	require.NoError(t, target.ChangeRole(entity.RoleAdmin, actor))
	assert.Equal(t, entity.RoleAdmin, target.Role())
}

func TestChangeRole_OwnerCannotChangeOwnRole(t *testing.T) {
	owner := makeUser(t, entity.RoleOwner)
	err := owner.ChangeRole(entity.RoleAdmin, owner)
	assert.ErrorIs(t, err, apperror.ErrInsufficientPerms)
}
