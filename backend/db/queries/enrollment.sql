-- ---- Enrollments ----

-- name: CreateEnrollment :one
INSERT INTO enrollments (id, school_id, student_id, course_id, source, source_id, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (school_id, student_id, course_id) DO UPDATE SET
  status     = 'active',
  updated_at = now()
RETURNING *;

-- name: GetEnrollment :one
SELECT * FROM enrollments WHERE school_id = $1 AND id = $2;

-- name: GetEnrollmentByStudentAndCourse :one
SELECT * FROM enrollments
WHERE school_id = $1 AND student_id = $2 AND course_id = $3;

-- name: ListStudentEnrollments :many
SELECT
  sqlc.embed(e),
  sqlc.embed(c)
FROM enrollments e
JOIN courses c ON c.id = e.course_id AND c.school_id = e.school_id
WHERE e.school_id = $1 AND e.student_id = $2 AND e.status = 'active'
ORDER BY e.enrolled_at DESC;

-- name: ListCourseEnrollments :many
SELECT * FROM enrollments
WHERE school_id = $1 AND course_id = $2
ORDER BY enrolled_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateEnrollmentProgress :one
UPDATE enrollments SET
  progress_pct = sqlc.arg(progress_pct),
  completed_at = CASE
    WHEN sqlc.arg(progress_pct)::int = 100 AND completed_at IS NULL THEN now()
    ELSE completed_at
  END
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- ---- Lecture Progress ----

-- name: UpsertLectureProgress :one
INSERT INTO lecture_progress (enrollment_id, school_id, lecture_id, is_completed, position_sec)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (enrollment_id, lecture_id) DO UPDATE SET
  is_completed = EXCLUDED.is_completed,
  position_sec = EXCLUDED.position_sec,
  updated_at   = now()
RETURNING *;

-- name: GetLectureProgress :one
SELECT * FROM lecture_progress
WHERE school_id = $1 AND enrollment_id = $2 AND lecture_id = $3;

-- name: ListLectureProgressByEnrollment :many
SELECT * FROM lecture_progress
WHERE school_id = $1 AND enrollment_id = $2;
