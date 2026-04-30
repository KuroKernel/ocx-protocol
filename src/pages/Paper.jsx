import React from "react";
import { Link } from "react-router-dom";
import Diagram from "../components/Diagram";

const sections = [
  { n: "1", title: "Introduction", body: "Why AI inference verification matters now: regulatory pressure (EU AI Act Article 14, RBI frameworks), and why existing answers (zkML, TEEs, audit logs, Certificate Transparency) don't solve the practical case." },
  { n: "2", title: "Related work", body: "Position OCX against verifiable computation (zkML, zkVM), trusted hardware (SGX, H100 CC), append-only logs (CT), and the inference-determinism literature." },
  { n: "3", title: "Protocol", body: "Receipt schema as a CBOR map with eleven integer-keyed fields. Canonical encoding. Ed25519 signing over a domain-separated message. Environment binding." },
  { n: "4", title: "Implementation", body: "Three-language stack: Go (canonical encoder), Python (deterministic GPU inference), Rust (offline verifier with C FFI). OpenAI-compatible HTTP endpoint serving signed receipts in headers." },
  { n: "5", title: "Empirical results", body: "Cross-language byte parity (8/8). Verification latency (79.4 µs median). Twelve byte-identical inference groups across H100 and MI300X. 11,000 sequential warm-model inferences with zero failures. Cross-vendor byte-identity at single-GPU; boundary line drawn at multi-GPU collectives." },
  { n: "6", title: "Soundness", body: "Threat model. Hypergeometric soundness lemma. Risk-weighted sampling. Replay irrelevance lemma. Monte Carlo validation across 70 cells, 700K trials. Comparison table to zkML / TEE / CT / audit logs." },
  { n: "7", title: "Limitations", body: "Honest scope: not zero-knowledge, not vendor-portable at multi-GPU TP, not a defense against hardware vendor compromise, vLLM under load not yet measured, closed-source models out of scope." },
  { n: "8", title: "Conclusion", body: "Two open questions: when does cross-vendor byte-identity hold across more substrates, and can the challenge coin be removed from the trust path entirely via witness consensus." },
];

export default function Paper() {
  return (
    <>
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-16 sm:pb-20 lg:pb-28">
        <div className="max-w-[60ch]">
          <p className="text-[13px] text-stone-500 mono mb-8 sm:mb-10">v1 draft · April 2026 · forthcoming on IACR ePrint</p>
          <h1 className="text-display-lg font-medium text-ink leading-[1.05]">
            Deterministic Frontier-Scale Language Model Inference with Signed&nbsp;Receipts.
          </h1>
          <p className="mt-8 sm:mt-10 text-stone-500 mono text-[13px] flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <span>Aishwary Singh</span>
            <span aria-hidden="true">·</span>
            <span>OCX Protocol</span>
            <span aria-hidden="true">·</span>
            <Link to="/contact" className="link">
              Contact
            </Link>
          </p>
          <div className="mt-10 sm:mt-14 flex flex-col xs:flex-row xs:items-baseline gap-4 xs:gap-6">
            <a
              href="/paper.pdf"
              className="btn w-full xs:w-auto"
              target="_blank"
              rel="noopener noreferrer"
            >
              Download PDF · 467 KB · 11 pp
            </a>
            <a
              href="https://eprint.iacr.org/"
              className="link-mute text-[14px] py-2 xs:py-0"
              target="_blank"
              rel="noopener noreferrer"
            >
              IACR ePrint →
            </a>
          </div>
        </div>
      </section>

      <section className="container-wide py-16 sm:py-20 lg:py-28">
        <div className="max-w-[68ch]">
          <p className="label mb-6 sm:mb-8">Abstract</p>
          <p className="text-stone-700 leading-[1.85] text-[16px] sm:text-[17px]">
            We describe a protocol that produces byte-identical outputs from
            frontier-scale language model inference and binds each output to
            a portable, offline-verifiable signed receipt. The construction
            has three parts. First, an inference substrate that runs models
            up to seventy-two billion dense parameters and forty-seven
            billion mixture-of-experts active parameters on NVIDIA H100,
            with cross-vendor extension to AMD Instinct MI300X. Output
            hashes match byte-for-byte across fresh process launches in
            every configuration measured; at single-GPU bf16 with eager
            attention the AMD and NVIDIA hashes are themselves
            byte-identical, and at two-GPU tensor-parallel they differ as
            predicted by the underlying NCCL-ring versus RCCL-fabric
            all-reduce topology. Second, a canonical CBOR receipt schema
            with an Ed25519 signature over a domain-separated message,
            implemented in Go, Python, and Rust with cross-language
            byte-identity verified end-to-end. Third, a probabilistic
            spot-check verifier that re-executes a small sample of receipts
            and rejects on mismatch; we prove a soundness lemma of the form
            1 − (1−f)<sup>k</sup> and validate it empirically across seventy
            adversary-verifier configurations with seven hundred thousand
            Monte Carlo trials. Verification costs about eighty
            microseconds per receipt on a single core. Eleven thousand
            sequential warm-model inferences ran without a single
            byte-identity failure. The contribution is the construction
            itself: a primitive that gives issuer-independent fabrication
            soundness for AI inference at production cost, without a
            hardware-vendor dependency and without zero-knowledge proofs.
          </p>
        </div>
      </section>

      <section className="py-16 sm:py-24 lg:py-32">
        <Diagram
          src="/diagrams/02-threat-model.html"
          ratio="1200 / 980"
          minH={1080}
          label="Figure 1 · Threat model"
          title="OCX Threat Model · Trust Boundaries"
        />
      </section>

      {/* Sections list — on mobile each section becomes a stacked block;
          on lg, the original 12-col grid layout returns. */}
      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-24 lg:py-32">
          <p className="label mb-8 sm:mb-12">Sections</p>
          <div className="space-y-10 sm:space-y-12">
            {sections.map((s) => (
              <article
                key={s.n}
                className="lg:grid lg:grid-cols-12 lg:gap-12"
              >
                <div className="flex items-baseline gap-4 lg:contents">
                  <span className="lg:col-span-1 mono text-stone-400 text-[13px] lg:pt-1">
                    {String(s.n).padStart(2, "0")}
                  </span>
                  <h3 className="lg:col-span-3 font-medium text-ink text-[16px] sm:text-[17px]">
                    {s.title}
                  </h3>
                </div>
                <p className="mt-3 lg:mt-0 lg:col-span-8 text-stone-600 leading-[1.7] text-[15px]">
                  {s.body}
                </p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="container-wide py-24 sm:py-32 lg:py-40">
        <div className="max-w-[60ch]">
          <h2 className="text-display-md font-medium tracking-tight text-ink">
            Every measurement points at a committed file.
          </h2>
          <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
            Receipts under <span className="mono text-ink break-all">examples/gpu-verifier/results/</span>.
            Test plan and aggregated results under <span className="mono text-ink break-all">whitepaper-tests/</span>.
            Monte Carlo soundness data under{" "}
            <span className="mono text-ink break-all">adversarial_soundness.jsonl</span>.
            A reviewer can run the listed commands on the listed commit and
            observe the same hashes.
          </p>
          <Link
            to="/spec"
            className="mt-10 sm:mt-12 inline-flex items-center text-ink underline underline-offset-[6px] decoration-stone-400 hover:decoration-ink"
          >
            Read the receipt specification →
          </Link>
        </div>
      </section>
    </>
  );
}
