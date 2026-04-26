# OCX API — Stripe billing + verification backend

Python / FastAPI service that:

- creates Stripe Checkout sessions when a customer clicks "Subscribe"
- handles Stripe webhooks (subscription created, updated, canceled, invoice paid/failed)
- generates and stores hashed API keys; emails the plaintext to the customer
- exposes `POST /v1/verify` — the actual product, gated by API key + tier limits
- exposes `POST /v1/billing/portal` — Stripe-hosted self-serve billing
- tracks per-period usage so we can enforce `monthly_receipts` per tier

Total: ~900 lines of Python plus a Postgres schema and a Dockerfile. Deploys to Railway as one service.

---

## Files

```
api/
├── README.md             ← this file
├── pyproject.toml        ← Python deps
├── .env.example          ← env vars template
├── Dockerfile            ← multi-stage build (Rust verifier + Python app)
├── railway.json          ← Railway service config
├── schema.sql            ← Postgres schema (run once on a fresh DB)
└── app/
    ├── main.py           ← FastAPI entry, route registration, CORS
    ├── config.py         ← settings via env
    ├── db.py             ← SQLAlchemy + Postgres session
    ├── models.py         ← Customer, Subscription, Usage, StripeEvent
    ├── auth.py           ← API key generation, bearer auth dependency
    ├── tiers.py          ← tier definitions + monthly receipt limits
    ├── email.py          ← Postmark sender (logs to stdout if unconfigured)
    ├── ffi_verify.py     ← ctypes binding for libocx-verify
    └── routes/
        ├── checkout.py   ← POST /v1/checkout/create-session
        ├── portal.py     ← POST /v1/billing/portal
        ├── webhooks.py   ← POST /v1/webhooks/stripe (the load-bearing piece)
        └── verify.py     ← POST /v1/verify   +   GET /v1/account
```

---

## End-to-end setup (first deployment)

This sequence takes about 30 minutes.

### 1. Set up Stripe Products + Prices

In the Stripe Dashboard (https://dashboard.stripe.com/test/products):

For each tier, create a **Product** with one **recurring monthly Price** in USD:

| Product name | Description | Price |
|---|---|---|
| `OCX Starter` | Up to 100,000 receipts / month | **$499 USD / month** |
| `OCX Growth` | Up to 1,000,000 receipts / month | **$1,499 USD / month** |
| `OCX Scale` | Up to 10,000,000 receipts / month | **$4,999 USD / month** |

After creating each Price, copy its **Price ID** (looks like `price_1Q...`). You'll need three of them.

### 2. Set up the Stripe webhook endpoint

In the Dashboard → Developers → Webhooks → "Add endpoint":

- **Endpoint URL**: `https://api.ocx.world/v1/webhooks/stripe` (your future production URL)
- **Events to send**: select these six exactly:
  - `checkout.session.completed`
  - `customer.subscription.created`
  - `customer.subscription.updated`
  - `customer.subscription.deleted`
  - `invoice.paid`
  - `invoice.payment_failed`

After creating, click "Reveal" on the **Signing Secret** — copy it. Looks like `whsec_...`.

You can do this AFTER deploying; Stripe lets you create a placeholder endpoint and update the URL later.

### 3. Set up Railway

```bash
# Install Railway CLI (one time)
npm i -g @railway/cli

# From the repo root:
cd /path/to/ocx-protocol
railway login
railway init                # create a new project
railway add --plugin postgresql   # provisions Postgres + sets DATABASE_URL
```

In the Railway dashboard for this project:

1. **Settings → Service → Source**: point at this GitHub repo, `Dockerfile path = api/Dockerfile`, `root directory = /` (the Dockerfile uses `api/...` paths).
2. **Variables**: add the env vars from `.env.example` with real values:
   - `STRIPE_SECRET_KEY` — from Dashboard → Developers → API keys (start with test mode `sk_test_...`)
   - `STRIPE_WEBHOOK_SECRET` — from step 2 above
   - `STRIPE_PRICE_STARTER`, `STRIPE_PRICE_GROWTH`, `STRIPE_PRICE_SCALE` — from step 1
   - `APP_URL=https://ocx.world`
   - `POSTMARK_TOKEN` — optional; leave blank to log emails instead of sending
   - `POSTMARK_FROM_EMAIL=hello@ocx.world` — must be a verified sender in Postmark
3. **Settings → Networking**: enable a public domain. Map it to `api.ocx.world` via your DNS provider (CNAME → Railway-provided host).

### 4. Initialize the database

After the first deploy, run the schema once:

```bash
railway run psql $DATABASE_URL -f api/schema.sql
```

(Or paste the contents of `api/schema.sql` into Railway's Postgres data console.)

### 5. Update the Stripe webhook endpoint URL

Now that the service is live, edit the webhook in the Stripe Dashboard and change the URL to:

```
https://api.ocx.world/v1/webhooks/stripe
```

Stripe will start delivering events.

### 6. Verify end-to-end

From the website (`ocx.world/pricing`), click any "Subscribe" button. You should be redirected to Stripe Checkout. Use a Stripe test card:

```
4242 4242 4242 4242   any future date   any 3-digit CVC   any postal
```

After successful payment:

- Browser redirects to `ocx.world/welcome?session_id=cs_test_...`
- Stripe sends `checkout.session.completed` to your webhook
- Webhook handler creates a customer record + generates an API key
- If `POSTMARK_TOKEN` is set: customer receives the welcome email with key
- If not: check `railway logs` — the email is printed there

Test the API key:

```bash
curl https://api.ocx.world/v1/account \
  -H "Authorization: Bearer ocx_live_<key from email>"
```

Expected: `{"email": "...", "tier": "starter", "monthly_limit": 100000, ...}`

---

## Local development

```bash
# Postgres (via docker)
docker run -d --name ocx-pg -e POSTGRES_PASSWORD=dev -p 5432:5432 postgres:16
docker exec ocx-pg psql -U postgres -c "CREATE DATABASE ocx;"
docker exec -i ocx-pg psql -U postgres -d ocx < api/schema.sql

# Build the verifier (one time)
cd libocx-verify && cargo build --release && cd ..

# Python deps
cd api
pip install -e .
cp .env.example .env       # then edit .env with your test Stripe keys

# Stripe CLI for local webhook forwarding (one time)
brew install stripe/stripe-cli/stripe   # or download from https://stripe.com/docs/stripe-cli
stripe login

# Run the API
export OCX_VERIFY_LIB=$PWD/../libocx-verify/target/release/liblibocx_verify.so
uvicorn app.main:app --reload --port 8000

# In a second terminal — forward Stripe events to localhost:
stripe listen --forward-to localhost:8000/v1/webhooks/stripe
# Copy the whsec_... it prints into your .env as STRIPE_WEBHOOK_SECRET, restart uvicorn.

# In a third terminal — frontend pointing at the local API:
cd ..
REACT_APP_API_BASE=http://localhost:8000 npm start
```

Now click "Subscribe" on `localhost:3000/pricing`. You'll be sent to Stripe-hosted checkout, pay with the test card, and the local webhook gets the event. The API key gets logged to your uvicorn console (since `POSTMARK_TOKEN` is unset).

---

## API surface

### Public — no auth

`POST /v1/checkout/create-session`
```json
{ "tier": "starter" }
→ 200
{ "url": "https://checkout.stripe.com/c/pay/cs_test_...", "session_id": "cs_test_..." }
```

`POST /v1/webhooks/stripe`
- Receives Stripe events. Verifies signature against `STRIPE_WEBHOOK_SECRET`.
- Idempotent — replays return 200 without re-processing.

### Authenticated — `Authorization: Bearer ocx_live_...`

`POST /v1/billing/portal`
```json
→ 200
{ "url": "https://billing.stripe.com/p/session/..." }
```

`GET /v1/account`
```json
→ 200
{
  "email": "alice@example.com",
  "tier": "starter",
  "api_key_prefix": "ocx_live_xyz123",
  "monthly_limit": 100000,
  "usage": {
    "receipts_verified": 42,
    "period_start": "2026-04-01T00:00:00+00:00",
    "period_end": "2026-05-01T00:00:00+00:00"
  }
}
```

`POST /v1/verify`
```json
{
  "cbor_hex": "a801582 ...",
  "public_key_hex": "015e10ec ..."
}
→ 200
{
  "verified": true,
  "error_code": 0,
  "error_name": "OCX_SUCCESS",
  "elapsed_us": 78.4,
  "tier": "starter",
  "usage": { "receipts_verified": 43, "monthly_limit": 100000, ... }
}
```
- Returns `429 Too Many Requests` if monthly tier limit exceeded.
- Returns `402 Payment Required` if subscription canceled.
- Returns `503 Service Unavailable` if libocx-verify is not loaded.

---

## Reliability notes for the operator

**Webhook idempotency.** Stripe retries every event for ~3 days on 5xx responses. Our handler records every `event.id` in the `stripe_events` table BEFORE returning 200. A retry of the same event ID is a no-op. If the handler crashes mid-execution, the DB rolls back the marker too, so Stripe's next retry re-runs cleanly.

**Email failure isolation.** If Postmark is down when `checkout.session.completed` fires, the customer record is still created (so they're not charged without an account) but the email isn't sent. The webhook returns 5xx → Stripe retries. On the retry, the customer already exists → idempotency skips → 200 returned. The customer can email support to get the API key reset.

**API key rotation.** No automatic UI yet. To rotate manually:

```sql
-- Generate a new key (run from the API service, not psql):
python -c "from app.auth import generate_api_key; p,h,pre = generate_api_key(); print(p); print(h); print(pre)"

-- Then update the customer:
UPDATE customers
SET api_key_hash = '<the printed hash>',
    api_key_prefix = '<the printed prefix>',
    updated_at = now()
WHERE email = 'alice@example.com';
```

Send the plaintext key (the first printed line) to the customer.

**Tier downgrade after cancellation.** When `customer.subscription.deleted` fires, we set `customer.tier = 'canceled'`. The next `/v1/verify` call returns `402 Payment Required`. If they re-subscribe, `checkout.session.completed` updates `tier` back.

**Per-period usage reset.** Usage rows are keyed by `(customer_id, period_start)`. When a subscription's `current_period_start` advances (Stripe sends `customer.subscription.updated` at every billing cycle), the next `/v1/verify` call creates a new usage row with the new period — old usage is preserved for analytics.

---

## What's NOT in this service yet

These are intentionally deferred to keep the surface small:

- **Self-serve API key rotation UI.** Customers can request rotation via email; we run the SQL above. UI later.
- **Webhook retries from our side.** If Postmark fails, we log it and return 5xx; Stripe handles the retry. We don't have an internal retry queue.
- **Per-API-key short-window rate limits.** Only monthly limits are enforced. If a customer floods us with requests, the monthly budget burns through quickly. Per-second/per-minute limits would need Redis or similar.
- **Multi-key per customer.** One key per customer right now. Multi-key (separate prod/staging/dev keys) is a future feature.
- **Detailed audit log.** `usage` aggregates counters per period; we don't keep an entry per individual `/v1/verify` call. Adding that is straightforward but storage-expensive.

---

## Cost expectations

For the first 100 paying customers:

- Railway: ~$10-30/month (free Postgres allowance + small service)
- Postmark: free for first 100 emails/month, ~$15/month for 10K emails
- Stripe: 2.9% + 30¢ per successful charge
- Domain (api.ocx.world): existing
- Total operating cost at 10 paying customers: < $50/month
- At 100 paying customers: < $200/month
