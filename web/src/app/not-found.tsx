import Link from "next/link";
import { getTranslations } from "next-intl/server";

export default async function NotFound() {
  const t = await getTranslations("common");
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-4 p-6 text-center">
      <h1 className="text-2xl font-semibold">{t("notFound")}</h1>
      <Link href="/" className="underline underline-offset-4">
        {t("backHome")}
      </Link>
    </main>
  );
}
