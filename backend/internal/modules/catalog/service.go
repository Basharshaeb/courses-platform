// Package catalog owns courses and their curriculum: sections, lectures and content blocks.
//
// Callers authorize first (school membership and role, see school.Service.Authorize); every
// method here runs in a tenant transaction for the given school.
package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

var (
	errCourseNotFound  = apperr.NotFound("course_not_found", "course not found")
	errSectionNotFound = apperr.NotFound("section_not_found", "section not found")
	errLectureNotFound = apperr.NotFound("lecture_not_found", "lecture not found")
	errBlockNotFound   = apperr.NotFound("block_not_found", "block not found")
	errSlugTaken       = apperr.Conflict("slug_taken", "another course already uses this URL name")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Lecture is a lecture with the number of its content blocks.
type Lecture struct {
	store.Lecture
	BlockCount int32
}

// Section is a section with its lectures in display order.
type Section struct {
	store.CourseSection
	Lectures []Lecture
}

type CourseDetail struct {
	Course   store.Course
	Sections []Section
}

func (s *Service) tx(ctx context.Context, schoolID uuid.UUID, fn func(q *store.Queries) error) error {
	return db.WithTenantTx(ctx, s.pool, schoolID, func(q *store.Queries, _ pgx.Tx) error { return fn(q) })
}

// notFound maps "no rows" to e and passes other errors through.
func notFound(err, e error) error {
	if db.IsNotFound(err) {
		return e
	}
	return err
}

func cleanTitle(title string) (string, error) {
	t := strings.TrimSpace(title)
	if t == "" || len([]rune(t)) > 200 {
		return "", apperr.BadRequest("invalid_title", "title must be 1-200 characters")
	}
	return t, nil
}

func cleanLocale(l string) string {
	if l == "en" {
		return "en"
	}
	return "ar"
}

// ---- Courses ----

type CreateCourseInput struct {
	Title, Slug, Subtitle, Description, Locale string
	AuthorID                                   uuid.UUID
}

func (s *Service) ListCourses(ctx context.Context, schoolID uuid.UUID) ([]store.ListCoursesRow, error) {
	var out []store.ListCoursesRow
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		var err error
		out, err = q.ListCourses(ctx, schoolID)
		return err
	})
	return out, err
}

// CreateCourse creates a draft course. Without a slug one is derived from the title.
func (s *Service) CreateCourse(ctx context.Context, schoolID uuid.UUID, in CreateCourseInput) (store.Course, error) {
	title, err := cleanTitle(in.Title)
	if err != nil {
		return store.Course{}, err
	}
	slug := strings.TrimSpace(in.Slug)
	if slug != "" {
		if err := ValidateSlug(slug); err != nil {
			return store.Course{}, err
		}
	}
	var c store.Course
	err = s.tx(ctx, schoolID, func(q *store.Queries) error {
		if slug == "" {
			if slug, err = uniqueSlug(ctx, q, schoolID, Slugify(title)); err != nil {
				return err
			}
		}
		var author *uuid.UUID
		if in.AuthorID != uuid.Nil {
			author = &in.AuthorID
		}
		c, err = q.CreateCourse(ctx, store.CreateCourseParams{
			ID:              uuid.Must(uuid.NewV7()),
			SchoolID:        schoolID,
			Slug:            slug,
			Title:           title,
			Subtitle:        strings.TrimSpace(in.Subtitle),
			Description:     strings.TrimSpace(in.Description),
			Locale:          cleanLocale(in.Locale),
			AuthorAccountID: author,
		})
		if err != nil {
			if isSlugConflict(err) {
				return errSlugTaken
			}
			return fmt.Errorf("catalog: create course: %w", err)
		}
		return nil
	})
	return c, err
}

// GetCourse returns a course with its full curriculum.
func (s *Service) GetCourse(ctx context.Context, schoolID, courseID uuid.UUID) (CourseDetail, error) {
	var out CourseDetail
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		c, err := q.GetCourse(ctx, store.GetCourseParams{SchoolID: schoolID, ID: courseID})
		if err != nil {
			return notFound(err, errCourseNotFound)
		}
		sections, err := loadCurriculum(ctx, q, schoolID, courseID)
		if err != nil {
			return err
		}
		out = CourseDetail{Course: c, Sections: sections}
		return nil
	})
	return out, err
}

type UpdateCourseInput struct {
	Title, Slug, Subtitle, Description, Locale *string
}

func (s *Service) UpdateCourse(ctx context.Context, schoolID, courseID uuid.UUID, in UpdateCourseInput) (store.Course, error) {
	p := store.UpdateCourseParams{SchoolID: schoolID, ID: courseID}
	if in.Title != nil {
		t, err := cleanTitle(*in.Title)
		if err != nil {
			return store.Course{}, err
		}
		p.Title = &t
	}
	if in.Slug != nil {
		slug := strings.TrimSpace(*in.Slug)
		if err := ValidateSlug(slug); err != nil {
			return store.Course{}, err
		}
		p.Slug = &slug
	}
	if in.Subtitle != nil {
		v := strings.TrimSpace(*in.Subtitle)
		p.Subtitle = &v
	}
	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		p.Description = &v
	}
	if in.Locale != nil {
		v := cleanLocale(*in.Locale)
		p.Locale = &v
	}
	var c store.Course
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		var err error
		c, err = q.UpdateCourse(ctx, p)
		if isSlugConflict(err) {
			return errSlugTaken
		}
		return notFound(err, errCourseNotFound)
	})
	return c, err
}

// DeleteCourse removes a course with its sections, lectures and blocks.
func (s *Service) DeleteCourse(ctx context.Context, schoolID, courseID uuid.UUID) error {
	return s.tx(ctx, schoolID, func(q *store.Queries) error {
		n, err := q.DeleteCourse(ctx, store.DeleteCourseParams{SchoolID: schoolID, ID: courseID})
		if err != nil {
			return err
		}
		if n == 0 {
			return errCourseNotFound
		}
		return nil
	})
}

// PublishCourse makes a course visible on the storefront. It needs a published lecture.
func (s *Service) PublishCourse(ctx context.Context, schoolID, courseID uuid.UUID) (store.Course, error) {
	var c store.Course
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		if _, err := q.GetCourse(ctx, store.GetCourseParams{SchoolID: schoolID, ID: courseID}); err != nil {
			return notFound(err, errCourseNotFound)
		}
		n, err := q.CountPublishedLectures(ctx, store.CountPublishedLecturesParams{SchoolID: schoolID, CourseID: courseID})
		if err != nil {
			return err
		}
		if n == 0 {
			return apperr.Conflict("course_empty", "add at least one published lecture before publishing")
		}
		c, err = q.SetCourseStatus(ctx, store.SetCourseStatusParams{Status: StatusPublished, SchoolID: schoolID, ID: courseID})
		return err
	})
	return c, err
}

// UnpublishCourse moves a course back to draft.
func (s *Service) UnpublishCourse(ctx context.Context, schoolID, courseID uuid.UUID) (store.Course, error) {
	var c store.Course
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		var err error
		c, err = q.SetCourseStatus(ctx, store.SetCourseStatusParams{Status: StatusDraft, SchoolID: schoolID, ID: courseID})
		return notFound(err, errCourseNotFound)
	})
	return c, err
}

// ---- Curriculum ----

func loadCurriculum(ctx context.Context, q *store.Queries, schoolID, courseID uuid.UUID) ([]Section, error) {
	secs, err := q.ListSections(ctx, store.ListSectionsParams{SchoolID: schoolID, CourseID: courseID})
	if err != nil {
		return nil, err
	}
	lecs, err := q.ListLectures(ctx, store.ListLecturesParams{SchoolID: schoolID, CourseID: courseID})
	if err != nil {
		return nil, err
	}
	bySection := make(map[uuid.UUID][]Lecture, len(secs))
	for _, l := range lecs {
		bySection[l.Lecture.SectionID] = append(bySection[l.Lecture.SectionID], Lecture{Lecture: l.Lecture, BlockCount: l.BlockCount})
	}
	out := make([]Section, 0, len(secs))
	for _, sec := range secs {
		ls := bySection[sec.ID]
		if ls == nil {
			ls = []Lecture{}
		}
		out = append(out, Section{CourseSection: sec, Lectures: ls})
	}
	return out, nil
}

// SectionOrder is one section and its lectures, in display order.
type SectionOrder struct {
	ID         uuid.UUID
	LectureIDs []uuid.UUID
}

var errInvalidOrder = apperr.BadRequest("invalid_order", "the order must list every section and lecture of the course exactly once")

// ReorderCurriculum sets section order and lecture order; lectures may move between sections.
func (s *Service) ReorderCurriculum(ctx context.Context, schoolID, courseID uuid.UUID, order []SectionOrder) ([]Section, error) {
	var out []Section
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		if _, err := q.GetCourse(ctx, store.GetCourseParams{SchoolID: schoolID, ID: courseID}); err != nil {
			return notFound(err, errCourseNotFound)
		}
		current, err := loadCurriculum(ctx, q, schoolID, courseID)
		if err != nil {
			return err
		}
		sectionIDs, lectureIDs, lectureSections, positions, err := validateOrder(current, order)
		if err != nil {
			return err
		}
		if _, err := q.SetSectionPositions(ctx, store.SetSectionPositionsParams{SchoolID: schoolID, CourseID: courseID, Ids: sectionIDs}); err != nil {
			return fmt.Errorf("catalog: reorder sections: %w", err)
		}
		if len(lectureIDs) > 0 {
			if _, err := q.MoveLectures(ctx, store.MoveLecturesParams{
				SchoolID: schoolID, CourseID: courseID, Ids: lectureIDs, SectionIds: lectureSections, Positions: positions,
			}); err != nil {
				return fmt.Errorf("catalog: reorder lectures: %w", err)
			}
		}
		out, err = loadCurriculum(ctx, q, schoolID, courseID)
		return err
	})
	return out, err
}

// validateOrder checks that order is a permutation of the current curriculum and flattens it.
func validateOrder(current []Section, order []SectionOrder) (sectionIDs, lectureIDs, lectureSections []uuid.UUID, positions []int32, err error) {
	knownSections := make(map[uuid.UUID]bool, len(current))
	knownLectures := make(map[uuid.UUID]bool)
	for _, sec := range current {
		knownSections[sec.ID] = true
		for _, l := range sec.Lectures {
			knownLectures[l.ID] = true
		}
	}
	if len(order) != len(knownSections) {
		return nil, nil, nil, nil, errInvalidOrder
	}
	seen := make(map[uuid.UUID]bool, len(knownSections)+len(knownLectures))
	for _, so := range order {
		if !knownSections[so.ID] || seen[so.ID] {
			return nil, nil, nil, nil, errInvalidOrder
		}
		seen[so.ID] = true
		sectionIDs = append(sectionIDs, so.ID)
		for i, lid := range so.LectureIDs {
			if !knownLectures[lid] || seen[lid] {
				return nil, nil, nil, nil, errInvalidOrder
			}
			seen[lid] = true
			lectureIDs = append(lectureIDs, lid)
			lectureSections = append(lectureSections, so.ID)
			positions = append(positions, int32(i)) //nolint:gosec // bounded by the request schema (maxItems)
		}
	}
	if len(lectureIDs) != len(knownLectures) {
		return nil, nil, nil, nil, errInvalidOrder
	}
	return sectionIDs, lectureIDs, lectureSections, positions, nil
}

// ---- Sections ----

func (s *Service) CreateSection(ctx context.Context, schoolID, courseID uuid.UUID, title string) (Section, error) {
	t, err := cleanTitle(title)
	if err != nil {
		return Section{}, err
	}
	var out Section
	err = s.tx(ctx, schoolID, func(q *store.Queries) error {
		if _, err := q.GetCourse(ctx, store.GetCourseParams{SchoolID: schoolID, ID: courseID}); err != nil {
			return notFound(err, errCourseNotFound)
		}
		sec, err := q.CreateSection(ctx, store.CreateSectionParams{
			ID: uuid.Must(uuid.NewV7()), SchoolID: schoolID, CourseID: courseID, Title: t,
		})
		if err != nil {
			return fmt.Errorf("catalog: create section: %w", err)
		}
		out = Section{CourseSection: sec, Lectures: []Lecture{}}
		return nil
	})
	return out, err
}

func (s *Service) UpdateSection(ctx context.Context, schoolID, courseID, sectionID uuid.UUID, title string) (Section, error) {
	t, err := cleanTitle(title)
	if err != nil {
		return Section{}, err
	}
	var out Section
	err = s.tx(ctx, schoolID, func(q *store.Queries) error {
		if _, err := q.UpdateSection(ctx, store.UpdateSectionParams{Title: t, SchoolID: schoolID, CourseID: courseID, ID: sectionID}); err != nil {
			return notFound(err, errSectionNotFound)
		}
		secs, err := loadCurriculum(ctx, q, schoolID, courseID)
		if err != nil {
			return err
		}
		for _, sec := range secs {
			if sec.ID == sectionID {
				out = sec
			}
		}
		return nil
	})
	return out, err
}

// DeleteSection removes a section with its lectures.
func (s *Service) DeleteSection(ctx context.Context, schoolID, courseID, sectionID uuid.UUID) error {
	return s.tx(ctx, schoolID, func(q *store.Queries) error {
		n, err := q.DeleteSection(ctx, store.DeleteSectionParams{SchoolID: schoolID, CourseID: courseID, ID: sectionID})
		if err != nil {
			return err
		}
		if n == 0 {
			return errSectionNotFound
		}
		return nil
	})
}

// ---- Lectures ----

func (s *Service) CreateLecture(ctx context.Context, schoolID, courseID, sectionID uuid.UUID, title string) (Lecture, error) {
	t, err := cleanTitle(title)
	if err != nil {
		return Lecture{}, err
	}
	var out Lecture
	err = s.tx(ctx, schoolID, func(q *store.Queries) error {
		if _, err := q.GetSection(ctx, store.GetSectionParams{SchoolID: schoolID, CourseID: courseID, ID: sectionID}); err != nil {
			return notFound(err, errSectionNotFound)
		}
		l, err := q.CreateLecture(ctx, store.CreateLectureParams{
			ID: uuid.Must(uuid.NewV7()), SchoolID: schoolID, CourseID: courseID, SectionID: sectionID, Title: t,
		})
		if err != nil {
			return fmt.Errorf("catalog: create lecture: %w", err)
		}
		out = Lecture{Lecture: l}
		return nil
	})
	return out, err
}

// LectureDetail is a lecture with its content blocks in display order.
type LectureDetail struct {
	Lecture Lecture
	Blocks  []Block
}

func (s *Service) GetLecture(ctx context.Context, schoolID, courseID, lectureID uuid.UUID) (LectureDetail, error) {
	var out LectureDetail
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		l, err := q.GetLecture(ctx, store.GetLectureParams{SchoolID: schoolID, CourseID: courseID, ID: lectureID})
		if err != nil {
			return notFound(err, errLectureNotFound)
		}
		blocks, err := listBlocks(ctx, q, schoolID, lectureID)
		if err != nil {
			return err
		}
		out = LectureDetail{Lecture: Lecture{Lecture: l, BlockCount: int32(len(blocks))}, Blocks: blocks} //nolint:gosec // block count is small
		return nil
	})
	return out, err
}

type UpdateLectureInput struct {
	Title         *string
	IsFreePreview *bool
	Status        *string
}

func (s *Service) UpdateLecture(ctx context.Context, schoolID, courseID, lectureID uuid.UUID, in UpdateLectureInput) (Lecture, error) {
	p := store.UpdateLectureParams{SchoolID: schoolID, CourseID: courseID, ID: lectureID, IsFreePreview: in.IsFreePreview}
	if in.Title != nil {
		t, err := cleanTitle(*in.Title)
		if err != nil {
			return Lecture{}, err
		}
		p.Title = &t
	}
	if in.Status != nil {
		if *in.Status != StatusDraft && *in.Status != StatusPublished {
			return Lecture{}, apperr.BadRequest("invalid_status", "status must be draft or published")
		}
		p.Status = in.Status
	}
	var out Lecture
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		l, err := q.UpdateLecture(ctx, p)
		if err != nil {
			return notFound(err, errLectureNotFound)
		}
		n, err := q.CountBlocks(ctx, store.CountBlocksParams{SchoolID: schoolID, LectureID: lectureID})
		if err != nil {
			return err
		}
		out = Lecture{Lecture: l, BlockCount: n}
		return nil
	})
	return out, err
}

func (s *Service) DeleteLecture(ctx context.Context, schoolID, courseID, lectureID uuid.UUID) error {
	return s.tx(ctx, schoolID, func(q *store.Queries) error {
		n, err := q.DeleteLecture(ctx, store.DeleteLectureParams{SchoolID: schoolID, CourseID: courseID, ID: lectureID})
		if err != nil {
			return err
		}
		if n == 0 {
			return errLectureNotFound
		}
		return nil
	})
}

// ---- Storefront ----

// PublicSection is a section with its published lectures.
type PublicSection struct {
	store.CourseSection
	Lectures []store.Lecture
}

type PublicCourse struct {
	Course   store.Course
	Sections []PublicSection
}

// ListPublished returns the published courses of a school, newest first.
func (s *Service) ListPublished(ctx context.Context, schoolID uuid.UUID) ([]store.ListPublishedCoursesRow, error) {
	var out []store.ListPublishedCoursesRow
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		var err error
		out, err = q.ListPublishedCourses(ctx, schoolID)
		return err
	})
	return out, err
}

// GetPublished returns a published course with its outline. Sections without published
// lectures are left out.
func (s *Service) GetPublished(ctx context.Context, schoolID uuid.UUID, slug string) (PublicCourse, error) {
	var out PublicCourse
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		c, err := q.GetPublishedCourseBySlug(ctx, store.GetPublishedCourseBySlugParams{SchoolID: schoolID, Slug: slug})
		if err != nil {
			return notFound(err, errCourseNotFound)
		}
		secs, err := q.ListSections(ctx, store.ListSectionsParams{SchoolID: schoolID, CourseID: c.ID})
		if err != nil {
			return err
		}
		lecs, err := q.ListPublishedLectures(ctx, store.ListPublishedLecturesParams{SchoolID: schoolID, CourseID: c.ID})
		if err != nil {
			return err
		}
		bySection := make(map[uuid.UUID][]store.Lecture, len(secs))
		for _, l := range lecs {
			bySection[l.SectionID] = append(bySection[l.SectionID], l)
		}
		out = PublicCourse{Course: c, Sections: []PublicSection{}}
		for _, sec := range secs {
			if ls := bySection[sec.ID]; len(ls) > 0 {
				out.Sections = append(out.Sections, PublicSection{CourseSection: sec, Lectures: ls})
			}
		}
		return nil
	})
	return out, err
}
