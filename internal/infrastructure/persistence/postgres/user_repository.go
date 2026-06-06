package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const selectFields = `id, email, password_hash, name, role, is_active, created_at, updated_at`

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+selectFields+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+selectFields+` FROM users WHERE email = $1`, email.String())
	return scanUser(row)
}

func (r *UserRepository) Save(ctx context.Context, u *entity.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			email        = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			name         = EXCLUDED.name,
			role         = EXCLUDED.role,
			is_active    = EXCLUDED.is_active,
			updated_at   = EXCLUDED.updated_at`,
		u.ID().UUID(), u.Email().String(), u.PasswordHash().PHC(),
		u.Name(), string(u.Role()), u.IsActive(),
		u.CreatedAt(), u.UpdatedAt(),
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// SaveFirstOwner uses the uq_users_owner_role partial unique index to enforce atomicity.
func (r *UserRepository) SaveFirstOwner(ctx context.Context, u *entity.User) (bool, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'owner', $5, $6, $7)`,
		u.ID().UUID(), u.Email().String(), u.PasswordHash().PHC(),
		u.Name(), u.IsActive(), u.CreatedAt(), u.UpdatedAt(),
	)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

var _ repository.UserRepository = (*UserRepository)(nil)

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*entity.User, error) {
	var (
		id        uuid.UUID
		email     string
		phc       string
		name      string
		role      string
		isActive  bool
		createdAt time.Time
		updatedAt time.Time
	)

	err := row.Scan(&id, &email, &phc, &name, &role, &isActive, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, err
	}

	vo, err := valueobject.NewEmail(email)
	if err != nil {
		return nil, err
	}

	return entity.Reconstitute(
		valueobject.UserIDFromUUID(id),
		vo,
		valueobject.NewPasswordHashFromPHC(phc),
		name,
		entity.Role(role),
		isActive,
		createdAt,
		updatedAt,
	), nil
}
