package enterprise

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// ResourceDiscovery implements the OCX Protocol standard for resource discovery
type ResourceDiscovery struct {
	ProviderConnectors map[string]ProviderConnector
	ResourceCache      map[string]CacheKey
	CacheTTL           time.Duration
	mu                 sync.RWMutex
}

// CacheKey represents cached resource discovery results
type CacheKey struct {
	Resources []AvailableResource
	ExpiresAt time.Time
}

// NewResourceDiscovery creates a new resource discovery engine
func NewResourceDiscovery() *ResourceDiscovery {
	return &ResourceDiscovery{
		ProviderConnectors: make(map[string]ProviderConnector),
		ResourceCache:      make(map[string]CacheKey),
		CacheTTL:           5 * time.Minute,
	}
}

// Discover finds available resources matching the specifications
func (rd *ResourceDiscovery) Discover(ctx context.Context, spec ResourceSpec, region Region, 
	duration float64, filters *DiscoveryFilter) ([]AvailableResource, error) {
	
	// Generate cache key
	cacheKey := rd.generateCacheKey(spec, region, duration, filters)
	
	// Check cache first
	rd.mu.RLock()
	if cached, exists := rd.ResourceCache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		rd.mu.RUnlock()
		return cached.Resources, nil
	}
	rd.mu.RUnlock()
	
	fmt.Printf("🔍 Discovering %dx %s in %s\n", spec.Quantity, spec.ResourceType, region)
	
	// Discover from all providers
	var allResources []AvailableResource
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	providers := []string{"aws", "gcp", "azure", "runpod", "lambda-labs", "coreweave"}
	
	for _, provider := range providers {
		// Skip excluded providers
		if filters != nil && contains(filters.ExcludeProviders, provider) {
			continue
		}
		
		// Skip if provider preferences specified and this provider not in preferences
		if filters != nil && len(filters.ProviderPreferences) > 0 && !contains(filters.ProviderPreferences, provider) {
			continue
		}
		
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			
			resources, err := rd.discoverFromProvider(ctx, p, spec, region, duration, filters)
			if err != nil {
				fmt.Printf("⚠️  Provider %s discovery failed: %v\n", p, err)
				return
			}
			
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(provider)
	}
	
	wg.Wait()
	
	// Sort by price and availability
	sortResources(allResources)
	
	// Cache results
	rd.mu.Lock()
	rd.ResourceCache[cacheKey] = CacheKey{
		Resources: allResources,
		ExpiresAt: time.Now().Add(rd.CacheTTL),
	}
	rd.mu.Unlock()
	
	fmt.Printf("✅ Found %d available resources\n", len(allResources))
	return allResources, nil
}

// discoverFromProvider discovers resources from a specific provider
func (rd *ResourceDiscovery) discoverFromProvider(ctx context.Context, provider string, 
	spec ResourceSpec, region Region, duration float64, filters *DiscoveryFilter) ([]AvailableResource, error) {
	
	// Simulate provider-specific availability and pricing
	basePrices := map[string]map[ResourceType]float64{
		"aws": {
			ResourceTypeA100:   3.2,
			ResourceTypeH100:   8.5,
			ResourceTypeV100:   2.1,
			ResourceTypeTPUV4:  4.5,
			ResourceTypeTPUV5:  6.8,
			ResourceTypeMI300X: 3.8,
		},
		"gcp": {
			ResourceTypeA100:   3.0,
			ResourceTypeH100:   8.2,
			ResourceTypeV100:   2.0,
			ResourceTypeTPUV4:  4.2,
			ResourceTypeTPUV5:  6.5,
			ResourceTypeMI300X: 3.6,
		},
		"azure": {
			ResourceTypeA100:   3.1,
			ResourceTypeH100:   8.3,
			ResourceTypeV100:   2.05,
			ResourceTypeTPUV4:  4.3,
			ResourceTypeTPUV5:  6.6,
			ResourceTypeMI300X: 3.7,
		},
		"runpod": {
			ResourceTypeA100:   1.8,
			ResourceTypeH100:   4.9,
			ResourceTypeV100:   1.2,
			ResourceTypeTPUV4:  2.5,
			ResourceTypeTPUV5:  3.8,
			ResourceTypeMI300X: 2.2,
		},
		"lambda-labs": {
			ResourceTypeA100:   1.9,
			ResourceTypeH100:   5.1,
			ResourceTypeV100:   1.3,
			ResourceTypeTPUV4:  2.6,
			ResourceTypeTPUV5:  3.9,
			ResourceTypeMI300X: 2.3,
		},
		"coreweave": {
			ResourceTypeA100:   2.1,
			ResourceTypeH100:   5.8,
			ResourceTypeV100:   1.4,
			ResourceTypeTPUV4:  2.8,
			ResourceTypeTPUV5:  4.2,
			ResourceTypeMI300X: 2.5,
		},
	}
	
	providerPrices := basePrices[provider]
	basePrice, exists := providerPrices[spec.ResourceType]
	if !exists {
		basePrice = 3.0 // Default price
	}
	
	// Regional pricing multiplier
	regionMultipliers := map[Region]float64{
		RegionUSEast:       1.0,
		RegionUSWest:       1.05,
		RegionEUWest:       1.1,
		RegionEUCentral:    1.08,
		RegionAsiaPacific:  1.15,
		RegionAsiaSingapore: 1.2,
		RegionAsiaTokyo:    1.25,
		RegionMiddleEast:   1.3,
		RegionLATAM:        1.4,
	}
	
	finalPrice := basePrice * regionMultipliers[region]
	
	// Apply filters
	if filters != nil {
		if filters.MaxPricePerHour != nil && finalPrice > *filters.MaxPricePerHour {
			return []AvailableResource{}, nil
		}
		if filters.MinAvailability != nil {
			// Simulate availability check
			simulatedAvailability := simulateAvailability(provider, spec, region)
			if simulatedAvailability < *filters.MinAvailability {
				return []AvailableResource{}, nil
			}
		}
	}
	
	// Simulate availability and setup time
	availability := simulateAvailability(provider, spec, region)
	setupTime := simulateSetupTime(provider, spec, region)
	
	// Check if we have sufficient quantity
	maxAvailable := simulateMaxQuantity(provider, spec, region)
	if maxAvailable < spec.Quantity {
		return []AvailableResource{}, nil
	}
	
	// Create available resource
	resource := AvailableResource{
		ResourceID:              fmt.Sprintf("%s_%s_%s_%d", provider, region, spec.ResourceType, time.Now().Unix()),
		ProviderID:              provider,
		ResourceSpec:            spec,
		Region:                  region,
		PricePerHour:            finalPrice,
		AvailabilityPercentage:  availability,
		EstimatedSetupTimeMinutes: setupTime,
		SupportedFrameworks:     []string{"pytorch", "tensorflow", "jax", "huggingface", "transformers"},
		LastUpdated:             time.Now(),
	}
	
	return []AvailableResource{resource}, nil
}

// generateCacheKey creates a cache key for resource discovery
func (rd *ResourceDiscovery) generateCacheKey(spec ResourceSpec, region Region, 
	duration float64, filters *DiscoveryFilter) string {
	
	key := fmt.Sprintf("%s_%d_%s_%.2f", spec.ResourceType, spec.Quantity, region, duration)
	
	if filters != nil {
		if filters.MaxPricePerHour != nil {
			key += fmt.Sprintf("_max_%.2f", *filters.MaxPricePerHour)
		}
		if filters.MinAvailability != nil {
			key += fmt.Sprintf("_min_avail_%.2f", *filters.MinAvailability)
		}
		if len(filters.ProviderPreferences) > 0 {
			key += fmt.Sprintf("_providers_%v", filters.ProviderPreferences)
		}
		if len(filters.ExcludeProviders) > 0 {
			key += fmt.Sprintf("_exclude_%v", filters.ExcludeProviders)
		}
	}
	
	return key
}

// sortResources sorts resources by price and availability
func sortResources(resources []AvailableResource) {
	// Sort by price first, then by availability (descending)
	for i := 0; i < len(resources)-1; i++ {
		for j := i + 1; j < len(resources); j++ {
			// Compare by price first
			if resources[i].PricePerHour > resources[j].PricePerHour {
				resources[i], resources[j] = resources[j], resources[i]
			} else if resources[i].PricePerHour == resources[j].PricePerHour {
				// If prices are equal, sort by availability (higher is better)
				if resources[i].AvailabilityPercentage < resources[j].AvailabilityPercentage {
					resources[i], resources[j] = resources[j], resources[i]
				}
			}
		}
	}
}

// simulateAvailability simulates resource availability for a provider
func simulateAvailability(provider string, spec ResourceSpec, region Region) float64 {
	// Base availability by provider
	baseAvailability := map[string]float64{
		"aws":         99.5,
		"gcp":         99.3,
		"azure":       99.1,
		"runpod":      95.0,
		"lambda-labs": 97.0,
		"coreweave":   96.0,
	}
	
	availability := baseAvailability[provider]
	
	// Adjust based on resource type (rarer resources have lower availability)
	resourceMultipliers := map[ResourceType]float64{
		ResourceTypeA100:     1.0,
		ResourceTypeH100:     0.85,  // H100s are rarer
		ResourceTypeV100:     1.1,   // V100s are more common
		ResourceTypeTPUV4:    0.9,   // TPUs are specialized
		ResourceTypeTPUV5:    0.8,   // Newer TPUs are rarer
		ResourceTypeMI300X:   0.9,   // AMD GPUs are less common
		ResourceTypeCPUIntel: 1.2,   // CPUs are very common
		ResourceTypeCPUAMD:   1.15,  // AMD CPUs are common
	}
	
	availability *= resourceMultipliers[spec.ResourceType]
	
	// Adjust based on quantity (larger requests have lower availability)
	if spec.Quantity > 100 {
		availability *= 0.9
	} else if spec.Quantity > 50 {
		availability *= 0.95
	}
	
	// Add some randomness
	randomFactor := 0.95 + (0.1 * math.Mod(float64(time.Now().UnixNano()), 1.0))
	availability *= randomFactor
	
	// Ensure availability is within reasonable bounds
	if availability > 99.99 {
		availability = 99.99
	}
	if availability < 50.0 {
		availability = 50.0
	}
	
	return availability
}

// simulateSetupTime simulates setup time for a provider
func simulateSetupTime(provider string, spec ResourceSpec, region Region) float64 {
	// Base setup time by provider
	baseSetupTime := map[string]float64{
		"aws":         5.0,   // AWS is fast
		"gcp":         7.0,   // GCP is reasonably fast
		"azure":       8.0,   // Azure is a bit slower
		"runpod":      15.0,  // Smaller providers are slower
		"lambda-labs": 12.0,
		"coreweave":   18.0,
	}
	
	setupTime := baseSetupTime[provider]
	
	// Adjust based on resource type
	resourceMultipliers := map[ResourceType]float64{
		ResourceTypeA100:     1.0,
		ResourceTypeH100:     1.2,   // H100s take longer to provision
		ResourceTypeV100:     0.9,   // V100s are faster
		ResourceTypeTPUV4:    1.5,   // TPUs take longer
		ResourceTypeTPUV5:    1.8,   // Newer TPUs take even longer
		ResourceTypeMI300X:   1.3,   // AMD GPUs take longer
		ResourceTypeCPUIntel: 0.7,   // CPUs are fast
		ResourceTypeCPUAMD:   0.8,   // AMD CPUs are fast
	}
	
	setupTime *= resourceMultipliers[spec.ResourceType]
	
	// Adjust based on quantity
	if spec.Quantity > 100 {
		setupTime *= 1.5
	} else if spec.Quantity > 50 {
		setupTime *= 1.2
	}
	
	// Add some randomness
	randomFactor := 0.8 + (0.4 * math.Mod(float64(time.Now().UnixNano()), 1.0))
	setupTime *= randomFactor
	
	return setupTime
}

// simulateMaxQuantity simulates maximum available quantity for a provider
func simulateMaxQuantity(provider string, spec ResourceSpec, region Region) int {
	// Base max quantity by provider
	baseMaxQuantity := map[string]int{
		"aws":         1000,
		"gcp":         800,
		"azure":       600,
		"runpod":      200,
		"lambda-labs": 300,
		"coreweave":   150,
	}
	
	maxQuantity := baseMaxQuantity[provider]
	
	// Adjust based on resource type
	resourceMultipliers := map[ResourceType]int{
		ResourceTypeA100:     1,
		ResourceTypeH100:     1,   // H100s are rarer
		ResourceTypeV100:     2,   // V100s are more common
		ResourceTypeTPUV4:    1,   // TPUs are specialized
		ResourceTypeTPUV5:    1,   // Newer TPUs are rarer
		ResourceTypeMI300X:   1,   // AMD GPUs are less common
		ResourceTypeCPUIntel: 5,   // CPUs are very common
		ResourceTypeCPUAMD:   4,   // AMD CPUs are common
	}
	
	maxQuantity *= resourceMultipliers[spec.ResourceType]
	
	// Add some randomness
	randomFactor := 0.5 + (1.0 * math.Mod(float64(time.Now().UnixNano()), 1.0))
	maxQuantity = int(float64(maxQuantity) * randomFactor)
	
	return maxQuantity
}

// contains checks if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
