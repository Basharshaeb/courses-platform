import Link from "next/link";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { buttonVariants } from "@/components/ui/button";
import { serverApi } from "@/lib/api/server";

export async function generateMetadata() {
  const { data } = await (await serverApi()).GET("/storefront/school");
  return { title: data?.name ?? "Teatchable" };
}

/** Storefront chrome for a school. The school comes from the request host, not the URL param. */
export default async function SchoolLayout({ children }: LayoutProps<"/site/[host]">) {
  const { data: school, response } = await (await serverApi()).GET("/storefront/school");
  if (response.status === 404) notFound();
  if (!school) throw new Error(`storefront/school failed: ${response.status}`);

  const t = await getTranslations("school");
  return (
    <>
      <header className="flex items-center justify-between border-b px-6 py-4">
        <Link href="/" className="text-lg font-semibold">
          {school.name}
        </Link>
        <nav className="flex items-center gap-2">
          <LocaleSwitcher />
          <Link href="/library" className={buttonVariants({ variant: "ghost", size: "sm" })}>
            {t("library")}
          </Link>
          <Link href="/login" className={buttonVariants({ variant: "outline", size: "sm" })}>
            {t("login")}
          </Link>
        </nav>
      </header>
      {children}
    </>
  );
}
