import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { loadVerifier, STATUS_DESCRIPTIONS } from "../lib/ocxVerifier";

const sampleReceipt =
  "a8015820b082cb9696105ed3ca4471ccff22e815cce68967d7b79299aabcb0f4d97faa55025820f128bc7ed21906c81d05ea0691ae40bf563f62bd97cbe55a14f4453997aa0861035820c487fa84a08ef8721134660f8f97667c386d5da96c0a9105edbff6bdd5b5c088041820051a69ec611d061a69ec611e07766f63782d6770752d76657269666965722d74702d7630085840d3ce37eb0e6e456f654d5badc6f7242676914efcc13e31b1c45012faeb3f3b12fe8cdac2e039ea1329fc1fd69fd02bd05dac9feecefac99e7c080c288dfeb503";
const samplePubkey =
  "015e10ecbdfc329e6673a1bff0b18043c6ec82067127b2ed7e303a5127498861";

/* ==================================================================
   Verify — pastes-and-verdict.
   The form runs the canonical libocx-verify Rust crate compiled to
   WebAssembly. Everything happens locally: no server, no network,
   no telemetry. The wasm binary is fetched once on demand.
   ================================================================== */
export default function Verify() {
  const [cbor, setCbor] = useState("");
  const [pubkey, setPubkey] = useState("");
  const [result, setResult] = useState(null);
  const [busy, setBusy] = useState(false);
  const [verifier, setVerifier] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const verifyButtonRef = useRef(null);

  // Load the wasm verifier on first paint. The promise is memoized in
  // ocxVerifier.js so subsequent route visits don't refetch.
  useEffect(() => {
    let cancelled = false;
    loadVerifier()
      .then((v) => {
        if (!cancelled) setVerifier(v);
      })
      .catch((err) => {
        if (!cancelled) setLoadError(err.message || String(err));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const verify = () => {
    if (!verifier) return;
    setBusy(true);
    setResult(null);
    // Push the actual call to the next animation frame so the "Verifying…"
    // state has a chance to paint. The wasm verify itself runs in <1ms
    // for valid receipts; without this the UI looks like nothing happened.
    requestAnimationFrame(() => {
      try {
        const t0 =
          typeof performance !== "undefined" ? performance.now() : Date.now();
        const out = verifier.verifyHex(cbor, pubkey);
        const t1 =
          typeof performance !== "undefined" ? performance.now() : Date.now();
        setResult({ ...out, elapsed_ms: t1 - t0 });
      } catch (e) {
        setResult({
          ok: false,
          status_code: "INVALID_HEX_INPUT",
          message: e.message || String(e),
          receipt: null,
          bytes_total: 0,
          signed_message_bytes: 0,
          elapsed_ms: 0,
        });
      } finally {
        setBusy(false);
      }
    });
  };

  const loadSample = () => {
    setCbor(sampleReceipt);
    setPubkey(samplePubkey);
    setResult(null);
  };

  const clear = () => {
    setCbor("");
    setPubkey("");
    setResult(null);
  };

  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-12 sm:pb-16 lg:pb-24">
        <div className="max-w-[60ch]">
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Paste a receipt. Get a verdict.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            The canonical Rust verifier ({verifier ? verifier.version : "libocx-verify"}),
            compiled to WebAssembly and running entirely in your browser.
            No server in the trust path. No login. No tracker.
          </p>

          {/* Load state — shown only while the wasm fetches. Removes itself
              once the verifier is ready. Stays out of the way otherwise. */}
          {!verifier && !loadError && (
            <p className="mt-6 text-[13px] mono text-stone-500">
              <Spinner /> Loading verifier (~50 KB compressed) …
            </p>
          )}
          {loadError && (
            <p className="mt-6 text-[13px] mono text-red-700">
              Verifier failed to load: {loadError}
            </p>
          )}
        </div>
      </section>

      <section className="container-wide py-12 sm:py-16 lg:py-24">
        <div className="max-w-[800px]">
          <div className="flex flex-wrap items-baseline justify-between gap-3 mb-8 sm:mb-10">
            <p className="label">Receipt</p>
            <button
              onClick={loadSample}
              className="text-[13px] text-stone-500 hover:text-ink underline underline-offset-[5px] decoration-stone-300 hover:decoration-ink transition-colors py-1"
              style={{ touchAction: "manipulation" }}
            >
              Load sample (Mixtral 8x7B)
            </button>
          </div>

          <label className="block">
            <span className="text-[13px] font-medium text-ink mono">
              Receipt CBOR (hex)
            </span>
            <textarea
              value={cbor}
              onChange={(e) => setCbor(e.target.value)}
              placeholder="a801582 ..."
              rows={8}
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              autoComplete="off"
              className="mt-3 block w-full p-4 sm:p-5 border border-stone-300 focus:border-ink focus:outline-none font-mono text-[13px] sm:text-[12px] leading-[1.7] text-stone-700 resize-y bg-paper"
            />
          </label>

          <label className="block mt-8 sm:mt-10">
            <span className="text-[13px] font-medium text-ink mono">
              Public key · 32 bytes hex
            </span>
            <input
              type="text"
              inputMode="text"
              value={pubkey}
              onChange={(e) => setPubkey(e.target.value)}
              placeholder="015e10ec ..."
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              autoComplete="off"
              className="mt-3 block w-full p-4 sm:p-5 border border-stone-300 focus:border-ink focus:outline-none font-mono text-[13px] sm:text-[12px] text-stone-700 bg-paper"
            />
          </label>

          <div className="mt-8 sm:mt-10 flex flex-col xs:flex-row xs:items-baseline gap-3 xs:gap-6">
            <button
              ref={verifyButtonRef}
              onClick={verify}
              disabled={!verifier || busy || !cbor || !pubkey}
              className="btn w-full xs:w-auto disabled:opacity-30"
            >
              {busy ? "Verifying…" : "Verify"}
            </button>
            <button onClick={clear} className="link-mute text-[14px] py-2 xs:py-0">
              Clear
            </button>
          </div>

          {result && <ResultPanel result={result} />}
        </div>
      </section>

      <section className="bg-ash mt-12 sm:mt-16 lg:mt-24">
        <div className="container-wide py-20 sm:py-32 lg:py-40">
          <div className="max-w-[60ch]">
            <p className="label mb-6 sm:mb-8">Run it locally</p>
            <h2 className="text-display-md font-medium tracking-tight text-ink">
              The same verifier, on your own machine.
            </h2>
            <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
              The verification you just ran in the browser is the same
              canonical Rust crate, with one difference: this build skips
              the redundant <span className="mono text-ink">ring</span>{" "}
              cross-check (which doesn&rsquo;t cross-compile to wasm
              cleanly) and relies on{" "}
              <span className="mono text-ink">ed25519-dalek</span> alone
              for signature math. The native build runs both libraries.
              Either way the receipt either verifies or it doesn&rsquo;t —
              the math is identical.
            </p>
          </div>
          <div className="mt-10 sm:mt-12 max-w-[800px] -mx-4 sm:mx-0">
            <div className="code-block">
              <pre className="whitespace-pre">
{`git clone https://github.com/KuroKernel/ocx-protocol
cd ocx-protocol/libocx-verify
cargo build --release

# Python:
import ctypes
lib = ctypes.CDLL("./target/release/liblibocx_verify.so")
ok  = lib.ocx_verify_receipt_detailed(
        cbor_bytes, len(cbor_bytes), pubkey, ctypes.byref(err)
      )`}
              </pre>
            </div>
          </div>
          <Link
            to="/spec"
            className="mt-10 sm:mt-12 inline-flex items-center text-ink underline underline-offset-[6px] decoration-stone-400 hover:decoration-ink"
          >
            Read the verifier ABI →
          </Link>
        </div>
      </section>
    </>
  );
}

/* ==================================================================
   ResultPanel — shows the verifier's verdict.
   Two states:
     • ok    — minimal triumph: status code, elapsed time, parsed fields
     • fail  — quiet failure: status code + plain-language explanation
   Both mirror the same skeleton so the layout doesn't jump.
   ================================================================== */
function ResultPanel({ result }) {
  const tone = result.ok ? "text-ink" : "text-stone-700";
  const description =
    STATUS_DESCRIPTIONS[result.status_code] || result.message;

  return (
    <div className="mt-10 sm:mt-12 border border-stone-200 bg-ash">
      {/* Header strip — status code on the left, elapsed time on the right. */}
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 sm:px-8 py-5 border-b border-stone-200">
        <div className="flex items-baseline gap-3">
          <span className="text-[11px] uppercase tracking-[0.22em] font-medium text-stone-500">
            Result
          </span>
          <span className={`mono text-[12px] sm:text-[13px] truncate ${tone}`}>
            OCX_{result.status_code}
          </span>
        </div>
        {typeof result.elapsed_ms === "number" && (
          <span className="mono text-[12px] text-stone-500 tabular-nums">
            {result.elapsed_ms < 1
              ? `${(result.elapsed_ms * 1000).toFixed(0)} µs`
              : `${result.elapsed_ms.toFixed(2)} ms`}
          </span>
        )}
      </div>

      {/* Body — different shape for success vs failure. */}
      {result.ok ? (
        <div className="px-5 sm:px-8 py-6 sm:py-8 space-y-6">
          <div>
            <p className="text-stone-600 leading-[1.7] text-[14px] sm:text-[15px]">
              {description}
            </p>
          </div>
          {result.receipt && <ReceiptDisplay receipt={result.receipt} />}
          <ByteFooter result={result} />
        </div>
      ) : (
        <div className="px-5 sm:px-8 py-6 sm:py-7 text-[14px] text-stone-700 leading-[1.7]">
          <p>{description}</p>
          {result.message && result.message !== description && (
            <p className="mt-3 text-stone-500 mono text-[12px] break-all">
              {result.message}
            </p>
          )}
          {result.receipt && (
            <details className="mt-6 group">
              <summary className="cursor-pointer text-[13px] text-stone-500 hover:text-ink underline underline-offset-[5px] decoration-stone-300">
                Show parsed fields
              </summary>
              <div className="mt-5">
                <ReceiptDisplay receipt={result.receipt} />
              </div>
            </details>
          )}
          <ByteFooter result={result} className="mt-6" />
        </div>
      )}
    </div>
  );
}

/* ReceiptDisplay — the parsed CBOR fields. Reads as a definition list
   with the labels left and the values right; values that are hashes
   show in mono with truncation + a copy affordance. */
function ReceiptDisplay({ receipt }) {
  return (
    <dl className="grid grid-cols-1 sm:grid-cols-[max-content_1fr] gap-x-8 gap-y-3 text-[13px]">
      <Field label="artifact_hash">
        <Hash value={receipt.artifact_hash} />
      </Field>
      <Field label="input_hash">
        <Hash value={receipt.input_hash} />
      </Field>
      <Field label="output_hash">
        <Hash value={receipt.output_hash} />
      </Field>
      <Field label="cycles_used">
        <span className="mono tabular-nums text-ink">
          {Number(receipt.cycles_used).toLocaleString("en-US")}
        </span>
      </Field>
      <Field label="started_at">
        <Time epoch={receipt.started_at} />
      </Field>
      <Field label="finished_at">
        <Time epoch={receipt.finished_at} />
      </Field>
      <Field label="duration">
        <span className="mono tabular-nums text-ink">
          {receipt.duration_seconds} s
        </span>
      </Field>
      <Field label="issuer_key_id">
        <span className="mono text-ink break-all">{receipt.issuer_key_id}</span>
      </Field>
      <Field label="signature">
        <Hash value={receipt.signature_hex} />
      </Field>
      {receipt.has_prev_receipt_hash && (
        <Field label="prev_receipt_hash">
          <span className="mono text-stone-500">[present]</span>
        </Field>
      )}
      {receipt.has_request_digest && (
        <Field label="request_digest">
          <span className="mono text-stone-500">[present]</span>
        </Field>
      )}
      {receipt.witness_count > 0 && (
        <Field label="witness_signatures">
          <span className="mono tabular-nums text-stone-500">
            {receipt.witness_count}
          </span>
        </Field>
      )}
      {receipt.has_vdf_proof && (
        <Field label="vdf_proof">
          <span className="mono text-stone-500">
            present
            {receipt.vdf_iterations &&
              ` · ${Number(receipt.vdf_iterations).toLocaleString("en-US")} iterations`}
          </span>
        </Field>
      )}
    </dl>
  );
}

function Field({ label, children }) {
  return (
    <>
      <dt className="text-[11px] uppercase tracking-[0.18em] text-stone-500 font-medium pt-1">
        {label}
      </dt>
      <dd className="text-stone-700 break-all">{children}</dd>
    </>
  );
}

/* Hash — long hex string with a Copy button that reveals on hover and
   stays visible on tap (touch). Doesn't truncate; the break-all does
   the work so the entire hash stays selectable. */
function Hash({ value }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      /* clipboard may be denied — silent */
    }
  };
  return (
    <span className="inline-flex items-baseline gap-2 group">
      <span className="mono text-ink break-all">{value}</span>
      <button
        type="button"
        onClick={copy}
        aria-label="Copy"
        className="mono text-[10px] uppercase tracking-[0.18em] text-stone-400 hover:text-ink opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </span>
  );
}

/* Time — Unix epoch seconds rendered as both ISO and a relative phrase.
   Times are intentionally shown in UTC to match the canonical receipt
   semantics (no timezone field is part of the receipt). */
function Time({ epoch }) {
  if (!epoch || epoch === 0) return <span className="mono text-stone-400">—</span>;
  const d = new Date(epoch * 1000);
  const iso = d.toISOString().replace("T", " ").replace(/\.\d+Z$/, "Z");
  return (
    <span className="mono text-ink tabular-nums">
      {iso}
      <span className="ml-2 text-stone-500">· {epoch}</span>
    </span>
  );
}

/* ByteFooter — the small "200 B receipt · 152 B signed message" line at
   the bottom of the result panel. Tells you exactly what the verifier
   processed, in case you want to cross-check against the spec. */
function ByteFooter({ result, className = "" }) {
  const parts = [];
  if (result.bytes_total) parts.push(`${result.bytes_total} B receipt`);
  if (result.signed_message_bytes)
    parts.push(`${result.signed_message_bytes} B signed message`);
  if (parts.length === 0) return null;
  return (
    <p className={`text-[12px] mono text-stone-500 ${className}`}>
      {parts.join(" · ")}
    </p>
  );
}

/* Spinner — tiny inline glyph for the wasm-loading line. SVG so it
   doesn't import any animation library. Honors prefers-reduced-motion
   via the global override in index.css. */
function Spinner() {
  return (
    <svg
      width="11"
      height="11"
      viewBox="0 0 11 11"
      style={{
        display: "inline-block",
        verticalAlign: "-1px",
        marginRight: "0.4em",
        animation: "ocx-spin 0.9s linear infinite",
      }}
      aria-hidden="true"
    >
      <circle
        cx="5.5"
        cy="5.5"
        r="4"
        fill="none"
        stroke="#8C8C8C"
        strokeWidth="1.2"
        strokeDasharray="6 18"
        strokeLinecap="round"
      />
      <style>{`@keyframes ocx-spin { to { transform: rotate(360deg); } }`}</style>
    </svg>
  );
}
