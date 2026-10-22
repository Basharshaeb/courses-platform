"use client";

import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

export type Problem = components["schemas"]["Problem"];

/** Browser client. Same-origin: the edge (or next.config rewrites in dev) sends /api to Go. */
export const api = createClient<paths>({ baseUrl: "/api/v1", credentials: "same-origin" });
