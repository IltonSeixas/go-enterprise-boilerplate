package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrationLockKey identifies the Postgres advisory lock used to serialize
// migrations across concurrently starting replicas. Arbitrary but fixed so
// every replica targets the same lock.
const migrationLockKey = 72147483647

// Migrate applies all embedded SQL migrations in lexical filename order.
// It holds a session-level Postgres advisory lock for the duration of the
// run so that multiple replicas starting concurrently apply migrations
// one at a time instead of racing on the same DDL statements.
func Migrate(ctx context.Context, pool *pgxpool.Pool) (err error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for migration lock: %w", err)
	}
	defer conn.Release()

	if _, lockErr := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); lockErr != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", lockErr)
	}
	defer func() {
		if _, unlockErr := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockKey); unlockErr != nil {
			err = errors.Join(err, fmt.Errorf("release migration advisory lock: %w", unlockErr))
		}
	}()

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		sql, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := conn.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}
