"""FastAPI entry point. Wires routes, CORS, and basic health checks."""
from __future__ import annotations

import logging
import os

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .config import get_settings
from .routes import checkout, portal, verify, webhooks

s = get_settings()
logging.basicConfig(level=getattr(logging, s.log_level.upper(), logging.INFO))
log = logging.getLogger("ocx.boot")


def _maybe_init_db() -> None:
    """If OCX_INIT_DB=1 is set, apply schema.sql before serving traffic.
    Idempotent (CREATE TABLE IF NOT EXISTS). Set the env var, redeploy,
    confirm the log line, then unset and redeploy again."""
    if os.environ.get("OCX_INIT_DB") != "1":
        return
    from pathlib import Path
    import psycopg
    schema_paths = [
        Path("/opt/ocx/schema.sql"),
        Path(__file__).resolve().parent.parent / "schema.sql",
    ]
    schema_path = next((p for p in schema_paths if p.exists()), None)
    if schema_path is None:
        log.error("OCX_INIT_DB=1 but schema.sql not found in %s", schema_paths)
        return
    sql = schema_path.read_text()
    # Prefer DATABASE_PUBLIC_URL for the one-shot init — Railway's private
    # IPv6 network sometimes isn't reachable from a fresh container before
    # routing tables settle. Public URL works either way; we only use it
    # for this idempotent CREATE TABLE pass, not steady-state traffic.
    init_url = os.environ.get("DATABASE_PUBLIC_URL") or s.database_url
    log.warning("OCX_INIT_DB=1 → applying %s (%d bytes) via %s",
                schema_path, len(sql), "public" if init_url is not s.database_url else "private")
    try:
        with psycopg.connect(init_url, autocommit=True, connect_timeout=20) as conn:
            with conn.cursor() as cur:
                cur.execute(sql)
                cur.execute(
                    "SELECT tablename FROM pg_tables "
                    "WHERE schemaname = current_schema() ORDER BY tablename"
                )
                tables = [r[0] for r in cur.fetchall()]
        log.warning("OCX_INIT_DB done. tables: %s", tables)
    except Exception:
        log.exception("OCX_INIT_DB failed")


app = FastAPI(
    title="OCX Protocol API",
    description="Verified inference. Cryptographic receipts for AI inference.",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)

# CORS — allow the frontend (ocx.world) to POST to /v1/checkout and /v1/billing/portal.
# Webhooks come straight from Stripe (no browser), so don't need CORS.
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        s.app_url,
        "http://localhost:3000",
        "http://localhost:5173",
    ],
    allow_credentials=False,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["Authorization", "Content-Type"],
    max_age=3600,
)

app.include_router(checkout.router, tags=["billing"])
app.include_router(portal.router, tags=["billing"])
app.include_router(webhooks.router, tags=["webhooks"])
app.include_router(verify.router, tags=["verify"])


@app.on_event("startup")
def _on_startup():
    # Run AFTER the app is up and serving — so Railway healthcheck never
    # blocks on Postgres reachability. If init fails, we log and keep
    # serving (routes that need the DB will just 500 until re-run).
    _maybe_init_db()


@app.get("/health", tags=["meta"])
def health():
    """Liveness probe — always 200 if the process is up."""
    return {"status": "ok", "service": "ocx-api"}


@app.get("/", tags=["meta"])
def root():
    return {
        "service": "ocx-api",
        "docs": "/docs",
        "endpoints": [
            "POST /v1/checkout/create-session",
            "POST /v1/billing/portal",
            "POST /v1/webhooks/stripe",
            "POST /v1/verify",
            "GET  /v1/account",
        ],
    }
