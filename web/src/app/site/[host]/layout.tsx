import Link from "next/link";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { buttonVariants } from "@/components/ui/button";
import { serverApi } from "@/lib/api/server";

export async function generateMetadata() {
  const { data } = await (await serverApi()).GET("/storefront/school");
  return {
    title: data?.name ? `${data.name} — أكاديمية تعليمية احترافية` : "Teatchable",
    description: "تعلّم المهارات التقنية وبناء الأنظمة مع أفضل المناهج والشهادات المعتمدة.",
  };
}

/** Storefront chrome for a school. The school comes from the request host, not the URL param. */
export default async function SchoolLayout({ children }: LayoutProps<"/site/[host]">) {
  const { data: school, response } = await (await serverApi()).GET("/storefront/school");
  if (response.status === 404) notFound();
  if (!school) throw new Error(`storefront/school failed: ${response.status}`);

  const t = await getTranslations("school");

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      {/* Sticky Top Navigation Bar */}
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <Link href="/" className="flex items-center gap-2.5 group">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary text-primary-foreground font-black text-sm shadow-xs group-hover:scale-105 transition-transform">
              {school.name.slice(0, 1)}
            </span>
            <span className="text-lg font-bold tracking-tight group-hover:text-primary transition-colors">
              {school.name}
            </span>
          </Link>

          <nav className="flex items-center gap-3 sm:gap-4">
            <Link
              href="/"
              className="text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors hidden sm:inline-block"
            >
              جميع الدورات
            </Link>

            <Link
              href="/library"
              className={buttonVariants({ variant: "ghost", size: "sm", className: "text-xs font-semibold" })}
            >
              {t("library")}
            </Link>

            <div className="h-4 w-px bg-border/60 mx-1 hidden sm:block" />

            <LocaleSwitcher />

            <Link
              href="/login"
              className={buttonVariants({ variant: "outline", size: "sm", className: "text-xs font-semibold" })}
            >
              {t("login")}
            </Link>

            <Link
              href="/signup"
              className={buttonVariants({ size: "sm", className: "text-xs font-semibold shadow-xs hidden sm:inline-flex" })}
            >
              {t("signup")}
            </Link>
          </nav>
        </div>
      </header>

      {/* Page Body */}
      <div className="flex-1">{children}</div>

      {/* Premium Footer */}
      <footer className="border-t bg-card py-12 px-6">
        <div className="mx-auto max-w-6xl flex flex-col sm:flex-row items-center justify-between gap-6 text-xs text-muted-foreground">
          <div className="flex items-center gap-3">
            <span className="flex h-6 w-6 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold text-xs">
              {school.name.slice(0, 1)}
            </span>
            <span className="font-semibold text-foreground">{school.name}</span>
            <span>• جميع الحقوق محفوظة © {new Date().getFullYear()}</span>
          </div>

          <div className="flex items-center gap-4">
            <span>🔒 منصة تعليمية مشفرة ببروتوكول SSL</span>
            <span>•</span>
            <span>دعم فني متواصل للطلاب</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
