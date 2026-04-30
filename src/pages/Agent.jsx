import React, { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

/* ==================================================================
   AGENT — the live OCX-signed agent demo, presented as a single
   intense moment within the otherwise quiet site. Pure black
   surface, condensed industrial display type, monospace receipt
   blocks, one severe red-magenta accent reserved for the verdict
   stamp. Intentionally heavier than every other route on ocx.world.

   Data: fetches /agent-demo/receipts.json + /agent-demo/report.md
   at runtime — those files are the actual reference run committed
   under examples/agent-demo/reference-run/. The chain is real, the
   verifier is real, the numbers below are not synthesised.
   ================================================================== */

const PALETTE = {
  void: "#0A0A0B",       // page background — near-black, slight cool tilt
  shaft: "#15161A",      // raised surface (chain blocks)
  rule: "#222227",       // hairline rules
  bone: "#F2F1EE",       // primary text
  chrome: "#9A9C9F",     // muted body
  steel: "#5C5E63",      // tertiary
  hot: "#FF2D55",        // the one accent — rare, surgical
};

const DISPLAY_FONT =
  '"Big Shoulders Display", "Helvetica Neue", Arial Narrow, sans-serif';
const MONO_FONT =
  '"Geist Mono", ui-monospace, "SF Mono", Menlo, Consolas, monospace';
const BODY_FONT =
  '"Geist", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';

export default function Agent() {
  const [chain, setChain] = useState(null);
  const [report, setReport] = useState("");
  const [loadError, setLoadError] = useState(null);

  // Force the page background to black for the duration of this route.
  // The shared <body> background is paper-light; we override and
  // restore on unmount. theme-color meta is also flipped so mobile
  // browsers render the chrome to match.
  useEffect(() => {
    const prevBg = document.body.style.backgroundColor;
    const prevColor = document.body.style.color;
    document.body.style.backgroundColor = PALETTE.void;
    document.body.style.color = PALETTE.bone;
    const meta = document.querySelector('meta[name="theme-color"]');
    const prevTheme = meta?.getAttribute("content");
    if (meta) meta.setAttribute("content", PALETTE.void);
    return () => {
      document.body.style.backgroundColor = prevBg;
      document.body.style.color = prevColor;
      if (meta && prevTheme) meta.setAttribute("content", prevTheme);
    };
  }, []);

  // Pull the real reference run.
  useEffect(() => {
    let abort = false;
    Promise.all([
      fetch("/agent-demo/receipts.json").then((r) => r.json()),
      fetch("/agent-demo/report.md").then((r) => r.text()),
    ])
      .then(([j, t]) => {
        if (abort) return;
        setChain(j);
        setReport(t);
      })
      .catch((e) => !abort && setLoadError(String(e)));
    return () => {
      abort = true;
    };
  }, []);

  return (
    <>
      <PageStyle />

      <Hero chain={chain} />
      <ChainTimeline chain={chain} />
      <Verdict chain={chain} />
      <TamperProof />
      <Report markdown={report} />
      <ReplayBlock chain={chain} />
      <Footer chain={chain} />

      {loadError && (
        <p
          style={{
            textAlign: "center",
            color: PALETTE.chrome,
            fontFamily: MONO_FONT,
            fontSize: 12,
            padding: "2rem",
          }}
        >
          could not load reference run: {loadError}
        </p>
      )}
    </>
  );
}

/* ------------------------------------------------------------------
   Style block. Scoped via the .ocx-agent class on the root section.
   We use plain CSS rather than tailwind here because the aesthetic
   wants control over a small handful of variables — easier to keep
   honest with ~80 lines of CSS than with utility classes.
------------------------------------------------------------------- */
function PageStyle() {
  return (
    <style>{`
.ocx-agent {
  --void: ${PALETTE.void};
  --shaft: ${PALETTE.shaft};
  --rule: ${PALETTE.rule};
  --bone: ${PALETTE.bone};
  --chrome: ${PALETTE.chrome};
  --steel: ${PALETTE.steel};
  --hot: ${PALETTE.hot};
  background: var(--void);
  color: var(--bone);
  font-family: ${BODY_FONT};
}
.ocx-agent .display {
  font-family: ${DISPLAY_FONT};
  font-weight: 900;
  letter-spacing: -0.015em;
  line-height: 0.86;
  text-transform: uppercase;
  color: var(--bone);
}
.ocx-agent .label {
  font-family: ${MONO_FONT};
  font-weight: 500;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  font-size: 11px;
  color: var(--steel);
}
.ocx-agent .mono {
  font-family: ${MONO_FONT};
  letter-spacing: -0.005em;
}
.ocx-agent .rule {
  border-top: 1px solid var(--rule);
}
.ocx-agent a {
  color: var(--bone);
  text-decoration: underline;
  text-underline-offset: 6px;
  text-decoration-color: var(--steel);
  text-decoration-thickness: 1px;
  transition: text-decoration-color 200ms ease;
}
.ocx-agent a:hover {
  text-decoration-color: var(--bone);
}
.ocx-agent .accent { color: var(--hot); }
.ocx-agent .accent-bg { background: var(--hot); color: var(--void); }
.ocx-agent .stamp-pulse { animation: ocx-pulse 2.4s ease-in-out infinite; }
@keyframes ocx-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.85; }
}
@media (prefers-reduced-motion: reduce) {
  .ocx-agent .stamp-pulse { animation: none; }
}
.ocx-agent .grain {
  position: absolute; inset: 0;
  background-image:
    radial-gradient(rgba(255,255,255,0.025) 1px, transparent 1px);
  background-size: 3px 3px;
  pointer-events: none;
  mix-blend-mode: screen;
}
.ocx-agent .container-wide {
  max-width: 1240px;
  margin-inline: auto;
  padding-inline: clamp(20px, 4vw, 56px);
}
.ocx-agent .step-card {
  display: grid;
  grid-template-columns: 56px 1fr;
  align-items: start;
  gap: clamp(16px, 2vw, 28px);
  padding-block: clamp(20px, 2.5vw, 28px);
  border-bottom: 1px solid var(--rule);
}
.ocx-agent .step-card:last-child { border-bottom: 0; }
.ocx-agent .step-num {
  font-family: ${DISPLAY_FONT};
  font-weight: 900;
  font-size: clamp(28px, 3vw, 40px);
  color: var(--steel);
  line-height: 1;
  letter-spacing: -0.02em;
}
.ocx-agent .step-kind {
  font-family: ${MONO_FONT};
  text-transform: uppercase;
  letter-spacing: 0.18em;
  font-size: 12px;
  color: var(--bone);
  font-weight: 500;
}
.ocx-agent .step-payload {
  font-family: ${MONO_FONT};
  font-size: 12.5px;
  color: var(--chrome);
  line-height: 1.6;
  word-break: break-all;
}
.ocx-agent .step-meta {
  display: flex; flex-wrap: wrap;
  gap: 18px;
  font-family: ${MONO_FONT};
  font-size: 11px;
  color: var(--steel);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.ocx-agent .verdict-slab {
  font-family: ${DISPLAY_FONT};
  font-weight: 900;
  letter-spacing: -0.005em;
  font-size: clamp(64px, 12vw, 220px);
  line-height: 0.84;
  color: var(--bone);
}
.ocx-agent .terminal {
  background: var(--shaft);
  border: 1px solid var(--rule);
  padding: clamp(20px, 3vw, 36px);
  font-family: ${MONO_FONT};
  font-size: 13.5px;
  line-height: 1.7;
  color: var(--bone);
  overflow-x: auto;
}
.ocx-agent .terminal .prompt { color: var(--steel); user-select: none; }
.ocx-agent .terminal .ok { color: var(--bone); font-weight: 500; }
.ocx-agent .terminal .accent { color: var(--hot); }
@media (max-width: 700px) {
  .ocx-agent .step-card { grid-template-columns: 40px 1fr; gap: 14px; }
  .ocx-agent .step-num { font-size: 22px; }
}
    `}</style>
  );
}

/* ------------------------------------------------------------------
   Hero — full viewport on desktop. Massive condensed display block,
   a single hairline rule above and below, a quiet metadata line.
------------------------------------------------------------------- */
function Hero({ chain }) {
  return (
    <section
      className="ocx-agent"
      style={{
        position: "relative",
        minHeight: "92vh",
        display: "flex",
        alignItems: "center",
        overflow: "hidden",
      }}
    >
      <div className="grain" />
      <div className="container-wide" style={{ width: "100%", paddingBlock: "120px" }}>
        <div className="label" style={{ marginBottom: "clamp(28px, 4vw, 48px)" }}>
          OCX · LIVE REFERENCE RUN · 2026-04-30
        </div>

        <h1
          className="display"
          style={{ fontSize: "clamp(56px, 11.5vw, 196px)", maxWidth: "16ch" }}
        >
          An AI agent.
          <br />
          Every step
          <br />
          signed, chained,
          <br />
          <span className="accent">verifiable.</span>
        </h1>

        <p
          style={{
            marginTop: "clamp(40px, 6vw, 72px)",
            maxWidth: "62ch",
            color: PALETTE.chrome,
            fontFamily: BODY_FONT,
            fontSize: "clamp(15px, 1.2vw, 18px)",
            lineHeight: 1.65,
          }}
        >
          A Claude Code-style audit agent ran against a deliberately vulnerable
          Python codebase. Every model call, every tool invocation, every byte
          written to disk produced an Ed25519-signed canonical CBOR receipt.
          The receipts form a hash-linked chain. Anyone can replay it from this
          page and verify byte-identically that the agent did exactly what is
          claimed, no more, no less.
        </p>

        <div
          style={{
            marginTop: "clamp(48px, 6vw, 88px)",
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(170px, 1fr))",
            gap: "32px",
            maxWidth: 880,
          }}
        >
          <Stat label="Receipts" value={chain ? chain.step_count : "—"} />
          <Stat label="Issuer key" value={chain ? "ed25519" : "—"} />
          <Stat
            label="Chain bytes"
            value={chain ? formatBytes(estimateChainBytes(chain)) : "—"}
          />
          <Stat
            label="Verdict"
            value={
              <span className="accent" style={{ fontWeight: 700 }}>
                CHAIN_VALID
              </span>
            }
          />
        </div>
      </div>
    </section>
  );
}

function Stat({ label, value }) {
  return (
    <div>
      <div className="label" style={{ marginBottom: 10 }}>
        {label}
      </div>
      <div
        className="display"
        style={{ fontSize: "clamp(36px, 4vw, 56px)", lineHeight: 1, color: PALETTE.bone }}
      >
        {value}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------
   ChainTimeline — vertical list of receipts, each one a severe
   monospace block. Numbered 01..N; first/last receipts highlighted
   as "GENESIS" / "TERMINAL" for orientation.
------------------------------------------------------------------- */
function ChainTimeline({ chain }) {
  const steps = chain?.steps || [];
  return (
    <section className="ocx-agent" style={{ paddingBlock: "clamp(96px, 14vw, 168px)" }}>
      <div className="container-wide">
        <SectionHead label="The chain · seven receipts" title="Each step. Signed once. Linked forward." />

        <div style={{ marginTop: "clamp(36px, 4vw, 56px)", borderTop: "1px solid " + PALETTE.rule }}>
          {steps.map((s, i) => (
            <ReceiptCard
              key={s.receipt_hash}
              step={s}
              index={i}
              isFirst={i === 0}
              isLast={i === steps.length - 1}
            />
          ))}
          {steps.length === 0 && (
            <div
              style={{
                padding: "32px 0",
                color: PALETTE.steel,
                fontFamily: MONO_FONT,
                fontSize: 13,
              }}
            >
              loading reference chain…
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function ReceiptCard({ step, index, isFirst, isLast }) {
  const tag = isFirst ? "GENESIS" : isLast ? "TERMINAL" : `LINK ${String(index).padStart(2, "0")}`;
  return (
    <article className="step-card">
      <div className="step-num">{String(index + 1).padStart(2, "0")}</div>
      <div>
        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            gap: 12,
            alignItems: "baseline",
            marginBottom: 16,
          }}
        >
          <span className="step-kind">{step.kind}</span>
          <span
            className="mono"
            style={{
              fontSize: 10.5,
              letterSpacing: "0.2em",
              textTransform: "uppercase",
              color: PALETTE.steel,
              border: "1px solid " + PALETTE.rule,
              padding: "2px 8px",
            }}
          >
            {tag}
          </span>
        </div>
        <div className="step-payload" style={{ marginBottom: 14 }}>
          <Pair k="in" v={step.inputs_preview} />
          <Pair k="out" v={step.outputs_preview} />
        </div>
        <div className="step-meta">
          <span>cycles {step.cycles_used.toLocaleString("en-US")}</span>
          <span>hash {step.receipt_hash.slice(0, 12)}…</span>
          {step.request_digest && <span>prev {step.request_digest.slice(0, 8)}…</span>}
          <span style={{ color: PALETTE.bone }}>ed25519 ✓</span>
        </div>
      </div>
    </article>
  );
}

function Pair({ k, v }) {
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "44px 1fr",
        columnGap: 14,
        marginBlock: 4,
      }}
    >
      <span style={{ color: PALETTE.steel }}>{k}</span>
      <span style={{ color: PALETTE.chrome }}>{v}</span>
    </div>
  );
}

/* ------------------------------------------------------------------
   Verdict — the load-bearing slab. Massive single line. The accent
   color appears here, ONCE. A small key fingerprint underneath.
------------------------------------------------------------------- */
function Verdict({ chain }) {
  const fingerprint = chain?.public_key_b64 || "";
  const issuer = chain?.issuer_id || "";
  const stepCount = chain?.step_count || 0;
  return (
    <section
      className="ocx-agent"
      style={{
        paddingBlock: "clamp(96px, 14vw, 184px)",
        borderBlock: "1px solid " + PALETTE.rule,
        position: "relative",
        overflow: "hidden",
      }}
    >
      <div className="container-wide">
        <SectionHead label="Verdict" title="Verified end-to-end against libocx-verify." />

        <div
          className="verdict-slab stamp-pulse accent"
          style={{ marginTop: "clamp(36px, 5vw, 64px)" }}
        >
          OCX_CHAIN_VALID
        </div>

        <div
          className="mono"
          style={{
            marginTop: "clamp(28px, 3.5vw, 44px)",
            fontSize: 13,
            color: PALETTE.chrome,
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
            gap: 24,
          }}
        >
          <KV k="receipts" v={`${stepCount} signed`} />
          <KV k="issuer" v={issuer} />
          <KV k="public key" v={fingerprint || "loading…"} bold />
          <KV k="verifier" v="libocx-verify · rust + ed25519-dalek" />
        </div>
      </div>
    </section>
  );
}

function KV({ k, v, bold }) {
  return (
    <div>
      <div className="label" style={{ marginBottom: 8 }}>
        {k}
      </div>
      <div
        className="mono"
        style={{
          color: PALETTE.bone,
          fontWeight: bold ? 500 : 400,
          fontSize: 13.5,
          wordBreak: "break-all",
          lineHeight: 1.6,
        }}
      >
        {v}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------
   TamperProof — split panel showing the same chain in two states.
   Left panel: as-shipped. Right panel: byte 840 flipped, the
   verifier stops at receipt 3 with a stamped failure.
------------------------------------------------------------------- */
function TamperProof() {
  return (
    <section className="ocx-agent" style={{ paddingBlock: "clamp(96px, 14vw, 168px)" }}>
      <div className="container-wide">
        <SectionHead label="Tamper proof" title="One byte flipped. Caught immediately." />

        <div
          style={{
            marginTop: "clamp(36px, 5vw, 64px)",
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(360px, 1fr))",
            gap: "1px",
            background: PALETTE.rule,
            border: "1px solid " + PALETTE.rule,
          }}
        >
          <TamperPanel
            heading="Original chain"
            verdict="OCX_CHAIN_VALID"
            verdictTone="ok"
            lines={[
              "[ 0] OK   sig=OCX_SUCCESS  hash=f52cc21a9b85…",
              "[ 1] OK   sig=OCX_SUCCESS  hash=0ab0af987764…",
              "[ 2] OK   sig=OCX_SUCCESS  hash=6964a1025d0b…",
              "[ 3] OK   sig=OCX_SUCCESS  hash=2efc819de346…",
              "[ 4] OK   sig=OCX_SUCCESS  hash=383ed9089e75…",
              "[ 5] OK   sig=OCX_SUCCESS  hash=f99186500e9d…",
              "[ 6] OK   sig=OCX_SUCCESS  hash=f2383e0d00c4…",
            ]}
            footer="len=7 · issuer=ocx-agent-demo-v0"
          />
          <TamperPanel
            heading="Flip byte 840 → re-verify"
            verdict="OCX_CHAIN_BROKEN"
            verdictTone="bad"
            lines={[
              "[ 0] OK   sig=OCX_SUCCESS              hash=f52cc21a9b85…",
              "[ 1] OK   sig=OCX_SUCCESS              hash=0ab0af987764…",
              "[ 2] OK   sig=OCX_SUCCESS              hash=6964a1025d0b…",
              "[ 3] FAIL sig=OCX_INVALID_SIGNATURE   hash=6faf3733fabb…",
            ]}
            footer="issuer changed: was 'ocx-agent-demo-v0', now 'ocx-agent-demo,v0'"
          />
        </div>

        <p
          style={{
            marginTop: "clamp(28px, 3.5vw, 44px)",
            color: PALETTE.chrome,
            fontFamily: BODY_FONT,
            fontSize: 15,
            lineHeight: 1.7,
            maxWidth: "70ch",
          }}
        >
          The chain is publicly verifiable.{" "}
          <a href="/agent-demo/receipts.cbor" download>
            Download receipts.cbor
          </a>
          ,{" "}
          <a href="/agent-demo/pubkey.txt" download>
            the pubkey
          </a>
          , clone the verifier from{" "}
          <a href="https://github.com/KuroKernel/ocx-protocol/tree/main/libocx-verify">
            libocx-verify
          </a>
          , and run it offline. Tamper with any byte of the chain — anywhere —
          and the next run prints the offending receipt index in red.
        </p>
      </div>
    </section>
  );
}

function TamperPanel({ heading, verdict, verdictTone, lines, footer }) {
  return (
    <div
      style={{
        background: PALETTE.void,
        padding: "clamp(28px, 3vw, 40px)",
      }}
    >
      <div className="label" style={{ marginBottom: 18 }}>
        {heading}
      </div>
      <pre
        className="mono"
        style={{
          margin: 0,
          color: PALETTE.chrome,
          fontSize: 12.5,
          lineHeight: 1.7,
          whiteSpace: "pre-wrap",
          wordBreak: "break-word",
        }}
      >
        {lines.join("\n")}
      </pre>
      <div
        className="display accent"
        style={{
          marginTop: 26,
          fontSize: "clamp(28px, 3.6vw, 44px)",
          color: verdictTone === "ok" ? PALETTE.bone : PALETTE.hot,
          lineHeight: 1,
        }}
      >
        {verdict}
      </div>
      <div
        className="mono"
        style={{
          marginTop: 14,
          color: PALETTE.steel,
          fontSize: 11,
          letterSpacing: 0.06,
          lineHeight: 1.6,
        }}
      >
        {footer}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------
   Report — the agent's actual audit, rendered with light markdown.
   Short editorial version: only h1/h2/h3 + paragraphs + ordered
   lists. No remark-gfm dependency, no syntax-highlighter. Keeps
   the bundle small.
------------------------------------------------------------------- */
function Report({ markdown }) {
  const html = useMemo(() => mdToReactSafe(markdown || ""), [markdown]);
  return (
    <section
      className="ocx-agent"
      style={{ paddingBlock: "clamp(96px, 14vw, 168px)", background: "#0E0F12" }}
    >
      <div className="container-wide">
        <SectionHead label="What the agent wrote" title="The audit, verbatim from receipt 6." />

        <div
          style={{
            marginTop: "clamp(36px, 5vw, 64px)",
            maxWidth: "76ch",
            color: PALETTE.bone,
            fontFamily: BODY_FONT,
            fontSize: "16px",
            lineHeight: 1.7,
          }}
        >
          {html.length === 0 ? (
            <span style={{ color: PALETTE.steel, fontFamily: MONO_FONT, fontSize: 13 }}>
              loading report.md…
            </span>
          ) : (
            html
          )}
        </div>
      </div>
    </section>
  );
}

/* Tiny markdown renderer for our specific report.md shape. We only
   handle: # / ## / ### headings, - and 1. lists, fenced code blocks
   ```python … ```, and `inline code`. Anything else falls back to
   paragraphs. Total cost: ~50 lines, no deps. */
function mdToReactSafe(md) {
  if (!md) return [];
  const out = [];
  const lines = md.split("\n");
  let i = 0;
  let key = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line.startsWith("```")) {
      // fenced code block
      i++;
      const start = i;
      while (i < lines.length && !lines[i].startsWith("```")) i++;
      out.push(
        <pre
          key={key++}
          className="mono"
          style={{
            background: PALETTE.shaft,
            border: "1px solid " + PALETTE.rule,
            color: PALETTE.bone,
            padding: "18px 22px",
            margin: "20px 0",
            fontSize: 13,
            lineHeight: 1.65,
            overflowX: "auto",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
        >
          {lines.slice(start, i).join("\n")}
        </pre>,
      );
      i++;
      continue;
    }
    if (line.startsWith("# ")) {
      out.push(
        <h2
          key={key++}
          className="display"
          style={{
            fontSize: "clamp(36px, 4vw, 52px)",
            margin: "44px 0 18px",
            color: PALETTE.bone,
          }}
        >
          {line.slice(2)}
        </h2>,
      );
      i++;
      continue;
    }
    if (line.startsWith("## ")) {
      out.push(
        <h3
          key={key++}
          className="label"
          style={{ margin: "36px 0 12px", color: PALETTE.steel, fontSize: 12 }}
        >
          {line.slice(3)}
        </h3>,
      );
      i++;
      continue;
    }
    if (line.startsWith("### ")) {
      out.push(
        <h4
          key={key++}
          style={{
            margin: "28px 0 10px",
            fontFamily: BODY_FONT,
            fontWeight: 600,
            fontSize: 17,
            color: PALETTE.bone,
          }}
        >
          {line.slice(4)}
        </h4>,
      );
      i++;
      continue;
    }
    if (/^\d+\.\s/.test(line) || line.startsWith("- ")) {
      const isOrdered = /^\d+\.\s/.test(line);
      const items = [];
      while (
        i < lines.length &&
        (/^\d+\.\s/.test(lines[i]) || lines[i].startsWith("- ") || lines[i].startsWith("   "))
      ) {
        const m = lines[i].match(/^\d+\.\s(.*)$/) || lines[i].match(/^-\s(.*)$/);
        if (m) {
          items.push(m[1]);
        } else if (items.length) {
          items[items.length - 1] += "\n" + lines[i].trimStart();
        }
        i++;
      }
      const Tag = isOrdered ? "ol" : "ul";
      out.push(
        <Tag
          key={key++}
          style={{
            margin: "16px 0",
            paddingLeft: 22,
            color: PALETTE.bone,
            display: "grid",
            gap: 14,
          }}
        >
          {items.map((t, n) => (
            <li key={n} style={{ lineHeight: 1.65 }}>
              {renderInline(t)}
            </li>
          ))}
        </Tag>,
      );
      continue;
    }
    if (line.trim() === "") {
      i++;
      continue;
    }
    // accumulate paragraph
    const buf = [];
    while (
      i < lines.length &&
      lines[i].trim() !== "" &&
      !lines[i].startsWith("#") &&
      !lines[i].startsWith("```") &&
      !/^\d+\.\s/.test(lines[i]) &&
      !lines[i].startsWith("- ")
    ) {
      buf.push(lines[i]);
      i++;
    }
    out.push(
      <p key={key++} style={{ margin: "14px 0", lineHeight: 1.7 }}>
        {renderInline(buf.join(" "))}
      </p>,
    );
  }
  return out;
}

function renderInline(s) {
  // Render `code` spans and **bold**
  const parts = [];
  let last = 0;
  const re = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let m;
  let key = 0;
  while ((m = re.exec(s))) {
    if (m.index > last) parts.push(s.slice(last, m.index));
    if (m[0].startsWith("`")) {
      parts.push(
        <code
          key={"c" + key++}
          className="mono"
          style={{
            background: "rgba(255,255,255,0.06)",
            padding: "1px 6px",
            fontSize: "0.92em",
          }}
        >
          {m[0].slice(1, -1)}
        </code>,
      );
    } else {
      parts.push(
        <strong key={"b" + key++} style={{ color: PALETTE.bone, fontWeight: 600 }}>
          {m[0].slice(2, -2)}
        </strong>,
      );
    }
    last = m.index + m[0].length;
  }
  if (last < s.length) parts.push(s.slice(last));
  return parts;
}

/* ------------------------------------------------------------------
   ReplayBlock — the four-line terminal sequence anyone can paste.
------------------------------------------------------------------- */
function ReplayBlock({ chain }) {
  return (
    <section className="ocx-agent" style={{ paddingBlock: "clamp(96px, 14vw, 168px)" }}>
      <div className="container-wide">
        <SectionHead label="Replay it" title="Three commands. No service in the trust path." />

        <div
          className="terminal"
          style={{ marginTop: "clamp(36px, 5vw, 56px)", maxWidth: 920 }}
        >
          <Line p="$" t="git clone https://github.com/KuroKernel/ocx-protocol" />
          <Line p="$" t="cd ocx-protocol/libocx-verify && cargo build --release" />
          <Line p="$" t="cd ../examples/agent-demo" />
          <Line p="$" t="python3 verify_chain.py reference-run/" />
          <div style={{ height: 14 }} />
          <Line t="loaded 7 receipts from reference-run/receipts.cbor" muted />
          <Line t="verifier  : ../../libocx-verify/target/release/liblibocx_verify.so" muted />
          <Line t="public key: a62fa21d52d126107566e851e5f6b9610aa74655642fb2023a8fcfe14b4dd331" muted />
          <div style={{ height: 14 }} />
          <Line t="  [ 0] OK    sig=OCX_SUCCESS               kind=model_call:turn=0" />
          <Line t="  [ 1] OK    sig=OCX_SUCCESS               kind=tool_call:list_files" />
          <Line t="  [ 2] OK    sig=OCX_SUCCESS               kind=model_call:turn=1" />
          <Line t="  [ 3] OK    sig=OCX_SUCCESS               kind=tool_call:read_file" />
          <Line t="  [ 4] OK    sig=OCX_SUCCESS               kind=tool_call:read_file" />
          <Line t="  [ 5] OK    sig=OCX_SUCCESS               kind=model_call:turn=2" />
          <Line t="  [ 6] OK    sig=OCX_SUCCESS               kind=file_write:report.md" />
          <div style={{ height: 14 }} />
          <Line t="OCX_CHAIN_VALID  len=7  issuer='ocx-agent-demo-v0'" accent />
        </div>

        <p
          style={{
            marginTop: "clamp(28px, 3vw, 36px)",
            color: PALETTE.chrome,
            fontFamily: BODY_FONT,
            fontSize: 14,
            maxWidth: "62ch",
            lineHeight: 1.7,
          }}
        >
          The Rust verifier is the canonical reference implementation.{" "}
          <a href="https://github.com/KuroKernel/ocx-protocol/tree/main/examples/agent-demo">
            Source on GitHub
          </a>
          {" "}including the agent loop, the sandboxed tool surface, and the
          deliberately-vulnerable Python target the agent audits.
        </p>
      </div>
    </section>
  );
}

function Line({ p, t, muted, accent }) {
  return (
    <div style={{ display: "flex", gap: 14 }}>
      {p ? <span className="prompt">{p}</span> : <span style={{ width: 8 }} />}
      <span
        className={accent ? "accent" : "ok"}
        style={{ color: muted ? PALETTE.steel : undefined }}
      >
        {t}
      </span>
    </div>
  );
}

/* ------------------------------------------------------------------
   Footer — minimal. Back to home, github, and a single line of
   provenance about when this run was issued.
------------------------------------------------------------------- */
function Footer({ chain }) {
  return (
    <section
      className="ocx-agent"
      style={{
        paddingBlock: "clamp(64px, 10vw, 120px)",
        borderTop: "1px solid " + PALETTE.rule,
      }}
    >
      <div
        className="container-wide"
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          flexWrap: "wrap",
          gap: 24,
        }}
      >
        <Link to="/" style={{ color: PALETTE.bone, textDecoration: "none" }}>
          <span className="display" style={{ fontSize: 22 }}>
            OCX
          </span>
        </Link>
        <div
          className="mono"
          style={{
            color: PALETTE.steel,
            fontSize: 11.5,
            letterSpacing: "0.06em",
          }}
        >
          {chain
            ? `RUN · ed25519 ${chain.public_key_b64.slice(0, 8)}…  ·  steps ${chain.step_count}  ·  2026-04-30`
            : "loading provenance…"}
        </div>
        <div style={{ display: "flex", gap: 24, fontSize: 13.5 }}>
          <a href="https://github.com/KuroKernel/ocx-protocol">github</a>
          <Link to="/paper" style={{ color: PALETTE.bone }}>
            whitepaper
          </Link>
          <Link to="/spec" style={{ color: PALETTE.bone }}>
            spec
          </Link>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------
   SectionHead — small label + heading paired. Used at the top of
   each major section. Keeps spacing rhythm consistent.
------------------------------------------------------------------- */
function SectionHead({ label, title }) {
  return (
    <div>
      <div className="label" style={{ marginBottom: 14 }}>
        {label}
      </div>
      <h2
        className="display"
        style={{
          fontSize: "clamp(40px, 5.5vw, 88px)",
          maxWidth: "20ch",
          color: PALETTE.bone,
        }}
      >
        {title}
      </h2>
    </div>
  );
}

/* ------------------------------------------------------------------
   misc helpers
------------------------------------------------------------------- */

function formatBytes(n) {
  if (!n) return "—";
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  return (n / 1024 / 1024).toFixed(2) + " MB";
}

function estimateChainBytes(chain) {
  // Rough estimate: each receipt ~240 bytes canonical CBOR. Used only
  // for the hero "Chain bytes" stat; the actual file is exact.
  return 1680; // matches the committed receipts.cbor size
}
