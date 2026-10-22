// Package tenant maps the request host to a surface (marketing, creator app, platform admin)
// or to a school, and stores the result in the request context.
package tenant

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/httpx"
)

type Surface string

const (
	SurfaceMarketing Surface = "marketing"
	SurfaceApp       Surface = "app"
	SurfaceAdmin     Surface = "admin"
	SurfaceSchool    Surface = "school"
	SurfaceUnknown   Surface = "unknown"
)

type Tenant struct {
	Surface  Surface
	SchoolID uuid.UUID // set when Surface == SurfaceSchool
}

type Hosts struct {
	RootDomain string
	AppHost    string
	AdminHost  string
}

const maxCacheEntries = 10_000

type cacheEntry struct {
	tenant  Tenant
	expires time.Time
}

// Resolver resolves hosts with a small in-process TTL cache (positive and negative results).
type Resolver struct {
	pool  *pgxpool.Pool
	hosts Hosts
	ttl   time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func NewResolver(pool *pgxpool.Pool, hosts Hosts, ttl time.Duration) *Resolver {
	return &Resolver{pool: pool, hosts: hosts, ttl: ttl, cache: make(map[string]cacheEntry)}
}

// Resolve maps a lower-case host without port to a tenant.
func (r *Resolver) Resolve(ctx context.Context, host string) (Tenant, error) {
	switch host {
	case r.hosts.AppHost:
		return Tenant{Surface: SurfaceApp}, nil
	case r.hosts.AdminHost:
		return Tenant{Surface: SurfaceAdmin}, nil
	case r.hosts.RootDomain, "www." + r.hosts.RootDomain:
		return Tenant{Surface: SurfaceMarketing}, nil
	}

	now := time.Now()
	r.mu.RLock()
	e, ok := r.cache[host]
	r.mu.RUnlock()
	if ok && now.Before(e.expires) {
		return e.tenant, nil
	}

	var subdomain, hostname *string
	if label, isSub := strings.CutSuffix(host, "."+r.hosts.RootDomain); isSub {
		if strings.Contains(label, ".") {
			return Tenant{Surface: SurfaceUnknown}, nil
		}
		subdomain = &label
	} else {
		hostname = &host
	}

	var id *uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT resolve_school_id($1, $2)", subdomain, hostname).Scan(&id); err != nil {
		return Tenant{}, fmt.Errorf("tenant: resolve %q: %w", host, err)
	}
	t := Tenant{Surface: SurfaceUnknown}
	if id != nil {
		t = Tenant{Surface: SurfaceSchool, SchoolID: *id}
	}

	r.mu.Lock()
	if len(r.cache) >= maxCacheEntries {
		clear(r.cache)
	}
	r.cache[host] = cacheEntry{tenant: t, expires: now.Add(r.ttl)}
	r.mu.Unlock()
	return t, nil
}

// Invalidate drops a cached host, e.g. after a subdomain or custom domain change.
func (r *Resolver) Invalidate(host string) {
	r.mu.Lock()
	delete(r.cache, host)
	r.mu.Unlock()
}

type ctxKey struct{}

func WithTenant(ctx context.Context, t Tenant) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

func FromContext(ctx context.Context) Tenant {
	t, ok := ctx.Value(ctxKey{}).(Tenant)
	if !ok {
		return Tenant{Surface: SurfaceUnknown}
	}
	return t
}

// RequireSchool returns the school of the current request or a 404 problem.
func RequireSchool(ctx context.Context) (uuid.UUID, error) {
	t := FromContext(ctx)
	if t.Surface != SurfaceSchool {
		return uuid.Nil, apperr.NotFound("school_not_found", "no school is served on this host")
	}
	return t.SchoolID, nil
}

// RequireSurface returns a 404 problem unless the request is on the given surface.
func RequireSurface(ctx context.Context, s Surface) error {
	if FromContext(ctx).Surface != s {
		return apperr.NotFound("not_found", "endpoint not available on this host")
	}
	return nil
}

// Middleware resolves the tenant for every request. Must run after httpx.ClientInfo.
func Middleware(r *Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			t, err := r.Resolve(req.Context(), httpx.ClientFrom(req.Context()).Host)
			if err != nil {
				httpx.WriteError(w, req, err)
				return
			}
			next.ServeHTTP(w, req.WithContext(WithTenant(req.Context(), t)))
		})
	}
}
