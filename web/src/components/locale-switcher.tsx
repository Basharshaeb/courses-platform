"use client";

import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { localeCookie } from "@/i18n/config";
import { Button } from "@/components/ui/button";

export function LocaleSwitcher() {
  const locale = useLocale();
  const t = useTranslations("common");
  const router = useRouter();

  function toggle() {
    const next = locale === "ar" ? "en" : "ar";
    document.cookie = `${localeCookie}=${next}; path=/; max-age=31536000; samesite=lax`;
    router.refresh();
  }

  return (
    <Button variant="ghost" size="sm" onClick={toggle}>
      {t("language")}
    </Button>
  );
}
