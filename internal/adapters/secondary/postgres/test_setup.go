package postgres

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	packagePool      *pgxpool.Pool
	packageContainer *postgres.PostgresContainer
	poolOnce         sync.Once
)

// SetupPackageContainer starts a single PostgreSQL 16-alpine container per
// package using sync.Once. Returns a pool connected to the containerized DB.
// If Docker is not available, the test is skipped via t.Skip.
func SetupPackageContainer(t testing.TB) *pgxpool.Pool {
	t.Helper()
	poolOnce.Do(func() {
		ctx := context.Background()

		ctr, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("hourglass_test"),
			postgres.WithUsername("hourglass"),
			postgres.WithPassword("hourglass"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			t.Skip("Docker not available, skipping integration test")
			return
		}
		packageContainer = ctr

		connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Skip("failed to get connection string, skipping integration test")
			return
		}

		pool, err := pgxpool.New(ctx, connStr)
		if err != nil {
			t.Skip("failed to create pool, skipping integration test")
			return
		}
		packagePool = pool

		// Cleanup is handled by testcontainers' Ryuk resource reaper when
		// the test process exits.  Do NOT register t.Cleanup here — doing so
		// ties the container lifetime to the first caller's test function,
		// which breaks subsequent test functions in the same package that
		// share the container via sync.Once.
	})
	return packagePool
}

// PackageTestPool returns the cached package-level pool. Panics if
// SetupPackageContainer has not been called first.
func PackageTestPool() *pgxpool.Pool {
	if packagePool == nil {
		panic("PackageTestPool: SetupPackageContainer must be called first")
	}
	return packagePool
}
