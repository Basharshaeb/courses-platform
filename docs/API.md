# Teatchable API Specification & Developer Guide

Welcome to the **Teatchable API** documentation. The API powers our multi-tenant online academy platform, providing endpoints for creators to build and monetize courses, and for students to purchase, learn, and track progress.

The API contract is defined using OpenAPI 3.0.3 in [`api/openapi/openapi.yaml`](file:///d:/teatchable/api/openapi/openapi.yaml) and serves as the single source of truth for both the Go backend (`oapi-codegen`) and the Next.js TypeScript client (`openapi-fetch`).

---

## 1. Architectural Foundations

### Same-Origin Architecture
All API requests are served under `/api/v1` on **every** host (see [docs/adr/0003-same-origin-api.md](file:///d:/teatchable/docs/adr/0003-same-origin-api.md)):
- `*.yourapp.com` & Custom Domains: Storefront, course sales, checkout, and student learning.
- `app.yourapp.com`: Creator dashboard, course curriculum editor, orders, and academy settings.
- `admin.yourapp.com`: Platform administration and global governance.

This eliminates 3rd-party cookie blocking issues across browsers (Safari ITP, Chrome Privacy Sandbox) and allows all session cookies to remain first-party.

### Tenant Resolution & Row-Level Security (RLS)
The edge proxy (Caddy) sets `X-Forwarded-Host`. The Go API resolves the host to a `school_id`, injects it into the request context, and executes:
```sql
SET LOCAL app.school_id = '<school-uuid>';
```
Every database query in Postgres enforces Row-Level Security, preventing any cross-tenant data leaks at the database engine level.

### Authentication & Sessions
- **Session Tokens**: 32-byte cryptographically secure random bytes (`crypto/rand`).
- **Storage**: Only SHA-256 hashes of session tokens are stored in Postgres.
- **Cookies**: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` (over HTTPS).
- **CSRF Protection**: SameSite enforcement combined with strict `Origin` header verification on all mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`).

### RFC 9457 Problem Errors
Errors are returned with `Content-Type: application/problem+json` and stable machine-readable codes:
```json
{
  "type": "https://yourapp.com/errors/invalid_request",
  "title": "Validation Failed",
  "status": 400,
  "code": "slug_taken",
  "detail": "another course already uses this URL name",
  "requestId": "req_01a16312e1"
}
```

---

## 2. Global Endpoints

### Health & Readiness Checks
Used by Kubernetes probes and edge monitors.

#### `GET /healthz`
Liveness probe. Returns HTTP 200 if the process is up.
```json
{
  "status": "ok"
}
```

#### `GET /readyz`
Readiness probe. Verifies PostgreSQL connection pool ping.
- **200 OK**: Database reachable.
- **503 Service Unavailable**: Database disconnected.

#### `GET /api/v1/internal/tls-allowed?domain={domain}`
Called by Caddy's `on_demand_tls` module before issuing an automatic Let's Encrypt certificate for a custom domain.
- **Query Parameter**: `domain` (string, max 253 chars).
- **200 OK**: Domain is verified and belongs to an active school.
- **403 Forbidden**: Domain unverified.

---

## 3. Creator Accounts & Authentication (App Host)

Endpoints mounted on `app.yourapp.com`.

### `POST /api/v1/accounts/signup`
Creates a creator account and initializes their primary school with a unique subdomain.
- **Request Body**:
```json
{
  "email": "creator@demo.test",
  "password": "strong-password-123",
  "name": "Khalid Al-Otaibi",
  "schoolName": "أكاديمية السحابة",
  "subdomain": "cloud-academy",
  "locale": "ar"
}
```
- **Responses**:
  - `201 Created`: Account & School created. Sets session cookie `sid`.
  - `400 Bad Request`: Validation failure (`invalid_subdomain`).
  - `409 Conflict`: `email_taken` or `subdomain_taken`.

### `POST /api/v1/accounts/login`
Authenticates a creator or school staff member.
- **Request Body**:
```json
{
  "email": "creator@demo.test",
  "password": "strong-password-123"
}
```
- **Responses**:
  - `200 OK`: Sets session cookie `sid`. Returns `AccountSession`.
  - `401 Unauthorized`: Invalid email or password.

### `GET /api/v1/accounts/me`
Retrieves the authenticated creator account and their school memberships.
- **Security**: Requires creator session cookie.
- **Response**:
```json
{
  "account": {
    "id": "01948834-4b57-73d7-83d3-9bc7c4930be6",
    "email": "creator@demo.test",
    "name": "Demo Creator",
    "locale": "ar",
    "createdAt": "2026-10-21T10:00:00Z"
  },
  "schools": [
    {
      "schoolId": "01948834-4b67-7b89-a2e4-df192cb266d1",
      "name": "أكاديمية التجربة",
      "subdomain": "demo",
      "role": "owner"
    }
  ]
}
```

### `POST /api/v1/accounts/logout`
Revokes the current session and clears the session cookie.

---

## 4. Student Authentication (School Host)

Endpoints mounted on `{school}.yourapp.com` or custom domains.

### `POST /api/v1/auth/signup`
Registers a student scoped to the current school.
- **Request Body**:
```json
{
  "email": "student@demo.test",
  "password": "student-password-123",
  "name": "Fahad Al-Harbi",
  "locale": "ar"
}
```
- **Response `201 Created`**: Returns student profile and sets first-party session cookie.

### `POST /api/v1/auth/login`
Authenticates a student within the school.

### `GET /api/v1/me`
Retrieves the logged-in student's profile.
```json
{
  "id": "01948834-58a1-7c9f-86f3-42e1a49688bc",
  "email": "student@demo.test",
  "name": "Fahad Al-Harbi",
  "locale": "ar",
  "createdAt": "2026-10-21T11:00:00Z"
}
```

---

## 5. Public Storefront

Publicly accessible on the school host without authentication.

### `GET /api/v1/storefront/school`
Returns public branding and configuration for the school resolved on the host.
```json
{
  "id": "01948834-4b67-7b89-a2e4-df192cb266d1",
  "name": "أكاديمية التجربة",
  "subdomain": "demo",
  "defaultLocale": "ar",
  "currency": "SAR"
}
```

### `GET /api/v1/storefront/courses`
Returns all published courses of the school, sorted newest first.
```json
[
  {
    "id": "01a16312-e1eb-77c4-bb7a-141d13993f86",
    "slug": "go-microservices",
    "title": "بناء الأنظمة الموزعة والخدمات المصغرة بلغة Go",
    "subtitle": "دليل شامل وعملي لاحتراف تطوير الأنظمة السحابية عالية الأداء.",
    "locale": "ar",
    "lectureCount": 4,
    "publishedAt": "2026-10-22T10:05:44Z"
  }
]
```

### `GET /api/v1/storefront/courses/{slug}`
Fetches full public course syllabus, including sections, lectures, and free preview indicators.
```json
{
  "id": "01a16312-e1eb-77c4-bb7a-141d13993f86",
  "slug": "go-microservices",
  "title": "بناء الأنظمة الموزعة والخدمات المصغرة بلغة Go",
  "subtitle": "دليل شامل وعملي لاحتراف تطوير الأنظمة السحابية عالية الأداء.",
  "description": "تفاصيل ومخرجات الدورة التدريبية...",
  "locale": "ar",
  "sections": [
    {
      "id": "01a16312-e1ee-7d52-873b-e85df6495b6c",
      "title": "الوحدة 1: الأساسيات المعمارية والبيئة التطويرية",
      "lectures": [
        {
          "id": "01a16312-e1ef-79d1-81ee-bbd31e9c1db4",
          "title": "مقدمة إلى الأنظمة الموزعة والتحديات الهندسية",
          "isFreePreview": true,
          "durationSec": 870
        }
      ]
    }
  ]
}
```

---

## 6. Creator Course & Curriculum Builder

Mounted on `app.yourapp.com/api/v1/admin/schools/{schoolId}/...`.
Requires school membership with `courses.view` or `courses.edit` permissions (`owner`, `admin`, or `author`).

### Courses CRUD

| Method | Endpoint | Description | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/admin/schools/{schoolId}/courses` | List all courses (draft & published) | `courses.view` |
| `POST` | `/admin/schools/{schoolId}/courses` | Create a new draft course | `courses.edit` |
| `GET` | `/admin/schools/{schoolId}/courses/{courseId}` | Get course detail & syllabus | `courses.view` |
| `PATCH`| `/admin/schools/{schoolId}/courses/{courseId}` | Update course title, slug, subtitle | `courses.edit` |
| `DELETE`| `/admin/schools/{schoolId}/courses/{courseId}` | Delete a course | `courses.edit` |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/publish` | Publish course to storefront | `courses.edit` |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/unpublish` | Revert course to draft | `courses.edit` |

### Curriculum Management (Sections & Lectures)

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/sections` | Add section |
| `PATCH` | `/admin/schools/{schoolId}/courses/{courseId}/sections/{sectionId}` | Rename section |
| `DELETE` | `/admin/schools/{schoolId}/courses/{courseId}/sections/{sectionId}` | Delete section & lectures |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/sections/{sectionId}/lectures` | Add lecture to section |
| `GET` | `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}` | Get lecture content blocks |
| `PATCH` | `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}` | Update lecture & free preview |
| `DELETE` | `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}` | Delete lecture |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/curriculum/reorder` | Bulk reorder sections & lectures |

### Content Blocks

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks` | Add content block (`text`, `video`, `pdf`, `embed`) |
| `PATCH`| `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks/{blockId}` | Update block content |
| `DELETE`| `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks/{blockId}`| Remove block |
| `POST` | `/admin/schools/{schoolId}/courses/{courseId}/lectures/{lectureId}/blocks/reorder` | Reorder blocks |

---

## 7. Media & Video Uploads

### `POST /api/v1/admin/schools/{schoolId}/media/uploads`
Prepares a direct client-to-CDN upload for video or storage files.
- **Request Body**:
```json
{
  "filename": "lesson-1-intro.mp4",
  "kind": "video"
}
```
- **Response `200 OK`**:
```json
{
  "assetId": "01a16312-f100-7ec8-9b8e-3d192840ac90",
  "provider": "bunny",
  "uploadUrl": "https://video.bunnycdn.com/tusupload",
  "headers": {
    "LibraryId": "123456",
    "VideoId": "01a16312-f100-7ec8-9b8e-3d192840ac90",
    "AuthorizationSignature": "sha256_hash_here",
    "AuthorizationExpire": "1730000000"
  }
}
```
The client uploads directly to Bunny Stream via `tus-js-client` without routing video payload through the Go API.

---

## 8. Webhooks

Incoming webhooks are idempotent; each event is recorded in `inbound_webhook_events(gateway, event_id)` and processed asynchronously via River jobs.

| Provider | Endpoint | Signature Check |
| :--- | :--- | :--- |
| **Moyasar** | `POST /api/v1/webhooks/moyasar` | HTTP Basic Auth / API Secret verification |
| **Stripe** | `POST /api/v1/webhooks/stripe` | `Stripe-Signature` HMAC SHA-256 header |
| **Bunny** | `POST /api/v1/webhooks/bunny` | Bunny Webhook Secret authentication |
