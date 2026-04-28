import React from "react";
import { Link, useSearchParams } from "react-router-dom";

export default function Welcome() {
  const [params] = useSearchParams();
  // LemonSqueezy redirects with `?order_number=...&order_id=...&...`.
  // We surface whichever id the provider passes for support reference.
  const orderId =
    params.get("order_id") ||
    params.get("order_number") ||
    params.get("checkout") ||
    params.get("subscription") ||
    null;

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-16 sm:pb-24 lg:pb-32">
        <div className="max-w-[60ch]">
          <p className="text-[11px] uppercase tracking-[0.22em] text-stone-500 mb-6 sm:mb-8">
            Subscription confirmed
          </p>
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Check your email.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            Your API key has just been sent to the email address you used at
            checkout. We don&rsquo;t store it — that email is the only place it
            exists in plain text. Save it somewhere safe.
          </p>
        </div>
      </section>

      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-24 lg:py-32">
          <div className="max-w-[80ch]">
            <p className="label mb-6 sm:mb-8">Next steps</p>

            <div className="space-y-10 sm:space-y-12">
              <Step
                n="1"
                title="Find the email"
                body="Subject line is &lsquo;Your OCX API key&rsquo;. From hello@ocx.world. Check spam if you don't see it within a minute."
              />
              <Step
                n="2"
                title="Make a test call"
                body="Use the curl example from the email. Send a real receipt and verify it returns OCX_SUCCESS."
              />
              <Step
                n="3"
                title="Manage your subscription"
                body={
                  <>
                    Update your payment method, change tier, or cancel anytime
                    from <Link to="/account" className="link">your account</Link>.
                  </>
                }
              />
            </div>

            {orderId && (
              <p className="mt-12 sm:mt-16 text-[12px] mono text-stone-500 break-all">
                order_ref: {orderId}
              </p>
            )}
          </div>
        </div>
      </section>

      <section className="container-wide py-24 sm:py-32 lg:py-40">
        <div className="max-w-[40ch]">
          <h2 className="text-display-md font-medium tracking-tight text-ink">
            Lost your API key?
          </h2>
          <p className="mt-5 sm:mt-6 text-stone-600 leading-relaxed">
            Email{" "}
            <a href="mailto:hhaishwary@gmail.com" className="link break-all">
              hhaishwary@gmail.com
            </a>{" "}
            from the address you subscribed with. We&apos;ll rotate the key and send the new one.
          </p>
        </div>
      </section>
    </>
  );
}

function Step({ n, title, body }) {
  return (
    <article className="lg:grid lg:grid-cols-12 lg:gap-8">
      <div className="flex items-baseline gap-4 lg:contents">
        <span className="lg:col-span-1 mono text-stone-400 text-[14px] lg:pt-1">
          {String(n).padStart(2, "0")}
        </span>
        <h3 className="lg:col-span-3 font-medium text-ink text-[16px] sm:text-[17px]">
          {title}
        </h3>
      </div>
      <div className="mt-3 lg:mt-0 lg:col-span-8 text-stone-600 leading-[1.7] text-[15px]">
        {body}
      </div>
    </article>
  );
}
