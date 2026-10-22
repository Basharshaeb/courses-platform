package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/store"
)

// Block kinds. Video blocks point to a media asset once the media module exists.
const (
	KindText  = "text"
	KindPDF   = "pdf"
	KindVideo = "video"
	KindEmbed = "embed"
)

// BlockData is the jsonb content of a block; which fields apply depends on the kind.
type BlockData struct {
	HTML     string `json:"html,omitempty"`
	URL      string `json:"url,omitempty"`
	FileKey  string `json:"fileKey,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// Block is a lecture block with decoded data.
type Block struct {
	store.LectureBlock
	Content BlockData
}

// htmlPolicy keeps formatting from the rich text editor and strips scripts, styles and handlers.
var htmlPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("dir").Matching(regexp.MustCompile(`^(rtl|ltr|auto)$`)).Globally()
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}()

// SanitizeHTML cleans rich text before it is stored.
func SanitizeHTML(html string) string { return strings.TrimSpace(htmlPolicy.Sanitize(html)) }

// normalizeBlock validates data for kind and drops fields that do not apply to it.
func normalizeBlock(kind string, d BlockData) (BlockData, error) {
	switch kind {
	case KindText:
		return BlockData{HTML: SanitizeHTML(d.HTML)}, nil
	case KindEmbed:
		u, err := url.Parse(strings.TrimSpace(d.URL))
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return BlockData{}, apperr.BadRequest("invalid_block", "embed URL must be an https link")
		}
		return BlockData{URL: u.String()}, nil
	case KindPDF:
		return BlockData{FileKey: strings.TrimSpace(d.FileKey), Filename: strings.TrimSpace(d.Filename)}, nil
	case KindVideo:
		return BlockData{}, nil
	}
	return BlockData{}, apperr.BadRequest("invalid_block", "unknown block kind")
}

func decodeBlock(b store.LectureBlock) (Block, error) {
	var d BlockData
	if len(b.Data) > 0 {
		if err := json.Unmarshal(b.Data, &d); err != nil {
			return Block{}, fmt.Errorf("catalog: decode block %s: %w", b.ID, err)
		}
	}
	return Block{LectureBlock: b, Content: d}, nil
}

func listBlocks(ctx context.Context, q *store.Queries, schoolID, lectureID uuid.UUID) ([]Block, error) {
	rows, err := q.ListBlocks(ctx, store.ListBlocksParams{SchoolID: schoolID, LectureID: lectureID})
	if err != nil {
		return nil, err
	}
	out := make([]Block, 0, len(rows))
	for _, r := range rows {
		b, err := decodeBlock(r)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// lectureInCourse checks that the lecture exists in the course (block routes are nested under it).
func lectureInCourse(ctx context.Context, q *store.Queries, schoolID, courseID, lectureID uuid.UUID) error {
	_, err := q.GetLecture(ctx, store.GetLectureParams{SchoolID: schoolID, CourseID: courseID, ID: lectureID})
	return notFound(err, errLectureNotFound)
}

// CreateBlock appends a block to a lecture.
func (s *Service) CreateBlock(ctx context.Context, schoolID, courseID, lectureID uuid.UUID, kind string, data BlockData) (Block, error) {
	d, err := normalizeBlock(kind, data)
	if err != nil {
		return Block{}, err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return Block{}, err
	}
	var out Block
	err = s.tx(ctx, schoolID, func(q *store.Queries) error {
		if err := lectureInCourse(ctx, q, schoolID, courseID, lectureID); err != nil {
			return err
		}
		b, err := q.CreateBlock(ctx, store.CreateBlockParams{
			ID: uuid.Must(uuid.NewV7()), SchoolID: schoolID, LectureID: lectureID, Kind: kind, Data: raw,
		})
		if err != nil {
			return fmt.Errorf("catalog: create block: %w", err)
		}
		out, err = decodeBlock(b)
		return err
	})
	return out, err
}

// UpdateBlock replaces the data of a block; the kind stays the same.
func (s *Service) UpdateBlock(ctx context.Context, schoolID, courseID, lectureID, blockID uuid.UUID, data BlockData) (Block, error) {
	var out Block
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		if err := lectureInCourse(ctx, q, schoolID, courseID, lectureID); err != nil {
			return err
		}
		cur, err := q.GetBlock(ctx, store.GetBlockParams{SchoolID: schoolID, LectureID: lectureID, ID: blockID})
		if err != nil {
			return notFound(err, errBlockNotFound)
		}
		d, err := normalizeBlock(cur.Kind, data)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		b, err := q.UpdateBlockData(ctx, store.UpdateBlockDataParams{Data: raw, SchoolID: schoolID, LectureID: lectureID, ID: blockID})
		if err != nil {
			return fmt.Errorf("catalog: update block: %w", err)
		}
		out, err = decodeBlock(b)
		return err
	})
	return out, err
}

func (s *Service) DeleteBlock(ctx context.Context, schoolID, courseID, lectureID, blockID uuid.UUID) error {
	return s.tx(ctx, schoolID, func(q *store.Queries) error {
		if err := lectureInCourse(ctx, q, schoolID, courseID, lectureID); err != nil {
			return err
		}
		n, err := q.DeleteBlock(ctx, store.DeleteBlockParams{SchoolID: schoolID, LectureID: lectureID, ID: blockID})
		if err != nil {
			return err
		}
		if n == 0 {
			return errBlockNotFound
		}
		return nil
	})
}

// ReorderBlocks sets the order of a lecture's blocks; ids must list every block exactly once.
func (s *Service) ReorderBlocks(ctx context.Context, schoolID, courseID, lectureID uuid.UUID, ids []uuid.UUID) ([]Block, error) {
	var out []Block
	err := s.tx(ctx, schoolID, func(q *store.Queries) error {
		if err := lectureInCourse(ctx, q, schoolID, courseID, lectureID); err != nil {
			return err
		}
		current, err := listBlocks(ctx, q, schoolID, lectureID)
		if err != nil {
			return err
		}
		known := make(map[uuid.UUID]bool, len(current))
		for _, b := range current {
			known[b.ID] = true
		}
		if len(ids) != len(known) {
			return apperr.BadRequest("invalid_order", "the order must list every block of the lecture exactly once")
		}
		for _, id := range ids {
			if !known[id] {
				return apperr.BadRequest("invalid_order", "the order must list every block of the lecture exactly once")
			}
			delete(known, id)
		}
		if _, err := q.SetBlockPositions(ctx, store.SetBlockPositionsParams{SchoolID: schoolID, LectureID: lectureID, Ids: ids}); err != nil {
			return fmt.Errorf("catalog: reorder blocks: %w", err)
		}
		out, err = listBlocks(ctx, q, schoolID, lectureID)
		return err
	})
	return out, err
}
