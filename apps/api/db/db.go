// Package db owns the Postgres connection pool and migration runner.
//
// One pgxpool per process. Every caller takes a *pgxpool.Pool and uses
// pgx directly for parameterized queries; we do not wrap every query in
// a repository — Phase 1 has too few queries to justify the indirection.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrations are embedded at build time so the prod binary doesn't need
// filesystem access to apply them. infra/migrations/ is symlinked or copied
// into apps/api/db/migrations at build time, but for now (Phase 1) we
// embed from the relative path via go:embed pattern below.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// Open dials Postgres and returns a ready-to-use pool. Caller must defer Close.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if url == "" {
		return nil, errors.New("db: DATABASE_URL is empty")
	}

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	// Conservative pool defaults; tune later from env if we hit limits.
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}

	// Ping to surface DSN/auth errors immediately rather than on first query.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return pool, nil
}

// Migrate runs all pending up-migrations. Idempotent.
//
// Uses goose against the embedded migrations directory. We open a
// dedicated *sql.DB from the pool for goose (it needs database/sql, not
// pgx native) — closing the *sql.DB does not close the underlying pool.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	stdDB := stdlib.OpenDBFromPool(pool)
	defer stdDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	goose.SetBaseFS(migrationFS)
	goose.SetTableName("schema_migrations")

	if err := goose.UpContext(ctx, stdDB, "migrations"); err != nil {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// Tx is a tiny helper that runs fn inside a transaction. Commits on
// success, rolls back on error or panic.
func Tx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() {
		// Rollback is a no-op after commit; safe to always call.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
