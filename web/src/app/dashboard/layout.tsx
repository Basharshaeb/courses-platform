import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";

export default async function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  const t = await getTranslations("common");
  return (
    <>
      <header className="flex items-center justify-between border-b px-6 py-4">
        <span className="text-lg font-semibold">{t("brand")}</span>
        <LocaleSwitcher />
      </header>
      {children}
    </>
  );
}
