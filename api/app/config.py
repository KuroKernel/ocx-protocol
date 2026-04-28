"""Settings — read from environment, validated by Pydantic."""
from __future__ import annotations

from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    # LemonSqueezy — Merchant of Record. All fields optional so the app
    # can boot before LS is wired up; the relevant routes return 503
    # ("provider not configured") until every required field is set.
    ls_api_key: str | None = Field(default=None, description="LemonSqueezy API key (Bearer token).")
    ls_store_id: str | None = Field(default=None, description="LemonSqueezy Store ID (numeric).")
    ls_webhook_secret: str | None = Field(default=None, description="Webhook signing secret. Webhook endpoint 503s until set.")
    ls_variant_starter: str | None = Field(default=None, description="Variant ID for the OCX Starter monthly plan.")
    ls_variant_growth: str | None = Field(default=None, description="Variant ID for the OCX Growth monthly plan.")
    ls_variant_scale: str | None = Field(default=None, description="Variant ID for the OCX Scale monthly plan.")

    @property
    def ls_configured(self) -> bool:
        """True iff every LS field needed for /v1/checkout is set.
        The webhook handler additionally requires ls_webhook_secret."""
        return bool(
            self.ls_api_key and self.ls_store_id and
            self.ls_variant_starter and self.ls_variant_growth and self.ls_variant_scale
        )

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
        return f"{self.app_url.rstrip('/')}{self.success_path}"

    @property
    def cancel_url(self) -> str:
        return f"{self.app_url.rstrip('/')}{self.cancel_path}"

    def variant_id_for_tier(self, tier: str) -> str | None:
        return {
            "starter": self.ls_variant_starter,
            "growth": self.ls_variant_growth,
            "scale": self.ls_variant_scale,
        }.get(tier)

    def tier_for_variant_id(self, variant_id: str) -> str | None:
        return {
            self.ls_variant_starter: "starter",
            self.ls_variant_growth: "growth",
            self.ls_variant_scale: "scale",
        }.get(variant_id)


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
