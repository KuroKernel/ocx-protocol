# Temporal Primitives: The Building Blocks

## Overview

Five patterns underlie most of cryptography. Four are well-understood. The fifth—time—is underexplored.

```
┌─────────────────────────────────────────────────────────────────┐
│                    CRYPTOGRAPHIC PATTERNS                        │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  COMMITMENT     │  ONE-WAYNESS    │  VERIFIABILITY              │
│  (binding)      │  (asymmetry)    │  (checking)                 │
├─────────────────┴─────────────────┴─────────────────────────────┤
│                    ADVERSARIAL THINKING                          │
│            (assume the worst about everyone)                     │
├─────────────────────────────────────────────────────────────────┤
│                    TIME-BOUND ASYMMETRY                          │
│              (the underexplored primitive)                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Primitive 1: Commitment

### The Pattern
Bind yourself to a decision before knowing outcomes.

### How It Works
```
Phase 1 - Commit:    H = hash(secret + nonce)    → publish H
Phase 2 - Wait:      [time passes, events happen]
Phase 3 - Reveal:    publish secret + nonce      → anyone can verify H = hash(secret + nonce)
```

### Why It Matters
- Prevents changing your mind after seeing results
- Foundation of fair coin flips, auctions, voting
- Creates temporal ordering without timestamps

### In OCX Context
Every receipt is a commitment. Publishing a receipt hash binds the transaction to that moment—cannot be altered later.

---

## Primitive 2: One-Wayness

### The Pattern
Easy to compute forward, hard to reverse.

### How It Works
```
Forward:    password → hash(password)     [instant]
Reverse:    hash → password               [years/impossible]

Forward:    private_key → public_key      [instant]
Reverse:    public_key → private_key      [impossible with current math]
```

### Why It Matters
- Enables proving knowledge without revealing it
- Creates asymmetry between creator and attacker
- Foundation of all public-key cryptography

### The Key Insight
One-wayness in space (computation) is well-understood. One-wayness in TIME is different—and possibly more powerful.

---

## Primitive 3: Verifiability

### The Pattern
Check a claim without redoing all the work.

### How It Works
```
Prover:     [does complex computation] → produces proof
Verifier:   [checks proof] → accepts/rejects

Work ratio:  Prover = 1000x  |  Verifier = 1x
```

### Why It Matters
- Scalability: one proof, unlimited verifiers
- Trust: verifier doesn't need to trust prover
- Efficiency: checking < computing

### Examples
| System | Prover Work | Verifier Work |
|--------|-------------|---------------|
| Bitcoin block | 10 minutes mining | 1 second validation |
| ZK-SNARK | Minutes to generate | Milliseconds to verify |
| Digital signature | Sign once | Verify unlimited times |

---

## Primitive 4: Adversarial Thinking

### The Pattern
Design assuming everyone (including yourself) might be malicious.

### How It Works
```
Question:   "What if the server lies?"
Answer:     → Use cryptographic proof, not trust

Question:   "What if the user lies?"
Answer:     → Require committed stake, not promises

Question:   "What if I get hacked?"
Answer:     → Time-delay withdrawals, not instant access
```

### Why It Matters
- Security = assuming the worst
- Trust = knowing attacks are unprofitable
- Systems survive by making honesty the dominant strategy

### The Byzantine Generals Problem
How do you coordinate when:
- Messages might not arrive
- Messages might be fake
- Some generals might be traitors

Cryptography's answer: make coordination verifiable, not assumed.

---

## Primitive 5: Time-Bound Asymmetry (The Underexplored One)

### The Pattern
Create asymmetry through **duration**, not just computation.

### How It Differs from Computational Asymmetry

| Computational Asymmetry | Temporal Asymmetry |
|------------------------|-------------------|
| More CPUs = faster | More CPUs = same time |
| Parallelizable | Sequential only |
| Hardware-dependent | Universal |
| Measured in operations | Measured in wall-clock |

### The Key Primitive: Verifiable Delay Functions (VDFs)

```
┌─────────────────────────────────────────────────────────────┐
│                         VDF                                  │
├─────────────────────────────────────────────────────────────┤
│  Input:   x (any data)                                      │
│  Output:  y, proof                                          │
│                                                             │
│  Properties:                                                │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ SEQUENTIAL:  Cannot parallelize. 1000 CPUs = same time ││
│  │ VERIFIABLE:  Output checkable in O(log n) time         ││
│  │ UNIQUE:      Same input always gives same output       ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  Computing:  [================================] T seconds   │
│  Verifying:  [==] milliseconds                              │
└─────────────────────────────────────────────────────────────┘
```

### How VDFs Work (Simplified)

**The Core Idea**: Repeated squaring in a group where shortcuts are unknown.

```
y = x^(2^T) mod N

To compute:  must square T times (no shortcut known)
To verify:   use proof to check in log(T) operations
```

**Why Sequential?**
- Each squaring depends on previous result
- x² → (x²)² → ((x²)²)² → ...
- Cannot compute step 5 before step 4

**Why Verifiable?**
- Proof uses number-theoretic properties
- Verifier checks consistency without re-squaring

### Current VDF Constructions

| Construction | Based On | Status |
|--------------|----------|--------|
| Wesolowski | RSA groups | Practical, trusted setup |
| Pietrzak | RSA groups | Practical, trusted setup |
| Continuous VDF | Class groups | No trusted setup, slower |

---

## Time-Lock Puzzles

### The Pattern
Encrypt data that "unlocks itself" after time T.

### How It Works
```
Encrypt:
  1. Generate VDF input x
  2. Compute y = VDF(x, T)  [takes time T]
  3. key = hash(y)
  4. ciphertext = encrypt(data, key)
  5. Publish: x, ciphertext

Decrypt (after time T):
  1. Compute y = VDF(x, T)  [takes time T, no shortcut]
  2. key = hash(y)
  3. data = decrypt(ciphertext, key)
```

### Properties
- No trusted party holds the key
- Data genuinely inaccessible until time elapsed
- Anyone can decrypt after time T (no special permission)

### Use Cases
| Use Case | How Time-Lock Helps |
|----------|-------------------|
| Fair lottery | Winning number locked until betting closes |
| Sealed-bid auction | Bids locked until auction ends |
| Will/testament | Contents locked until death + delay |
| Whistleblower | Evidence locked until safe to release |

---

## Proof of Sequential Work

### The Pattern
Prove that work happened *in sequence*, not in parallel.

### How It Works
```
Step 1:  h₁ = hash(input)
Step 2:  h₂ = hash(h₁)
Step 3:  h₃ = hash(h₂)
...
Step n:  hₙ = hash(hₙ₋₁)

Each step depends on previous.
Cannot compute h₃ without h₂.
Cannot parallelize.
```

### Why It's Different from Hash Chains
- Hash chains can be computed quickly (hashing is fast)
- Sequential work uses SLOW operations
- The slowness is the feature, not a bug

### Combined with VDFs
```
Step 1:  y₁ = VDF(input, T)     [T seconds, sequential]
Step 2:  y₂ = VDF(y₁, T)        [T seconds, sequential]
Step 3:  y₃ = VDF(y₂, T)        [T seconds, sequential]

Total: 3T seconds (cannot compress to less)
```

---

## Comparison: Computational vs Temporal Hardness

```
┌────────────────────────────────────────────────────────────────────┐
│                    COMPUTATIONAL HARDNESS                           │
│                    (e.g., Bitcoin mining)                           │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1 CPU:     [================================] 10 minutes           │
│  10 CPUs:   [===] 1 minute                                         │
│  100 CPUs:  [=] 6 seconds                                          │
│                                                                     │
│  → More resources = faster                                          │
│  → Favors wealthy attackers                                         │
│  → Hardware arms race                                               │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│                    TEMPORAL HARDNESS                                │
│                    (e.g., VDF computation)                          │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1 CPU:     [================================] 10 minutes           │
│  10 CPUs:   [================================] 10 minutes           │
│  100 CPUs:  [================================] 10 minutes           │
│                                                                     │
│  → More resources = same time                                       │
│  → Democratic: everyone waits equally                               │
│  → No hardware arms race                                            │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

---

## Synthesis: What Temporal Primitives Enable

| Capability | Without Time Primitive | With Time Primitive |
|------------|----------------------|-------------------|
| Commit to decision | Need reveal step | Auto-reveal after delay |
| Prove elapsed time | Trusted timestamp server | Trustless VDF proof |
| Fair randomness | Multi-party protocol | Single-party VDF output |
| Slow down attackers | Increase key size | Increase time parameter |
| Process verification | Inspect artifact | Verify temporal proof |

---

## The Big Insight

Traditional cryptography creates **spatial asymmetry**: easy to encrypt, hard to decrypt. Easy to sign, hard to forge.

Temporal cryptography creates **temporal asymmetry**: fast to set up, slow to break. Fast to verify, slow to compute.

These are *different* resources. An attacker with infinite compute power can break spatial asymmetry. An attacker with infinite compute power *cannot* break temporal asymmetry—time still passes at the same rate.

This difference might be load-bearing for new kinds of systems.

---

*Next: [02-thompson-insight.md](./02-thompson-insight.md) — Why we should trust processes, not artifacts*
