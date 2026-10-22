package api_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/teatchable/backend/internal/api"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/tenant"
	"github.com/teatchable/backend/internal/testutil/pgtest"
)

var srv *httptest.Server

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}
	ctx := context.Background()
	env, stop, err := pgtest.Start(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := config.Config{
		RootDomain: "localhost", AppHost: "app.localhost", AdminHost: "admin.localhost",
		TrustProxy: true, SessionTTL: time.Hour,
	}
	schools := school.NewService(env.AppPool)
	h, err := api.NewRouter(api.Deps{
		Config:   cfg,
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pool:     env.AppPool,
		Resolver: tenant.NewResolver(env.AppPool, tenant.Hosts{RootDomain: cfg.RootDomain, AppHost: cfg.AppHost, AdminHost: cfg.AdminHost}, 0),
		Identity: identity.NewService(env.AppPool, schools, cfg.SessionTTL),
		Schools:  schools,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	srv = httptest.NewServer(h)
	code := m.Run()
	srv.Close()
	stop()
	os.Exit(code)
}

type response struct {
	status int
	body   map[string]any
	cookie *http.Cookie
}

// call sends a request as if it came through the edge proxy for host.
func call(t *testing.T, method, host, path, body string, cookie *http.Cookie, origin string) response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+"/api/v1"+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-Host", host)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	out := response{status: res.StatusCode}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	for _, c := range res.Cookies() {
		if c.Name == "sid" {
			out.cookie = c
		}
	}
	return out
}

func expect(t *testing.T, r response, status int, code string) {
	t.Helper()
	if r.status != status {
		t.Fatalf("status = %d, want %d (body %v)", r.status, status, r.body)
	}
	if code != "" && r.body["code"] != code {
		t.Fatalf("code = %v, want %s", r.body["code"], code)
	}
}

const (
	appHost = "app.localhost:3000"
	appOrig = "http://app.localhost:3000"
)

func TestCreatorAndStudentFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Creator signs up on the app host and gets a school.
	signup := `{"email":"Owner@Flow.test","password":"password123","name":"Owner","schoolName":"Flow Academy","subdomain":"flow","locale":"en"}`
	r := call(t, "POST", appHost, "/accounts/signup", signup, nil, appOrig)
	expect(t, r, http.StatusCreated, "")
	creator := r.cookie
	if creator == nil || !creator.HttpOnly {
		t.Fatalf("expected httpOnly session cookie, got %+v", creator)
	}
	if r.body["account"].(map[string]any)["email"] != "owner@flow.test" {
		t.Errorf("email not normalized: %v", r.body["account"])
	}

	expect(t, call(t, "POST", appHost, "/accounts/signup", signup, nil, appOrig), http.StatusConflict, "email_taken")
	expect(t, call(t, "POST", appHost, "/accounts/signup",
		`{"email":"x@flow.test","password":"password123","name":"x","schoolName":"x","subdomain":"flow"}`, nil, appOrig),
		http.StatusConflict, "subdomain_taken")
	expect(t, call(t, "POST", appHost, "/accounts/signup",
		`{"email":"bad","password":"short","name":"x","schoolName":"x","subdomain":"ok-sub"}`, nil, appOrig),
		http.StatusBadRequest, "invalid_request")

	r = call(t, "GET", appHost, "/accounts/me", "", creator, "")
	expect(t, r, http.StatusOK, "")
	if n := len(r.body["schools"].([]any)); n != 1 {
		t.Fatalf("schools = %d, want 1", n)
	}

	// Creator endpoints do not exist on school hosts; creator cookie is not a student session.
	expect(t, call(t, "POST", "flow.localhost:3000", "/accounts/login", `{"email":"owner@flow.test","password":"password123"}`, nil, ""), http.StatusNotFound, "not_found")
	expect(t, call(t, "GET", "flow.localhost:3000", "/me", "", creator, ""), http.StatusUnauthorized, "unauthorized")

	// Storefront resolves by host.
	r = call(t, "GET", "flow.localhost:3000", "/storefront/school", "", nil, "")
	expect(t, r, http.StatusOK, "")
	if r.body["name"] != "Flow Academy" {
		t.Errorf("school name = %v", r.body["name"])
	}
	expect(t, call(t, "GET", "missing.localhost:3000", "/storefront/school", "", nil, ""), http.StatusNotFound, "school_not_found")

	// Student signs up in the school; the session only works on that school's host.
	r = call(t, "POST", "flow.localhost:3000", "/auth/signup", `{"email":"learner@flow.test","password":"password123","name":"Learner"}`, nil, "http://flow.localhost:3000")
	expect(t, r, http.StatusCreated, "")
	student := r.cookie
	expect(t, call(t, "GET", "flow.localhost:3000", "/me", "", student, ""), http.StatusOK, "")
	expect(t, call(t, "GET", appHost, "/accounts/me", "", student, ""), http.StatusUnauthorized, "unauthorized")

	// A second school does not see the first school's students.
	r = call(t, "POST", appHost, "/accounts/signup", `{"email":"other@flow.test","password":"password123","name":"Other","schoolName":"Other","subdomain":"other"}`, nil, appOrig)
	expect(t, r, http.StatusCreated, "")
	expect(t, call(t, "GET", "other.localhost:3000", "/me", "", student, ""), http.StatusUnauthorized, "unauthorized")
	expect(t, call(t, "POST", "other.localhost:3000", "/auth/login", `{"email":"learner@flow.test","password":"password123"}`, nil, ""), http.StatusUnauthorized, "unauthorized")
	// Same email may register independently in another school.
	expect(t, call(t, "POST", "other.localhost:3000", "/auth/signup", `{"email":"learner@flow.test","password":"password123","name":"Learner"}`, nil, ""), http.StatusCreated, "")

	// CSRF: cross-origin mutations are rejected.
	expect(t, call(t, "POST", "flow.localhost:3000", "/auth/logout", "", student, "http://evil.test"), http.StatusForbidden, "cross_origin_request")

	// Logout revokes the session.
	expect(t, call(t, "POST", "flow.localhost:3000", "/auth/logout", "", student, "http://flow.localhost:3000"), http.StatusNoContent, "")
	expect(t, call(t, "GET", "flow.localhost:3000", "/me", "", student, ""), http.StatusUnauthorized, "unauthorized")

	// Wrong password and unknown user give the same answer.
	a := call(t, "POST", appHost, "/accounts/login", `{"email":"owner@flow.test","password":"wrong-password"}`, nil, "")
	b := call(t, "POST", appHost, "/accounts/login", `{"email":"nobody@flow.test","password":"wrong-password"}`, nil, "")
	expect(t, a, http.StatusUnauthorized, "unauthorized")
	expect(t, b, http.StatusUnauthorized, "unauthorized")
	if a.body["detail"] != b.body["detail"] {
		t.Errorf("login errors differ: %v vs %v", a.body["detail"], b.body["detail"])
	}
}

func TestTLSAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	call(t, "POST", appHost, "/accounts/signup", `{"email":"tls@flow.test","password":"password123","name":"T","schoolName":"T","subdomain":"tls-school"}`, nil, appOrig)
	expect(t, call(t, "GET", "edge", "/internal/tls-allowed?domain=tls-school.localhost", "", nil, ""), http.StatusOK, "")
	expect(t, call(t, "GET", "edge", "/internal/tls-allowed?domain=unknown.example.com", "", nil, ""), http.StatusNotFound, "domain_not_allowed")
	expect(t, call(t, "GET", "edge", "/internal/tls-allowed", "", nil, ""), http.StatusBadRequest, "invalid_request")
}
