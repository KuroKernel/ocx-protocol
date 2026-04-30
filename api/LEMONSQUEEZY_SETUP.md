# LemonSqueezy onboarding — pause until OpenKitaab Pvt Ltd bank account is wired

**Blocked on:** OpenKitaab Pvt Ltd bank account setup. ~10-day window.
**State of the code:** fully shipped. Backend, schema, webhook handler,
frontend pricing, copy — all wired up for LS. Health checks pass; the
checkout endpoint returns a clean 503 ("Checkout is not yet configured.
Email hello@kitaab.live.") until the env vars below are set on Railway.

When the bank account is ready, do these six steps in order. About 30
minutes end-to-end.

## 1. Sign up

https://www.lemonsqueezy.com → Sign up.

Seller of record: **OpenKitaab Private Limited**, India.

LS holds VAT/GST/sales-tax liability globally. They sell to the customer.
You sell to LS. They pay out monthly to your Indian bank in INR or USD.

## 2. Create three Products

Each product → one variant → Subscription billing → Monthly → USD.

| Product name | Price/month | Notes |
|---|---|---|
| OCX Starter | **$999** | 100,000 receipts/month |
| OCX Growth | **$4,999** | 1,000,000 receipts/month |
| OCX Scale  | **$9,999** | 10,000,000 receipts/month |

Add a short description on each. Product images optional.

Note the **Variant ID** on each (numeric, in the Variants tab URL).

## 3. Create the API key

Settings → API → "Create API Key". One-time-shown Bearer token. Copy.

## 4. Register the production webhook

Settings → Webhooks → New webhook.

- **URL:** `https://ocx-api-production.up.railway.app/v1/webhooks/lemonsqueezy`
  (or `https://api.ocx.world/v1/webhooks/lemonsqueezy` if the CNAME is in by then)
- **Events** (all of these):
  - subscription_created
  - subscription_updated
  - subscription_cancelled
  - subscription_resumed
  - subscription_expired
  - subscription_payment_success
  - subscription_payment_failed
- Copy the signing secret. Shown ONCE.

## 5. Set env vars on Railway → ocx-api

```
LS_API_KEY          = <Bearer token from step 3>
LS_STORE_ID         = <numeric, top of any Products page URL>
LS_WEBHOOK_SECRET   = <from step 4>
LS_VARIANT_STARTER  = <numeric, Starter Variants tab>
LS_VARIANT_GROWTH   = <numeric, Growth Variants tab>
LS_VARIANT_SCALE    = <numeric, Scale Variants tab>
```

No redeploy needed — `get_settings()` re-reads env on demand. The 503
on `/v1/checkout/create-session` will flip to a real LS checkout URL
the next time someone clicks Subscribe.

## 6. Smoke test

LS test mode → buy your own Starter subscription with a 4242 test card.
Confirm:

- redirect to ocx.world/welcome with `?order_id=...`
- welcome email arrives at the email used at checkout
- ocx.world/account → paste key → tier `starter`, usage 0/100,000
- LS dashboard shows the test subscription
- Webhook delivery log on the LS side shows 200s

## What is NOT blocked by this

- The whitepaper, the GPU verifier work, MI300X results, vLLM evidence
- The redesigned site, /verify/<hash> redirect, all marketing copy
- Enterprise sales via "Talk to sales" + wire transfer (no LS in path)
- Kitaab integration (Kitaab still calls ocx-server-production directly)
- The receipt-as-URL flow (`/r/<id>` page with WASM verifier) when ready

Resume here when the bank account lands.
