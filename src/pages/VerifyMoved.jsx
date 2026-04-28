import React, { useState } from "react";
import { Link, useParams } from "react-router-dom";

/* ==================================================================
   VerifyMoved — landing page for legacy /verify/<hash> URLs that
   were embedded in receipts issued before the in-browser verifier
   was retired. Shows the hash the visitor came in with and routes
   them to a contact channel. NO verification UI. The visitor is
   typically an auditor / GST officer / accountant who clicked a
   link from a Kitaab-issued receipt; the page exists so the URL
   never feels broken.
   ================================================================== */
export default function VerifyMoved() {
  const { hash } = useParams();
  const display = (hash || "").trim();
  const looksHashy = /^[0-9a-fA-F]{32,128}$/.test(display);

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-12 sm:pb-16 lg:pb-24">
        <div className="max-w-[60ch]">
          <p className="text-[11px] uppercase tracking-[0.22em] text-stone-500 mb-6 sm:mb-8">
            Receipt verification
          </p>
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Online verification is moving.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            The browser-based receipt verifier that lived at this URL has been
            retired while we ship a redesigned, audit-grade replacement. The
            cryptographic receipt this link refers to is unchanged, signed,
            and still independently verifiable.
          </p>
        </div>
      </section>

      {display && (
        <section className="container-wide pb-12 sm:pb-16">
          <div className="max-w-[800px]">
            <p className="label mb-5 sm:mb-6">Receipt hash</p>
            <HashCard value={display} valid={looksHashy} />
          </div>
        </section>
      )}

      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-24 lg:py-32">
          <div className="max-w-[80ch]">
            <p className="label mb-6 sm:mb-8">Verify this receipt</p>

            <div className="space-y-10 sm:space-y-12">
              <Step
                n="1"
                title="Email the issuer with the hash above"
                body={
                  <>
                    If the URL you clicked was on a document, invoice, or
                    audit trail issued by <span className="text-ink">Kitaab</span>,
                    send the hash to{" "}
                    <a href="mailto:hello@kitaab.live" className="link break-all">
                      hello@kitaab.live
                    </a>{" "}
                    and you&rsquo;ll receive a signed verification report
                    within one business day.
                  </>
                }
              />
              <Step
                n="2"
                title="Verify offline using the canonical Rust verifier"
                body={
                  <>
                    The reference implementation is open source and verifies a
                    receipt against the issuer&rsquo;s public key in roughly
                    80 microseconds. Build it once, link via C&nbsp;FFI from
                    Python, Go, or Node:
                  </>
                }
              />
            </div>

            <div className="mt-10 sm:mt-12 max-w-[760px] -mx-4 sm:mx-0">
              <div className="code-block">
                <pre className="whitespace-pre">
{`git clone https://github.com/KuroKernel/ocx-protocol
cd ocx-protocol/libocx-verify
cargo build --release

# Python (linked via C FFI):
import ctypes
lib = ctypes.CDLL("./target/release/liblibocx_verify.so")
ok  = lib.ocx_verify_receipt_detailed(
        cbor_bytes, len(cbor_bytes), pubkey, ctypes.byref(err)
      )`}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="container-wide py-20 sm:py-28 lg:py-40">
        <div className="max-w-[60ch]">
          <h2 className="text-display-md font-medium tracking-tight text-ink">
            What happens next.
          </h2>
          <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
            A redesigned per-receipt verification page is in active
            development. It will fetch the receipt CBOR, run the canonical
            verifier in your browser via WebAssembly, and render a
            human-readable transcript. We will redirect this URL to it
            transparently the moment it ships. No re-issuance of receipts is
            required.
          </p>
          <p className="mt-6 sm:mt-8 text-stone-500 text-[14px] leading-relaxed">
            Status updates: <Link to="/" className="link">ocx.world</Link>{" "}
            · Source:{" "}
            <a href="https://github.com/KuroKernel/ocx-protocol" className="link">
              github.com/KuroKernel/ocx-protocol
            </a>
          </p>
        </div>
      </section>
    </>
  );
}

/* HashCard — renders the hash the user came in with, with a Copy button.
   If the hash doesn't look like hex (someone pasted garbage into the URL)
   we render it but flag it visually so a confused user doesn't waste time
   sending a malformed string to support. */
function HashCard({ value, valid }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      /* clipboard may be denied */
    }
  };
  return (
    <div className="border border-stone-200 bg-paper">
      <div className="px-5 sm:px-8 py-4 sm:py-5 border-b border-stone-200 flex items-center justify-between gap-3">
        <span className="text-[11px] uppercase tracking-[0.22em] font-medium text-stone-500">
          {valid ? "From this URL" : "Unrecognized format"}
        </span>
        <button
          type="button"
          onClick={copy}
          className="mono text-[11px] uppercase tracking-[0.18em] text-stone-500 hover:text-ink transition-colors"
          aria-label="Copy hash"
        >
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <div className="px-5 sm:px-8 py-5 sm:py-6 mono text-[13px] sm:text-[14px] text-ink break-all leading-[1.6]">
        {value}
      </div>
    </div>
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
