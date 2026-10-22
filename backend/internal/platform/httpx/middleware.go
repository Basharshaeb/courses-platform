package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey int

const clientKey ctxKey = iota

// Client describes who sent the request, as seen through the edge proxy.
type Client struct {
	Host      string // lower-case host without port
	HostPort  string // host as sent, including port (used for Origin checks)
	IP        string
	UserAgent string
}

// ClientFrom returns the Client stored by the ClientInfo middleware.
func ClientFrom(ctx context.Context) Client {
	c, _ := ctx.Value(clientKey).(Client)
	return c
}

// ClientInfo resolves the request host and client IP. When trustProxy is set the
// X-Forwarded-Host / X-Forwarded-For headers from the edge proxy are honored.
func ClientInfo(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hostPort := r.Host
			ip := remoteIP(r.RemoteAddr)
			if trustProxy {
				if fh := firstHeaderValue(r.Header.Get("X-Forwarded-Host")); fh != "" {
					hostPort = fh
				}
				if ff := firstHeaderValue(r.Header.Get("X-Forwarded-For")); ff != "" {
					ip = ff
				}
			}
			c := Client{
				Host:      StripPort(strings.ToLower(hostPort)),
				HostPort:  strings.ToLower(hostPort),
				IP:        ip,
				UserAgent: r.UserAgent(),
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientKey, c)))
		})
	}
}

// RequireSameOrigin rejects unsafe requests whose Origin header names another host.
// Together with SameSite=Lax cookies this is the CSRF defense (docs/adr/0003-same-origin-api.md).
func RequireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, ClientFrom(r.Context()).HostPort) {
				WriteProblem(w, r, http.StatusForbidden, "cross_origin_request", "origin does not match host")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs one line per request.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			c := ClientFrom(r.Context())
			log.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("host", c.Host),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}

// Recoverer turns panics into 500 problem responses.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				slog.ErrorContext(r.Context(), "panic", "panic", rec, "stack", string(debug.Stack()))
				WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func StripPort(hostPort string) string {
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		return h
	}
	return hostPort
}

func remoteIP(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

func firstHeaderValue(v string) string {
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
