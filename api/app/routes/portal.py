"""POST /v1/billing/portal — returns the LemonSqueezy customer portal URL
for the authenticated subscriber. They can update card, cancel, or pause
from the LS-hosted page.

LemonSqueezy gives us a per-subscription `urls.customer_portal` link in
every subscription webhook event; we cache that on the Subscription row
at write time, so this endpoint just reads it. No live LS API call
needed.

Authentication: bearer API key (the customer's plaintext key from the
welcome email).
"""
from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..auth import get_current_customer
from ..db import get_db
from ..models import Customer, Subscription

log = logging.getLogger("ocx.portal")
router = APIRouter()


class PortalResponse(BaseModel):
    url: str


@router.post("/v1/billing/portal", response_model=PortalResponse)
def billing_portal(
    customer: Customer = Depends(get_current_customer),
    db: Session = Depends(get_db),
):
    sub = db.execute(
        select(Subscription)
        .where(Subscription.customer_id == customer.id)
        .order_by(Subscription.updated_at.desc())
    ).scalars().first()

    if sub is None or not sub.portal_url:
        # Either we never received a subscription webhook for this
        # customer, or the provider didn't include a portal URL on the
        # most recent event.
        raise HTTPException(
            status.HTTP_404_NOT_FOUND,
            "No customer portal URL on file. Email hello@ocx.world.",
        )

    return PortalResponse(url=sub.portal_url)
