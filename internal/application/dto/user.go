package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
)

type UpdateProfileInput struct {
	Name string
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type ChangeRoleInput struct {
	Role entity.Role
}

type UserOutput struct {
	ID        uuid.UUID
	Email     string
	Name      string
	Role      entity.Role
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListUsersInput struct {
	Page     int32
	PageSize int32
}

type PaginationOutput struct {
	Page       int32
	PageSize   int32
	TotalItems int64
	TotalPages int32
}

type ListUsersOutput struct {
	Items      []UserOutput
	Pagination PaginationOutput
}
