# Teatchable Web Frontend (`web/`)

This directory contains the Next.js 16 (App Router + Turbopack) application. It powers all frontend surfaces across the platform using multi-tenant host routing:
- **`app.yourapp.com`**: Creator Dashboard (course builder, curriculum editor, order management, settings).
- **`{subdomain}.yourapp.com` & Custom Domains**: School Storefront, sales landing pages, checkout, and the student learning player.
- **`admin.yourapp.com`**: Platform Admin portal.
- **`yourapp.com`**: Platform marketing and creator acquisition landing page.

---

## 1. Directory Structure

```
web/
├── public/                 # Static assets, logos, and fonts
├── src/
│   ├── app/
│   │   ├── dashboard/      # App host (app.*) - Creator admin UI
│   │   │   └── schools/[schoolId]/
│   │   │       ├── courses/           # Course listing & curriculum editor
│   │   │       └── settings/          # School settings, domains, payments
│   │   ├── marketing/      # Root host - Public landing page
│   │   ├── platform/       # Admin host (admin.*) - Superadmin console
│   │   ├── site/[host]/    # Tenant host - Student storefront & learning player
│   │   │   ├── courses/[slug]/        # Course sales page
│   │   │   ├── checkout/[planId]/     # Payment checkout page
│   │   │   ├── learn/[courseSlug]/    # Interactive lecture player
│   │   │   └── library/               # Enrolled courses library
│   │   ├── layout.tsx      # Root HTML layout with RTL dir and fonts
│   │   └── globals.css     # Design tokens, CSS variables, Tailwind directives
│   ├── components/         # Reusable UI components
│   │   ├── ui/             # Primitives (button, card, dialog, input, badge, tabs)
│   │   ├── course-player.tsx          # Full-screen course learning interface
│   │   ├── curriculum-editor.tsx      # Section & lecture organizer
│   │   └── lecture-editor.tsx         # Multi-block lesson content editor
│   ├── lib/
│   │   ├── api/            # Type-safe API client (openapi-fetch + schema.d.ts)
│   │   ├── auth/           # Session context and cookie utilities
│   │   └── utils.ts        # cn() class merge helper (clsx + tailwind-merge)
│   ├── proxy.ts            # Host-based URL rewrite engine
│   └── middleware.ts       # Edge middleware routing requests via proxy.ts
└── tailwind.config.ts      # Tailwind design system configuration
```

---

## 2. Mandatory Code Standards & Frontend Best Practices

### A. Server Components vs. Client Components
1. **Server Components by Default**:
   - All page layouts and data-fetching views in `src/app/**/page.tsx` must be async Server Components.
   - Fetch data directly using `serverApi()` which forwards host headers and session cookies to the Go backend.
2. **Client Components (`"use client"`)**:
   - Add `"use client"` only at the leaf component level when:
     - Component requires local state (`useState`, `useReducer`).
     - Component requires event listeners (`onClick`, `onChange`, `onSubmit`).
     - Component utilizes browser APIs (video playback, media recorder, drag-and-drop).
3. **No Waterfalls**:
   - Fetch independent data concurrently using `Promise.all([fetchA(), fetchB()])`.
   - Wrap slow-loading UI sections with React `<Suspense fallback={<Skeleton />}>`.

### B. RTL-First CSS Standards (Strict Arabic & English Support)
The application is designed Arabic-first with seamless English switching:
1. **Never use physical directions in Tailwind classes**:
   - ❌ **Forbidden**: `ml-4`, `mr-2`, `pl-6`, `pr-3`, `left-0`, `right-4`, `text-left`, `text-right`
   - ✅ **Required**: `ms-4` (margin-start), `me-2` (margin-end), `ps-6` (padding-start), `pe-3` (padding-end), `start-0`, `end-4`, `text-start`, `text-end`
2. **Icons with Directionality**:
   - Arrows, chevrons, and navigation icons must use `rtl:rotate-180` so they dynamically flip when switching between LTR and RTL:
   ```tsx
   <ChevronLeft className="h-4 w-4 rtl:rotate-180" />
   ```
3. **Typography**:
   - Arabic text uses high-legibility modern fonts (`Cairo`, `Tajawal`).
   - Line-heights for Arabic must be slightly more generous (`leading-relaxed` or `leading-loose`) to prevent diacritic clipping.

### C. Type-Safe API Consumption (`openapi-fetch`)
1. **Zero Manual Types & Zero `any`**:
   - All API models are generated into `src/lib/api/schema.d.ts` directly from `api/openapi/openapi.yaml`.
   - Use the typed client:
   ```ts
   import { api } from "@/lib/api/client";

   const { data, error } = await api.GET("/api/v1/courses", {
     params: { query: { status: "published" } },
   });
   ```
2. **Error Handling**:
   - All backend errors follow RFC 9457 Problem Details (`{ type, title, status, code, detail }`).
   - Handle errors gracefully by displaying localized error messages mapped from `error.code`.

### D. Design System & Aesthetics
1. **Glassmorphism & Depth**:
   - Use subtle translucent backgrounds with blur for sticky headers and floating sidebars:
     `bg-white/80 dark:bg-slate-900/80 backdrop-blur-md border-b border-border/60`
2. **Curated Color Tokens**:
   - Never use raw Tailwind colors (`bg-blue-500`, `text-red-600`) directly in core components.
   - Use CSS semantic tokens: `bg-primary`, `text-primary-foreground`, `bg-muted`, `border-border`, `bg-card`.
3. **Micro-Animations & Feedback**:
   - Buttons, cards, and interactive elements must feature subtle transition states (`transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md active:scale-95`).
   - Form buttons must render a spinning `<Loader2 className="animate-spin" />` during network operations and disable repeated clicks.

### E. Forms & Validation
- Standardize all user input forms on `react-hook-form` with `@hookform/resolvers/zod`.
- Validate client inputs with Zod schemas matching backend constraints (e.g., minimum password lengths, valid slugs, positive price numbers).

---

## 3. Host-Based Routing Workflow

Next.js detects the request host (`Host` header or `X-Forwarded-Host`) in `src/proxy.ts`:
- If host starts with `app.` → rewrites to `/dashboard`
- If host starts with `admin.` → rewrites to `/platform`
- If host is root marketing domain → rewrites to `/marketing`
- All other hosts (subdomains like `demo.localhost` or custom domains like `academy.org`) → rewrites to `/site/[host]`

This allows full previewing locally:
- `http://localhost:3000` → Marketing
- `http://app.localhost:3000` → Creator Dashboard
- `http://demo.localhost:3000` → Demo School Storefront

---

## 4. Linting, Quality Checks & Testing

Before committing code, verify type safety and ESLint conformance:

```bash
# Run TypeScript compilation check
pnpm typecheck

# Run ESLint check
pnpm lint

# Build production bundle
pnpm build
```

The CI pipeline runs all three checks and will reject any commit containing ESLint warnings or TypeScript type errors.
