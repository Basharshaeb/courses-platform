# Teatchable Strategic Product Roadmap & Feature Plan

This document outlines the architectural plan, phased deliverables, and implementation milestones for expanding **Teatchable** into a premier course platform for the Saudi/GCC and global markets.

---

## 1. Roadmap Overview & Milestones

```
  Phase 1: Production MVP (Weeks 1-4)
  └── Complete Moyasar & Stripe Checkout ──▶ Bunny Stream TUS Uploads ──▶ ZATCA Invoices ──▶ Email Dispatcher
        │
  Phase 2: Monetize & Assess (Weeks 5-10)
  └── Subscriptions & Dunning ──▶ Quizzes & Assignments ──▶ PDF Certificates ──▶ Comments & Discussion
        │
  Phase 3: Scale & Expand (Weeks 11-18)
  └── Email Broadcasts & Automations ──▶ Affiliate System ──▶ Block Page Builder ──▶ AI Course Assistant
```

---

## 2. Phase 1: MVP Finalization (Production Readiness)

### Milestone 1.1: Complete Checkout & Payment Gateway Integration
- **Goal**: Enable creators to collect payments using Saudi mada cards, Apple Pay, STC Pay, and international credit cards.
- **Backend Components**:
  - `commerce.Service.CreateCheckoutSession`: Generate signed Moyasar payment session or Stripe PaymentIntent.
  - Verification webhook: `POST /api/v1/webhooks/moyasar` & `POST /api/v1/webhooks/stripe` to mark orders `paid` idempotently.
  - Auto-enrollment grant upon payment verification.
- **Frontend Components**:
  - Checkout page: `web/src/app/site/[host]/checkout/[planId]/page.tsx`.
  - Moyasar form integration: mada and Apple Pay button with dynamic fee calculation.
  - Order success page: `web/src/app/site/[host]/thank-you/[orderId]/page.tsx`.

### Milestone 1.2: Direct-to-CDN Video Uploading (Bunny Stream)
- **Goal**: Upload multi-gigabyte video lectures directly from the creator's browser without loading the Go API server.
- **Backend**:
  - `POST /admin/schools/{schoolId}/media/uploads`: Generates SHA-256 Bunny TUS authorization signature.
  - Webhook listener: `POST /webhooks/bunny` to record encoding complete, thumbnail URL, and exact duration.
- **Frontend**:
  - Reusable drag-and-drop video uploader in `LectureEditor` utilizing `tus-js-client`.
  - Real-time progress bar (percentage, upload speed, time remaining).

### Milestone 1.3: Tax Settings & ZATCA Phase 1 Invoicing
- **Goal**: Issue legally compliant simplified tax invoices with QR codes for Saudi creators.
- **Backend**:
  - TLV (Tag-Length-Value) Base64 encoder for ZATCA Phase 1 QR: Seller Name, VAT Number, Timestamp, Total, Tax Amount.
  - River Job: `invoice.generate` calling Gotenberg (headless Chromium) with Arabic RTL HTML template to produce PDF.
  - Save generated PDF to S3/R2 and attach link to student order details.

### Milestone 1.4: Transactional Emails (SES & Mailpit)
- **Goal**: Send branded transactional notifications.
- **Job Triggers**:
  - Account signup verification & Magic login links.
  - Order receipt with invoice PDF attachment.
  - Course enrollment confirmation with direct lesson link.
- **Tech**: River worker running `email.send` via AWS SES SDK (points to Mailpit on `localhost:51025` in development).

---

## 3. Phase 2: Monetization & Engagement

### Milestone 2.1: Recurring Subscriptions & Installment Plans
- **Schema Additions**:
  - `subscriptions` (school_id, student_id, pricing_plan_id, status, next_billing_at, installments_paid).
- **Billing Engine (River Periodic Job)**:
  - Runs every 5 minutes: Queries `subscriptions` where `next_billing_at <= now()`.
  - Initiates off-session charge using saved payment token.
  - Dunning process: Auto-retry after +1 day, +3 days, and +7 days. On exhaustion, sets status to `past_due` and revokes enrollment.
- **Student Portal**:
  - Manage payment methods, cancel subscription, download past receipts.

### Milestone 2.2: Quizzes, Question Banks & Assignments
- **Schema Additions**:
  - `quizzes` (lecture_block_id, pass_score_pct, max_attempts, shuffle).
  - `quiz_questions` (type: single_choice, multiple_choice, true_false, short_answer).
  - `quiz_attempts` & `quiz_answers`.
- **Learning Player Experience**:
  - Interactive quiz taker inside the lesson player.
  - Instant scoring with answer explanations.
  - Requirement: passing quiz unlocks subsequent drip lessons.

### Milestone 2.3: Automated Certificates of Completion
- **Features**:
  - Certificate template designer in creator dashboard (upload background image, set custom signature, title).
  - Public verification URL: `https://{school}.yourapp.com/certificates/verify/{code}` allowing employers to verify student credentials.
  - River Job: `certificate.issue` triggered when `enrollment.progress_pct == 100`.

### Milestone 2.4: Lecture Discussions & Comments
- **Features**:
  - Threaded comment discussion forum beneath each lecture.
  - Creator moderation controls: approve, pin, or delete comments.
  - Real-time email notification to the creator when a student posts a question.

### Milestone 2.5: Custom Domain Automation
- **Features**:
  - Creator adds `academy.example.com` in dashboard settings.
  - Background job verifies DNS CNAME record pointing to edge proxy.
  - Caddy issues on-demand SSL certificate automatically via ACME / Let's Encrypt.

---

## 4. Phase 3: Scale & Growth Marketing

### Milestone 3.1: Email Marketing Broadcasts & Automations
- Segment students by course, enrollment date, or completion status.
- Schedule broadcasts and automated sequences (e.g., send congratulatory email 3 days after completing Unit 1).

### Milestone 3.2: Affiliate Referral Marketing
- Affiliate codes and unique referral tracking links.
- Cookie tracking window (configurable 30–90 days).
- Commission calculation on paid orders with monthly creator payout reports.

### Milestone 3.3: Visual Block Page Builder
- Integration of Puck block page editor for creating custom high-converting landing pages, sales funnels, and course pages.

### Milestone 3.4: AI Course Creator Toolkit
- AI audio transcription & bilingual subtitle generator (Arabic + English).
- Automated quiz question generation from video lecture transcripts.
- AI Course Outline generator assisting creators during initial setup.

---

## 5. Technical Requirements & Definition of Done

For each feature to be considered complete, the following standard applies:
1. **Database**: Migrations include composite unique keys `(id, school_id)` and strict RLS tenant isolation.
2. **OpenAPI**: Endpoints declared in `api/openapi/openapi.yaml` with strict schemas; code generated via `task gen`.
3. **Backend**: Strict-server Go handler, transport-agnostic module service, unit & integration tests passing.
4. **Frontend**: Accessible, RTL-first Next.js UI using shadcn components; zero TypeScript or ESLint warnings.
5. **Observability**: Structured `slog` logging with tenant ID, request ID, and River queue tracing.
