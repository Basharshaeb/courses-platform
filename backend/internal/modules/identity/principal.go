package identity

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/httpx"
	"github.com/teatchable/backend/internal/platform/tenant"
)

type Kind string

const (
	KindAccount Kind = "account"
	KindStudent Kind = "student"
)

// Principal is the authenticated subject of a request.
type Principal struct {
	Kind     Kind
	ID       uuid.UUID
	SchoolID *uuid.UUID // students only

	tokenHash []byte
}

type ctxKey struct{}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// RequireAccount returns the creator account of the request or a 401 problem.
func RequireAccount(ctx context.Context) (Principal, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok || p.Kind != KindAccount {
		return Principal{}, apperr.Unauthorized("login required")
	}
	return p, nil
}

// RequireStudent returns the student of the request or a 401 problem.
func RequireStudent(ctx context.Context) (Principal, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok || p.Kind != KindStudent {
		return Principal{}, apperr.Unauthorized("login required")
	}
	return p, nil
}

// Cookies builds session cookies. Over HTTPS the __Host- prefix pins the cookie to the exact host.
type Cookies struct {
	Secure bool
}

func (c Cookies) Name() string {
	if c.Secure {
		return "__Host-sid"
	}
	return "sid"
}

func (c Cookies) Session(s NewSession) string {
	return (&http.Cookie{ //nolint:gosec // Secure comes from COOKIE_SECURE; off only for local http dev
		Name:     c.Name(),
		Value:    s.Token,
		Path:     "/",
		Expires:  s.ExpiresAt,
		MaxAge:   int(time.Until(s.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

func (c Cookies) Cleared() string {
	return (&http.Cookie{ //nolint:gosec // Secure comes from COOKIE_SECURE; off only for local http dev
		Name:     c.Name(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

// Middleware loads the session cookie into the request context. A session is only honored on
// the surface it belongs to: accounts on the creator app host, students on their own school.
func Middleware(svc *Service, cookies Cookies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ck, err := r.Cookie(cookies.Name())
			if err != nil || ck.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			p, ok, err := svc.Authenticate(r.Context(), ck.Value)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			if ok && allowedOnSurface(p, tenant.FromContext(r.Context())) {
				r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, p))
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowedOnSurface(p Principal, t tenant.Tenant) bool {
	switch p.Kind {
	case KindAccount:
		return t.Surface == tenant.SurfaceApp
	case KindStudent:
		return t.Surface == tenant.SurfaceSchool && p.SchoolID != nil && *p.SchoolID == t.SchoolID
	}
	return false
}
