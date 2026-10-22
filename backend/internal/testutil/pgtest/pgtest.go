// Package pgtest starts a throwaway Postgres (testcontainers) with migrations applied, for
// integration tests. Tests using it are skipped with -short.
package pgtest

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/teatchable/backend/internal/platform/db"
)

const (
	appUser     = "teatchable_app"
	appPassword = "teatchable_app" //nolint:gosec // throwaway test container credential
)

type Env struct {
	OwnerURL string
	AppURL   string
	// AppPool connects as the RLS-enforced application role, like the API does.
	AppPool *pgxpool.Pool
}

// Start launches Postgres, creates the application role and runs all migrations.
// The returned func stops the container.
func Start(ctx context.Context) (*Env, func(), error) {
	ctr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("teatchable"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("pgtest: start container: %w", err)
	}
	stop := func() { _ = testcontainers.TerminateContainer(ctr) }

	ownerURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		stop()
		return nil, nil, err
	}

	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		stop()
		return nil, nil, err
	}
	_, err = conn.Exec(ctx, fmt.Sprintf(
		"CREATE ROLE app_rw NOLOGIN; CREATE ROLE %s LOGIN PASSWORD '%s' IN ROLE app_rw;", appUser, appPassword))
	_ = conn.Close(ctx)
	if err != nil {
		stop()
		return nil, nil, fmt.Errorf("pgtest: create roles: %w", err)
	}

	if err := db.Migrate(ctx, ownerURL, io.Discard); err != nil {
		stop()
		return nil, nil, fmt.Errorf("pgtest: migrate: %w", err)
	}

	u, err := url.Parse(ownerURL)
	if err != nil {
		stop()
		return nil, nil, err
	}
	u.User = url.UserPassword(appUser, appPassword)
	appURL := u.String()

	pool, err := db.Connect(ctx, appURL)
	if err != nil {
		stop()
		return nil, nil, err
	}
	return &Env{OwnerURL: ownerURL, AppURL: appURL, AppPool: pool}, func() {
		pool.Close()
		stop()
	}, nil
}
