// Package api wires HTTP: router, middleware, OpenAPI validation and the strict handlers that
// map generated request/response types to module services. Modules stay transport agnostic.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	oapimw "github.com/oapi-codegen/nethttp-middleware"

	"github.com/teatchable/backend/internal/api/gen"
	"github.com/teatchable/backend/internal/modules/catalog"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/httpx"
	"github.com/teatchable/backend/internal/platform/tenant"
)

const basePath = "/api/v1"

type Deps struct {
	Config   config.Config
	Log      *slog.Logger
	Pool     *pgxpool.Pool
	Resolver *tenant.Resolver
	Identity *identity.Service
	Schools  *school.Service
	Catalog  *catalog.Service
}

// Server implements gen.StrictServerInterface.
type Server struct {
	identity *identity.Service
	schools  *school.Service
	catalog  *catalog.Service
	resolver *tenant.Resolver
	cookies  identity.Cookies
}

var _ gen.StrictServerInterface = (*Server)(nil)

// NewRouter builds the full HTTP handler.
func NewRouter(d Deps) (http.Handler, error) {
	spec, err := gen.GetSpec()
	if err != nil {
		return nil, err
	}
	cookies := identity.Cookies{Secure: d.Config.CookieSecure}
	srv := &Server{identity: d.Identity, schools: d.Schools, catalog: d.Catalog, resolver: d.Resolver, cookies: cookies}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpx.ClientInfo(d.Config.TrustProxy))
	r.Use(httpx.RequestLogger(d.Log))
	r.Use(httpx.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := d.Pool.Ping(req.Context()); err != nil {
			httpx.WriteProblem(w, req, http.StatusServiceUnavailable, "db_unavailable", "")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route(basePath, func(r chi.Router) {
		r.Use(tenant.Middleware(d.Resolver))
		r.Use(httpx.RequireSameOrigin)
		r.Use(identity.Middleware(d.Identity, cookies))
		r.Use(oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{
			DoNotValidateServers: true,
			Prefix:               basePath,
			Options:              openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
			ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, req *http.Request, opts oapimw.ErrorHandlerOpts) {
				httpx.WriteProblem(w, req, opts.StatusCode, "invalid_request", validationDetail(err))
			},
		}))

		strict := gen.NewStrictHandlerWithOptions(srv, nil, gen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
				httpx.WriteProblem(w, req, http.StatusBadRequest, "invalid_request", err.Error())
			},
			ResponseErrorHandlerFunc: httpx.WriteError,
		})
		// Routes are relative to the basePath mount.
		gen.HandlerWithOptions(strict, gen.ChiServerOptions{
			BaseRouter: r,
			ErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
				httpx.WriteProblem(w, req, http.StatusBadRequest, "invalid_request", err.Error())
			},
		})
	})

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		httpx.WriteProblem(w, req, http.StatusNotFound, "not_found", "")
	})
	return r, nil
}

// validationDetail keeps the first line of a kin-openapi error, which names the failing field.
func validationDetail(err error) string {
	var re *openapi3filter.RequestError
	if errors.As(err, &re) {
		if re.Parameter != nil && re.Err != nil {
			return "parameter " + re.Parameter.Name + ": " + firstLine(re.Err)
		}
		if re.Err != nil {
			return firstLine(re.Err)
		}
	}
	return firstLine(err)
}

func firstLine(err error) string {
	s, _, _ := strings.Cut(err.Error(), "\n")
	return s
}
