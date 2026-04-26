import React, { useEffect, useRef, useState } from "react";

/**
 * Embeds a self-contained diagram HTML file via iframe.
 *
 * Mobile-first behaviour
 * ----------------------
 * The underlying SVG diagrams have a min-width of ~960–1080px so they
 * remain readable. On phone-sized viewports we let the iframe scroll
 * horizontally (so the user can pan), reduce vertical real estate so
 * the page doesn't become a wall, and surface a clear "open in a new
 * tab" affordance for users who want to dig in.
 *
 * Props
 * -----
 * src    — public URL of the diagram HTML
 * ratio  — CSS aspect-ratio for desktop sizing (e.g. "1240 / 920")
 * minH   — desktop min-height in pixels
 * title  — accessible title; shown as caption hint
 * label  — short eyebrow shown above the iframe (mobile only)
 * wide   — when true, lifts iframe out of the standard container
 *          width to use a 1600px max
 */
export default function Diagram({
  src,
  ratio,
  minH = 720,
  title,
  label,
  wide = true,
}) {
  const [loaded, setLoaded] = useState(false);
  const [hintVisible, setHintVisible] = useState(true);
  const wrapRef = useRef(null);

  // Hide the "scroll →" hint as soon as the user scrolls horizontally.
  useEffect(() => {
    const wrap = wrapRef.current;
    if (!wrap) return;
    const onScroll = () => {
      if (wrap.scrollLeft > 4) setHintVisible(false);
    };
    wrap.addEventListener("scroll", onScroll, { passive: true });
    return () => wrap.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <div className={wide ? "mx-auto max-w-[1600px] px-4 sm:px-6 lg:px-8" : ""}>
      {label && (
        <p className="md:hidden mb-3 text-[11px] font-medium uppercase tracking-[0.22em] text-stone-500">
          {label}
        </p>
      )}

      <figure className="relative">
        {/* The actual scroll container.
            • Mobile: capped at ~62dvh, scrolls horizontally inside the
              wrapper so the user can pan across the SVG.
            • Desktop: aspect-ratio + minH governs height. */}
        <div
          ref={wrapRef}
          className={[
            "relative bg-[#f5f5f5] border border-stone-200 overflow-x-auto overflow-y-hidden scroll-touch",
            "md:overflow-hidden",
          ].join(" ")}
          style={{
            // Mobile sizing — short and panning. Desktop sizing comes
            // from the inline media-query overrides below.
            height: undefined,
          }}
        >
          {/* Mobile-only sizing: bounded height with horizontal scroll. */}
          <div className="md:hidden">
            <iframe
              src={src}
              title={title}
              loading="lazy"
              onLoad={() => setLoaded(true)}
              className="block bg-[#f5f5f5]"
              style={{
                // Keep the iframe at its content's intrinsic width so the
                // wrapper can scroll across it. Height is bounded so the
                // mobile reader isn't trapped in a giant region.
                width: "1080px",
                height: "min(62dvh, 540px)",
                border: 0,
              }}
            />
          </div>

          {/* Desktop sizing — preserves the original behaviour. */}
          <div className="hidden md:block">
            <iframe
              src={src}
              title={title}
              loading="lazy"
              scrolling="no"
              onLoad={() => setLoaded(true)}
              className="block w-full bg-[#f5f5f5]"
              style={{
                aspectRatio: ratio,
                minHeight: `${minH}px`,
                border: 0,
              }}
            />
          </div>

          {/* Skeleton state — sits behind the iframe until it fires onLoad.
              Subtle pulse, no spinning glyphs. */}
          {!loaded && (
            <div
              aria-hidden="true"
              className="absolute inset-0 pointer-events-none"
              style={{
                background:
                  "linear-gradient(90deg, #f5f5f5 0%, #ececec 50%, #f5f5f5 100%)",
                backgroundSize: "200% 100%",
                animation: "ocx-skeleton 1.6s ease-in-out infinite",
              }}
            />
          )}

          {/* Mobile-only: scroll hint badge — fades away the moment the
              user pans. Sits over the diagram, top-left, hairline border. */}
          <div
            className={[
              "md:hidden absolute top-3 left-3 px-2.5 py-1.5 bg-paper/90 backdrop-blur-sm",
              "border border-stone-300 text-[10px] font-medium uppercase tracking-[0.18em] text-stone-600",
              "transition-opacity duration-300",
              hintVisible && loaded ? "opacity-100" : "opacity-0",
            ].join(" ")}
          >
            ← Drag to pan →
          </div>

          {/* Open-in-new-tab affordance — small, top-right. Always there. */}
          <a
            href={src}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`Open ${title || "diagram"} in a new tab`}
            className={[
              "absolute top-3 right-3 inline-flex items-center gap-1.5",
              "px-2.5 py-1.5 bg-paper/90 backdrop-blur-sm border border-stone-300",
              "text-[11px] font-medium text-ink",
              "hover:bg-paper hover:border-ink transition-colors duration-150",
            ].join(" ")}
            style={{ touchAction: "manipulation" }}
          >
            Open <span aria-hidden="true">↗</span>
          </a>
        </div>

        {/* Caption — visible on every viewport. Quiet. */}
        {title && (
          <figcaption className="mt-3 text-[12px] text-stone-500 mono">
            {title}
          </figcaption>
        )}
      </figure>

      {/* Skeleton keyframes — kept inline so a single component import
          carries everything; the entire site has no other CSS modules. */}
      <style>{`
        @keyframes ocx-skeleton {
          0%   { background-position: 100% 0; }
          100% { background-position: -100% 0; }
        }
        @media (prefers-reduced-motion: reduce) {
          [class*="ocx-skeleton"], [style*="ocx-skeleton"] {
            animation: none !important;
            background: #f5f5f5 !important;
          }
        }
      `}</style>
    </div>
  );
}
