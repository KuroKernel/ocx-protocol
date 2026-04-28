"""POST /v1/checkout/create-session — creates a Stripe Checkout session
for the requested tier and returns the URL the browser should redirect to.
"""
from __future__ import annotations

import logging

import stripe
from fastapi import APIRouter, HTTPException, status
from pydantic import BaseModel, EmailStr

from ..config import get_settings

log = logging.getLogger("ocx.checkout")
router = APIRouter()


class CreateSessionBody(BaseModel):
    tier: str  # "starter" | "growth" | "scale"
    email: EmailStr | None = None  # optional pre-fill


class CreateSessionResponse(BaseModel):
    url: str
    session_id: str


@router.post("/v1/checkout/create-session", response_model=CreateSessionResponse)
def create_session(body: CreateSessionBody):
    s = get_settings()
    stripe.api_key = s.stripe_secret_key

    price_id = s.price_id_for_tier(body.tier)
    if not price_id:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, f"Unknown tier: {body.tier}")

    try:
        session = stripe.checkout.Session.create(
            mode="subscription",
            line_items=[{"price": price_id, "quantity": 1}],
            success_url=s.success_url,
            cancel_url=s.cancel_url,
            customer_email=body.email,
            allow_promotion_codes=True,
            billing_address_collection="auto",
            # tax id collection is fine to enable; comment out if you don't want it
            tax_id_collection={"enabled": True},
            # Force USD-only checkout. The marketing page advertises USD
            # prices, the Price objects are USD, the customer sees USD at
            # checkout — no IP-based local-currency conversion. Belt-and-
            # suspenders against the account-level Adaptive Pricing toggle
            # being flipped back on.
            adaptive_pricing={"enabled": False},
            subscription_data={
                "metadata": {"ocx_tier": body.tier},
            },
            # Carry the tier through to the webhook (covers the rare case
            # where Stripe events arrive without subscription_data metadata
            # populated — we still know what the customer paid for).
            metadata={"ocx_tier": body.tier},
        )
    except stripe.error.StripeError as e:
        log.exception("stripe.checkout.Session.create failed")
        raise HTTPException(status.HTTP_502_BAD_GATEWAY, f"Stripe error: {e.user_message or str(e)}") from e

    return CreateSessionResponse(url=session.url, session_id=session.id)
