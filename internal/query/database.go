package query

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DatabaseQueryEngine implements OCX-QL with database integration
type DatabaseQueryEngine struct {
	db *sql.DB
}

// NewDatabaseQueryEngine creates a new database-backed query engine
func NewDatabaseQueryEngine(db *sql.DB) *DatabaseQueryEngine {
	return &DatabaseQueryEngine{db: db}
}

// QueryResult represents the result of a query
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
	Time    time.Duration   `json:"execution_time"`
}

// ExecuteQuery executes an OCX-QL query against the database
func (e *DatabaseQueryEngine) ExecuteQuery(query *OCXQuery) (*QueryResult, error) {
	start := time.Now()
	
	// Generate SQL from OCX-QL
	sqlQuery, err := e.generateSQL(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}
	
	// Execute query
	rows, err := e.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	
	// Scan rows
	var resultRows [][]interface{}
	for rows.Next() {
		// Create slice of interface{} for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		resultRows = append(resultRows, values)
	}
	
	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
		Time:    time.Since(start),
	}, nil
}

// generateSQL converts OCX-QL to SQL
func (e *DatabaseQueryEngine) generateSQL(query *OCXQuery) (string, error) {
	switch query.Type {
	case "SELECT":
		return e.generateSelectSQL(query)
	case "RESERVE":
		return e.generateReserveSQL(query)
	case "ANALYZE":
		return e.generateAnalyzeSQL(query)
	default:
		return "", fmt.Errorf("unsupported query type: %s", query.Type)
	}
}

func (e *DatabaseQueryEngine) generateSelectSQL(query *OCXQuery) (string, error) {
	var sql strings.Builder
	
	// SELECT clause
	sql.WriteString("SELECT ")
	if len(query.Fields) == 0 {
		sql.WriteString("*")
	} else {
		for i, field := range query.Fields {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(field)
		}
	}
	
	// FROM clause
	sql.WriteString(" FROM ")
	sql.WriteString(query.Table)
	
	// WHERE clause
	if len(query.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		for i, condition := range query.Conditions {
			if i > 0 {
				sql.WriteString(" AND ")
			}
			sql.WriteString(condition.Field)
			sql.WriteString(" ")
			sql.WriteString(condition.Operator)
			sql.WriteString(" ")
			sql.WriteString(fmt.Sprintf("'%v'", condition.Value))
		}
	}
	
	// ORDER BY clause
	if query.OrderBy != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(query.OrderBy)
		if query.OrderDesc {
			sql.WriteString(" DESC")
		}
	}
	
	// LIMIT clause
	if query.Limit > 0 {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit))
	}
	
	return sql.String(), nil
}

func (e *DatabaseQueryEngine) generateReserveSQL(query *OCXQuery) (string, error) {
	// RESERVE queries are for finding available compute units
	var sql strings.Builder
	
	sql.WriteString(`
		SELECT cu.unit_id, cu.provider_id, cu.hardware_type, cu.gpu_model, 
		       cu.gpu_memory_gb, cu.base_price_per_hour_usdc, cu.current_availability,
		       p.operator_address, p.geographic_region, p.reputation_score
		FROM compute_units cu
		JOIN providers p ON cu.provider_id = p.provider_id
		WHERE cu.current_availability = 'available'
	`)
	
	// Add hardware type filter
	if len(query.Conditions) > 0 {
		for _, condition := range query.Conditions {
			if condition.Field == "hardware_type" {
				sql.WriteString(fmt.Sprintf(" AND cu.hardware_type = '%v'", condition.Value))
			}
			if condition.Field == "max_price" {
				sql.WriteString(fmt.Sprintf(" AND cu.base_price_per_hour_usdc <= %v", condition.Value))
			}
			if condition.Field == "min_reputation" {
				sql.WriteString(fmt.Sprintf(" AND p.reputation_score >= %v", condition.Value))
			}
		}
	}
	
	sql.WriteString(" ORDER BY cu.base_price_per_hour_usdc ASC")
	
	return sql.String(), nil
}

func (e *DatabaseQueryEngine) generateAnalyzeSQL(query *OCXQuery) (string, error) {
	// ANALYZE queries are for performance analytics
	var sql strings.Builder
	
	sql.WriteString(`
		SELECT 
			DATE_TRUNC('hour', timestamp) as hour,
			AVG(gpu_utilization_percent) as avg_gpu_util,
			AVG(cpu_utilization_percent) as avg_cpu_util,
			AVG(gpu_temperature_celsius) as avg_temp,
			COUNT(*) as metric_count
		FROM session_metrics sm
		JOIN compute_sessions cs ON sm.session_id = cs.session_id
		WHERE cs.session_status = 'active'
	`)
	
	// Add time range filter
	if len(query.Conditions) > 0 {
		for _, condition := range query.Conditions {
			if condition.Field == "start_time" {
				sql.WriteString(fmt.Sprintf(" AND sm.timestamp >= '%v'", condition.Value))
			}
			if condition.Field == "end_time" {
				sql.WriteString(fmt.Sprintf(" AND sm.timestamp <= '%v'", condition.Value))
			}
		}
	}
	
	sql.WriteString(`
		GROUP BY DATE_TRUNC('hour', timestamp)
		ORDER BY hour DESC
	`)
	
	return sql.String(), nil
}

// GetAvailableUnits returns available compute units for matching
func (e *DatabaseQueryEngine) GetAvailableUnits(hardwareType string, maxPrice float64, minReputation float64) (*QueryResult, error) {
	query := &OCXQuery{
		Type: "RESERVE",
		Conditions: []QueryCondition{
			{Field: "hardware_type", Operator: "=", Value: hardwareType},
			{Field: "max_price", Operator: "<=", Value: maxPrice},
			{Field: "min_reputation", Operator: ">=", Value: minReputation},
		},
	}
	
	return e.ExecuteQuery(query)
}

// GetProviderStats returns statistics for a provider
func (e *DatabaseQueryEngine) GetProviderStats(providerID string) (*QueryResult, error) {
	query := &OCXQuery{
		Type: "SELECT",
		Table: "providers",
		Fields: []string{"provider_id", "operator_address", "reputation_score", "status", "registration_timestamp"},
		Conditions: []QueryCondition{
			{Field: "provider_id", Operator: "=", Value: providerID},
		},
	}
	
	return e.ExecuteQuery(query)
}

// GetOrderHistory returns order history for analysis
func (e *DatabaseQueryEngine) GetOrderHistory(requesterID string, limit int) (*QueryResult, error) {
	query := &OCXQuery{
		Type: "SELECT",
		Table: "compute_orders",
		Fields: []string{"order_id", "required_hardware_type", "max_price_per_hour_usdc", "order_status", "placed_at"},
		Conditions: []QueryCondition{
			{Field: "requester_id", Operator: "=", Value: requesterID},
		},
		OrderBy: "placed_at",
		OrderDesc: true,
		Limit: limit,
	}
	
	return e.ExecuteQuery(query)
}
