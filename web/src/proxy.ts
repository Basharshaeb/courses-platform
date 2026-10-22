import { NextResponse, type NextRequest } from "next/server";
import { surfaceFor } from "@/lib/hosts";

// Routes each host to its surface: app.* → /dashboard, admin.* → /platform, root → /marketing,
// any other host (school subdomain or custom domain) → /site/{host}. The browser URL is unchanged.
export function proxy(request: NextRequest) {
  const host = (request.headers.get("x-forwarded-host") ?? request.headers.get("host") ?? "").toLowerCase();
  const surface = surfaceFor(host);
  const prefix = surface === "school" ? `/site/${encodeURIComponent(host)}` : `/${surface}`;

  const url = request.nextUrl.clone();
  url.pathname = prefix + (url.pathname === "/" ? "" : url.pathname);
  return NextResponse.rewrite(url);
}

export const config = {
  // Skip the API (proxied to Go), Next internals and files with an extension.
  matcher: ["/((?!api/|_next/|.*\\..*).*)"],
};
