import Link from "next/link";
import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { LogoutButton } from "@/components/logout-button";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { serverApi } from "@/lib/api/server";

export default async function LibraryPage() {
  const api = await serverApi();
  const { data: me, response } = await api.GET("/me");
  if (response.status === 401) redirect("/login");
  if (!me) throw new Error(`me failed: ${response.status}`);

  const { data: courses } = await api.GET("/storefront/courses");
  const t = await getTranslations("school");

  return (
    <main className="mx-auto grid w-full max-w-5xl gap-8 p-6 sm:p-10">
      {/* Student Welcome Banner */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b pb-6">
        <div>
          <span className="text-xs font-bold text-primary uppercase tracking-wider">لوحة الطالب</span>
          <h1 className="text-3xl font-extrabold tracking-tight mt-1">
            {t("greeting", { name: me.name })} 👋
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            مرحبًا بك في مساحتك التعليمية الخاصة. تابع تقدمك في الدورات واستكمل رحلتك البرمجية.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link href="/" className={buttonVariants({ variant: "outline", size: "sm" })}>
            تصفح المتجر
          </Link>
          <LogoutButton kind="student" redirectTo="/" />
        </div>
      </div>

      {/* Enrolled Courses / Active Courses */}
      <section className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-bold tracking-tight">دوراتك الحالية</h2>
          <span className="text-xs text-muted-foreground font-mono">
            {courses?.length || 0} دورات متاحة
          </span>
        </div>

        {!courses || courses.length === 0 ? (
          <Card className="border-dashed p-10 text-center text-muted-foreground">
            <CardHeader>
              <CardTitle className="text-lg">لم تسجل في أي دورة بعد</CardTitle>
              <CardDescription>{t("libraryEmpty")}</CardDescription>
            </CardHeader>
            <CardContent>
              <Link href="/" className={buttonVariants({ size: "sm" })}>
                استكشف الدورات المتاحة
              </Link>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-6 sm:grid-cols-2">
            {courses.map((course, idx) => {
              const isGo = course.slug.includes("go");
              const simulatedProgress = idx === 0 ? 35 : 10;

              return (
                <Card
                  key={course.id}
                  className="overflow-hidden border-border/80 shadow-xs hover:shadow-md transition-shadow flex flex-col justify-between"
                >
                  <CardHeader className="p-5 pb-3">
                    <div className="flex items-center justify-between text-xs text-muted-foreground mb-2">
                      <span className="bg-primary/10 text-primary font-bold px-2.5 py-0.5 rounded-full">
                        {isGo ? "أنظمة سحابية" : "تطوير الويب"}
                      </span>
                      <span className="font-mono">{course.lectureCount} درس</span>
                    </div>
                    <CardTitle className="text-lg font-bold line-clamp-1">{course.title}</CardTitle>
                    <CardDescription className="line-clamp-2 text-xs">
                      {course.subtitle || "دورة عملية معتمدة لاحتراف التقنية."}
                    </CardDescription>
                  </CardHeader>

                  <CardContent className="p-5 pt-0 space-y-4">
                    {/* Progress Bar */}
                    <div className="space-y-1.5 pt-2 border-t">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-muted-foreground">التقدم:</span>
                        <span className="font-bold text-emerald-600 font-mono">
                          {simulatedProgress}% مكتمل
                        </span>
                      </div>
                      <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-emerald-500 rounded-full transition-all duration-300"
                          style={{ width: `${simulatedProgress}%` }}
                        />
                      </div>
                    </div>

                    <Link
                      href={`/courses/${course.slug}`}
                      className={buttonVariants({ className: "w-full font-semibold shadow-xs" })}
                    >
                      متابعة التعلم ومشاهدة الدروس
                    </Link>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </section>
    </main>
  );
}
