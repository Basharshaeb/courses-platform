import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { serverApi } from "@/lib/api/server";

export default async function SchoolHome() {
  const api = await serverApi();
  const { data: school } = await api.GET("/storefront/school");
  const { data: courses } = await api.GET("/storefront/courses");
  const t = await getTranslations("school");

  return (
    <div className="flex flex-col min-h-screen bg-background text-foreground">
      {/* Hero Header with Rich Aesthetics */}
      <section className="relative overflow-hidden border-b bg-gradient-to-b from-primary/10 via-background/60 to-background py-20 sm:py-28 px-6 text-center">
        {/* Ambient background glows */}
        <div className="absolute top-0 right-1/3 -z-10 h-96 w-96 rounded-full bg-primary/15 blur-3xl opacity-70" />
        <div className="absolute bottom-0 left-1/3 -z-10 h-80 w-80 rounded-full bg-sky-500/10 blur-3xl opacity-60" />

        <div className="mx-auto max-w-4xl space-y-6">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-3.5 py-1 text-xs font-semibold text-primary">
            <span>✨</span>
            <span>أكاديمية تعليمية احترافية معتمدة</span>
          </div>

          <h1 className="text-4xl sm:text-6xl font-extrabold tracking-tight leading-tight">
            {school?.name || "أكاديمية التعليم الرقمي"}
          </h1>

          <p className="text-lg sm:text-xl text-muted-foreground max-w-2xl mx-auto leading-relaxed">
            اكتسب المهارات التقنية الأكثر طلباً في سوق العمل مع مناهج عملية، مشاريع حقيقية، وشهادات
            إتمام معتمدة.
          </p>

          <div className="flex flex-wrap items-center justify-center gap-4 pt-4">
            <Link href="/library" className={buttonVariants({ variant: "outline", size: "lg", className: "px-6 font-semibold" })}>
              {t("library")}
            </Link>
            <Link href="/signup" className={buttonVariants({ size: "lg", className: "px-8 font-semibold shadow-lg shadow-primary/20" })}>
              ابدأ التعلم الآن مجاناً
            </Link>
          </div>

          {/* Quick Metrics Bar */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-10 max-w-3xl mx-auto border-t border-border/60 text-start">
            <div className="p-3">
              <span className="text-2xl font-extrabold text-foreground">1,400+</span>
              <p className="text-xs text-muted-foreground mt-0.5">طالب ملتحق</p>
            </div>
            <div className="p-3">
              <span className="text-2xl font-extrabold text-foreground">4.9 ★</span>
              <p className="text-xs text-muted-foreground mt-0.5">متوسط التقييم</p>
            </div>
            <div className="p-3">
              <span className="text-2xl font-extrabold text-foreground">100%</span>
              <p className="text-xs text-muted-foreground mt-0.5">مشاريع عملية</p>
            </div>
            <div className="p-3">
              <span className="text-2xl font-extrabold text-foreground">مدى الحياة</span>
              <p className="text-xs text-muted-foreground mt-0.5">وصول دائم ومستمر</p>
            </div>
          </div>
        </div>
      </section>

      {/* Courses Catalog Section */}
      <main className="mx-auto w-full max-w-6xl px-6 py-16">
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 mb-10">
          <div>
            <div className="flex items-center gap-2 text-xs font-bold text-primary uppercase tracking-wider mb-1">
              <span>المسارات التدريبية</span>
            </div>
            <h2 className="text-3xl font-extrabold tracking-tight">الدورات التدريبية المتاحة</h2>
            <p className="text-sm text-muted-foreground mt-1">
              اختر المسار الأنسب لطموحك المهني وابدأ المشاهدة فوراً
            </p>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs bg-muted px-3 py-1.5 rounded-full font-medium">
              إجمالي {courses?.length || 0} دورات منشورة
            </span>
          </div>
        </div>

        {!courses || courses.length === 0 ? (
          <Card className="border-dashed p-12 text-center text-muted-foreground">
            <CardHeader>
              <CardTitle className="text-lg">لا توجد دورات منشورة حالياً</CardTitle>
              <CardDescription>تابعنا قريباً، سيتم إضافة دورات تعليمية جديدة في الأكاديمية.</CardDescription>
            </CardHeader>
          </Card>
        ) : (
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
            {courses.map((course, idx) => {
              const isGo = course.slug.includes("go");
              return (
                <Card
                  key={course.id}
                  className="group flex flex-col justify-between overflow-hidden rounded-2xl border-border/80 transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:border-primary/40 bg-card"
                >
                  {/* Course Thumbnail Banner with Rich Gradient */}
                  <div
                    className={`h-48 w-full relative flex items-center justify-center p-6 text-white overflow-hidden ${
                      isGo
                        ? "bg-gradient-to-br from-sky-700 via-blue-900 to-slate-950"
                        : idx % 2 === 0
                        ? "bg-gradient-to-br from-indigo-700 via-purple-900 to-slate-950"
                        : "bg-gradient-to-br from-emerald-700 via-teal-900 to-slate-950"
                    }`}
                  >
                    <div className="absolute inset-0 bg-black/20 group-hover:bg-black/0 transition-colors" />

                    <div className="relative z-10 text-center space-y-1">
                      <span className="text-3xl font-black opacity-30 select-none tracking-widest">
                        {isGo ? "GO // CLOUD" : "REACT // NEXT"}
                      </span>
                    </div>

                    <div className="absolute top-3.5 start-3.5 z-20">
                      <span className="text-[11px] font-bold bg-white/20 backdrop-blur-md px-2.5 py-1 rounded-full text-white border border-white/20">
                        {course.lectureCount} درس تعليمي
                      </span>
                    </div>

                    <div className="absolute top-3.5 end-3.5 z-20">
                      <span className="text-[11px] font-bold bg-emerald-500/90 text-white px-2.5 py-1 rounded-full shadow-xs">
                        مجاني
                      </span>
                    </div>
                  </div>

                  <CardHeader className="p-6 space-y-2 flex-1">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span className="text-amber-500 font-bold">★ 4.9 (120+)</span>
                      <span className="font-mono text-[11px]">مستوى احترافي</span>
                    </div>

                    <CardTitle className="text-lg font-bold group-hover:text-primary transition-colors line-clamp-2 leading-snug">
                      {course.title}
                    </CardTitle>

                    <CardDescription className="line-clamp-2 text-xs leading-relaxed">
                      {course.subtitle || "دورة عملية تطبيقية لاحتراف هذا المساق وبناء مشاريع للإنتاج."}
                    </CardDescription>
                  </CardHeader>

                  <CardContent className="p-6 pt-0 border-t bg-muted/10 space-y-3">
                    <div className="flex items-center justify-between text-xs pt-3 font-medium">
                      <span className="text-muted-foreground">الشهادة:</span>
                      <span className="text-emerald-600 font-semibold">شهادة إتمام معتمدة</span>
                    </div>

                    <Link
                      href={`/courses/${course.slug}`}
                      className={buttonVariants({
                        className: "w-full font-semibold shadow-xs transition-all",
                      })}
                    >
                      عرض تفاصيل الدورة والمنهج
                    </Link>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </main>

      {/* Why Choose Our Academy Section */}
      <section className="border-t bg-muted/20 py-20 px-6">
        <div className="mx-auto max-w-6xl space-y-12">
          <div className="text-center space-y-3 max-w-2xl mx-auto">
            <h2 className="text-3xl font-extrabold tracking-tight">لماذا تتعلّم معنا؟</h2>
            <p className="text-muted-foreground text-sm">
              نوفر لك بيئة تعليمية احترافية تركّز على ما يحتاجه سوق العمل الحقيقي.
            </p>
          </div>

          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors">
              <span className="text-3xl">💻</span>
              <h3 className="font-bold text-base">مناهج تطبيقية 100%</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                بعيداً عن الحشو النظري؛ كل درس يركز على كود حقيقي، وتحديات برمجية واقعية.
              </p>
            </Card>

            <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors">
              <span className="text-3xl">📜</span>
              <h3 className="font-bold text-base">شهادات معتمدة</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                احصل على شهادة إتمام رقمية برقم تسلسلي يمكن التحقق منها ومشاركتها في سيرتك الذاتية.
              </p>
            </Card>

            <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors">
              <span className="text-3xl">⚡</span>
              <h3 className="font-bold text-base">وصول غير محدود</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                تعلّم بالوتيرة التي تناسبك مع وصول مدى الحياة لجميع المحاضرات والتحديثات.
              </p>
            </Card>

            <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors">
              <span className="text-3xl">💬</span>
              <h3 className="font-bold text-base">نقاشات وتواصل مع المدرب</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                اسأل في أي وقت واحصل على إجابات مباشرة وتوجيهات عملية لتجاوز أي عقبة برمجية.
              </p>
            </Card>
          </div>
        </div>
      </section>
    </div>
  );
}
