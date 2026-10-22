-- Runs once when the local Postgres volume is created.
-- app_rw: privileges granted by migrations. teatchable_app: login used by api/worker (RLS enforced).
CREATE ROLE app_rw NOLOGIN;
CREATE ROLE teatchable_app LOGIN PASSWORD 'teatchable_app' IN ROLE app_rw;
