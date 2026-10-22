"use client";

import { useState } from "react";
import Link from "next/link";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { components } from "@/lib/api/schema";

type StorefrontCourse = components["schemas"]["StorefrontCourse"];

interface FlatLecture {
  id: string;
  title: string;
  sectionTitle: string;
  isFreePreview: boolean;
  durationSec?: number;
}

export function CoursePlayer({
  course,
  allLectures,
  currentIndex,
}: {
  course: StorefrontCourse;
  activeLectureId?: string;
  allLectures: FlatLecture[];
  currentIndex: number;
}) {
  const currentLecture = allLectures[currentIndex] || allLectures[0];

  const [completedLectures, setCompletedLectures] = useState<Record<string, boolean>>({});
  const [activeTab, setActiveTab] = useState<"notes" | "resources" | "qa">("notes");
  const [qaInput, setQaInput] = useState("");
  const [qaList, setQaList] = useState([
    {
      author: "محمد السالم",
      time: "منذ ساعتين",
      text: "هل يُفضّل تطبيق هذا الكود في بيئة Docker محلياً أم مباشرة على نظام التشغيل؟",
      answer: "يُفضّل استخدام Docker لضمان تطابق البيئة التطويرية مع بيئة الإنتاج.",
    },
  ]);

  const isCompleted = !!completedLectures[currentLecture.id];

  function toggleComplete() {
    setCompletedLectures((prev) => ({
      ...prev,
      [currentLecture.id]: !prev[currentLecture.id],
    }));
  }

  const prevLecture = currentIndex > 0 ? allLectures[currentIndex - 1] : null;
  const nextLecture = currentIndex < allLectures.length - 1 ? allLectures[currentIndex + 1] : null;

  const completedCount = Object.values(completedLectures).filter(Boolean).length;
  const progressPct = Math.round((completedCount / allLectures.length) * 100);

  function handleAddQuestion(e: React.FormEvent) {
    e.preventDefault();
    if (!qaInput.trim()) return;
    setQaList([
      {
        author: "طالب متدرب",
        time: "الآن",
        text: qaInput.trim(),
        answer: "شكرًا لسؤالك! سيقوم المدرب بالإجابة في أقرب وقت.",
      },
      ...qaList,
    ]);
    setQaInput("");
  }

  return (
    <div className="flex h-screen flex-col bg-background overflow-hidden text-foreground selection:bg-primary/20 selection:text-primary">
      {/* Top Learning Bar */}
      <header className="h-14 border-b bg-card/70 backdrop-blur-md flex items-center justify-between px-4 sm:px-6 shrink-0 z-20">
        <div className="flex items-center gap-3">
          <Link
            href={`/courses/${course.slug}`}
            className="text-xs font-semibold text-muted-foreground hover:text-foreground flex items-center gap-1.5 transition-colors"
          >
            <span>←</span>
            <span>تفاصيل الدورة</span>
          </Link>
          <span className="text-muted-foreground/30">|</span>
          <span className="text-sm font-bold truncate max-w-[200px] sm:max-w-md">
            {course.title}
          </span>
        </div>

        {/* Course Progress Indicator */}
        <div className="flex items-center gap-4">
          <div className="hidden sm:flex items-center gap-2">
            <span className="text-xs text-muted-foreground">نسبة إنجازك:</span>
            <div className="w-28 h-2 bg-muted rounded-full overflow-hidden">
              <div
                className="h-full bg-emerald-500 transition-all duration-500 rounded-full"
                style={{ width: `${progressPct}%` }}
              />
            </div>
            <span className="text-xs font-bold text-emerald-600 font-mono">{progressPct}%</span>
          </div>

          <Button
            size="sm"
            variant={isCompleted ? "default" : "outline"}
            onClick={toggleComplete}
            className={`text-xs h-8 gap-1.5 font-semibold transition-all ${
              isCompleted ? "bg-emerald-600 hover:bg-emerald-700 text-white" : ""
            }`}
          >
            <span>{isCompleted ? "✓ مكتمل" : "تحديد كمكتمل"}</span>
          </Button>
        </div>
      </header>

      {/* Main Workspace (Player + Outline) */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Area: Video Player & Tabs */}
        <main className="flex-1 overflow-y-auto p-4 sm:p-8 flex flex-col justify-between">
          <div className="mx-auto w-full max-w-4xl space-y-6">
            {/* Header info */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b pb-4">
              <div>
                <span className="text-xs font-bold text-primary uppercase tracking-wider">
                  {currentLecture.sectionTitle}
                </span>
                <div className="flex items-center gap-3 mt-1">
                  <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">
                    {currentLecture.title}
                  </h1>
                  {currentLecture.isFreePreview && (
                    <span className="text-[11px] font-bold bg-sky-500/10 text-sky-600 border border-sky-500/20 px-2.5 py-0.5 rounded-full">
                      معاينة مجانية
                    </span>
                  )}
                </div>
              </div>

              <div className="text-xs text-muted-foreground font-mono">
                درس {currentIndex + 1} من {allLectures.length}
              </div>
            </div>

            {/* Video Player Display Container */}
            <div className="relative rounded-2xl overflow-hidden border border-border/80 bg-slate-950 shadow-2xl">
              <div className="relative aspect-video w-full flex flex-col items-center justify-center p-8 text-white">
                {/* Simulated ambient lighting */}
                <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-black/60 pointer-events-none" />

                <div className="relative z-10 flex flex-col items-center gap-3">
                  <div className="h-20 w-20 rounded-full bg-primary/90 text-primary-foreground flex items-center justify-center shadow-xl shadow-primary/30 cursor-pointer hover:scale-110 hover:bg-primary transition-all">
                    <span className="text-3xl mr-1">▶</span>
                  </div>
                  <h3 className="font-bold text-lg tracking-wide">{currentLecture.title}</h3>
                  <p className="text-xs text-slate-400">مشغل الفيديو عالي الدقة (1080p Full HD)</p>
                </div>

                {/* Video Controls Bar */}
                <div className="absolute bottom-0 inset-x-0 p-4 bg-gradient-to-t from-black via-black/80 to-transparent flex items-center justify-between text-xs text-slate-300 z-10 font-mono">
                  <div className="flex items-center gap-3">
                    <span className="cursor-pointer hover:text-white">▶ تشغيل</span>
                    <span className="text-slate-500">|</span>
                    <span>00:00 / 14:30</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="bg-white/10 px-2 py-0.5 rounded cursor-pointer hover:bg-white/20">
                      1.0x
                    </span>
                    <span className="cursor-pointer hover:text-white">HD</span>
                    <span className="cursor-pointer hover:text-white">⛶ ملء الشاشة</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Interactive Tabbed Content (Notes / Resources / Q&A) */}
            <div className="space-y-4 pt-2">
              <div className="flex border-b border-border/80 gap-6 text-sm font-semibold">
                <button
                  type="button"
                  onClick={() => setActiveTab("notes")}
                  className={`pb-3 transition-colors relative ${
                    activeTab === "notes"
                      ? "text-primary border-b-2 border-primary"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  📝 الشرح والملاحظات
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("resources")}
                  className={`pb-3 transition-colors relative ${
                    activeTab === "resources"
                      ? "text-primary border-b-2 border-primary"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  📁 المرفقات والأكواد
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("qa")}
                  className={`pb-3 transition-colors relative ${
                    activeTab === "qa"
                      ? "text-primary border-b-2 border-primary"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  💬 النقاش والأسئلة ({qaList.length})
                </button>
              </div>

              {activeTab === "notes" && (
                <div className="rounded-xl border bg-card p-6 text-sm text-muted-foreground space-y-3 leading-relaxed">
                  <h4 className="font-bold text-foreground text-base">ملاحظات الدرس وتفاصيل التطبيق</h4>
                  <p>
                    في هذا الدرس العملي، نستعرض المفاهيم المعمارية، وكيفية تنظيم الكود المصدري وفق
                    معايير هندسية تضمن قابلية التوسع والصيانة. ننصح بتطبيق كل خطوة بشكل عملي وكتابة
                    الأكواد يدوياً.
                  </p>
                  <p>
                    تأكد من مراجعة المرفقات في التبويب المجاور لتحميل الكود المصدري الكامل لهذا
                    الدرس.
                  </p>
                </div>
              )}

              {activeTab === "resources" && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between p-4 rounded-xl border bg-card hover:border-primary/40 transition-colors">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">📦</span>
                      <div>
                        <h5 className="font-bold text-sm">source-code-lecture-{currentIndex + 1}.zip</h5>
                        <span className="text-xs text-muted-foreground">الكود المصدري الكامل • 1.4 MB</span>
                      </div>
                    </div>
                    <Button size="sm" variant="outline" className="text-xs">
                      تحميل الملف
                    </Button>
                  </div>

                  <div className="flex items-center justify-between p-4 rounded-xl border bg-card hover:border-primary/40 transition-colors">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">📄</span>
                      <div>
                        <h5 className="font-bold text-sm">lecture-cheatsheet-summary.pdf</h5>
                        <span className="text-xs text-muted-foreground">ملخص المفاهيم والروابط • 450 KB</span>
                      </div>
                    </div>
                    <Button size="sm" variant="outline" className="text-xs">
                      تحميل PDF
                    </Button>
                  </div>
                </div>
              )}

              {activeTab === "qa" && (
                <div className="space-y-4">
                  <form onSubmit={handleAddQuestion} className="space-y-2">
                    <textarea
                      value={qaInput}
                      onChange={(e) => setQaInput(e.target.value)}
                      placeholder="اطرح سؤالك أو استفسارك حول هذا الدرس هنا…"
                      rows={3}
                      className="w-full rounded-xl border border-input bg-card p-3 text-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    />
                    <Button type="submit" size="sm" disabled={!qaInput.trim()}>
                      نشر السؤال
                    </Button>
                  </form>

                  <div className="space-y-3 pt-2">
                    {qaList.map((item, idx) => (
                      <Card key={idx} className="p-4 space-y-2 bg-muted/20">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-bold">{item.author}</span>
                          <span className="text-muted-foreground font-mono">{item.time}</span>
                        </div>
                        <p className="text-sm font-medium">{item.text}</p>
                        {item.answer && (
                          <div className="rounded-lg bg-primary/5 border border-primary/15 p-3 text-xs space-y-1 text-primary">
                            <span className="font-bold">رد المدرب:</span>
                            <p className="text-muted-foreground">{item.answer}</p>
                          </div>
                        )}
                      </Card>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Navigation Footer */}
          <div className="mx-auto w-full max-w-4xl pt-8 border-t mt-12 flex items-center justify-between">
            {prevLecture ? (
              <Link
                href={`/learn/${course.slug}/${prevLecture.id}`}
                className={buttonVariants({ variant: "outline", size: "sm", className: "gap-2" })}
              >
                <span>←</span>
                <span>الدرس السابق: {prevLecture.title}</span>
              </Link>
            ) : (
              <div />
            )}

            {nextLecture ? (
              <Link
                href={`/learn/${course.slug}/${nextLecture.id}`}
                className={buttonVariants({ size: "sm", className: "gap-2 font-semibold shadow-xs" })}
              >
                <span>الدرس التالي: {nextLecture.title}</span>
                <span>→</span>
              </Link>
            ) : (
              <Link
                href={`/courses/${course.slug}`}
                className={buttonVariants({ size: "sm", className: "font-semibold" })}
              >
                🎉 إتمام المساق
              </Link>
            )}
          </div>
        </main>

        {/* Sidebar Curriculum (Right side in RTL) */}
        <aside className="w-80 border-s bg-muted/15 overflow-y-auto hidden md:flex flex-col shrink-0">
          <div className="p-4 border-b bg-muted/30 flex items-center justify-between">
            <span className="font-bold text-sm">محتوى الدورة</span>
            <span className="text-xs font-mono text-muted-foreground">
              {completedCount} / {allLectures.length} مكتمل
            </span>
          </div>

          <div className="p-3 space-y-5">
            {course.sections.map((sec, sIdx) => (
              <div key={sec.id} className="space-y-1.5">
                <div className="px-2.5 text-xs font-bold text-muted-foreground">
                  قسم {sIdx + 1}: {sec.title}
                </div>
                <div className="space-y-1">
                  {sec.lectures.map((lec) => {
                    const isActive = lec.id === currentLecture.id;
                    const isDone = !!completedLectures[lec.id];
                    return (
                      <Link
                        key={lec.id}
                        href={`/learn/${course.slug}/${lec.id}`}
                        className={`flex items-center justify-between px-3 py-2.5 text-xs rounded-xl transition-all ${
                          isActive
                            ? "bg-primary text-primary-foreground font-semibold shadow-xs"
                            : "hover:bg-muted text-foreground"
                        }`}
                      >
                        <div className="flex items-center gap-2.5 truncate max-w-[200px]">
                          <span
                            className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full text-[10px] ${
                              isDone
                                ? "bg-emerald-500 text-white font-bold"
                                : isActive
                                ? "border border-white/60 text-white"
                                : "border border-border text-muted-foreground"
                            }`}
                          >
                            {isDone ? "✓" : ""}
                          </span>
                          <span className="truncate">{lec.title}</span>
                        </div>

                        {lec.isFreePreview && !isActive && (
                          <span className="text-[10px] text-sky-600 bg-sky-500/10 px-1.5 py-0.5 rounded font-medium">
                            مجاني
                          </span>
                        )}
                      </Link>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </aside>
      </div>
    </div>
  );
}
