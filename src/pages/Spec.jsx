import React from "react";
import Diagram from "../components/Diagram";

export default function Spec() {
  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-14 sm:pb-20 lg:pb-28">
        <div className="max-w-[60ch]">
          <p className="text-[13px] mono text-stone-500 mb-8 sm:mb-10">v1 · April 2026</p>
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            The receipt format.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-600 text-base sm:text-lg leading-relaxed">
            Two hundred bytes on the wire. Eleven integer-keyed fields in
            canonical CBOR. Ed25519 signature over a domain-separated
            message. Three independent implementations produce byte-identical
            output.
          </p>
        </div>
      </section>

      <section className="py-14 sm:py-20 lg:py-28">
        <Diagram
          src="/diagrams/05-receipt-structure.html"
          ratio="1240 / 1320"
          minH={1480}
          label="Receipt · anatomy"
          title="OCX Receipt Structure · 11 fields, 200 bytes"
        />
      </section>

      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-32 lg:py-40">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-20">
            <div className="lg:col-span-5">
              <p className="label mb-6 sm:mb-8">Canonical encoding</p>
              <h2 className="text-display-md font-medium tracking-tight text-ink">
                RFC 8949 deterministic CBOR.
              </h2>
              <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
                Shortest-form integer encoding. Definite-length maps and
                arrays. Map keys sorted by canonical byte representation.
                No indefinite-length items. No floating-point in the
                signed core.
              </p>
              <p className="mt-4 text-stone-600 leading-[1.7]">
                Integer keys (rather than text keys) are deliberate: removes
                UTF-8 normalisation ambiguity and saves about fifty bytes
                per entry.
              </p>
            </div>
            <div className="lg:col-span-7 -mx-4 sm:mx-0">
              <div className="code-block">
                <pre className="whitespace-pre">
{`# Canonical signing message:

m  =  "OCXv1|receipt|"  ||  canonical_cbor(signed_map)


# Signature:

signature  =  Ed25519.sign(m, private_key)


# Wire envelope:

{
  "core":      <canonical_cbor_bytes>,
  "signature": <64 bytes>,
  "host_info": {
    driver, cuda_version, torch_version,
    gpu_uuid, parallelism_config, …
  }
}`}
                </pre>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="container-wide py-20 sm:py-32 lg:py-40">
        <div className="max-w-[60ch]">
          <p className="label mb-6 sm:mb-8">Verifier ABI</p>
          <h2 className="text-display-md font-medium tracking-tight text-ink">
            One C entry point.
          </h2>
        </div>
        <div className="mt-10 sm:mt-12 lg:mt-16 max-w-[800px] -mx-4 sm:mx-0">
          <div className="code-block">
            <pre className="whitespace-pre">
{`bool ocx_verify_receipt_detailed(
    const uint8_t* cbor_data,
    size_t          cbor_len,
    const uint8_t*  public_key,   /* 32 bytes */
    int*            error_code_out
);`}
            </pre>
          </div>
        </div>
        <p className="mt-8 sm:mt-10 text-stone-600 leading-[1.7] max-w-[60ch]">
          Returns true on success. On failure, sets <span className="mono text-ink break-all">error_code_out</span>{" "}
          to one of seven values: <span className="mono text-ink break-all">OCX_INVALID_CBOR</span>,{" "}
          <span className="mono text-ink break-all">OCX_NON_CANONICAL_CBOR</span>,{" "}
          <span className="mono text-ink break-all">OCX_MISSING_FIELD</span>,{" "}
          <span className="mono text-ink break-all">OCX_INVALID_FIELD_VALUE</span>,{" "}
          <span className="mono text-ink break-all">OCX_INVALID_SIGNATURE</span>,{" "}
          <span className="mono text-ink break-all">OCX_HASH_MISMATCH</span>,{" "}
          <span className="mono text-ink break-all">OCX_INVALID_TIMESTAMP</span>.
          Linked into Python (ctypes) and Go (cgo) clients.
        </p>
      </section>
    </>
  );
}
