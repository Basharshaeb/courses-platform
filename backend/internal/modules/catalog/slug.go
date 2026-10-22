package catalog

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
)

const (
	maxSlugLen = 80
	// maxGeneratedSlugLen caps generated slugs, leaving room for a "-N" de-duplication suffix.
	maxGeneratedSlugLen = 60
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateSlug checks the URL name of a course.
func ValidateSlug(slug string) error {
	if len(slug) > maxSlugLen || !slugRe.MatchString(slug) {
		return apperr.BadRequest("invalid_slug", "URL name must be lower-case letters, digits and single dashes (max 80)")
	}
	return nil
}

// Slugify turns a title into a URL name using ASCII letters and digits only. Titles without
// any (for example Arabic ones) give "course"; the creator can rename it.
func Slugify(title string) string {
	words := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	var b strings.Builder
	for _, w := range words {
		if b.Len() == 0 {
			b.WriteString(w[:min(len(w), maxGeneratedSlugLen)])
			continue
		}
		if b.Len()+1+len(w) > maxGeneratedSlugLen {
			break
		}
		b.WriteByte('-')
		b.WriteString(w)
	}
	if b.Len() == 0 {
		return "course"
	}
	return b.String()
}

// uniqueSlug returns base, or base-2, base-3, … if taken in the school.
func uniqueSlug(ctx context.Context, q *store.Queries, schoolID uuid.UUID, base string) (string, error) {
	for i := 1; i <= 50; i++ {
		slug := base
		if i > 1 {
			slug = fmt.Sprintf("%s-%d", base, i)
		}
		taken, err := q.CourseSlugExists(ctx, store.CourseSlugExistsParams{SchoolID: schoolID, Slug: slug})
		if err != nil {
			return "", fmt.Errorf("catalog: slug lookup: %w", err)
		}
		if !taken {
			return slug, nil
		}
	}
	return fmt.Sprintf("%s-%s", base, uuid.Must(uuid.NewV7()).String()[24:]), nil
}

func isSlugConflict(err error) bool {
	c, ok := db.UniqueViolation(err)
	return ok && c == "courses_school_id_slug_key"
}
