import { getTranslations } from "next-intl/server";
import { AuthCard } from "@/components/auth-card";
import { AuthForm } from "@/components/auth-form";

export default async function CreatorLoginPage() {
  const t = await getTranslations("auth");
  return (
    <AuthCard title={t("creatorLoginTitle")} footer={{ text: t("noAccount"), href: "/signup", link: t("signup") }}>
      <AuthForm mode="creator-login" redirectTo="/" />
    </AuthCard>
  );
}
