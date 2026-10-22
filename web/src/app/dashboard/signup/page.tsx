import { getTranslations } from "next-intl/server";
import { AuthCard } from "@/components/auth-card";
import { AuthForm } from "@/components/auth-form";
import { hosts } from "@/lib/hosts";

export default async function CreatorSignupPage() {
  const t = await getTranslations("auth");
  return (
    <AuthCard title={t("creatorSignupTitle")} footer={{ text: t("haveAccount"), href: "/login", link: t("login") }}>
      <AuthForm mode="creator-signup" redirectTo="/" subdomainSuffix={`.${hosts.rootDomain}`} />
    </AuthCard>
  );
}
