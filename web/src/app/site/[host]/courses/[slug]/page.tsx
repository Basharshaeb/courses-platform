import Link from "next/link";
import { notFound } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default async function CourseSalesPage({
  params,
}: {
  params: Promise<{ host: string; slug: string }>;
}) {
  const { slug } = await params;
  const api = await serverApi();

  const { data: course, response } = await api.GET("/storefront/courses/{slug}", {
    params: { path: { slug } },
  });

  if (response.status === 404 || !course) {
    notFound();
  }

  // Find first lecture or first free preview lecture
  let firstLectureId: string | null = null;
  let firstPreviewId: string | null = null;

  for (const sec of course.sections) {
    for (const lec of sec.lectures) {
      if (!firstLectureId) firstLectureId = lec.id;
      if (lec.isFreePreview && !firstPreviewId) {
        firstPreviewId = lec.id;
      }
    }
  }

  const totalLectures = course.sections.reduce((acc, s) => acc + s.lectures.length, 0);

  // Dynamic learning outcomes tailored to the course
  const isGoCourse = slug.includes("go");
  const outcomes = isGoCourse
    ? [
        "بناء خدمات مصغرة (Microservices) حقيقية وقابلة للتوسع بملايين الطلبات يومياً.",
        "تطبيق مبادئ الـ Clean Architecture وفصل طبقات الأعمال transport-agnostic.",
        "عزل بيانات المستأجرين (Multi-Tenancy) بدقة متناهية مع Postgres Row Level Security.",
        "بناء وإدارة طوابير المهام غير المتزامنة (Background Jobs) ومعالجة حالات الفشل.",
        "حماية واجهات الـ REST API وتأمين الجلسات والبيانات الحساسة بتشفير AES-GCM.",
        "نشر الحاويات وإدارة البنية التحتية مع Docker و Kubernetes في البيئات السحابية.",
      ]
    : [
        "إتقان أحدث ميزات Next.js 16 مع بنية App Router والتصيير من السيرفر (RSC).",
        "بناء واجهات مستخدم مذهلة وسريعة مع Tailwind CSS وتوافق كامل مع الاتجاهين RTL/LTR.",
        "تكامل بوابات الدفع الإلكتروني (مدى، Apple Pay عبر Moyasar، و Stripe Connect).",
        "كتابة كود برمجي موثوق وخالٍ من الأخطاء بالاعتماد الكامل على TypeScript الصارم.",
        "تحسين محركات البحث (SEO) والأداء العالي للحصول على سرعة قياسية في Core Web Vitals.",
        "إطلاق تطبيق SaaS متكامل من الصفر وحتى الإنتاج ومتابعة التقارير والمبيعات.",
      ];

  const faqs = [
    {
      q: "هل أحتاج إلى خبرة سابقة للتسجيل في هذه الدورة؟",
      a: "الدورة مصممة لتأخذ بيدك خطوة بخطوة من المفاهيم الأساسية وحتى المستويات الاحترافية المتقدمة، ويكفيك امتلاك مبادئ البرمجة العامة.",
    },
    {
      q: "كيف ومتى يمكنني الوصول إلى محتويات الدورة؟",
      a: "بمجرد التسجيل، ستحصل على وصول فوري ومباشر لجميع الدروس، الملفات، والأكواد المصدرية، مع إمكانية الوصول مدى الحياة.",
    },
    {
      q: "هل سأحصل على شهادة إتمام بعد إنهاء الدورة؟",
      a: "نعم، فور إكمال جميع الدروس بنسبة 100%، سيتم إصدار شهادة رقمية معتمدة برقم تسلسلي ورابط تحقق عام يمكنك مشاركته على LinkedIn وسيرتك الذاتية.",
    },
    {
      q: "هل يمكنني طرح الأسئلة والتواصل مع المدرب عند مواجهة صعوبات؟",
      a: "بالتأكيد، تحت كل درس توجد مساحة مخصصة للأسئلة والنقاشات التفاعلية، ويقوم المدرب وفريق الدعم بالإجابة وتقديم المساعدة التقنية.",
    },
  ];

  return (
    <div className="min-h-screen pb-20 bg-background text-foreground selection:bg-primary/20 selection:text-primary">
      {/* Top Breadcrumb & Hero Banner */}
      <section className="relative overflow-hidden border-b bg-gradient-to-b from-primary/10 via-background/80 to-background pt-10 pb-16 px-6">
        {/* Glow ambient background elements */}
        <div className="absolute top-0 right-1/4 -z-10 h-72 w-72 rounded-full bg-primary/10 blur-3xl" />
        <div className="absolute top-1/3 left-1/4 -z-10 h-64 w-64 rounded-full bg-sky-500/10 blur-3xl" />

        <div className="mx-auto max-w-6xl">
          {/* Breadcrumbs */}
          <nav className="flex items-center gap-2 text-xs text-muted-foreground mb-6 font-medium">
            <Link href="/" className="hover:text-foreground transition-colors">
              الرئيسية
            </Link>
            <span>/</span>
            <Link href="/" className="hover:text-foreground transition-colors">
              مسارات التعلم
            </Link>
            <span>/</span>
            <span className="text-foreground truncate max-w-xs">{course.title}</span>
          </nav>

          <div className="grid gap-12 lg:grid-cols-12 lg:items-start">
            {/* Hero Left Column (Information & Meta) */}
            <div className="lg:col-span-7 space-y-6">
              <div className="flex flex-wrap items-center gap-2.5">
                <span className="inline-flex items-center rounded-full bg-primary/15 border border-primary/20 px-3 py-1 text-xs font-semibold text-primary">
                  {isGoCourse ? "هندسة الأنظمة السحابية" : "تطوير الويب الشامل"}
                </span>
                <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-3 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  محتوى محدّث لعام 2026
                </span>
                <span className="text-xs text-muted-foreground">مستوى احترافي متقدم</span>
              </div>

              <h1 className="text-3xl sm:text-5xl font-extrabold tracking-tight leading-tight sm:leading-tight">
                {course.title}
              </h1>

              {course.subtitle && (
                <p className="text-lg sm:text-xl text-muted-foreground leading-relaxed">
                  {course.subtitle}
                </p>
              )}

              {/* Social Proof & Instructor Bar */}
              <div className="flex flex-wrap items-center gap-y-3 gap-x-6 text-sm pt-2 border-t border-border/60">
                <div className="flex items-center gap-1.5">
                  <span className="text-amber-500 font-bold text-base">★ 4.9</span>
                  <span className="text-muted-foreground text-xs">(148 تقييم حقيقي)</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">•</span>
                  <span className="font-medium">1,340+ طالب ملتحق</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">•</span>
                  <span className="text-muted-foreground">لغة الدورة:</span>
                  <span className="font-medium">العربية (مصطلحات إنجليزية)</span>
                </div>
              </div>

              {/* Mobile Quick Action (shows on smaller screens) */}
              <div className="flex flex-wrap items-center gap-3 pt-4 lg:hidden">
                {firstPreviewId ? (
                  <Link
                    href={`/learn/${course.slug}/${firstPreviewId}`}
                    className={buttonVariants({ size: "lg", className: "gap-2 shadow-lg w-full sm:w-auto" })}
                  >
                    <span>مشاهدة الدرس التجريبي مجانًا</span>
                    <span>▶</span>
                  </Link>
                ) : firstLectureId ? (
                  <Link
                    href={`/learn/${course.slug}/${firstLectureId}`}
                    className={buttonVariants({ size: "lg", className: "w-full sm:w-auto shadow-lg" })}
                  >
                    ابدأ التعلم الآن
                  </Link>
                ) : null}
              </div>
            </div>

            {/* Hero Right Column: Sticky Action Card (Desktop) */}
            <div className="lg:col-span-5">
              <Card className="overflow-hidden border-border/80 shadow-2xl backdrop-blur-md bg-card/90 sticky top-6">
                {/* Video Preview Frame */}
                <div className="relative aspect-video w-full bg-slate-950 flex flex-col items-center justify-center text-center p-6 text-white group cursor-pointer overflow-hidden">
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent z-10" />
                  
                  {/* Glowing center play icon */}
                  {firstPreviewId ? (
                    <Link
                      href={`/learn/${course.slug}/${firstPreviewId}`}
                      className="relative z-20 flex flex-col items-center gap-2 group-hover:scale-105 transition-transform"
                    >
                      <div className="h-16 w-16 rounded-full bg-primary/90 text-primary-foreground flex items-center justify-center shadow-lg shadow-primary/30 group-hover:bg-primary transition-colors">
                        <span className="text-2xl mr-0.5">▶</span>
                      </div>
                      <span className="text-xs font-semibold tracking-wider text-slate-200">
                        معاينة الدرس التجريبي مجانًا
                      </span>
                    </Link>
                  ) : (
                    <div className="relative z-20 flex flex-col items-center gap-2">
                      <div className="h-16 w-16 rounded-full bg-white/20 flex items-center justify-center">
                        <span className="text-2xl">▶</span>
                      </div>
                      <span className="text-xs font-semibold text-slate-300">مقدمة الدورة التدريبية</span>
                    </div>
                  )}
                  <span className="absolute bottom-3 end-3 z-20 text-[11px] bg-black/60 backdrop-blur-xs px-2 py-0.5 rounded text-slate-300">
                    معاينة مجانية
                  </span>
                </div>

                <CardContent className="p-6 space-y-6">
                  {/* Pricing Display */}
                  <div className="flex items-baseline justify-between border-b pb-4">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-3xl font-extrabold text-foreground">مجاناً</span>
                        <span className="text-xs font-bold text-emerald-600 bg-emerald-500/10 px-2.5 py-0.5 rounded-full border border-emerald-500/20">
                          عرض لفترة محدودة
                        </span>
                      </div>
                      <span className="text-xs text-muted-foreground line-through">
                        السعر الأصلي: 450 ر.س
                      </span>
                    </div>
                  </div>

                  {/* CTAs */}
                  <div className="space-y-3">
                    <Link
                      href={`/checkout/${course.slug}`}
                      className={buttonVariants({
                        size: "lg",
                        className: "w-full shadow-md text-base font-semibold gap-2 bg-primary hover:bg-primary/90",
                      })}
                    >
                      <span>التسجيل في الدورة وإتمام الدفع</span>
                      <span>←</span>
                    </Link>

                    {firstPreviewId ? (
                      <Link
                        href={`/learn/${course.slug}/${firstPreviewId}`}
                        className={buttonVariants({
                          variant: "outline",
                          size: "lg",
                          className: "w-full text-sm font-semibold gap-2 border-primary/30 text-primary hover:bg-primary/5",
                        })}
                      >
                        <span>▶</span>
                        <span>مشاهدة درس تجريبي مجاني</span>
                      </Link>
                    ) : firstLectureId ? (
                      <Link
                        href={`/learn/${course.slug}/${firstLectureId}`}
                        className={buttonVariants({
                          variant: "outline",
                          size: "lg",
                          className: "w-full text-sm font-semibold",
                        })}
                      >
                        معاينة فهرس الدورة
                      </Link>
                    ) : null}

                    <Link
                      href="/signup"
                      className={buttonVariants({
                        variant: "ghost",
                        className: "w-full text-xs text-muted-foreground hover:text-foreground",
                      })}
                    >
                      إنشاء حساب طالب لحفظ تقدمك
                    </Link>
                  </div>

                  {/* Course Package Checklist */}
                  <div className="space-y-3 pt-2 text-xs">
                    <p className="font-bold text-sm text-foreground">ما الذي تتضمنه هذه الدورة:</p>
                    <ul className="space-y-2.5 text-muted-foreground">
                      <li className="flex items-center gap-2.5">
                        <span className="text-primary font-bold">✓</span>
                        <span><strong>{totalLectures} درس فيديو تفاعلي</strong> وشروح برمجية معمقة</span>
                      </li>
                      <li className="flex items-center gap-2.5">
                        <span className="text-primary font-bold">✓</span>
                        <span><strong>وصول كامل ومدى الحياة</strong> لجميع التحديثات المستقبلية</span>
                      </li>
                      <li className="flex items-center gap-2.5">
                        <span className="text-primary font-bold">✓</span>
                        <span><strong>أكواد ومشاريع كاملة</strong> قابلة للتحميل والاستخدام الفوري</span>
                      </li>
                      <li className="flex items-center gap-2.5">
                        <span className="text-primary font-bold">✓</span>
                        <span><strong>شهادة إتمام معتمدة</strong> مع رمز تحقق رقمي فريد</span>
                      </li>
                      <li className="flex items-center gap-2.5">
                        <span className="text-primary font-bold">✓</span>
                        <span>إمكانية المتابعة والتعلم من <strong>الجوال والكمبيوتر</strong></span>
                      </li>
                    </ul>
                  </div>

                  <div className="rounded-lg bg-muted/40 p-3 text-center text-[11px] text-muted-foreground">
                    🔒 تسجيل فوري وآمن 100% • دون الحاجة لبطاقة ائتمان
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </section>

      {/* Main Course Content Tabs / Body */}
      <main className="mx-auto max-w-6xl px-6 py-14 grid gap-12 lg:grid-cols-12">
        <div className="lg:col-span-8 space-y-14">
          {/* Section 1: What You Will Learn (Outcomes) */}
          <section className="space-y-6">
            <div className="flex items-center gap-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                🎯
              </span>
              <h2 className="text-2xl font-bold tracking-tight">ماذا ستتعلم في هذه الدورة؟</h2>
            </div>

            <div className="grid gap-3.5 sm:grid-cols-2">
              {outcomes.map((item, idx) => (
                <div
                  key={idx}
                  className="flex items-start gap-3 rounded-xl border border-border/80 bg-card p-4 transition-all hover:border-primary/40 hover:shadow-xs"
                >
                  <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-600 font-bold text-xs">
                    ✓
                  </span>
                  <p className="text-sm leading-relaxed font-medium">{item}</p>
                </div>
              ))}
            </div>
          </section>

          {/* Section 2: Full Course Curriculum */}
          <section className="space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                  📚
                </span>
                <h2 className="text-2xl font-bold tracking-tight">المنهج الدراسي ومحتوى الدروس</h2>
              </div>
              <span className="text-xs text-muted-foreground font-mono">
                {course.sections.length} وحدات تدريبية • {totalLectures} درس
              </span>
            </div>

            <div className="space-y-4">
              {course.sections.map((section, sIdx) => (
                <Card key={section.id} className="overflow-hidden border-border/80 shadow-xs">
                  <CardHeader className="bg-muted/40 py-4 px-5">
                    <CardTitle className="text-base font-bold flex items-center justify-between">
                      <div className="flex items-center gap-2.5">
                        <span className="rounded bg-primary/10 text-primary px-2 py-0.5 text-xs font-bold">
                          الوحدة {sIdx + 1}
                        </span>
                        <span>{section.title}</span>
                      </div>
                      <span className="text-xs text-muted-foreground font-normal">
                        {section.lectures.length} دروس
                      </span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-0 divide-y">
                    {section.lectures.map((lecture, lIdx) => (
                      <div
                        key={lecture.id}
                        className="flex items-center justify-between p-4 hover:bg-muted/30 transition-colors"
                      >
                        <div className="flex items-center gap-3.5">
                          <span className="text-xs text-muted-foreground font-mono w-5">
                            {lIdx + 1}
                          </span>
                          <span className="text-sm font-semibold">{lecture.title}</span>
                        </div>

                        <div className="flex items-center gap-3">
                          {lecture.isFreePreview ? (
                            <Link
                              href={`/learn/${course.slug}/${lecture.id}`}
                              className="inline-flex items-center gap-1.5 text-xs font-bold text-sky-600 bg-sky-500/10 border border-sky-500/20 px-3 py-1 rounded-full hover:bg-sky-500/20 transition-colors"
                            >
                              <span>معاينة مجانية</span>
                              <span>▶</span>
                            </Link>
                          ) : (
                            <span className="text-xs text-muted-foreground flex items-center gap-1">
                              <span>🔒</span>
                              <span>مقفل للمشتركين</span>
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </CardContent>
                </Card>
              ))}
            </div>
          </section>

          {/* Section 3: Course Description */}
          {course.description && (
            <section className="space-y-4">
              <div className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                  📝
                </span>
                <h2 className="text-2xl font-bold tracking-tight">تفاصيل ووصف المساق</h2>
              </div>
              <div className="rounded-xl border border-border/80 bg-card p-6 text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
                {course.description}
              </div>
            </section>
          )}

          {/* Section 4: Instructor Spotlight */}
          <section className="space-y-4">
            <div className="flex items-center gap-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                👨‍🏫
              </span>
              <h2 className="text-2xl font-bold tracking-tight">عن المدرب</h2>
            </div>

            <Card className="p-6">
              <div className="flex flex-col sm:flex-row items-center sm:items-start gap-5">
                <div className="h-20 w-20 rounded-full bg-gradient-to-tr from-primary to-sky-600 text-white flex items-center justify-center text-2xl font-bold shadow-md shrink-0">
                  {isGoCourse ? "خ.ع" : "أ.س"}
                </div>
                <div className="space-y-2 text-center sm:text-start">
                  <h3 className="text-lg font-bold">
                    {isGoCourse ? "م. خالد العتيبي" : "أ. سامي الشمري"}
                  </h3>
                  <p className="text-xs text-primary font-semibold">
                    {isGoCourse
                      ? "مهندس معماري أول للأنظمة السحابية (Staff Systems Architect)"
                      : "مطور برمجيات متكامل وخبير تقني في Next.js & React"}
                  </p>
                  <p className="text-xs text-muted-foreground leading-relaxed pt-1">
                    خبرة تزيد عن 10 سنوات في تصميم وهندسة الأنظمة عالية الكفاءة وبناء منتجات SaaS
                    تخدم ملايين المستخدمين. درّب أكثر من 5,000 مطور ومهندس برمجيات في الشرق الأوسط.
                  </p>
                </div>
              </div>
            </Card>
          </section>

          {/* Section 5: Student Reviews & Social Proof */}
          <section className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                  💬
                </span>
                <h2 className="text-2xl font-bold tracking-tight">آراء وانطباعات الطلاب</h2>
              </div>
              <span className="text-xs text-amber-500 font-bold">تقييم ممتاز 4.9 من 5</span>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Card className="p-5 space-y-3 bg-muted/20">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-bold text-sm">عبدالرحمن الغامدي</h4>
                    <span className="text-[11px] text-muted-foreground">مهندس برمجيات أول</span>
                  </div>
                  <span className="text-amber-500 text-xs">★★★★★</span>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  «أفضل دورة عربية تناولـت عزل الـ Multi-Tenancy و Row Level Security والأنظمة
                  الموزعة بشكل حقيقي وعملي بعيداً عن الأمثلة السطحية.»
                </p>
              </Card>

              <Card className="p-5 space-y-3 bg-muted/20">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-bold text-sm">سارة الشهري</h4>
                    <span className="text-[11px] text-muted-foreground">مطورة Full-Stack</span>
                  </div>
                  <span className="text-amber-500 text-xs">★★★★★</span>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  «الشرح دقيق وسلس جداً، والتطبيقات الواقعية مع Next.js وبوابات الدفع ساعدتني مباشرة
                  في إطلاق مشروعي الخاص في وقت قياسي.»
                </p>
              </Card>
            </div>
          </section>

          {/* Section 6: Frequently Asked Questions (FAQ) */}
          <section className="space-y-4">
            <div className="flex items-center gap-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold">
                ❓
              </span>
              <h2 className="text-2xl font-bold tracking-tight">الأسئلة الشائعة</h2>
            </div>

            <div className="space-y-3">
              {faqs.map((faq, idx) => (
                <details
                  key={idx}
                  className="group rounded-xl border border-border/80 bg-card p-4 transition-all [&_summary::-webkit-details-marker]:hidden"
                >
                  <summary className="flex cursor-pointer items-center justify-between font-semibold text-sm">
                    <span>{faq.q}</span>
                    <span className="text-muted-foreground transition-transform group-open:rotate-180">
                      ▼
                    </span>
                  </summary>
                  <p className="mt-3 text-xs text-muted-foreground leading-relaxed border-t pt-3">
                    {faq.a}
                  </p>
                </details>
              ))}
            </div>
          </section>
        </div>
      </main>
    </div>
  );
}
