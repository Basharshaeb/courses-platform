-- name: CreateAccount :one
INSERT INTO accounts (id, email, name, password_hash, locale)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountByEmail :one
SELECT * FROM accounts WHERE email = $1;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: CreateStudent :one
INSERT INTO students (id, school_id, email, name, password_hash, locale)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStudentByEmail :one
SELECT * FROM students WHERE school_id = $1 AND email = $2;

-- name: GetStudentByID :one
SELECT * FROM students WHERE school_id = $1 AND id = $2;

-- name: CreateSession :exec
INSERT INTO sessions (id, token_hash, subject_type, subject_id, school_id, ip, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetActiveSession :one
SELECT * FROM sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: DeleteStaleSessions :execrows
DELETE FROM sessions WHERE expires_at < sqlc.arg(before) OR revoked_at < sqlc.arg(before);
