"""POST /v1/billing/portal — returns a one-time URL to the Stripe-hosted
Customer Portal where the customer can update payment method, change tier,
download invoices, or cancel.

Authentication: bearer API key (the customer's plaintext key from the
welcome email). We look up their Stripe customer ID and create a portal
session.
"""
from __future__ import annotations

import logging

import stripe
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

from ..auth import get_current_customer
from ..config import get_settings
from ..models import Customer

log = logging.getLogger("ocx.portal")
router = APIRouter()


class PortalResponse(BaseModel):
    url: str


@router.post("/v1/billing/portal", response_model=PortalResponse)
def billing_portal(customer: Customer = Depends(get_current_customer)):
    s = get_settings()
    stripe.api_key = s.stripe_secret_key
    return_url = f"{s.app_url.rstrip('/')}/account"
    try:
        session = stripe.billing_portal.Session.create(
            customer=customer.stripe_customer_id,
            return_url=return_url,
        )
    except stripe.error.StripeError as e:
        log.exception("stripe.billing_portal.Session.create failed")
        raise HTTPException(status.HTTP_502_BAD_GATEWAY, f"Stripe error: {e.user_message or str(e)}") from e
    return PortalResponse(url=session.url)
