import React from "react";

// Pure typographic wordmark. No symbol. Per design context: the wordmark IS the brand.
// Letter-spacing tuned for Geist Medium at small sizes; tightens further at display sizes.

export const Wordmark = ({ size = "sm", as: Tag = "span" }) => {
  const sizes = {
    sm: "text-[15px] tracking-[-0.04em]",
    md: "text-2xl tracking-[-0.045em]",
    lg: "text-4xl tracking-[-0.05em]",
    xl: "text-6xl tracking-[-0.055em]",
  };
  return (
    <Tag className={`wordmark ${sizes[size]}`}>OCX</Tag>
  );
};

export default Wordmark;
