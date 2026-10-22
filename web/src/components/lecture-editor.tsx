"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { components } from "@/lib/api/schema";

type LectureDetail = components["schemas"]["LectureDetail"];
type BlockKind = components["schemas"]["BlockKind"];

export function LectureEditor({
  schoolId,
  courseId,
  initialData,
}: {
  schoolId: string;
  courseId: string;
  initialData: LectureDetail;
}) {
  const router = useRouter();
  const [data, setData] = useState(initialData);
  const [isFreePreview, setIsFreePreview] = useState(data.lecture.isFreePreview);
  const [updatingMeta, setUpdatingMeta] = useState(false);

  // New block state
  const [selectedKind, setSelectedKind] = useState<BlockKind>("text");
  const [textHtml, setTextHtml] = useState("");
  const [embedUrl, setEmbedUrl] = useState("");
  const [pdfFilename, setPdfFilename] = useState("");
  const [addingBlock, setAddingBlock] = useState(false);

  async function handleToggleFreePreview() {
    setUpdatingMeta(true);
    const nextVal = !isFreePreview;
    const { error } = await api.PATCH("/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}", {
      params: { path: { schoolId, courseId, lectureId: data.lecture.id } },
      body: { isFreePreview: nextVal },
    });
    setUpdatingMeta(false);

    if (error) {
      alert(error.detail || "تعذر تعديل حالة المعاينة المجانية");
      return;
    }

    setIsFreePreview(nextVal);
    router.refresh();
  }

  async function handleAddBlock(e: React.FormEvent) {
    e.preventDefault();
    setAddingBlock(true);

    let blockData = {};
    if (selectedKind === "text") {
      if (!textHtml.trim()) {
        setAddingBlock(false);
        return;
      }
      blockData = { html: textHtml };
    } else if (selectedKind === "embed") {
      if (!embedUrl.trim()) {
        setAddingBlock(false);
        return;
      }
      blockData = { url: embedUrl.trim() };
    } else if (selectedKind === "pdf") {
      if (!pdfFilename.trim()) {
        setAddingBlock(false);
        return;
      }
      blockData = { filename: pdfFilename.trim(), fileKey: "mock-pdf-key" };
    }

    const { data: newBlock, error } = await api.POST(
      "/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks",
      {
        params: { path: { schoolId, courseId, lectureId: data.lecture.id } },
        body: {
          kind: selectedKind,
          data: blockData,
        },
      }
    );
    setAddingBlock(false);

    if (error) {
      alert(error.detail || "تعذر إضافة محتوى الدرس");
      return;
    }

    if (newBlock) {
      setTextHtml("");
      setEmbedUrl("");
      setPdfFilename("");
      setData((prev) => ({
        ...prev,
        blocks: [...prev.blocks, newBlock],
      }));
      router.refresh();
    }
  }

  async function handleDeleteBlock(blockId: string) {
    if (!confirm("هل أنت متأكد من حذف هذا المحتوى؟")) return;

    const { error } = await api.DELETE(
      "/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks/{blockId}",
      {
        params: { path: { schoolId, courseId, lectureId: data.lecture.id, blockId } }
      }
    );

    if (error) {
      alert(error.detail || "تعذر حذف المحتوى");
      return;
    }

    setData((prev) => ({
      ...prev,
      blocks: prev.blocks.filter((b) => b.id !== blockId),
    }));
    router.refresh();
  }

  return (
    <div className="grid gap-6">
      {/* Lecture Header Card */}
      <Card>
        <CardContent className="flex flex-col gap-4 p-6 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <span className="text-xs font-semibold text-muted-foreground uppercase">محرر الدرس</span>
            <h2 className="text-xl font-bold mt-1">{data.lecture.title}</h2>
          </div>

          <div className="flex items-center gap-3">
            <label className="flex items-center gap-2 cursor-pointer text-sm font-medium">
              <input
                type="checkbox"
                checked={isFreePreview}
                onChange={handleToggleFreePreview}
                disabled={updatingMeta}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <span>متاح كمعاينة مجانية للزوار</span>
            </label>
          </div>
        </CardContent>
      </Card>

      {/* Existing Blocks */}
      <div className="grid gap-4">
        <h3 className="text-lg font-semibold">محتويات الدرس ({data.blocks.length})</h3>

        {data.blocks.length === 0 ? (
          <Card className="border-dashed p-8 text-center text-muted-foreground">
            لا توجد كتل محتوى مضافة لهذا الدرس بعد. أضف نصاً أو فيديو أدناه!
          </Card>
        ) : (
          data.blocks.map((block, idx) => (
            <Card key={block.id} className="relative overflow-hidden">
              <CardHeader className="bg-muted/30 py-2.5 px-4 flex flex-row items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono text-muted-foreground">#{idx + 1}</span>
                  <span className="text-xs font-semibold uppercase tracking-wider bg-primary/10 text-primary px-2 py-0.5 rounded">
                    {block.kind === "text"
                      ? "مقال / نص"
                      : block.kind === "video"
                      ? "فيديو"
                      : block.kind === "pdf"
                      ? "مستند PDF"
                      : "تضمين خارجي"}
                  </span>
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-xs text-destructive hover:bg-destructive/10"
                  onClick={() => handleDeleteBlock(block.id)}
                >
                  حذف الكتلة
                </Button>
              </CardHeader>
              <CardContent className="p-4">
                {block.kind === "text" && (
                  <div
                    className="prose dark:prose-invert max-w-none text-sm"
                    dangerouslySetInnerHTML={{ __html: block.data.html || "" }}
                  />
                )}
                {block.kind === "embed" && (
                  <div className="text-sm font-mono bg-muted p-3 rounded">
                    URL: {block.data.url}
                  </div>
                )}
                {block.kind === "pdf" && (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>📄</span>
                    <span>{block.data.filename || "مستند PDF"}</span>
                  </div>
                )}
                {block.kind === "video" && (
                  <div className="text-sm text-muted-foreground">
                    <span>🎬 فيديو مرفوع معرّف الأصول: {block.mediaAssetId || "قيد المعالجة"}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          ))
        )}
      </div>

      {/* Add New Block Form */}
      <Card className="border-primary/20">
        <CardHeader>
          <CardTitle className="text-base font-semibold">إضافة محتوى جديد إلى الدرس</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleAddBlock} className="grid gap-4">
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant={selectedKind === "text" ? "default" : "outline"}
                onClick={() => setSelectedKind("text")}
              >
                نص ومقال
              </Button>
              <Button
                type="button"
                size="sm"
                variant={selectedKind === "embed" ? "default" : "outline"}
                onClick={() => setSelectedKind("embed")}
              >
                تضمين (YouTube/Vimeo)
              </Button>
              <Button
                type="button"
                size="sm"
                variant={selectedKind === "pdf" ? "default" : "outline"}
                onClick={() => setSelectedKind("pdf")}
              >
                مستند PDF
              </Button>
            </div>

            {selectedKind === "text" && (
              <div className="grid gap-2">
                <Label htmlFor="block-text">المحتوى النصي (يدعم تنسيق HTML والمقالات)</Label>
                <textarea
                  id="block-text"
                  value={textHtml}
                  onChange={(e) => setTextHtml(e.target.value)}
                  rows={6}
                  placeholder="اكتب الشرح أو المقال هنا…"
                  className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-sans"
                  required
                />
              </div>
            )}

            {selectedKind === "embed" && (
              <div className="grid gap-2">
                <Label htmlFor="block-url">رابط التضمين (HTTPS URL)</Label>
                <Input
                  id="block-url"
                  value={embedUrl}
                  onChange={(e) => setEmbedUrl(e.target.value)}
                  placeholder="https://www.youtube.com/embed/…"
                  required
                />
              </div>
            )}

            {selectedKind === "pdf" && (
              <div className="grid gap-2">
                <Label htmlFor="block-pdf">اسم الملف المرفق</Label>
                <Input
                  id="block-pdf"
                  value={pdfFilename}
                  onChange={(e) => setPdfFilename(e.target.value)}
                  placeholder="ملف الشرائح - الوحدة الأولى.pdf"
                  required
                />
              </div>
            )}

            <Button type="submit" disabled={addingBlock} className="w-full sm:w-auto">
              {addingBlock ? "جارٍ الإضافة…" : "إضافة الكتلة للدرس"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
