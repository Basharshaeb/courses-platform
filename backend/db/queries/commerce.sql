-- ---- Products ----

-- name: CreateProduct :one
INSERT INTO products (id, school_id, kind, ref_id, name)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE school_id = $1 AND id = $2;

-- name: GetProductByRef :one
SELECT * FROM products WHERE school_id = $1 AND kind = $2 AND ref_id = $3;

-- name: ListProducts :many
SELECT * FROM products
WHERE school_id = $1
ORDER BY created_at DESC;

-- name: UpdateProduct :one
UPDATE products SET
  name   = coalesce(sqlc.narg(name), name),
  status = coalesce(sqlc.narg(status), status)
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- ---- Pricing Plans ----

-- name: CreatePricingPlan :one
INSERT INTO pricing_plans (id, school_id, product_id, type, amount, currency, tax_inclusive, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPricingPlan :one
SELECT * FROM pricing_plans WHERE school_id = $1 AND id = $2;

-- name: ListPricingPlansByProduct :many
SELECT * FROM pricing_plans
WHERE school_id = $1 AND product_id = $2 AND active = true
ORDER BY amount ASC;

-- name: DeactivatePricingPlans :exec
UPDATE pricing_plans SET active = false
WHERE school_id = $1 AND product_id = $2;

-- ---- Orders ----

-- name: CreateOrder :one
INSERT INTO orders (
  id, school_id, student_id, status, subtotal, discount, tax, total,
  currency, gateway, gateway_order_ref, buyer_name, buyer_email, idempotency_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE school_id = $1 AND id = $2;

-- name: GetOrderByRef :one
SELECT * FROM orders WHERE school_id = $1 AND gateway_order_ref = $2;

-- name: GetOrderByIdempotency :one
SELECT * FROM orders WHERE school_id = $1 AND idempotency_key = $2;

-- name: ListOrders :many
SELECT * FROM orders
WHERE school_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateOrderStatus :one
UPDATE orders SET
  status            = sqlc.arg(status),
  gateway_order_ref = coalesce(sqlc.narg(gateway_order_ref), gateway_order_ref)
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- ---- Order Items ----

-- name: CreateOrderItem :one
INSERT INTO order_items (id, school_id, order_id, product_id, pricing_plan_id, amount, tax)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListOrderItems :many
SELECT * FROM order_items
WHERE school_id = $1 AND order_id = $2;

-- ---- Payments ----

-- name: CreatePayment :one
INSERT INTO payments (id, school_id, order_id, gateway, gateway_payment_id, amount, currency, status, raw_response)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments SET
  status       = sqlc.arg(status),
  failure_code = sqlc.narg(failure_code),
  raw_response = coalesce(sqlc.narg(raw_response), raw_response)
WHERE school_id = sqlc.arg(school_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: GetPaymentByGatewayID :one
SELECT * FROM payments WHERE school_id = $1 AND gateway_payment_id = $2;

-- ---- Gateway Connections ----

-- name: UpsertGatewayConnection :one
INSERT INTO gateway_connections (
  id, school_id, gateway, mode, credentials_encrypted, publishable_key, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (school_id, gateway, mode) DO UPDATE SET
  credentials_encrypted = EXCLUDED.credentials_encrypted,
  publishable_key       = EXCLUDED.publishable_key,
  status                = EXCLUDED.status,
  updated_at            = now()
RETURNING *;

-- name: GetGatewayConnection :one
SELECT * FROM gateway_connections
WHERE school_id = $1 AND gateway = $2 AND mode = $3;

-- ---- Inbound Webhooks ----

-- name: InsertInboundWebhook :one
INSERT INTO inbound_webhook_events (id, gateway, event_id, payload)
VALUES ($1, $2, $3, $4)
ON CONFLICT (gateway, event_id) DO NOTHING
RETURNING *;

-- name: MarkInboundWebhookProcessed :exec
UPDATE inbound_webhook_events SET
  processed_at = now(),
  error        = sqlc.narg(error)
WHERE id = $1;
