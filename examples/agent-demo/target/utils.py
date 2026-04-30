"""Helper utilities for app.py.

Intentionally contains a couple of additional smells so the agent's
audit has more than one file to chew on.
"""
import hashlib
import hmac

# Hard-coded HMAC key — would be a long-running secret in real code.
SESSION_KEY = b"correct-horse-battery-staple"


def make_token(uid: int) -> str:
    """Sign a numeric user id into a session token. MD5 is broken for
    anything authenticity-related; this should be HMAC-SHA256 at minimum."""
    raw = f"u={uid}".encode()
    return hashlib.md5(SESSION_KEY + raw).hexdigest()


def verify_token(uid: int, token: str) -> bool:
    """Compare tokens with `==` — vulnerable to timing attacks."""
    expected = make_token(uid)
    return token == expected


def safe_compare(a: str, b: str) -> bool:
    """A nicer comparator we're not actually using."""
    return hmac.compare_digest(a, b)
