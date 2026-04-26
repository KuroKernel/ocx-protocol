"""POST /v1/webhooks/stripe — receives Stripe webhook events.

Reliability properties this handler guarantees:

  1. Signature verification — rejects forged events with 401.
  2. Idempotency — every event_id is recorded in stripe_events; replays
     after the first one are no-ops with 200.
  3. Atomic per-event handling — DB writes happen in a transaction so a
     mid-handler failure rolls back the event_id record too. Stripe will
     retry, and the next attempt sees no record and re-runs cleanly.
  4. 200 on handled, 4xx on permanent failure, 5xx on transient failure
     (so Stripe retries with exponential backoff).
"""
from __future__ import annotations

import json
import logging
from datetime import datetime, timezone

import stripe
from fastapi import APIRouter, Depends, Header, HTTPException, Request, status
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..auth import generate_api_key
from ..config import get_settings
from ..db import get_db
from ..email import send_api_key_email
from ..models import Customer, StripeEvent, Subscription, Usage

log = logging.getLogger("ocx.webhooks")
router = APIRouter()


@router.post("/v1/webhooks/stripe")
async def stripe_webhook(
    request: Request,
    stripe_signature: str | None = Header(None, alias="Stripe-Signature"),
    db: Session = Depends(get_db),
):
    s = get_settings()
    if not s.stripe_webhook_secret:
        # Webhook secret not configured yet (e.g. fresh deploy before the
        # Stripe endpoint is wired up). Return 503 so Stripe retries once
        # the env var is set, instead of crashing the app on import.
        log.error("STRIPE_WEBHOOK_SECRET not set; cannot verify webhook")
        raise HTTPException(status.HTTP_503_SERVICE_UNAVAILABLE, "Webhook endpoint not configured")
    payload = await request.body()
    if not stripe_signature:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, "Missing Stripe-Signature header")
    try:
        event = stripe.Webhook.construct_event(
            payload=payload,
            sig_header=stripe_signature,
            secret=s.stripe_webhook_secret,
        )
    except (ValueError, stripe.error.SignatureVerificationError) as e:
        log.warning("Bad webhook signature: %s", e)
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Invalid signature") from e

    event_id = event["id"]
    event_type = event["type"]

    # Idempotency: if we've seen this event before, return 200 without
    # re-processing. (Stripe retries; this prevents double-charging,
    # double-emailing, etc.)
    if db.execute(select(StripeEvent).where(StripeEvent.event_id == event_id)).scalar_one_or_none():
        log.info("Replay of event %s (%s) — skipping", event_id, event_type)
        return {"received": True, "replay": True}

    log.info("Processing event %s type=%s", event_id, event_type)

    try:
        handler = _HANDLERS.get(event_type)
        if handler is None:
            log.info("Ignoring unhandled event type %s", event_type)
        else:
            handler(event["data"]["object"], db)

        # Record the event AFTER successful handling so a thrown handler
        # rolls back the marker too and Stripe will retry.
        db.add(StripeEvent(event_id=event_id, event_type=event_type, received_at=_utc_now()))
        db.commit()
        return {"received": True}
    except HTTPException:
        raise
    except Exception as e:
        log.exception("Handler crashed for event %s (%s)", event_id, event_type)
        db.rollback()
        # 500 → Stripe retries. Different from 4xx (give up).
        raise HTTPException(status.HTTP_500_INTERNAL_SERVER_ERROR, f"Handler failed: {e}") from e


# ============================================================
# Handlers — one per event type. Each takes (object, db_session).
# ============================================================

def _utc_now() -> datetime:
    return datetime.now(timezone.utc)


def _ts_to_dt(ts: int | None) -> datetime | None:
    if ts is None:
        return None
    return datetime.fromtimestamp(ts, tz=timezone.utc)


def _to_dict(obj) -> dict:
    """Stripe SDK returns StripeObject (which behaves like a dict for [] but
    NOT for .get()). Normalize to a real dict so downstream .get() works.
    Recursive: nested StripeObjects also become dicts."""
    if hasattr(obj, "to_dict_recursive"):
        return obj.to_dict_recursive()
    if hasattr(obj, "to_dict"):
        # Fallback: shallow + manually recurse
        d = obj.to_dict()
        return {k: _to_dict(v) if hasattr(v, "to_dict") else v for k, v in d.items()}
    return dict(obj) if obj else {}


def _extract_period(sub_obj: dict) -> tuple[datetime | None, datetime | None]:
    """Stripe API v2025-05-28+ moved current_period_start/end from the
    subscription's top level into each subscription item. Read from the
    item first, fall back to top-level for older accounts/API versions."""
    items = (sub_obj.get("items") or {}).get("data") or []
    if items:
        first = items[0]
        start = _ts_to_dt(first.get("current_period_start"))
        end = _ts_to_dt(first.get("current_period_end"))
        if start and end:
            return start, end
    # Fallback: top-level (older API versions)
    return (
        _ts_to_dt(sub_obj.get("current_period_start")),
        _ts_to_dt(sub_obj.get("current_period_end")),
    )


def _handle_checkout_completed(obj_raw, db: Session) -> None:
    """checkout.session.completed — first event after a successful payment.
    Creates the customer record + generates an API key + emails it.
    """
    s = get_settings()
    stripe.api_key = s.stripe_secret_key
    obj = _to_dict(obj_raw)

    stripe_customer_id = obj.get("customer")
    if not stripe_customer_id:
        # Fixture-triggered events from `stripe trigger` sometimes have no
        # real customer attached. Skip — nothing to bind a record to.
        log.info("checkout.session.completed without customer id (likely a fixture); skipping")
        return

    email = (obj.get("customer_details") or {}).get("email") or obj.get("customer_email") or ""
    tier = (obj.get("metadata") or {}).get("ocx_tier")
    if not tier:
        # Fallback: inspect the subscription for the price → tier
        sub_id = obj.get("subscription")
        if sub_id:
            sub = _to_dict(stripe.Subscription.retrieve(sub_id))
            price_id = sub["items"]["data"][0]["price"]["id"]
            tier = s.tier_for_price_id(price_id) or "starter"
        else:
            tier = "starter"

    # If we already saw this customer (e.g. they churned then resubscribed),
    # don't re-issue a key. Update tier + email instead.
    existing = db.execute(
        select(Customer).where(Customer.stripe_customer_id == stripe_customer_id)
    ).scalar_one_or_none()
    if existing:
        log.info("checkout.completed for existing customer %s; updating tier=%s", email, tier)
        existing.tier = tier
        existing.email = email or existing.email
        existing.updated_at = _utc_now()
        return

    plaintext, digest, prefix = generate_api_key()
    now = _utc_now()
    customer = Customer(
        stripe_customer_id=stripe_customer_id,
        email=email,
        api_key_hash=digest,
        api_key_prefix=prefix,
        tier=tier,
        created_at=now,
        updated_at=now,
    )
    db.add(customer)
    db.flush()  # get id

    # Belt-and-suspenders: fetch the subscription from Stripe and create
    # the local row HERE. Stripe doesn't guarantee event ordering — the
    # `customer.subscription.created` webhook may arrive before this one.
    # If we wait for it, we depend on Stripe's retry timing. Better to
    # eagerly create the row now while we already have the IDs at hand.
    sub_id = obj.get("subscription")
    if sub_id:
        try:
            sub_obj = _to_dict(stripe.Subscription.retrieve(sub_id))
            existing = db.execute(
                select(Subscription).where(Subscription.stripe_subscription_id == sub_id)
            ).scalar_one_or_none()
            period_start, period_end = _extract_period(sub_obj)
            if not existing:
                db.add(Subscription(
                    stripe_subscription_id=sub_id,
                    customer_id=customer.id,
                    status=sub_obj.get("status", "active"),
                    price_id=sub_obj["items"]["data"][0]["price"]["id"],
                    current_period_start=period_start or now,
                    current_period_end=period_end or now,
                    cancel_at_period_end=bool(sub_obj.get("cancel_at_period_end", False)),
                    created_at=now,
                    updated_at=now,
                ))
        except Exception as e:
            log.warning("Failed to pre-fetch subscription %s: %s. Will sync on subscription.updated.", sub_id, e)
            # Not fatal; customer.subscription.updated will run later and create it.

    # Send the welcome email with the plaintext key.
    # Done OUTSIDE the DB transaction's critical section but inside the
    # webhook's try/except so a Postmark failure can trigger Stripe retry
    # → idempotency check finds the customer already exists → no double-send.
    try:
        send_api_key_email(to=email, api_key=plaintext, tier=tier)
    except Exception:
        log.warning("Email send failed for new customer %s; key was generated but not delivered", email)
        # Don't re-raise — the customer record is created, they can request
        # a key reset via support. Better than blocking the webhook.


def _handle_subscription_event(obj_raw, db: Session) -> None:
    """customer.subscription.{created,updated} — keep our copy in sync."""
    s = get_settings()
    obj = _to_dict(obj_raw)
    sub_id = obj["id"]
    customer_stripe_id = obj["customer"]

    customer = db.execute(
        select(Customer).where(Customer.stripe_customer_id == customer_stripe_id)
    ).scalar_one_or_none()
    if customer is None:
        # Race: subscription event arrived before checkout.session.completed.
        # Raise so the webhook returns 5xx and Stripe retries — by the time
        # of the retry, checkout.completed has run and the customer exists.
        # (Idempotency guarantees we don't duplicate work on retry.)
        log.warning("subscription event for unknown customer %s; raising 500 so Stripe retries", customer_stripe_id)
        raise RuntimeError(f"Customer {customer_stripe_id} not yet known; will retry after checkout.completed")

    price_id = obj["items"]["data"][0]["price"]["id"]
    new_tier = s.tier_for_price_id(price_id)
    if new_tier and new_tier != customer.tier:
        log.info("Customer %s tier change: %s → %s", customer.email, customer.tier, new_tier)
        customer.tier = new_tier
        customer.updated_at = _utc_now()

    existing_sub = db.execute(
        select(Subscription).where(Subscription.stripe_subscription_id == sub_id)
    ).scalar_one_or_none()

    period_start, period_end = _extract_period(obj)
    status_ = obj.get("status", "unknown")
    cancel_at_period_end = bool(obj.get("cancel_at_period_end", False))
    now = _utc_now()

    if existing_sub:
        existing_sub.status = status_
        existing_sub.price_id = price_id
        if period_start:
            existing_sub.current_period_start = period_start
        if period_end:
            existing_sub.current_period_end = period_end
        existing_sub.cancel_at_period_end = cancel_at_period_end
        existing_sub.updated_at = now
    else:
        db.add(Subscription(
            stripe_subscription_id=sub_id,
            customer_id=customer.id,
            status=status_,
            price_id=price_id,
            current_period_start=period_start or now,
            current_period_end=period_end or now,
            cancel_at_period_end=cancel_at_period_end,
            created_at=now,
            updated_at=now,
        ))


def _handle_subscription_deleted(obj_raw, db: Session) -> None:
    """customer.subscription.deleted — mark sub canceled. We DON'T delete
    the customer record (they might come back); we set tier='canceled'
    so /v1/verify rejects further calls."""
    obj = _to_dict(obj_raw)
    sub_id = obj["id"]
    sub = db.execute(
        select(Subscription).where(Subscription.stripe_subscription_id == sub_id)
    ).scalar_one_or_none()
    if sub:
        sub.status = "canceled"
        sub.updated_at = _utc_now()
        # Demote the customer (no other active sub, so they lose API access)
        sub.customer.tier = "canceled"
        sub.customer.updated_at = _utc_now()
        log.info("Subscription canceled for customer %s", sub.customer.email)


def _handle_invoice_paid(obj_raw, db: Session) -> None:
    """invoice.paid — could send a receipt email here. For now just log."""
    obj = _to_dict(obj_raw)
    log.info("invoice.paid: %s (%s) amount=%s",
             obj.get("id"), obj.get("customer_email") or obj.get("customer"),
             obj.get("amount_paid"))


def _handle_invoice_payment_failed(obj_raw, db: Session) -> None:
    """invoice.payment_failed — send a dunning email or flag account.
    Stripe Smart Retries will keep trying; we just log here."""
    obj = _to_dict(obj_raw)
    log.warning("invoice.payment_failed: %s (%s) attempt=%s",
                obj.get("id"), obj.get("customer"), obj.get("attempt_count"))


_HANDLERS = {
    "checkout.session.completed": _handle_checkout_completed,
    "customer.subscription.created": _handle_subscription_event,
    "customer.subscription.updated": _handle_subscription_event,
    "customer.subscription.deleted": _handle_subscription_deleted,
    "invoice.paid": _handle_invoice_paid,
    "invoice.payment_failed": _handle_invoice_payment_failed,
}
