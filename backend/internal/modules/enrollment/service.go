package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Enroll grants a student access to a course.
func (s *Service) Enroll(ctx context.Context, schoolID, studentID, courseID uuid.UUID, source string, sourceID *uuid.UUID) (store.Enrollment, error) {
	var enr store.Enrollment
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		enr, err = q.CreateEnrollment(ctx, store.CreateEnrollmentParams{
			ID:        uuid.Must(uuid.NewV7()),
			SchoolID:  schoolID,
			StudentID: studentID,
			CourseID:  courseID,
			Source:    source,
			SourceID:  sourceID,
			Status:    "active",
		})
		return err
	})
	if err != nil {
		return store.Enrollment{}, fmt.Errorf("enrollment: enroll: %w", err)
	}
	return enr, nil
}

// CheckAccess verifies that a student is actively enrolled in a course.
func (s *Service) CheckAccess(ctx context.Context, schoolID, studentID, courseID uuid.UUID) (*store.Enrollment, error) {
	var enr store.Enrollment
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		enr, err = q.GetEnrollmentByStudentAndCourse(ctx, store.GetEnrollmentByStudentAndCourseParams{
			SchoolID:  schoolID,
			StudentID: studentID,
			CourseID:  courseID,
		})
		return err
	})
	if db.IsNotFound(err) {
		return nil, apperr.Forbidden("not_enrolled")
	}
	if err != nil {
		return nil, fmt.Errorf("enrollment: check access: %w", err)
	}
	if enr.Status != "active" {
		return nil, apperr.Forbidden("enrollment_not_active")
	}
	return &enr, nil
}

// ListStudentCourses returns all active enrollments for a student.
func (s *Service) ListStudentCourses(ctx context.Context, schoolID, studentID uuid.UUID) ([]store.ListStudentEnrollmentsRow, error) {
	var rows []store.ListStudentEnrollmentsRow
	err := db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error {
		var err error
		rows, err = q.ListStudentEnrollments(ctx, store.ListStudentEnrollmentsParams{
			SchoolID:  schoolID,
			StudentID: studentID,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("enrollment: list student courses: %w", err)
	}
	return rows, nil
}

// UpdateProgress marks a lecture progress and recalculates course completion percentage.
func (s *Service) UpdateProgress(ctx context.Context, schoolID, studentID, courseID, lectureID uuid.UUID, isCompleted bool, positionSec int32) (int32, error) {
	enr, err := s.CheckAccess(ctx, schoolID, studentID, courseID)
	if err != nil {
		return 0, err
	}

	var progressPct int32
	err = db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, tx pgx.Tx) error {
		// 1. Upsert lecture progress
		_, err := q.UpsertLectureProgress(ctx, store.UpsertLectureProgressParams{
			EnrollmentID: enr.ID,
			SchoolID:     schoolID,
			LectureID:    lectureID,
			IsCompleted:  isCompleted,
			PositionSec:  positionSec,
		})
		if err != nil {
			return err
		}

		// 2. Count total published lectures in the course
		var totalLectures int64
		err = tx.QueryRow(ctx, `
			SELECT count(*) FROM lectures
			WHERE school_id = $1 AND course_id = $2 AND status = 'published'
		`, schoolID, courseID).Scan(&totalLectures)
		if err != nil {
			return err
		}

		if totalLectures == 0 {
			progressPct = 100
		} else {
			// Count completed lectures for this enrollment
			var completedCount int64
			err = tx.QueryRow(ctx, `
				SELECT count(*) FROM lecture_progress lp
				JOIN lectures l ON l.id = lp.lecture_id AND l.school_id = lp.school_id
				WHERE lp.enrollment_id = $1 AND lp.school_id = $2 AND l.course_id = $3 AND lp.is_completed = true
			`, enr.ID, schoolID, courseID).Scan(&completedCount)
			if err != nil {
				return err
			}
			progressPct = int32((completedCount * 100) / totalLectures)
			if progressPct > 100 {
				progressPct = 100
			}
		}

		// 3. Update enrollment progress
		_, err = q.UpdateEnrollmentProgress(ctx, store.UpdateEnrollmentProgressParams{
			SchoolID:    schoolID,
			ID:          enr.ID,
			ProgressPct: progressPct,
		})
		return err
	})

	if err != nil {
		return 0, fmt.Errorf("enrollment: update progress: %w", err)
	}

	return progressPct, nil
}
