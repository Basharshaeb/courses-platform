"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";

export function LogoutButton({ kind, redirectTo }: { kind: "account" | "student"; redirectTo: string }) {
  const t = useTranslations("common");
  const router = useRouter();

  async function logout() {
    await (kind === "account" ? api.POST("/accounts/logout") : api.POST("/auth/logout"));
    router.push(redirectTo);
    router.refresh();
  }

  return (
    <Button variant="outline" size="sm" onClick={logout}>
      {t("logout")}
    </Button>
  );
}
