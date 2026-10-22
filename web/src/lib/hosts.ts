// Host → surface mapping. Mirrors backend/internal/platform/tenant; keep the two in sync.

export type Surface = "marketing" | "dashboard" | "platform" | "school";

const rootDomain = (process.env.ROOT_DOMAIN ?? "localhost").toLowerCase();
const appHost = (process.env.APP_HOST ?? `app.${rootDomain}`).toLowerCase();
const adminHost = (process.env.ADMIN_HOST ?? `admin.${rootDomain}`).toLowerCase();

export function stripPort(host: string): string {
  return host.replace(/:\d+$/, "");
}

export function surfaceFor(hostWithPort: string): Surface {
  const host = stripPort(hostWithPort.toLowerCase());
  if (host === appHost) return "dashboard";
  if (host === adminHost) return "platform";
  if (host === rootDomain || host === `www.${rootDomain}`) return "marketing";
  return "school";
}

/** Origin of another surface, keeping the protocol and port of the current request. */
export function originFor(currentHost: string, targetHost: string, proto = "http"): string {
  const port = currentHost.match(/:(\d+)$/)?.[1];
  return `${proto}://${targetHost}${port ? `:${port}` : ""}`;
}

export const hosts = { rootDomain, appHost, adminHost };
