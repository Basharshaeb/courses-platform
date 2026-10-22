import { getTranslations } from "next-intl/server";

export default async function PlatformHome() {
  const t = await getTranslations("platform");
  return (
    <main className="mx-auto grid w-full max-w-4xl gap-2 p-6">
      <h1 className="text-2xl font-semibold">{t("title")}</h1>
      <p className="text-muted-foreground">{t("comingSoon")}</p>
    </main>
  );
}
