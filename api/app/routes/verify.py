"""POST /v1/verify — the actual product. Customer authenticates with
their bearer API key, posts a hex-encoded receipt + public key, gets back
a verdict. Usage is metered; over-tier requests get 429."""
from __future__ import annotations

import binascii
import logging
import time
from datetime import datetime, timedelta, timezone

from fastapi import APIRouter, Depends, HTTPException, Response, status
from pydantic import BaseModel, Field
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..auth import get_current_customer
from ..db import get_db
from ..ffi_verify import is_available, verify
from ..models import Customer, Subscription, Usage
from ..tiers import get_tier

log = logging.getLogger("ocx.verify")
router = APIRouter()


class VerifyBody(BaseModel):
    cbor_hex: str = Field(..., description="Receipt CBOR bytes, hex-encoded")
    public_key_hex: str = Field(..., description="Issuer public key, 64 hex chars (32 bytes)")


class VerifyResponse(BaseModel):
    verified: bool
    error_code: int
    error_name: str
    elapsed_us: float
    tier: str
    usage: dict


def _utc_now() -> datetime:
    return datetime.now(timezone.utc)


def _current_period(customer: Customer, db: Session) -> tuple[datetime, datetime]:
    """Return the current billing period for this customer.
    Falls back to a 30-day rolling window if no active subscription is
    found (e.g. legacy customer; shouldn't happen in practice)."""
    sub = db.execute(
        select(Subscription)
        .where(Subscription.customer_id == customer.id)
        .where(Subscription.status.in_(["active", "trialing", "past_due"]))
        .order_by(Subscription.current_period_end.desc())
    ).scalars().first()
    if sub:
        return sub.current_period_start, sub.current_period_end
    # Fallback
    now = _utc_now()
    return now.replace(day=1), now.replace(day=1) + timedelta(days=30)


def _get_or_create_usage(customer_id: int, period_start: datetime, period_end: datetime, db: Session) -> Usage:
    usage = db.execute(
        select(Usage)
        .where(Usage.customer_id == customer_id)
        .where(Usage.period_start == period_start)
    ).scalar_one_or_none()
    if usage is None:
        usage = Usage(
            customer_id=customer_id,
            period_start=period_start,
            period_end=period_end,
            receipts_verified=0,
            bytes_processed=0,
        )
        db.add(usage)
        db.flush()
    return usage


@router.post("/v1/verify", response_model=VerifyResponse)
def verify_receipt(
    body: VerifyBody,
    response: Response,
    customer: Customer = Depends(get_current_customer),
    db: Session = Depends(get_db),
):
    if not is_available():
        raise HTTPException(
            status.HTTP_503_SERVICE_UNAVAILABLE,
            "Verifier unavailable on this server (libocx-verify not loaded)",
        )

    if customer.tier == "canceled":
        raise HTTPException(
            status.HTTP_402_PAYMENT_REQUIRED,
            "Subscription canceled. Re-subscribe to continue verifying.",
        )

    tier = get_tier(customer.tier)
    if tier is None:
        raise HTTPException(status.HTTP_403_FORBIDDEN, f"Unknown tier on account: {customer.tier}")

    # Decode hex inputs
    try:
        cbor_bytes = binascii.unhexlify(body.cbor_hex.replace(" ", "").replace("\n", ""))
        pubkey_bytes = binascii.unhexlify(body.public_key_hex.replace(" ", "").replace("\n", ""))
    except (binascii.Error, ValueError) as e:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, f"Invalid hex input: {e}") from e

    # Per-period usage check
    period_start, period_end = _current_period(customer, db)
    usage = _get_or_create_usage(customer.id, period_start, period_end, db)
    if usage.receipts_verified >= tier.monthly_receipts:
        # Tier limit hit. Tell them when the period resets.
        response.headers["X-RateLimit-Reset"] = str(int(period_end.timestamp()))
        raise HTTPException(
            status.HTTP_429_TOO_MANY_REQUESTS,
            f"Tier limit reached ({tier.monthly_receipts:,} receipts/month). Resets at {period_end.isoformat()}.",
        )

    # Do the actual verification
    t0 = time.perf_counter()
    result = verify(cbor_bytes, pubkey_bytes)
    elapsed_us = (time.perf_counter() - t0) * 1_000_000

    # Increment usage AFTER successful call (failures still count, see comment)
    # Rationale: failed verifications still consume verifier compute. We bill
    # for compute, not for true-positive verifications. This also prevents an
    # abusive client from flooding us with garbage to use compute "for free."
    usage.receipts_verified += 1
    usage.bytes_processed += len(cbor_bytes)
    usage.last_request_at = _utc_now()
    db.commit()

    response.headers["X-RateLimit-Limit"] = str(tier.monthly_receipts)
    response.headers["X-RateLimit-Remaining"] = str(tier.monthly_receipts - usage.receipts_verified)
    response.headers["X-RateLimit-Reset"] = str(int(period_end.timestamp()))

    return VerifyResponse(
        verified=result.ok,
        error_code=result.error_code,
        error_name=result.error_name,
        elapsed_us=round(elapsed_us, 2),
        tier=customer.tier,
        usage={
            "receipts_verified": usage.receipts_verified,
            "monthly_limit": tier.monthly_receipts,
            "period_start": period_start.isoformat(),
            "period_end": period_end.isoformat(),
        },
    )


@router.get("/v1/account")
def account(customer: Customer = Depends(get_current_customer), db: Session = Depends(get_db)):
    """GET /v1/account — basic info about the calling customer (tier, key prefix, usage)."""
    period_start, period_end = _current_period(customer, db)
    usage = _get_or_create_usage(customer.id, period_start, period_end, db)
    db.commit()
    tier = get_tier(customer.tier)
    return {
        "email": customer.email,
        "tier": customer.tier,
        "api_key_prefix": customer.api_key_prefix,
        "monthly_limit": tier.monthly_receipts if tier else 0,
        "usage": {
            "receipts_verified": usage.receipts_verified,
            "period_start": period_start.isoformat(),
            "period_end": period_end.isoformat(),
        },
    }
