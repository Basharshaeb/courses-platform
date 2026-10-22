package db_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/teatchable/backend/internal/modules/catalog"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/store"
	"github.com/teatchable/backend/internal/testutil/pgtest"
)

var env *pgtest.Env

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}
	e, stop, err := pgtest.Start(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	env = e
	code := m.Run()
	stop()
	os.Exit(code)
}

// TestRowLevelSecurity proves the application role only sees rows of the tenant set in the
// transaction, and cannot write rows for another tenant.
func TestRowLevelSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	schools := school.NewService(env.AppPool)
	ids := identity.NewService(env.AppPool, schools, time.Hour)

	newSchool := func(sub string) uuid.UUID {
		out, _, err := ids.SignupAccount(ctx, identity.AccountSignupInput{
			Email: sub + "@owner.test", Password: "password123", Name: sub, SchoolName: sub, Subdomain: sub,
		}, identity.ClientMeta{})
		if err != nil {
			t.Fatalf("signup %s: %v", sub, err)
		}
		return out.Schools[0].SchoolID
	}
	schoolA, schoolB := newSchool("rls-a"), newSchool("rls-b")

	studentA, _, err := ids.SignupStudent(ctx, schoolA, identity.StudentSignupInput{Email: "s@a.test", Password: "password123", Name: "A"}, identity.ClientMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ids.SignupStudent(ctx, schoolB, identity.StudentSignupInput{Email: "s@b.test", Password: "password123", Name: "B"}, identity.ClientMeta{}); err != nil {
		t.Fatal(err)
	}

	countStudents := func(tx pgx.Tx) (n int) {
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM students").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("no tenant sees nothing", func(t *testing.T) {
		err := db.WithTx(ctx, env.AppPool, func(q *store.Queries, tx pgx.Tx) error {
			if n := countStudents(tx); n != 0 {
				t.Errorf("students visible without tenant: %d", n)
			}
			var schoolsVisible int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM schools").Scan(&schoolsVisible); err != nil {
				return err
			}
			if schoolsVisible != 0 {
				t.Errorf("schools visible without tenant: %d", schoolsVisible)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("tenant sees only own rows", func(t *testing.T) {
		err := db.WithTenantTx(ctx, env.AppPool, schoolB, func(q *store.Queries, tx pgx.Tx) error {
			if n := countStudents(tx); n != 1 {
				t.Errorf("school B sees %d students, want 1", n)
			}
			_, err := q.GetStudentByID(ctx, store.GetStudentByIDParams{SchoolID: schoolA, ID: studentA.ID})
			if !db.IsNotFound(err) {
				t.Errorf("school B read student of A: err=%v", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cannot insert into another tenant", func(t *testing.T) {
		err := db.WithTenantTx(ctx, env.AppPool, schoolA, func(q *store.Queries, _ pgx.Tx) error {
			_, err := q.CreateStudent(ctx, store.CreateStudentParams{
				ID: uuid.Must(uuid.NewV7()), SchoolID: schoolB, Email: "evil@b.test", Name: "x", Locale: "ar",
			})
			return err
		})
		if err == nil {
			t.Fatal("expected row level security violation")
		}
	})
}

// TestCatalogIsolation covers the catalog tables: another tenant sees none of a school's
// curriculum and cannot attach its own rows to it, even with known ids.
func TestCatalogIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()
	schools := school.NewService(env.AppPool)
	ids := identity.NewService(env.AppPool, schools, time.Hour)
	cat := catalog.NewService(env.AppPool)

	newSchool := func(sub string) uuid.UUID {
		out, _, err := ids.SignupAccount(ctx, identity.AccountSignupInput{
			Email: sub + "@owner.test", Password: "password123", Name: sub, SchoolName: sub, Subdomain: sub,
		}, identity.ClientMeta{})
		if err != nil {
			t.Fatalf("signup %s: %v", sub, err)
		}
		return out.Schools[0].SchoolID
	}
	schoolA, schoolB := newSchool("cat-a"), newSchool("cat-b")

	course, err := cat.CreateCourse(ctx, schoolA, catalog.CreateCourseInput{Title: "A course"})
	if err != nil {
		t.Fatal(err)
	}
	section, err := cat.CreateSection(ctx, schoolA, course.ID, "S")
	if err != nil {
		t.Fatal(err)
	}
	lecture, err := cat.CreateLecture(ctx, schoolA, course.ID, section.ID, "L")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cat.CreateBlock(ctx, schoolA, course.ID, lecture.ID, catalog.KindText, catalog.BlockData{HTML: "<p>x</p>"}); err != nil {
		t.Fatal(err)
	}

	t.Run("other tenant sees nothing", func(t *testing.T) {
		err := db.WithTenantTx(ctx, env.AppPool, schoolB, func(_ *store.Queries, tx pgx.Tx) error {
			for _, table := range []string{"courses", "course_sections", "lectures", "lecture_blocks"} {
				var n int
				if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil { //nolint:gosec // fixed table names
					return err
				}
				if n != 0 {
					t.Errorf("school B sees %d rows in %s", n, table)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	// Foreign keys bypass RLS; the composite (id, school_id) keys stop cross-tenant references.
	attach := map[string]func(q *store.Queries) error{
		"section on foreign course": func(q *store.Queries) error {
			_, err := q.CreateSection(ctx, store.CreateSectionParams{ID: uuid.Must(uuid.NewV7()), SchoolID: schoolB, CourseID: course.ID, Title: "x"})
			return err
		},
		"lecture in foreign section": func(q *store.Queries) error {
			_, err := q.CreateLecture(ctx, store.CreateLectureParams{ID: uuid.Must(uuid.NewV7()), SchoolID: schoolB, CourseID: course.ID, SectionID: section.ID, Title: "x"})
			return err
		},
		"block in foreign lecture": func(q *store.Queries) error {
			_, err := q.CreateBlock(ctx, store.CreateBlockParams{ID: uuid.Must(uuid.NewV7()), SchoolID: schoolB, LectureID: lecture.ID, Kind: "text", Data: []byte("{}")})
			return err
		},
	}
	for name, fn := range attach {
		t.Run(name, func(t *testing.T) {
			err := db.WithTenantTx(ctx, env.AppPool, schoolB, func(q *store.Queries, _ pgx.Tx) error { return fn(q) })
			if err == nil {
				t.Fatal("expected foreign key violation")
			}
		})
	}

	t.Run("lecture cannot move to a section of another course", func(t *testing.T) {
		other, err := cat.CreateCourse(ctx, schoolA, catalog.CreateCourseInput{Title: "Other"})
		if err != nil {
			t.Fatal(err)
		}
		otherSection, err := cat.CreateSection(ctx, schoolA, other.ID, "S")
		if err != nil {
			t.Fatal(err)
		}
		err = db.WithTenantTx(ctx, env.AppPool, schoolA, func(q *store.Queries, _ pgx.Tx) error {
			_, err := q.MoveLectures(ctx, store.MoveLecturesParams{
				SchoolID: schoolA, CourseID: course.ID, Ids: []uuid.UUID{lecture.ID}, SectionIds: []uuid.UUID{otherSection.ID}, Positions: []int32{0},
			})
			return err
		})
		if err == nil {
			t.Fatal("expected foreign key violation")
		}
	})
}
