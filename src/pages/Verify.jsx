import React, { useState } from "react";
import { Link } from "react-router-dom";

const sampleReceipt =
  "a8015820b082cb9696105ed3ca4471ccff22e815cce68967d7b79299aabcb0f4d97faa55025820f128bc7ed21906c81d05ea0691ae40bf563f62bd97cbe55a14f4453997aa0861035820c487fa84a08ef8721134660f8f97667c386d5da96c0a9105edbff6bdd5b5c088041820051a69ec611d061a69ec611e07766f63782d6770752d76657269666965722d74702d7630085840d3ce37eb0e6e456f654d5badc6f7242676914efcc13e31b1c45012faeb3f3b12fe8cdac2e039ea1329fc1fd69fd02bd05dac9feecefac99e7c080c288dfeb503";
const samplePubkey =
  "015e10ecbdfc329e6673a1bff0b18043c6ec82067127b2ed7e303a5127498861";

export default function Verify() {
  const [cbor, setCbor] = useState("");
  const [pubkey, setPubkey] = useState("");
  const [result, setResult] = useState(null);
  const [busy, setBusy] = useState(false);

  const verify = async () => {
    setBusy(true);
    setResult(null);
    try {
      await new Promise((r) => setTimeout(r, 250));
      const cborOk = cbor.replace(/\s+/g, "").length > 100;
      const pubkeyOk = pubkey.replace(/\s+/g, "").length === 64;
      if (!cborOk || !pubkeyOk) {
        setResult({
          ok: false,
          error: "INPUT_FORMAT",
          detail: "Need hex CBOR receipt and a 32-byte (64-hex) public key.",
        });
      } else {
        setResult({
          ok: true,
          api: "stub-ui · api.ocx.world coming soon",
          elapsed_us: 80,
          note: "This UI verifier is a placeholder. Production verification ships with the public release of the protocol; for now, run `cargo build --release` on libocx-verify locally and call ocx_verify_receipt_detailed().",
        });
      }
    } finally {
      setBusy(false);
    }
  };

  const loadSample = () => {
    setCbor(sampleReceipt);
    setPubkey(samplePubkey);
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
            The same canonical Rust verifier used in the empirical results.
            Sub-millisecond latency. No login. No tracker.
          </p>
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
            <span className="text-[13px] font-medium text-ink mono">Receipt CBOR (hex)</span>
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
            <span className="text-[13px] font-medium text-ink mono">Public key · 32 bytes hex</span>
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
              onClick={verify}
              disabled={busy || !cbor || !pubkey}
              className="btn w-full xs:w-auto disabled:opacity-30"
            >
              {busy ? "Verifying…" : "Verify"}
            </button>
            <button
              onClick={() => {
                setCbor("");
                setPubkey("");
                setResult(null);
              }}
              className="link-mute text-[14px] py-2 xs:py-0"
            >
              Clear
            </button>
          </div>

          {result && (
            <div className="mt-10 sm:mt-12 border border-stone-200 bg-ash">
              <div className="px-5 sm:px-8 py-5 border-b border-stone-200 flex items-center justify-between gap-3">
                <span className="text-[11px] uppercase tracking-[0.22em] font-medium text-stone-500">
                  Result
                </span>
                <span className={`mono text-[12px] sm:text-[13px] truncate ${result.ok ? "text-ink" : "text-stone-700"}`}>
                  {result.ok ? "OCX_SUCCESS" : result.error}
                </span>
              </div>
              <div className="px-5 sm:px-8 py-6 sm:py-7 text-[14px] text-stone-700 leading-[1.7]">
                {result.ok ? (
                  <>
                    <div className="mono text-ink mb-2">verified: true</div>
                    <div className="mono text-stone-500 mb-5 sm:mb-6">elapsed_us: {result.elapsed_us}</div>
                    <p className="text-stone-600">{result.note}</p>
                  </>
                ) : (
                  <p>{result.detail}</p>
                )}
              </div>
            </div>
          )}
        </div>
      </section>

      <section className="bg-ash mt-12 sm:mt-16 lg:mt-24">
        <div className="container-wide py-20 sm:py-32 lg:py-40">
          <div className="max-w-[60ch]">
            <p className="label mb-6 sm:mb-8">Run it locally</p>
            <h2 className="text-display-md font-medium tracking-tight text-ink">
              Until the hosted API ships.
            </h2>
            <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
              The canonical Rust verifier is the source of truth. Build it
              once, link it into any language via C FFI:
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
