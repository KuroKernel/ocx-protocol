# OCX Protocol - Stub Replacement Guide
**Complete Guide for Replacing Stubs with Production Implementations**

## 🎯 **STUB REPLACEMENT PRIORITY MATRIX**

### **🔴 CRITICAL (Must Replace for Production)**
1. **Cloud Provider APIs** - Market Intelligence System
2. **Blockchain Integration** - Payment & Settlement System
3. **Authentication System** - Enterprise API Security

### **🟡 HIGH (Important for Full Functionality)**
4. **DSL Compute Execution** - Workload Execution Engine
5. **Hardware Verification** - Trust & Verification System
6. **Database Production Setup** - Data Persistence

### **🟢 MEDIUM (Enhancement Features)**
7. **KYC Integration** - Compliance System
8. **Payment Rails** - Multi-Rail Settlement
9. **Monitoring & Observability** - Production Monitoring

## 🔧 **DETAILED STUB REPLACEMENT INSTRUCTIONS**

### **1. CLOUD PROVIDER APIs** 🔴 **CRITICAL**

#### **Current Stub Location**: `internal/marketintelligence/connectors/`

#### **AWS Connector Replacement**:
```go
// REPLACE THIS STUB:
func (a *AWSConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
    // Mock pricing data
    basePrices := map[string]float64{
        "A100": 3.2,
        "H100": 8.5,
    }
    return map[string]interface{}{
        "on_demand_price": basePrice * multiplier,
        "spot_price": onDemandPrice * spotDiscount,
    }, nil
}

// WITH REAL AWS API INTEGRATION:
func (a *AWSConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
    // 1. Get EC2 pricing from AWS Pricing API
    pricingClient := pricing.NewFromConfig(a.awsConfig)
    
    // 2. Query for specific instance types
    input := &pricing.GetProductsInput{
        ServiceCode: aws.String("AmazonEC2"),
        Filters: []types.Filter{
            {
                Type:  types.FilterTypeTermMatch,
                Field: aws.String("instanceType"),
                Value: aws.String(a.getInstanceType(resourceType)),
            },
            {
                Type:  types.FilterTypeTermMatch,
                Field: aws.String("location"),
                Value: aws.String(a.getLocation(region)),
            },
        },
    }
    
    result, err := pricingClient.GetProducts(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to get AWS pricing: %w", err)
    }
    
    // 3. Parse pricing data
    return a.parsePricingData(result), nil
}
```

#### **Required AWS SDK Integration**:
```bash
# Add to go.mod
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/service/pricing
go get github.com/aws/aws-sdk-go-v2/service/ec2
go get github.com/aws/aws-sdk-go-v2/config
```

#### **GCP Connector Replacement**:
```go
// REPLACE STUB WITH REAL GCP API:
func (g *GCPConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
    // 1. Initialize GCP client
    client, err := compute.NewInstancesRESTClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCP client: %w", err)
    }
    
    // 2. Get machine type pricing
    req := &computepb.GetMachineTypeRequest{
        Project:    g.projectID,
        Zone:       g.getZone(region),
        MachineType: g.getMachineType(resourceType),
    }
    
    resp, err := client.GetMachineType(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to get machine type: %w", err)
    }
    
    // 3. Parse pricing from response
    return g.parseGCPPricing(resp), nil
}
```

#### **Required GCP SDK Integration**:
```bash
# Add to go.mod
go get cloud.google.com/go/compute/apiv1
go get cloud.google.com/go/billing/apiv1
go get google.golang.org/api/option
```

### **2. BLOCKCHAIN INTEGRATION** 🔴 **CRITICAL**

#### **Current Stub Location**: `internal/settlement/usdc.go`

#### **USDC Integration Replacement**:
```go
// REPLACE THIS STUB:
func (u *USDCSettlement) CreateEscrow(ctx context.Context, orderID string, amount *big.Int, requesterAddr, providerAddr string) (*EscrowTransaction, error) {
    // Mock escrow creation
    return &EscrowTransaction{
        ID: fmt.Sprintf("escrow_%d", time.Now().Unix()),
        Status: "pending",
    }, nil
}

// WITH REAL BLOCKCHAIN INTEGRATION:
func (u *USDCSettlement) CreateEscrow(ctx context.Context, orderID string, amount *big.Int, requesterAddr, providerAddr string) (*EscrowTransaction, error) {
    // 1. Connect to blockchain
    client, err := ethclient.Dial(u.rpcURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to blockchain: %w", err)
    }
    
    // 2. Load USDC contract
    usdcContract, err := NewUSDC(common.HexToAddress(u.usdcAddress), client)
    if err != nil {
        return nil, fmt.Errorf("failed to load USDC contract: %w", err)
    }
    
    // 3. Create escrow transaction
    auth, err := bind.NewKeyedTransactorWithChainID(u.privateKey, u.chainID)
    if err != nil {
        return nil, fmt.Errorf("failed to create transactor: %w", err)
    }
    
    // 4. Execute escrow creation
    tx, err := usdcContract.Transfer(auth, common.HexToAddress(u.escrowAddress), amount)
    if err != nil {
        return nil, fmt.Errorf("failed to create escrow transaction: %w", err)
    }
    
    // 5. Wait for confirmation
    receipt, err := bind.WaitMined(ctx, client, tx)
    if err != nil {
        return nil, fmt.Errorf("failed to confirm transaction: %w", err)
    }
    
    return &EscrowTransaction{
        ID: orderID,
        TxHash: tx.Hash().Hex(),
        Status: "confirmed",
        Amount: amount,
    }, nil
}
```

#### **Required Blockchain Dependencies**:
```bash
# Add to go.mod
go get github.com/ethereum/go-ethereum
go get github.com/ethereum/go-ethereum/accounts/abi/bind
go get github.com/ethereum/go-ethereum/ethclient
go get github.com/ethereum/go-ethereum/common
```

### **3. AUTHENTICATION SYSTEM** 🔴 **CRITICAL**

#### **Current Stub Location**: `internal/enterprise/api.go`

#### **JWT Authentication Replacement**:
```go
// REPLACE THIS STUB:
func (e *EnterpriseAPI) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
            http.Error(w, "API key or token required", http.StatusUnauthorized)
            return
        }
        // Simplified token validation
        clientID := "demo_client"
    }
}

// WITH REAL JWT VALIDATION:
func (e *EnterpriseAPI) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract token
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }
        
        // 2. Remove "Bearer " prefix
        if !strings.HasPrefix(tokenString, "Bearer ") {
            http.Error(w, "Invalid token format", http.StatusUnauthorized)
            return
        }
        tokenString = tokenString[7:]
        
        // 3. Parse and validate JWT
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return e.jwtSecret, nil
        })
        
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // 4. Validate claims
        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            clientID := claims["client_id"].(string)
            ctx := context.WithValue(r.Context(), "client_id", clientID)
            next.ServeHTTP(w, r.WithContext(ctx))
        } else {
            http.Error(w, "Invalid token claims", http.StatusUnauthorized)
            return
        }
    }
}
```

#### **Required JWT Dependencies**:
```bash
# Add to go.mod
go get github.com/golang-jwt/jwt/v5
go get github.com/gorilla/mux
```

### **4. DSL COMPUTE EXECUTION** 🟡 **HIGH PRIORITY**

#### **Current Stub Location**: `internal/consensus/telemetry/verification.go`

#### **Container Integration Replacement**:
```go
// REPLACE THIS STUB:
func (vc *VerificationChallenge) GenerateWorkProof(workloadID string, metrics map[string]float64) string {
    workSignature := map[string]interface{}{
        "workload_id": workloadID,
        "cpu_cycles":  metrics["cpu_utilization"] * 1000000, // MOCK
        "memory_ops":  metrics["memory_usage_gb"] * 1024 * 1024, // MOCK
        "gpu_flops":   metrics["gpu_utilization"] * 2000000000, // MOCK
    }
    // Mock proof generation
}

// WITH REAL CONTAINER INTEGRATION:
func (vc *VerificationChallenge) GenerateWorkProof(workloadID string, metrics map[string]float64) string {
    // 1. Connect to Docker daemon
    client, err := docker.NewClientWithOpts(docker.FromEnv)
    if err != nil {
        return ""
    }
    
    // 2. Get container stats
    stats, err := client.ContainerStats(context.Background(), workloadID, false)
    if err != nil {
        return ""
    }
    
    // 3. Parse real metrics
    var containerStats types.Stats
    json.NewDecoder(stats.Body).Decode(&containerStats)
    
    // 4. Calculate real work proof
    cpuDelta := containerStats.CPUStats.CPUUsage.TotalUsage - vc.previousCPU
    systemDelta := containerStats.CPUStats.SystemUsage - vc.previousSystem
    cpuPercent := float64(cpuDelta) / float64(systemDelta) * float64(len(containerStats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
    
    workSignature := map[string]interface{}{
        "workload_id": workloadID,
        "cpu_cycles":  cpuDelta,
        "memory_ops":  containerStats.MemoryStats.Usage,
        "gpu_flops":   vc.getGPUUtilization(workloadID),
        "timestamp":   time.Now().Unix(),
    }
    
    // 5. Generate cryptographic proof
    return vc.generateCryptographicProof(workSignature)
}
```

#### **Required Container Dependencies**:
```bash
# Add to go.mod
go get github.com/docker/docker
go get github.com/docker/docker/api/types
go get github.com/docker/docker/client
```

### **5. HARDWARE VERIFICATION** 🟡 **HIGH PRIORITY**

#### **Current Stub Location**: `internal/verification/hardware.go`

#### **GPU Verification Replacement**:
```go
// REPLACE THIS STUB:
func (hv *HardwareVerifier) VerifyGPU(providerID, gpuID string) (*GPUVerificationResult, error) {
    // Mock GPU verification
    return &GPUVerificationResult{
        GPUID: gpuID,
        Verified: true, // Always returns true
        Model: "A100",
        Memory: "80GB",
    }, nil
}

// WITH REAL GPU VERIFICATION:
func (hv *HardwareVerifier) VerifyGPU(providerID, gpuID string) (*GPUVerificationResult, error) {
    // 1. Connect to NVIDIA Management Library
    nvml.Init()
    defer nvml.Shutdown()
    
    // 2. Get device handle
    device, err := nvml.DeviceGetHandleByIndex(0)
    if err != nil {
        return nil, fmt.Errorf("failed to get device handle: %w", err)
    }
    
    // 3. Get device information
    name, err := device.GetName()
    if err != nil {
        return nil, fmt.Errorf("failed to get device name: %w", err)
    }
    
    memoryInfo, err := device.GetMemoryInfo()
    if err != nil {
        return nil, fmt.Errorf("failed to get memory info: %w", err)
    }
    
    // 4. Verify GPU specifications
    verified := hv.verifyGPUSpecs(name, memoryInfo)
    
    return &GPUVerificationResult{
        GPUID: gpuID,
        Verified: verified,
        Model: name,
        Memory: fmt.Sprintf("%dMB", memoryInfo.Total/1024/1024),
        DriverVersion: hv.getDriverVersion(),
        CUDAVersion: hv.getCUDAVersion(),
    }, nil
}
```

#### **Required Hardware Dependencies**:
```bash
# Add to go.mod
go get github.com/NVIDIA/go-nvml
go get github.com/shirou/gopsutil/v3/cpu
go get github.com/shirou/gopsutil/v3/mem
```

## 🚀 **IMPLEMENTATION CHECKLIST**

### **Phase 1: Critical APIs** (Week 1-2)
- [ ] AWS EC2 Pricing API integration
- [ ] GCP Compute Engine API integration
- [ ] Azure Compute API integration
- [ ] RunPod API integration
- [ ] JWT authentication implementation
- [ ] Database production setup

### **Phase 2: Blockchain Integration** (Week 3-4)
- [ ] USDC smart contract integration
- [ ] Escrow contract deployment
- [ ] Transaction broadcasting
- [ ] Wallet integration
- [ ] Multi-rail settlement

### **Phase 3: Compute Execution** (Week 5-6)
- [ ] Docker container integration
- [ ] Kubernetes orchestration
- [ ] GPU monitoring
- [ ] Hardware verification
- [ ] Performance metrics

### **Phase 4: Production Deployment** (Week 7-8)
- [ ] Infrastructure setup
- [ ] Security hardening
- [ ] Monitoring implementation
- [ ] Load testing
- [ ] Production deployment

## 🎯 **SUCCESS METRICS**

### **API Integration Success**
- [ ] Real-time pricing data from all providers
- [ ] < 100ms API response times
- [ ] 99.9% API availability
- [ ] Rate limiting compliance

### **Blockchain Integration Success**
- [ ] Real USDC transactions
- [ ] < 30s transaction confirmation
- [ ] 99.9% transaction success rate
- [ ] Gas optimization

### **Compute Execution Success**
- [ ] Real container execution
- [ ] Real hardware verification
- [ ] Real performance metrics
- [ ] SLA compliance tracking

## 🎉 **CONCLUSION**

This guide provides the complete roadmap for replacing all stubs with production-ready implementations. The OCX Protocol system is architecturally sound and ready for production deployment once these integrations are complete.

**🚀 Ready to build the future of compute!**
