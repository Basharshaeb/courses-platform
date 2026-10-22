import Link from "next/link";
import { redirect } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CreateCourseForm } from "@/components/course-form";

export default async function SchoolCoursesPage({
  params,
}: {
  params: Promise<{ schoolId: string }>;
}) {
  const { schoolId } = await params;
  const api = await serverApi();

  const { response: authRes } = await api.GET("/accounts/me");
  if (authRes.status === 401) redirect("/login");

  const { data: courses, response } = await api.GET("/admin/schools/{schoolId}/courses", {
    params: { path: { schoolId } },
  });

  if (response.status === 404) {
    redirect("/");
  }

  return (
    <main className="mx-auto grid w-full max-w-5xl gap-6 p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
            <Link href="/" className="hover:underline">
              لوحة التحكم
            </Link>
            <span>/</span>
            <span>الدورات</span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight">إدارة الدورات التدريبية</h1>
        </div>
        <CreateCourseForm schoolId={schoolId} />
      </div>

      {!courses || courses.length === 0 ? (
        <Card className="border-dashed p-12 text-center">
          <CardHeader>
            <CardTitle className="text-xl">لا توجد دورات مضافة بعد</CardTitle>
            <CardDescription>
              ابدأ الآن بإنشاء أول دورة تدريبية لك في الأكاديمية وأضف الدروس والمحتوى.
            </CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {courses.map((course) => (
            <Card key={course.id} className="flex flex-col justify-between transition-shadow hover:shadow-md">
              <CardHeader>
                <div className="flex items-center justify-between gap-2 mb-1">
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                      course.status === "published"
                        ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
                        : "bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20"
                    }`}
                  >
                    {course.status === "published" ? "منشورة" : "مسودة"}
                  </span>
                  <span className="text-xs text-muted-foreground" dir="ltr">
                    /{course.slug}
                  </span>
                </div>
                <CardTitle className="text-lg line-clamp-1">{course.title}</CardTitle>
                <CardDescription>
                  {course.sectionCount} أقسام · {course.lectureCount} درس
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-0">
                <Link
                  href={`/schools/${schoolId}/courses/${course.id}`}
                  className={buttonVariants({ variant: "outline", size: "sm", className: "w-full" })}
                >
                  تعديل المنهج والمحتوى
                </Link>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </main>
  );
}
