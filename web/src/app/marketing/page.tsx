import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardTitle } from "@/components/ui/card";
import { hosts, originFor } from "@/lib/hosts";
import { requestHost, requestProto } from "@/lib/api/server";

export default async function MarketingPage() {
  const t = await getTranslations();
  const app = originFor(await requestHost(), hosts.appHost, await requestProto());

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      {/* Sticky Blurred Header */}
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <div className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary text-primary-foreground font-black text-sm shadow-xs">
              T
            </span>
            <span className="text-xl font-bold tracking-tight">
              {t("common.brand")}
            </span>
          </div>

          <nav className="flex items-center gap-3">
            <LocaleSwitcher />
            <Link
              href={`${app}/login`}
              className={buttonVariants({ variant: "outline", size: "sm", className: "text-xs font-semibold" })}
            >
              {t("marketing.login")}
            </Link>
            <Link
              href={`${app}/signup`}
              className={buttonVariants({ size: "sm", className: "text-xs font-semibold shadow-xs" })}
            >
              {t("marketing.cta")}
            </Link>
          </nav>
        </div>
      </header>

      {/* Hero Section with Ambient Lights */}
      <section className="relative overflow-hidden border-b bg-gradient-to-b from-primary/10 via-background/60 to-background py-24 sm:py-32 px-6 text-center">
        <div className="absolute top-0 right-1/4 -z-10 h-96 w-96 rounded-full bg-primary/15 blur-3xl" />
        <div className="absolute bottom-0 left-1/4 -z-10 h-80 w-80 rounded-full bg-sky-500/10 blur-3xl" />

        <div className="mx-auto max-w-4xl space-y-6">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-4 py-1.5 text-xs font-semibold text-primary">
            <span>🚀</span>
            <span>المنصة الرائدة لصناع المحتوى والأكاديميات في الخليج</span>
          </div>

          <h1 className="text-4xl sm:text-6xl font-extrabold tracking-tight leading-tight sm:leading-tight">
            {t("marketing.title")}
          </h1>

          <p className="max-w-2xl mx-auto text-lg sm:text-xl text-muted-foreground leading-relaxed">
            {t("marketing.subtitle")}
          </p>

          <div className="flex flex-wrap items-center justify-center gap-4 pt-4">
            <Link
              href={`${app}/signup`}
              className={buttonVariants({
                size: "lg",
                className: "px-8 font-bold shadow-xl shadow-primary/20 text-base",
              })}
            >
              {t("marketing.cta")}
            </Link>
            <a
              href="http://demo.localhost:3000"
              className={buttonVariants({
                variant: "outline",
                size: "lg",
                className: "px-6 font-semibold",
              })}
            >
              معاينة متجر الأكاديمية الحي
            </a>
          </div>

          {/* Social Proof Badges */}
          <div className="flex flex-wrap items-center justify-center gap-6 pt-10 text-xs text-muted-foreground font-medium">
            <span className="flex items-center gap-1.5">
              <span>💳</span>
              <span>مدى، Apple Pay، وبطاقات الائتمان</span>
            </span>
            <span>•</span>
            <span className="flex items-center gap-1.5">
              <span>⚡</span>
              <span>بث فيديو مشفر عالي السرعة (Bunny Stream)</span>
            </span>
            <span>•</span>
            <span className="flex items-center gap-1.5">
              <span>🔒</span>
              <span>عزل تام للبيانات (Multi-Tenancy RLS)</span>
            </span>
          </div>
        </div>
      </section>

      {/* Feature Pillars Grid */}
      <main className="mx-auto w-full max-w-6xl px-6 py-20 space-y-16">
        <div className="text-center space-y-3 max-w-2xl mx-auto">
          <h2 className="text-3xl font-extrabold tracking-tight">كل ما تحتاجه لإطلاق أكاديميتك والربح منها</h2>
          <p className="text-sm text-muted-foreground">
            بنية تحتية هندسية متكاملة مصممة خصيصاً لتمنحك ولطلابك تجربة تعليمية استثنائية.
          </p>
        </div>

        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-primary/10 text-primary flex items-center justify-center text-2xl font-bold">
              🎓
            </div>
            <CardTitle className="text-lg font-bold">محرر المناهج التفاعلي</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              نظم دوراتك في وحدات وأقسام، وأضف الفيديوهات، المقالات النصية التفاعلية، وملفات الـ PDF بسهولة.
            </p>
          </Card>

          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-emerald-500/10 text-emerald-600 flex items-center justify-center text-2xl font-bold">
              💰
            </div>
            <CardTitle className="text-lg font-bold">مدفوعات سعودية وخليجية</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              تحصيل مباشر لأرباحك عبر ميسر لدعم شبكة مدى السعودية، Apple Pay، وبطاقات الائتمان دون وسيط.
            </p>
          </Card>

          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-sky-500/10 text-sky-600 flex items-center justify-center text-2xl font-bold">
              📺
            </div>
            <CardTitle className="text-lg font-bold">بث فيديو آمن ضد القرصنة</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              تشفير كامل لروابط المشاهدة عبر تقنية Signed HLS Tokens لمنع سرقة المحتوى أو تحميله بدون تصريح.
            </p>
          </Card>

          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-amber-500/10 text-amber-600 flex items-center justify-center text-2xl font-bold">
              🌐
            </div>
            <CardTitle className="text-lg font-bold">نطاق مخصص وهوية بصرية</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              اربط أكاديميتك بنطاقك الخاص (yourdomain.com) مع شهادات أمان SSL مجانية تلقائية لكل مدرسة.
            </p>
          </Card>

          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-purple-500/10 text-purple-600 flex items-center justify-center text-2xl font-bold">
              📊
            </div>
            <CardTitle className="text-lg font-bold">تتبع الطلاب والإنجاز</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              لوحة تحكم فورية تعرض نسب إكمال الدورات، ثواني المشاهدة، وتفاعل الطلاب مع كل درس.
            </p>
          </Card>

          <Card className="p-6 space-y-3 border-border/80 bg-card hover:border-primary/40 transition-colors shadow-xs">
            <div className="h-12 w-12 rounded-xl bg-rose-500/10 text-rose-600 flex items-center justify-center text-2xl font-bold">
              🧾
            </div>
            <CardTitle className="text-lg font-bold">فواتير ضريبية ZATCA</CardTitle>
            <p className="text-xs text-muted-foreground leading-relaxed">
              إصدار فواتير ضريبية متوافقة مع متطلبات هيئة الزكاة والضريبة والجمارك مع رمز الاستجابة السريع (QR Code).
            </p>
          </Card>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-card py-10 px-6 text-center text-xs text-muted-foreground">
        <p>© {new Date().getFullYear()} Teatchable — جميع الحقوق محفوظة.</p>
      </footer>
    </div>
  );
}
