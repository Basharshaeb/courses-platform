import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const apiInternalUrl = process.env.API_INTERNAL_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  // Every page here is per-request (tenant from host, locale from cookie), so Cache Components
  // stays off for now. Revisit for cached storefront pages (docs/PLAN.md, phase 2).
  cacheComponents: false,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  // Local stand-in for the edge proxy: /api/* on every host goes to the Go API
  // (docs/adr/0003-same-origin-api.md). In production Caddy does this before Next.js.
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiInternalUrl}/api/:path*` }];
  },
};

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
