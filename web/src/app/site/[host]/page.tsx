import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { buttonVariants } from "@/components/ui/button";
import { serverApi } from "@/lib/api/server";

export default async function SchoolHome() {
  const { data: school } = await (await serverApi()).GET("/storefront/school");
  const t = await getTranslations("school");
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-6 px-6 py-24 text-center">
      <h1 className="text-4xl font-bold tracking-tight">{t("heroTitle", { name: school?.name ?? "" })}</h1>
      <p className="text-lg text-muted-foreground">{t("heroSubtitle")}</p>
      <Link href="/signup" className={buttonVariants({ size: "lg" })}>
        {t("signup")}
      </Link>
    </main>
  );
}
