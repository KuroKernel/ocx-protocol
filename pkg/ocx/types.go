// types.go — OCX Protocol v0.1 (foundational schemas)
// go 1.22+

package ocx

import "time"

// ---------- Core primitives ----------

type ID = string // ULID (base32 Crockford). Example: "01J8Z3TF6X9H3W1M6A6J1KSTQH"

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

var V010 = Version{Major: 0, Minor: 1, Patch: 0}

type Hash struct {
	Alg   string `json:"alg"`   // e.g., "sha256"
	Value string `json:"value"` // hex/base64
}

type Sig struct {
	Alg   string `json:"alg"` // e.g., "ed25519"
	KeyID ID     `json:"key_id"`
	// Detached signature over the canonical JSON of the message envelope
	// (excluding this Sig field).
	SigB64 string `json:"sig_b64"`
}

type Money struct {
	Currency string `json:"currency"` // "USD","EUR","INR","USDC" (fiat or token)
	Amount   string `json:"amount"`   // decimal string to avoid float issues
	Scale    int    `json:"scale"`    // 2 for cents, 6 for micro, etc.
}

// ---------- Identity & compliance (referenced, implemented in id.go) ----------

type PartyRef struct {
	PartyID   ID      `json:"party_id"` // maps to an Identity record
	Role      string  `json:"role"`     // "provider","buyer","arbiter","issuer"
	KYC       *KYCRef `json:"kyc,omitempty"`
	AttestRef *ID     `json:"attest_ref,omitempty"`
}

type KYCRef struct {
	Status  string    `json:"status"` // "none","pending","verified","restricted"
	Schema  string    `json:"schema"` // e.g., "OCX-KYC-2025-01"
	Updated time.Time `json:"updated"`
	Issuer  *ID       `json:"issuer,omitempty"` // identity provider
}

// ---------- Resource model ----------

type GPUArch string

const (
	GPUArchNVIDIA GPUArch = "nvidia"
	GPUArchAMD    GPUArch = "amd"
	GPUArchINTEL  GPUArch = "intel"
	// room for "ascend","habana","tpu-proxy" (if proxied)
)

type Interconnect string

const (
	InterconnectNVLink Interconnect = "nvlink"
	InterconnectPCIe   Interconnect = "pcie"
	InterconnectIB     Interconnect = "infiniband"
	InterconnectEther  Interconnect = "ethernet"
)

type GPU struct {
	Model        string       `json:"model"`        // "H100-80GB","A100-40GB","MI300X"
	VRAMGiB      int          `json:"vram_gib"`     // 80
	Count        int          `json:"count"`        // number in this slice
	Arch         GPUArch      `json:"arch"`         // "nvidia"
	Interconnect Interconnect `json:"interconnect"` // "nvlink"
	TF32TFLOPS   float32      `json:"tf32_tflops"`  // optional perf hints
	FP16TFLOPS   float32      `json:"fp16_tflops"`
	BWGBps       float32      `json:"bw_gbps"` // HBM/PCIe effective
}

type CPU struct {
	Model   string `json:"model"` // "EPYC 9654"
	Sockets int    `json:"sockets"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
}

type Memory struct {
	RAMGiB int `json:"ram_gib"`
}

type Storage struct {
	SSDGiB         int `json:"ssd_gib"`
	ThroughputMBps int `json:"throughput_mb_s"`
}

type Network struct {
	EgressMbps  int    `json:"egress_mbps"`
	IngressMbps int    `json:"ingress_mbps"`
	Region      string `json:"region"` // "eu-de-1","us-west-2","in-central-1"
	Zone        string `json:"zone"`   // provider-defined zone
	PublicIP    bool   `json:"public_ip"`
}

type RuntimeSpec struct {
	OS            string   `json:"os"`             // "ubuntu-22.04"
	Kernel        string   `json:"kernel"`         // "6.8.x"
	Container     bool     `json:"container"`      // supports OCI
	Drivers       []string `json:"drivers"`        // e.g., "cuda-12.4","rocm-6.1"
	Frameworks    []string `json:"frameworks"`     // "pytorch-2.3", "jax-0.4"
	SecureCompute bool     `json:"secure_compute"` // SEV/TDX/CC
}

type Fleet struct {
	FleetID     ID        `json:"fleet_id"`
	Owner       PartyRef  `json:"owner"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Labels      []string  `json:"labels,omitempty"`
	Description string    `json:"description,omitempty"`
	// Aggregate capabilities:
	GPU      GPU          `json:"gpu"`
	CPU      CPU          `json:"cpu"`
	Memory   Memory       `json:"memory"`
	Storage  Storage      `json:"storage"`
	Network  Network      `json:"network"`
	Runtime  RuntimeSpec  `json:"runtime"`
	Capacity int          `json:"capacity"` // max concurrent leases this fleet can handle
	SLA      *SLASpec     `json:"sla,omitempty"`
	Attest   *Attestation `json:"attest,omitempty"`
}

// ---------- Market objects ----------

type PriceUnit string

const (
	PricePerGPUHour PriceUnit = "gpu_hour"
	PricePerJob     PriceUnit = "job"
)

type Offer struct {
	OfferID    ID        `json:"offer_id"`
	Version    Version   `json:"version"`
	Provider   PartyRef  `json:"provider"`
	FleetID    ID        `json:"fleet_id"`
	Unit       PriceUnit `json:"unit"`
	UnitPrice  Money     `json:"unit_price"` // e.g., INR 160.00 per gpu_hour
	MinHours   int       `json:"min_hours"`
	MaxHours   int       `json:"max_hours"`
	MinGPUs    int       `json:"min_gpus"`
	MaxGPUs    int       `json:"max_gpus"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidTo    time.Time `json:"valid_to"`
	TermsHash  *Hash     `json:"terms_hash,omitempty"` // hash of T&Cs doc
	Compliance []string  `json:"compliance,omitempty"` // "HIPAA","GDPR","ITAR"
	Sig        *Sig      `json:"sig,omitempty"`
}

type OrderState string

const (
	OrderPending  OrderState = "pending"
	OrderAccepted OrderState = "accepted"
	OrderActive   OrderState = "active"
	OrderClosed   OrderState = "closed"
	OrderFailed   OrderState = "failed"
)

type Order struct {
	OrderID       ID         `json:"order_id"`
	Version       Version    `json:"version"`
	Buyer         PartyRef   `json:"buyer"`
	OfferID       ID         `json:"offer_id"`
	RequestedGPUs int        `json:"requested_gpus"`
	Hours         int        `json:"hours"` // requested duration
	BudgetCap     *Money     `json:"budget_cap,omitempty"`
	State         OrderState `json:"state"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Sig           *Sig       `json:"sig,omitempty"`
}

type LeaseState string

const (
	LeaseProvisioning LeaseState = "provisioning"
	LeaseRunning      LeaseState = "running"
	LeasePaused       LeaseState = "paused"
	LeaseCompleted    LeaseState = "completed"
	LeaseBreached     LeaseState = "breached"
	LeaseCancelled    LeaseState = "cancelled"
)

type Lease struct {
	LeaseID      ID         `json:"lease_id"`
	Version      Version    `json:"version"`
	OrderID      ID         `json:"order_id"`
	FleetID      ID         `json:"fleet_id"`
	AssignedGPUs int        `json:"assigned_gpus"`
	StartAt      time.Time  `json:"start_at"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	State        LeaseState `json:"state"`
	Access       AccessSpec `json:"access"` // how the buyer connects
	Policy       PolicyRef  `json:"policy"` // runtime policies to enforce
	SLA          *SLASpec   `json:"sla,omitempty"`
	Sig          *Sig       `json:"sig,omitempty"`
}

type AccessSpec struct {
	Method    string   `json:"method"`      // "ssh","tls-grpc","ocx-agent"
	Endpoints []string `json:"endpoints"`   // host:port or URLs
	CACertPEM string   `json:"ca_cert_pem"` // mTLS bootstrap
	CredsRef  ID       `json:"creds_ref"`   // id of ephemeral credential bundle
}

type PolicyRef struct {
	PolicyID ID     `json:"policy_id"`
	Revision string `json:"revision"` // e.g., git sha or semver
	Hash     Hash   `json:"hash"`
}

type SLASpec struct {
	AvailabilityPct float32 `json:"availability_pct"` // e.g., 99.5
	MinEgressMbps   int     `json:"min_egress_mbps"`
	MaxJitterMs     int     `json:"max_jitter_ms"`
	Remedy          string  `json:"remedy"` // credit/backoff rules
}

// ---------- Telemetry & metering ----------

type MeterRecord struct {
	RecordID  ID        `json:"record_id"`
	LeaseID   ID        `json:"lease_id"`
	Timestamp time.Time `json:"ts"`
	GPUHours  int       `json:"gpu_hours"` // integral or fixed-pt
	EgressGB  int       `json:"egress_gb"`
	CPUHours  int       `json:"cpu_hours"`
	Notes     string    `json:"notes,omitempty"`
	Sig       *Sig      `json:"sig,omitempty"` // signed by provider/agent
}

type InvoiceState string

const (
	InvOpen    InvoiceState = "open"
	InvPaid    InvoiceState = "paid"
	InvDispute InvoiceState = "dispute"
	InvVoid    InvoiceState = "void"
)

type Invoice struct {
	InvoiceID ID            `json:"invoice_id"`
	Version   Version       `json:"version"`
	LeaseID   ID            `json:"lease_id"`
	Issuer    PartyRef      `json:"issuer"` // provider or clearing house
	Recipient PartyRef      `json:"recipient"`
	Lines     []InvoiceLine `json:"lines"`
	Total     Money         `json:"total"`
	State     InvoiceState  `json:"state"`
	IssuedAt  time.Time     `json:"issued_at"`
	DueAt     time.Time     `json:"due_at"`
	Sig       *Sig          `json:"sig,omitempty"`
}

type InvoiceLine struct {
	Description string `json:"description"`
	Unit        string `json:"unit"` // "gpu_hour","egress_gb"
	Qty         string `json:"qty"`  // decimal string
	UnitPrice   Money  `json:"unit_price"`
	LineTotal   Money  `json:"line_total"`
}

// ---------- Attestation & compliance ----------

type Attestation struct {
	AttestID  ID                `json:"attest_id"`
	Version   Version           `json:"version"`
	Issuer    PartyRef          `json:"issuer"`  // e.g., "OCX-CA","Intel-SGX","CloudProvider"
	Subject   ID                `json:"subject"` // FleetID or LeaseID
	Claims    map[string]string `json:"claims"`  // "sev":"true","tdx":"true","fw":"nvidia-xxx"
	IssuedAt  time.Time         `json:"issued_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Sig       *Sig              `json:"sig,omitempty"`
}

// ---------- Dispute / arbitration ----------

type DisputeState string

const (
	DisputeOpen      DisputeState = "open"
	DisputeReview    DisputeState = "review"
	DisputeResolved  DisputeState = "resolved"
	DisputeEscalated DisputeState = "escalated"
	DisputeRejected  DisputeState = "rejected"
)

type Dispute struct {
	DisputeID ID           `json:"dispute_id"`
	LeaseID   ID           `json:"lease_id"`
	RaisedBy  PartyRef     `json:"raised_by"`
	Reason    string       `json:"reason"` // "SLA breach","billing error","abuse"
	Evidence  []Hash       `json:"evidence,omitempty"`
	State     DisputeState `json:"state"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Arbiter   *PartyRef    `json:"arbiter,omitempty"`
	Decision  string       `json:"decision,omitempty"`
	Sig       *Sig         `json:"sig,omitempty"`
}

// ---------- Envelope for on-wire messages ----------

type MsgKind string

const (
	KindOffer   MsgKind = "offer"
	KindOrder   MsgKind = "order"
	KindLease   MsgKind = "lease"
	KindMeter   MsgKind = "meter"
	KindInvoice MsgKind = "invoice"
	KindDispute MsgKind = "dispute"
	KindAttest  MsgKind = "attest"
)

type Envelope struct {
	ID       ID        `json:"id"` // message id (ULID)
	Kind     MsgKind   `json:"kind"`
	Version  Version   `json:"version"`
	IssuedAt time.Time `json:"issued_at"`
	Prev     *ID       `json:"prev,omitempty"` // chain messages if needed
	Payload  any       `json:"payload"`        // one of the above structs
	Hash     Hash      `json:"hash"`           // canonical JSON payload hash
	Sig      Sig       `json:"sig"`            // signer = issuer PartyRef.KeyID
}
