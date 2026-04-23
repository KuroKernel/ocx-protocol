# OCX Verifiable AI — Strategic Fork Analysis

**Date:** 2026-04-23
**Context:** Written after the cross-architecture determinism test measured that x86 and ARM diverge at token ~55 due to SIMD reduction-order differences. See `STATUS.md` for the technical measurement. This doc is about the strategic response.
**Input paths proposed by founder:**
1. Restricted envelope (same-arch verifier/prover)
2. Architecture-neutral engineering (GGML_CPU_GENERIC or equivalent)
3. Consistency trust model (bounded drift)

---

## Premise challenge (before picking a path)

The "does OCX need cross-machine byte-identity?" question is actually two questions:

1. **Does Kitaab need it?** Probably not. Kitaab SMEs need signed server-side outputs plus audit logs. "Trust Kitaab the signer" is the working trust model for invoice signing, payments, credit scores. A regulator can subpoena the model + prompt + receipt and check against server logs. Byte-identity is a nice-to-have, not a blocker.

2. **Does OCX-as-infrastructure need it?** Yes, if the pitch is "trust-minimized computation primitive." No, if the pitch is "signed attestations of computation from a trusted authority." These are different businesses.

This matters because the three paths all serve OCX-as-infra. If near-term revenue is Kitaab, you might be over-investing in the OCX primitive relative to what Kitaab actually needs. Worth naming before choosing.

---

## The hidden Path 4 — Environment-pinned receipts

Not surfaced in the original three paths, but should be on the table.

The receipt itself carries a signed environment spec:

```
environment: {
  arch: "x86_64",
  cpu_features: ["avx2", "fma"],
  llama_cpp_version: "0.3.20",
  os: "linux-6.12",
  python: "3.10.12"
}
```

Verifier reads the spec:
- If verifier matches spec → runs the bit-identity check. Pass/fail.
- If not → returns "environment mismatch, cannot verify." Not failure, just scoped.

This is Path 1 formalized into a protocol. Same commercial timeline, same limits, but legible. A third party can read a receipt and know whether their machine can verify it without guessing. It also makes honesty enforceable: a prover can't hide what environment they used.

Takes a weekend to add. Ships with Path 1.

---

## The three paths, with inversion analysis

### Path 1: Restricted envelope

- **Time to revenue:** 3 months.
- **Inversion (what makes this fail?):** You anchor the whole protocol on environment-matched verifiers, then a big buyer says "we need ARM Graviton for cost reasons, your system is useless to us." You've painted yourself into x86-only and retrofitting is painful.
- **Mitigation:** Path 4 (environment pinning) makes the constraint legible and limits commitment. You can ship for x86 today and add ARM as a separately-certified environment later without protocol break.

### Path 2: Architecture-neutral engineering (GGML_CPU_GENERIC)

- **Time to usable result:** 12-18 months, uncertain.
- **Perf hit stated (2-5x):** optimistic. Real architecture-neutrality may require either software FP emulation (10-100x slower) or explicit reduction-tree scaffolding in every kernel (enormous engineering work). "Pure C no SIMD" is NOT automatically arch-neutral, because compilers still generate different FP op orders per target.
- **Inversion:** You invest 12 months in ggml kernel work, llama.cpp upstream rejects your patches because they don't care about determinism, you now maintain a fork forever. Bootstrap death.
- This is a research program, not a startup play.

### Path 3: Consistency trust model

- **Time to design + ship:** 6-12 months, heavy design work.
- **The hard technical question:** what IS a consistency relation for LLM outputs? Candidates: logit-fingerprint at N sampled positions, cosine similarity on embedded output, a probabilistic test for "same model distribution." All of these are research questions with no off-the-shelf answer. You'd be publishing papers, not shipping product.
- **Inversion:** You spend 9 months designing a consistency protocol, then realize the trust story is too subtle for SME buyers ("your invoice is verified... 95% confidence"). The crypto framework gets complicated. No one understands it. No one buys it.
- **However:** this is the path that unlocks GPU verification, and GPUs are where production AI actually runs. It's the only path that survives first contact with real enterprise AI workloads.

---

## Recommendation

**Ship Path 1 + Path 4 (environment-pinned restricted envelope) now. Start Path 3 as a research track in parallel. Kill Path 2.**

Reasoning:

- **Path 1 + Path 4 is 3 months to a shippable primitive.** Same-architecture verifiers work for 100% of Kitaab's near-term use cases because Kitaab controls the compute. The environment spec in the receipt makes the limits honest and gives headroom to add architectures later without breaking the protocol.
- **Path 3 is a research track, not a product.** It's the right long-term direction because "consistency" is what actually generalizes to GPUs and heterogeneous infra. But it's 6-12 months of design work before you know what the primitive even looks like. Run it as R&D. Let Kitaab revenue fund it. Aim for a publishable result in 6 months.
- **Path 2 is the trap.** All the work, none of the payoff. Even if it succeeds you're shipping slow inference that still doesn't solve the GPU question. A bootstrapped company should not take this.

The pattern: ship the narrow, honest version today; invest in the conceptually harder, broader version as research; refuse the middle path that costs a lot and leaves you in a dead end.

---

## What to do tomorrow

1. **Today:** Add the environment spec to `OCXReceipt`. 30-minute change in `ocx_ai_verifier.py`. Receipt now says *exactly* what environment it was produced in.
2. **This week:** Write a one-page "OCX Trust Model v1" that states "same-environment verifier required, architecture-pinned, see receipt's `environment` field." Put it in the OCX repo. This is the commercial pitch.
3. **This month:** First real Kitaab integration that uses OCX receipts for invoice signing. Find out if SMEs actually care.
4. **Next quarter:** Start Path 3 research. Hire or collaborate with someone who does ML determinism research (there are maybe 20 people worldwide who take this seriously). Aim for a consistency-model writeup in 6 months.
5. **Don't:** Start ggml kernel work. It's 12 months you won't get back.

---

## The deeper strategic question

All three paths assume the primitive is the product. The harder founder question:

**Is the primitive the product, or is the product the application (Kitaab) built on top of the primitive?**

- If the application is the product, you're spending too much brain on the primitive and not enough on the thing that earns revenue. The primitive becomes plumbing, built just well enough to enable Kitaab, not polished as a standalone offering.
- If the primitive is the product, you need a clearer buyer in year 1 than "future enterprises that will care about this." The "verifiable AI" category barely exists. Regulators mandating AI audit trails is earlier than you'd think, but still a few years out.

The answer to this question changes the path choice:

- **Application-first (Kitaab is the product):** Path 1 + Path 4 is plenty. Path 3 can wait. Don't over-engineer the primitive.
- **Primitive-first (OCX is the product):** Path 1 + Path 4 is the commercial wedge; Path 3 is the R&D that earns technical credibility and future-proofs you for GPUs. Fund Path 3 seriously.

This doc is written assuming primitive-first, which is the posture most of the technical work has been aimed at. If it's actually application-first, the strategic answer simplifies further: ship Path 1 in 3 months, stop there, put remaining cycles on Kitaab features that drive revenue.

---

## Summary

- **Technical finding this week:** Byte-identity holds within a CPU architecture, breaks across x86 ↔ ARM at token ~55 due to SIMD reduction order. Measured, characterized, documented in STATUS.md.
- **Strategic response:** Ship Path 1 + Path 4 (environment-pinned restricted envelope) as the commercial primitive. Start Path 3 (consistency model) as a research track. Reject Path 2 (arch-neutral kernels) as a bootstrap trap.
- **First concrete step:** Add a signed `environment` field to `OCXReceipt`. 30 minutes of work. Makes the architecture boundary legible to every verifier.
- **Biggest unresolved question:** Is OCX the product, or is Kitaab? The answer changes how much to invest in Path 3 and whether to even bother naming OCX as a standalone offering.
