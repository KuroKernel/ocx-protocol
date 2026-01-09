# Witness Protocol Design Questions

Before building, these questions need answers.

---

## 1. Witness Network Topology

```
Option A: Federated (like early email)
┌─────────┐  ┌─────────┐  ┌─────────┐
│ Kitaab  │  │ Bank X  │  │ Govt Y  │   ← Organizations run witnesses
│ Witness │  │ Witness │  │ Witness │
└─────────┘  └─────────┘  └─────────┘
     │            │            │
     └────────────┼────────────┘
                  ▼
           Witness Consensus

Option B: Decentralized (like Bitcoin nodes)
┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐
│ W │ │ W │ │ W │ │ W │ │ W │  ← Anyone can run a witness
└───┘ └───┘ └───┘ └───┘ └───┘
        All peers, no hierarchy

Option C: Centralized first, decentralize later
        OCX runs 3-5 witnesses initially
        Open to partners (banks, NBFCs, govt)
        Eventually open to anyone
```

**Question: Which model fits the KITAAB vision?**

---

## 2. Trust Threshold

How many witnesses must agree for an anchor to be "valid"?

```
Option A: Simple majority (>50% of witnesses)

Option B: Supermajority (>66% like BFT consensus)

Option C: Configurable per use case
          - Low stakes (message): 1 witness enough
          - High stakes (₹1Cr invoice): Need 5+ witnesses

Option D: All available witnesses must sign (strongest but slowest)
```

**Question: What's the trust model?**

---

## 3. Latency vs Finality Tradeoff

```
Real-time anchoring:
  → Witness signs immediately on receipt
  → Low latency (<100ms)
  → But witnesses could miss each other's attestations

Batched anchoring:
  → Collect all requests for N seconds
  → All witnesses sign the batch Merkle root
  → Higher latency but guaranteed consistency

Hybrid:
  → Immediate "provisional" anchor (1 witness)
  → "Confirmed" anchor after batch consensus
```

**Question: What's acceptable latency for Kitaab use cases?**

---

## 4. External Chain Anchoring (Bitcoin/Ethereum)

```
Option A: No external anchoring
          Our witness network IS the authority
          Simpler, no dependencies, no fees

Option B: Optional anchoring
          Users can pay to anchor to Bitcoin
          "Premium immutability" for high-value proofs

Option C: Periodic batch anchoring
          OCX anchors daily Merkle root to Bitcoin
          Users don't pay, OCX absorbs cost
          Long-term insurance against witness compromise

Option D: Multiple chains
          Bitcoin + Ethereum + Solana
          Redundancy across ecosystems
```

**Question: Do we need external chain anchoring at all?**

---

## 5. What Gets Witnessed

```
Option A: Only content hashes
          Witness sees: hash, timestamp, creator pubkey
          Witness doesn't see: actual content
          Maximum privacy

Option B: Content hash + metadata
          Witness sees: hash + type + size + optional tags
          Enables richer queries ("show all invoices from June")

Option C: Full content (encrypted)
          Witness stores encrypted blob
          Can decrypt with creator's permission
          Enables content retrieval, not just verification
```

**Question: Privacy vs functionality - where do you want to be?**

---

## 6. Identity Model

```
Option A: Public keys only
          Pseudonymous, like Bitcoin addresses
          Privacy-preserving but no real-world identity link

Option B: DIDs (Decentralized Identifiers)
          W3C standard, interoperable
          Can link to Aadhaar/other identity systems optionally

Option C: Aadhaar-linked from start
          Every anchor tied to verified Indian identity
          Maximum accountability, minimum privacy

Option D: Layered
          Base layer: public keys (pseudonymous)
          Optional: link to DID/Aadhaar for higher trust
```

**Question: For the billion-Indian vision, what's the identity model?**

---

## Initial Assumptions (To Be Confirmed)

Based on KITAAB vision, initial guesses:

| Question | Assumed Answer | Reasoning |
|----------|---------------|-----------|
| Topology | Option C (centralized first) | Ship fast, decentralize later |
| Trust threshold | Option C (configurable) | Different stakes need different assurance |
| Latency | Hybrid | Immediate provisional, batch confirmed |
| External anchoring | Option B or C | Nice-to-have, not core |
| What's witnessed | Option B (hash + metadata) | Privacy with queryability |
| Identity | Option D (layered) | Pseudonymous base, optional Aadhaar |

---

## Use Cases the Protocol Must Support

From KITAAB vision document:

| Use Case | Activity Anchored | Stakes |
|----------|------------------|--------|
| SMB Credit | Invoice + payment history | High (₹lakhs-crores) |
| Gig Workers | Task completions, deliveries | Medium |
| Freelancers | Project receipts | Medium |
| Creators | Content authorship | Medium |
| Domestic Help | Attendance attestation | Low-Medium |
| Communication | Message authenticity | Variable |
| Anti-Deepfake | Media provenance | High (societal) |

The protocol must be generic enough to handle all of these.
