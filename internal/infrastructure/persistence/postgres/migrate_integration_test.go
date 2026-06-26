//go:build integration

package postgres_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/infrastructure/persistence/postgres"
)

const concurrentReplicas = 8

func TestMigrate_ConcurrentReplicas_ApplyMigrationsExactlyOnceWithoutError(t *testing.T) {
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

	// Each simulated replica gets its own pool, mirroring how independent
	// app instances would each open their own connection pool in production.
	pools := make([]*pgxpool.Pool, concurrentReplicas)
	for i := range pools {
		pool, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		t.Cleanup(pool.Close)
		pools[i] = pool
	}

	var wg sync.WaitGroup
	errs := make([]error, concurrentReplicas)
	for i, pool := range pools {
		wg.Add(1)
		go func(i int, pool *pgxpool.Pool) {
			defer wg.Done()
			errs[i] = postgres.Migrate(ctx, pool)
		}(i, pool)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "replica %d failed to migrate", i)
	}

	verifyPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer verifyPool.Close()

	var userTableCount int
	err = verifyPool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_name = 'users'`,
	).Scan(&userTableCount)
	require.NoError(t, err)
	require.Equal(t, 1, userTableCount, "users table must exist exactly once")

	var auditTableCount int
	err = verifyPool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_name = 'audit_log'`,
	).Scan(&auditTableCount)
	require.NoError(t, err)
	require.Equal(t, 1, auditTableCount, "audit_log table must exist exactly once")
}
