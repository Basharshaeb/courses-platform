"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function CreateCourseForm({ schoolId }: { schoolId: string }) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [subtitle, setSubtitle] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;

    setLoading(true);
    setError(null);

    const { data, error: err } = await api.POST("/admin/schools/{schoolId}/courses", {
      params: { path: { schoolId } },
      body: {
        title: title.trim(),
        subtitle: subtitle.trim() || undefined,
        description: description.trim() || undefined,
        locale: "ar",
      },
    });

    setLoading(false);

    if (err) {
      setError(err.detail || "حدث خطأ أثناء إنشاء الدورة");
      return;
    }

    if (data) {
      setOpen(false);
      setTitle("");
      setSubtitle("");
      setDescription("");
      router.refresh();
      router.push(`/schools/${schoolId}/courses/${data.id}`);
    }
  }

  if (!open) {
    return (
      <Button onClick={() => setOpen(true)} className="gap-2">
        <span>+</span>
        <span>إنشاء دورة جديدة</span>
      </Button>
    );
  }

  return (
    <Card className="border-primary/20 shadow-md">
      <CardHeader>
        <CardTitle className="text-lg">إنشاء دورة تدريبية جديدة</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="grid gap-4">
          {error && <p className="text-sm font-medium text-destructive">{error}</p>}
          <div className="grid gap-2">
            <Label htmlFor="course-title">عنوان الدورة *</Label>
            <Input
              id="course-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="مثال: مقدمة في لغة Go وبناء الأنظمة"
              required
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="course-subtitle">وصف مختصر</Label>
            <Input
              id="course-subtitle"
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder="ملخص قصير يظهر للطلاب في المتجر"
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="course-desc">التفاصيل الكاملة</Label>
            <textarea
              id="course-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="ماذا سيتعلم الطالب في هذه الدورة؟"
            />
          </div>

          <div className="flex items-center gap-2 pt-2">
            <Button type="submit" disabled={loading}>
              {loading ? "جارٍ الإنشاء…" : "إنشاء الدورة"}
            </Button>
            <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={loading}>
              إلغاء
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
