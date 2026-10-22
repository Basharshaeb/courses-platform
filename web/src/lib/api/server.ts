import "server-only";

import { headers } from "next/headers";
import createClient from "openapi-fetch";
import type { paths } from "./schema";

const apiInternalUrl = process.env.API_INTERNAL_URL ?? "http://localhost:8080";

/**
 * Server-side API client for the current request. Calls the Go API on the internal network and
 * forwards the browser's host (tenant) and cookie (session), as the edge proxy would.
 */
export async function serverApi() {
  const h = await headers();
  const forwarded: Record<string, string> = {
    "x-forwarded-host": h.get("x-forwarded-host") ?? h.get("host") ?? "",
  };
  const cookie = h.get("cookie");
  if (cookie) forwarded.cookie = cookie;
  const ua = h.get("user-agent");
  if (ua) forwarded["user-agent"] = ua;

  return createClient<paths>({
    baseUrl: `${apiInternalUrl}/api/v1`,
    headers: forwarded,
    cache: "no-store",
  });
}

/** Current request host including port, e.g. "demo.localhost:3000". */
export async function requestHost(): Promise<string> {
  const h = await headers();
  return (h.get("x-forwarded-host") ?? h.get("host") ?? "").toLowerCase();
}

export async function requestProto(): Promise<string> {
  return (await headers()).get("x-forwarded-proto") ?? "http";
}
