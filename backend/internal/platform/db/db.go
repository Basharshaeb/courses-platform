// Package db owns the Postgres pool and transaction helpers.
//
// Every query that touches tenant tables must run inside WithTenantTx so that row level
// security sees the current school (see docs/adr/0001-multi-tenancy-with-rls.md).
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/store"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// WithTx runs fn in a transaction without tenant context (global tables only).
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(q *store.Queries, tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return fn(store.New(tx), tx)
	})
}

// WithTenantTx runs fn in a transaction scoped to schoolID for row level security.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, schoolID uuid.UUID, fn func(q *store.Queries, tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.school_id', $1, true)", schoolID.String()); err != nil {
			return fmt.Errorf("db: set tenant: %w", err)
		}
		return fn(store.New(tx), tx)
	})
}

// IsNotFound reports whether err means "no rows".
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// UniqueViolation returns the violated constraint name if err is a unique violation.
func UniqueViolation(err error) (constraint string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName, true
	}
	return "", false
}
