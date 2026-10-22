package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/teatchable/backend/db/migrations"
)

// Migrate applies goose schema migrations and River's queue migrations. url must connect as the
// schema owner. Progress is written to out.
func Migrate(ctx context.Context, url string, out io.Writer) error {
	provider, closeDB, err := gooseProvider(url)
	if err != nil {
		return err
	}
	defer closeDB()

	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Fprintln(out, "applied", r.Source.Path, r.Duration)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrate: %w", err)
	}
	for _, v := range res.Versions {
		fmt.Fprintln(out, "applied river migration", v.Version)
	}
	return nil
}

// MigrateDown rolls back the most recent goose migration.
func MigrateDown(ctx context.Context, url string, out io.Writer) error {
	provider, closeDB, err := gooseProvider(url)
	if err != nil {
		return err
	}
	defer closeDB()
	res, err := provider.Down(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "rolled back", res.Source.Path)
	return nil
}

// MigrateStatus prints the state of every goose migration.
func MigrateStatus(ctx context.Context, url string, out io.Writer) error {
	provider, closeDB, err := gooseProvider(url)
	if err != nil {
		return err
	}
	defer closeDB()
	statuses, err := provider.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		fmt.Fprintf(out, "%-8s %s\n", s.State, s.Source.Path)
	}
	return nil
}

func gooseProvider(url string) (*goose.Provider, func(), error) {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return nil, nil, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return provider, func() { _ = sqlDB.Close() }, nil
}
