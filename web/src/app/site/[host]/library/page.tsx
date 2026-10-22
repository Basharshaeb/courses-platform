import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { LogoutButton } from "@/components/logout-button";
import { serverApi } from "@/lib/api/server";

export default async function LibraryPage() {
  const { data: me, response } = await (await serverApi()).GET("/me");
  if (response.status === 401) redirect("/login");
  if (!me) throw new Error(`me failed: ${response.status}`);

  const t = await getTranslations("school");
  return (
    <main className="mx-auto grid w-full max-w-4xl gap-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("greeting", { name: me.name })}</h1>
        <LogoutButton kind="student" redirectTo="/" />
      </div>
      <h2 className="text-lg font-medium">{t("libraryTitle")}</h2>
      <p className="text-muted-foreground">{t("libraryEmpty")}</p>
    </main>
  );
}
