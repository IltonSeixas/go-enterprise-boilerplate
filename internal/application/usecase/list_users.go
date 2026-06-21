package usecase

import (
	"context"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type ListUsers struct {
	users repository.UserRepository
}

func NewListUsers(users repository.UserRepository) *ListUsers {
	return &ListUsers{users: users}
}

func (uc *ListUsers) Execute(ctx context.Context, callerRole entity.Role, in dto.ListUsersInput) (dto.ListUsersOutput, error) {
	if !callerRole.CanManageRoles() {
		return dto.ListUsersOutput{}, apperror.ErrInsufficientPerms
	}

	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	switch {
	case pageSize < 1:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}

	offset := int64(page-1) * int64(pageSize)

	users, total, err := uc.users.FindPaginated(ctx, offset, int64(pageSize))
	if err != nil {
		return dto.ListUsersOutput{}, err
	}

	var totalPages int32
	if total > 0 {
		totalPages = int32((total-1)/int64(pageSize) + 1)
	}

	items := make([]dto.UserOutput, len(users))
	for i, user := range users {
		items[i] = dto.UserOutput{
			ID:        user.ID().UUID(),
			Email:     user.Email().String(),
			Name:      user.Name(),
			Role:      user.Role(),
			IsActive:  user.IsActive(),
			CreatedAt: user.CreatedAt(),
			UpdatedAt: user.UpdatedAt(),
		}
	}

	return dto.ListUsersOutput{
		Items: items,
		Pagination: dto.PaginationOutput{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}
