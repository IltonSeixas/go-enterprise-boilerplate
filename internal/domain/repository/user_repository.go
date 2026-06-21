package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error)
	Save(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	// SaveFirstOwner atomically saves user only if no owner exists yet.
	// Returns (true, nil) on success, (false, nil) if an owner already exists.
	SaveFirstOwner(ctx context.Context, user *entity.User) (bool, error)
	// FindPaginated returns a page of users ordered by created_at, id together
	// with the total number of users in the full collection.
	FindPaginated(ctx context.Context, offset, limit int64) ([]*entity.User, int64, error)
}
