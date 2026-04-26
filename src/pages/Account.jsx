import React, { useState } from "react";
import { Link } from "react-router-dom";

const API_BASE = process.env.REACT_APP_API_BASE || "/api";

export default function Account() {
  const [apiKey, setApiKey] = useState("");
  const [account, setAccount] = useState(null);
  const [loadingAccount, setLoadingAccount] = useState(false);
  const [loadingPortal, setLoadingPortal] = useState(false);
  const [error, setError] = useState(null);

  const lookup = async () => {
    setError(null);
    setLoadingAccount(true);
    setAccount(null);
    try {
      const res = await fetch(`${API_BASE}/v1/account`, {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error(`Lookup failed (${res.status}): ${await res.text()}`);
      setAccount(await res.json());
    } catch (e) {
      setError(e.message);
    } finally {
      setLoadingAccount(false);
    }
  };

  const openPortal = async () => {
    setError(null);
    setLoadingPortal(true);
    try {
      const res = await fetch(`${API_BASE}/v1/billing/portal`, {
        method: "POST",
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      if (!res.ok) throw new Error(`Portal failed (${res.status}): ${await res.text()}`);
      const { url } = await res.json();
      window.location.href = url;
    } catch (e) {
      setError(e.message);
      setLoadingPortal(false);
    }
  };

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-12 sm:pb-16 lg:pb-24">
        <div className="max-w-[60ch]">
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Your account.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            Look up tier and current usage, or open the Stripe-hosted billing
            portal to update payment method, change tier, or download invoices.
          </p>
        </div>
      </section>

      <section className="container-wide py-12 sm:py-16 lg:py-24">
        <div className="max-w-[600px]">
          <label className="block">
            <span className="text-[12px] font-medium text-stone-600 mono uppercase tracking-[0.12em]">
              API key
            </span>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="ocx_live_..."
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              autoComplete="off"
              className="mt-3 block w-full p-4 border border-stone-300 focus:border-ink focus:outline-none font-mono text-[14px] bg-paper"
            />
          </label>

          <div className="mt-7 sm:mt-8 flex flex-col xs:flex-row xs:items-center gap-3 xs:gap-6">
            <button
              onClick={lookup}
              disabled={!apiKey || loadingAccount}
              className="btn-ghost w-full xs:w-auto disabled:opacity-40"
            >
              {loadingAccount ? "Loading…" : "Look up account"}
            </button>
            <button
              onClick={openPortal}
              disabled={!apiKey || loadingPortal}
              className="btn w-full xs:w-auto disabled:opacity-40"
            >
              {loadingPortal ? "Opening…" : "Open billing portal"}
            </button>
          </div>

          {error && (
            <p className="mt-5 sm:mt-6 text-[13px] text-red-700 mono break-words">
              {error}
            </p>
          )}

          {account && (
            <div className="mt-10 sm:mt-12 border border-stone-200 bg-ash p-6 sm:p-8 lg:p-10 space-y-4 sm:space-y-5">
              <Row label="Email" value={account.email} />
              <Row label="Tier" value={account.tier} />
              <Row label="Key prefix" value={account.api_key_prefix} mono />
              <Row
                label="Period"
                value={`${account.usage.period_start.slice(0, 10)} → ${account.usage.period_end.slice(0, 10)}`}
                mono
              />
              <Row
                label="Used this period"
                value={`${account.usage.receipts_verified.toLocaleString()} / ${account.monthly_limit.toLocaleString()}`}
                mono
              />
            </div>
          )}
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
          <p className="mt-5 sm:mt-6 text-stone-500 text-[14px] leading-relaxed">
            Or browse <Link to="/pricing" className="link">pricing</Link> for tier details.
          </p>
        </div>
      </section>
    </>
  );
}

function Row({ label, value, mono }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-1 sm:gap-4 sm:items-baseline">
      <span className="text-[11px] uppercase tracking-[0.18em] text-stone-500">
        {label}
      </span>
      <span
        className={`sm:col-span-2 break-all ${mono ? "mono text-[13px]" : ""} text-ink`}
      >
        {value}
      </span>
    </div>
  );
}
