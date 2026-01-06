# OCX Protocol: Frontier Roadmap

## Where OCX Could Actually Matter

---

## Level 1: Where We Are Now (Solved)

**"Did this transaction happen as claimed?"**

- Invoice created → receipt generated → bank verifies
- This is valuable but eventually commoditizable
- **Status**: Production-ready

---

## Level 2: Verifiable AI Decisions (Target: 12-18 months)

### The Problem Nobody's Solving

When KITAAB's AI says "this business has 85% repayment probability" - how does anyone **prove** that:
1. The model that ran is the approved model (not a patched version)
2. The input data wasn't manipulated before inference
3. The output wasn't changed after generation

### OCX Solution

```
Model artifact hash + Input data hash + Inference execution
    → OCX Receipt
    → Verifiable AI decision
```

### Why This Matters

RBI is moving toward AI governance for lending. EU AI Act requires explainability.

> "Every credit decision made by KITAAB's AI has a cryptographic receipt proving exactly which model, which data, which output."

### Implementation Plan

1. **Model Registry** - Hash and register approved model artifacts
2. **Inference Wrapper** - Capture input/output through D-MVM
3. **Decision Receipts** - Special receipt type for AI decisions
4. **Audit Trail** - Link decision receipts to underlying data receipts

---

## Level 3: Composable Trust Chains (Target: 2-3 years)

### The Idea

OCX receipts that **reference other OCX receipts**:

```
Receipt A: Raw invoice created
    ↓
Receipt B: Invoice verified against GST portal (references A)
    ↓
Receipt C: Payment confirmed via bank statement (references B)
    ↓
Receipt D: AI credit score generated (references A, B, C)
    ↓
Receipt E: Loan disbursed (references D)
```

### What This Creates

A **cryptographically linked chain of trust** from raw transaction → credit decision → money movement.

Not a blockchain (no consensus, no tokens, no gas fees). Just **receipts that point to receipts**.

### Why This Is Powerful

Auditor doesn't check individual transactions. They verify the CHAIN.

> "Show me the provenance of this loan decision."
>
> *Here's the chain: invoice → GST match → bank confirmation → AI scoring → disbursement. Every link is cryptographically attested.*

### Implementation Plan

1. **Chain Verification** - Verify linked receipts recursively (IN PROGRESS)
2. **Multi-Reference Support** - Single receipt references multiple parents
3. **Chain Visualization** - Graph representation of trust chains
4. **Provenance Queries** - "Show me everything that led to this receipt"

---

## Level 4: Cross-Entity Trust Networks (Target: 3-5 years)

### The Idea

KITAAB's OCX receipts can be **verified by other KITAAB users**.

**Scenario:**
- Business A (KITAAB user) sends invoice to Business B (also KITAAB user)
- Both sides generate OCX receipts
- The receipts **reference each other**

```
A's Receipt: "I sent invoice #123 to B"
B's Receipt: "I received invoice #123 from A"
    ↓
Cross-referenced verification without central authority
```

### What This Creates

A **trust network** where businesses verify each other's claims.

The more KITAAB users in a supply chain, the stronger the verification.

### The Network Effect Moat

> Clone KITAAB's code → you get the software
> Clone KITAAB's network → impossible, it's the relationships

### Implementation Plan

1. **Entity Registry** - Public key infrastructure for businesses
2. **Cross-Reference Protocol** - Standard for mutual receipt references
3. **Network Discovery** - Find other KITAAB users in supply chain
4. **Reputation Aggregation** - Trust scores based on network verification

---

## Level 5: The Sovereign Verification Layer (Target: 5-10 years)

### The Nation-Scale Vision

India Stack has:
- Aadhaar (identity)
- UPI (payments)
- GSTN (tax)
- Account Aggregator (data)

**What's missing?**

**Verification layer.** A way to cryptographically attest that data flowing through India Stack is authentic and untampered.

### OCX as Infrastructure

Every AA data pull → OCX receipt
Every GST filing → OCX receipt
Every UPI transaction → OCX receipt (downstream)

> "The verification primitive for India Stack"

### What This Means

Not "startup that got acquired."

**"Protocol that became a standard."**

---

## Technical Frontiers (What's Still Hard)

| Frontier | Status | OCX Relevance |
|----------|--------|---------------|
| **Deterministic execution at scale** | Unsolved cleanly | D-MVM is novel |
| **Recursive verification** | Emerging (RISC0 working on it) | Receipts-of-receipts |
| **Cross-party attestation without blockchain** | Nobody doing well | Trust networks |
| **AI model provenance** | Regulatory pressure, no solution | OCX + model hashing |
| **Offline-verifiable proofs** | ZK is expensive, TEE needs hardware | OCX is pure software, cheap |
| **Sub-second verification** | ZK is slow (minutes) | OCX is <5ms |

---

## Moat Progression

```
Year 1 (Now):
    KITAAB + OCX = "Our transactions are verifiable"
    Moat: Data accumulation begins

Year 2-3:
    OCX receipts reference each other
    Moat: Trust chains can't be replicated without history

Year 3-5:
    Cross-entity verification network
    Moat: Network effects (every new user strengthens all users)

Year 5+:
    OCX as India Stack primitive
    Moat: Protocol becomes standard, you wrote it
```

---

## Implementation Priorities

### Immediate (Next 3 Months)

- [x] D-MVM implementation
- [x] Receipt generation (v1.1 with chaining)
- [x] Single receipt verification
- [ ] **Receipt chain verification** ← CURRENT FOCUS
- [ ] File provisional patent on D-MVM + receipt structure
- [ ] Production deployment with KITAAB soft launch

### Medium-Term (6-12 Months)

- [ ] Multi-reference receipts (receipt references N parents)
- [ ] AI decision attestation (model hash + input + output)
- [ ] CA verification dashboard
- [ ] Chain visualization tools

### Long-Term (12-24 Months)

- [ ] Cross-entity receipts (A's invoice ↔ B's PO)
- [ ] Entity registry and PKI
- [ ] Open verification protocol
- [ ] Protocol specification publication

---

## The Core Insight

**Tech has zero moat** when it's:
- A feature
- An API wrapper
- A UI improvement

**Tech has deep moat** when it's:
- A protocol that others build on
- A data structure that compounds
- A network that strengthens with each node

OCX is the second kind. But only if we **design for composability and accumulation**, not just "verification as a feature."

---

## Current Focus: Receipt Chain Verification

### What Exists (v1.1)
- `prev_receipt_hash` field in receipt structure
- Single receipt verification (Ed25519 + constraints)

### What We're Building
- Chain traversal and verification
- Recursive validation (verify receipt + all ancestors)
- Chain integrity checks (no gaps, no forks)
- Efficient batch verification for long chains

### Success Criteria
- Verify chain of N receipts in O(N) time
- Detect tampering at any point in chain
- Support branching (receipt with multiple children)
- API endpoint for chain verification
