"""Tier definitions and per-tier limits.

Limits are checked in /v1/verify after looking up the customer's tier.
Keep this in lockstep with the tier descriptions on the website pricing page.
"""
from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Tier:
    name: str
    monthly_receipts: int
    sla: str
    description: str


TIERS: dict[str, Tier] = {
    "starter": Tier(
        name="Starter",
        monthly_receipts=100_000,
        sla="99.5%",
        description="Up to 100,000 receipts / month",
    ),
    "growth": Tier(
        name="Growth",
        monthly_receipts=1_000_000,
        sla="99.9%",
        description="Up to 1,000,000 receipts / month",
    ),
    "scale": Tier(
        name="Scale",
        monthly_receipts=10_000_000,
        sla="99.95%",
        description="Up to 10,000,000 receipts / month",
    ),
    "enterprise": Tier(
        name="Enterprise",
        monthly_receipts=10**12,  # effectively unlimited
        sla="99.99%",
        description="Custom limits",
    ),
}


def get_tier(name: str) -> Tier | None:
    return TIERS.get(name)
