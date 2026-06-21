package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type UserRepository struct {
	mu    sync.RWMutex
	store map[uuid.UUID]*entity.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{store: make(map[uuid.UUID]*entity.User)}
}

func (r *UserRepository) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.store[id]
	if !ok {
		return nil, apperror.ErrUserNotFound
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(_ context.Context, email valueobject.Email) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.store {
		if u.Email().String() == email.String() {
			return u, nil
		}
	}
	return nil, apperror.ErrUserNotFound
}

func (r *UserRepository) Save(_ context.Context, u *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[u.ID().UUID()] = u
	return nil
}

func (r *UserRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

func (r *UserRepository) Count(_ context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.store)), nil
}

// SaveFirstOwner acquires a write lock before checking, preventing TOCTOU races.
func (r *UserRepository) SaveFirstOwner(_ context.Context, u *entity.User) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.store {
		if existing.Role() == entity.RoleOwner {
			return false, nil
		}
	}
	r.store[u.ID().UUID()] = u
	return true, nil
}

func (r *UserRepository) FindPaginated(_ context.Context, offset, limit int64) ([]*entity.User, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*entity.User, 0, len(r.store))
	for _, u := range r.store {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].CreatedAt().Equal(users[j].CreatedAt()) {
			return users[i].ID().UUID().String() < users[j].ID().UUID().String()
		}
		return users[i].CreatedAt().Before(users[j].CreatedAt())
	})

	total := int64(len(users))
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	page := make([]*entity.User, end-start)
	copy(page, users[start:end])
	return page, total, nil
}

var _ repository.UserRepository = (*UserRepository)(nil)
