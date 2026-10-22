# ADR 0003: Same-origin API on every school host

- Status: Accepted
- Date: 2026-10-15

## Context
Schools live on `{sub}.yourapp.com` and on custom domains. Browsers block third-party cookies, so a
session cookie set by `api.yourapp.com` would not be sent from `academy.example.com`.

## Decision
- The edge proxy (Caddy) routes `https://{any school host}/api/*` to the Go API and everything else
  to Next.js. The API is always same-origin for the browser.
- Sessions are opaque random tokens (SHA-256 hash stored in DB, cached in Redis) in an
  `httpOnly; Secure; SameSite=Lax` host-only cookie. No JWTs in the browser.
- The API resolves the tenant from `X-Forwarded-Host` (or `Host`).
- Next.js server components call the API on the internal network, forwarding the cookie and
  `X-Forwarded-Host`.
- Mobile apps and the public API use `api.yourapp.com` with `Authorization: Bearer`.
- CSRF: SameSite=Lax plus an `Origin` header check on mutating requests.

## Consequences
- Works on any custom domain with no CORS and no third-party cookies.
- Revoking a session is immediate (delete the row / cache key).
- Local dev must emulate the proxy: Next.js rewrites `/api/*` to the Go API.
