/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,jsx}", "./public/index.html"],
  theme: {
    // Mobile-first explicit breakpoints. xs catches very narrow phones
    // (older iPhone SE at 320px); the rest match Tailwind defaults.
    screens: {
      xs: "375px",
      sm: "640px",
      md: "768px",
      lg: "1024px",
      xl: "1280px",
      "2xl": "1536px",
    },
    extend: {
      colors: {
        // Pure achromatic. No cream. No blue. No accent.
        // OKLCH-defined for perceptual uniformity, then expressed as hex
        // for build-time stability.
        ink: "#141414", // oklch(15% 0 0)
        paper: "#FCFCFC", // oklch(99% 0 0) — barely off-white
        ash: "#F2F2F2", // oklch(95.5% 0 0) — for occasional section variation
        stone: {
          100: "#F2F2F2",
          200: "#E4E4E4",
          300: "#D1D1D1",
          400: "#B8B8B8",
          500: "#8C8C8C",
          600: "#6B6B6B",
          700: "#525252",
          800: "#3A3A3A",
          900: "#2A2A2A",
        },
      },
      fontFamily: {
        sans: [
          "Geist",
          "-apple-system",
          "BlinkMacSystemFont",
          "Segoe UI",
          "system-ui",
          "sans-serif",
        ],
        mono: [
          "Geist Mono",
          "ui-monospace",
          "SFMono-Regular",
          "Menlo",
          "Consolas",
          "monospace",
        ],
      },
      letterSpacing: {
        tightest: "-0.045em",
        snug: "-0.02em",
      },
      maxWidth: {
        "screen-narrow": "640px",
        "screen-reading": "720px",
        "screen-wide": "1280px",
      },
      fontSize: {
        // Display headlines — Geist sans, fluid sizing.
        // Floor at 320px viewport, ceiling at 1536px+.
        // Tuned so an iPhone SE displays them at the lower bound
        // without becoming a wall of text.
        "display-xl": [
          "clamp(2.75rem, 1rem + 8vw, 7.5rem)",
          { lineHeight: "1.0", letterSpacing: "-0.045em" },
        ],
        "display-lg": [
          "clamp(2.25rem, 1rem + 4vw, 4.5rem)",
          { lineHeight: "1.05", letterSpacing: "-0.035em" },
        ],
        "display-md": [
          "clamp(1.75rem, 1rem + 2.5vw, 3rem)",
          { lineHeight: "1.1", letterSpacing: "-0.03em" },
        ],
      },
      spacing: {
        // Safe-area-aware spacing tokens, used on sticky/fixed
        // elements that need to clear the iPhone notch + home indicator.
        "safe-top": "env(safe-area-inset-top)",
        "safe-bottom": "env(safe-area-inset-bottom)",
      },
    },
  },
  plugins: [],
};
