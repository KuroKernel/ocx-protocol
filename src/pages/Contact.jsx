import React, { useState } from "react";
import { Link } from "react-router-dom";

/* ==================================================================
   Contact form. POSTs to Netlify Forms (form name: "contact").
   The form's static counterpart lives in public/index.html so
   Netlify can detect it at build time on a single-page React app.
   Submissions show up in Netlify dashboard → Forms, and can be
   wired to email or Slack via Netlify's notification settings.
   ================================================================== */

const FORM_NAME = "contact";

function encode(data) {
  return Object.keys(data)
    .map(
      (key) =>
        encodeURIComponent(key) + "=" + encodeURIComponent(data[key]),
    )
    .join("&");
}

export default function Contact() {
  const [state, setState] = useState({
    name: "",
    email: "",
    company: "",
    message: "",
    "bot-field": "", // honeypot
  });
  const [status, setStatus] = useState("idle"); // idle | sending | sent | error
  const [error, setError] = useState(null);

  const onChange = (e) =>
    setState((s) => ({ ...s, [e.target.name]: e.target.value }));

  const onSubmit = async (e) => {
    e.preventDefault();
    setStatus("sending");
    setError(null);
    try {
      await fetch("/", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: encode({ "form-name": FORM_NAME, ...state }),
      });
      setStatus("sent");
    } catch (err) {
      setStatus("error");
      setError(err.message || "Submission failed.");
    }
  };

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-12 sm:pb-16 lg:pb-24">
        <div className="max-w-[60ch]">
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Get in touch.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            Pilot, partnership, integration, or research. One short note,
            we'll reply within a business day. For receipt-verification
            issues from a Kitaab-issued document, please include the receipt
            hash.
          </p>
        </div>
      </section>

      <section className="container-wide py-12 sm:py-16 lg:py-24">
        <div className="max-w-[600px]">
          {status === "sent" ? (
            <SentPanel onReset={() => {
              setState({ name: "", email: "", company: "", message: "", "bot-field": "" });
              setStatus("idle");
            }} />
          ) : (
            <form
              name={FORM_NAME}
              method="POST"
              data-netlify="true"
              netlify-honeypot="bot-field"
              onSubmit={onSubmit}
              className="space-y-7 sm:space-y-8"
            >
              {/* Netlify form-name hidden input — required for SPA detection */}
              <input type="hidden" name="form-name" value={FORM_NAME} />
              {/* Honeypot — bots fill this, humans don't see it */}
              <p className="hidden">
                <label>
                  Don't fill this out: <input name="bot-field" onChange={onChange} />
                </label>
              </p>

              <Field
                label="Name"
                name="name"
                value={state.name}
                onChange={onChange}
                required
                autoComplete="name"
              />
              <Field
                label="Email"
                name="email"
                type="email"
                value={state.email}
                onChange={onChange}
                required
                autoComplete="email"
              />
              <Field
                label="Company (optional)"
                name="company"
                value={state.company}
                onChange={onChange}
                autoComplete="organization"
              />
              <FieldArea
                label="What are you looking at?"
                name="message"
                value={state.message}
                onChange={onChange}
                required
                rows={6}
                placeholder="A few sentences is plenty. Tell us what you're shipping, what you'd verify, and what's blocking you."
              />

              <div className="pt-3 sm:pt-4 flex flex-col xs:flex-row xs:items-center gap-4 xs:gap-6">
                <button
                  type="submit"
                  disabled={status === "sending"}
                  className="btn w-full xs:w-auto disabled:opacity-40"
                >
                  {status === "sending" ? "Sending…" : "Send"}
                </button>
                <Link to="/" className="link-mute text-[14px] py-2 xs:py-0">
                  Or back home →
                </Link>
              </div>

              {status === "error" && (
                <p className="mt-4 text-[13px] text-red-700 mono break-words">
                  {error || "Something went wrong. Try again or copy the address from the footer."}
                </p>
              )}
            </form>
          )}
        </div>
      </section>
    </>
  );
}

/* ------------------------------------------------------------------
   Field — single-line input. Achromatic, generous tap targets, no
   floating labels (those break copy-paste in subtle ways). Stays in
   the existing site rhythm.
------------------------------------------------------------------- */
function Field({ label, name, value, onChange, required, type = "text", autoComplete }) {
  return (
    <label className="block">
      <span className="text-[12px] font-medium text-stone-600 mono uppercase tracking-[0.12em]">
        {label}
        {required && <span className="text-stone-400 ml-1">*</span>}
      </span>
      <input
        type={type}
        name={name}
        value={value}
        onChange={onChange}
        required={required}
        autoComplete={autoComplete}
        autoCapitalize="off"
        autoCorrect="off"
        spellCheck={type === "email" ? false : undefined}
        className="mt-3 block w-full p-4 border border-stone-300 focus:border-ink focus:outline-none text-[15px] bg-paper"
      />
    </label>
  );
}

/* ------------------------------------------------------------------
   FieldArea — multi-line. Same visuals as Field, taller. Resize is
   on; the user can drag the corner if they need more room.
------------------------------------------------------------------- */
function FieldArea({ label, name, value, onChange, required, rows = 5, placeholder }) {
  return (
    <label className="block">
      <span className="text-[12px] font-medium text-stone-600 mono uppercase tracking-[0.12em]">
        {label}
        {required && <span className="text-stone-400 ml-1">*</span>}
      </span>
      <textarea
        name={name}
        rows={rows}
        value={value}
        onChange={onChange}
        required={required}
        placeholder={placeholder}
        className="mt-3 block w-full p-4 border border-stone-300 focus:border-ink focus:outline-none text-[15px] bg-paper leading-[1.6] resize-y"
      />
    </label>
  );
}

/* ------------------------------------------------------------------
   SentPanel — quiet confirmation. No bouncing checkmark, no toast.
   Just a clean state change with a path forward.
------------------------------------------------------------------- */
function SentPanel({ onReset }) {
  return (
    <div className="border border-stone-200 bg-ash p-8 sm:p-10">
      <p className="label mb-5">Sent</p>
      <h2 className="text-display-sm font-medium text-ink leading-tight">
        Thanks. We'll write back within a business day.
      </h2>
      <p className="mt-5 text-stone-600 leading-relaxed text-[15px]">
        Replies come from <span className="mono text-ink">hhaishwary@gmail.com</span>.
        If you don't see one in 24 hours, check spam.
      </p>
      <div className="mt-8 flex flex-col xs:flex-row xs:items-baseline gap-4 xs:gap-6">
        <button
          type="button"
          onClick={onReset}
          className="link-mute text-[14px] underline underline-offset-[5px] decoration-stone-300 hover:decoration-ink py-2 xs:py-0"
          style={{ background: "transparent", border: "none", cursor: "pointer", padding: 0 }}
        >
          Send another →
        </button>
        <Link to="/" className="link-mute text-[14px] py-2 xs:py-0">
          Back home →
        </Link>
      </div>
    </div>
  );
}
