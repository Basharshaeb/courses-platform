# Teatchable Backend (`backend/`)

This directory contains the Go modular monolith powering the Teatchable platform. It serves the HTTP API (`cmd/api`), executes asynchronous background tasks via River (`cmd/worker`), and handles database migrations and seed data (`cmd/migrate`).

---

## 1. Architectural Structure

```
backend/
├── cmd/
│   ├── api/             # HTTP server entrypoint (chi router, middleware, graceful shutdown)
│   ├── worker/          # River background job worker
│   └── migrate/         # Goose migrations runner and test seeder
├── db/
│   ├── migrations/      # Goose SQL migration files (numbered sequentially)
│   └── queries/         # SQL queries compiled by sqlc into type-safe Go code
├── internal/
│   ├── api/             # HTTP transport adapters implementing OpenAPI generated interfaces
│   │   ├── gen/         # Generated OpenAPI server & types (oapi-codegen) - DO NOT EDIT
│   │   ├── handlers.go  # Core route handlers (auth, schools, TLS validation)
│   │   ├── catalog_handlers.go # Course, section, lecture management handlers
│   │   └── server.go    # Chi router setup, OpenAPI validation, RFC 9457 error wiring
│   ├── modules/         # Domain business logic (pure Go, zero HTTP dependencies)
│   │   ├── catalog/     # Courses, sections, lectures, content blocks
│   │   ├── commerce/    # Products, pricing plans, orders, Moyasar & Stripe gateways
│   │   ├── enrollment/  # Student enrollments, lecture progress tracking
│   │   ├── identity/    # Users, hashed sessions, password verification, permissions
│   │   ├── media/       # Bunny Stream CDN video uploads, signed tokens
│   │   └── school/      # Multi-tenant schools, domains, customizations
│   ├── platform/        # Shared infrastructure (database pool, logger, apperr, config)
│   │   ├── apperr/      # Standard error types mapped to RFC 9457 error codes
│   │   ├── db/          # pgxpool connection manager, tenant tx wrapper
│   │   ├── httpx/       # RFC 9457 problem writers, security middleware, client info
│   │   └── tenant/      # Tenant context resolution & surface detection
│   └── store/           # Generated sqlc database repository - DO NOT EDIT
└── Taskfile.yml         # Task runner definitions for local development
```

---

## 2. Mandatory Code Standards & Best Practices

### A. Modular Boundaries & Dependency Direction
1. **Modules must remain transport-agnostic**:
   - Files in `internal/modules/*` must **NEVER** import `net/http`, `github.com/go-chi/chi`, or `internal/api/gen`.
   - Domain services accept standard Go types (`context.Context`, `uuid.UUID`, domain structs) and return domain results or `apperr` errors.
2. **Cross-Module Communication**:
   - Cross-module calls must go through exported Service interfaces (e.g., `commerce.Service` calling `enrollment.Service.GrantEnrollment`).
   - Direct cross-table SQL joins across module domains are **strictly forbidden** to prevent tight coupling.
   - For eventual consistency or heavy workloads, use River background jobs.

### B. Multi-Tenancy & Row-Level Security (RLS)
The database enforces strict tenant isolation using PostgreSQL Row-Level Security:
1. **Always use `db.WithTenantTx`**:
   ```go
   err := db.WithTenantTx(ctx, s.pool, schoolID, func(tx pgx.Tx, q *store.Queries) error {
       // All queries executed within this transaction inherit app.school_id
       return q.UpdateCourse(ctx, arg)
   })
   ```
2. **Double Defense**:
   - Even though PostgreSQL RLS prevents selecting or modifying rows belonging to another `school_id`, every SQL query in `db/queries/*.sql` **MUST** also explicitly include `WHERE school_id = $1`.
3. **Composite Foreign Keys**:
   - Foreign key checks in PostgreSQL bypass RLS. Therefore, child tables must always reference parent tables via composite keys:
   ```sql
   FOREIGN KEY (course_id, school_id) REFERENCES courses (id, school_id) ON DELETE CASCADE
   ```

### C. Error Handling & RFC 9457
Never expose raw database errors or internal stack traces to HTTP clients.
- Use the typed errors in `internal/platform/apperr`:
  - `apperr.NotFound("course_not_found", "The requested course does not exist")`
  - `apperr.BadRequest("slug_taken", "This course slug is already in use")`
  - `apperr.Forbidden("enrollment_required", "You must be enrolled to access this lecture")`
  - `apperr.Unauthorized("session_expired", "Please log in again")`
- The HTTP adapter (`internal/api`) intercepts these errors and renders a compliant `application/problem+json` response with request tracing IDs.

### D. Financial Data & Currency
- **Never use floating point numbers (`float32`, `float64`) for money**.
- All amounts must be stored as 64-bit integers (`int64` / `bigint`) representing the currency's smallest minor unit (e.g., **Halalas** for SAR, **Cents** for USD).
- 100.00 SAR = `10000` Halalas.

### E. Concurrency & Context Propagation
1. Always pass `ctx context.Context` as the first argument to functions performing I/O.
2. Respect context cancellation: check `ctx.Done()` in long-running loops.
3. When spawning background goroutines, **never** pass the request `ctx` directly (it will be cancelled when the HTTP request finishes). Use `context.WithoutCancel(ctx)` or pass a background context to River jobs.

### F. Structured Logging (`log/slog`)
Always use structured logging with contextual attributes:
```go
slog.InfoContext(ctx, "order created successfully",
    "school_id", schoolID,
    "order_id", order.ID,
    "amount_minor", order.AmountMinor,
    "currency", order.Currency,
)
```
Avoid `fmt.Printf`, `log.Println`, or unformatted log statements.

---

## 3. Database Workflows (`sqlc` & `goose`)

### Adding a New Migration
1. Create a new SQL file in `db/migrations/`:
   ```bash
   # Format: 0000X_description.sql
   ```
2. Include both `-- +goose Up` and `-- +goose Down` blocks.
3. Every tenant table must include:
   ```sql
   school_id uuid NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
   ALTER TABLE <table_name> ENABLE ROW LEVEL SECURITY;
   ALTER TABLE <table_name> FORCE ROW LEVEL SECURITY;
   CREATE POLICY <table_name>_school_isolation ON <table_name>
       USING (school_id = current_setting('app.school_id', true)::uuid)
       WITH CHECK (school_id = current_setting('app.school_id', true)::uuid);
   ```

### Writing SQL Queries (`sqlc`)
1. Place queries in `db/queries/{domain}.sql`.
2. Annotate each query with `-- name: OperationName :one` (or `:many`, `:exec`, `:execrows`).
3. Run code generation:
   ```bash
   task gen
   ```
4. Never manually edit files in `internal/store/`.

---

## 4. Testing Guidelines

1. **Table-Driven Tests**: Use idiomatic Go table-driven tests for domain logic and validation functions.
2. **Integration Tests with Real Postgres**:
   - We test against real PostgreSQL to verify RLS policies and transaction isolation.
   - Run tests with race detection:
   ```bash
   task test
   # Or: go test -v -race ./...
   ```
3. **No Mocks for Database**:
   - Instead of mocking `store.Querier`, execute real transactions against an isolated test database and roll back at the end of each test.

---

## 5. Useful Commands

| Task | Command |
| :--- | :--- |
| **Run API Server** | `task api` |
| **Run Background Worker** | `task worker` |
| **Run Database Migrations** | `task migrate` |
| **Generate Code (`sqlc` & `oapi-codegen`)** | `task gen` |
| **Run Linter** | `golangci-lint run ./...` |
| **Format Code** | `gofmt -w . && goimports -w .` |
