# ADR 0001: Shared database multi-tenancy with Postgres RLS

- Status: Accepted
- Date: 2026-10-15

## Context
The platform hosts many independent schools (tenants). Each school owns courses, students,
orders and content. A data leak between schools is the worst-case failure for the product.
We expect thousands of small schools and a few large ones.

## Decision
- One shared Postgres database. Every tenant-owned table has `school_id uuid not null`.
- Row Level Security is enabled on every tenant table with policy
  `school_id = current_setting('app.school_id', true)::uuid`.
- The API connects with a role that does **not** have `BYPASSRLS`. Each request transaction runs
  `SELECT set_config('app.school_id', $1, true)` (transaction-local) before touching tenant data.
- Platform-wide operations (platform admin, some worker jobs, host resolution) use a separate
  role / explicit privileged path.
- Queries still filter by `school_id` explicitly (defense in depth); RLS is the safety net.
- Students are scoped per school (`unique(school_id, email)`), creators/staff are global `accounts`
  linked to schools through `school_members`.

## Consequences
- Simple ops (one DB, one migration path) and cheap per-tenant cost.
- Every tx must go through `db.WithTenantTx`; forgetting it returns zero rows instead of leaking data.
- Cross-tenant integration tests are mandatory for every new tenant table.
- A very large tenant can later be moved to its own database if needed.
