//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/postgres"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("boilerplate"),
		tcpostgres.WithUsername("boilerplate"),
		tcpostgres.WithPassword("boilerplate"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, postgres.Migrate(ctx, pool))

	return pool
}

func newTestUser(t *testing.T, email string) *entity.User {
	t.Helper()

	emailVO, err := valueobject.NewEmail(email)
	require.NoError(t, err)

	hash := valueobject.NewPasswordHashFromPHC(
		"$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$c29tZWhhc2g",
	)

	user, err := entity.NewUser(emailVO, hash, "Test User", entity.RoleUser)
	require.NoError(t, err)
	return user
}

func TestUserRepository_SaveAndFindByID_RoundTrips(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	user := newTestUser(t, "round-trip@example.com")
	require.NoError(t, repo.Save(ctx, user))

	found, err := repo.FindByID(ctx, user.ID().UUID())
	require.NoError(t, err)
	require.Equal(t, user.Email().String(), found.Email().String())
	require.Equal(t, user.Name(), found.Name())
	require.Equal(t, user.Role(), found.Role())
}

func TestUserRepository_FindByID_NotFound_ReturnsDomainError(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	_, err := repo.FindByID(ctx, valueobject.NewUserID().UUID())
	require.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestUserRepository_FindByEmail_ReturnsSavedUser(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	user := newTestUser(t, "by-email@example.com")
	require.NoError(t, repo.Save(ctx, user))

	emailVO, err := valueobject.NewEmail("by-email@example.com")
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, emailVO)
	require.NoError(t, err)
	require.Equal(t, user.ID().UUID(), found.ID().UUID())
}

func TestUserRepository_Delete_RemovesUser(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	user := newTestUser(t, "to-delete@example.com")
	require.NoError(t, repo.Save(ctx, user))
	require.NoError(t, repo.Delete(ctx, user.ID().UUID()))

	_, err := repo.FindByID(ctx, user.ID().UUID())
	require.ErrorIs(t, err, apperror.ErrUserNotFound)
}

func TestUserRepository_SaveFirstOwner_OnlySucceedsOnce(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	first := newTestUser(t, "owner-one@example.com")
	created, err := repo.SaveFirstOwner(ctx, first)
	require.NoError(t, err)
	require.True(t, created, "first owner must be created")

	second := newTestUser(t, "owner-two@example.com")
	created, err = repo.SaveFirstOwner(ctx, second)
	require.NoError(t, err)
	require.False(t, created, "a second owner must not be created")
}

func TestUserRepository_FindPaginated_RespectsLimitAndReturnsTotal(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	for i := 0; i < 3; i++ {
		user := newTestUser(t, "page-"+string(rune('a'+i))+"@example.com")
		require.NoError(t, repo.Save(ctx, user))
	}

	users, total, err := repo.FindPaginated(ctx, 0, 2)
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, int64(3), total)
}

func TestUserRepository_Count_ReflectsSavedUsers(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := postgres.NewUserRepository(pool)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	require.NoError(t, repo.Save(ctx, newTestUser(t, "counted@example.com")))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}
