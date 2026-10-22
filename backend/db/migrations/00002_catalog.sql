-- +goose Up

-- Catalog: courses → sections → lectures → content blocks.
--
-- Child tables reference their parent with composite foreign keys that include school_id.
-- Foreign key checks bypass row level security, so without school_id in the key a tenant could
-- attach rows to another school's course by guessing its id.

CREATE TABLE courses (
  id                uuid PRIMARY KEY,
  school_id         uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
  slug              text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
  title             text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  subtitle          text NOT NULL DEFAULT '',
  description       text NOT NULL DEFAULT '',
  cover_key         text,
  locale            text NOT NULL DEFAULT 'ar' CHECK (locale IN ('ar', 'en')),
  status            text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
  author_account_id uuid REFERENCES accounts (id) ON DELETE SET NULL,
  settings          jsonb NOT NULL DEFAULT '{}',
  published_at      timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (school_id, slug),
  UNIQUE (id, school_id)
);
CREATE INDEX courses_school_id_created_at_idx ON courses (school_id, created_at);
CREATE TRIGGER courses_updated_at BEFORE UPDATE ON courses FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_sections (
  id         uuid PRIMARY KEY,
  school_id  uuid NOT NULL,
  course_id  uuid NOT NULL,
  position   integer NOT NULL CHECK (position >= 0),
  title      text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  drip_type  text NOT NULL DEFAULT 'none' CHECK (drip_type IN ('none', 'days', 'date')),
  drip_days  integer CHECK (drip_days >= 0),
  drip_date  timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (course_id, school_id) REFERENCES courses (id, school_id) ON DELETE CASCADE,
  UNIQUE (id, course_id, school_id)
);
CREATE INDEX course_sections_course_id_position_idx ON course_sections (course_id, position);
CREATE TRIGGER course_sections_updated_at BEFORE UPDATE ON course_sections FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lectures (
  id              uuid PRIMARY KEY,
  school_id       uuid NOT NULL,
  course_id       uuid NOT NULL,
  section_id      uuid NOT NULL,
  position        integer NOT NULL CHECK (position >= 0),
  title           text NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
  is_free_preview boolean NOT NULL DEFAULT false,
  status          text NOT NULL DEFAULT 'published' CHECK (status IN ('draft', 'published')),
  duration_sec    integer CHECK (duration_sec >= 0),
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  -- The section must belong to the same course (and school); moving a lecture keeps that true.
  FOREIGN KEY (section_id, course_id, school_id) REFERENCES course_sections (id, course_id, school_id) ON DELETE CASCADE,
  UNIQUE (id, school_id)
);
CREATE INDEX lectures_course_id_idx ON lectures (course_id);
CREATE INDEX lectures_section_id_position_idx ON lectures (section_id, position);
CREATE TRIGGER lectures_updated_at BEFORE UPDATE ON lectures FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lecture_blocks (
  id             uuid PRIMARY KEY,
  school_id      uuid NOT NULL,
  lecture_id     uuid NOT NULL,
  position       integer NOT NULL CHECK (position >= 0),
  kind           text NOT NULL CHECK (kind IN ('text', 'pdf', 'video', 'embed')),
  data           jsonb NOT NULL DEFAULT '{}',
  -- Points to media_assets(id) once the media module lands (Bunny Stream / S3 uploads).
  media_asset_id uuid,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (lecture_id, school_id) REFERENCES lectures (id, school_id) ON DELETE CASCADE
);
CREATE INDEX lecture_blocks_lecture_id_position_idx ON lecture_blocks (lecture_id, position);
CREATE TRIGGER lecture_blocks_updated_at BEFORE UPDATE ON lecture_blocks FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE courses ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON courses
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE course_sections ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON course_sections
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE lectures ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON lectures
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

ALTER TABLE lecture_blocks ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON lecture_blocks
  USING (school_id = app_current_school_id())
  WITH CHECK (school_id = app_current_school_id());

-- +goose Down
DROP TABLE IF EXISTS lecture_blocks;
DROP TABLE IF EXISTS lectures;
DROP TABLE IF EXISTS course_sections;
DROP TABLE IF EXISTS courses;
