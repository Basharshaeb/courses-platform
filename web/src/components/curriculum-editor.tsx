"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api/client";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { components } from "@/lib/api/schema";

type CourseDetail = components["schemas"]["CourseDetail"];

export function CurriculumEditor({
  schoolId,
  initialData,
}: {
  schoolId: string;
  initialData: CourseDetail;
}) {
  const router = useRouter();
  const [data, setData] = useState(initialData);
  const [statusLoading, setStatusLoading] = useState(false);
  const [newSectionTitle, setNewSectionTitle] = useState("");
  const [addingSection, setAddingSection] = useState(false);
  const [activeSectionId, setActiveSectionId] = useState<string | null>(null);
  const [newLectureTitle, setNewLectureTitle] = useState("");

  const course = data.course;
  const isPublished = course.status === "published";

  async function togglePublish() {
    setStatusLoading(true);
    const endpoint = isPublished
      ? `/admin/schools/{schoolId}/courses/{courseId}/unpublish`
      : `/admin/schools/{schoolId}/courses/{courseId}/publish`;

    const { data: updated, error } = await api.POST(endpoint, {
      params: { path: { schoolId, courseId: course.id } },
    });
    setStatusLoading(false);

    if (error) {
      alert(error.detail || "حدث خطأ أثناء تغيير حالة الدورة");
      return;
    }

    if (updated) {
      setData((prev) => ({
        ...prev,
        course: { ...prev.course, status: updated.status, publishedAt: updated.publishedAt },
      }));
      router.refresh();
    }
  }

  async function handleAddSection(e: React.FormEvent) {
    e.preventDefault();
    if (!newSectionTitle.trim()) return;

    setAddingSection(true);
    const { data: sec, error } = await api.POST("/admin/schools/{schoolId}/courses/{courseId}/sections", {
      params: { path: { schoolId, courseId: course.id } },
      body: { title: newSectionTitle.trim() },
    });
    setAddingSection(false);

    if (error) {
      alert(error.detail || "تعذر إضافة القسم");
      return;
    }

    if (sec) {
      setNewSectionTitle("");
      setData((prev) => ({
        ...prev,
        sections: [...prev.sections, sec],
      }));
      router.refresh();
    }
  }

  async function handleDeleteSection(sectionId: string) {
    if (!confirm("هل أنت متأكد من حذف هذا القسم وجميع دروسه؟")) return;

    const { error } = await api.DELETE("/admin/schools/{schoolId}/courses/{courseId}/sections/{sectionId}", {
      params: { path: { schoolId, courseId: course.id, sectionId } },
    });

    if (error) {
      alert(error.detail || "تعذر حذف القسم");
      return;
    }

    setData((prev) => ({
      ...prev,
      sections: prev.sections.filter((s) => s.id !== sectionId),
    }));
    router.refresh();
  }

  async function handleAddLecture(e: React.FormEvent, sectionId: string) {
    e.preventDefault();
    if (!newLectureTitle.trim()) return;

    const { data: lec, error } = await api.POST(
      "/admin/schools/{schoolId}/courses/{courseId}/sections/{sectionId}/lectures",
      {
        params: { path: { schoolId, courseId: course.id, sectionId } },
        body: { title: newLectureTitle.trim() },
      }
    );

    if (error) {
      alert(error.detail || "تعذر إضافة الدرس");
      return;
    }

    if (lec) {
      setNewLectureTitle("");
      setActiveSectionId(null);
      setData((prev) => ({
        ...prev,
        sections: prev.sections.map((s) =>
          s.id === sectionId ? { ...s, lectures: [...s.lectures, lec] } : s
        ),
      }));
      router.refresh();
    }
  }

  async function handleDeleteLecture(lectureId: string) {
    if (!confirm("هل أنت متأكد من حذف هذا الدرس؟")) return;

    const { error } = await api.DELETE("/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}", {
      params: { path: { schoolId, courseId: course.id, lectureId } },
    });

    if (error) {
      alert(error.detail || "تعذر حذف الدرس");
      return;
    }

    setData((prev) => ({
      ...prev,
      sections: prev.sections.map((s) => ({
        ...s,
        lectures: s.lectures.filter((l) => l.id !== lectureId),
      })),
    }));
    router.refresh();
  }

  return (
    <div className="grid gap-6">
      {/* Course Action Header */}
      <Card>
        <CardContent className="flex flex-col gap-4 p-6 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="flex items-center gap-3">
              <span
                className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${
                  isPublished
                    ? "bg-emerald-500/10 text-emerald-600 border border-emerald-500/20"
                    : "bg-amber-500/10 text-amber-600 border border-amber-500/20"
                }`}
              >
                {isPublished ? "منشورة للطلاب" : "مسودة خاصة"}
              </span>
              <span className="text-sm text-muted-foreground" dir="ltr">
                /{course.slug}
              </span>
            </div>
            <h2 className="text-xl font-bold mt-1">{course.title}</h2>
            {course.subtitle && <p className="text-sm text-muted-foreground">{course.subtitle}</p>}
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant={isPublished ? "outline" : "default"}
              onClick={togglePublish}
              disabled={statusLoading}
            >
              {statusLoading ? "جارٍ التحديث…" : isPublished ? "إلغاء النشر (تحويل لمسودة)" : "نشر الدورة الآن"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Curriculum Sections */}
      <div className="grid gap-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">المنهج الدراسي والدروس</h3>
          <span className="text-sm text-muted-foreground">
            {data.sections.length} أقسام ·{" "}
            {data.sections.reduce((acc, s) => acc + s.lectures.length, 0)} درس
          </span>
        </div>

        {data.sections.map((section, sIdx) => (
          <Card key={section.id} className="overflow-hidden border-border/80">
            <CardHeader className="bg-muted/40 py-3 px-4 flex flex-row items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-muted-foreground">قسم {sIdx + 1}:</span>
                <CardTitle className="text-base font-semibold">{section.title}</CardTitle>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-xs text-destructive hover:bg-destructive/10"
                  onClick={() => handleDeleteSection(section.id)}
                >
                  حذف القسم
                </Button>
              </div>
            </CardHeader>
            <CardContent className="p-4 grid gap-3">
              {section.lectures.length === 0 ? (
                <p className="text-xs text-muted-foreground py-2 text-center">لا توجد دروس في هذا القسم بعد.</p>
              ) : (
                <div className="divide-y rounded-md border bg-background">
                  {section.lectures.map((lec, lIdx) => (
                    <div
                      key={lec.id}
                      className="flex items-center justify-between p-3 hover:bg-muted/30 transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <span className="text-xs text-muted-foreground font-mono">{lIdx + 1}</span>
                        <span className="font-medium text-sm">{lec.title}</span>
                        {lec.isFreePreview && (
                          <span className="text-[10px] bg-sky-500/10 text-sky-600 border border-sky-500/20 px-2 py-0.5 rounded-full font-medium">
                            معاينة مجانية
                          </span>
                        )}
                        <span className="text-xs text-muted-foreground">
                          ({lec.blockCount} كتل محتوى)
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Link
                          href={`/schools/${schoolId}/courses/${course.id}/lectures/${lec.id}`}
                          className={buttonVariants({ variant: "outline", size: "sm" })}
                        >
                          محتوى الدرس
                        </Link>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-destructive hover:bg-destructive/10 text-xs px-2"
                          onClick={() => handleDeleteLecture(lec.id)}
                        >
                          ✕
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Add Lecture inside Section */}
              {activeSectionId === section.id ? (
                <form
                  onSubmit={(e) => handleAddLecture(e, section.id)}
                  className="flex items-center gap-2 pt-2"
                >
                  <Input
                    value={newLectureTitle}
                    onChange={(e) => setNewLectureTitle(e.target.value)}
                    placeholder="عنوان الدرس الجديد…"
                    className="h-8 text-sm"
                    autoFocus
                  />
                  <Button type="submit" size="sm" className="h-8">
                    إضافة
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-8"
                    onClick={() => setActiveSectionId(null)}
                  >
                    إلغاء
                  </Button>
                </form>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full border-dashed text-xs text-muted-foreground hover:text-foreground"
                  onClick={() => {
                    setActiveSectionId(section.id);
                    setNewLectureTitle("");
                  }}
                >
                  + إضافة درس إلى هذا القسم
                </Button>
              )}
            </CardContent>
          </Card>
        ))}

        {/* Add New Section Form */}
        <Card className="border-dashed p-4">
          <form onSubmit={handleAddSection} className="flex flex-col sm:flex-row items-center gap-2">
            <Input
              value={newSectionTitle}
              onChange={(e) => setNewSectionTitle(e.target.value)}
              placeholder="عنوان القسم الجديد (مثال: الوحدة الأولى: الأساسيات)…"
              disabled={addingSection}
            />
            <Button type="submit" disabled={addingSection || !newSectionTitle.trim()} className="w-full sm:w-auto shrink-0">
              {addingSection ? "جارٍ الإضافة…" : "+ إضافة قسم جديد"}
            </Button>
          </form>
        </Card>
      </div>
    </div>
  );
}
