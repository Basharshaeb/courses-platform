// Package identity owns creator accounts, students and sessions.
package identity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/crypto"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

type Service struct {
	pool       *pgxpool.Pool
	schools    *school.Service
	sessionTTL time.Duration
}

func NewService(pool *pgxpool.Pool, schools *school.Service, sessionTTL time.Duration) *Service {
	return &Service{pool: pool, schools: schools, sessionTTL: sessionTTL}
}

// ClientMeta is stored with a session for the user's device list and audits.
type ClientMeta struct {
	IP        string
	UserAgent string
}

// NewSession is returned after login/signup. Token is the raw cookie value.
type NewSession struct {
	Token     string
	ExpiresAt time.Time
}

var errInvalidCredentials = apperr.Unauthorized("invalid email or password")

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func defaultLocale(l string) string {
	if l == "en" {
		return "en"
	}
	return "ar"
}

// ---- Accounts (creators) ----

type AccountSignupInput struct {
	Email, Password, Name, SchoolName, Subdomain, Locale string
}

type AccountWithSchools struct {
	Account store.Account
	Schools []store.ListAccountSchoolsRow
}

// SignupAccount creates a creator account, its first school and a session in one transaction.
func (s *Service) SignupAccount(ctx context.Context, in AccountSignupInput, meta ClientMeta) (AccountWithSchools, NewSession, error) {
	sub := strings.ToLower(strings.TrimSpace(in.Subdomain))
	if err := school.ValidateSubdomain(sub); err != nil {
		return AccountWithSchools{}, NewSession{}, err
	}
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return AccountWithSchools{}, NewSession{}, err
	}
	locale := defaultLocale(in.Locale)
	schoolID := uuid.Must(uuid.NewV7())

	var out AccountWithSchools
	var sess NewSession
	err = db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		acc, err := q.CreateAccount(ctx, store.CreateAccountParams{
			ID:           uuid.Must(uuid.NewV7()),
			Email:        normalizeEmail(in.Email),
			Name:         strings.TrimSpace(in.Name),
			PasswordHash: &hash,
			Locale:       locale,
		})
		if err != nil {
			if c, ok := db.UniqueViolation(err); ok && c == "accounts_email_key" {
				return apperr.Conflict("email_taken", "an account with this email already exists")
			}
			return fmt.Errorf("identity: create account: %w", err)
		}
		sch, err := s.schools.Provision(ctx, q, school.ProvisionInput{
			ID: schoolID, OwnerID: acc.ID, Name: strings.TrimSpace(in.SchoolName), Subdomain: sub, Locale: locale,
		})
		if err != nil {
			return err
		}
		sess, err = s.createSession(ctx, q, "account", acc.ID, nil, meta)
		if err != nil {
			return err
		}
		out = AccountWithSchools{
			Account: acc,
			Schools: []store.ListAccountSchoolsRow{{SchoolID: sch.ID, Name: sch.Name, Subdomain: sch.Subdomain, Role: "owner"}},
		}
		return nil
	})
	return out, sess, err
}

// LoginAccount checks credentials and opens a session.
func (s *Service) LoginAccount(ctx context.Context, email, password string, meta ClientMeta) (AccountWithSchools, NewSession, error) {
	q := store.New(s.pool)
	acc, err := q.GetAccountByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if db.IsNotFound(err) {
			crypto.BurnPasswordCheck(password)
			return AccountWithSchools{}, NewSession{}, errInvalidCredentials
		}
		return AccountWithSchools{}, NewSession{}, err
	}
	if err := checkPassword(password, acc.PasswordHash); err != nil {
		return AccountWithSchools{}, NewSession{}, err
	}
	sess, err := s.createSession(ctx, q, "account", acc.ID, nil, meta)
	if err != nil {
		return AccountWithSchools{}, NewSession{}, err
	}
	schools, err := s.schools.ListForAccount(ctx, acc.ID)
	if err != nil {
		return AccountWithSchools{}, NewSession{}, err
	}
	return AccountWithSchools{Account: acc, Schools: schools}, sess, nil
}

// GetAccount returns an account with its schools.
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (AccountWithSchools, error) {
	acc, err := store.New(s.pool).GetAccountByID(ctx, accountID)
	if err != nil {
		if db.IsNotFound(err) {
			return AccountWithSchools{}, apperr.Unauthorized("account no longer exists")
		}
		return AccountWithSchools{}, err
	}
	schools, err := s.schools.ListForAccount(ctx, acc.ID)
	if err != nil {
		return AccountWithSchools{}, err
	}
	return AccountWithSchools{Account: acc, Schools: schools}, nil
}

// ---- Students ----

type StudentSignupInput struct {
	Email, Password, Name, Locale string
}

// SignupStudent creates a student in schoolID and opens a session.
func (s *Service) SignupStudent(ctx context.Context, schoolID uuid.UUID, in StudentSignupInput, meta ClientMeta) (store.Student, NewSession, error) {
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return store.Student{}, NewSession{}, err
	}
	var st store.Student
	var sess NewSession
	err = db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		st, err = q.CreateStudent(ctx, store.CreateStudentParams{
			ID:           uuid.Must(uuid.NewV7()),
			SchoolID:     schoolID,
			Email:        normalizeEmail(in.Email),
			Name:         strings.TrimSpace(in.Name),
			PasswordHash: &hash,
			Locale:       defaultLocale(in.Locale),
		})
		if err != nil {
			if c, ok := db.UniqueViolation(err); ok && c == "students_school_id_email_key" {
				return apperr.Conflict("email_taken", "a student with this email already exists")
			}
			return fmt.Errorf("identity: create student: %w", err)
		}
		sess, err = s.createSession(ctx, q, "student", st.ID, &schoolID, meta)
		return err
	})
	return st, sess, err
}

// LoginStudent checks credentials within schoolID and opens a session.
func (s *Service) LoginStudent(ctx context.Context, schoolID uuid.UUID, email, password string, meta ClientMeta) (store.Student, NewSession, error) {
	var st store.Student
	var sess NewSession
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		st, err = q.GetStudentByEmail(ctx, store.GetStudentByEmailParams{SchoolID: schoolID, Email: normalizeEmail(email)})
		if err != nil {
			if db.IsNotFound(err) {
				crypto.BurnPasswordCheck(password)
				return errInvalidCredentials
			}
			return err
		}
		if err := checkPassword(password, st.PasswordHash); err != nil {
			return err
		}
		if st.Status != "active" {
			return apperr.Forbidden("this account is blocked")
		}
		sess, err = s.createSession(ctx, q, "student", st.ID, &schoolID, meta)
		return err
	})
	return st, sess, err
}

// GetStudent returns a student of schoolID.
func (s *Service) GetStudent(ctx context.Context, schoolID, studentID uuid.UUID) (store.Student, error) {
	var st store.Student
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		st, err = q.GetStudentByID(ctx, store.GetStudentByIDParams{SchoolID: schoolID, ID: studentID})
		return err
	})
	if db.IsNotFound(err) {
		return store.Student{}, apperr.Unauthorized("student no longer exists")
	}
	return st, err
}

// ---- Sessions ----

func checkPassword(password string, hash *string) error {
	if hash == nil {
		crypto.BurnPasswordCheck(password)
		return errInvalidCredentials
	}
	ok, err := crypto.VerifyPassword(password, *hash)
	if err != nil {
		return fmt.Errorf("identity: verify password: %w", err)
	}
	if !ok {
		return errInvalidCredentials
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, q *store.Queries, subjectType string, subjectID uuid.UUID, schoolID *uuid.UUID, meta ClientMeta) (NewSession, error) {
	token, hash, err := crypto.NewToken()
	if err != nil {
		return NewSession{}, err
	}
	expires := time.Now().Add(s.sessionTTL)
	err = q.CreateSession(ctx, store.CreateSessionParams{
		ID:          uuid.Must(uuid.NewV7()),
		TokenHash:   hash,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		SchoolID:    schoolID,
		Ip:          nonEmpty(meta.IP),
		UserAgent:   nonEmpty(truncate(meta.UserAgent, 512)),
		ExpiresAt:   expires,
	})
	if err != nil {
		return NewSession{}, fmt.Errorf("identity: create session: %w", err)
	}
	return NewSession{Token: token, ExpiresAt: expires}, nil
}

// Authenticate resolves a session cookie to a principal. Unknown or expired tokens return ok=false.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, bool, error) {
	if token == "" {
		return Principal{}, false, nil
	}
	hash := crypto.HashToken(token)
	sess, err := store.New(s.pool).GetActiveSession(ctx, hash)
	if err != nil {
		if db.IsNotFound(err) {
			return Principal{}, false, nil
		}
		return Principal{}, false, err
	}
	return Principal{
		Kind:      Kind(sess.SubjectType),
		ID:        sess.SubjectID,
		SchoolID:  sess.SchoolID,
		tokenHash: hash,
	}, true, nil
}

// Logout revokes the session of p.
func (s *Service) Logout(ctx context.Context, p Principal) error {
	return store.New(s.pool).RevokeSession(ctx, p.tokenHash)
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
