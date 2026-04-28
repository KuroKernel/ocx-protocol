# OCX API — billing + verification backend

Python / FastAPI service. Wires up:

- **LemonSqueezy** as Merchant of Record. LS sells, holds VAT/GST liability,
  pays out monthly to OpenKitaab Pvt Ltd.
- API-key issuance after a successful subscription event (welcome email
  is the only place the plaintext key exists).
- `POST /v1/verify` — gated by API key + tier limits, calls `libocx-verify`
  via C FFI.
- `GET  /v1/account` — current tier + usage.
- `POST /v1/billing/portal` — returns the LS-hosted customer portal link.
- Per-period usage tracking for tier limits.

Total: ~900 lines of Python plus a Postgres schema and a Dockerfile.
Deploys to Railway as one service.

---

## Files

```
api/
├── README.md             ← this file
├── pyproject.toml        ← Python deps
├── .env.example          ← env vars template
├── Dockerfile            ← multi-stage (Rust verifier + Python app)
├── railway.json          ← Railway service config
├── schema.sql            ← Postgres schema (run once on a fresh DB)
└── app/
    ├── main.py           ← FastAPI entry, route registration, CORS
    ├── config.py         ← settings via env
    ├── db.py             ← SQLAlchemy + Postgres session
    ├── models.py         ← Customer, Subscription, Usage, ProviderEvent
    ├── auth.py           ← API key generation + bearer auth dependency
    ├── tiers.py          ← tier definitions + monthly receipt limits
    ├── email.py          ← Postmark sender (stdout fallback)
    ├── ffi_verify.py     ← ctypes binding for libocx-verify
    ├── lemonsqueezy.py   ← LS API client + signature verification
    └── routes/
        ├── checkout.py   ← POST /v1/checkout/create-session
        ├── portal.py     ← POST /v1/billing/portal
        ├── webhooks.py   ← POST /v1/webhooks/lemonsqueezy
        └── verify.py     ← POST /v1/verify   +   GET /v1/account
```

---

## Local dev

```bash
# 1. Postgres
docker run -d --name ocx-pg -e POSTGRES_PASSWORD=dev -p 5432:5432 postgres:16

# 2. Schema
psql -h localhost -U postgres -d postgres -c 'CREATE DATABASE ocx;'
psql -h localhost -U postgres -d ocx -f schema.sql

# 3. Env
cp .env.example .env  # fill in LS_*

# 4. Deps + run
pip install -e .
uvicorn app.main:app --reload --port 8000
```

For local webhook testing, use the [LemonSqueezy CLI](https://docs.lemonsqueezy.com/help/webhooks)
or `ngrok` pointed at `http://localhost:8000/v1/webhooks/lemonsqueezy`.

---

## Provider design

`models.py` uses `provider` + `provider_customer_id` + `provider_subscription_id`
columns rather than vendor-named fields. The webhook handler keys on
`provider="lemonsqueezy"` everywhere. Swapping to Razorpay or Paddle is
a matter of writing a new `lemonsqueezy.py`-equivalent + updating the
webhook router. Customer / Subscription / Usage rows survive unchanged.

---

## Deployment

Railway service `ocx-api`:

```bash
railway up --service ocx-api
```

Env vars live in Railway dashboard. Postgres is a sibling service in
the same project; `DATABASE_URL` is injected automatically.
