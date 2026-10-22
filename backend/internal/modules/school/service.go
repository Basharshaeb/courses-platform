// Package school owns schools, their domains, branding and staff.
package school

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

var subdomainRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$`)

// reserved subdomains are platform hosts or likely future ones.
var reserved = map[string]bool{
	"app": true, "admin": true, "api": true, "www": true, "edge": true, "mail": true, "smtp": true,
	"static": true, "cdn": true, "assets": true, "media": true, "help": true, "support": true,
	"status": true, "blog": true, "docs": true, "dashboard": true, "billing": true, "auth": true,
	"login": true, "signup": true, "account": true, "accounts": true, "dev": true, "staging": true,
	"test": true, "internal": true, "webhooks": true,
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type ProvisionInput struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	Subdomain string
	Locale    string
}

// ValidateSubdomain checks format and reserved names.
func ValidateSubdomain(sub string) error {
	if !subdomainRe.MatchString(sub) {
		return apperr.BadRequest("invalid_subdomain", "subdomain must be 3-32 lower-case letters, digits or dashes")
	}
	if reserved[sub] {
		return apperr.Conflict("subdomain_taken", "this subdomain is reserved")
	}
	return nil
}

// Provision creates a school and makes the owner a member. It must run inside a tenant
// transaction for in.ID (see db.WithTenantTx) so the RLS insert check passes.
func (s *Service) Provision(ctx context.Context, q *store.Queries, in ProvisionInput) (store.School, error) {
	if err := ValidateSubdomain(in.Subdomain); err != nil {
		return store.School{}, err
	}
	sch, err := q.CreateSchool(ctx, store.CreateSchoolParams{
		ID:             in.ID,
		OwnerAccountID: in.OwnerID,
		Name:           in.Name,
		Subdomain:      in.Subdomain,
		DefaultLocale:  in.Locale,
	})
	if err != nil {
		if c, ok := db.UniqueViolation(err); ok && c == "schools_subdomain_key" {
			return store.School{}, apperr.Conflict("subdomain_taken", "this subdomain is already in use")
		}
		return store.School{}, fmt.Errorf("school: create: %w", err)
	}
	if err := q.AddSchoolMember(ctx, store.AddSchoolMemberParams{SchoolID: sch.ID, AccountID: in.OwnerID, Role: "owner"}); err != nil {
		return store.School{}, fmt.Errorf("school: add owner: %w", err)
	}
	return sch, nil
}

// Get returns a school by id.
func (s *Service) Get(ctx context.Context, schoolID uuid.UUID) (store.School, error) {
	var sch store.School
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		sch, err = q.GetSchool(ctx, schoolID)
		return err
	})
	if db.IsNotFound(err) {
		return store.School{}, apperr.NotFound("school_not_found", "school not found")
	}
	return sch, err
}

// ListForAccount returns the schools an account is a member of.
func (s *Service) ListForAccount(ctx context.Context, accountID uuid.UUID) ([]store.ListAccountSchoolsRow, error) {
	return store.New(s.pool).ListAccountSchools(ctx, accountID)
}
