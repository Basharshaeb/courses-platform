import { redirect } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { LogoutButton } from "@/components/logout-button";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { requestHost, requestProto, serverApi } from "@/lib/api/server";
import { hosts, originFor } from "@/lib/hosts";

export default async function DashboardHome() {
  const api = await serverApi();
  const { data, response } = await api.GET("/accounts/me");
  if (response.status === 401) redirect("/login");
  if (!data) throw new Error(`accounts/me failed: ${response.status}`);

  const t = await getTranslations("dashboard");
  const host = await requestHost();
  const proto = await requestProto();

  return (
    <main className="mx-auto grid w-full max-w-4xl gap-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("welcome", { name: data.account.name })}</h1>
        <LogoutButton kind="account" redirectTo="/login" />
      </div>

      <section className="grid gap-3">
        <h2 className="text-lg font-medium">{t("yourSchools")}</h2>
        <div className="grid gap-4 sm:grid-cols-2">
          {data.schools.map((s) => {
            const site = originFor(host, `${s.subdomain}.${hosts.rootDomain}`, proto);
            return (
              <Card key={s.schoolId}>
                <CardHeader>
                  <CardTitle>{s.name}</CardTitle>
                  <CardDescription dir="ltr" className="text-start">
                    {s.subdomain}.{hosts.rootDomain} · {t(`role.${s.role}`)}
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex items-center gap-2">
                  <a href={`/schools/${s.schoolId}/courses`} className={buttonVariants({ size: "sm" })}>
                    إدارة الدورات
                  </a>
                  <a href={site} target="_blank" rel="noreferrer" className={buttonVariants({ variant: "outline", size: "sm" })}>
                    {t("visit")}
                  </a>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>

      <p className="text-sm text-muted-foreground">{t("nextSteps")}</p>
    </main>
  );
}
