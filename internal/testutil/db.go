package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDSN = "postgres://ser:ser@localhost:5432/ser?sslmode=disable"

// TestPool returns a *pgxpool.Pool connected to the dev database.
// It skips the test if running with -short (unit-test mode) or if
// the database is unreachable.
func TestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping repo test: requires Postgres (run without -short)")
	}

	pool, err := pgxpool.New(context.Background(), testDSN)
	if err != nil {
		t.Fatalf("failed to create pgxpool: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping repo test: cannot reach Postgres: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// TxForTest starts a transaction and returns a *pgx.Tx that will be
// rolled back when the test completes. This keeps tests isolated.
func TxForTest(t *testing.T, pool *pgxpool.Pool) pgx.Tx {
	t.Helper()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})
	return tx
}
