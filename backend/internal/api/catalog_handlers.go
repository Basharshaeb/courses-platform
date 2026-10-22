package api

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/teatchable/backend/internal/api/gen"
	"github.com/teatchable/backend/internal/modules/catalog"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/tenant"
	"github.com/teatchable/backend/internal/store"
)

// authorizeSchool guards /admin/schools/{schoolId}/…: app host, account session, and a role in
// that school that grants p (non-members get 404, members without p get 403).
func (s *Server) authorizeSchool(ctx context.Context, schoolID uuid.UUID, p school.Permission) (identity.Principal, error) {
	if err := tenant.RequireSurface(ctx, tenant.SurfaceApp); err != nil {
		return identity.Principal{}, err
	}
	acc, err := identity.RequireAccount(ctx)
	if err != nil {
		return identity.Principal{}, err
	}
	if _, err := s.schools.Authorize(ctx, schoolID, acc.ID, p); err != nil {
		return identity.Principal{}, err
	}
	return acc, nil
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// ---- Courses ----

func (s *Server) ListCourses(ctx context.Context, req gen.ListCoursesRequestObject) (gen.ListCoursesResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesView); err != nil {
		return nil, err
	}
	rows, err := s.catalog.ListCourses(ctx, req.SchoolId)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListCourses200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.CourseSummary{
			Id:           r.Course.ID,
			Slug:         r.Course.Slug,
			Title:        r.Course.Title,
			Status:       gen.CourseStatus(r.Course.Status),
			SectionCount: int(r.SectionCount),
			LectureCount: int(r.LectureCount),
			PublishedAt:  utcPtr(r.Course.PublishedAt),
			UpdatedAt:    r.Course.UpdatedAt.UTC(),
		})
	}
	return out, nil
}

func (s *Server) CreateCourse(ctx context.Context, req gen.CreateCourseRequestObject) (gen.CreateCourseResponseObject, error) {
	acc, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit)
	if err != nil {
		return nil, err
	}
	b := req.Body
	c, err := s.catalog.CreateCourse(ctx, req.SchoolId, catalog.CreateCourseInput{
		Title:       b.Title,
		Slug:        deref(b.Slug),
		Subtitle:    deref(b.Subtitle),
		Description: deref(b.Description),
		Locale:      localeOrEmpty(b.Locale),
		AuthorID:    acc.ID,
	})
	if err != nil {
		return nil, err
	}
	return gen.CreateCourse201JSONResponse(toCourse(c)), nil
}

func (s *Server) GetCourse(ctx context.Context, req gen.GetCourseRequestObject) (gen.GetCourseResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesView); err != nil {
		return nil, err
	}
	d, err := s.catalog.GetCourse(ctx, req.SchoolId, req.CourseId)
	if err != nil {
		return nil, err
	}
	return gen.GetCourse200JSONResponse{Course: toCourse(d.Course), Sections: toSections(d.Sections)}, nil
}

func (s *Server) UpdateCourse(ctx context.Context, req gen.UpdateCourseRequestObject) (gen.UpdateCourseResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	b := req.Body
	in := catalog.UpdateCourseInput{Title: b.Title, Slug: b.Slug, Subtitle: b.Subtitle, Description: b.Description}
	if b.Locale != nil {
		l := string(*b.Locale)
		in.Locale = &l
	}
	c, err := s.catalog.UpdateCourse(ctx, req.SchoolId, req.CourseId, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateCourse200JSONResponse(toCourse(c)), nil
}

func (s *Server) DeleteCourse(ctx context.Context, req gen.DeleteCourseRequestObject) (gen.DeleteCourseResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	if err := s.catalog.DeleteCourse(ctx, req.SchoolId, req.CourseId); err != nil {
		return nil, err
	}
	return gen.DeleteCourse204Response{}, nil
}

func (s *Server) PublishCourse(ctx context.Context, req gen.PublishCourseRequestObject) (gen.PublishCourseResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	c, err := s.catalog.PublishCourse(ctx, req.SchoolId, req.CourseId)
	if err != nil {
		return nil, err
	}
	return gen.PublishCourse200JSONResponse(toCourse(c)), nil
}

func (s *Server) UnpublishCourse(ctx context.Context, req gen.UnpublishCourseRequestObject) (gen.UnpublishCourseResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	c, err := s.catalog.UnpublishCourse(ctx, req.SchoolId, req.CourseId)
	if err != nil {
		return nil, err
	}
	return gen.UnpublishCourse200JSONResponse(toCourse(c)), nil
}

func (s *Server) ReorderCurriculum(ctx context.Context, req gen.ReorderCurriculumRequestObject) (gen.ReorderCurriculumResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	order := make([]catalog.SectionOrder, 0, len(req.Body.Sections))
	for _, so := range req.Body.Sections {
		order = append(order, catalog.SectionOrder{ID: so.Id, LectureIDs: so.LectureIds})
	}
	secs, err := s.catalog.ReorderCurriculum(ctx, req.SchoolId, req.CourseId, order)
	if err != nil {
		return nil, err
	}
	return gen.ReorderCurriculum200JSONResponse{Sections: toSections(secs)}, nil
}

// ---- Sections ----

func (s *Server) CreateSection(ctx context.Context, req gen.CreateSectionRequestObject) (gen.CreateSectionResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	sec, err := s.catalog.CreateSection(ctx, req.SchoolId, req.CourseId, req.Body.Title)
	if err != nil {
		return nil, err
	}
	return gen.CreateSection201JSONResponse(toSection(sec)), nil
}

func (s *Server) UpdateSection(ctx context.Context, req gen.UpdateSectionRequestObject) (gen.UpdateSectionResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	sec, err := s.catalog.UpdateSection(ctx, req.SchoolId, req.CourseId, req.SectionId, req.Body.Title)
	if err != nil {
		return nil, err
	}
	return gen.UpdateSection200JSONResponse(toSection(sec)), nil
}

func (s *Server) DeleteSection(ctx context.Context, req gen.DeleteSectionRequestObject) (gen.DeleteSectionResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	if err := s.catalog.DeleteSection(ctx, req.SchoolId, req.CourseId, req.SectionId); err != nil {
		return nil, err
	}
	return gen.DeleteSection204Response{}, nil
}

// ---- Lectures ----

func (s *Server) CreateLecture(ctx context.Context, req gen.CreateLectureRequestObject) (gen.CreateLectureResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	l, err := s.catalog.CreateLecture(ctx, req.SchoolId, req.CourseId, req.SectionId, req.Body.Title)
	if err != nil {
		return nil, err
	}
	return gen.CreateLecture201JSONResponse(toLecture(l)), nil
}

func (s *Server) GetLecture(ctx context.Context, req gen.GetLectureRequestObject) (gen.GetLectureResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesView); err != nil {
		return nil, err
	}
	d, err := s.catalog.GetLecture(ctx, req.SchoolId, req.CourseId, req.LectureId)
	if err != nil {
		return nil, err
	}
	return gen.GetLecture200JSONResponse{Lecture: toLecture(d.Lecture), Blocks: toBlocks(d.Blocks)}, nil
}

func (s *Server) UpdateLecture(ctx context.Context, req gen.UpdateLectureRequestObject) (gen.UpdateLectureResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	b := req.Body
	in := catalog.UpdateLectureInput{Title: b.Title, IsFreePreview: b.IsFreePreview}
	if b.Status != nil {
		st := string(*b.Status)
		in.Status = &st
	}
	l, err := s.catalog.UpdateLecture(ctx, req.SchoolId, req.CourseId, req.LectureId, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateLecture200JSONResponse(toLecture(l)), nil
}

func (s *Server) DeleteLecture(ctx context.Context, req gen.DeleteLectureRequestObject) (gen.DeleteLectureResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	if err := s.catalog.DeleteLecture(ctx, req.SchoolId, req.CourseId, req.LectureId); err != nil {
		return nil, err
	}
	return gen.DeleteLecture204Response{}, nil
}

// ---- Blocks ----

func (s *Server) CreateBlock(ctx context.Context, req gen.CreateBlockRequestObject) (gen.CreateBlockResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	b, err := s.catalog.CreateBlock(ctx, req.SchoolId, req.CourseId, req.LectureId, string(req.Body.Kind), fromBlockData(req.Body.Data))
	if err != nil {
		return nil, err
	}
	return gen.CreateBlock201JSONResponse(toBlock(b)), nil
}

func (s *Server) UpdateBlock(ctx context.Context, req gen.UpdateBlockRequestObject) (gen.UpdateBlockResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	b, err := s.catalog.UpdateBlock(ctx, req.SchoolId, req.CourseId, req.LectureId, req.BlockId, fromBlockData(&req.Body.Data))
	if err != nil {
		return nil, err
	}
	return gen.UpdateBlock200JSONResponse(toBlock(b)), nil
}

func (s *Server) DeleteBlock(ctx context.Context, req gen.DeleteBlockRequestObject) (gen.DeleteBlockResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	if err := s.catalog.DeleteBlock(ctx, req.SchoolId, req.CourseId, req.LectureId, req.BlockId); err != nil {
		return nil, err
	}
	return gen.DeleteBlock204Response{}, nil
}

func (s *Server) ReorderBlocks(ctx context.Context, req gen.ReorderBlocksRequestObject) (gen.ReorderBlocksResponseObject, error) {
	if _, err := s.authorizeSchool(ctx, req.SchoolId, school.PermCoursesEdit); err != nil {
		return nil, err
	}
	blocks, err := s.catalog.ReorderBlocks(ctx, req.SchoolId, req.CourseId, req.LectureId, req.Body.BlockIds)
	if err != nil {
		return nil, err
	}
	return gen.ReorderBlocks200JSONResponse(toBlocks(blocks)), nil
}

// ---- Storefront ----

func (s *Server) ListStorefrontCourses(ctx context.Context, _ gen.ListStorefrontCoursesRequestObject) (gen.ListStorefrontCoursesResponseObject, error) {
	schoolID, err := tenant.RequireSchool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.catalog.ListPublished(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	out := make(gen.ListStorefrontCourses200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, gen.StorefrontCourseSummary{
			Id:           r.Course.ID,
			Slug:         r.Course.Slug,
			Title:        r.Course.Title,
			Subtitle:     r.Course.Subtitle,
			Locale:       gen.Locale(r.Course.Locale),
			LectureCount: int(r.LectureCount),
			PublishedAt:  utcPtr(r.Course.PublishedAt),
		})
	}
	return out, nil
}

func (s *Server) GetStorefrontCourse(ctx context.Context, req gen.GetStorefrontCourseRequestObject) (gen.GetStorefrontCourseResponseObject, error) {
	schoolID, err := tenant.RequireSchool(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.catalog.GetPublished(ctx, schoolID, req.Slug)
	if err != nil {
		return nil, err
	}
	sections := make([]gen.StorefrontSection, 0, len(c.Sections))
	for _, sec := range c.Sections {
		lectures := make([]gen.StorefrontLecture, 0, len(sec.Lectures))
		for _, l := range sec.Lectures {
			var dur *int
			if l.DurationSec != nil {
				d := int(*l.DurationSec)
				dur = &d
			}
			lectures = append(lectures, gen.StorefrontLecture{Id: l.ID, Title: l.Title, IsFreePreview: l.IsFreePreview, DurationSec: dur})
		}
		sections = append(sections, gen.StorefrontSection{Id: sec.ID, Title: sec.Title, Lectures: lectures})
	}
	return gen.GetStorefrontCourse200JSONResponse{
		Id:          c.Course.ID,
		Slug:        c.Course.Slug,
		Title:       c.Course.Title,
		Subtitle:    c.Course.Subtitle,
		Description: c.Course.Description,
		Locale:      gen.Locale(c.Course.Locale),
		Sections:    sections,
	}, nil
}

// ---- Mapping ----

func toCourse(c store.Course) gen.Course {
	return gen.Course{
		Id:          c.ID,
		Slug:        c.Slug,
		Title:       c.Title,
		Subtitle:    c.Subtitle,
		Description: c.Description,
		Locale:      gen.Locale(c.Locale),
		Status:      gen.CourseStatus(c.Status),
		PublishedAt: utcPtr(c.PublishedAt),
		CreatedAt:   c.CreatedAt.UTC(),
		UpdatedAt:   c.UpdatedAt.UTC(),
	}
}

func toSections(secs []catalog.Section) []gen.Section {
	out := make([]gen.Section, 0, len(secs))
	for _, sec := range secs {
		out = append(out, toSection(sec))
	}
	return out
}

func toSection(sec catalog.Section) gen.Section {
	lectures := make([]gen.Lecture, 0, len(sec.Lectures))
	for _, l := range sec.Lectures {
		lectures = append(lectures, toLecture(l))
	}
	return gen.Section{Id: sec.ID, Title: sec.Title, Position: int(sec.Position), Lectures: lectures}
}

func toLecture(l catalog.Lecture) gen.Lecture {
	return gen.Lecture{
		Id:            l.ID,
		SectionId:     l.SectionID,
		Title:         l.Title,
		Position:      int(l.Position),
		IsFreePreview: l.IsFreePreview,
		Status:        gen.LectureStatus(l.Status),
		BlockCount:    int(l.BlockCount),
	}
}

func toBlocks(blocks []catalog.Block) []gen.Block {
	out := make([]gen.Block, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, toBlock(b))
	}
	return out
}

func toBlock(b catalog.Block) gen.Block {
	return gen.Block{
		Id:           b.ID,
		Kind:         gen.BlockKind(b.Kind),
		Position:     int(b.Position),
		MediaAssetId: b.MediaAssetID,
		Data: gen.BlockData{
			Html:     nonEmpty(b.Content.HTML),
			Url:      nonEmpty(b.Content.URL),
			FileKey:  nonEmpty(b.Content.FileKey),
			Filename: nonEmpty(b.Content.Filename),
		},
	}
}

func fromBlockData(d *gen.BlockData) catalog.BlockData {
	if d == nil {
		return catalog.BlockData{}
	}
	return catalog.BlockData{HTML: deref(d.Html), URL: deref(d.Url), FileKey: deref(d.FileKey), Filename: deref(d.Filename)}
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
