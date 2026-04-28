"""POST /v1/checkout/create-session — creates a LemonSqueezy hosted
Checkout for the requested tier and returns the URL the browser should
redirect to."""
from __future__ import annotations

import logging

from fastapi import APIRouter, HTTPException, status
from pydantic import BaseModel, EmailStr

from ..config import get_settings
from ..lemonsqueezy import LemonSqueezyError, create_checkout

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

    variant_id = s.variant_id_for_tier(body.tier)
    if not variant_id:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, f"Unknown tier: {body.tier}")

    try:
        url, checkout_id = create_checkout(
            api_key=s.ls_api_key,
            store_id=s.ls_store_id,
            variant_id=variant_id,
            email=body.email,
            redirect_url=s.success_url,
            cancel_url=s.cancel_url,
            # Echoed back on every subscription event so the webhook
            # handler can resolve which OCX tier the customer paid for
            # without having to look up the variant id.
            custom={"ocx_tier": body.tier},
        )
    except LemonSqueezyError as e:
        log.exception("LemonSqueezy create_checkout failed")
        raise HTTPException(
            status.HTTP_502_BAD_GATEWAY, f"Checkout provider error: {e}"
        ) from e

    return CreateSessionResponse(url=url, session_id=checkout_id)
