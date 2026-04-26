import React, { useState } from "react";

// Backend URL — defaults to "/api" which Netlify rewrites to the Railway
// service (see netlify.toml). Override via REACT_APP_API_BASE for local dev.
const API_BASE = process.env.REACT_APP_API_BASE || "/api";

const tiers = [
  {
    key: "free",
    name: "Open source",
    price: "Free",
    cadence: "MIT-licensed",
    sub: "Self-hosted protocol & SDKs",
    features: [
      "Canonical CBOR receipt schema",
      "Rust verifier (libocx-verify) with C FFI",
      "Go, Python, Rust SDKs",
      "Self-hosted, no limits",
      "Community support",
    ],
    cta: { kind: "link", label: "Read the spec", href: "/spec" },
  },
  {
    key: "starter",
    name: "Starter",
    price: "$499",
    cadence: "/month",
    sub: "For teams shipping their first verified product",
    features: [
      "100,000 receipts / month",
      "Hosted verifier API",
      "99.5% uptime SLA",
      "Email support · 48h response",
      "Audit logs · 30-day retention",
    ],
    cta: { kind: "checkout", tier: "starter", label: "Subscribe — $499/mo" },
  },
  {
    key: "growth",
    name: "Growth",
    price: "$1,499",
    cadence: "/month",
    sub: "For production AI deployments at meaningful scale",
    features: [
      "1,000,000 receipts / month",
      "Hosted verifier API",
      "99.9% uptime SLA",
      "Priority support · 8h response",
      "Audit logs · 90-day retention",
      "Custom issuer keys",
    ],
    cta: { kind: "checkout", tier: "growth", label: "Subscribe — $1,499/mo" },
    highlight: true,
  },
  {
    key: "scale",
    name: "Scale",
    price: "$4,999",
    cadence: "/month",
    sub: "For high-throughput inference services",
    features: [
      "10,000,000 receipts / month",
      "Hosted verifier API",
      "99.95% uptime SLA",
      "Dedicated support engineer",
      "Audit logs · 1-year retention",
      "Custom integrations",
      "Quarterly architecture reviews",
    ],
    cta: { kind: "checkout", tier: "scale", label: "Subscribe — $4,999/mo" },
  },
];

export default function Pricing() {
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const onSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    const formData = new FormData(e.target);
    const payload = Object.fromEntries(formData.entries());
    try {
      const subject = encodeURIComponent(`OCX enterprise inquiry — ${payload.company}`);
      const body = encodeURIComponent(
        `Name: ${payload.name}\nCompany: ${payload.company}\nRole: ${payload.role}\nEmail: ${payload.email}\n\nUse case:\n${payload.usecase}\n\nVolume estimate: ${payload.volume}\n`
      );
      window.location.href = `mailto:hhaishwary@gmail.com?subject=${subject}&body=${body}`;
      setSubmitted(true);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-12 sm:pb-16 lg:pb-24">
        <div className="max-w-[60ch]">
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Open protocol. Commercial trust&nbsp;layer.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            The protocol is MIT-licensed and free forever. Anyone can run it.
            The hosted verifier and the canonical key registry are commercial
            products for organisations that prefer not to operate their own.
          </p>
        </div>
      </section>

      {/* Four-tier grid: Free + Starter + Growth + Scale.
          Mobile: single column with the highlight tier visually pulled
          forward via its border weight. Tablet: 2×2 grid. Desktop: 4 across. */}
      <section className="container-wide pb-10 sm:pb-12 lg:pb-16">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 sm:gap-6">
          {tiers.map((t) => (
            <Tier key={t.key} {...t} />
          ))}
        </div>
      </section>

      {/* Enterprise — full-width dominant dark card.
          Mobile: tighter padding, stacked grid. */}
      <section className="container-wide pb-20 sm:pb-32 lg:pb-40">
        <div className="bg-ink text-paper p-8 sm:p-12 lg:p-16">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-16">
            <div className="lg:col-span-5">
              <p className="text-[11px] font-medium text-stone-400 uppercase tracking-[0.22em] mb-6 sm:mb-8">
                Enterprise
              </p>
              <div className="text-display-md font-medium tracking-tight text-paper">
                Custom
              </div>
              <p className="mt-5 sm:mt-6 text-stone-300 leading-[1.7] max-w-[40ch]">
                For AI deployments in production. Volume pricing, dedicated
                infrastructure, and the trust controls procurement teams need
                to sign.
              </p>
              <a
                href="#contact"
                className="mt-8 sm:mt-12 inline-flex items-center justify-center gap-2 px-7 py-4 text-[15px] font-medium border border-paper bg-paper text-ink hover:bg-stone-200 transition-colors duration-150 w-full xs:w-auto"
                style={{ minHeight: "48px", touchAction: "manipulation" }}
              >
                Contact sales →
              </a>
            </div>
            <div className="lg:col-span-7">
              <ul className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 sm:gap-x-10 gap-y-4 sm:gap-y-5">
                {[
                  "Unlimited receipts",
                  "On-premises deployment option",
                  "Compliance documentation pack",
                  "Named support engineer",
                  "Custom protocol extensions",
                  "Private key registry",
                  "24/7 incident response",
                  "Custom MSA + DPA",
                ].map((f) => (
                  <li key={f} className="text-[14px] sm:text-[15px] text-stone-200 flex items-baseline gap-3">
                    <span className="text-stone-500 mono text-[10px]">→</span>
                    {f}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Contact section — generic framing, no industry-specific copy */}
      <section id="contact" className="bg-ash" style={{ scrollMarginTop: "5rem" }}>
        <div className="container-wide py-20 sm:py-32 lg:py-48">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-24">
            <div className="lg:col-span-5">
              <p className="label mb-6 sm:mb-8">Talk to us</p>
              <h2 className="text-display-md font-medium tracking-tight text-ink leading-[1.1]">
                Tell us what you&rsquo;re shipping.
              </h2>
              <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
                Whether you&rsquo;re looking at the Starter tier or a custom
                Enterprise deployment, the fastest path is a short message
                describing what you want to make verifiable. We reply within
                24 hours, IST business days.
              </p>
              <p className="mt-5 sm:mt-6 text-stone-500 text-[14px] leading-[1.7]">
                Currently taking design-partner conversations on the Enterprise
                tier. First five pilots receive founder-level direct integration
                support and co-development on protocol extensions.
              </p>
            </div>
            <div className="lg:col-span-7">
              {submitted ? (
                <div className="border border-ink p-8 sm:p-12 bg-paper">
                  <p className="text-display-md font-medium text-ink">Email opened.</p>
                  <p className="mt-5 sm:mt-6 text-stone-600 leading-relaxed">
                    We reply within 24 hours, IST business days.
                  </p>
                </div>
              ) : (
                <form
                  onSubmit={onSubmit}
                  className="bg-paper border border-stone-200 p-6 sm:p-10 lg:p-12 space-y-6 sm:space-y-8"
                >
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-5 sm:gap-6">
                    <Field label="Name" name="name" autoComplete="name" required />
                    <Field
                      label="Email"
                      name="email"
                      type="email"
                      autoComplete="email"
                      inputMode="email"
                      required
                    />
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-5 sm:gap-6">
                    <Field label="Company" name="company" autoComplete="organization" required />
                    <Field label="Role" name="role" autoComplete="organization-title" required />
                  </div>
                  <Field
                    label="What are you shipping?"
                    name="usecase"
                    required
                    textarea
                    placeholder="What inference workload do you want to make verifiable?"
                  />
                  <Field
                    label="Estimated volume"
                    name="volume"
                    placeholder="e.g. 500K inferences / month"
                  />
                  <button
                    type="submit"
                    disabled={submitting}
                    className="btn w-full sm:w-auto"
                  >
                    {submitting ? "Sending…" : "Send inquiry"}
                  </button>
                  <p className="text-stone-500 text-[12px] mono leading-relaxed">
                    Sent directly to hhaishwary@gmail.com. No CRM. No tracker.
                  </p>
                </form>
              )}
            </div>
          </div>
        </div>
      </section>
    </>
  );
}

function Tier({ name, price, cadence, sub, features, cta, highlight }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const cardClass = highlight
    ? "bg-paper border-2 border-ink p-7 sm:p-8 lg:p-10 flex flex-col relative"
    : "bg-paper border border-stone-200 p-7 sm:p-8 lg:p-10 flex flex-col";

  const startCheckout = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/v1/checkout/create-session`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tier: cta.tier }),
      });
      if (!res.ok) {
        const detail = await res.text();
        throw new Error(`Checkout failed (${res.status}): ${detail}`);
      }
      const { url } = await res.json();
      window.location.href = url;
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  };

  return (
    <div className={cardClass}>
      {highlight && (
        <div className="absolute -top-3 left-7 sm:left-8 lg:left-10 bg-ink text-paper text-[10px] font-medium uppercase tracking-[0.22em] px-3 py-1.5">
          Most popular
        </div>
      )}
      <div className="mb-7 sm:mb-8">
        <p className="text-[11px] font-medium text-stone-500 uppercase tracking-[0.22em] mb-5 sm:mb-6">
          {name}
        </p>
        <div className="flex items-baseline gap-2">
          <span className="text-3xl sm:text-3xl lg:text-4xl font-medium text-ink tracking-tight">
            {price}
          </span>
          {cadence && <span className="text-stone-500 text-[14px]">{cadence}</span>}
        </div>
        <p className="mt-3 text-[14px] text-stone-600 leading-snug min-h-[2.5rem]">{sub}</p>
      </div>
      <ul className="space-y-3 flex-1 mb-8 sm:mb-10">
        {features.map((f) => (
          <li key={f} className="text-[14px] text-stone-700 flex items-baseline gap-3">
            <span className="text-stone-400 mono text-[10px]">→</span>
            {f}
          </li>
        ))}
      </ul>
      {cta.kind === "checkout" ? (
        <>
          <button
            onClick={startCheckout}
            disabled={loading}
            className={
              highlight
                ? "btn w-full justify-center disabled:opacity-50"
                : "btn-ghost w-full justify-center disabled:opacity-50"
            }
          >
            {loading ? "Redirecting…" : cta.label}
          </button>
          {error && (
            <p className="mt-3 text-[12px] text-red-700 mono break-words">{error}</p>
          )}
        </>
      ) : (
        <a href={cta.href} className="link text-[14px] font-medium self-start py-1">
          {cta.label} →
        </a>
      )}
    </div>
  );
}

function Field({ label, name, type = "text", autoComplete, inputMode, required, textarea, placeholder }) {
  return (
    <label className="block">
      <span className="text-[12px] font-medium text-stone-600 mono uppercase tracking-[0.12em]">
        {label}
      </span>
      {textarea ? (
        <textarea
          name={name}
          required={required}
          rows={4}
          placeholder={placeholder}
          className="mt-2 block w-full p-4 border border-stone-300 focus:border-ink focus:outline-none bg-paper text-ink resize-y"
        />
      ) : (
        <input
          name={name}
          type={type}
          required={required}
          placeholder={placeholder}
          autoComplete={autoComplete}
          inputMode={inputMode}
          className="mt-2 block w-full p-4 border border-stone-300 focus:border-ink focus:outline-none bg-paper text-ink"
        />
      )}
    </label>
  );
}
