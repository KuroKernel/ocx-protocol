# OCX — the four math wedges

What you must know to engineer this at world-class level. Not the
textbook tour — the narrow set of results that make every reviewer
question answerable from authority instead of guess.

Three of the four are in the security / adversarial / cryptography
register that suits the natural reader of this doc. The fourth
(IEEE 754) is the dull-but-load-bearing one.

---

## 1. Sampling theory — hypergeometric and its binomial bound

### Layman frame

A poacher's-ground inspector. 10,000 hunters in a forest. Some
fraction `f` of them are poaching illegally. You can only check a
few. How many do you check before you're 99.99% sure you'll catch
at least one poacher?

That's the entire soundness math. The hunters are receipts. The
poachers are fraudulent ones. The inspector's check is a
re-execution. The number you check is `k`.

### Technical core

Drawing `k` items without replacement from `N` where `fN` are bad:

```
P(miss all)    = C((1−f)N, k) / C(N, k)        ← hypergeometric (exact)
              ≤ (1−f)^k                         ← binomial (upper bound)

P(catch ≥ 1)  ≥ 1 − (1−f)^k

k             ≥ ln(1/ε) / (−ln(1−f))
              ≈ ln(1/ε) / f          for small f
```

`ε` is the miss probability you're willing to accept. 4 nines = ε = 10⁻⁴.

### Concrete numbers

| Fraud rate `f` | Samples for 4 nines | Samples for 5 nines |
|---|---:|---:|
| 1% | 916 | 1,145 |
| 0.1% | 9,210 | 11,500 |
| 0.01% | 92,099 | 115,000 |

Cost per sample at 72B = 2-5 GPU-seconds. 916 samples = 30-75 GPU
minutes. Affordable.

### The reflex test

"If fraud rate is 0.1% and I want 5 nines, how many samples?"
Answer in 10 seconds: ln(10⁵)/0.001 ≈ 11.5/0.001 = 11,500.

### Cheapest path to mastery

Wikipedia hypergeometric distribution + Coupon Collector lecture
note. ~1 hour total. High-school combinatorics with a fancy name.

### The deeper thing

`(1−f)^k` is an approximation. The exact hypergeometric is tighter
when `k/N` is large. For OCX-scale (sampling 10% of receipts), the
binomial is a slightly loose but always-conservative bound. Knowing
which regime you're in is engineering judgment, not formula
recall.

---

## 2. Hash functions — collision resistance and the birthday bound

### Layman frame

A digital fingerprint. You hand a SHA-256 hash function any
document, any size; it gives you back exactly 32 bytes. Two
different documents in practice never produce the same fingerprint.
"In practice never" means: brute-forcing a collision takes more
compute than the entire Bitcoin network has ever done. So if two
fingerprints match, the documents are the same — within a tolerance
of "the universe ends before someone finds an exception."

### Technical core

- SHA-256 outputs 256 bits.
- Collision attack: birthday bound, ~2¹²⁸ work to find any pair
  `(x, y)` with `H(x) = H(y)`.
- Preimage attack: ~2²⁵⁶ work to invert a given hash.
- The protocol relies on collision resistance + second-preimage
  resistance, NOT on hash being a random oracle.

### The reflex test

"Why is the receipt hash 32 bytes and not 16?"
Answer: 16 bytes = 128-bit hash. Birthday-bound security 2⁶⁴.
Brute-forceable in months on rented GPUs. 32 bytes = 256-bit hash.
Birthday-bound security 2¹²⁸. Computationally infeasible.

### Cheapest path

Bellare-Rogaway 1993 abstract + intro (random oracle model). BLAKE3
design paper. ~30 minutes each, skip the formal proofs.

### The deeper thing

Random Oracle Model (ROM) is a heuristic; real hash functions are
not random oracles. OCX security only depends on collision and
second-preimage resistance, both of which SHA-256 has under standard
assumptions. Reviewers who say "you're using ROM" haven't read the
construction carefully.

---

## 3. Ed25519 — what it commits to, what's malleable, why nonce reuse kills you

### Layman frame

A wax seal that only the person holding the matching ring can press.
Anyone can recognize the seal as real. Change the document by one
bit and the seal becomes invalid. The seal is 64 bytes, the ring
is 32 bytes, both fit on a business card. Verifying a seal takes
microseconds; making one takes microseconds; forging one without
the ring is impossible within the lifetime of the universe.

### Technical core

- Deterministic: `Sign(message, sk)` always produces the same 64-byte
  `(R, s)`. No randomness needed at sign time.
- The signing message in OCX is `domain_separator || canonical_bytes`.
  Domain separator = `"OCXv1|receipt|"`. Without it, signatures from
  another protocol with the same canonical encoding can be replayed.
- Verification: ~80 µs on a modern CPU. Public key 32 bytes.
- Malleability gotcha: signatures must use canonical `s` (with
  `s < ℓ`). Non-canonical `s` = the same signature in two encodings
  = malleability. This bug has hit Solana, Monero, others.

### The reflex test

"Why does OCX prepend `OCXv1|receipt|` to every signing message?"
Answer: domain separation. Without it, an Ed25519 signature on an
OCX receipt is technically a valid signature on any other 32-byte
payload that produces the same canonical encoding under a different
protocol — cross-protocol replay attack.

### Cheapest path

djb's Ed25519 paper sections 1-3 (10 pages, ~1 hour). RFC 8032 for
wire format. Skip the formal security proof.

### The deeper thing

Ed25519 is non-malleable under SUF-CMA (strong unforgeability under
chosen-message attack) **only** when canonical-`s` checking is
enforced. Non-canonical `s` was specifically called out as a
malleability surface in Solana's audit. Saying "Ed25519 is
non-malleable" without the caveat reveals you haven't engineered
against this.

---

## 4. IEEE 754 floating-point determinism — the boring one, the actual moat

### Layman frame (the version for someone who likes the outdoors, not numerical analysis)

A computer cannot store every number exactly. It uses a fixed-size
slot — for bf16, that slot is 16 bits — and approximates everything
to fit. Every approximation introduces a tiny error.

When you add 100 such approximated numbers, the order of addition
matters. `(a + b) + c` is not always equal to `a + (b + c)` because
each addition rounds slightly differently. Different orders →
different totals → different bytes coming out the other end.

A GPU is 80,000 threads adding numbers in parallel. The order they
land in the final sum depends on hardware scheduling. Naively,
this means GPU outputs are non-deterministic — you run the same
input twice, you get slightly different bytes.

**The OCX moat: NVIDIA H100 and AMD MI300X both follow IEEE 754
strictly enough that, on a single GPU with deterministic kernels,
the same input gives the same bytes out — across vendors.** Nobody
else has shown this at 72B. The reason: nobody else cares about
byte-identity hard enough to engineer for it. They want correct
answers, not bit-exact answers.

This is the part that's tedious to engineer and impossible to fake.
Crypto is solved; FP determinism on heterogeneous GPUs is the
research result.

### Technical core

- IEEE 754 bf16: 1 sign bit, 8 exponent bits, 7 mantissa bits.
  Same dynamic range as fp32, ~256× less precision.
- Float addition is **not associative** (rounding-error compounded).
- GPU all-reduce sums across devices. NCCL (NVIDIA) uses ring
  topology; RCCL (AMD) uses fabric topology. **Different reduction
  order → different sum → different bytes.**
- Single-GPU bf16 inference: deterministic by IEEE 754, no all-reduce
  needed, byte-identical across vendors.
- Multi-GPU TP bf16 inference: NOT byte-identical across NCCL vs
  RCCL. The boundary line is at the all-reduce, not at the model.

### The reflex test

"Why does single-GPU H100 vs single-GPU MI300X give the same
output_hash but 2× H100 vs 2× MI300X gives different ones?"

Answer: single-GPU has no inter-device reduction. Operations are
deterministic, bf16 arithmetic is bit-defined by IEEE 754. Two
different vendors implementing 754 strictly produce identical bits.
Multi-GPU TP requires an all-reduce across devices, and NCCL's
ring topology vs RCCL's fabric topology produces different
addition orders → different sums → different output bytes. The
boundary is at the all-reduce, not at the model architecture.

### Cheapest path

Goldberg 1991, "What Every Computer Scientist Should Know About
Floating Point" — 70 pages, ~3 hours. The only "textbook" thing
on the list. Then PyTorch's reproducibility doc (1 page). Then
NVIDIA's Blackwell whitepaper section on numerics, ~20 pages.

### The deeper thing nobody else is engineering at this level

Cross-vendor byte-identity is a fact about **two competing chip
vendors both implementing IEEE 754 strictly at the bit level**, not
about model architecture or weights. The empirical result on
Qwen-72B is the only published demonstration. RISC Zero, zkML
researchers, the Web3 verifiability crowd — none of them are
working on this. They're working on cryptographic arguments for
arbitrary computation. OCX's bet is that for ML specifically,
deterministic execution is enough, and that bet rests entirely on
this fourth wedge.

---

## The wedge in one sentence

**Sampling defends cost. Hash + Ed25519 defend integrity. IEEE 754
defends the moat.** Everyone in this space has the first three.
Nobody has the fourth at frontier model scale. That's where the
edge lives.

---

## How to study, for someone whose natural taste is jungles, gunpowder, alpinism, cryptography, and hacking

Three of the four wedges (sampling, hashes, signatures) are in your
register already. Sampling is hunter math. Hashes and signatures are
hacking math. You will enjoy these. Spend a weekend.

The fourth (IEEE 754) is dull-but-bounded. You don't need to become
a numerical-analysis expert. You need to know **why** cross-vendor
byte-identity is hard, **why** all-reduce breaks it, and **why**
single-GPU survives it. That's a one-weekend read of Goldberg's
70-page paper, not a year of textbooks.

If the fourth wedge bores you to tears, that's a real signal — most
people won't engineer at this level for the same reason. The world
needs a small handful of people who do, and that's where the moat
lives. You don't have to love it. You have to know it well enough
that nobody can embarrass you on it. That's two-three days of work,
not a career.

---

## The deeper question — should I delete everything or do the work?

The work isn't wasted regardless of OCX's commercial outcome.
Mastering these four areas makes you a stronger engineer for
whatever's next: Kitaab, the next paper, the next company, the next
research collaboration.

The math is real. The empirical result is real. The whitepaper
exists, the receipts verify, the cross-vendor demonstration runs.
What's uncertain is the market timing for direct OCX commercial
sales. That's a separate question from whether the technical
contribution holds.

Delete the standalone B2B SaaS posture. Keep the research. Spend
the weekend on the four wedges. See where it goes.
