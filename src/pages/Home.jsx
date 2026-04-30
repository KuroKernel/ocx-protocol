import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { ContactButton } from "../components/Layout";

/* ==================================================================
   Motion primitives — small, achromatic, in-house.
   Three things only:
     • Reveal       — fade + 8px lift when element enters viewport
     • CountUp      — number rolls from 0 to its final value
     • LifecycleStrip — the receipt's "verify forever" timeline
   All respect prefers-reduced-motion via the global override in index.css.
   ================================================================== */

// IntersectionObserver hook. Fires once when the element first enters the
// viewport. We don't unmount-animate — protocol pages don't need exit motion.
function useInView(options = { threshold: 0.2, rootMargin: "0px 0px -8% 0px" }) {
  const ref = useRef(null);
  const [inView, setInView] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (typeof IntersectionObserver === "undefined") {
      setInView(true); // server / unsupported — render visible
      return;
    }
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setInView(true);
          obs.unobserve(entry.target);
        }
      },
      options
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [options.threshold, options.rootMargin]); // eslint-disable-line
  return [ref, inView];
}

// Mount tick — flips true on the second animation frame so the initial
// `opacity: 0` state has time to paint before the transition fires.
function useMounted() {
  const [m, setM] = useState(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => requestAnimationFrame(() => setM(true)));
    return () => cancelAnimationFrame(id);
  }, []);
  return m;
}

/* Fade + 8px lift. Applies on mount (default) or on intersection (when
   `onView` is set). Stagger via the `delay` prop. */
function Reveal({ children, delay = 0, onView = false, as: Tag = "div", className = "" }) {
  const mounted = useMounted();
  const [ref, inView] = useInView();
  const visible = onView ? inView : mounted;
  return (
    <Tag
      ref={onView ? ref : null}
      className={className}
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? "translate3d(0,0,0)" : "translate3d(0,8px,0)",
        transition: `opacity 600ms cubic-bezier(0.22, 1, 0.36, 1) ${delay}ms, transform 700ms cubic-bezier(0.22, 1, 0.36, 1) ${delay}ms`,
        willChange: visible ? "auto" : "opacity, transform",
      }}
    >
      {children}
    </Tag>
  );
}

/* The lifecycle strip — three dots on a hairline, traced from left to
   right when in view. Communicates "issued once, verifiable forever." */
function LifecycleStrip() {
  const [ref, inView] = useInView();
  return (
    <div ref={ref} className="mt-10 sm:mt-12 pt-10 sm:pt-12 border-t border-stone-200">
      <div className="grid grid-cols-3 gap-4 sm:gap-6 mb-5 sm:mb-6">
        <LifecycleLabel
          eyebrow="Issued"
          when="t = 0"
          note="signed once"
          delay={300}
          show={inView}
        />
        <LifecycleLabel
          eyebrow="Verified"
          when="+ 1 day"
          note="79 µs · offline"
          delay={650}
          show={inView}
        />
        <LifecycleLabel
          eyebrow="Verified again"
          when="+ 5 years"
          note="same 200 bytes"
          delay={1000}
          show={inView}
          focal
        />
      </div>
      <svg
        viewBox="0 0 600 24"
        preserveAspectRatio="none"
        className="block w-full h-6"
        aria-hidden="true"
      >
        {/* Faint baseline always present */}
        <line
          x1="0" y1="12" x2="600" y2="12"
          stroke="#E4E4E4" strokeWidth="1"
        />
        {/* Animated trace — strokeDasharray draws from 0 to full when in view */}
        <line
          x1="0" y1="12" x2="600" y2="12"
          stroke="#141414" strokeWidth="1"
          strokeDasharray="600"
          strokeDashoffset={inView ? 0 : 600}
          style={{
            transition: "stroke-dashoffset 1400ms cubic-bezier(0.22, 1, 0.36, 1) 200ms",
          }}
        />
        {/* Three dots — appear via opacity at calibrated times */}
        <circle
          cx="0" cy="12" r="4" fill="#141414"
          style={{
            opacity: inView ? 1 : 0,
            transition: "opacity 300ms ease-out 100ms",
          }}
        />
        <circle
          cx="300" cy="12" r="4" fill="#141414"
          style={{
            opacity: inView ? 1 : 0,
            transition: "opacity 300ms ease-out 750ms",
          }}
        />
        <circle
          cx="600" cy="12" r="5" fill="#141414"
          style={{
            opacity: inView ? 1 : 0,
            transform: inView ? "scale(1)" : "scale(0.6)",
            transformOrigin: "600px 12px",
            transition:
              "opacity 300ms ease-out 1300ms, transform 600ms cubic-bezier(0.22, 1, 0.36, 1) 1300ms",
          }}
        />
      </svg>
    </div>
  );
}

function LifecycleLabel({ eyebrow, when, note, delay, show, focal }) {
  return (
    <div
      style={{
        opacity: show ? 1 : 0,
        transform: show ? "translate3d(0,0,0)" : "translate3d(0,4px,0)",
        transition: `opacity 500ms cubic-bezier(0.22, 1, 0.36, 1) ${delay}ms, transform 600ms cubic-bezier(0.22, 1, 0.36, 1) ${delay}ms`,
      }}
    >
      <p className="text-[10px] sm:text-[11px] font-medium uppercase tracking-[0.18em] text-stone-500">
        {eyebrow}
      </p>
      <p className={`mt-2 text-[14px] sm:text-[15px] ${focal ? "font-medium text-ink" : "text-ink"}`}>
        {when}
      </p>
      <p className="mt-1 text-[12px] sm:text-[13px] text-stone-500 mono">
        {note}
      </p>
    </div>
  );
}

/* ==================================================================
   PAGE
   ================================================================== */

export default function Home() {
  return (
    <>
      {/* ============================================================
          HERO — fade-up sequence on first paint.
          ============================================================ */}
      <section className="container-wide pt-20 sm:pt-32 lg:pt-48 pb-20 sm:pb-32 lg:pb-44 min-h-screen-safe flex flex-col justify-center">
        <Reveal as="h1" className="text-display-xl font-medium text-ink max-w-[18ch]">
          A protocol for verifiable AI&nbsp;inference.
        </Reveal>
        <Reveal
          delay={140}
          as="p"
          className="mt-8 sm:mt-12 lg:mt-16 text-base sm:text-lg lg:text-xl text-stone-600 leading-[1.6] max-w-[60ch]"
        >
          OCX produces byte-identical outputs from frontier-scale language
          models, binds each output to a 200-byte cryptographic receipt, and
          lets anyone verify offline in microseconds. No hardware-vendor
          trust. No zero-knowledge proofs.
        </Reveal>
        <Reveal
          delay={280}
          className="mt-10 sm:mt-16 lg:mt-20 flex flex-col xs:flex-row xs:items-center gap-4 xs:gap-6 sm:gap-8"
        >
          <Link to="/paper" className="btn w-full xs:w-auto">Read the whitepaper</Link>
          <ContactButton
            className="link-mute text-[14px] py-2 xs:py-0"
            idleLabel="Get in touch →"
          />
        </Reveal>
      </section>

      {/* ============================================================
          RECEIPT — the marquee artifact.
          New: a lifecycle strip at the bottom that traces "issued →
          verified → verified again, +5 years" when scrolled into view.
          That motion IS the punchline of the protocol.
          ============================================================ */}
      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-28 lg:py-36">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-20 items-start">
            <Reveal onView className="lg:col-span-5">
              <p className="label mb-6 sm:mb-8">A receipt</p>
              <h2 className="text-display-md font-medium tracking-tight text-ink">
                Two hundred bytes.
              </h2>
              <p className="mt-6 sm:mt-8 text-stone-600 leading-[1.7]">
                A canonical CBOR map with eleven integer-keyed fields, signed
                by Ed25519 over a domain-separated message. Three independent
                implementations — Go, Python, Rust — produce byte-identical
                output. Verification is a single FFI call.
              </p>
              <Link
                to="/spec"
                className="mt-8 sm:mt-10 inline-flex items-center text-ink underline underline-offset-[6px] decoration-stone-400 hover:decoration-ink"
              >
                Read the specification →
              </Link>
            </Reveal>
            <Reveal onView delay={120} className="lg:col-span-7 -mx-4 sm:mx-0">
              <div className="code-block bg-paper">
                <pre className="whitespace-pre">
{`a8                              # CBOR map of 8 entries
01 5820                         # key 1: program_hash (sha256, 32B)
   b082cb96 96105ed3 ca4471cc ff22e815
   cce68967 d7b79299 aabcb0f4 d97faa55
02 5820                         # key 2: input_hash
   f128bc7e d21906c8 1d05ea06 91ae40bf
   563f62bd 97cbe55a 14f44539 97aa0861
03 5820                         # key 3: output_hash
   c487fa84 a08ef872 1134660f 8f97667c
   386d5da9 6c0a9105 edbff6bd d5b5c088
04 1820                         # key 4: cycles_used = 32
05 1a 69ec611d                  # key 5: started_at
06 1a 69ec611e                  # key 6: finished_at
07 75 6f63782d776562736974652d  # key 7: issuer_id
   73 616d706c652d7631
08 5840                         # key 8: ed25519 signature, 64B
   eb299f3d 9f67d0de 3af87263 49bf95a1
   4073c09f bdaf7b6d 90a3f8da cc844c5c
   b017c73f a4660cde d7322bd8 465868f5
   66cf4162 0c621167 03154bbc 4eaf0e05`}
                </pre>
                <div className="mt-6 pt-6 border-t border-stone-200 text-[12px] text-stone-500 leading-relaxed">
                  Real signed receipt · 211 wire bytes · the canonical Rust
                  verifier (200 LOC, MIT) confirms the signature offline in
                  microseconds.
                  {" "}
                  <Link to="/agent" className="link">
                    Watch a live agent run produce one →
                  </Link>
                </div>
              </div>
            </Reveal>
          </div>

          {/* Lifecycle strip — pedagogical motion. The receipt verifies
              today, in five years, by anyone, offline. Animates in once
              the user scrolls past the artifact. */}
          <Reveal onView delay={200}>
            <LifecycleStrip />
          </Reveal>
        </div>
      </section>

      {/* ============================================================
          EMPIRICAL — prose, not a stat wall.
          Numbers live inside sentences where they do real work.
          Reads like an IACR abstract, not a landing page brag.
          ============================================================ */}
      <section className="container-wide py-20 sm:py-28 lg:py-36">
        <Reveal onView as="p" className="label mb-6 sm:mb-8">
          Empirical results
        </Reveal>
        <Reveal onView delay={80} as="h2" className="text-display-md font-medium tracking-tight text-ink max-w-[20ch]">
          What we measured.
        </Reveal>
        <Reveal onView delay={160} className="mt-8 sm:mt-12 max-w-[68ch] text-stone-700 leading-[1.85] text-base sm:text-[17px] space-y-5">
          <p>
            We ran the protocol across NVIDIA H100 and AMD MI300X — single-GPU
            bf16, tensor-parallel, pipeline-parallel, and CPU offload. The
            largest model tested was{" "}
            <span className="text-ink font-medium">
              seventy-two billion dense parameters
            </span>
            . Eleven thousand sequential warm-model inferences ran without a
            single byte-identity failure across fresh process launches.
          </p>
          <p>
            Median verification latency on a single core, offline, is{" "}
            <span className="text-ink font-medium">
              seventy-nine microseconds
            </span>
            . The receipt is two hundred bytes on the wire — eleven CBOR
            fields plus a sixty-four-byte Ed25519 signature. The Rust
            verifier is seven kilobytes, no_std, links into any language via
            C FFI.
          </p>
        </Reveal>
        <Reveal onView delay={240}>
          <Link
            to="/paper"
            className="mt-10 sm:mt-12 inline-flex items-center text-ink underline underline-offset-[6px] decoration-stone-400 hover:decoration-ink"
          >
            Read the empirical section of the paper →
          </Link>
        </Reveal>
      </section>

      {/* ============================================================
          SOUNDNESS
          ============================================================ */}
      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-28 lg:py-36">
          <div className="max-w-[68ch]">
            <Reveal onView as="p" className="label mb-6 sm:mb-8">
              Soundness
            </Reveal>
            <Reveal onView as="h2" delay={80} className="text-display-md font-medium tracking-tight text-ink">
              Cheating gets exponentially harder.
            </Reveal>
            <Reveal onView delay={180} className="mt-10 sm:mt-12 py-8 sm:py-10 px-6 sm:px-10 bg-paper border-l-2 border-ink">
              <p className="text-2xl sm:text-3xl lg:text-4xl text-ink mono leading-[1.4]">
                P[catch] = 1 − (1 − f)<sup className="text-[0.55em] align-super">k</sup>
              </p>
              <p className="mt-4 text-[13px] mono text-stone-500">
                f = fraction of fabricated receipts &nbsp;·&nbsp; k = independent probes
              </p>
            </Reveal>
            <Reveal onView delay={260} as="p" className="mt-10 sm:mt-12 text-stone-600 leading-[1.8]">
              An issuer fabricating one percent of receipts is caught with
              probability 0.634 in a hundred probes. Compounded across a
              year of independent verification — say, ten thousand probes
              from independent verifiers — the escape probability falls
              below 10<sup>−43</sup>. That is smaller than guessing a
              private key.
            </Reveal>
            <Reveal onView delay={320} as="p" className="mt-5 text-stone-600 leading-[1.8]">
              The soundness lemma is proved formally in Section 6 of the
              paper and validated empirically across seventy
              adversary-verifier configurations with seven hundred
              thousand Monte Carlo trials.
            </Reveal>
          </div>
        </div>
      </section>

      {/* ============================================================
          COMPARISON — table.
          ============================================================ */}
      <section className="container-wide py-20 sm:py-28 lg:py-36">
        <Reveal onView as="p" className="label mb-6 sm:mb-8">
          Compared to
        </Reveal>
        <Reveal onView as="h2" delay={80} className="text-display-md font-medium tracking-tight text-ink max-w-[20ch]">
          The trust-cost frontier.
        </Reveal>
        <Reveal onView delay={160} as="p" className="mt-6 sm:mt-8 text-stone-600 leading-[1.7] max-w-[60ch]">
          Two questions decide a verification scheme: how long does it take
          to verify, and who do you have to trust to believe the answer.
        </Reveal>

        <Reveal onView delay={220} className="mt-12 sm:mt-16 -mx-4 sm:mx-0">
          <table className="w-full text-left">
            <thead>
              <tr className="border-y border-ink">
                <th className="py-4 pr-6 pl-4 sm:pl-0 text-[11px] uppercase tracking-[0.18em] text-stone-500 font-medium">
                  Approach
                </th>
                <th className="py-4 px-6 text-[11px] uppercase tracking-[0.18em] text-stone-500 font-medium">
                  Verification
                </th>
                <th className="hidden sm:table-cell py-4 px-6 text-[11px] uppercase tracking-[0.18em] text-stone-500 font-medium">
                  Trust required
                </th>
                <th className="py-4 pl-6 pr-4 sm:pr-0 text-[11px] uppercase tracking-[0.18em] text-stone-500 font-medium">
                  Scale today
                </th>
              </tr>
            </thead>
            <tbody>
              <ComparisonRow
                approach="OCX"
                verification="79 µs"
                trust="Math · Ed25519 + SHA-256"
                scale="Frontier · 72B"
                focal
              />
              <ComparisonRow
                approach="TEEs · SGX, SEV, TDX"
                verification="microseconds"
                trust="Hardware vendor"
                scale="Frontier"
              />
              <ComparisonRow
                approach="zkML · Halo2, EZKL"
                verification="minutes (proving)"
                trust="Math · trusted setup"
                scale="Toy · ≤ 1B"
              />
              <ComparisonRow
                approach="Audit logs · SIEM"
                verification="seconds (search)"
                trust="Log producer"
                scale="Frontier · weak claim"
              />
            </tbody>
          </table>
        </Reveal>

        <Reveal onView delay={280} as="p" className="mt-10 sm:mt-12 text-stone-500 text-[14px] leading-[1.7] max-w-[60ch]">
          OCX is the only approach that delivers microsecond verification
          with no vendor in the trust path, at frontier model scale.
        </Reveal>
      </section>

      {/* ============================================================
          CTA
          ============================================================ */}
      <section className="bg-ash">
        <div className="container-wide py-20 sm:py-32 lg:py-44">
          <Reveal onView className="max-w-[44ch]">
            <h2 className="text-display-lg font-medium tracking-tightest text-ink">
              Read the paper.
            </h2>
            <div className="mt-8 sm:mt-12 flex flex-col xs:flex-row xs:items-baseline gap-4 xs:gap-6 sm:gap-8">
              <Link to="/paper" className="btn w-full xs:w-auto">
                Whitepaper · 467 KB PDF
              </Link>
              <a
                href="https://github.com/KuroKernel/ocx-protocol"
                className="link-mute text-[14px] py-2 xs:py-0"
              >
                Source on GitHub →
              </a>
            </div>
          </Reveal>
        </div>
      </section>
    </>
  );
}

/* ------------------------------------------------------------------
   ComparisonRow — focal row gets a left ink bar that grows from 0 to
   full height when the table enters view. Tiny but communicative:
   "this is the answer."
------------------------------------------------------------------- */
function ComparisonRow({ approach, verification, trust, scale, focal }) {
  const [ref, inView] = useInView();
  return (
    <tr
      ref={ref}
      className={[
        "border-b border-stone-200 align-baseline",
        focal ? "bg-ash" : "",
      ].join(" ")}
    >
      <td
        className={[
          "py-5 sm:py-6 pr-6 pl-4 sm:pl-0",
          focal ? "font-medium text-ink" : "text-ink",
        ].join(" ")}
      >
        {focal && (
          <span
            className="inline-block w-1 bg-ink mr-3 align-middle"
            style={{
              height: inView ? "1rem" : "0px",
              transition: "height 600ms cubic-bezier(0.22, 1, 0.36, 1) 200ms",
            }}
          />
        )}
        {approach}
      </td>
      <td className="py-5 sm:py-6 px-6 mono text-[13px] sm:text-[14px] text-stone-700">
        {verification}
      </td>
      <td className="hidden sm:table-cell py-5 sm:py-6 px-6 text-[14px] text-stone-700">
        {trust}
      </td>
      <td className="py-5 sm:py-6 pl-6 pr-4 sm:pr-0 text-[13px] sm:text-[14px] text-stone-700">
        {scale}
      </td>
    </tr>
  );
}
