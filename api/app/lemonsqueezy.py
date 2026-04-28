"""Minimal LemonSqueezy client.

LemonSqueezy is the Merchant of Record. OCX is wired up as a Store with
three Variants (Starter, Growth, Scale). LS hosts checkout, holds the
VAT/GST liability, and pays out to OpenKitaab Pvt Ltd monthly.

We only need three API surfaces here:

  • create_checkout(variant_id, email, custom)  →  hosted checkout URL
  • get_subscription(sub_id)                    →  subscription object
  • verify_webhook_signature(body, signature)   →  HMAC-SHA256 check

All other operations (list customers, refunds, plan changes, etc.) go
through the LemonSqueezy dashboard or are out of scope for this
codebase. Subscription cancellation is handled customer-side via the
LS-hosted "Update payment method" portal URL we receive on each
subscription event.
"""
from __future__ import annotations

import hashlib
import hmac
import logging
from typing import Any

import httpx

LS_API_BASE = "https://api.lemonsqueezy.com/v1"
JSON_API = "application/vnd.api+json"

log = logging.getLogger("ocx.lemonsqueezy")


class LemonSqueezyError(Exception):
    pass


def _client(api_key: str) -> httpx.Client:
    return httpx.Client(
        base_url=LS_API_BASE,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Accept": JSON_API,
            "Content-Type": JSON_API,
        },
        timeout=httpx.Timeout(15.0, connect=5.0),
    )


def create_checkout(
    *,
    api_key: str,
    store_id: str,
    variant_id: str,
    email: str | None,
    redirect_url: str,
    cancel_url: str,
    custom: dict[str, Any] | None = None,
) -> tuple[str, str]:
    """Create a LemonSqueezy hosted Checkout. Returns (checkout_url, checkout_id).

    The `custom` payload is echoed back on every webhook event for that
    customer / subscription, so we use it to carry the OCX tier across
    the checkout boundary.
    """
    body = {
        "data": {
            "type": "checkouts",
            "attributes": {
                "checkout_data": {
                    "email": email,
                    "custom": custom or {},
                },
                "checkout_options": {
                    "embed": False,
                    "media": True,
                    "logo": True,
                },
                "product_options": {
                    "redirect_url": redirect_url,
                    "receipt_button_text": "Open my OCX account",
                    "receipt_link_url": redirect_url,
                },
                "expires_at": None,
            },
            "relationships": {
                "store": {"data": {"type": "stores", "id": str(store_id)}},
                "variant": {"data": {"type": "variants", "id": str(variant_id)}},
            },
        }
    }
    with _client(api_key) as c:
        r = c.post("/checkouts", json=body)
        if r.status_code >= 400:
            log.error("LS create_checkout failed: %s %s", r.status_code, r.text[:500])
            raise LemonSqueezyError(f"LS {r.status_code}: {r.text[:200]}")
        d = r.json()["data"]
        return d["attributes"]["url"], d["id"]


def get_subscription(*, api_key: str, subscription_id: str) -> dict[str, Any]:
    with _client(api_key) as c:
        r = c.get(f"/subscriptions/{subscription_id}")
        if r.status_code >= 400:
            raise LemonSqueezyError(f"LS {r.status_code}: {r.text[:200]}")
        return r.json()["data"]


def verify_webhook_signature(*, body: bytes, signature: str | None, secret: str) -> bool:
    """LS sends `X-Signature` as hex of HMAC-SHA256(body, secret).
    Constant-time compare. Empty / missing signature → False."""
    if not signature or not secret:
        return False
    digest = hmac.new(secret.encode("utf-8"), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(digest, signature)
