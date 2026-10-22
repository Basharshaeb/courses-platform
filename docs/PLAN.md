# Plan — Multi-tenant Course Platform (Teachable-style), Go APIs + Next.js

## Context
`D:\teatchable` is empty (greenfield). Goal: SaaS where creators sign up, get their own school
(`school.yourapp.com` or custom domain), build courses (video, text, quizzes), sell them
(one-time, subscriptions, payment plans), and students learn on a branded site.
Target market: Saudi/GCC first, global later → Arabic + English (RTL), mada/Apple Pay/STC Pay, 15% VAT, ZATCA invoices.

### Locked decisions
| Area | Choice |
|---|---|
| Tenancy | Multi-tenant SaaS, shared Postgres DB, `school_id` on every tenant row + Postgres RLS |
| Backend | Go modular monolith: `api` binary + `worker` binary, same codebase |
| Frontend | Next.js (App Router) + TypeScript, Tailwind + shadcn/ui, next-intl (ar/en, RTL) |
| Payments | Stripe (Connect), Moyasar, Tap, PayPal — behind one `Gateway` interface |
| Merchant model | Creator is merchant of record (connects own gateway / Stripe Connect). Platform takes fee via Stripe `application_fee` or monthly fee invoice. Avoids SAMA payment-facilitator licensing. |
| Video | Bunny Stream (direct TUS upload, HLS, token-signed playback) |
| Files | S3-compatible object storage (Cloudflare R2 or S3), presigned URLs |
| Infra | Docker + Kubernetes, Helm, Terraform, GitHub Actions; KSA region for PDPL (e.g. GCP Dammam) |
| Mobile | Later phase (Flutter/React Native) on the same API |

---

## 1. Roles & Feature Scope

**Roles**
- Platform admin (you): manage schools, SaaS plans, fees, moderation, impersonation, feature flags.
- School staff (`accounts` + `school_members`): owner, admin, author, support, finance.
- Student (`students`, scoped per school — same email can exist in many schools, like Teachable).
- Affiliate (a student or external person with referral code).

**Creator features**
- School setup: name, subdomain, logo, colors, fonts, default locale, currency, timezone, custom domain.
- Course builder: sections → lectures → content blocks (video, rich text, PDF, downloadable file, quiz, assignment, embed, code). Drag & drop reorder, free preview lectures, draft/publish, drip (days after enrollment / fixed date), completion rules, comments toggle.
- Products & pricing: free, one-time, subscription (monthly/yearly), payment plan (N installments), bundles, coaching, digital downloads.
- Checkout: coupons (percent/fixed, limits, expiry), order bumps, upsells, VAT handling, thank-you page.
- Site/page builder: home, sales page per course, custom pages, nav menu, block-based editor.
- Students: list, filter, import CSV, manual enroll/unenroll, progress view, export.
- Quizzes & certificates (templated PDF, public verify URL).
- Community: lecture comments & discussions with moderation.
- Email marketing: transactional + broadcasts + automations (trigger → delay → email).
- Affiliates: codes, commission %, cookie window, payouts report.
- Analytics: revenue, MRR, churn, conversion, enrollments, completion, video engagement.
- Integrations: outgoing webhooks, public API keys, Zapier (later).
- SaaS billing: creator picks plan (Free/Basic/Pro) with limits + transaction fee %.

**Student features**
- Catalog, sales pages, checkout (mada, Apple Pay, STC Pay, cards, PayPal), receipts/VAT invoices.
- Library ("My courses"), lecture player with resume, progress tracking, quizzes, certificates, comments.
- Account: profile, password/magic link/Google login, payment methods, subscriptions (cancel), data export/delete.

---

## 2. Architecture

```
                  ┌──────────────── Caddy edge (K8s, on-demand TLS) ───────────────┐
 browser ───────▶ │ *.yourapp.com, custom domains, app., admin.                     │
                  │   /api/*  ───────────────▶  Go API (cmd/api)                     │
                  │   /*      ───────────────▶  Next.js (web)  ──server fetch──▶ API │
                  └─────────────────────────────────────────────────────────────────┘
 Go API ──▶ Postgres (RLS, River jobs)   Go API ──▶ Redis (cache, sessions, rate limit)
 Go Worker (River) ──▶ Bunny Stream / S3 / SES / Gotenberg (PDF) / gateways / webhooks
 Gateways & Bunny ──webhooks──▶ /api/webhooks/{provider}
```

Key points:
- **Same-origin API**: Caddy routes `{any school host}/api/*` → Go API. Cookies stay first-party on every school/custom domain (no 3rd-party cookie problem). Mobile/public API uses `api.yourapp.com` with Bearer tokens.
- **Host → school resolution**: Go middleware reads `X-Forwarded-Host`, looks up `school_domains` (Redis cache, 5 min TTL, busted on change), puts `school_id` in context, and runs `SET LOCAL app.school_id = …` in each tx for RLS.
- **Custom domains**: creator adds CNAME → `edge.yourapp.com` + TXT verify; Caddy `on_demand_tls` calls `GET /internal/tls-allowed?domain=` before issuing certs; cert storage shared in Redis. Wildcard `*.yourapp.com` via DNS-01 challenge.
- **Hosts**: `yourapp.com` marketing, `app.yourapp.com` creator dashboard, `admin.yourapp.com` platform admin (IP allowlist + MFA), `{sub}.yourapp.com` / custom = school.
- **Async**: River (Postgres-native job queue) with transactional enqueue — job inserted in same tx as the business change (no separate outbox needed).
- **PDFs** (invoices, certificates): Gotenberg (headless Chromium) renders HTML→PDF — correct Arabic shaping/RTL (Go PDF libs handle Arabic poorly).
- **Caching**: Redis for host map, sessions, rate limits, course outlines. Next.js ISR for public pages; API calls Next `revalidateTag` endpoint on publish.

---

## 3. Go Backend

### Stack
| Concern | Library |
|---|---|
| Go | latest stable (1.26+) |
| Router | `go-chi/chi/v5` |
| API contract | OpenAPI 3.1 spec-first, `oapi-codegen` (strict server, chi) + `kin-openapi` request validation |
| DB | Postgres 17/18, `jackc/pgx/v5`, `sqlc` (type-safe queries), `pressly/goose` migrations |
| Jobs | `riverqueue/river` (+ periodic jobs) |
| Cache | `redis/go-redis/v9` |
| Logging/obs | `log/slog` JSON, OpenTelemetry traces/metrics, Sentry |
| Auth crypto | `golang.org/x/crypto/argon2` (argon2id), crypto/rand opaque tokens |
| Config | `caarlos0/env` |
| Payments | `stripe/stripe-go`; custom REST clients for Moyasar, Tap, PayPal |
| AWS/S3/SES | `aws-sdk-go-v2` (works with R2) |
| Tests | stdlib `testing`, `testcontainers-go`, `golangci-lint` |
| Dev | `air` hot reload, Taskfile |

### Repo layout (monorepo at `D:\teatchable`)
```
backend/
  cmd/api/main.go            # HTTP server
  cmd/worker/main.go         # River workers + periodic jobs
  cmd/migrate/main.go        # goose wrapper + seed
  internal/
    platform/                # shared infra, no business logic
      config/ db/ (pool, WithTx, tenant SET LOCAL) cache/ httpx/ (problem+json, pagination, idempotency)
      auth/ (session middleware, RBAC) tenant/ (host resolver) crypto/ (AES-GCM envelope, KMS)
      jobs/ (River client) storage/ (S3) mailer/ pdf/ (Gotenberg) ratelimit/ observability/
    modules/
      identity/      accounts, students, sessions, OAuth (Google/Apple), magic links, MFA
      school/        schools, domains, branding, settings, staff & roles
      catalog/       courses, sections, lectures, blocks, bundles
      media/         Bunny Stream + S3 uploads, webhooks
      enrollment/    enrollments, progress, drip unlocks
      assessment/    quizzes, questions, attempts, assignments
      commerce/      products, pricing plans, coupons, checkout, orders
      payments/      gateway interface + stripe/ moyasar/ tap/ paypal/ adapters, inbound webhooks
      billing/       subscriptions, installments, dunning, invoices, VAT/ZATCA
      saas/          creator plans, platform fee ledger, creator billing
      certificates/
      community/     comments, discussions, moderation
      marketing/     email lists, broadcasts, automations, affiliates
      pages/         site builder pages, navigation, themes
      analytics/     events ingest, rollups, reports
      integrations/  outgoing webhooks, API keys
      admin/         platform admin ops, impersonation, audit
  db/migrations/*.sql
  db/queries/<module>/*.sql
  sqlc.yaml
api/openapi/                 # spec split per module, bundled by redocly → openapi.yaml
web/                         # Next.js
deploy/ (helm/, caddy/Caddyfile, terraform/)
docker-compose.yml           # postgres, redis, minio, mailpit, gotenberg, stripe-cli
Taskfile.yml
docs/ (PLAN.md, adr/)
.github/workflows/
```

### Module pattern (repeat for every module)
- `handler.go` — implements generated strict-server interface; maps DTO ↔ service; no SQL.
- `service.go` — business rules, authorization checks, opens tx via `db.WithTx(ctx, schoolID, fn)`, enqueues River jobs in same tx.
- `repo` — sqlc-generated `Queries`; every query also filters by `school_id` (defense in depth on top of RLS).
- `jobs.go` — River job args + workers for the module.
- Modules talk via exported service interfaces only (no cross-module table access); cross-module side effects via jobs (e.g. `OrderPaid` → `enrollment.Grant`, `billing.CreateInvoice`, `marketing.TrackAffiliate`).

### Conventions
- IDs: UUIDv7. Money: `bigint` minor units + `currency char(3)` (SAR halalas). Times: `timestamptz` UTC.
- Errors: RFC 9457 `application/problem+json` with stable `code` (e.g. `course_not_found`).
- Pagination: cursor-based (`?cursor=&limit=`).
- `Idempotency-Key` header required on checkout/refund/charge endpoints (stored 24h in Redis + DB).
- Rate limits: Redis token bucket per IP, per student, per school (login, OTP, checkout stricter).
- Versioning: `/api/v1`.

---

## 4. Data Model (Postgres)
All tenant tables have `school_id uuid not null` + RLS policy `school_id = current_setting('app.school_id')::uuid`. API DB role has no `BYPASSRLS`; platform-admin and some worker jobs use a separate role.

**Platform / identity**
- `platform_admins` (id, email, password_hash, mfa_secret)
- `accounts` (id, email unique, name, password_hash, locale, email_verified_at, mfa_enabled) — creators/staff
- `students` (id, school_id, email, name, password_hash null, locale, status, metadata jsonb, unique(school_id,email))
- `oauth_identities` (id, subject_type, subject_id, provider, provider_uid)
- `sessions` (id, subject_type account|student|admin, subject_id, school_id null, token_hash, ip, user_agent, expires_at, revoked_at)
- `magic_links` / `email_verifications` / `password_resets` (token_hash, expires_at, used_at)

**School**
- `schools` (id, owner_account_id, name, subdomain unique, default_locale, currency, timezone, branding jsonb, settings jsonb, saas_plan_id, status)
- `school_domains` (id, school_id, hostname unique, verification_token, verified_at, tls_status, is_primary)
- `school_members` (school_id, account_id, role owner|admin|author|support|finance, permissions jsonb)

**Catalog & content**
- `courses` (id, school_id, slug, title, subtitle, description, cover_key, locale, status draft|published|archived, author_account_id, settings jsonb {drip, comments, completion_rule}, published_at)
- `course_sections` (id, course_id, position, title, drip_type none|days|date, drip_days, drip_date)
- `lectures` (id, course_id, section_id, position, title, is_free_preview, status, duration_sec)
- `lecture_blocks` (id, lecture_id, position, kind video|text|pdf|file|quiz|assignment|embed|code, data jsonb, media_asset_id null)
- `media_assets` (id, school_id, provider bunny|s3, provider_ref, kind, status uploading|processing|ready|failed, filename, size_bytes, duration_sec, thumbnail_key)
- `bundles`, `bundle_items` (bundle_id, course_id)

**Commerce**
- `products` (id, school_id, kind course|bundle|coaching|download|membership, ref_id, name, status)
- `pricing_plans` (id, school_id, product_id, type free|one_time|subscription|payment_plan, amount, currency, interval month|year, interval_count, installments, trial_days, tax_inclusive, active)
- `coupons` (id, school_id, code, type percent|fixed, value, applies_to jsonb, max_redemptions, per_student_limit, starts_at, expires_at)
- `coupon_redemptions` (coupon_id, order_id, student_id)
- `orders` (id, school_id, student_id, status pending|paid|failed|refunded|partially_refunded|canceled, subtotal, discount, tax, total, currency, gateway, coupon_id, affiliate_id, buyer_vat_number, billing_address jsonb, idempotency_key)
- `order_items` (order_id, product_id, pricing_plan_id, amount, tax)
- `payments` (id, school_id, order_id, subscription_id null, gateway, gateway_payment_id, amount, status, failure_code, raw jsonb)
- `refunds` (id, payment_id, amount, reason, gateway_refund_id, status)
- `payment_methods` (id, school_id, student_id, gateway, token_encrypted, brand, last4, exp, is_default)
- `gateway_connections` (id, school_id, gateway, mode test|live, credentials_encrypted bytea, external_account_id, status, webhook_secret_encrypted)
- `inbound_webhook_events` (id, gateway, event_id, unique(gateway,event_id), payload jsonb, processed_at, error)

**Billing**
- `subscriptions` (id, school_id, student_id, pricing_plan_id, gateway, gateway_subscription_id null, payment_method_id, status trialing|active|past_due|canceled|completed, current_period_start/end, next_billing_at, installments_paid, cancel_at_period_end, failed_attempts)
- `invoices` (id, school_id, order_id/payment_id, number (gapless per school), seller_vat_number, vat_rate, vat_amount, total, zatca_uuid, zatca_qr, pdf_key, issued_at)
- `tax_settings` (school_id, vat_registered, vat_number, vat_rate, prices_include_tax, country)

**Learning**
- `enrollments` (id, school_id, student_id, course_id, source order|subscription|bundle|manual|free, source_id, status active|expired|revoked, enrolled_at, expires_at, completed_at, progress_pct)
- `lecture_progress` (enrollment_id, lecture_id, completed_at, position_sec, watched_sec, updated_at; pk(enrollment_id, lecture_id))
- `quizzes` (id, lecture_block_id, pass_score, max_attempts, shuffle) / `quiz_questions` (type single|multi|true_false|short_text, body, options jsonb, answer jsonb, points) / `quiz_attempts` / `quiz_answers`
- `assignments` / `assignment_submissions` (file_key, text, grade, feedback)
- `certificate_templates` (school_id, html, background_key) / `certificates` (id, enrollment_id, serial, verify_code unique, pdf_key, issued_at)
- `comments` (id, school_id, lecture_id, author_type, author_id, parent_id, body, status visible|hidden|pending)

**Site & marketing**
- `pages` (id, school_id, slug, kind home|sales|checkout|thank_you|custom, draft_blocks jsonb, published_blocks jsonb, seo jsonb, published_at) / `page_versions`
- `nav_menus` (school_id, location header|footer, items jsonb)
- `email_templates`, `broadcasts`, `broadcast_recipients`, `automations` (trigger, steps jsonb), `automation_runs`, `email_events` (sent|delivered|opened|clicked|bounced|complained), `email_suppressions`
- `affiliates` (id, school_id, student_id null, name, email, code, commission_pct, cookie_days, status) / `affiliate_referrals` / `affiliate_commissions` (order_id, amount, status pending|approved|paid)

**Platform / ops**
- `saas_plans` (id, name, price, interval, transaction_fee_bps, fixed_fee, limits jsonb {courses, students, staff, custom_domain, ...})
- `school_saas_subscriptions` (school_id, saas_plan_id, gateway, status, current_period_end)
- `platform_fee_ledger` (school_id, order_id, amount, currency, status accrued|invoiced|collected)
- `webhook_endpoints` (school_id, url, secret_encrypted, events[]) / `webhook_deliveries` (attempts, status, response_code)
- `api_keys` (school_id, prefix, key_hash, scopes[], last_used_at, revoked_at)
- `analytics_events` (partitioned monthly: school_id, kind, subject, props jsonb, occurred_at) + `daily_school_stats` rollups
- `audit_logs` (actor_type, actor_id, school_id, action, target, diff jsonb, ip, at)
- `feature_flags` (key, school_id null, enabled)

Indexes: `(school_id, created_at)` on big tables, `(school_id, slug)` unique on courses/pages, `(student_id, course_id)` unique on enrollments, `next_billing_at` partial index where status in (active, past_due).

---

## 5. API Surface (REST, `/api/v1`)
Auth: opaque session token (32 random bytes, SHA-256 stored, Redis-cached), httpOnly Secure SameSite=Lax cookie, host-only per school domain; `Authorization: Bearer` for mobile/API keys. CSRF: SameSite + `Origin` check on mutations.

| Group | Representative endpoints |
|---|---|
| Student auth (school host) | `POST /auth/signup`, `/auth/login`, `/auth/magic-link`, `/auth/oauth/{provider}`, `/auth/logout`, `/auth/password/reset` |
| Storefront (public) | `GET /storefront/school`, `/storefront/pages/{slug}`, `/storefront/courses`, `/storefront/courses/{slug}` (outline + free previews), `/storefront/pricing-plans/{id}` |
| Checkout | `POST /checkout/quote` (price+coupon+VAT), `POST /checkout/sessions` → `{order_id, gateway, client_config | redirect_url}`, `POST /checkout/sessions/{id}/confirm`, `GET /orders/{id}` |
| Student area | `GET /me`, `/me/enrollments`, `/me/courses/{id}/outline`, `/me/lectures/{id}` (content + signed video URL, drip-checked), `POST /me/lectures/{id}/progress`, `POST /me/quizzes/{id}/attempts`, `/me/certificates`, `/me/subscriptions/{id}/cancel`, `/me/payment-methods`, `/me/orders`, `/me/invoices/{id}/pdf`, `POST /me/export`, `DELETE /me` |
| Community | `GET/POST /me/lectures/{id}/comments`, `PATCH/DELETE /comments/{id}` |
| Creator auth (app host) | `POST /accounts/signup` (creates account + school), `/accounts/login`, `/accounts/mfa/*`, `GET /accounts/me/schools` |
| Creator admin | `/admin/schools/{schoolId}/…` : `courses` CRUD, `courses/{id}/sections`, `lectures`, `blocks`, `reorder`, `publish`; `media/uploads` (returns Bunny TUS signature or S3 presigned URL); `products`, `pricing-plans`, `bundles`, `coupons`; `students` (list/import/enroll/export); `orders`, `refunds`; `subscriptions`; `pages`, `nav`; `emails/broadcasts`, `automations`; `affiliates`; `quizzes`, `certificate-templates`; `comments/moderation`; `analytics/{revenue,enrollments,engagement}`; `settings/{branding,domains,tax,gateways,team}`; `webhooks`, `api-keys`; `billing/saas-plan` |
| Platform admin | `/platform/schools`, `/platform/schools/{id}/impersonate`, `/platform/saas-plans`, `/platform/fees`, `/platform/flags`, `/platform/audit` |
| Webhooks in | `POST /webhooks/stripe`, `/webhooks/moyasar`, `/webhooks/tap`, `/webhooks/paypal`, `/webhooks/bunny`, `/webhooks/ses` |
| Internal | `GET /internal/resolve-host`, `/internal/tls-allowed`, health `/healthz`, `/readyz`, `/metrics` |

Authorization: middleware checks school membership + role permission matrix (e.g. `author` can edit own courses only; `finance` sees orders/refunds; `support` sees students/comments).

---

## 6. Payments Design

```go
type Gateway interface {
    Name() GatewayName
    CreatePayment(ctx context.Context, in CreatePaymentInput) (*PaymentInit, error) // client config or redirect URL
    VerifyPayment(ctx context.Context, gatewayPaymentID string) (*PaymentResult, error)
    ChargeSaved(ctx context.Context, in ChargeSavedInput) (*PaymentResult, error)    // recurring / installments
    Refund(ctx context.Context, in RefundInput) (*RefundResult, error)
    ParseWebhook(r *http.Request, secret []byte) (*WebhookEvent, error)               // signature verified
}
```
Built per school from `gateway_connections` (credentials decrypted with envelope encryption, KMS master key).

| Gateway | Onboarding | Checkout | Recurring | Platform fee |
|---|---|---|---|---|
| Stripe | Connect (Express/Standard) onboarding link | PaymentIntent / Checkout on connected account | Saved PaymentMethod, `off_session` charges by our engine; handle `requires_action` → email student | `application_fee_amount` |
| Moyasar | Creator pastes publishable/secret keys | Moyasar payment form (mada, Apple Pay, STC Pay, cards), callback → server `GET /payments/{id}` verify amount+status+metadata.order_id | Card token (`save_card`) charged by our engine | Accrue in ledger, bill creator monthly |
| Tap | Creator keys (or Tap marketplace/destinations later) | `POST /v2/charges` redirect → verify via `GET /v2/charges/{id}`, webhook hashstring check | Saved card / customer token, merchant-initiated | Ledger (or Tap destinations split) |
| PayPal | Creator REST app credentials (or Partner/Multiparty later) | Orders v2 create → approve → capture | PayPal Subscriptions API synced via webhooks | Ledger |

Rules:
- Never trust redirects: order marked paid only after server-side verify or signed webhook. Check amount + currency + order_id metadata.
- Webhooks idempotent via `inbound_webhook_events` unique (gateway, event_id); processing as River job.
- One internal **billing engine** (River periodic job every 5 min): picks subscriptions/installments with `next_billing_at <= now`, charges saved token, extends period or starts dunning (retry +1d, +3d, +7d → `past_due` → `canceled` → enrollment access revoked). Emails at each step.
- Order flow: `pending` → verify → `paid` → jobs in same tx: grant enrollment(s), create invoice + PDF, record platform fee, affiliate commission, receipt email, outgoing webhook `order.paid`, analytics event.
- VAT: 15% default (configurable per school), tax-inclusive or exclusive prices; invoices in creator's name with their VAT number; ZATCA Phase 1 QR (TLV base64: seller, VAT no, timestamp, total, VAT) on simplified invoices; ZATCA Phase 2 (UBL XML + crypto stamp + Fatoora API reporting) in Phase 3 roadmap.
- Note: Stripe needs creator entity in a Stripe-supported country — Moyasar/Tap cover KSA-based creators.

---

## 7. Video & Files
- Upload: dashboard calls `POST /admin/schools/{id}/media/uploads {filename, kind}` → API creates Bunny video (collection per school) → returns TUS endpoint + signature (`sha256(libraryId + apiKey + expires + videoId)`) → browser uploads directly (tus-js-client, resumable) → Bunny webhook → `media_assets.status = ready`, duration/thumbnail stored.
- Playback: `GET /me/lectures/{id}` checks enrollment + drip + subscription status → returns Bunny embed/HLS URL with token auth (expires ~2h). Free-preview lectures signed for anyone.
- Captions: Bunny auto-captions (ar/en) later.
- Other files (PDF, downloads, images, certificates): S3/R2 private bucket, presigned PUT for upload, presigned GET (short TTL) for download; images served via CDN with resizing.
- Limits per SaaS plan (storage GB, video hours) enforced in media service.

---

## 8. Frontend (Next.js)
Stack: Next.js App Router + TS, Tailwind + shadcn/ui (RTL via `dir` + logical CSS props `ms-/me-/ps-/pe-`), next-intl (ar/en, `/[locale]` segment), TanStack Query, `openapi-typescript` + `openapi-fetch` (typed client from same OpenAPI spec), react-hook-form + zod, Tiptap (rich text, RTL), dnd-kit (curriculum drag & drop), Puck (block page builder), tus-js-client, hls.js / Bunny player, Recharts, Playwright. Fonts: IBM Plex Sans Arabic + Inter.

Host-based routing in `middleware.ts`:
- `app.yourapp.com` → `app/(dashboard)/…`
- `admin.yourapp.com` → `app/(platform)/…`
- `yourapp.com` → `app/(marketing)/…`
- anything else → resolve school → rewrite to `app/(school)/[host]/…`

Pages:
- **School (student)**: home, `/courses`, `/courses/[slug]` sales page, `/p/[slug]`, `/checkout/[planId]`, `/thank-you/[orderId]`, `/login`, `/signup`, `/library`, `/learn/[courseSlug]/[lectureId]` (player + sidebar outline + comments + mark complete), `/certificates/verify/[code]`, `/account/*`.
- **Creator dashboard**: onboarding wizard, overview stats, courses (list, curriculum builder, lecture editor, pricing, drip, settings), students, sales (orders, subscriptions, refunds), coupons, pages/site builder, theme, emails, affiliates, analytics, settings (branding, domain, gateways, tax, team, webhooks, API keys), SaaS billing.
- **Platform admin**: schools, plans, fees, flags, audit, impersonate.

Server components fetch API with forwarded cookie + `X-Forwarded-Host`; client components use same-origin `/api/v1`. Public pages ISR with tag revalidation.

---

## 9. Worker Jobs (River)
| Job | Trigger |
|---|---|
| `order.fulfill` (enroll, fee ledger, affiliate, receipt) | order paid |
| `invoice.generate` (HTML → Gotenberg PDF → S3, ZATCA QR) | order paid / renewal |
| `billing.run_due` / `billing.dunning` | periodic 5 min |
| `webhook.process_inbound` | gateway/Bunny webhook received |
| `webhook.deliver_outbound` (HMAC signed, exp backoff, 8 tries) | domain events |
| `email.send`, `email.broadcast_batch` (chunks of 500, SES) | various |
| `automation.step` | automation triggers + delays |
| `drip.notify` (lecture unlocked email) | periodic daily |
| `certificate.issue` | course completed |
| `domain.verify` (TXT/CNAME check) | periodic until verified |
| `analytics.rollup` | periodic hourly/daily |
| `students.import` / `students.export` CSV | admin action |
| `cleanup` (expired sessions, abandoned orders, orphan uploads) | periodic daily |

---

## 10. Security & Compliance
- Passwords argon2id; MFA (TOTP) for creators/admins; login + OTP rate limits; breached-password check optional.
- Opaque sessions, rotation on privilege change, revoke-all on password change.
- Tenant isolation: RLS + explicit `school_id` filters + integration tests that try cross-tenant access.
- Gateway keys, webhook secrets, card tokens: AES-256-GCM envelope encryption, master key in cloud KMS. Never store raw PAN (gateways tokenize → PCI SAQ-A).
- Webhook signature verification per provider; replay window check.
- Signed short-lived media URLs; hotlink protection on Bunny.
- Input validation from OpenAPI schema; HTML sanitization (bluemonday) for rich text/comments; CSP headers.
- Audit log for admin/staff actions and impersonation.
- Saudi PDPL: data residency in KSA region, privacy policy, consent, data export/delete endpoints; GDPR-compatible for later.
- Backups: Postgres PITR, daily snapshots, restore drill monthly.

---

## 11. Infrastructure & DevOps
- **Local**: `docker-compose.yml` (Postgres, Redis, MinIO, Mailpit, Gotenberg, stripe-cli), `task dev` runs api (air) + worker + web; seed script creates demo school, creator, student, course.
- **K8s** (Helm charts): `web`, `api`, `worker` Deployments (HPA on CPU/RPS), Caddy edge (on-demand TLS, Redis cert storage), Gotenberg. Managed Postgres (HA + PITR), managed Redis, R2/S3, Bunny, SES.
- **Terraform**: cluster, DB, Redis, buckets, DNS (Cloudflare), KMS, secrets.
- **CI (GitHub Actions)**: Go — `golangci-lint`, `go test -race`, sqlc diff check, OpenAPI lint/codegen diff; Web — eslint, tsc, unit tests, Playwright; build & push images; deploy staging on `main`, prod on tag (Argo CD or `helm upgrade`); migrations as pre-deploy Job.
- **Environments**: local, staging (gateway test modes), production.
- **Observability**: OTel → Grafana (Tempo/Prometheus/Loki) or Datadog; Sentry for Go + Next.js; alerts on 5xx rate, p95 latency, job failures, webhook backlog, billing-run failures.
- **NFR targets**: API p95 < 200 ms, 99.9% uptime, video fully offloaded to Bunny CDN, zero cross-tenant leaks.

---

## 12. Testing Strategy
- Unit: services with fakes for gateways/mailer/storage.
- Integration: `testcontainers-go` Postgres per package; RLS cross-tenant tests; sqlc queries.
- Contract: handlers validated against OpenAPI spec; generated TS client type-checks web.
- Payments: sandbox tests for each gateway (Stripe CLI webhooks, Moyasar/Tap test cards, PayPal sandbox); webhook replay fixtures.
- E2E (Playwright, ar + en, RTL screenshots): creator signup → create course → upload video → set price → publish → student checkout → watch → complete → certificate.
- Load: k6 on progress heartbeat, storefront, checkout.

---

## 13. Roadmap
| Phase | Weeks | Deliverables | Exit criteria |
|---|---|---|---|
| 0 Foundations | 1–2 | Monorepo, compose, CI, Go skeleton (config, db, RLS tx helper, problem+json, slog, OTel), migrations, OpenAPI pipeline + codegen, Next.js shell with ar/en RTL, host middleware | `task dev` boots all; health checks green; CI passes |
| 1 MVP | 3–10 | Creator signup + school + subdomain; branding; course builder (sections, lectures, video/text/PDF blocks, Bunny upload); student auth (password + magic link); free + one-time pricing; checkout with Moyasar + Stripe Connect; orders, enrollments; player + progress; default sales page template; transactional emails; creator students/sales views; basic platform admin | Private beta: real creator sells a course with mada end-to-end |
| 2 Monetize & learn | 11–18 | Tap + PayPal; billing engine (subscriptions, payment plans, dunning); coupons, bundles; quizzes, certificates, drip; comments; custom domains + on-demand TLS; VAT invoices + ZATCA Phase 1 QR; Puck page builder; SaaS plans + platform fee ledger; analytics dashboard | Public launch |
| 3 Growth | 19–26 | Email broadcasts + automations; affiliates; order bumps/upsells; coaching bookings; digital downloads; outgoing webhooks + API keys + Zapier; full team roles; ZATCA Phase 2; search (Postgres FTS → Meilisearch); Google/Apple login | Feature parity with Teachable core |
| 4 Expand | 27+ | Mobile apps (Flutter/RN, offline downloads), communities/memberships, live classes (Zoom/Meet), AI (captions, quiz generation, course outline assistant), multi-currency, optional platform-as-merchant (if licensed) | — |

Suggested team: 2 Go backend, 2 frontend, 1 product designer (RTL-savvy), 0.5 DevOps, 1 QA.

---

## 14. Execution after approval
1. Copy this plan to `D:\teatchable\docs\PLAN.md` + first ADRs (tenancy/RLS, merchant model, same-origin API).
2. Scaffold Phase 0: `backend/` (Go module, cmd/api, cmd/worker, internal/platform/*, first migration: accounts, schools, school_domains, school_members, students, sessions), `api/openapi/` (auth + school endpoints), `web/` (Next.js + Tailwind + shadcn + next-intl ar/en + host middleware), `docker-compose.yml`, `Taskfile.yml`, `.github/workflows/ci.yml`, `git init`.
3. Then Phase 1 modules in order: identity → school → catalog → media → commerce/payments (Moyasar first) → enrollment → player UI.

## Verification
- `docker compose up -d` then `task migrate seed dev` → API `/healthz` 200, web loads at `http://demo.localhost:3000` in ar (RTL) and en.
- `go test ./... -race` (incl. testcontainers RLS cross-tenant tests) and `golangci-lint run` pass.
- OpenAPI codegen diff empty in CI (spec, Go server, TS client in sync).
- Phase 1 E2E: Playwright script runs creator → course → Bunny upload → publish → student pays with Moyasar test mada card → server verifies payment → enrollment granted → lecture plays via signed URL → progress saved → receipt email in Mailpit.
- Webhook tests: `stripe listen --forward-to localhost/api/v1/webhooks/stripe`; Moyasar/Tap via tunnel to staging; replay same event twice → processed once.
