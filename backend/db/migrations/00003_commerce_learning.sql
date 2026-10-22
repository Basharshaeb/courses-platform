-- +goose Up

-- Composite unique key on students so child tables can reference (student_id, school_id)
ALTER TABLE students ADD CONSTRAINT students_id_school_id_key UNIQUE (id, school_id);

-- Media assets: Bunny Stream videos, S3/R2 files and images.
CREATE TABLE media_assets (
  id             uuid PRIMARY KEY,
  school_id      uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  provider       text NOT NULL CHECK (provider IN ('bunny', 's3')),
  provider_ref   text NOT NULL,
  kind           text NOT NULL CHECK (kind IN ('video', 'file', 'image')),
  status         text NOT NULL DEFAULT 'uploading' CHECK (status IN ('uploading', 'processing', 'ready', 'failed')),
  filename       text NOT NULL,
  size_bytes     bigint,
  duration_sec   integer CHECK (duration_sec >= 0),
  thumbnail_url  text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, school_id)
);
CREATE INDEX media_assets_school_id_created_at_idx ON media_assets (school_id, created_at);
CREATE TRIGGER media_assets_updated_at BEFORE UPDATE ON media_assets FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Add foreign key constraint to lecture_blocks now that media_assets exists.
ALTER TABLE lecture_blocks
  ADD CONSTRAINT lecture_blocks_media_asset_fkey
  FOREIGN KEY (media_asset_id, school_id) REFERENCES media_assets (id, school_id) ON DELETE SET NULL;

-- Commerce: Products and Pricing Plans
CREATE TABLE products (
  id         uuid PRIMARY KEY,
  school_id  uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  kind       text NOT NULL CHECK (kind IN ('course', 'bundle')),
  ref_id     uuid NOT NULL,
  name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
  status     text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (school_id, kind, ref_id),
  UNIQUE (id, school_id)
);
CREATE INDEX products_school_id_idx ON products (school_id);
CREATE TRIGGER products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE pricing_plans (
  id            uuid PRIMARY KEY,
  school_id     uuid NOT NULL,
  product_id    uuid NOT NULL,
  type          text NOT NULL CHECK (type IN ('free', 'one_time')),
  amount        bigint NOT NULL DEFAULT 0 CHECK (amount >= 0),
  currency      char(3) NOT NULL DEFAULT 'SAR',
  tax_inclusive boolean NOT NULL DEFAULT true,
  active        boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (product_id, school_id) REFERENCES products (id, school_id) ON DELETE CASCADE,
  UNIQUE (id, school_id)
);
CREATE INDEX pricing_plans_product_id_idx ON pricing_plans (product_id);
CREATE TRIGGER pricing_plans_updated_at BEFORE UPDATE ON pricing_plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Orders and Payments
CREATE TABLE orders (
  id                 uuid PRIMARY KEY,
  school_id          uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  student_id         uuid NOT NULL,
  status             text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed', 'refunded', 'canceled')),
  subtotal           bigint NOT NULL CHECK (subtotal >= 0),
  discount           bigint NOT NULL DEFAULT 0 CHECK (discount >= 0),
  tax                bigint NOT NULL DEFAULT 0 CHECK (tax >= 0),
  total              bigint NOT NULL CHECK (total >= 0),
  currency           char(3) NOT NULL DEFAULT 'SAR',
  gateway            text NOT NULL CHECK (gateway IN ('free', 'moyasar', 'stripe')),
  gateway_order_ref  text,
  buyer_name         text NOT NULL,
  buyer_email        text NOT NULL CHECK (buyer_email = lower(buyer_email)),
  idempotency_key    text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (student_id, school_id) REFERENCES students (id, school_id) ON DELETE CASCADE,
  UNIQUE (id, school_id),
  UNIQUE (school_id, idempotency_key)
);
CREATE INDEX orders_school_id_created_at_idx ON orders (school_id, created_at);
CREATE INDEX orders_student_id_idx ON orders (student_id);
CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE order_items (
  id              uuid PRIMARY KEY,
  school_id       uuid NOT NULL,
  order_id        uuid NOT NULL,
  product_id      uuid NOT NULL,
  pricing_plan_id uuid NOT NULL,
  amount          bigint NOT NULL CHECK (amount >= 0),
  tax             bigint NOT NULL DEFAULT 0,
  created_at      timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (order_id, school_id) REFERENCES orders (id, school_id) ON DELETE CASCADE,
  FOREIGN KEY (product_id, school_id) REFERENCES products (id, school_id) ON DELETE RESTRICT,
  FOREIGN KEY (pricing_plan_id, school_id) REFERENCES pricing_plans (id, school_id) ON DELETE RESTRICT
);
CREATE INDEX order_items_order_id_idx ON order_items (order_id);

CREATE TABLE payments (
  id                 uuid PRIMARY KEY,
  school_id          uuid NOT NULL,
  order_id           uuid NOT NULL,
  gateway            text NOT NULL CHECK (gateway IN ('free', 'moyasar', 'stripe')),
  gateway_payment_id text,
  amount             bigint NOT NULL CHECK (amount >= 0),
  currency           char(3) NOT NULL DEFAULT 'SAR',
  status             text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'failed', 'refunded')),
  failure_code       text,
  raw_response       jsonb NOT NULL DEFAULT '{}',
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (order_id, school_id) REFERENCES orders (id, school_id) ON DELETE CASCADE,
  UNIQUE (id, school_id)
);
CREATE INDEX payments_order_id_idx ON payments (order_id);
CREATE TRIGGER payments_updated_at BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Payment Gateway Credentials per School
CREATE TABLE gateway_connections (
  id                    uuid PRIMARY KEY,
  school_id             uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  gateway               text NOT NULL CHECK (gateway IN ('moyasar', 'stripe')),
  mode                  text NOT NULL DEFAULT 'test' CHECK (mode IN ('test', 'live')),
  credentials_encrypted bytea NOT NULL,
  publishable_key       text NOT NULL DEFAULT '',
  status                text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  UNIQUE (school_id, gateway, mode)
);
CREATE TRIGGER gateway_connections_updated_at BEFORE UPDATE ON gateway_connections FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Inbound Webhooks (Idempotent processing, global)
CREATE TABLE inbound_webhook_events (
  id           uuid PRIMARY KEY,
  gateway      text NOT NULL,
  event_id     text NOT NULL,
  payload      jsonb NOT NULL,
  processed_at timestamptz,
  error        text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (gateway, event_id)
);
CREATE INDEX inbound_webhook_events_gateway_event_idx ON inbound_webhook_events (gateway, event_id);

-- Learning: Enrollments & Progress
CREATE TABLE enrollments (
  id           uuid PRIMARY KEY,
  school_id    uuid NOT NULL,
  student_id   uuid NOT NULL,
  course_id    uuid NOT NULL,
  source       text NOT NULL DEFAULT 'order' CHECK (source IN ('order', 'free', 'manual')),
  source_id    uuid,
  status       text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
  enrolled_at  timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz,
  progress_pct integer NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (student_id, school_id) REFERENCES students (id, school_id) ON DELETE CASCADE,
  FOREIGN KEY (course_id, school_id) REFERENCES courses (id, school_id) ON DELETE CASCADE,
  UNIQUE (school_id, student_id, course_id),
  UNIQUE (id, school_id)
);
CREATE INDEX enrollments_student_id_idx ON enrollments (student_id);
CREATE INDEX enrollments_course_id_idx ON enrollments (course_id);
CREATE TRIGGER enrollments_updated_at BEFORE UPDATE ON enrollments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lecture_progress (
  enrollment_id uuid NOT NULL,
  school_id     uuid NOT NULL,
  lecture_id    uuid NOT NULL,
  is_completed  boolean NOT NULL DEFAULT false,
  position_sec  integer NOT NULL DEFAULT 0 CHECK (position_sec >= 0),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (enrollment_id, lecture_id),
  FOREIGN KEY (enrollment_id, school_id) REFERENCES enrollments (id, school_id) ON DELETE CASCADE,
  FOREIGN KEY (lecture_id, school_id) REFERENCES lectures (id, school_id) ON DELETE CASCADE
);
CREATE INDEX lecture_progress_lecture_id_idx ON lecture_progress (lecture_id);

-- Enable RLS on all tenant tables
ALTER TABLE media_assets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON media_assets
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE products ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON products
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE pricing_plans ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON pricing_plans
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON orders
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE order_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON order_items
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payments
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE gateway_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON gateway_connections
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE enrollments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON enrollments
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE lecture_progress ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON lecture_progress
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

-- +goose Down
ALTER TABLE lecture_blocks DROP CONSTRAINT IF EXISTS lecture_blocks_media_asset_fkey;
DROP TABLE IF EXISTS lecture_progress;
DROP TABLE IF EXISTS enrollments;
DROP TABLE IF EXISTS inbound_webhook_events;
DROP TABLE IF EXISTS gateway_connections;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS pricing_plans;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS media_assets;
