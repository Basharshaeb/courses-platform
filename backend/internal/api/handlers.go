package api

import (
	"context"
	"net/http"

	"github.com/teatchable/backend/internal/api/gen"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/httpx"
	"github.com/teatchable/backend/internal/platform/tenant"
	"github.com/teatchable/backend/internal/store"
)

func clientMeta(ctx context.Context) identity.ClientMeta {
	c := httpx.ClientFrom(ctx)
	return identity.ClientMeta{IP: c.IP, UserAgent: c.UserAgent}
}

func localeOrEmpty(l *gen.Locale) string {
	if l == nil {
		return ""
	}
	return string(*l)
}

// ---- Storefront ----

func (s *Server) GetStorefrontSchool(ctx context.Context, _ gen.GetStorefrontSchoolRequestObject) (gen.GetStorefrontSchoolResponseObject, error) {
	schoolID, err := tenant.RequireSchool(ctx)
	if err != nil {
		return nil, err
	}
	sch, err := s.schools.Get(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	return gen.GetStorefrontSchool200JSONResponse{
		Id:            sch.ID,
		Name:          sch.Name,
		Subdomain:     sch.Subdomain,
		DefaultLocale: gen.Locale(sch.DefaultLocale),
		Currency:      sch.Currency,
	}, nil
}

// ---- Creator accounts (app host) ----

func (s *Server) SignupAccount(ctx context.Context, req gen.SignupAccountRequestObject) (gen.SignupAccountResponseObject, error) {
	if err := tenant.RequireSurface(ctx, tenant.SurfaceApp); err != nil {
		return nil, err
	}
	b := req.Body
	out, sess, err := s.identity.SignupAccount(ctx, identity.AccountSignupInput{
		Email:      string(b.Email),
		Password:   b.Password,
		Name:       b.Name,
		SchoolName: b.SchoolName,
		Subdomain:  b.Subdomain,
		Locale:     localeOrEmpty(b.Locale),
	}, clientMeta(ctx))
	if err != nil {
		return nil, err
	}
	cookie := s.cookies.Session(sess)
	return gen.SignupAccount201JSONResponse{
		Body:    toAccountSession(out),
		Headers: gen.SignupAccount201ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *Server) LoginAccount(ctx context.Context, req gen.LoginAccountRequestObject) (gen.LoginAccountResponseObject, error) {
	if err := tenant.RequireSurface(ctx, tenant.SurfaceApp); err != nil {
		return nil, err
	}
	out, sess, err := s.identity.LoginAccount(ctx, string(req.Body.Email), req.Body.Password, clientMeta(ctx))
	if err != nil {
		return nil, err
	}
	cookie := s.cookies.Session(sess)
	return gen.LoginAccount200JSONResponse{
		Body:    toAccountSession(out),
		Headers: gen.LoginAccount200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *Server) LogoutAccount(ctx context.Context, _ gen.LogoutAccountRequestObject) (gen.LogoutAccountResponseObject, error) {
	p, err := identity.RequireAccount(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.identity.Logout(ctx, p); err != nil {
		return nil, err
	}
	cookie := s.cookies.Cleared()
	return gen.LogoutAccount204Response{Headers: gen.LogoutAccount204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (s *Server) GetCurrentAccount(ctx context.Context, _ gen.GetCurrentAccountRequestObject) (gen.GetCurrentAccountResponseObject, error) {
	p, err := identity.RequireAccount(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.identity.GetAccount(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	return gen.GetCurrentAccount200JSONResponse(toAccountSession(out)), nil
}

// ---- Students (school host) ----

func (s *Server) SignupStudent(ctx context.Context, req gen.SignupStudentRequestObject) (gen.SignupStudentResponseObject, error) {
	schoolID, err := tenant.RequireSchool(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	st, sess, err := s.identity.SignupStudent(ctx, schoolID, identity.StudentSignupInput{
		Email:    string(b.Email),
		Password: b.Password,
		Name:     b.Name,
		Locale:   localeOrEmpty(b.Locale),
	}, clientMeta(ctx))
	if err != nil {
		return nil, err
	}
	cookie := s.cookies.Session(sess)
	return gen.SignupStudent201JSONResponse{
		Body:    toStudent(st),
		Headers: gen.SignupStudent201ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *Server) LoginStudent(ctx context.Context, req gen.LoginStudentRequestObject) (gen.LoginStudentResponseObject, error) {
	schoolID, err := tenant.RequireSchool(ctx)
	if err != nil {
		return nil, err
	}
	st, sess, err := s.identity.LoginStudent(ctx, schoolID, string(req.Body.Email), req.Body.Password, clientMeta(ctx))
	if err != nil {
		return nil, err
	}
	cookie := s.cookies.Session(sess)
	return gen.LoginStudent200JSONResponse{
		Body:    toStudent(st),
		Headers: gen.LoginStudent200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (s *Server) LogoutStudent(ctx context.Context, _ gen.LogoutStudentRequestObject) (gen.LogoutStudentResponseObject, error) {
	p, err := identity.RequireStudent(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.identity.Logout(ctx, p); err != nil {
		return nil, err
	}
	cookie := s.cookies.Cleared()
	return gen.LogoutStudent204Response{Headers: gen.LogoutStudent204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (s *Server) GetCurrentStudent(ctx context.Context, _ gen.GetCurrentStudentRequestObject) (gen.GetCurrentStudentResponseObject, error) {
	p, err := identity.RequireStudent(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.identity.GetStudent(ctx, *p.SchoolID, p.ID)
	if err != nil {
		return nil, err
	}
	return gen.GetCurrentStudent200JSONResponse(toStudent(st)), nil
}

// ---- Internal ----

// CheckTLSAllowed answers Caddy's on_demand_tls "ask" request: 200 only for hosts we serve.
func (s *Server) CheckTLSAllowed(ctx context.Context, req gen.CheckTLSAllowedRequestObject) (gen.CheckTLSAllowedResponseObject, error) {
	t, err := s.resolver.Resolve(ctx, httpx.StripPort(req.Params.Domain))
	if err != nil {
		return nil, err
	}
	if t.Surface == tenant.SurfaceUnknown {
		return nil, apperr.New(http.StatusNotFound, "domain_not_allowed", "domain is not served by this platform")
	}
	return gen.CheckTLSAllowed200Response{}, nil
}

// ---- Mapping ----

func toAccountSession(a identity.AccountWithSchools) gen.AccountSession {
	schools := make([]gen.SchoolMembership, 0, len(a.Schools))
	for _, m := range a.Schools {
		schools = append(schools, gen.SchoolMembership{
			SchoolId:  m.SchoolID,
			Name:      m.Name,
			Subdomain: m.Subdomain,
			Role:      gen.SchoolMembershipRole(m.Role),
		})
	}
	return gen.AccountSession{
		Account: gen.Account{
			Id:        a.Account.ID,
			Email:     a.Account.Email,
			Name:      a.Account.Name,
			Locale:    gen.Locale(a.Account.Locale),
			CreatedAt: a.Account.CreatedAt.UTC(),
		},
		Schools: schools,
	}
}

func toStudent(st store.Student) gen.Student {
	return gen.Student{
		Id:        st.ID,
		Email:     st.Email,
		Name:      st.Name,
		Locale:    gen.Locale(st.Locale),
		CreatedAt: st.CreatedAt.UTC(),
	}
}
