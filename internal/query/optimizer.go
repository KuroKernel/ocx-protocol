package query

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// QueryOptimizer optimizes OCX-QL queries for performance
type QueryOptimizer struct {
	statisticsStore *Statistics
	indexRegistry   *IndexRegistry
}

// Statistics holds table and column statistics for optimization
type Statistics struct {
	TableStats map[string]*TableStats
}

type TableStats struct {
	RowCount      int64
	FieldStats   map[string]*FieldStats
	LastUpdated   time.Time
}

type FieldStats struct {
	DistinctValues int64
	NullCount      int64
	MinValue       interface{}
	MaxValue       interface{}
	MostCommon     []interface{}
}

// IndexRegistry manages available indexes
type IndexRegistry struct {
	Indexes map[string][]*Index
}

type Index struct {
	Name        string
	Table       string
	Fields     []string
	Type        string
	Selectivity float64
}

// QueryCondition represents a WHERE condition for optimization
type QueryConditionOld struct {
	Field   string
	Operator string
	Value    interface{}
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(statistics *Statistics, indexRegistry *IndexRegistry) *QueryOptimizer {
	return &QueryOptimizer{
		statisticsStore: statistics,
		indexRegistry:   indexRegistry,
	}
}

// OptimizeSelectQuery optimizes SELECT queries
func (opt *QueryOptimizer) OptimizeSelectQuery(ctx context.Context, query *Query) (*QueryPlan, error) {
	// 1. QueryCondition pushdown
	predicates := opt.extractQueryConditions(query.Where)
	
	// 2. Index selection based on predicates
	// bestIndex := opt.selectBestIndex(query.From, predicates)
	
	// 3. Cost estimation for different join orders
	joinPlans := opt.generateJoinPlans(query.From, predicates)
	
	// 4. Select lowest cost plan
	bestPlan := opt.selectLowestCostPlan(joinPlans)
	
	return bestPlan, nil
}

// OptimizeComputeQuery optimizes compute resource queries with geographic optimization
func (opt *QueryOptimizer) OptimizeComputeQuery(ctx context.Context, query *Query) (*QueryPlan, error) {
	plan := &QueryPlan{}
	
	// 1. Geographic filtering first (highest selectivity)
	if regions := opt.extractPreferredRegions(query); len(regions) > 0 {
		plan.Steps = append(plan.Steps, ExecutionStep{
			Operation: IndexScan,
			Table:     "compute_units",
			IndexHint: "idx_compute_units_region",
			QueryCondition: &QueryCondition{
				Field:   "geographic_region",
				Operator: "IN",
				Value:    regions,
			},
		})
	}
	
	// 2. Hardware type filtering (medium selectivity)
	if hardwareType := opt.extractHardwareType(query); hardwareType != "" {
		plan.Steps = append(plan.Steps, ExecutionStep{
			Operation: IndexScan,
			Table:     "compute_units",
			IndexHint: "idx_compute_units_hardware",
			QueryCondition: &QueryCondition{
				Field:   "hardware_type",
				Operator: "=",
				Value:    hardwareType,
			},
		})
	}
	
	// 3. Availability filtering (high volatility, check last)
	plan.Steps = append(plan.Steps, ExecutionStep{
		Operation: SequentialScan, // Availability changes too fast for indexes
		Table:     "compute_units",
		QueryCondition: &QueryCondition{
			Field:   "current_availability",
			Operator: "=",
			Value:    "available",
		},
	})
	
	// 4. Price sorting (most common ordering)
	if query.OrderBy != nil && len(query.OrderBy.Fields) > 0 {
		orderField := query.OrderBy.Fields[0].Field
		if orderField == "price" || orderField == "base_price_per_hour_usdc" {
			plan.Steps = append(plan.Steps, ExecutionStep{
				Operation: Sort,
				IndexHint: "idx_compute_units_price",
			})
		}
	}
	
	// 5. Reputation joining
	if minReputation := opt.extractMinReputation(query); minReputation > 0 {
		plan.Steps = append(plan.Steps, ExecutionStep{
			Operation: HashJoin,
			Table:     "provider_reputation_cache",
			QueryCondition: &QueryCondition{
				Field:   "overall_score",
				Operator: ">=",
				Value:    minReputation,
			},
		})
	}
	
	// 6. Result limiting
	if query.Limit > 0 {
		plan.Steps = append(plan.Steps, ExecutionStep{
			Operation: LimitStep,
			Limit:     query.Limit,
		})
	}
	
	plan.EstimatedCost = opt.calculatePlanCost(plan)
	plan.EstimatedLatency = opt.calculatePlanLatency(plan)
	
	return plan, nil
}

// Helper methods for query optimization

func (opt *QueryOptimizer) extractQueryConditions(where *WhereClause) []QueryCondition {
	if where == nil {
		return nil
	}
	
	predicates := make([]QueryCondition, len(where.QueryConditions))
	for i, condition := range where.QueryConditions {
		predicates[i] = QueryCondition{
			Field:   condition.Field,
			Operator: condition.Operator,
			Value:    condition.Value,
		}
	}
	
	return predicates
}

func (opt *QueryOptimizer) selectBestIndex(table string, predicates []QueryCondition) *Index {
	availableIndexes := opt.indexRegistry.GetIndexesForTable(table)
	
	// // bestIndex := // // bestIndex := // bestIndex := &Index{}Index{}Index{}
	bestSelectivity := 1.0
	
	for _, index := range availableIndexes {
		selectivity := opt.estimateSelectivity(index, predicates)
		if selectivity < bestSelectivity {
			bestSelectivity = selectivity
			// bestIndex = index
		}
	}
	
	return // bestIndex
}

func (opt *QueryOptimizer) estimateSelectivity(index *Index, predicates []QueryCondition) float64 {
	stats := opt.statisticsStore.GetTableStats(index.Table)
	
	// For compound indexes, estimate combined selectivity
	selectivity := 1.0
	for _, predicate := range predicates {
		if index.CoversField(predicate.Field) {
			columnStats := stats.GetFieldStats(predicate.Field)
			predicateSelectivity := opt.estimateQueryConditionSelectivity(predicate, columnStats)
			selectivity *= predicateSelectivity
		}
	}
	
	return selectivity
}

func (opt *QueryOptimizer) estimateQueryConditionSelectivity(predicate QueryCondition, columnStats *FieldStats) float64 {
	if columnStats == nil {
		return 0.1 // Default selectivity if no stats
	}
	
	switch predicate.Operator {
	case "=":
		// Equality: 1 / distinct_values
		return 1.0 / float64(columnStats.DistinctValues)
	case "!=":
		// Inequality: 1 - equality selectivity
		return 1.0 - (1.0 / float64(columnStats.DistinctValues))
	case ">", ">=", "<", "<=":
		// Range: approximately 1/3 for typical data distribution
		return 0.33
	case "IN":
		// IN clause: count of values / distinct_values
		if values, ok := predicate.Value.([]interface{}); ok {
			return float64(len(values)) / float64(columnStats.DistinctValues)
		}
		return 0.1
	case "LIKE":
		// Pattern matching: depends on pattern, use conservative estimate
		return 0.05
	default:
		return 0.1 // Conservative default
	}
}

func (opt *QueryOptimizer) generateJoinPlans(table string, predicates []QueryCondition) []*QueryPlan {
	// Simplified join plan generation
	// In a real implementation, this would consider multiple join orders
	plans := []*QueryPlan{}
	
	// Plan 1: Index scan on main table
	plan1 := &QueryPlan{
		Steps: []ExecutionStep{
			{
				Operation: IndexScan,
				Table:     table,
				QueryCondition: &predicates[0],
			},
		},
	}
	plan1.EstimatedCost = opt.calculatePlanCost(plan1)
	plans = append(plans, plan1)
	
	// Plan 2: Sequential scan (for small tables)
	plan2 := &QueryPlan{
		Steps: []ExecutionStep{
			{
				Operation: SequentialScan,
				Table:     table,
			},
		},
	}
	plan2.EstimatedCost = opt.calculatePlanCost(plan2)
	plans = append(plans, plan2)
	
	return plans
}

func (opt *QueryOptimizer) selectLowestCostPlan(plans []*QueryPlan) *QueryPlan {
	if len(plans) == 0 {
		return &QueryPlan{}
	}
	
	bestPlan := plans[0]
	for _, plan := range plans[1:] {
		if plan.EstimatedCost < bestPlan.EstimatedCost {
			bestPlan = plan
		}
	}
	
	return bestPlan
}

func (opt *QueryOptimizer) calculatePlanCost(plan *QueryPlan) float64 {
	cost := 0.0
	
	for _, step := range plan.Steps {
		switch step.Operation {
		case IndexScan:
			cost += 10.0 // Low cost for index access
		case SequentialScan:
			cost += 100.0 // Higher cost for full table scan
		case HashJoin:
			cost += 50.0 // Medium cost for hash join
		case NestedLoopJoin:
			cost += 200.0 // High cost for nested loop join
		case Sort:
			cost += 30.0 // Medium cost for sorting
		case LimitStep:
			cost += 5.0 // Low cost for limiting
		case Aggregate:
			cost += 40.0 // Medium cost for aggregation
		}
	}
	
	return cost
}

func (opt *QueryOptimizer) calculatePlanLatency(plan *QueryPlan) time.Duration {
	latency := time.Duration(0)
	
	for _, step := range plan.Steps {
		switch step.Operation {
		case IndexScan:
			latency += 5 * time.Millisecond
		case SequentialScan:
			latency += 50 * time.Millisecond
		case HashJoin:
			latency += 20 * time.Millisecond
		case NestedLoopJoin:
			latency += 100 * time.Millisecond
		case Sort:
			latency += 15 * time.Millisecond
		case LimitStep:
			latency += 1 * time.Millisecond
		case Aggregate:
			latency += 25 * time.Millisecond
		}
	}
	
	return latency
}

// Specialized methods for compute queries

func (opt *QueryOptimizer) extractPreferredRegions(query *Query) []string {
	if query.Reserve != nil && query.Reserve.ReserveOptions != nil {
		if regions, ok := query.Reserve.ReserveOptions["PREFERRED_REGIONS_regions"].([]string); ok {
			return regions
		}
	}
	return nil
}

func (opt *QueryOptimizer) extractHardwareType(query *Query) string {
	if query.Reserve != nil && query.Reserve.ComputeSpec != nil {
		if hardwareType, ok := query.Reserve.ComputeSpec["hardware_type"].(string); ok {
			return hardwareType
		}
	}
	return ""
}

func (opt *QueryOptimizer) extractMinReputation(query *Query) float64 {
	if query.Reserve != nil && query.Reserve.ReserveOptions != nil {
		if reputation, ok := query.Reserve.ReserveOptions["MIN_REPUTATION"].(float64); ok {
			return reputation
		}
	}
	return 0.0
}

// Index management methods

func (ir *IndexRegistry) GetIndexesForTable(table string) []*Index {
	if indexes, exists := ir.Indexes[table]; exists {
		return indexes
	}
	return []*Index{}
}

func (i *Index) CoversField(column string) bool {
	for _, col := range i.Fields {
		if col == column {
			return true
		}
	}
	return false
}

// Statistics management methods

func (s *Statistics) GetTableStats(table string) *TableStats {
	if stats, exists := s.TableStats[table]; exists {
		return stats
	}
	return &TableStats{
		RowCount:    1000, // Default estimate
		FieldStats: make(map[string]*FieldStats),
	}
}

func (ts *TableStats) GetFieldStats(column string) *FieldStats {
	if stats, exists := ts.FieldStats[column]; exists {
		return stats
	}
	return &FieldStats{
		DistinctValues: 100, // Default estimate
		NullCount:      0,
	}
}

// Example usage and test queries
func ExampleOptimizedQueries() {
	// Example 1: Find cheapest H100s in US with 95%+ reputation
	query1 := `
	SELECT unit_id, provider_id, base_price_per_hour_usdc, reputation_score
	FROM COMPUTE 
	WHERE hardware_type = 'gpu_training'
	  AND gpu_model = 'H100_SXM5' 
	  AND geographic_region IN ['us-west-1', 'us-west-2', 'us-east-1']
	  AND reputation_score >= 0.95
	  AND current_availability = 'available'
	ORDER BY base_price_per_hour_usdc ASC
	LIMIT 10
	`
	
	// Example 2: Reserve multi-GPU training cluster
	query2 := `
	RESERVE COMPUTE {
		"hardware_type": "gpu_training",
		"gpu_model": "H100_SXM5", 
		"required_units": 8,
		"interconnect": "NVLink",
		"estimated_duration_hours": 12,
		"max_price_per_hour_usdc": 25.00
	}
	WITH FAILOVER POLICY 'auto_migrate',
		 PREFERRED_REGIONS ['us-west-1', 'us-west-2'],
		 MIN_REPUTATION 0.90
	ESCROW 2400.00 USDC
	`
	
	// Example 3: Analyze pricing trends for inference workloads
	query3 := `
	ANALYZE PRICING 
	FOR HARDWARE_TYPE 'gpu_inference' 
		AND TIME_RANGE LAST 30 DAYS
	GROUPBY gpu_model, geographic_region
	`
	
	// These would be parsed and optimized by the query engine
	_ = query1
	_ = query2
	_ = query3
}
