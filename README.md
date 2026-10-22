# Teatchable

Multi-tenant course platform (Teachable-style): creators get their own school on
`{name}.yourapp.com` or a custom domain, build courses and sell them; students learn on the
school's branded site. Arabic + English (RTL), Saudi/GCC payments (mada, Apple Pay, STC Pay via Moyasar; Stripe Connect).

- **API Documentation**: [docs/API.md](docs/API.md)
- **Feature Roadmap & Milestones**: [docs/ROADMAP.md](docs/ROADMAP.md)
- **Architecture Plan**: [docs/PLAN.md](docs/PLAN.md)
- **Architecture Decision Records (ADRs)**: [docs/adr/](docs/adr/)

---

## Status: Phase 1 Foundations & Experience

Working end-to-end:

- **Go Modular Monolith (`backend/`)**:
  - `api` server on chi router with strictly typed handlers generated from OpenAPI 3.0.3.
  - `worker` background process running River job queue with periodic task scheduling.
  - PostgreSQL schema with **Row-Level Security (RLS)** per school; connection pool as non-superuser role `teatchable_app`.
  - Modules: `identity`, `school`, `catalog`, `commerce`, `enrollment`, `media`.
  - Request validation (`kin-openapi`), RFC 9457 Problem error format, structured `slog` logging.
- **Next.js 16 Web Application (`web/`)**:
  - Host-based routing (`src/proxy.ts`): `app.` (creator dashboard), `admin.` (platform), `{sub}.` / custom domain (school site), root (marketing).
  - Modern RTL-first design (Arabic & English via `next-intl`).
  - Creator Course & Curriculum Builder (Drag-and-drop sections, lectures, content blocks).
  - Public Storefront with rich course previews, syllabus outlines, and FAQ accordion.
  - Interactive Student Course Player with simulated video player, completion checkboxes, and Q&A tabs.
- **Developer Experience**:
  - Single contract `api/openapi/openapi.yaml` generating Go strict server and TypeScript client.
  - One-command setup and background runners via [Task](https://taskfile.dev).

---

## Layout

```
api/openapi/openapi.yaml   API contract (source of truth for Go server + TS client)
backend/                   Go module: cmd/{api,worker,migrate}, internal/{api,modules,platform,store}
  db/migrations            goose SQL migrations (embedded in binaries)
  db/queries               sqlc queries → internal/store (generated)
web/                       Next.js app: src/app/{marketing,dashboard,platform,site/[host]}
deploy/                    Postgres init roles, Caddy edge config
docs/                      API documentation, Roadmap, Architecture Plan, and ADRs
```

Backend modules (`internal/modules/*`) hold business logic and stay transport-agnostic;
`internal/api` maps generated HTTP types to module services.

---

## Prerequisites

- Go 1.27+, Node 22 + pnpm 10, Docker or local Postgres 16/17, [Task](https://taskfile.dev)
- For codegen: [sqlc](https://sqlc.dev) 1.31, `oapi-codegen` 2.8.
- For lint: `golangci-lint`, `eslint`.

---

## Quick Start

```bash
task setup
```

Run all services together (API server on `:8080`, worker, and Next.js on `:3000`):

```bash
task dev
```

Or run individual services:
```bash
task api      # Starts Go API on :8080
task worker   # Starts River background job worker
task web      # Starts Next.js dev server on :3000
```

### URLs and Demo Access

Browsers automatically resolve `*.localhost` to `127.0.0.1` (no hosts file needed):

| URL | Surface | Description |
| :--- | :--- | :--- |
| **http://localhost:3000** | Marketing | Public landing page for creator acquisition |
| **http://app.localhost:3000** | Creator Dashboard | Course curriculum editor & academy settings |
| **http://demo.localhost:3000** | School Storefront | Demo academy storefront & student learning player |
| **http://admin.localhost:3000** | Platform Admin | Platform administration (placeholder) |

Demo credentials seeded via `task seed`:

| Role | Email | Password |
| :--- | :--- | :--- |
| **Creator** (owns school `demo`) | `creator@demo.test` | `demo-password-123` |
| **Student** (enrolled in `demo`) | `student@demo.test` | `demo-password-123` |

---

## 📐 Engineering & Code Standards

### 1. Multi-Tenancy & Security Rules (Zero Data Leaks)
- **Every tenant table MUST contain `school_id uuid NOT NULL`**.
- **Child tables MUST reference parent tables with composite foreign keys**:
  ```sql
  FOREIGN KEY (course_id, school_id) REFERENCES courses (id, school_id) ON DELETE CASCADE
  ```
  Foreign key checks bypass RLS; composite foreign keys ensure that even an attacker guessing an ID cannot attach entities to another tenant.
- **Always use `db.WithTenantTx(ctx, pool, schoolID, fn)`**:
  Never execute tenant queries on a raw connection. The tx helper sets `SET LOCAL app.school_id = ...` inside the transaction.
- **Defense-in-Depth Query Filtering**:
  All `sqlc` queries MUST also explicitly filter by `WHERE school_id = $1` in addition to RLS.

### 2. Go Backend Standards
- **Transport Agnostic Modules**:
  Business logic lives in `internal/modules/{domain}` and must NEVER import `net/http` or HTTP-specific types.
  `internal/api` acts purely as an adapter between the generated OpenAPI types and module services.
- **Cross-Module Communication**:
  Modules communicate strictly via exported Service Go interfaces or asynchronous River jobs. Direct database cross-table joins across module domains are forbidden.
- **Error Handling**:
  Return errors that map to stable RFC 9457 codes using `apperr.BadRequest(code, detail)`, `apperr.NotFound(code, detail)`, or `apperr.Forbidden(code)`. Never return raw database errors to the client.
- **IDs & Money**:
  - IDs are UUIDv7 (`uuid.Must(uuid.NewV7())`).
  - Money is stored as `bigint` minor units (e.g., Halalas for SAR, Cents for USD). Never use floating point numbers for currency calculations.

### 3. Frontend & TypeScript Standards (Next.js 16)
- **Server Components by Default**:
  Keep data fetching in async Server Components (`src/app/**/page.tsx`). Add `"use client"` only when interactive state, hooks (`useState`, `useEffect`), or browser APIs are required.
- **Type-Safe API Consumption**:
  Use `serverApi()` on the server and `api` on the client (both typed with generated `schema.d.ts` from `openapi-fetch`). Never write manual `fetch` calls or `any` casts.
- **RTL & Logical CSS**:
  The application is Arabic-first and English-ready. Never use physical CSS directions like `ml-`, `mr-`, `left-`, or `right-`. Always use logical properties:
  - `ms-` (margin-inline-start), `me-` (margin-inline-end)
  - `ps-` (padding-inline-start), `pe-` (padding-inline-end)
  - `start-`, `end-` for positioning
  - `text-start`, `text-end`
- **Design Aesthetics**:
  Use curated color tokens (Tailwind CSS variables), subtle glassmorphism, responsive container grids, and micro-animations for hover states and progress bars.

### 4. Code Generation & Contract Drift Policy
- Never hand-edit files in:
  - `backend/internal/store/*.sql.go` (generated by `sqlc`)
  - `backend/internal/api/gen/*.go` (generated by `oapi-codegen`)
  - `web/src/lib/api/schema.d.ts` (generated by `openapi-typescript`)
- After changing any migration, query, or OpenAPI path, always run:
  ```bash
  task gen
  ```
  CI will fail if there is any codegen drift.

---

## Common Development Commands

| Command | Action |
| :--- | :--- |
| `task gen` | Regenerate sqlc queries, Go server interfaces, and TypeScript API types |
| `task migrate` | Run goose migrations and River queue setup |
| `task seed` | Seed local demo academy, courses, and accounts |
| `task test` | Run all unit and integration tests (race detector enabled) |
| `task lint` | Run `golangci-lint` on backend and ESLint + TypeScript check on web |
# courses-platform
