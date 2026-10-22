import { getTranslations } from "next-intl/server";
import { AuthCard } from "@/components/auth-card";
import { AuthForm } from "@/components/auth-form";

export default async function StudentLoginPage() {
  const t = await getTranslations("auth");
  return (
    <AuthCard title={t("studentLoginTitle")} footer={{ text: t("noAccount"), href: "/signup", link: t("signup") }}>
      <AuthForm mode="student-login" redirectTo="/library" />
    </AuthCard>
  );
}
