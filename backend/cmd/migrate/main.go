// Command migrate applies schema migrations (goose + River) and seeds demo data.
//
//	migrate up      apply all migrations
//	migrate down    roll back the last goose migration
//	migrate status  list goose migrations
//	migrate seed    create demo creator, school and student (idempotent)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/platform/tenant"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if err := run(context.Background(), cmd); err != nil {
		slog.Error("migrate failed", "cmd", cmd, "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch cmd {
	case "up":
		if err := db.Migrate(ctx, cfg.MigrateDatabaseURL, os.Stdout); err != nil {
			return err
		}
		fmt.Println("migrations up to date")
		return nil
	case "down":
		return db.MigrateDown(ctx, cfg.MigrateDatabaseURL, os.Stdout)
	case "status":
		return db.MigrateStatus(ctx, cfg.MigrateDatabaseURL, os.Stdout)
	case "seed":
		return seed(ctx, cfg)
	default:
		return fmt.Errorf("unknown command %q (want up, down, status, seed)", cmd)
	}
}

// Demo data for local development only. Credentials are documented in README.md.
const (
	demoCreatorEmail = "creator@demo.test"
	demoStudentEmail = "student@demo.test"
	demoPassword     = "demo-password-123"
	demoSubdomain    = "demo"
)

func seed(ctx context.Context, cfg config.Config) error {
	if cfg.IsProduction() {
		return errors.New("refusing to seed in production")
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	schools := school.NewService(pool)
	ids := identity.NewService(pool, schools, cfg.SessionTTL)
	meta := identity.ClientMeta{UserAgent: "seed"}

	_, _, err = ids.SignupAccount(ctx, identity.AccountSignupInput{
		Email: demoCreatorEmail, Password: demoPassword, Name: "Demo Creator",
		SchoolName: "أكاديمية التجربة", Subdomain: demoSubdomain, Locale: "ar",
	}, meta)
	if e, ok := apperr.As(err); ok && e.Code == "email_taken" {
		fmt.Println("demo creator exists")
	} else if err != nil {
		return err
	} else {
		fmt.Println("created creator", demoCreatorEmail, "with school", demoSubdomain)
	}

	resolver := tenant.NewResolver(pool, tenant.Hosts{RootDomain: cfg.RootDomain, AppHost: cfg.AppHost, AdminHost: cfg.AdminHost}, 0)
	t, err := resolver.Resolve(ctx, demoSubdomain+"."+cfg.RootDomain)
	if err != nil {
		return err
	}
	if t.Surface != tenant.SurfaceSchool {
		return errors.New("demo school not resolvable; is ROOT_DOMAIN set correctly?")
	}
	_, _, err = ids.SignupStudent(ctx, t.SchoolID, identity.StudentSignupInput{
		Email: demoStudentEmail, Password: demoPassword, Name: "Demo Student", Locale: "ar",
	}, meta)
	if e, ok := apperr.As(err); ok && e.Code == "email_taken" {
		fmt.Println("demo student exists")
	} else if err != nil {
		return err
	} else {
		fmt.Println("created student", demoStudentEmail)
	}
	return nil
}
