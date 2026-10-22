import { getTranslations } from "next-intl/server";
import { AuthCard } from "@/components/auth-card";
import { AuthForm } from "@/components/auth-form";

export default async function StudentSignupPage() {
  const t = await getTranslations("auth");
  return (
    <AuthCard title={t("studentSignupTitle")} footer={{ text: t("haveAccount"), href: "/login", link: t("login") }}>
      <AuthForm mode="student-signup" redirectTo="/library" />
    </AuthCard>
  );
}
