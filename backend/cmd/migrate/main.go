// Command migrate applies schema migrations (goose + River) and seeds demo data.
//
//	migrate up      apply all migrations
//	migrate down    roll back the last goose migration
//	migrate status  list goose migrations
//	migrate seed    create demo creator, school and student (idempotent)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"

	"github.com/teatchable/backend/internal/modules/catalog"
	"github.com/teatchable/backend/internal/modules/identity"
	"github.com/teatchable/backend/internal/modules/school"
	"github.com/teatchable/backend/internal/platform/apperr"
	"github.com/teatchable/backend/internal/platform/config"
	"github.com/teatchable/backend/internal/platform/db"
	"github.com/teatchable/backend/internal/platform/tenant"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if err := run(context.Background(), cmd); err != nil {
		slog.Error("migrate failed", "cmd", cmd, "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch cmd {
	case "up":
		if err := db.Migrate(ctx, cfg.MigrateDatabaseURL, os.Stdout); err != nil {
			return err
		}
		fmt.Println("migrations up to date")
		return nil
	case "down":
		return db.MigrateDown(ctx, cfg.MigrateDatabaseURL, os.Stdout)
	case "status":
		return db.MigrateStatus(ctx, cfg.MigrateDatabaseURL, os.Stdout)
	case "seed":
		return seed(ctx, cfg)
	default:
		return fmt.Errorf("unknown command %q (want up, down, status, seed)", cmd)
	}
}

// Demo data for local development only. Credentials are documented in README.md.
const (
	demoCreatorEmail = "creator@demo.test"
	demoStudentEmail = "student@demo.test"
	demoPassword     = "demo-password-123"
	demoSubdomain    = "demo"
)

func seed(ctx context.Context, cfg config.Config) error {
	if cfg.IsProduction() {
		return errors.New("refusing to seed in production")
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	schools := school.NewService(pool)
	ids := identity.NewService(pool, schools, cfg.SessionTTL)
	meta := identity.ClientMeta{UserAgent: "seed"}

	_, _, err = ids.SignupAccount(ctx, identity.AccountSignupInput{
		Email: demoCreatorEmail, Password: demoPassword, Name: "Demo Creator",
		SchoolName: "أكاديمية التجربة", Subdomain: demoSubdomain, Locale: "ar",
	}, meta)
	if e, ok := apperr.As(err); ok && e.Code == "email_taken" {
		fmt.Println("demo creator exists")
	} else if err != nil {
		return err
	} else {
		fmt.Println("created creator", demoCreatorEmail, "with school", demoSubdomain)
	}

	resolver := tenant.NewResolver(pool, tenant.Hosts{RootDomain: cfg.RootDomain, AppHost: cfg.AppHost, AdminHost: cfg.AdminHost}, 0)
	t, err := resolver.Resolve(ctx, demoSubdomain+"."+cfg.RootDomain)
	if err != nil {
		return err
	}
	if t.Surface != tenant.SurfaceSchool {
		return errors.New("demo school not resolvable; is ROOT_DOMAIN set correctly?")
	}
	_, _, err = ids.SignupStudent(ctx, t.SchoolID, identity.StudentSignupInput{
		Email: demoStudentEmail, Password: demoPassword, Name: "Demo Student", Locale: "ar",
	}, meta)
	if e, ok := apperr.As(err); ok && e.Code == "email_taken" {
		fmt.Println("demo student exists")
	} else if err != nil {
		return err
	} else {
		fmt.Println("created student", demoStudentEmail)
	}

	cats := catalog.NewService(pool)
	seedCourses(ctx, cats, t.SchoolID)

	return nil
}

func seedCourses(ctx context.Context, cats *catalog.Service, schoolID uuid.UUID) {
	courses, _ := cats.ListCourses(ctx, schoolID)
	for _, c := range courses {
		if c.Course.Slug == "go-microservices" {
			fmt.Println("demo courses already exist")
			return
		}
	}

	// Course 1
	c1, err := cats.CreateCourse(ctx, schoolID, catalog.CreateCourseInput{
		Title:       "بناء الأنظمة الموزعة والخدمات المصغرة بلغة Go",
		Slug:        "go-microservices",
		Subtitle:    "دليل شامل وعملي لاحتراف تطوير الأنظمة السحابية عالية الأداء والتوسع في بيئات الإنتاج.",
		Description: "في هذه الدورة التدريبية المتقدمة، ستتعلم كيفية بناء أنظمة موزعة وخدمات مصغرة (Microservices) حقيقية وقابلة للتوسع باستخدام لغة Go و Docker و PostgreSQL و Redis.\n\nسنتناول أفضل الممارسات في:\n• التصميم المعماري (Domain-Driven Design & Clean Architecture)\n• التواصل عالي الكفاءة عبر gRPC و REST مع OpenAPI 3.1\n• عزل البيانات والأنظمة متعددة المستأجرين (Multi-Tenancy with Row Level Security)\n• طوابير المهام الخلفية الموثوقة (Background Job Queues)\n• أمان التطبيقات والمصادقة وتشفير الجلسات (Argon2id & Opaque Tokens)\n• المراقبة وتتبع الأداء (OpenTelemetry & Prometheus)",
		Locale:      "ar",
	})
	if err == nil {
		s1, _ := cats.CreateSection(ctx, schoolID, c1.ID, "الوحدة 1: الأساسيات المعمارية والبيئة التطويرية")
		if s1.ID != uuid.Nil {
			l1, _ := cats.CreateLecture(ctx, schoolID, c1.ID, s1.ID, "مقدمة إلى الأنظمة الموزعة والتحديات الهندسية")
			_, _ = cats.UpdateLecture(ctx, schoolID, c1.ID, l1.ID, catalog.UpdateLectureInput{
				IsFreePreview: boolPtr(true),
			})
			html1 := "<h3>مرحبًا بك في عالم الأنظمة الموزعة!</h3><p>في هذا الدرس الاستهلالي، نستعرض المعمارية الهندسية، ولماذا تعتبر لغة <strong>Go</strong> الخيار الأول لكبرى الشركات العالمية (مثل Google و Uber و Docker) لبناء بنى تحتية موثوقة وسريعة للغاية.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c1.ID, l1.ID, "text", catalog.BlockData{
				HTML: html1,
			})

			l2, _ := cats.CreateLecture(ctx, schoolID, c1.ID, s1.ID, "إعداد بيئة التطوير وقاعدة البيانات Postgres مع RLS")
			html2 := "<p>شرح عملي لإعداد الحاويات عبر Docker Compose وتفعيل ميزة Row-Level Security في PostgreSQL لضمان أعلى مستويات الأمان وعزل بيانات العملاء في بيئة الـ Multi-Tenancy.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c1.ID, l2.ID, "text", catalog.BlockData{
				HTML: html2,
			})
		}

		s2, _ := cats.CreateSection(ctx, schoolID, c1.ID, "الوحدة 2: بناء محركات الأعمال وطوابير المهام")
		if s2.ID != uuid.Nil {
			l3, _ := cats.CreateLecture(ctx, schoolID, c1.ID, s2.ID, "تصميم الـ Domain Model وهندسة الطبقات في Go")
			_, _ = cats.UpdateLecture(ctx, schoolID, c1.ID, l3.ID, catalog.UpdateLectureInput{
				IsFreePreview: boolPtr(true),
			})
			html3 := "<p>تطبيق مبادئ الـ Clean Architecture في Go، وفصل طبقات الأعمال transport-agnostic modules عن واجهات الـ HTTP ومحركات قاعدة البيانات.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c1.ID, l3.ID, "text", catalog.BlockData{
				HTML: html3,
			})

			l4, _ := cats.CreateLecture(ctx, schoolID, c1.ID, s2.ID, "بناء طوابير المهام الخلفية الموثوقة مع River Queue")
			html4 := "<p>كيف تنفذ المهام غير المتزامنة مثل إرسال الإشعارات، معالجة الدفعات، وتوليد التقارير والفواتير بصورة آمنة ومقاومة للأخطاء.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c1.ID, l4.ID, "text", catalog.BlockData{
				HTML: html4,
			})
		}
		_, _ = cats.PublishCourse(ctx, schoolID, c1.ID)
		fmt.Println("seeded course 1: go-microservices")
	}

	// Course 2
	c2, err := cats.CreateCourse(ctx, schoolID, catalog.CreateCourseInput{
		Title:       "تطوير تطبيقات الويب المتكاملة بـ Next.js 16 و TypeScript",
		Slug:        "nextjs-fullstack-mastery",
		Subtitle:    "من الصفر إلى إطلاق منتج SaaS متكامل مع دعم كامل للغة العربية (RTL) وبوابات الدفع.",
		Description: "تعلم أحدث تقنيات بناء تطبيقات الويب الحديثة باستخدام Next.js 16 (App Router & Turbopack)، و Tailwind CSS، و TypeScript مع أفضل الممارسات المتبعة في كبرى الشركات التقنية.\n\nستبني مشروعاً متكاملاً يدعم:\n• التصيير من جانب السيرفر (Server Components & Streaming)\n• تعدد اللغات مع دعم اتجاه النصوص العربية RTL بالكامل عبر next-intl\n• ربط بوابات الدفع المحلية (مدى، Apple Pay عبر Moyasar) والدولية (Stripe)\n• واجهات مستخدم مذهلة وسريعة الاستجابة بأحدث معايير التصميم العالمي",
		Locale:      "ar",
	})
	if err == nil {
		s1, _ := cats.CreateSection(ctx, schoolID, c2.ID, "الوحدة 1: الأساسيات ومفاهيم App Router و RTL")
		if s1.ID != uuid.Nil {
			l1, _ := cats.CreateLecture(ctx, schoolID, c2.ID, s1.ID, "نظرة عامة على Next.js 16 ومحرك Turbopack")
			_, _ = cats.UpdateLecture(ctx, schoolID, c2.ID, l1.ID, catalog.UpdateLectureInput{
				IsFreePreview: boolPtr(true),
			})
			html1 := "<p>استكشاف الميزات الجديدة في Next.js 16، والسرعة الفائقة مع Turbopack، وكيفية تنظيم المسارات مع دعم التوجيه الديناميكي للنطاقات المتعددة.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c2.ID, l1.ID, "text", catalog.BlockData{
				HTML: html1,
			})

			l2, _ := cats.CreateLecture(ctx, schoolID, c2.ID, s1.ID, "تخصيص الواجهات والخطوط العربية مع Tailwind CSS")
			html2 := "<p>كيفية تصميم واجهات تدعم الاتجاهين RTL و LTR بسلاسة، واستخدام الخطوط العربية الحديثة مثل IBM Plex Sans Arabic و Tajawal.</p>"
			_, _ = cats.CreateBlock(ctx, schoolID, c2.ID, l2.ID, "text", catalog.BlockData{
				HTML: html2,
			})
		}
		_, _ = cats.PublishCourse(ctx, schoolID, c2.ID)
		fmt.Println("seeded course 2: nextjs-fullstack-mastery")
	}
}

func boolPtr(b bool) *bool { return &b }
