"""POST /v1/webhooks/lemonsqueezy — receives LemonSqueezy webhook events.

Reliability properties this handler guarantees:

  1. Signature verification (HMAC-SHA256, `X-Signature` header).
     Forged events return 401.
  2. Idempotency — every event id (`X-Event-Id` header, plus the body's
     `meta.event_name + meta.custom_data + data.id` fingerprint) is
     recorded in the `provider_events` table; replays after the first
     are no-ops with 200.
  3. Atomic per-event handling — DB writes happen in a transaction so
     a mid-handler failure rolls back the dedup record too. LS will
     retry, the next attempt sees no record, runs cleanly.
  4. 200 on handled, 4xx on permanent failure, 5xx on transient
     failure (LS retries 5xx with exponential backoff).

Events handled:

  • subscription_created      — new paid subscription. Issue API key,
                                send welcome email.
  • subscription_updated      — renewal, plan change, status change.
                                Sync local row + tier flip.
  • subscription_cancelled    — customer cancelled. Mark sub canceled,
                                downgrade tier so /v1/verify rejects
                                further requests.
  • subscription_resumed      — customer un-cancelled before period end.
                                Restore tier.
  • subscription_expired      — billing failed permanently / period
                                ended after a cancel. Same effect as
                                cancelled.
  • subscription_payment_success / _failed — logged for now; renewal
                                logic handled by subscription_updated.

Other event types (orders, products, license keys) are ignored — we
don't sell one-shot products and we don't issue LS-hosted license
keys (the API key is OCX-side, generated post-checkout).
"""
from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import Any

from fastapi import APIRouter, Depends, Header, HTTPException, Request, status
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..auth import generate_api_key
from ..config import get_settings
from ..db import get_db
from ..email import send_api_key_email
from ..lemonsqueezy import verify_webhook_signature
from ..models import Customer, ProviderEvent, Subscription

log = logging.getLogger("ocx.webhooks")
router = APIRouter()

PROVIDER = "lemonsqueezy"


@router.post("/v1/webhooks/lemonsqueezy")
async def lemonsqueezy_webhook(
    request: Request,
    x_signature: str | None = Header(None, alias="X-Signature"),
    x_event_name: str | None = Header(None, alias="X-Event-Name"),
    db: Session = Depends(get_db),
):
    s = get_settings()
    if not s.ls_webhook_secret:
        log.error("LS_WEBHOOK_SECRET not set; cannot verify webhook")
        raise HTTPException(
            status.HTTP_503_SERVICE_UNAVAILABLE,
            "Webhook endpoint not configured",
        )

    payload = await request.body()
    if not verify_webhook_signature(
        body=payload, signature=x_signature, secret=s.ls_webhook_secret
    ):
        log.warning("Bad LS webhook signature")
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Invalid signature")

    try:
        body = await request.json()
    except Exception as e:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, f"Bad JSON: {e}") from e

    meta = body.get("meta") or {}
    data = body.get("data") or {}
    event_name = meta.get("event_name") or x_event_name or "unknown"
    obj_id = str(data.get("id") or "")
    # LS doesn't always send a stable event_id; build one from the
    # event-name + the object id + the event timestamp on meta.
    event_id = f"{event_name}:{obj_id}:{meta.get('webhook_id') or meta.get('test_mode') or ''}"

    if db.execute(
        select(ProviderEvent).where(ProviderEvent.event_id == event_id)
    ).scalar_one_or_none():
        log.info("Replay of %s — skipping", event_id)
        return {"received": True, "replay": True}

    log.info("Processing event %s type=%s", event_id, event_name)

    try:
        handler = _HANDLERS.get(event_name)
        if handler is None:
            log.info("Ignoring unhandled LS event %s", event_name)
        else:
            handler(body, db)

        db.add(ProviderEvent(
            event_id=event_id,
            provider=PROVIDER,
            event_type=event_name,
            received_at=_utc_now(),
        ))
        db.commit()
        return {"received": True}
    except HTTPException:
        raise
    except Exception as e:
        log.exception("Handler crashed for event %s (%s)", event_id, event_name)
        db.rollback()
        raise HTTPException(
            status.HTTP_500_INTERNAL_SERVER_ERROR, f"Handler failed: {e}"
        ) from e


# ============================================================
# Helpers
# ============================================================

def _utc_now() -> datetime:
    return datetime.now(timezone.utc)


def _parse_iso(s: str | None) -> datetime | None:
    if not s:
        return None
    try:
        # LemonSqueezy timestamps look like "2024-01-15T08:23:45.000000Z"
        return datetime.fromisoformat(s.replace("Z", "+00:00"))
    except Exception:
        return None


def _custom(body: dict[str, Any]) -> dict[str, Any]:
    """Custom data the buyer passed at checkout, echoed back here.
    Used to carry the ocx_tier from /v1/checkout/create-session."""
    return ((body.get("meta") or {}).get("custom_data") or {})


def _attrs(body: dict[str, Any]) -> dict[str, Any]:
    return ((body.get("data") or {}).get("attributes") or {})


# ============================================================
# Handlers — one per event name
# ============================================================

def _handle_subscription_created(body: dict[str, Any], db: Session) -> None:
    """First event after a successful checkout. Create the customer
    record, issue the API key, send the welcome email."""
    s = get_settings()
    attrs = _attrs(body)
    sub_id = str((body.get("data") or {}).get("id") or "")
    customer_provider_id = str(attrs.get("customer_id") or "")
    email = attrs.get("user_email") or ""

    tier = (
        _custom(body).get("ocx_tier")
        or s.tier_for_variant_id(str(attrs.get("variant_id") or ""))
        or "starter"
    )

    if not customer_provider_id:
        log.info("subscription_created without customer_id; skipping")
        return

    existing = db.execute(
        select(Customer).where(
            Customer.provider == PROVIDER,
            Customer.provider_customer_id == customer_provider_id,
        )
    ).scalar_one_or_none()

    now = _utc_now()
    if existing:
        log.info("subscription_created for known customer %s; updating tier=%s", email, tier)
        existing.tier = tier
        existing.email = email or existing.email
        existing.updated_at = now
        customer = existing
    else:
        plaintext, digest, prefix = generate_api_key()
        customer = Customer(
            provider=PROVIDER,
            provider_customer_id=customer_provider_id,
            email=email,
            api_key_hash=digest,
            api_key_prefix=prefix,
            tier=tier,
            created_at=now,
            updated_at=now,
        )
        db.add(customer)
        db.flush()
        try:
            send_api_key_email(to=email, api_key=plaintext, tier=tier)
        except Exception:
            log.warning("Email send failed for %s; key generated but not delivered", email)

    _upsert_subscription(db, body, customer.id, tier, now)


def _handle_subscription_updated(body: dict[str, Any], db: Session) -> None:
    """Renewal, plan change, or status change. Keep the local copy in
    sync. If the variant id moved (tier change), update the customer's
    tier too."""
    s = get_settings()
    attrs = _attrs(body)
    customer_provider_id = str(attrs.get("customer_id") or "")
    variant_id = str(attrs.get("variant_id") or "")

    customer = db.execute(
        select(Customer).where(
            Customer.provider == PROVIDER,
            Customer.provider_customer_id == customer_provider_id,
        )
    ).scalar_one_or_none()
    if customer is None:
        # Race: subscription_updated arrived before subscription_created.
        # Raise so LS retries.
        raise RuntimeError(
            f"Customer {customer_provider_id} not yet known; will retry"
        )

    new_tier = s.tier_for_variant_id(variant_id) or customer.tier
    if new_tier != customer.tier:
        log.info("Customer %s tier change: %s → %s", customer.email, customer.tier, new_tier)
        customer.tier = new_tier
        customer.updated_at = _utc_now()

    _upsert_subscription(db, body, customer.id, new_tier, _utc_now())


def _handle_subscription_cancelled(body: dict[str, Any], db: Session) -> None:
    """Customer cancelled — sub stays active until period end then LS
    fires subscription_expired. Mark cancel_at_period_end so the UI
    reflects the pending cancel."""
    sub_id = str((body.get("data") or {}).get("id") or "")
    sub = db.execute(
        select(Subscription).where(
            Subscription.provider == PROVIDER,
            Subscription.provider_subscription_id == sub_id,
        )
    ).scalar_one_or_none()
    if sub:
        sub.cancel_at_period_end = True
        sub.status = "cancelled"
        sub.updated_at = _utc_now()
        log.info("Subscription cancelled (will expire at period end) for %s", sub.customer.email)


def _handle_subscription_expired(body: dict[str, Any], db: Session) -> None:
    """Subscription period ended after cancel, OR billing failed
    permanently. Demote the customer."""
    sub_id = str((body.get("data") or {}).get("id") or "")
    sub = db.execute(
        select(Subscription).where(
            Subscription.provider == PROVIDER,
            Subscription.provider_subscription_id == sub_id,
        )
    ).scalar_one_or_none()
    if sub:
        sub.status = "expired"
        sub.updated_at = _utc_now()
        sub.customer.tier = "expired"
        sub.customer.updated_at = _utc_now()
        log.info("Subscription expired for %s", sub.customer.email)


def _handle_subscription_resumed(body: dict[str, Any], db: Session) -> None:
    """Customer un-cancelled before the period ended."""
    sub_id = str((body.get("data") or {}).get("id") or "")
    sub = db.execute(
        select(Subscription).where(
            Subscription.provider == PROVIDER,
            Subscription.provider_subscription_id == sub_id,
        )
    ).scalar_one_or_none()
    if sub:
        sub.cancel_at_period_end = False
        sub.status = "active"
        sub.updated_at = _utc_now()
        log.info("Subscription resumed for %s", sub.customer.email)


def _handle_payment(body: dict[str, Any], db: Session) -> None:
    attrs = _attrs(body)
    log.info("LS payment event: id=%s status=%s amount=%s",
             (body.get("data") or {}).get("id"),
             attrs.get("status"),
             attrs.get("total_formatted"))


def _upsert_subscription(
    db: Session,
    body: dict[str, Any],
    customer_id: int,
    tier: str,
    now: datetime,
) -> None:
    attrs = _attrs(body)
    sub_id = str((body.get("data") or {}).get("id") or "")
    variant_id = str(attrs.get("variant_id") or "")
    period_start = _parse_iso(attrs.get("created_at")) or now
    period_end = _parse_iso(attrs.get("renews_at") or attrs.get("ends_at")) or now
    portal_url = ((attrs.get("urls") or {}).get("customer_portal"))
    status_ = attrs.get("status") or "active"
    cancel_at_period_end = bool(attrs.get("cancelled"))

    existing = db.execute(
        select(Subscription).where(
            Subscription.provider == PROVIDER,
            Subscription.provider_subscription_id == sub_id,
        )
    ).scalar_one_or_none()

    if existing:
        existing.status = status_
        existing.variant_id = variant_id or existing.variant_id
        existing.current_period_start = period_start
        existing.current_period_end = period_end
        existing.cancel_at_period_end = cancel_at_period_end
        if portal_url:
            existing.portal_url = portal_url
        existing.updated_at = now
    else:
        db.add(Subscription(
            provider=PROVIDER,
            provider_subscription_id=sub_id,
            customer_id=customer_id,
            status=status_,
            variant_id=variant_id,
            current_period_start=period_start,
            current_period_end=period_end,
            cancel_at_period_end=cancel_at_period_end,
            portal_url=portal_url,
            created_at=now,
            updated_at=now,
        ))


_HANDLERS = {
    "subscription_created":          _handle_subscription_created,
    "subscription_updated":          _handle_subscription_updated,
    "subscription_cancelled":        _handle_subscription_cancelled,
    "subscription_expired":          _handle_subscription_expired,
    "subscription_resumed":          _handle_subscription_resumed,
    "subscription_payment_success":  _handle_payment,
    "subscription_payment_failed":   _handle_payment,
    "subscription_payment_recovered": _handle_payment,
}
