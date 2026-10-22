import { cookies, headers } from "next/headers";
import { getRequestConfig } from "next-intl/server";
import { defaultLocale, isLocale, localeCookie, type Locale } from "./config";

// Locale comes from the NEXT_LOCALE cookie, then Accept-Language, then Arabic.
// URL-prefixed locales (/en/...) can come later for SEO on storefront pages.
async function resolveLocale(): Promise<Locale> {
  const fromCookie = (await cookies()).get(localeCookie)?.value;
  if (isLocale(fromCookie)) return fromCookie;

  const accept = (await headers()).get("accept-language") ?? "";
  for (const part of accept.split(",")) {
    const tag = part.split(";")[0].trim().slice(0, 2).toLowerCase();
    if (isLocale(tag)) return tag;
  }
  return defaultLocale;
}

export default getRequestConfig(async () => {
  const locale = await resolveLocale();
  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default,
  };
});

