-- +goose Up

-- Application role. The API/worker log in as a user that is a member of app_rw.
-- app_rw does NOT have BYPASSRLS, so row level security always applies to it.
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_rw') THEN
    CREATE ROLE app_rw NOLOGIN;
  END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO app_rw;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_rw;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO app_rw;

-- Tenant id of the current transaction, set by db.WithTenantTx via set_config('app.school_id', ..., true).
CREATE FUNCTION app_current_school_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.school_id', true), '')::uuid
$$;

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END
$$;
-- +goose StatementEnd

-- Creators and school staff. Global (not tenant scoped).
CREATE TABLE accounts (
  id                uuid PRIMARY KEY,
  email             text NOT NULL UNIQUE CHECK (email = lower(email)),
  name              text NOT NULL,
  password_hash     text,
  locale            text NOT NULL DEFAULT 'ar' CHECK (locale IN ('ar', 'en')),
  email_verified_at timestamptz,
  mfa_enabled       boolean NOT NULL DEFAULT false,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER accounts_updated_at BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Tenant root.
CREATE TABLE schools (
  id               uuid PRIMARY KEY,
  owner_account_id uuid NOT NULL REFERENCES accounts (id),
  name             text NOT NULL,
  subdomain        text NOT NULL UNIQUE CHECK (subdomain ~ '^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$'),
  default_locale   text NOT NULL DEFAULT 'ar' CHECK (default_locale IN ('ar', 'en')),
  currency         char(3) NOT NULL DEFAULT 'SAR',
  timezone         text NOT NULL DEFAULT 'Asia/Riyadh',
  branding         jsonb NOT NULL DEFAULT '{}',
  settings         jsonb NOT NULL DEFAULT '{}',
  status           text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX schools_owner_account_id_idx ON schools (owner_account_id);
CREATE TRIGGER schools_updated_at BEFORE UPDATE ON schools FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE school_domains (
  id                 uuid PRIMARY KEY,
  school_id          uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  hostname           text NOT NULL UNIQUE CHECK (hostname = lower(hostname)),
  verification_token text NOT NULL,
  verified_at        timestamptz,
  tls_status         text NOT NULL DEFAULT 'pending' CHECK (tls_status IN ('pending', 'active', 'failed')),
  is_primary         boolean NOT NULL DEFAULT false,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX school_domains_school_id_idx ON school_domains (school_id);
CREATE TRIGGER school_domains_updated_at BEFORE UPDATE ON school_domains FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE school_members (
  school_id   uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  account_id  uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  role        text NOT NULL CHECK (role IN ('owner', 'admin', 'author', 'support', 'finance')),
  permissions jsonb NOT NULL DEFAULT '{}',
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (school_id, account_id)
);
CREATE INDEX school_members_account_id_idx ON school_members (account_id);

-- Students belong to exactly one school; the same email can exist in many schools.
CREATE TABLE students (
  id            uuid PRIMARY KEY,
  school_id     uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  email         text NOT NULL CHECK (email = lower(email)),
  name          text NOT NULL,
  password_hash text,
  locale        text NOT NULL DEFAULT 'ar' CHECK (locale IN ('ar', 'en')),
  status        text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked')),
  metadata      jsonb NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (school_id, email)
);
CREATE INDEX students_school_id_created_at_idx ON students (school_id, created_at);
CREATE TRIGGER students_updated_at BEFORE UPDATE ON students FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Opaque session tokens. Only the SHA-256 of the token is stored.
CREATE TABLE sessions (
  id           uuid PRIMARY KEY,
  token_hash   bytea NOT NULL UNIQUE,
  subject_type text NOT NULL CHECK (subject_type IN ('account', 'student', 'platform_admin')),
  subject_id   uuid NOT NULL,
  school_id    uuid REFERENCES schools (id) ON DELETE CASCADE,
  ip           text,
  user_agent   text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz NOT NULL,
  revoked_at   timestamptz,
  CHECK ((subject_type = 'student') = (school_id IS NOT NULL))
);
CREATE INDEX sessions_subject_idx ON sessions (subject_type, subject_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Row level security: every tenant table only exposes rows of app_current_school_id().
ALTER TABLE schools ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON schools
  USING (id = app_current_school_id())
  WITH CHECK (id = app_current_school_id());

ALTER TABLE school_domains ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON school_domains
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE school_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON school_members
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE students ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON students
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

-- Cross-tenant lookups that must work before a tenant is known. SECURITY DEFINER runs as the
-- migration owner (bypasses RLS); each function returns only the minimum needed.

-- Host -> school id. p_subdomain is set for {sub}.ROOT_DOMAIN hosts, p_hostname for custom domains.
CREATE FUNCTION resolve_school_id(p_subdomain text, p_hostname text) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT s.id
  FROM schools s
  WHERE s.status = 'active'
    AND (
      (p_subdomain IS NOT NULL AND s.subdomain = lower(p_subdomain))
      OR (p_hostname IS NOT NULL AND s.id = (
        SELECT d.school_id FROM school_domains d
        WHERE d.hostname = lower(p_hostname) AND d.verified_at IS NOT NULL
      ))
    )
  LIMIT 1
$$;

-- Schools an account belongs to (creator dashboard school switcher).
CREATE FUNCTION account_school_memberships(p_account_id uuid)
RETURNS TABLE (school_id uuid, name text, subdomain text, role text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT s.id, s.name, s.subdomain, m.role
  FROM school_members m
  JOIN schools s ON s.id = m.school_id
  WHERE m.account_id = p_account_id AND s.status <> 'deleted'
  ORDER BY m.created_at
$$;

REVOKE EXECUTE ON FUNCTION resolve_school_id(text, text) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION account_school_memberships(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION resolve_school_id(text, text) TO app_rw;
GRANT EXECUTE ON FUNCTION account_school_memberships(uuid) TO app_rw;

-- +goose Down
DROP FUNCTION IF EXISTS account_school_memberships(uuid);
DROP FUNCTION IF EXISTS resolve_school_id(text, text);
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS school_members;
DROP TABLE IF EXISTS school_domains;
DROP TABLE IF EXISTS schools;
DROP TABLE IF EXISTS accounts;
DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS app_current_school_id();
