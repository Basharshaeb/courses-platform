-- ---- Courses ----

-- name: CreateCourse :one
INSERT INTO courses (id, school_id, slug, title, subtitle, description, locale, author_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListCourses :many
SELECT sqlc.embed(c),
  (SELECT count(*) FROM course_sections s WHERE s.school_id = c.school_id AND s.course_id = c.id)::int AS section_count,
  (SELECT count(*) FROM lectures l WHERE l.school_id = c.school_id AND l.course_id = c.id)::int AS lecture_count
FROM courses c
WHERE c.school_id = $1
ORDER BY c.created_at DESC, c.id DESC;

-- name: GetCourse :one
SELECT * FROM courses WHERE school_id = $1 AND id = $2;

-- name: CourseSlugExists :one
SELECT EXISTS (SELECT 1 FROM courses WHERE school_id = $1 AND slug = $2);

-- name: UpdateCourse :one
UPDATE courses SET
  title       = coalesce(sqlc.narg(title), title),
  slug        = coalesce(sqlc.narg(slug), slug),
  subtitle    = coalesce(sqlc.narg(subtitle), subtitle),
  description = coalesce(sqlc.narg(description), description),
  locale      = coalesce(sqlc.narg(locale), locale)
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: SetCourseStatus :one
UPDATE courses SET
  status       = sqlc.arg(status),
  published_at = CASE WHEN sqlc.arg(status)::text = 'published' THEN now() ELSE published_at END
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteCourse :execrows
DELETE FROM courses WHERE school_id = $1 AND id = $2;

-- ---- Sections ----

-- name: CreateSection :one
INSERT INTO course_sections (id, school_id, course_id, position, title)
SELECT sqlc.arg(id)::uuid, sqlc.arg(school_id)::uuid, sqlc.arg(course_id)::uuid,
  coalesce(max(s.position) + 1, 0), sqlc.arg(title)::text
FROM course_sections s
WHERE s.school_id = sqlc.arg(school_id) AND s.course_id = sqlc.arg(course_id)
RETURNING *;

-- name: ListSections :many
SELECT * FROM course_sections
WHERE school_id = $1 AND course_id = $2
ORDER BY position, id;

-- name: GetSection :one
SELECT * FROM course_sections WHERE school_id = $1 AND course_id = $2 AND id = $3;

-- name: UpdateSection :one
UPDATE course_sections SET title = sqlc.arg(title)
WHERE school_id = sqlc.arg(school_id) AND course_id = sqlc.arg(course_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSection :execrows
DELETE FROM course_sections WHERE school_id = $1 AND course_id = $2 AND id = $3;

-- name: SetSectionPositions :execrows
UPDATE course_sections AS s SET position = (o.ord - 1)::int
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE s.school_id = sqlc.arg(school_id) AND s.course_id = sqlc.arg(course_id) AND s.id = o.id;

-- ---- Lectures ----

-- name: CreateLecture :one
INSERT INTO lectures (id, school_id, course_id, section_id, position, title)
SELECT sqlc.arg(id)::uuid, sqlc.arg(school_id)::uuid, sqlc.arg(course_id)::uuid, sqlc.arg(section_id)::uuid,
  coalesce(max(l.position) + 1, 0), sqlc.arg(title)::text
FROM lectures l
WHERE l.school_id = sqlc.arg(school_id) AND l.section_id = sqlc.arg(section_id)
RETURNING *;

-- name: ListLectures :many
SELECT sqlc.embed(l),
  (SELECT count(*) FROM lecture_blocks b WHERE b.school_id = l.school_id AND b.lecture_id = l.id)::int AS block_count
FROM lectures l
WHERE l.school_id = $1 AND l.course_id = $2
ORDER BY l.position, l.id;

-- name: GetLecture :one
SELECT * FROM lectures WHERE school_id = $1 AND course_id = $2 AND id = $3;

-- name: UpdateLecture :one
UPDATE lectures SET
  title           = coalesce(sqlc.narg(title), title),
  is_free_preview = coalesce(sqlc.narg(is_free_preview), is_free_preview),
  status          = coalesce(sqlc.narg(status), status)
WHERE school_id = sqlc.arg(school_id) AND course_id = sqlc.arg(course_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteLecture :execrows
DELETE FROM lectures WHERE school_id = $1 AND course_id = $2 AND id = $3;

-- name: MoveLectures :execrows
UPDATE lectures AS l SET
  section_id = (sqlc.arg(section_ids)::uuid[])[o.ord::int],
  position   = (sqlc.arg(positions)::int[])[o.ord::int]
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE l.school_id = sqlc.arg(school_id) AND l.course_id = sqlc.arg(course_id) AND l.id = o.id;

-- name: CountPublishedLectures :one
SELECT count(*)::int FROM lectures WHERE school_id = $1 AND course_id = $2 AND status = 'published';

-- ---- Lecture blocks ----

-- name: CreateBlock :one
INSERT INTO lecture_blocks (id, school_id, lecture_id, position, kind, data, media_asset_id)
SELECT sqlc.arg(id)::uuid, sqlc.arg(school_id)::uuid, sqlc.arg(lecture_id)::uuid,
  coalesce(max(b.position) + 1, 0), sqlc.arg(kind)::text, sqlc.arg(data)::jsonb, sqlc.narg(media_asset_id)::uuid
FROM lecture_blocks b
WHERE b.school_id = sqlc.arg(school_id) AND b.lecture_id = sqlc.arg(lecture_id)
RETURNING *;

-- name: ListBlocks :many
SELECT * FROM lecture_blocks
WHERE school_id = $1 AND lecture_id = $2
ORDER BY position, id;

-- name: GetBlock :one
SELECT * FROM lecture_blocks WHERE school_id = $1 AND lecture_id = $2 AND id = $3;

-- name: CountBlocks :one
SELECT count(*)::int FROM lecture_blocks WHERE school_id = $1 AND lecture_id = $2;

-- name: UpdateBlockData :one
UPDATE lecture_blocks SET data = sqlc.arg(data)
WHERE school_id = sqlc.arg(school_id) AND lecture_id = sqlc.arg(lecture_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteBlock :execrows
DELETE FROM lecture_blocks WHERE school_id = $1 AND lecture_id = $2 AND id = $3;

-- name: SetBlockPositions :execrows
UPDATE lecture_blocks AS b SET position = (o.ord - 1)::int
FROM unnest(sqlc.arg(ids)::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE b.school_id = sqlc.arg(school_id) AND b.lecture_id = sqlc.arg(lecture_id) AND b.id = o.id;

-- ---- Storefront (published content only) ----

-- name: ListPublishedCourses :many
SELECT sqlc.embed(c),
  (SELECT count(*) FROM lectures l
   WHERE l.school_id = c.school_id AND l.course_id = c.id AND l.status = 'published')::int AS lecture_count
FROM courses c
WHERE c.school_id = $1 AND c.status = 'published'
ORDER BY c.published_at DESC, c.id DESC;

-- name: GetPublishedCourseBySlug :one
SELECT * FROM courses WHERE school_id = $1 AND slug = $2 AND status = 'published';

-- name: ListPublishedLectures :many
SELECT * FROM lectures
WHERE school_id = $1 AND course_id = $2 AND status = 'published'
ORDER BY position, id;
