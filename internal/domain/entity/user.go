package entity

import (
	"strings"
	"time"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type Role string

const (
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) CanManageRoles() bool {
	return r == RoleOwner || r == RoleAdmin
}

// ParseRole validates and converts a raw string into a known Role.
func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleUser:
		return Role(s), nil
	default:
		return "", apperror.ErrInvalidRole
	}
}

type User struct {
	id           valueobject.UserID
	email        valueobject.Email
	passwordHash valueobject.PasswordHash
	name         string
	role         Role
	isActive     bool
	createdAt    time.Time
	updatedAt    time.Time
}

func NewUser(
	email valueobject.Email,
	hash valueobject.PasswordHash,
	name string,
	role Role,
) (*User, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &User{
		id:           valueobject.NewUserID(),
		email:        email,
		passwordHash: hash,
		name:         strings.TrimSpace(name),
		role:         role,
		isActive:     true,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func Reconstitute(
	id valueobject.UserID,
	email valueobject.Email,
	hash valueobject.PasswordHash,
	name string,
	role Role,
	isActive bool,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		passwordHash: hash,
		name:         name,
		role:         role,
		isActive:     isActive,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (u *User) UpdateName(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	u.name = strings.TrimSpace(name)
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdatePassword(hash valueobject.PasswordHash) {
	u.passwordHash = hash
	u.updatedAt = time.Now().UTC()
}

func (u *User) ChangeRole(newRole Role, actor *User) error {
	if actor.role != RoleOwner || actor.id == u.id {
		return apperror.ErrInsufficientPerms
	}
	u.role = newRole
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now().UTC()
}

func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now().UTC()
}

func (u *User) ID() valueobject.UserID                 { return u.id }
func (u *User) Email() valueobject.Email               { return u.email }
func (u *User) PasswordHash() valueobject.PasswordHash { return u.passwordHash }
func (u *User) Name() string                           { return u.name }
func (u *User) Role() Role                             { return u.role }
func (u *User) IsActive() bool                         { return u.isActive }
func (u *User) CreatedAt() time.Time                   { return u.createdAt }
func (u *User) UpdatedAt() time.Time                   { return u.updatedAt }

func validateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) == 0 || len(trimmed) > 100 {
		return apperror.ErrInvalidName
	}
	return nil
}
