import { notFound } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import { CoursePlayer } from "@/components/course-player";

export default async function CoursePlayerPage({
  params,
}: {
  params: Promise<{ host: string; courseSlug: string; lectureId: string }>;
}) {
  const { courseSlug, lectureId } = await params;
  const api = await serverApi();

  const { data: course, response } = await api.GET("/storefront/courses/{slug}", {
    params: { path: { slug: courseSlug } },
  });

  if (response.status === 404 || !course) {
    notFound();
  }

  interface FlatLecture {
    id: string;
    title: string;
    sectionTitle: string;
    isFreePreview: boolean;
    durationSec?: number;
  }

  const allLectures: FlatLecture[] = [];
  for (const s of course.sections) {
    for (const l of s.lectures) {
      allLectures.push({
        id: l.id,
        title: l.title,
        sectionTitle: s.title,
        isFreePreview: l.isFreePreview,
        durationSec: l.durationSec,
      });
    }
  }

  const currentIndex = allLectures.findIndex((l) => l.id === lectureId);
  if (currentIndex === -1 && allLectures.length > 0) {
    // If not found, show first lecture
    return (
      <CoursePlayer
        course={course}
        activeLectureId={allLectures[0].id}
        allLectures={allLectures}
        currentIndex={0}
      />
    );
  }

  return (
    <CoursePlayer
      course={course}
      activeLectureId={lectureId}
      allLectures={allLectures}
      currentIndex={currentIndex >= 0 ? currentIndex : 0}
    />
  );
}
