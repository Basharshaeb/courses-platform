import Link from "next/link";
import { redirect } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import { CurriculumEditor } from "@/components/curriculum-editor";

export default async function CourseDetailPage({
  params,
}: {
  params: Promise<{ schoolId: string; courseId: string }>;
}) {
  const { schoolId, courseId } = await params;
  const api = await serverApi();

  const { response: authRes } = await api.GET("/accounts/me");
  if (authRes.status === 401) redirect("/login");

  const { data: detail, response } = await api.GET("/admin/schools/{schoolId}/courses/{courseId}", {
    params: { path: { schoolId, courseId } },
  });

  if (response.status === 404 || !detail) {
    redirect(`/schools/${schoolId}/courses`);
  }

  return (
    <main className="mx-auto grid w-full max-w-5xl gap-6 p-6">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link href="/" className="hover:underline">
          لوحة التحكم
        </Link>
        <span>/</span>
        <Link href={`/schools/${schoolId}/courses`} className="hover:underline">
          الدورات
        </Link>
        <span>/</span>
        <span className="text-foreground font-medium">{detail.course.title}</span>
      </div>

      <CurriculumEditor schoolId={schoolId} initialData={detail} />
    </main>
  );
}
