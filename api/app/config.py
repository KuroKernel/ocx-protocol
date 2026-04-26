"""Settings — read from environment, validated by Pydantic."""
from __future__ import annotations

from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    # Stripe
    stripe_secret_key: str = Field(..., description="sk_test_... or sk_live_...")
    stripe_webhook_secret: str | None = Field(default=None, description="whsec_... from the endpoint config; webhooks 503 until set")
    stripe_price_starter: str = Field(...)
    stripe_price_growth: str = Field(...)
    stripe_price_scale: str = Field(...)

    # App
    app_url: str = "https://ocx.world"
    success_path: str = "/welcome"
    cancel_path: str = "/pricing"

    # Database
    database_url: str = "postgresql+psycopg://postgres:dev@localhost:5432/ocx"

    # Email (optional — log to stdout if Postmark not configured)
    postmark_token: str | None = None
    postmark_from_email: str = "hello@ocx.world"

    # Verifier (optional)
    ocx_verify_lib: str | None = None

    # Misc
    log_level: str = "info"
    port: int = 8000

    @property
    def success_url(self) -> str:
        return f"{self.app_url.rstrip('/')}{self.success_path}?session_id={{CHECKOUT_SESSION_ID}}"

    @property
    def cancel_url(self) -> str:
        return f"{self.app_url.rstrip('/')}{self.cancel_path}"

    def price_id_for_tier(self, tier: str) -> str | None:
        return {
            "starter": self.stripe_price_starter,
            "growth": self.stripe_price_growth,
            "scale": self.stripe_price_scale,
        }.get(tier)

    def tier_for_price_id(self, price_id: str) -> str | None:
        return {
            self.stripe_price_starter: "starter",
            self.stripe_price_growth: "growth",
            self.stripe_price_scale: "scale",
        }.get(price_id)


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
