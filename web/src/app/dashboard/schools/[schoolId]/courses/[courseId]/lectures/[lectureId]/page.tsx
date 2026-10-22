import Link from "next/link";
import { redirect } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import { LectureEditor } from "@/components/lecture-editor";

export default async function LectureDetailPage({
  params,
}: {
  params: Promise<{ schoolId: string; courseId: string; lectureId: string }>;
}) {
  const { schoolId, courseId, lectureId } = await params;
  const api = await serverApi();

  const { response: authRes } = await api.GET("/accounts/me");
  if (authRes.status === 401) redirect("/login");

  const { data: detail, response } = await api.GET(
    "/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}",
    {
      params: { path: { schoolId, courseId, lectureId } },
    }
  );

  if (response.status === 404 || !detail) {
    redirect(`/schools/${schoolId}/courses/${courseId}`);
  }

  return (
    <main className="mx-auto grid w-full max-w-4xl gap-6 p-6">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link href="/" className="hover:underline">
          لوحة التحكم
        </Link>
        <span>/</span>
        <Link href={`/schools/${schoolId}/courses`} className="hover:underline">
          الدورات
        </Link>
        <span>/</span>
        <Link href={`/schools/${schoolId}/courses/${courseId}`} className="hover:underline">
          المنهج
        </Link>
        <span>/</span>
        <span className="text-foreground font-medium">{detail.lecture.title}</span>
      </div>

      <LectureEditor schoolId={schoolId} courseId={courseId} initialData={detail} />
    </main>
  );
}
