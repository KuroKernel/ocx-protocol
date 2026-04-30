import React, { useEffect, useState, useCallback } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import { Wordmark } from "./Logo";

const CONTACT_EMAIL = "hhaishwary@gmail.com";

const navItems = [
  { to: "/paper", label: "Whitepaper" },
  { to: "/spec", label: "Specification" },
];

/* Click-to-copy contact button. Replaces `mailto:` (which opens a
   webmail tab on systems without a default mail client). One click
   copies the address to the clipboard and briefly flips the label
   to "Copied · email" so the user knows what's now in their
   clipboard without trusting the silent action. */
function useCopyContact() {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(CONTACT_EMAIL);
    } catch {
      // clipboard may be denied (older browsers, no https) — fall
      // back to a hidden textarea + execCommand
      try {
        const ta = document.createElement("textarea");
        ta.value = CONTACT_EMAIL;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
      } catch {
        /* give up silently */
      }
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }, []);
  return { copied, copy };
}

/* ------------------------------------------------------------------
   Header
   ------------------------------------------------------------------
   • Sticky on every viewport — feels native, lets the wordmark and
     navigation stay reachable on long pages.
   • Hairline border-bottom only appears AFTER the user scrolls a
     few pixels (Linear / Apple style — flat at the top, defined
     once you commit to the page).
   • Mobile shows a hamburger that opens a full-width slide-down
     menu. The menu locks body scroll while open, dismisses on
     route change, and respects the safe-area inset for the notch.
------------------------------------------------------------------- */
const Header = () => {
  const [scrolled, setScrolled] = useState(false);
  const [open, setOpen] = useState(false);
  const location = useLocation();

  // Hairline appears after 4px of scroll — enough to feel intentional,
  // not enough to flicker on rubber-band overshoot.
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 4);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Close the mobile menu whenever the route changes.
  useEffect(() => {
    setOpen(false);
  }, [location.pathname, location.hash]);

  // Lock body scroll while the mobile menu is open.
  useEffect(() => {
    if (open) {
      const prev = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = prev;
      };
    }
  }, [open]);

  // Esc closes the menu (keyboard users).
  useEffect(() => {
    if (!open) return;
    const onKey = (e) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const toggle = useCallback(() => setOpen((o) => !o), []);

  return (
    <header
      className={[
        "sticky top-0 z-40 bg-paper/95 backdrop-blur-[6px]",
        "transition-[border-color] duration-200",
        scrolled ? "border-b border-stone-200" : "border-b border-transparent",
      ].join(" ")}
      style={{ paddingTop: "env(safe-area-inset-top)" }}
    >
      <div className="container-wide flex items-center justify-between h-14 sm:h-16">
        <Link
          to="/"
          aria-label="OCX home"
          className="flex items-center -ml-1 px-1 py-2 rounded-sm"
        >
          <Wordmark size="sm" />
        </Link>

        {/* Desktop nav */}
        <nav className="hidden md:flex items-center gap-8 lg:gap-10">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `text-[14px] tracking-snug transition-colors duration-150 ${
                  isActive ? "text-ink" : "text-stone-500 hover:text-ink"
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          {/* Contact — click copies the email to clipboard. No mailto. */}
          <ContactButton
            className="hidden sm:inline-flex text-[14px] font-medium text-ink hover:text-stone-600 transition-colors duration-150 px-2 py-2"
            idleLabel="Contact →"
          />

          {/* Hamburger — only on viewports below md */}
          <button
            type="button"
            onClick={toggle}
            aria-expanded={open}
            aria-controls="mobile-nav"
            aria-label={open ? "Close menu" : "Open menu"}
            className="md:hidden inline-flex items-center justify-center w-11 h-11 -mr-2 text-ink"
            style={{ touchAction: "manipulation" }}
          >
            <Hamburger open={open} />
          </button>
        </div>
      </div>

      {/* Mobile menu — slide-down panel beneath the header */}
      <MobileMenu open={open} onClose={() => setOpen(false)} />
    </header>
  );
};

/* Hamburger glyph — two hairlines that morph into an X.
   Pure CSS transform; respects prefers-reduced-motion via the
   global override in index.css. */
const Hamburger = ({ open }) => (
  <span className="relative block w-5 h-5" aria-hidden="true">
    <span
      className={[
        "absolute left-0 right-0 h-px bg-ink transition-transform duration-200 ease-out",
        open ? "top-1/2 rotate-45 -translate-y-1/2" : "top-[6px]",
      ].join(" ")}
    />
    <span
      className={[
        "absolute left-0 right-0 h-px bg-ink transition-transform duration-200 ease-out",
        open ? "top-1/2 -rotate-45 -translate-y-1/2" : "bottom-[6px]",
      ].join(" ")}
    />
  </span>
);

const MobileMenu = ({ open, onClose }) => {
  return (
    <div
      id="mobile-nav"
      className={[
        "md:hidden fixed inset-x-0 z-30 bg-paper",
        "border-b border-stone-200",
        "transition-[opacity,transform] duration-200 ease-out",
        open
          ? "opacity-100 translate-y-0 pointer-events-auto"
          : "opacity-0 -translate-y-2 pointer-events-none",
      ].join(" ")}
      style={{
        // Sits directly under the (sticky) header. 56px on phones,
        // 64px on small tablets where header height grows.
        top: "calc(3.5rem + env(safe-area-inset-top))",
        // Cover the rest of the viewport — feels like a sheet, not a chip.
        bottom: 0,
      }}
      aria-hidden={!open}
    >
      <nav className="container-wide pt-6 pb-10 flex flex-col">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onClose}
            className={({ isActive }) =>
              [
                "flex items-center justify-between py-4 border-b border-stone-200",
                "text-[18px] tracking-snug transition-colors duration-150",
                isActive ? "text-ink" : "text-stone-700",
              ].join(" ")
            }
          >
            <span>{item.label}</span>
            <span className="text-stone-400 text-[16px]">→</span>
          </NavLink>
        ))}
        <Link
          to="/account"
          onClick={onClose}
          className="flex items-center justify-between py-4 border-b border-stone-200 text-[18px] text-stone-700"
        >
          <span>Account</span>
          <span className="text-stone-400 text-[16px]">→</span>
        </Link>
        <ContactButton
          onClick={onClose}
          className="mt-8 inline-flex items-center justify-center px-6 py-4 border border-ink bg-ink text-paper text-[15px] font-medium"
          idleLabel="Contact →"
          dark
        />
        <a
          href="https://github.com/KuroKernel/ocx-protocol"
          className="mt-4 text-center text-stone-500 text-[14px] py-3"
        >
          GitHub ↗
        </a>
      </nav>
    </div>
  );
};

/* ------------------------------------------------------------------
   Footer
   ------------------------------------------------------------------
   Mobile: stacked, generous tap targets, copyright + email at the
   bottom. Desktop: original single-row layout.
------------------------------------------------------------------- */
const Footer = () => (
  <footer className="border-t border-stone-200 mt-24 sm:mt-32 lg:mt-56">
    <div
      className="container-wide py-10 sm:py-10 flex flex-col sm:flex-row sm:flex-wrap sm:items-baseline sm:justify-between gap-y-8 sm:gap-y-4 sm:gap-x-10"
      style={{ paddingBottom: "max(2.5rem, env(safe-area-inset-bottom))" }}
    >
      <div className="flex flex-col sm:flex-row sm:items-baseline gap-y-5 gap-x-10">
        <Wordmark size="sm" />
        <div className="flex flex-wrap items-baseline gap-x-8 gap-y-3">
          <Link to="/paper" className="link-mute text-[14px] sm:text-[13px]">Whitepaper</Link>
          <Link to="/spec" className="link-mute text-[14px] sm:text-[13px]">Spec</Link>
          <a
            href="https://github.com/KuroKernel/ocx-protocol"
            className="link-mute text-[14px] sm:text-[13px]"
          >
            GitHub
          </a>
        </div>
      </div>
      <div className="flex flex-col sm:flex-row sm:items-baseline gap-y-3 gap-x-8 text-stone-500 text-[13px]">
        <ContactButton
          className="link-mute break-all"
          idleLabel={CONTACT_EMAIL}
          copiedLabel={CONTACT_EMAIL + " · copied"}
        />
        <span>© 2026</span>
      </div>
    </div>
  </footer>
);

/* ------------------------------------------------------------------
   ContactButton — single shared component for everywhere we used to
   stick a `mailto:` link. Click copies CONTACT_EMAIL to clipboard
   and flashes the label briefly to confirm. No new tab, no mailto
   handler, no popup.
   - idleLabel:    text shown when not in the "just-copied" state
   - copiedLabel:  text shown for ~1.8s after a click (default:
                   "Copied · email")
   - dark:         flips the contrast of the success flash for use
                   on a dark/inked button
------------------------------------------------------------------- */
export function ContactButton({
  className = "",
  idleLabel = "Contact →",
  copiedLabel,
  dark = false,
  onClick,
}) {
  const { copied, copy } = useCopyContact();
  const handle = (e) => {
    copy();
    if (onClick) onClick(e);
  };
  const flash = copiedLabel || `Copied · ${CONTACT_EMAIL}`;
  return (
    <button
      type="button"
      onClick={handle}
      aria-live="polite"
      className={className}
      style={{ cursor: "pointer", appearance: "none" }}
      title="Click to copy email address"
    >
      {copied ? flash : idleLabel}
    </button>
  );
}

export default function Layout({ children }) {
  return (
    <div className="min-h-screen flex flex-col bg-paper text-ink">
      <Header />
      <main className="flex-1">{children}</main>
      <Footer />
    </div>
  );
}
