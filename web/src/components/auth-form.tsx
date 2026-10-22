"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useLocale, useTranslations } from "next-intl";
import { api, type Problem } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Mode = "creator-login" | "creator-signup" | "student-login" | "student-signup";

const knownErrors = new Set([
  "email_taken",
  "subdomain_taken",
  "invalid_subdomain",
  "invalid_request",
  "unauthorized",
  "forbidden",
]);

export function AuthForm({
  mode,
  redirectTo,
  subdomainSuffix,
}: {
  mode: Mode;
  redirectTo: string;
  /** Shown after the subdomain input, e.g. ".yourapp.com". */
  subdomainSuffix?: string;
}) {
  const t = useTranslations("auth");
  const te = useTranslations("errors");
  const locale = useLocale() as "ar" | "en";
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const isSignup = mode.endsWith("signup");
  const isCreatorSignup = mode === "creator-signup";

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setPending(true);
    const f = new FormData(e.currentTarget);
    const email = String(f.get("email") ?? "");
    const password = String(f.get("password") ?? "");
    const name = String(f.get("name") ?? "");

    let problem: Problem | undefined;
    switch (mode) {
      case "creator-login":
        ({ error: problem } = await api.POST("/accounts/login", { body: { email, password } }));
        break;
      case "creator-signup":
        ({ error: problem } = await api.POST("/accounts/signup", {
          body: {
            email,
            password,
            name,
            locale,
            schoolName: String(f.get("schoolName") ?? ""),
            subdomain: String(f.get("subdomain") ?? "").toLowerCase(),
          },
        }));
        break;
      case "student-login":
        ({ error: problem } = await api.POST("/auth/login", { body: { email, password } }));
        break;
      case "student-signup":
        ({ error: problem } = await api.POST("/auth/signup", { body: { email, password, name, locale } }));
        break;
    }

    setPending(false);
    if (problem) {
      setError(knownErrors.has(problem.code) ? te(problem.code) : te("generic"));
      return;
    }
    router.push(redirectTo);
    router.refresh();
  }

  return (
    <form onSubmit={submit} className="grid gap-4" noValidate={false}>
      {isSignup && (
        <Field id="name" label={t("name")}>
          <Input id="name" name="name" required maxLength={100} autoComplete="name" />
        </Field>
      )}
      {isCreatorSignup && (
        <>
          <Field id="schoolName" label={t("schoolName")}>
            <Input id="schoolName" name="schoolName" required maxLength={100} />
          </Field>
          <Field id="subdomain" label={t("subdomain")} hint={t("subdomainHint")}>
            <div className="flex items-center gap-2" dir="ltr">
              <Input
                id="subdomain"
                name="subdomain"
                required
                minLength={3}
                maxLength={32}
                pattern="[a-z0-9][a-z0-9-]{1,30}[a-z0-9]"
                autoCapitalize="none"
              />
              {subdomainSuffix && <span className="text-sm text-muted-foreground">{subdomainSuffix}</span>}
            </div>
          </Field>
        </>
      )}
      <Field id="email" label={t("email")}>
        <Input id="email" name="email" type="email" required autoComplete="email" dir="ltr" />
      </Field>
      <Field id="password" label={t("password")}>
        <Input
          id="password"
          name="password"
          type="password"
          required
          minLength={isSignup ? 8 : 1}
          autoComplete={isSignup ? "new-password" : "current-password"}
          dir="ltr"
        />
      </Field>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <Button type="submit" disabled={pending} className="w-full">
        {pending ? t("submitting") : isSignup ? t("signup") : t("login")}
      </Button>
    </form>
  );
}

function Field({ id, label, hint, children }: { id: string; label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
