import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { buttonVariants } from "@/components/ui/button";
import { hosts, originFor } from "@/lib/hosts";
import { requestHost, requestProto } from "@/lib/api/server";

export default async function MarketingPage() {
  const t = await getTranslations();
  const app = originFor(await requestHost(), hosts.appHost, await requestProto());

  return (
    <>
      <header className="flex items-center justify-between border-b px-6 py-4">
        <span className="text-lg font-semibold">{t("common.brand")}</span>
        <div className="flex items-center gap-2">
          <LocaleSwitcher />
          <Link href={`${app}/login`} className={buttonVariants({ variant: "ghost", size: "sm" })}>
            {t("marketing.login")}
          </Link>
        </div>
      </header>
      <main className="flex flex-1 flex-col items-center justify-center gap-6 px-6 py-24 text-center">
        <h1 className="max-w-2xl text-4xl font-bold tracking-tight sm:text-5xl">{t("marketing.title")}</h1>
        <p className="max-w-xl text-lg text-muted-foreground">{t("marketing.subtitle")}</p>
        <Link href={`${app}/signup`} className={buttonVariants({ size: "lg" })}>
          {t("marketing.cta")}
        </Link>
      </main>
    </>
  );
}
