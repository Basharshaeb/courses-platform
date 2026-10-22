package school

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

// Role of an account in a school (school_members.role).
type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleAuthor  Role = "author"
	RoleSupport Role = "support"
	RoleFinance Role = "finance"
)

// Permission is an action a school member may be allowed to take.
type Permission string

const (
	// PermCoursesView lets a member see courses in the dashboard.
	PermCoursesView Permission = "courses.view"
	// PermCoursesEdit lets a member create, edit, reorder, publish and delete courses.
	PermCoursesEdit Permission = "courses.edit"
)

var rolePermissions = map[Role]map[Permission]bool{
	RoleOwner:   {PermCoursesView: true, PermCoursesEdit: true},
	RoleAdmin:   {PermCoursesView: true, PermCoursesEdit: true},
	RoleAuthor:  {PermCoursesView: true, PermCoursesEdit: true},
	RoleSupport: {PermCoursesView: true},
	RoleFinance: {PermCoursesView: true},
}

// Can reports whether the role grants p.
func (r Role) Can(p Permission) bool { return rolePermissions[r][p] }

// Authorize checks that accountID is a member of schoolID with permission p and returns its role.
// Non-members get a 404 so school ids cannot be probed; members lacking p get a 403.
func (s *Service) Authorize(ctx context.Context, schoolID, accountID uuid.UUID, p Permission) (Role, error) {
	var role string
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		role, err = q.GetMemberRole(ctx, store.GetMemberRoleParams{SchoolID: schoolID, AccountID: accountID})
		return err
	})
	if db.IsNotFound(err) {
		return "", apperr.NotFound("school_not_found", "school not found")
	}
	if err != nil {
		return "", fmt.Errorf("school: member role: %w", err)
	}
	if !Role(role).Can(p) {
		return "", apperr.Forbidden("your role in this school does not allow this action")
	}
	return Role(role), nil
}
