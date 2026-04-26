"""Email sending. Postmark by default; falls back to stdout logging if
POSTMARK_TOKEN is not set."""
from __future__ import annotations

import logging
from typing import Any

import httpx

from .config import get_settings

log = logging.getLogger("ocx.email")


def send_api_key_email(*, to: str, api_key: str, tier: str) -> None:
    """Sends the post-checkout welcome email with the new API key."""
    s = get_settings()
    subject = "Your OCX API key"
    text_body = f"""\
Welcome to OCX.

Your API key — keep this somewhere safe. We don't store it; this is the
only time you'll see it.

  {api_key}

Tier: {tier}

Use it on every request as:

  Authorization: Bearer {api_key}

Quick test:

  curl https://api.ocx.world/v1/verify \\
    -H "Authorization: Bearer {api_key}" \\
    -H "Content-Type: application/json" \\
    -d '{{"cbor_hex": "<paste a receipt>", "public_key_hex": "<32 bytes>"}}'

Manage your billing, change tier, or download invoices anytime:
https://ocx.world/account

— Aishwary, OCX Protocol
hhaishwary@gmail.com
"""
    _send(to=to, subject=subject, text_body=text_body)


def _send(*, to: str, subject: str, text_body: str) -> None:
    s = get_settings()
    if not s.postmark_token:
        # Dev / unconfigured: log to stdout instead of sending.
        log.warning("POSTMARK_TOKEN unset — would send email:\nTo: %s\nSubject: %s\n\n%s",
                    to, subject, text_body)
        return
    payload: dict[str, Any] = {
        "From": s.postmark_from_email,
        "To": to,
        "Subject": subject,
        "TextBody": text_body,
        "MessageStream": "outbound",
    }
    headers = {
        "X-Postmark-Server-Token": s.postmark_token,
        "Accept": "application/json",
        "Content-Type": "application/json",
    }
    try:
        r = httpx.post("https://api.postmarkapp.com/email", json=payload, headers=headers, timeout=10.0)
        r.raise_for_status()
        log.info("Email sent to %s (Postmark id=%s)", to, r.json().get("MessageID"))
    except Exception as e:
        log.exception("Postmark send failed for %s: %s", to, e)
        # Re-raise so webhook handler can decide whether to retry
        raise
