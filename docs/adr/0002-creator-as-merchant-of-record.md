# ADR 0002: Creator is merchant of record

- Status: Accepted
- Date: 2026-10-15

## Context
Payments must support Stripe, Moyasar, Tap and PayPal, with Saudi methods (mada, Apple Pay, STC Pay).
If the platform collects money on behalf of creators it becomes a payment facilitator, which in
Saudi Arabia requires SAMA licensing and a payout system.

## Decision
- Each school connects its own gateway account: Stripe Connect (Express/Standard), or its own
  Moyasar / Tap / PayPal API credentials stored encrypted in `gateway_connections`.
- Invoices are issued in the creator's name with the creator's VAT number.
- Platform fee: Stripe `application_fee_amount`; other gateways accrue fees in
  `platform_fee_ledger` and the platform bills the creator monthly.
- All gateways sit behind one `payments.Gateway` interface; recurring charges run through one
  internal billing engine (except PayPal Subscriptions).

## Consequences
- No SAMA payment-facilitator license needed at launch; PCI scope stays SAQ-A (gateways tokenize).
- Creators must complete gateway onboarding before selling paid products.
- Collecting fees from non-Stripe gateways depends on creator billing; unpaid fees can suspend a school.
- Platform-as-merchant can be revisited later (roadmap phase 4) if licensed.
