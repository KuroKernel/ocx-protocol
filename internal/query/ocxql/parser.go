// internal/query/ocxql/parser.go
package ocxql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// OCX-QL is a domain-specific language for compute resource management
// It's NOT SQL - it's designed specifically for compute workloads

// OCXQLQuery represents a parsed OCX-QL query
type OCXQLQuery struct {
	// Resource specifications (not SELECT - this is compute-specific)
	Resources map[string]int `json:"resources"`
	
	// Compute-specific constraints
	Region        string  `json:"region,omitempty"`
	Availability  float64 `json:"availability,omitempty"`  // SLA percentage
	MaxPrice      float64 `json:"max_price,omitempty"`
	MinPrice      float64 `json:"min_price,omitempty"`
	
	// Performance requirements (compute-specific)
	MinMemory     int     `json:"min_memory_gb,omitempty"`
	MinBandwidth  string  `json:"min_bandwidth,omitempty"`  // e.g., "400GB/s"
	MaxLatency    int     `json:"max_latency_ms,omitempty"`
	PowerEfficiency float64 `json:"min_power_efficiency,omitempty"` // FLOPS/Watt
	
	// Workload-specific requirements
	WorkloadType  string `json:"workload_type,omitempty"`  // "training", "inference", "hpc"
	Interconnect  string `json:"interconnect,omitempty"`   // "nvlink", "infinity_fabric", "custom"
	Resilience    string `json:"resilience,omitempty"`     // "single_az", "multi_az", "multi_region"
	
	// Scheduling preferences
	StartTime     *time.Time `json:"start_time,omitempty"`
	Duration      time.Duration `json:"duration,omitempty"`
	Priority      string    `json:"priority,omitempty"`    // "low", "normal", "high", "urgent"
	
	// Budget constraints
	MaxBudget     float64 `json:"max_budget,omitempty"`
	BudgetPeriod  string  `json:"budget_period,omitempty"` // "hourly", "daily", "monthly"
}

// OCXQLParser parses OCX-QL queries
type OCXQLParser struct {
	// Compute-specific resource types
	validResources map[string]bool
	validRegions   map[string]bool
	validWorkloads map[string]bool
}

// NewOCXQLParser creates a new OCX-QL parser
func NewOCXQLParser() *OCXQLParser {
	return &OCXQLParser{
		validResources: map[string]bool{
			"H100":     true,
			"A100":     true,
			"V100":     true,
			"TPU_V4":   true,
			"TPU_V5":   true,
			"CPU":      true,
			"ASIC":     true,
			"FPGA":     true,
		},
		validRegions: map[string]bool{
			"us-east":     true,
			"us-west":     true,
			"eu-west":     true,
			"eu-central":  true,
			"asia-pacific": true,
			"asia-south":  true,
			"mena":        true,
			"latam":       true,
		},
		validWorkloads: map[string]bool{
			"training":    true,
			"inference":   true,
			"hpc":         true,
			"rendering":   true,
			"simulation":  true,
			"mining":      true,
		},
	}
}

// Parse parses an OCX-QL query string
func (p *OCXQLParser) Parse(query string) (*OCXQLQuery, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	
	result := &OCXQLQuery{
		Resources: make(map[string]int),
	}
	
	// OCX-QL syntax is compute-specific, not SQL-like
	// Format: RESOURCE quantity [constraints...]
	
	lines := strings.Split(query, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if err := p.parseLine(line, result); err != nil {
			return nil, fmt.Errorf("parse error at '%s': %w", line, err)
		}
	}
	
	return result, nil
}

// parseLine parses a single line of OCX-QL
func (p *OCXQLParser) parseLine(line string, query *OCXQLQuery) error {
	// Handle resource specifications: "H100 200" or "200x H100"
	if p.isResourceSpec(line) {
		return p.parseResourceSpec(line, query)
	}
	
	// Handle constraints: "region: mena", "sla: 99.99%", etc.
	if strings.Contains(line, ":") {
		return p.parseConstraint(line, query)
	}
	
	// Handle workload specifications: "for training", "workload: inference"
	if strings.HasPrefix(strings.ToLower(line), "for ") || strings.HasPrefix(strings.ToLower(line), "workload:") {
		return p.parseWorkloadSpec(line, query)
	}
	
	// Handle budget constraints: "budget: $1000/hour", "max: $5000"
	if strings.Contains(strings.ToLower(line), "budget") || strings.Contains(strings.ToLower(line), "max:") {
		return p.parseBudgetConstraint(line, query)
	}
	
	return fmt.Errorf("unrecognized syntax: %s", line)
}

// isResourceSpec checks if line specifies compute resources
func (p *OCXQLParser) isResourceSpec(line string) bool {
	// Check for patterns like "H100 200", "200x H100", "200 * H100"
	words := strings.Fields(line)
	if len(words) < 2 {
		return false
	}
	
	// Check if any word is a valid resource type
	for _, word := range words {
		cleanWord := strings.Trim(word, "x*")
		if p.validResources[strings.ToUpper(cleanWord)] {
			return true
		}
	}
	
	return false
}

// parseResourceSpec parses resource specifications
func (p *OCXQLParser) parseResourceSpec(line string, query *OCXQLQuery) error {
	words := strings.Fields(line)
	
	var quantity int
	var resourceType string
	
	// Parse different formats: "H100 200", "200x H100", "200 * H100"
	for _, word := range words {
		cleanWord := strings.Trim(word, "x*")
		
		// Check if this is a number
		if qty, err := strconv.Atoi(cleanWord); err == nil {
			quantity = qty
		} else if p.validResources[strings.ToUpper(cleanWord)] {
			resourceType = strings.ToUpper(cleanWord)
		}
	}
	
	if quantity == 0 || resourceType == "" {
		return fmt.Errorf("invalid resource specification: %s", line)
	}
	
	query.Resources[resourceType] = quantity
	return nil
}

// parseConstraint parses constraint specifications
func (p *OCXQLParser) parseConstraint(line string, query *OCXQLQuery) error {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid constraint format: %s", line)
	}
	
	key := strings.TrimSpace(strings.ToLower(parts[0]))
	value := strings.TrimSpace(parts[1])
	
	switch key {
	case "region":
		if !p.validRegions[value] {
			return fmt.Errorf("invalid region: %s", value)
		}
		query.Region = value
		
	case "sla", "availability":
		// Parse percentage: "99.99%", "99.99"
		value = strings.TrimSuffix(value, "%")
		if sla, err := strconv.ParseFloat(value, 64); err == nil {
			query.Availability = sla
		} else {
			return fmt.Errorf("invalid SLA value: %s", value)
		}
		
	case "max_price", "price_max":
		// Parse price: "$2.50", "2.50"
		value = strings.TrimPrefix(value, "$")
		if price, err := strconv.ParseFloat(value, 64); err == nil {
			query.MaxPrice = price
		} else {
			return fmt.Errorf("invalid price value: %s", value)
		}
		
	case "min_price", "price_min":
		value = strings.TrimPrefix(value, "$")
		if price, err := strconv.ParseFloat(value, 64); err == nil {
			query.MinPrice = price
		} else {
			return fmt.Errorf("invalid price value: %s", value)
		}
		
	case "memory", "min_memory":
		// Parse memory: "80GB", "80"
		value = strings.TrimSuffix(strings.ToUpper(value), "GB")
		if memory, err := strconv.Atoi(value); err == nil {
			query.MinMemory = memory
		} else {
			return fmt.Errorf("invalid memory value: %s", value)
		}
		
	case "bandwidth", "min_bandwidth":
		query.MinBandwidth = value
		
	case "latency", "max_latency":
		// Parse latency: "100ms", "100"
		value = strings.TrimSuffix(strings.ToLower(value), "ms")
		if latency, err := strconv.Atoi(value); err == nil {
			query.MaxLatency = latency
		} else {
			return fmt.Errorf("invalid latency value: %s", value)
		}
		
	case "power", "efficiency":
		if efficiency, err := strconv.ParseFloat(value, 64); err == nil {
			query.PowerEfficiency = efficiency
		} else {
			return fmt.Errorf("invalid power efficiency value: %s", value)
		}
		
	case "interconnect":
		query.Interconnect = value
		
	case "resilience":
		query.Resilience = value
		
	case "priority":
		query.Priority = value
		
	default:
		return fmt.Errorf("unknown constraint: %s", key)
	}
	
	return nil
}

// parseWorkloadSpec parses workload specifications
func (p *OCXQLParser) parseWorkloadSpec(line string, query *OCXQLQuery) error {
	line = strings.ToLower(line)
	
	// Handle "for training", "for inference", etc.
	if strings.HasPrefix(line, "for ") {
		workload := strings.TrimPrefix(line, "for ")
		if p.validWorkloads[workload] {
			query.WorkloadType = workload
			return nil
		}
	}
	
	// Handle "workload: training", "workload: inference", etc.
	if strings.HasPrefix(line, "workload:") {
		workload := strings.TrimSpace(strings.TrimPrefix(line, "workload:"))
		if p.validWorkloads[workload] {
			query.WorkloadType = workload
			return nil
		}
	}
	
	return fmt.Errorf("invalid workload type: %s", line)
}

// parseBudgetConstraint parses budget constraints
func (p *OCXQLParser) parseBudgetConstraint(line string, query *OCXQLQuery) error {
	line = strings.ToLower(line)
	
	// Handle "budget: $1000/hour", "max: $5000"
	if strings.Contains(line, "budget:") {
		parts := strings.Split(line, "budget:")
		if len(parts) == 2 {
			budgetStr := strings.TrimSpace(parts[1])
			// Parse budget and period
			if strings.Contains(budgetStr, "/") {
				budgetParts := strings.Split(budgetStr, "/")
				if len(budgetParts) == 2 {
					amountStr := strings.TrimPrefix(strings.TrimSpace(budgetParts[0]), "$")
					period := strings.TrimSpace(budgetParts[1])
					
					if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
						query.MaxBudget = amount
						query.BudgetPeriod = period
						return nil
					}
				}
			}
		}
	}
	
	// Handle "max: $5000"
	if strings.HasPrefix(line, "max:") {
		amountStr := strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "max:")), "$")
		if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
			query.MaxBudget = amount
			return nil
		}
	}
	
	return fmt.Errorf("invalid budget constraint: %s", line)
}

// Example OCX-QL queries:
// H100 200
// region: mena
// sla: 99.99%
// max_price: $2.50
// for training
// interconnect: nvlink
// resilience: multi_az
// budget: $1000/hour
