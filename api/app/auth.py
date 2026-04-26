"""API key generation and bearer-token authentication.

Plaintext key format:   ocx_live_<43 char base32>
DB-stored as:           SHA-256 hex digest of the plaintext
Display prefix:         first 12 chars of the plaintext, used in dashboards/emails

The plaintext is shown to the customer ONCE in the post-checkout email.
After that we never see it again — only the hash is in the DB.
"""
from __future__ import annotations

import hashlib
import secrets
from typing import Annotated

from fastapi import Depends, Header, HTTPException, status
from sqlalchemy import select
from sqlalchemy.orm import Session

from .db import get_db
from .models import Customer

KEY_PREFIX = "ocx_live_"
RANDOM_BYTES = 32  # → 43 base32-encoded chars after stripping padding


def generate_api_key() -> tuple[str, str, str]:
    """Returns (plaintext, sha256_hex, display_prefix)."""
    raw = secrets.token_urlsafe(RANDOM_BYTES)  # url-safe base64
    plaintext = f"{KEY_PREFIX}{raw}"
    digest = hashlib.sha256(plaintext.encode("utf-8")).hexdigest()
    display_prefix = plaintext[:16]  # "ocx_live_xxxxxxx"
    return plaintext, digest, display_prefix


def hash_key(plaintext: str) -> str:
    return hashlib.sha256(plaintext.encode("utf-8")).hexdigest()


def get_current_customer(
    authorization: Annotated[str | None, Header()] = None,
    db: Session = Depends(get_db),
) -> Customer:
    """FastAPI dependency. Looks up the customer from the bearer token."""
    if not authorization:
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Missing Authorization header")
    parts = authorization.strip().split(" ", 1)
    if len(parts) != 2 or parts[0].lower() != "bearer":
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Authorization must be 'Bearer ocx_live_...'")
    token = parts[1]
    if not token.startswith(KEY_PREFIX):
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Invalid API key format")
    digest = hash_key(token)
    customer = db.execute(
        select(Customer).where(Customer.api_key_hash == digest)
    ).scalar_one_or_none()
    if customer is None:
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "API key not found")
    return customer
