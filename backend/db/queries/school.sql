-- name: CreateSchool :one
INSERT INTO schools (id, owner_account_id, name, subdomain, default_locale)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AddSchoolMember :exec
INSERT INTO school_members (school_id, account_id, role)
VALUES ($1, $2, $3);

-- name: GetSchool :one
SELECT * FROM schools WHERE id = $1;

-- name: ListAccountSchools :many
SELECT school_id::uuid AS school_id, name::text AS name, subdomain::text AS subdomain, role::text AS role
FROM account_school_memberships(sqlc.arg(account_id)::uuid);

-- name: GetMemberRole :one
SELECT role FROM school_members WHERE school_id = $1 AND account_id = $2;
